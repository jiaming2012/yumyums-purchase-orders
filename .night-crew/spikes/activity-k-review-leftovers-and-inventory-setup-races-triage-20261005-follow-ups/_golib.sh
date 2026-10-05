# _golib.sh — shared plumbing for Activity K's Go spikes. Sourced, not run.
# The sourcing script sets SPIKE_TAG (k2) BEFORE sourcing.
#
# Works in a THROWAWAY git worktree off `dev` (../hq-worktrees/spike-<tag>-20261007) and runs Go
# tests in the worktree against a FRESH spike-owned database hq_test_spike_<tag>_go on :5434
# (yumyums-test-pg, role hqtest) — the package's TestMain migrates it. Worktree removed and database
# dropped on EVERY exit path (trap). 🛑 NEVER :5433 (decision 155).
: "${SPIKE_TAG:?set SPIKE_TAG before sourcing}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[1]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
WT="$(dirname -- "$REPO_ROOT")/hq-worktrees/spike-${SPIKE_TAG}-20261007"
GO_DB="hq_test_spike_${SPIKE_TAG}_go"
GO_URL="postgres://hqtest:hqtest@localhost:5434/$GO_DB?sslmode=disable"
PG=(psql -X -h localhost -p 5434 -U hqtest -d postgres -v ON_ERROR_STOP=1 -qtA)
export PGPASSWORD="${PGPASSWORD:-hqtest}"
export PATH="/usr/local/go/bin:$PATH"
LOG="${SPIKE_LOG_DIR:-$(mktemp -d /tmp/spike-${SPIKE_TAG}.XXXXXX)}"
T0=$(date +%s)
MAIN_SHA=""

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
leg()        { printf '\n── %s ──\n' "$1"; }

cleanup() {
  local rc=$?
  set +e
  stop_server 2>/dev/null
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
  git -C "$REPO_ROOT" rev-parse --verify -q dev >/dev/null || cannot_run "no local branch 'dev'"
  [ -z "$(git -C "$REPO_ROOT" status --porcelain -- backend)" ] || cannot_run "main tree backend/ is dirty BEFORE the spike — commit the sitting's edits first (the E2 lesson)"
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
# migrate_go_db — packages whose TestMain does NOT migrate (recipes) need the schema applied
# first: boot the built server ONCE against the fresh database until "database migrations
# applied successfully", then stop it (the J2 recipe). Port MIG_PORT defaults to 8344.
MIG_PORT="${MIG_PORT:-8344}"
SERVER_PID=""
# `wait` on a TERMed child returns 143; under set -e that would abort the script, so it is swallowed.
stop_server() {
  if [ -n "$SERVER_PID" ]; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  SERVER_PID=""
  return 0
}
migrate_go_db() {
  [ -x "$LOG/hqserver" ] || ( cd "$WT/backend" && go build -o "$LOG/hqserver" ./cmd/server/ ) || cannot_run "go build ./cmd/server failed"
  if ss -ltn 2>/dev/null | grep -q ":$MIG_PORT "; then cannot_run "port $MIG_PORT already in use"; fi
  ( cd "$WT/backend" && exec env PORT=$MIG_PORT DB_URL="$GO_URL" \
      STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
      MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= \
      SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= STORAGE_KEY= STORAGE_SECRET= STORAGE_BUCKET= STORAGE_REGION= STORAGE_ENDPOINT= \
      "$LOG/hqserver" ) >"$LOG/server.log" 2>&1 &
  SERVER_PID=$!
  local i
  for i in $(seq 1 120); do
    grep -q "database migrations applied successfully" "$LOG/server.log" 2>/dev/null && break
    kill -0 "$SERVER_PID" 2>/dev/null || { tail -20 "$LOG/server.log" >&2; cannot_run "server exited before migrating"; }
    sleep 0.5
  done
  grep -q "database migrations applied successfully" "$LOG/server.log" || { tail -20 "$LOG/server.log" >&2; cannot_run "migrations did not finish in 60s"; }
  stop_server
  echo "#   migrated $GO_DB: goose version $(psql -X -h localhost -p 5434 -U hqtest -d "$GO_DB" -qtA -c "select max(version_id) from goose_db_version")"
}
go_run() { # <label> <pkg> <-run regex> → echoes exit; log in $LOG/<label>.log
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
