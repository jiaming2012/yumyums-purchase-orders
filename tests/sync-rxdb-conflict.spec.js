// tests/sync-rxdb-conflict.spec.js — the conflict handler contract.
//
// Card `sync-rxdb-replication-and-conflict-handler` (overnight-20260801, C1).
// Ledger T-22 decision 50.
//
// ===========================================================================
// WHY THIS SUITE NEEDS NO BROWSER, NO DATABASE AND NO REPLICATION.
// ===========================================================================
// "Headless and testable is a REQUIREMENT, not a nicety" — C2 builds the
// user-visible half against this handler's behaviour, so the behaviour has to
// be pinned by something that runs in the ordinary suite rather than by a
// hand-run two-device ritual.
//
// The whole decision is a pure function of the three states RxDB already hands
// the handler (`assumedMasterState`, `newDocumentState`, `realMasterState`), so
// `sync-rxdb/conflict-handler.js` imports NOTHING and every rule below is
// exercised by calling it with three plain objects. The one thing that is not a
// pure function — "is this really what RxDB's default does?" — is measured
// against the REAL committed engine: `vendor/rxdb.bundle.js` imports cleanly in
// Node (verified), so the reproduction group below calls the actual
// `defaultConflictHandler` rather than a description of it.
//
// ===========================================================================
// THE RED THIS FILE REPRODUCES BEFORE IT FIXES.
// ===========================================================================
// RxDB's default is `resolve: (i) => i.realMasterState` — UNCONDITIONAL
// master-wins. Not last-write-wins: no clock participates, and a strictly-later
// local write is discarded silently. Observed four times by the spike
// (`proof-lww.js`) and reproduced here a fifth, directly against the shipped
// bundle, in `reproduction` below.
//
// ===========================================================================
// ANTI-VACUOUS DISCIPLINE (B-22/B-23/B-24, armed 2026-07-29).
// ===========================================================================
// Most of this file is data-driven off CASES. A guard printing PASS is not
// evidence until its subject set is shown non-empty — so the first test asserts
// the case table's exact size and its exact case names. Emptying CASES reds the
// suite instead of silently passing zero assertions.

const { test, expect } = require('@playwright/test');
const path = require('path');
const { pathToFileURL } = require('url');

const REPO_ROOT = path.resolve(__dirname, '..');
const HANDLER_PATH = path.join(REPO_ROOT, 'sync-rxdb', 'conflict-handler.js');
const BUNDLE_PATH = path.join(REPO_ROOT, 'vendor', 'rxdb.bundle.js');

// Loaded INSIDE each test, never at file scope: a top-level await that throws
// makes the whole file fail to load, and a spec file reporting zero executed
// tests is this repo's signature silent failure (B-09, B-14, B-16). Per-test
// loading turns a missing module into N real, counted failures.
async function loadHandler() {
  return import(pathToFileURL(HANDLER_PATH).href);
}
async function loadBundle() {
  return import(pathToFileURL(BUNDLE_PATH).href);
}

// ---------------------------------------------------------------------------
// A `checklist_templates` row (collection `templates`). Two different people
// can legitimately touch two different fields of it in the same minute: the
// owner renames the checklist while a manager flips the approval requirement.
// Under RxDB's default one of those two edits is destroyed.
// ---------------------------------------------------------------------------
const T_ASSUMED = {
  id: 'tpl-1',
  name: 'Opening Checklist',
  requires_approval: false,
  created_by: 'u-owner',
  created_at: '2026-08-01T10:00:00.000Z',
  updated_by: 'u-owner',
  updated_at: '2026-08-01T10:00:00.000Z',
  archived_at: null,
  _deleted: false,
  _rev: '1-aaa',
  _meta: { lwt: 1 },
  _attachments: {},
};

// A `submission_responses` row (collection `responses`) — one answer, the
// granularity the conflict sheet works at.
const R_ASSUMED = {
  id: 'rsp-1',
  submission_id: null,
  field_id: 'fld-prep-sink-temp',
  value: { _v: 41, _fail_note: null },
  answered_by: 'u-dana',
  answered_at: '2026-08-01T18:12:00.000Z',
  _deleted: false,
  _rev: '1-bbb',
  _meta: { lwt: 2 },
  _attachments: {},
};

// A `checklist_submissions` row (collection `checklists`). `status` is an enum
// with four values, so — unlike a boolean — it can genuinely clash: the crew
// completes the checklist offline while the approver approves it in the office.
const C_ASSUMED = {
  id: 'sub-1',
  template_id: 'tpl-1',
  template_snapshot: { name: 'Opening Checklist', fields: [] },
  submitted_by: 'u-crew',
  submitted_at: '2026-08-01T17:00:00.000Z',
  status: 'pending',
  reviewed_by: null,
  reviewed_at: null,
  idempotency_key: null,
  updated_by: 'u-crew',
  updated_at: '2026-08-01T17:00:00.000Z',
  _deleted: false,
  _rev: '1-ccc',
  _meta: { lwt: 3 },
  _attachments: {},
};

const clone = (o) => JSON.parse(JSON.stringify(o));
const withFields = (base, patch) => Object.assign(clone(base), patch);

