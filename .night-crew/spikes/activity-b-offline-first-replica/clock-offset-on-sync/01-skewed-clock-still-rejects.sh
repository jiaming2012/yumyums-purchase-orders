#!/usr/bin/env bash
# 01-skewed-clock-still-rejects.sh — spike: §5.1's clock-offset design closes
# the real hole. (a) the PostgREST pull response's Date header is a usable
# serverNow source; (b) RED ANALOG — the naive local expiry check ACCEPTS a
# dead code under a 2-days-slow device clock (the defect class §5.1 names);
# (c) the offset-adjusted check REJECTS the same code under the same skew.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   (a)+(b)+(c) all held.
#   exit 1   a leg failed — the premise is not proven. If it is leg (a), the
#            card needs a now() RPC instead of the Date header; that finding
#            re-shapes the card, record it.
#   exit 2   could not run (substrate, tooling).
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase project only; never :5433, never :5434, no hosted project.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"   # substrate_up, psqlq, apply_all, reset_bare, fail, cannot_run, newuuid

substrate_up
echo
echo "── built schema: reset_bare + apply_all (the committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — that is Card 1's gate, not this spike's premise"

RESTP="$("${DC[@]}" port rest 3000 | awk -F: '{print $NF}')"
[ -n "$RESTP" ] || cannot_run "could not resolve PostgREST host port"

echo
echo "── seed the dangerous row: expired ~1 day ago, INSIDE the 2-day replica window ──"
CODE="$(newuuid)"
psqlq <<SQL >/dev/null || cannot_run "seeding the per-run code failed"
insert into public.codes (id, token_hash, campaign_id, expires_at)
values ('$CODE', 'spike-clock-$CODE',
        'a0000000-0000-4000-8000-000000000001',
        now() - interval '1 day');
SQL
echo "  code=$CODE expires_at=now()-1d"

cleanup() { psqlq <<<"delete from public.codes where id = '$CODE';" >/dev/null 2>&1 || true; }
trap cleanup EXIT

echo
echo "── device JWT (secret read from the compose file, never re-typed) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" \
  || cannot_run "minting the device token failed"

echo
echo "── the three legs (node — pure local arithmetic on real wire data) ──"
node "$SCRIPT_DIR/js/clock-offset.mjs" "http://127.0.0.1:$RESTP" "$TOKEN" "$CODE" \
  || fail "a leg red — see the node log above"

printf '\n✅ VERDICT: GREEN — Date header is a usable serverNow source; naive check provably accepts the dead code under skew; offset-adjusted check rejects it.\n'
