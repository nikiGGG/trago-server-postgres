---
work_package_id: WP05
title: Fresh integrated verification and common evidence
dependencies:
- WP04
requirement_refs:
- C-003
- C-004
- FR-001
- FR-002
- FR-003
- FR-004
- FR-005
- FR-006
- NFR-001
- NFR-002
- NFR-003
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
base_branch: kitty/mission-postgres-dashboard-review-fixes-01M5ZJTF
base_commit: 6e4c888f7eca13cea44336b6f6a60b357bbd4710
created_at: '2026-10-27T22:29:58.770167+00:00'
subtasks:
- T020
- T021
- T022
- T023
- T024
history: []
agent_profile: implementer-ivan
authoritative_surface: docs/reviews/
create_intent:
- docs/reviews/postgres-dashboard-remediation-evidence-2026-10-28.md
execution_mode: code_change
owned_files:
- docs/reviews/postgres-dashboard-remediation-evidence-2026-10-28.md
role: implementer
tags: []
task_type: implement
tracker_refs: []
---

# WP05: Fresh integrated verification and common evidence

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Read any recorded rejection feedback before restarting.

## Objective and success criteria

Execute the complete verification contract on the remediation source revision and retain one secret-free evidence record for both reviews.

**Independent acceptance**: Evidence identifies tested commit/patch, actual PG17, zero skipped PG cases, real Docker bindings/cleanup, all required exits and full backend/frontend results; every claimed pass has retained execution support.

**Requirement refs**: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, NFR-001, NFR-002, NFR-003, C-003, C-004.

## Context and constraints

Read `kitty-specs/postgres-dashboard-review-fixes-01M5ZJTF/spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/remediation.md`, `quickstart.md` and `tasks.md` in that same mission directory. All implementation paths below are repository-root-relative.

Preserve GraphQL/UI fields, integer-bound behavior, local-time semantics, per-span rounding, filtering, zero fill and SQLite/PostgreSQL text/bytea persistence. No dependency/ORM upgrade, destructive conversion, production access, deployment, image publication or external manifest edits. Keep task 114 open.

## Branch strategy and dependencies

**Planning/base**: `plan/postgres-dashboard`. **Final merge target**: `plan/postgres-dashboard`.

**Depends on**: WP04. Use the runtime checkout/lane from `lanes.json`; this is a single-branch mission, so execute sequentially and do not allocate an ad-hoc worktree.

After `/spec-kitty.analyze` has persisted its report, claim through `spec-kitty agent action implement WP05 --agent codex --mission postgres-dashboard-review-fixes-01M5ZJTF`. Task authoring does not claim or implement this WP.

## Ownership and integration contract

This is an execution/evidence package, not permission to rewrite sibling source. Route any discovered code defect to the owning package and rerun affected checks. It is independently reviewable by inspecting logs and reproducing commands.

- `docs/reviews/postgres-dashboard-remediation-evidence-2026-10-28.md` (planned new file)

Sibling owned files are read-only here. A discovered sibling defect returns to its owning WP; do not conceal it with evidence edits. Record any narrowly necessary ownership amendment before proceeding.

## Subtasks and detailed guidance

### T020 — Prepare the actual verification environment and identify source

**Deliverable**: Prepare the actual verification environment and identify source.

**Implementation**

Create the common evidence file. Record git commit, dirty patch identity if applicable, timestamp, Go/CGO/compiler/Docker/Node/Yarn/gqlgen/goimports versions and sanitized log references.

**Boundaries and edge cases**

Run make download-tools, make install, make generate and NODE_OPTIONS=--openssl-legacy-provider make build-js according to quickstart.md. Use real ui/build assets; retain dependency/lockfile constraints.

**Verification and done evidence**

Every preparation command has an exit record. Missing tools or failed preparation remain open; no stubs or historical approvals may replace execution.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T020. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T021 — Run fresh SQLite and regression discrimination checks

**Deliverable**: Run fresh SQLite and regression discrimination checks.

**Implementation**

