// Command check-postgres-results is the enforcement half of the dedicated
// PostgreSQL gate (FR-006 / NFR-002). It validates the `go test -json` result
// stream emitted by hack/test-databases.sh (with TRAGGO_TEST_GO_JSON=1) against
// an explicit, reviewed manifest of required live PostgreSQL suites.
//
// A green exit proves that every expected real-engine suite root and its
// required children ran to a terminal pass, that no observed descendant of a
// manifest root was skipped or failed, that every mandatory package completed,
// and that the result stream was well-formed and complete. Server-free tests
// (DSN parsing / invalid-connection unit tests) never satisfy the manifest and
// never count toward live execution.
//
// It reads JSON test events from a file path argument (or stdin when no
// argument is given) and exits non-zero on any violation. Only sanitized test
// identities and counts are reported; connection secrets are never read or
// printed.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// modulePrefix is the fully qualified package prefix emitted by `go test -json`.
const modulePrefix = "github.com/traggo/server/"

// serverFreeTests never establish a connection to the configured server. They
// are excluded from live-execution counts and cannot satisfy a manifest entry.
var serverFreeTests = map[string]bool{
	"test.TestPostgresDSNForDatabase":       true,
	"test.TestPostgresConfigFromEnv":        true,
	"test.TestPostgresDB_invalidConnection": true,
}

// requiredTest is one required live PostgreSQL suite root and its expected
// children. Children are subtest identities relative to the root in go-test
// sanitized form (spaces become underscores), e.g. "fractional_same-second_identities".
type requiredTest struct {
	name     string
	children []string
}

// requiredPackage pairs a mandatory package (short name relative to the module,
// e.g. "statistics") with its required live suite roots. A package may carry no
// required roots (e.g. model) yet must still complete for the gate to pass.
type requiredPackage struct {
	pkg   string
	tests []requiredTest
}

// postgresManifest is the reviewed, non-empty expected live-suite manifest,
// keyed by the exact Go `Package` and `Test` identities. Package names are
// relative to module github.com/traggo/server. It is reconciled with the final
// WP02 fractional fixture identities and WP03 database regressions.
var postgresManifest = []requiredPackage{
	{
		pkg: "statistics",
		tests: []requiredTest{
			{
				name: "TestStats_postgres",
				children: []string{
					"fractional_same-second_identities",
					"fractional_duplicate_and_empty",
					"fractional_sum-of-rounded_vs_rounded-sum",
				},
			},
			{
				name: "TestStats_postgres_sessionTimeZone",
				children: []string{
					"fractional_same-second_identities",
					"fractional_duplicate_and_empty",
					"fractional_sum-of-rounded_vs_rounded-sum",
				},
			},
			{name: "TestStats2_daily_postgres"},
			{name: "TestStats2_daily_postgres_sessionTimeZone"},
			{name: "TestStats2_single_fractional_postgres"},
			{name: "TestStats2_single_fractional_postgres_sessionTimeZone"},
			{name: "TestMidnightCrossing_manualHalfOpen_postgres"},
		},
	},
	{
		pkg: "dashboard",
		tests: []requiredTest{
			{name: "TestDashboardEntryPersistence_text"},
			{name: "TestDashboardEntryPersistence_bytea"},
			{name: "TestDashboardLifecycle_pgReconnect"},
			{
				name:     "TestStartupMigration_reopen_textAndBytea",
				children: []string{"text_schema", "bytea_schema"},
			},
			{name: "TestDashboardAPI_postgres"},
			{name: "TestStatsGraphQLShapes_postgres"},
		},
	},
	{
		pkg: "device",
		tests: []requiredTest{
			{name: "TestDevicePersistence_text"},
			{name: "TestDevicePersistence_bytea"},
			{name: "TestDeviceLifecycle_pgReconnect"},
		},
	},
	{
		pkg: "test",
		tests: []requiredTest{
			{name: "TestPostgresDB_isolation"},
			{name: "TestPostgresDB_cleanup"},
		},
	},
	{
		pkg: "database",
		tests: []requiredTest{
			{name: "TestMigrationFailure_postgres"},
			{name: "TestInitializationReopen_postgres"},
		},
	},
	{
		pkg:   "model",
		tests: []requiredTest{},
	},
}

// testEvent is one line of a `go test -json` event stream.
type testEvent struct {
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
}

// observedResult accumulates the raw events needed for validation.
type observedResult struct {
	// terminal holds the last terminal action ("pass", "fail", "skip") per test
	// identity, keyed by package+"\x00"+test.
	terminal map[string]string
	// ran records whether a "run" action was observed for a test identity.
	ran map[string]bool
	// pkgTerminal holds the last package-level terminal action per package.
	pkgTerminal map[string]string
	// malformed holds line numbers that were not valid JSON.
	malformed []int
	// allTests maps a short-package name to the set of observed (unqualified)
	// test identities under that package.
	allTests map[string]map[string]bool
}