// ---------------------------------------------------------------------------
// THE CASE TABLE. Each case names the three states and the contract:
//   expectDoc    — the fields the resolved document must carry
//   expectClashes— the clashes that must be reported, field-by-field
// Written out here rather than derived from the module: a test that derives its
// expectations from the artefact under test passes for any artefact.
// ---------------------------------------------------------------------------
const CASES = [
  {
    name: 'different fields, different people — BOTH survive',
    assumed: T_ASSUMED,
    fork: withFields(T_ASSUMED, { name: 'Opening Checklist (AM)' }),
    master: withFields(T_ASSUMED, { requires_approval: true, _rev: '2-mmm' }),
    expectDoc: { name: 'Opening Checklist (AM)', requires_approval: true },
    expectClashes: [],
  },
  {
    name: 'only the fork changed a field — the fork value survives',
    assumed: T_ASSUMED,
    fork: withFields(T_ASSUMED, { name: 'Opening Checklist (AM)' }),
    master: withFields(T_ASSUMED, { archived_at: '2026-08-01T20:00:00.000Z', _rev: '2-mmm' }),
    expectDoc: { name: 'Opening Checklist (AM)', archived_at: '2026-08-01T20:00:00.000Z' },
    expectClashes: [],
  },
  {
    name: 'only master changed a field — the master value survives',
    assumed: T_ASSUMED,
    fork: withFields(T_ASSUMED, { name: 'Opening Checklist' }), // untouched
    master: withFields(T_ASSUMED, { name: 'Opening — do not rename', _rev: '2-mmm' }),
    expectDoc: { name: 'Opening — do not rename' },
    expectClashes: [],
  },
  {
    name: 'genuine same-field clash — master wins AND the fork value is recorded',
    assumed: R_ASSUMED,
    fork: withFields(R_ASSUMED, { value: { _v: 39, _fail_note: null } }),
    master: withFields(R_ASSUMED, { value: { _v: 45, _fail_note: null }, _rev: '2-mmm' }),
    expectDoc: { value: { _v: 45, _fail_note: null } },
    expectClashes: [
      { field: 'value', discarded: { _v: 39, _fail_note: null }, winner: { _v: 45, _fail_note: null } },
    ],
  },
  {
    name: 'both sides made the SAME edit — convergent, not a clash',
    assumed: R_ASSUMED,
    fork: withFields(R_ASSUMED, { value: { _v: 45, _fail_note: null } }),
    master: withFields(R_ASSUMED, { value: { _v: 45, _fail_note: null }, _rev: '2-mmm' }),
    expectDoc: { value: { _v: 45, _fail_note: null } },
    expectClashes: [],
  },
  {
    name: 'a clash on one field does not destroy a clean merge on another',
    assumed: T_ASSUMED,
    fork: withFields(T_ASSUMED, { name: 'FORK NAME', archived_at: '2026-08-02T00:00:00.000Z' }),
    master: withFields(T_ASSUMED, { name: 'MASTER NAME', requires_approval: true, _rev: '2-mmm' }),
    expectDoc: {
      name: 'MASTER NAME',
      requires_approval: true,
      archived_at: '2026-08-02T00:00:00.000Z',
    },
    expectClashes: [{ field: 'name', discarded: 'FORK NAME', winner: 'MASTER NAME' }],
  },
  {
    name: 'a field the fork invented (present in NEITHER assumed nor master) survives',
    assumed: T_ASSUMED,
    fork: withFields(T_ASSUMED, { crew_note: 'walk-in door sticking' }),
    master: withFields(T_ASSUMED, { requires_approval: true, _rev: '2-mmm' }),
    expectDoc: { crew_note: 'walk-in door sticking', requires_approval: true },
    expectClashes: [],
  },
  {
    name: 'nested values compare by VALUE, not by identity',
    assumed: R_ASSUMED,
    // Structurally identical to assumed — a different object, the same answer.
    fork: withFields(R_ASSUMED, { value: { _v: 41, _fail_note: null } }),
    master: withFields(R_ASSUMED, { value: { _v: 45, _fail_note: null }, _rev: '2-mmm' }),
    expectDoc: { value: { _v: 45, _fail_note: null } },
    // The fork did NOT actually change the answer, so master's edit is not a
    // clash. A shallow/identity compare would report one here.
    expectClashes: [],
  },
  {
    // `_deleted` is NOT reserved — it is a real user act and it merges by the
    // ordinary rule. The fork deleted the row, master only renamed it, so the
    // delete is UNCONTESTED and survives (and carries master's rename with it).
    //
    // 🛑 THIS CASE PINS AN OPEN QUESTION, not a settled rule. Master's rename
    // lands on a tombstone and NOTHING is reported, because no field clashed —
    // the one silent-loss path left in the merge.
    //
    // It is unreachable today on EXACTLY ONE ground: **no HQ page writes
    // through RxDB**, so no fork exists to carry `_deleted: true`. Full stop.
    // An earlier draft of this comment also claimed HQ hard-deletes none of the
    // four mirrored tables — THAT IS FALSE and must not be re-derived. HQ
    // hard-deletes three of the four from live paths: `saveResponse` deletes
    // the `submission_responses` row on a null value
    // (`backend/internal/workflow/repository.go:811` — **that is unchecking a
    // checkbox**, the tool's highest-frequency write), and `unsubmitChecklist`
    // deletes `submission_rejections` then `checklist_submissions`
    // (`repository.go:1289`, `:1297`). Only `checklist_templates` is
    // delete-free (it ARCHIVES via `archived_at`).
    //
    // So the severity is high, not marginal: once `sync-hard-cutover` opens the
    // write path, an offline crew member unchecking a box while a manager edits
    // the same response row annihilates the manager's edit silently — the most
    // LIKELY conflict on this schema, not a corner case. "Should an intentional
    // delete beat a concurrent edit?" is a product question belonging to
    // `sync-hard-cutover`, the card that introduces the write path. Pinned here
    // so that card changes it visibly rather than by accident. See the header
    // of sync-rxdb/conflict-handler.js.
    //
    // 🛑 Worth writing down because it is not obvious: WITH a baseline a
    // `_deleted` clash is UNREACHABLE. It is a boolean, so "both sides changed
    // it from the same baseline" forces both to have flipped to the same value,
    // which is the convergent case. `_deleted` can therefore only be reported
    // in the no-baseline fallback — asserted separately below.
    name: '_deleted participates in the merge — an UNCONTESTED local delete survives',
    assumed: T_ASSUMED,
    fork: withFields(T_ASSUMED, { _deleted: true }),
    master: withFields(T_ASSUMED, { _deleted: false, name: 'MASTER NAME', _rev: '2-mmm' }),
    expectDoc: { _deleted: true, name: 'MASTER NAME' },
    expectClashes: [],
  },
  {
    // The real-world clash the conflict sheet exists for: the crew marks the
    // checklist complete on the truck while the approver approves it in the
    // office. `reviewed_by`/`reviewed_at` are fork-untouched, so they merge
    // cleanly and only `status` is reported.
    name: 'an enum clash on checklists — status completed vs approved',
    assumed: C_ASSUMED,
    fork: withFields(C_ASSUMED, { status: 'completed' }),
    master: withFields(C_ASSUMED, {
      status: 'approved', reviewed_by: 'u-mgr',
      reviewed_at: '2026-08-01T19:00:00.000Z', _rev: '2-mmm',
    }),
    expectDoc: { status: 'approved', reviewed_by: 'u-mgr' },
    expectClashes: [{ field: 'status', discarded: 'completed', winner: 'approved' }],
  },
];

