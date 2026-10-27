# Planning investigation and decisions

Repository-source inspection at eb57894caf93636aedcb9e6eda797b26e245b0db, 2026-10-28. No runtime results are claimed.

| Decision | Evidence and rationale | Alternative rejected |
| --- | --- | --- |
| Integer range identity; original response bounds | statistics/summary.go formats PG bounds to seconds, groups returned strings and matches exact input. DB timestamps cannot represent all accepted nanoseconds. | Adding fractions to to_char or typed scans still couples identity to DB precision. |
| RFC3339Nano response serialization | model/time.go parses fractions and OmitTimeZone preserves nanoseconds, but MarshalGQL drops them. | Resolver-only fix leaves response wire incorrect; no schema change is needed. |
| Preserve duration and range builder | summary.go uses clipped per-span rounding; summary2.go delegates after existing range generation. | Rewriting aggregation/range parsing expands behavior and numerical risk. |
| Production New owns migration and cleanup | database/database.go ignores AutoMigrate error; main.go already terminates on New error. model/all.go already puts User first. | Duplicate harness migration masks production failure; no ORM upgrade/framework. |
| External-package DB integration tests | test/postgres.go imports database; reverse internal import cycles. | Bad DSNs never prove migration failure; mocks alone cannot satisfy real-engine acceptance. |
| Same launcher in CI and locally | Existing workflow and launcher omit publish host address. | Separate CI service mapping maintains two security/lifecycle implementations. |
| Remove DSN output, constrain errors | Launcher prints full DSN and streams child output. URL and keyword forms are supported. | Naive keyword splitting breaks quoted credentials; URL-only redaction is insufficient. |
| Extend gate coverage and check actual major/skips | PG_PACKAGES omits database/model; image major can be overridden. | Image labels and historical PG14/18 runs are not PG17 execution evidence. |

No unresolved product or technology selection remains. Exact real-engine migration-conflict fixture mechanics are verified during implementation: begin with a users view/table conflict and use equivalent migration-time schema conflict if GORM introspection requires it. This does not defer a functional requirement. Tool availability, UI generation and actual PG17 behavior remain future execution uncertainties. No dependencies change or external research is required. Any discovered requirement conflict returns to specify rather than weakening acceptance.
