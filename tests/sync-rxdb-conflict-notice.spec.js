// tests/sync-rxdb-conflict-notice.spec.js — the MODEL contract for the
// overwritten-answers UI.
//
// Card `sync-rxdb-conflict-notice-ui` (overnight-20260801, C2).
//
// ===========================================================================
// WHY THIS SUITE NEEDS NO BROWSER.
// ===========================================================================
// The feature's substance is arithmetic. Three surfaces print a number, the
// counting rule has eight clauses, and both amendments exist because two of
// those numbers were conflated — so the counting lives in an import-free module
// and is proven by calling it with plain records. The SCREENS are proven
// separately, and by screenshot, in tests/states-sync-rxdb-conflict-notice.spec.js.
//
// ===========================================================================
// ANTI-VACUOUS DISCIPLINE (B-22/B-23/B-24).
// ===========================================================================
// A guard printing PASS is not evidence until its subject set is shown
// non-empty. Every data-driven block below asserts its fixture's exact size
// first, so emptying a fixture reds the suite rather than silently asserting
// nothing.

const { test, expect } = require('@playwright/test');
const path = require('path');
const { pathToFileURL } = require('url');

const REPO_ROOT = path.resolve(__dirname, '..');
const MODULE_PATH = path.join(REPO_ROOT, 'sync-rxdb', 'conflict-notice.js');

// Loaded INSIDE each test: a top-level await that throws makes the whole file
// fail to load, and a spec reporting zero executed tests is this repo's
// signature silent failure (B-09, B-14, B-16).
async function load() {
  return import(pathToFileURL(MODULE_PATH).href);
}

// ---------------------------------------------------------------------------
// Fixtures. `overwritten_at` is written with an explicit -04:00 offset so the
// clock rendering is deterministic wherever the suite runs.
// ---------------------------------------------------------------------------

const DOC_A = 'sub-9f31c4b2-0000-0000-0000-000000000001';
const DOC_B = 'sub-2b70ae00-0000-0000-0000-000000000002';

function rec(over) {
  return Object.assign({
    id: 'r-1',
    doc_id: DOC_A,
    collection: 'responses',
    submission_id: DOC_A,
    field_id: 'fld-cooler',
    field_label: 'Walk-in cooler temp',
    field_type: 'temperature',
    display_unit: '°F',
    field_removed: false,
    checklist_name: 'Opening — Truck A',
    checklist_date: 'Mon Jul 27',
    discarded_value: 38,
    current_value: 41,
    overwritten_by: 'u-dana',
    overwritten_by_name: 'Dana M.',
    overwritten_at: '2026-07-27T18:12:00-04:00',
    status: 'open',
    undone: false,
    failure: null,
  }, over);
}

// The success plate's data: two answers on one checklist, one already restored.
const SUCCESS = [
  rec({ id: 'a', field_id: 'fld-cooler' }),
  rec({
    id: 'b',
    field_id: 'fld-sanitizer',
    field_label: 'Sanitizer concentration',
    field_type: 'number',
    display_unit: 'ppm',
    discarded_value: 200,
    current_value: 150,
    overwritten_at: '2026-07-27T18:13:00-04:00',
    status: 'restored',
  }),
];

const NOW = '2026-07-28T09:00:00-04:00';

// ---------------------------------------------------------------------------
// A-3 (decision 95) — the frozen label, and its dependency on R-C.
// ---------------------------------------------------------------------------

// `template_snapshot` is json.Marshal of the whole template
// (backend/internal/workflow/repository.go:695) and `Field.Label` is on it
// (model.go:44-57). The shape below is that structure.
const SNAPSHOT = {
  id: 'tpl-1',
  name: 'Opening — Truck A',
  sections: [
    {
      id: 'sec-1',
      title: 'Cold holding',
      fields: [
        { id: 'fld-cooler', type: 'temperature', label: 'Walk-in cooler temp' },
        {
          id: 'fld-prep',
          type: 'temperature',
          label: 'Prep sink temperature',
          sub_steps: [
            { id: 'fld-prep-a', type: 'checkbox', label: 'Sanitizer added' },
          ],
        },
      ],
    },
  ],
};

