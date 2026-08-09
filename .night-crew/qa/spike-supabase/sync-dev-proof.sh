#!/usr/bin/env bash
# sync-dev-proof.sh — PROVE Card 1's two done_when items WITHOUT touching :5433.
# Card `sync-live-in-dev-substrate` (Activity 5, run 20260810).
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#
#   exit 0   GREEN. Both done_when items proved:
#            (1) The /sync/rest proxy DOOR: 503 with the 4 HQ_SYNC_* vars UNSET
#                (red-first), 200 with them SET (green). Both captured.
#            (2) The RELAY carries a real write: a field written through the REAL
#                /saveResponse path fires the NOTIFY, the relay projects it into
#                the substrate, and the row ARRIVES in the substrate within the
#                Spike-A convergence bound. Polled and asserted.
#   exit 1   RED — ran and a done_when item is DISPROVEN (door stays 503 with vars
#            set, or the relay never lands the row). A RED IS A FINDING, not a park.
#   exit 2   COULD NOT RUN — infra/setup (Docker down, substrate won't reconcile,
#            scratch HQ never healthy, migrator failed, login failed, /saveResponse
#            not 204). NO verdict.
#   exit 3   verdict reached but the substrate could not be restored (carried rows
#            or FDW options left behind — WILL red the Go RLS suites).
#   exit 64  usage error.
#
# WHY THIS EXISTS SEPARATELY FROM sync-dev-up.sh
#
# sync-dev-up.sh is the DELIVERABLE — it stands the persistent data plane up in
# the operator's dev environment, which genuinely lives on :5433. This proof must
# NOT touch :5433 (B-164), so it uses the SPIKE-F MODEL: a FRESH scratch HQ
# Postgres booted by HQ's own binary (real migrations) on a Docker-ASSIGNED
# EPHEMERAL port, the substrate consumed in RECONCILE mode, and a
# snapshot→verify→restore of every substrate table + FDW option it touches. The
# MECHANISM this proof exercises is byte-identical to what sync-dev-up.sh wires;
# only the HQ coordinate differs (ephemeral scratch here, live dev there).
#
# USAGE
#   .night-crew/qa/spike-supabase/sync-dev-proof.sh
#   .night-crew/qa/spike-supabase/sync-dev-proof.sh --keep
#
#   🛑 GATE ON THIS SCRIPT'S EXIT CODE, never on `task` (B-163).

set -euo pipefail

SPIKE_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SPIKE_DIR/../../.." && pwd)"

HQ_COMPOSE="$SPIKE_DIR/docker-compose.hq-real.yml"     # reuse spike-c/f's scratch-HQ compose
HQ_PROJECT="sync-dev-proof-hq"                          # its OWN project — disjoint from spike-c/f
HQ_DB_USER="hq"
HQ_DB_NAME="hq_real"
HQ_DB_PASS="9a1e5c7b3d206f84e5b90c1a7d43f68e"          # throwaway; docker-compose.hq-real.yml banner

SYNC_TABLE="hq_sync_checklists"                         # the substrate table the relay projects into
FDW_SERVER="hq_pg"
FDW_ROLE="hq_sync_fdw"
FDW_ROLE_PASS="b2-rowvis-suite-throwaway"              # matches the substrate's existing user mapping
REALTIME_HOST="realtime-dev.localhost"
APP_SLUG="operations"
SUPERADMIN_EMAIL="jamal@yumyums.kitchen"
SUPERADMIN_PASSWORD="test123"

HQ_API_PORT="${SYNC_DEV_PROOF_API_PORT:-8473}"         # NOT 8471/8472 (spike C/F), NOT 8089/8199 (dev/PW)
CONVERGE_MS="${SYNC_DEV_PROOF_CONVERGE_MS:-15000}"

KEEP=0
for arg in "$@"; do
  case "$arg" in
    --keep) KEEP=1 ;;
    *) echo "usage: $(basename "$0") [--keep]" >&2; exit 64 ;;
  esac
done

for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin "$HOME/.local/go/bin"; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH

