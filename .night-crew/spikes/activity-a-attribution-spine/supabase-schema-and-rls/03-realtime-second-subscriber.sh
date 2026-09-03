#!/usr/bin/env bash
# 03-realtime-second-subscriber.sh — spike: the card's done_when leg — "a row
# changed on one client appears on a second subscriber" — through the real
# Realtime websocket, with the publication membership §7.1 warns about and the
# RLS-per-subscriber rule §7.2 imposes both in play.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   the second subscriber (an authenticated-role websocket client,
#            rtprobe) JOINed public.codes with NO SYS-ERR, and the redemption
#            UPDATE fired by "client one" arrived as a postgres_changes frame
#            within the observation window.
#   exit 1   the subscription carried a SYS-ERR, or the event never arrived.
#   exit 2   could not run (substrate, build, or websocket transport).
#
# rtprobe's own discipline is inherited: a join that replies "ok" is NOT a
# subscription — the READY line only counts bindings with no late system-frame
# error, and this script refuses to assert on a label that carried SYS-ERR.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
QA="$REPO_ROOT/.night-crew/qa/spike-supabase"
WINDOW="${SPIKE_RT_WINDOW:-20s}"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH
command -v go >/dev/null 2>&1 || cannot_run "go is not on PATH"

echo "# target coordinates (read-only statement before any write):"
echo "#   compose project : spike-supabase (throwaway). NOT :5433, NOT :5434, no hosted project."

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
RT_PORT="$("${DC[@]}" port realtime 4000 | awk -F: '{print $NF}')"
case "$RT_PORT" in ''|*[!0-9]*|0) cannot_run "could not resolve the Realtime host port" ;; esac
echo "#   db container=$DB_CID realtime=127.0.0.1:$RT_PORT"

psqlq() { docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -qtA; }

echo
echo "── schema + seed ──"
docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q -f - \
  < "$SCRIPT_DIR/sql/qr-schema.sql" || cannot_run "qr-schema.sql would not apply (spike 01 owns that premise)"
CAMPAIGN='11111111-1111-1111-1111-111111111111'
CODE='55555555-5555-5555-5555-555555555555'
psqlq <<SQL || cannot_run "seeding failed"
insert into public.campaigns (id, name, face_value) values ('$CAMPAIGN', 'spike-rt', 2.00);
insert into public.codes (id, token_hash, campaign_id, expires_at)
  values ('$CODE', 'spike-rt-hash-1', '$CAMPAIGN', now() + interval '1 day');
SQL
echo "  seeded code=$CODE (unredeemed)"

echo
echo "── second subscriber: authenticated-role token + rtprobe on public.codes ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-b -role authenticated -ttl 30m)" \
  || cannot_run "minting the subscriber token failed"
echo "  minted role=authenticated sub=device-b (the role RLS must let read codes, §7.2)"

WORK="$(mktemp -d)"
trap 'kill "${PROBE_PID:-}" 2>/dev/null || true; rm -rf "$WORK"' EXIT

# Build, then run the BINARY — `go run &` hands the shell a supervisor pid.
(cd "$QA" && go build -o "$WORK/rtprobe" ./rtprobe) || cannot_run "rtprobe would not build"

"$WORK/rtprobe" \
  -addr "127.0.0.1:$RT_PORT" -host realtime-dev.localhost -token "$TOKEN" \
  -window "$WINDOW" \
  -bind 'codes|public|codes|' \
  > "$WORK/rtprobe.log" 2>&1 &
PROBE_PID=$!

for _ in $(seq 1 30); do
  grep -q '^RTP READY' "$WORK/rtprobe.log" && break
  kill -0 "$PROBE_PID" 2>/dev/null || break
  sleep 1
done
grep -q '^RTP READY' "$WORK/rtprobe.log" || {
  sed 's/^/  | /' "$WORK/rtprobe.log"
  cannot_run "rtprobe never reached READY — websocket did not connect or the join never resolved"
}
grep -E '^RTP (CONNECT|BIND|JOIN-OK|JOIN-ERR|SYS-ERR|READY)' "$WORK/rtprobe.log" | sed 's/^/  /'
grep -q '^RTP JOIN-OK label=codes' "$WORK/rtprobe.log" || fail "the codes binding never JOINed"
if grep -q '^RTP SYS-ERR label=codes' "$WORK/rtprobe.log"; then
  fail "the codes subscription carried a SYS-ERR — joined but NOT subscribed (the §7.1/§7.2 failure shape)"
fi

echo
echo "── client one redeems: UPDATE the code server-side (the §5.3 propagation event) ──"
psqlq <<SQL || cannot_run "the redemption UPDATE failed"
update public.codes
   set redeemed_by = 'device-a', redeemed_at = now(), updated_at = now()
 where id = '$CODE';
SQL
echo "  updated code=$CODE → redeemed_by=device-a"

echo
echo "── waiting out the observation window ($WINDOW) ──"
set +e; wait "$PROBE_PID"; PROBE_RC=$?; set -e
echo "  rtprobe exit=$PROBE_RC"
grep -E '^RTP (EVENT|COUNT|DONE)' "$WORK/rtprobe.log" | sed 's/^/  /'
[ "$PROBE_RC" = 0 ] || cannot_run "rtprobe exited $PROBE_RC (3 = could not run); its log is above"

grep -q "^RTP EVENT label=codes type=UPDATE table=public.codes" "$WORK/rtprobe.log" \
  || fail "the redemption UPDATE never arrived at the second subscriber within $WINDOW"

printf '\n✅ VERDICT: GREEN — a row changed by one client arrived at a second Realtime subscriber (publication + RLS both live)\n'
exit 0
