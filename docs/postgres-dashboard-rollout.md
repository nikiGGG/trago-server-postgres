# Traggo server — PostgreSQL dashboard release and deployment runbook

Status: **handoff / draft — NOT yet executed.** This document is the release and
deployment handoff for making dashboard statistics, tables and charts work
against PostgreSQL 17 while preserving SQLite behavior. It satisfies **FR-008**
("Document build, immutable image identification, staging validation, Kubernetes
rollout, observation, and rollback"). Completing this work package (WP06) does
**not** deploy anything, does **not** publish an image, and does **not** mark the
external task 114 as deployed. Live rollout and the external manifest change are
separate execution steps that require explicit follow-up instructions and access
to the owning repositories.

**This is a documentation-only package. Nothing in this runbook has been run
against a live cluster or a production database.**

---

## 1. Scope and source of truth

- Requirement: **FR-008** from `kitty-specs/postgres-dashboard-01M5JRH2/spec.md`.
- Task: Vikunja task 114, "Исправить статистику и диаграммы Traggo при работе
  с PostgreSQL".
- Deployment context (from the task, not independently verified): production
  runs Traggo **0.8.3** at commit
  `6321119c3c2d55f04e2e4967f6492aabd6067b76` on CNPG PostgreSQL 17, with a
  deployed workaround that converted `dashboard_entries.type`,
  `dashboard_entries.interval` and `devices.type` from `text` to `bytea`.
- Mission scope: backend compatibility only. **No** UI redesign, new chart
  types, new GraphQL fields, ORM upgrade, database migration tool, or
  SQLite→PostgreSQL data transfer. Existing dialect behavior is preserved where
  possible.

The authoritative test/CI evidence for the FRs referenced here is:

| Requirement | Evidence location |
| --- | --- |
| FR-001/002/003 (stats, clipping, rounding, Include/Exclude, zero-fill, user isolation) | `statistics/summary.go` (dialect fragments), `statistics/summary_test.go`, `statistics/summary2_test.go` |
| FR-001/002 (session-TimeZone independence) | `statistics/summary2_test.go` — `TestStats2_daily_postgres_sessionTimeZone` |
| FR-004 (SQL/Scan errors surface to resolvers) | `statistics/summary.go` (`Raw(...).Scan(...).Error`), covered by the shared fixtures |
| FR-005/006 (text + legacy bytea enum persistence) | `model/dashboard.go`, `model/device.go`, `dashboard/postgres_persistence_test.go` |
| FR-006 (dashboard/device persistence across restart) | `dashboard/postgres_acceptance_test.go` — `TestDashboardAPI_sqlite`, `TestDashboardAPI_postgres` (close + reopen) |
| FR-007 (mandatory SQLite **and** PostgreSQL CI) | `.github/workflows/build.yml` (`postgres-test` job), `Makefile` (`test-postgres`), `hack/test-databases.sh` |
| FR-008 (this document) | `docs/postgres-dashboard-rollout.md`, `README.md` |

---

## 2. Supported database behavior and configuration (T021)

### 2.1 SQLite remains the default

The default database dialect is **SQLite** (`sqlite3`). This is the value baked
into the application defaults in `config/config.go`
(`DatabaseDialect` default `"sqlite3"`) and in `.env.sample`. Existing SQLite
installations keep working with **no configuration or schema change**. No
automatic text→bytea conversion is performed for SQLite and none is required.

### 2.2 PostgreSQL 17 configuration

PostgreSQL 17 is selected via the `TRAGGO_DATABASE_DIALECT` environment
variable. Connection settings are supplied through the existing
`TRAGGO_DATABASE_CONNECTION` variable. Credentials are **not** shown here and
must be provided by the deployer at runtime.

```
TRAGGO_DATABASE_DIALECT=postgres
TRAGGO_DATABASE_CONNECTION=host=<host> port=5432 user=<user> dbname=<dbname> sslmode=<mode> password=<password>
```

- `TRAGGO_DATABASE_CONNECTION` for PostgreSQL is a lib/pq connection string
  (the GORM `jinzhu/gorm/dialects/postgres` dialect uses lib/pq). Both URL form
  (`postgres://user:pass@host:5432/db?sslmode=disable`) and keyword/value form
  are accepted.
- The application opens the configured database and runs `AutoMigrate` on the
  existing schema (see `database/database.go`). No destructive column
  conversion is performed.
- The config file loader (`.env`, `.env.local`, `/etc/traggo/server.ini`) and
  the `TRAGGO_*` prefix are unchanged.

> Note on driver auth: the pinned lib/pq v1.0.0 does **not** implement
> PostgreSQL's default SCRAM-SHA-256 authentication (default since PG15). The
> **test** harness and CI use `trust` auth on disposable servers only. A real
> deployment must configure an authentication method the pinned driver supports
> (e.g. password/md5 via a compatible server config, or a server that still
> allows it). This is an infrastructure decision, not an application one.

### 2.3 text/bytea compatibility (no automatic column conversion)

The enum fields `DashboardType`, `Interval` (both in `model/dashboard.go`) and
`DeviceType` (`model/device.go`) are now scanned via `scanStringEnum`, which
accepts **both** a Go `string` (normal `text` columns and SQLite) **and** a byte
slice (`[]byte`, PostgreSQL legacy `bytea` columns). The `Value()` methods
continue to write the plain string value.

Consequences:

- **Normal `text` schema works** (fresh PostgreSQL installs).
- **The legacy `bytea` workaround works** — the deployed
  `dashboard_entries.type`, `dashboard_entries.interval` and `devices.type`
  bytea columns are readable and updatable with no schema change.
- **No automatic column conversion is required or performed.** Do not convert
  the existing bytea columns back to text as part of this fix. FR-006 explicitly
  states no destructive or automatic text/bytea conversion is required for
  initial compatibility.
- Unsupported types (including `nil`/NULL) produce a defined error rather than
  silent defaulting (FR-005).

### 2.4 Local user-time semantics

Durations are computed in the local wall-clock terms of the stored user times
(`start_user_time` / `end_user_time`), preserving the existing local user-time
semantics (FR-002):

- Each span's duration is **clipped** to the requested range and **rounded to
  whole seconds per span** before summation (rounding after SUM would change
  results).
- **Running spans** (no end time) remain excluded.
- A completed span crossing a day boundary contributes **only its overlap** to
  each requested day.
- PostgreSQL uses `ROUND(EXTRACT(EPOCH FROM (clipped_end - clipped_start)))`;
  SQLite keeps its original rounded `julianday` arithmetic. Both are equivalent
  at whole-second granularity.
- The daily/weekly values are **independent of the PostgreSQL session
  TimeZone** (verified by `TestStats2_daily_postgres_sessionTimeZone`, WP03's
  session-TZ fix). A span `23:30 → 00:30` across midnight yields the preserved
  Stats2 second-inclusive split (1799 s on Friday / 1800 s on Saturday), not a
  silently changed half-open split.

### 2.5 Targeted verification for this section

The behavior described above is covered by the shared cross-database fixtures
that run against **both** SQLite (always) and a disposable PostgreSQL 17
(mandatory). Run them locally with:

```
make test-postgres
```

`make test-postgres` invokes `hack/test-databases.sh`, which either starts a
disposable `postgres:17` container (default, trust auth) or reuses
`TRAGGO_TEST_POSTGRES_DSN`, and forces `TRAGGO_TEST_POSTGRES_REQUIRED=1` so a
green run provably executed real PostgreSQL fixtures (never SQLite-only). The
identical suite is the `postgres-test` CI job in `.github/workflows/build.yml`.

---

## 3. Immutable image release procedure (T022)

### 3.1 Reconcile the checkout baseline with the deployed revision

Before building the image, confirm the build source is a **superset** of what is
currently deployed (task 114 baseline). The two baselines differ:

| Baseline | Commit | Meaning |
| --- | --- | --- |
| Deployed (task 114) | `6321119c3c2d55f04e2e4967f6492aabd6067b76` (Traggo 0.8.3) | What production currently runs |
| Planning baseline | `8ca9bb6aa414ae621488a5a158a6235e89cfa57b` | The checkout when this mission was planned |

For this mission's accepted revision (the integration branch after WP01..WP05
are merged), `6321119c` is an **ancestor** of `HEAD` — verified with
`git merge-base --is-ancestor 6321119c3c2d55f04e2e4967f6492aabd6067b76 HEAD`
returns success. The accepted revision therefore includes the deployed code plus
all later upstream changes and the mission's compatibility work. **No upstream
code is missing.** Still, as a release gate, record the exact source commit that
the image is built from (see §3.3) so the image's `BuildCommit` label and the
deployed manifest digest are traceable.

