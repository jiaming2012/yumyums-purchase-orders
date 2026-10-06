// ADVERSARIAL REVIEW SCRATCH — run 20261006. Copied into my own worktree's tests/ only.
const { test } = require('@playwright/test');
const crypto = require('crypto'); const fs = require('fs'); const path = require('path');
const DIR = process.env.ADV_DIR; const OUT = process.env.ADV_OUT; const MODE = process.env.ADV_MODE;
const K = Number(process.env.ADV_PAGES || 6);
const LIB = path.join(__dirname, '..', 'lib', 'html5-qrcode.min.js');
const sha = (s) => crypto.createHash('sha256').update(s).digest('hex');
const manifest = () => JSON.parse(fs.readFileSync(path.join(DIR, 'manifest.json'), 'utf8'));
const b64 = (f) => fs.readFileSync(path.join(DIR, f)).toString('base64');

const HELPERS = () => {
  window.advFile = (b, name) => new File([Uint8Array.from(atob(b), (c) => c.charCodeAt(0))], name || 'p.png', { type: 'image/png' });
  let seq = 0;
  window.advScan = async (file, size, cfg) => {
    const h = document.createElement('div'); h.id = 'adv-' + (seq++); h.style.cssText = `position:absolute;left:-9999px;width:${size}px;height:${size}px;overflow:hidden`; document.body.appendChild(h);
    // eslint-disable-next-line no-undef
    try { return await new Html5Qrcode(h.id, cfg).scanFile(file, false); } catch (e) { return null; } finally { h.remove(); }
  };
  window.advTurn = async (file, k) => {
    const bmp = await createImageBitmap(file); const c = document.createElement('canvas');
    c.width = k % 2 ? bmp.height : bmp.width; c.height = k % 2 ? bmp.width : bmp.height; const x = c.getContext('2d');
    x.imageSmoothingEnabled = false; x.fillStyle = '#fff'; x.fillRect(0, 0, c.width, c.height); x.translate(c.width / 2, c.height / 2); x.rotate(k * Math.PI / 2); x.drawImage(bmp, -bmp.width / 2, -bmp.height / 2);
    const blob = await new Promise((r) => c.toBlob(r, 'image/png')); return new File([blob], 't.png', { type: 'image/png' });
  };
  // "photo-like" perturbations
  window.advPerturb = async (file, kind) => {
    const bmp = await createImageBitmap(file); const c = document.createElement('canvas'); let x; let type = 'image/jpeg'; let q = 0.6;
    if (kind === 'half') { c.width = 256; c.height = 256; x = c.getContext('2d'); x.imageSmoothingQuality = 'high'; x.drawImage(bmp, 0, 0, 256, 256); type = 'image/png'; }
    else if (kind === 'jpeg') { c.width = bmp.width; c.height = bmp.height; x = c.getContext('2d'); x.drawImage(bmp, 0, 0); q = 0.5; }
    else if (kind === 'screenshot') { c.width = 1080; c.height = 1920; x = c.getContext('2d'); x.fillStyle = '#f2f2f2'; x.fillRect(0, 0, 1080, 1920); x.fillStyle = '#222'; x.font = '40px sans-serif'; x.fillText('Yumyums: show this code at the window', 60, 300); x.imageSmoothingQuality = 'high'; x.drawImage(bmp, 140, 500, 800, 800); q = 0.7; }
    else if (kind === 'tilt') { c.width = 900; c.height = 900; x = c.getContext('2d'); x.fillStyle = '#ddd'; x.fillRect(0, 0, 900, 900); x.translate(450, 450); x.rotate(7 * Math.PI / 180); x.imageSmoothingQuality = 'high'; x.drawImage(bmp, -300, -300, 600, 600); q = 0.7; }
    const blob = await new Promise((r) => c.toBlob(r, type, q)); return new File([blob], 'x', { type });
  };
  window.advShard = async (items, plan) => {
    const res = [];
    for (const it of items) {
      const f = window.advFile(it.b); const r = { i: it.i };
      if (plan.rot) { r.t = [await window.advScan(f, 640)]; for (let k = 1; k <= 3; k++) r.t.push(await window.advScan(await window.advTurn(f, k), 640)); }
      else r.t = [await window.advScan(f, 640)];
      if (plan.s320) r.s320 = await window.advScan(f, 320);
      // eslint-disable-next-line no-undef
      if (plan.qr) r.qr = await window.advScan(f, 640, { formatsToSupport: [Html5QrcodeSupportedFormats.QR_CODE], verbose: false });
      // eslint-disable-next-line no-undef
      if (plan.rotqr) { const cfg = { formatsToSupport: [Html5QrcodeSupportedFormats.QR_CODE], verbose: false }; r.q = [await window.advScan(f, 640, cfg)]; for (let k = 1; k <= 3; k++) r.q.push(await window.advScan(await window.advTurn(f, k), 640, cfg)); }
      if (plan.perturb) { r.p = {}; for (const kind of plan.perturb) { const pf = await window.advPerturb(f, kind); const cfg = plan.pqr ? { formatsToSupport: [Html5QrcodeSupportedFormats.QR_CODE], verbose: false } : undefined; r.p[kind] = [await window.advScan(pf, 640, cfg)]; for (let k = 1; k <= 3; k++) r.p[kind].push(await window.advScan(await window.advTurn(pf, k), 640, cfg)); } }
      res.push(r);
    }
    return res;
  };
};

