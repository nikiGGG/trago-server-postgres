---
work_package_id: WP06
title: Release and deployment handoff
dependencies:
- WP05
requirement_refs:
- FR-008
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
subtasks: []
phase: Phase 6 - Release and deployment handoff
history: []
agent_profile: implementer-ivan
authoritative_surface: docs/
create_intent:
- docs/postgres-dashboard-rollout.md
execution_mode: code_change
owned_files:
- README.md
- docs/postgres-dashboard-rollout.md
role: implementer
tags: []
task_type: implement
tracker_refs: []
---
# Work Package Prompt: WP06 – Release and deployment handoff

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Agent/tool: `qwen`. Read recorded review feedback before restarting rejected work.

## Objectives & Success Criteria

Deliver release and deployment handoff and demonstrate FR-008. All listed subtasks must have reviewable evidence.

## Context & Constraints

Read `../spec.md`, `../plan.md`, `../research.md`, `../tasks.md` and `../handoff.md` relative to this prompt. The user requested planning here and implementation on another device. Do not deploy or modify a live database as part of this package. Preserve SQLite and existing GraphQL/UI contracts. No ORM upgrade, chart redesign or automatic text/bytea migration.

## Branch Strategy

Planning/integration branch: `plan/postgres-dashboard`. Final reviewed PR destination: `master`. Honor the runtime workspace returned for this single-branch mission; do not create an untracked worktree or run several writers concurrently.

**Dependencies**: WP05.
**Requirement Refs**: FR-008.

## Subtasks & Detailed Guidance

### T021 – Document supported database behavior and configuration

Document PostgreSQL and SQLite configuration, local user-time semantics and text/bytea compatibility. State that SQLite remains the default and no automatic column conversion is required. Include targeted verification and all relevant FR evidence links.

**Owned surfaces**: `README.md`, `docs/postgres-dashboard-rollout.md`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T022 – Prepare the immutable image release procedure

Reconcile checkout baseline with task 114 deployed revision before building. Document existing image build tooling, recording source commit and image digest, staging prerequisites and rollback image. Verify instructions without publishing or deploying as a side effect of documentation work.

**Owned surfaces**: `README.md`, `docs/postgres-dashboard-rollout.md`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T023 – Prepare the external Kubernetes manifest change procedure

Locate the owning automation repository on the implementation device and inspect automation-scripts/k8s-cluster/traggo.yaml plus workaround commit 65b29d3. Document a concrete image digest change and retention of existing bytea workaround. If repository access is absent, record the external follow-up clearly instead of inventing a successful manifest update.

**Owned surfaces**: `README.md`, `docs/postgres-dashboard-rollout.md`.

**Verification**: show the behavior described above using actual database results or a concrete documentation/configuration diff. Check each command exit status and fixture write error. Preserve useful failure output without secrets.

**Done evidence**: record changed files, exact checks, observed results and any external follow-up in the review handoff. Do not mark a check passed solely because the test skipped.

### T024 – Define operational acceptance and rollback evidence

Provide a staging/production checklist covering selected-week Daily table and StackedBarChart data, three Cyrillic series, source-duration totals, Include/Exclude, create/edit/reopen and persistence after process restart. Include SQL/Scan log checks and image rollback. Live rollout and external writes require subsequent explicit execution instructions; completing this package does not declare task 114 deployed.

**Owned surfaces**: `README.md`, `docs/postgres-dashboard-rollout.md`.

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
