// marketing/sync/harness/clock-harness.mjs — the node half of card
// clock-offset-on-sync's standalone gate (see clock-run.sh for the substrate
// half and the verdict contract). Sibling of harness.mjs (Card 2) and
// push-harness.mjs (Card 3).
//
// GREEN mode drives the PRODUCTION modules (../clock.js, ../replicas.js,
// ../pull-replication.js) — dynamically imported so the red mode, which
// predates the clock module in git history, never touches them. One clock per
// device: the pull handlers capture offset = serverNow − deviceNow from every
// successful pull response's Date header (spike-proven source, 196 ms recovery
// under a 2-day skew), and every offline expires_at comparison runs adjusted.
//
// RED mode is the deliberately naive inline probe (raw deviceNow < expires_at,
// NO offset — never the production code) that must FAIL its assertions:
//   red-skew — device clock 2 days BEHIND: the dead code is ACCEPTED (§5.1's
//              hole — a rolled-back clock resurrects dead codes); device clock
//              2 days AHEAD ("fast"): the valid code is FALSELY REJECTED.
//
// Direction note (stated in the merge-intent): naive acceptance of a DEAD code
// requires deviceNow < expires_at < serverNow — the device clock reading
// EARLIER than the server. The slate/E-KR4 phrase "clock ≥2 days fast" is read
// as "wrong by ≥2 days"; BOTH skew signs are exercised, red and green.
//
// Discipline inherited from the family: request logs are enumerated (B-216),
// never inferred; the window bounds each request actually sent are parsed out
// of the logged URLs and asserted, not assumed.

import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../.night-crew/qa/spike-supabase/rxdb/spike-env.js';

const MODE = process.argv[2] || 'green';
const env = (k) => {
  const v = process.env[k];
  if (!v) { console.error(`missing env ${k}`); process.exit(2); }
  return v;
};
const JWT = env('C4_JWT');
const DEAD = env('C4_DEAD');
const DEAD_HASH = env('C4_DEAD_HASH');
const VALID = env('C4_VALID');
const VALID_HASH = env('C4_VALID_HASH');

const REST = `http://127.0.0.1:${REST_PORT}`;
const SKEW_MS = 2 * 24 * 3600 * 1000; // the done_when's ≥2 days, exactly 2d
const DAY_MS = 24 * 3600 * 1000;
const OFFSET_TOL_MS = 10_000;     // spike tolerance: Date header is whole-second
const WINDOW_TOL_MS = 15 * 60_000; // window bounds separate by DAYS; 15min is generous

// seed.sql fixture contract (fixed UUIDs; supabase/seed.sql) — same as harness.mjs
const FIX = (n) => `c0000000-0000-4000-8000-00000000000${n}`;
const SEED_LIVE = [FIX(1), FIX(2), FIX(4), FIX(5)]; // expire 2028 (0004 redeemed)

