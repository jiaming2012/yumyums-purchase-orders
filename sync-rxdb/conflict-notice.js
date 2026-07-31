// sync-rxdb/conflict-notice.js — the MODEL half of the overwritten-answers UI.
//
// Card `sync-rxdb-conflict-notice-ui` (overnight-20260801, C2).
// Ledger T-22 **decision 50** (the user-visible half), T-26 **82** (A-1, A-2),
// T-28 **95** (A-3), **96** (retention), **97** (the cap), T-27 **89** (the
// record is local and per-device).
//
// ===========================================================================
// WHY THE MODEL IS A SEPARATE, IMPORT-FREE MODULE.
// ===========================================================================
// The whole feature is arithmetic plus copy. The counting rule has EIGHT
// clauses, three surfaces print a number, and the amendments exist because two
// of those numbers were conflated. So the counting is a pure function of a list
// of plain records, testable with no browser, no IndexedDB and no replication —
// exactly the property C1's conflict handler was built for and for the same
// reason.
//
// The one thing imported is the retention window, and that is deliberate:
// decision 96 says the number is read from ONE named constant and no surface
// restates the literal. It is imported here and re-exported so the renderer and
// the sweep read the same object rather than a copy.
//
// ===========================================================================
// THE COUNTING RULE, IMPLEMENTED. (UI-SPEC §"The counting rule", rules 1-8.)
// ===========================================================================
//   1. banner headline  = ANSWER rows across the sheet (what was overwritten).
//   2. group chip base  = ANSWER rows in that group.
//   3. what was DONE to a row never changes 1 or 2. Rows leave only on Dismiss
//      or expiry.
//   4. UNIDENTIFIED rows are counted separately — a third banner line and the
//      chip's `+N`. chip base + `+N` = rows drawn.
//   5. the record is keyed by document id + field id, so a repeat clash on the
//      same question REPLACES the record it already has rather than adding a
//      row. `recordIdFor()` is that key.
//   6. banner second line = ANSWER rows still to review. Reviewed = restored,
//      kept or dismissed. Untouched, FAILED and in-flight are all still to
//      review.
//   7. `Restore all N` acts only on rows still to review — and, since A-3, only
//      on the ones there is somewhere to write back to.
//   8. collapse hides the Restore/Keep pair and nothing else.
//
// ===========================================================================
// 🛑 A-3 (decision 95) AND ITS DEPENDENCY ON B1's RECORDED-NOT-FIXED ITEM R-C.
// ===========================================================================
// A removed question keeps its label, struck through and read-only. The label
// comes from the submission's own `template_snapshot` — `json.Marshal(tmpl)` of
// the whole template (`backend/internal/workflow/repository.go:695`), with
// `Field.Label` on the marshalled struct (`model.go:44-57`) — so the discarded
// document carries its own frozen label for a field the LIVE template has since
// dropped.
//
// That makes the snapshot's SHAPE load-bearing, which promotes B1's R-C from a
// recorded-not-fixed item to a dependency of this card: `template_snapshot` is
// declared `{type:'object'}` with **no nested `properties`**, the committed
// vendor bundle ships no dev-mode or validation plugin, and **nothing rejects a
// malformed value**. `fieldLabelFromSnapshot` therefore assumes NOTHING. It is
// total: every input returns a string or null, it never throws, it never
// recurses without a bound, and a snapshot that cannot be read for labels falls
// back to the raw field id rather than rendering nothing.

import { CONFLICT_RECORD_RETENTION_DAYS } from '../sync-schema/collections.js';

/**
 * Decision 96. ONE named constant, re-exported rather than re-declared, so
 * every surface — the empty state's copy, the sweep, the tests — reads the same
 * value. No surface restates the literal.
 */
export const RETENTION_DAYS = CONFLICT_RECORD_RETENTION_DAYS;

/**
 * Decision 97. The sheet caps at this many GROUPS with an "and N more" line.
 * Rows below the line are NOT dropped: they stay in the store, they stay in the
 * banner's totals, and the line says how many are down there.
 */
export const MAX_GROUPS = 10;

