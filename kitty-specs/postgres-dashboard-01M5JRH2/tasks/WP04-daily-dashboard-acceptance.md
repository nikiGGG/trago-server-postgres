---
work_package_id: WP04
title: Daily statistics and dashboard acceptance
dependencies:
- WP02
- WP03
requirement_refs:
- FR-001
- FR-002
- FR-003
- FR-006
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
base_branch: kitty/mission-postgres-dashboard-01M5JRH2
base_commit: 75ae7a88927253743c62f0629bad6d8a7df14721
created_at: '2026-10-25T13:39:17.599208+00:00'
subtasks: []
phase: Phase 4 - Daily statistics and dashboard acceptance
history: []
agent_profile: implementer-ivan
authoritative_surface: statistics/
create_intent:
- statistics/summary2_test.go
- dashboard/postgres_acceptance_test.go
execution_mode: code_change
owned_files:
- statistics/summary2_test.go
- dashboard/postgres_acceptance_test.go
role: implementer
tags: []
task_type: implement
tracker_refs: []
---
# Work Package Prompt: WP04 – Daily statistics and dashboard acceptance

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Agent/tool: `qwen`. Read recorded review feedback before restarting rejected work.

## Objectives & Success Criteria

Deliver daily statistics and dashboard acceptance and demonstrate FR-001, FR-002, FR-003, FR-006. All listed subtasks must have reviewable evidence.

## Context & Constraints

Read `../spec.md`, `../plan.md`, `../research.md`, `../tasks.md` and `../handoff.md` relative to this prompt. The user requested planning here and implementation on another device. Do not deploy or modify a live database as part of this package. Preserve SQLite and existing GraphQL/UI contracts. No ORM upgrade, chart redesign or automatic text/bytea migration.

## Branch Strategy

Planning/integration branch: `plan/postgres-dashboard`. Final reviewed PR destination: `master`. Honor the runtime workspace returned for this single-branch mission; do not create an untracked worktree or run several writers concurrently.

**Dependencies**: WP02, WP03.
**Requirement Refs**: FR-001, FR-002, FR-003, FR-006.

## Subtasks & Detailed Guidance

### T013 – Author the deterministic Cyrillic weekly fixture

Seed synthetic work tags работа=дом, работа=офис and работа=подработка across a fixed week, including an empty day and a completed span crossing midnight. Calculate and document expected daily/series totals independently from SQL. Avoid copying private deployment records.

**Owned surfaces**: `statistics/summary2_test.go`, `dashboard/postgres_acceptance_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T014 – Verify Stats2 Daily ranges on both engines

Use fixed now and explicit week settings to call Stats2 with Daily. Assert range boundaries, values, zero filling, Include/Exclude and weekly totals. Preserve the existing second-inclusive range builder: distinguish its outcome from manually supplied adjacent midnight bounds. Test non-UTC input offsets and PostgreSQL session TimeZone changes.

**Owned surfaces**: `statistics/summary2_test.go`, `dashboard/postgres_acceptance_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T015 – Verify dashboard API persistence and chart selections

Test resolver/API operations preserving Table and StackedBarChart configuration, Daily interval, named ranges and filters across reconnect. Confirm both stats endpoints retain their GraphQL shapes. Use shared fixtures from WP01 and enum compatibility from WP02; add no UI redesign or new fields.

**Owned surfaces**: `statistics/summary2_test.go`, `dashboard/postgres_acceptance_test.go`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T016 – Record acceptance and regression evidence

Run targeted statistics/dashboard/device/model suites on both backends. Record any actual driver or timestamp limitations and fix acceptance failures in owned files; coordinate changes to WP03-owned SQL through a reviewed follow-up. Provide a repeatable manual chart/restart check for WP06.

**Owned surfaces**: `statistics/summary2_test.go`, `dashboard/postgres_acceptance_test.go`.

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
