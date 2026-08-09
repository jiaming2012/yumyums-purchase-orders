#!/usr/bin/env bash
# sync-dev-up.sh — bring the RxDB DATA PLANE up as a PERSISTENT dev service.
# Card `sync-live-in-dev-substrate` (Activity 5, run 20260810), legs 1+2 + FDW.
#
# ═══════════════════════════════════════════════════════════════════════════
# WHAT THIS IS — and how it differs from demo:sync / the spikes
#
# The spikes and `task demo:sync` stand the data plane up INSIDE one script run
# and tear it down at the bottom. This card's job is the opposite: make the data
# plane STAY UP in the operator's dev environment, between runs, so the
# `workflows.html` `/sync/*` read path is live whenever the dev server is.
#
# This script productionizes the PROVEN mechanics (Spike A substrate GREEN, the
# LISTEN/NOTIFY relay GREEN in spikes C/D/E) into a persistent dev SERVICE:
#
#   1. SUBSTRATE — Spike A's PostgREST + Realtime, brought up in RECONCILE mode
#      (env-up.sh; idempotent, never destroys). This is the never-destroy-on-
#      restart posture the slate's precondition names.
#   2. RELAY — backend/cmd/spikec-relay (wrapping sync.RunSpikeCRelay) started
#      as a PERSISTENT BACKGROUND process (nohup, pidfile), LISTENing on the dev
#      HQ Postgres for submission_responses writes and projecting them into the
#      substrate. The relay trigger (sql/spike-c-relay-trigger.sql) is applied to
#      the dev HQ so /saveResponse writes fire the NOTIFY.
#   3. FDW POINTING — sql/persistent-dev-fdw-pointing.sql applied to BOTH the dev
#      HQ (HALF A: hq_sync_fdw LOGIN) and the substrate (HALF B: hq_pg repointed
#      at the dev HQ), so the production per-user RLS resolves against real dev
#      data — the persistent answer to the spike's repoint+restore.
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 B-164 — THE DEV HQ COORDINATE, AND WHY IT IS REFUSED BY DEFAULT
#
# The relay and the FDW both need the operator's LIVE dev HQ Postgres. On this
# box that genuinely lives on the Windows box at :5433 (yumyums-dev-pg), reached
# over Tailscale/LAN — the SAME cluster that serves https://hq.yumyums.kitchen.
# A probe there destroyed the prod database on 2026-08-06 (B-141/B-143, dec 155).
#
# So this script REFUSES a bare :5433 target unless the operator sets the explicit
# override HQ_SYNC_DEV_ALLOW_5433=1. The operator CAN knowingly point the
# persistent dev service at their real dev cluster — that IS the deliverable — but
# never by default and never from an unattended run. Set:
#
#   HQ_SYNC_DEV_HQ_DSN   the dev HQ Postgres DSN the relay LISTENs on and the FDW
#                        resolves against (e.g. postgres://yumyums:yumyums@
#                        100.70.200.55:5433/yumyums?sslmode=disable over Tailscale).
#   HQ_SYNC_DEV_FDW_HOST the host the SUBSTRATE (Docker) reaches that same dev HQ
#                        Postgres at from inside the Docker network (a Tailscale
#                        IP, or host.docker.internal if the DSN host is local).
#
# The MECHANISM is proven end-to-end WITHOUT :5433 by sync-dev-proof.sh, which
# uses a fresh scratch HQ on an ephemeral port (the spike-f model).
#
# USAGE
#   HQ_SYNC_DEV_HQ_DSN=... HQ_SYNC_DEV_FDW_HOST=... sync-dev-up.sh          # up (reconcile)
#   HQ_SYNC_DEV_HQ_DSN=... HQ_SYNC_DEV_FDW_HOST=... sync-dev-up.sh --status # report only
#   sync-dev-up.sh --down                                                   # stop the relay (substrate stays up)
#   sync-dev-up.sh --env                                                    # print the 4 HQ_SYNC_* vars for the dev server
#
#   or via the repo Taskfile:  task sync:dev:up | sync:dev:status | sync:dev:down | sync:dev:env

