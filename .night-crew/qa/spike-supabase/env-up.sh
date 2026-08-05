#!/usr/bin/env bash
# env-up.sh — take this machine to "Supabase + RxDB both up, schema applied,
# healthy", UNATTENDED. Night-crew card C1 `spike-a-environment-up`.
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#
#   exit 0   the environment is up: three containers healthy, ALL THREE fixture
#            schemas applied (spike_notes AND the hq_* bridge tables AND their
#            five policies — each asserted by name, because asserting only
#            spike_notes let a stack with no hq_* tables at all exit 0, which is
#            the same "container up, no schema" conflation relocated to the
#            other half of the schema), PostgREST discriminating by RLS,
#            Realtime SUBSCRIBED, and a real RxDB database round-tripped.
#   exit !=0 it is not. Every failure path below prints WHAT failed and WHERE
#            to look, then exits non-zero.
#
# There is deliberately no "warn and continue" anywhere in this file, and no
# leg is advisory. A step that cannot decide is a FAILURE, never a pass. A
# spike script that silently no-ops is the exact defect class this cycle
# exists to retire — if you are about to add `|| true` to a health assertion,
# you are about to destroy the only thing this script is for.
#
# There is exactly ONE self-healing action (step 6's reconcile leg: restart
# Realtime when it is CHANNEL_ERROR against a healthy db). It performs a named
# remedy and then RE-ASSERTS; a failing re-assertion is still RED. That is a
# reconciler, not a `|| true`, and the distinction is the whole file.
# ═══════════════════════════════════════════════════════════════════════════
#
# ⚠ LOCAL SPIKE ONLY. This drives the throwaway `spike-supabase` compose
#   project defined in docker-compose.supabase.yml, whose credentials are
#   throwaway values committed to git on purpose (see that file's banner).
#   It NEVER touches HQ prod, the hosted world (there is no supabase.com
#   account involved at any point), HQ's night-crew environment
#   (docker-compose.nc.yml), or the dev Postgres on :5433. The `-p
#   spike-supabase` project name on every compose call is what guarantees it.
#
# USAGE
#   .night-crew/qa/spike-supabase/env-up.sh            # idempotent: reconcile + verify
#   .night-crew/qa/spike-supabase/env-up.sh --fresh    # destroy volumes first, then bring up
#   .night-crew/qa/spike-supabase/env-up.sh --health   # verify only; do not touch the stack
#
#   or via the repo Taskfile:  task spike:up | spike:up:fresh | spike:health | spike:down
#
#   --fresh is the honest "clean machine" test: it does `down --volumes`, so
#   Postgres re-runs initdb and every schema is re-applied from source. Use it
#   when the question is "does this come up from nothing", which is C1's
#   question. Plain `--fresh`-less runs answer the cheaper question "is it up
#   now", which is what a later card's `task demo:sync` will want.
#
# WHAT IT DOES NOT DO
#   Teardown. Per W1's runbook, destroying the stack stays a deliberate,
#   separate act (`task spike:down`), so an unattended verify can never eat
#   another session's running substrate.
#
#   🛑 THAT CLAIM WAS ONCE FALSE, AND IS ONLY TRUE NOW BECAUSE OF TWO FIXES.
#   Until run 20260806 the db service had NO VOLUME for PGDATA — the data
#   directory lived in the container's writable layer — and the compose file's
#   relative bind mounts made the container config hash depend on the ABSOLUTE
#   host path it was resolved from. So a plain `up -d` from a different checkout
#   of this repo (a worktree, a second clone) RECREATED the db container and
#   silently destroyed every table in it, while `docker ps` went right on saying
#   `Up (healthy)`. Both mechanisms are fixed: PGDATA is on the `spike-db-data`
#   named volume, and every compose call below passes a stable
#   `--project-directory`. Do not undo either and then leave this paragraph up.

set -euo pipefail

# --------------------------------------------------------------------------
# Locate the repo root from this script's own path, so the script works from
# any cwd, from a git worktree, and from a Taskfile whose dir differs.
# --------------------------------------------------------------------------
SPIKE_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SPIKE_DIR/../../.." && pwd)"

