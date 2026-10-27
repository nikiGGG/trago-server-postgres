# Work packages: PostgreSQL dashboard review remediation

**Mission**: `postgres-dashboard-review-fixes-01M5ZJTF`. **Date**: 2026-10-28 (Europe/Samara).

**Inputs**: spec.md, plan.md, research.md, data-model.md, contracts/remediation.md and quickstart.md in this mission directory. Tests are explicitly required. No implementation or fresh acceptance is claimed.

## Decomposition and execution contract

IC-01 → WP01; IC-02 → WP02; IC-03 → WP03; IC-04 → WP04 (gate wiring) and WP05 (execution/evidence); IC-05 → WP06. This separates code ownership from evidence collection and historical report reconciliation. WP06 uses execution_mode planning_artifact because its owned deliverables are confined to docs/ and kitty-specs/; WP01–WP05 use code_change. Prompts stay concise (3–5 concrete subtasks each), below the preferred 200-line target without merging unrelated ownership domains.

WP01, WP02 and WP03 have no logical dependencies and disjoint ownership. Execute sequentially for the canonical single-branch topology: WP01 → WP02 → WP03 → WP04 → WP05 → WP06. WP04 depends on all three fixes; WP05 depends on WP04; WP06 depends on verified WP05 evidence.

Planning/base and final merge target: `plan/postgres-dashboard`. No agents or worktrees are created by task generation. Run `/spec-kitty.analyze` before implementation; follow canonical runtime allocation afterward.

Completion is event-sourced. Txxx rows below are references, not checkboxes. Use `spec-kitty agent tasks mark-status <Txxx> --status done --mission postgres-dashboard-review-fixes-01M5ZJTF` during execution.

Launcher/CI interface: WP01 owns complete default PG packages, TRAGGO_TEST_PG_PACKAGES override and TRAGGO_TEST_GO_JSON=1; JSON test events stay parseable, stage messages use stderr. WP04 owns Makefile/workflow and the skip/count result checker. No file is owned by both packages.

## Work Package WP01: Safe disposable PostgreSQL launcher (Priority: P1)

**Goal**: Make the local disposable launcher enforce loopback publishing, credential-safe output and actual PostgreSQL 17 execution.

**Independent Test**: Executable shell regressions pass for default/custom ports, URL/quoted keyword DSNs, setup/test failures, wrong major and cleanup; emitted sentinels never appear in captured output.

**Prompt**: [tasks/WP01-safe-postgres-launcher.md](tasks/WP01-safe-postgres-launcher.md).

**Requirement Refs**: FR-001, FR-002, FR-005, NFR-001, C-003, C-004.

### Included Subtasks

T001 Reproduce publishing and credential leaks with command doubles (WP01)

T002 Enforce loopback publishing and preserve lifecycle behavior (WP01)

T003 Remove credentials from launcher-controlled output (WP01)

T004 Require the connected server to be major 17 (WP01)

T005 Run and document focused launcher checks (WP01)


### Implementation Notes

Own launcher behavior and shell tests together. WP04 owns Makefile/workflow wiring. Own the complete default package list and JSON/environment interface; WP04 wires the targets and result checker. Leave production migration helpers to WP03.

### Dependencies

Depends on: None.

### Parallel Opportunities

Disjoint root package; logically independent of the other roots, but use sequential execution on the shared branch.

### Risks and Mitigations

Quoted lib/pq passwords cannot be redacted by whitespace splitting; command doubles must not substitute for the real Docker checks in WP05.

**Prompt size**: 189 lines; 5 subtasks.

## Work Package WP02: Lossless fractional statistics and GraphQL bounds (Priority: P1)

**Goal**: Decouple aggregate identity from database timestamp formatting and preserve accepted precision through the serialized GraphQL response.

**Independent Test**: Shared SQLite/PG17 fixtures return independent expected seconds and original bounds; scalar and actual GraphQL response assertions preserve nanoseconds, and existing integer/filter/timezone cases remain passing.

**Prompt**: [tasks/WP02-fractional-range-identity.md](tasks/WP02-fractional-range-identity.md).

**Requirement Refs**: FR-003, FR-005, C-001.

### Included Subtasks

T006 Add shared fractional-range reproduction fixtures (WP02)

T007 Group aggregate results by parameterized integer range identity (WP02)

T008 Preserve accepted precision in scalar and wire serialization (WP02)

T009 Protect Stats2 and existing compatibility semantics (WP02)

T010 Verify focused regression coverage on both engines (WP02)


### Implementation Notes

Keep query/group reconstruction, scalar serialization and their regression tests in one package: resolver-only fixes cannot satisfy the wire contract. Do not rewrite Stats2 range generation or touch schema.graphql.

### Dependencies

Depends on: None.

### Parallel Opportunities

Disjoint root package; logically independent of the other roots, but use sequential execution on the shared branch.

### Risks and Mitigations