func newObservedResult() *observedResult {
	return &observedResult{
		terminal:    map[string]string{},
		ran:         map[string]bool{},
		pkgTerminal: map[string]string{},
		allTests:    map[string]map[string]bool{},
	}
}

// identity builds the composite key for a test event.
func identity(pkg, test string) string { return pkg + "\x00" + test }

// ingest parses an event stream, recording the raw observations used for
// validation. A non-empty line that is not valid JSON is recorded as malformed.
func (o *observedResult) ingest(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var ev testEvent
		if err := json.Unmarshal([]byte(text), &ev); err != nil {
			o.malformed = append(o.malformed, line)
			continue
		}
		short := strings.TrimPrefix(ev.Package, modulePrefix)
		if short == ev.Package {
			// Package not under the module; skip (e.g. third-party output).
			continue
		}
		switch ev.Action {
		case "run":
			o.ran[identity(ev.Package, ev.Test)] = true
			if ev.Test != "" {
				if o.allTests[short] == nil {
					o.allTests[short] = map[string]bool{}
				}
				o.allTests[short][ev.Test] = true
			}
		case "pass", "fail", "skip":
			if ev.Test == "" {
				o.pkgTerminal[short] = ev.Action
			} else {
				o.terminal[identity(ev.Package, ev.Test)] = ev.Action
				if o.allTests[short] == nil {
					o.allTests[short] = map[string]bool{}
				}
				o.allTests[short][ev.Test] = true
			}
		}
	}
}

// violation records a gate failure.
type violation struct {
	message string
}

// report summarizes a validation run.
type report struct {
	violations []string
	// livePassed counts manifest roots that ran to a terminal pass.
	livePassed int
	// liveRoots is the total number of required manifest roots.
	liveRoots int
	// childrenRequired is the total number of required child identities.
	childrenRequired int
	// childrenPassed counts required children that ran to a terminal pass.
	childrenPassed int
	// skippedDescendants counts observed descendants of a manifest root that
	// were skipped (these are violations).
	skippedDescendants int
	// skippedServerFree counts skips of server-free tests (informational only).
	skippedServerFree int
}

func (r *report) addf(format string, args ...interface{}) {
	r.violations = append(r.violations, fmt.Sprintf(format, args...))
}

// shortPackageName returns the module-relative short name of a fully qualified
// package path, or "" if the path is outside the module.
func shortPackageName(pkg string) string {
	if !strings.HasPrefix(pkg, modulePrefix) {
		return ""
	}
	return strings.TrimPrefix(pkg, modulePrefix)
}