COMPOSE_FILE="$REPO_ROOT/docker-compose.supabase.yml"
PROJECT="spike-supabase"

# --------------------------------------------------------------------------
# 🛑 PATH-STABLE COMPOSE INVOCATION. Do not simplify this away.
#
# docker-compose.supabase.yml bind-mounts its two initdb scripts by RELATIVE
# path. Compose resolves those against the *project directory* and bakes the
# resulting ABSOLUTE host path into the container config hash — so running
# `up -d` from a git worktree and then from the main checkout makes compose
# RECREATE the db container, every time, for no reason. Reproduced three times
# on run 20260806, and until PGDATA got a named volume that recreate was total
# data loss (it is the most likely cause of this card's own headline finding:
# "three containers, five days old, no schema").
#
# `git rev-parse --git-common-dir` points every linked worktree at the SAME
# main .git, so its parent is a stable anchor shared by the whole clone. We
# pass it as --project-directory while still passing THIS checkout's compose
# file with -f, so path resolution is stable but the file's *content* is
# whatever the caller has checked out.
#
# Override with SPIKE_ANCHOR=/some/path if you deliberately want a second,
# independently-anchored stack. Fall back to REPO_ROOT when git is unavailable.
# --------------------------------------------------------------------------
if [ -n "${SPIKE_ANCHOR:-}" ]; then
  ANCHOR="$SPIKE_ANCHOR"
else
  ANCHOR=""
  if command -v git >/dev/null 2>&1; then
    _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
    [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
  fi
  [ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
fi
[ -d "$ANCHOR/.night-crew/qa/spike-supabase/initdb" ] \
  || ANCHOR="$REPO_ROOT"   # anchor must actually contain the bind-mount sources

DC=(docker compose -p "$PROJECT" --project-directory "$ANCHOR" -f "$COMPOSE_FILE")

# Go is not on a non-interactive shell's PATH on this box, and the token is
# minted by the Go minter on purpose (HQ's Go backend is the token authority).
# Without this the minter dies `go: not found` and it LOOKS like a substrate
# failure when it is a PATH failure.
#
# 🛑 NOT a hardcoded home directory. A script whose whole claim is "clean
# machine, unattended" cannot carry its author's $HOME as a constant. We prepend
# the conventional install locations that actually exist, and if none of them
# yields `go` the preflight below still names it by name.
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin "$HOME/.local/go/bin"; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH

DB_HEALTHY_TIMEOUT="${SPIKE_DB_TIMEOUT:-300}"   # seconds; initdb on a cold volume is slow
HTTP_READY_TIMEOUT="${SPIKE_HTTP_TIMEOUT:-180}" # seconds; realtime runs migrations on boot

MODE="reconcile"
case "${1:-}" in
  --fresh)  MODE="fresh" ;;
  --health) MODE="health" ;;
  "")       MODE="reconcile" ;;
  *) echo "usage: $(basename "$0") [--fresh|--health]" >&2; exit 64 ;;
esac

STEP=0
step() { STEP=$((STEP + 1)); printf '\n── %d. %s ───────────────────────────────\n' "$STEP" "$1"; }
fail() { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit "${2:-1}"; }

printf '# env-up.sh — mode=%s repo=%s\n' "$MODE" "$REPO_ROOT"
printf '# compose anchor (--project-directory, path-stability fix) = %s\n' "$ANCHOR"
printf '# started %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --------------------------------------------------------------------------
# 0. Preflight. Missing tooling is a RED verdict with a name on it, not a
#    confusing failure three steps later.
# --------------------------------------------------------------------------
step "preflight — required tooling"
# NOTE: psql is deliberately absent from this list — it runs INSIDE the db
# container via `docker exec`, so the host never needs a Postgres client.
for bin in docker curl node npm go; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    fail "required tool not on PATH: $bin"
  fi
  printf '  %-6s %s\n' "$bin" "$(command -v "$bin")"
done
if ! docker compose version >/dev/null 2>&1; then
  fail "Compose v2 unavailable — this needs 'docker compose', not 'docker-compose'"
