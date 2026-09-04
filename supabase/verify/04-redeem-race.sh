#!/usr/bin/env bash
# 04-redeem-race.sh — Card 2 gate (`redeem-rpc-race-proof`, run 20260904): the
# atomic arbiter is race-proof against the IN-REPO migration. Two concurrent
# clients firing at one code get EXACTLY ONE ok=true, every loser gets
# 'already_used', an expired code refuses both with 'expired', a pre-redeemed
# fixture answers 'already_used', an unknown uuid answers 'not_found' (GAP-1's
# taxonomy, §9/§19 — never a NULL reason), and a device-role JWT can drive the
# whole thing through PostgREST /rpc/redeem. Successor of the spike's
# 01-race-atomic.sh (throwaway; design input only) — this harness is
# self-contained under supabase/ and repeatable.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   default mode, all legs held:
#              R  the harness DETECTS the defect class: a separately-named naive
#                 check-then-update analog (TOCTOU window widened with
#                 pg_sleep) double-wins under two concurrent clients, then the
#                 analog is dropped. Without this leg the greens below would be
#                 unfalsifiable (greenfield red-first rule).
#              G  public.redeem: ROUNDS×2 concurrent clients (each client its
#                 own connection — one docker-exec psql per client), fresh
#                 per-run code per round: exactly one winner per round,
#                 0 double-wins, 0 zero-win rounds, every loser 'already_used'.
#              E  seeded expired fixture …0003 refuses BOTH concurrent clients
#                 with 'expired' and stays unredeemed.
#              A  seeded pre-redeemed fixture …0004 answers 'already_used'.
#              N  an unknown uuid answers (false, 'not_found') — GAP-1 closed:
#                 the reason is never NULL, so a forged code cannot read as a
#                 system outage downstream (§18 edge-case 3).
#              H  seeded race-target fixture …0002 (staged unredeemed first —
#                 explicit fixture action on the throwaway substrate): a single
#                 client redeems it, the row lands redeemed_by that device, and
#                 codes.updated_at ADVANCES (the replication checkpoint key —
#                 a redemption invisible to the pull tick would be a defect).
#              P  through PostgREST as role=authenticated: /rpc/redeem wins a
#                 fresh code (ok=true), answers 'already_used' on the second
#                 call, and an anonymous caller cannot execute it.
#   exit 1   a leg failed — the single-use premise does not hold on the built
#            migration (or the harness cannot detect the defect class).
#   exit 2   could not run.
#
#   --red-analog   THE RED-FIRST PROBE. Installs the naive check-then-update
#            body AS public.redeem (identical signature), then runs the SAME
#            assertion legs G/E/A/N/H. The probe does NOT invert its meaning —
#            red is red: against the naive body the race leg fails with
#            observed double-wins and the script exits 1, and THAT log is the
#            red-first evidence. (If the naive body somehow survived every
#            leg, the harness could not detect the defect class — that also
#            exits 1, with its own message; there is no path to exit 0 here.)
#            Whatever this mode leaves behind is overwritten by the migration's
#            drop-and-create on the next apply_all.
#
# Rounds default to 20 (the card's done_when: "20 rounds with 0 double-wins");
# override with RACE_ROUNDS for a longer soak.
#
# Re-runnable on a warm substrate: race codes are seeded with per-run uuids and
# deleted again on a green finish (so 01-structure.sh's exactly-5-codes seed
# assertion stays true afterwards); on a red they are left in place for
# post-mortem. Fixtures …0002/…0003/…0004 are the seed contract's Card 2 arms.
#
# USAGE
#   04-redeem-race.sh               # the full gate
#   04-redeem-race.sh --red-analog  # naive body as public.redeem → expect exit 1
#   RACE_ROUNDS=100 04-redeem-race.sh

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

MODE="full"
case "${1:-}" in
  --red-analog) MODE="red" ;;
  "") ;;
  *) echo "usage: $(basename "$0") [--red-analog]" >&2; exit 64 ;;
esac

ROUNDS="${RACE_ROUNDS:-20}"

# Seed fixture contract (supabase/seed.sql — by value, do not renumber):
CODE_RACE='c0000000-0000-4000-8000-000000000002'   # LOW active unredeemed — race/happy target
CODE_EXP='c0000000-0000-4000-8000-000000000003'    # LOW EXPIRED unredeemed — 'expired' arm
CODE_USED='c0000000-0000-4000-8000-000000000004'   # LOW active REDEEMED — 'already_used' arm
CAMPAIGN_LOW='a0000000-0000-4000-8000-000000000001'

substrate_up