const CASE_NAMES = CASES.map((c) => c.name);

test.describe('sync-rxdb conflict handler — the case table is real', () => {
  test('the case table is non-empty and names exactly the decided rules', () => {
    // B-22/B-23/B-24: a guard's PASS is not evidence until its subject set is
    // shown non-empty. Every data-driven test below iterates CASES.
    expect(CASES.length).toBe(10);
    expect(new Set(CASE_NAMES).size).toBe(CASES.length); // no duplicate names
    for (const c of CASES) {
      expect(c.assumed, `${c.name}: no assumed state`).toBeTruthy();
      expect(c.fork, `${c.name}: no fork state`).toBeTruthy();
      expect(c.master, `${c.name}: no master state`).toBeTruthy();
      expect(Object.keys(c.expectDoc).length, `${c.name}: asserts nothing`).toBeGreaterThan(0);
    }
    // At least one case must actually clash, or "master-wins fallback" is never
    // exercised and the whole clash half of the suite is vacuous.
    expect(CASES.filter((c) => c.expectClashes.length > 0).length).toBeGreaterThanOrEqual(3);
  });
});

// ===========================================================================
// REPRODUCTION — the defect, measured against the SHIPPED engine.
// ===========================================================================
test.describe('reproduction — RxDB 17.4.0 default conflict handler (decision 50)', () => {
  test('the default is unconditional master-wins: a strictly-LATER local write is discarded', async () => {
    const { defaultConflictHandler } = await loadBundle();

    // The fork's write is strictly later in wall-clock time than master's.
    const assumed = clone(R_ASSUMED);
    const master = withFields(R_ASSUMED, {
      value: { _v: 45, _fail_note: null },
      answered_at: '2026-08-01T18:20:00.000Z', // T1 — EARLIER
      _rev: '2-mmm',
    });
    const fork = withFields(R_ASSUMED, {
      value: { _v: 39, _fail_note: null },
      answered_at: '2026-08-01T18:30:00.000Z', // T2 > T1 — LATER
    });

    const out = await defaultConflictHandler.resolve(
      { assumedMasterState: assumed, newDocumentState: fork, realMasterState: master },
      'replication-resolve-conflict',
    );

    // The later write loses. No clock participated.
    expect(out.value).toEqual({ _v: 45, _fail_note: null });
    expect(out.answered_at).toBe('2026-08-01T18:20:00.000Z');
    expect(out).toEqual(master);
    // And nothing about the discarded value is carried anywhere in the output.
    expect(JSON.stringify(out)).not.toContain('39');
  });

  test('the default destroys an edit to a DIFFERENT field than the one master changed', async () => {
    const { defaultConflictHandler } = await loadBundle();
    const fork = withFields(T_ASSUMED, { name: 'Opening Checklist (AM)' });
    const master = withFields(T_ASSUMED, { requires_approval: true, _rev: '2-mmm' });

    const out = await defaultConflictHandler.resolve(
      { assumedMasterState: clone(T_ASSUMED), newDocumentState: fork, realMasterState: master },
      'replication-resolve-conflict',
    );

    // The rename is gone even though master never touched `name`. THIS is the
    // loss the field-level three-way merge exists to stop.
    expect(out.name).toBe('Opening Checklist');
    expect(out.requires_approval).toBe(true);
  });

  test('assumedMasterState is present in the input RxDB actually hands the handler', async () => {
    // Decision 50 turns on this fact. It is declared OPTIONAL in
    // `conflict-handling.d.ts:10`, so the rule needs an absent-case fallback —
    // asserted separately below. Here: the engine accepts and ignores it.
    const { defaultConflictHandler } = await loadBundle();
    const seen = [];
    const probe = {
      isEqual: defaultConflictHandler.isEqual,
      resolve(input, ctx) {
        seen.push({ keys: Object.keys(input).sort(), ctx });
        return defaultConflictHandler.resolve(input, ctx);
      },
    };
    await probe.resolve(
      {
        assumedMasterState: clone(T_ASSUMED),
        newDocumentState: withFields(T_ASSUMED, { name: 'x' }),
        realMasterState: withFields(T_ASSUMED, { requires_approval: true }),
      },
      'replication-resolve-conflict',
    );
    expect(seen).toHaveLength(1);
    expect(seen[0].keys).toEqual(['assumedMasterState', 'newDocumentState', 'realMasterState']);
  });
});

