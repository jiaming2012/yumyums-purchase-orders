#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# verify-test-harness.sh — the two reds of card H1 `test-harness-fail-loud`,
# made re-runnable.
#
# This script does not test the product. It tests whether the TEST HARNESS is
# capable of reporting failure at all. Both checks below were RED on
# `overnight-20260729-2`'s base commit, and both describe a way this repo's
# suite has already announced success while measuring nothing:
#
#   A. `task test` omitted `bdd:gen` from its `deps`, so a fresh worktree ran
#      19 of 20 spec files — one whole Playwright project contributing zero
#      tests, with no error and no skip line. (BACKLOG B-09, ledger T-25 d.73)
#
#   B. `DB_TEST_URL` set but unreachable was a `t.Skip`, and a non-verbose
#      `go test` prints a bare `ok <pkg> 2.9s` for a package whose every test
#      skipped. A reviewer DROPPING a card's database mid-run was therefore
#      indistinguishable from that card passing. (BACKLOG B-16, ledger T-27
#      d.90)
#
# Run it from the repo root:  bash scripts/verify-test-harness.sh
# Exit status is the verdict. Nothing here is piped, so nothing here can have
# its exit status swallowed by a `| tail`.
# ─────────────────────────────────────────────────────────────────────────────

set -u

cd "$(dirname "$0")/.."
REPO_ROOT="$PWD"

# `go` is not on PATH in non-login shells on this host; a Playwright webServer
# already died once with `go: not found` and the run still reported green.
export PATH=/usr/local/go/bin:$PATH

FAILURES=0
pass() { printf '  \033[32mPASS\033[0m  %s\n' "$1"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; FAILURES=$((FAILURES + 1)); }

# ── Check A — `task test` must generate the BDD specs before running ────────
#
# `task --dry` resolves and prints the full dependency chain without executing
# it, so this check is instant and needs no database and no browser.
echo "── A · task test regenerates the BDD specs ─────────────────────────────"
DRY_OUT="$(task --dry test 2>&1)"
if printf '%s' "$DRY_OUT" | grep -q 'npx bddgen'; then
	pass "task test's dependency chain runs 'npx bddgen'"
else
	fail "task test's dependency chain does NOT run 'npx bddgen' — the generated
        spec file under .features-gen/ is absent on a clean worktree, so the
        suite silently runs 19 of 20 spec files and reports success."
fi

# Corroborating count, reported (not graded — it needs node_modules present).
if [ -d "$REPO_ROOT/node_modules/@playwright" ]; then
	echo "    spec files Playwright currently resolves:"
	npx playwright test --list --reporter=list 2>/dev/null | grep -E '^Total:' | sed 's/^/      /'
fi

# ── Check B — a set-but-unreachable DB_TEST_URL must FAIL, not skip ─────────
#
# The DSN below names a database that DOES NOT EXIST on a Postgres that DOES.
# That is deliberate, and it is the exact shape of the B-16 incident: a
# reviewer ran `DROP DATABASE hq_test_go_c` while the implementer was still
# using it, and the implementer's suite went on reporting `ok`.
#
# It is also the FAST reproduction. `pgxpool.New` is lazy, so the failure
# surfaces at `Ping` either way — but pointing at a genuinely dead *port*
# (e.g. 127.0.0.1:5599) does not fail fast on this WSL2 host: closed ports
# there black-hole the SYN instead of refusing it, so every package sat for
# ~120s before reporting `ok` anyway. Measured 2026-07-29, pre-fix:
#
#     ok  github.com/yumyums/hq/internal/workflow   120.047s
#     ok  github.com/yumyums/hq/internal/receipt    120.168s
#
# Same verdict, 500x slower, and the 120s wait is an artifact of the host
# rather than of the harness. Set H1_DEAD_PORT=1 to use that variant anyway.
echo
echo "── B · DB_TEST_URL set + unreachable ⇒ non-zero exit ───────────────────"
if [ "${H1_DEAD_PORT:-0}" = "1" ]; then
	DEAD_URL='postgres://yumyums:yumyums@127.0.0.1:5599/hq_test_go_does_not_exist?sslmode=disable'
	echo "    (H1_DEAD_PORT=1 — dead port; expect ~120s per package on this host)"
else
	DEAD_URL='postgres://yumyums:yumyums@127.0.0.1:5433/hq_test_go_dropped_by_a_reviewer?sslmode=disable'
fi

# The eight sites this card converted, by package. `-count=1` defeats the test
# cache: a cached `ok` from a run with a LIVE database would otherwise be
# replayed here and hand us a false green inside the very script written to
# catch false greens.
DB_PKGS="./internal/workflow/ ./internal/receipt/ ./internal/inventory/ ./internal/auth/ ./internal/purchasing/ ./internal/recipes/ ./internal/sync/"

cd "$REPO_ROOT/backend"
# shellcheck disable=SC2086
DB_TEST_URL="$DEAD_URL" DATABASE_URL='' TEST_DATABASE_URL='' \
	go test -count=1 -p 1 $DB_PKGS >/tmp/h1-deadport.log 2>&1
DEAD_STATUS=$?
cd "$REPO_ROOT"

if [ "$DEAD_STATUS" -ne 0 ]; then
	pass "go test exited $DEAD_STATUS with DB_TEST_URL=$DEAD_URL"
else
	fail "go test exited 0 with DB_TEST_URL=$DEAD_URL — a dropped or unreachable
        database reports 'ok'. Destroying the test environment is
        indistinguishable from passing it."
fi
echo "    (per-package output: /tmp/h1-deadport.log)"

# ── Check B2 — the UNSET case must still SKIP, not fail ─────────────────────
#
# The bug is the SYMMETRY, not the skip. A contributor with no database must
# still be able to run the tests that need none. This check is the guard on
# over-correcting: it is red if someone converts the unset case too.
echo
echo "── B2 · DB_TEST_URL unset ⇒ still skips (no over-correction) ───────────"
cd "$REPO_ROOT/backend"
env -u DB_TEST_URL -u DATABASE_URL -u TEST_DATABASE_URL \
	go test -count=1 -p 1 ./internal/recipes/ ./internal/sync/ >/tmp/h1-unset.log 2>&1
UNSET_STATUS=$?
cd "$REPO_ROOT"

if [ "$UNSET_STATUS" -eq 0 ]; then
	pass "go test exited 0 with DB_TEST_URL unset (skip-on-unset preserved)"
else
	fail "go test exited $UNSET_STATUS with DB_TEST_URL unset — skip-on-unset was
        over-corrected into a failure. A contributor without a database must
        still be able to run the unit tests. See /tmp/h1-unset.log."
fi

echo
if [ "$FAILURES" -eq 0 ]; then
	printf '\033[32mharness OK\033[0m — the suite is capable of reporting failure.\n'
	exit 0
fi
printf '\033[31m%d harness check(s) FAILED\033[0m — a green suite from this tree does not mean what it says.\n' "$FAILURES"
exit 1
