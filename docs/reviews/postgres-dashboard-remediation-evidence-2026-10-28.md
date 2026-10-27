# PostgreSQL Dashboard remediation — revision-bound verification evidence

- **Mission**: `postgres-dashboard-review-fixes-01M5ZJTF`
- **Work package**: WP05 (fresh integrated verification and common evidence)
- **Date**: 2026-10-28 (Europe/Samara)
- **Author**: Implementer Ivan (implementer profile), executed on a lane worktree

## Scope and honesty statement

This document is the single, secret-free evidence record for the remediation
revision produced by WP01–WP04. It records what was actually executed and what
could **not** be executed in this environment, with every claimed pass backed
by a retained command, exit code and (where applicable) test/skip counts.

**Two hard environment/verification blockers are flagged up front and kept
open (not fabricated):**

1. **Actual PostgreSQL 17 could not be executed.** No `postgres:17` image is
   available in this environment and none could be obtained:
   - Docker Hub `postgres:17` pull fails: `tls: failed to verify certificate:
     x509: certificate signed by unknown authority` (corporate network).
   - Corporate mirror `docker-proxy.artifactory.tcsbank.ru/postgres:17` requires
     an auth token; configuring that auth (`dp artifacts docker configure`) is
     blocked by the host auto-mode policy (credential/system change).
   - The only locally available PostgreSQL image is **postgres:18** (wrong
     major). Consequently the positive PG17 lifecycle gate (T022) is **NOT
     verified**; the "zero skipped PG cases" acceptance criterion is **not** met.
2. **A real defect was discovered in WP01's owned launcher** that blocks *all*
   real PostgreSQL execution through the launcher, on both the disposable and
   the reused-DSN paths (see "Discovered defect routed to WP01" below). This is
   routed back to WP01 per the WP05 ownership contract; it is **not** amended
   here.

## Tested source revision

- **Tested commit**: `38ae5e9e3818030b1da410a652fdb012877d2973`
  (`feat(WP04): wire launcher CI gates and enforce PostgreSQL live-suite manifest`)
- **Branch**: `kitty/mission-postgres-dashboard-review-fixes-01M5ZJTF-lane-e`
- **Working tree**: clean at time of verification — `git status --porcelain`
  empty (no dirty patch identity).
- **Diff against mission base** `6e4c888`: includes all WP01–WP04 remediation
  source (18 files, +2002/−198): launcher `hack/test-databases.sh(_test.sh)`,
  checker `hack/check-postgres-results.go(_test.go)`, Makefile/CI wiring,
  fractional statistics, migration-error propagation, and their regressions.
- The generated Go (`make generate-go`) and UI GraphQL (`make generate-js`)
  outputs are gitignored; regeneration produced no tracked diff, so the tested
  revision matches the committed remediation source.

## Environment / tool versions

| Tool | Version | Notes |
| --- | --- | --- |
| Go | `go1.25.5 darwin/arm64` | module directive `go 1.24.0`; CI mentions 1.25 |
| CGO | `CGO_ENABLED=1`, `CC=cc` | |
| gqlgen | `v0.17.85` | installed via `make download-tools` |
| goimports | `v0.1.10` | installed via `make download-tools` |
| Node | `v22.20.0` | |
| Yarn | `1.22.22` | |
| Docker | `27.3.1` (daemon running) | |
| lib/pq | `v1.0.0` (go.mod) | pinned |
| psql client | 14.15 (Homebrew) | not used for the server |

Sanitized command/result logs for each block below were retained locally
(scratch path `/tmp/wp05-evidence/*.txt`) for reviewer reproduction; no secret
or credential-bearing capture is retained anywhere.

---

## T020 — Prepare the verification environment and identify source

All preparation commands ran with `PATH` prefixed with `$(go env GOPATH)/bin`
so the freshly installed `gqlgen` / `goimports` are resolvable.

| Command | Exit | Result |
| --- | --- | --- |
| `make download-tools` | 0 | installed gqlgen v0.17.85, goimports v0.1.10 |
| `make install-go` (`go mod download`) | 0 | deps resolved |
| `make install-js` (`cd ui && yarn`) | 0 | node_modules installed; optional `fsevents` native-module build warning (yarn: "This module is OPTIONAL, you can safely ignore this error") |
| `make generate-go` (`gqlgen`) | 0 | generated GraphQL sources, no tracked diff |
| `make generate-js` (`cd ui && yarn generate`) | 0 | generated 42 UI GraphQL files under `ui/src/gql/__generated__/` |
| `NODE_OPTIONS=--openssl-legacy-provider make build-js` | **2 on first attempt**, then **0** | first attempt failed: `Cannot find file '../gql/__generated__/globalTypes'` (generated types absent). After `make generate-js`, re-ran and compiled successfully: `build/static/js/...` real UI assets produced. |

