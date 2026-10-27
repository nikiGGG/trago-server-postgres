# Mission Review Report: postgres-dashboard-01M5JRH2

**Reviewer**: Nessy CLI (spec-kitty-mission-review)
**Date**: 2026-10-27
**Mission**: `postgres-dashboard-01M5JRH2` — "postgres dashboard"
**Baseline commit**: `fb2538f66aa1df6ed2b7abc636d03e9cf4643047` (mission_merge_commit `dc074f5…`)
**HEAD at review**: `69fc2ceb650e7c5379f5cba192f70f29e8112ad8`
**WPs reviewed**: WP01–WP06 (24 subtasks, all `done`; 6/6 merged)

---

## Gate Results

Gates 1–4 from the spec-kitty mission-review doctrine apply to missions **in the spec-kitty CLI repo itself**. This is a `software-dev` mission in the downstream **Go product repo** (`github.com/traggo/server`), so the Python contract/architectural/e2e suites do not exist here. The applicable equivalents were run instead.

| Gate | Command | Exit | Result |
|------|---------|------|--------|
| 1 — Contract tests | N/A (no `tests/contract/` in this repo) | — | N/A |
| 2 — Architectural tests | `go build ./...` + `go vet ./<changed pkgs>` | 0 / 0 | PASS |
| 3 — Cross-repo E2E | N/A (no spec-kitty-e2e scenario for a product mission) | — | N/A |
| 4 — Issue Matrix | `issue-matrix.json` absent; spec references **no GitHub issues** (only Vikunja task 114) | — | N/A (acceptance-matrix.json present, thorough) |

**Local verification run**: `go build ./...` exit 0; `go vet` exit 0; `go test -count=1 ./statistics/... ./model/... ./dashboard/... ./device/... ./database/...` all **PASS** (fresh, non-cached). PG-gated suites `SKIP` cleanly without a DSN and **FAIL (exit 1)** under `TRAGGO_TEST_POSTGRES_REQUIRED=1` without one — FR-007's no-silent-skip contract holds.

---

## FR Coverage Matrix

All 8 FRs trace spec → WP → test → code with real (non-synthetic) evidence. Adequacy assessment:

| FR | WP Owner | Test Evidence | Adequacy | Finding |
|----|---------------|----------|---------|
| FR-001 | WP03, WP04 | `TestStatsGraphQLShapes_sqlite/_postgres` (real resolver calls on both engines); shared `statsFixtures` | ADEQUATE | — |
| FR-002 | WP03, WP04 | Boundary fixtures; `TestStats_postgres_sessionTimeZone`, `TestStats2_daily_postgres_sessionTimeZone` (force `SET TIME ZONE 'Asia/Yekaterinburg'`); hand-calc `cyrillicDailySecs` | ADEQUATE | — |
| FR-003 | WP03, WP04 | `multiple_*`, `exclude_precedence`, `user_isolation`, zero-fill fixtures | ADEQUATE | — |
| FR-004 | WP03 | `TestStats_errorPropagation` — drops `time_spans` *after* seeding; asserts resolver error | ADEQUATE | — |
| FR-005 | WP02 | `TestDashboardType_Scan`, `TestInterval_Scan`, `TestDeviceType_Scan` (string/[]byte/nil/unsupported/no-mutate) + real PG text **and** bytea round-trips | ADEQUATE | — |
| FR-006 | WP02, WP04 | File-SQLite reopen, PG reconnect via `database.New`, `TestStartupMigration_reopen_textAndBytea` | ADEQUATE | DRIFT-1 (below) |
| FR-007 | WP01, WP05 | `postgres-test` CI job (`postgres:17`, trust auth, `REQUIRED=1`), `hack/test-databases.sh`, harness self-tests | ADEQUATE | — |
| FR-008 | WP06 | `docs/postgres-dashboard-rollout.md` (build/immutable ID/staging/rollout/rollback), `README.md` | ADEQUATE | — |

**WP03 review signal**: WP03 had exactly one rejection cycle (`changes_requested` → backward rewind, `force:true` for the rewind only — a normal rework path, not a forced approval). No `ReviewerSelfApproval`, no arbiter override anywhere. The reject was a genuine, high-quality adversarial finding (session-TimeZone dependence). The fix **is** shipped in code (`(?::timestamptz, ...)`, `to_char(... AT TIME ZONE 'UTC') || '+00:00'`, explicit `queryHandle.Error` check) and is locked by the non-UTC regression tests. Verified.

---

## Drift Findings

