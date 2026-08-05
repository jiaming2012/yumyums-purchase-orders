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
# Check B grades all seven DB-backed packages INDIVIDUALLY as of run 20260806.
# Before that it graded their disjunction — one aggregate `go test` over the
# whole list, passing on any non-zero exit — so six of the seven could lose
# fail-loud with the check still printing PASS. See the block above Check B
# for the demonstration. (BACKLOG B-22.)
#
# A2 and B2 are the guards on the two fixes over-shooting: A2 grades the
# spec-file COUNT (A grades only the mechanism, and a mechanism that runs and
# emits nothing still passes A); B2 grades that the *unset* case still skips
# across every site the fix touched. Both were added in the H1 repair round
# after G6 falsified the originals. Each was itself falsified before it shipped
# — A2 by raising its floor, B2 by G6's `requested :=` reordering.
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

# ── Check A2 — and the spec-file count must actually be the whole repo ──────
#
# Check A grades the MECHANISM (`bddgen` is in the dep chain). A2 grades the
# PROPERTY the mechanism exists to deliver: *the suite runs every spec file the
# repo has*. Those are not the same assertion. A tree where `bddgen` ran and
# emitted zero `.feature` files — a moved features dir, a renamed glob, a
# generator that exits 0 on an empty input set — passes Check A and is exactly
# the 19-of-20 failure A was written to catch, wearing a green badge.
#
# `bddgen` is run first, deliberately: it is idempotent (~2s), it is what
# `task test`'s dep chain now does, and without it the count on a FRESH
# worktree would be 19 for the very reason Check A already covers. A2 is the
# floor on the generated tree, not a second copy of A.
#
# The floor is 20 (19 static under ./tests + 1 generated under .features-gen/).
# It ratchets UP as spec files are added; it must never ratchet down silently.
# H1_MIN_SPEC_FILES overrides it — for proving this check can go red, and for
# nothing else.
MIN_SPEC_FILES="${H1_MIN_SPEC_FILES:-20}"
echo
echo "── A2 · Playwright resolves at least $MIN_SPEC_FILES spec files ────────────────────"
if [ ! -d "$REPO_ROOT/node_modules/@playwright" ]; then
	fail "node_modules/@playwright is absent, so the spec-file count cannot be
        taken. This is a FAIL and not a skip: an ungraded check is not a
        passed check. Run 'npm ci' and re-run."
else
	npx bddgen >/tmp/h1-bddgen.log 2>&1
	npx playwright test --list --reporter=list >/tmp/h1-list.log 2>&1
	TOTAL_LINE="$(grep -E '^Total: [0-9]+ tests in [0-9]+ files' /tmp/h1-list.log | tail -1)"
	SPEC_FILES="$(printf '%s' "$TOTAL_LINE" | sed -nE 's/^Total: [0-9]+ tests in ([0-9]+) files.*/\1/p')"
	if [ -z "$SPEC_FILES" ]; then
		fail "'npx playwright test --list' printed no parseable 'Total: N tests in M
        files' line, so the spec-file count could not be graded. See
        /tmp/h1-list.log and /tmp/h1-bddgen.log."
	elif [ "$SPEC_FILES" -ge "$MIN_SPEC_FILES" ]; then
		pass "$TOTAL_LINE (floor: $MIN_SPEC_FILES)"
	else
		fail "$TOTAL_LINE — BELOW the floor of $MIN_SPEC_FILES. Spec files the repo has are
        not reaching the runner. A suite that silently resolves fewer files
        than exist reports success for tests it never ran; that is the B-09
        incident. See /tmp/h1-list.log."
	fi
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
#
# ── Why this check is a LOOP and not one `go test` (B-22) ───────────────────
#
# Until run 20260806 this check ran ONE aggregate invocation over all seven
# packages and passed on `DEAD_STATUS -ne 0`. `go test` exits non-zero if ANY
# package fails, so that assertion was a DISJUNCTION: it graded "at least one
# of the seven still fails loud", while the property the check exists to
# defend is "ALL seven still fail loud". Six of the seven could lose the gate
# and this check still printed PASS.
#
# That is not a hypothetical. Demonstrated on 20260806 by removing fail-loud
# from six packages — `requested := dbURL != ""` -> `requested := false` in
# the four TestMains (receipt, inventory, auth, purchasing) and
# `t.Fatal(testdb.Reason(...))` -> `t.Skip(...)` in the two helper packages
# (recipes, sync), leaving only internal/workflow honest. Probed one by one
# the mutated tree read:
#
#     workflow exit 1 · receipt exit 0 · inventory exit 0 · auth exit 0
#     purchasing exit 0 · recipes exit 0 · sync exit 0
#
# and the aggregate check reported, verbatim:
#
#     PASS  go test exited 1 with DB_TEST_URL=postgres://…/hq_test_go_dropped_by_a_reviewer
#
# A check written to catch a harness that reports success while measuring
# nothing was itself reporting success while measuring one seventh of what it
# claimed. This is exactly the shape of the two-package Check B2 that G6
# falsified during H1 — same defect, other check. (BACKLOG B-22.)
#
# The loop is affordable precisely because the DSN above is the live-server /
# missing-database variant: ~0.02s of Postgres round-trip per package, so the
# seven cost about as much as the one did. Under H1_DEAD_PORT=1 the loop costs
# seven black-holed TCP timeouts (~110s each, ~13min); that variant is
# diagnostic only and stays opt-in for that reason.
echo
echo "── B · DB_TEST_URL set + unreachable ⇒ non-zero exit, EVERY package ────"
if [ "${H1_DEAD_PORT:-0}" = "1" ]; then
	DEAD_URL='postgres://yumyums:yumyums@127.0.0.1:5599/hq_test_go_does_not_exist?sslmode=disable'
	echo "    (H1_DEAD_PORT=1 — dead port; expect ~120s PER PACKAGE on this host)"
