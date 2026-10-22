# Implementation plan: PostgreSQL dashboards and SQLite compatibility

Status: implementation plan authored; implementation and rollout have not started. Source of requirements: `spec.md`, Vikunja task 114. Source investigation: `research.md`.

## Implementation agent

The user selected Qwen CLI to implement this mission. Register both `codex` and `qwen` in the project agent configuration; Codex prepared the specification and plan. Use `qwen` as the agent identity for later Spec Kitty implementation commands. Finalize tasks before starting implementation; do not launch implementation during this planning session.

## Technical Context

Go module `github.com/traggo/server`, Go 1.24 language baseline, CI Go 1.25; GORM v1.9.2, lib/pq, go-sqlite3; GraphQL resolvers and existing React dashboard. Target PostgreSQL 17/CNPG and existing SQLite. Preserve the GraphQL schema, UI behavior, config defaults, local user-time calculations, and database data. No production schema migration is planned.

## Architecture and data flow

Dashboard UI → existing `stats` or `stats2` → range creation (Stats2 only) → shared Stats query → dialect-specific duration expression → scan/normalize results → existing range and tag zero-fill → existing tables/charts.

1. In `statistics/summary.go`, choose fixed fragments from the active dialect. PostgreSQL CTE bounds must be explicitly cast to timestamp types compatible with the observed user-time columns, avoiding VALUES parameters resolving as text. Keep values parameterized and retain existing SQLite binding behavior.
2. Use `dates CROSS JOIN time_span_tags AS tst JOIN (...) AS ts ON ts.id = tst.time_span_id` or an equivalent explicitly joined overlap query. Qualify tag keys, user ownership and correlations (`tstx.time_span_id = tsx.id`). Group by both query bounds, tag key and value. Maintain Include OR/Exclude OR without joining filter rows into aggregate rows.
3. Compute each span's clipped local user-time duration using CASE bounds. PostgreSQL uses rounded EXTRACT(EPOCH FROM interval); SQLite retains rounded julianday arithmetic. Do not alter timestamp storage, counting of active timers, per-span rounding, or existing range endpoint conventions.
4. Capture the returned `Raw(...).Scan(...)` handle and check `.Error` before grouping/zero filling. Test a deliberate SQL execution failure, ensuring an actual error reaches GraphQL. Check timestamp decoding with real driver results: prefer typed bound scanning and explicit normalization if necessary, retaining API timestamps. Do not depend on PostgreSQL string formatting or session TimeZone.
5. Make DashboardType, Interval and DeviceType Scanners accept string and byte slice consistently, retaining Valuer strings. Return descriptive errors for unsupported types, including nil unless schema evidence establishes a different existing contract. Verify both normal text and legacy bytea read/write paths before accepting this approach. Do not change column types automatically.

## Implementation sequence

These are sequencing units for a later `/spec-kitty.tasks` pass, not finalized work packages.

| Unit | Scope | Depends on | Requirements / exit evidence |
| --- | --- | --- | --- |
| 1 | Add disposable PostgreSQL test harness and shared fixtures; characterize current SQLite outputs and reproduce PostgreSQL failure | — | FR-007; failing PostgreSQL reproduction, unchanged SQLite expectations |
| 2 | Fix enum loading; verify dashboard/device persistence on SQLite, PostgreSQL text, PostgreSQL bytea | 1 | FR-005/006; round-trip tests including reconnect |
| 3 | Add SQL dialect fragments, explicit joins/grouping/types, error propagation and necessary result normalization | 1 | FR-001–004; shared Stats fixtures pass on both engines |
| 4 | Exercise Stats2 daily/week selection and dashboard GraphQL lifecycle, add mandatory PostgreSQL 17 CI job | 2, 3 | FR-001–007; CI and API acceptance evidence |
| 5 | Write build/deployment runbook, produce candidate image and validate staging in the later execution phase | 4 | FR-008; image digest, rollout/rollback checklist, verified manifest diff |

## Test strategy

Refactor shared fixtures to run against SQLite and an explicitly configured disposable PostgreSQL test instance. Keep `test.InMemoryDB` behavior for unrelated tests. PostgreSQL tests must use isolated schemas or databases, fail on setup/seed errors, clean up connections, and prevent fixed-ID collisions; avoid parallel execution sharing a schema. A local run may explicitly skip absent PostgreSQL configuration, but CI's PostgreSQL job must require it and fail if no PostgreSQL tests execute. Use PostgreSQL 17 service health checks and CGO-enabled SQLite; generate existing GraphQL sources before Go tests.

Run identical expectation fixtures rather than treating either engine as the sole oracle:

