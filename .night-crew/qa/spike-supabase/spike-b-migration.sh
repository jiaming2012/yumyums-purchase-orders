#!/usr/bin/env bash
# spike-b-migration.sh — SPIKE B. Night-crew card S `spike-b-migration-rehearsal`.
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#
#   exit 0   HQ-shaped data landed in the substrate AND surfaced in RxDB:
#            a fresh scratch Postgres carrying a subset of HQ's real schema came
#            up, was fixtured, its rows were transformed and loaded THROUGH
#            PostgREST into spike A's Supabase substrate, RLS discriminated over
#            THOSE MIGRATED ROWS on both axes (identity and live entitlement),
#            and a real RxDB client replicating with a real signed token ended up
#            holding exactly the migrated rows it was entitled to, byte-for-byte,
#            and not one row more.
#   exit !=0 it did not. The failing leg is named and the reason printed.
#
# 🛑 A RED VERDICT IS A SUCCESSFUL SPIKE. This card exists to find out whether
#    the migration works, not to make it work. If a leg reds: record it with the
#    captured output and stop. Debugging the compose file or this script is
#    legitimate. Rewriting the goal so it passes is not.
#
# There is deliberately no "warn and continue" anywhere in this file and no
# advisory leg. A step that cannot decide is a FAILURE. That is spike A's rule
# (env-up.sh:18-27) carried forward unchanged; if you are about to add `|| true`
# to an assertion you are about to destroy the only thing this script is for.
# ═══════════════════════════════════════════════════════════════════════════
#
# ⚠ CONTAINERS — the standing rule, absolute.
#   The HQ-shaped Postgres is a FRESH SCRATCH CONTAINER of its own, under the
#   compose project `spike-b-hq`, on a DOCKER-ASSIGNED EPHEMERAL host port.
#     * NEVER :5433. That cluster is PRODUCTION — a probe against it destroyed
#       the prod database on 2026-08-06. No command in this file resolves any
#       database name against :5433.
#     * NEVER :5434 (`yumyums-test-pg`) — the Playwright/Go proof substrate.
#     * NEVER 5432.
#   It is created at the top of this script and DESTROYED at the bottom, so the
#   script is re-runnable from nothing. `--keep` suspends only the teardown.
#
# ⚠ SPIKE A'S STACK IS CONSUMED, NOT MODIFIED.
#   env-up.sh is called in its default RECONCILE mode — idempotent, brings up
#   what is missing, destroys nothing. It is NOT called with --fresh by default,
#   because --fresh does `down --volumes` on the `spike-supabase` project and
#   would eat another session's running substrate; the card's isolation rule says
#   this card must not collide with spike A's stack. Pass --fresh-substrate to
#   opt in deliberately.
#
#   "Fresh" on this card's own side is guaranteed three ways that need nobody
#   else's stack destroyed: the HQ-shaped Postgres is created and destroyed every
#   run; each RxDB database is created new under a per-run name; and the
#   migration target is reset to empty (scoped to exactly this fixture's keys)
#   before every load.
#
# USAGE
#   .night-crew/qa/spike-supabase/spike-b-migration.sh
#   .night-crew/qa/spike-supabase/spike-b-migration.sh --keep              # leave the scratch pg up
#   .night-crew/qa/spike-supabase/spike-b-migration.sh --fresh-substrate   # also rebuild spike A's stack from nothing
#
#   or via the repo Taskfile:  task spike:migration

set -euo pipefail

SPIKE_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SPIKE_DIR/../../.." && pwd)"

HQ_COMPOSE="$SPIKE_DIR/docker-compose.hq-source.yml"
HQ_PROJECT="spike-b-hq"

KEEP=0
SUBSTRATE_ARGS=()
for arg in "$@"; do
  case "$arg" in
    --keep)             KEEP=1 ;;
    --fresh-substrate)  SUBSTRATE_ARGS=(--fresh) ;;
    *) echo "usage: $(basename "$0") [--keep] [--fresh-substrate]" >&2; exit 64 ;;
  esac
done

# Go is not on a non-interactive shell's PATH on this box, and the JWT is minted
# by the Go minter on purpose (HQ's Go backend is the token authority). Without
# this the minter dies `go: not found` and it LOOKS like a substrate failure when
# it is a PATH failure. Same list env-up.sh uses, and for the same reason it is
# not a hardcoded $HOME.
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin "$HOME/.local/go/bin"; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH

STEP=0
step() { STEP=$((STEP + 1)); printf '\n══ %d. %s ═══════════════════════════════\n' "$STEP" "$1"; }
fail() { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit "${2:-1}"; }

