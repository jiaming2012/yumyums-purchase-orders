// adversarial claim-2 check: independent decoder + finder-pattern truth on MY fresh PNGs
const fs = require('fs'); const path = require('path'); const { PNG } = require('pngjs'); const ZX = require('@zxing/library'); const jsQR = require('jsqr');
const FPF = require('@zxing/library/cjs/core/qrcode/detector/FinderPatternFinder').default;
const [dir, bulkf, key, vkey, pkey] = process.argv.slice(2); const m = require(path.join(dir, 'manifest.json')); const bulk = bulkf === '-' ? null : require(bulkf);
const load = (f) => { const p = PNG.sync.read(fs.readFileSync(f)); const g = new Uint8ClampedArray(p.width * p.height); for (let i = 0; i < g.length; i++) g[i] = (306 * p.data[4 * i] + 601 * p.data[4 * i + 1] + 117 * p.data[4 * i + 2] + 0x200) >> 10; return { w: p.width, h: p.height, g }; };
const pad = (img, n) => { const w = img.w + 2 * n, h = img.h + 2 * n; const g = new Uint8ClampedArray(w * h).fill(255); for (let y = 0; y < img.h; y++) g.set(img.g.subarray(y * img.w, (y + 1) * img.w), (y + n) * w + n); return { w, h, g }; };
const rot90 = (img) => { const g = new Uint8ClampedArray(img.w * img.h); for (let y = 0; y < img.h; y++) for (let x = 0; x < img.w; x++) g[x * img.h + (img.h - 1 - y)] = img.g[y * img.w + x]; return { w: img.h, h: img.w, g }; };
const bmpOf = (img) => new ZX.BinaryBitmap(new ZX.HybridBinarizer(new ZX.RGBLuminanceSource(img.g, img.w, img.h)));
const qr = (img) => { try { return new ZX.QRCodeReader().decode(bmpOf(img), new Map()).getText(); } catch (e) { return null; } };
const js = (img) => { const d = new Uint8ClampedArray(img.w * img.h * 4); for (let i = 0; i < img.g.length; i++) { d[4 * i] = d[4 * i + 1] = d[4 * i + 2] = img.g[i]; d[4 * i + 3] = 255; } const r = jsQR(d, img.w, img.h, { inversionAttempts: 'dontInvert' }); return r ? r.data : null; };
const t = { n: 0, node: 0, jsqr: 0, jsqrWrong: 0, agree: 0, brFail: 0, brFailJsqr: 0, brFailFinderBad: 0, brOkFinderBad: 0, nodeAny: 0, nodeNever: [] };
const B = bulk ? new Map(bulk.map((x) => [x.i, x])) : null;
for (const r of m) {
  if (B && !B.has(r.i)) continue; const pay = r[pkey || 'payload']; const v = r[vkey || 'card_v']; const mods = 17 + 4 * v;
  const im = load(path.join(dir, r[key || 'card'])); const P = pad(im, 100); const px = im.w / (mods + 8); const off = 4 * px; t.n++;
  const nodeOk = qr(P) === pay; if (nodeOk) t.node++;
  const j = js(P); if (j === pay) t.jsqr++; else if (j !== null) t.jsqrWrong++;
  let finderOk = false; try { const info = new FPF(bmpOf(P).getBlackMatrix(), null).find(new Map()); const got = [info.getTopLeft(), info.getTopRight(), info.getBottomLeft()].map((p) => [p.getX(), p.getY()]); const truth = [[3.5, 3.5], [mods - 3.5, 3.5], [3.5, mods - 3.5]].map(([x, y]) => [100 + off + x * px, 100 + off + y * px]); finderOk = got.every((g) => truth.some((q) => Math.hypot(g[0] - q[0], g[1] - q[1]) < 1.5 * px)); } catch (e) { finderOk = false; }
  let R = P, any = nodeOk; for (let k = 0; k < 3; k++) { R = rot90(R); if (qr(R) === pay) any = true; } if (any) t.nodeAny++; else t.nodeNever.push(r.i);
  if (B) { const x = B.get(r.i); const brOk = (x.q ? x.q[0] : x.t[0]) === pay; if (brOk === nodeOk) t.agree++; if (!brOk) { t.brFail++; if (j === pay) t.brFailJsqr++; if (!finderOk) t.brFailFinderBad++; } else if (!finderOk) t.brOkFinderBad++; }
}
console.log(JSON.stringify(t));
