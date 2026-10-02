# lib.sh — shared bits for the dish-merge-and-erasure-backstop spikes. Sourced, not run.
#
# 🛑 The ONLY database server these spikes may touch is localhost:5434 (container
# yumyums-test-pg, role hqtest). :5433 is the dev AND production cluster; a spike
# pointed there destroyed the production database on 2026-08-06. The port is
# hard-coded and guarded below — do not parameterise it.
#
# Each spike creates its own database, hq_test_spike_i3_20261003, migrates it with the
# repo's own goose runner (by building backend/cmd/server and running it once until
# "database migrations applied successfully" appears — the migrations are embedded in
# the binary, so this is the same code path production takes), seeds the minimum rows,
# and DROPs the database on exit (trap).

export PGHOST=localhost PGPORT=5434 PGUSER=hqtest PGPASSWORD=hqtest
SPIKE_DB=hq_test_spike_i3_20261003
SPIKE_PORT=8313

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

[ "$PGPORT" = "5434" ] || cannot_run "refusing: PGPORT=$PGPORT is not the test cluster (5434)"
[ "$SPIKE_DB" = "hq_test_spike_i3_20261003" ] || cannot_run "refusing: scratch db name drifted"

export PATH="/usr/local/go/bin:$PATH"
LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$LIB_DIR/../../../.." && pwd)"
BACKEND="$REPO_ROOT/backend"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/spike-i3.XXXXXX")"
SERVER_PID=""

# Admin psql (maintenance db) and spike psql (scratch db). -X skips ~/.psqlrc.
padmin() { psql -X -q -v ON_ERROR_STOP=1 -d postgres "$@"; }
pq()     { psql -X -q -v ON_ERROR_STOP=1 -d "$SPIKE_DB" "$@"; }
pqa()    { pq -At "$@"; }   # unaligned, tuples only

spike_cleanup() {
  local rc=$?
  if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then kill "$SERVER_PID" 2>/dev/null || true; wait "$SERVER_PID" 2>/dev/null || true; fi
  padmin -c "DROP DATABASE IF EXISTS $SPIKE_DB WITH (FORCE);" >/dev/null 2>&1 \
    && echo "# cleanup: dropped $SPIKE_DB" || echo "# cleanup: DROP DATABASE $SPIKE_DB failed — drop it by hand on :5434" >&2
  rm -rf "$WORK"
  exit $rc
}

spike_db_create() {
  trap spike_cleanup EXIT
  padmin -c "select 1" >/dev/null 2>&1 || cannot_run "cannot reach yumyums-test-pg on :5434 (task test:db:up)"
  local who; who="$(padmin -Atc "select current_user || '@' || inet_server_port()")"
  [ "$who" = "hqtest@5432" ] || cannot_run "connected as $who, expected hqtest on the test container"
  padmin -c "DROP DATABASE IF EXISTS $SPIKE_DB WITH (FORCE);" >/dev/null 2>&1
  padmin -c "CREATE DATABASE $SPIKE_DB;" >/dev/null
  echo "# created $SPIKE_DB on :5434"
}

# Migrate by running the real server once against the scratch db. Every secret the
# Playwright stack blanks (playwright.config.js webServer command) is blanked here too,
# and E2E_DISABLE_SCHEDULERS=1 keeps the workers quiet for the second it is up.
spike_db_migrate() {
  ( cd "$BACKEND" && go build -o "$WORK/hqserver" ./cmd/server/ ) || cannot_run "go build ./cmd/server failed"
  ( cd "$BACKEND" && PORT=$SPIKE_PORT DB_URL="postgres://hqtest:hqtest@localhost:5434/$SPIKE_DB?sslmode=disable" \
      STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
      MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= \
      SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= STORAGE_KEY= STORAGE_SECRET= STORAGE_BUCKET= STORAGE_REGION= STORAGE_ENDPOINT= \
      "$WORK/hqserver" >"$WORK/server.log" 2>&1 ) &
  SERVER_PID=$!
  local i
  for i in $(seq 1 120); do
    grep -q "database migrations applied successfully" "$WORK/server.log" 2>/dev/null && break
    kill -0 "$SERVER_PID" 2>/dev/null || { tail -20 "$WORK/server.log" >&2; cannot_run "server exited before migrating"; }
    sleep 0.5
  done
  grep -q "database migrations applied successfully" "$WORK/server.log" || { tail -20 "$WORK/server.log" >&2; cannot_run "migrations did not finish in 60s"; }
  kill "$SERVER_PID" 2>/dev/null || true; wait "$SERVER_PID" 2>/dev/null || true; SERVER_PID=""
  echo "# migrated: goose version $(pqa -c "select max(version_id) from goose_db_version") ($(pqa -c "select count(*) from goose_db_version where is_applied") applied)"
}