/**
 * Collapse trigger (UI-SPEC "several conflicts at once"): more than one group,
 * or any one group past two rows. When it applies it applies to the WHOLE
 * sheet — a mixed sheet reads as a bug, not as a rule.
 */
export const COLLAPSE_MAX_ROWS_PER_GROUP = 2;

const MS_PER_DAY = 86400000;

// ---------------------------------------------------------------------------
// A-3 — the frozen label, read defensively out of `template_snapshot`.
// ---------------------------------------------------------------------------

// Bounds, because R-C means the value is whatever happened to be stored. A
// cyclic object (a snapshot round-tripped through something that reconstructed
// references), a 10-million-entry array, or a getter that throws are all
// reachable inputs, and none of them may take the sheet down with them.
const MAX_SNAPSHOT_DEPTH = 8;
const MAX_SNAPSHOT_NODES = 20000;

function isPlainish(v) {
  return !!v && typeof v === 'object' && !Array.isArray(v);
}

/**
 * The question's original label for `fieldId`, out of the submission's own
 * frozen `template_snapshot`.
 *
 * TOTAL BY CONSTRUCTION — this is A-3's dependency on R-C, not defensive
 * decoration. Returns a non-empty string, or `null` for every other input:
 * a non-object snapshot, a missing/typed-wrong `sections`, a section that is
 * not an object, a missing/typed-wrong `fields`, a field with no `id`, a
 * `label` that is not a non-empty string, a structure deeper or larger than the
 * bounds above, a cycle, or a property accessor that throws.
 *
 * `null` means "the snapshot carries no label for that id", which is exactly
 * the condition A-3 names for falling back to the raw field id.
 *
 * @param {unknown} snapshot
 * @param {unknown} fieldId
 * @returns {string|null}
 */
export function fieldLabelFromSnapshot(snapshot, fieldId) {
  const found = findFieldInSnapshot(snapshot, fieldId);
  if (!found) return null;
  const label = found.label;
  return typeof label === 'string' && label.trim() !== '' ? label : null;
}

/**
 * The whole frozen field object for `fieldId`, or null. Same totality contract
 * as `fieldLabelFromSnapshot` — the renderer wants `type` as well as `label`,
 * and walking the snapshot twice would double the cost of the one operation
 * that has to survive a malformed input.
 *
 * @returns {{id:string,label:unknown,type:unknown}|null}
 */
export function findFieldInSnapshot(snapshot, fieldId) {
  if (typeof fieldId !== 'string' || fieldId === '') return null;
  let budget = MAX_SNAPSHOT_NODES;
  const seen = new Set();

  // Iterative-with-explicit-depth rather than free recursion: `sub_steps` is a
  // real nested list (`Field.SubSteps []Field`), so the walk must recurse, but
  // an unbounded one on an attacker-shaped or merely corrupt object is a stack
  // overflow inside a render.
  const walkFields = (fields, depth) => {
    if (depth > MAX_SNAPSHOT_DEPTH || !Array.isArray(fields)) return null;
    for (const f of fields) {
      if (budget-- <= 0) return null;
      if (!isPlainish(f) || seen.has(f)) continue;
      seen.add(f);
      let id;
      let label;
      let type;
      try {
        id = f.id;
        label = f.label;
        type = f.type;
      } catch (err) {
        continue; // a throwing getter is a malformed snapshot, not a crash
      }
      if (id === fieldId) return { id: fieldId, label, type };
      let sub;
      try {
        sub = f.sub_steps;
      } catch (err) {
        sub = undefined;
      }
      const hit = walkFields(sub, depth + 1);
      if (hit) return hit;
    }
    return null;
  };

  try {
    if (!isPlainish(snapshot)) return null;
    const sections = snapshot.sections;
    if (!Array.isArray(sections)) return null;
    for (const s of sections) {
      if (budget-- <= 0) return null;
      if (!isPlainish(s) || seen.has(s)) continue;
      seen.add(s);
      const hit = walkFields(s.fields, 1);
      if (hit) return hit;
    }
    return null;
  } catch (err) {
    // Any surprise at all — a proxy, a frozen exotic object, a getter on
    // `sections` — degrades to "no label", which the caller renders as the raw
    // field id. It must never take the sheet down.
    return null;
  }
}

