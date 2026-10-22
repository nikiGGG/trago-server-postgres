# Tasks: PostgreSQL dashboards with SQLite compatibility

Implementation host: another device, using Qwen CLI. All work packages are initially unstarted. See `handoff.md` before claiming work. Tests are explicitly required by task 114 and FR-007.

## Execution order

WP01 → WP02 and WP03 → WP04 → WP05 → WP06. WP02 and WP03 are logically independent after WP01, but use sequential execution for this single-branch mission. The integration branch is `plan/postgres-dashboard`; eventual PR destination is `master`. No agents are launched by this planning work.

## Work Package WP01: Cross-database fixtures and failure reproduction

**Goal**: Cross-database fixtures and failure reproduction. **Priority**: P1. **Dependencies**: None.
**Requirement Refs**: FR-007.
**Prompt**: [tasks/WP01-cross-database-fixtures.md](tasks/WP01-cross-database-fixtures.md).
**Independent test**: the prompt’s success criteria and package-specific verification must pass before approval.

| Subtask | Description |
| --- | --- |
| T001 | Create an isolated PostgreSQL test harness |
| T002 | Run existing statistics fixtures against both engines |
| T003 | Characterize boundaries and reproduce the reported SQL failure |
| T004 | Verify harness isolation and document invocation |

**Implementation notes**: follow the prompt in order, use only disposable database fixtures and report real test results. Keep package scope reviewable.

## Work Package WP02: Dashboard and device enum persistence

**Goal**: Dashboard and device enum persistence. **Priority**: P1. **Dependencies**: WP01.
**Requirement Refs**: FR-005, FR-006.
**Prompt**: [tasks/WP02-enum-persistence.md](tasks/WP02-enum-persistence.md).
**Independent test**: the prompt’s success criteria and package-specific verification must pass before approval.

| Subtask | Description |
| --- | --- |
| T005 | Accept database string and byte values |
| T006 | Prove compatibility with PostgreSQL text and bytea |
| T007 | Verify dashboard and device lifecycle persistence |
| T008 | Check startup migration behavior on existing schemas |

**Implementation notes**: follow the prompt in order, use only disposable database fixtures and report real test results. Keep package scope reviewable.

## Work Package WP03: Portable statistics query and error propagation

**Goal**: Portable statistics query and error propagation. **Priority**: P1. **Dependencies**: WP01.
**Requirement Refs**: FR-001, FR-002, FR-003, FR-004.
**Prompt**: [tasks/WP03-portable-statistics.md](tasks/WP03-portable-statistics.md).
**Independent test**: the prompt’s success criteria and package-specific verification must pass before approval.

| Subtask | Description |
| --- | --- |
| T009 | Introduce dialect-specific duration and range typing |
| T010 | Make joins grouping and filter correlations portable |
| T011 | Propagate query errors and normalize driver results |
| T012 | Pass the shared statistics regression suite |

**Implementation notes**: follow the prompt in order, use only disposable database fixtures and report real test results. Keep package scope reviewable.

## Work Package WP04: Daily statistics and dashboard acceptance

**Goal**: Daily statistics and dashboard acceptance. **Priority**: P1. **Dependencies**: WP02, WP03.
**Requirement Refs**: FR-001, FR-002, FR-003, FR-006.
**Prompt**: [tasks/WP04-daily-dashboard-acceptance.md](tasks/WP04-daily-dashboard-acceptance.md).
**Independent test**: the prompt’s success criteria and package-specific verification must pass before approval.

| Subtask | Description |
| --- | --- |
| T013 | Author the deterministic Cyrillic weekly fixture |
| T014 | Verify Stats2 Daily ranges on both engines |
| T015 | Verify dashboard API persistence and chart selections |
| T016 | Record acceptance and regression evidence |

**Implementation notes**: follow the prompt in order, use only disposable database fixtures and report real test results. Keep package scope reviewable.

## Work Package WP05: Mandatory SQLite and PostgreSQL CI

**Goal**: Mandatory SQLite and PostgreSQL CI. **Priority**: P1. **Dependencies**: WP04.
**Requirement Refs**: FR-007.
**Prompt**: [tasks/WP05-dual-database-ci.md](tasks/WP05-dual-database-ci.md).
**Independent test**: the prompt’s success criteria and package-specific verification must pass before approval.

| Subtask | Description |
| --- | --- |
| T017 | Add a health-checked PostgreSQL 17 CI service |
| T018 | Require actual PostgreSQL execution |
| T019 | Provide repeatable local verification commands |
| T020 | Validate the complete required pipeline |

**Implementation notes**: follow the prompt in order, use only disposable database fixtures and report real test results. Keep package scope reviewable.

## Work Package WP06: Release and deployment handoff

**Goal**: Release and deployment handoff. **Priority**: P1. **Dependencies**: WP05.
**Requirement Refs**: FR-008.
**Prompt**: [tasks/WP06-release-runbook.md](tasks/WP06-release-runbook.md).
**Independent test**: the prompt’s success criteria and package-specific verification must pass before approval.

| Subtask | Description |
| --- | --- |
| T021 | Document supported database behavior and configuration |
| T022 | Prepare the immutable image release procedure |
| T023 | Prepare the external Kubernetes manifest change procedure |
| T024 | Define operational acceptance and rollback evidence |

**Implementation notes**: follow the prompt in order, use only disposable database fixtures and report real test results. Keep package scope reviewable.

## Coverage

| Requirement | Packages |
| --- | --- |
| FR-001 | WP03, WP04 |
| FR-002 | WP03, WP04 |
| FR-003 | WP03, WP04 |
| FR-004 | WP03 |
| FR-005 | WP02 |
| FR-006 | WP02, WP04 |
| FR-007 | WP01, WP05 |
| FR-008 | WP06 |

## Completion boundary

Six packages, 24 subtasks. WP01–WP05 deliver and verify application compatibility; WP06 provides release instructions. Deployment and the external manifest execution remain separately recorded operational work. Do not close Vikunja task 114 before its live acceptance criteria have actually been verified.
