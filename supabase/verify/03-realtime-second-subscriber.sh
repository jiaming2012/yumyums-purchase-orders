#!/usr/bin/env bash
# 03-realtime-second-subscriber.sh — Card 1 gate: a server-side UPDATE to a
# code arrives at a SECOND authenticated Realtime subscriber inside the
# observation window, with the publication membership (§7.1) and the
# RLS-per-subscriber rule (§7.2) both in play — driven against the IN-REPO
# migration + seed. Adapted from the spike's 03-realtime-second-subscriber.sh
# (read-only source).
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   the second subscriber (an authenticated-role websocket client,
#            rtprobe) JOINed public.codes with NO SYS-ERR, and the server-side
#            UPDATE arrived as a postgres_changes frame within the window.
#   exit 1   the subscription carried a SYS-ERR, or the event never arrived.
#   exit 2   could not run (substrate, build, or websocket transport).
#
# rtprobe's discipline is inherited: a join that replies "ok" is NOT a
# subscription — READY only counts bindings with no late system-frame error,
# and this script refuses to assert on a label that carried SYS-ERR.
#
# Non-destructive by design: the fired UPDATE is a pure `updated_at` checkpoint
# touch on seeded code …0001 — it redeems nothing, so the seed fixtures keep
# their contract state for Card 2 and re-runs.

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

WINDOW="${CARD1_RT_WINDOW:-20s}"

command -v go >/dev/null 2>&1 || cannot_run "go is not on PATH"

substrate_up

RT_PORT="$("${DC[@]}" port realtime 4000 | awk -F: '{print $NF}')"
case "$RT_PORT" in ''|*[!0-9]*|0) cannot_run "could not resolve the Realtime host port" ;; esac
echo "#   realtime        : 127.0.0.1:$RT_PORT"

echo
echo "── migration + seed (idempotent) ──"
apply_all || cannot_run "the migration would not apply (01-structure.sh owns that premise)"

CODE='c0000000-0000-4000-8000-000000000001'   # LOW campaign, active — the touch target
RUNID="$(date +%s)-$$"

echo
echo "── second subscriber: authenticated-role token + rtprobe on public.codes ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub "rt-sub-$RUNID" -role authenticated -ttl 30m)" \
  || cannot_run "minting the subscriber token failed"
echo "  minted role=authenticated sub=rt-sub-$RUNID (the role the codes_select_device policy admits)"

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
echo "── client one: server-side UPDATE on the code (pure updated_at checkpoint touch) ──"
psqlq <<SQL >/dev/null || cannot_run "the server-side UPDATE failed"
update public.codes set updated_at = now() where id = '$CODE';
SQL
echo "  updated code=$CODE → updated_at=now() (nothing redeemed; seed contract preserved)"

echo
echo "── waiting out the observation window ($WINDOW) ──"
set +e; wait "$PROBE_PID"; PROBE_RC=$?; set -e
echo "  rtprobe exit=$PROBE_RC"
grep -E '^RTP (EVENT|COUNT|DONE)' "$WORK/rtprobe.log" | sed 's/^/  /'
[ "$PROBE_RC" = 0 ] || cannot_run "rtprobe exited $PROBE_RC (3 = could not run); its log is above"

grep -q "^RTP EVENT label=codes type=UPDATE table=public.codes" "$WORK/rtprobe.log" \
  || fail "the UPDATE never arrived at the second subscriber within $WINDOW"

printf '\n✅ VERDICT: GREEN — a server-side UPDATE arrived at a second authenticated Realtime subscriber (publication + per-subscriber RLS both live, in-repo migration)\n'
exit 0
