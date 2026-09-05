#!/usr/bin/env bash
# marketing/sync/harness/f2-run.sh — the standalone F-2 guard gate for card
# requires-online-replication (run 20260906; B-345 precedent: the runnable
# script IS the verdict). This gate IS the owed GAP-1 validation run: spike 03
# (f2-push-poison-and-guard) re-executed against the SHIPPED guard — the guard
# inside ../push-replication.js and the COMMITTED landing-path migration
# (20260906000200), wrapper-free. See f2-harness.mjs for the legs.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  green: the guard diverts before redeem(), both attempts land,
#           the audit row reaches the server — or, in red mode, the probe
#           failed AS EXPECTED.
#   exit 1  a leg failed (green) — or the red probe PASSED (the assertion
#           does not catch the defect class it exists for).
#   exit 2  could not run.
#
# USAGE
#   f2-run.sh                # green — the PRIMARY gate: shipped modules
#   f2-run.sh red-unflagged  # the 64-hex row WITHOUT unverified_code: the
#                            # guard's discriminator lost → redeem-first path
#                            # → head-of-line poison reds the assertions
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase project only (RECONCILE mode, never --fresh); never :5433,
# never :5434, no hosted project. lib.sh prints resolved coordinates read-only
# before any write.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../.." && pwd)"

MODE="${1:-green}"
case "$MODE" in green|red-unflagged) ;; *) echo "usage: f2-run.sh [green|red-unflagged]" >&2; exit 64 ;; esac

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply"

echo
echo "── landing-path DDL enumerated (the committed 20260906000200 migration) ──"
psqlq <<'SQL' | sed 's/^/#   /'
select 'code_id nullable: ' || (is_nullable = 'YES')
  from information_schema.columns
 where table_schema='public' and table_name='scan_attempts' and column_name='code_id';
select 'token_hash column: ' || count(*)
  from information_schema.columns
 where table_schema='public' and table_name='scan_attempts' and column_name='token_hash';
select 'check constraint : ' || coalesce(string_agg(conname, ', '), '(none)')
  from pg_constraint
 where conrelid = 'public.scan_attempts'::regclass and conname = 'scan_attempts_names_a_code';
SQL

echo
echo "── device JWT (throwaway secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN_A="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

# Node module resolution: borrow the proven QA rxdb node_modules via symlink
# (walk-up resolution; nothing installed here; the link is gitignored).
[ -e "$SCRIPT_DIR/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/node_modules"

echo
echo "── mode: $MODE ──"
set +e
F2_JWT="$TOKEN_A" F2_DB_CID="$DB_CID" node "$SCRIPT_DIR/f2-harness.mjs" "$MODE"
NODE_EXIT=$?
set -e

if [ "$MODE" = "green" ]; then
  if [ "$NODE_EXIT" -eq 0 ]; then
    printf '\n✅ VERDICT: GREEN — the F-2 guard holds on the shipped surface: unverified attempts divert BEFORE redeem(), land on the distinct path, and never strand the queue.\n'
  else
    printf '\n🛑 VERDICT: RED — a leg failed (node exit %s); see the log above.\n' "$NODE_EXIT"
  fi
else
  if [ "$NODE_EXIT" -ne 0 ]; then
    printf '\n🔴 RED-FIRST DEMONSTRATED (%s): the unflagged 64-hex row poisons the queue and the assertions red (node exit %s). This non-zero exit IS the evidence.\n' "$MODE" "$NODE_EXIT"
  else
    printf '\n🛑 RED PROBE PASSED (%s): the assertion does NOT catch the defect class — the harness is not evidence. Fix the harness.\n' "$MODE"
  fi
fi
exit "$NODE_EXIT"