**Conclusion**: real UI build assets (`ui/build/`) and generated GraphQL sources
are present, not stubs. The only deviation was the expected initial `build-js`
failure before `generate-js` had produced the UI types.

---

## T021 — Fresh SQLite and regression discrimination checks

### Fresh SQLite suite (DSN absent)

```
go test -count=1 ./model/... ./statistics/... ./dashboard/... ./device/... ./database/... ./test/...
```
Exit **0**. All packages `ok`:
`model`, `statistics`, `dashboard`, `dashboard/convert`, `dashboard/dbrange`,
`dashboard/entry`, `dashboard/util`, `device`, `database`, `test`,
`test/fake`.

Verbose re-run counted **74 optional PostgreSQL SKIPs** (these are the
postgres-only cases skipped because no DSN was set). These are **SQLite-only
evidence** and prove nothing about PostgreSQL 17; they are recorded separately
and not counted toward any PG acceptance. The dedicated PG gate (T022) requires
zero skips and was not satisfied.

### Four-defect discrimination evidence (SQLite variants)

| Defect (source report) | SQLite-variant regression | Result |
| --- | --- | --- |
| DRIFT-1 fractional ranges (Codex, Nessy) | `TestStats/fractional_same-second_identities`, `fractional_duplicate_and_empty`, `fractional_sum-of-rounded_vs_rounded-sum`, `fractional_rounding_0.4s->_0`, `fractional_rounding_0.8s->_1`, `TestStats2_single_fractional_sqlite`, `TestStats/boundaries_1..3` | all PASS |
| Startup migration / silent failure (Codex RISK-1; Nessy DRIFT-1/RISK-1) | `TestNew_closesOwnedPoolOnMigrationError`, `TestNew_cleanupRunsOnce`, `TestNew_migrationCauseSurvivesCloseFailure`, `TestMigrationFailure_sqlite` | all PASS |
| Launcher loopback + credential redaction (Codex RISK-1/RISK-2) | `make test-database-launcher` (shell regression suite, command doubles) | PASS — see below |

### `make test-database-launcher`

```
./hack/test-databases_test.sh
```
Exit **0**: `16 cases run, 0 failed, 0 skipped`.

This shell suite (no Docker, no Go toolchain; synthetic `docker`/`go` doubles)
executes the launcher safety contract: loopback-only publish for default and
custom ports with exactly one self-started cleanup and none for a reused server;
URL and quoted-keyword DSN sentinel passwords absent from captured output on
success, setup failure, wrong-major and test-failure paths (including a child
that deliberately echoes the sentinel); actual-major-17 requirement (17
accepted, 16/18 rejected); JSON mode keeps child events on stdout and stage
messages on stderr; `TRAGGO_TEST_PG_PACKAGES` override; and **failure-injection**
assertions that still reject the reviewed broad-publish and DSN-logging defects.

**Conclusion**: all four reviewed defects have observable discrimination
evidence on SQLite / launcher doubles. FR-003 and FR-004 SQLite regressions
pass; FR-001/FR-002 launcher regressions pass.

---

## T022 — Actual PostgreSQL 17 and real launcher lifecycle checks

### Blocked — positive PG17 gate NOT executed

`make test-postgres` (default image `postgres:17`) was attempted and **failed**
with exit 2 because the image could not be pulled:

```
==> Starting disposable postgres:17 on 127.0.0.1:55432 (trust auth)
Unable to find image 'postgres:17' locally
docker: Error response from daemon: Get "https://registry-1.docker.io/v2/":
  tls: failed to verify certificate: x509: certificate signed by unknown authority.
==> Removing disposable PostgreSQL container traggo-test-pg-<pid>
```

The checker (`go run ./hack/check-postgres-results.go`) correctly reported the
gate as failed: `required live suites: 0/18 passed`, `required children: 0/8
passed`, and 33 violations (all six mandatory packages missing a completion
event, all required live suites "never ran"). This demonstrates the gate does
**not** accept a run with no real PG17 execution.