# Fixed ids so every leg can name its rows without round-tripping.
U_ID=44444444-4444-4444-8444-444444444444
DISH_A=aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
DISH_B=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
CAMP_ID=cccccccc-cccc-4ccc-8ccc-cccccccccccc
SUB_ID=55555555-5555-4555-8555-555555555555
SHORT=SPKE23       # 0083's alphabet: 23456789ABCDEFGHJKLMNPQRSTUVWXYZ

spike_seed() {
  pq <<SQL
insert into users (id, email, first_name, last_name, roles, status) values ('$U_ID', 'spike-i3@example.test', 'Spike', 'I3', array['admin'], 'active');  -- 0017 split display_name, 0023 made role a TEXT[]
insert into menu_items (id, master_id, name, menu, menu_group, last_seen) values
  ('$DISH_A', 'spike-a', 'Dish A', 'Main', 'Wings', current_date),
  ('$DISH_B', 'spike-b', 'Dish B', 'Main', 'Wings', current_date);
insert into campaigns_admin (id, slug, name, offer_text, face_value_cents, requires_online, item_id, ends_at, created_by)
  values ('$CAMP_ID', 'spike-wings', 'Spike Wings', '\$2 off', 200, false, '$DISH_A', now() + interval '30 days', '$U_ID');
insert into qr_codes (short, campaign_id, channel, item_id, created_by) values ('$SHORT', '$CAMP_ID', 'truck_sign', '$DISH_A', '$U_ID');
SQL
  spike_seed_subscriber
  echo "# seeded: user, dishes A/B, campaign(item_id=A), code $SHORT(item_id=A), subscriber(source_short=$SHORT) + 1 event"
}

spike_seed_subscriber() {
  pq <<SQL
insert into subscribers (id, display_name, phone_e164, source, source_short, joined_at) values ('$SUB_ID', 'Sub One', '+15555550123', 'qr', '$SHORT', now());
insert into subscriber_events (subscriber_id, kind) values ('$SUB_ID', 'signed_up');
SQL
}

# try_sql <label> <sql> — runs <sql> in its own transaction (a DO block), prints
# "<label> → OK" or "<label> → SQLSTATE <code> constraint=<name> table=<t>", and sets
# TRY_STATE / TRY_CONSTRAINT for the caller to assert on. Never aborts the script.
try_sql() {
  local label="$1" sql="$2" out
  out="$(pq -At <<SQL 2>&1
do \$\$
declare s text; c text; t text;
begin
  $sql
  raise notice 'RESULT OK';
exception when others then
  get stacked diagnostics s = returned_sqlstate, c = constraint_name, t = table_name;
  raise notice 'RESULT % % %', s, coalesce(nullif(c,''),'-'), coalesce(nullif(t,''),'-');
end \$\$;
SQL
)" || true
  local line; line="$(printf '%s\n' "$out" | sed -n 's/^NOTICE:  RESULT //p' | head -1)"
  [ -n "$line" ] || { printf '%s\n' "$out" >&2; fail "$label: no RESULT line (psql error above)"; }
  if [ "$line" = "OK" ]; then
    TRY_STATE=OK; TRY_CONSTRAINT=""; echo "$label → OK"
  else
    TRY_STATE="${line%% *}"; local rest="${line#* }"; TRY_CONSTRAINT="${rest%% *}"; local tbl="${rest#* }"
    echo "$label → SQLSTATE $TRY_STATE constraint=$TRY_CONSTRAINT table=$tbl"
  fi
}

# fk_name <table> <column> — the actual FK constraint name from pg_constraint.
fk_name() {
  pqa -c "select c.conname from pg_constraint c join pg_attribute a on a.attrelid=c.conrelid and a.attnum = any(c.conkey)
          where c.contype='f' and c.conrelid='$1'::regclass and a.attname='$2'"
}
