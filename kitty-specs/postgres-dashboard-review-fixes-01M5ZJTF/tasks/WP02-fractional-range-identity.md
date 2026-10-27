---
work_package_id: WP02
title: Lossless fractional statistics and GraphQL bounds
dependencies: []
requirement_refs:
- C-001
- FR-003
- FR-005
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
base_branch: kitty/mission-postgres-dashboard-review-fixes-01M5ZJTF
base_commit: 6e4c888f7eca13cea44336b6f6a60b357bbd4710
created_at: '2026-10-27T22:15:31.656608+00:00'
subtasks:
- T006
- T007
- T008
- T009
- T010
history: []
agent_profile: implementer-ivan
authoritative_surface: statistics/
create_intent: []
execution_mode: code_change
owned_files:
- statistics/summary.go
- statistics/summary_test.go
- statistics/summary2_test.go
- model/time.go
- model/time_test.go
- graphql/handler_test.go
role: implementer
tags: []
task_type: implement
tracker_refs: []
---

# WP02: Lossless fractional statistics and GraphQL bounds

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Read any recorded rejection feedback before restarting.

## Objective and success criteria

Decouple aggregate identity from database timestamp formatting and preserve accepted precision through the serialized GraphQL response.

**Independent acceptance**: Shared SQLite/PG17 fixtures return independent expected seconds and original bounds; scalar and actual GraphQL response assertions preserve nanoseconds, and existing integer/filter/timezone cases remain passing.

**Requirement refs**: FR-003, FR-005, C-001.

## Context and constraints

Read `kitty-specs/postgres-dashboard-review-fixes-01M5ZJTF/spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/remediation.md`, `quickstart.md` and `tasks.md` in that same mission directory. All implementation paths below are repository-root-relative.

Preserve GraphQL/UI fields, integer-bound behavior, local-time semantics, per-span rounding, filtering, zero fill and SQLite/PostgreSQL text/bytea persistence. No dependency/ORM upgrade, destructive conversion, production access, deployment, image publication or external manifest edits. Keep task 114 open.

## Branch strategy and dependencies

**Planning/base**: `plan/postgres-dashboard`. **Final merge target**: `plan/postgres-dashboard`.

**Depends on**: None. Use the runtime checkout/lane from `lanes.json`; this is a single-branch mission, so execute sequentially and do not allocate an ad-hoc worktree.

After `/spec-kitty.analyze` has persisted its report, claim through `spec-kitty agent action implement WP02 --agent codex --mission postgres-dashboard-review-fixes-01M5ZJTF`. Task authoring does not claim or implement this WP.

## Ownership and integration contract

Keep query/group reconstruction, scalar serialization and their regression tests in one package: resolver-only fixes cannot satisfy the wire contract. Do not rewrite Stats2 range generation or touch schema.graphql.

- `statistics/summary.go`
- `statistics/summary_test.go`
- `statistics/summary2_test.go`
- `model/time.go`
- `model/time_test.go`
- `graphql/handler_test.go`

Sibling owned files are read-only here. A discovered sibling defect returns to its owning WP; do not conceal it with evidence edits. Record any narrowly necessary ownership amendment before proceeding.

## Subtasks and detailed guidance

### T006 — Add shared fractional-range reproduction fixtures

**Deliverable**: Add shared fractional-range reproduction fixtures.

**Implementation**

Extend statistics/summary_test.go for both engines using one completed tagged span 10:00:00–10:00:10Z. Assert explicit numeric results rather than equality between engines.

**Boundaries and edge cases**

Ranges relative to start: 0.200–1.000 => 1; 0.000–1.800 => 2; 0.200–1.800 => 2; 0.800–1.200 => 0; 0.200000001–1.800000001 => 2. Request the same-whole-second identity pair together.

**Verification and done evidence**

Reviewed source fails original-bound/nonempty aggregate assertions. Include empty ranges, duplicate identical ranges and multiple spans that distinguish sum-of-rounded durations from rounding the sum.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T006. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T007 — Group aggregate results by parameterized integer range identity

**Deliverable**: Group aggregate results by parameterized integer range identity.

**Implementation**

