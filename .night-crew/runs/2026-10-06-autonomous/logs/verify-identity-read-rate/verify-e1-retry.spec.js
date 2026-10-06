// verify-e1-retry.spec.js — reviewer probe (run 20261006): would a "turn it and try again" step in
// the page fix the photo path, using ONLY the vendored decoder the page already ships?
// SCRATCH MEASUREMENT, NOT A LANDED CHANGE. Nothing in marketing/ is edited: the retry is done
// from the test, in the shipped marketing.html, with the page's own Html5Qrcode#scanFile.
//   attempt 0: the PNG as drawn (identical to the page today — validated code-for-code against
//              the real #scan-file path in verify-e1.spec.js)
//   attempt k: the same picture turned k×90° on a canvas (imageSmoothingEnabled=false), re-encoded
//              as PNG, handed to scanFile again
// Also: the server's picture drawn at an exact whole number of pixels per module (go-qrcode
// at 488 px = 61 × 8) — is the uneven module width the cause?  V_ONLY=<i,j> picks variants by index.
const { test } = require('@playwright/test');
const { execFileSync } = require('child_process');
const crypto = require('crypto');
const path = require('path');
const fs = require('fs');
const ALPHA = '23456789abcdefghjkmnpqrstuvwxyz';
const N = Number(process.env.V_N || 250);
const SEED = process.env.VSEED || 'reviewer-retry';
const OUT = process.env.V_OUT; const QRGEN = process.env.V_QRGEN;
function stream(label) { let c = 0; let buf = Buffer.alloc(0); return () => { if (!buf.length) buf = crypto.createHash('sha256').update(`${SEED}|${label}|${c++}`).digest(); const b = buf[0]; buf = buf.subarray(1); return b; }; }
function tokenFrom(next) { let t = ''; while (t.length < 16) { const b = next(); if (b < 248) t += ALPHA[b % 31]; } return t; }
function uuidFrom(next) { const b = Buffer.from(Array.from({ length: 16 }, next)); b[6] = (b[6] & 0x0f) | 0x40; b[8] = (b[8] & 0x3f) | 0x80; const h = b.toString('hex'); return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`; }
const b64 = (o) => Buffer.from(JSON.stringify(o)).toString('base64url');
const BASE = 'https://hq.yumyums.kitchen/r/';
const OFFERS = [['Free side of wings', 2], ['$2 off any 6pc wings', 2], ['Free drink with any combo', 3], ['BOGO 6pc wings', 9.5], ['Free fries', 4]];
const full = (t, c, i) => `${BASE}${t}#o=${b64({ label: OFFERS[i % OFFERS.length][0], campaign_id: c, expires_at: `2026-1${i % 3}-${String(10 + (i % 19)).padStart(2, '0')}`, face_value: OFFERS[i % OFFERS.length][1] })}`;
const VARIANTS = [
  ['card Low 512 (as built)', 'L', '512', full],
  ['card Low, exact 8 px/module (488 px)', 'L', '488', full],
  ['card Medium 512 (the "negative control")', 'M', '512', full],
  ['bare identity Low 512', 'L', '512', (t) => `${BASE}${t}`],
];
test('verify-e1 retry reader', async ({ page }) => {
  test.setTimeout(3000000);
  fs.mkdirSync(OUT, { recursive: true });
  const nt = stream('token'); const nc = stream('campaign');
  const tokens = Array.from({ length: N }, () => tokenFrom(nt)); const camps = Array.from({ length: N }, () => uuidFrom(nc));
  await page.goto('/login.html');
  await page.fill('input[type="email"]', 'jamal@yumyums.kitchen');
  await page.fill('input[type="password"]', 'test123');
  await page.click('button.btn');
  await page.waitForURL((u) => !u.pathname.includes('login'));
  await page.goto('/marketing.html');
  await page.waitForFunction(() => window.MarketingScan && window.MarketingScan.booted === true);
  const ONLY = process.env.V_ONLY ? process.env.V_ONLY.split(',').map(Number) : null;
  for (const [name, level, size, pf] of VARIANTS.filter((_, i) => !ONLY || ONLY.includes(i))) {
    const items = tokens.map((t, i) => { const payload = pf(t, camps[i], i); const file = path.join(OUT, `r-${i}.png`); execFileSync(QRGEN, [payload, file, size, level]); return { payload, data: fs.readFileSync(file).toString('base64') }; });
    const got = await page.evaluate(async (files) => {
      const scan = async (file, tag) => {
        const h = document.createElement('div'); h.id = 'vr-' + tag; h.style.cssText = 'position:absolute;left:-9999px;width:640px;height:640px;overflow:hidden'; document.body.appendChild(h);
        // eslint-disable-next-line no-undef
        try { return await new Html5Qrcode(h.id).scanFile(file, false); } catch (e) { return null; } finally { h.remove(); }
      };
      const turned = async (file, k) => {
        const bmp = await createImageBitmap(file); const c = document.createElement('canvas'); c.width = bmp.width; c.height = bmp.height; const x = c.getContext('2d');
        x.imageSmoothingEnabled = false; x.fillStyle = '#fff'; x.fillRect(0, 0, c.width, c.height); x.translate(c.width / 2, c.height / 2); x.rotate(k * Math.PI / 2); x.drawImage(bmp, -bmp.width / 2, -bmp.height / 2);
        const blob = await new Promise((r) => c.toBlob(r, 'image/png')); return new File([blob], 't.png', { type: 'image/png' });
      };
      const res = [];
      for (let i = 0; i < files.length; i++) {
        const f = new File([Uint8Array.from(atob(files[i]), (c) => c.charCodeAt(0))], `p${i}.png`, { type: 'image/png' });
        const texts = [await scan(f, i + '-0')];
        for (let k = 1; k <= 3; k++) texts.push(await scan(await turned(f, k), i + '-' + k));
        res.push(texts);
      }
      return res;
    }, items.map((x) => x.data));
    const isTok = (t) => typeof t === 'string' && /\/r\/([^/?#]+)(?=[?#]|$)/.test(t); // the page's own TOKEN_PATTERN gate
    const first = got.filter((g, i) => g[0] === items[i].payload).length;
    const each = [1, 2, 3].map((k) => got.filter((g, i) => g[k] === items[i].payload).length);
    // retry policy as a page would implement it: take the first attempt whose text passes the page's /r/<token> gate
    let retryOk = 0; let retryWrong = 0; let attemptsUsed = [0, 0, 0, 0];
    got.forEach((g, i) => { const k = g.findIndex(isTok); if (k < 0) return; attemptsUsed[k]++; if (g[k] === items[i].payload) retryOk++; else retryWrong++; });
    const stray = got.flat().filter((t, j) => t !== null && t !== items[Math.floor(j / 4)].payload);
    console.log(`VERIFY RETRY ${name.padEnd(42)} N=${N}  as drawn ${first} (${(100 * first / N).toFixed(1)}%)  turned 90/180/270: ${each.join('/')}  RETRY-UNTIL-/r/-TOKEN ${retryOk} (${(100 * retryOk / N).toFixed(1)}%) wrong-token-accepted=${retryWrong} read-on-attempt#=${JSON.stringify(attemptsUsed)}  non-payload texts seen=${JSON.stringify([...new Set(stray)].slice(0, 6))}`);
  }
});
