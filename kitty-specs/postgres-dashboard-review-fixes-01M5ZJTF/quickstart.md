# Implementation-time verification

These checks are planned, not executed. Run from repository root on the remediation revision with disposable synthetic databases; retain sanitized logs/exits/counts.

1. Record git revision/status/patch identity and tool versions. Check Go (module 1.24, CI 1.25), CGO compiler, Docker, Node/Yarn, gqlgen, goimports. Keep lockfiles/dependencies; resolve missing tools instead of claiming partial acceptance.
2. Follow existing workflow preparation: make download-tools, make install, make generate, then NODE_OPTIONS=--openssl-legacy-provider make build-js. Record actual environment/commands. Require generated GraphQL and real ui/build assets, not stubs. Resolve encountered lifecycle-script/runtime compatibility questions explicitly.
3. Reproduce reviewed defects with regressions, then run targeted fixes and proposed launcher-regression Makefile target. bash -n alone is insufficient.
4. With DSN absent run fresh SQLite/backend checks: go test -count=1 ./model/... ./statistics/... ./dashboard/... ./device/... ./database/... ./test/... . Record optional PG skips separately; this is SQLite evidence only.
5. Run make test-postgres against actual PostgreSQL 17, required mode, expanded database/model packages and verbose/JSON PG execution/skip evidence. Record actual version. Inspect all published HostIp addresses while running; verify connectivity, custom port, and container removal after success and induced failure.
6. Direct required-mode tests with DSN unset must fail: TRAGGO_TEST_POSTGRES_REQUIRED=1 go test -count=1 ./statistics/... ./database/... ./test/... . Repeat with unusable synthetic loopback DSN and bounded connect timeout, including launcher reuse. Exercise wrong-major failure and URL/quoted keyword output sentinels. Record intended negative results separately.
7. Run complete make lint and make test. Inspect goimports filenames even if exit zero; resolve all formatting findings. Retain race/coverage and frontend results. Code changes invalidate affected results; final acceptance must identify final code revision.
8. Write shared docs/reviews evidence; append dated reconciliation to both reports and update affected acceptance provenance from actual events. Keep missing checks open and F-5/F-6 external follow-ups explicit. Later docs-only commits cite tested code revision. Keep task 114 open.

No release/deployment command, production DSN or external manifest change belongs here. Existing build workflow supplies canonical full pipeline; any environment workaround must be documented without weakening acceptance.
