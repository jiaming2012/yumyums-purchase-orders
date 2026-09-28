#!/usr/bin/env bash
# 01-race-atomic.sh — spike: the cycle's core correctness premise (§6, E-KR1).
# A single conditional UPDATE is a real mutex: two concurrent clients firing at
# one code get EXACTLY ONE ok=true — and the test proves it can detect the
# defect class by first showing a naive check-then-update DOUBLE-WINNING.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   all five legs held:
#            R  red analog — naive_redeem double-wins under two concurrent
#               clients (the harness DETECTS the defect; greenfield red-first)
#            G  redeem_verbatim: ROUNDS×2 concurrent clients, exactly one
#               winner per round, 0 double-wins; every loser 'already_used'
#            E  expired code → (f,'expired') and the row stays unredeemed
#            N  unknown uuid: verbatim's reason IS NULL (the pinned gap —
#               §9/§19 name 'not_found', verbatim cannot produce it)
#            V  redeem_v2 returns 'not_found' for the unknown uuid and races
#               identically (1 winner) on a fresh code
#   exit 1   a leg failed.
#   exit 2   could not run.
#
# Rounds default to 20 (the card's done_when: "passes 20× with 0 double-wins").

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
QA="$REPO_ROOT/.night-crew/qa/spike-supabase"
SCHEMA_SQL="$REPO_ROOT/.night-crew/spikes/activity-a-attribution-spine/supabase-schema-and-rls/sql/qr-schema.sql"
ROUNDS="${SPIKE_RACE_ROUNDS:-20}"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

echo "# target coordinates (read-only statement before any write):"
echo "#   compose project : spike-supabase (throwaway). NOT :5433, NOT :5434, no hosted project."
[ -f "$SCHEMA_SQL" ] || cannot_run "goal-1 schema fixture missing: $SCHEMA_SQL (this goal rides goal 1's real fixture, not a stub)"

echo
echo "── substrate up ──"
"$QA/env-up.sh" || cannot_run "env-up.sh did not reach GREEN"

ANCHOR=""
if command -v git >/dev/null 2>&1; then
  _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
  [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
fi
[ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
DC=(docker compose -p spike-supabase --project-directory "$ANCHOR" -f "$REPO_ROOT/docker-compose.supabase.yml")
DB_CID="$("${DC[@]}" ps -q db)"; [ -n "$DB_CID" ] || cannot_run "no db container"
echo "#   db container=$DB_CID"

psqlq() { docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -qtA; }
# one concurrent CLIENT = one docker-exec psql = one fresh connection (autocommit)
client() { docker exec "$DB_CID" psql -U supabase_admin -d postgres -qtA -c "$1"; }

echo
echo "── schema + redeem functions ──"
docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q -f - < "$SCHEMA_SQL" \
  || cannot_run "qr-schema.sql would not apply (goal 1's spike 01 owns that premise)"
docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q -f - < "$SCRIPT_DIR/sql/redeem-fns.sql" \
  || fail "redeem-fns.sql would not apply"

CAMPAIGN='11111111-1111-1111-1111-111111111111'
psqlq <<SQL || cannot_run "campaign seed failed"
insert into public.campaigns (id, name, face_value) values ('$CAMPAIGN', 'spike-race', 2.00);
SQL

seed_code() { # $1 uuid  $2 interval-expr for expires_at
  psqlq <<SQL
insert into public.codes (id, token_hash, campaign_id, expires_at)
  values ('$1', 'hash-$1', '$CAMPAIGN', now() + $2);
SQL
}
uuidn() { printf '%08d-0000-4000-8000-%012d' "$1" "$1"; }

race() { # $1 fn  $2 code-uuid → prints two result lines "t|" / "f|reason"
  local out_a="$WORK/a.$$" out_b="$WORK/b.$$"
  client "select ok||'|'||coalesce(reason,'<NULL>') from public.$1('$2','device-a');" > "$out_a" &
  local pa=$!
  client "select ok||'|'||coalesce(reason,'<NULL>') from public.$1('$2','device-b');" > "$out_b" &
  local pb=$!
  wait "$pa" || true; wait "$pb" || true
  cat "$out_a" "$out_b" | grep .
}
WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT

echo
echo "── leg R: the red analog — naive check-then-update DOUBLE-WINS ──"
NAIVE_DOUBLE=0
for try in 1 2 3; do
  CODE="$(uuidn $((9000 + try)))"
  seed_code "$CODE" "interval '1 day'" || cannot_run "seed failed (leg R)"
  RES="$(race naive_redeem "$CODE")"
  WINS="$(echo "$RES" | grep -c '^true|' || true)"
  echo "  try $try: winners=$WINS  [$(echo "$RES" | tr '\n' ' ')]"
  if [ "$WINS" = 2 ]; then NAIVE_DOUBLE=1; break; fi
done
[ "$NAIVE_DOUBLE" = 1 ] || fail "the naive TOCTOU function never double-won in 3 tries — the harness cannot demonstrate the defect class, so the green legs below would be unfalsifiable"
echo "  ✓ defect class demonstrated: both clients won on one code (this is what the atomic UPDATE must prevent)"

echo
echo "── leg G: redeem_verbatim — $ROUNDS rounds × 2 concurrent clients ──"
DOUBLE=0; ZERO=0
for i in $(seq 1 "$ROUNDS"); do
  CODE="$(uuidn "$i")"
  seed_code "$CODE" "interval '1 day'" || cannot_run "seed failed (round $i)"
  RES="$(race redeem_verbatim "$CODE")"
  WINS="$(echo "$RES" | grep -c '^true|' || true)"
  LOSER_REASON="$(echo "$RES" | grep '^false|' | head -1 | cut -d'|' -f2)"
  printf '  round %02d: winners=%s loser_reason=%s\n' "$i" "$WINS" "${LOSER_REASON:-<none>}"
  case "$WINS" in
    1) [ "$LOSER_REASON" = "already_used" ] || fail "round $i: the loser's reason was '${LOSER_REASON:-<empty>}', expected 'already_used'" ;;
    2) DOUBLE=1 ;;
    0) ZERO=1 ;;
  esac