// ---------------------------------------------------------------------------
// Record identity — counting rule 5.
// ---------------------------------------------------------------------------

/**
 * The durable record is keyed by **document id + field id**. A restore that
 * conflicts in turn REPLACES the record it already has and updates the existing
 * row in place; without this key the closed recovery loop would grow a
 * duplicate row per retry and the counting rule would not hold.
 */
export function recordIdFor(docId, fieldId) {
  return `${docId || ''}::${fieldId || ''}`;
}

// ---------------------------------------------------------------------------
// Row classification.
// ---------------------------------------------------------------------------

/** The three renderings a row can have. */
export const KIND_ANSWER = 'answer';
export const KIND_REMOVED = 'removed';
export const KIND_UNIDENTIFIED = 'unidentified';

/**
 * Which of the three a record is.
 *
 * `unidentified` is DERIVED, not flagged: it is precisely "the diff yields
 * nothing showable", which on a stored record is "there is no discarded value".
 * Deriving it means a record cannot claim to be recoverable while carrying
 * nothing to recover.
 */
export function rowKind(rec) {
  if (!rec) return KIND_UNIDENTIFIED;
  if (rec.discarded_value === null || rec.discarded_value === undefined) {
    return KIND_UNIDENTIFIED;
  }
  return rec.field_removed ? KIND_REMOVED : KIND_ANSWER;
}

/** Counting rule 6. Reviewed = restored, kept or dismissed. */
export function isHandled(rec) {
  return rec.status === 'restored' || rec.status === 'kept';
}

/**
 * Counting rule 7 as A-3 leaves it. The batch acts on rows still to review AND
 * with somewhere to write back to.
 *
 * 🛑 This is where decision 95's accepted consequence becomes visible: under
 * Reading A a removed-field row is in the chip base and in "still to review",
 * but it can never be restored — so a group can legitimately print
 * `Restore all 1 of mine` under a chip reading `2 answers`. Triage signed that
 * off on the ground that the struck-through, read-only row makes the mismatch
 * "legible on screen rather than arithmetic". It is not a bug to be reconciled.
 */
export function isBatchTarget(rec) {
  return rowKind(rec) === KIND_ANSWER && !isHandled(rec);
}

// ---------------------------------------------------------------------------
// Value rendering.
// ---------------------------------------------------------------------------

const FREE_TEXT_TYPES = ['text', 'textarea', 'note', 'longtext'];

/**
 * One answer as a crew member would read it.
 *
 * NEVER truncated and never ellipsised — the value is the thing being
 * recovered, so hiding any of it hides the payload they came for. Long content
 * is the stylesheet's problem (`overflow-wrap:anywhere`), not this function's.
 */
export function formatValue(value, opts = {}) {
  const type = opts.type;
  const unit = typeof opts.unit === 'string' && opts.unit !== '' ? opts.unit : null;
  if (value === null || value === undefined || value === '') return '—';
  if (typeof value === 'boolean') return value ? 'Yes' : 'No';
  if (typeof value === 'number') {
    // NBSP between the figure and its unit, exactly as the mockup draws
    // `41&nbsp;°F`: a temperature that wrapped between the number and the unit
    // would read as two things on a phone.
    return Number.isFinite(value) ? `${value}${unit ? `\u00a0${unit}` : ''}` : '—';
  }
  if (typeof value === 'string') {
    const t = value.trim();
    if (t === '') return '—';
    if (t.toLowerCase() === 'yes') return 'Yes';
    if (t.toLowerCase() === 'no') return 'No';
    if (FREE_TEXT_TYPES.includes(type)) return `“${t}”`;
    return unit ? `${t}\u00a0${unit}` : t;
  }
  if (isPlainish(value)) {
    // HQ bundles a value with its metadata: `{_v, _fail_note}`,
    // `{_v, _correction_photo}`, `{value, sub_steps}` (CLAUDE.md's persistence
    // rule). The answer is the `_v`/`value`; the rest is bookkeeping and is not
    // what was overwritten from the crew member's point of view.
    if ('_v' in value) return formatValue(value._v, opts);
    if ('value' in value) return formatValue(value.value, opts);
  }
  try {
    return JSON.stringify(value);
  } catch (err) {
    return '—';
  }
}