STEP=0
step() { STEP=$((STEP + 1)); printf '\n══ %d. %s ═══════════════════════════════\n' "$STEP" "$1"; }
cannot_run() { printf '\n🛑 COULD NOT RUN (not a verdict) — %s\n' "$1" >&2; exit 2; }
red()        { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }

ANCHOR=""
if command -v git >/dev/null 2>&1; then
  _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
  [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
fi
[ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
HQDC=(docker compose -p "$HQ_PROJECT" --project-directory "$ANCHOR" -f "$HQ_COMPOSE")
SUBDC=(docker compose -p spike-supabase --project-directory "$ANCHOR" -f "$REPO_ROOT/docker-compose.supabase.yml")

RUN_ID="${SYNC_DEV_PROOF_RUN_ID:-p$(date -u +%Y%m%d%H%M%S)}"
WORK="$(mktemp -d -t sync-dev-proof-XXXXXX)"
SENTINEL="syncdev-$RUN_ID-$(date +%s%N)"

printf '# sync-dev-proof.sh — Card 1 done_when proof (spike-f model, NO :5433)\n'
printf '# repo    %s\n# run     %s\n# started %s\n' "$REPO_ROOT" "$RUN_ID" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --------------------------------------------------------------------------
# Teardown — registered BEFORE anything is created. Restores the substrate.
# --------------------------------------------------------------------------
SERVER_PID=""; RELAY_PID=""; SUB_DB_CID=""
FDW_REPOINTED=0; ORIG_FDW_HOST=""; ORIG_FDW_PORT=""; ORIG_FDW_DBNAME=""
BASELINE_SYNC=""

subpsql() { docker exec -i "$SUB_DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -tAc "$1"; }

teardown() {
  local rc=$?
  set +e
  [ -n "$RELAY_PID" ]  && { kill "$RELAY_PID"  2>/dev/null; wait "$RELAY_PID"  2>/dev/null; }
  [ -n "$SERVER_PID" ] && { kill "$SERVER_PID" 2>/dev/null; wait "$SERVER_PID" 2>/dev/null; }

  if [ -n "$SUB_DB_CID" ]; then
    printf '\n── teardown (1/3): removing carried syncdev- rows from %s ──\n' "$SYNC_TABLE"
    subpsql "delete from public.$SYNC_TABLE where id like 'spikec-%' or id like 'syncdev-%'" >/dev/null 2>&1
    if [ -n "$BASELINE_SYNC" ]; then
      local after; after="$(subpsql "select coalesce(string_agg(id, ',' order by id), '') from public.$SYNC_TABLE" 2>/dev/null)"
      if [ "$after" = "$BASELINE_SYNC" ]; then
        printf '  VERIFIED: %s id-set is byte-identical to the pre-run baseline.\n' "$SYNC_TABLE"
      else
        printf '  🛑 %s DID NOT RETURN TO BASELINE.\n     baseline: %s\n     after   : %s\n' "$SYNC_TABLE" "$BASELINE_SYNC" "$after"; rc=3
      fi
    fi
    if [ "$FDW_REPOINTED" = "1" ] && [ -n "$ORIG_FDW_HOST" ]; then
      printf '\n── teardown (2/3): restoring FDW server %s -> %s:%s/%s ──\n' "$FDW_SERVER" "$ORIG_FDW_HOST" "$ORIG_FDW_PORT" "$ORIG_FDW_DBNAME"
      subpsql "alter server $FDW_SERVER options (set host '$ORIG_FDW_HOST', set port '$ORIG_FDW_PORT', set dbname '$ORIG_FDW_DBNAME')" >/dev/null 2>&1 \
        || { printf '  🛑 could NOT restore the FDW server options — repair by hand.\n'; rc=3; }
      subpsql "select postgres_fdw_disconnect_all()" >/dev/null 2>&1
      printf '  FDW options now: %s\n' "$(subpsql "select string_agg(o,'|' order by o) from pg_foreign_server, unnest(srvoptions) o where srvname='$FDW_SERVER'" 2>/dev/null)"
    fi
  fi

  if [ "$KEEP" = "1" ]; then
    printf '\n(--keep) scratch HQ Postgres LEFT RUNNING (project %s)\n' "$HQ_PROJECT"
  else
    printf '\n── teardown (3/3): destroying the scratch HQ Postgres (project %s) ──\n' "$HQ_PROJECT"
    "${HQDC[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || printf '  ⚠ teardown of %s did not complete cleanly\n' "$HQ_PROJECT"
    printf '  torn down.\n'
  fi
  rm -rf "$WORK"
  exit $rc
}
trap teardown EXIT

# --------------------------------------------------------------------------
step "preflight"
# --------------------------------------------------------------------------
for bin in docker curl node go npx python3; do
  command -v "$bin" >/dev/null 2>&1 || cannot_run "required tool not on PATH: $bin"
done
docker info >/dev/null 2>&1 || cannot_run "the Docker daemon is not reachable"
[ -f "$HQ_COMPOSE" ] || cannot_run "compose file missing: $HQ_COMPOSE"
[ -f "$REPO_ROOT/workflows.html" ] || cannot_run "workflows.html missing at repo root"
if grep -Eq '^[[:space:]]*-[[:space:]]*"?(5432|5433|5434):' "$HQ_COMPOSE"; then
  cannot_run "$HQ_COMPOSE publishes a FIXED host port in 5432-5434; this proof needs a Docker-assigned ephemeral port"
fi

# --------------------------------------------------------------------------
step "substrate — reconcile (never destroy)"
# --------------------------------------------------------------------------
"$SPIKE_DIR/env-up.sh" || cannot_run "the substrate did not come up — env-up.sh returned non-zero"
SUB_DB_CID="$("${SUBDC[@]}" ps -q db 2>/dev/null)"
[ -n "$SUB_DB_CID" ] || cannot_run "could not resolve the substrate db container"
REST_PORT="$("${SUBDC[@]}" port rest 3000 2>/dev/null | sed 's/.*://')"
RT_PORT="$("${SUBDC[@]}" port realtime 4000 2>/dev/null | sed 's/.*://')"
case "$REST_PORT" in ''|*[!0-9]*) cannot_run "substrate PostgREST port unresolved" ;; esac
case "$RT_PORT" in ''|*[!0-9]*) cannot_run "substrate Realtime port unresolved" ;; esac
REST_BASE="http://127.0.0.1:$REST_PORT"; RT_BASE="http://127.0.0.1:$RT_PORT"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET from docker-compose.supabase.yml"
printf '  PostgREST %s · Realtime %s\n' "$REST_BASE" "$RT_BASE"

# --------------------------------------------------------------------------
step "substrate baseline + FDW snapshot (so restore can be VERIFIED)"
# --------------------------------------------------------------------------
BASELINE_SYNC="$(subpsql "select coalesce(string_agg(id, ',' order by id), '') from public.$SYNC_TABLE")" \
  || cannot_run "could not read the $SYNC_TABLE baseline"
case "$BASELINE_SYNC" in *spikec-*|*syncdev-*) cannot_run "the substrate holds leftover spikec-/syncdev- rows from an aborted run — remove them first" ;; esac
FDW_OPTS="$(subpsql "select string_agg(o,'|' order by o) from pg_foreign_server, unnest(srvoptions) o where srvname='$FDW_SERVER'")" \
  || cannot_run "could not read the $FDW_SERVER FDW options"
