#!/usr/bin/env bash
# 01-hybrid-qr-encodes-and-the-shipped-scanner-reads-it.sh — spike for card E1
# (identity-code-and-qr, D-KR3, handoff §10 / #10): which SERVER-generated hybrid QR encodings the
# SHIPPED scanner page reads. First run (2026-10-05T13:12Z, exit 1): the 234-char payload at
# Medium/512 (version 11) answered `decodeError` — html5-qrcode draws a picked file onto the hidden
# 320 px #scan-file-surface, so module DENSITY, not pixel size, is the bound. This run generates
# SEVERAL variants with the repo's own go-qrcode and measures each through the shipped page:
#   full-M-512    the 234-char payload (full JSON descriptor), Medium, 512 px   — the first run's red
#   full-M-1024   the same at 1024 px                                           — is pixel size the bound?
#   full-L-512    the same payload at Low correction                             — one version smaller
#   full-L-nb     Low, no quiet zone in the PNG (the surface supplies white)     — two fewer modules a side
#   compact-M-512 a compact descriptor that KEEPS campaign_id (base64url of short-key JSON), Medium
#   compact-L-512 the compact descriptor at Low
# Offline, each must decode to the token's hash (compact variants decode to `unknownCode` because
# the shipped reader cannot parse them — still a decode; the full ones to `embeddedOffer`). Online,
# the first decodable variant must resolve the server's list. The worktree is throwaway.
# Coordinates: TEST_DB_NAME=hq_test_spike_e1_20261006, TEST_PORT=8331, :5434. NEVER :5433.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the CARD's encoding (card-L-512: 16-char token, date-only expires_at, Low) decodes offline
#           to embeddedOffer AND a hash-matching variant resolves offerReady online — the card can ship
#           a server-made hybrid QR the shipped reader accepts; the variant table says the margins
#   exit 1  no full-descriptor variant decodes, or none resolves online — "🛑 VERDICT: RED" (the card
#           must then change the reader's surface or the descriptor contract, and the slate says so)
#   exit 2  could not run (precondition missing; nothing was measured)
set -euo pipefail
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/_lib.sh"
preflight
worktree_up

TOKEN="spike-e1-identity-token"
FULL_JSON='{"label":"Free side of wings","campaign_id":"a0000000-0000-4000-8000-000000000001","expires_at":"2028-01-01T00:00:00Z","face_value":2}'
# Compact candidate (the card may lock this and replace parseEmbeddedOffer): short keys, the campaign
# uuid as 22-char base64url of its 16 bytes, a date, cents. The shipped reader will NOT parse it.
COMPACT_JSON="{\"l\":\"Free side of wings\",\"c\":\"$(node -e 'process.stdout.write(Buffer.from("a0000000000040008000000000000001","hex").toString("base64url"))')\",\"e\":\"2028-01-01\",\"v\":200}"
b64url() { printf %s "$1" | base64 -w0 | tr '+/' '-_' | tr -d '='; }
FULL_PAYLOAD="https://hq.yumyums.kitchen/r/${TOKEN}#o=$(b64url "$FULL_JSON")"
# The card's candidate: a 16-char identity token (80 bits of base32 — the scanner hashes it, the
# hash is the key) and the SAME descriptor keys the shipped reader parses, with a date-only
# expires_at. Expected: version 9 at Low (≤ 230 bytes).
CARD_TOKEN="ic7k3m9q2x5p8w4z"
CARD_JSON='{"label":"Free side of wings","campaign_id":"a0000000-0000-4000-8000-000000000001","expires_at":"2028-01-01","face_value":2}'
CARD_PAYLOAD="https://hq.yumyums.kitchen/r/${CARD_TOKEN}#o=$(b64url "$CARD_JSON")"
COMPACT_PAYLOAD="https://hq.yumyums.kitchen/r/${TOKEN}#o=$(b64url "$COMPACT_JSON")"
FIX="$WT/tests/fixtures"

