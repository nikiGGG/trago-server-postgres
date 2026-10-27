---
work_package_id: WP01
title: Safe disposable PostgreSQL launcher
dependencies: []
requirement_refs:
- C-003
- C-004
- FR-001
- FR-002
- FR-005
- NFR-001
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
base_branch: kitty/mission-postgres-dashboard-review-fixes-01M5ZJTF
base_commit: 6e4c888f7eca13cea44336b6f6a60b357bbd4710
created_at: '2026-10-27T22:15:15.455062+00:00'
subtasks:
- T001
- T002
- T003
- T004
- T005
history: []
agent_profile: implementer-ivan
authoritative_surface: hack/
create_intent:
- hack/test-databases_test.sh
execution_mode: code_change
owned_files:
- hack/test-databases.sh
- hack/test-databases_test.sh
role: implementer
tags: []
task_type: implement
tracker_refs: []
---

# WP01: Safe disposable PostgreSQL launcher

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Read any recorded rejection feedback before restarting.

## Objective and success criteria

Make the local disposable launcher enforce loopback publishing, credential-safe output and actual PostgreSQL 17 execution.

**Independent acceptance**: Executable shell regressions pass for default/custom ports, URL/quoted keyword DSNs, setup/test failures, wrong major and cleanup; emitted sentinels never appear in captured output.

**Requirement refs**: FR-001, FR-002, FR-005, NFR-001, C-003, C-004.

## Context and constraints

Read `kitty-specs/postgres-dashboard-review-fixes-01M5ZJTF/spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/remediation.md`, `quickstart.md` and `tasks.md` in that same mission directory. All implementation paths below are repository-root-relative.

Preserve GraphQL/UI fields, integer-bound behavior, local-time semantics, per-span rounding, filtering, zero fill and SQLite/PostgreSQL text/bytea persistence. No dependency/ORM upgrade, destructive conversion, production access, deployment, image publication or external manifest edits. Keep task 114 open.

## Branch strategy and dependencies

**Planning/base**: `plan/postgres-dashboard`. **Final merge target**: `plan/postgres-dashboard`.

**Depends on**: None. Use the runtime checkout/lane from `lanes.json`; this is a single-branch mission, so execute sequentially and do not allocate an ad-hoc worktree.

After `/spec-kitty.analyze` has persisted its report, claim through `spec-kitty agent action implement WP01 --agent codex --mission postgres-dashboard-review-fixes-01M5ZJTF`. Task authoring does not claim or implement this WP.

## Ownership and integration contract

Own launcher behavior and shell tests together. WP04 owns Makefile/workflow wiring. Own the complete default package list and JSON/environment interface; WP04 wires the targets and result checker. Leave production migration helpers to WP03.

- `hack/test-databases.sh`
- `hack/test-databases_test.sh` (planned new file)

Sibling owned files are read-only here. A discovered sibling defect returns to its owning WP; do not conceal it with evidence edits. Record any narrowly necessary ownership amendment before proceeding.

## Subtasks and detailed guidance

### T001 — Reproduce publishing and credential leaks with command doubles

**Deliverable**: Reproduce publishing and credential leaks with command doubles.

**Implementation**

Create hack/test-databases_test.sh using an isolated PATH and synthetic docker/go commands. Capture both streams and exit status without printing sentinel values. Exercise the unmodified launcher to show regression assertions distinguish the reviewed behavior.

**Boundaries and edge cases**

Include default port, custom port, reused URL DSN, quoted keyword password with spaces/escapes, malformed input, setup failure and child-emitted credentials on successful and failing runs.

**Verification and done evidence**

At least the missing HostIp and DSN-output assertions fail against reviewed source; the test harness itself exits nonzero when an assertion fails.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T001. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T002 — Enforce loopback publishing and preserve lifecycle behavior

**Deliverable**: Enforce loopback publishing and preserve lifecycle behavior.

**Implementation**

Change publishing to 127.0.0.1:${PORT}:5432. Retain custom port configuration, health checks, supplied-DSN reuse and cleanup trap. Avoid Docker service assumptions in launcher comments.

