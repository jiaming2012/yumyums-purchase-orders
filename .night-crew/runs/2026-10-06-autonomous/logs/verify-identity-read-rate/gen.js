// gen.js — draws N codes per variant with the card's own go-qrcode (backend/cmd/qrgen) and writes
// results-<key>.json for analyze.js. Token stream differs from the browser probe's (SEED label).
//   node gen.js <qrgen-bin> <outdir> <N> <seed> <key,key,...>
const { execFileSync } = require('child_process'); const crypto = require('crypto'); const fs = require('fs'); const path = require('path');
const [bin, out, Ns, SEED, keys] = process.argv.slice(2); const N = Number(Ns);
const ALPHA = '23456789abcdefghjkmnpqrstuvwxyz';
function stream(label) { let c = 0; let buf = Buffer.alloc(0); return () => { if (!buf.length) buf = crypto.createHash('sha256').update(`${SEED}|${label}|${c++}`).digest(); const b = buf[0]; buf = buf.subarray(1); return b; }; }
function tokenFrom(next) { let t = ''; while (t.length < 16) { const b = next(); if (b < 248) t += ALPHA[b % 31]; } return t; }
function uuidFrom(next) { const b = Buffer.from(Array.from({ length: 16 }, next)); b[6] = (b[6] & 0x0f) | 0x40; b[8] = (b[8] & 0x3f) | 0x80; const h = b.toString('hex'); return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`; }
const b64 = (o) => Buffer.from(JSON.stringify(o)).toString('base64url');
const BASE = 'https://hq.yumyums.kitchen/r/';
const OFFERS = [['Free side of wings', 2], ['$2 off any 6pc wings', 2], ['Free drink with any combo', 3], ['BOGO 6pc wings', 9.5], ['Free fries', 4]];
const full = (t, c, i) => `${BASE}${t}#o=${b64({ label: OFFERS[i % OFFERS.length][0], campaign_id: c, expires_at: `2026-1${i % 3}-${String(10 + (i % 19)).padStart(2, '0')}`, face_value: OFFERS[i % OFFERS.length][1] })}`;
const bare = (t) => `${BASE}${t}`;
const V = { cardL: ['L', 512, full], cardM: ['M', 512, full], cardQ: ['Q', 512, full], cardH: ['H', 512, full], bareL: ['L', 512, bare], bareM: ['M', 512, bare], bareQ: ['Q', 512, bare], bareH: ['H', 512, bare], cardL1024: ['L', 1024, full], cardL488: ['L', 488, full], cardL976: ['L', 976, full], bareL518: ['L', 518, bare], bareL296: ['L', 296, bare] };
fs.mkdirSync(out, { recursive: true });
const nt = stream('token'); const nc = stream('campaign');
const tokens = Array.from({ length: N }, () => tokenFrom(nt)); const camps = Array.from({ length: N }, () => uuidFrom(nc));
for (const key of keys.split(',')) {
  const [level, size, pf] = V[key];
  const rows = tokens.map((t, i) => { const payload = pf(t, camps[i], i); const file = `${key}-${String(i).padStart(3, '0')}.png`; const o = execFileSync(bin, [payload, path.join(out, file), String(size), level], { encoding: 'utf8' }); return { i, token: t, payload, file, version: Number(/version=(\d+)/.exec(o)[1]), modules: Number(/modules=(\d+)/.exec(o)[1]) }; });
  fs.writeFileSync(path.join(out, `results-${key}.json`), JSON.stringify(rows));
}