> Release gate: confirm the target image revision is reachable from
> `origin/plan/postgres-dashboard` (or the intended tag), not from a local-only
> commit.

### 3.2 Existing image build tooling (verified)

The repository builds immutable Docker images with **GoReleaser**, driven by the
`make release` / `make release-snapshot` targets, plus a local binary target.

| Target / file | What it does | Verified present |
| --- | --- | --- |
| `make build-bin-local` | Local binary `build/traggo-server` (requires `pre-build`/UI assets; CGO_ENABLED=1, static, prod build info via ldflags) | Yes (`Makefile`) |
| `make release-snapshot` | GoReleaser snapshot build — builds images **without publishing**. Use for a disposable test image. | Yes (`Makefile`) |
| `make release` | GoReleaser release — builds **and** publishes the images/manifest (requires a git tag and Docker Hub creds). **Do not run in this WP.** | Yes (`Makefile`) |
| `make build` | Declared `.PHONY` but has **no recipe** (empty target). It does not build an image; use `make build-bin-local` or `make release-snapshot`. | Yes (`Makefile`, empty) |
| `.goreleaser.yml` | Defines builds (linux-amd64/arm64/armv7, windows) and the `dockers`/`docker_manifests` image set | Yes |
| `docker/Dockerfile` | Final runtime image: `FROM scratch`, copies the built `traggo` binary, `EXPOSE 3030` | Yes |
| `docker/Dockerfile.build` | `goreleaser-cross` builder image | Yes |
| `docker/Dockerfile.dev` | Local dev multi-stage build | Yes |
| `docker-compose.yml` | Local compose using `docker/Dockerfile.dev` (SQLite `.traggodata`) | Yes |

