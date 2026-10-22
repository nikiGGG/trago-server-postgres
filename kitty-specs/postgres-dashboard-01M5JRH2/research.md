# Repository findings and decisions

## Evidence

- `statistics/summary.go`: `julianday` duration arithmetic; `dates JOIN time_span_tags` without an ON condition; selected `query_end` absent from GROUP BY; unqualified `id` in the exclusion subquery; ignored `Raw(...).Scan(...)` result and return of `r.DB.Error`.
- `statistics/summary2.go`: creates relative ranges using settings and delegates to Stats. It needs integration coverage rather than a second SQL implementation.
- `model/dashboard.go`, `model/device.go`: enum Valuers write strings; Scanners accept only `[]byte`. This explains the task's reported text-column workaround. Reading both `string` and `[]byte` is the proposed application fix; bytea writes still require an actual PostgreSQL round-trip test.
- `test/db.go`: all current shared test fixtures use in-memory SQLite. PostgreSQL needs a separate disposable database/schema, checked fixture writes, deterministic cleanup, and no production DSN.
- `model/time.go`: OmitTimeZone preserves wall-clock fields and places them in UTC. `time/range.go` builds adjacent ranges with second-inclusive ends. Preserve these semantics; do not silently change to elapsed UTC duration or half-open ranges in this compatibility mission.
- `statistics/summary.go`: range result parser assumes one timestamp string format; range sorting compares one row's Start with another row's End. Typed timestamp results and repeated-start/unsorted ranges need characterization. Fix only if reproduced in the planned cross-database fixtures.
- `.github/workflows/build.yml`, `Makefile`: generated GraphQL sources and UI assets are prerequisites to the current full suite. Retain that setup in CI before tests.
- Source checkout is `8ca9bb6`, whereas task 114 reports an older deployed revision. Release must identify which upstream changes are included.

## Design choice

Keep aggregation in SQL and share query structure/filter generation. Select a small, fixed duration and range-cast fragment from `DB.Dialect().GetName()` for PostgreSQL versus SQLite. This minimizes application memory and avoids an unrelated aggregation rewrite. Use explicit CROSS JOIN/ON joins, qualified columns and full grouping. Only code-owned SQL fragments vary; all user values remain bound parameters.

Use PostgreSQL `ROUND(EXTRACT(EPOCH FROM (clipped_end - clipped_start)))` for seconds, with explicit bound-range timestamp types aligned with actual column definitions. Keep SQLite's existing `ROUND((julianday(clipped_end) - julianday(clipped_start)) * 86400, 0)` calculation. Preserve per-span rounding; rounding after SUM would change results.

A single universal SQL expression would require changing timestamp storage or loading raw spans into Go. Both expand scope unnecessarily. An ORM/driver upgrade is also unnecessary for the initial fix.

## Primary references

- [PostgreSQL 17 date/time arithmetic](https://www.postgresql.org/docs/17/functions-datetime.html): timestamp subtraction and extracting epoch seconds.
- [GORM v1 error handling](https://v1.gorm.io/docs/error_handling.html) and [method chaining](https://v1.gorm.io/docs/method_chaining.html): inspect the returned query handle's Error field; chain operations clone DB objects.

## Unverified items

No production database or cluster was accessed. SQL failures are reported by the task and supported by source inspection, but have not been reproduced against PostgreSQL in this planning session. Verify PostgreSQL timestamp types, session TimeZone independence, legacy bytea writes, and the exact deployment manifest/image before implementation is accepted.
