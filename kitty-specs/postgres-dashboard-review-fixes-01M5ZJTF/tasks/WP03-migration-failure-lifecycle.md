---
work_package_id: WP03
title: Production migration failure and startup resource lifecycle
dependencies: []
requirement_refs:
- C-002
- C-003
- FR-004
- FR-005
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
base_branch: kitty/mission-postgres-dashboard-review-fixes-01M5ZJTF
base_commit: 6e4c888f7eca13cea44336b6f6a60b357bbd4710
created_at: '2026-10-27T22:15:50.124176+00:00'
subtasks:
- T011
- T012
- T013
- T014
- T015
history: []
agent_profile: implementer-ivan
authoritative_surface: database/
create_intent:
- main_test.go
- database/migration_test.go
- database/migration_external_test.go
execution_mode: code_change
owned_files:
- database/database.go
- database/database_test.go
- database/migration_test.go
- database/migration_external_test.go
- main_test.go
- test/postgres.go
- test/postgres_test.go
role: implementer
tags: []
task_type: implement
tracker_refs: []
---

# WP03: Production migration failure and startup resource lifecycle

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Read any recorded rejection feedback before restarting.

## Objective and success criteria

Return real AutoMigrate failures from production New, close the owned pool and prevent successful startup while preserving healthy SQLite/text/bytea initialization.

**Independent acceptance**: Actual SQLite and PG17 schema conflicts after successful connection cause nil/error from public New; retained owned-pool Ping fails, cleanup failure preserves migration cause, and startup subprocess exits before success.

**Requirement refs**: FR-004, FR-005, C-002, C-003.

## Context and constraints

Read `kitty-specs/postgres-dashboard-review-fixes-01M5ZJTF/spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/remediation.md`, `quickstart.md` and `tasks.md` in that same mission directory. All implementation paths below are repository-root-relative.

Preserve GraphQL/UI fields, integer-bound behavior, local-time semantics, per-span rounding, filtering, zero fill and SQLite/PostgreSQL text/bytea persistence. No dependency/ORM upgrade, destructive conversion, production access, deployment, image publication or external manifest edits. Keep task 114 open.

## Branch strategy and dependencies

**Planning/base**: `plan/postgres-dashboard`. **Final merge target**: `plan/postgres-dashboard`.

**Depends on**: None. Use the runtime checkout/lane from `lanes.json`; this is a single-branch mission, so execute sequentially and do not allocate an ad-hoc worktree.

After `/spec-kitty.analyze` has persisted its report, claim through `spec-kitty agent action implement WP03 --agent codex --mission postgres-dashboard-review-fixes-01M5ZJTF`. Task authoring does not claim or implement this WP.

## Ownership and integration contract

Own production initialization and its harness integration together. Main production fatal boundary already exists; test it without broad main.go refactoring. Keep enum persistence implementations unchanged.

- `database/database.go`
- `database/database_test.go`
- `database/migration_test.go` (planned new file)
- `database/migration_external_test.go` (planned new file)
- `main_test.go`
- `test/postgres.go`
- `test/postgres_test.go`

Sibling owned files are read-only here. A discovered sibling defect returns to its owning WP; do not conceal it with evidence edits. Record any narrowly necessary ownership amendment before proceeding.

## Subtasks and detailed guidance

### T011 — Create real post-connect migration-conflict fixtures

**Deliverable**: Create real post-connect migration-conflict fixtures.

**Implementation**

Add database/migration_external_test.go in package database_test. Use synthetic isolated SQLite and PG databases, connect successfully, then introduce a view named users or equivalent deterministic schema conflict.

Name the PostgreSQL public-New migration-error regression TestMigrationFailure_postgres. It must connect to the configured disposable server and execute the real migration failure, not merely test a malformed/unusable DSN. WP04 registers this exact root in its expected live-suite manifest.

**Boundaries and edge cases**

Assert the intended migration statement fails rather than connection opening. Use existing PostgreSQL isolation/required-mode harness without importing test from internal package database.

**Verification and done evidence**

Public database.New returns the regression failure against reviewed code. Missing DSN in required mode fails rather than skipping the new PG cases.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T011. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T012 — Propagate migration errors and close the owned pool

**Deliverable**: Propagate migration errors and close the owned pool.

