# lib.sh — shared plumbing for the mms-send-on-signup spikes. Sourced, not run.
#
# Both spikes work in a THROWAWAY git worktree off `dev` (../hq-worktrees/spike-e2-20261006) and
# run Go tests in the worktree against a FRESH spike-owned database hq_test_spike_e2_go on :5434
# (yumyums-test-pg, role hqtest) — internal/marketing's TestMain migrates it. The worktree is
# removed and the database dropped on EVERY exit path (trap). No spike talks to any provider.
# 🛑 NEVER :5433 — that is the dev AND production cluster (decision 155).
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
WT="$(dirname -- "$REPO_ROOT")/hq-worktrees/spike-e2-20261006"
GO_DB="hq_test_spike_e2_go"
GO_URL="postgres://hqtest:hqtest@localhost:5434/$GO_DB?sslmode=disable"
PG=(psql -X -h localhost -p 5434 -U hqtest -d postgres -v ON_ERROR_STOP=1 -qtA)
export PGPASSWORD="${PGPASSWORD:-hqtest}"
export PATH="/usr/local/go/bin:$PATH"
LOG="${SPIKE_LOG_DIR:-$(mktemp -d /tmp/spike-e2.XXXXXX)}"
T0=$(date +%s)
MAIN_SHA=""

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
leg()        { printf '\n── %s ──\n' "$1"; }

cleanup() {
  local rc=$?
  set +e
  if [ -d "$WT" ]; then
    git -C "$REPO_ROOT" worktree remove --force "$WT" >/dev/null 2>&1 || rm -rf "$WT"
    git -C "$REPO_ROOT" worktree prune >/dev/null 2>&1
  fi
  "${PG[@]}" -c "drop database if exists $GO_DB with (force)" >/dev/null 2>&1
  printf '# cleanup: worktree %s removed=%s; %s dropped; logs in %s; wall-clock %ss\n' \
    "$WT" "$([ -d "$WT" ] && echo no || echo yes)" "$GO_DB" "$LOG" "$(( $(date +%s) - T0 ))"
  exit "$rc"
}

preflight() {
  command -v go   >/dev/null || cannot_run "go not on PATH (export PATH=/usr/local/go/bin:\$PATH)"
  command -v psql >/dev/null || cannot_run "psql not on PATH"
  "${PG[@]}" -c 'select 1' >/dev/null 2>&1 || cannot_run "yumyums-test-pg (:5434, role hqtest) not reachable — task test:db:up"
  local who; who="$("${PG[@]}" -c "select current_user || '@' || inet_server_port()")"
  [ "$who" = "hqtest@5432" ] || [ "$who" = "hqtest@5434" ] || cannot_run "connected as $who — expected the hqtest role on the test container"
  git -C "$REPO_ROOT" rev-parse --verify -q dev >/dev/null || cannot_run "no local branch 'dev'"
  MAIN_SHA="$(git -C "$REPO_ROOT" rev-parse HEAD)"
  echo "# target coordinates (read-only statement before any write):"
  echo "#   worktree : $WT  (detached, off $(git -C "$REPO_ROOT" rev-parse --short dev) = dev)"
  echo "#   go db    : $GO_URL  (fresh per leg; TestMain migrates)"
  echo "#   logs     : $LOG"
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
  echo "#   HEAD=$(git -C "$WT" rev-parse --short HEAD)"
}

fresh_go_db() {
  "${PG[@]}" -c "drop database if exists $GO_DB with (force)" >/dev/null
  "${PG[@]}" -c "create database $GO_DB" >/dev/null
}
# go_run <label> <pkg> <-run regex> → echoes exit; log in $LOG/<label>.log
go_run() {
  local label="$1" pkg="$2" run="$3" rc=0
  ( cd "$WT/backend" && DB_TEST_URL="$GO_URL" go test "$pkg" -run "$run" -count=1 -v ) >"$LOG/$label.log" 2>&1 || rc=$?
  echo "$rc"
}
summary() { grep -E "^(--- (PASS|FAIL|SKIP)|ok|FAIL|PASS|\[build failed\])" "$LOG/$1.log" | sed 's/^/#   /'; }
passes()  { grep -c "^--- PASS" "$LOG/$1.log" || true; }
fails()   { grep -c "^--- FAIL" "$LOG/$1.log" || true; }
skips()   { grep -c "^--- SKIP" "$LOG/$1.log" || true; }
built()   { ! grep -q "^\[build failed\]\|cannot find package\|undefined:\|syntax error" "$LOG/$1.log"; }

main_tree_untouched() {
  [ "$(git -C "$REPO_ROOT" rev-parse HEAD)" = "$MAIN_SHA" ] || fail "main tree HEAD moved during the spike"
  [ -z "$(git -C "$REPO_ROOT" status --porcelain -- backend)" ] || fail "main tree backend/ is dirty — a mutation escaped the worktree"
}
