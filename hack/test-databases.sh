#!/usr/bin/env bash
#
# test-databases.sh -- run the shared Traggo regression/acceptance fixtures
# against BOTH SQLite (always executed) and a disposable PostgreSQL server
# (mandatory, FR-007). This mirrors the `postgres-test` job in
# .github/workflows/build.yml, so the same command can be used locally and in CI.
#
# The fixtures live in the `statistics`, `device`, `dashboard` and `test`
# packages. Each package always runs its SQLite cases, and (when the PostgreSQL
# harness is available) its identical PostgreSQL cases in the SAME `go test`
# invocation. That is how this script "runs the SQLite suite AND the PostgreSQL
# suite" in one go.
#
# Environment variables:
#   TRAGGO_TEST_POSTGRES_DSN      (optional) lib/pq connection string to a
#                                 disposable, test-only PostgreSQL server. The
#                                 database named in the DSN (usually
#                                 "postgres") is used only as a maintenance
#                                 target to CREATE DATABASE / DROP DATABASE for
#                                 one isolated database per test; it never
#                                 receives fixtures. When set, the script reuses
#                                 that server (this is the GitHub Actions path,
#                                 where a PostgreSQL service container is
#                                 already running). When unset, the script
#                                 starts its own disposable container and
#                                 removes it afterwards.
#   TRAGGO_TEST_POSTGRES_REQUIRED Set to 1 to turn a missing/unusable DSN or
#                                 any skipped PostgreSQL suite into a hard
#                                 failure. This script always forces it to 1 so
#                                 a green run provably executed real PostgreSQL
#                                 fixtures (T018).
#   TRAGGO_TEST_PG_IMAGE          (optional) Docker image used when the script
#                                 starts its own disposable server. Defaults to
#                                 postgres:17 to match the CI service container.
#                                 Override locally when postgres:17 cannot be
#                                 pulled (e.g. a corporate mirror only carries a
#                                 different minor or major version).
#   TRAGGO_TEST_PG_PORT           (optional) host port for a self-started
#                                 container. Defaults to 55432.
#
# Disposable-database assumptions:
#   * NEVER point TRAGGO_TEST_POSTGRES_DSN at a production, shared or
#     long-lived database. The harness issues `CREATE DATABASE` and
#     `DROP DATABASE ... WITH (FORCE)` for a random database per test.
#   * lib/pq v1.0.0 (pinned in go.mod) does NOT implement PostgreSQL's default
#     SCRAM-SHA-256 authentication (the default since PG15). A fresh server
#     therefore runs with `POSTGRES_HOST_AUTH_METHOD=trust` so the pinned
#     driver can connect. Never use trust authentication on a real database.
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Packages that carry the shared cross-database regression/acceptance fixtures.
PG_PACKAGES=(
  ./statistics/...
  ./device/...
  ./dashboard/...
  ./test/...
)

DSN="${TRAGGO_TEST_POSTGRES_DSN:-}"
CONTAINER=""

cleanup() {
  if [[ -n "${CONTAINER}" ]]; then
    echo "==> Removing disposable PostgreSQL container ${CONTAINER}"
    docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

if [[ -z "${DSN}" ]]; then
  IMAGE="${TRAGGO_TEST_PG_IMAGE:-postgres:17}"
  PORT="${TRAGGO_TEST_PG_PORT:-55432}"
  CONTAINER="traggo-test-pg-$$"
  echo "==> Starting disposable ${IMAGE} on 127.0.0.1:${PORT} (trust auth)"
  docker run -d \
    --name "${CONTAINER}" \
    -e POSTGRES_HOST_AUTH_METHOD=trust \
    -p "${PORT}:5432" \
    "${IMAGE}" >/dev/null

  # Health check: wait for pg_isready inside the container.
  for _ in $(seq 1 60); do
    if docker exec "${CONTAINER}" pg_isready -U postgres >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
  if ! docker exec "${CONTAINER}" pg_isready -U postgres >/dev/null 2>&1; then
    echo "error: disposable PostgreSQL ${IMAGE} did not become ready" >&2
    exit 1
  fi
  DSN="postgres://postgres@127.0.0.1:${PORT}/postgres?sslmode=disable"
else
  echo "==> Reusing PostgreSQL server from TRAGGO_TEST_POSTGRES_DSN"
fi

echo "==> Running regression/acceptance fixtures against SQLite AND PostgreSQL"
echo "    TRAGGO_TEST_POSTGRES_DSN=${DSN}"
echo "    TRAGGO_TEST_POSTGRES_REQUIRED=1"
(
  cd "${ROOT_DIR}"
  TRAGGO_TEST_POSTGRES_DSN="${DSN}" \
  TRAGGO_TEST_POSTGRES_REQUIRED=1 \
    go test -count=1 "${PG_PACKAGES[@]}"
)