else
	DEAD_URL='postgres://yumyums:yumyums@127.0.0.1:5433/hq_test_go_dropped_by_a_reviewer?sslmode=disable'
fi

# The eight sites this card converted, by package. `-count=1` defeats the test
# cache: a cached `ok` from a run with a LIVE database would otherwise be
# replayed here and hand us a false green inside the very script written to
# catch false greens.
DB_PKGS="./internal/workflow/ ./internal/receipt/ ./internal/inventory/ ./internal/auth/ ./internal/purchasing/ ./internal/recipes/ ./internal/sync/"

# The number of packages $DB_PKGS is expected to carry, asserted separately
# below. Without it, deleting entries from $DB_PKGS SHRINKS both Check B and
# Check B2 in silence and both keep printing PASS — a check that grades fewer
# things than it used to, reporting the same green. That is the identical
# failure mode to the one this card just fixed, one level up, so it is graded
# rather than trusted. One assertion covers both checks because they
# deliberately share the list (see Check B2's comment).
#
# It ratchets UP as DB-backed packages are added. H1_DB_PKG_COUNT overrides it
# — for proving this assertion can go red, and for nothing else.
EXPECTED_DB_PKGS="${H1_DB_PKG_COUNT:-7}"

cd "$REPO_ROOT/backend"
DEAD_PROBED=0
DEAD_SILENT=""
for p in $DB_PKGS; do
	DEAD_PROBED=$((DEAD_PROBED + 1))
	PKG_LOG="/tmp/h1-deadport-$(basename "$p").log"
	DB_TEST_URL="$DEAD_URL" DATABASE_URL='' TEST_DATABASE_URL='' \
		go test -count=1 -p 1 "$p" >"$PKG_LOG" 2>&1
	PKG_STATUS=$?
	if [ "$PKG_STATUS" -ne 0 ]; then
		printf '    %-26s exit %-3s fails loud\n' "$p" "$PKG_STATUS"
	else
		printf '    %-26s exit %-3s REPORTED ok  <-- silent\n' "$p" "$PKG_STATUS"
		DEAD_SILENT="$DEAD_SILENT $p"
	fi
done
cd "$REPO_ROOT"

if [ -z "$DEAD_SILENT" ]; then
	pass "all $DEAD_PROBED DB-backed packages exited non-zero, individually, with
        DB_TEST_URL=$DEAD_URL"
else
	fail "these packages exited 0 with DB_TEST_URL=$DEAD_URL:$DEAD_SILENT
        For them a dropped or unreachable database reports 'ok'. Destroying the
        test environment is indistinguishable from passing it. Note that an
        AGGREGATE 'go test' over all $DEAD_PROBED packages would still have exited
        non-zero here, on the strength of the packages that DID fail, and would
        have printed PASS — that disjunction is B-22 and is why this check
        iterates."
fi

# The count assertion is deliberately its own graded line. Folded into the loop
# above it would be vacuous: a $DB_PKGS trimmed to one package makes "all 1
# packages exited non-zero" true.
if [ "$DEAD_PROBED" -eq "$EXPECTED_DB_PKGS" ]; then
	pass "Check B iterated $DEAD_PROBED packages (expected $EXPECTED_DB_PKGS)"
else
	fail "Check B iterated $DEAD_PROBED package(s) but expected $EXPECTED_DB_PKGS. \$DB_PKGS has
        changed size. If a DB-backed package was legitimately added or removed,
        update EXPECTED_DB_PKGS in the same commit and say why; if it was not,
        this check — and Check B2, which shares the list — has quietly narrowed
        the ground it grades while still printing PASS."
fi
echo "    (per-package output: /tmp/h1-deadport-<pkg>.log)"

# ── Check B2 — the UNSET case must still SKIP, not fail ─────────────────────
#
# The bug is the SYMMETRY, not the skip. A contributor with no database must
# still be able to run the tests that need none. This check is the guard on
# over-correcting: it is red if someone converts the unset case too.
#
# It runs the SAME $DB_PKGS list as Check B, and that is the point. An earlier
# revision of this script ran only ./internal/recipes/ ./internal/sync/ — the
# two HELPER-based packages, whose unset path was structurally unchanged by
# this card (they t.Skip on unset *before* they ever touch the DSN, so they
# cannot regress). The five TestMain packages, which this check omitted, are
# precisely where the unset path is delicate: each one computes
#
#     requested := dbURL != ""
#
# and that line MUST run BEFORE the `if dbURL == "" { dbURL = <fallback> }`.
# Move it one line down and `requested` becomes true for everybody, the
# fallback DSN is treated as a statement of intent, and a contributor with no
# database gets a hard failure instead of a skip. That mutation was applied to
# receipt/worker_test.go and the two-package version of this check still
# printed PASS — the over-correction B2 exists to catch, reported green. The
# list below is the fix.
echo
echo "── B2 · DB_TEST_URL unset ⇒ still skips (no over-correction) ───────────"
cd "$REPO_ROOT/backend"
# shellcheck disable=SC2086
env -u DB_TEST_URL -u DATABASE_URL -u TEST_DATABASE_URL \
	go test -count=1 -p 1 $DB_PKGS >/tmp/h1-unset.log 2>&1
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
