# Postgres Dashboard acceptance provenance (reconciliation note)

**Mission**: `postgres-dashboard-01M5JRH2` — "postgres dashboard"
**Date**: 2026-10-28 (Europe/Samara)
**Author / curator**: curator-carla (WP06, `postgres-dashboard-review-fixes-01M5JRH2`)

This document records the acceptance-provenance corrections applied by WP06 to
`acceptance-matrix.json`, why they were made, and what remains open. It references actual event
log records; it does **not** rewrite the event log or create an approval history.

## What was corrected and why

The parent `acceptance-matrix.json` recorded `verified_by: "qwen-review"` for every criterion.
`qwen-review` is **not** an event-recorded identity. The recorded reviewers in
`status.events.jsonl` (2026-10-25) are:

- a human reviewer (`reviewer: "user"`, tool `"Nikita Gryaznev"`) who approved WP01–WP06, and
- an automated reviewer agent (`reviewer: "qwen"`) that recorded a subsequent batch approval.

Per FR-007 reconciliation guidance, recorded agent/user names must not be presented as
independent human review. `verified_by` was therefore corrected to the recorded human reviewer
`"user"`. `qwen` is an automated reviewer and is not presented as independent human review; the
`qwen-review` label was dropped as non-authoritative.

The FR-006 criterion note was also updated: the parent mission's own review (Nessy CLI,
`kitty-specs/postgres-dashboard-01M5JRH2/mission-review.md`) identified that startup migration
errors were swallowed by `database.New` (DRIFT-1/RISK-1). That gap was closed by remediation WP03
(`postgres-dashboard-review-fixes-01M5JRH2`), which propagates AutoMigrate errors and releases the
owned pool — referenced rather than re-asserted here, with the actual-PG17 component left open.

## Reconciliation evidence

Both source reports carry dated reconciliation addenda (2026-10-28) that map every source-local
finding to the single WP05 evidence record:

- Common evidence: `docs/reviews/postgres-dashboard-remediation-evidence-2026-10-28.md`
- Tested remediation revision: `38ae5e9e3818030b1da410a652fdb012877d2973`
- Reports: `docs/reviews/postgres-dashboard-mission-review-2026-10-27.md` (Codex),
  `kitty-specs/postgres-dashboard-01M5JRH2/mission-review.md` (Nessy CLI).

The parent `overall_verdict` remains `pass` (historical). The remediation scoped verdict is:
defects fixed with SQLite / launcher-suite regression evidence; **the actual PostgreSQL 17 gate
(FR-006 / NFR-002, "zero skipped PG cases") remains OPEN** in this environment.

## Explicitly deferred and still open

- **Codex F-5 (hardening)** and **F-6 (operational delivery)** remain explicitly deferred
  follow-ups; no closure is claimed.
- **Vikunja task 114** stays **open**.
- **Actual PostgreSQL 17 execution** (with zero skipped PG cases and real `HostIp`/cleanup
  inspection) is **not verified** here — no `postgres:17` image was obtainable (Docker Hub
  TLS-blocked; corporate mirror auth blocked).
- **WP01 launcher version-probe defect** (dotfile helper) was fixed on lane-a (commit `659cd04`,
  approved review-cycle-3, event `01M5ZRCEXM0H2RWWRVE6HSGAK9`) but that fix is not yet merged onto
  `plan/postgres-dashboard`; the real PG17 gate remains unexecuted regardless.

This documentation implies **no** deployment, image publication, external manifest edit, or
Vikunja task closure.