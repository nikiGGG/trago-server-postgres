# Mission Review Report: PostgreSQL dashboard review fixes

**Reviewed-HEAD verdict: FAIL (historical).** The credential-redaction findings below concern the original reviewed revision. Code fixes are now prepared, with tool-dependent validation pending as described in the implementation follow-up. The user has accepted the existing PostgreSQL 18 verification in place of a separate PostgreSQL 17 run; missing PG17 execution is no longer a blocker.

**Implementation follow-up (2026-10-28):** the user subsequently authorized code fixes and requested execution of tool-dependent checks on another device. RISK-1/2 filtering, RISK-3 checker coverage, and TEST-1's rounding fixture have now been changed in the working tree, with regression coverage. Fresh launcher checks pass 27/27 here. Go/real-engine/full-pipeline verification of these changes is pending on the other device; the original reviewed-HEAD findings and verdict below remain historical evidence, not a claim about fresh execution of the modified Go code. See [verification handoff](../../docs/reviews/postgres-dashboard-followup-verification.md). The launcher now defaults to PostgreSQL 18 and accepts actual majors 17/18, implementing the user's version-substitution decision.

**Reviewer:** Codex (automated, this session)
**Date:** 2026-10-28
**Mission:** `postgres-dashboard-review-fixes-01M5ZJTF`
**Reviewed HEAD:** `ce09ee1a4ae149cdaedd19d73ce4db77110a98ab`
**Comparison baseline:** `6e4c888f7eca13cea44336b6f6a60b357bbd4710`
**WPs reviewed:** WP01–WP06, all `done`; integration merge `5b42fb1` into `plan/postgres-dashboard`.

## Scope and evidence

The latest merged mission is the review target; the parent mission already has its own report. Read this mission's spec, plan, tasks, remediation contract, WP review cycles, status events, acceptance matrix, shared execution evidence, and both reconciled parent reports. Compared merged implementation and regression files against the baseline cited by WP05. `meta.json` lacks `baseline_merge_commit`; the baseline above is explicitly reconstructed from WP05's evidence, not supplied by metadata. The merge's first parent already contains most implementation, so comparing only that parent would miss the substantive changes.

Relevant implementation is live: GraphQL calls statistics resolvers; Stats2 delegates to Stats; the time scalar serializes responses; application initialization calls `database.New`; CI calls the launcher and result checker. No new dead production module was identified. The integer range identity, RFC3339Nano serialization, and migration error propagation follow the plan. No schema/API-field expansion, ORM upgrade, destructive enum conversion, deployment, or production database activity was observed in the reviewed diff.

Fresh verification here is deliberately distinguished from WP05's historical execution. This environment has no `go`, `docker`, or `yarn` on PATH. Existing evidence records successful SQLite/full-pipeline execution at `38ae5e9e3818030b1da410a652fdb012877d2973`, before the subsequent launcher helper-name fix `659cd04`. It explicitly records missing PG17 acceptance. No fresh Go, SQLite, PostgreSQL, frontend, race, or coverage pass is claimed here.

## Operator decision — PostgreSQL version substitution (2026-10-28)

After receiving this review, the user instructed: “i want to skip проверка PostgreSQL 17 cause already checked with PostgreSQL 18.” This explicitly accepts the previously recorded PostgreSQL 18 verification instead of requiring an additional PostgreSQL 17 run for this mission. This user instruction supersedes the original version-specific review requirement; no additional approval or exception form is required.

**Disposition:** the PostgreSQL-17-only portion of Gate 3 and DRIFT-1 is **WAIVED BY USER / PG18 ACCEPTED**, and F-01 no longer requests a PG17 rerun. The original analysis below is retained as historical context under the original spec. Statements there that PG18 cannot satisfy the original requirement describe that earlier contract, not the current operator-approved acceptance decision. PostgreSQL 17 is not claimed to have been tested. No new database run occurred during this amendment. The later implementation follow-up updates the launcher to accept majors 17/18 and default to 18.