// ===========================================================================
// THE HQ HANDLER — field-level three-way merge (decision 50).
// ===========================================================================
test.describe('HQ conflict handler — field-level three-way merge', () => {
  for (const c of CASES) {
    test(`resolve: ${c.name}`, async () => {
      const { resolveConflict } = await loadHandler();
      const { document, clashes } = resolveConflict({
        assumedMasterState: clone(c.assumed),
        newDocumentState: clone(c.fork),
        realMasterState: clone(c.master),
      });

      for (const [k, v] of Object.entries(c.expectDoc)) {
        expect(document[k], `${c.name}: field "${k}"`).toEqual(v);
      }
      expect(
        clashes.map((x) => ({ field: x.field, discarded: x.discarded, winner: x.winner })),
        c.name,
      ).toEqual(c.expectClashes);
    });
  }

  test('the resolved document keeps master\'s RxDB metadata, never the fork\'s', async () => {
    const { resolveConflict } = await loadHandler();
    const fork = withFields(T_ASSUMED, { name: 'FORK', _rev: '9-fff', _meta: { lwt: 999 } });
    const master = withFields(T_ASSUMED, { requires_approval: true, _rev: '2-mmm', _meta: { lwt: 2 } });
    const { document, clashes } = resolveConflict({
      assumedMasterState: clone(T_ASSUMED),
      newDocumentState: fork,
      realMasterState: master,
    });
    expect(document._rev).toBe('2-mmm');
    expect(document._meta).toEqual({ lwt: 2 });
    // Metadata divergence is protocol substrate, never a user-visible clash.
    expect(clashes.map((c) => c.field)).not.toContain('_rev');
    expect(clashes.map((c) => c.field)).not.toContain('_meta');
    expect(clashes.map((c) => c.field)).not.toContain('_attachments');
  });

  test('provenance (who-and-when) is merged but NEVER reported as a clash', async () => {
    // Decision 79's columns describe who last wrote the ROW, not an answer.
    // Both sides always change them when both sides edit, so reporting them
    // would put a clash on every single merge and swamp C2's sheet with rows
    // from which nothing recoverable was lost.
    const { resolveConflict, PROVENANCE_FIELDS } = await loadHandler();
    expect(PROVENANCE_FIELDS).toContain('updated_by');
    expect(PROVENANCE_FIELDS).toContain('answered_by');
    expect(PROVENANCE_FIELDS).toContain('rejected_by');

    const fork = withFields(T_ASSUMED, {
      name: 'FORK NAME', updated_by: 'u-crew', updated_at: '2026-08-01T18:30:00.000Z',
    });
    const master = withFields(T_ASSUMED, {
      requires_approval: true, updated_by: 'u-mgr', updated_at: '2026-08-01T18:20:00.000Z',
      _rev: '2-mmm',
    });
    const { document, clashes } = resolveConflict({
      assumedMasterState: clone(T_ASSUMED), newDocumentState: fork, realMasterState: master,
    });
    expect(document.name).toBe('FORK NAME');
    expect(document.requires_approval).toBe(true);
    expect(document.updated_by).toBe('u-mgr'); // master's, silently
    expect(clashes).toEqual([]);
  });

  test('an unchanged row on both sides resolves to master with no clashes', async () => {
    const { resolveConflict } = await loadHandler();
    const { document, clashes } = resolveConflict({
      assumedMasterState: clone(T_ASSUMED),
      newDocumentState: clone(T_ASSUMED),
      realMasterState: withFields(T_ASSUMED, { _rev: '2-mmm' }),
    });
    expect(clashes).toEqual([]);
    expect(document.name).toBe('Opening Checklist');
  });

  test('the resolved document never carries a field absent from all three inputs', async () => {
    const { resolveConflict } = await loadHandler();
    for (const c of CASES) {
      const { document } = resolveConflict({
        assumedMasterState: clone(c.assumed),
        newDocumentState: clone(c.fork),
        realMasterState: clone(c.master),
      });
      const known = new Set([
        ...Object.keys(c.assumed), ...Object.keys(c.fork), ...Object.keys(c.master),
      ]);
      for (const k of Object.keys(document)) {
        expect(known.has(k), `${c.name}: invented field "${k}"`).toBe(true);
      }
    }
  });

  test('no clock, no timestamp, participates in any decision (decision 50 rejected LWW)', async () => {
    const { resolveConflict } = await loadHandler();
    // Same clash, run twice with the fork's timestamps swung from far-earlier
    // to far-later. Under genuine last-write-wins the outcome would flip.
    const results = ['1999-01-01T00:00:00.000Z', '2099-01-01T00:00:00.000Z'].map((ts) =>
      resolveConflict({
        assumedMasterState: clone(R_ASSUMED),
        newDocumentState: withFields(R_ASSUMED, { value: { _v: 39 }, answered_at: ts }),
        realMasterState: withFields(R_ASSUMED, { value: { _v: 45 }, _rev: '2-mmm' }),
      }),
    );
    expect(results[0].document.value).toEqual({ _v: 45 });
    expect(results[1].document.value).toEqual({ _v: 45 });
    expect(results[0].clashes).toEqual(results[1].clashes);
  });
});

