#!/usr/bin/env bash
#
# test-databases_test.sh -- executable shell regressions for the safe
# PostgreSQL launcher (hack/test-databases.sh). These tests run WITHOUT Docker
# and WITHOUT a real Go toolchain: synthetic `docker` and `go` command doubles
# are placed first on PATH, the launcher is executed, and its stdout, stderr and
# exit status are asserted against the safety contract (FR-001, FR-002,
# NFR-001):
#
#   * loopback-only publishing (127.0.0.1:${PORT}:5432) for default and custom
#     ports, with exactly one cleanup for self-started containers and none for
#     a reused server;
#   * no credential-bearing DSN or password in any captured output across
#     URL/quoted-keyword DSNs, success, setup failure, wrong-major and test
#     failure paths, including a child that deliberately echoes a sentinel;
#   * actual PostgreSQL major 17 or 18 is required (both accepted, other majors
#     rejected) on both the disposable and reused paths;
#   * JSON mode keeps child events machine-parseable on stdout with stage
#     messages on stderr;
#   * TRAGGO_TEST_PG_PACKAGES overrides the default six-package list.
#
# The harness itself exits nonzero on the first failed assertion.
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LAUNCHER="${SCRIPT_DIR}/test-databases.sh"

TESTS_RUN=0
TESTS_SKIPPED=0
TESTS_FAILED=0

fail() {
  echo "ASSERT FAILED: $*" >&2
  TESTS_FAILED=$((TESTS_FAILED + 1))
  return 1
}

assert_contains() { # haystack needle
  if [[ "$1" != *"$2"* ]]; then
    fail "expected to find [$2] in output:"$'\n'"$1"
  fi
}

assert_not_contains() { # haystack needle
  if [[ "$1" == *"$2"* ]]; then
    fail "did not expect [$2] in output:"$'\n'"$1"
  fi
}

assert_eq() { # actual expected
  if [[ "$1" != "$2" ]]; then
    fail "expected [$2] got [$1]"
  fi
}

# ---- Build isolated command doubles -----------------------------------------

TEST_DIR="$(mktemp -d)"
trap 'rm -rf "${TEST_DIR}"' EXIT

FAKE_DIR="${TEST_DIR}/bin"
mkdir -p "${FAKE_DIR}"
DOCKER_LOG="${TEST_DIR}/docker.log"
GO_LOG="${TEST_DIR}/go.log"
: >"${DOCKER_LOG}"
: >"${GO_LOG}"

cat >"${FAKE_DIR}/docker" <<'FAKE'
#!/usr/bin/env bash
LOG="${FAKE_DOCKER_LOG:?FAKE_DOCKER_LOG must be set}"
echo "docker $*" >>"${LOG}"
op="${1:-}"
case "${op}" in
  run)
    echo "cont-12345"
    ;;
  exec)
    if [[ -n "${FAKE_DOCKER_NOT_READY:-}" ]]; then
      echo "pg_isready: no response" >&2
      exit 1
    fi
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
FAKE

cat >"${FAKE_DIR}/go" <<'FAKE'
#!/usr/bin/env bash
LOG="${FAKE_GO_LOG:?FAKE_GO_LOG must be set}"
echo "go $*" >>"${LOG}"
sub="${1:-}"
shift || true
case "${sub}" in
  run)
    if [[ -n "${FAKE_GO_RUN_EXIT:-}" ]]; then
      printf "error: connection refused %s\n" "${FAKE_CHILD_SECRET:-}" >&2
      exit "${FAKE_GO_RUN_EXIT}"
    fi
    printf 'major=%s\n' "${FAKE_GO_MAJOR:-17}"
    ;;
  test)
    # Use real JSON encoding so quotes/backslashes/control characters exercise
    # the same representation as go test -json. Plain diagnostics use stderr.
    if [[ -n "${FAKE_CHILD_SECRET:-}" ]]; then
      echo "child leaked: ${FAKE_CHILD_SECRET}" >&2
      python3 -c 'import json, os; print(json.dumps({"Action":"output", "Test":"TestX_postgres", "Output":os.environ["FAKE_CHILD_SECRET"]+"\n"}, separators=(",", ":")))'
    fi
    if [[ "$*" == *"-json"* ]]; then
      printf '{"Action":"run","Test":"TestX_postgres"}\n'
      printf '{"Action":"pass","Test":"TestX_postgres"}\n'
      printf '{"Action":"pass","Test":"TestY_sqlite"}\n'
    else
      echo "=== RUN   TestX_postgres"
      echo "--- PASS: TestX_postgres"
      echo "PASS"
    fi
    exit "${FAKE_GO_TEST_EXIT:-0}"
    ;;
  *)
    exit 0
    ;;
esac
FAKE