ORIG_FDW_HOST="$(printf '%s' "$FDW_OPTS" | tr '|' '\n' | sed -n 's/^host=//p')"
ORIG_FDW_PORT="$(printf '%s' "$FDW_OPTS" | tr '|' '\n' | sed -n 's/^port=//p')"
ORIG_FDW_DBNAME="$(printf '%s' "$FDW_OPTS" | tr '|' '\n' | sed -n 's/^dbname=//p')"
[ -n "$ORIG_FDW_HOST" ] && [ -n "$ORIG_FDW_PORT" ] && [ -n "$ORIG_FDW_DBNAME" ] \
  || cannot_run "could not parse the FDW options: $FDW_OPTS"
printf '  %s baseline: %s row(s); FDW %s -> %s:%s/%s (snapshotted)\n' \
  "$SYNC_TABLE" "$(subpsql "select count(*) from public.$SYNC_TABLE")" "$FDW_SERVER" "$ORIG_FDW_HOST" "$ORIG_FDW_PORT" "$ORIG_FDW_DBNAME"

# --------------------------------------------------------------------------
step "scratch HQ Postgres — fresh, EMPTY, Docker-assigned ephemeral port"
# --------------------------------------------------------------------------
"${HQDC[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
"${HQDC[@]}" up -d || cannot_run "'docker compose up -d' failed for project $HQ_PROJECT"
HQ_CID="$("${HQDC[@]}" ps -q hqreal || true)"
[ -n "$HQ_CID" ] || cannot_run "the hqreal service has no container"
deadline=$(( $(date +%s) + 120 ))
while :; do
  health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$HQ_CID" 2>/dev/null || echo none)"
  [ "$health" = "healthy" ] && break
  state="$(docker inspect -f '{{.State.Status}}' "$HQ_CID" 2>/dev/null || echo missing)"
  case "$state" in exited|dead|missing) cannot_run "hqreal is '$state' — docker logs $HQ_CID" ;; esac
  [ "$(date +%s)" -lt "$deadline" ] || cannot_run "hqreal never became healthy within 120s"
  sleep 2
