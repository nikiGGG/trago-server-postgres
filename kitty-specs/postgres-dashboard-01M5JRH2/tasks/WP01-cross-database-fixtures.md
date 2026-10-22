---
work_package_id: WP01
title: Cross-database fixtures and failure reproduction
dependencies: []
requirement_refs:
- FR-007
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
subtasks: []
phase: Phase 1 - Cross-database fixtures and failure reproduction
history: []
agent_profile: implementer-ivan
authoritative_surface: test/
create_intent:
- test/postgres.go
execution_mode: code_change
owned_files:
- test/db.go
- test/postgres.go
- statistics/summary_test.go
role: implementer
tags: []
task_type: implement
tracker_refs: []
---
# Work Package Prompt: WP01 – Cross-database fixtures and failure reproduction

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Agent/tool: `qwen`. Read recorded review feedback before restarting rejected work.

## Objectives & Success Criteria

Deliver cross-database fixtures and failure reproduction and demonstrate FR-007. All listed subtasks must have reviewable evidence.

## Context & Constraints

Read `../spec.md`, `../plan.md`, `../research.md`, `../tasks.md` and `../handoff.md` relative to this prompt. The user requested planning here and implementation on another device. Do not deploy or modify a live database as part of this package. Preserve SQLite and existing GraphQL/UI contracts. No ORM upgrade, chart redesign or automatic text/bytea migration.

## Branch Strategy

Planning/integration branch: `plan/postgres-dashboard`. Final reviewed PR destination: `master`. Honor the runtime workspace returned for this single-branch mission; do not create an untracked worktree or run several writers concurrently.

**Dependencies**: None.
**Requirement Refs**: FR-007.

## Subtasks & Detailed Guidance

### T001 – Create an isolated PostgreSQL test harness

Extend test helpers with an explicit disposable test DSN and isolated schema/database per test. Keep InMemoryDB unchanged for unrelated tests. Fail fixture creation and seed operations immediately on errors; close connections and clean up. Never accept a production DSN implicitly. Ensure fixed IDs do not conflict with generated sequences.

**Owned surfaces**: `test/db.go`, `test/postgres.go`, `statistics/summary_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T002 – Run existing statistics fixtures against both engines

Refactor existing statistics fixtures into a reusable suite with the same expected values for SQLite and PostgreSQL 17. Preserve assertions and user isolation. Make PostgreSQL absence an explicit local skip, with a required-mode switch that turns absence or no executed tests into failure for CI.

**Owned surfaces**: `test/db.go`, `test/postgres.go`, `statistics/summary_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T003 – Characterize boundaries and reproduce the reported SQL failure

Record current SQLite behavior for empty/reversed/equal ranges, empty keys, active spans, exact boundaries and per-span fractional rounding. Capture the PostgreSQL failure from the current query. Keep known failing reproduction explicitly opt-in until WP03 fixes it so the foundation commit remains usable.

**Owned surfaces**: `test/db.go`, `test/postgres.go`, `statistics/summary_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T004 – Verify harness isolation and document invocation

Repeat fixture runs on both backends; ensure cleanup and reliable failure on an invalid test connection. Document the exact env variables and targeted commands in the test helper comments. Do not change production SQL in this package.

**Owned surfaces**: `test/db.go`, `test/postgres.go`, `statistics/summary_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

## Test Strategy

Follow the package-specific verification above and the plan test matrix. Use disposable PostgreSQL 17 and SQLite; never use production connection settings. Generate existing GraphQL sources before backend tests. Run focused checks first and the required full pipeline at WP05. Keep expected durations independent of the SQL implementation.

## Risks & Mitigations

Driver timestamp formats, PostgreSQL parameter types, local-time boundary conventions, legacy bytea writes and test fixture isolation are known risks. Diagnose the failing layer before changing behavior. Any requirement change must be reflected in the spec and plan for review.

## Review Guidance

Verify the package scope against its requirement refs and acceptance criteria. Review both database paths, error handling and cleanup. Require concrete validation evidence and no unrequested production side effects. Return actionable feedback if any required case is unverified.

## Activity Log

Planning prompt authored by Codex; implementation has not started. Status changes belong in the Spec Kitty event log, not manually invented frontmatter lanes.