# Readiness is doubled too: exhausting retries must not cost a real minute.
cat >"${FAKE_DIR}/sleep" <<'FAKE'
#!/usr/bin/env bash
exit 0
FAKE
chmod +x "${FAKE_DIR}/docker" "${FAKE_DIR}/go" "${FAKE_DIR}/sleep"

# run_launcher executes the launcher with the doubled PATH, capturing stdout,
# stderr and exit status into globals. Extra env assignments are passed through
# as the first arguments (e.g. run_launcher TRAGGO_TEST_PG_PORT=56789).
run_launcher() {
  local extra=()
  while [[ "$#" -gt 0 && "$1" == *=* ]]; do
    extra+=("$1")
    shift
  done
  : >"${DOCKER_LOG}"
  : >"${GO_LOG}"
  set +e
  env PATH="${FAKE_DIR}:$PATH" FAKE_DOCKER_LOG="${DOCKER_LOG}" FAKE_GO_LOG="${GO_LOG}" \
    ${extra[@]+"${extra[@]}"} bash "${LAUNCHER}" >"${TEST_DIR}/out" 2>"${TEST_DIR}/err"
  LAUNCHER_EXIT=$?
  set -e
  LAUNCHER_STDOUT="$(cat "${TEST_DIR}/out")"
  LAUNCHER_STDERR="$(cat "${TEST_DIR}/err")"
  LAUNCHER_OUT="${LAUNCHER_STDOUT}"$'\n'"${LAUNCHER_STDERR}"
  TESTS_RUN=$((TESTS_RUN + 1))
}

# has_broad_publish returns 0 (true) if the doubled docker log contains a
# Docker `-p` mapping whose host portion is NOT 127.0.0.1 (a bare "PORT:PORT",
# "0.0.0.0:PORT:PORT", or any explicit non-loopback host).
has_broad_publish() {
  local toks t
  toks="$(tr ' ' '\n' <"${DOCKER_LOG}" | grep -A1 -- '^-p$' | grep -v -- '^-p$' | grep -v '^--$')"
  while read -r t; do
    [[ -z "${t}" ]] && continue
    if [[ "${t}" != 127.0.0.1:* ]]; then
      return 0
    fi
  done <<<"${toks}"
  return 1
}

SENTINEL="s3nt1n3l-pw-9x2"

# ---- Test 1: default port, loopback publish, cleanup ------------------------
run_launcher
assert_eq "${LAUNCHER_EXIT}" "0"
assert_contains "$(cat "${DOCKER_LOG}")" "-p 127.0.0.1:55432:5432 postgres:18"
# Ensure no non-loopback publish mapping is ever used.
if has_broad_publish; then
  fail "publish mapping was not restricted to loopback in:"$'\n'"$(cat "${DOCKER_LOG}")"
fi
# Exactly one cleanup for a self-started container.
assert_eq "$(grep -c 'docker rm -f traggo-test-pg-' "${DOCKER_LOG}" || true)" "1"
# No DSN/password anywhere.
assert_not_contains "${LAUNCHER_OUT}" "postgres://postgres@127.0.0.1"

# ---- Test 2: custom port stays on loopback ----------------------------------
run_launcher TRAGGO_TEST_PG_PORT=56789
assert_eq "${LAUNCHER_EXIT}" "0"
assert_contains "$(cat "${DOCKER_LOG}")" "-p 127.0.0.1:56789:5432 postgres:18"
assert_contains "${LAUNCHER_STDOUT}" "on 127.0.0.1:56789"

# ---- Test 3: reused URL DSN success leaks nothing ---------------------------
run_launcher TRAGGO_TEST_POSTGRES_DSN="postgres://u:${SENTINEL}@127.0.0.1:55432/postgres?sslmode=disable"
assert_eq "${LAUNCHER_EXIT}" "0"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"
assert_not_contains "${LAUNCHER_OUT}" "postgres://u:${SENTINEL}@127.0.0.1"
assert_not_contains "${LAUNCHER_OUT}" "postgres://"
# Reused server must NOT trigger container cleanup.
assert_eq "$(grep -c 'docker rm -f' "${DOCKER_LOG}" || true)" "0"
assert_contains "${LAUNCHER_STDOUT}" "Reusing PostgreSQL server"

# ---- Test 4: reused URL DSN test-failure preserves exit and leaks nothing ---
run_launcher TRAGGO_TEST_POSTGRES_DSN="postgres://u:${SENTINEL}@127.0.0.1:55432/postgres?sslmode=disable" FAKE_GO_TEST_EXIT=7
assert_eq "${LAUNCHER_EXIT}" "7"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"
assert_not_contains "${LAUNCHER_OUT}" "postgres://"

# ---- Test 5: quoted keyword password (spaces/escapes) leaks nothing ---------
run_launcher TRAGGO_TEST_POSTGRES_DSN="host=127.0.0.1 port=55432 user=u password='${SENTINEL} with spaces' dbname=postgres sslmode=disable"
assert_eq "${LAUNCHER_EXIT}" "0"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"
assert_not_contains "${LAUNCHER_OUT}" "postgres://"