// ---------------------------------------------------------------------------
// Attribution.
// ---------------------------------------------------------------------------

/**
 * A-2.3 — the collapsed view carries the SAME attribution the expanded view
 * does: name AND time, on every `Now shows` row. r1 drew a bare `Dana M.` in
 * the collapsed batch view, so the riskiest action on the sheet carried the
 * least information.
 *
 * A null actor is NOT a bug and must NOT be papered over with an invented name:
 * a server-side or migration touch has no human actor (C1's `describeConflict`
 * returns null for exactly that), and "someone else" is the honest rendering.
 */
export function formatWho(name, at, opts = {}) {
  const who = typeof name === 'string' && name.trim() !== '' ? name.trim() : 'someone else';
  const time = formatClock(at, opts);
  return time ? `${who}, ${time}` : who;
}

export function formatClock(at, opts = {}) {
  if (!at) return '';
  const d = at instanceof Date ? at : new Date(at);
  if (Number.isNaN(d.getTime())) return '';
  try {
    return d.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      timeZone: opts.timeZone || 'America/New_York',
    });
  } catch (err) {
    return '';
  }
}

function formatDay(at, opts = {}) {
  if (!at) return '';
  const d = at instanceof Date ? at : new Date(at);
  if (Number.isNaN(d.getTime())) return '';
  try {
    return d.toLocaleDateString('en-US', {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
      timeZone: opts.timeZone || 'America/New_York',
    });
  } catch (err) {
    return '';
  }
}

// ---------------------------------------------------------------------------
// Retention.
// ---------------------------------------------------------------------------

/**
 * Records inside the window, newest first. Decision 96's window, read from the
 * one constant.
 */
export function withinRetention(records, opts = {}) {
  const days = typeof opts.retentionDays === 'number' ? opts.retentionDays : RETENTION_DAYS;
  const now = opts.now ? new Date(opts.now).getTime() : Date.now();
  const floor = now - days * MS_PER_DAY;
  return (records || []).filter((r) => {
    if (!r) return false;
    const t = new Date(r.overwritten_at).getTime();
    if (Number.isNaN(t)) return true; // an unreadable stamp is kept, not silently swept
    return t >= floor;
  });
}

// ---------------------------------------------------------------------------
// The sheet model. This is what the renderer draws and what the tests assert.
// ---------------------------------------------------------------------------

/**
 * @param {Array<object>} records  durable conflict records
 * @param {object} [opts] `{now, retentionDays, maxGroups, timeZone, snapshots}`
 *   `snapshots` is `{[submissionId]: template_snapshot}` — the LIVE label
 *   source. A record's own frozen `field_label` is the fallback beneath it and
 *   the raw field id is the fallback beneath that (A-3).
 */
