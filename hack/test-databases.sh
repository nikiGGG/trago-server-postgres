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
#   * The actual connected server must be PostgreSQL major 17. The default
#     `postgres:17` image alone is not accepted evidence; the real
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
#                                 postgres:17. Override locally when postgres:17
#                                 cannot be pulled (e.g. a corporate mirror only
#                                 carries a different minor version).
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

# dsn_password extracts the password from a lib/pq connection string. Both URL
# (postgres://...) and keyword/value forms are supported. Keyword values may be
# single-quoted (with doubled single-quote escaping) and may contain spaces and
# '=' characters, so a naive whitespace split is deliberately avoided. The
# password is printed to stdout (empty if the DSN carries none).
dsn_password() {
  local dsn="$1"
  printf '%s' "${dsn}" | perl - "${dsn}" <<'PERL'
    my $dsn = $ARGV[0];
    my $n = length $dsn;
    my $i = 0;
    my $pw = '';
    my $found = 0;
    if ($dsn =~ m{^(postgres|postgresql)://}) {
      (my $userinfo) = $dsn =~ m{^[^:]+://([^@]*)@};
      if (defined $userinfo && $userinfo =~ /:/) {
        (my $enc) = $userinfo =~ /^[^:]*:(.*)$/;
        $enc =~ s/%([0-9A-Fa-f]{2})/chr(hex($1))/ge;
        $pw = $enc; $found = 1;
      }
    } else {
      while ($i < $n) {
        $i++ while $i < $n && substr($dsn,$i,1) =~ /[ \t]/;
        last if $i >= $n;
        my $eq = index($dsn, '=', $i);
        last if $eq < 0;
        my $key = substr($dsn, $i, $eq - $i);
        $i = $eq + 1;
        my $val = '';
        if ($i < $n && substr($dsn,$i,1) eq "'") {
          $i++;
          my $valchars = '';
          my $end = index($dsn, "'", $i);
          while ($end >= 0) {
            if (substr($dsn,$end+1,1) eq "'") {
              $valchars .= substr($dsn, $i, $end - $i) . "'";
              $i = $end + 2;
              $end = index($dsn, "'", $i);
            } else {
              $valchars .= substr($dsn, $i, $end - $i);
              $i = $end + 1;
              $end = -1;
            }
          }
          $val = $valchars;
        } else {
          my $ws = $i;
          $ws++ while $ws < $n && substr($dsn,$ws,1) !~ /[ \t]/;
          $val = substr($dsn, $i, $ws - $i);
          $i = $ws;
        }
        if ($key eq 'password') { $pw = $val; $found = 1; }
      }
    }
    print $pw if $found;
PERL
}

# redact_output replaces the full DSN and the password (both supported forms)
# with fixed markers. Child output is replayed only through this filter so a
# child that echoes a credential cannot leak it into captured launcher output.
redact_output() {
  local dsn="$1" pw="$2"
  perl -e '
    my ($dsn, $pw) = @ARGV;
    while (my $line = <STDIN>) {
      if (length($dsn)) { $line =~ s/\Q$dsn\E/<redacted DSN>/g; }
      if (length($pw))  { $line =~ s/\Q$pw\E/<redacted>/g; }
      print $line;
    }
  ' "${dsn}" "${pw}"
}

# check_server_version queries the real server_version_num through the supplied
# connection and fails unless it is PostgreSQL major 17.
check_server_version() {
  local dsn="$1"
  local helper out major
  helper="${SCRIPT_DIR}/.pgversion-$$.go"
  cat >"${helper}" <<'GOEOF'
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq" // driver registration for the version probe
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pgversion <dsn>")
		os.Exit(2)
	}
	db, err := sql.Open("postgres", os.Args[1])
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
  out="$(cd "${ROOT_DIR}" && go run "${helper}" "${dsn}")" || {
    echo "error: could not determine the connected PostgreSQL server version" >&2
    return 1
  }
  major="$(printf '%s\n' "${out}" | sed -n 's/^major=\([0-9][0-9]*\)$/\1/p')"
  if [[ -z "${major}" ]]; then
    echo "error: could not determine the connected PostgreSQL server major version" >&2
    return 1
  fi
  if [[ "${major}" != "17" ]]; then
    echo "error: connected PostgreSQL server is major ${major}, but major 17 is required" >&2
    return 1
  fi
}

# run_suite executes the child `go test` with the DSN and required mode supplied
# through the environment, captures both streams privately, replays them through
# the redaction filter and returns the child's original exit status.
run_suite() {
  local dsn="$1" pw status json_flag=""
  pw="$(dsn_password "${dsn}")"
  if [[ -n "${JSON}" ]]; then json_flag="-json"; fi
  CAPTURE_STDOUT="$(mktemp)"
  CAPTURE_STDERR="$(mktemp)"
  set +e
  (
    cd "${ROOT_DIR}"
    TRAGGO_TEST_POSTGRES_DSN="${dsn}" \
    TRAGGO_TEST_POSTGRES_REQUIRED=1 \
      go test -count=1 ${json_flag} "${PG_PACKAGES[@]}"
  ) >"${CAPTURE_STDOUT}" 2>"${CAPTURE_STDERR}"
  status=$?
  set -e
  redact_output "${dsn}" "${pw}" <"${CAPTURE_STDOUT}"
  redact_output "${dsn}" "${pw}" <"${CAPTURE_STDERR}" >&2
  rm -f "${CAPTURE_STDOUT}" "${CAPTURE_STDERR}"
  CAPTURE_STDOUT=""
  CAPTURE_STDERR=""
  return "${status}"
}

if [[ -z "${DSN}" ]]; then
  IMAGE="${TRAGGO_TEST_PG_IMAGE:-postgres:17}"
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

# Require the connected server to actually be major 17 on both paths.
check_server_version "${DSN}" || exit $?

stage "==> Running regression/acceptance fixtures against SQLite AND PostgreSQL (required)"
run_suite "${DSN}"