echo
echo "── migration + seed (idempotent, server-side as supabase_admin) ──"
apply_all || cannot_run "the migrations would not apply (01-structure.sh owns that premise)"

RUNID="$(date +%s)-$$"
WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT
SEEDED_FILE="$WORK/seeded-codes"; : > "$SEEDED_FILE"

# one concurrent CLIENT = one docker-exec psql = one fresh connection
# (autocommit) — the spike-proven pattern the card names.
client() { docker exec "$DB_CID" psql -U supabase_admin -d postgres -qtA -c "$1"; }

# race FN CODE → prints the two clients' result lines ("true|<NULL>" / "false|reason")
race() {
  local out_a="$WORK/a.$$" out_b="$WORK/b.$$" pa pb
  client "select ok||'|'||coalesce(reason,'<NULL>') from public.$1('$2','device-$RUNID-a');" > "$out_a" &
  pa=$!
  client "select ok||'|'||coalesce(reason,'<NULL>') from public.$1('$2','device-$RUNID-b');" > "$out_b" &
  pb=$!
  wait "$pa" || true; wait "$pb" || true
  # `|| true`: an empty result set must surface as a zero-winner ASSERTION
  # failure downstream, not kill the script silently under set -e/pipefail.
  cat "$out_a" "$out_b" | grep . || true
}

seed_code() { # $1 uuid — a fresh, active, unredeemed per-run code in the LOW campaign
  psqlq <<SQL >/dev/null
insert into public.codes (id, token_hash, campaign_id, expires_at)
  values ('$1', 'race-$RUNID-$1', '$CAMPAIGN_LOW', now() + interval '1 day');
SQL
  echo "$1" >> "$SEEDED_FILE"
}

# The naive check-then-update body — the DEFECT CLASS this card exists to rule
# out. pg_sleep widens the TOCTOU window so two clients reliably overlap.
# Used twice: leg R installs it under its own name (and drops it again);
# --red-analog installs it AS public.redeem so the shipped legs demonstrably red.
naive_body() { # $1 = function name to (re)create with the naive body
  docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q <<SQL
drop function if exists public.$1(uuid, text);
create function public.$1(p_code uuid, p_device text)
returns table (ok boolean, reason text) language plpgsql as \$fn\$
declare v_free boolean;
begin
  select (c.redeemed_by is null and c.expires_at > now()) into v_free
    from public.codes c where c.id = p_code;
  perform pg_sleep(0.4);  -- the TOCTOU window, widened on purpose
  if coalesce(v_free, false) then
    update public.codes set redeemed_by = p_device, redeemed_at = now()
     where id = p_code;
    return query select true, null::text;
  else
    return query select false, 'unavailable'::text;
  end if;
end \$fn\$;
SQL
}