**Implementation**

Check AutoMigrate(model.All()...).Error in database.New. On failure close the newly opened pool and return nil plus wrapped migration cause. Emit success only after migration succeeds.

**Boundaries and edge cases**

Keep New signature, model.All foreign-key ordering and successful initialization semantics. A narrow internal post-open initializer is allowed to make pool ownership observable without replacing real public-New integration coverage.

**Verification and done evidence**

Public-New real-engine fixtures pass after the fix. Errors retain migration context and callers never receive a usable initialized handle on failure.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T012. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T013 — Prove pool cleanup and migration-cause priority

**Deliverable**: Prove pool cleanup and migration-cause priority.

**Implementation**

Use database/migration_test.go to retain the exact owned pool used during failure. Verify Ping fails after cleanup and test a controlled Close failure without substituting an unrelated fixture connection.

**Boundaries and edge cases**

Use a narrow test seam only where needed; restore all hooks. Pair controlled cleanup-failure coverage with real-engine tests and, where useful, PostgreSQL session observation.

**Verification and done evidence**

Cleanup runs once and a cleanup error cannot replace or hide the migration cause. A test must fail if failed initialization leaks the pool.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T013. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T014 — Exercise startup fatal behavior in a subprocess

**Deliverable**: Exercise startup fatal behavior in a subprocess.

**Implementation**

Create main_test.go to drive initDatabase with a real migration-conflict database in a subprocess, since zerolog Fatal terminates the process. Assert nonzero exit and migration error context.

**Boundaries and edge cases**

Assert no default-user creation or startup-success message after the error. In database/database_test.go restore mutable hooks including the existing mkdirAll override to prevent test-order coupling.

**Verification and done evidence**

The subprocess distinguishes a real migration failure from invalid DSN/open failure and cannot terminate the parent test process.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T014. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T015 — Make the harness defer migration ownership to production New

**Deliverable**: Make the harness defer migration ownership to production New.

**Implementation**

Remove redundant migrations from test/postgres.go once production New handles errors. Correct stale migration/ordering comments; keep isolated DB creation/removal and required-mode failures.

**Boundaries and edge cases**

Exercise healthy fresh/reopen on both engines, including text and legacy bytea enum read/write and physical-column assertions from existing suites. Do not convert columns or upgrade GORM.

Name the database package's healthy PostgreSQL fresh/reopen regression TestInitializationReopen_postgres, also in database/migration_external_test.go. WP04 requires its live-server execution separately from the expected migration-error case; existing dashboard/device persistence suites retain their own coverage.

**Verification and done evidence**

Focused database/main/test checks and existing dashboard/device persistence cases pass with both engines; provide observed cleanup and startup evidence for review.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T015. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

## Verification strategy

Tests are explicitly required by FR-005/006. Run focused checks first, with generated sources and real UI assets where required. WP05 owns final integrated evidence. Use only synthetic, disposable fixtures.

For engine claims, record the actual connected version and execution counts. Optional PostgreSQL skips in a SQLite run prove nothing about PG17. The dedicated gate requires zero skipped PostgreSQL cases.

Capture reviewed-defect reproduction before applying each fix, then demonstrate the same behavior passes. For documentation packages, validate source-qualified evidence links and historical preservation instead of inventing new runtime claims.

## Risks and mitigations

A view named users may be skipped by GORM introspection; prove that the failure happens during AutoMigrate. Internal database tests importing test would form a cycle.

Missing tools/services and failed required checks remain unresolved verification. Do not weaken assertions, change requirements or claim historical evidence as a fresh result.

## Definition of done and review guidance

Actual SQLite and PG17 schema conflicts after successful connection cause nil/error from public New; retained owned-pool Ping fails, cleanup failure preserves migration cause, and startup subprocess exits before success.

Reviewer checks the complete requirement mapping, declared file ownership and every subtask oracle. Require evidence that exposes the defect, preserves compatibility and distinguishes intended negative checks from failed acceptance.

Do not mark the WP ready solely because a command exited zero if it skipped required cases or emitted unresolved formatting findings. Post-test source changes require affected checks to run again.

## Activity log

2026-10-28 (Europe/Samara): Codex authored this planning prompt. Implementation and executable acceptance have not started; task states belong to the canonical event log.