done
HQ_PORT="$("${HQDC[@]}" port hqreal 5432 2>/dev/null | sed 's/.*://' || true)"
case "$HQ_PORT" in ''|*[!0-9]*) cannot_run "could not resolve hqreal:5432 host port" ;; esac
case "$HQ_PORT" in 5432|5433|5434) cannot_run "Docker assigned protected port $HQ_PORT — re-run" ;; esac
HQ_DSN="postgres://$HQ_DB_USER:$HQ_DB_PASS@127.0.0.1:$HQ_PORT/$HQ_DB_NAME?sslmode=disable"
printf '  scratch HQ on host port %s (ephemeral, not 5432/5433/5434)\n' "$HQ_PORT"
srcpsql() { docker exec -i "$HQ_CID" psql -U "$HQ_DB_USER" -d "$HQ_DB_NAME" -v ON_ERROR_STOP=1 -tAc "$1"; }

# --------------------------------------------------------------------------
step "RED-FIRST — the /sync/rest door with the 4 HQ_SYNC_* vars UNSET must 503"
# --------------------------------------------------------------------------
# Boot HQ's server WITHOUT any HQ_SYNC_* var. The proxy is registered (main.go)
# but LoadProxyConfig() returns empty URLs → the room 503s. This IS the red-first:
# the door is closed until the wiring the card delivers opens it.
( cd "$REPO_ROOT/backend" && go build -o "$WORK/hq-server" ./cmd/server ) \
  || cannot_run "'go build ./cmd/server' failed"
API="http://127.0.0.1:$HQ_API_PORT"
if curl -fsS --max-time 2 "$API/api/v1/health" >/dev/null 2>&1; then
  cannot_run "something is ALREADY serving $API — set SYNC_DEV_PROOF_API_PORT"
fi
boot_server() {  # $@ = extra env assignments (the HQ_SYNC_* vars, or none)
  env PORT="$HQ_API_PORT" DB_URL="$HQ_DSN" STATIC_DIR="$REPO_ROOT" \
    SUPERADMIN_CONFIG="$REPO_ROOT/backend/config/superadmins.yaml" \
    TEMPLATE_CONFIG="$REPO_ROOT/backend/config/templates.yaml" \
    TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
    MERCURY_API_KEY= ANTHROPIC_API_KEY= \
    ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= \
    SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= \
    "$@" "$WORK/hq-server" > "$WORK/hq-server.log" 2>&1 &
  SERVER_PID=$!
  local dl=$(( $(date +%s) + 90 ))
  until curl -fsS "$API/api/v1/health" >/dev/null 2>&1; do
    kill -0 "$SERVER_PID" 2>/dev/null || { echo "--- hq-server.log ---"; cat "$WORK/hq-server.log"; cannot_run "HQ server exited before health"; }
    [ "$(date +%s)" -lt "$dl" ] || { echo "--- hq-server.log ---"; cat "$WORK/hq-server.log"; cannot_run "HQ server never answered health within 90s"; }
    sleep 1
  done
}
# 🛑 `wait` returns the WAITED-FOR process's exit status, and a process killed by
# SIGTERM exits 143 — under `set -e` that aborts the whole script (measured: the
# proof stopped dead right after the red-first capture with rc=143). The teardown
# trap survives it because it runs under `set +e`; stop_server runs under `set -e`,
# so it must swallow the signal-exit itself. `|| true`, not removing the wait — we
# still want to reap the child before booting the second server on the same port.
stop_server() { [ -n "$SERVER_PID" ] && { kill "$SERVER_PID" 2>/dev/null || true; wait "$SERVER_PID" 2>/dev/null || true; SERVER_PID=""; }; }