// 🛑 B1's recorded-not-fixed item R-C, promoted to a DEPENDENCY by A-3:
// `checklists.template_snapshot` is declared `{type:'object'}` with NO nested
// `properties`, the committed vendor bundle ships no dev-mode or validation
// plugin, and NOTHING REJECTS A MALFORMED VALUE. A-3 makes the snapshot's shape
// load-bearing for the UI, so every one of these has to return null rather than
// throw, recurse forever, or render nothing.
const MALFORMED = [
  ['undefined', undefined],
  ['null', null],
  ['a string', 'not a snapshot'],
  ['a number', 42],
  ['an array', [{ id: 'fld-cooler', label: 'nope' }]],
  ['no sections key', { id: 'tpl' }],
  ['sections is a string', { sections: 'oops' }],
  ['sections is an object', { sections: { 0: { fields: [] } } }],
  ['a section is null', { sections: [null] }],
  ['a section is a string', { sections: ['oops'] }],
  ['fields is missing', { sections: [{ id: 's' }] }],
  ['fields is a number', { sections: [{ fields: 7 }] }],
  ['a field is null', { sections: [{ fields: [null] }] }],
  ['a field has no id', { sections: [{ fields: [{ label: 'orphan' }] }] }],
  ['the label is a number', { sections: [{ fields: [{ id: 'fld-cooler', label: 7 }] }] }],
  ['the label is empty', { sections: [{ fields: [{ id: 'fld-cooler', label: '   ' }] }] }],
  ['the label is an object', { sections: [{ fields: [{ id: 'fld-cooler', label: { en: 'x' } }] }] }],
  ['sub_steps is a string', { sections: [{ fields: [{ id: 'x', sub_steps: 'oops' }] }] }],
];