Notable partial launcher behavior observed even on the failed run: loopback
publish intent (`127.0.0.1:55432`) and cleanup after failure
(`Removing disposable PostgreSQL container`) — consistent with FR-001.

The corporate mirror path was also tried and requires an auth token that could
not be configured in this session (see honesty statement). Only **postgres:18**
is locally available.

**Therefore: real PostgreSQL 17 execution, real HostIp inspection on a live
PG17 container, custom-port real binding, and success/failure container cleanup
of a real PG17 container are NOT verified in this environment. This gap remains
open (NFR-002, FR-006, SC-002 not satisfied).**

### Discovered defect routed to WP01 (blocks all real PG via launcher)

Attempting a real launcher run against the locally available postgres:18 server
(and, by the same code path, any real server) exposed a defect in WP01's owned
`hack/test-databases.sh`:

- `hack/test-databases.sh:191` names the version probe helper
  `helper="${SCRIPT_DIR}/.pgversion-$$.go"` — a **dotfile**.
- `hack/test-databases.sh:223` then runs `go run "${helper}"`.
- The Go toolchain **ignores source files whose names begin with `.`**, so the
  real `go run` fails with
  `package command-line-arguments: no Go files in <hack dir>`.
- Verified independently: `go run .dotprobe.go` → "no Go files in <dir>",
  while `go run probe.go` succeeds.
- The shell regression suite does **not** catch this because it substitutes a
  fake `go` double that never compiles the helper.