**Boundaries and edge cases**

Do not create a container on reused-DSN paths. Cleanup must run after ready failure, failed tests and successful tests, preserving the original exit status.

**Verification and done evidence**

Doubles assert exact publish arguments and one cleanup for self-started containers; no cleanup targets an externally supplied server.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T002. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T003 — Remove credentials from launcher-controlled output

**Deliverable**: Remove credentials from launcher-controlled output.

**Implementation**

Remove the full DSN echo and pass connection settings through environment. Prevent secret-bearing child errors at source where possible. If replay requires capture/redaction, keep captures private and remove them on every exit.

**Boundaries and edge cases**

Handle both URL userinfo and lib/pq quoted keyword values. Preserve useful fixed stage/failure messages. Never enable tracing or put raw credentials into evidence.

**Verification and done evidence**

All success/setup/test-failure cases capture zero sentinel-password or complete credential-bearing DSN occurrences across stdout and stderr, including child output; original failure exits survive.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T003. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T004 — Require the connected server to be major 17

**Deliverable**: Require the connected server to be major 17.

**Implementation**

Check actual server_version_num through the connected server on both disposable and reused paths. Default image postgres:17 is not sufficient. Keep the pinned driver-compatible disposable authentication boundary.

**Boundaries and edge cases**

Use existing dependencies/tools; if a helper is needed, keep it within the owned launcher/test files. Reject wrong major before treating fixtures as accepted execution.

**Verification and done evidence**

Synthetic version responses show 17 accepted and other majors rejected; actual-version verification remains a required WP05 integration check. Support TRAGGO_TEST_PG_PACKAGES (space-separated paths; default includes all six packages) and TRAGGO_TEST_GO_JSON=1 for WP04; JSON mode keeps child result events parseable and sends fixed stage messages to stderr.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T004. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T005 — Run and document focused launcher checks

**Deliverable**: Run and document focused launcher checks.

**Implementation**

Run bash -n on both owned scripts and execute bash hack/test-databases_test.sh. Keep execution deterministic without Docker by using doubles; retain red/green evidence in the review handoff.

**Boundaries and edge cases**

Check executable modes if direct execution is used. Verify temporary captures and doubles are removed after induced failures. Do not edit CI or the Makefile in this package.

**Verification and done evidence**

All assertions pass after fixes; failure-injection assertions still reject a launcher that reintroduces broad publishing or credential logging.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T005. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

## Verification strategy

Tests are explicitly required by FR-005/006. Run focused checks first, with generated sources and real UI assets where required. WP05 owns final integrated evidence. Use only synthetic, disposable fixtures.

For engine claims, record the actual connected version and execution counts. Optional PostgreSQL skips in a SQLite run prove nothing about PG17. The dedicated gate requires zero skipped PostgreSQL cases.

Capture reviewed-defect reproduction before applying each fix, then demonstrate the same behavior passes. For documentation packages, validate source-qualified evidence links and historical preservation instead of inventing new runtime claims.

## Risks and mitigations

Quoted lib/pq passwords cannot be redacted by whitespace splitting; command doubles must not substitute for the real Docker checks in WP05.

Missing tools/services and failed required checks remain unresolved verification. Do not weaken assertions, change requirements or claim historical evidence as a fresh result.

## Definition of done and review guidance

Executable shell regressions pass for default/custom ports, URL/quoted keyword DSNs, setup/test failures, wrong major and cleanup; emitted sentinels never appear in captured output.

Reviewer checks the complete requirement mapping, declared file ownership and every subtask oracle. Require evidence that exposes the defect, preserves compatibility and distinguishes intended negative checks from failed acceptance.

Do not mark the WP ready solely because a command exited zero if it skipped required cases or emitted unresolved formatting findings. Post-test source changes require affected checks to run again.

## Activity log

2026-10-28 (Europe/Samara): Codex authored this planning prompt. Implementation and executable acceptance have not started; task states belong to the canonical event log.
