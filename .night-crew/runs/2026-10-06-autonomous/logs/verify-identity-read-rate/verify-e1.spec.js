// verify-e1.spec.js — INDEPENDENT re-measurement of the identity-code read rate (reviewer probe,
// run 20261006). Not part of the suite: copied into a scratch worktree's tests/ as
// zz-verify-e1.spec.js, run alone, then removed.
//
// Differences from the session's density-probe on purpose:
//   * tokens are drawn from a DIFFERENT, reproducible stream (sha256 counter over VSEED);
//   * leg REAL drives the shipped page's own control: setInputFiles('#scan-file') → the page's
//     change listener → onFilePicked → new Html5Qrcode('scan-file-surface').scanFile → doScan →
//     #scan-result[data-kind,data-token-hash]. A read counts only when the rendered hash is
//     SHA-256(token). Fresh page load per code; offline after boot (as the signed spike did);
//   * leg SHORT is the session's shortcut (private Html5Qrcode on a 640 px div) on the SAME PNGs,
//     so the two can be compared code by code.
//
// env: V_N (codes), V_LEGS (comma list: real,short), V_VARIANTS (comma list of variant keys),
//      V_OUT (output dir for PNGs + results JSON), V_REPEAT (re-scans of each real-path failure)
const { test } = require('@playwright/test');
const { execFileSync } = require('child_process');
const crypto = require('crypto');
const path = require('path');
const fs = require('fs');

const ALPHA = '23456789abcdefghjkmnpqrstuvwxyz';
const N = Number(process.env.V_N || 160);
const SEED = process.env.VSEED || 'verify-e1-20261006-reviewer';
const OUT = process.env.V_OUT;
const LEGS = (process.env.V_LEGS || 'real,short').split(',');
const WANT = (process.env.V_VARIANTS || 'cardL').split(',');
const REPEAT = Number(process.env.V_REPEAT || 3);
const QRGEN = process.env.V_QRGEN;