# ── the shipped assertion legs, all against public.redeem ────────────────────
race_legs() {
  echo
  echo "── leg G: public.redeem — $ROUNDS rounds × 2 concurrent clients, fresh code each ──"
  local i CODE RES WINS LOSER_REASON DOUBLE=0 ZERO=0
  for i in $(seq 1 "$ROUNDS"); do
    CODE="$(newuuid)"
    seed_code "$CODE" || cannot_run "seed failed (round $i)"
    RES="$(race redeem "$CODE")"
    WINS="$(echo "$RES" | grep -c '^true|' || true)"
    # `|| true`: on a DOUBLE-WIN there is no false| line at all — pipefail must
    # not kill the script before the double-win assertion below can report it
    # (this exact silent death was observed on the first red-analog run).
    LOSER_REASON="$(echo "$RES" | grep '^false|' | head -1 | cut -d'|' -f2 || true)"
    printf '  round %02d: winners=%s loser_reason=%s\n' "$i" "$WINS" "${LOSER_REASON:-<none>}"
    case "$WINS" in
      1) [ "$LOSER_REASON" = "already_used" ] || fail "round $i: the loser's reason was '${LOSER_REASON:-<empty>}', expected 'already_used'" ;;
      2) DOUBLE=1; break ;;
      0) ZERO=1; break ;;
    esac
  done
  [ "$DOUBLE" = 0 ] || fail "a round produced TWO winners — the single-use premise is falsified (this is the defect class the atomic UPDATE must prevent)"
  [ "$ZERO" = 0 ]  || fail "a round produced ZERO winners — the arbiter lost a redemption"
  echo "  ✓ $ROUNDS rounds, exactly one winner each, 0 double-wins, every loser 'already_used'"

  echo
  echo "── leg E: seeded expired fixture …0003 refuses BOTH clients with 'expired' ──"
  RES="$(race redeem "$CODE_EXP")"
  echo "  [$(echo "$RES" | tr '\n' ' ')]"
  [ "$(echo "$RES" | grep -c '^false|expired$')" = 2 ] || fail "expired fixture: expected both clients to get (false, expired), got: $RES"
  local STILL
  STILL="$(echo "select c.redeemed_by is null from public.codes c where c.id='$CODE_EXP';" | psqlq)"
  [ "$STILL" = t ] || fail "the expired fixture got redeemed_by set anyway"
  echo "  ✓ both refused, row stays unredeemed"

  echo
  echo "── leg A: seeded pre-redeemed fixture …0004 answers 'already_used' ──"
  local R_USED
  R_USED="$(client "select ok||'|'||coalesce(reason,'<NULL>') from public.redeem('$CODE_USED','device-$RUNID-a');")"
  echo "  redeem(…0004) → $R_USED"
  [ "$R_USED" = "false|already_used" ] || fail "pre-redeemed fixture: expected (false, already_used), got '$R_USED'"

  echo
  echo "── leg N: unknown uuid answers 'not_found' — GAP-1's taxonomy, never NULL ──"
  local GHOST R_GHOST
  GHOST="$(newuuid)"   # never seeded — no row anywhere
  R_GHOST="$(client "select ok||'|'||coalesce(reason,'<NULL>') from public.redeem('$GHOST','device-$RUNID-a');")"
  echo "  redeem($GHOST) → $R_GHOST"
  [ "$R_GHOST" = "false|not_found" ] || fail "unknown uuid: expected (false, not_found), got '$R_GHOST' — a NULL/other reason makes a forged code read as a system outage downstream (GAP-1, §18 edge-case 3)"

  echo
  echo "── leg H: seeded race-target fixture …0002 — happy path + updated_at advances ──"
  echo "  (fixture staging on the throwaway substrate: …0002 reset to unredeemed)"
  echo "update public.codes set redeemed_at=null, redeemed_by=null where id='$CODE_RACE';" | psqlq >/dev/null
  local TS_BEFORE R_HAPPY ROW
  TS_BEFORE="$(echo "select updated_at from public.codes where id='$CODE_RACE';" | psqlq)"
  R_HAPPY="$(client "select ok||'|'||coalesce(reason,'<NULL>') from public.redeem('$CODE_RACE','device-$RUNID-happy');")"
  echo "  redeem(…0002) → $R_HAPPY"
  [ "$R_HAPPY" = "true|<NULL>" ] || fail "happy path on …0002: expected (true, NULL), got '$R_HAPPY'"
  ROW="$(echo "select coalesce(redeemed_by,'<null>')||'|'||(redeemed_at is not null)||'|'||(updated_at > '$TS_BEFORE'::timestamptz) from public.codes where id='$CODE_RACE';" | psqlq)"
  echo "  row after: redeemed_by|redeemed_at set|updated_at advanced = $ROW"
  [ "$ROW" = "device-$RUNID-happy|true|true" ] || fail "…0002 after redeem: expected redeemed_by=device-$RUNID-happy, redeemed_at set, updated_at ADVANCED (the replication checkpoint key must move on redemption), got '$ROW'"
}

if [ "$MODE" = "red" ]; then
  echo
  echo "── RED-ANALOG MODE: installing the naive check-then-update body AS public.redeem ──"
  echo "   (identical signature; the shipped legs below must DETECT the defect and exit 1)"
  naive_body redeem || cannot_run "could not install the naive analog as public.redeem"
  race_legs
  # Reaching here means the naive TOCTOU body survived every assertion — the
  # harness cannot detect the defect class, so its greens prove nothing.
  fail "the naive check-then-update analog passed every leg — the harness cannot detect the defect class it exists to rule out"
fi

echo
echo "── leg R: the harness detects the defect class (naive analog double-wins) ──"
naive_body redeem_naive_analog_card2 || cannot_run "could not create the naive analog"
NAIVE_DOUBLE=0
for try in 1 2 3; do
  CODE="$(newuuid)"
  seed_code "$CODE" || cannot_run "seed failed (leg R)"
  RES="$(race redeem_naive_analog_card2 "$CODE")"
  WINS="$(echo "$RES" | grep -c '^true|' || true)"
  echo "  try $try: winners=$WINS  [$(echo "$RES" | tr '\n' ' ')]"
  if [ "$WINS" = 2 ]; then NAIVE_DOUBLE=1; break; fi
done
echo "drop function if exists public.redeem_naive_analog_card2(uuid, text);" | psqlq >/dev/null
[ "$NAIVE_DOUBLE" = 1 ] || fail "the naive TOCTOU analog never double-won in 3 tries — the harness cannot demonstrate the defect class, so the green legs would be unfalsifiable"
echo "  ✓ defect class demonstrated: both clients won on one code (analog dropped again)"

