#!/usr/bin/env bash
# 03-f2-push-poison-and-guard.sh — spike: F-2, measured rather than inferred.
#
# The card's F-2 claim is currently an inference from reading three files:
#   submit-flow.js:240   code_id: SUBMIT_CTX.code_id || SUBMIT_CTX.token_hash
#                        (an unknown code has no code_id → the 64-hex hash)
#   migration 0001       scan_attempts.code_id  uuid not null
#                        redeem(p_code uuid, p_device text)
#   push-replication.js  throws on any non-200 → RxDB retries forever
# Nobody has run it. This spike runs it:
#
#   (a) WHICH endpoint refuses the 64-hex code_id, and with what status —
#       /rpc/redeem (p_code uuid) or /scan_attempts (code_id uuid not null).
#       Which one fails FIRST decides where the guard goes.
#   (b) BLAST RADIUS — with the poison row queued first, does a legitimate
#       attempt behind it land? Head-of-line poisoning, or just a dropped row?
#   (c) A GUARD clears it — and the rejected alternative is DEMONSTRATED
#       rejected, not asserted: "skip until arbitration" also drains the queue,
#       and is rejected because it strands the audit row on-device. Decision 166
#       ratified unknown→false BECAUSE every such attempt is audit-flagged; a
#       guard that never lands the row falsifies the ratification's own premise.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the poison reproduced AND a guard cleared it with the audit row
#           landed.   exit 1  a leg disagreed.   exit 2  could not run.
#
# The extra DDL leg (c) needs is applied INSIDE this spike, on the throwaway
# substrate, and is printed as the migration the card owes. The committed
# migrations are never touched.
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase compose project only; never :5433, never :5434, no hosted
# project.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — Activity A's gate, not this spike's premise"

echo
echo "── enumerated: the column types F-2 collides with ──"
psqlq <<'SQL' | sed 's/^/#   /'
select 'scan_attempts.' || column_name || ' ' || data_type
       || case when is_nullable = 'NO' then ' NOT NULL' else ' NULL' end
  from information_schema.columns
 where table_schema='public' and table_name='scan_attempts' and column_name in ('id','code_id','token_hash')
 union all
select 'redeem(' || pg_get_function_arguments(p.oid) || ')'
  from pg_proc p join pg_namespace n on n.oid = p.pronamespace
 where n.nspname='public' and p.proname='redeem';
SQL

echo
echo "── device JWT (sub MUST equal device_id — the RLS with-check) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

echo
echo "── legs (a) poison, (b) blast radius, (c) guard vs the rejected alternative ──"
node "$SCRIPT_DIR/js/f2-push-poison.mjs" "$TOKEN" "$DB_CID" \
  || fail "a leg disagreed — see the node log above"

printf '\n✅ VERDICT: GREEN — the poison reproduces, its blast radius is measured, and a guard clears it with the audit row landed.\n'
