# lib.sh — shared bits for the merge-target-and-blanking-guards spikes. Sourced, not run.
#
# 🛑 The ONLY database server these spikes may touch is localhost:5434 (container
# yumyums-test-pg, role hqtest). :5433 is the dev AND production cluster; a spike pointed
# there destroyed the production database on 2026-08-06. Hard-coded and guarded below.
#
# Spike 1 creates hq_test_spike_j2_20261005, migrates it with the repo's own goose runner (the
# built server, run once until "database migrations applied successfully"), seeds the minimum
# rows, and DROPs the database on exit (trap). Spike 2 uses a worktree + hq_test_spike_j2_go.

export PGHOST=localhost PGPORT=5434 PGUSER=hqtest PGPASSWORD=hqtest
SPIKE_DB=hq_test_spike_j2_20261005
GO_DB=hq_test_spike_j2_go
SPIKE_PORT=8322

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
leg()        { printf '\n── %s ──\n' "$1"; }

[ "$PGPORT" = "5434" ] || cannot_run "refusing: PGPORT=$PGPORT is not the test cluster (5434)"

export PATH="/usr/local/go/bin:$PATH"
LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$LIB_DIR/../../../.." && pwd)"
BACKEND="$REPO_ROOT/backend"
WT="$(dirname -- "$REPO_ROOT")/hq-worktrees/spike-j2-20261005"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/spike-j2.XXXXXX")"
SERVER_PID=""
T0=$(date +%s)

padmin() { psql -X -q -v ON_ERROR_STOP=1 -d postgres "$@"; }
pq()     { psql -X -q -v ON_ERROR_STOP=1 -d "$SPIKE_DB" "$@"; }
pqa()    { pq -At "$@"; }

stop_server() {
  [ -n "$SERVER_PID" ] || return 0
  kill -TERM "$SERVER_PID" 2>/dev/null || true
  local i=0; while kill -0 "$SERVER_PID" 2>/dev/null && [ $i -lt 50 ]; do sleep 0.1; i=$((i+1)); done
  kill -0 "$SERVER_PID" 2>/dev/null && { kill -KILL "$SERVER_PID" 2>/dev/null || true; sleep 0.2; }
  SERVER_PID=""
}

spike_cleanup() {
  local rc=$?
  set +e
  stop_server
  if [ -d "$WT" ]; then
    git -C "$REPO_ROOT" worktree remove --force "$WT" >/dev/null 2>&1 || rm -rf "$WT"
    git -C "$REPO_ROOT" worktree prune >/dev/null 2>&1
  fi
  padmin -c "DROP DATABASE IF EXISTS $SPIKE_DB WITH (FORCE);" >/dev/null 2>&1
  padmin -c "DROP DATABASE IF EXISTS $GO_DB WITH (FORCE);" >/dev/null 2>&1
  echo "# cleanup: $SPIKE_DB and $GO_DB dropped; worktree removed=$([ -d "$WT" ] && echo no || echo yes); wall-clock $(( $(date +%s) - T0 ))s"
  rm -rf "$WORK"
  exit $rc
}

spike_preflight() {
  trap spike_cleanup EXIT
  command -v go >/dev/null || cannot_run "go not on PATH"
  command -v psql >/dev/null || cannot_run "psql not on PATH"
  padmin -c "select 1" >/dev/null 2>&1 || cannot_run "cannot reach yumyums-test-pg on :5434 (task test:db:up)"
  local who; who="$(padmin -Atc "select current_user || '@' || inet_server_port()")"
  [ "$who" = "hqtest@5432" ] || cannot_run "connected as $who, expected hqtest on the test container"
}

spike_db_create() {
  padmin -c "DROP DATABASE IF EXISTS $SPIKE_DB WITH (FORCE);" >/dev/null 2>&1
  padmin -c "CREATE DATABASE $SPIKE_DB;" >/dev/null
  echo "# created $SPIKE_DB on :5434"
}

