// clock-offset.mjs — the three legs of the skewed-clock spike. Pure Node, no
// RxDB: the offline expiry comparison is plain local arithmetic, which is the
// point — this is exactly what the offline path runs (§5.1).
//
// argv: <restOrigin> <deviceJwt> <codeId>
// exit 0 all legs held; exit 1 a leg failed; exit 2 could not run.

const [restOrigin, jwt, codeId] = process.argv.slice(2);
if (!restOrigin || !jwt || !codeId) {
  console.error('usage: clock-offset.mjs <restOrigin> <jwt> <codeId>');
  process.exit(2);
}

const SKEW_MS = 2 * 24 * 3600 * 1000; // device clock 2 days SLOW — §5.1's dangerous direction
const fail = (msg) => { console.error(`RED: ${msg}`); process.exit(1); };

// ---------------------------------------------------------------------------
// Leg (a) — serverNow from the pull response the sync already makes.
// This is the SAME request shape the pull handler uses; no extra endpoint.
// ---------------------------------------------------------------------------
const t0 = Date.now();
const res = await fetch(
  `${restOrigin}/codes?select=id,expires_at&id=eq.${codeId}`,
  { headers: { Authorization: `Bearer ${jwt}` } }
).catch((e) => { console.error(`could not reach PostgREST: ${e.message}`); process.exit(2); });
const rtt = Date.now() - t0;

if (res.status !== 200) fail(`pull request answered HTTP ${res.status}, not 200`);
const dateHeader = res.headers.get('date');
if (!dateHeader) fail('PostgREST response carries NO Date header — the card needs a now() RPC instead');
const serverNow = Date.parse(dateHeader);
if (Number.isNaN(serverNow)) fail(`Date header not parseable: ${dateHeader}`);

const rows = await res.json();
if (!Array.isArray(rows) || rows.length !== 1) fail(`expected exactly the seeded row, got ${JSON.stringify(rows)}`);
const expiresAt = Date.parse(rows[0].expires_at);
if (Number.isNaN(expiresAt)) fail(`expires_at not parseable: ${rows[0].expires_at}`);

// The skewed device clock, injected: deviceNow() = real now - 2 days.
const realNow = Date.now();
const deviceNow = realNow - SKEW_MS;

// offset captured "on successful sync" (§5.1): serverNow - deviceNow.
const offset = serverNow - deviceNow;
const recoveredSkewErr = Math.abs(offset - SKEW_MS);

console.log(`# leg (a) — serverNow signal`);
console.log(`  Date header      : ${dateHeader}`);
console.log(`  parsed serverNow : ${new Date(serverNow).toISOString()} (rtt ${rtt}ms)`);
console.log(`  skewed deviceNow : ${new Date(deviceNow).toISOString()} (injected -2d)`);
console.log(`  computed offset  : ${offset}ms (expected ~${SKEW_MS}ms, err ${recoveredSkewErr}ms)`);
// Date header is whole-second; allow generous slack for rtt + rounding.
if (recoveredSkewErr > 10_000) fail(`offset did not recover the injected skew (err ${recoveredSkewErr}ms > 10s)`);
console.log('  leg (a) HELD — the pull response Date header recovers the skew to <10s');

// ---------------------------------------------------------------------------
// Leg (b) — RED ANALOG: the naive check under the slow clock ACCEPTS the dead
// code. Without this the green below is unfalsifiable.
// ---------------------------------------------------------------------------
console.log(`# leg (b) — naive check (deviceNow < expires_at), code expired ${new Date(expiresAt).toISOString()}`);
const naiveAccepts = deviceNow < expiresAt;
console.log(`  naive verdict    : ${naiveAccepts ? 'ACCEPT (wrong — code is dead)' : 'reject'}`);
if (!naiveAccepts) fail('the naive check did NOT accept the dead code — the red analog is broken, the defect class was not demonstrated');
console.log('  leg (b) HELD — defect class demonstrated: slow clock + naive check = dead code accepted');

// ---------------------------------------------------------------------------
// Leg (c) — the §5.1 fix: offset-adjusted comparison rejects.
// ---------------------------------------------------------------------------
console.log(`# leg (c) — adjusted check (deviceNow + offset < expires_at)`);
const adjustedAccepts = deviceNow + offset < expiresAt;
console.log(`  adjusted verdict : ${adjustedAccepts ? 'ACCEPT' : 'REJECT (correct)'}`);
if (adjustedAccepts) fail('the offset-adjusted check still accepted the dead code');
console.log('  leg (c) HELD — offset-adjusted comparison rejects the dead code under the same skew');

process.exit(0);