PostgreSQL stores less precision than accepted timestamps; exact identity must use input position while arithmetic retains native DB precision. Duplicate requests must not multiply aggregate seconds.

**Prompt size**: 193 lines; 5 subtasks.

## Work Package WP03: Production migration failure and startup resource lifecycle (Priority: P1)

**Goal**: Return real AutoMigrate failures from production New, close the owned pool and prevent successful startup while preserving healthy SQLite/text/bytea initialization.

**Independent Test**: Actual SQLite and PG17 schema conflicts after successful connection cause nil/error from public New; retained owned-pool Ping fails, cleanup failure preserves migration cause, and startup subprocess exits before success.

**Prompt**: [tasks/WP03-migration-failure-lifecycle.md](tasks/WP03-migration-failure-lifecycle.md).

**Requirement Refs**: FR-004, FR-005, C-002, C-003.

### Included Subtasks

T011 Create real post-connect migration-conflict fixtures (WP03)

T012 Propagate migration errors and close the owned pool (WP03)

T013 Prove pool cleanup and migration-cause priority (WP03)

T014 Exercise startup fatal behavior in a subprocess (WP03)

T015 Make the harness defer migration ownership to production New (WP03)


### Implementation Notes

Own production initialization and its harness integration together. Main production fatal boundary already exists; test it without broad main.go refactoring. Keep enum persistence implementations unchanged.

### Dependencies

Depends on: None.

### Parallel Opportunities

Disjoint root package; logically independent of the other roots, but use sequential execution on the shared branch.

### Risks and Mitigations

A view named users may be skipped by GORM introspection; prove that the failure happens during AutoMigrate. Internal database tests importing test would form a cycle.

**Prompt size**: 199 lines; 5 subtasks.

## Work Package WP04: Mandatory PostgreSQL and launcher CI gates (Priority: P1)

**Goal**: Run the same safe launcher in CI, include model/database regressions and reject skipped PostgreSQL tests in the dedicated gate.

**Independent Test**: CI invokes explicit shell-regression and test-postgres targets, covers all six packages, and a gate result checker rejects injected PG skips or zero PG execution.

**Prompt**: [tasks/WP04-mandatory-database-gates.md](tasks/WP04-mandatory-database-gates.md).

**Requirement Refs**: FR-001, FR-005, FR-006, NFR-002, C-004.

### Included Subtasks

T016 Wire explicit regression and complete database targets (WP04)

T017 Replace CI service-container setup with the safe launcher (WP04)

T018 Reject skipped PostgreSQL execution from JSON results (WP04)

T019 Verify gate failure semantics and focused integration (WP04)


### Implementation Notes

WP01 owns launcher files. Wire expanded packages/output through environment or launcher arguments supported by WP01; coordinate this interface in the task plan, not by overlapping ownership. The agreed launcher interface is TRAGGO_TEST_PG_PACKAGES (space-separated package paths) and TRAGGO_TEST_GO_JSON=1.

### Dependencies

Depends on: WP01, WP02, WP03.

### Parallel Opportunities

Wait for dependency evidence; do not run concurrent writers on this single-branch mission.

### Risks and Mitigations

go test success alone does not prove PostgreSQL cases executed. Skip detection must use test-result events and recognize nested PostgreSQL subtests without treating unrelated SQLite runs as PG evidence.

**Prompt size**: 177 lines; 4 subtasks.

## Work Package WP05: Fresh integrated verification and common evidence (Priority: P1)

**Goal**: Execute the complete verification contract on the remediation source revision and retain one secret-free evidence record for both reviews.

**Independent Test**: Evidence identifies tested commit/patch, actual PG17, zero skipped PG cases, real Docker bindings/cleanup, all required exits and full backend/frontend results; every claimed pass has retained execution support.

**Prompt**: [tasks/WP05-revision-bound-verification.md](tasks/WP05-revision-bound-verification.md).

**Requirement Refs**: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, NFR-001, NFR-002, NFR-003, C-003, C-004.

### Included Subtasks

T020 Prepare the actual verification environment and identify source (WP05)

T021 Run fresh SQLite and regression discrimination checks (WP05)

T022 Execute actual PG17 and real launcher lifecycle checks (WP05)

T023 Record credential and required-mode negative checks (WP05)

T024 Complete full pipeline and freeze the evidence revision (WP05)


### Implementation Notes

This is an execution/evidence package, not permission to rewrite sibling source. Route any discovered code defect to the owning package and rerun affected checks. It is independently reviewable by inspecting logs and reproducing commands.

### Dependencies

Depends on: WP04.

### Parallel Opportunities

Wait for dependency evidence; do not run concurrent writers on this single-branch mission.

### Risks and Mitigations

Missing tools, generation, real frontend assets or server access are unresolved verification. Later code edits invalidate affected results; docs-only commits may cite the tested code revision explicitly.

**Prompt size**: 193 lines; 5 subtasks.

