package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// ev marshals a single `go test -json` event line.
func ev(action, pkg, test string) string {
	b, _ := json.Marshal(testEvent{Action: action, Package: pkg, Test: test})
	return string(b)
}

// eventBuilder assembles a JSON event stream as a list of lines.
type eventBuilder struct {
	lines []string
}

func (b *eventBuilder) add(action, pkg, test string) {
	b.lines = append(b.lines, ev(action, pkg, test))
}

func (b *eventBuilder) rp(pkg, test string) {
	b.add("run", pkg, test)
	b.add("pass", pkg, test)
}

func (b *eventBuilder) pkgPass(short string) {
	b.add("pass", modulePrefix+short, "")
}

func (b *eventBuilder) String() string { return strings.Join(b.lines, "\n") + "\n" }

// goodStream builds a fully passing stream: every manifest root and required
// child runs to a terminal pass, all six mandatory packages complete, and the
// server-free Postgres-named unit tests pass.
func goodStream() string {
	b := &eventBuilder{}
	for _, p := range postgresManifest {
		for _, rt := range p.tests {
			b.rp(modulePrefix+p.pkg, rt.name)
			for _, child := range rt.children {
				b.rp(modulePrefix+p.pkg, rt.name+"/"+child)
			}
		}
		b.pkgPass(p.pkg)
	}
	// Server-free tests (excluded from live execution) still pass.
	b.rp(modulePrefix+"test", "TestPostgresDSNForDatabase")
	b.rp(modulePrefix+"test", "TestPostgresConfigFromEnv")
	b.rp(modulePrefix+"test", "TestPostgresDB_invalidConnection")
	return b.String()
}