function stream(label) { let c = 0; let buf = Buffer.alloc(0); return () => { if (!buf.length) buf = crypto.createHash('sha256').update(`${SEED}|${label}|${c++}`).digest(); const b = buf[0]; buf = buf.subarray(1); return b; }; }
function tokenFrom(next) { let t = ''; while (t.length < 16) { const b = next(); if (b < 248) t += ALPHA[b % 31]; } return t; }
function uuidFrom(next) { const b = Buffer.from(Array.from({ length: 16 }, next)); b[6] = (b[6] & 0x0f) | 0x40; b[8] = (b[8] & 0x3f) | 0x80; const h = b.toString('hex'); return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`; }
const b64 = (o) => Buffer.from(JSON.stringify(o)).toString('base64url');
const sha = (s) => crypto.createHash('sha256').update(s).digest('hex');
const BASE = 'https://hq.yumyums.kitchen/r/';
// A realistic first-touch offer mix (identity.go's key order: label, campaign_id, expires_at, face_value).
const OFFERS = [['Free side of wings', 2], ['$2 off any 6pc wings', 2], ['Free drink with any combo', 3], ['BOGO 6pc wings', 9.5], ['Free fries', 4]];
const full = (t, c, i) => `${BASE}${t}#o=${b64({ label: OFFERS[i % OFFERS.length][0], campaign_id: c, expires_at: `2026-1${i % 3}-${String(10 + (i % 19)).padStart(2, '0')}`, face_value: OFFERS[i % OFFERS.length][1] })}`;
const VARIANTS = {
  cardL: { level: 'L', size: 512, payload: full },
  cardM: { level: 'M', size: 512, payload: full },
  bareL: { level: 'L', size: 512, payload: (t) => `${BASE}${t}` },
  bareM: { level: 'M', size: 512, payload: (t) => `${BASE}${t}` },
  bareQ: { level: 'Q', size: 512, payload: (t) => `${BASE}${t}` },
  bareH: { level: 'H', size: 512, payload: (t) => `${BASE}${t}` },
  cardL1024: { level: 'L', size: 1024, payload: full },
  cardL244: { level: 'L', size: 244, payload: full },
};

test('verify-e1 read rate', async ({ page }) => {
  test.setTimeout(3000000);
  fs.mkdirSync(OUT, { recursive: true });
  const nt = stream('token'); const nc = stream('campaign');
  const tokens = Array.from({ length: N }, () => tokenFrom(nt));
  const camps = Array.from({ length: N }, () => uuidFrom(nc));

  await page.goto('/login.html');
  await page.fill('input[type="email"]', 'jamal@yumyums.kitchen');
  await page.fill('input[type="password"]', 'test123');
  await page.click('button.btn');
  await page.waitForURL((u) => !u.pathname.includes('login'));
  const open = async () => {
    await page.context().setOffline(false);
    await page.goto('/marketing.html');
    await page.waitForFunction(() => window.MarketingScan && window.MarketingScan.booted === true);
  };
  await open();
  const env = await page.evaluate(() => ({ barcodeDetector: 'BarcodeDetector' in window, ua: navigator.userAgent, surfaceW: document.getElementById('scan-file-surface').clientWidth, dpr: window.devicePixelRatio }));
  console.log('VERIFY ENV ' + JSON.stringify(env));

  const realScan = async (file) => {
    await open();
    await page.context().setOffline(true);
    await page.setInputFiles('#scan-file', file);
    await page.waitForSelector('#scan-result[data-kind]', { state: 'attached' });
    const r = page.locator('#scan-result');
    return { kind: await r.getAttribute('data-kind'), hash: await r.getAttribute('data-token-hash') };
  };

  for (const key of WANT) {
    const v = VARIANTS[key];
    const items = tokens.map((t, i) => {
      const payload = v.payload(t, camps[i], i);
      const file = path.join(OUT, `${key}-${String(i).padStart(3, '0')}.png`);
      const o = execFileSync(QRGEN, [payload, file, String(v.size), v.level], { encoding: 'utf8' });
      return { i, token: t, payload, file, version: Number(/version=(\d+)/.exec(o)[1]), modules: Number(/modules=(\d+)/.exec(o)[1]) };
    });
    const rows = items.map((x) => ({ i: x.i, token: x.token, payload: x.payload, file: path.basename(x.file), version: x.version, modules: x.modules }));

    if (LEGS.includes('real')) {
      for (const x of items) {
        const got = await realScan(x.file);
        const ok = got.hash === sha(x.token);
        rows[x.i].real = { kind: got.kind, ok, wrongHash: !!got.hash && !ok };
        if (!ok && REPEAT > 0) {
          rows[x.i].realRepeats = [];
          for (let k = 0; k < REPEAT; k++) { const g = await realScan(x.file); rows[x.i].realRepeats.push({ kind: g.kind, ok: g.hash === sha(x.token) }); }
        }
      }
      // determinism of SUCCESS too: re-scan the first 15 successes once each
      let re = 0; let reOk = 0;
      for (const x of items) { if (re >= 15) break; if (!rows[x.i].real.ok) continue; re++; const g = await realScan(x.file); if (g.hash === sha(x.token)) reOk++; }
      const ok = rows.filter((r) => r.real.ok).length;
      const kinds = {}; rows.forEach((r) => { kinds[r.real.kind] = (kinds[r.real.kind] || 0) + 1; });
      const fails = rows.filter((r) => !r.real.ok);
      const flips = fails.filter((r) => (r.realRepeats || []).some((g) => g.ok)).length;
      console.log(`VERIFY REAL  ${key.padEnd(10)} v=${[...new Set(items.map((x) => x.version))].join('/')} len=${[...new Set(items.map((x) => x.payload.length))].join('/')}  read ${ok}/${N} (${(100 * ok / N).toFixed(1)}%)  kinds=${JSON.stringify(kinds)}  wrong-hash=${rows.filter((r) => r.real.wrongHash).length}  failures-that-ever-read-on-${REPEAT}-repeats=${flips}/${fails.length}  successes-rescanned-ok=${reOk}/${re}`);
      console.log(`VERIFY REAL  ${key.padEnd(10)} marks ${rows.map((r) => (r.real.ok ? '.' : (r.real.wrongHash ? 'W' : 'x'))).join('')}`);
    }

    if (LEGS.includes('short')) {
      await open();
      const got = await page.evaluate(async (files) => {
        const res = [];
        for (let i = 0; i < files.length; i++) {
          const one = async (cfg) => {
            const h = document.createElement('div'); h.id = 'vprobe-' + i + '-' + (cfg ? 'q' : 'a'); h.style.cssText = 'position:absolute;left:-9999px;width:640px;height:640px;overflow:hidden'; document.body.appendChild(h);
            const f = new File([Uint8Array.from(atob(files[i]), (c) => c.charCodeAt(0))], `p${i}.png`, { type: 'image/png' });
            // eslint-disable-next-line no-undef
            try { return { text: await new Html5Qrcode(h.id, cfg).scanFile(f, false) }; } catch (e) { return { text: null, err: String(e && e.message ? e.message : e).slice(0, 120) }; } finally { h.remove(); }
          };
          // eslint-disable-next-line no-undef
          res.push({ all: await one(undefined), qr: await one({ formatsToSupport: [Html5QrcodeSupportedFormats.QR_CODE], verbose: false }) });
        }
        return res;
      }, items.map((x) => fs.readFileSync(x.file).toString('base64')));
      got.forEach((g, i) => { rows[i].short = { ok: g.all.text === items[i].payload, text: g.all.text === items[i].payload ? undefined : g.all.text, err: g.all.err }; rows[i].shortQr = { ok: g.qr.text === items[i].payload, text: g.qr.text === items[i].payload ? undefined : g.qr.text, err: g.qr.err }; });
      const ok = rows.filter((r) => r.short.ok).length; const okq = rows.filter((r) => r.shortQr.ok).length;
      const wrong = rows.filter((r) => !r.short.ok && r.short.text !== null && r.short.text !== undefined);
      const wrongq = rows.filter((r) => !r.shortQr.ok && r.shortQr.text !== null && r.shortQr.text !== undefined);
      console.log(`VERIFY SHORT ${key.padEnd(10)} v=${[...new Set(items.map((x) => x.version))].join('/')} read ${ok}/${N} (${(100 * ok / N).toFixed(1)}%)  wrong-text=${wrong.length} ${JSON.stringify(wrong.map((r) => r.short.text).slice(0, 6))}   QR-only read ${okq}/${N} (${(100 * okq / N).toFixed(1)}%) wrong-text=${wrongq.length}`);
      if (LEGS.includes('real')) console.log(`VERIFY AGREE ${key.padEnd(10)} real-vs-shortcut disagreements: ${rows.filter((r) => r.real.ok !== r.short.ok).length}/${N}`);
    }
    fs.writeFileSync(path.join(OUT, `results-${key}.json`), JSON.stringify(rows, null, 1));
  }
});