fi
printf '  compose %s\n' "$(docker compose version --short 2>/dev/null || echo '?')"
printf '  go      %s\n' "$(go version)"
printf '  node    %s\n' "$(node -v)"
[ -f "$COMPOSE_FILE" ] || fail "compose file missing: $COMPOSE_FILE"
if ! docker info >/dev/null 2>&1; then
  fail "the Docker daemon is not reachable — 'docker info' failed"
fi
echo "  docker daemon: reachable"

# --------------------------------------------------------------------------
# 1. Bring the stack up (or tear it down first, under --fresh).
# --------------------------------------------------------------------------
if [ "$MODE" = "health" ]; then
  step "skipping bring-up (--health): verifying whatever is already running"
else
  if [ "$MODE" = "fresh" ]; then
    step "fresh — destroying the spike stack and its volumes"
    echo "  (this affects ONLY the '$PROJECT' compose project)"
    "${DC[@]}" down --volumes --remove-orphans || fail "'docker compose down --volumes' failed"
  fi

  step "bringing the stack up (idempotent)"
  # `up -d` pulls anything missing, so a genuinely clean machine needs no
  # separate pull step. ~3.3 GB on a cold cache, dominated by supabase/postgres.
  "${DC[@]}" up -d || fail "'docker compose up -d' failed — see 'docker compose -p $PROJECT logs'"
  "${DC[@]}" ps
fi

# --------------------------------------------------------------------------
# 2. Wait for Postgres to be genuinely ready.
#
#    The compose healthcheck is an AUTHENTICATED query, not pg_isready:
#    pg_isready answers on the temporary bootstrap server that runs during
#    initdb, so dependents would start while migrate.sh is still installing
#    the anon/authenticated/authenticator role set. Waiting on the container's
#    health state is therefore waiting on the right signal.
# --------------------------------------------------------------------------
step "waiting for db to report healthy (timeout ${DB_HEALTHY_TIMEOUT}s)"
DB_CID="$("${DC[@]}" ps -q db || true)"
[ -n "$DB_CID" ] || fail "the db service has no container — 'docker compose -p $PROJECT ps -a' will say why"
deadline=$(( $(date +%s) + DB_HEALTHY_TIMEOUT ))
while :; do
  state="$(docker inspect -f '{{.State.Status}}' "$DB_CID" 2>/dev/null || echo missing)"
  health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$DB_CID" 2>/dev/null || echo none)"
  [ "$health" = "healthy" ] && { echo "  db: $state/$health"; break; }
  case "$state" in
    exited|dead|missing) fail "db container is '$state' — 'docker logs $DB_CID' will say why" ;;
  esac
  [ "$(date +%s)" -lt "$deadline" ] || fail "db never became healthy within ${DB_HEALTHY_TIMEOUT}s (last state=$state health=$health)"
  sleep 3
done

# --------------------------------------------------------------------------
# 3. Apply every fixture schema. All three files are idempotent by
#    construction, so this is safe on a warm stack as well as a cold one.
#
#    ON_ERROR_STOP=1 is load-bearing: without it psql reports success while
#    having skipped past a failed statement — the schema-shaped version of the
#    silent no-op this script exists to prevent.
# --------------------------------------------------------------------------
if [ "$MODE" = "health" ]; then
  step "skipping schema apply (--health) — the health assertion must find the schema already there"
else
step "applying fixture schemas"
apply_sql() {
  local f="$1"
  [ -f "$SPIKE_DIR/sql/$f" ] || fail "fixture missing: $SPIKE_DIR/sql/$f"
  printf '  → %s\n' "$f"
  if ! docker exec -i "$DB_CID" \
        psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q -f - \
        < "$SPIKE_DIR/sql/$f"; then
    fail "applying sql/$f failed"
  fi
}
# W1's substrate contract (public.spike_notes) — the RxDB replication shape.
apply_sql spike-fixture.sql
# The HQ-bridge tables. Fixture first (tables/grants/seed, RLS OFF — the
# deliberate red state), policies second (RLS ON — the green state). Applying
# them in the other order is meaningless.
apply_sql hq-bridge-fixture.sql
apply_sql hq-bridge-policies.sql
fi