const fail = (msg) => { console.error(`RED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (150s) — a leg never finished'), 150_000);
const sortedIds = async (col) => (await col.find().exec()).map((d) => d.id).sort();
const eqSets = (a, b) => a.length === b.length && a.every((v, i) => v === b[i]);
const windowOf = (url) => {
  const m = /expires_at=gt\.([^&]+)/.exec(url);
  if (!m) fail(`request URL carries no expires_at bound: ${url}`);
  return Date.parse(decodeURIComponent(m[1]));
};
const iso = (ms) => new Date(ms).toISOString();

// ═══════════════════════════════════════════════════════════════════════════
// RED PROBE — the naive deviceNow comparison, inline on purpose.
// ═══════════════════════════════════════════════════════════════════════════

if (MODE === 'red-skew') {
  console.log('# red-skew — naive `deviceNow < expires_at` (no offset) under ±2d clock skew');
  const res = await fetch(
    `${REST}/codes?select=id,expires_at&id=in.(${DEAD},${VALID})&order=id.asc`,
    { headers: { Authorization: `Bearer ${JWT}` } }
  );
  if (res.status !== 200) fail(`pull request answered HTTP ${res.status}`);
  const dateHeader = res.headers.get('date');
  console.log(`  pull response Date header: ${dateHeader} (the serverNow source the naive check IGNORES)`);
  const rows = await res.json();
  const byId = Object.fromEntries(rows.map((r) => [r.id, Date.parse(r.expires_at)]));
  const deadExp = byId[DEAD];
  const validExp = byId[VALID];
  if (!Number.isFinite(deadExp) || !Number.isFinite(validExp)) fail(`seeded rows not readable: ${JSON.stringify(rows)}`);

  const realNow = Date.now();
  const behindNow = realNow - SKEW_MS; // clock set BACK 2 days — §5.1's dangerous direction
  const aheadNow = realNow + SKEW_MS;  // clock 2 days ahead — the "fast watch" direction

  const behindAcceptsDead = behindNow < deadExp;   // naive verdict on the dead code
  const aheadAcceptsValid = aheadNow < validExp;   // naive verdict on the valid code
  console.log(`  dead code  expires ${iso(deadExp)} (expired ~1d ago)`);
  console.log(`  valid code expires ${iso(validExp)} (~1d from now)`);
  console.log(`  deviceNow -2d (${iso(behindNow)}) → naive verdict on DEAD : ${behindAcceptsDead ? 'ACCEPT (wrong — the code is dead)' : 'reject'}`);
  console.log(`  deviceNow +2d (${iso(aheadNow)}) → naive verdict on VALID: ${aheadAcceptsValid ? 'accept' : 'REJECT (wrong — the code is live)'}`);

  // The properties a correct offline expiry check must satisfy — the naive
  // comparison must red them (that failure IS the demonstration).
  if (behindAcceptsDead || !aheadAcceptsValid) {
    fail(
      `naive deviceNow comparison under ±2d skew — ` +
      `dead code ${behindAcceptsDead ? 'ACCEPTED' : 'rejected'} with the clock 2 days behind (§5.1's hole); ` +
      `valid code ${aheadAcceptsValid ? 'accepted' : 'FALSELY REJECTED'} with the clock 2 days ahead`
    );
  }
  console.log('  naive comparison held under both skews (probe unexpectedly green)');
  clearTimeout(hardTimeout);
  process.exit(0);
}

// ═══════════════════════════════════════════════════════════════════════════
// GREEN — the production clock through the production replicas, every clause.
// ═══════════════════════════════════════════════════════════════════════════

if (MODE !== 'green') fail(`unknown mode ${MODE}`);

const { createSyncClock } = await import('../clock.js');
const { startCodesReplica, startOffersReplica, resolveOffers, marketingCollectionSpec } =
  await import('../replicas.js');

addRxPlugin(RxDBDevModePlugin);
async function makeDb() {
  const db = await createRxDatabase({
    name: `c4_green_${Math.random().toString(36).slice(2)}`,
    storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
  });
  await db.addCollections(marketingCollectionSpec());
  return db;
}

const T0 = Date.now(); // real wall clock at harness start — the assertion anchor
const wantCodes = [...SEED_LIVE, DEAD, VALID].sort(); // corrected §5.3 window
const wantOffers = [...SEED_LIVE, VALID].sort();      // corrected live-only window

async function runReplicaPair({ label, deviceNow, offers }) {
  const persisted = [];
  const clock = createSyncClock({ deviceNow, persist: (s) => persisted.push(s) });
  const db = await makeDb();
  const codesLog = [];
  const deps = {
    replicateRxCollection, restUrl: REST, bearer: JWT, fetchImpl: fetch, clock,
  };
  const repCodes = startCodesReplica({
    ...deps, collection: db.codes, stream$: new Subject().asObservable(),
    batchSize: 2, requestLog: codesLog,
  });
  repCodes.error$.subscribe((e) => console.log(`!! [${label}] codes replication error:`, e.message || e));
  await repCodes.awaitInitialReplication();

  let repOffers = null;
  const offersLog = [];
  if (offers) {
    repOffers = startOffersReplica({
      ...deps, collection: db.offers, stream$: new Subject().asObservable(),
      batchSize: 2, requestLog: offersLog,
    });
    repOffers.error$.subscribe((e) => console.log(`!! [${label}] offers replication error:`, e.message || e));
    await repOffers.awaitInitialReplication();
  }
  return { clock, db, codesLog, offersLog, persisted, repCodes, repOffers };
}

