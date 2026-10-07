# _k2lib.sh — the commands behind every log in this directory (card K2,
# dish-merge-shapes-and-backstop-tests, run 20261007). Sourced from the worktree root.
# Go database hq_test_go_k2 on :5434 (yumyums-test-pg, role hqtest). NEVER :5433.
export PATH="/usr/local/go/bin:$PATH"
export PGPASSWORD=hqtest
K2_DB=hq_test_go_k2
K2_URL="postgres://hqtest:hqtest@localhost:5434/$K2_DB?sslmode=disable"
K2_LOG=.night-crew/runs/2026-10-07-autonomous/logs/dish-merge-shapes-and-backstop-tests
K2_TMP=/tmp/k2-20261007
K2_PG=(psql -X -h localhost -p 5434 -U hqtest -d postgres -v ON_ERROR_STOP=1 -qtA)
mkdir -p "$K2_TMP"

k2_fresh_db() {
  "${K2_PG[@]}" -c "drop database if exists $K2_DB with (force)" >/dev/null
  "${K2_PG[@]}" -c "create database $K2_DB" >/dev/null
}
# The recipes package's TestMain does not migrate: boot the built server once until it has
# (the spike's migrate_go_db recipe), then stop it.
k2_migrate() {
  ( cd backend && go build -o "$K2_TMP/hqserver" ./cmd/server/ ) || return 2
  ( cd backend && exec env PORT=8345 DB_URL="$K2_URL" \
      STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
      MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= \
      SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= STORAGE_KEY= STORAGE_SECRET= STORAGE_BUCKET= STORAGE_REGION= STORAGE_ENDPOINT= \
      "$K2_TMP/hqserver" ) >"$K2_TMP/server.log" 2>&1 &
  local pid=$! i
  for i in $(seq 1 120); do
    grep -q "database migrations applied successfully" "$K2_TMP/server.log" 2>/dev/null && break
    kill -0 "$pid" 2>/dev/null || { tail -20 "$K2_TMP/server.log" >&2; return 2; }
    sleep 0.5
  done
  kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
  grep -q "database migrations applied successfully" "$K2_TMP/server.log" || return 2
  echo "# migrated $K2_DB: goose version $(psql -X -h localhost -p 5434 -U hqtest -d "$K2_DB" -qtA -c 'select max(version_id) from goose_db_version')"
}
# k2_run <log name> <pkg> <-run regex>: writes $K2_LOG/<name>.log with the command, the full
# -v output and the exit line.
k2_run() {
  local name="$1" pkg="$2" run="$3" rc=0 out="$K2_LOG/$1.log"
  {
    echo "# $(date -u +%FT%TZ) · tree: $(git rev-parse --short HEAD) + working tree · db: $K2_URL"
    echo "# \$ cd backend && DB_TEST_URL=$K2_URL go test $pkg -run '$run' -count=1 -v"
  } >"$out"
  ( cd backend && DB_TEST_URL="$K2_URL" go test "$pkg" -run "$run" -count=1 -v ) >>"$out" 2>&1 || rc=$?
  echo "# exit=$rc pass=$(grep -c '^--- PASS' "$out") fail=$(grep -c '^--- FAIL' "$out") skip=$(grep -c '^--- SKIP' "$out")" | tee -a "$out"
  grep -E "^(--- (PASS|FAIL|SKIP)|ok|FAIL|\[build failed\])" "$out" | sed 's/^/#   /'
  return 0
}