With DSN absent run go test -count=1 ./model/... ./statistics/... ./dashboard/... ./device/... ./database/... ./test/... and focused GraphQL/startup tests. Run make test-database-launcher.

**Boundaries and edge cases**

Collect red/green regression records from WP01–WP03. If reproduction must be repeated, use an isolated scratch source/patch and never disturb the active branch or expose credentials.

**Verification and done evidence**

Record exact command exits, counts and optional PG skips separately as SQLite-only evidence. All four reviewed defects have observable discrimination evidence.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T021. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T022 — Execute actual PG17 and real launcher lifecycle checks

**Deliverable**: Execute actual PG17 and real launcher lifecycle checks.

**Implementation**

Run make test-postgres in required mode against actual major 17 and record server_version and server_version_num, package coverage, total/PG execution and zero PG skip counts.

**Boundaries and edge cases**

During default/custom-port self-launch runs inspect every Docker published HostIp; require only 127.0.0.1 and prove connectivity. Observe container removal after success and induced test failure.

**Verification and done evidence**

Record sanitized runtime observations and container lifecycle identities. Image tags and command-double results alone do not satisfy real binding, version or cleanup acceptance.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T022. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T023 — Record credential and required-mode negative checks

**Deliverable**: Record credential and required-mode negative checks.

**Implementation**

Capture both streams for URL and quoted-keyword synthetic sentinels across success, malformed/setup failure and test failure, including child credential echoes. Assert no sentinel or full credential DSN is present.

**Boundaries and edge cases**

Run direct required tests with no DSN, unusable loopback DSN with bounded timeout, launcher reuse failure and wrong actual major. Treat their expected nonzero exits as negative-check success.

**Verification and done evidence**

The record proves no green dedicated run without real PG17 execution. Do not store raw secret-bearing captures; retain sanitized assertions and log identities.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T023. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T024 — Complete full pipeline and freeze the evidence revision

**Deliverable**: Complete full pipeline and freeze the evidence revision.

**Implementation**

Run make lint and make test after preparation. Inspect goimports filenames even if the command exits zero. Record backend race/coverage and frontend outcomes explicitly.

**Boundaries and edge cases**

Resolve required failures through the owning WP, then repeat affected checks on final source. Identify the tested code commit and any docs-only successor rather than claiming an untested commit passed.

**Verification and done evidence**

All acceptance commands exit zero, formatting has no unresolved findings and all required evidence fields are populated. If an environment failure persists, leave this WP unapproved and relevant gaps open.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T024. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

## Verification strategy

Tests are explicitly required by FR-005/006. Run focused checks first, with generated sources and real UI assets where required. WP05 owns final integrated evidence. Use only synthetic, disposable fixtures.

For engine claims, record the actual connected version and execution counts. Optional PostgreSQL skips in a SQLite run prove nothing about PG17. The dedicated gate requires zero skipped PostgreSQL cases.

Capture reviewed-defect reproduction before applying each fix, then demonstrate the same behavior passes. For documentation packages, validate source-qualified evidence links and historical preservation instead of inventing new runtime claims.

## Risks and mitigations

Missing tools, generation, real frontend assets or server access are unresolved verification. Later code edits invalidate affected results; docs-only commits may cite the tested code revision explicitly.

Missing tools/services and failed required checks remain unresolved verification. Do not weaken assertions, change requirements or claim historical evidence as a fresh result.

## Definition of done and review guidance

Evidence identifies tested commit/patch, actual PG17, zero skipped PG cases, real Docker bindings/cleanup, all required exits and full backend/frontend results; every claimed pass has retained execution support.

Reviewer checks the complete requirement mapping, declared file ownership and every subtask oracle. Require evidence that exposes the defect, preserves compatibility and distinguishes intended negative checks from failed acceptance.

Do not mark the WP ready solely because a command exited zero if it skipped required cases or emitted unresolved formatting findings. Post-test source changes require affected checks to run again.

## Activity log

2026-10-28 (Europe/Samara): Codex authored this planning prompt. Implementation and executable acceptance have not started; task states belong to the canonical event log.
