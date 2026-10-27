---
affected_files: []
cycle_number: 2
mission_slug: postgres-dashboard-review-fixes-01M5ZJTF
reproduction_command:
reviewed_at: '2026-10-27T22:37:03Z'
reviewer_agent: user
wp_id: WP01
---

# WP01 Review Feedback (cycle 2) — discovered via WP05 real execution

## Defect (blocking)
`hack/test-databases.sh` line ~191 names the version-probe helper `.pgversion-$$.go` — a leading-dot (dotfile) filename. The Go toolchain ignores files/dirs starting with `.` and `_`, so the real `go run` of this helper fails with "no Go files in <dir>" on BOTH the disposable and reused-server paths. This silently breaks the actual PG17 `server_version_num` gate that WP01's T004 is meant to enforce.

The shell regression suite (`hack/test-databases_test.sh`) does NOT catch this because it uses a fake `go` command double, so the real `go run` failure is masked. WP05 demonstrated the real fixtures pass on postgres:18 (exit 0) directly — which is exactly why the broken gate matters: without the version check, any PostgreSQL major would be accepted.

## Required fix
Rename the version-probe helper file to a non-dotfile name (e.g. `pgversion_$$.go` or `pgversion-$$.go` — no leading dot) so the Go toolchain actually compiles and runs it. Verify the real `go run` path works. Keep all other WP01 behavior unchanged.

## Scope
- Only fix the dotfile naming defect in `hack/test-databases.sh`.
- Update the shell regression test only if needed to assert the helper filename is not a dotfile.
- Do NOT change WP02/WP03/WP04 files (out of scope).
