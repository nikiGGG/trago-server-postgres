# Remediation contracts

Existing schema.graphql remains authoritative; no endpoint/field changes. These assertions implement spec.md, not new requirements.

| Requirement | Contract and evidence |
| --- | --- |
| FR-001 / NFR-001 | Default/custom-port launcher and CI publish on 127.0.0.1 only. Command doubles plus real Docker HostIp inspection, connectivity and success/failure cleanup. |
| FR-002 / NFR-001 | URL/quoted keyword passwords and full DSNs absent from captured stdout/stderr on success/setup/test failure, including controlled child echoes. Preserve exit status and useful stage messages; executable regressions wired into CI. |
| FR-003 / C-001 | Original fractional identity survives DB precision and serialized GraphQL response. Shared SQLite/PG17 fixtures, scalar round trip and actual GraphQL response test; integer/local-time/filter/user/TZ/zero-fill regressions unchanged. |
| FR-004 | Successful open then real AutoMigrate failure returns nil/error, closes owned pool and prevents startup. Engine tests, observable pool cleanup, controlled cleanup-failure test, startup subprocess. |
| FR-005 | Regressions fail reviewed defects and pass fixes. Healthy fresh/reopen/text/bytea coverage remains passing. |
| FR-006 / NFR-002 | Actual PG17 required run, all PG cases execute and zero PG skips; full lint/test after generation/UI preparation. Missing/unusable DSN and wrong-major negative checks fail. |
| FR-007 / NFR-003 | Both dated report addenda map every source-local finding to common revision-bound evidence/verdict; event-based acceptance attribution and historical outcomes preserved. |

## Independent fractional oracle

For one tagged span 10:00:00–10:00:10Z, independently calculate clipped seconds and round per span:

| Bounds relative to 10:00:00Z | Seconds | Purpose |
| --- | --- | --- |
| 0.200–1.000 | 1 | Reviewed reproduction |
| 0.000–1.800 | 2 | Fractional end |
| 0.200–1.800 | 2 | Both fractional |
| 0.800–1.200 | 0 | Distinct identity with same respective whole seconds |
| 0.200000001–1.800000001 | 2 | Identity finer than PostgreSQL storage |
| 20.200–21.800 | Existing empty/zero-fill | No overlap |

Send the same-second-identity pair together and assert both bounds and independent totals. Add multiple spans distinguishing sum-of-rounded durations from rounded total. Avoid ambiguous half-second floating thresholds. Repeat relevant local-user offset/non-UTC session cases; preserve Stats2 builder conventions.

## Gate and evidence semantics

Missing DSN in direct required tests must fail. Launcher absence of DSN intentionally provisions PostgreSQL and is not that negative case. Unusable supplied DSN fails direct tests and launcher; wrong actual major fails dedicated PG17 gate. Keep intended negative exits separate from acceptance commands, which all must exit zero.

Evidence records tested commit, any working-tree patch identity, timestamp, tools, actual server_version/server_version_num, sanitized commands/environment, exits, test/skip counts, log locations and real binding/cleanup. No secrets. Docs-only follow-up commits may cite tested code commit. Label local/remote execution accurately; historical PG14/18 or absent-DSN runs do not establish fresh PG17 acceptance.
