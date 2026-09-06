#!/usr/bin/env bash
# lib-hq.sh — shared HQ-server bring-up for the sync-coordinates-provisioning
# spikes. Sourced AFTER supabase/verify/lib.sh (needs $REPO_ROOT, $DC, fail,
# cannot_run and the PATH repair).
#
# What this adds over the sibling spikes: the real Go backend — the shipped
# main.go wiring (auth.Middleware, TokenHandler, ProxyHandler) — because this
# goal's premise IS the HQ door, not bare PostgREST.
#
# 🛑 TARGET DISCIPLINE (HQ side). The HQ database is a SPIKE-OWNED name,
# hq_test_spike_prov, on :5434 (`hqtest` role — yumyums-test-pg, the TEST-ONLY
# container from docker-compose.test.yml). scripts/reset-e2e-db.js's name guard
# is on the path, so a mistyped name refuses rather than dropping something
# real. NEVER :5433 (dev AND production — B-141/B-143, ledger decision 155).
# The spike-owned name means a concurrent `task test` run on hq_test_e2e
# cannot collide with us in either direction.

SPIKE_DB_NAME="hq_test_spike_prov"
HQ_PORT="${HQ_PORT:-8339}"
HQ_ORIGIN="http://127.0.0.1:${HQ_PORT}"
HQ_LOG="${HQ_LOG:-$SCRIPT_DIR/.hq-server.log}"
HQ_PID=""
HQ2_PID=""

ADMIN_EMAIL="jamal@yumyums.kitchen"   # bootstrapped from backend/config/superadmins.yaml
ADMIN_PASSWORD="test123"              # its committed dev_password — test stacks only

# Resolve the substrate's PostgREST host port + the committed throwaway secret.
resolve_substrate_rest() {
  REST_PORT_HOST="$("${DC[@]}" port rest 3000 | awk -F: '{print $NF}')"
  [ -n "$REST_PORT_HOST" ] || cannot_run "could not resolve the substrate's PostgREST host port"
  REST_DIRECT="http://127.0.0.1:${REST_PORT_HOST}"
  JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
  [ -n "$JWT_SECRET" ] || cannot_run "could not read the committed throwaway JWT_SECRET from docker-compose.supabase.yml"
}

# Bring the TEST pg container up (idempotent) and reset the spike-owned DB.
# The Taskfile brings it up under compose project `yumyums-test` — a bare
# `docker compose up` here would fight over the container name (measured:
# correction 1 in the ledger), so reuse a running container and otherwise
# invoke it the Taskfile's way.
hq_db_reset() {
  if ! docker ps --format '{{.Names}}' | grep -qx 'yumyums-test-pg'; then
    ( cd "$REPO_ROOT" && docker compose -p yumyums-test -f docker-compose.test.yml up -d >/dev/null 2>&1 ) \
      || cannot_run "docker compose -p yumyums-test up failed — yumyums-test-pg is a precondition"
  fi
  local tries=0
  until docker exec yumyums-test-pg pg_isready -U hqtest >/dev/null 2>&1; do
    tries=$((tries + 1)); [ "$tries" -le 30 ] || cannot_run "yumyums-test-pg never became ready"
    sleep 1
  done
  TEST_DB_NAME="$SPIKE_DB_NAME" node "$REPO_ROOT/scripts/reset-e2e-db.js" \
    || cannot_run "reset-e2e-db.js refused or failed for $SPIKE_DB_NAME"
}

# psql into the spike-owned HQ database (docker exec — no host psql needed).
hq_psql() { docker exec -i yumyums-test-pg psql -U hqtest -d "$SPIKE_DB_NAME" -v ON_ERROR_STOP=1 -qtA; }

hq_down() {
  for pid in "$HQ_PID" "$HQ2_PID"; do
    [ -n "$pid" ] || continue
    kill -TERM -- "-$pid" 2>/dev/null || true
    pkill -TERM -P "$pid" 2>/dev/null || true
  done
}

_wait_health() {
  local origin="$1" tries=0
  until curl -sf "$origin/api/v1/health" >/dev/null 2>&1; do
    tries=$((tries + 1)); [ "$tries" -le 90 ] || return 1
    sleep 1
  done
}

# Start the real server with the sync door CONFIGURED (secret + REST upstream).
hq_up() {
  local db_url="postgres://hqtest:hqtest@localhost:5434/${SPIKE_DB_NAME}?sslmode=disable&TimeZone=America/New_York"
  ( cd "$REPO_ROOT/backend" && exec setsid env \
      PORT="$HQ_PORT" DB_URL="$db_url" STATIC_DIR=../ \
      SUPERADMIN_CONFIG=config/superadmins.yaml \
      TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
      MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= \
      ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= SMTP_ADDR= \
      SMTP_USERNAME= SMTP_PASSWORD= STORAGE_KEY= STORAGE_SECRET= \
      STORAGE_BUCKET= STORAGE_REGION= STORAGE_ENDPOINT= \
      HQ_SYNC_JWT_SECRET="$JWT_SECRET" \
      HQ_SYNC_REST_URL="$REST_DIRECT" \
      go run ./cmd/server/ ) > "$HQ_LOG" 2>&1 &
  HQ_PID=$!
  _wait_health "$HQ_ORIGIN" || { tail -20 "$HQ_LOG" >&2; cannot_run "the HQ server never answered /api/v1/health on :$HQ_PORT (log tail above)"; }
  echo "#   HQ server        : $HQ_ORIGIN (pid $HQ_PID, spike DB $SPIKE_DB_NAME, door → $REST_DIRECT)"
}

# Start a second, DEGRADED instance: sync door deliberately UNCONFIGURED
# (no HQ_SYNC_JWT_SECRET, no HQ_SYNC_REST_URL) — the fail-closed legs.
hq_up_degraded() {
  HQ2_PORT="${HQ2_PORT:-8340}"
  HQ2_ORIGIN="http://127.0.0.1:${HQ2_PORT}"
  local db_url="postgres://hqtest:hqtest@localhost:5434/${SPIKE_DB_NAME}?sslmode=disable&TimeZone=America/New_York"
  ( cd "$REPO_ROOT/backend" && exec setsid env \
      PORT="$HQ2_PORT" DB_URL="$db_url" STATIC_DIR=../ \
      SUPERADMIN_CONFIG=config/superadmins.yaml \
      TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
      MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= \
      ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= SMTP_ADDR= \
      SMTP_USERNAME= SMTP_PASSWORD= STORAGE_KEY= STORAGE_SECRET= \
      STORAGE_BUCKET= STORAGE_REGION= STORAGE_ENDPOINT= \
      go run ./cmd/server/ ) > "${HQ_LOG}.degraded" 2>&1 &
  HQ2_PID=$!
  _wait_health "$HQ2_ORIGIN" || { tail -20 "${HQ_LOG}.degraded" >&2; cannot_run "the degraded HQ server never answered health on :$HQ2_PORT"; }
  echo "#   degraded server  : $HQ2_ORIGIN (pid $HQ2_PID — sync door env UNSET on purpose)"
}

# Login as the bootstrapped superadmin; sets $COOKIE (the hq_session value).
hq_login() {
  local origin="${1:-$HQ_ORIGIN}" jar
  jar="$(mktemp)"
  local body
  body="$(curl -s -c "$jar" -X POST "$origin/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")" \
    || cannot_run "login POST failed against $origin"
  COOKIE="$(awk '$6 == "hq_session" {print $7}' "$jar")"
  rm -f "$jar"
  [ -n "$COOKIE" ] || { echo "login body: $body" >&2; fail "login did not set hq_session — the bootstrapped superadmin is the shipped path and it refused"; }
}