## Work Package WP06: Reconcile historical reviews and acceptance provenance (Priority: P2)

**Goal**: Append consistent evidence-backed dispositions to both source reports and correct affected acceptance attribution without rewriting historical results.

**Independent Test**: Every source-local finding maps to the same WP05 evidence and scoped verdict; historical verdicts remain readable, provenance follows actual events and operational follow-ups/task 114 remain open.

**Prompt**: [tasks/WP06-review-reconciliation.md](tasks/WP06-review-reconciliation.md).

**Requirement Refs**: FR-007, NFR-003, C-003, C-004.

### Included Subtasks

T025 Audit source-local finding and acceptance provenance (WP06)

T026 Append dated reconciliation addenda to both reviews (WP06)

T027 Correct acceptance provenance and preserve operational follow-ups (WP06)


### Implementation Notes

Documentation reconciliation owns report/acceptance files only. Read status.events.jsonl and prior review cycles for provenance; never rewrite the event log to create a desired approval history.

### Dependencies

Depends on: WP05.

### Parallel Opportunities

Wait for dependency evidence; do not run concurrent writers on this single-branch mission.

### Risks and Mitigations

Finding IDs collide across reports. Recorded tool/user names are not proof of independent human review, and a configured image or historical PG14/18 run cannot establish PG17 acceptance.

**Prompt size**: 154 lines; 3 subtasks.

## Requirements Coverage Summary

| Requirement ID | Covered By Work Package(s) |
| --- | --- |
| FR-001 | WP01, WP04, WP05 |
| FR-002 | WP01, WP05 |
| FR-003 | WP02, WP05 |
| FR-004 | WP03, WP05 |
| FR-005 | WP01, WP02, WP03, WP04, WP05 |
| FR-006 | WP04, WP05 |
| FR-007 | WP06 |
| NFR-001 | WP01, WP05 |
| NFR-002 | WP04, WP05 |
| NFR-003 | WP05, WP06 |
| C-001 | WP02 |
| C-002 | WP03 |
| C-003 | WP01, WP03, WP05, WP06 |
| C-004 | WP01, WP04, WP05, WP06 |

## Subtask Index (Reference)

| Subtask ID | Deliverable | Work Package |
| --- | --- | --- |
| T001 | Reproduce publishing and credential leaks with command doubles | WP01 |
| T002 | Enforce loopback publishing and preserve lifecycle behavior | WP01 |
| T003 | Remove credentials from launcher-controlled output | WP01 |
| T004 | Require the connected server to be major 17 | WP01 |
| T005 | Run and document focused launcher checks | WP01 |
| T006 | Add shared fractional-range reproduction fixtures | WP02 |
| T007 | Group aggregate results by parameterized integer range identity | WP02 |
| T008 | Preserve accepted precision in scalar and wire serialization | WP02 |
| T009 | Protect Stats2 and existing compatibility semantics | WP02 |
| T010 | Verify focused regression coverage on both engines | WP02 |
| T011 | Create real post-connect migration-conflict fixtures | WP03 |
| T012 | Propagate migration errors and close the owned pool | WP03 |
| T013 | Prove pool cleanup and migration-cause priority | WP03 |
| T014 | Exercise startup fatal behavior in a subprocess | WP03 |
| T015 | Make the harness defer migration ownership to production New | WP03 |
| T016 | Wire explicit regression and complete database targets | WP04 |
| T017 | Replace CI service-container setup with the safe launcher | WP04 |
| T018 | Reject skipped PostgreSQL execution from JSON results | WP04 |
| T019 | Verify gate failure semantics and focused integration | WP04 |
| T020 | Prepare the actual verification environment and identify source | WP05 |
| T021 | Run fresh SQLite and regression discrimination checks | WP05 |
| T022 | Execute actual PG17 and real launcher lifecycle checks | WP05 |
| T023 | Record credential and required-mode negative checks | WP05 |
| T024 | Complete full pipeline and freeze the evidence revision | WP05 |
| T025 | Audit source-local finding and acceptance provenance | WP06 |
| T026 | Append dated reconciliation addenda to both reviews | WP06 |
| T027 | Correct acceptance provenance and preserve operational follow-ups | WP06 |

## Completion boundary

Six packages, 27 subtasks. WP01–WP03 deliver the four scoped fixes and regressions; WP04 makes coverage mandatory; WP05 produces fresh acceptance evidence; WP06 reconciles the reports. No smaller subset establishes complete remediation acceptance.

Both reviews must retain historical verdicts and add source-qualified dispositions using the same revision-bound evidence. Required failures or missing tools/services keep the relevant gap open. F-5 hardening and F-6 operational delivery stay deferred; task 114 stays open.

Charter action context loaded for tasks: project charter absent; no governance setup introduced. Built-in decision traceability, specification fidelity and locality applied through explicit boundaries, requirement mappings and observable acceptance criteria.
