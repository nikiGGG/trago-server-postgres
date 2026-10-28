#!/usr/bin/env bash
#
# test-databases.sh -- run the shared Traggo regression/acceptance fixtures
# against BOTH SQLite (always executed) and a disposable PostgreSQL server
# (mandatory, FR-007). This mirrors the `postgres-test` job in
# .github/workflows/build.yml, so the same command can be used locally and in CI.
#
# The fixtures live in the `statistics`, `device`, `dashboard`, `test`,
# `database` and `model` packages. Each package always runs its SQLite cases,
# and (when the PostgreSQL harness is available) its identical PostgreSQL cases
# in the SAME `go test` invocation. That is how this script "runs the SQLite
# suite AND the PostgreSQL suite" in one go.
#
# Security and lifecycle guarantees (FR-001, FR-002, NFR-001):
#   * A self-started server is published on IPv4 loopback ONLY
#     (127.0.0.1:${PORT}:5432). It is never published on 0.0.0.0 or an
#     external interface, even when a custom host port is used.
#   * Connection settings are passed to the child through the environment, never
#     echoed to output. The full DSN and any password are redacted from captured
#     child output before it is replayed, and temporary captures are removed on
#     every exit.
#   * The actual connected server must be PostgreSQL major 17 or 18. The default
#     `postgres:18` image alone is not accepted evidence; the real
#     server_version_num is checked on both the disposable and the reused path.
#   * Shell tracing is never enabled here.
#
# Environment variables:
#   TRAGGO_TEST_POSTGRES_DSN      (optional) lib/pq connection string to a
#                                 disposable, test-only PostgreSQL server. The
#                                 database named in the DSN (usually
#                                 "postgres") is used only as a maintenance
#                                 target to CREATE DATABASE / DROP DATABASE for
#                                 one isolated database per test; it never
#                                 receives fixtures. When set, the script reuses
#                                 that server. When unset, the script starts its
#                                 own disposable container and removes it
#                                 afterwards.
#   TRAGGO_TEST_POSTGRES_REQUIRED Set to 1 to turn a missing/unusable DSN or
#                                 any skipped PostgreSQL suite into a hard
#                                 failure. This script always forces it to 1 so
#                                 a green run provably executed real PostgreSQL
#                                 fixtures.
#   TRAGGO_TEST_PG_IMAGE          (optional) Docker image used when the script
#                                 starts its own disposable server. Defaults to
#                                 postgres:18. Override locally when postgres:18
#                                 cannot be pulled (e.g. a corporate mirror only
#                                 carries a different supported major version).
#   TRAGGO_TEST_PG_PORT           (optional) host port for a self-started
#                                 container. Defaults to 55432.
#   TRAGGO_TEST_PG_PACKAGES       (optional) space-separated package paths to
#                                 test. Defaults to the six shared suites
#                                 (statistics, device, dashboard, test,
#                                 database, model).
#   TRAGGO_TEST_GO_JSON           Set to 1 to run the child with `go test -json`
#                                 so test events stay machine-parseable on
#                                 stdout; fixed stage messages go to stderr.
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