test.describe('A-3 — the frozen label out of template_snapshot (decision 95)', () => {
  test('the malformed-snapshot fixture is non-empty and covers every level', async () => {
    // B-22/B-23/B-24: this table IS the subject set of the totality test below.
    // Emptying it would make that test pass having asserted nothing.
    expect(MALFORMED.length).toBe(18);
    expect(MALFORMED.map((m) => m[0])).toContain('a field has no id');
    expect(MALFORMED.map((m) => m[0])).toContain('sections is a string');
  });

  test('reads a top-level field label', async () => {
    const { fieldLabelFromSnapshot } = await load();
    expect(fieldLabelFromSnapshot(SNAPSHOT, 'fld-cooler')).toBe('Walk-in cooler temp');
    expect(fieldLabelFromSnapshot(SNAPSHOT, 'fld-prep')).toBe('Prep sink temperature');
  });

  test('reads a label nested in sub_steps — Field.SubSteps is a real nested list', async () => {
    const { fieldLabelFromSnapshot } = await load();
    expect(fieldLabelFromSnapshot(SNAPSHOT, 'fld-prep-a')).toBe('Sanitizer added');
  });

  test('returns null — never throws — for an id the snapshot does not carry', async () => {
    const { fieldLabelFromSnapshot } = await load();
    expect(fieldLabelFromSnapshot(SNAPSHOT, 'fld-nope')).toBeNull();
    expect(fieldLabelFromSnapshot(SNAPSHOT, '')).toBeNull();
    expect(fieldLabelFromSnapshot(SNAPSHOT, null)).toBeNull();
  });

  for (const [name, snap] of MALFORMED) {
    test(`malformed snapshot (${name}) returns null and does not throw`, async () => {
      const { fieldLabelFromSnapshot, findFieldInSnapshot } = await load();
      // The contract is TOTALITY: no throw, and no label. `findFieldInSnapshot`
      // may legitimately FIND the field in the three cases where the structure
      // is sound and only the label is junk — that is exactly the branch that
      // must degrade to the raw field id rather than rendering `7` or
      // `[object Object]` as a question title.
      expect(() => fieldLabelFromSnapshot(snap, 'fld-cooler')).not.toThrow();
      expect(() => findFieldInSnapshot(snap, 'fld-cooler')).not.toThrow();
      expect(fieldLabelFromSnapshot(snap, 'fld-cooler')).toBeNull();
    });
  }

  test('a CYCLIC snapshot terminates rather than blowing the stack', async () => {
    const { fieldLabelFromSnapshot } = await load();
    const field = { id: 'fld-loop', label: 'Loop' };
    field.sub_steps = [field];
    const snap = { sections: [{ fields: [field] }] };
    expect(fieldLabelFromSnapshot(snap, 'fld-loop')).toBe('Loop');
    expect(fieldLabelFromSnapshot(snap, 'fld-missing')).toBeNull();
  });

  test('a THROWING accessor is a malformed snapshot, not a crash', async () => {
    const { fieldLabelFromSnapshot } = await load();
    const bomb = {};
    Object.defineProperty(bomb, 'id', { get() { throw new Error('boom'); }, enumerable: true });
    const snap = { sections: [{ fields: [bomb, { id: 'fld-cooler', label: 'Walk-in cooler temp' }] }] };
    expect(fieldLabelFromSnapshot(snap, 'fld-cooler')).toBe('Walk-in cooler temp');
  });

  test('a row falls back to the RAW FIELD ID when the snapshot carries no label', async () => {
    const { buildSheetModel } = await load();
    // No snapshot, and the record's own frozen label absent too — A-3's fallback
    // ladder bottoms out at the raw id, rendered exactly as revision 2 drew it.
    const m = buildSheetModel(
      [rec({ id: 'z', field_id: 'fld_prep_sink_temp', field_label: null, field_removed: true, current_value: undefined })],
      { now: NOW },
    );
    const row = m.groups[0].rows[0];
    expect(row.kind).toBe('removed');
    expect(row.labelSource).toBe('fallback');
    expect(row.label).toBe('fld_prep_sink_temp');
  });

  test('a removed row PREFERS the frozen label, and is read-only', async () => {
    const { buildSheetModel } = await load();
    const m = buildSheetModel(
      [rec({
        id: 'z',
        field_id: 'fld-prep',
        field_label: 'Prep sink temperature',
        field_removed: true,
        discarded_value: 72,
      })],
      { now: NOW, snapshots: { [DOC_A]: SNAPSHOT } },
    );
    const row = m.groups[0].rows[0];
    expect(row.kind).toBe('removed');
    expect(row.labelSource).toBe('snapshot');
    expect(row.label).toBe('Prep sink temperature');
    // Read-only: never a batch target, so `Restore all N` can never write it.
    expect(row.isBatchTarget).toBe(false);
  });

  test('counting follows READING A — a removed row is in the chip base, not the +N', async () => {
    // Decision 95's recorded consequence. The headline counts what was TAKEN
    // from the crew member; the `+N` line keeps meaning only "we couldn't
    // identify". Pooling a perfectly-identified removed question with a genuine
    // unknown was the worse outcome.
    const { buildSheetModel, bannerModel } = await load();
    const m = buildSheetModel(
      [rec({ id: 'a' }), rec({ id: 'z', field_id: 'fld-prep', field_label: 'Prep sink temperature', field_removed: true, discarded_value: 72 })],
      { now: NOW },
    );
    const g = m.groups[0];
    expect(g.chipBase).toBe(2);
    expect(g.chipPlus).toBe(0);
    expect(g.rows.length).toBe(2);
    expect(bannerModel(m).headline).toBe('2 answers were overwritten');
    expect(bannerModel(m).unid).toBeNull();
    // ...and the accepted mismatch decision 95 signed off: 2 in the chip, 1
    // restorable. A-3 makes it legible on screen rather than arithmetic.
    expect(g.batchCount).toBe(1);
  });
});

// ---------------------------------------------------------------------------
// The counting rule, clause by clause.
// ---------------------------------------------------------------------------