# --------------------------------------------------------------------------
# 4. Resolve the Docker-assigned host ports and wait for both HTTP services.
#    W1's compose publishes bare container ports on purpose, so two stacks can
#    coexist; the host side is therefore never a constant.
#
#    🛑 IT IS NOT EVEN CONSTANT FOR THE LIFETIME OF ONE `up`. Docker frees an
#    ephemeral published port on stop and picks a NEW one on start, so
#    `docker compose restart realtime` moves the Realtime host port. Measured
#    on run 20260806's fix round: 50959 → 50135 across a single restart. That
#    is why resolution lives in a FUNCTION and the reconcile leg re-runs it —
#    the first version of that leg reused the pre-restart port and timed out for
#    180s against a Realtime that was already healthy on a different port.
# --------------------------------------------------------------------------
port_of() {
  local svc="$1" cport="$2" out p
  out="$("${DC[@]}" port "$svc" "$cport" 2>/dev/null || true)"
  [ -n "$out" ] || fail "could not resolve the published host port for $svc:$cport"
  p="${out##*:}"
  # 🛑 `-n` is NOT enough. For a service that is DEFINED but not RUNNING,
  #    `docker compose port` prints `:0` — so `${out##*:}` is the string "0",
  #    which is non-empty and sails through. The failure then surfaced 180s
  #    later as a wait_http timeout naming the wrong thing. Demand a real,
  #    non-zero port number and fail here, immediately, with the cause.
  case "$p" in
    ''|*[!0-9]*) fail "$svc:$cport resolved to a non-numeric host port ('$out') — is the service running? 'docker compose -p $PROJECT ps -a'" ;;
  esac
  [ "$p" -gt 0 ] || fail "$svc:$cport resolved to port 0 ('$out') — the service is defined but not running; 'docker compose -p $PROJECT ps -a' will say why"
  echo "$p"
}
resolve_ports() {
  step "resolving host ports"
  DBP="$(port_of db 5432)"
  RESTP="$(port_of rest 3000)"
  RTP="$(port_of realtime 4000)"
  printf '  db=%s rest=%s realtime=%s\n' "$DBP" "$RESTP" "$RTP"
}
resolve_ports

wait_http() {
  local name="$1" url="$2" dl code
  dl=$(( $(date +%s) + HTTP_READY_TIMEOUT ))
  while :; do
    # 🛑 `|| echo 000` here is a BUG, and it cost this card a false green on
    #    the first fresh run: curl ALREADY prints `000` on a connection
    #    failure, so the fallback appended a second one and `$code` became
    #    `000\n000` — which is not equal to `000`, so the gate passed and the
    #    health assertion ran against a Realtime that had not finished booting.
    #    Let curl's own `000` stand; use `|| true` only to survive `set -e`.
    code="$(curl -s -o /dev/null -m 5 -w '%{http_code}' "$url" 2>/dev/null || true)"
    code="${code:-000}"
    # Any HTTP status means the server is listening and answering. A 401/404
    # from Realtime's root is a live server; 000 is no server at all.
    [ "$code" != "000" ] && { echo "  $name: HTTP $code"; return 0; }
    [ "$(date +%s)" -lt "$dl" ] || fail "$name never answered on $url within ${HTTP_READY_TIMEOUT}s"
    sleep 3
  done
}
step "waiting for PostgREST and Realtime to answer (timeout ${HTTP_READY_TIMEOUT}s each)"
wait_http PostgREST "http://127.0.0.1:${RESTP}/"
wait_http Realtime  "http://127.0.0.1:${RTP}/"

# --------------------------------------------------------------------------
# 5. RxDB side — install from the committed lockfile.
#
#    `npm ci`, not `npm install`: node_modules/ is gitignored, so a clean
#    clone has none, and the lockfile is the only thing that makes the install
#    reproducible. This is an ISOLATED npm project — the repo-root
#    package.json is the Playwright environment for every card and is not
#    touched here.
# --------------------------------------------------------------------------
step "installing the RxDB harness (npm ci, isolated project)"
if [ -d "$SPIKE_DIR/rxdb/node_modules" ]; then
  echo "  node_modules present — skipping install"