boot_server   # NO HQ_SYNC_* vars
TABLES="$(srcpsql "select count(*) from information_schema.tables where table_schema='public'")"
[ "$TABLES" -gt 30 ] || cannot_run "only $TABLES public tables — HQ's migrator did not run"
# A session, so the proxy reaches the room (an unauthenticated call 401s before 503).
LOGIN_CODE="$(curl -sS -o "$WORK/login.json" -w '%{http_code}' -c "$WORK/cookies.txt" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$SUPERADMIN_EMAIL\",\"password\":\"$SUPERADMIN_PASSWORD\"}" "$API/api/v1/auth/login")"
[ "$LOGIN_CODE" = "200" ] || { cat "$WORK/login.json"; cannot_run "login returned HTTP $LOGIN_CODE"; }
RED_CODE="$(curl -sS -o "$WORK/red.out" -w '%{http_code}' -b "$WORK/cookies.txt" "$API/sync/rest/")"
RED_BODY="$(cat "$WORK/red.out")"
printf '  RED-FIRST: GET /sync/rest/ (vars UNSET) -> HTTP %s  body=%s\n' "$RED_CODE" "$RED_BODY"
[ "$RED_CODE" = "503" ] || red "the door did NOT 503 with the vars unset (got $RED_CODE) — the red-first is vacuous, the door was already open"
stop_server

# --------------------------------------------------------------------------
step "wire the FDW at THIS scratch HQ (persistent-dev-fdw-pointing.sql, both halves)"
# --------------------------------------------------------------------------
srcpsql "select 1 from pg_views where viewname='hq_sync_field_templates'" | grep -q 1 \
  || cannot_run "hq_sync_field_templates absent in the scratch HQ — migration 0073 did not run"
# HALF A on the scratch HQ (give hq_sync_fdw LOGIN) — via the SHIPPED sql file.
docker exec -i "$HQ_CID" psql -U "$HQ_DB_USER" -d "$HQ_DB_NAME" -v ON_ERROR_STOP=1 -q \
  -v fdw_password="$FDW_ROLE_PASS" -v hq_host="host.docker.internal" -v hq_port="$HQ_PORT" -v hq_dbname="$HQ_DB_NAME" \
  -f - < "$SPIKE_DIR/sql/persistent-dev-fdw-pointing.sql" \
  || cannot_run "applying the FDW SQL (HALF A) to the scratch HQ failed"
# HALF B on the substrate (repoint hq_pg at the scratch HQ) — same shipped file.
subpsql "select 1" >/dev/null
docker exec -i "$SUB_DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q \
  -v fdw_password="$FDW_ROLE_PASS" -v hq_host="host.docker.internal" -v hq_port="$HQ_PORT" -v hq_dbname="$HQ_DB_NAME" \
  -f - < "$SPIKE_DIR/sql/persistent-dev-fdw-pointing.sql" \
  || cannot_run "applying the FDW SQL (HALF B) to the substrate failed"
FDW_REPOINTED=1
FT_COUNT="$(subpsql "select count(*) from public.hq_field_templates" 2>"$WORK/fdw.err")" \
  || { echo "--- fdw.err ---"; cat "$WORK/fdw.err"; cannot_run "the FDW could not connect to the scratch HQ after the repoint"; }
printf '  FDW resolves through the shipped SQL: hq_field_templates has %s row(s)\n' "$FT_COUNT"