# Anchor compose the way env-up.sh does, so the project directory is identical
# whether this runs from the main checkout or a worktree. This file has no bind
# mounts (deliberately — see its banner), so nothing here is path-sensitive; the
# anchor is for consistency with the sibling stack, not necessity.
ANCHOR=""
if command -v git >/dev/null 2>&1; then
  _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
  [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
fi
[ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
HQDC=(docker compose -p "$HQ_PROJECT" --project-directory "$ANCHOR" -f "$HQ_COMPOSE")

export SPIKE_B_RUN_ID="${SPIKE_B_RUN_ID:-b$(date -u +%Y%m%d%H%M%S)}"

printf '# spike-b-migration.sh — HQ-shaped Postgres -> Supabase -> RxDB\n'
printf '# repo   %s\n' "$REPO_ROOT"
printf '# anchor %s\n' "$ANCHOR"
printf '# run    %s\n' "$SPIKE_B_RUN_ID"
printf '# started %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --------------------------------------------------------------------------
# Teardown. Registered BEFORE the container is created, so an abort between
# `up` and the first assertion still cleans up. It must never alter the exit
# status — the exit status is the verdict.
# --------------------------------------------------------------------------
teardown() {
  local rc=$?
  if [ "$KEEP" = "1" ]; then
    printf '\n(--keep) scratch HQ-shaped Postgres left running: docker compose -p %s ... down --volumes\n' "$HQ_PROJECT"
    return $rc
  fi
  printf '\n── teardown: destroying the scratch HQ-shaped Postgres (project %s) ──\n' "$HQ_PROJECT"
  "${HQDC[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || \
    printf '  ⚠ teardown of project %s did not complete cleanly — check `docker ps -a`\n' "$HQ_PROJECT"
  printf '  torn down.\n'
  return $rc
}
trap teardown EXIT

# --------------------------------------------------------------------------
step "preflight — required tooling"
# --------------------------------------------------------------------------
for bin in docker curl node npm go; do
  command -v "$bin" >/dev/null 2>&1 || fail "required tool not on PATH: $bin"
  printf '  %-6s %s\n' "$bin" "$(command -v "$bin")"
done
docker compose version >/dev/null 2>&1 || fail "Compose v2 unavailable — needs 'docker compose'"
docker info >/dev/null 2>&1 || fail "the Docker daemon is not reachable — 'docker info' failed"
[ -f "$HQ_COMPOSE" ] || fail "compose file missing: $HQ_COMPOSE"
for f in sql/hq-source-schema.sql sql/hq-source-fixture.sql rxdb/hq-bridge-env.js rxdb/hq-migrate.js rxdb/hq-verify.js; do
  [ -f "$SPIKE_DIR/$f" ] || fail "spike asset missing: $SPIKE_DIR/$f"
done

# 🛑 ISOLATION ASSERTION, NOT A COMMENT. The card's absolute rule is that this
# spike never resolves a database against the shared :5433 cluster or the :5434
# test container. Assert that the compose file this script is about to run
# publishes NO fixed host port at all, so it cannot possibly bind one of them.
if grep -Eq '^[[:space:]]*-[[:space:]]*"?(5432|5433|5434):' "$HQ_COMPOSE"; then
  fail "docker-compose.hq-source.yml publishes a FIXED host port in the 5432-5434 range. \
:5433 is the PRODUCTION cluster (a probe there destroyed the prod DB on 2026-08-06) and \
:5434 is yumyums-test-pg. This spike must use a Docker-assigned ephemeral port."
fi
echo "  isolation: hq-source compose publishes no fixed host port (Docker-assigned only)"

# --------------------------------------------------------------------------
step "substrate — spike A's Supabase + RxDB environment"
# --------------------------------------------------------------------------
echo "  delegating to env-up.sh (${SUBSTRATE_ARGS[*]:-reconcile}); its exit status gates this leg"
"$SPIKE_DIR/env-up.sh" "${SUBSTRATE_ARGS[@]}" \
  || fail "the substrate did not come up — env-up.sh returned non-zero. Its own output above names the leg. Spike B cannot migrate into a substrate that is not there."

# --------------------------------------------------------------------------
step "HQ-shaped source — fresh scratch Postgres (project $HQ_PROJECT)"
# --------------------------------------------------------------------------
# Always from nothing: down --volumes first, so a leftover container from an
# aborted run can never make this look like it came up clean.
"${HQDC[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
"${HQDC[@]}" up -d || fail "'docker compose up -d' failed for project $HQ_PROJECT"
"${HQDC[@]}" ps

HQ_CID="$("${HQDC[@]}" ps -q hqsrc || true)"
[ -n "$HQ_CID" ] || fail "the hqsrc service has no container — 'docker compose -p $HQ_PROJECT ps -a' will say why"
export SPIKE_B_HQ_CID="$HQ_CID"

deadline=$(( $(date +%s) + 120 ))
while :; do
  state="$(docker inspect -f '{{.State.Status}}' "$HQ_CID" 2>/dev/null || echo missing)"
  health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$HQ_CID" 2>/dev/null || echo none)"
  [ "$health" = "healthy" ] && { echo "  hqsrc: $state/$health"; break; }
  case "$state" in
    exited|dead|missing) fail "hqsrc container is '$state' — 'docker logs $HQ_CID' will say why" ;;
  esac
  [ "$(date +%s)" -lt "$deadline" ] || fail "hqsrc never became healthy within 120s (last state=$state health=$health)"
  sleep 2
done

HQ_PORT="$("${HQDC[@]}" port hqsrc 5432 2>/dev/null | sed 's/.*://' || true)"
case "$HQ_PORT" in
  ''|*[!0-9]*) fail "could not resolve the Docker-assigned host port for hqsrc:5432 (got '$HQ_PORT')" ;;