leg "generate — go-qrcode inside the worktree's backend module, six variants"
mkdir -p "$WT/backend/cmd/spike-qrgen"
cp "$SCRIPT_DIR/qrgen.go" "$WT/backend/cmd/spike-qrgen/main.go"
echo "#   full payload    ($(printf %s "$FULL_PAYLOAD" | wc -c) chars): ${FULL_PAYLOAD:0:70}…"
echo "#   compact payload ($(printf %s "$COMPACT_PAYLOAD" | wc -c) chars): ${COMPACT_PAYLOAD:0:70}…"
echo "#   card payload    ($(printf %s "$CARD_PAYLOAD" | wc -c) chars): $CARD_PAYLOAD"
VARIANTS="["
gen() { # name token payload size level border full(true|false) desc
  local name="$1" token="$2" payload="$3" size="$4" level="$5" border="$6" full="$7" desc="$8"
  local line
  line="$(cd "$WT/backend" && go run ./cmd/spike-qrgen "$payload" "$FIX/spike-e1-$name.png" "$size" "$level" "$border")" || fail "$name did not encode"
  echo "#   $name: $line"
  local ver mod
  ver="$(sed -n 's/^version=\([0-9]*\).*/\1/p' <<<"$line")"; mod="$(sed -n 's/.*modules=\([0-9]*\).*/\1/p' <<<"$line")"
  VARIANTS="$VARIANTS{\"name\":\"$name\",\"token\":\"$token\",\"fullDescriptor\":$full,\"desc\":\"$desc\",\"enc\":{\"version\":$ver,\"modules\":$mod,\"size\":$size,\"level\":\"$level\",\"border\":$border,\"payload_len\":$(printf %s "$payload" | wc -c)}},"
}
gen card-L-512    "$CARD_TOKEN" "$CARD_PAYLOAD"    512  L 1 true  "THE CARD'S ENCODING: 16-char token, full JSON with date-only expires_at, Low, 512 px"
gen full-M-512    "$TOKEN"      "$FULL_PAYLOAD"    512  M 1 true  "full JSON descriptor, Medium, 512 px (the first run's red)"
gen full-M-1024   "$TOKEN"      "$FULL_PAYLOAD"    1024 M 1 true  "full JSON descriptor, Medium, 1024 px"
gen full-L-512    "$TOKEN"      "$FULL_PAYLOAD"    512  L 1 true  "full JSON descriptor, Low, 512 px"
gen full-L-nb     "$TOKEN"      "$FULL_PAYLOAD"    512  L 0 true  "full JSON descriptor, Low, 512 px, no quiet zone"
gen compact-M-512 "$TOKEN"      "$COMPACT_PAYLOAD" 512  M 1 false "compact descriptor (keeps campaign_id), Medium, 512 px"
gen compact-L-512 "$TOKEN"      "$COMPACT_PAYLOAD" 512  L 1 false "compact descriptor (keeps campaign_id), Low, 512 px"
VARIANTS="${VARIANTS%,}]"
printf %s "$VARIANTS" >"$FIX/spike-e1-variants.json"
node -e 'JSON.parse(require("fs").readFileSync(process.argv[1],"utf8"))' "$FIX/spike-e1-variants.json" || cannot_run "variants.json malformed"

leg "spec — copy spike-e1.spec.js into the worktree's tests/"
cp "$SCRIPT_DIR/spike-e1.spec.js" "$WT/tests/spike-e1.spec.js"
echo "#   $WT/tests/spike-e1.spec.js (7 offline variant specs + 1 online)"

leg "run — through the shipped page"
RC="$(pw_run e1 tests/spike-e1.spec.js)"
OUT="$(pw_outcomes e1 2>/dev/null || true)"
[ -s "$LOG/e1.json" ] || cannot_run "no Playwright JSON — the stack did not come up; see $LOG/e1.log"
echo "#   variant table (offline, shipped reader):"
grep -o 'SPIKE-E1-VARIANT {.*}' "$LOG/e1.log" | sed 's/SPIKE-E1-VARIANT //' | sed 's/^/#     /'
grep -o 'SPIKE-E1-ONLINE {.*}' "$LOG/e1.log" | sed 's/^/#   /'
exitline "playwright tests/spike-e1.spec.js (8 specs)" "$RC" "any"
N_FULL_OK="$(grep -o 'SPIKE-E1-VARIANT {.*}' "$LOG/e1.log" | grep '"name":"full-' | grep -c '"decoded":true' || true)"
N_ANY_OK="$(grep -o 'SPIKE-E1-VARIANT {.*}' "$LOG/e1.log" | grep -c '"decoded":true' || true)"
CARD_OK="$(grep -o 'SPIKE-E1-VARIANT {.*}' "$LOG/e1.log" | grep '"name":"card-L-512"' | grep -c '"decoded":true' || true)"
echo "#   card encoding decoded: $CARD_OK / 1; full-descriptor variants decoded: $N_FULL_OK / 5; any variant decoded: $N_ANY_OK / 7"
has "$OUT" "[E1-SPIKE-2]=true" || fail "ONLINE leg red — no variant that decoded to its token's hash resolved offerReady from the server lookup (see $LOG/e1.log)"
[ "$CARD_OK" = 1 ] || fail "the card's encoding (card-L-512) did not decode offline to embeddedOffer through the shipped reader (see the variant table)"

main_tree_untouched
echo
echo "✅ GREEN — the card's encoding decodes offline to embeddedOffer and resolves offerReady online through the shipped scanner on dev@$(git -C "$REPO_ROOT" rev-parse --short dev);"
echo "   $N_FULL_OK of 5 full-descriptor and $N_ANY_OK of 7 variants overall decoded. The variant table above is the build fact the card encodes to."
