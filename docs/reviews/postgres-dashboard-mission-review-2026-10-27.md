# Mission Review Report: postgres-dashboard (Codex review)

**Reviewer**: Codex (automated reviewer agent)
**Date**: 2026-10-27
**Mission**: `postgres-dashboard-01M5JRH2` — "postgres dashboard"
**Reviewed revision**: `69fc2ceb650e7c5379f5cba192f70f29e8112ad8`
**Historical verdict**: **FAIL**

> **Reconstruction provenance (WP06, 2026-10-28):** The original Codex review output was not
> persisted into the repository (`docs/reviews/` contained no file under this name in the parent
> mission history). This file is created by work package WP06
> (`postgres-dashboard-review-fixes-01M5JRH2`) to serve as the durable Codex review record. The
> findings and the historical FAIL verdict below are reconstructed from the authoritative
> reconciliation record in `kitty-specs/postgres-dashboard-review-fixes-01M5JRH2/spec.md`
> (Goal + "Verification and reconciliation contract") and from the shared remediation evidence
> (`docs/reviews/postgres-dashboard-remediation-evidence-2026-10-28.md`). No verification claim is
> fabricated; dispositions are recorded by the reconciliation addendum appended below.

---

## Findings (as recorded in the reconciliation record)

| ID | Finding | Historical verdict component |
| --- | --- | --- |
| RISK-1 | Network exposure — the disposable PostgreSQL launcher could publish on a non-loopback address, exposing an unauthenticated test database. | FAIL component |
| RISK-2 | Credential disclosure — launcher output could print the connection DSN and/or its embedded password. | FAIL component |
| DRIFT-1 | Fractional boundary loss — fractional range statistics lose original range identity / correct clipped per-span totals. | FAIL component |
| GAP-1 / F-4 | Missing PostgreSQL 17 and full-pipeline execution — the reviewed revision was not executed on actual PostgreSQL 17 with zero skipped PostgreSQL cases, nor with a complete lint/test run. | FAIL / open verification |
| Silent-startup-failure row | Swallowed startup migration error — `database.New` ignores the `AutoMigrate` error, so a startup migration failure is not surfaced and the application may proceed against an incompatible schema. | FAIL component |
| F-5 | Hardening follow-up (external). | Deferred follow-up |
| F-6 | Operational delivery follow-up (external). | Deferred follow-up |

---

## Reconciliation addendum (2026-10-28)

**WP**: WP06 `postgres-dashboard-review-fixes-01M5JRH2`
**Common evidence**: `docs/reviews/postgres-dashboard-remediation-evidence-2026-10-28.md` (WP05)
**Tested remediation revision**: `38ae5e9e3818030b1da410a652fdb012877d2973`
(`feat(WP04): wire launcher CI gates and enforce PostgreSQL live-suite manifest`, branch
`kitty/mission-postgres-dashboard-review-fixes-01M5JRH2-lane-e`)
**Curator**: curator-carla (WP06)

This addendum reconciles each source-local Codex finding above to the WP05 evidence. The
historical FAIL verdict remains readable above; the current scoped verdict is derived from the
shared evidence below. Where a component is marked **OPEN**, the finding is **not** claimed
verified — it stays open.

| Codex finding | Maps to (remediation) | WP05 evidence | Current disposition |
| --- | --- | --- | --- |
| RISK-1 (network exposure) | FR-001 / NFR-001 | `make test-database-launcher` → **16 cases run, 0 failed, 0 skipped** (loopback-only publish for default and custom ports, cleanup on success and failure, reused-DSN no-cleanup); failed real run still showed loopback publish intent (`127.0.0.1:55432`) and cleanup. | **FIXED** (launcher/regressions). Real PG17 `HostIp` inspection of a live container remains **OPEN** (PG17 image unavailable). |
| RISK-2 (credential disclosure) | FR-002 / NFR-001 | Launcher shell-suite cases 3–6, 10–12, 15; zero sentinel passwords / credential DSNs in captured stdout/stderr across success, setup-failure, test-failure and controlled child-echo paths. | **FIXED** |
| DRIFT-1 (fractional boundary loss) | FR-003 / C-001 | SQLite fractional regressions all PASS (`TestStats/fractional_*`, `TestStats2_single_fractional_sqlite`, `boundaries_1..3`). | **FIXED on SQLite**; shared real-PG17 fractional fixtures **NOT verified** (PG17 image unavailable) → PG17 component **OPEN**. |
| GAP-1 / F-4 (missing PG17 + full pipeline) | FR-006 / NFR-002 | Full `make lint` (go vet PASS; goimports 0 findings; yarn format/lint PASS) and `make test` (backend race+coverage all `ok`; frontend 2/2) both exit 0. Actual PG17 gate `make test-postgres` **NOT executed** (image unavailable; Docker Hub TLS block, mirror auth blocked). | Full pipeline portion **PASS**. Actual PostgreSQL 17 with zero skips **OPEN** (NFR-002 / SC-002). |
| Silent-startup-failure row | FR-004 / C-002 | `TestNew_closesOwnedPoolOnMigrationError`, `TestNew_cleanupRunsOnce`, `TestNew_migrationCauseSurvivesCloseFailure`, `TestMigrationFailure_sqlite` all PASS (SQLite). | **FIXED on SQLite**; real PG17 migration-failure path **NOT verified** → PG17 component **OPEN**. |
| F-5 (hardening) | — | — | **DEFERRED** (explicit separate follow-up). |
| F-6 (operational delivery) | — | — | **DEFERRED** (explicit separate follow-up). Vikunja task 114 remains **open**. |

### Related real-launcher defect (routed to WP01)

WP05 also identified a defect in WP01's owned launcher: `hack/test-databases.sh:191` named the
version-probe helper `.pgversion-$$.go` (a dotfile), which the Go toolchain ignores, so `go run`
on the real helper failed and blocked real version gating on both launcher paths. This was
routed to WP01 and is **not** amended by this WP.

Post-WP05, WP01's cycle-2 fix renamed the helper to a non-dotfile `pgversion_$$.go` and added a
regression guard rejecting dotfile/underscore helper names (commit `659cd04` on
`kitty/mission-postgres-dashboard-review-fixes-01M5ZJTF-lane-a`; approved review-cycle-3, event
`01M5ZRCEXM0H2RWWRVE6HSGAK9`). **The real PostgreSQL 17 gate nevertheless remains OPEN** — no
`postgres:17` image was obtainable in this environment, so actual PG17 execution has still not
been performed. The WP01 fix has not yet been merged onto `plan/postgres-dashboard` at the time
of this addendum.

### Current scoped verdict

All four defects are addressed in code with SQLite and/or launcher-suite regression evidence
(FR-001, FR-002, FR-003-on-SQLite, FR-004-on-SQLite, FR-005, FR-006-pipeline PASS). **The actual
PostgreSQL 17 gate (FR-006 / NFR-002, "zero skipped PG cases") remains OPEN** due to environment
constraints, and the PG17-dependent components of FR-003/FR-004/FR-001 (real `HostIp`
inspection) are **not verified**. No deployment, image publication, external manifest edit or
task-114 closure is implied by this documentation.