### DRIFT-1: `database.New` discards the AutoMigrate error — startup-migration errors are not actually verified end-to-end
**Type**: PUNTED-FR (partial) / verification gap
**Severity**: MEDIUM (non-blocking)
**Spec/plan reference**: FR-006; plan.md "…checking startup migration errors" (verification boundary); T008
**Evidence**:
- `database/database.go`: `db.AutoMigrate(model.All()...)` — return value **ignored** (pre-existing).
- `dashboard/postgres_persistence_test.go` `openRawPostgres`: `gdb, err := database.New(...)`; `require.NoError(t, err)` — `database.New` only returns the `gorm.Open` error, so this assertion cannot detect an AutoMigrate failure.
- `test/postgres.go` comment confirms: *"database.New ignores the AutoMigrate error, so surface it here"* — the harness's own AutoMigrate calls (not the production path) are the only error-checked ones.

**Analysis**: T008 (and WP06's FR-006 evidence table) claim startup migration is checked on text and bytea schemas. What is genuinely verified is **no destructive column conversion** (column type retained after reopen) and **data readability** — both good. But the specific plan item "checking startup migration errors" is not actually exercised through the production `database.New` path, because that path swallows the error. In production, a startup `AutoMigrate` failure on an existing PostgreSQL schema would be logged only as a debug line ("Auto migrating schema's") and the app would proceed against a possibly-incompatible schema. This is a genuine silent-failure hole the mission scoped but did not close.

---

## Risk Findings

### RISK-1: Silent startup-migration failure (production)
**Type**: ERROR-PATH / silent default
**Severity**: MEDIUM
**Location**: `database/database.go` (`db.AutoMigrate(...)`), plus any migration-error check relying on `database.New`'s returned error.
**Trigger**: A real PostgreSQL schema where AutoMigrate fails at startup (e.g. an index/column it cannot create on the bytea workaround shape).
**Analysis**: No error surfaces; the app starts and may serve empty/wrong statistics. Existing tests (T008) empirically pass because the bytea/text reopen happens to work; they would not catch a non-catastrophic AutoMigrate error. Recommendation (follow-up): surface the AutoMigrate error from `database.New` and add a T008 variant that asserts migration-error propagation (not just column retention).

### RISK-2: Stale harness comment contradicts shipped `model.All()` ordering
**Type**: CROSS-WP-INTEGRATION / doc drift
**Severity**: LOW
**Location**: `test/postgres.go` — "model.All() declares TagDefinition before User, and TagDefinition carries REFERENCES users(id)…" — this is now **false**: WP02's `model/all.go` reorders `User` before `TagDefinition`. The redundant harness `AutoMigrate(&model.User{})` is harmless, but the comment no longer describes the code. Non-blocking.

---

## Silent Failure Candidates

| Location | Condition | Silent result | Spec impact |
|----------|---------------|-------------|
| `database/database.go` (`db.AutoMigrate`) | Startup migration error on existing schema | proceeds, no error returned | FR-006 / T008 — migration error not surfaced |
| `statistics/summary.go` tail | Post-`Scan` `group`/zero-fill operations | previously `r.DB.Error` now `return nil` | **None** — verified those ops issue no DB queries; explicit `Scan(...).Error` is checked. Not a regression; an improvement. |

No other silent-default paths were found in the new production code.

---

## Security Notes

| Finding | Location | Risk class | Recommendation |
|---------|----------|------------|----------------|
| None blocking | — | — | — |

The mission introduces **no** production subprocess, shell, network, or credential handling. `SET TIME ZONE '"+sessionTZ+"'"` in tests inlines a fixed constant (timezone names cannot contain `'`), documented and never user input. The CI `postgres:17` service uses `trust` auth **only** for a disposable container, explicitly flagged as never-for-production. The runbook deliberately avoids committing credentials. No injection, path-traversal, or TOCTOU vectors in scope.

---

## Final Verdict

**PASS WITH NOTES**

### Verdict rationale
All 8 FRs are implemented and covered by real, behavior-constraining tests on both SQLite and disposable PostgreSQL; the mission's single critical bug (WP03 session-TimeZone dependence) was correctly rejected, fixed in code, and locked by non-UTC regression tests with no forced/self-approval. Production blast radius is small and contained (`statistics/summary.go`, `model/{dashboard,device,all}.go`). Build, vet, and the full SQLite suite pass fresh; the CI design for mandatory PostgreSQL 17 is sound and its no-silent-skip contract is verified. FR-008 is documented honestly with explicitly unverified external inputs. The only findings are MEDIUM/LOW and non-blocking: `database.New` swallows the AutoMigrate error (DRIFT-1 / RISK-1), so startup-migration *error propagation* is not truly verified; plus minor stale-comment and acceptance-matrix reviewer-attribution (`verified_by: qwen-review` vs. the human reviewer recorded in the event log) hygiene notes.

### Open items (non-blocking)
1. **DRIFT-1/RISK-1** — surface the AutoMigrate error from `database.New` and add a T008 assertion that a migration error propagates. Highest-value follow-up.
2. **RISK-2** — correct the stale `model.All()` ordering comment in `test/postgres.go`.
3. **Process** — align `acceptance-matrix.json` `verified_by` with the recorded reviewer (human `user`) for accuracy.
4. **External** (already documented in runbook §6, not a gap): image digest, `traggo.yaml`/`65b29d3` manifest change, and live rollout remain separate execution steps pending owning-repo access.

---

## Retrospective Reminder

`retrospective.yaml` **exists** (schema v1, authored by runtime terminus, provenance `runtime_post_completion`, `created_at 2026-10-25T14:00:18Z`); no `RetrospectiveCaptureFailed` events in the log. Proceed to surface findings:

- `spec-kitty retrospect summary` — cross-mission aggregation (read-only).
- `spec-kitty agent retrospect synthesize --mission postgres-dashboard-01M5JRH2` — inspect proposals (dry-run by default); use `--apply` only to commit staged proposals.

---

## Reconciliation addendum (2026-10-28)

**WP**: WP06 `postgres-dashboard-review-fixes-01M5JRH2`
**Common evidence**: `docs/reviews/postgres-dashboard-remediation-evidence-2026-10-28.md` (WP05)
**Tested remediation revision**: `38ae5e9e3818030b1da410a652fdb012877d2973`
(`feat(WP04): wire launcher CI gates and enforce PostgreSQL live-suite manifest`, branch
`kitty/mission-postgres-dashboard-review-fixes-01M5JRH2-lane-e`)
**Curator**: curator-carla (WP06)

This addendum reconciles each source-local finding from **this** (Nessy CLI) report to the WP05
evidence. The historical **PASS WITH NOTES** verdict above remains readable and unchanged. Where a
component is marked **OPEN**, it is not claimed verified.

| Finding (this report) | Maps to (remediation) | WP05 evidence | Current disposition |
| --- | --- | --- | --- |
| DRIFT-1 / RISK-1 — `database.New` swallows the AutoMigrate error | FR-004 / C-002 | `TestNew_closesOwnedPoolOnMigrationError`, `TestNew_cleanupRunsOnce`, `TestNew_migrationCauseSurvivesCloseFailure`, `TestMigrationFailure_sqlite` all PASS (SQLite). This also closes the Codex "silent-startup-failure" row. | **FIXED on SQLite**; real PG17 migration-failure path **NOT verified** (PG17 image unavailable) → PG17 component **OPEN**. |
| RISK-2 — stale harness comment (`model.All()` ordering in `test/postgres.go`) | WP03 / T015 behavior | Harness now documents `database.New` running `AutoMigrate(model.All()...)` itself with propagation, and `model.All()` order already topologically sorted (users before tag_definitions); harness no longer owns a separate migration step. | **FIXED** |
| Attribution hygiene — `acceptance-matrix.json` `verified_by: "qwen-review"` vs the human reviewer in the event log | FR-007 / T027 | Recorded parent-mission reviewers per `status.events.jsonl` (2026-10-25): human `user` approved WP01–WP06; an automated `qwen` review followed. Corrected in the parent acceptance-matrix and explained in `acceptance.md`. | **FIXED** (see `acceptance.md`); note `qwen` is an automated reviewer and is not presented as independent human review. |

### Current scoped verdict

This report's open items are closed as follows: DRIFT-1/RISK-1 is **fixed on SQLite** with the
PG17 component still **OPEN**; RISK-2 is **fixed**; attribution hygiene is **fixed** via T027.
The mandatory PostgreSQL 17 gate (FR-006 / NFR-002, "zero skipped PG cases") was **not executed**
in this environment (no `postgres:17` image obtainable; Docker Hub TLS-blocked, mirror auth
blocked) and therefore remains **OPEN**. External items (image digest, `traggo.yaml`/`65b29d3`
manifest change, live rollout) remain separate follow-ups; Vikunja task 114 stays open. No
deployment, image publication, external manifest edit or task-114 closure is implied.