// ---------------------------------------------------------------------------
// leg A — FAST clock (+2d, the done_when direction): capture, storage, windows
// ---------------------------------------------------------------------------
console.log('# leg A — device clock 2 days FAST (deviceNow = real + 2d); codes + offers replicas, one clock');
const fastDeviceNow = () => Date.now() + SKEW_MS;
const fast = await runReplicaPair({ label: 'fast', deviceNow: fastDeviceNow, offers: true });

// A1 — the offset was captured from a real pull's Date header, on EVERY pull.
const totalPulls = fast.codesLog.length + fast.offersLog.length;
console.log(`  pulls: ${fast.codesLog.length} codes + ${fast.offersLog.length} offers; clock.captures = ${fast.clock.captures}`);
if (fast.clock.captures < 1) fail('no offset capture happened on a successful pull');
if (fast.clock.captures !== totalPulls) fail(`captures ${fast.clock.captures} != successful pulls ${totalPulls} — "on every successful pull" does not hold`);
const offErr = Math.abs(fast.clock.offsetMs - (-SKEW_MS));
console.log(`  offset = ${fast.clock.offsetMs}ms (expected ~${-SKEW_MS}ms, err ${offErr}ms)`);
if (offErr > OFFSET_TOL_MS) fail(`offset did not recover the +2d skew (err ${offErr}ms > ${OFFSET_TOL_MS}ms)`);

// A2 — the state is handed to the injected persist on every capture (this is
// what Cards 5/6 store beside the checkpoint) and round-trips.
if (fast.persisted.length !== fast.clock.captures) fail(`persist fired ${fast.persisted.length}x for ${fast.clock.captures} captures`);
const lastPersisted = fast.persisted[fast.persisted.length - 1];
if (lastPersisted.offset_ms !== fast.clock.offsetMs) fail('persisted state disagrees with the live clock');
console.log(`  persisted ${fast.persisted.length} states; last = {offset_ms: ${lastPersisted.offset_ms}, captured_at_server: ${iso(lastPersisted.captured_at_server)}}`);

// A3 — window bounds: request #1 pre-capture (unadjusted fast clock ⇒ floor ≈
// real now), later requests corrected (floor ≈ real now − 2d). Enumerated.
console.log(`  codes request windows (${fast.codesLog.length} requests):`);
for (const [i, r] of fast.codesLog.entries()) console.log(`    #${i + 1} expires_at > ${iso(windowOf(r.url))}`);
const w1 = windowOf(fast.codesLog[0].url);
const wLast = windowOf(fast.codesLog[fast.codesLog.length - 1].url);
if (Math.abs(w1 - T0) > WINDOW_TOL_MS) fail(`first (pre-capture) codes window ${iso(w1)} not ≈ real now ${iso(T0)} — the fast clock's unadjusted floor should sit at real now (+2d − 2d)`);
if (Math.abs(wLast - (T0 - 2 * DAY_MS)) > WINDOW_TOL_MS) fail(`last codes window ${iso(wLast)} not ≈ real now − 2d — the capture did not correct the window`);
console.log('  leg A HELD — offset captured on every pull (Date header), persisted, and fed back into the very next request\'s window');