Remaining F-01 work is documentation reconciliation with this decision and any separately missing real launcher HostIp/custom-port/cleanup evidence. The version substitution alone does not establish those lifecycle observations. These retained evidence limitations are not being used to reintroduce the waived PG17-only blocker. RISK-1, RISK-2, RISK-3 and TEST-1 are independent of the PostgreSQL major version and remain open.

## Gate Results

The legacy skill's `spec-kitty` Python contract/architectural/cross-repository scenario paths concern Spec Kitty's own implementation. This repository is a Go application using Spec Kitty; those external suites are not application acceptance tests. Their applicability is recorded explicitly instead of treating nonexistent suites as successful checks.

### Gate 1 — Contract tests

- Command: `make test-database-launcher`.
- Exit: **0**; **16 cases, 0 failures, 0 skips**, fresh on reviewed HEAD.
- Command: `bash -n hack/test-databases.sh hack/test-databases_test.sh`.
- Exit: **0**.
- Result: existing shell suite **PASS**, security contract **FAIL** under additional synthetic reproductions (RISK-1). Go contract/regression execution is **UNVERIFIED** here because the toolchain is absent. Spec Kitty's `tests/contract/` is not applicable to this repository.

### Gate 2 — Architectural tests

- Command: `git diff --check`; exit **0**.
- Result: **UNVERIFIED** for fresh Go compilation/vet; this whitespace check establishes no architectural pass. Spec Kitty's `tests/architectural/` is not applicable. WP05 records historical `make lint` and `make test` exit 0, including race/coverage and frontend 2/2, at its stated source revision.

### Gate 3 — Integration / PostgreSQL 17

**Current disposition:** PG17-specific execution **WAIVED BY USER / PG18 ACCEPTED**, as recorded above. The command and failure below are the historical review attempt, not a continuing demand for PG17 execution.

- Command: `make test-postgres`.
- Exit: **2** from make; child failures report `go: command not found` and `docker: command not found` (127).
- Result: **FAIL / acceptance unverified**. No engine tests executed here. This local environmental failure is not itself a database code defect. Separately, WP05 expressly records that real PG17 execution, HostIp inspection, custom-port connectivity, and successful/failed real-container cleanup were not established. PostgreSQL 18 results cannot satisfy spec C-004 or FR-006.
- No `mission-exception.md` was found. The acceptance-matrix note about operator direction to use PG18 is not a revised specification or the schema-qualified exception described in the skill. No exception can establish that PG17 tests actually ran.
- Spec Kitty's external scenario suite is not applicable: this mission does not modify Spec Kitty cross-repository behavior.

### Gate 4 — Issue Matrix

- Inspection: no `issue-matrix.json` or legacy `issue-matrix.md` in this mission; **0 applicable GitHub issue rows**.
- Exit: not applicable (artifact applicability inspection).
- Result: **N/A**. The spec references parent review findings and Vikunja task 114, not a GitHub issue closure matrix. Follow-ups are recorded below; task 114 remains open. The existing acceptance matrix is assessed separately in DRIFT-1.

## FR Coverage Matrix

“Adequate” below describes assertion structure, not fresh execution or PG17 acceptance.