test.describe('the counting rule (UI-SPEC rules 1-8)', () => {
  test('rule 1/2 — headline counts answer rows; the chip is the same figure per group', async () => {
    const { buildSheetModel, bannerModel } = await load();
    const m = buildSheetModel(SUCCESS, { now: NOW });
    expect(m.groups).toHaveLength(1);
    expect(m.totals.answers).toBe(2);
    expect(m.groups[0].chipBase).toBe(2);
    expect(bannerModel(m).headline).toBe('2 answers were overwritten');
  });

  test('rule 3 — what was DONE to a row never changes the headline', async () => {
    const { buildSheetModel, bannerModel } = await load();
    const untouched = buildSheetModel(SUCCESS.map((r) => ({ ...r, status: 'open' })), { now: NOW });
    const worked = buildSheetModel(
      [{ ...SUCCESS[0], status: 'kept' }, { ...SUCCESS[1], status: 'restored' }],
      { now: NOW },
    );
    expect(bannerModel(untouched).headline).toBe(bannerModel(worked).headline);
    expect(worked.groups[0].chipBase).toBe(2);
    expect(worked.groups[0].rows).toHaveLength(2);
  });

  test('rule 4 — unidentifiable rows are counted SEPARATELY, never folded in', async () => {
    const { buildSheetModel, bannerModel } = await load();
    const m = buildSheetModel(
      [rec({ id: 'a' }), rec({ id: 'u', field_id: 'fld-x', discarded_value: null })],
      { now: NOW },
    );
    const b = bannerModel(m);
    expect(b.headline).toBe('1 answer was overwritten');
    expect(b.open).toBe('1 still to review');
    expect(b.unid).toBe("+ 1 change we couldn't identify");
    expect(m.groups[0].chipBase).toBe(1);
    expect(m.groups[0].chipPlus).toBe(1);
    // chip base + N = rows drawn
    expect(m.groups[0].chipBase + m.groups[0].chipPlus).toBe(m.groups[0].rows.length);
  });

  test('rule 5 — the record id is document + field, so a retry REPLACES a row', async () => {
    const { recordIdFor, createMemoryConflictStore } = await load();
    expect(recordIdFor('doc-1', 'fld-1')).toBe('doc-1::fld-1');
    expect(recordIdFor('doc-1', 'fld-1')).not.toBe(recordIdFor('doc-1', 'fld-2'));
    const store = createMemoryConflictStore([]);
    await store.upsert({ id: recordIdFor('doc-1', 'fld-1'), discarded_value: 1 });
    await store.upsert({ id: recordIdFor('doc-1', 'fld-1'), discarded_value: 2 });
    const all = await store.all();
    expect(all).toHaveLength(1);
    expect(all[0].discarded_value).toBe(2);
  });

  test('rule 6 — reviewed is restored/kept; untouched, FAILED and in-flight are not', async () => {
    const { buildSheetModel, bannerModel } = await load();
    const cases = [
      ['open', false], ['restoring', false], ['failed', false],
      ['restored', true], ['kept', true],
    ];
    expect(cases).toHaveLength(5); // the subject set, pinned
    for (const [status, handled] of cases) {
      const m = buildSheetModel([rec({ id: 'a', status })], { now: NOW });
      expect(m.totals.handled, `status=${status}`).toBe(handled ? 1 : 0);
      expect(m.totals.still, `status=${status}`).toBe(handled ? 0 : 1);
    }
    // 🛑 A-1 rule 3's hard case, drawn on the error plate: BOTH a failed restore
    // and an in-flight restore are still to review. If either counted as
    // handled, the definition would be contradicted by its own screen.
    const err = buildSheetModel(
      [rec({ id: 'a', status: 'failed', failure: 'offline' }), rec({ id: 'b', field_id: 'fld-sink', status: 'restoring' })],
      { now: NOW },
    );
    expect(bannerModel(err).open).toBe('2 still to review');
  });

  test('rule 6 wording — nothing handled / some handled / all handled', async () => {
    const { buildSheetModel, bannerModel } = await load();
    const none = bannerModel(buildSheetModel(SUCCESS.map((r) => ({ ...r, status: 'open' })), { now: NOW }));
    expect(none.open).toBe('2 still to review');
    const some = bannerModel(buildSheetModel(SUCCESS, { now: NOW }));
    expect(some.open).toEqual({ lead: '1 still to review', hand: '· 1 handled' });
    const all = bannerModel(buildSheetModel(SUCCESS.map((r) => ({ ...r, status: 'kept' })), { now: NOW }));
    expect(all.open).toBe('All 2 reviewed');
  });

  test('rule 7 — Restore all N is the still-to-review figure, never the chip base', async () => {
    const { buildSheetModel } = await load();
    const m = buildSheetModel(
      [
        rec({ id: 'a' }),
        rec({ id: 'b', field_id: 'f2', status: 'kept' }),
        rec({ id: 'c', field_id: 'f3', status: 'restored' }),
        rec({ id: 'd', field_id: 'f4' }),
      ],
      { now: NOW },
    );
    expect(m.groups[0].chipBase).toBe(4);
    // HOW IT FAILS: if this were 4, one tap would re-write the row the crew
    // member deliberately KEPT — silently reversing their own decision, which is
    // the opposite of what an override that "states what it destroys" is for.
    expect(m.groups[0].batchCount).toBe(2);
    expect(m.groups[0].batchIds.sort()).toEqual(['a', 'd']);
  });

  test('rule 8 — collapse triggers on >1 group OR >2 rows, and applies sheet-wide', async () => {
    const { buildSheetModel } = await load();
    const twoRowsOneGroup = buildSheetModel(SUCCESS, { now: NOW });
    expect(twoRowsOneGroup.collapsed).toBe(false);
    const threeRows = buildSheetModel([...SUCCESS, rec({ id: 'c', field_id: 'f3' })], { now: NOW });
    expect(threeRows.collapsed).toBe(true);
    const twoGroups = buildSheetModel(
      [rec({ id: 'a' }), rec({ id: 'b', doc_id: DOC_B, submission_id: DOC_B, checklist_name: 'Closing — Truck A' })],
      { now: NOW },
    );
    expect(twoGroups.collapsed).toBe(true);
    expect(twoGroups.groups).toHaveLength(2);
  });
});