export function buildSheetModel(records, opts = {}) {
  const maxGroups = typeof opts.maxGroups === 'number' ? opts.maxGroups : MAX_GROUPS;
  const live = withinRetention(records, opts);

  // Group by document. The grouping is not cosmetic — it is the shape
  // `conflict$` arrives in (one event per document).
  const byDoc = new Map();
  for (const rec of live) {
    const key = rec.doc_id || rec.submission_id || rec.id;
    if (!byDoc.has(key)) byDoc.set(key, []);
    byDoc.get(key).push(rec);
  }

  const groups = [];
  for (const [docId, recs] of byDoc) {
    recs.sort((a, b) => String(a.overwritten_at).localeCompare(String(b.overwritten_at)));
    const rows = recs.map((rec) => buildRow(rec, opts));
    const answers = rows.filter((r) => r.kind !== KIND_UNIDENTIFIED);
    const extras = rows.filter((r) => r.kind === KIND_UNIDENTIFIED);
    const targets = rows.filter((r) => r.isBatchTarget);
    const first = recs[0] || {};
    groups.push({
      docId,
      name: first.checklist_name || 'Checklist',
      day: first.checklist_date || formatDay(first.overwritten_at, opts),
      docChip: shortDocId(docId),
      chipBase: answers.length,
      chipPlus: extras.length,
      rows,
      // Counting rule 7 — the batch's N is the still-to-review, restorable set.
      batchCount: targets.length,
      batchIds: targets.map((r) => r.id),
      batchWho: batchAttribution(targets).who,
      batchRange: batchAttribution(targets).range,
      newest: recs.reduce(
        (acc, r) => (String(r.overwritten_at) > acc ? String(r.overwritten_at) : acc),
        '',
      ),
    });
  }

  // Two orderings, deliberately, because they answer different questions.
  // WHICH ten survive the cap is decided newest-first — an offline stretch that
  // produced 13 checklists' worth of clashes must not hide tonight's behind
  // last week's. HOW they are drawn is chronological, which is the order the
  // signed `edge-many` plate draws them in (Opening before Closing) and the
  // order a shift actually happened in.
  groups.sort((a, b) => b.newest.localeCompare(a.newest));

  const allRows = groups.flatMap((g) => g.rows);
  const answerRows = allRows.filter((r) => r.kind !== KIND_UNIDENTIFIED);
  const totals = {
    // rule 1 — what was overwritten in the window, across the WHOLE sheet,
    // including the groups below the cap line (decision 97).
    answers: answerRows.length,
    // rule 6
    still: answerRows.filter((r) => !r.handled).length,
    handled: answerRows.filter((r) => r.handled).length,
    // rule 4 — counted separately, never folded into either figure above
    unidentified: allRows.length - answerRows.length,
    checklists: groups.length,
  };

  const shown = groups.slice(0, maxGroups).sort((a, b) => a.newest.localeCompare(b.newest));
  const hidden = groups.slice(maxGroups);

  return {
    status: live.length === 0 ? 'empty' : 'ready',
    retentionDays: typeof opts.retentionDays === 'number' ? opts.retentionDays : RETENTION_DAYS,
    totals,
    groups: shown,
    hiddenGroups: hidden.length,
    hiddenAnswers: hidden.reduce((n, g) => n + g.chipBase + g.chipPlus, 0),
    // Collapse when the sheet holds more than one group OR any group holds more
    // than two rows — and then it applies to the whole sheet.
    collapsed:
      groups.length > 1 || groups.some((g) => g.rows.length > COLLAPSE_MAX_ROWS_PER_GROUP),
  };
}

// The group header's monospace chip. A submission id is a uuid; the chip exists
// so a crew member can quote the row to a manager, not so it can be retyped.
function shortDocId(docId) {
  const s = String(docId || '');
  if (s.length <= 12) return s;
  return `sub_${s.replace(/^sub[-_]/i, '').replace(/-/g, '').slice(0, 6)}`;
}

/**
 * A-2's batch attribution, as two parts rather than one string.
 *
 * The name and the time range are separated because the label reads
 * "replaces 3 of Dana M.'s answers · 6:12–6:14 PM" — gluing them produced
 * "3 of Dana M., 6:12 PM–6:14 PM's answers", which is what the first render
 * back showed and is not English.
 */
function batchAttribution(targets) {
  const names = [...new Set(targets.map((t) => t.whoName).filter(Boolean))];
  const times = targets.map((t) => t.whoTime).filter(Boolean);
  const who = names.length === 1 ? names[0] : names.length === 0 ? 'someone else' : 'others';
  let range = '';
  if (times.length === 1 || (times.length > 1 && times[0] === times[times.length - 1])) {
    range = times[0];
  } else if (times.length > 1) {
    const a = times[0];
    const b = times[times.length - 1];
    // "6:12–6:14 PM", not "6:12 PM–6:14 PM", when the meridiem is shared —
    // the mockup's own wording, and one fewer thing to read at 6am.
    const ma = a.slice(-2);
    range = ma === b.slice(-2) ? `${a.slice(0, -3)}–${b}` : `${a}–${b}`;
  }
  return { who, range };
}