command -v python3 >/dev/null || { echo "error: Python 3 is required for credential-safe output" >&2; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Packages that carry the shared cross-database regression/acceptance fixtures.
DEFAULT_PG_PACKAGES=(./statistics/... ./device/... ./dashboard/... ./test/... ./database/... ./model/...)

JSON="${TRAGGO_TEST_GO_JSON:-}"
if [[ -n "${TRAGGO_TEST_PG_PACKAGES:-}" ]]; then
  # shellcheck disable=SC2206
  read -r -a PG_PACKAGES <<<"${TRAGGO_TEST_PG_PACKAGES}"
else
  PG_PACKAGES=("${DEFAULT_PG_PACKAGES[@]}")
fi

# stage prints a fixed, credential-free status message. In JSON mode stage text
# is sent to stderr so stdout remains a machine-parseable JSON event stream.
stage() {
  if [[ -n "${JSON}" ]]; then
    echo "$*" >&2
  else
    echo "$*"
  fi
}

DSN="${TRAGGO_TEST_POSTGRES_DSN:-}"
CONTAINER=""
CAPTURE_STDOUT=""
CAPTURE_STDERR=""
VERSION_HELPER=""

cleanup() {
  if [[ -n "${CONTAINER}" ]]; then
    stage "==> Removing disposable PostgreSQL container ${CONTAINER}"
    docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${CAPTURE_STDOUT}" ]]; then
    rm -f "${CAPTURE_STDOUT}" "${CAPTURE_STDERR}"
  fi
  if [[ -n "${VERSION_HELPER}" ]]; then
    rm -f "${VERSION_HELPER}"
  fi
}
trap cleanup EXIT

# Redact both plain diagnostics and JSON-escaped test output. Never pass the
# DSN or password in process arguments. Python 3 uses only its standard library.
redact_output() {
  TRAGGO_TEST_POSTGRES_DSN="$1" python3 "${SCRIPT_DIR}/redact-postgres-output.py"
}

# check_server_version queries the real server_version_num through the supplied
# connection and fails unless it is PostgreSQL major 17 or 18.
check_server_version() {
  local dsn="$1"
  local helper out major status=0
  helper="${SCRIPT_DIR}/pgversion_$$.go"
  cat >"${helper}" <<'GOEOF'
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq" // driver registration for the version probe
)

func main() {
	if os.Getenv("TRAGGO_TEST_POSTGRES_DSN") == "" {
		fmt.Fprintln(os.Stderr, "missing PostgreSQL connection settings")
		os.Exit(2)
	}
	db, err := sql.Open("postgres", os.Getenv("TRAGGO_TEST_POSTGRES_DSN"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open connection: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	var num int
	if err := db.QueryRow("SHOW server_version_num").Scan(&num); err != nil {
		fmt.Fprintf(os.Stderr, "error: query server_version_num: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("major=%d\n", num/10000)
}
GOEOF
  VERSION_HELPER="${helper}"
  CAPTURE_STDOUT="$(mktemp)"
  CAPTURE_STDERR="$(mktemp)"
  (
    cd "${ROOT_DIR}"
    TRAGGO_TEST_POSTGRES_DSN="${dsn}" go run "${helper}"
  ) >"${CAPTURE_STDOUT}" 2>"${CAPTURE_STDERR}" || status=$?
  # Sanitize even failed setup diagnostics before replaying them.
  out="$(redact_output "${dsn}" <"${CAPTURE_STDOUT}")" || return 1
  redact_output "${dsn}" <"${CAPTURE_STDERR}" >&2 || return 1
  if [[ "${status}" -ne 0 ]]; then
    echo "error: could not determine the connected PostgreSQL server version" >&2
    return "${status}"
  fi
  major="$(printf '%s\n' "${out}" | sed -n 's/^major=\([0-9][0-9]*\)$/\1/p')"
  if [[ -z "${major}" ]]; then
    echo "error: could not determine the connected PostgreSQL server major version" >&2
    return 1
  fi
  if [[ "${major}" != "17" && "${major}" != "18" ]]; then
    echo "error: connected PostgreSQL server is major ${major}, but major 17 or 18 is required" >&2
    return 1
  fi
}

# run_suite executes the child `go test` with the DSN and required mode supplied
# through the environment, captures both streams privately, replays them through
# the redaction filter and returns the child's original exit status.
run_suite() {
  local dsn="$1" status json_flag=""
  if [[ -n "${JSON}" ]]; then json_flag="-json"; fi
  # Reuse the probe captures so cleanup tracks all private files.
  set +e
  (
    cd "${ROOT_DIR}"
    TRAGGO_TEST_POSTGRES_DSN="${dsn}" \
    TRAGGO_TEST_POSTGRES_REQUIRED=1 \
      go test -count=1 ${json_flag} "${PG_PACKAGES[@]}"
  ) >"${CAPTURE_STDOUT}" 2>"${CAPTURE_STDERR}"
  status=$?
  set -e
  redact_output "${dsn}" <"${CAPTURE_STDOUT}"
  redact_output "${dsn}" <"${CAPTURE_STDERR}" >&2
  rm -f "${CAPTURE_STDOUT}" "${CAPTURE_STDERR}"
  CAPTURE_STDOUT=""
  CAPTURE_STDERR=""
  return "${status}"
}

if [[ -z "${DSN}" ]]; then
  IMAGE="${TRAGGO_TEST_PG_IMAGE:-postgres:18}"
  PORT="${TRAGGO_TEST_PG_PORT:-55432}"
  CONTAINER="traggo-test-pg-$$"
  stage "==> Starting disposable ${IMAGE} on 127.0.0.1:${PORT} (trust auth)"
  docker run -d \
    --name "${CONTAINER}" \
    -e POSTGRES_HOST_AUTH_METHOD=trust \
    -p "127.0.0.1:${PORT}:5432" \
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
  stage "==> Reusing PostgreSQL server from TRAGGO_TEST_POSTGRES_DSN"
fi

# Require the connected server to actually be major 17 or 18 on both paths.
check_server_version "${DSN}" || exit $?

stage "==> Running regression/acceptance fixtures against SQLite AND PostgreSQL (required)"
run_suite "${DSN}"