else
  ( cd "$SPIKE_DIR/rxdb" && npm ci --no-audit --no-fund ) \
    || fail "'npm ci' failed in $SPIKE_DIR/rxdb — the RxDB half cannot come up"
fi

# --------------------------------------------------------------------------
# 6. The health assertion. This is the leg that decides the verdict: it proves
#    the schema is really applied, that RLS really discriminates, that
#    Realtime really subscribes, and that RxDB really runs — as opposed to
#    "three containers are in `docker ps`", which is not the same claim and
#    conflating the two is precisely what this card exists to prevent.
# --------------------------------------------------------------------------
step "health assertion — rest, realtime, schema, rxdb"
HC_OUT="$(mktemp)"
trap 'rm -f "$HC_OUT"' EXIT

run_healthcheck() {
  ( cd "$SPIKE_DIR/rxdb" && node healthcheck.js ) 2>&1 | tee "$HC_OUT"
  return "${PIPESTATUS[0]}"
}

if ! run_healthcheck; then
  # ------------------------------------------------------------------------
  # RECONCILE LEG — the one self-healing action in this script, and it is
  # narrow on purpose.
  #
  # A db container recreate invalidates Realtime's logical-replication slot.
  # Realtime does not recover: its WebSocket answers CHANNEL_ERROR (transport
  # failure) FOREVER — measured across three consecutive invocations, all
  # EXIT=1 — while `docker compose up -d` prints
  #   Container spike-supabase-realtime-1  Running
  # and does nothing, because compose will not restart a running-but-broken
  # service. Only `docker compose -p spike-supabase restart realtime` recovers
  # it. Without this leg the script could BREAK the stack (by recreating the
  # db) and then report RED about it forever, unable to reconcile what it
  # itself did — which is not a reconciler.
  #
  # 🛑 This is NOT a "warn and continue" and it is NOT `|| true`. It performs a
  # named, bounded REMEDY and then RE-ASSERTS. If the re-assertion fails, the
  # verdict is RED. Nothing here can turn a failing leg into a pass.
  # ------------------------------------------------------------------------
  if [ "$MODE" != "health" ] && grep -q 'FAIL  realtime' "$HC_OUT"; then
    step "reconcile — realtime is CHANNEL_ERROR against a healthy db; restarting it"
    echo "  (compose reports a broken-but-running service as 'Running' and will not"
    echo "   restart it; an explicit 'restart realtime' is the only thing that recovers"
    echo "   a Realtime whose replication slot died with a db recreate)"
    "${DC[@]}" restart realtime || fail "'docker compose restart realtime' failed"
    # 🛑 RE-RESOLVE. A restart moves the ephemeral published port (measured:
    #    50959 → 50135). Waiting on the pre-restart port burns the full 180s
    #    timeout against a service that is already healthy somewhere else.
    resolve_ports
    wait_http Realtime "http://127.0.0.1:${RTP}/"
    step "re-asserting health after the realtime restart"
    if ! run_healthcheck; then
      fail "the health assertion still failed after restarting realtime — see the FAIL lines above. If 'realtime' is still CHANNEL_ERROR, check 'docker compose -p $PROJECT logs realtime'; if a schema leg failed, re-run WITHOUT --health so the fixtures are applied."
    fi
  else
    fail "the health assertion failed — see the FAIL lines above. Remedies by leg: 'schema' → re-run without --health so sql/*.sql are applied (or --fresh); 'realtime' CHANNEL_ERROR → 'docker compose -p $PROJECT restart realtime' (plain 'up -d' will NOT do it — it reports the broken service as merely Running); 'rest' → check 'docker compose -p $PROJECT logs rest'."
  fi
fi

printf '\n══════════════════════════════════════════════════════════\n'
printf '✅ VERDICT: GREEN — Supabase + RxDB up, schema applied, healthy\n'
printf '   db=%s  rest=%s  realtime=%s   (project: %s)\n' "$DBP" "$RESTP" "$RTP" "$PROJECT"
printf '   teardown is a separate, deliberate act:  task spike:down\n'
printf '   finished %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf '══════════════════════════════════════════════════════════\n'
exit 0
