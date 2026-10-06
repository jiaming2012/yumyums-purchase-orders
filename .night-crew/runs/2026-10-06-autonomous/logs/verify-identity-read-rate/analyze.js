// analyze.js — reviewer root-cause probe (run 20261006). Node, scratch-local deps only
// (@zxing/library 0.19.1 — the library html5-qrcode vendors — jsqr, pngjs). No browser.
//
//   node analyze.js <dir-with-results-<key>.json-and-PNGs> <key>
//
// For every PNG the browser probe drew it rebuilds what the shipped page hands its decoder
// (html5-qrcode scanFile: the image at NATIVE size on a canvas padded by 100 transparent px,
// which zxing's canvas luminance source reads as white; HybridBinarizer; MultiFormatReader with
// every format and TRY_HARDER=false) and then varies ONE thing at a time.
const fs = require('fs');
const path = require('path');
const { PNG } = require('pngjs');
const ZX = require('@zxing/library');
const jsQR = require('jsqr');

const dir = process.argv[2]; const key = process.argv[3];
const rows = JSON.parse(fs.readFileSync(path.join(dir, `results-${key}.json`), 'utf8'));
const ALL = [ZX.BarcodeFormat.QR_CODE, ZX.BarcodeFormat.AZTEC, ZX.BarcodeFormat.CODABAR, ZX.BarcodeFormat.CODE_39, ZX.BarcodeFormat.CODE_93, ZX.BarcodeFormat.CODE_128, ZX.BarcodeFormat.DATA_MATRIX, ZX.BarcodeFormat.MAXICODE, ZX.BarcodeFormat.ITF, ZX.BarcodeFormat.EAN_13, ZX.BarcodeFormat.EAN_8, ZX.BarcodeFormat.PDF_417, ZX.BarcodeFormat.RSS_14, ZX.BarcodeFormat.RSS_EXPANDED, ZX.BarcodeFormat.UPC_A, ZX.BarcodeFormat.UPC_E, ZX.BarcodeFormat.UPC_EAN_EXTENSION];

function load(file) { const p = PNG.sync.read(fs.readFileSync(file)); const g = new Uint8ClampedArray(p.width * p.height); for (let i = 0; i < g.length; i++) g[i] = (306 * p.data[4 * i] + 601 * p.data[4 * i + 1] + 117 * p.data[4 * i + 2] + 0x200) >> 10; return { w: p.width, h: p.height, g }; }
function pad(img, n) { const w = img.w + 2 * n; const h = img.h + 2 * n; const g = new Uint8ClampedArray(w * h).fill(255); for (let y = 0; y < img.h; y++) g.set(img.g.subarray(y * img.w, (y + 1) * img.w), (y + n) * w + n); return { w, h, g }; }
function rot90(img) { const g = new Uint8ClampedArray(img.w * img.h); for (let y = 0; y < img.h; y++) for (let x = 0; x < img.w; x++) g[x * img.h + (img.h - 1 - y)] = img.g[y * img.w + x]; return { w: img.h, h: img.w, g }; }
function scaleNN(img, f) { const w = Math.round(img.w * f); const h = Math.round(img.h * f); const g = new Uint8ClampedArray(w * h); for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) g[y * w + x] = img.g[Math.min(img.h - 1, Math.floor(y / f)) * img.w + Math.min(img.w - 1, Math.floor(x / f))]; return { w, h, g }; }
function zx(img, { formats = ALL, tryHarder = false, binarizer = 'hybrid', pure = false } = {}) {
  const hints = new Map(); hints.set(ZX.DecodeHintType.POSSIBLE_FORMATS, formats); hints.set(ZX.DecodeHintType.TRY_HARDER, tryHarder); if (pure) hints.set(ZX.DecodeHintType.PURE_BARCODE, true);
  const src = new ZX.RGBLuminanceSource(img.g, img.w, img.h);
  const bmp = new ZX.BinaryBitmap(binarizer === 'hybrid' ? new ZX.HybridBinarizer(src) : new ZX.GlobalHistogramBinarizer(src));
  try { const r = new ZX.MultiFormatReader(); r.setHints(hints); const res = r.decode(bmp); return { text: res.getText(), fmt: res.getBarcodeFormat() }; } catch (e) { return { text: null, err: e.constructor.name || e.name }; }
}
// where inside the QR reader does it die?
function stage(img) {
  const bmp = new ZX.BinaryBitmap(new ZX.HybridBinarizer(new ZX.RGBLuminanceSource(img.g, img.w, img.h)));
  try { new ZX.QRCodeReader().decode(bmp, new Map()); return 'ok'; } catch (e) { return (e.constructor.name || e.name) + (e.message ? ':' + String(e.message).slice(0, 40) : ''); }
}
function js(img) { const d = new Uint8ClampedArray(img.w * img.h * 4); for (let i = 0; i < img.g.length; i++) { d[4 * i] = d[4 * i + 1] = d[4 * i + 2] = img.g[i]; d[4 * i + 3] = 255; } const r = jsQR(d, img.w, img.h, { inversionAttempts: 'dontInvert' }); return r ? r.data : null; }