# --------------------------------------------------------------------------
step "GREEN — the /sync/rest door WITH the 4 HQ_SYNC_* vars set must 200"
# --------------------------------------------------------------------------
boot_server \
  HQ_SYNC_REST_URL="$REST_BASE" \
  HQ_SYNC_REALTIME_URL="$RT_BASE" \
  HQ_SYNC_JWT_SECRET="$JWT_SECRET" \
  HQ_SYNC_REALTIME_HOST="$REALTIME_HOST"
LOGIN_CODE="$(curl -sS -o "$WORK/login.json" -w '%{http_code}' -c "$WORK/cookies.txt" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$SUPERADMIN_EMAIL\",\"password\":\"$SUPERADMIN_PASSWORD\"}" "$API/api/v1/auth/login")"
[ "$LOGIN_CODE" = "200" ] || { cat "$WORK/login.json"; cannot_run "login returned HTTP $LOGIN_CODE"; }
SESSION="$(awk '$6=="hq_session"{print $7}' "$WORK/cookies.txt" | tail -1)"
GREEN_CODE="$(curl -sS -o "$WORK/green.out" -w '%{http_code}' -b "$WORK/cookies.txt" "$API/sync/rest/")"
GREEN_BODY="$(head -c 200 "$WORK/green.out")"
printf '  GREEN: GET /sync/rest/ (vars SET) -> HTTP %s  body(head)=%s\n' "$GREEN_CODE" "$GREEN_BODY"
case "$GREEN_CODE" in
  503) red "DONE_WHEN 1 DISPROVEN — the door still 503s WITH the 4 vars set (leg-2 wiring not effective)" ;;
  200) printf '  ✅ done_when 1: the door is OPEN (200, not 503) with the 4 HQ_SYNC_* vars set\n' ;;
  *)   red "the door returned HTTP $GREEN_CODE (expected 200) with the vars set" ;;
esac

# --------------------------------------------------------------------------
step "the relay's prerequisites — a real field + the relay trigger on the scratch HQ"
# --------------------------------------------------------------------------
HQ_USER_ID="$(srcpsql "select id from users where email='$SUPERADMIN_EMAIL'")"
[ -n "$HQ_USER_ID" ] || cannot_run "superadmin not upserted into users"
FIELD_ID="$(srcpsql "select id from checklist_fields where type='text' order by id limit 1")"
[ -n "$FIELD_ID" ] || FIELD_ID="$(srcpsql "select id from checklist_fields order by id limit 1")"
[ -n "$FIELD_ID" ] || cannot_run "no checklist_fields — config/templates.yaml did not seed"
# Apply the relay trigger (the SAME sql sync-dev-up.sh applies to the dev HQ).
docker exec -i "$HQ_CID" psql -U "$HQ_DB_USER" -d "$HQ_DB_NAME" -v ON_ERROR_STOP=1 -q \
  -f - < "$SPIKE_DIR/sql/spike-c-relay-trigger.sql" \
  || cannot_run "applying the relay trigger to the scratch HQ failed"
srcpsql "select 1 from pg_trigger where tgname='spike_c_relay_notify'" | grep -q 1 \
  || cannot_run "the relay trigger did not register"
printf '  field=%s · trigger spike_c_relay_notify present\n' "$FIELD_ID"

# --------------------------------------------------------------------------
step "start the relay as a service (the SAME mechanism sync:dev:relay wires)"
# --------------------------------------------------------------------------
( cd "$REPO_ROOT/backend" && go build -o "$WORK/spikec-relay" ./cmd/spikec-relay ) \
  || cannot_run "'go build ./cmd/spikec-relay' failed"
SERVICE_TOKEN="$(cd "$SPIKE_DIR" && go run ./mintjwt -secret "$JWT_SECRET" -sub "sync-dev-relay" -role service_role -ttl 60m)" \
  || cannot_run "minting the service_role token failed"
env SPIKE_C_HQ_DSN="$HQ_DSN" SPIKE_C_REST_BASE="$REST_BASE" \
  SPIKE_C_SERVICE_TOKEN="$SERVICE_TOKEN" SPIKE_C_SYNC_TABLE="$SYNC_TABLE" SPIKE_C_APP_SLUG="$APP_SLUG" \
  "$WORK/spikec-relay" > "$WORK/relay.log" 2>&1 &
