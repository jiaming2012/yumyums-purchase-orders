// analyze2.js — WHERE does the vendored library (zxing-js) lose a pristine code? (reviewer, run 20261006)
//   node analyze2.js <dir> <key>
// For each PNG: (1) the shipped recipe restricted to QR; (2) the symbol's exact module grid, read
// straight off the PNG at module centres, handed to zxing's own QR Decoder (no locating step at
// all); (3) zxing's finder-pattern search, compared with where the three corner squares really are;
// (4) the four rotations; (5) jsQR as an independent decoder.
const fs = require('fs'); const path = require('path'); const { PNG } = require('pngjs');
const ZX = require('@zxing/library'); const jsQR = require('jsqr');
const Decoder = require('@zxing/library/cjs/core/qrcode/decoder/Decoder').default;
const FPF = require('@zxing/library/cjs/core/qrcode/detector/FinderPatternFinder').default;
const dir = process.argv[2]; const key = process.argv[3];
const rows = JSON.parse(fs.readFileSync(path.join(dir, `results-${key}.json`), 'utf8'));
function load(file) { const p = PNG.sync.read(fs.readFileSync(file)); const g = new Uint8ClampedArray(p.width * p.height); for (let i = 0; i < g.length; i++) g[i] = (306 * p.data[4 * i] + 601 * p.data[4 * i + 1] + 117 * p.data[4 * i + 2] + 0x200) >> 10; return { w: p.width, h: p.height, g }; }
function pad(img, n) { const w = img.w + 2 * n; const h = img.h + 2 * n; const g = new Uint8ClampedArray(w * h).fill(255); for (let y = 0; y < img.h; y++) g.set(img.g.subarray(y * img.w, (y + 1) * img.w), (y + n) * w + n); return { w, h, g }; }
function rot90(img) { const g = new Uint8ClampedArray(img.w * img.h); for (let y = 0; y < img.h; y++) for (let x = 0; x < img.w; x++) g[x * img.h + (img.h - 1 - y)] = img.g[y * img.w + x]; return { w: img.h, h: img.w, g }; }
const bmpOf = (img) => new ZX.BinaryBitmap(new ZX.HybridBinarizer(new ZX.RGBLuminanceSource(img.g, img.w, img.h)));
function qr(img) { try { return new ZX.QRCodeReader().decode(bmpOf(img), new Map()).getText(); } catch (e) { return null; } }
function js(img) { const d = new Uint8ClampedArray(img.w * img.h * 4); for (let i = 0; i < img.g.length; i++) { d[4 * i] = d[4 * i + 1] = d[4 * i + 2] = img.g[i]; d[4 * i + 3] = 255; } const r = jsQR(d, img.w, img.h, { inversionAttempts: 'dontInvert' }); return r ? r.data : null; }
const t = { n: rows.length, shipped: 0, exact: 0, exactAmongFail: 0, fail: 0, r90: 0, r180: 0, r270: 0, any: 0, jsqr: 0, finderWrong: 0, finderWrongAmongFail: 0, finderNone: 0, finderWrongAmongOk: 0 };
const per = [];
for (const r of rows) {
  const im = load(path.join(dir, r.file)); const mods = r.modules; const px = im.w / (mods + 8); const off = 4 * px; // this go-qrcode scales by nearest neighbour: pitch = size / (modules + 8), NOT an integer
  const P = pad(im, 100); const o = { i: r.i };
  o.shipped = qr(P) === r.payload;
  // exact grid → zxing's own Decoder
  const bm = new ZX.BitMatrix(mods, mods); for (let y = 0; y < mods; y++) for (let x = 0; x < mods; x++) if (im.g[Math.floor(off + (y + 0.5) * px) * im.w + Math.floor(off + (x + 0.5) * px)] < 128) bm.set(x, y);
  try { o.exact = new Decoder().decode(bm, new Map()).getText() === r.payload; } catch (e) { o.exact = false; o.exactErr = e.constructor.name; }
  // zxing's finder search vs truth
  const truth = [[3.5, 3.5], [mods - 3.5, 3.5], [3.5, mods - 3.5]].map(([x, y]) => [100 + off + x * px, 100 + off + y * px]);
  try { const info = new FPF(bmpOf(P).getBlackMatrix(), null).find(new Map()); const got = [info.getTopLeft(), info.getTopRight(), info.getBottomLeft()].map((p) => [p.getX(), p.getY()]);
    o.finderOk = got.every((g) => truth.some((q) => Math.hypot(g[0] - q[0], g[1] - q[1]) < 1.5 * px)); if (!o.finderOk) o.finderGot = got.map((g) => g.map((v) => +((v - 100 - off) / px).toFixed(1)));
  } catch (e) { o.finderOk = false; o.finderNone = true; }
  let R = P; const rots = []; for (let k = 0; k < 3; k++) { R = rot90(R); rots.push(qr(R) === r.payload); }
  [o.r90, o.r180, o.r270] = rots; o.any = o.shipped || rots.some(Boolean);
  o.jsqr = js(P) === r.payload;
  for (const k of ['shipped', 'exact', 'r90', 'r180', 'r270', 'any', 'jsqr']) if (o[k]) t[k]++;
  if (!o.shipped) { t.fail++; if (o.exact) t.exactAmongFail++; if (!o.finderOk) t.finderWrongAmongFail++; } else if (!o.finderOk) t.finderWrongAmongOk++;
  if (!o.finderOk) t.finderWrong++; if (o.finderNone) t.finderNone++;
  per.push(o);
}
const pct = (k) => `${k}/${t.n} (${(100 * k / t.n).toFixed(1)}%)`;
{ const im0 = load(path.join(dir, rows[0].file)); console.log(`ANALYZE2 ${key}: ${t.n} PNGs, version ${[...new Set(rows.map((r) => r.version))].join('/')}, ${im0.w}px, module pitch ${(im0.w / (rows[0].modules + 8)).toFixed(3)} px`); }
console.log(`  shipped recipe (zxing, QR only, as drawn)                 ${pct(t.shipped)}`);
console.log(`  exact module grid → zxing's own Decoder (no locating)     ${pct(t.exact)}   — of the ${t.fail} shipped-recipe failures: ${t.exactAmongFail} decode`);
console.log(`  independent decoder (jsQR)                                ${pct(t.jsqr)}`);
console.log(`  zxing's finder search picked 3 points that are NOT the 3 corner squares: ${t.finderWrong}/${t.n} — among shipped failures ${t.finderWrongAmongFail}/${t.fail}, among shipped successes ${t.finderWrongAmongOk}/${t.n - t.fail}`);
console.log(`  example wrong picks (module coordinates; true corners are 3.5 / ${rows[0].modules - 3.5}): ${JSON.stringify(per.filter((o) => o.finderGot).slice(0, 4).map((o) => o.finderGot))}`);
console.log(`  same pixels rotated: 90° ${pct(t.r90)} · 180° ${pct(t.r180)} · 270° ${pct(t.r270)}`);
console.log(`  RETRY READER — as drawn, else 90°, else 180°, else 270°   ${pct(t.any)}`);
fs.writeFileSync(path.join(dir, `analysis2-${key}.json`), JSON.stringify(per));
