#!/usr/bin/env bash
# 02-web-share-files-enumerated.sh — spike: the code sheet's primary action is
# "Share" (navigator.share with the PNG as a File). That API is NOT uniformly
# present, so the card needs the fallback matrix as a SET, not a guess (B-216):
# for each Playwright engine (chromium, webkit, firefox) record typeof
# navigator.share, typeof navigator.canShare, and canShare({files:[png]}).
# Premise falsified only if the enumeration cannot run; the FINDING is the
# matrix, and the card builds Save PNG / Copy link as the fallback wherever
# canShare is not true. (iOS Safari itself is not an engine Playwright ships;
# desktop WebKit is the nearest proxy and is labelled as such.)
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  all three engines enumerated.   exit 2  could not run.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
command -v node >/dev/null || cannot_run "node not on PATH"
[ -d "$REPO_ROOT/node_modules/playwright" ] || cannot_run "playwright not installed"
cat > /tmp/spike-h2-share.$$.cjs <<JS
const pw = require(process.argv[2] + '/node_modules/playwright');
(async () => {
  const rows = [];
  for (const name of ['chromium', 'webkit', 'firefox']) {
    let br; try { br = await pw[name].launch(); } catch (e) { rows.push([name, 'LAUNCH-FAILED', e.message.split('\n')[0]]); continue; }
    const pg = await br.newPage();
    const r = await pg.evaluate(() => {
      const f = new File([new Uint8Array([137,80,78,71])], 'code.png', { type: 'image/png' });
      let can = 'n/a'; try { can = navigator.canShare ? String(navigator.canShare({ files: [f] })) : 'no canShare'; } catch (e) { can = 'throws: ' + e.message; }
      return [typeof navigator.share, typeof navigator.canShare, can];
    });
    rows.push([name, ...r]); await br.close();
  }
  console.log('engine\tshare\tcanShare\tcanShare({files:[png]})');
  for (const r of rows) console.log(r.join('\t'));
  const launched = rows.filter(r => r[1] !== 'LAUNCH-FAILED').length;
  if (launched < 3) { console.log('ENUMERATION INCOMPLETE'); process.exit(2); }
})().catch(e => { console.log('ERR ' + e.message); process.exit(2); });
JS
echo "# the finding is the matrix below (desktop engines; webkit stands in for iOS Safari, labelled as a proxy):"
node /tmp/spike-h2-share.$$.cjs "$REPO_ROOT" || { rm -f /tmp/spike-h2-share.$$.cjs; cannot_run "an engine failed to launch — run npx playwright install"; }
rm -f /tmp/spike-h2-share.$$.cjs
echo "✅ GREEN — enumerated; the card ships Save PNG + Copy link as the fallback wherever canShare is not 'true'"