GoReleaser produces these image names and tags:

- Per-arch: `traggo/server:amd64-latest`, `traggo/server:amd64-<version>`;
  `arm64-...`; `armv7-...`.
- Manifests: `traggo/server:latest`, `traggo/server:<version>`.

Version is resolved by GoReleaser from the git tag. **This fork has no git tags**
(`git describe --tags --abbrev=0` returns nothing, `git tag -l` is empty), so a
real `make release` requires a tag first (e.g. `v0.8.4` or a fork-specific
version), which is a release-time decision to confirm. For staging, a snapshot
image is sufficient.

The `BuildVersion`/`BuildCommit`/`BuildDate` strings are injected at link time
via ldflags (`main.BuildVersion`, etc.) and logged at startup (`main.go`), so a
built image's exact source commit is traceable from its logs.

### 3.3 Record source commit and image digest (verified procedure, digest is a follow-up)

Because this WP must not build or publish, the concrete image digest is **not
known here** and is recorded as a follow-up to capture at build time. The
procedure to record it:

1. Build the accepted revision (source commit `$COMMIT`):
   ```
   git checkout <accepted-revision>   # the merged integration branch / tag
   make release-snapshot              # staging/test image (no publish)
   ```
   For a publishable image (separate release step):
   ```
   git tag v<version> && make release
   ```
2. Record `$COMMIT`:
   ```
   git rev-parse HEAD                 # the exact source commit
   git log -1 --format='%H %s'        # human-readable confirmation
   ```
3. Record the image digest (not its mutable tag — digests are immutable):
   ```
   docker images --digests traggo/server
   # or after pulling the deployed image:
   docker inspect --format '{{index .RepoDigests 0}}' traggo/server:latest
   ```
