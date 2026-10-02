# _lib.sh — shared plumbing for the atomic-scan-dedupe spikes. Sourced, not run.
# Creates the spike-owned database hq_test_spike_i4_20261003 on :5434 (yumyums-test-pg,
# role hqtest), migrates it with the repo's own goose runner (by booting the backend once
# and stopping it at "database migrations applied successfully"), seeds one user / one
# campaign / one qr_code, and DROPS the database on exit via trap. Never :5433.
SPIKE_DB="hq_test_spike_i4_20261003"
PGHOST=localhost; PGPORT=5434; PGUSER=hqtest; export PGHOST PGPORT PGUSER
export PGPASSWORD=hqtest
SPIKE_PORT=8314
SHORT="ABCDEF"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/spike-i4.XXXXXX")"
SERVER_PID=""
export PATH="/usr/local/go/bin:$PATH"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

q()  { psql -X -q -v ON_ERROR_STOP=1 -At -d "$SPIKE_DB" "$@"; }   # query the spike DB
qp() { psql -X -q -v ON_ERROR_STOP=1 -At -d postgres  "$@"; }     # query the maintenance DB

# stop the migrating server: TERM, poll up to 5s, then KILL (it is not our child, so wait() cannot reap it)
stop_server() {
  [ -n "$SERVER_PID" ] || return 0
  kill -TERM "$SERVER_PID" 2>/dev/null || true
  local i=0; while kill -0 "$SERVER_PID" 2>/dev/null && [ $i -lt 50 ]; do sleep 0.1; i=$((i+1)); done
  kill -0 "$SERVER_PID" 2>/dev/null && { kill -KILL "$SERVER_PID" 2>/dev/null || true; sleep 0.2; }
  kill -0 "$SERVER_PID" 2>/dev/null && echo "# WARNING: hq-server pid $SERVER_PID still alive" >&2
  SERVER_PID=""
}

cleanup() {
  local rc=$?
  stop_server
  [ "$PGPORT" = "5434" ] || { echo "refusing cleanup on port $PGPORT" >&2; exit 2; }
  qp -c "DROP DATABASE IF EXISTS $SPIKE_DB WITH (FORCE)" >/dev/null 2>&1 && echo "# dropped $SPIKE_DB" || echo "# WARNING: could not drop $SPIKE_DB" >&2
  rm -rf "$WORK"
  exit $rc
}

spike_db_up() {
  [ "$PGPORT" = "5434" ] || cannot_run "PGPORT is $PGPORT, must be 5434"
  qp -c "select 1" >/dev/null 2>&1 || cannot_run "cannot reach :5434 as hqtest (task test:db:up?)"
  qp -c "DROP DATABASE IF EXISTS $SPIKE_DB WITH (FORCE)" >/dev/null
  qp -c "CREATE DATABASE $SPIKE_DB" >/dev/null
  trap cleanup EXIT
  echo "# created $SPIKE_DB on :5434; building backend to migrate it"
  (cd "$REPO_ROOT/backend" && go build -o "$WORK/hq-server" ./cmd/server/) || cannot_run "go build failed"
  # `exec` so the backgrounded subshell BECOMES the server and $! is its pid — without it
  # `cd && server &` backgrounds a wrapper shell, and killing the wrapper orphans the server.
  (cd "$REPO_ROOT/backend" && exec env PORT=$SPIKE_PORT DB_URL="postgres://hqtest:hqtest@localhost:5434/$SPIKE_DB?sslmode=disable" \
     STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
     MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= \
     SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= STORAGE_KEY= STORAGE_SECRET= STORAGE_BUCKET= STORAGE_REGION= STORAGE_ENDPOINT= \
     "$WORK/hq-server") >"$WORK/server.log" 2>&1 &
  SERVER_PID=$!
  local i=0
  until grep -q "database migrations applied successfully" "$WORK/server.log"; do
    kill -0 "$SERVER_PID" 2>/dev/null || { tail -20 "$WORK/server.log" >&2; cannot_run "server exited before migrating"; }
    i=$((i+1)); [ $i -lt 120 ] || { tail -20 "$WORK/server.log" >&2; cannot_run "migrations did not finish in 120s"; }
    sleep 1
  done
  stop_server
  echo "# migrated: goose version $(q -c 'select max(version_id) from goose_db_version')"
  q <<SQL
INSERT INTO users (id, email, first_name, last_name, roles, status) VALUES ('00000000-0000-0000-0000-000000000001','spike@example.com','Spike','Runner',ARRAY['admin'],'active');
INSERT INTO campaigns_admin (id, slug, name, offer_text, face_value_cents, requires_online, ends_at, created_by)
  VALUES ('00000000-0000-0000-0000-0000000000c1','spike','Spike','\$1 off',100,false,now()+interval '30 days','00000000-0000-0000-0000-000000000001');
INSERT INTO qr_codes (campaign_id, short, channel, created_by)
  VALUES ('00000000-0000-0000-0000-0000000000c1','$SHORT','flyer','00000000-0000-0000-0000-000000000001');
SQL
  echo "# seeded user + campaign + qr_code short=$SHORT"
}

# fire N concurrent psql clients each running the SQL in $1; prints the qr_scans row count.
# Process spawn staggers the clients by tens of ms, which alone narrows the race, so each
# client first sleeps until one shared start line 1.5s out (server clock) — then all N
# statements land within a few ms of each other, which is what 12 pooled goroutines did.
blast() {
  local sql="$1" n="${2:-12}" start
  q -c "TRUNCATE qr_scans"
  start="$(q -c "select (clock_timestamp() + interval '1.5 seconds')::text")"
  { printf "select pg_sleep(greatest(0, extract(epoch from ('%s'::timestamptz - clock_timestamp()))));\n" "$start"
    printf '%s\n' "$sql"; } >"$WORK/stmt.sql"
  seq "$n" | xargs -P "$n" -I{} psql -X -q -v ON_ERROR_STOP=1 -d "$SPIKE_DB" -f "$WORK/stmt.sql" >/dev/null
  q -c "select count(*) from qr_scans"
}
