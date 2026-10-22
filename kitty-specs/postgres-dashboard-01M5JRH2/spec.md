# PostgreSQL dashboards with SQLite compatibility

Source: Vikunja task 114, “Исправить статистику и диаграммы Traggo при работе с PostgreSQL”, read with all available comments (none). Planning baseline: repository commit `8ca9bb6`; the task reports a deployment based on Traggo 0.8.3, commit `6321119c3c2d55f04e2e4967f6492aabd6067b76`. These baselines differ and must be reconciled before release.

## Goal and scope

Make existing dashboard statistics, tables, and charts work against PostgreSQL 17, while preserving SQLite behavior and the existing GraphQL/UI contract. This mission covers backend compatibility, regression tests, and a release/runbook plan. The current request authorizes planning; implementation and live rollout are subsequent work.

The task reports 796 completed spans in CNPG PostgreSQL 17, empty charts, and `pq: syntax error at or near "WHERE"`. A deployed workaround converted `dashboard_entries.type`, `dashboard_entries.interval`, and `devices.type` from text to bytea. Both normal text schemas and the existing bytea workaround must remain usable without data loss.

## User scenarios and acceptance

1. A user opens a dashboard containing a table and StackedBarChart, selects Daily and a week, and sees correct daily seconds on PostgreSQL and SQLite, without SQL or Scan errors.
2. Tag key `работа` has separate `дом`, `офис`, and `подработка` series. Each daily value and weekly series total matches an independent calculation from clipped source intervals.
3. Include filters match any supplied key/value pair; Exclude filters remove any matching span. Exclude wins when both apply. Filtering another tag must not multiply displayed durations.
4. A completed span crossing midnight contributes only its overlap to each requested day. Empty ranges remain present with the existing zero-fill behavior. Other users' data is absent.
5. Users create, edit, save, and reopen dashboards, chart types, intervals, named ranges, and filters. Values survive closing/reopening database connections and an application restart.
6. Existing SQLite installations continue working without configuration or schema changes. PostgreSQL text and legacy bytea enum columns support reading and updating stored dashboard/device values.

## Functional requirements

- **FR-001**: Both `stats` and `stats2` execute successfully on PostgreSQL 17 and SQLite, retaining their GraphQL response shapes and daily/week selection behavior.
- **FR-002**: Duration is clipped to each requested range and rounded per span to whole seconds before summation, preserving current local user-time semantics. Running spans remain excluded. Inclusive endpoint behavior and empty keys/ranges must be characterized before changing the query.
- **FR-003**: Preserve Include OR, Exclude OR, exclusion precedence, tag grouping, zero fill, and user isolation. Correct SQL aliases and joins must not change these semantics.
- **FR-004**: SQL execution and scan failures return resolver errors rather than successful empty/zero statistics.
- **FR-005**: DashboardType, Interval, and DeviceType load database string and byte-slice values and persist consistently across supported PostgreSQL schemas and SQLite. Unsupported input types produce a defined error; establish and test nil handling.
- **FR-006**: Dashboard/device operations persist after reconnect/restart. No destructive or automatic text/bytea conversion is required for initial compatibility.
- **FR-007**: The same regression fixtures run on actual PostgreSQL 17 and SQLite in CI; PostgreSQL failures or missing configuration must fail its dedicated CI job.
- **FR-008**: Document build, immutable image identification, staging validation, Kubernetes rollout, observation, and rollback. Updating the external manifest is a later execution step in its owning repository.

## Non-goals

No UI redesign, new chart types, new GraphQL fields, ORM upgrade, database migration tool, SQLite-to-PostgreSQL data transfer, or general MySQL support project. Preserve existing dialect behavior where possible; do not claim new support for untested databases. Avoid unrelated sorting/tag-key cleanup unless required by a reproduced acceptance failure.

## Verification boundaries and open release inputs

The production record count and workaround are task evidence, not independently verified here. Use deterministic synthetic fixtures rather than copying private production data. Confirm the deployed image baseline, exact timestamp column types, bytea parameter write behavior, owning path for `automation-scripts/k8s-cluster/traggo.yaml` (task cites commit `65b29d3`), image registry/tag policy, and deployment access before release. These inputs do not block repository planning. No live database changes or deployment occur during planning.