// withoutEvents returns the lines of stream with every event for the given
// (package, test) identity removed (covers run and terminal events together).
func withoutEvents(stream, pkg, test string) string {
	var out []string
	for _, l := range strings.Split(strings.TrimSuffix(stream, "\n"), "\n") {
		if l == "" {
			continue
		}
		var e testEvent
		if err := json.Unmarshal([]byte(l), &e); err != nil {
			continue
		}
		if e.Package == pkg && e.Test == test {
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n") + "\n"
}

// replaceTerminal rewrites the terminal action of a single (package, test)
// identity. Only the first run event is kept and the terminal becomes the given
// action.
func replaceTerminal(stream, pkg, test, action string) string {
	var out []string
	seen := false
	for _, l := range strings.Split(strings.TrimSuffix(stream, "\n"), "\n") {
		if l == "" {
			continue
		}
		var e testEvent
		if err := json.Unmarshal([]byte(l), &e); err == nil && e.Package == pkg && e.Test == test {
			if !seen {
				out = append(out, ev("run", pkg, test))
				out = append(out, ev(action, pkg, test))
				seen = true
			}
			continue
		}
		out = append(out, l)
	}
	if !seen {
		// Identity absent: append run + terminal so the stream stays well-formed
		// for the specific mutation being tested.
		out = append(out, ev("run", pkg, test))
		out = append(out, ev(action, pkg, test))
	}
	return strings.Join(out, "\n") + "\n"
}

// runChecker validates a stream and returns the exit code and printed report.
func runChecker(t *testing.T, stream string) (int, string) {
	t.Helper()
	obs := newObservedResult()
	obs.ingest(strings.NewReader(stream))
	rep := obs.validate()
	var buf bytes.Buffer
	rep.printReport(&buf)
	code := 0
	if len(rep.violations) > 0 {
		code = 1
	}
	return code, buf.String()
}

func TestManifestIsNonEmptyAndCoversAllSixPackages(t *testing.T) {
	if len(postgresManifest) != 6 {
		t.Fatalf("manifest must cover exactly six mandatory packages, got %d", len(postgresManifest))
	}
	seen := map[string]bool{}
	totalRoots := 0
	totalChildren := 0
	for _, p := range postgresManifest {
		if seen[p.pkg] {
			t.Fatalf("duplicate package %q in manifest", p.pkg)
		}
		seen[p.pkg] = true
		for _, rt := range p.tests {
			totalRoots++
			if rt.name == "" {
				t.Fatalf("package %s has an empty required root", p.pkg)
			}
			totalChildren += len(rt.children)
		}
	}
	if totalRoots == 0 {
		t.Fatal("manifest must contain at least one required live suite root")
	}
	if totalChildren == 0 {
		t.Fatal("manifest must enumerate at least the fractional/reopen child identities")
	}
	for _, pkg := range []string{"statistics", "dashboard", "device", "test", "database", "model"} {
		if !seen[pkg] {
			t.Fatalf("mandatory package %s missing from manifest", pkg)
		}
	}
}

func TestCompleteManifestPasses(t *testing.T) {
	code, out := runChecker(t, goodStream())
	if code != 0 {
		t.Fatalf("expected pass for complete manifest, got exit %d:\n%s", code, out)
	}
	if !strings.Contains(out, "required live suites:") {
		t.Fatalf("report must include live-suite summary:\n%s", out)
	}
}

func TestOnlyServerFreePostgresNamedTestsPassingRejected(t *testing.T) {
	// All six packages complete, but only the server-free Postgres-named tests
	// pass. No manifest root runs -> the gate must reject.
	b := &eventBuilder{}
	b.rp(modulePrefix+"test", "TestPostgresDSNForDatabase")
	b.rp(modulePrefix+"test", "TestPostgresConfigFromEnv")
	b.rp(modulePrefix+"test", "TestPostgresDB_invalidConnection")
	for _, p := range postgresManifest {
		b.pkgPass(p.pkg)
	}
	code, out := runChecker(t, b.String())
	if code == 0 {
		t.Fatalf("server-free-only stream must be rejected:\n%s", out)
	}
	for _, want := range []string{"zero required live PostgreSQL suites completed", "never ran"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in rejection output:\n%s", want, out)
		}
	}
}

func TestRequiredSuiteAbsentRejected(t *testing.T) {
	stream := withoutEvents(goodStream(), modulePrefix+"statistics", "TestStats2_daily_postgres")
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("missing required suite must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "required live suite statistics.TestStats2_daily_postgres never ran") {
		t.Fatalf("expected missing-suite violation:\n%s", out)
	}
}

func TestRequiredChildAbsentDespiteParentPassRejected(t *testing.T) {
	stream := withoutEvents(goodStream(), modulePrefix+"statistics",
		"TestStats_postgres/fractional_duplicate_and_empty")
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("missing required fractional child despite parent pass must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "required child statistics.TestStats_postgres/fractional_duplicate_and_empty never ran") {
		t.Fatalf("expected missing-child violation:\n%s", out)
	}
}

func TestReopenChildAbsentDespiteParentPassRejected(t *testing.T) {
	stream := withoutEvents(goodStream(), modulePrefix+"dashboard",
		"TestStartupMigration_reopen_textAndBytea/bytea_schema")
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("missing bytea_schema reopen child must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "required child dashboard.TestStartupMigration_reopen_textAndBytea/bytea_schema never ran") {
		t.Fatalf("expected missing reopen-child violation:\n%s", out)
	}
}

func TestNestedLiveChildSkippedRejected(t *testing.T) {
	stream := replaceTerminal(goodStream(), modulePrefix+"statistics",
		"TestStats_postgres/fractional_same-second_identities", "skip")
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("skipped required live child must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "required child statistics.TestStats_postgres/fractional_same-second_identities was skipped") {
		t.Fatalf("expected skipped-child violation:\n%s", out)
	}
}

func TestDeeperNestedDescendantSkippedRejected(t *testing.T) {
	stream := goodStream() + ev("run", modulePrefix+"statistics", "TestStats_postgres/extra/nested") + "\n" +
		ev("skip", modulePrefix+"statistics", "TestStats_postgres/extra/nested") + "\n"
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("skipped deeper-nested descendant of a manifest root must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "descendant statistics.TestStats_postgres/extra/nested of required suite was skipped") {
		t.Fatalf("expected nested-descendant-skip violation:\n%s", out)
	}
}

func TestMandatoryPackageOmittedRejected(t *testing.T) {
	stream := withoutEvents(goodStream(), modulePrefix+"model", "")
	// withoutEvents matches Test=="" for the package-level pass event too; ensure
	// the model package-level pass is gone.
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("omitted mandatory package must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "mandatory package model is missing a terminal completion event") {
		t.Fatalf("expected omitted-package violation:\n%s", out)
	}
}

func TestZeroLiveExecutionRejected(t *testing.T) {
	// Packages pass but no manifest root ever runs.
	b := &eventBuilder{}
	for _, p := range postgresManifest {
		b.pkgPass(p.pkg)
	}
	code, out := runChecker(t, b.String())
	if code == 0 {
		t.Fatalf("zero live execution must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "zero required live PostgreSQL suites completed") {
		t.Fatalf("expected zero-live-execution violation:\n%s", out)
	}
}

func TestFailedRequiredSuiteRejected(t *testing.T) {
	stream := replaceTerminal(goodStream(), modulePrefix+"device", "TestDeviceLifecycle_pgReconnect", "fail")
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("failed required suite must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "required live suite device.TestDeviceLifecycle_pgReconnect failed") {
		t.Fatalf("expected failed-suite violation:\n%s", out)
	}
}

func TestMalformedJSONRejected(t *testing.T) {
	stream := goodStream() + "this is not json\n"
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("malformed JSON must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "malformed JSON") {
		t.Fatalf("expected malformed-JSON violation:\n%s", out)
	}
}

func TestTruncatedStreamMissingTerminalRejected(t *testing.T) {
	// Dashboard package runs TestDashboardAPI_postgres but the terminal pass is
	// never emitted (truncated stream).
	stream := withoutEvents(goodStream(), modulePrefix+"dashboard", "TestDashboardAPI_postgres") +
		ev("run", modulePrefix+"dashboard", "TestDashboardAPI_postgres") + "\n"
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("truncated stream (missing terminal) must be rejected:\n%s", out)
	}
	if !strings.Contains(out, "required live suite dashboard.TestDashboardAPI_postgres is missing a terminal pass event") {
		t.Fatalf("expected missing-terminal violation:\n%s", out)
	}
}

func TestWrongPackageIdentityDoesNotSatisfyManifest(t *testing.T) {
	// A test literally named TestStats_postgres in the dashboard package must
	// NOT satisfy the statistics package's manifest entry.
	stream := withoutEvents(goodStream(), modulePrefix+"statistics", "TestStats_postgres") +
		ev("run", modulePrefix+"dashboard", "TestStats_postgres") + "\n" +
		ev("pass", modulePrefix+"dashboard", "TestStats_postgres") + "\n"
	code, out := runChecker(t, stream)
	if code == 0 {
		t.Fatalf("wrong-package identity must not satisfy a manifest entry:\n%s", out)
	}
	if !strings.Contains(out, "required live suite statistics.TestStats_postgres never ran") {
		t.Fatalf("expected statistics.TestStats_postgres missing violation:\n%s", out)
	}
}

func TestRunReadsFromFileArgument(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/results.json"
	if err := os.WriteFile(path, []byte(goodStream()), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"check-postgres-results", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected pass reading from file, got exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "required live suites:") {
		t.Fatalf("expected summary in stdout:\n%s", stdout.String())
	}
}

// Explicit identities avoid a self-fulfilling test that only enumerates the manifest.
func TestStats2FractionalSuitesAreMandatory(t *testing.T) {
	for _, name := range []string{
		"TestStats2_single_fractional_postgres",
		"TestStats2_single_fractional_postgres_sessionTimeZone",
	} {
		t.Run(name, func(t *testing.T) {
			for _, action := range []string{"absent", "skip"} {
				stream := withoutEvents(goodStream(), modulePrefix+"statistics", name)
				if action == "skip" {
					stream = replaceTerminal(stream, modulePrefix+"statistics", name, "skip")
				}
				code, out := runChecker(t, stream)
				if code == 0 || !strings.Contains(out, name) {
					t.Fatalf("%s suite must fail: %s", action, out)
				}
			}
		})
	}
}

func TestUnlistedPostgresTestsCannotSkipOrDisappearMidRun(t *testing.T) {
	for _, name := range []string{"TestFuture_postgres", "TestFuture_pgReconnect/nested"} {
		for _, action := range []string{"skip", "fail", "incomplete", "skip-without-run"} {
			stream := goodStream()
			if action != "skip-without-run" {
				stream += ev("run", modulePrefix+"statistics", name) + "\n"
			}
			if action != "incomplete" {
				terminal := action
				if action == "skip-without-run" {
					terminal = "skip"
				}
				stream += ev(terminal, modulePrefix+"statistics", name) + "\n"
			}
			code, out := runChecker(t, stream)
			if code == 0 || !strings.Contains(out, name) {
				t.Fatalf("%s %s must fail: %s", name, action, out)
			}
		}
	}
}

func TestUnlistedPostgresPassAndServerFreeSkipsAllowed(t *testing.T) {
	stream := goodStream() + ev("run", modulePrefix+"statistics", "TestFuture_postgres") + "\n" +
		ev("pass", modulePrefix+"statistics", "TestFuture_postgres") + "\n"
	stream = replaceTerminal(stream, modulePrefix+"test", "TestPostgresConfigFromEnv", "skip")
	stream += ev("skip", modulePrefix+"test", "TestPostgresConfigFromEnv/server_free_child") + "\n"
	code, out := runChecker(t, stream)
	if code != 0 {
		t.Fatalf("passing live suite and explicitly server-free skips should pass: %s", out)
	}
}
