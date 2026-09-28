#!/usr/bin/env bash
# 01-decode-and-hash-chain.sh — spike: the §12 scan chain headless. A QR PNG in
# the #10 hybrid shape (URL-wrapped identity token, token = the committed seed
# fixture's label) decodes in REAL Chromium via the vendored single-file
# html5-qrcode (the no-build premise), the extracted token's in-page WebCrypto
# SHA-256 equals the committed supabase/seed.sql literal (the replica-key
# contract), cross-checked against Node's createHash.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  decode + extraction + both hashes align with the committed contract.
#   exit 1  a leg failed — decode broken in the no-build context, or the hash
#           scheme disagrees (every replica lookup would silently miss).
#   exit 2  could not run (npm/network, browser missing).
#
# Camera capture (getUserMedia) is deliberately NOT here — environmental, needs
# a physical camera; the card's attended verification covers it (ledger).
# Touches nothing outside this directory. No substrate, no DB.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

command -v node >/dev/null 2>&1 || cannot_run "node not on PATH"

# The committed contract of record (supabase/seed.sql): token → sha256 hex.
TOKEN='card1-test-code-fixture-1'
SEED_LITERAL='c5a1641409efd198e5a55417f209eda33500fd199f1fa7fa0d8a2567ee1f9680'
PAYLOAD="https://hq.yumyums.kitchen/r/$TOKEN"

echo "── spike-local deps (qrcode + html5-qrcode; nothing at repo root) ──"
(cd "$SCRIPT_DIR" && npm install --no-audit --no-fund --silent) || cannot_run "npm install failed — network?"

echo
echo "── generate the QR + Node-side hash cross-check ──"
GEN="$(cd "$SCRIPT_DIR" && node gen-qr.mjs "$TOKEN" "$SCRIPT_DIR/web/qr.png")" || cannot_run "QR generation failed"
echo "  $GEN"
NODE_HASH="$(echo "$GEN" | node -e 'process.stdin.once("data",(d)=>console.log(JSON.parse(d).nodeHash))')"
[ "$NODE_HASH" = "$SEED_LITERAL" ] \
  || fail "Node sha256($TOKEN) = $NODE_HASH does not equal the committed seed literal — the seed contract itself is off"
echo "  node createHash == committed seed literal ✓"

echo
echo "── vendor the single-file candidate (the no-build premise) ──"
SRC="$SCRIPT_DIR/node_modules/html5-qrcode/html5-qrcode.min.js"
[ -f "$SRC" ] || fail "html5-qrcode ships no single-file html5-qrcode.min.js — the no-build premise fails for this candidate"
cp "$SRC" "$SCRIPT_DIR/web/html5-qrcode.min.js"
BYTES="$(wc -c < "$SCRIPT_DIR/web/html5-qrcode.min.js" | tr -d ' ')"
echo "  vendored html5-qrcode.min.js: $BYTES bytes (the no-build weight the card inherits)"

echo
echo "── Chromium: decode → extract → WebCrypto digest (repo-root Playwright) ──"
cd "$REPO_ROOT"
SPIKE_EXPECT_PAYLOAD="$PAYLOAD" SPIKE_EXPECT_TOKEN="$TOKEN" SPIKE_EXPECT_HASH="$SEED_LITERAL" \
  npx --no-install playwright test -c "$SCRIPT_DIR/playwright.config.js" \
  || fail "the browser chain red — see the Playwright log above"

printf '\n✅ VERDICT: GREEN — html5-qrcode decodes as ONE vendored file in real Chromium and the on-device WebCrypto hash IS the committed replica-key contract.\n'
