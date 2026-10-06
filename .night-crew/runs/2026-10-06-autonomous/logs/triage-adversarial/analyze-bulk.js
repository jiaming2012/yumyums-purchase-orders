// node analyze-bulk.js <bulk.json> <manifest.json> <payloadKey>
const [f, mf, pk] = process.argv.slice(2); const r = require(f); const m = require(mf); const PK = pk || 'payload';
const TOK = /\/r\/([^/?#]+)(?=[?#]|$)/; const tokOf = (t) => (typeof t === 'string' && TOK.test(t)) ? TOK.exec(t)[1] : null;
const pct = (a, n) => `${a}/${n} (${(100 * a / n).toFixed(2)}%)`;
const N = r.length; const P = (x) => m[x.i][PK];
const rep = (name, sel) => {
  if (!r[0][sel]) return; const arr = (x) => x[sel];
  const drawn = r.filter((x) => arr(x)[0] === P(x)).length;
  console.log(`[${name}] as drawn exact-read ${pct(drawn, N)}; fail 1 in ${(N / (N - drawn)).toFixed(1)}`);
  if (arr(r[0]).length < 4) return;
  console.log(`[${name}] per turn 0/90/180/270 exact: ${[0, 1, 2, 3].map((k) => r.filter((x) => arr(x)[k] === P(x)).length).join(' / ')}`);
  // policy: first attempt whose text passes the /r/ token gate
  let ok = 0, wrongTok = 0, none = 0; const used = [0, 0, 0, 0]; const missed = [];
  r.forEach((x) => { const k = arr(x).findIndex((t) => tokOf(t)); if (k < 0) { none++; missed.push(x.i); return; } used[k]++; if (tokOf(arr(x)[k]) === m[x.i].token) ok++; else wrongTok++; });
  console.log(`[${name}] RETRY (drawn, then 90/180/270; first /r/ text wins): read ${pct(ok, N)}  NOT READ ${none}  wrong-token-accepted ${wrongTok}  read-on-attempt ${JSON.stringify(used)}`);
  const anyExactButTextDiff = r.filter((x) => arr(x).some((t) => tokOf(t) && t !== P(x))).length;
  const stray = r.flatMap((x) => arr(x).filter((t) => t !== null && t !== P(x)));
  console.log(`[${name}] texts that are not the payload: ${stray.length} scans, ${new Set(stray).size} distinct, any passing the /r/ gate: ${anyExactButTextDiff}; sample ${JSON.stringify([...new Set(stray)].slice(0, 5))}`);
  const strayDrawn = r.filter((x) => arr(x)[0] !== null && arr(x)[0] !== P(x)).length;
  console.log(`[${name}] as-drawn failures: no-text ${r.filter((x) => arr(x)[0] === null).length}, non-code text ${strayDrawn}`);
  const byV = {}; r.forEach((x) => { const v = 'v' + (m[x.i].card_v) + ' len' + P(x).length; byV[v] = byV[v] || [0, 0, 0]; byV[v][0]++; if (arr(x)[0] === P(x)) byV[v][1]++; if (arr(x).some((t) => tokOf(t))) byV[v][2]++; });
  console.log(`[${name}] by symbol [n, drawn, retry]: ${JSON.stringify(byV)}`);
  if (missed.length) console.log(`[${name}] never-read ids: ${JSON.stringify(missed.slice(0, 60))}`);
  return missed;
};
rep('all formats (shipped config)', 't'); rep('QR-only config', 'q');
if (r[0].s320 !== undefined) { const a = r.filter((x) => x.s320 === P(x)).length; const d = r.filter((x) => (x.s320 === P(x)) !== (x.t[0] === P(x))).length; console.log(`[320 surface] as drawn ${pct(a, N)}; codes whose outcome differs from 640: ${d}`); }
if (r[0].p) for (const kind of Object.keys(r[0].p)) { const drawn = r.filter((x) => x.p[kind][0] === P(x)).length; const any = r.filter((x) => x.p[kind].some((t) => tokOf(t) === m[x.i].token)).length; const wrong = r.filter((x) => x.p[kind].some((t) => tokOf(t) && tokOf(t) !== m[x.i].token)).length; const orig = r.filter((x) => x.t[0] === P(x)).length; const flip = r.filter((x) => (x.p[kind][0] === P(x)) !== (x.t[0] === P(x))).length; console.log(`[perturb ${kind}] as drawn ${pct(drawn, N)} (pristine same codes ${orig}; outcome differs for ${flip} codes)  retry ${pct(any, N)}  wrong-token ${wrong}`); }