4. Keep the **previous deployed image digest** (the current
   `traggo/server:...` digest at `6321119c`) as the rollback image.

### 3.4 Staging prerequisites

Before any rollout, confirm:

- A staging Kubernetes namespace with a CNPG PostgreSQL 17 cluster matching the
  production topology (text-schema clean install **and** a replica of the bytea
  workaround schema, to prove both paths).
- Image registry/tag policy confirmed (which registry and whether
  `traggo/server` or a fork mirror is used by the deployment).
- `make test-postgres` green on the accepted revision, plus `make lint` and
  `make test` green (generated GraphQL sources and UI assets first:
  `make download-tools && make install && make generate && make build-js`).
- Existing CNPG backup procedures available (follow them before any later
  schema operation — none is planned for this fix).

### 3.5 Rollback image

The rollback image is the **current deployed digest** (the image running
`6321119c` / Traggo 0.8.3). Because this fix does not change the database
schema (no destructive migration, bytea columns untouched), rolling back the
image requires **no data restoration**: the previous image can read the same
data. See §5 for the rollback procedure.

---

## 4. External Kubernetes manifest change procedure (T023)

> **Unverified in this environment.** The owning automation repository is **not
> present or accessible** from this device. We did not find `automation-scripts`,
> `traggo.yaml`, or workaround commit `65b29d3` in this repository or in the
> local workspace tree. Nothing below was executed; it is the recorded external
> follow-up, not a claim of a completed manifest update.

The production deployment is owned by a **separate automation repository** (not
this one). From the task evidence:

- Owning file: `automation-scripts/k8s-cluster/traggo.yaml`.
- Workaround commit: `65b29d3` (introduced the bytea workaround in the manifest
  / deployment tooling).

### 4.1 Required external follow-up

1. **Obtain access** to the owning automation repository from the deployment
   owner / SRE. Confirm the exact repository URL and the current state of
   `automation-scripts/k8s-cluster/traggo.yaml`.
2. Inspect the `traggo.yaml` image reference and workaround commit `65b29d3`.
3. Make an **image-only** change to the manifest: point the deployment's image
   at the **new immutable digest** recorded in §3.3 (the accepted revision),
   **retaining the existing bytea workaround** (do not remove the bytea columns
   or their handling — FR-006 requires the workaround to keep working). Do not
   introduce any schema/data migration in the manifest.
4. Retain the **previous image digest** in the manifest comment/history so the
   rollback step (see §5) can restore it.
5. Apply the manifest to **staging first**, validate against §5, then promote to
   production.
6. Commit the manifest change in the **external** repository. Because that
   repository is outside this package's control, the exact diff, PR/merge
   process and final digest are **not recorded here** — they are the deployer's
   responsibility and must be attached to the task when done.

The manifest update is a **separate execution step** that requires access to the
owning repository. It has **not** happened as part of this WP and **must not be
invented** here.

---

## 5. Operational acceptance and rollback evidence (T024)

This checklist is the staging/production acceptance gate. It mirrors the manual
chart/restart check documented in the header of
`dashboard/postgres_acceptance_test.go` and the deterministic Cyrillic fixture in
`statistics/summary2_test.go`, so each step maps to a real, verified behavior.

### 5.1 Setup for the check

- Deploy the accepted image (via the manifest from §4, or the release procedure
  in §3) to a server connected to PostgreSQL 17 (text schema) and, separately,
  confirm the same steps against the bytea-workaround schema.
- Seed synthetic data only (no private production records), or use a controlled
  test user with known spans.

### 5.2 Acceptance checklist

Run **all** of the following, on a **clean install** and **again after an
application restart**:

1. **Selected-week Daily table + StackedBarChart.**
   Open a dashboard; add a Table (VerticalTable) and a StackedBarChart, each
   with interval **Daily**, a named range (or `now-7d..now`) and tag key
   `работа`. Select the week Mon..Sun and confirm both charts show daily buckets
   (7 columns). Empty days show zero-height bars (zero-fill). A span crossing
   midnight is counted in both adjacent days.
