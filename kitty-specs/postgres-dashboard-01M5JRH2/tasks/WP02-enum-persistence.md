---
work_package_id: WP02
title: Dashboard and device enum persistence
dependencies:
- WP01
requirement_refs:
- FR-005
- FR-006
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
base_branch: kitty/mission-postgres-dashboard-01M5JRH2
base_commit: 75ae7a88927253743c62f0629bad6d8a7df14721
created_at: '2026-10-25T13:30:25.751209+00:00'
subtasks: []
phase: Phase 2 - Dashboard and device enum persistence
history: []
agent_profile: implementer-ivan
authoritative_surface: model/
create_intent:
- model/dashboard_test.go
- dashboard/postgres_persistence_test.go
- device/postgres_persistence_test.go
execution_mode: code_change
owned_files:
- model/dashboard.go
- model/device.go
- model/dashboard_test.go
- model/device_test.go
- dashboard/postgres_persistence_test.go
- device/postgres_persistence_test.go
role: implementer
tags: []
task_type: implement
tracker_refs: []
---
# Work Package Prompt: WP02 – Dashboard and device enum persistence

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Agent/tool: `qwen`. Read recorded review feedback before restarting rejected work.

## Objectives & Success Criteria

Deliver dashboard and device enum persistence and demonstrate FR-005, FR-006. All listed subtasks must have reviewable evidence.

## Context & Constraints

Read `../spec.md`, `../plan.md`, `../research.md`, `../tasks.md` and `../handoff.md` relative to this prompt. The user requested planning here and implementation on another device. Do not deploy or modify a live database as part of this package. Preserve SQLite and existing GraphQL/UI contracts. No ORM upgrade, chart redesign or automatic text/bytea migration.

## Branch Strategy

Planning/integration branch: `plan/postgres-dashboard`. Final reviewed PR destination: `master`. Honor the runtime workspace returned for this single-branch mission; do not create an untracked worktree or run several writers concurrently.

**Dependencies**: WP01.
**Requirement Refs**: FR-005, FR-006.

## Subtasks & Detailed Guidance

### T005 – Accept database string and byte values

Update DashboardType, Interval and DeviceType scanners to accept string and []byte. Preserve their Valuer representation. Define errors for nil and unsupported types after inspecting schema constraints. Add unit cases for all three scanners, including error inputs and value round trips.

**Owned surfaces**: `model/dashboard.go`, `model/device.go`, `model/dashboard_test.go`, `model/device_test.go`, `dashboard/postgres_persistence_test.go`, `device/postgres_persistence_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T006 – Prove compatibility with PostgreSQL text and bytea

Create test schema variants for normal text and legacy bytea columns used by task 114. Exercise inserts, updates and reads for dashboard entry type/interval and device type with the actual pq driver. If binding fails on bytea, fix the application binding boundary without automatic destructive schema conversion.

**Owned surfaces**: `model/dashboard.go`, `model/device.go`, `model/dashboard_test.go`, `model/device_test.go`, `dashboard/postgres_persistence_test.go`, `device/postgres_persistence_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T007 – Verify dashboard and device lifecycle persistence

Exercise create/edit/reopen of dashboard entries, chart types, intervals, named ranges, filters and layouts, plus device reads. Close/reopen database connections; use file-backed SQLite for persistence tests. Check persisted fields and other-user isolation after reconnect.

**Owned surfaces**: `model/dashboard.go`, `model/device.go`, `model/dashboard_test.go`, `model/device_test.go`, `dashboard/postgres_persistence_test.go`, `device/postgres_persistence_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T008 – Check startup migration behavior on existing schemas

Initialize both PostgreSQL schema variants with database.New and reopen them. Assert existing values/types are retained and surface migration failure in test setup. Include an application restart smoke procedure for the final runbook; reconnect alone must not be reported as process-restart evidence.

**Owned surfaces**: `model/dashboard.go`, `model/device.go`, `model/dashboard_test.go`, `model/device_test.go`, `dashboard/postgres_persistence_test.go`, `device/postgres_persistence_test.go`.

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