// validate performs the manifest enforcement over the observed stream.
func (o *observedResult) validate() *report {
	rep := &report{}

	if len(o.malformed) > 0 {
		rep.addf("malformed JSON on %d line(s): %v", len(o.malformed), o.malformed)
	}

	// Build a lookup of required roots and a set of all required root names per
	// short package for descendant scanning.
	requiredRoots := map[string]map[string]requiredTest{} // shortPkg -> rootName -> requiredTest
	for _, p := range postgresManifest {
		requiredRoots[p.pkg] = map[string]requiredTest{}
		for _, rt := range p.tests {
			requiredRoots[p.pkg][rt.name] = rt
		}
		rep.liveRoots += len(p.tests)
		for _, rt := range p.tests {
			rep.childrenRequired += len(rt.children)
		}
	}

	// 1. Required roots: run event followed by a terminal pass.
	for shortPkg, tests := range requiredRoots {
		pkg := modulePrefix + shortPkg
		for name := range tests {
			key := identity(pkg, name)
			if !o.ran[key] {
				rep.addf("required live suite %s.%s never ran", shortPkg, name)
				continue
			}
			switch o.terminal[key] {
			case "pass":
				rep.livePassed++
			case "skip":
				rep.addf("required live suite %s.%s was skipped", shortPkg, name)
			case "fail":
				rep.addf("required live suite %s.%s failed", shortPkg, name)
			default:
				rep.addf("required live suite %s.%s is missing a terminal pass event (truncated stream)", shortPkg, name)
			}
		}
	}

	// 2. Required children: run event followed by a terminal pass.
	for shortPkg, tests := range requiredRoots {
		pkg := modulePrefix + shortPkg
		for _, rt := range tests {
			for _, child := range rt.children {
				full := rt.name + "/" + child
				key := identity(pkg, full)
				if !o.ran[key] {
					rep.addf("required child %s.%s never ran", shortPkg, full)
					continue
				}
				switch o.terminal[key] {
				case "pass":
					rep.childrenPassed++
				case "skip":
					rep.addf("required child %s.%s was skipped", shortPkg, full)
				case "fail":
					rep.addf("required child %s.%s failed", shortPkg, full)
				default:
					rep.addf("required child %s.%s is missing a terminal pass event (truncated stream)", shortPkg, full)
				}
			}
		}
	}

	// 3. Every observed descendant of a manifest root must not be skipped or
	// failed, and must have a terminal pass (covers deeper nested subtests).
	for shortPkg, tests := range requiredRoots {
		for _, rt := range tests {
			prefix := rt.name + "/"
			for test := range o.allTests[shortPkg] {
				if !strings.HasPrefix(test, prefix) {
					continue
				}
				key := identity(modulePrefix+shortPkg, test)
				switch o.terminal[key] {
				case "skip":
					rep.skippedDescendants++
					rep.addf("descendant %s.%s of required suite was skipped", shortPkg, test)
				case "fail":
					rep.addf("descendant %s.%s of required suite failed", shortPkg, test)
				case "":
					if o.ran[key] {
						rep.addf("descendant %s.%s of required suite is missing a terminal event (truncated stream)", shortPkg, test)
					}
				}
			}
		}
	}

	// Also enforce newly added PostgreSQL suites outside the manifest. Missing
	// expected roots are checked above; observed live tests cannot silently skip
	// merely because the manifest has not yet been updated for their names.
	for shortPkg, tests := range o.allTests {
		for name := range tests {
			root := strings.SplitN(name, "/", 2)[0]
			if serverFreeTests[shortPkg+"."+root] {
				continue
			}
			if _, required := requiredRoots[shortPkg][root]; required {
				continue // already checked, including descendants
			}
			lower := strings.ToLower(root)
			if !strings.Contains(lower, "postgres") && !strings.Contains(lower, "_pg") {
				continue
			}
			key := identity(modulePrefix+shortPkg, name)
			switch o.terminal[key] {
			case "pass":
			case "skip":
				rep.skippedDescendants++
				rep.addf("live PostgreSQL test %s.%s was skipped", shortPkg, name)
			case "fail":
				rep.addf("live PostgreSQL test %s.%s failed", shortPkg, name)
			default:
				rep.addf("live PostgreSQL test %s.%s is missing a terminal pass event", shortPkg, name)
			}
		}
	}

	// 4. Mandatory packages must complete (terminal package-level pass).
	for _, p := range postgresManifest {
		switch o.pkgTerminal[p.pkg] {
		case "pass":
		case "skip":
			rep.addf("mandatory package %s was skipped (not completed)", p.pkg)
		case "fail":
			rep.addf("mandatory package %s failed to complete", p.pkg)
		default:
			rep.addf("mandatory package %s is missing a terminal completion event (omitted or truncated)", p.pkg)
		}
	}

	// 5. Zero completed live suite roots is never acceptable.
	if rep.livePassed == 0 {
		rep.addf("zero required live PostgreSQL suites completed")
	}

	// 6. Informational skip count for server-free tests (not a violation).
	for _, p := range postgresManifest {
		pkg := modulePrefix + p.pkg
		for name := range o.allTests[p.pkg] {
			if serverFreeTests[p.pkg+"."+name] && o.terminal[identity(pkg, name)] == "skip" {
				rep.skippedServerFree++
			}
		}
	}

	return rep
}

// printReport writes the sanitized summary to w.
func (r *report) printReport(w io.Writer) {
	fmt.Fprintf(w, "PostgreSQL result manifest summary:\n")
	fmt.Fprintf(w, "  required live suites: %d/%d passed\n", r.livePassed, r.liveRoots)
	fmt.Fprintf(w, "  required children:    %d/%d passed\n", r.childrenPassed, r.childrenRequired)
	fmt.Fprintf(w, "  live-suite skips:     %d (any is a failure)\n", r.skippedDescendants)
	fmt.Fprintf(w, "  server-free skips:    %d (informational, excluded)\n", r.skippedServerFree)
	if len(r.violations) > 0 {
		sort.Strings(r.violations)
		fmt.Fprintf(w, "violations (%d):\n", len(r.violations))
		for _, v := range r.violations {
			fmt.Fprintf(w, "  - %s\n", v)
		}
	}
}

func run(args []string, stdout, stderr io.Writer) int {
	var reader io.Reader
	if len(args) > 1 {
		f, err := os.Open(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "error: open %s: %v\n", args[1], err)
			return 2
		}
		defer f.Close()
		reader = f
	} else {
		reader = os.Stdin
	}

	obs := newObservedResult()
	obs.ingest(reader)
	rep := obs.validate()
	rep.printReport(stdout)

	if len(rep.violations) > 0 {
		fmt.Fprintln(stderr, "error: PostgreSQL gate failed")
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}