done
[ "$DOUBLE" = 0 ] || fail "a round produced TWO winners — the atomic premise is falsified"
[ "$ZERO" = 0 ]  || fail "a round produced ZERO winners — the arbiter lost a redemption"
echo "  ✓ $ROUNDS rounds, exactly one winner each, every loser 'already_used'"

echo
echo "── leg E: an expired code refuses both clients with 'expired' ──"
CODE="$(uuidn 9998)"
seed_code "$CODE" "- interval '1 hour'" || cannot_run "seed failed (leg E)"
RES="$(race redeem_verbatim "$CODE")"
echo "  [$(echo "$RES" | tr '\n' ' ')]"
[ "$(echo "$RES" | grep -c '^false|expired$')" = 2 ] || fail "expired code: expected both clients to get (f,expired), got: $RES"
STILL="$(echo "select redeemed_by is null from public.codes where id='$CODE';" | psqlq)"
[ "$STILL" = t ] || fail "an expired code got redeemed_by set anyway"

echo
echo "── leg N: unknown uuid — pin the verbatim §6 gap (reason is NULL, not 'not_found') ──"
GHOST='99999999-9999-4999-8999-999999999999'
R_VERB="$(client "select ok||'|'||coalesce(reason,'<NULL>') from public.redeem_verbatim('$GHOST','device-a');")"
echo "  redeem_verbatim(unknown) → $R_VERB"
[ "$R_VERB" = "false|<NULL>" ] || fail "expected the verbatim draft to return (f, NULL) for an unknown code — it returned '$R_VERB'; re-characterize the gap before building on it"

echo "── leg V: redeem_v2 closes the gap and races identically ──"
R_V2="$(client "select ok||'|'||coalesce(reason,'<NULL>') from public.redeem_v2('$GHOST','device-a');")"
echo "  redeem_v2(unknown) → $R_V2"
[ "$R_V2" = "false|not_found" ] || fail "redeem_v2 expected (f, not_found), got '$R_V2'"
CODE="$(uuidn 9999)"
seed_code "$CODE" "interval '1 day'" || cannot_run "seed failed (leg V)"
RES="$(race redeem_v2 "$CODE")"
WINS="$(echo "$RES" | grep -c '^true|' || true)"
echo "  v2 race: winners=$WINS  [$(echo "$RES" | tr '\n' ' ')]"
[ "$WINS" = 1 ] || fail "redeem_v2 race produced $WINS winners, expected exactly 1"

printf '\n✅ VERDICT: GREEN — atomic redeem() is race-proof (%s rounds, 0 double-wins), the defect class is detectable, and the not_found gap is pinned + closed by the v2 draft\n' "$ROUNDS"
exit 0