**Impact**: the real `check_server_version` gate can never execute against a
real server, on either the disposable or the reused-DSN path. This means even if
`postgres:17` were available, the launcher's real version gate would still fail.
It is a genuine WP01 defect. **Routed to WP01** per the WP05 ownership contract
("a discovered sibling defect returns to its owning WP; do not conceal it with
evidence edits"). It is not amended in this WP. Real wrong-major gating via the
launcher is therefore also unverifiable until WP01 is fixed.

---

## T023 — Credential and required-mode negative checks

| Negative check | Command (sanitized) | Expected | Result |
| --- | --- | --- | --- |
| Missing DSN in required mode | `TRAGGO_TEST_POSTGRES_REQUIRED=1 go test -count=1 ./statistics/... ./database/... ./test/...` (no DSN) | nonzero | **exit 1** — PG tests FAIL, not skip: `TRAGGO_TEST_POSTGRES_DSN not set but TRAGGO_TEST_POSTGRES_REQUIRED=1: PostgreSQL tests are required and cannot run` |
| Unusable loopback DSN | `TRAGGO_TEST_POSTGRES_REQUIRED=1 TRAGGO_TEST_POSTGRES_DSN="postgres://postgres@127.0.0.1:59999/postgres?sslmode=disable&connect_timeout=3" go test -count=1 ./database/... ./test/...` | nonzero | **exit 1** — `dial tcp 127.0.0.1:59999: connect: connection refused`; no green run |
| Wrong actual major (shell-suite gate) | `make test-database-launcher` (case `FAKE_GO_MAJOR=18`) | reject 18 | PASS — `major 18, but major 17 is required`, reused path does not trigger cleanup |
| Wrong actual major (real, un-gated) | `TRAGGO_TEST_POSTGRES_REQUIRED=1 TRAGGO_TEST_POSTGRES_DSN="postgres://postgres@127.0.0.1:55432/postgres?sslmode=disable" go test -count=1 ./database/... ./test/...` against the real postgres:18 | note | **exit 0** — the real fixtures themselves PASS on major 18. This is the exact reason a version gate is mandatory; the gate (which should reject 18) is broken by the WP01 dotfile defect and cannot currently be exercised against a real server. |
| Credential sentinels, URL / quoted keyword, success/test-failure/setup-failure/child echo | `make test-database-launcher` cases 3–6, 10–12, 15 | zero sentinels in captured stdout/stderr | PASS — sentinel `s3nt1n3l-pw-9x2` and full credential DSNs never appear in any captured output across success, malformed/setup failure, and test failure, including controlled child echoes |
| Launcher reuse failure | `make test-database-launcher` cases 4, 8, 10 (reused DSN) | failure exits preserved, no leak | PASS — reused-DSN test failure preserves exit (7), wrong-major rejected, no cleanup of external server |

**Conclusion**: negative checks prove that missing/unusable PostgreSQL
configuration cannot yield a green dedicated run, and credential sentinels never
appear in launcher output. No green dedicated run was produced without real
PostgreSQL execution (the only green direct run against a real server used the
wrong major 18 and is explicitly a negative, not acceptance, record). No raw
secret-bearing capture is stored.

---

## T024 — Complete full pipeline and freeze the evidence revision

| Command | Exit | Result |
| --- | --- | --- |
| `go test -count=1 ./hack/...` (WP04 checker) | 0 | `ok github.com/traggo/server/hack` |
| `make lint` | 0 | `go vet ./...` PASS; `goimports -l` **0 findings** (the long filename list in the make log is the command echo, not findings — verified separately, exit 0, empty output); `yarn format:check` PASS; `yarn lint:check` (tslint) PASS |
| `make test` | 0 | backend `go test --race -coverprofile=coverage.txt -covermode=atomic ./...` all packages `ok` (harmless cgo linker `LC_DYSYMTAB` warnings on darwin/arm64; not errors). Representative coverage: dashboard 85.4%, database 100%, statistics 93.1%, graphql 100%, server 100%, device 96.6%, model 91.5%, test 76.9%. `coverage.txt` generated (361 KB, gitignored). Frontend `(cd ui && CI=true yarn test)` → `Test Suites: 1 passed, 1 total; Tests: 2 passed, 2 total` PASS. |

**Conclusion**: `make lint` and `make test` both exit 0 with no unresolved
formatting findings; backend race + coverage and frontend outcomes are recorded.

---

## Requirement mapping and status

| Requirement | Status | Evidence |
| --- | --- | --- |
| FR-001 (loopback publish / cleanup) | **PASS (launcher regressions + partial real observation)** | `make test-database-launcher` 16/16; `make test-postgres` failure run showed loopback-publish intent and cleanup. Real PG17 HostIp inspection **not** executed (blocked). |
| FR-002 (credential-safe output) | **PASS** | launcher shell suite cases 3–6, 10–12, 15; zero sentinels/DSNs in captured output. |
| FR-003 (fractional ranges) | **PASS (SQLite) / NOT verified on PG17** | SQLite fractional regressions pass; PG17 shared fixtures blocked (image unavailable). |
| FR-004 (migration error propagation) | **PASS (SQLite) / NOT verified on PG17** | `TestNew_closesOwnedPoolOnMigrationError`, `TestMigrationFailure_sqlite`, etc. pass; real PG17 migration failure blocked. |
| FR-005 (regressions distinguish defects) | **PASS** | discrimination evidence in T021 and T023 for all four defects. |
| FR-006 (PG17 + full pipeline) | **PARTIAL / OPEN** | full lint/test PASS; required PG17 gate with zero skips **NOT executed** (image unavailable + WP01 launcher defect). |
| FR-007 (reconcile reports) | **Not owned here** | report addenda are owned by the reporting WPs; this document is the shared evidence they cite. |
| NFR-001 (no broad publish, zero sentinels) | **PASS (launcher regressions)** | 16/16 shell cases; real HostIp inspection blocked. |
| NFR-002 (zero PG skips, required exits 0) | **OPEN** | PG17 gate could not run; no green PG17 run produced. |
| NFR-003 (revision-bound evidence w/ exits) | **PASS** | this document. |
| C-003 (synthetic/disposable only) | **PASS** | only disposable/test fixtures used; no production access. |
| C-004 (fresh vs historical evidence) | **PASS (honest)** | only freshly executed evidence recorded; PG17 explicitly not claimed. |

## Open gaps (must be resolved before acceptance of FR-006/NFR-002)

1. **Actual PostgreSQL 17 execution** — needs a `postgres:17` server image (or
   DSN) reachable from this environment. Not obtainable here (Docker Hub TLS
   block; mirror auth blocked by policy).
2. **WP01 launcher version-probe defect** — `hack/test-databases.sh:191`
   dotfile helper `.pgversion-$$.go` is ignored by the Go tool; `go run` fails.
   Blocks real version gating on both launcher paths. **Routed to WP01.**
3. Once (1) and (2) are resolved, re-run `make test-postgres` on real PG17,
   inspect live published HostIp (all 127.0.0.1), prove container cleanup after
   success and induced failure, and confirm zero skipped PG cases.

No fabricated or historical evidence is presented as a fresh PG17 result.