function buildRow(rec, opts) {
  const kind = rowKind(rec);
  const snapshot = (opts.snapshots || {})[rec.submission_id];
  const found = findFieldInSnapshot(snapshot, rec.field_id);

  // A-3's fallback ladder, in order:
  //   1. the live/frozen snapshot's own label for this field id
  //   2. the label frozen onto the record when it was written
  //   3. the raw field id  — and ONLY here, rendered exactly as r2 drew it
  const label =
    (found && typeof found.label === 'string' && found.label.trim() !== ''
      ? found.label.trim()
      : null)
    || (typeof rec.field_label === 'string' && rec.field_label.trim() !== ''
      ? rec.field_label.trim()
      : null);
  const type = (found && typeof found.type === 'string' ? found.type : null) || rec.field_type;
  const vopts = { type, unit: rec.display_unit };

  const handled = isHandled(rec);
  return {
    id: rec.id,
    docId: rec.doc_id || rec.submission_id,
    fieldId: rec.field_id,
    kind,
    // `labelSource` is what the renderer keys the A-3 rendering off, and what
    // the tests assert: 'snapshot' draws the struck-through question title,
    // 'fallback' draws the muted monospace raw id exactly as r2 drew it.
    labelSource: label ? 'snapshot' : 'fallback',
    label:
      kind === KIND_UNIDENTIFIED
        ? "A change we couldn't identify"
        : label || rec.field_id,
    yours: kind === KIND_UNIDENTIFIED ? null : formatValue(rec.discarded_value, vopts),
    // The UNFORMATTED discarded value, for the restore write. `yours` is what a
    // crew member reads ("38 °F", "Yes"); writing that string back would turn a
    // number into a string and a boolean into a word — the restore has to put
    // back what was actually there.
    rawDiscarded: rec.discarded_value,
    theirs: formatValue(rec.current_value, vopts),
    hasTheirs: rec.current_value !== undefined,
    whoName:
      typeof rec.overwritten_by_name === 'string' && rec.overwritten_by_name.trim() !== ''
        ? rec.overwritten_by_name.trim()
        : null,
    whoTime: formatClock(rec.overwritten_at, opts),
    who: formatWho(rec.overwritten_by_name, rec.overwritten_at, opts),
    status: rec.status || 'open',
    handled,
    undone: !!rec.undone,
    failure: rec.failure || null,
    isBatchTarget: isBatchTarget(rec),
  };
}

// ---------------------------------------------------------------------------
// Banner copy. A-1 — TWO figures, and a third line when there is one to carry.
// ---------------------------------------------------------------------------

/**
 * The banner, or `null` when there is nothing to say.
 *
 * 🛑 A-1's whole content: `headline` and `open` are two DIFFERENT questions and
 * r1 answered both with one number. The headline reports what was taken from
 * the crew member in the retention window and is stable under Restore/Keep; the
 * open line reports what is still unhandled and is the one that moves. A run
 * that prints one of them reinstates the defect decision 82 was filed against.
 */
export function bannerModel(model) {
  if (!model || model.status !== 'ready') return null;
  const t = model.totals;
  if (t.answers === 0 && t.unidentified === 0) return null;

  const headline = t.answers === 1
    ? '1 answer was overwritten'
    : `${t.answers} answers were overwritten`;

  let open;
  if (t.answers === 0) open = 'Nothing of yours to review';
  else if (t.still === 0) open = `All ${t.answers} reviewed`;
  else if (t.handled === 0) open = `${t.still} still to review`;
  else open = { lead: `${t.still} still to review`, hand: `· ${t.handled} handled` };

  const unid = t.unidentified === 0
    ? null
    : t.unidentified === 1
      ? "+ 1 change we couldn't identify"
      : `+ ${t.unidentified} changes we couldn't identify`;

  const cause = t.checklists > 1
    ? `Across ${t.checklists} checklists, while you were offline.`
    : 'Someone edited them while your phone was offline.';

  return { headline, open, unid, cause };
}

// ---------------------------------------------------------------------------
// The durable store.
//
// 🛑 THE PRECONDITION, STATED ONCE AND LOUDLY (UI-SPEC "the recovery path"):
// `conflict$` is a plain RxJS `Subject` — no replay, no buffer — and RxDB
// persists nothing about a resolved conflict. The discarded value must be
// written to durable local storage THE INSTANT the event arrives or a reload
// destroys the thing this whole screen exists to recover.
//
// Decision 89 puts that record in a LOCAL-ONLY RxDB collection: personal undo,
// per device, no server table, no endpoint, no replication of the record
// itself. `createRxdbConflictStore` is that binding. `createMemoryConflictStore`
// is the same interface over an array — it is what the state-enumeration spec
// drives, and it is why every state below can be forced without IndexedDB.
// ---------------------------------------------------------------------------

