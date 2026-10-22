---
work_package_id: WP05
title: Mandatory SQLite and PostgreSQL CI
dependencies:
- WP04
requirement_refs:
- FR-007
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
subtasks: []
phase: Phase 5 - Mandatory SQLite and PostgreSQL CI
history: []
agent_profile: implementer-ivan
authoritative_surface: .github/workflows/
create_intent:
- hack/test-databases.sh
execution_mode: code_change
owned_files:
- .github/workflows/build.yml
- Makefile
- hack/test-databases.sh
role: implementer
tags: []
task_type: implement
tracker_refs: []
---
# Work Package Prompt: WP05 – Mandatory SQLite and PostgreSQL CI

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Agent/tool: `qwen`. Read recorded review feedback before restarting rejected work.

## Objectives & Success Criteria

Deliver mandatory sqlite and postgresql ci and demonstrate FR-007. All listed subtasks must have reviewable evidence.

## Context & Constraints

Read `../spec.md`, `../plan.md`, `../research.md`, `../tasks.md` and `../handoff.md` relative to this prompt. The user requested planning here and implementation on another device. Do not deploy or modify a live database as part of this package. Preserve SQLite and existing GraphQL/UI contracts. No ORM upgrade, chart redesign or automatic text/bytea migration.

## Branch Strategy

Planning/integration branch: `plan/postgres-dashboard`. Final reviewed PR destination: `master`. Honor the runtime workspace returned for this single-branch mission; do not create an untracked worktree or run several writers concurrently.

**Dependencies**: WP04.
**Requirement Refs**: FR-007.

## Subtasks & Detailed Guidance

### T017 – Add a health-checked PostgreSQL 17 CI service

Add a dedicated CI job using a disposable PostgreSQL 17 service and explicit test credentials. Preserve the existing build pipeline and SQLite coverage. Generate GraphQL sources before backend tests; retain CGO support. Do not require cluster credentials or production networking.

**Owned surfaces**: `.github/workflows/build.yml`, `Makefile`, `hack/test-databases.sh`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T018 – Require actual PostgreSQL execution

Wire WP01 required-mode settings so the PostgreSQL job fails for unavailable connection, failed setup or skipped backend suites. Ensure text/bytea and acceptance suites execute. Do not permit a green job that ran SQLite only.

**Owned surfaces**: `.github/workflows/build.yml`, `Makefile`, `hack/test-databases.sh`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T019 – Provide repeatable local verification commands

Add a scoped Make target or hack script matching CI setup and test invocation. Keep existing make test behavior compatible; avoid installing unnecessary tool upgrades. Document env names and disposable database assumptions in the script.

**Owned surfaces**: `.github/workflows/build.yml`, `Makefile`, `hack/test-databases.sh`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T020 – Validate the complete required pipeline

Run targeted backend jobs and existing lint/test commands with generated sources and UI assets. Inspect job configuration and prove required-mode rejects an invalid PostgreSQL DSN. Record commands, versions and results; do not claim remote CI passed until it actually ran.

**Owned surfaces**: `.github/workflows/build.yml`, `Makefile`, `hack/test-databases.sh`.

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
