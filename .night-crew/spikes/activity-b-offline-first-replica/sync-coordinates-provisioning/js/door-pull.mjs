// door-pull.mjs — the auth matrix + the three shipped replicas, through the
// real HQ door. Everything under test is shipped code:
//   * backend: auth.Middleware + sync.ProxyHandler (running in the real server
//     this script talks to — nothing mocked on that side)
//   * marketing/sync/replicas.js — startCodesReplica / startOffersReplica /
//     startCampaignsReplica / createCampaignPolicySource, scan-page order
//
// The only things written here are the cookie-attaching fetchImpl (Node has no
// cookie jar; a real browser's same-origin fetch sends the cookie natively —
// the card's Playwright e2e exercises that half) and the assertions.
//
// argv: <hqOrigin> <cookieValue> <validBridgeToken> <restDirect>
// exit 0 all measurements agreed.

import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import {
  startCodesReplica, startOffersReplica, startCampaignsReplica,
  createCampaignPolicySource, marketingCollectionSpec,
} from '../../../../../marketing/sync/replicas.js';
import { createSyncClock } from '../../../../../marketing/sync/clock.js';

const [ORIGIN, COOKIE, VALID_TOKEN, REST_DIRECT] = process.argv.slice(2);
if (!ORIGIN || !COOKIE || !VALID_TOKEN || !REST_DIRECT) {
  console.error('usage: door-pull.mjs <hqOrigin> <cookie> <validToken> <restDirect>');
  process.exit(2);
}

const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (120s)'), 120_000);

const DOOR = `${ORIGIN}/sync/rest`;
const GARBAGE = 'not-a-jwt-the-door-must-not-care';
const probe = `/campaigns?select=id&limit=1`;

// ---------------------------------------------------------------------------
// The auth matrix — a set, not a sample.
// ---------------------------------------------------------------------------
console.log('── auth matrix (each row measured, statuses enumerated) ──');
const rows = [
  ['(a) cookie + garbage bearer   ', `${DOOR}${probe}`,
    { Cookie: `hq_session=${COOKIE}`, Authorization: `Bearer ${GARBAGE}` }, 200],
  ['(b) no cookie + VALID bearer  ', `${DOOR}${probe}`,
    { Authorization: `Bearer ${VALID_TOKEN}` }, 401],
  ['(c) no cookie + no bearer     ', `${DOOR}${probe}`, {}, 401],
  ['(d) garbage bearer DIRECT     ', `${REST_DIRECT}${probe}`,
    { Authorization: `Bearer ${GARBAGE}` }, null], // expectation: anything BUT 200
];
for (const [label, url, headers, want] of rows) {
  const res = await fetch(url, { headers: { Accept: 'application/json', ...headers } });
  const body = await res.text();
  console.log(`  ${label} → HTTP ${res.status}  ${body.slice(0, 80).replace(/\n/g, ' ')}`);
  if (want === null) {
    if (res.status === 200) fail(`row (d): PostgREST accepted a garbage bearer directly — then row (a) proves nothing about substitution`);
  } else if (res.status !== want) {
    fail(`row ${label.trim()} expected HTTP ${want}, measured ${res.status}`);
  }
  if (want === 200) {
    const parsed = JSON.parse(body);
    if (!Array.isArray(parsed) || parsed.length < 1) {
      fail('row (a) answered 200 but ZERO rows — the silent-empty failure mode (RLS filtered the authenticated pull to nothing)');
    }
  }
}
console.log('  → the cookie is the credential; the client bearer is provably inert at the door.');

// ---------------------------------------------------------------------------
// The three shipped replicas, scan-page order, through the door.
// ---------------------------------------------------------------------------
console.log('\n── startSync shape: three shipped replicas + policy source, through the door ──');
addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `spike_door_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections(marketingCollectionSpec());

const clock = createSyncClock({});
const cookieFetch = (url, opts = {}) => fetch(url, {
  ...opts,
  headers: { ...(opts.headers || {}), Cookie: `hq_session=${COOKIE}` },
});
const deps = (collection, replicationIdentifier) => ({
  replicateRxCollection,
  collection,
  restUrl: DOOR,               // the coordinate the card will write
  bearer: GARBAGE,             // deliberately garbage — proven inert above
  fetchImpl: cookieFetch,
  stream$: new Subject(),
  clock,
  replicationIdentifier,
});

// scan-page.js order: policy source exists first; campaigns starts and
// attaches adjacently (build-fact 1 of the sibling ledger); then the rest.
const policySource = createCampaignPolicySource(db.campaigns);
const campaignsRep = startCampaignsReplica(deps(db.campaigns, 'marketing-campaigns-pull'));
policySource.attach(campaignsRep);
const codesRep = startCodesReplica(deps(db.codes, 'marketing-codes-pull'));
const offersRep = startOffersReplica(deps(db.offers, 'marketing-offers-pull'));

await Promise.all([
  codesRep.awaitInitialReplication(),
  offersRep.awaitInitialReplication(),
  campaignsRep.awaitInitialReplication(),
]);
await new Promise((r) => setTimeout(r, 400)); // policy source settle tick + margin

const nCodes = (await db.codes.find().exec()).length;
const nOffers = (await db.offers.find().exec()).length;
const nCampaigns = (await db.campaigns.find().exec()).length;
console.log(`  codes replica     : ${nCodes} rows`);
console.log(`  offers replica    : ${nOffers} rows`);
console.log(`  campaigns replica : ${nCampaigns} rows`);
console.log(`  policy source     : attached=${policySource.attached()} size=${policySource.size()} unresolved=${policySource.unresolved()} lastError=${policySource.lastError()}`);
console.log(`  clock captures    : ${clock.captures} (a successful pull calibrated §5.1's clock through the door)`);

if (nCodes < 1) fail('codes replica delivered zero rows through the door');
if (nOffers < 1) fail('offers replica delivered zero rows through the door');
if (nCampaigns < 1) fail('campaigns replica delivered zero rows through the door');
if (!policySource.attached()) fail('policy source did not attach — done_when clause 2 premise');
if (policySource.size() < 1) fail('policy source Map is empty after initial replication');
if (policySource.unresolved() !== false) fail(`policy source unresolved()=${policySource.unresolved()} after a clean initial replication`);
if (clock.captures < 1) fail('no clock capture — the Date header did not survive the proxy hop');

console.log('  → done_when clause 2\'s premise holds against the real source: attached, populated, resolved.');

await db.close();
clearTimeout(hardTimeout);
process.exit(0);