| FR | WP owner | Implementation and test evidence | Adequacy / finding |
| --- | --- | --- | --- |
| FR-001 loopback/lifecycle | 01, 04, 05 | `hack/test-databases.sh:269` publishes `127.0.0.1:${PORT}:5432`; CI reuses launcher; shell default/custom-port and cleanup assertions | PARTIAL: doubles pass; actual HostIp/connectivity/lifecycle evidence absent, DRIFT-1 |
| FR-002 credential-safe output | 01, 05 | `dsn_password`, `redact_output`, `run_suite`; shell sentinel cases | PARTIAL: supported DSNs expose passwords, RISK-1; probe stderr bypasses filter, RISK-2 |
| FR-003 fractional identity/totals | 02, 05 | `statistics/summary.go` range_id grouping and input-bound reconstruction; `summary_test.go:294` shared five-range independent oracle, duplicate/empty fixtures; `summary2_test.go:331`; `model/time_test.go`; HTTP GraphQL assertion `graphql/handler_test.go:61` | Core assertions ADEQUATE for identities, clipping and wire precision; PG17 unverified; rounding discrimination hole TEST-1 |
| FR-004 startup migration/cleanup | 03, 05 | `database/database.go:60`; real SQLite/PG schema-conflict tests in `migration_external_test.go`; owned-pool Ping/cleanup/cause tests in `migration_test.go`; startup fatal subprocess in `main_test.go:29` | ADEQUATE test design: public New conflicts distinguish post-connect migration failure; hooks separately prove owned-pool closure; PG17 execution unverified |
| FR-005 constraining regressions | 01–05 | Launcher doubles, shared real-engine fixtures, scalar/wire tests, migration conflict/cleanup/startup tests, checker tests | PARTIAL: RISK-1/2, TEST-1 and RISK-3 expose gaps |
| FR-006 executed PG17/full pipeline | 04, 05 | Makefile target, workflow postgres-test job, checker, WP05 common evidence | MISSING required successful PG17 execution; historical full pipeline pass only; DRIFT-1 and RISK-3 |
| FR-007 consistent reconciliation/provenance | 06 | Both report addenda and parent acceptance artifacts cite WP05; remediation acceptance matrix contradicts open PG17 status | PARTIAL: DRIFT-1; actor labels do not prove independent human review |

NFR-001 fails the credential boundary (RISK-1); NFR-002 remains unestablished on PG17; NFR-003 is weakened by contradictory acceptance claims. C-001/002 are supported by the design and test structure, with execution limits above. C-003 operational boundaries were preserved. C-004 is violated by treating PG18 as passing PG17 acceptance.

## Drift Findings

### DRIFT-1 — Required PG17 verification is absent but acceptance declares pass

**Type:** NFR-MISS / acceptance drift
**Severity:** HIGH, blocking
**References:** FR-006/007, NFR-002/003, C-004, SC-002/004/005.

`acceptance-matrix.json:58–65` marks FR-006 `pass`, cites PostgreSQL 18, and drops “17” from its requirement description. Lines 90–98 likewise mark NFR-002 confirmed while explicitly acknowledging the required major is still open. The top-level verdict is `pass`. In contrast, WP05's evidence sections T022 and Open gaps expressly state PG17 execution and real lifecycle checks are missing. Both parent report addenda still leave that gate OPEN. An acceptance note saying “Per operator direction” is not evidence of execution on the required engine; the unchanged spec explicitly rejects PG18 as a substitute.

WP05 also predates the launcher helper rename; its evidence still describes that now-fixed defect as open. Current code correctly uses `pgversion_$$.go`, but no retained positive PG17 run after that fix closes the execution gap. WP05 review cycle 1 approved truthful reporting of an open gap, not fulfillment of that gap. All WPs becoming `done` does not change the requirement.

**Follow-up F-01:** retain a successful revision-bound PG17 `make test-postgres` run with zero PG skips, actual server version, real loopback HostIp/custom-port checks, success/failure cleanup, and required negatives after the final launcher changes. Reconcile the remediation acceptance matrix and both historical addenda against that same evidence. Until then, preserve the gap as unresolved.

## Risk Findings

### RISK-1 — Valid lib/pq passwords survive the launcher redaction

**Type:** BOUNDARY-CONDITION / credential disclosure
**Severity:** HIGH, blocking
**Location:** `hack/test-databases.sh:118–167`, `:174–183`, `:243`.
**References:** FR-002/005, NFR-001.

