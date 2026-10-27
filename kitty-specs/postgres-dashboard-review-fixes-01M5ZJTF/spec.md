# PostgreSQL dashboard review remediation

**Mission**: `postgres-dashboard-review-fixes-01M5ZJTF`
**Created**: 2026-10-28 (Europe/Samara)
**Status**: Specified; implementation and verification pending
**Target branch**: `plan/postgres-dashboard`
**Parent mission**: `postgres-dashboard-01M5JRH2`

**Input**: “Fix the postgres-dashboard mission review findings: bind the disposable PostgreSQL port to loopback, remove DSN credential logging, preserve fractional range boundaries, and propagate startup AutoMigrate errors. Add regression coverage and verify SQLite, PostgreSQL 17 with no skips, and the full lint/test pipeline. Reconcile both review reports using the resulting evidence”.

## Goal and scope

Close the four defects and verification gap in the parent mission reviews. Deliver safe disposable PostgreSQL execution, correct fractional-range statistics, observable startup migration failures, and evidence sufficient to reconcile both reports. This specification defines the remediation; it does not claim fixes or verification have run.

Source reviews, both inspecting revision `69fc2ceb650e7c5379f5cba192f70f29e8112ad8`:

- `docs/reviews/postgres-dashboard-mission-review-2026-10-27.md` (Codex): FAIL; RISK-1 network exposure, RISK-2 credential disclosure, DRIFT-1 fractional boundary loss, GAP-1 missing PostgreSQL 17/full-pipeline execution; also identifies swallowed startup migration errors.
- `kitty-specs/postgres-dashboard-01M5JRH2/mission-review.md` (Nessy CLI): PASS WITH NOTES; DRIFT-1/RISK-1 startup migration errors, RISK-2 stale harness comment, reviewer-attribution hygiene. Its recorded local PostgreSQL cases skipped without a DSN; those runs do not establish PostgreSQL 17 acceptance.

Finding IDs are local to each report. Reconciliation must identify both source and ID.

## User scenarios and testing

### Story 1 — Run database checks safely (P1)

A developer runs the disposable launcher or reuses a test server without exposing unauthenticated database access or writing credentials to captured output.

**Independent test**: Exercise setup, reused-DSN execution, failure, and cleanup paths with synthetic credentials; inspect publish arguments and stdout/stderr. Confirm actual published host addresses in an integration run.

1. Without a supplied DSN, the disposable database publishes on IPv4 loopback only, including with a custom host port. The documented command connects and cleans up its container.
2. With either a URL DSN or lib/pq keyword DSN containing a sentinel password, successful and failed runs never print that password or the credential-bearing DSN. Useful stage/failure messages remain available.
3. The CI disposable database uses the same loopback publishing boundary. Missing or unusable configuration fails required PostgreSQL verification.

### Story 2 — Read fractional-range statistics (P1)

A dashboard user requests valid fractional ranges and receives each original range and its correct aggregate, rather than empty entries or merged identities.

**Independent test**: Run shared fixtures on SQLite and PostgreSQL 17, asserting original response bounds and independently calculated per-span rounded seconds through both existing statistics paths where they accept ranges.

1. A completed tagged span `10:00:00Z–10:00:10Z`, requested as `10:00:00.200Z–10:00:01Z`, returns one second and preserves the requested bounds.
2. Fractional ends and ranges with both bounds fractional preserve their identities and clipped, per-span rounded durations.
3. Two distinct ranges whose starts and ends fall within the same respective whole seconds remain separate, including when their expected aggregates differ.
4. Empty fractional ranges retain existing zero-fill behavior. Integer-bound, local-user-time, non-UTC PostgreSQL session-timezone, filtering, grouping, and user-isolation regressions continue passing.

### Story 3 — Reject failed startup migrations (P1)

An operator receives a startup error when migration fails, rather than an apparently initialized application using an incompatible schema.

**Independent test**: Arrange a deterministic real migration failure on each supported engine, call production `database.New`, and assert its returned error. Exercise the startup caller's error path and verify failed initialization releases its connection pool.