// ===========================================================================
// THE assumedMasterState-ABSENT FALLBACK.
// ===========================================================================
test.describe('HQ conflict handler — the assumedMasterState-absent fallback', () => {
  test('with no baseline, master wins on EVERY field and every difference is recorded', async () => {
    const { resolveConflict } = await loadHandler();
    const fork = withFields(T_ASSUMED, { name: 'FORK NAME', archived_at: '2026-08-02T00:00:00.000Z' });
    const master = withFields(T_ASSUMED, { requires_approval: true, _rev: '2-mmm' });

    const { document, clashes, baseline } = resolveConflict({
      assumedMasterState: undefined,
      newDocumentState: fork,
      realMasterState: master,
    });

    // Identical WINNER to RxDB's default — never worse.
    expect(document).toEqual(master);
    expect(baseline).toBe('absent');
    // Strictly better on recoverability: every field the fork held differently
    // is surfaced, because without a baseline none of them can be shown to be
    // uncontested.
    // `requires_approval` is in the list too, and that is the fallback working
    // as decided rather than a fixture slip: the fork holds `false` and master
    // holds `true`, and with no baseline nothing can show WHICH side moved it.
    expect(clashes.map((c) => c.field).sort())
      .toEqual(['archived_at', 'name', 'requires_approval']);
    expect(clashes.find((c) => c.field === 'name').discarded).toBe('FORK NAME');
    expect(clashes.find((c) => c.field === 'archived_at').discarded).toBe('2026-08-02T00:00:00.000Z');
  });

  test('with no baseline, a diverging _deleted IS reported (the only path that reaches it)', async () => {
    // With a baseline this is unreachable — see the CASES note on `_deleted`.
    const { resolveConflict } = await loadHandler();
    const { document, clashes } = resolveConflict({
      newDocumentState: withFields(T_ASSUMED, { _deleted: true }),
      realMasterState: withFields(T_ASSUMED, { _deleted: false, _rev: '2-mmm' }),
    });
    expect(document._deleted).toBe(false); // master wins
    expect(clashes).toEqual([{ field: '_deleted', discarded: true, winner: false }]);
  });

  test('with no baseline and an identical fork, nothing is reported', async () => {
    const { resolveConflict } = await loadHandler();
    const master = withFields(T_ASSUMED, { _rev: '2-mmm' });
    const { document, clashes, baseline } = resolveConflict({
      assumedMasterState: null,
      newDocumentState: clone(T_ASSUMED),
      realMasterState: master,
    });
    expect(baseline).toBe('absent');
    expect(clashes).toEqual([]);
    expect(document).toEqual(master);
  });

  test('with no baseline, provenance and metadata are still never reported', async () => {
    const { resolveConflict } = await loadHandler();
    const fork = withFields(T_ASSUMED, { updated_by: 'u-crew', _rev: '9-fff' });
    const master = withFields(T_ASSUMED, { updated_by: 'u-mgr', _rev: '2-mmm' });
    const { clashes } = resolveConflict({
      newDocumentState: fork, realMasterState: master,
    });
    expect(clashes).toEqual([]);
  });

  test('the baseline flag distinguishes a three-way merge from the fallback', async () => {
    const { resolveConflict } = await loadHandler();
    const three = resolveConflict({
      assumedMasterState: clone(T_ASSUMED),
      newDocumentState: withFields(T_ASSUMED, { name: 'A' }),
      realMasterState: withFields(T_ASSUMED, { requires_approval: true }),
    });
    expect(three.baseline).toBe('assumed-master');
  });

  test('a missing realMasterState throws loudly rather than guessing', async () => {
    const { resolveConflict } = await loadHandler();
    expect(() => resolveConflict({ newDocumentState: clone(T_ASSUMED) })).toThrow(/realMasterState/);
  });
});

// ===========================================================================
// THE RxConflictHandler ADAPTER — what actually gets handed to RxDB.
// ===========================================================================
test.describe('HQ conflict handler — the RxDB adapter', () => {
  test('isEqual is the injected one, byte-for-byte — the handler does not redefine equality', async () => {
    const { createHQConflictHandler } = await loadHandler();
    const calls = [];
    const isEqual = (a, b, ctx) => { calls.push(ctx); return a.id === b.id; };
    const handler = createHQConflictHandler({ isEqual });
    expect(handler.isEqual({ id: 'x' }, { id: 'x' }, 'downstream-check-if-equal-0')).toBe(true);
    expect(handler.isEqual({ id: 'x' }, { id: 'y' }, 'upstream-check-if-equal')).toBe(false);
    expect(calls).toEqual(['downstream-check-if-equal-0', 'upstream-check-if-equal']);
  });

  test('constructed with the REAL RxDB isEqual, the reproduced loss no longer happens', async () => {
    // The direct "the red is fixed" test: same inputs as the reproduction group.
    const { defaultConflictHandler } = await loadBundle();
    const { createHQConflictHandler } = await loadHandler();
    const handler = createHQConflictHandler({ isEqual: defaultConflictHandler.isEqual });

    const out = await handler.resolve(
      {
        assumedMasterState: clone(T_ASSUMED),
        newDocumentState: withFields(T_ASSUMED, { name: 'Opening Checklist (AM)' }),
        realMasterState: withFields(T_ASSUMED, { requires_approval: true, _rev: '2-mmm' }),
      },
      'replication-resolve-conflict',
    );
    // Both survive, where the default kept only master's.
    expect(out.name).toBe('Opening Checklist (AM)');
    expect(out.requires_approval).toBe(true);
  });

  test('resolve returns a bare document (RxDB overwrites _rev/_meta/_attachments itself)', async () => {
    const { createHQConflictHandler } = await loadHandler();
    const handler = createHQConflictHandler();
    const out = await handler.resolve({
      assumedMasterState: clone(R_ASSUMED),
      newDocumentState: withFields(R_ASSUMED, { value: { _v: 39 } }),
      realMasterState: withFields(R_ASSUMED, { value: { _v: 45 }, _rev: '2-mmm' }),
    }, 'replication-resolve-conflict');
    expect(out.id).toBe('rsp-1');
    expect(out.value).toEqual({ _v: 45 });
    expect(out.clashes).toBeUndefined(); // the doc is a document, not a report
  });

  test('onClash fires once per clashing field, with the discarded value recoverable', async () => {
    const { createHQConflictHandler } = await loadHandler();
    const seen = [];
    const handler = createHQConflictHandler({ onClash: (e) => seen.push(e) });
    await handler.resolve({
      assumedMasterState: clone(T_ASSUMED),
      newDocumentState: withFields(T_ASSUMED, {
        name: 'FORK NAME', requires_approval: true, archived_at: '2026-08-02T00:00:00.000Z',
      }),
      realMasterState: withFields(T_ASSUMED, {
        name: 'MASTER NAME', requires_approval: true, _rev: '2-mmm',
      }),
    }, 'replication-resolve-conflict');

    expect(seen).toHaveLength(1);
    expect(seen[0].id).toBe('tpl-1');
    expect(seen[0].field).toBe('name');
    expect(seen[0].discarded).toBe('FORK NAME');
    expect(seen[0].winner).toBe('MASTER NAME');
    expect(seen[0].baseline).toBe('assumed-master');
  });

  test('onClash does NOT fire when nothing was lost', async () => {
    const { createHQConflictHandler } = await loadHandler();
    const seen = [];
    const handler = createHQConflictHandler({ onClash: (e) => seen.push(e) });
    await handler.resolve({
      assumedMasterState: clone(T_ASSUMED),
      newDocumentState: withFields(T_ASSUMED, { name: 'FORK NAME' }),
      realMasterState: withFields(T_ASSUMED, { requires_approval: true, _rev: '2-mmm' }),
    }, 'replication-resolve-conflict');
    expect(seen).toEqual([]);
  });

  test('a throwing onClash never breaks replication', async () => {
    // The notice is a courtesy; losing it must not wedge the sync loop.
    const { createHQConflictHandler } = await loadHandler();
    const handler = createHQConflictHandler({
      onClash: () => { throw new Error('C2 UI blew up'); },
    });
    const out = await handler.resolve({
      assumedMasterState: clone(R_ASSUMED),
      newDocumentState: withFields(R_ASSUMED, { value: { _v: 39 } }),
      realMasterState: withFields(R_ASSUMED, { value: { _v: 45 }, _rev: '2-mmm' }),
    }, 'replication-resolve-conflict');
    expect(out.value).toEqual({ _v: 45 });
  });
});

