#!/usr/bin/env bash
# marketing/sync/harness/refusal-run.sh — the B-432 validation gate for card
# refusal-holds-before-sync (run 20260906-2; B-345 precedent: the runnable
# script IS the verdict).
#
# THIS GATE IS THE OWED GAP-1 VALIDATION RUN (goal ledger Comebacks): spike 02
# (`refusal-holds-during-window`) re-executed with the PROTOTYPE policy
# replaced by the SHIPPED policy source — the f2-run.sh precedent, wrapper-free
# against the shipped code. See refusal-harness.mjs for the seven legs.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  green: the refusal holds through the window on the shipped source,
#           decision 166 survives, the codes-arrive-first sub-case is refused,
#           and the discriminator lands distinguishable — or, in red mode, the
#           preserved pre-card shape failed the assertions AS EXPECTED.
#   exit 1  a leg failed (green) — or the red probe PASSED (the assertions do
#           not catch the defect class they exist for).
#   exit 2  could not run.
#
# USAGE
#   refusal-run.sh                # green — the PRIMARY gate: shipped modules
#   refusal-run.sh red-preserved  # the PRE-CARD policy shape (Map-or-null) in
#                                 # place of the shipped source: B-432 returns
#                                 # and the assertions must red
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase project only (RECONCILE mode, never --fresh); never :5433,
# never :5434, no hosted project. lib.sh prints resolved coordinates read-only
# before any write.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../.." && pwd)"

MODE="${1:-green}"
case "$MODE" in green|red-preserved) ;; *) echo "usage: refusal-run.sh [green|red-preserved]" >&2; exit 64 ;; esac

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply"

echo
echo "── the shipped artifacts under test (identity pinned, not assumed) ──"
for f in marketing/sync/replicas.js marketing/sync/push-replication.js \
         marketing/submit-machine.js lib/xstate.umd.min.js; do
  [ -f "$REPO_ROOT/$f" ] || cannot_run "$f missing — this gate tests the SHIPPED artifact, not a copy"
  printf '#   %-40s sha256 %s\n' "$f" "$(shasum -a 256 "$REPO_ROOT/$f" | cut -c1-16)…"
done

echo
echo "── discriminator DDL enumerated (the committed 20260906000300 migration) ──"
psqlq <<'SQL' | sed 's/^/#   /'
select 'policy_unresolved column: ' || count(*)
  from information_schema.columns
 where table_schema='public' and table_name='scan_attempts' and column_name='policy_unresolved';
select 'check constraint        : ' || coalesce(string_agg(pg_get_constraintdef(oid), '; '), '(none)')
  from pg_constraint
 where conrelid = 'public.scan_attempts'::regclass and conname = 'scan_attempts_names_a_code';
SQL
psqlq <<'SQL' | grep -qx '1' || cannot_run "public.scan_attempts has no policy_unresolved column — migration 20260906000300 did not apply, and the client MUST NOT send the field before it does (PGRST204, the F-2 poison class)"
select count(*)::text from information_schema.columns
 where table_schema='public' and table_name='scan_attempts' and column_name='policy_unresolved';
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
RH_JWT="$TOKEN_A" RH_DB_CID="$DB_CID" node "$SCRIPT_DIR/refusal-harness.mjs" "$MODE"
NODE_EXIT=$?
set -e

if [ "$MODE" = "green" ]; then
  if [ "$NODE_EXIT" -eq 0 ]; then
    printf '\n✅ VERDICT: GREEN — B-432 is closed on the SHIPPED policy source: the refusal holds through the window AND the codes-arrive-first sub-case, decision 166 survives, and the discriminator lands distinguishable.\n'
  else
    printf '\n🛑 VERDICT: RED — a leg failed (node exit %s); see the log above.\n' "$NODE_EXIT"
  fi
else
  if [ "$NODE_EXIT" -ne 0 ]; then
    printf '\n🔴 RED-FIRST DEMONSTRATED (%s): the PRE-CARD policy shape fails these assertions (node exit %s). This non-zero exit IS the evidence that the green run measures something.\n' "$MODE" "$NODE_EXIT"
  else
    printf '\n🛑 RED PROBE PASSED (%s): the assertions do NOT catch the defect class — the harness is not evidence. Fix the harness.\n' "$MODE"
  fi
fi
exit "$NODE_EXIT"