Add range_id per input position to dates CTE rows. Keep typed PostgreSQL bounds and dialect duration fragments. Scan ID, tag key/value and seconds, then attach aggregates by ID to responses built from original Go bounds.

**Boundaries and edge cases**

Remove boundStartExpr/boundEndExpr and string-bound reconstruction only after confirming uses. Preserve one aggregate query, completed-span overlap, user isolation, Include/Exclude precedence and tag zero fill.

**Verification and done evidence**

Distinct fractional inputs cannot merge or disappear, including submicrosecond differences. Duplicate ranges each return the original total, without duplicate-join amplification.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T007. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T008 — Preserve accepted precision in scalar and wire serialization

**Deliverable**: Preserve accepted precision in scalar and wire serialization.

**Implementation**

Use RFC3339Nano in model.Time.MarshalGQL while retaining accepted parsing, UTC response convention and whole-second formatting. Add scalar round-trip cases in model/time_test.go.

**Boundaries and edge cases**

Add an actual serialized statistics GraphQL response assertion in graphql/handler_test.go, using existing handler/auth fixtures. Assert fractional start/end strings survive JSON decoding; a direct resolver result is insufficient.

**Verification and done evidence**

Nanosecond bounds remain exact on wire and whole-second snapshots remain compatible. Do not hand-edit generated GraphQL sources or add fields.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T008. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T009 — Protect Stats2 and existing compatibility semantics

**Deliverable**: Protect Stats2 and existing compatibility semantics.

**Implementation**

Extend statistics/summary2_test.go where its current static/relative inputs support fractional bounds. Preserve the existing second-inclusive daily builder and its delegation to Stats; no parser expansion.

**Boundaries and edge cases**

Retain integer bounds, filtering, grouping, user offsets, non-UTC PostgreSQL sessions, running-span behavior and empty-entry shape. Expectations avoid ambiguous half-second floating thresholds.

**Verification and done evidence**

Both engines pass relevant existing and new cases. Unsorted-range redesign and additional general query/scan hardening remain deferred unless a scoped assertion requires them.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T009. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T010 — Verify focused regression coverage on both engines

**Deliverable**: Verify focused regression coverage on both engines.

**Implementation**

After existing generation preparation, run go test -count=1 ./model/... ./statistics/... ./graphql/... with disposable SQLite and configured PG17. Use required mode for PostgreSQL and record actual test names/counts.

**Boundaries and edge cases**

Do not call an absent-DSN skip a PG pass. If PG17 is unavailable, preserve the verification gap for WP05. Resolve package failures within ownership and rerun affected checks.

**Verification and done evidence**

Supply red/green findings, exact response examples and successful relevant checks for independent review; WP05 still owns complete integrated execution.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T010. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

## Verification strategy

Tests are explicitly required by FR-005/006. Run focused checks first, with generated sources and real UI assets where required. WP05 owns final integrated evidence. Use only synthetic, disposable fixtures.

For engine claims, record the actual connected version and execution counts. Optional PostgreSQL skips in a SQLite run prove nothing about PG17. The dedicated gate requires zero skipped PostgreSQL cases.

Capture reviewed-defect reproduction before applying each fix, then demonstrate the same behavior passes. For documentation packages, validate source-qualified evidence links and historical preservation instead of inventing new runtime claims.

## Risks and mitigations

PostgreSQL stores less precision than accepted timestamps; exact identity must use input position while arithmetic retains native DB precision. Duplicate requests must not multiply aggregate seconds.

Missing tools/services and failed required checks remain unresolved verification. Do not weaken assertions, change requirements or claim historical evidence as a fresh result.

## Definition of done and review guidance

Shared SQLite/PG17 fixtures return independent expected seconds and original bounds; scalar and actual GraphQL response assertions preserve nanoseconds, and existing integer/filter/timezone cases remain passing.

Reviewer checks the complete requirement mapping, declared file ownership and every subtask oracle. Require evidence that exposes the defect, preserves compatibility and distinguishes intended negative checks from failed acceptance.

Do not mark the WP ready solely because a command exited zero if it skipped required cases or emitted unresolved formatting findings. Post-test source changes require affected checks to run again.

## Activity log

2026-10-28 (Europe/Samara): Codex authored this planning prompt. Implementation and executable acceptance have not started; task states belong to the canonical event log.