async function bulk(browser, items, plan) {
  const shards = Array.from({ length: K }, () => []); items.forEach((it, j) => shards[j % K].push(it));
  const out = await Promise.all(shards.map(async (sh) => {
    const ctx = await browser.newContext(); const page = await ctx.newPage(); await page.goto('about:blank');
    await page.addScriptTag({ path: LIB }); await page.evaluate(HELPERS);
    const res = [];
    for (let o = 0; o < sh.length; o += 40) res.push(...await page.evaluate(([its, pl]) => window.advShard(its, pl), [sh.slice(o, o + 40).map((it) => ({ i: it.i, b: b64(it.file) })), plan]));
    await ctx.close(); return res;
  }));
  return out.flat().sort((a, b) => a.i - b.i);
}

test('adv', async ({ browser, page }) => {
  test.setTimeout(6 * 3600 * 1000);
  const m = MODE === 'env' ? [] : manifest(); const lo = Number(process.env.ADV_LO || 0); const hi = Number(process.env.ADV_HI || m.length); const rows = m.slice(lo, hi);
  if (MODE === 'bulk') {
    const variant = process.env.ADV_VARIANT || 'card'; const plan = JSON.parse(process.env.ADV_PLAN || '{"rot":true}');
    const t0 = Date.now();
    const res = await bulk(browser, rows.map((r) => ({ i: r.i, file: r[variant] })), plan);
    fs.writeFileSync(OUT, JSON.stringify(res)); console.log(`ADV bulk ${variant} n=${res.length} plan=${JSON.stringify(plan)} secs=${((Date.now() - t0) / 1000).toFixed(0)}`);
    return;
  }
  if (MODE === 'env') {
    await page.goto('about:blank');
    console.log('ADV ENV ' + JSON.stringify(await page.evaluate(() => ({ bd: 'BarcodeDetector' in window, ua: navigator.userAgent }))));
    return;
  }
  if (MODE === 'real') {
    // the shipped page's own control, K logged-in contexts. ADV_PATCH=<file> swaps marketing/scan-page.js for a scratch copy.
    const want = process.env.ADV_ONLY ? new Set(JSON.parse(fs.readFileSync(process.env.ADV_ONLY, 'utf8'))) : null;
    const variant = process.env.ADV_VARIANT || 'card'; const reps = Number(process.env.ADV_REPS || 1);
    const todo = rows.filter((r) => !want || want.has(r.i)); const shards = Array.from({ length: K }, () => []); todo.forEach((r, j) => shards[j % K].push(r));
    const t0 = Date.now();
    const out = await Promise.all(shards.map(async (sh) => {
      const ctx = await browser.newContext({ baseURL: test.info().project.use.baseURL, serviceWorkers: 'block' }); const p = await ctx.newPage();
      if (process.env.ADV_PATCH) await p.route('**/marketing/scan-page.js', (route) => route.fulfill({ contentType: 'text/javascript', body: fs.readFileSync(process.env.ADV_PATCH, 'utf8') }));
      await p.goto('/login.html'); await p.fill('input[type="email"]', 'jamal@yumyums.kitchen'); await p.fill('input[type="password"]', 'test123'); await p.click('button.btn'); await p.waitForURL((u) => !u.pathname.includes('login'));
      const res = [];
      for (const r of sh) {
        const got = [];
        for (let k = 0; k < reps; k++) {
          await ctx.setOffline(false); await p.goto('/marketing.html'); await p.waitForFunction(() => window.MarketingScan && window.MarketingScan.booted === true);
          await ctx.setOffline(true);
          await p.setInputFiles('#scan-file', path.join(DIR, r[variant]));
          await p.waitForSelector('#scan-result[data-kind]', { state: 'attached' });
          const el = p.locator('#scan-result'); got.push({ kind: await el.getAttribute('data-kind'), hash: await el.getAttribute('data-token-hash'), text: (await el.innerText().catch(() => '')).slice(0, 160) });
        }
        res.push({ i: r.i, want: sha(r.token), got });
      }
      await ctx.close(); return res;
    }));
    const res = out.flat().sort((a, b) => a.i - b.i); fs.writeFileSync(OUT, JSON.stringify(res));
    const ok = res.filter((r) => r.got[0].hash === r.want).length; const wrong = res.filter((r) => r.got.some((g) => g.hash && g.hash !== r.want)).length;
    const kinds = {}; res.forEach((r) => { kinds[r.got[0].kind] = (kinds[r.got[0].kind] || 0) + 1; });
    const unstable = res.filter((r) => new Set(r.got.map((g) => g.kind + '|' + g.hash)).size > 1).length;
    console.log(`ADV real ${variant} patch=${process.env.ADV_PATCH ? 'YES' : 'no'} n=${res.length} read=${ok} wrongHash=${wrong} kinds=${JSON.stringify(kinds)} reps=${reps} unstable-across-reps=${unstable} secs=${((Date.now() - t0) / 1000).toFixed(0)}`);
  }
});
