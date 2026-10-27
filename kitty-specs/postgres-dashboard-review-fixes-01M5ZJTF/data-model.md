# Data and lifecycle model

No persisted tables, columns, enum types or GraphQL fields are added.

| Value | Fields | Invariants |
| --- | --- | --- |
| Requested range | Original model.Time start/end | Existing start ≤ end validation; retain all accepted precision and local-user-time meaning. |
| Query range | Integer range_id; dialect-bound start/end | Internal ID per input position; parameterized, never persisted/exposed; DB rounding cannot redefine identity. |
| Aggregate | range_id; tag key/value; seconds | Existing filtering/user isolation; sum of per-span rounded clipped durations. IDs cannot merge. |
| Response range | Original bounds and entries | Existing UTC response convention and zero/tag fill; fractional precision retained in serialized response. |
| Initialization | Open pool → migration → ready or closed failure | Only ready yields usable handle; failure returns nil/error and closes owned pool. Migration atomicity is not promised. |
| Disposable fixture | Maintenance DSN, random DB, pool | Synthetic isolated data; cleanup after success/failure; DSN excluded from logs/evidence. |
| Evidence | Revision/patch, tools, actual version, commands/exits, tests/skips | Common record supports both addenda; historical provenance retained. |

Existing user/span/tag and dashboard/device relationships stay unchanged. Healthy PostgreSQL text/bytea columns retain physical types and support read/write after reopen. No domain event, callback or external payload is introduced.