1. Successful connection followed by AutoMigrate failure returns a non-nil error and no usable initialized handle; application startup does not proceed as successful.
2. Cleanup closes resources opened during failed initialization. The migration failure remains the reported cause if cleanup also fails.
3. Healthy fresh initialization and reconnect succeed on both engines. PostgreSQL text and legacy bytea enum schemas remain readable and writable without destructive conversion.

### Story 4 — Assess one evidence-backed outcome (P2)

A reviewer follows both reports to the remediation revision and determines resolved findings and remaining external work.

**Independent test**: Follow each reconciliation entry to a regression and execution record identifying revision, command, environment, outcome, and skips.

1. Fresh SQLite, actual PostgreSQL major version 17, and complete lint/test runs succeed on the remediation source revision. Dedicated PostgreSQL verification has zero skipped PostgreSQL cases, including new migration-error regressions.
2. Both reports gain dated reconciliation addenda referring to the same evidence and current verdict. Historical commands, outcomes, and original verdicts remain identifiable.
3. Missing tools, failed commands, unavailable services, and skipped tests remain unresolved verification; configuration and past approvals never substitute for execution.

### Edge cases

- URL/keyword DSNs, quoted passwords, setup/test failures, and custom host ports.
- Fractional starts, ends, both bounds, distinct subsecond identities, and empty ranges. Preserve accepted timestamp precision in response identity, including precision finer than PostgreSQL timestamp storage; do not silently redefine accepted input precision.
- Non-UTC sessions and user offsets. Use independent numeric expectations away from ambiguous floating-point rounding thresholds.
- Database open succeeds but migration fails; cleanup fails; healthy text/bytea reopen.
- Unsorted ranges and additional query/scan-error hardening remain explicit non-blocking follow-ups unless required for the scoped fixes.

## Requirements

### Functional requirements

| ID | Requirement | Priority | Status |
| --- | --- | --- | --- |
| FR-001 | Restrict launcher and CI disposable PostgreSQL published ports to loopback, retaining working connectivity, custom-port support, and cleanup. | P1 | Open |
| FR-002 | Remove credential-bearing DSN logging on supported launcher paths, including launcher-controlled failure output; cover URL and keyword DSNs. | P1 | Open |
| FR-003 | Preserve original fractional range identities and correct clipped, per-span rounded totals on SQLite and PostgreSQL through the existing statistics API. Distinct ranges cannot merge or disappear. | P1 | Open |
| FR-004 | Return production AutoMigrate failures to startup callers, prevent successful startup after failed migration, and release resources opened by failed initialization. | P1 | Open |
| FR-005 | Add behavior-constraining regressions for FR-001–004, including shared real-engine fractional fixtures and real SQLite/PostgreSQL 17 migration failures. Tests must distinguish fixed from defective behavior. | P1 | Open |
| FR-006 | Execute and retain fresh SQLite, required PostgreSQL 17, and full lint/test evidence after required generation/UI preparation. Missing/unusable PostgreSQL configuration must fail its dedicated gate. | P1 | Open |
| FR-007 | Reconcile both reports and affected acceptance evidence against the same remediation evidence; correct stale migration/order claims and inaccurate reviewer attribution using actual event records. | P2 | Open |

### Non-functional requirements

| ID | Requirement / measurable threshold | Category | Priority | Status |
| --- | --- | --- | --- | --- |
| NFR-001 | Zero non-loopback published database addresses; zero sentinel-password occurrences in captured launcher stdout/stderr on tested success/failure paths. | Security | P1 | Open |
| NFR-002 | Zero skipped PostgreSQL cases in required PostgreSQL 17 verification. Every required command exits 0; formatting findings must also be resolved even when the tool exits 0. | Verification | P1 | Open |
| NFR-003 | Every scoped review finding has a status and revision-bound evidence reference; command records include exit status and applicable test/skip summaries. | Auditability | P2 | Open |

### Constraints