// ---------------------------------------------------------------------------
// leg B — done_when (E-KR4): fast clock, OFFLINE — the expired code is
// rejected, the valid code still resolves (no false rejections).
// ---------------------------------------------------------------------------
console.log('# leg B — done_when: offline expiry under the fast clock');
const codesIdsFast = await sortedIds(fast.db.codes);
const offersIdsFast = await sortedIds(fast.db.offers);
if (!eqSets(codesIdsFast, wantCodes)) fail(`codes replica set mismatch — want [${wantCodes}] got [${codesIdsFast}]`);
if (!eqSets(offersIdsFast, wantOffers)) fail(`offers replica set mismatch — want [${wantOffers}] got [${offersIdsFast}] (an unadjusted fast window would have dropped the valid +1d offer)`);
await fast.repCodes.cancel();
await fast.repOffers.cancel();
console.log('  offline (both replications cancelled)');

const deadDoc = await fast.db.codes.findOne(DEAD).exec();
const validDoc = await fast.db.codes.findOne(VALID).exec();
if (!deadDoc || !validDoc) fail('per-run rows missing from the codes replica');
const naiveDead = fastDeviceNow() < Date.parse(deadDoc.expires_at);
const naiveValid = fastDeviceNow() < Date.parse(validDoc.expires_at);
console.log(`  dead code:  naive=${naiveDead ? 'accept' : 'reject'}  adjusted isExpired=${fast.clock.isExpired(deadDoc.expires_at)}`);
console.log(`  valid code: naive=${naiveValid ? 'accept' : 'REJECT (the false rejection the clock prevents)'}  adjusted isExpired=${fast.clock.isExpired(validDoc.expires_at)}`);
if (!fast.clock.isExpired(deadDoc.expires_at)) fail('done_when broken — the expired code is NOT rejected offline under the fast clock');
if (fast.clock.isExpired(validDoc.expires_at)) fail('false rejection — the valid code reads expired under the fast clock');

const offersAdj = await resolveOffers(fast.db.offers, VALID_HASH, { now: fast.clock.now });
if (offersAdj.length !== 1 || offersAdj[0].code_id !== VALID)
  fail(`valid customer's offer did not resolve offline with the adjusted clock — got ${JSON.stringify(offersAdj)}`);
const offersNaive = await resolveOffers(fast.db.offers, VALID_HASH, { now: fastDeviceNow });
if (offersNaive.length !== 0)
  fail('contrast leg broken — the raw fast clock should falsely drop the +1d offer, but it resolved');
const offersDead = await resolveOffers(fast.db.offers, DEAD_HASH, { now: fast.clock.now });
if (offersDead.length !== 0) fail('the dead code resolved as an offer');
console.log(`  resolveOffers(valid hash, clock.now) → [${offersAdj[0].code_id}]; (raw fast deviceNow) → []; (dead hash) → []`);
console.log('  leg B HELD — clock ≥2d fast: expired code rejected offline, valid code still resolves');

// ---------------------------------------------------------------------------
// leg C — BEHIND clock (−2d, the spike's dangerous direction): the §5.1 hole
// is closed — the naive check accepts the dead code, the adjusted check
// rejects it, same data, same skew.
// ---------------------------------------------------------------------------
console.log('# leg C — device clock 2 days BEHIND (deviceNow = real − 2d); codes replica');
const behindDeviceNow = () => Date.now() - SKEW_MS;
const behind = await runReplicaPair({ label: 'behind', deviceNow: behindDeviceNow, offers: false });
const offErrB = Math.abs(behind.clock.offsetMs - SKEW_MS);
console.log(`  offset = ${behind.clock.offsetMs}ms (expected ~${SKEW_MS}ms, err ${offErrB}ms)`);
if (offErrB > OFFSET_TOL_MS) fail(`offset did not recover the −2d skew (err ${offErrB}ms > ${OFFSET_TOL_MS}ms)`);
console.log(`  codes request windows (${behind.codesLog.length} requests):`);
for (const [i, r] of behind.codesLog.entries()) console.log(`    #${i + 1} expires_at > ${iso(windowOf(r.url))}`);
const wb1 = windowOf(behind.codesLog[0].url);
const wbLast = windowOf(behind.codesLog[behind.codesLog.length - 1].url);
if (Math.abs(wb1 - (T0 - 4 * DAY_MS)) > WINDOW_TOL_MS) fail(`first (pre-capture) codes window ${iso(wb1)} not ≈ real now − 4d`);
if (Math.abs(wbLast - (T0 - 2 * DAY_MS)) > WINDOW_TOL_MS) fail(`last codes window ${iso(wbLast)} not ≈ real now − 2d — the capture did not correct the window`);