const T = {
  'A shipped recipe (native, +100px pad, all formats, hybrid)': (im) => zx(pad(im, 100)),
  'B QR format only': (im) => zx(pad(im, 100), { formats: [ZX.BarcodeFormat.QR_CODE] }),
  'C QR only + TRY_HARDER': (im) => zx(pad(im, 100), { formats: [ZX.BarcodeFormat.QR_CODE], tryHarder: true }),
  'D QR only + PURE_BARCODE, no pad': (im) => zx(im, { formats: [ZX.BarcodeFormat.QR_CODE], pure: true }),
  'E QR only, GlobalHistogram binarizer': (im) => zx(pad(im, 100), { formats: [ZX.BarcodeFormat.QR_CODE], binarizer: 'global' }),
  'F QR only, no pad (raw PNG)': (im) => zx(im, { formats: [ZX.BarcodeFormat.QR_CODE] }),
  'G QR only, pad 103 (grid off the 8px block lattice)': (im) => zx(pad(im, 103), { formats: [ZX.BarcodeFormat.QR_CODE] }),
  'H QR only, pad 300 (huge quiet zone)': (im) => zx(pad(im, 300), { formats: [ZX.BarcodeFormat.QR_CODE] }),
  'I QR only, rotated 90deg': (im) => zx(rot90(pad(im, 100)), { formats: [ZX.BarcodeFormat.QR_CODE] }),
  'J QR only, x0.5 nearest (4px modules)': (im) => zx(pad(scaleNN(im, 0.5), 100), { formats: [ZX.BarcodeFormat.QR_CODE] }),
  'K QR only, x2 nearest (16px modules)': (im) => zx(pad(scaleNN(im, 2), 100), { formats: [ZX.BarcodeFormat.QR_CODE] }),
  'L QR only, x0.75 nearest (6px modules)': (im) => zx(pad(scaleNN(im, 0.75), 100), { formats: [ZX.BarcodeFormat.QR_CODE] }),
};
const tally = {}; const per = []; const stages = {}; let jsOk = 0; let jsWrong = 0; let agree = 0; let agreeN = 0; const wrongTexts = [];
for (const r of rows) {
  const im = load(path.join(dir, r.file)); const o = { i: r.i };
  for (const [name, f] of Object.entries(T)) { const res = f(im); const ok = res.text === r.payload; o[name[0]] = ok; tally[name] = (tally[name] || 0) + (ok ? 1 : 0); if (res.text !== null && !ok) wrongTexts.push({ i: r.i, test: name[0], fmt: res.fmt, text: res.text.slice(0, 60) }); }
  const j = js(pad(im, 100)); o.jsqr = j === r.payload; if (o.jsqr) jsOk++; else if (j !== null) jsWrong++;
  if (!o.B) { const s = stage(pad(im, 100)); o.stage = s; stages[s] = (stages[s] || 0) + 1; }
  const browser = r.short ? r.short.ok : (r.real ? r.real.ok : undefined);
  if (browser !== undefined) { agreeN++; if (browser === o.A) agree++; }
  o.retry = o.B || o.I || o.J || o.K; // a "try again another way" reader
  per.push(o);
}
const n = rows.length; const pct = (k) => `${String(k).padStart(4)}/${n} (${(100 * k / n).toFixed(1)}%)`;
console.log(`ANALYZE ${key}: ${n} PNGs, version ${[...new Set(rows.map((r) => r.version))].join('/')}, payload ${[...new Set(rows.map((r) => r.payload.length))].join('/')} chars`);
const one = load(path.join(dir, rows[0].file)); console.log(`  PNG ${one.w}x${one.h}; symbol ${rows[0].modules} modules + 8 quiet = ${rows[0].modules + 8}; ${one.w}/${rows[0].modules + 8} → ${Math.floor(one.w / (rows[0].modules + 8))} px per module exactly, ${(one.w - Math.floor(one.w / (rows[0].modules + 8)) * (rows[0].modules + 8)) / 2} px extra white each side; distinct grey levels in the file: ${[...new Set(one.g)].join(',')}`);
if (agreeN) console.log(`  node recipe A vs the BROWSER's own result for the same PNG: agree on ${agree}/${agreeN}`);
for (const name of Object.keys(T)) console.log(`  ${name.padEnd(58)} ${pct(tally[name])}`);
console.log(`  ${'INDEPENDENT decoder (jsQR), same padded pixels'.padEnd(58)} ${pct(jsOk)}  wrong-text ${jsWrong}`);
console.log(`  ${'jsQR reads, among the PNGs recipe B could NOT read'.padEnd(58)} ${per.filter((o) => !o.B && o.jsqr).length}/${per.filter((o) => !o.B).length}`);
console.log(`  ${'retry reader: B, else rot90, else x0.5, else x2'.padEnd(58)} ${pct(per.filter((o) => o.retry).length)}`);
console.log(`  where zxing's QR reader gives up on recipe-B failures: ${JSON.stringify(stages)}`);
console.log(`  decoded-but-not-the-payload (any recipe): ${wrongTexts.length} ${JSON.stringify(wrongTexts.slice(0, 8))}`);
fs.writeFileSync(path.join(dir, `analysis-${key}.json`), JSON.stringify(per));
