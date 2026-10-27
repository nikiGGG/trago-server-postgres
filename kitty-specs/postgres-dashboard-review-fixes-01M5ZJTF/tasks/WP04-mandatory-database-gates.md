---
work_package_id: WP04
title: Mandatory PostgreSQL and launcher CI gates
dependencies:
- WP01
- WP02
- WP03
requirement_refs:
- C-004
- FR-001
- FR-005
- FR-006
- NFR-002
planning_base_branch: plan/postgres-dashboard
merge_target_branch: plan/postgres-dashboard
branch_strategy: Planning artifacts for this mission were generated on plan/postgres-dashboard. During /spec-kitty.implement this WP may branch from a dependency-specific base, but completed changes must merge back into plan/postgres-dashboard unless the human explicitly redirects the landing branch.
subtasks:
- T016
- T017
- T018
- T019
history: []
agent_profile: implementer-ivan
authoritative_surface: .github/workflows/
create_intent:
- hack/check-postgres-results.go
- hack/check-postgres-results_test.go
execution_mode: code_change
owned_files:
- Makefile
- .github/workflows/build.yml
- hack/check-postgres-results.go
- hack/check-postgres-results_test.go
role: implementer
tags: []
task_type: implement
tracker_refs: []
---

# WP04: Mandatory PostgreSQL and launcher CI gates

## ⚡ Do This First: Load Agent Profile

Use the `spk-doctrine-profile-load` skill (legacy `/ad-hoc-profile-load`) to load `implementer-ivan` before implementation. Read any recorded rejection feedback before restarting.

## Objective and success criteria

Run the same safe launcher in CI, include model/database regressions and reject skipped PostgreSQL tests in the dedicated gate.

**Independent acceptance**: CI invokes explicit shell-regression and test-postgres targets, covers all six packages, and a gate result checker rejects injected PG skips or zero PG execution.

**Requirement refs**: FR-001, FR-005, FR-006, NFR-002, C-004.

## Context and constraints

Read `kitty-specs/postgres-dashboard-review-fixes-01M5ZJTF/spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/remediation.md`, `quickstart.md` and `tasks.md` in that same mission directory. All implementation paths below are repository-root-relative.

Preserve GraphQL/UI fields, integer-bound behavior, local-time semantics, per-span rounding, filtering, zero fill and SQLite/PostgreSQL text/bytea persistence. No dependency/ORM upgrade, destructive conversion, production access, deployment, image publication or external manifest edits. Keep task 114 open.

## Branch strategy and dependencies

**Planning/base**: `plan/postgres-dashboard`. **Final merge target**: `plan/postgres-dashboard`.

**Depends on**: WP01, WP02, WP03. Use the runtime checkout/lane from `lanes.json`; this is a single-branch mission, so execute sequentially and do not allocate an ad-hoc worktree.

After `/spec-kitty.analyze` has persisted its report, claim through `spec-kitty agent action implement WP04 --agent codex --mission postgres-dashboard-review-fixes-01M5ZJTF`. Task authoring does not claim or implement this WP.

## Ownership and integration contract

WP01 owns launcher files. Wire expanded packages/output through environment or launcher arguments supported by WP01; coordinate this interface in the task plan, not by overlapping ownership. The agreed launcher interface is TRAGGO_TEST_PG_PACKAGES (space-separated package paths) and TRAGGO_TEST_GO_JSON=1.

- `Makefile`
- `.github/workflows/build.yml`
- `hack/check-postgres-results.go` (planned new file)
- `hack/check-postgres-results_test.go` (planned new file)

Sibling owned files are read-only here. A discovered sibling defect returns to its owning WP; do not conceal it with evidence edits. Record any narrowly necessary ownership amendment before proceeding.

## Subtasks and detailed guidance

### T016 — Wire explicit regression and complete database targets

**Deliverable**: Wire explicit regression and complete database targets.

**Implementation**

Add make test-database-launcher to execute bash hack/test-databases_test.sh. Expand mandatory PG package invocation to statistics, device, dashboard, test, database and model.

**Boundaries and edge cases**

