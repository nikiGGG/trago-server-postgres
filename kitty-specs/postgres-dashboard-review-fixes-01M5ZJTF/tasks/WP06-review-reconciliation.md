---
work_package_id: WP06
title: Reconcile historical reviews and acceptance provenance
dependencies:
- WP05
requirement_refs:
- C-003
- C-004
- FR-007
- NFR-003
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
subtasks:
- T025
- T026
- T027
history: []
agent_profile: curator-carla
authoritative_surface: docs/reviews/
create_intent:
- kitty-specs/postgres-dashboard-01M5JRH2/acceptance.md
execution_mode: planning_artifact
owned_files:
- docs/reviews/postgres-dashboard-mission-review-2026-10-27.md
- kitty-specs/postgres-dashboard-01M5JRH2/mission-review.md
- kitty-specs/postgres-dashboard-01M5JRH2/acceptance-matrix.json
- kitty-specs/postgres-dashboard-01M5JRH2/acceptance.md
role: curator
tags: []
task_type: implement
tracker_refs: []
---

# WP06: Reconcile historical reviews and acceptance provenance

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `curator-carla` before implementation. Read any recorded rejection feedback before restarting.

## Objective and success criteria

Append consistent evidence-backed dispositions to both source reports and correct affected acceptance attribution without rewriting historical results.

**Independent acceptance**: Every source-local finding maps to the same WP05 evidence and scoped verdict; historical verdicts remain readable, provenance follows actual events and operational follow-ups/task 114 remain open.

**Requirement refs**: FR-007, NFR-003, C-003, C-004.

## Context and constraints

Read `kitty-specs/postgres-dashboard-review-fixes-01M5ZJTF/spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/remediation.md`, `quickstart.md` and `tasks.md` in that same mission directory. All implementation paths below are repository-root-relative.

Preserve GraphQL/UI fields, integer-bound behavior, local-time semantics, per-span rounding, filtering, zero fill and SQLite/PostgreSQL text/bytea persistence. No dependency/ORM upgrade, destructive conversion, production access, deployment, image publication or external manifest edits. Keep task 114 open.

## Branch strategy and dependencies

**Planning/base**: `plan/postgres-dashboard`. **Final merge target**: `plan/postgres-dashboard`.

**Depends on**: WP05. Use the runtime checkout/lane from `lanes.json`; this is a single-branch mission, so execute sequentially and do not allocate an ad-hoc worktree.

After `/spec-kitty.analyze` has persisted its report, claim through `spec-kitty agent action implement WP06 --agent codex --mission postgres-dashboard-review-fixes-01M5ZJTF`. Task authoring does not claim or implement this WP.

## Ownership and integration contract

This planning_artifact package owns only docs/ and kitty-specs/ report/acceptance surfaces; update them using canonical Spec Kitty artifact placement. Read status.events.jsonl and prior review cycles for provenance; never rewrite the event log to create a desired approval history.

- `docs/reviews/postgres-dashboard-mission-review-2026-10-27.md`
- `kitty-specs/postgres-dashboard-01M5JRH2/mission-review.md`
- `kitty-specs/postgres-dashboard-01M5JRH2/acceptance-matrix.json`
- `kitty-specs/postgres-dashboard-01M5JRH2/acceptance.md` (planned new file)

Sibling owned files are read-only here. A discovered sibling defect returns to its owning WP; do not conceal it with evidence edits. Record any narrowly necessary ownership amendment before proceeding.

## Subtasks and detailed guidance

### T025 — Audit source-local finding and acceptance provenance

**Deliverable**: Audit source-local finding and acceptance provenance.

**Implementation**

Read both full source reports, parent acceptance-matrix.json, recorded status events and relevant review-cycle artifacts. Build a source-qualified reconciliation map before editing current conclusions.

**Boundaries and edge cases**

Codex IDs include RISK-1, RISK-2, DRIFT-1, GAP-1/F-4 and the silent-startup-failure row. Nessy IDs include DRIFT-1/RISK-1, RISK-2 and attribution hygiene; preserve their distinct meanings.

**Verification and done evidence**

Every source-local finding from the spec reconciliation table has an explicit destination and evidence reference; no identity is inferred solely from display labels.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T025. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T026 — Append dated reconciliation addenda to both reviews

**Deliverable**: Append dated reconciliation addenda to both reviews.

**Implementation**

Append addenda to docs/reviews/postgres-dashboard-mission-review-2026-10-27.md and the parent mission-review.md, linking the single WP05 evidence record and tested revision.

**Boundaries and edge cases**

Map each finding to fixed, still open or explicitly deferred with concrete tests/execution support. Keep original commands, observations and historical FAIL/PASS WITH NOTES verdicts identifiable.

**Verification and done evidence**

Both current scoped verdicts agree with actual fresh evidence. Missing verification keeps the corresponding gap open rather than being softened into a passing claim.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T026. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T027 — Correct acceptance provenance and preserve operational follow-ups

**Deliverable**: Correct acceptance provenance and preserve operational follow-ups.

**Implementation**

Update affected parent acceptance-matrix.json fields compatibly and add a dated explanatory acceptance.md when needed. Preserve original records and reference actual event IDs/timestamps/tool identities.

**Boundaries and edge cases**

Correct stale migration/ordering claims with references to WP03 behavior. Do not present recorded agent/user names as independent human review. Keep Codex F-5 hardening and F-6 operational delivery explicitly deferred and Vikunja task 114 open.

**Verification and done evidence**

Validate JSON syntax, links, source-local coverage and tested revision references. No deployment, image publication, external manifest edit or task closure is implied by the documentation.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T027. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

## Verification strategy

Tests are explicitly required by FR-005/006. Run focused checks first, with generated sources and real UI assets where required. WP05 owns final integrated evidence. Use only synthetic, disposable fixtures.

For engine claims, record the actual connected version and execution counts. Optional PostgreSQL skips in a SQLite run prove nothing about PG17. The dedicated gate requires zero skipped PostgreSQL cases.

Capture reviewed-defect reproduction before applying each fix, then demonstrate the same behavior passes. For documentation packages, validate source-qualified evidence links and historical preservation instead of inventing new runtime claims.

## Risks and mitigations

Finding IDs collide across reports. Recorded tool/user names are not proof of independent human review, and a configured image or historical PG14/18 run cannot establish PG17 acceptance.

Missing tools/services and failed required checks remain unresolved verification. Do not weaken assertions, change requirements or claim historical evidence as a fresh result.

## Definition of done and review guidance

Every source-local finding maps to the same WP05 evidence and scoped verdict; historical verdicts remain readable, provenance follows actual events and operational follow-ups/task 114 remain open.

Reviewer checks the complete requirement mapping, declared file ownership and every subtask oracle. Require evidence that exposes the defect, preserves compatibility and distinguishes intended negative checks from failed acceptance.

Do not mark the WP ready solely because a command exited zero if it skipped required cases or emitted unresolved formatting findings. Post-test source changes require affected checks to run again.

## Activity log

2026-10-28 (Europe/Samara): Codex authored this planning prompt. Implementation and executable acceptance have not started; task states belong to the canonical event log.