// ===========================================================================
// describeConflict — what C2 calls on a `conflict$` emission.
// ===========================================================================
test.describe('describeConflict — the conflict$ contract C2 builds against', () => {
  // `conflict$` emits `{input, output}` per DOCUMENT and carries the document
  // id (upstream.js:333 — verified in the shipped bundle: the emission happens
  // inside `Object.entries(...).map(...)`, one per document).
  const emission = () => ({
    input: {
      assumedMasterState: clone(R_ASSUMED),
      newDocumentState: withFields(R_ASSUMED, { value: { _v: 39 }, answered_by: 'u-crew' }),
      realMasterState: withFields(R_ASSUMED, {
        value: { _v: 45 }, answered_by: 'u-dana',
        answered_at: '2026-08-01T18:12:00.000Z', _rev: '2-mmm',
      }),
    },
    output: withFields(R_ASSUMED, { value: { _v: 45 }, _rev: '3-nnn' }),
  });

  test('extracts the document id, the baseline mode and every discarded value', async () => {
    const { describeConflict } = await loadHandler();
    const d = describeConflict(emission());
    expect(d.id).toBe('rsp-1');
    expect(d.baseline).toBe('assumed-master');
    expect(d.clashes).toHaveLength(1);
    expect(d.clashes[0].field).toBe('value');
    expect(d.clashes[0].discarded).toEqual({ _v: 39 });
    expect(d.clashes[0].winner).toEqual({ _v: 45 });
  });

  test('carries who-and-when for the WINNING write, so the sheet can say "Dana M., 6:12 PM"', async () => {
    const { describeConflict } = await loadHandler();
    const d = describeConflict(emission());
    expect(d.winnerActor).toBe('u-dana');
    expect(d.winnerAt).toBe('2026-08-01T18:12:00.000Z');
  });

  test('honours a custom primary path', async () => {
    const { describeConflict } = await loadHandler();
    const e = emission();
    e.input.newDocumentState.pk = 'other-key';
    expect(describeConflict(e, { primaryPath: 'pk' }).id).toBe('other-key');
  });

  test('reports nothing when the merge lost nothing', async () => {
    const { describeConflict } = await loadHandler();
    const e = emission();
    e.input.newDocumentState = withFields(R_ASSUMED, { submission_id: 'sub-9' });
    expect(describeConflict(e).clashes).toEqual([]);
  });
});