spike_db_migrate() {
  ( cd "$BACKEND" && go build -o "$WORK/hqserver" ./cmd/server/ ) || cannot_run "go build ./cmd/server failed"
  if ss -ltn 2>/dev/null | grep -q ":$SPIKE_PORT "; then cannot_run "port $SPIKE_PORT already in use"; fi
  ( cd "$BACKEND" && exec env PORT=$SPIKE_PORT DB_URL="postgres://hqtest:hqtest@localhost:5434/$SPIKE_DB?sslmode=disable" \
      STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
      MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= \
      SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= STORAGE_KEY= STORAGE_SECRET= STORAGE_BUCKET= STORAGE_REGION= STORAGE_ENDPOINT= \
      "$WORK/hqserver" ) >"$WORK/server.log" 2>&1 &
  SERVER_PID=$!
  local i
  for i in $(seq 1 120); do
    grep -q "database migrations applied successfully" "$WORK/server.log" 2>/dev/null && break
    kill -0 "$SERVER_PID" 2>/dev/null || { tail -20 "$WORK/server.log" >&2; cannot_run "server exited before migrating"; }
    sleep 0.5
  done
  grep -q "database migrations applied successfully" "$WORK/server.log" || { tail -20 "$WORK/server.log" >&2; cannot_run "migrations did not finish in 60s"; }
  stop_server
  echo "# migrated: goose version $(pqa -c "select max(version_id) from goose_db_version") ($(pqa -c "select count(*) from goose_db_version where is_applied") applied)"
}

# Fixed ids so every leg can name its rows.
U_ID=44444444-4444-4444-8444-444444444444
DISH_A=aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa   # unattached: no recipe, campaign or code names it
DISH_C=cccccccc-cccc-4ccc-8ccc-cccccccccccc   # a campaign names it
BOGUS=99999999-9999-4999-8999-999999999999    # no menu_items row
CAMP_ID=dddddddd-dddd-4ddd-8ddd-dddddddddddd
SHORT=SPKJ25

spike_seed() {
  pq <<SQL
insert into users (id, email, first_name, last_name, roles, status) values ('$U_ID', 'spike-j2@example.test', 'Spike', 'J2', array['admin'], 'active');
insert into menu_items (id, master_id, name, menu, menu_group, last_seen) values
  ('$DISH_A', 'spike-a', 'Dish A (unattached)', 'Main', 'Wings', current_date),
  ('$DISH_C', 'spike-c', 'Dish C (campaign)',   'Main', 'Wings', current_date);
insert into daily_menu_sales (menu_item_id, business_date, units_sold, gross_amount) values ('$DISH_A', current_date, 7, 84.00);
insert into campaigns_admin (id, slug, name, offer_text, face_value_cents, requires_online, item_id, ends_at, created_by)
  values ('$CAMP_ID', 'spike-wings', 'Spike Wings', '\$2 off', 200, false, '$DISH_C', now() + interval '30 days', '$U_ID');
insert into qr_codes (short, campaign_id, channel, created_by) values ('$SHORT', '$CAMP_ID', 'truck_sign', '$U_ID');
SQL
  echo "# seeded: user, dish A (unattached, 1 daily_menu_sales row), dish C (campaign item_id), code $SHORT; target $BOGUS names no dish"
}

# try_sql <label> <sql> — runs <sql> in one transaction (DO block); sets TRY_STATE / TRY_CONSTRAINT.
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

# MergeMenuItem's statement sequence, verbatim from backend/internal/recipes/repository.go
# (recipes → campaigns_admin → qr_codes re-point, then DELETE the source). $1=target $2=source.
merge_sql() {
  cat <<SQL
  update recipes set menu_item_id = '$1', updated_at = now() where menu_item_id = '$2';
  update campaigns_admin set item_id = '$1', updated_at = now() where item_id = '$2';
  update qr_codes set item_id = '$1' where item_id = '$2';
  delete from menu_items where id = '$2';
SQL
}