// ---------------------------------------------------------------------------
// Decision 96 — retention, from ONE named constant.
// ---------------------------------------------------------------------------

test.describe('retention (decision 96)', () => {
  test('the window is READ from sync-schema, not restated', async () => {
    const mod = await load();
    const schema = await import(
      pathToFileURL(path.join(REPO_ROOT, 'sync-schema', 'collections.js')).href
    );
    expect(mod.RETENTION_DAYS).toBe(schema.CONFLICT_RECORD_RETENTION_DAYS);
    // Identity, not equality of two literals: changing the constant in
    // sync-schema must move this, or "one named constant" is not true.
    expect(typeof mod.RETENTION_DAYS).toBe('number');
  });

  test('records outside the window are swept and are NOT in any count', async () => {
    const { buildSheetModel, RETENTION_DAYS, sweepExpired, createMemoryConflictStore } = await load();
    const old = new Date(Date.parse(NOW) - (RETENTION_DAYS + 1) * 86400000).toISOString();
    const fresh = rec({ id: 'a' });
    const stale = rec({ id: 'old', field_id: 'f-old', overwritten_at: old });
    const m = buildSheetModel([fresh, stale], { now: NOW });
    expect(m.totals.answers).toBe(1);

    const store = createMemoryConflictStore([fresh, stale]);
    expect(await sweepExpired(store, { now: NOW })).toBe(1);
    expect(await store.all()).toHaveLength(1);
  });

  test('an unreadable timestamp is KEPT, not silently swept', async () => {
    const { buildSheetModel } = await load();
    const m = buildSheetModel([rec({ id: 'a', overwritten_at: 'not-a-date' })], { now: NOW });
    expect(m.totals.answers).toBe(1);
  });
});

// ---------------------------------------------------------------------------
// Decision 97 — the 10-group cap.
// ---------------------------------------------------------------------------

test.describe('the cap (decision 97)', () => {
  const many = () => {
    const out = [];
    for (let i = 0; i < 13; i++) {
      const doc = `sub-${String(i).padStart(4, '0')}-0000-0000-0000-00000000000${i % 10}`;
      out.push(rec({
        id: `g${i}`,
        doc_id: doc,
        submission_id: doc,
        checklist_name: `Checklist ${i}`,
        overwritten_at: `2026-07-2${(i % 7) + 1}T18:12:00-04:00`,
      }));
    }
    return out;
  };

  test('caps at MAX_GROUPS groups', async () => {
    const { buildSheetModel, MAX_GROUPS } = await load();
    expect(MAX_GROUPS).toBe(10);
    const src = many();
    expect(src).toHaveLength(13); // subject set, pinned
    const m = buildSheetModel(src, { now: '2026-08-01T09:00:00-04:00' });
    expect(m.groups).toHaveLength(10);
    expect(m.hiddenGroups).toBe(3);
    expect(m.hiddenAnswers).toBe(3);
  });

  test('rows below the line are NOT dropped and the banner reports the TRUE total', async () => {
    const { buildSheetModel, bannerModel } = await load();
    const m = buildSheetModel(many(), { now: '2026-08-01T09:00:00-04:00' });
    // HOW IT FAILS: a cap that also capped the count would under-report the loss
    // on the one night the feature matters. 13, not 10.
    expect(m.totals.answers).toBe(13);
    expect(bannerModel(m).headline).toBe('13 answers were overwritten');
    expect(m.totals.checklists).toBe(13);
  });
});

// ---------------------------------------------------------------------------
// Values and attribution.
// ---------------------------------------------------------------------------