Have Makefile pass TRAGGO_TEST_PG_PACKAGES and TRAGGO_TEST_GO_JSON=1 to the launcher. Launcher defaults must include the same complete set (WP01 owns those defaults). Keep one source of truth where practical.

**Verification and done evidence**

make target expansion shows database/model included and the shell suite is explicitly runnable without a server. Existing build/test/release targets retain behavior.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T016. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T017 — Replace CI service-container setup with the safe launcher

**Deliverable**: Replace CI service-container setup with the safe launcher.

**Implementation**

In postgres-test remove the PostgreSQL services block and hardcoded service DSN. Use the Ubuntu host Docker runtime and make test-postgres so local and CI share publishing, readiness/version checks and cleanup.

**Boundaries and edge cases**

Run the shell-regression target in CI. Retain Go tool installation/module preparation/GraphQL generation and any real assets required by included packages. Preserve separate full build/release behavior.

**Verification and done evidence**

No independent broad port mapping remains in the workflow. There is no supplied service DSN that silently bypasses the self-launch integration path.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T017. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T018 — Reject skipped PostgreSQL execution from JSON results

**Deliverable**: Reject skipped PostgreSQL execution from JSON results.

**Implementation**

Add a standard-library Go result checker in hack/check-postgres-results.go, invoked by the Makefile on sanitized launcher JSON results. Add meaningful result-stream tests in the paired test file.

**Boundaries and edge cases**

Recognize package/test/subtest PG identities based on actual suite naming. Fail on PG skip, no executed PG cases, malformed/truncated event stream or failed command; preserve nonzero test status and readable counts.

**Verification and done evidence**

Synthetic streams demonstrate pass, PG skip, zero PG, failed test and malformed results. Dedicated output reports PG execution/skip counts without exposing connection secrets.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T018. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

### T019 — Verify gate failure semantics and focused integration

**Deliverable**: Verify gate failure semantics and focused integration.

**Implementation**

Run make test-database-launcher and focused checker tests. Check direct required-mode tests with DSN absent and an unusable synthetic loopback DSN with bounded timeout; both must fail.

**Boundaries and edge cases**

Check launcher reuse with bad DSN and actual wrong major also fails. Absence of DSN in the launcher intentionally provisions a server and is not the missing-DSN negative test.

**Verification and done evidence**

Separate intended negative exits from passing acceptance commands. Fresh actual PG17/full pipeline results are recorded by WP05; gate wiring alone does not close FR-006.

Record the changed files, exact sanitized command, exit and applicable test/skip counts for T019. Completion is recorded through the Spec Kitty event log, not markdown checkboxes.

## Verification strategy

Tests are explicitly required by FR-005/006. Run focused checks first, with generated sources and real UI assets where required. WP05 owns final integrated evidence. Use only synthetic, disposable fixtures.

For engine claims, record the actual connected version and execution counts. Optional PostgreSQL skips in a SQLite run prove nothing about PG17. The dedicated gate requires zero skipped PostgreSQL cases.

Capture reviewed-defect reproduction before applying each fix, then demonstrate the same behavior passes. For documentation packages, validate source-qualified evidence links and historical preservation instead of inventing new runtime claims.

## Risks and mitigations

go test success alone does not prove PostgreSQL cases executed. Skip detection must use test-result events and recognize nested PostgreSQL subtests without treating unrelated SQLite runs as PG evidence.

Missing tools/services and failed required checks remain unresolved verification. Do not weaken assertions, change requirements or claim historical evidence as a fresh result.

## Definition of done and review guidance

CI invokes explicit shell-regression and test-postgres targets, covers all six packages, and a gate result checker rejects injected PG skips or zero PG execution.

Reviewer checks the complete requirement mapping, declared file ownership and every subtask oracle. Require evidence that exposes the defect, preserves compatibility and distinguishes intended negative checks from failed acceptance.

Do not mark the WP ready solely because a command exited zero if it skipped required cases or emitted unresolved formatting findings. Post-test source changes require affected checks to run again.

## Activity log

2026-10-28 (Europe/Samara): Codex authored this planning prompt. Implementation and executable acceptance have not started; task states belong to the canonical event log.