The custom password extractor handles SQL-style doubled quotes, whereas lib/pq keyword values use backslash escapes. It also extracts URL passwords only from userinfo and ignores `?password=...`. Both are accepted connection-string forms in the pinned driver: see [lib/pq v1.0.0 parseOpts](https://github.com/lib/pq/blob/v1.0.0/conn.go#L418-L487) and [ParseURL query handling](https://github.com/lib/pq/blob/v1.0.0/url.go#L65-L68).

Fresh full-launcher reproductions used an isolated fake `go`: `run` prints `major=17`, and `test` prints the synthetic plaintext password. No actual server or credentials were used. For both inputs below the launcher returned **0** and emitted the complete secret on stdout:

| DSN | Child-emitted secret still visible |
| --- | --- |
| `host=localhost user=test password='synthetic\'quoted-secret'` | `synthetic'quoted-secret` |
| `postgres://test@127.0.0.1/db?password=synthetic-query-secret` | `synthetic-query-secret` |

Full-DSN replacement cannot hide a decoded password emitted alone. Existing doubles exercise the accepted subset of the custom extractor and miss these cases. This demonstrates that the stated child-output security boundary fails; it does not claim every real Go test prints a password.

**Follow-up F-02:** use driver-compatible connection parsing, cover these valid DSNs and JSON-escaped child output, and require all relevant output paths to keep sentinel passwords absent.

### RISK-2 — Version-probe failure output bypasses credential filtering

**Type:** ERROR-PATH / credential disclosure boundary
**Severity:** MEDIUM
**Location:** `hack/test-databases.sh:208–224`.

`check_server_version` captures stdout only. Its Go helper prints raw driver errors to stderr before `run_suite` creates the redacted captures. A synthetic probe that emits `probe failure: synthetic-probe-secret` to stderr and exits 1 causes the launcher to repeat that line unchanged, followed by its generic error message. The ordinary shell setup-failure double prints only a fixed “connection refused”, so it cannot detect credential-bearing setup diagnostics.

This is a demonstrated filtering bypass, not proof that a normal authentication failure in lib/pq reveals the password. The pinned driver's malformed-URL/option-error paths can include supplied values; the spec expressly includes malformed/setup failures in the redaction boundary.

**Follow-up F-03:** capture and sanitize version-probe diagnostics before replay and add a setup-failure sentinel assertion.

### RISK-3 — New Stats2 PostgreSQL suites can skip without failing the checker

**Type:** CROSS-WP-INTEGRATION / verification gap
**Severity:** MEDIUM
**Location:** `hack/check-postgres-results.go:65–88`, `:254–361`; `statistics/summary2_test.go:340–360`.

The manifest requires Stats2 daily suites but omits `TestStats2_single_fractional_postgres` and `TestStats2_single_fractional_postgres_sessionTimeZone`, both introduced by WP02 for this mission. Validation rejects skips only for required roots and their observed descendants. A skip for either omitted top-level test is outside those loops; a package-level pass plus all manifest roots/children passing still passes validation. Thus “zero skipped PostgreSQL cases” is not enforced for the new Stats2 cases. Current DSN-required behavior normally prevents these particular skips, but the checker cannot detect future/conditional skip regressions in these tests.

This finding is established by the manifest and validation control flow, not by executing a compiled checker here (Go is unavailable).

**Follow-up F-04:** include both Stats2 fractional roots and reject skips of every classified live PostgreSQL case; exercise a stream with all mandatory suites passing and one omitted top-level PG suite skipped.

## Test Adequacy Findings

### TEST-1 — The rounding fixture cannot distinguish sum-of-rounded from rounded-sum

**Severity:** MEDIUM
**Location:** `statistics/summary_test.go:331–344`.
**References:** FR-005, C-001, plan's independent per-span rounding oracle.

The `fractional sum-of-rounded vs rounded-sum` fixture seeds two 0.6-second spans but tags one `review` and the other `work`. SQL groups by tag value, leaving one span in each aggregate. Both `SUM(ROUND(duration))` and `ROUND(SUM(duration))` return 1 in each group; the fixture would pass if aggregation rounding were moved outside SUM. Production currently uses the correct `sum(round(...))`, so this is a missing regression constraint, not an observed wrong result.

**Follow-up F-05:** put both spans in the same key/value group and assert 2 seconds; demonstrate the alternative expression returns 1 and fails the assertion on both engines.

## Silent Failure Candidates

| Location | Condition | Result / assessment |
| --- | --- | --- |
| `statistics/summary.go`, attachAggregates | Scanned range ID outside the request slice | Silently continues; SQL currently generates trusted IDs, so not a demonstrated normal-path defect; optional hardening |
| `hack/test-databases.sh:99–102` | `docker rm -f` fails | Error suppressed; orphan container possible; actual lifecycle inspection remains required |
| `database/migration_external_test.go`, dropDisposablePostgres | Maintenance open/drop fails | Test cleanup ignores errors; disposable DB leakage can go unnoticed |
| `database/database.go:60–70` | Migration fails and pool Close also fails | Migration remains returned cause; cleanup failure logged at debug; intentionally preserves the required primary error |

The original silent AutoMigrate success path is removed. Query/scan errors in Stats are now returned. No new normal production silent-success migration path was found.

## Security Notes

| Finding | Risk class | Recommendation |
| --- | --- | --- |
| RISK-1 | Credential disclosure | Driver-compatible parsing and sentinel coverage before accepting FR-002 |
| RISK-2 | Unfiltered setup diagnostics | Capture/sanitize the probe's stderr |
| Version probe passes DSN through `go run ... "$dsn"` at line 223; Perl helpers also receive it in argv | Local process-argument exposure | Prefer environment/private input as planned; stdout/stderr contract alone does not protect process inspection |
| SQL range IDs are trusted loop integers; range bounds/tags/user remain parameters | No new SQL injection found | Preserve parameter binding |
| Published port explicitly uses IPv4 loopback; no CI service container remains | Correct static boundary | Complete real HostIp/lifecycle verification |

## Review History and Process

WP01 cycle 2 rejected the dot-prefixed helper; cycle 3 approved the non-dotfile fix and filename guard. That fix is present in merged HEAD. WP02/03 reviews explicitly recorded optional PG skips. WP05/06 approved evidence and reconciliation while retaining PG17 as open. Final status events on 2026-10-28 force all six packages from approved to done with `auto-approval:WPxx:20261028` evidence. These transitions establish completion status, not missing engine execution.

Review records label approvals as `user`, with tool metadata `Nikita Gryaznev`, alongside automated qwen profile claims. Those records establish recorded actor labels; they do not independently prove a human or separate reviewer performed the analysis. Parent acceptance prose is more categorical about human independence than the event evidence warrants.

## Final Verdict

**FAIL.** RISK-1 remains a blocking violation of the credential-safety requirement. Missing PostgreSQL 17 execution is no longer a blocker: the user explicitly accepted existing PostgreSQL 18 verification instead. RISK-2, RISK-3 and TEST-1 remain concrete output-filtering and regression-coverage gaps independent of that version substitution. The core fractional-identity and migration-error changes are credible from source and test inspection, subject to the recorded execution limits.

Follow-ups F-01–F-05 above are recorded locally for remediation; no external issue or message was created. Existing parent F-5 general hardening and F-6 operational delivery remain separate, and Vikunja task 114 remains open. No implementation or test changes were made during this review.

## Retrospective Reminder

The canonical order is mission review → author/verify retrospective → surface findings. No retrospective existed for this mission at review start. Ran `spec-kitty retrospect create --mission postgres-dashboard-review-fixes-01M5ZJTF --json` (exit 0), which created [retrospective.yaml](retrospective.yaml) and appended its event to `status.events.jsonl`. The installed runtime writes the record in the mission directory, rather than the legacy skill's `.kittify/missions/<id>/` location. This was an automated explicit-create invocation; the generated `created_by.kind: human` is CLI attribution, not evidence of human authorship.

`spec-kitty retrospect summary` exited 0 and found two completed mission records. `spec-kitty agent retrospect synthesize --mission postgres-dashboard-review-fixes-01M5ZJTF` exited 0 in dry-run mode: no planned applications or applied proposals. No `--apply` was used.

The generated retrospective incorrectly says WP01 completed without rejection cycles despite the explicit cycle-2 rejection; it separately recognizes two implementation cycles. Treat the original review-cycle files and events as the authority. The automatic retrospective is not an independent acceptance verdict and does not override this report.
