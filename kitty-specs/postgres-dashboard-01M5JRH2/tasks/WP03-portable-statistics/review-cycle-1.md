---
affected_files: []
cycle_number: 1
mission_slug: postgres-dashboard-01M5JRH2
reproduction_command:
reviewed_at: '2026-10-25T13:24:52Z'
reviewer_agent: user
wp_id: WP03
---

# WP03 Review Feedback — Portable statistics query and error propagation

Reviewer verdict: **REJECT** (blocking correctness issue on the PostgreSQL path).

Summary of what passed: portable SQL structure (explicit CROSS JOIN + ON, qualified keys/IDs, GROUP BY bounds+key+value), parameterization (no user value concatenated), per-span rounding on both dialects, error propagation (`.Error` checked before zero-fill, FR-004 test drops `time_spans`), running-span exclusion, user isolation, Include/Exclude precedence — all verified. SQLite suite PASS, PostgreSQL suite PASS on the UTC-default disposable server, `go vet` and `gofmt` clean. The blocking problem is the PostgreSQL session-TimeZone correctness gap below.

---

## Issue 1 (BLOCKING) — PostgreSQL results depend on the session TimeZone; dashboard statistics are wrong on any non-UTC server

The WP03 CTE range bounds are cast as `?::timestamp` (i.e. `timestamp without time zone`):

```go
// boundRow()
return "(?::timestamp, ?::timestamp)"
```

But the `time_spans.start_user_time` / `end_user_time` columns are migrated as **`timestamp with time zone`**. Confirmed two independent ways:

1. jinzhu/gorm v1.9.2 `dialect_postgres.go` maps a `time.Time` field to `"timestamp with time zone"`.
2. Live inspection of a harness-migrated schema: `SELECT data_type ... WHERE table_name='time_spans' AND column_name='start_user_time'` returns `timestamp with time zone`.

When PostgreSQL compares a `timestamp without time zone` operand against a `timestamptz` column, it coerces the `timestamp` to `timestamptz` using the **session TimeZone**. So the effective range bound shifts by the session offset.

Empirical proof (ran the exact WP03 PG query on a disposable PG, same 1h span `10:00–11:00 UTC`, range `10:00–11:00`):

- `SET TIME ZONE 'Etc/UTC'` → returns `type/review` **3600 s** (correct).
- `SET TIME ZONE 'Asia/Yekaterinburg'` (+05) → returns **0 rows** (span entirely excluded; bound became `05:00–06:00 UTC`).

The WP03 tests only pass because the disposable server defaulted to UTC (this is the implementer's own stated caveat). On any PostgreSQL deployment whose session/`TimeZone` is not UTC — which is common and outside the app's control — the dashboard ranges shift by the server offset and statistics become empty or wrong. That is exactly the failure class this mission exists to fix, and it directly violates:

- plan.md line 49: *"check PostgreSQL session TimeZone independence"*;
- plan.md line 68 (risk): *"Session timezone changes totals or returned ranges | Preserve local wall-clock meaning; verify multiple PostgreSQL session timezones"*;
- plan.md line 24: *"Do not depend on … session TimeZone"*;
- spec FR-002: *"preserving current local user-time semantics"*.

**How to fix (concrete):**

1. Cast the CTE bounds to `timestamptz` to match the column type, e.g. `(?::timestamptz, ?::timestamptz)`. The parameter is already `rg.Start.OmitTimeZone()` (UTC-located), so it encodes as an absolute instant; comparing two `timestamptz` values performs no session-TZ coercion and is correct under any TimeZone.
2. Make the SELECT output normalization TZ-independent as well, e.g. `to_char(dates.query_start AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') || '+00:00'` (and likewise for the end bound). Today `to_char` on a `timestamp without time zone` is already session-TZ independent, but once the bound becomes `timestamptz` the `AT TIME ZONE 'UTC'` is required so the returned wall-clock matches the OmitTimeZone/UTC convention the `group()` parser expects.
3. Add a regression test (T012 / plan test-matrix item) that runs the shared statistics suite under a non-UTC session TimeZone (e.g. `SET TIME ZONE 'Asia/Yekaterinburg'`) and asserts the same fixed expected seconds as UTC. That locks the fix and removes the "verified only on UTC server" caveat.

---

## Issue 2 (test gap, tied to Issue 1) — session-TimeZone independence is unverified

The plan's test matrix explicitly lists session-TimeZone independence, and the implementer's handoff confirms it was not tested (the disposable server defaulted to UTC). Until a non-UTC session-TimeZone regression test exists, FR-002's "preserving current local user-time semantics" on PostgreSQL is unverified. This is resolved by the fix + test in Issue 1.