| ID | Constraint | Category | Priority | Status |
| --- | --- | --- | --- | --- |
| C-001 | Preserve GraphQL/UI shape, integer-bound behavior, filtering, local-time semantics, per-span rounding, zero fill, and session-timezone correctness. | Compatibility | P1 | Open |
| C-002 | Preserve SQLite and PostgreSQL text/bytea persistence; no destructive column conversion, ORM upgrade, or new migration framework. | Data integrity | P1 | Open |
| C-003 | Use synthetic data and disposable databases. No production DB access, deployment, image publishing, external manifest changes, or closing Vikunja task 114. | Scope | P1 | Open |
| C-004 | Distinguish historical from fresh evidence. PostgreSQL 14/18 runs or configured `postgres:17` images alone cannot establish PostgreSQL 17 acceptance. | Evidence | P1 | Open |

### Key concepts

- **Range identity**: original start/end timestamps defining one response range, independent of formatting and database timestamp precision.
- **Startup migration failure**: failure after connection opens but before successful schema initialization; this is a startup error.
- **Reconciliation evidence**: revision-bound commands, results, regressions, and statuses explaining each historical finding after remediation.

## Verification and reconciliation contract

Retain an evidence artifact with tested commit, any working-tree patch identity, tool versions, actual PostgreSQL `server_version`, commands, exit codes, and test/skip counts. Evidence must contain no secrets. A later documentation-only commit may explicitly reference the tested code revision instead of claiming its own revision was tested.

Required checks include `make test-postgres` on PostgreSQL 17 in required mode, database startup regressions included in mandatory PostgreSQL execution, and complete `make lint` / `make test` after repository-required installation, generation, and UI build preparation. Record full-pipeline backend race/coverage and frontend results. Negative checks must demonstrate that missing/unusable PostgreSQL configuration cannot yield a green dedicated run. Complete reproducible local evidence is acceptable; distinguish local execution from remote CI and link any remote run actually used.

| Source | Finding | Required closure evidence |
| --- | --- | --- |
| Codex | RISK-1 | Loopback regression, actual server binding, working launcher/cleanup, CI mapping assessment/fix. |
| Codex | RISK-2 | URL/keyword sentinel passwords absent from successful/failed captured output. |
| Codex | DRIFT-1 | Fractional start/end/both-bound and distinct-range fixtures with correct bounds/seconds on both engines. |
| Codex | GAP-1 / F-4 | Fresh PostgreSQL 17 with no skips and full lint/test results on the remediation revision. |
| Nessy | DRIFT-1 / RISK-1 | Production migration error/cleanup regressions; healthy reopen and text/bytea results. Also closes Codex's startup silent-failure row. |
| Nessy | RISK-2 / attribution note | Correct harness explanation and evidence attribution grounded in recorded events, retaining provenance. |
| Both | External items; Codex F-5/F-6 | Explicit separate follow-ups for hardening/operations; no unsupported deployment or task-completion claim. |

Each addendum must map source-local findings to fixed, still open, or explicitly deferred status, cite shared evidence, and explain the resulting scoped verdict. Missing verification keeps the relevant gap open. Preserve historical reports rather than silently replacing their past observations.

## Success criteria

- **SC-001**: All four defects have regressions distinguishing fixed behavior from reviewed behavior; all pass.
- **SC-002**: SQLite and actual PostgreSQL 17 return expected fractional results; required PostgreSQL cases execute with zero skips.
- **SC-003**: Migration failures surface and release resources; healthy initialization/persistence cases pass on both engines.
- **SC-004**: Full lint/test validation completes with no unresolved failures or formatting findings and retained revision-bound evidence.
- **SC-005**: Both reports and affected acceptance evidence agree on scoped remediation status while preserving historical outcomes and external follow-ups.

## Non-goals and decisions

No UI redesign, new GraphQL fields, MySQL support expansion, production migration, live rollout, or general range-ordering redesign. No unresolved product decision blocks planning. SQL representation, regression harness mechanics, and evidence layout belong to plan/tasks. Environment failures must not weaken acceptance criteria. This phase produces the specification; implementation, actual validation, and report reconciliation occur in subsequent mission phases.