test.describe('value + attribution rendering', () => {
  test('values render as a crew member reads them, and are NEVER truncated', async () => {
    const { formatValue } = await load();
    const long = 'RESTOCK-2026-07-27T18:44:02Z-TRUCKA-OPENING-9f31c4b2e7d5a1c8f0e3b6d94a2c7e15';
    expect(formatValue(long)).toBe(long); // no ellipsis, no slice
    // NBSP, as the mockup draws `41&nbsp;°F` — a temperature must not wrap
    // between the figure and its unit.
    expect(formatValue(38, { unit: '°F' })).toBe('38\u00a0°F');
    expect(formatValue(true)).toBe('Yes');
    expect(formatValue(false)).toBe('No');
    expect(formatValue('yes')).toBe('Yes');
    expect(formatValue(null)).toBe('—');
    expect(formatValue('')).toBe('—');
    expect(formatValue('Emptied', { type: 'text' })).toBe('“Emptied”');
    // HQ bundles a value with its metadata — the ANSWER is the `_v`.
    expect(formatValue({ _v: 39, _fail_note: { note: 'x' } })).toBe('39');
  });

  // ─────────────────────────────────────────────────────────────────────────
  // 🛑 G6 FINDING F-2 — `formatValue` must be TOTAL, for the same reason
  // `findFieldInSnapshot` is.
  //
  // The `{_v}` / `{value}` unwrap recursed with no depth bound and read the
  // property with no `try`. `formatValue` runs on EVERY conflict row, on BOTH
  // `discarded_value` and `current_value` — the same "must never take the sheet
  // down" commitment `findFieldInSnapshot` was built to honour, on the same hot
  // path — and the module already wraps `JSON.stringify` in `try/catch` for
  // exactly this class of input, so the omission was an oversight.
  //
  // 🛑 REACHABILITY, MEASURED RATHER THAN ASSERTED. G6's finding says a
  // 30 000-deep `_v` chain "is JSON-representable, so it survives a store
  // round-trip". On this runtime (node 20.20 / V8) it does NOT: binary-searched
  // on the box that ran this fix,
  //
  //     structuredClone (the IndexedDB serialiser)  overflows past ~1 954 deep
  //     JSON.stringify                              overflows past ~4 174 deep
  //     the OLD unbounded formatValue               overflowed past ~8 397 deep
  //
  // — so a chain deep enough to break the renderer could not have come back out
  // of the local store on this engine. The bound is kept anyway, and the reason
  // is recorded rather than dressed up: those three numbers are stack-budget
  // and engine dependent, they are within one order of magnitude of each other,
  // `sync-hard-cutover` gives this function a network-fed producer whose payload
  // is parsed rather than cloned, and the SECOND half of F-2 — a throwing
  // accessor — is reachable today with nothing validating the stored value
  // (B1's R-C). Exposure today is nil either way: dormant, no producer.
  //
  // RED BEFORE THE FIX: `RangeError: Maximum call stack size exceeded` on the
  // deep chain, and the getter's own `Error` propagating out — both through
  // `buildSheetModel`, not merely in isolation.
  // ─────────────────────────────────────────────────────────────────────────
  test('F-2 — a deep `_v` chain is BOUNDED, not a stack overflow', async () => {
    const { formatValue, MAX_VALUE_UNWRAP_DEPTH } = await load();

    let deep = 'the answer';
    for (let i = 0; i < 30000; i++) deep = { _v: deep };

    let out;
    expect(() => { out = formatValue(deep); }).not.toThrow();
    expect(typeof out).toBe('string');

    // The same chain through `{value}`, the other bundle shape.
    let deepV = 'the answer';
    for (let i = 0; i < 30000; i++) deepV = { value: deepV };
    expect(() => formatValue(deepV)).not.toThrow();

    // The bound is a NAMED constant, mirroring `MAX_SNAPSHOT_DEPTH` — the
    // convention `findFieldInSnapshot` already set, not a new one.
    expect(typeof MAX_VALUE_UNWRAP_DEPTH).toBe('number');

    // A chain SHORTER than the bound still unwraps all the way to the answer —
    // the bound must not cost the ordinary `{_v, _fail_note}` case anything.
    let shallow = 39;
    for (let i = 0; i < MAX_VALUE_UNWRAP_DEPTH; i++) shallow = { _v: shallow };
    expect(formatValue(shallow)).toBe('39');

    // A cycle is the same class and must also be bounded.
    const cyclic = {};
    cyclic._v = cyclic;
    expect(() => formatValue(cyclic)).not.toThrow();
  });

  test('F-2 — a throwing getter on `_v` degrades, it does not propagate', async () => {
    const { formatValue } = await load();
    const boobytrapped = {};
    Object.defineProperty(boobytrapped, '_v', {
      enumerable: true,
      get() { throw new Error('malformed stored value'); },
    });
    expect(() => formatValue(boobytrapped)).not.toThrow();
    expect(formatValue(boobytrapped)).toBe('—');

    const onValue = {};
    Object.defineProperty(onValue, 'value', {
      enumerable: true,
      get() { throw new Error('malformed stored value'); },
    });
    expect(() => formatValue(onValue)).not.toThrow();
  });

  test('F-2 — and it holds THROUGH buildSheetModel, which is where it renders', async () => {
    const { buildSheetModel } = await load();

    let deep = 'the answer';
    for (let i = 0; i < 30000; i++) deep = { _v: deep };
    const thrower = {};
    Object.defineProperty(thrower, '_v', {
      enumerable: true,
      get() { throw new Error('malformed stored value'); },
    });

    // Both columns the sheet draws: what was discarded AND what the server now
    // shows. `formatValue` runs on each of them for every row.
    let model;
    expect(() => {
      model = buildSheetModel(
        [
          rec({ id: 'deep-discarded', field_id: 'f1', discarded_value: deep }),
          rec({ id: 'deep-current', field_id: 'f2', current_value: deep }),
          rec({ id: 'throws-discarded', field_id: 'f3', discarded_value: thrower }),
          rec({ id: 'throws-current', field_id: 'f4', current_value: thrower }),
        ],
        { now: NOW },
      );
    }).not.toThrow();
    expect(model.groups[0].rows).toHaveLength(4);
    for (const row of model.groups[0].rows) {
      expect(typeof row.yours, `row ${row.id} yours`).toBe('string');
      expect(typeof row.theirs, `row ${row.id} theirs`).toBe('string');
    }
  });

  test('a null actor degrades to "someone else" — never an invented name', async () => {
    const { formatWho, buildSheetModel } = await load();
    expect(formatWho(null, null)).toBe('someone else');
    const m = buildSheetModel([rec({ id: 'a', overwritten_by_name: null })], { now: NOW });
    expect(m.groups[0].rows[0].who).toMatch(/^someone else, /);
  });

  test('A-2.3 — every row carries name AND time, collapsed or not', async () => {
    const { buildSheetModel } = await load();
    const m = buildSheetModel([...SUCCESS, rec({ id: 'c', field_id: 'f3' })], { now: NOW });
    expect(m.collapsed).toBe(true);
    expect(m.groups[0].rows).toHaveLength(3);
    for (const row of m.groups[0].rows) {
      // r1's counter-example was a bare "Dana M." in the collapsed view — the
      // riskiest action on the sheet carrying the least information.
      expect(row.who, `row ${row.id}`).toMatch(/^Dana M\., \d{1,2}:\d{2} (AM|PM)$/);
    }
  });
});