2. **Three Cyrillic series.** The chart shows the three series for tag key
   `работа`: `дом`, `офис`, `подработка`, with the per-day seconds matching the
   independent hand calculation in `statistics/summary2_test.go`
   (`cyrillicDailySecs` / `cyrillicWeeklyTotals`). The known fixture week
   (Mon 2019-06-10 .. Sun 2019-06-16) yields, e.g., `дом` weekly total 23400 s,
   `офис` 32400 s, `подработка` 35999 s (the last reflecting the midnight-cross
   span split 1799 s Friday / 1800 s Saturday under Stats2's second-inclusive
   daily builder).
3. **Source-duration totals vs dashboard totals.** Sum the clipped source
   intervals for a representative series/week independently and compare with the
   dashboard's displayed total. They must match within the preserved
   per-span whole-second rounding.
4. **Include / Exclude.** On the StackedBarChart apply Include
   `["работа"=дом]` and Exclude `["работа"=подработка]`; leave the Table
   unfiltered. Confirmed series are present (with zero-fill on empty days);
   excluded series never appear; Include matches any supplied key/value; Exclude
   removes any matching span; Exclude wins when both apply; filtering another tag
   must not multiply displayed durations.
5. **Create / edit / reopen.** Create, edit, save and reopen dashboards, chart
   types, intervals, named ranges and filters. Values survive closing/reopening
   database connections and an application restart. (Automated on both engines
   by `TestDashboardAPI_sqlite` and `TestDashboardAPI_postgres` in
   `dashboard/postgres_acceptance_test.go`.)
6. **Persistence after process restart.** Restart the application (Kubernetes
   rollout or stop/start the binary), reopen the dashboard and confirm the chart
   types, Daily interval, named range, Include/Exclude filters and layout all
   survived the restart. On the bytea-workaround schema, confirm dashboard/device
   enum values are readable and updateable (FR-005/006).
7. **SQL/Scan error log checks.** Confirm both the `stats` and `stats2` queries
   return rows with **no** `pq: syntax error at or near "WHERE"` and **no** Scan
   errors in the application and database logs (FR-004: failures surface as
   resolver errors, not silently empty/zero statistics).
8. **User isolation.** Confirm another user's identical-looking data is absent
   from the checked dashboard (fixture includes a user-2 `работа=дом` span that
   must never leak into user 1's statistics).

### 5.3 Rollback procedure

On any acceptance failure (wrong totals, SQL/Scan errors, persistence loss,
chart/filter breakage):

1. **Revert the image** to the previous deployed digest (the rollback image from
   §3.3 / §4.1) by restoring the prior manifest image reference in the external
   automation repo and reapplying it (standard `kubectl rollout` / manifest
   apply).
2. Because this fix performs **no schema migration and no column conversion**,
   schema compatibility permits rollback **without data restoration** — the old
   image reads the same data. Only re-run §5.2 after rollback to confirm
   recovery.
3. Do not leave the failed digest in staging/production; record the failure
   evidence (which checklist item failed, logs) in the task before retrying.

### 5.4 Scope boundary (important)

Completing this WP **does not** deploy task 114 and **does not** authorize a
production rollout or external manifest write. Live rollout and external writes
require **subsequent, explicit execution instructions** (including access to the
owning automation repo and confirmation of the registry/image policy).

---

## 6. Follow-ups / unverified items

| Item | Status |
| --- | --- |
| Exact new image digest | Not computed (no build in this WP). Capture at build time via §3.3. |
| Owning automation repo URL and `automation-scripts/k8s-cluster/traggo.yaml` state | Not accessible from this environment. Requires access (§4). |
| Workaround commit `65b29d3` details | Cited by the task, not verifiable here. Confirm in the owning repo. |
| Release version/tag | No git tags in this fork; decide version (e.g. `v0.8.4` or fork-specific) before `make release`. |
| Registry/image tag policy used by the deployment | To be confirmed with the deployment owner; this repo's tooling publishes `traggo/server:<version>` etc. |
| Production record count / bytea workaround | Task evidence only; use synthetic fixtures (done). |

These do not block repository work; they are the handoff inputs a deployer must
confirm before any rollout.