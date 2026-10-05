# _pwlib.sh — shared plumbing for Activity K's Playwright spikes. Sourced, not run.
# The sourcing script sets SPIKE_TAG (k1 | k3 | k4) and PW_PORT BEFORE sourcing.
#
# Each spike works in a THROWAWAY git worktree off `dev` (../hq-worktrees/spike-<tag>-20261007)
# so a spike-only spec never touches the main tree, and runs Playwright against a spike-owned
# stack: TEST_DB_NAME=hq_test_spike_<tag>_20261007, TEST_PORT=$PW_PORT, on :5434 (yumyums-test-pg,
# role hqtest). Worktree removed and database dropped on EVERY exit path (trap).
# 🛑 NEVER :5433 — that is the dev AND production cluster (decision 155).
: "${SPIKE_TAG:?set SPIKE_TAG before sourcing}"; : "${PW_PORT:?set PW_PORT before sourcing}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[1]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
WT="$(dirname -- "$REPO_ROOT")/hq-worktrees/spike-${SPIKE_TAG}-20261007"
PW_DB="hq_test_spike_${SPIKE_TAG}_20261007"
PG=(psql -X -h localhost -p 5434 -U hqtest -d postgres -v ON_ERROR_STOP=1 -qtA)
export PGPASSWORD="${PGPASSWORD:-hqtest}"
export PATH="/usr/local/go/bin:$PATH"
LOG="${SPIKE_LOG_DIR:-$(mktemp -d /tmp/spike-${SPIKE_TAG}.XXXXXX)}"
T0=$(date +%s)
MAIN_SHA=""

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
leg()        { printf '\n── %s ──\n' "$1"; }
exitline()   { printf '#   %-56s exit=%s  (expected %s)\n' "$1" "$2" "$3"; }

cleanup() {
  local rc=$?
  set +e
  if [ -d "$WT" ]; then
    git -C "$REPO_ROOT" worktree remove --force "$WT" >/dev/null 2>&1 || rm -rf "$WT"
    git -C "$REPO_ROOT" worktree prune >/dev/null 2>&1
  fi
  "${PG[@]}" -c "drop database if exists $PW_DB with (force)" >/dev/null 2>&1
  printf '# cleanup: worktree %s removed=%s; %s dropped; logs in %s; wall-clock %ss\n' \
    "$WT" "$([ -d "$WT" ] && echo no || echo yes)" "$PW_DB" "$LOG" "$(( $(date +%s) - T0 ))"
  exit "$rc"
}

preflight() {
  command -v go   >/dev/null || cannot_run "go not on PATH (export PATH=/usr/local/go/bin:\$PATH)"
  command -v node >/dev/null || cannot_run "node not on PATH"
  command -v psql >/dev/null || cannot_run "psql not on PATH"
  [ -d "$REPO_ROOT/node_modules/@playwright" ] || cannot_run "no node_modules in $REPO_ROOT (npm ci first)"
  [ -e "$REPO_ROOT/.night-crew/qa/spike-supabase/rxdb/node_modules/rxdb" ] || cannot_run "no vendored rxdb under .night-crew/qa/spike-supabase/rxdb/node_modules (B-477)"
  "${PG[@]}" -c 'select 1' >/dev/null 2>&1 || cannot_run "yumyums-test-pg (:5434, role hqtest) not reachable — task test:db:up"
  git -C "$REPO_ROOT" rev-parse --verify -q dev >/dev/null || cannot_run "no local branch 'dev'"
  if ss -ltn 2>/dev/null | grep -q ":$PW_PORT "; then cannot_run "port $PW_PORT already in use"; fi
  MAIN_SHA="$(git -C "$REPO_ROOT" rev-parse HEAD)"
  echo "# target coordinates (read-only statement before any write):"
  echo "#   worktree   : $WT  (detached, off $(git -C "$REPO_ROOT" rev-parse --short dev) = dev)"
  echo "#   playwright : TEST_DB_NAME=$PW_DB TEST_PORT=$PW_PORT  on :5434/hqtest"
  echo "#   logs       : $LOG"
}

worktree_up() {
  trap cleanup EXIT
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
  leg "warm build (backend, in the worktree)"
  ( cd "$WT/backend" && go build -o /dev/null ./cmd/server/ ) || cannot_run "backend did not compile in the worktree"
  echo "#   backend compiled; build cache warm for the Playwright legs"
}

pw_run() {
  local label="$1"; shift
  local rc=0
  ( cd "$WT" && TEST_DB_NAME="$PW_DB" TEST_PORT="$PW_PORT" PLAYWRIGHT_JSON_OUTPUT_NAME="$LOG/$label.json" \
      npx playwright test "$@" --retries=0 --reporter=line,json ) >"$LOG/$label.log" 2>&1 || rc=$?
  echo "$rc"
}
pw_outcomes() {
  node -e '
    const r = JSON.parse(require("fs").readFileSync(process.argv[1], "utf8"));
    const out = [];
    const walk = (s) => { (s.specs||[]).forEach(sp => out.push(`${sp.title.match(/^\[[A-Z0-9-]+[a-z]?\]/)?.[0] ?? sp.title}=${sp.ok}`)); (s.suites||[]).forEach(walk); };
    (r.suites||[]).forEach(walk);
    console.log(out.join("\n"));
  ' "$LOG/$1.json"
}
has() { grep -qxF -- "$2" <<<"$1"; }

main_tree_untouched() {
  [ "$(git -C "$REPO_ROOT" rev-parse HEAD)" = "$MAIN_SHA" ] || fail "main tree HEAD moved during the spike"
  [ -z "$(git -C "$REPO_ROOT" status --porcelain -- marketing tests backend inventory.html)" ] || fail "main tree marketing/, tests/, backend/ or inventory.html is dirty — a mutation escaped the worktree"
}