// ---------------------------------------------------------------------------
// The record written from a conflict$ emission.
// ---------------------------------------------------------------------------

test.describe('recording a conflict', () => {
  test('one record per clash, keyed by document + field, carrying BOTH values', async () => {
    const { recordsFromConflict, recordIdFor } = await load();
    const described = {
      id: 'rsp-1',
      baseline: 'assumed-master',
      winnerActor: 'u-dana',
      winnerAt: '2026-07-27T18:12:00-04:00',
      clashes: [{ field: 'value', discarded: 38, winner: 41 }],
      master: { submission_id: DOC_A },
    };
    const rows = recordsFromConflict(described, { actorName: 'Dana M.', collection: 'responses' });
    expect(rows).toHaveLength(1);
    expect(rows[0].id).toBe(recordIdFor('rsp-1', 'value'));
    expect(rows[0].discarded_value).toBe(38);
    // The `Now shows` value. r2's schema carried no field for it, and every
    // plate draws it.
    expect(rows[0].current_value).toBe(41);
    expect(rows[0].overwritten_by_name).toBe('Dana M.');
    expect(rows[0].status).toBe('open');
  });

  test('an emission with no clashes writes nothing', async () => {
    const { recordsFromConflict } = await load();
    expect(recordsFromConflict({ id: 'x', clashes: [] })).toEqual([]);
    expect(recordsFromConflict(null)).toEqual([]);
  });
});