- Existing duration, clipping, zero-fill, multiple tag/value and Include/Exclude cases; combined filters, exclusion precedence, nonmatching filters, multiple matching filter rows and user isolation.
- Deterministic weekly Daily fixture with `работа=дом`, `работа=офис`, `работа=подработка`. Include normal days, an empty day and a midnight-crossing span; publish expected per-day/per-series seconds and independent clipped totals.
- Cross-midnight example: a completed span 23:30:00–00:30:00 split by half-open test ranges at midnight produces 1,800 seconds per range. Separately exercise Stats2's current second-inclusive range builder and record its existing boundary outcome without silently changing it.
- Empty range list, reversed/equal bounds, empty keys, no matches, active spans, fractional-second rounding, exact day boundaries, equal starts with different ends, and ordered/unsorted ranges. Establish existing SQLite behavior first; avoid unrelated semantics changes.
- Stats2 with fixed `now`, Daily interval, selected week, user week-start settings and non-UTC input offsets. Compare response boundaries and seconds on both databases; check PostgreSQL session TimeZone independence.
- Scanners: string and byte slice for all enum types, unsupported types and nil; Valuer round trips. Dashboard create/update/reopen with chart types, intervals, named ranges, filters and layout fields; persisted device read. Run PostgreSQL variants on text and legacy bytea columns and SQLite on disk for reconnect/restart checks.
- Force query execution failure after successful fixture setup; expect non-nil error and no fabricated statistics. Assert fixture write errors so failed inserts cannot masquerade as successful empty results.

Validation commands during implementation: generate GraphQL using the repository tooling, run targeted model/statistics/dashboard/device suites on both databases, then the required `make lint` and `make test` pipeline with generated UI assets. This planning session does not claim these tests passed.

## Data compatibility and rollout

Do not convert production bytea columns back to text as part of this fix. First prove updates and reopens work against the deployed workaround and a clean text schema. If bytea writes fail, resolve binding compatibility explicitly before release; scanner acceptance alone is insufficient. Avoid destructive AutoMigrate effects by testing schema startup on both PostgreSQL shapes and checking startup migration errors.

Before building, identify the intended source baseline relative to task 114's deployed commit. Build an immutable image from the accepted revision using current repository release tooling, record commit and image digest, and keep the existing image digest for rollback. Document `TRAGGO_DATABASE_DIALECT=postgres` and the existing connection configuration without exposing credentials; SQLite remains the default.

In the later deployment phase, locate the external automation repository and inspect `automation-scripts/k8s-cluster/traggo.yaml` and workaround commit `65b29d3`. Prepare an image-only manifest change unless verified evidence requires more. Validate a staging environment against PostgreSQL 17 and a disposable SQLite deployment, then roll out the accepted digest. Check table/StackedBarChart Daily data for a selected week, Cyrillic series, Include/Exclude, create/edit/reopen, and persistence after application restart. Observe application/DB logs for SQL and Scan errors. Compare representative source-span totals without logging private data. On failure revert the image/manifest; schema compatibility must permit rollback without data restoration. Follow existing CNPG backup procedures before any later schema operation.

## Risks, assumptions and decisions

| Risk | Mitigation / decision |
| --- | --- |
| PostgreSQL VALUES parameters infer text; timestamp result shapes differ | Explicit type fragments and tests using the actual drivers; normalize only at the database boundary |
| Session timezone changes totals or returned ranges | Preserve local wall-clock meaning; verify multiple PostgreSQL session timezones |
| bytea workaround is readable but not writable | Full dashboard/device mutation and reconnect tests on both PostgreSQL schema variants |
| PostgreSQL fixed-ID seeds disturb sequences or fail silently | Check inserts and sequence allocation; isolated fixtures and cleanup |
| Range builder uses second-inclusive ends | Characterize existing behavior and preserve it; defer any independent precision/endpoint redesign |
| Broad range ordering or colon-tag bugs surface | Fix only if a required cross-database acceptance case fails; document any follow-up separately |
| Source baseline differs from deployment | Reconcile versions before building and record the image's exact revision |

## Completion criteria and handoff

Map all FR-001–008 to test/runbook evidence. Require passing SQLite and PostgreSQL CI, schema compatibility evidence, the task's dashboard acceptance checks, and a concrete rollout/rollback record before declaring task 114 complete. Next workflow step is `/spec-kitty.tasks` to author bounded work packages from this plan. Planning completion does not mark the Vikunja task done or authorize a production rollout.

## Spec Kitty environment status

Project agents: Codex and Qwen. The specification is substantive and committed locally on `plan/postgres-dashboard`. The installed CLI changed from 3.2.5 to 3.2.7 during runtime refresh; its plan scaffold command now reports that no mission types are registered (`software-dev` unavailable). This plan is authored from repository evidence and the committed specification; CLI phase completion has not been recorded. Repair mission-type registration before using the runtime implementation loop. Qwen command files are registered globally, but the `qwen` executable was not found on this shell PATH.