const codesIdsBehind = await sortedIds(behind.db.codes);
if (!eqSets(codesIdsBehind, wantCodes)) fail(`codes replica set mismatch (behind) — want [${wantCodes}] got [${codesIdsBehind}]`);
await behind.repCodes.cancel();
console.log('  offline (replication cancelled)');

const deadDocB = await behind.db.codes.findOne(DEAD).exec();
const validDocB = await behind.db.codes.findOne(VALID).exec();
const naiveAcceptsDeadB = behindDeviceNow() < Date.parse(deadDocB.expires_at);
console.log(`  dead code:  naive=${naiveAcceptsDeadB ? 'ACCEPT (the §5.1 hole)' : 'reject'}  adjusted isExpired=${behind.clock.isExpired(deadDocB.expires_at)}`);
if (!naiveAcceptsDeadB) fail('the naive check did not accept the dead code under the behind clock — the defect class is not live on this data, the leg proves nothing');
if (!behind.clock.isExpired(deadDocB.expires_at)) fail('§5.1 hole OPEN — the adjusted check accepted the dead code under the behind clock');
if (behind.clock.isExpired(validDocB.expires_at)) fail('false rejection — the valid code reads expired under the behind clock');
console.log('  leg C HELD — same row, same skew: naive accepts the dead code, the adjusted check rejects it; the valid code still resolves');

// ---------------------------------------------------------------------------
// leg D — the state contract: reload-offline round-trip, headerless capture,
// fail-closed parsing.
// ---------------------------------------------------------------------------
console.log('# leg D — state round-trip + capture edge cases (pure local)');
const rebooted = createSyncClock({ deviceNow: behindDeviceNow, initialState: behind.clock.state() });
if (rebooted.captures !== 0) fail('a rebooted clock should start with zero captures');
if (rebooted.offsetMs !== behind.clock.offsetMs) fail('initialState did not round-trip the offset');
if (!rebooted.isExpired(deadDocB.expires_at)) fail('a device reloaded OFFLINE from persisted state accepted the dead code');
console.log(`  reloaded-offline clock (initialState, 0 captures) still rejects the dead code (offset ${rebooted.offsetMs}ms)`);

const before = rebooted.offsetMs;
const headerless = rebooted.captureFromResponse({ status: 200, headers: { get: () => null } });
if (headerless !== null) fail('a headerless response should not produce a capture');
if (rebooted.offsetMs !== before || rebooted.captures !== 0) fail('a failed capture disturbed the clock state');
console.log('  headerless response → capture returns null, prior offset retained (observable, not silent)');

if (!rebooted.isExpired(undefined) || !rebooted.isExpired('not-a-date'))
  fail('an unreadable expires_at must read as EXPIRED (fail-closed) — it read as valid');
console.log('  unreadable expires_at → expired (fail-closed)');
console.log('  leg D HELD');

// ---------------------------------------------------------------------------
// the enumerated request logs (B-216)
// ---------------------------------------------------------------------------
console.log(`# request log — fast codes (${fast.codesLog.length}), fast offers (${fast.offersLog.length}), behind codes (${behind.codesLog.length})`);
console.log(`# sample pull URL (fast codes #1): ${fast.codesLog[0].url}`);

clearTimeout(hardTimeout);
await fast.db.close();
await behind.db.close();
console.log('ALL LEGS HELD');
process.exit(0);
