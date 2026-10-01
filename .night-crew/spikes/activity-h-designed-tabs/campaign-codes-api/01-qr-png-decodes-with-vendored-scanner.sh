#!/usr/bin/env bash
# 01-qr-png-decodes-with-vendored-scanner.sh — spike: the campaign QR the card
# will mint (skip2/go-qrcode PNG of the short URL) is (a) small — version ≤ 3,
# i.e. ≤ 29 modules a side, so it scans off a truck sign — and (b) decodable by
# the exact scanner library the tablets already ship (lib/html5-qrcode.min.js,
# scanFile), decoding to the identical URL string.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  both legs held.   exit 1  a leg failed.   exit 2  could not run.
#
# Isolation: a throwaway Go module + a Playwright page under the session
# scratchpad / TMPDIR. Nothing in the repo is written. Network: `go get` of the
# qrcode module (first run only).
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
export PATH="/usr/local/go/bin:$PATH"
command -v go >/dev/null || cannot_run "go not on PATH"
command -v node >/dev/null || cannot_run "node not on PATH"
[ -f "$REPO_ROOT/lib/html5-qrcode.min.js" ] || cannot_run "vendored lib/html5-qrcode.min.js missing"
[ -d "$REPO_ROOT/node_modules/playwright" ] || cannot_run "playwright not installed at repo root"
URL="https://hq.yumyums.kitchen/q/7KQ2M3"
W="${TMPDIR:-/tmp}/spike-h1-qr-$$"; mkdir -p "$W"; trap 'rm -rf "$W"' EXIT
cat > "$W/main.go" <<'GO'
package main
import ("fmt";"os";qrcode "github.com/skip2/go-qrcode")
func main(){
  q, err := qrcode.New(os.Args[1], qrcode.Medium); if err != nil { fmt.Println("ERR", err); os.Exit(3) }
  bm := q.Bitmap(); modules := len(bm) - 2*4 // Bitmap includes the 4-module quiet zone each side
  if err := q.WriteFile(1024, os.Args[2]); err != nil { fmt.Println("ERR", err); os.Exit(3) }
  fmt.Printf("modules=%d\n", modules)
}
GO
( cd "$W" && go mod init spikeqr >/dev/null 2>&1 && GOFLAGS=-mod=mod go get github.com/skip2/go-qrcode@v0.0.0-20200617195104-da1b6568686e >/dev/null 2>&1 ) || cannot_run "go get github.com/skip2/go-qrcode failed (network?)"
OUT="$(cd "$W" && GOFLAGS=-mod=mod go run . "$URL" "$W/code.png")" || cannot_run "go run failed: $OUT"
echo "# generated: $OUT (URL length $(printf %s "$URL" | wc -c | tr -d ' '))"
MODULES="$(printf %s "$OUT" | sed -n 's/^modules=//p')"
[ -n "$MODULES" ] || fail "could not read module count"
[ "$MODULES" -le 29 ] || fail "QR is $MODULES modules a side (> 29 = version > 3) — too dense for a sign"
cat > "$W/decode.cjs" <<JS
const { chromium } = require(process.argv[2] + '/node_modules/playwright');
const fs = require('fs');
(async () => {
  const br = await chromium.launch();
  const pg = await br.newPage();
  await pg.setContent('<div id="s" style="width:320px;height:320px"></div><input type="file" id="f">');
  await pg.addScriptTag({ path: process.argv[2] + '/lib/html5-qrcode.min.js' });
  await pg.setInputFiles('#f', process.argv[3]);
  const text = await pg.evaluate(async () => {
    const f = document.getElementById('f').files[0];
    const h = new Html5Qrcode('s');
    return await h.scanFile(f, false);
  });
  console.log('decoded=' + text);
  await br.close();
})().catch(e => { console.log('ERR ' + e.message); process.exit(3); });
JS
DEC="$(node "$W/decode.cjs" "$REPO_ROOT" "$W/code.png")" || cannot_run "playwright decode failed: $DEC"
echo "# $DEC"
[ "$DEC" = "decoded=$URL" ] || fail "scanner decoded '$DEC', expected '$URL'"
echo "✅ GREEN — $MODULES modules a side (version ≤ 3); html5-qrcode.scanFile decoded the exact URL"