set -euo pipefail

SPIKE_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SPIKE_DIR/../../.." && pwd)"

RELAY_SYNC_TABLE="${HQ_SYNC_DEV_SYNC_TABLE:-hq_sync_checklists}"   # the substrate table the relay projects into
RELAY_APP_SLUG="${HQ_SYNC_DEV_APP_SLUG:-operations}"              # spike B finding #1 — constant, labelled
FDW_ROLE_PASS="${HQ_SYNC_DEV_FDW_PASS:-b2-rowvis-suite-throwaway}" # MUST match the substrate's existing user mapping
RUNDIR="${HQ_SYNC_DEV_RUNDIR:-$SPIKE_DIR/.persistent-dev}"        # pidfile + relay log live here (gitignored)
PIDFILE="$RUNDIR/relay.pid"
RELAY_LOG="$RUNDIR/relay.log"

# Go is not on a non-interactive shell's PATH on this box; the relay build and
# the token minter both need it. Same list env-up.sh / the spikes use.
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin "$HOME/.local/go/bin"; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH

MODE="up"
case "${1:-}" in
  --status) MODE="status" ;;
  --down)   MODE="down" ;;
  --env)    MODE="env" ;;
  "")       MODE="up" ;;
  *) echo "usage: $(basename "$0") [--status|--down|--env]" >&2; exit 64 ;;
esac

STEP=0
step() { STEP=$((STEP + 1)); printf '\n── %d. %s ───────────────────────────────\n' "$STEP" "$1"; }
fail() { printf '\n🛑 sync-dev-up: %s\n' "$1" >&2; exit "${2:-1}"; }

ANCHOR=""
if command -v git >/dev/null 2>&1; then
  _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
  [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
fi
[ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
SUBDC=(docker compose -p spike-supabase --project-directory "$ANCHOR" -f "$REPO_ROOT/docker-compose.supabase.yml")

subpsql() {
  local cid; cid="$("${SUBDC[@]}" ps -q db 2>/dev/null)"
  [ -n "$cid" ] || return 1
  docker exec -i "$cid" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -tAc "$1"
}

# --------------------------------------------------------------------------
# --env — print the four HQ_SYNC_* vars the dev server must carry. This is the
# EXACT set backend/Taskfile.yml's dev targets export; printing them here gives
# the operator a copy-pasteable source that resolves the live substrate ports.
# --------------------------------------------------------------------------
print_env() {
  local rest_port rt_port
  rest_port="$("${SUBDC[@]}" port rest 3000 2>/dev/null | sed 's/.*://' || true)"
  rt_port="$("${SUBDC[@]}" port realtime 4000 2>/dev/null | sed 's/.*://' || true)"
  local jwt_secret
  jwt_secret="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
  case "$rest_port" in ''|*[!0-9]*) fail "substrate PostgREST port not resolvable — is the substrate up? (task sync:dev:up)" ;; esac
  case "$rt_port"   in ''|*[!0-9]*) fail "substrate Realtime port not resolvable — is the substrate up? (task sync:dev:up)" ;; esac
  cat <<EOF
# The four HQ_SYNC_* vars the /sync proxy needs (proxy.go + jwtbridge_handler.go).
# backend/Taskfile.yml's dev / dev:tailscale / dev:lan targets export these; this
# resolves the CURRENT substrate host ports. DEV ONLY — never docker-compose.prod.yml.
export HQ_SYNC_REST_URL="http://127.0.0.1:${rest_port}"
export HQ_SYNC_REALTIME_URL="http://127.0.0.1:${rt_port}"
export HQ_SYNC_JWT_SECRET="${jwt_secret}"
export HQ_SYNC_REALTIME_HOST="realtime-dev.localhost"
EOF
}

if [ "$MODE" = "env" ]; then
  print_env
  exit 0
fi