RELAY_PID=$!
deadline=$(( $(date +%s) + 30 ))
until grep -q SPIKE_C_RELAY_READY "$WORK/relay.log" 2>/dev/null; do
  kill -0 "$RELAY_PID" 2>/dev/null || { echo "--- relay.log ---"; cat "$WORK/relay.log"; cannot_run "the relay exited before readiness"; }
  [ "$(date +%s)" -lt "$deadline" ] || { echo "--- relay.log ---"; cat "$WORK/relay.log"; cannot_run "the relay never announced readiness within 30s"; }
  sleep 1
done
printf '  relay READY (pid %s)\n' "$RELAY_PID"

# --------------------------------------------------------------------------
step "DONE_WHEN 2 — write ONE field through the REAL /saveResponse; poll the substrate"
# --------------------------------------------------------------------------
SAVE_CODE="$(curl -sS -o "$WORK/save.out" -w '%{http_code}' -b "$WORK/cookies.txt" \
  -H 'Content-Type: application/json' \
  -d "{\"field_id\":\"$FIELD_ID\",\"value\":\"$SENTINEL\"}" "$API/api/v1/workflow/saveResponse")"
case "$SAVE_CODE" in 200|204) printf '  POST /api/v1/workflow/saveResponse -> %s\n' "$SAVE_CODE" ;; *) cat "$WORK/save.out"; cannot_run "POST /saveResponse returned HTTP $SAVE_CODE" ;; esac
RESP_ID="$(srcpsql "select id from submission_responses where field_id='$FIELD_ID' and answered_by='$HQ_USER_ID' and submission_id is null order by answered_at desc limit 1")"
[ -n "$RESP_ID" ] || cannot_run "the write did not land as a draft submission_responses row in the scratch HQ"
printf '  wrote submission_responses %s (sentinel=%s); polling the substrate for spikec-%s ...\n' "$RESP_ID" "$SENTINEL" "$RESP_ID"

# Poll the substrate — the relay projects the row keyed by id `spikec-<respid>`.
# 🛑 macOS `date` has no %N (nanoseconds) — `date +%s%N` yields e.g. "1786287909N",
# which broke the elapsed-ms arithmetic. python3 (asserted in preflight) gives a
# portable millisecond epoch on both macOS and Linux.
now_ms() { python3 -c 'import time;print(int(time.time()*1000))'; }
started="$(now_ms)"
deadline=$(( $(date +%s) + (CONVERGE_MS / 1000) + 2 ))
FOUND=""
while :; do
  FOUND="$(subpsql "select body from public.$SYNC_TABLE where id='spikec-$RESP_ID'" 2>/dev/null || true)"
  [ -n "$FOUND" ] && break
  if ! kill -0 "$RELAY_PID" 2>/dev/null; then echo "--- relay.log ---"; cat "$WORK/relay.log"; red "the relay died before the row arrived"; fi
  [ "$(date +%s)" -lt "$deadline" ] || { echo "--- relay.log ---"; cat "$WORK/relay.log"; red "DONE_WHEN 2 DISPROVEN — the written field never arrived in the substrate within ${CONVERGE_MS}ms"; }
  sleep 0.25
done
elapsed_ms=$(( $(now_ms) - started ))
case "$FOUND" in
  *"$SENTINEL"*) printf '  ✅ done_when 2: the field ARRIVED in the substrate in %sms — spikec-%s carries the sentinel\n' "$elapsed_ms" "$RESP_ID" ;;
  *) red "a row arrived but does not carry the sentinel (got: $(printf '%s' "$FOUND" | head -c 200))" ;;
esac

printf '\n══════════════════════════════════════════════════════════\n'
printf '✅ VERDICT: GREEN — both done_when items proved (spike-f model, NO :5433)\n'
printf '   (1) door: 503 vars-unset (red-first) -> 200 vars-set (green)\n'
printf '   (2) relay: real /saveResponse field arrived in the substrate in %sms\n' "$elapsed_ms"
printf '   FDW pointing + door + relay use the SHIPPED files sync:dev:up wires.\n'
printf '   finished %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf '══════════════════════════════════════════════════════════\n'
exit 0