// ===========================================================================
// 🛑 G6 CORRECTION (C2, overnight-20260801) — THE HANDLER AND `describeConflict`
// MUST AGREE.
// ===========================================================================
// `describeConflict` re-runs `resolveConflict` to derive the row set C2 renders
// (`conflict-handler.js:411`, call at `:414`). It used to run with ITS OWN
// `opts`, defaulted to `{}` — so a caller who customised `reservedFields` or
// `provenanceFields` at `createHQConflictHandler` got a `conflict$` clash list
// that DISAGREED with what the handler had actually done.
//
// The symptom is not abstract: a custom `provenanceFields` entry is suppressed
// as a clash by `resolve()` — that is the whole point of the list — and, without
// threading, came back as a ROW ON THE OVERWRITTEN-ANSWERS SHEET. A row for a
// value nothing lost, on the one screen whose entire thesis is that its numbers
// are true.
//
// Pinned with a CUSTOMISED FIELD, per the correction's own wording, on both
// halves of the option surface.
test.describe('describeConflict agrees with the handler it was configured beside (G6)', () => {
  const forkAndMaster = (extraFork, extraMaster) => ({
    input: {
      assumedMasterState: clone(R_ASSUMED),
      newDocumentState: withFields(R_ASSUMED, Object.assign({ value: { _v: 39 } }, extraFork)),
      realMasterState: withFields(R_ASSUMED, Object.assign({ value: { _v: 45 }, _rev: '2-mmm' }, extraMaster)),
    },
    output: withFields(R_ASSUMED, { value: { _v: 45 }, _rev: '3-nnn' }),
  });

  test('a handler exposes the merge options it was constructed with', async () => {
    const { createHQConflictHandler, conflictOptsOf } = await loadHandler();
    const h = createHQConflictHandler({ provenanceFields: ['edited_by'] });
    expect(h.mergeOpts).toBeTruthy();
    expect(h.mergeOpts.provenanceFields).toEqual(['edited_by']);
    expect(conflictOptsOf(h)).toBe(h.mergeOpts);
    // A default-constructed handler must thread an EMPTY object, not undefined,
    // so the previous behaviour is reproduced exactly where nothing was customised.
    expect(conflictOptsOf(createHQConflictHandler())).toEqual({});
    // Anything that is not one of ours degrades to {} rather than throwing.
    expect(conflictOptsOf(null)).toEqual({});
    expect(conflictOptsOf({})).toEqual({});
  });

  test('a CUSTOM provenanceFields entry is suppressed by BOTH the handler and describeConflict', async () => {
    const { createHQConflictHandler, describeConflict, conflictOptsOf } = await loadHandler();
    const opts = { provenanceFields: ['edited_by'] };
    const handler = createHQConflictHandler(opts);
    const e = forkAndMaster({ edited_by: 'u-crew' }, { edited_by: 'u-dana' });

    // What the handler ACTUALLY did: `value` clashed, `edited_by` did not.
    const seen = [];
    createHQConflictHandler(Object.assign({ onClash: (c) => seen.push(c.field) }, opts))
      .resolve(clone(e.input));
    expect(seen).toEqual(['value']);

    // Threaded — the sheet reports the same thing.
    const threaded = describeConflict(e, conflictOptsOf(handler));
    expect(threaded.clashes.map((c) => c.field)).toEqual(['value']);

    // HOW IT FAILS: the un-threaded call is the defect. `edited_by` is not in
    // the DEFAULT provenance list, so it comes back as a second clash — a row
    // on the sheet for a value nothing lost.
    //
    // 🛑 G6 FINDING F-3 — this comment used to claim "this assertion is what
    // reddens if the threading is removed from `startHQReplication`", and that
    // was FALSE. Nothing here imports or calls `startHQReplication`; this pins
    // `describeConflict`'s own contract and nothing more, and deleting
    // `client.js`'s `conflictOptsOf(...)` argument reddened nothing in the
    // suite. The test that genuinely drives `startHQReplication` — and which
    // does red on that deletion — is at the bottom of this describe block.
    const unthreaded = describeConflict(e);
    expect(unthreaded.clashes.map((c) => c.field).sort()).toEqual(['edited_by', 'value']);
    expect(threaded.clashes.length).not.toBe(unthreaded.clashes.length);
  });

  test('a CUSTOM reservedFields entry is suppressed by BOTH the handler and describeConflict', async () => {
    const { createHQConflictHandler, describeConflict, conflictOptsOf } = await loadHandler();
    const opts = { reservedFields: ['_rev', '_meta', '_attachments', '_modified', 'device_id'] };
    const handler = createHQConflictHandler(opts);
    const e = forkAndMaster({ device_id: 'phone-a' }, { device_id: 'phone-b' });

    const seen = [];
    createHQConflictHandler(Object.assign({ onClash: (c) => seen.push(c.field) }, opts))
      .resolve(clone(e.input));
    expect(seen).toEqual(['value']);

    expect(describeConflict(e, conflictOptsOf(handler)).clashes.map((c) => c.field))
      .toEqual(['value']);
    expect(describeConflict(e).clashes.map((c) => c.field).sort())
      .toEqual(['device_id', 'value']);
  });

  test('the call SHAPE startHQReplication uses is the one that threads (simulated, not driven)', async () => {
    // 🛑 NAMED FOR WHAT IT IS (F-3). This reproduces the shape of the call by
    // hand — `conflictOptsOf(db[key].conflictHandler)` — with the collection
    // shaped as RxDB shapes it. It does NOT import `client.js` and it does NOT
    // red if the threading is deleted from it. The test below does.
    const { createHQConflictHandler, describeConflict, conflictOptsOf } = await loadHandler();
    const handler = createHQConflictHandler({ provenanceFields: ['edited_by'] });
    const db = { responses: { conflictHandler: handler } };  // RxDB hangs it here

    const e = forkAndMaster({ edited_by: 'u-crew' }, { edited_by: 'u-dana' });
    const described = describeConflict(e, conflictOptsOf(db.responses.conflictHandler));
    expect(described.clashes.map((c) => c.field)).toEqual(['value']);
  });

  // ─────────────────────────────────────────────────────────────────────────
  // 🛑 G6 FINDING F-3 — the threading, ACTUALLY DRIVEN.
  //
  // `startHQReplication`'s own comment says it "lives here so the shape is
  // reviewable now and so C2 can drive it in a test". C2 did not drive it: no
  // test in the repo imported or called it, so `client.js:411`'s
  // `conflictOptsOf(db[key] && db[key].conflictHandler)` was unpinned and its
  // deletion reddened nothing.
  //
  // Driving it needs the one seam it cannot be given by hand: `client.js`
  // statically imports `replicateSupabase` from the committed vendor bundle,
  // and calling the real one needs a real RxDB collection, a storage adapter and
  // IndexedDB. So the bundle SPECIFIER is swapped for a stub in a CHILD process
  // via a Node ESM resolve hook — `client.js` itself is loaded verbatim, from
  // disk, unmodified. A child process rather than `module.register()` in-band,
  // because a loader hook registered in the shared Playwright worker would also
  // intercept `tests/sync-rxdb-client.spec.js`'s legitimate imports of the real
  // bundle, and a stub leaking into the pin test is worse than no test at all.
  // ─────────────────────────────────────────────────────────────────────────
  test('startHQReplication REALLY threads the handler options into conflict$ (driven)', async () => {
    const { execFileSync } = require('child_process');
    const fs = require('fs');
    const os = require('os');

    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'hq-f3-'));
    const url = (p) => pathToFileURL(p).href;

    // The stub bundle. Exports exactly the six names `client.js` imports.
    // `replicateSupabase` returns a replication state whose `conflict$` emits
    // one event on subscribe — which is the seam under test.
    fs.writeFileSync(path.join(dir, 'stub-bundle.mjs'), `
      export function createClient() { return {}; }
      export async function createRxDatabase() { throw new Error('unused by startHQReplication'); }
      export function getRxStorageDexie() { return {}; }
      export const defaultConflictHandler = { isEqual: (a, b) => JSON.stringify(a) === JSON.stringify(b) };
      export const VENDOR_BUILD = { rxdb: '17.4.0', supabaseJs: '2.109.0' };
      export function replicateSupabase(opts) {
        return {
          __opts: opts,
          conflict$: { subscribe(fn) { fn(globalThis.__HQ_CONFLICT_EVENT__); return { unsubscribe() {} }; } },
        };
      }
    `);

    // A resolve hook that swaps ONLY the exact vendor-bundle URL. Everything
    // else — client.js, conflict-handler.js, collections.js — resolves normally.
    fs.writeFileSync(path.join(dir, 'hook.mjs'), `
      const VENDOR = process.env.HQ_VENDOR_URL;
      const STUB = process.env.HQ_STUB_URL;
      export async function resolve(specifier, context, next) {
        const r = await next(specifier, context);
        if (r.url === VENDOR) return { url: STUB, format: 'module', shortCircuit: true };
        return r;
      }
    `);
    fs.writeFileSync(path.join(dir, 'register.mjs'), `
      import { register } from 'node:module';
      register(${JSON.stringify(url(path.join(dir, 'hook.mjs')))});
    `);

    // The driver. Imports the REAL client.js and calls the REAL
    // startHQReplication; the only thing faked is the engine underneath it.
    fs.writeFileSync(path.join(dir, 'drive.mjs'), `
      import { startHQReplication } from ${JSON.stringify(url(path.join(REPO_ROOT, 'sync-rxdb', 'client.js')))};
      import { createHQConflictHandler } from ${JSON.stringify(url(HANDLER_PATH))};
      import { REPLICATED_COLLECTIONS } from ${JSON.stringify(url(path.join(REPO_ROOT, 'sync-schema', 'collections.js')))};

      globalThis.__HQ_CONFLICT_EVENT__ = JSON.parse(process.argv[2]);

      // The customisation is on the handler RxDB hangs off the collection —
      // exactly where a caller of createHQSyncDatabase puts it.
      const handler = createHQConflictHandler({ provenanceFields: ['edited_by'] });
      const db = {};
      for (const key of Object.keys(REPLICATED_COLLECTIONS)) db[key] = { conflictHandler: handler };

      const seen = [];
      startHQReplication(db, {}, {
        // Required since card \`sync-replication-scope-per-checklist\` (20260802,
        // A1): replication refuses to start unscoped (preference
        // architecture/C-2). This test is about conflict-option threading, not
        // about the scope — any valid scope will do, and passing one keeps the
        // subject of the test unchanged.
        // \`templateId\` became mandatory in the G6 fix round (F-5): an omitted
        // one used to widen \`templates\` to every non-archived row.
        // \`userId\` became mandatory on card \`activate-fill-view-reads\` (C3,
        // run 20260808-2) — G6 finding F-2: a fill checkpoint with no crew
        // member in its key let the second user on a shared phone resume the
        // first's cursor.
        scope: {
          userId: 'usr-conflict-threading-probe',
          checklistId: 'chk-conflict-threading-probe',
          templateId: 'tpl-conflict-threading-probe',
        },
        onConflict: (described, key) => seen.push({ key, fields: described.clashes.map((c) => c.field).sort() }),
      });
      process.stdout.write(JSON.stringify(seen));
    `);

    const e = forkAndMaster({ edited_by: 'u-crew' }, { edited_by: 'u-dana' });
    const out = execFileSync(
      process.execPath,
      ['--import', url(path.join(dir, 'register.mjs')), path.join(dir, 'drive.mjs'), JSON.stringify(e)],
      {
        encoding: 'utf8',
        env: Object.assign({}, process.env, {
          HQ_VENDOR_URL: url(BUNDLE_PATH),
          HQ_STUB_URL: url(path.join(dir, 'stub-bundle.mjs')),
        }),
      },
    );
    const seen = JSON.parse(out);

    // Anti-vacuous (B-22/B-23/B-24): a driver that subscribed to nothing would
    // print `[]` and every assertion below would pass over an empty set.
    expect(seen.length).toBeGreaterThan(0);
    // One subscription per replicated collection, and the population is PINNED
    // rather than merely walked — dropping a collection from the loop must red
    // here, not slide past.
    expect(seen.map((s) => s.key).sort())
      .toEqual(['approvals', 'checklists', 'responses', 'templates']);
    // `conflict_records` is deliberately NOT here: decision 89 makes the record
    // LOCAL and per-device, so it lives in LOCAL_COLLECTIONS and nothing
    // replicates it. Asserting the exact set is what makes that visible.

    // 🔴 THE ASSERTION THAT REDS IF THE THREADING GOES. Delete
    // `conflictOptsOf(db[key] && db[key].conflictHandler)` from
    // `startHQReplication` and `edited_by` reappears as a second clash on every
    // collection — a row on the overwritten-answers sheet for a value nothing
    // lost, on the one screen whose whole thesis is that its numbers are true.
    for (const s of seen) {
      expect(s.fields, `collection ${s.key}`).toEqual(['value']);
      expect(s.fields, `collection ${s.key}`).not.toContain('edited_by');
    }

    fs.rmSync(dir, { recursive: true, force: true });
  });
});