# --------------------------------------------------------------------------
# --down — stop the persistent relay. The SUBSTRATE stays up (never-destroy).
# --------------------------------------------------------------------------
if [ "$MODE" = "down" ]; then
  if [ -f "$PIDFILE" ]; then
    pid="$(cat "$PIDFILE" 2>/dev/null || true)"
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      echo "relay stopped (pid $pid). The substrate stays up (never-destroy; use task spike:down to destroy it deliberately)."
    else
      echo "relay pidfile present but process not running; cleaning up."
    fi
    rm -f "$PIDFILE"
  else
    echo "no relay pidfile — nothing to stop. The substrate (if up) is untouched."
  fi
  exit 0
fi

# --------------------------------------------------------------------------
# --status — report substrate + relay + FDW pointing, touch nothing.
# --------------------------------------------------------------------------
if [ "$MODE" = "status" ]; then
  step "substrate"
  "${SUBDC[@]}" ps 2>/dev/null || echo "  (substrate not up — task sync:dev:up)"
  step "relay"
  if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE" 2>/dev/null)" 2>/dev/null; then
    echo "  RUNNING (pid $(cat "$PIDFILE")), log: $RELAY_LOG"
  else
    echo "  not running"
  fi
  step "FDW server hq_pg (substrate side)"
  subpsql "select string_agg(o,'|' order by o) from pg_foreign_server, unnest(srvoptions) o where srvname='hq_pg'" 2>/dev/null \
    | sed 's/^/  /' || echo "  (substrate not reachable)"
  exit 0
fi

# ══════════════════════════════════════════════════════════════════════════
# MODE=up — bring the persistent data plane up (reconcile).
# ══════════════════════════════════════════════════════════════════════════
mkdir -p "$RUNDIR"

# --- the dev HQ coordinate, refused by default if it is :5433 --------------
DEV_DSN="${HQ_SYNC_DEV_HQ_DSN:-}"
FDW_HOST="${HQ_SYNC_DEV_FDW_HOST:-}"
[ -n "$DEV_DSN" ] || fail "HQ_SYNC_DEV_HQ_DSN is unset — the relay and the FDW need the operator's live dev HQ Postgres DSN. See this script's header (B-164)." 64
[ -n "$FDW_HOST" ] || fail "HQ_SYNC_DEV_FDW_HOST is unset — the substrate (Docker) needs the host it reaches the dev HQ Postgres at. See this script's header." 64

# Parse host/port/dbname out of the DSN for the FDW SQL + the :5433 guard.
_dsn_noscheme="${DEV_DSN#*://}"
_dsn_hostpart="${_dsn_noscheme#*@}"          # host:port/db?params
_dsn_hostport="${_dsn_hostpart%%/*}"
_dsn_dbpart="${_dsn_hostpart#*/}"
DEV_DBNAME="${_dsn_dbpart%%\?*}"
DEV_PORT="${_dsn_hostport##*:}"
case "$_dsn_hostport" in *:*) : ;; *) DEV_PORT="5432" ;; esac

if [ "$DEV_PORT" = "5433" ] && [ "${HQ_SYNC_DEV_ALLOW_5433:-0}" != "1" ]; then
  fail "REFUSING: HQ_SYNC_DEV_HQ_DSN targets :5433 (yumyums-dev-pg = dev AND prod cluster; a probe there destroyed prod 2026-08-06). The dev HQ genuinely lives there, so if you MEAN it, set HQ_SYNC_DEV_ALLOW_5433=1 knowingly. Never from an unattended run. (B-164)" 64
fi

step "substrate — Spike A's PostgREST + Realtime (RECONCILE, never destroy)"
"$SPIKE_DIR/env-up.sh" || fail "the substrate did not come up — env-up.sh returned non-zero (its output names the leg)."

step "FDW pointing — HALF A (dev HQ: hq_sync_fdw LOGIN) + HALF B (substrate: hq_pg -> dev HQ)"
echo "  applying persistent-dev-fdw-pointing.sql to the dev HQ ($DEV_PORT/$DEV_DBNAME)"
psql "$DEV_DSN" -v ON_ERROR_STOP=1 -q \
  -v fdw_password="$FDW_ROLE_PASS" -v hq_host="$FDW_HOST" -v hq_port="$DEV_PORT" -v hq_dbname="$DEV_DBNAME" \
  -f "$SPIKE_DIR/sql/persistent-dev-fdw-pointing.sql" \
  || fail "applying the FDW SQL to the dev HQ failed"