esac
[ "$HQ_PORT" -gt 0 ] || fail "hqsrc:5432 resolved to port 0 — the service is defined but not running"
case "$HQ_PORT" in
  5432|5433|5434) fail "Docker assigned host port $HQ_PORT to the scratch Postgres, which collides with a protected port (5433 = PRODUCTION cluster, 5434 = yumyums-test-pg). Refusing to continue. Re-run; the ephemeral range will pick another." ;;
esac
printf '  container %s  ·  host port %s (Docker-assigned, ephemeral)\n' "${HQ_CID:0:12}" "$HQ_PORT"

# --------------------------------------------------------------------------
step "applying the HQ-shaped schema subset and its fixture"
# --------------------------------------------------------------------------
# ON_ERROR_STOP=1 is load-bearing: without it psql reports success having skipped
# past a failed statement — the schema-shaped version of the silent no-op.
apply_sql() {
  local f="$1"
  printf '  → %s\n' "$f"
  docker exec -i "$HQ_CID" psql -U hq -d hq_source -v ON_ERROR_STOP=1 -q -f - < "$SPIKE_DIR/sql/$f" \
    || fail "applying sql/$f into the scratch HQ-shaped Postgres failed"
}
apply_sql hq-source-schema.sql
apply_sql hq-source-fixture.sql

# Assert the fixture is really there. "Container up, no schema" is exactly the
# conflation spike A's card was written to detect; it is not repeated here.
counts="$(docker exec -i "$HQ_CID" psql -U hq -d hq_source -t -A -c \
  "select (select count(*) from users)||'/'||(select count(*) from app_permissions)||'/'||(select count(*) from checklist_submissions)||'/'||(select count(*) from submission_responses)")"
echo "  fixture counts users/permissions/submissions/responses = $counts"
[ "$counts" = "3/4/7/15" ] || fail "the HQ-shaped fixture did not land as expected (users/permissions/submissions/responses = $counts, expected 3/4/7/15)"

# --------------------------------------------------------------------------
step "MIGRATE — HQ-shaped rows into the substrate, through PostgREST"
# --------------------------------------------------------------------------
( cd "$SPIKE_DIR/rxdb" && node hq-migrate.js ) \
  || fail "the migration leg failed — see the FAIL line above. It names which assertion could not be made."

# --------------------------------------------------------------------------
step "VERIFY — do the migrated rows surface in RxDB?"
# --------------------------------------------------------------------------
( cd "$SPIKE_DIR/rxdb" && node hq-verify.js ) \
  || fail "the verification leg failed — see the FAIL line above. Landing and surfacing are separate claims; this is the second one."

printf '\n══════════════════════════════════════════════════════════\n'
printf '✅ VERDICT: GREEN — HQ-shaped data migrated into the Supabase substrate\n'
printf '   and surfaced in RxDB, discriminated on both authz axes,\n'
printf '   byte-for-byte identical to the HQ-shaped source.\n'
printf '   hq source: project %s, container %s, host port %s (ephemeral)\n' "$HQ_PROJECT" "${HQ_CID:0:12}" "$HQ_PORT"
printf '   finished %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf '══════════════════════════════════════════════════════════\n'
exit 0