race_legs

echo
echo "── leg P: the device-facing surface — /rpc/redeem through PostgREST ──"
command -v go >/dev/null 2>&1 || cannot_run "go is not on PATH — mintjwt cannot mint the device token"
REST_PORT="$("${DC[@]}" port rest 3000 | awk -F: '{print $NF}')"
case "$REST_PORT" in ''|*[!0-9]*|0) cannot_run "could not resolve PostgREST host port" ;; esac
REST="http://127.0.0.1:$REST_PORT"
echo "#   rest            : $REST"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"
DEVICE="device-rpc-$RUNID"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub "$DEVICE" -role authenticated -ttl 30m)" \
  || cannot_run "minting the device token failed"
echo "  minted role=authenticated sub=$DEVICE"
req() { curl -s -m 10 -w '\n%{http_code}' "$@"; }

CODE="$(newuuid)"
seed_code "$CODE" || cannot_run "seed failed (leg P)"

# First call retries briefly on non-200: PostgREST reloads its schema cache on
# the migration's `notify pgrst` asynchronously, and a fresh function can 404
# for a beat. The retry is warmup only — the ASSERTION afterwards is absolute.
HTTP=""; BODY=""
for attempt in 1 2 3 4 5 6 7 8 9 10; do
  R="$(req -X POST "$REST/rpc/redeem" \
        -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
        -d "{\"p_code\":\"$CODE\",\"p_device\":\"$DEVICE\"}")"
  HTTP="$(echo "$R" | tail -1)"; BODY="$(echo "$R" | sed '$d')"
  [ "$HTTP" = 200 ] && break
  echo "  (attempt $attempt: HTTP $HTTP — schema cache warmup, retrying)"
  sleep 1
done
echo "  device POST /rpc/redeem (fresh code): HTTP $HTTP  $BODY"
[ "$HTTP" = 200 ] || fail "device /rpc/redeem expected 200, got $HTTP: $BODY"
echo "$BODY" | grep -q '"ok"[[:space:]]*:[[:space:]]*true' || fail "device /rpc/redeem did not win the fresh code: $BODY"

R="$(req -X POST "$REST/rpc/redeem" \
      -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
      -d "{\"p_code\":\"$CODE\",\"p_device\":\"$DEVICE\"}")"
HTTP="$(echo "$R" | tail -1)"; BODY="$(echo "$R" | sed '$d')"
echo "  device POST /rpc/redeem (same code again): HTTP $HTTP  $BODY"
[ "$HTTP" = 200 ] || fail "second /rpc/redeem expected 200, got $HTTP: $BODY"
echo "$BODY" | grep -q '"reason"[[:space:]]*:[[:space:]]*"already_used"' || fail "second /rpc/redeem expected reason already_used, got: $BODY"

R="$(req -X POST "$REST/rpc/redeem" -H "Content-Type: application/json" \
      -d "{\"p_code\":\"$CODE\",\"p_device\":\"anon-caller\"}")"
HTTP="$(echo "$R" | tail -1)"; BODY="$(echo "$R" | sed '$d')"
echo "  anonymous POST /rpc/redeem: HTTP $HTTP  $(echo "$BODY" | head -c 200)"
if [ "$HTTP" = 200 ] && echo "$BODY" | grep -q '"ok"'; then
  fail "an ANONYMOUS caller executed redeem() — execute must be revoked from anon/PUBLIC"
fi
echo "  ✓ device wins, device re-call answers already_used, anonymous cannot execute"

echo
echo "── cleanup: deleting this run's per-run race codes (fixture state restored) ──"
SEEDED_COUNT="$(grep -c . "$SEEDED_FILE" || true)"
docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -qtA \
  -c "delete from public.codes where token_hash like 'race-$RUNID-%';" >/dev/null \
  || fail "cleanup of per-run race codes failed"
REMAIN="$(echo "select count(*) from public.codes where token_hash like 'race-$RUNID-%';" | psqlq)"
echo "  seeded this run: $SEEDED_COUNT · remaining after cleanup: $REMAIN"
[ "$REMAIN" = 0 ] || fail "per-run race codes survived cleanup"

printf '\n✅ VERDICT: GREEN — public.redeem is race-proof against the in-repo migration (%s rounds, 0 double-wins), the defect class is detectable (leg R), the full reason taxonomy holds (already_used / expired / not_found — GAP-1 closed), updated_at advances on redemption, and the device RPC surface discriminates\n' "$ROUNDS"
exit 0
