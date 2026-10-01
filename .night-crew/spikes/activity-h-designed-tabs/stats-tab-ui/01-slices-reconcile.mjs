// 01-slices-reconcile.mjs — spike: the handoff §5 metric definitions, applied to
// one fixture, make the three slices (by campaign / by channel / by item) sum
// to the overview for every column — scans, signups, redeemed, revenue,
// discount (implied → actual), net — under FIRST-TOUCH attribution; declines
// count in the orphan rate except duplicate_scan; Per $1 is null at zero
// discount. If the slices cannot be made to reconcile on paper, the Stats tab
// would show three totals that disagree. Exit 0 = every assertion held.
import assert from 'node:assert/strict';
const codes = [ // qr_codes
  { short: 'A1', campaign: 'welcome', channel: 'truck_sign', item: null },
  { short: 'A2', campaign: 'welcome', channel: 'flyer', item: null },
  { short: 'B1', campaign: 'wingwed', channel: 'truck_sign', item: 'wings6' },
  { short: 'B2', campaign: 'wingwed', channel: 'instagram', item: 'wings6' },
  { short: 'C1', campaign: 'catering', channel: 'google_ads', item: 'tray' },
];
const campaigns = { welcome: { face: 300 }, wingwed: { face: 200 }, catering: { face: 4000 } };
const scans = [ ['A1',3], ['A2',2], ['B1',4], ['B2',1], ['C1',2] ].flatMap(([s,n]) => Array.from({length:n}, () => ({ short: s })));
const subs = [ // subscribers with first-touch source_short
  { id: 's1', source_short: 'A1' }, { id: 's2', source_short: 'A2' }, { id: 's3', source_short: 'B1' }, { id: 's4', source_short: 'B1' }, { id: 's5', source_short: 'C1' }, { id: 's6', source_short: null },
];
const attempts = [ // accepted redemptions: subscriber, campaign redeemed, order match, decision
  { id: 'r1', sub: 's1', campaign: 'welcome', order: { amount: 1450, discount: 300 } },
  { id: 'r2', sub: 's2', campaign: 'welcome', order: { amount: 900, discount: 250 } },   // actual ≠ implied
  { id: 'r3', sub: 's3', campaign: 'wingwed', order: { amount: 1200, discount: 200 } },
  { id: 'r4', sub: 's4', campaign: 'wingwed', order: null, decision: { reason: 'customer_left' } },  // declined → orphan rate
  { id: 'r5', sub: 's5', campaign: 'catering', order: null, decision: null },                        // open orphan
  { id: 'r6', sub: 's3', campaign: 'wingwed', order: null, decision: { reason: 'duplicate_scan' } }, // declined, excluded
  { id: 'r7', sub: 's6', campaign: 'welcome', order: { amount: 800, discount: 300 } },               // no first-touch → "direct"
];
const codeOf = Object.fromEntries(codes.map(c => [c.short, c]));
const subOf = Object.fromEntries(subs.map(s => [s.id, s]));
const dimKey = (a, dim) => {
  const c = subOf[a.sub]?.source_short ? codeOf[subOf[a.sub].source_short] : null;
  if (dim === 'campaign') return a.campaign;
  if (dim === 'channel') return c ? c.channel : 'direct';
  if (dim === 'item') return c ? (c.item ?? 'any') : 'direct';
};
const money = (list) => {
  // CORRECTION (spike run 1 → red): discount is PER ROW — the matched order's
  // actual discount where there is one, the campaign's face value where there
  // is not — and every total is the sum of those rows. A slice-level basis
  // (actual if fully matched, else implied) made a fully-matched slice disagree
  // with a partly-matched overview by exactly the actual−implied gap (−1100 vs
  // −1150). `basis` is a LABEL (actual | implied | mixed), never a choice of math.
  const redeemed = list.length;
  const matched = list.filter(a => a.order);
  const revenue = matched.reduce((s, a) => s + a.order.amount, 0);
  const discount = list.reduce((s, a) => s + (a.order ? a.order.discount : campaigns[a.campaign].face), 0);
  const implied = list.reduce((s, a) => s + campaigns[a.campaign].face, 0);
  const basis = matched.length === list.length ? 'actual' : matched.length === 0 ? 'implied' : 'mixed';
  return { redeemed, revenue, discount, implied, basis, net: revenue - discount, per_dollar: discount ? Math.round(revenue / discount * 100) / 100 : null };
};
const slice = (dim) => {
  const out = {};
  for (const a of attempts) { const k = dimKey(a, dim); (out[k] ??= []).push(a); }
  const rows = Object.fromEntries(Object.entries(out).map(([k, l]) => [k, money(l)]));
  if (dim !== 'campaign') { // scans/signups attributed by code
    for (const c of codes) {
      const k = dim === 'channel' ? c.channel : (c.item ?? 'any');
      rows[k] ??= money([]);
      rows[k].scans = (rows[k].scans ?? 0) + scans.filter(s => s.short === c.short).length;
      rows[k].signups = (rows[k].signups ?? 0) + subs.filter(s => s.source_short === c.short).length;
    }
  } else {
    for (const c of codes) { rows[c.campaign].scans = (rows[c.campaign].scans ?? 0) + scans.filter(s => s.short === c.short).length; rows[c.campaign].signups = (rows[c.campaign].signups ?? 0) + subs.filter(s => s.source_short === c.short).length; }
  }
  return rows;
};
const sum = (rows, f) => Object.values(rows).reduce((s, r) => s + (r[f] ?? 0), 0);
const overview = { ...money(attempts), scans: scans.length, signups: subs.filter(s => s.source_short).length };
for (const dim of ['campaign', 'channel', 'item']) {
  const rows = slice(dim);
  for (const f of ['scans', 'signups', 'redeemed', 'revenue', 'discount', 'implied', 'net']) assert.equal(sum(rows, f), overview[f], `${dim}.${f} Σ=${sum(rows, f)} overview=${overview[f]}`);
  console.log(`# ${dim}: Σ scans=${sum(rows,'scans')} signups=${sum(rows,'signups')} redeemed=${sum(rows,'redeemed')} revenue=${sum(rows,'revenue')} discount=${sum(rows,'discount')} net=${sum(rows,'net')}`);
}
// the implied figure is still carried per row so the UI can show "implied −$X · actual −$Y" where they differ
console.log(`# overview: discount(per-row)=${overview.discount} implied=${overview.implied} basis=${overview.basis}`);
assert.ok(overview.discount !== overview.implied, 'fixture must exercise an actual≠implied row');
// orphan rate
const declinedCounted = attempts.filter(a => a.decision && a.decision.reason !== 'duplicate_scan').length;
const openOrphans = attempts.filter(a => !a.order && !a.decision).length;
const rate = (declinedCounted + openOrphans) / attempts.length;
console.log(`# orphan rate = (${openOrphans} open + ${declinedCounted} declined-counted) / ${attempts.length} = ${(rate*100).toFixed(1)}%  (duplicate_scan excluded)`);
assert.equal(declinedCounted, 1); assert.equal(openOrphans, 1);
assert.equal(money([]).per_dollar, null, 'per_dollar is null at zero discount');
console.log('✅ GREEN — slices reconcile to the overview under first-touch attribution; decision rule (correction): discount is summed PER ROW — actual where the order is matched, face value where it is not — on every slice and the overview alike; basis is a label (actual/implied/mixed)');