# ---- Test 6: lib/pq backslash-escaped quote leaks nothing -------------------
escaped_dsn="host=127.0.0.1 user=u password='it\\'s ${SENTINEL}' dbname=postgres"
run_launcher TRAGGO_TEST_POSTGRES_DSN="${escaped_dsn}"
assert_eq "${LAUNCHER_EXIT}" "0"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"

# ---- Test 7: wrong major is rejected (disposable path) ----------------------
run_launcher FAKE_GO_MAJOR=16
assert_eq "${LAUNCHER_EXIT}" "1"
assert_contains "${LAUNCHER_STDERR}" "major 16, but major 17 or 18 is required"
# Cleanup still runs for the self-started container after wrong-major failure.
assert_eq "$(grep -c 'docker rm -f traggo-test-pg-' "${DOCKER_LOG}" || true)" "1"

# ---- Test 8: wrong major is rejected (reused path) --------------------------
run_launcher TRAGGO_TEST_POSTGRES_DSN="postgres://u:${SENTINEL}@127.0.0.1:55432/postgres?sslmode=disable" FAKE_GO_MAJOR=19
assert_eq "${LAUNCHER_EXIT}" "1"
assert_contains "${LAUNCHER_STDERR}" "major 19, but major 17 or 18 is required"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"
assert_eq "$(grep -c 'docker rm -f' "${DOCKER_LOG}" || true)" "0"

# ---- Test 9: setup failure (container never ready) --------------------------
run_launcher FAKE_DOCKER_NOT_READY=1
assert_eq "${LAUNCHER_EXIT}" "1"
assert_contains "${LAUNCHER_STDERR}" "did not become ready"

# ---- Test 10: unusable connection (version probe fails) leaks nothing -------
run_launcher TRAGGO_TEST_POSTGRES_DSN="postgres://u:${SENTINEL}@127.0.0.1:55432/postgres?sslmode=disable" FAKE_GO_RUN_EXIT=1
assert_eq "${LAUNCHER_EXIT}" "1"
assert_contains "${LAUNCHER_STDERR}" "could not determine the connected PostgreSQL server version"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"
assert_not_contains "${LAUNCHER_OUT}" "postgres://"

# ---- Test 11: child-emitted credential is redacted on success ---------------
run_launcher TRAGGO_TEST_POSTGRES_DSN="postgres://u:${SENTINEL}@127.0.0.1:55432/postgres?sslmode=disable" FAKE_CHILD_SECRET="${SENTINEL}"
assert_eq "${LAUNCHER_EXIT}" "0"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"

# ---- Test 12: child-emitted credential is redacted on test failure ----------
run_launcher TRAGGO_TEST_POSTGRES_DSN="postgres://u:${SENTINEL}@127.0.0.1:55432/postgres?sslmode=disable" FAKE_CHILD_SECRET="${SENTINEL}" FAKE_GO_TEST_EXIT=3
assert_eq "${LAUNCHER_EXIT}" "3"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"

# ---- Test 13: JSON mode keeps events on stdout, stages on stderr ------------
run_launcher TRAGGO_TEST_GO_JSON=1
assert_eq "${LAUNCHER_EXIT}" "0"
assert_contains "$(cat "${GO_LOG}")" "go test -count=1 -json"
assert_contains "${LAUNCHER_STDOUT}" '{"Action":"run","Test":"TestX_postgres"}'
assert_contains "${LAUNCHER_STDOUT}" '{"Action":"pass","Test":"TestY_sqlite"}'
# Stage messages are on stderr, not stdout, in JSON mode.
assert_contains "${LAUNCHER_STDERR}" "Starting disposable"
assert_not_contains "${LAUNCHER_STDOUT}" "Starting disposable"

# ---- Test 14: PG_PACKAGES override is honored -------------------------------
run_launcher TRAGGO_TEST_PG_PACKAGES="./database/... ./model/..."
assert_eq "${LAUNCHER_EXIT}" "0"
assert_contains "$(cat "${GO_LOG}")" "go test -count=1 ./database/... ./model/..."
assert_not_contains "$(cat "${GO_LOG}")" "./statistics/..."

# ---- Test 15: default package list is the six shared suites -----------------
run_launcher
assert_contains "$(cat "${GO_LOG}")" "./statistics/... ./device/... ./dashboard/... ./test/... ./database/... ./model/..."

# ---- Additional credential-boundary and supported-major regressions --------
for major in 17 18; do
  run_launcher FAKE_GO_MAJOR="${major}"
  assert_eq "${LAUNCHER_EXIT}" "0"
done

