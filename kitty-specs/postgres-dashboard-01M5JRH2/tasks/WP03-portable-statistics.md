---
work_package_id: WP03
title: Portable statistics query and error propagation
dependencies:
- WP01
requirement_refs:
- FR-001
- FR-002
- FR-003
- FR-004
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
subtasks: []
phase: Phase 3 - Portable statistics query and error propagation
history: []
agent_profile: implementer-ivan
authoritative_surface: statistics/
create_intent: []
execution_mode: code_change
owned_files:
- statistics/summary.go
- statistics/summary_test.go
role: implementer
tags: []
task_type: implement
tracker_refs: []
---
# Work Package Prompt: WP03 – Portable statistics query and error propagation

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Agent/tool: `qwen`. Read recorded review feedback before restarting rejected work.

## Objectives & Success Criteria

Deliver portable statistics query and error propagation and demonstrate FR-001, FR-002, FR-003, FR-004. All listed subtasks must have reviewable evidence.

## Context & Constraints

Read `../spec.md`, `../plan.md`, `../research.md`, `../tasks.md` and `../handoff.md` relative to this prompt. The user requested planning here and implementation on another device. Do not deploy or modify a live database as part of this package. Preserve SQLite and existing GraphQL/UI contracts. No ORM upgrade, chart redesign or automatic text/bytea migration.

## Branch Strategy

Planning/integration branch: `plan/postgres-dashboard`. Final reviewed PR destination: `master`. Honor the runtime workspace returned for this single-branch mission; do not create an untracked worktree or run several writers concurrently.

**Dependencies**: WP01.
**Requirement Refs**: FR-001, FR-002, FR-003, FR-004.

## Subtasks & Detailed Guidance

### T009 – Introduce dialect-specific duration and range typing

Use the active GORM dialect to select fixed SQL fragments. PostgreSQL uses ROUND(EXTRACT(EPOCH FROM clipped_end - clipped_start)); SQLite retains julianday arithmetic and per-span rounding. Explicitly type PostgreSQL CTE range bounds to match inspected local user-time columns. Bind every user value; do not concatenate user input.

**Owned surfaces**: `statistics/summary.go`, `statistics/summary_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T010 – Make joins grouping and filter correlations portable

Replace implicit joins with explicit CROSS JOIN and ON relationships. Qualify ownership, keys and subquery IDs, including tstx.time_span_id = tsx.id. Group by both query bounds, key and value. Preserve Include OR/Exclude OR, exclusion precedence and running-span behavior without multiplying durations.

**Owned surfaces**: `statistics/summary.go`, `statistics/summary_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T011 – Propagate query errors and normalize driver results

Check Error on the returned Raw(...).Scan handle before zero filling. Verify PostgreSQL timestamp scan forms and normalize at the boundary if needed. Test a real execution error without breaking fixture setup. Characterize range ordering and only fix grouping/sorting if required fixtures reveal a correctness failure.

**Owned surfaces**: `statistics/summary.go`, `statistics/summary_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T012 – Pass the shared statistics regression suite

Enable the previously opt-in PostgreSQL reproduction as a normal passing regression. Cover clipping, fractional rounding, multiple tags, duplicate filter matches, empty results, same-start/different-end ranges and user isolation. Compare each backend with fixed expected seconds; do not merely compare engines to each other.

**Owned surfaces**: `statistics/summary.go`, `statistics/summary_test.go`.

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