/** The record a `describeConflict()` clash becomes. */
export function recordFromClash(described, clash, ctx = {}) {
  const docId = described.id;
  return {
    id: recordIdFor(docId, clash.field),
    doc_id: String(docId || ''),
    collection: ctx.collection || null,
    submission_id: ctx.submissionId || described.master?.submission_id || null,
    field_id: clash.field,
    field_label: ctx.fieldLabel || null,
    field_type: ctx.fieldType || null,
    display_unit: ctx.displayUnit || null,
    field_removed: !!ctx.fieldRemoved,
    checklist_name: ctx.checklistName || null,
    checklist_date: ctx.checklistDate || null,
    discarded_value: clash.discarded === undefined ? null : clash.discarded,
    current_value: clash.winner === undefined ? null : clash.winner,
    overwritten_by: described.winnerActor || null,
    overwritten_by_name: ctx.actorName || null,
    overwritten_at: described.winnerAt || ctx.now || new Date().toISOString(),
    status: 'open',
    undone: false,
    failure: null,
  };
}

/**
 * Every clash in one `conflict$` emission, as records.
 *
 * Counting rule 5 is honoured by `recordIdFor`: writing these with an upsert
 * REPLACES an existing record for the same document+field rather than adding a
 * second row, which is what keeps the count stable under Retry.
 */
export function recordsFromConflict(described, ctx = {}) {
  if (!described || !Array.isArray(described.clashes)) return [];
  return described.clashes.map((c) => recordFromClash(described, c, ctx));
}

/** An array-backed store. Same interface as the RxDB one. */
export function createMemoryConflictStore(seed = []) {
  let rows = seed.map((r) => ({ ...r }));
  const subs = new Set();
  const emit = () => subs.forEach((fn) => fn(rows.map((r) => ({ ...r }))));
  return {
    async all() {
      return rows.map((r) => ({ ...r }));
    },
    async upsert(rec) {
      const i = rows.findIndex((r) => r.id === rec.id);
      if (i >= 0) rows[i] = { ...rec };
      else rows.push({ ...rec });
      emit();
    },
    async patch(id, fields) {
      const i = rows.findIndex((r) => r.id === id);
      if (i < 0) return;
      rows[i] = { ...rows[i], ...fields };
      emit();
    },
    async remove(id) {
      rows = rows.filter((r) => r.id !== id);
      emit();
    },
    subscribe(fn) {
      subs.add(fn);
      return () => subs.delete(fn);
    },
  };
}

/**
 * The decision-89 store: RxDB's local-only `conflict_records` collection.
 *
 * Every method is wrapped, because an evicted or unreadable IndexedDB is a
 * DESIGNED state of this UI (the storage-error plate), not a stack trace. A
 * throw here surfaces as that plate; it must never take `workflows.html` down.
 */
export function createRxdbConflictStore(collection) {
  return {
    async all() {
      const docs = await collection.find().exec();
      return docs.map((d) => (typeof d.toJSON === 'function' ? d.toJSON() : d));
    },
    async upsert(rec) {
      await collection.upsert(rec);
    },
    async patch(id, fields) {
      const doc = await collection.findOne(id).exec();
      if (doc) await doc.patch(fields);
    },
    async remove(id) {
      const doc = await collection.findOne(id).exec();
      if (doc) await doc.remove();
    },
    subscribe(fn) {
      const sub = collection.find().$.subscribe((docs) => {
        fn(docs.map((d) => (typeof d.toJSON === 'function' ? d.toJSON() : d)));
      });
      return () => sub.unsubscribe();
    },
  };
}

/**
 * Decision 96's sweep. LOCAL — a client-side delete of records past the window.
 * There is no server-side retention job because there is no server table.
 */
export async function sweepExpired(store, opts = {}) {
  const all = await store.all();
  const keep = new Set(withinRetention(all, opts).map((r) => r.id));
  let n = 0;
  for (const r of all) {
    if (!keep.has(r.id)) {
      await store.remove(r.id);
      n += 1;
    }
  }
  return n;
}