for dsn in \
  "host=localhost user=u password='synthetic\\'quoted-secret'" \
  "postgres://u@localhost/db?password=synthetic%27quoted-secret" \
  "host=localhost user=u password = synthetic\\'quoted-secret"; do
  for status in 0 7; do
    run_launcher TRAGGO_TEST_POSTGRES_DSN="${dsn}" FAKE_CHILD_SECRET="synthetic'quoted-secret" FAKE_GO_TEST_EXIT="${status}" TRAGGO_TEST_GO_JSON=1
    assert_eq "${LAUNCHER_EXIT}" "${status}"
    assert_not_contains "${LAUNCHER_OUT}" "synthetic'quoted-secret"
    assert_not_contains "${LAUNCHER_OUT}" "${dsn}"
    printf '%s\n' "${LAUNCHER_STDOUT}" | python3 -c 'import json, sys; [json.loads(line) for line in sys.stdin if line.strip()]'
  done
done

# JSON escaping must not hide a password from the filter or corrupt events.
json_secret=$'synthetic"quoted\\secret\nnext-line'
run_launcher TRAGGO_TEST_POSTGRES_DSN='postgres://u@localhost/db?password=synthetic%22quoted%5Csecret%0Anext-line' FAKE_CHILD_SECRET="${json_secret}" TRAGGO_TEST_GO_JSON=1
assert_eq "${LAUNCHER_EXIT}" "0"
assert_not_contains "${LAUNCHER_OUT}" "synthetic"
printf '%s\n' "${LAUNCHER_STDOUT}" | python3 -c 'import json, sys; [json.loads(line) for line in sys.stdin if line.strip()]'

run_launcher TRAGGO_TEST_POSTGRES_DSN="postgres://u:${SENTINEL}@localhost/db" FAKE_GO_RUN_EXIT=9 FAKE_CHILD_SECRET="${SENTINEL}"
assert_eq "${LAUNCHER_EXIT}" "9"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"
assert_contains "${LAUNCHER_STDERR}" "could not determine"
# The real helper receives connection configuration through env, never argv.
assert_not_contains "$(cat "${GO_LOG}")" "${SENTINEL}"

# Unparseable input suppresses child diagnostics and fails closed.
run_launcher TRAGGO_TEST_POSTGRES_DSN="host=localhost password='unterminated" FAKE_CHILD_SECRET="unterminated"
assert_eq "${LAUNCHER_EXIT}" "1"
assert_contains "${LAUNCHER_STDERR}" "output suppressed"
assert_not_contains "${LAUNCHER_OUT}" "unterminated"

# ---- T005: failure injection must still reject the reviewed defects ---------
: >"${DOCKER_LOG}"
# A loopback-only mapping must not be flagged.
printf '%s\n' "docker run -d --name x -p 127.0.0.1:55432:5432 postgres:18" >>"${DOCKER_LOG}"
if has_broad_publish; then
  fail "loopback-only publish was mis-detected as broad"
fi
# A broad mapping (the reviewed defect) MUST be detected.
printf '%s\n' "docker run -d --name y -p 0.0.0.0:55432:5432 postgres:18" >>"${DOCKER_LOG}"
if ! has_broad_publish; then
  fail "failure injection: broad publish mapping was not detected"
fi

# A child that emits the complete credential-bearing DSN must be redacted too.
run_launcher TRAGGO_TEST_POSTGRES_DSN="postgres://u:${SENTINEL}@127.0.0.1:55432/postgres?sslmode=disable" \
  FAKE_CHILD_SECRET="postgres://u:${SENTINEL}@127.0.0.1:55432/postgres?sslmode=disable"
assert_eq "${LAUNCHER_EXIT}" "0"
assert_not_contains "${LAUNCHER_OUT}" "${SENTINEL}"
assert_not_contains "${LAUNCHER_OUT}" "postgres://u:${SENTINEL}"

# The version-probe helper file must NOT be a dotfile (leading '.' or '_').
# Go ignores such filenames, which would silently break the real `go run` of
# the PG17 server_version_num gate (the fake `go` double would mask it).
helper_assign="$(grep -oE 'helper="\$\{SCRIPT_DIR\}/[^"]+\.go"' "${LAUNCHER}" | head -1)"
if [[ -z "${helper_assign}" ]]; then
  fail "could not locate the version-probe helper assignment in the launcher"
fi
helper_name="${helper_assign##*/}"
helper_name="${helper_name%\"}"
case "${helper_name}" in
  .*|_*)
    fail "version-probe helper must not be a dotfile/underscore filename, got [${helper_name}]"
    ;;
esac

echo "test-databases_test.sh: ${TESTS_RUN} cases run, ${TESTS_FAILED} failed, ${TESTS_SKIPPED} skipped"
if [[ "${TESTS_FAILED}" -ne 0 ]]; then
  exit 1
fi
exit 0