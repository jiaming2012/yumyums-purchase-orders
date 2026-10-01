// 01-divert-predicate-diverts-on-unverified-alone.mjs — spike for B-440: the
// SHIPPED push handler diverts to the unverified-landing path on
// `doc.unverified_code` ALONE, so a row {unverified_code:true,
// offline_override:false} is sent to /scan_attempts where the tightened check
// constraint rejects it (HTTP 400) and the handler THROWS — the head-of-line
// poison. This is the red baseline scanner-polish flips (predicate must become
// unverified_code && offline_override). Exit 0 = the defect is reproduced as
// described (the spike proves the premise of the fix); exit 1 = it is not.
import { makePushHandler } from '../../../../marketing/sync/push-replication.js';
const requestLog = [];
const doc = { id: 'att-1', status: 'pending', code_id: 'f'.repeat(64), unverified_code: true, offline_override: false,
  scanned_at: new Date().toISOString(), pos_business_date: '2026-10-01', patched: null,
  async incrementalPatch(p) { this.patched = p; Object.assign(this, p); } };
const attemptsCollection = { findOne: () => ({ exec: async () => doc }) };
const codesCollection = { findOne: () => ({ exec: async () => null }) };
const fetchImpl = async (url, init) => {
  const body = JSON.parse(init.body);
  if (url.endsWith('/scan_attempts') && body.unverified_code && !body.offline_override) {
    return { status: 400, json: async () => ({ code: '23514', message: 'violates check constraint "scan_attempts_names_a_code"' }) };
  }
  return { status: 201, json: async () => ({}) };
};
const push = makePushHandler({ restUrl: 'http://spike', bearer: 't', deviceId: 'dev', fetchImpl, attemptsCollection, codesCollection, requestLog, winnerWaitMs: 10 });
let threw = null;
try { await push([{ newDocumentState: { id: 'att-1', status: 'pending' } }]); } catch (e) { threw = e; }
console.log('# request log kinds:', requestLog.map(r => r.kind).join(', ') || '(none)');
console.log('# handler threw:', threw ? threw.message : 'no');
console.log('# doc after:', JSON.stringify({ status: doc.status, landed: doc.landed ?? null }));
const diverted = requestLog.some(r => r.kind === 'land-unverified');
if (!diverted) { console.log('🛑 RED — the shipped handler did NOT divert on unverified_code alone; B-440 as filed does not reproduce'); process.exit(1); }
if (!threw) { console.log('🛑 RED — the divert happened but the 400 did not throw; the poison shape differs from B-440'); process.exit(1); }
if (doc.status !== 'pending') { console.log('🛑 RED — row left pending was expected (retry-forever); got ' + doc.status); process.exit(1); }
console.log('✅ GREEN — reproduced: divert on unverified_code alone → constraint 400 → throw → row stays pending (head-of-line poison). scanner-polish tightens the predicate to unverified_code && offline_override.');