echo "  applying persistent-dev-fdw-pointing.sql to the substrate (repoint hq_pg)"
sub_cid="$("${SUBDC[@]}" ps -q db)"
docker exec -i "$sub_cid" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q \
  -v fdw_password="$FDW_ROLE_PASS" -v hq_host="$FDW_HOST" -v hq_port="$DEV_PORT" -v hq_dbname="$DEV_DBNAME" \
  -f - < "$SPIKE_DIR/sql/persistent-dev-fdw-pointing.sql" \
  || fail "applying the FDW SQL to the substrate failed"

step "relay trigger — apply spike_c_relay_notify to the dev HQ's submission_responses"
psql "$DEV_DSN" -v ON_ERROR_STOP=1 -q -f "$SPIKE_DIR/sql/spike-c-relay-trigger.sql" \
  || fail "applying the relay trigger to the dev HQ failed"

step "relay — start as a persistent background service"
if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE" 2>/dev/null)" 2>/dev/null; then
  echo "  relay already running (pid $(cat "$PIDFILE")) — leaving it up (reconcile)"
else
  echo "  building backend/cmd/spikec-relay"
  ( cd "$REPO_ROOT/backend" && go build -o "$RUNDIR/spikec-relay" ./cmd/spikec-relay ) \
    || fail "'go build ./cmd/spikec-relay' failed"
  rest_port="$("${SUBDC[@]}" port rest 3000 2>/dev/null | sed 's/.*://')"
  case "$rest_port" in ''|*[!0-9]*) fail "substrate PostgREST port not resolvable" ;; esac
  jwt_secret="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
  service_token="$(cd "$SPIKE_DIR" && go run ./mintjwt -secret "$jwt_secret" -sub "sync-dev-relay" -role service_role -ttl 720h)" \
    || fail "minting the service_role token failed"
  echo "  starting relay (LISTEN on the dev HQ, project into substrate $RELAY_SYNC_TABLE)"
  nohup env SPIKE_C_HQ_DSN="$DEV_DSN" \
    SPIKE_C_REST_BASE="http://127.0.0.1:${rest_port}" \
    SPIKE_C_SERVICE_TOKEN="$service_token" \
    SPIKE_C_SYNC_TABLE="$RELAY_SYNC_TABLE" \
    SPIKE_C_APP_SLUG="$RELAY_APP_SLUG" \
    "$RUNDIR/spikec-relay" > "$RELAY_LOG" 2>&1 &
  relay_pid=$!
  echo "$relay_pid" > "$PIDFILE"
  deadline=$(( $(date +%s) + 30 ))
  until grep -q SPIKE_C_RELAY_READY "$RELAY_LOG" 2>/dev/null; do
    kill -0 "$relay_pid" 2>/dev/null || { echo "--- relay.log ---"; cat "$RELAY_LOG"; rm -f "$PIDFILE"; fail "the relay exited before announcing readiness"; }
    [ "$(date +%s)" -lt "$deadline" ] || { echo "--- relay.log ---"; cat "$RELAY_LOG"; fail "the relay never announced readiness within 30s"; }
    sleep 1
  done
  echo "  relay READY (pid $relay_pid, log $RELAY_LOG)"
fi

printf '\n══════════════════════════════════════════════════════════\n'
printf '✅ persistent sync data plane UP\n'
printf '   substrate: reconciled (task sync:dev:status to inspect)\n'
printf '   relay:     running against the dev HQ, projecting into %s\n' "$RELAY_SYNC_TABLE"
printf '   FDW:       hq_pg -> %s:%s/%s (persistent)\n' "$FDW_HOST" "$DEV_PORT" "$DEV_DBNAME"
printf '   dev server: run `task backend:dev:tailscale` — it now carries the 4 HQ_SYNC_* vars.\n'
printf '   (or source `task sync:dev:env` if you run the server by hand)\n'
printf '══════════════════════════════════════════════════════════\n'
