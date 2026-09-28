#!/usr/bin/env bash
# lib.sh — shared plumbing for the supabase/verify harnesses.
# Card 1 `supabase-schema-and-rls`, night-crew run 20260904. Sourced, not executed.
#
# Provides: substrate bring-up (RECONCILE mode only), read-only coordinate
# printing before any write (the decision-155 habit), psql/apply helpers, and
# the Go PATH repair the non-interactive shell needs for mintjwt/rtprobe.
#
# 🛑 TARGET DISCIPLINE. The ONLY database these harnesses touch is the committed
# LOCAL `spike-supabase` compose project (throwaway creds, committed on purpose).
# NEVER :5433 (the dev AND production cluster — destroying it has happened,
# B-141/B-143, ledger decision 155). NEVER :5434 (the Playwright/Go test pg).
# NEVER any hosted supabase.com project (none exists). env-up.sh is invoked with
# NO FLAGS — reconcile mode; `--fresh` from a harness could eat another
# session's stack and is forbidden here.

set -euo pipefail

VERIFY_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SUPABASE_DIR="$(cd -- "$VERIFY_DIR/.." && pwd)"
REPO_ROOT="$(cd -- "$SUPABASE_DIR/.." && pwd)"
QA="$REPO_ROOT/.night-crew/qa/spike-supabase"
MIGRATIONS_DIR="$SUPABASE_DIR/migrations"
SEED_FILE="$SUPABASE_DIR/seed.sql"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

# Go for mintjwt/rtprobe — non-interactive shells do not carry it (run-mechanics
# rule; without this the minter dies `go: not found` and it LOOKS like a
# substrate failure when it is a PATH failure).
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH

# Bring the substrate to GREEN (reconcile) and resolve the compose handle +
# db container id. Prints the resolved target coordinates read-only FIRST.
substrate_up() {
  echo "# target coordinates (read-only statement before any write):"
  echo "#   compose project : spike-supabase (throwaway LOCAL substrate)"
  echo "#   compose file    : $REPO_ROOT/docker-compose.supabase.yml"
  echo "#   mode            : RECONCILE (env-up.sh, no flags — never --fresh from a harness)"
  echo "#   NOT :5433 (dev/prod cluster), NOT :5434 (test pg), NOT any hosted supabase.com project"
  echo
  echo "── substrate up (env-up.sh, idempotent reconcile) ──"
  "$QA/env-up.sh" || cannot_run "env-up.sh did not reach GREEN — the substrate is a precondition, not this harness's premise"

  # Anchor compose exactly the way env-up.sh does (path-stability fix, run 20260806).
  ANCHOR=""
  if command -v git >/dev/null 2>&1; then
    _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
    [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
  fi
  [ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
  DC=(docker compose -p spike-supabase --project-directory "$ANCHOR" -f "$REPO_ROOT/docker-compose.supabase.yml")
  DB_CID="$("${DC[@]}" ps -q db)"
  [ -n "$DB_CID" ] || cannot_run "no db container after env-up GREEN"
  echo "#   db container    : $DB_CID"
  echo "#   db host port    : $("${DC[@]}" port db 5432 | awk -F: '{print $NF}') (Docker-assigned)"
  echo "#   db role         : supabase_admin, via docker exec (no host psql involved)"
}

# Quiet tuples-only query runner (reads SQL on stdin).
psqlq() { docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -qtA; }

# Apply one SQL file. ON_ERROR_STOP=1 is load-bearing: without it psql reports
# success while having skipped past a failed statement.
psqlf() {
  docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q -f - < "$1"
}

# Apply every migration in lexicographic filename order, then the TEST seed.
# Card 2 (`redeem-rpc-race-proof`) extends supabase/migrations/ with a
# later-sorting file and inherits this loop unchanged.
apply_all() {
  local f found=0
  for f in "$MIGRATIONS_DIR"/*.sql; do
    [ -e "$f" ] || break
    found=1
    echo "  → migrations/$(basename "$f")"
    psqlf "$f" || return 1
  done
  if [ "$found" != 1 ]; then
    echo "  no migration files in $MIGRATIONS_DIR" >&2
    return 1
  fi
  [ -f "$SEED_FILE" ] || { echo "  seed file missing: $SEED_FILE" >&2; return 1; }
  echo "  → seed.sql (TEST fixtures — local/test substrates only, never production)"
  psqlf "$SEED_FILE" || return 1
}

# Drop ONLY this card's objects — a test-fixture reset of the THROWAWAY
# substrate so the fresh leg (and the red-first probe) starts from bare.
# Touches nothing else in the stack: spike_notes, hq_*, _realtime.* stay put.
# Dropping a table also removes it from any publication, so this really is
# "the card's objects do not exist" afterwards.
reset_bare() {
  psqlq <<'SQL' >/dev/null
drop table if exists public.scan_attempts cascade;
drop table if exists public.codes cascade;
drop table if exists public.campaigns cascade;
drop table if exists public.marketing_settings cascade;
SQL
}

# Portable UUID for per-run rows (macOS and Linux both carry uuidgen; fall back
# to python3).
newuuid() {
  if command -v uuidgen >/dev/null 2>&1; then
    uuidgen | tr 'A-Z' 'a-z'
  else
    python3 -c 'import uuid; print(uuid.uuid4())'
  fi
}
