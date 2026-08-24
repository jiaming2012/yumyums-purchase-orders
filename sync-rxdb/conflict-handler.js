// sync-rxdb/conflict-handler.js — HQ's field-level three-way merge.
//
// Card `sync-rxdb-replication-and-conflict-handler` (overnight-20260801, C1).
// Ledger T-22 **decision 50**.
//
// ===========================================================================
// WHAT THIS REPLACES, AND WHY.
// ===========================================================================
// RxDB 17.4.0's default conflict handler is, in full:
//
//     { isEqual(a,b,ctx){ …deep-equal… }, resolve(i){ return i.realMasterState } }
//
// That is UNCONDITIONAL MASTER-WINS — not the last-write-wins the 2026-07-24
// explore session signed off on. No clock participates, and a strictly-LATER
// local write is discarded silently. Reproduced five times: four by the spike's
// `proof-lww.js`, once more in `tests/sync-rxdb-conflict.spec.js` directly
// against the shipped bundle.
//
// For a food-truck PWA whose whole reason for adopting RxDB is offline
// correctness, that means a crew member's 20 minutes of offline work vanishes
// the moment someone in the office touches the same row — and the close record
// then attributes the crew member's work to whoever edited from the office,
// which is the opposite of the product's stated core value.
//
// Decision 50's policy, implemented here:
//
//   * different fields edited by different people **all survive**;
//   * only a genuine SAME-FIELD clash falls back to master-wins, and then the
//     discarded value is **recoverable** — surfaced through `conflict$`.
//
// Genuine LWW was considered and rejected at triage: it is symmetric loss (it
// drops the manager's correction instead) and it makes each phone's clock the
// tiebreaker, reintroducing the skew risk the server's trigger-stamped
// `_modified` currently avoids. **Nothing in this file reads a timestamp for
// any decision.** `tests/sync-rxdb-conflict.spec.js` swings the fork's
// timestamps from 1999 to 2099 and asserts the outcome does not move.
//
// ===========================================================================
// THE MECHANISM.
// ===========================================================================
// `RxConflictHandlerInput` carries THREE states, not two
// (`rxdb/dist/types/types/conflict-handling.d.ts:10`):
//
//     assumedMasterState  A — what the fork believed master held when it forked
//     newDocumentState    F — the local fork's state (the write being pushed)
//     realMasterState     M — what master actually holds now
//
// With A in hand, "who changed what" is decidable per field:
//
//     forkChanged   = F[k] ≠ A[k]
//     masterChanged = M[k] ≠ A[k]
//
//     fork only        → F[k] wins   (the edit nobody contested survives)
//     master only      → M[k] wins
//     neither          → M[k]        (identical anyway)
//     both, same value → M[k]        (convergent — not a clash)
//     both, different  → M[k] wins AND the clash is RECORDED   ← decision 50
//
// The resolved document is `{...M, ...winners}`, so RxDB's own protocol fields
// come from master. RxDB overwrites `_meta`, `_rev` and `_attachments` on the
// output regardless (`upstream.js`, the `xv()` wrapper), so this file must not
// try to own them.
//
// ===========================================================================
// 🛑 THE assumedMasterState-ABSENT FALLBACK — STATED, NOT ASSUMED.
// ===========================================================================
// `assumedMasterState` is declared OPTIONAL in the type. The card required a
// DEFINED fallback rather than an assumption of presence, and required it to be
// a mechanism justifiable from decision 50 rather than a fresh product call.
//
// **THE RULE: with no baseline, master wins on every field, and EVERY field
// where the fork differs from master is recorded as a clash.**
//
// It is not a new policy — it is decision 50's own same-field-clash branch
// applied to the whole document. Without A, no field can be SHOWN to be
// uncontested, so treating each differing field as contested is the honest
// reading. Note the two properties that make it the right degradation:
//
//   * the WINNER is byte-identical to what RxDB's default would have chosen,
//     so this branch is never *worse* than the engine's own behaviour;
//   * every discarded value is still recoverable, so it is strictly *better*
//     than the default on the one axis decision 50 cares about.
//
// The alternative — merging optimistically without a baseline (letting the fork
// win where the two differ) — was rejected: with no A there is nothing to
// distinguish "the fork edited this" from "master edited this", so it would be
// inventing last-write-wins with no clock at all, which is the thing decision 50
// explicitly overturned.
//
// ===========================================================================
// WHAT IS DELIBERATELY NOT MERGED.
// ===========================================================================
// RESERVED_FIELDS — RxDB protocol substrate. Always taken from master, never
// reported. `_modified` is in the list belt-and-braces: decision 78 keeps it out
// of every declared schema, so it should never reach a document at all.
//
// PROVENANCE_FIELDS — decision 79's who-and-when. They are merged by the NORMAL
// rule (so a row the fork alone touched keeps the fork author's name) but are
// never REPORTED as a clash. Both sides always rewrite them whenever both sides
// edit, so reporting them would put a row on C2's sheet for every single merge,
// from which nothing recoverable was lost — the "a change we couldn't identify"
// noise decision 78 was already at pains to avoid.
//
// ===========================================================================
// 🛑 OPEN QUESTION INHERITED BY `sync-hard-cutover` — DELETES.
// ===========================================================================
// `_deleted` is NOT reserved: it is a real user act, so it merges by the
// ordinary rule. That is consistent, and it has one consequence decision 50
// does not cover, because decision 50 is written about FIELD edits and a delete
// is not a field edit:
//
//   fork sets `_deleted: true`, master edits some other field
//     → the delete is UNCONTESTED, so it survives — and master's edit lands on
//       a tombstone, i.e. it is annihilated, and NOTHING IS REPORTED because no
//       field clashed.
//
// This is the one silent-loss path left in the rule, and it is written down
// here rather than fixed for two reasons:
//
//   1. IT IS UNREACHABLE TODAY, ON EXACTLY ONE GROUND: **no HQ page writes
//      through RxDB.** This card is import + construction only, so no fork is
//      ever produced, so no fork can carry `_deleted: true`. That is the whole
//      of the reason. Nothing else holds this shut.
//
//      🛑 DO NOT re-derive the second ground an earlier draft of this comment
//      claimed — that HQ's domain hard-deletes none of the four mirrored
//      tables. THAT IS FALSE. HQ hard-deletes THREE of the four, from live
//      production paths:
//
//        * `saveResponse` (`backend/internal/workflow/repository.go:811`,
//          `POST /api/v1/workflow/saveResponse`, `main.go:558`) runs
//          `DELETE FROM submission_responses WHERE field_id=$1
//          AND answered_by=$2 AND submission_id IS NULL` whenever the value is
//          null — **that is unchecking a checkbox**, the highest-frequency
//          write in the workflows tool and the exact path CLAUDE.md's
//          persistence rule is built around. Responses are NOT merely upserted.
//        * `unsubmitChecklist` (`repository.go:1289` then `:1297`,
//          `POST /api/v1/workflow/unsubmitChecklist`, `main.go:563`) deletes
//          the `submission_rejections` rows and then the
//          `checklist_submissions` row. Submissions and rejections ARE deleted.
//        * a template edit that removes fields deletes the draft
//          `submission_responses` (`repository.go:321` — the "discards their
//          unsubmitted answers" warning at `workflows.html:1397`); and
//          `cleanupOldDrafts` (`repository.go:1334`) sweeps yesterday's drafts
//          on a schedule.
//
//      Only `checklist_templates` is delete-free (it ARCHIVES via
//      `archived_at` / `archiveTemplate`).
//
//      🛑 SO PRICE THE OPEN QUESTION ACCORDINGLY. The moment
//      `sync-hard-cutover` opens a write path this stops being unreachable
//      immediately, and it is not an exotic edge: on this schema an offline
//      crew member UNCHECKING A BOX while a manager edits the same response
//      row annihilates the manager's edit and reports nothing. That is the
//      MOST LIKELY conflict in the product, not a corner case.
//   2. "Should an intentional delete beat a concurrent edit, or should the edit
//      block the delete?" IS A PRODUCT QUESTION, and this card has no standing
//      to answer it. It belongs to `sync-hard-cutover`, which is the card that
//      introduces the write path — including whatever delete path it introduces.
//
// The current behaviour is pinned by a named test
// (`_deleted participates in the merge — an UNCONTESTED local delete survives`)
// so a future card changing it does so visibly rather than by accident.
//
// Note also, and it is not obvious: WITH a baseline a `_deleted` CLASH is
// unreachable. It is a boolean, so "both sides changed it from the same
// baseline" forces both to have flipped to the same value — the convergent
// case. `_deleted` can only be reported in the no-baseline fallback.
//
// 🛑 KNOWN AND ACCEPTED LIMIT, recorded rather than hidden: the schema
// (decision 79) carries ONE provenance pair per ROW, not per field. So a merged
// row that took `name` from the fork and `requires_approval` from master can
// only name one author, and it names master's. There is no per-field
// provenance to be honest with; inventing one is a schema change and a
// different card. C2's sheet reads the who-and-when of the WINNING write, which
// is what `describeConflict` returns, and that is accurate.
//
// ===========================================================================
// ZERO IMPORTS — ON PURPOSE.
// ===========================================================================
// This file imports nothing, so the whole decision is a pure function of three
// plain objects and the suite that pins it needs no browser, no database, no
// IndexedDB and no replication. "Headless and testable is a REQUIREMENT, not a
// nicety" — C2 builds the user-visible half against this behaviour.

// ---------------------------------------------------------------------------
// Field classes.
// ---------------------------------------------------------------------------

/** RxDB protocol substrate. Taken from master, never merged, never reported. */
export const RESERVED_FIELDS = ['_rev', '_meta', '_attachments', '_modified'];

/**
 * Decision 79's who-and-when, across all four replicated collections. Merged
 * by the normal rule; never reported as a clash. See the header.
 */
export const PROVENANCE_FIELDS = [
  'updated_by', 'updated_at',
  'answered_by', 'answered_at',
  'rejected_by', 'rejected_at',
];

/** The provenance pairs, in the order describeConflict prefers them. */
const PROVENANCE_PAIRS = [
  ['answered_by', 'answered_at'],
  ['rejected_by', 'rejected_at'],
  ['updated_by', 'updated_at'],
];

// ---------------------------------------------------------------------------
// Value equality.
//
// Structural, not referential. Two documents that arrived down two different
// wires are always different objects; comparing by identity would report a
// clash on every field of every conflict.
//
// JSON-shaped only, which is exactly what these collections hold: the four
// schemas declare strings, booleans, numbers, nulls and one polymorphic
// `value` union (`{type:['boolean','number','string','object','null']}`).
// `undefined` is treated as "absent", which is what makes a field the fork
// invented merge correctly.
// ---------------------------------------------------------------------------
export function valuesEqual(a, b) {
  if (a === b) return true;
  if (a === null || b === null || a === undefined || b === undefined) return false;
  if (typeof a !== 'object' || typeof b !== 'object') return false;
  const aArr = Array.isArray(a);
  if (aArr !== Array.isArray(b)) return false;
  if (aArr) {
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i++) if (!valuesEqual(a[i], b[i])) return false;
    return true;
  }
  const ak = Object.keys(a);
  const bk = Object.keys(b);
  if (ak.length !== bk.length) return false;
  for (const k of ak) {
    if (!Object.prototype.hasOwnProperty.call(b, k)) return false;
    if (!valuesEqual(a[k], b[k])) return false;
  }
  return true;
}

function mergeableFields(states, reserved) {
  const keys = new Set();
  for (const s of states) {
    if (!s) continue;
    for (const k of Object.keys(s)) if (!reserved.includes(k)) keys.add(k);
  }
  return [...keys];
}

// ---------------------------------------------------------------------------
// The merge itself.
// ---------------------------------------------------------------------------

/**
 * Resolve one conflict.
 *
 * @param {object} input  RxConflictHandlerInput — `{assumedMasterState?,
 *                        newDocumentState, realMasterState}`.
 * @param {object} [opts] `{reservedFields, provenanceFields, isValueEqual}`.
 * @returns {{document: object, clashes: Array<{field,discarded,winner}>,
 *            baseline: 'assumed-master'|'absent'}}
 *
 * `document` is what RxDB is handed back. `clashes` is what C2 renders — one
 * entry per field where a genuine same-field collision discarded fork work.
 * `baseline` names which of the two rules above produced the answer.
 */
export function resolveConflict(input, opts = {}) {
  const {
    reservedFields = RESERVED_FIELDS,
    provenanceFields = PROVENANCE_FIELDS,
    isValueEqual = valuesEqual,
  } = opts;

  const assumed = input ? input.assumedMasterState : undefined;
  const fork = (input && input.newDocumentState) || {};
  const master = input ? input.realMasterState : undefined;

  // Fail loud rather than guess. RxDB always supplies realMasterState on the
  // conflict path; an absent one means the caller is not RxDB and the whole
  // "master wins" half of the rule has nothing to point at.
  if (!master || typeof master !== 'object') {
    throw new Error(
      'sync-rxdb/conflict-handler: realMasterState is required to resolve a conflict',
    );
  }

  const haveBaseline = !!assumed && typeof assumed === 'object';
  const baseline = haveBaseline ? 'assumed-master' : 'absent';

  const document = Object.assign({}, master);
  const clashes = [];

  for (const k of mergeableFields([assumed, fork, master], reservedFields)) {
    const f = fork[k];
    const m = master[k];
    if (isValueEqual(f, m)) continue; // nothing to decide

    if (!haveBaseline) {
      // 🛑 THE FALLBACK. Master wins on every field — identical winner to
      // RxDB's own default — and every difference is recorded, because with no
      // baseline none of them can be shown to be uncontested. See the header.
      if (!provenanceFields.includes(k)) {
        clashes.push({ field: k, discarded: f, winner: m });
      }
      continue; // `document` already carries master's value
    }

    const a = assumed[k];
    const forkChanged = !isValueEqual(f, a);
    const masterChanged = !isValueEqual(m, a);

    if (forkChanged && !masterChanged) {
      // The edit nobody contested. THIS is the case RxDB's default destroys.
      document[k] = f;
    } else if (!forkChanged && masterChanged) {
      // Already master's — leave it.
    } else if (forkChanged && masterChanged) {
      // A genuine same-field clash. Master wins (decision 50) and the fork's
      // value is recorded so `conflict$` can surface it recoverably.
      if (!provenanceFields.includes(k)) {
        clashes.push({ field: k, discarded: f, winner: m });
      }
    }
    // !forkChanged && !masterChanged is unreachable here: the two would be
    // equal to `a` and therefore to each other, and the isValueEqual(f,m)
    // guard above would already have skipped the field.
  }

  return { document, clashes, baseline };
}

// ---------------------------------------------------------------------------
// The RxDB adapter.
// ---------------------------------------------------------------------------

/**
 * Build the object handed to `addCollections({..., conflictHandler})`.
 *
 * @param {object} [options]
 * @param {function} [options.isEqual]  RxDB's own `defaultConflictHandler.isEqual`.
 *        INJECTED rather than reimplemented: `isEqual` decides whether a
 *        conflict exists at all, and it strips `_meta`/`_attachments` in ways
 *        that are the engine's business, not ours. Redefining it would change
 *        WHEN the handler runs, which is not what decision 50 changed.
 * @param {function} [options.onClash]  called once per clashing field, eagerly,
 *        so C2 can write the local `conflict_records` row without waiting for a
 *        `conflict$` subscription to exist. Optional; `conflict$` +
 *        `describeConflict` is the same information derived later.
 */
export function createHQConflictHandler(options = {}) {
  const { isEqual, onClash, ...mergeOpts } = options;

  return {
    // ─── G6 CORRECTION (C2, overnight-20260801) ───────────────────────────
    // 🛑 THE CONFIGURED OPTIONS ARE PART OF THE HANDLER'S PUBLIC SHAPE.
    //
    // `describeConflict` re-runs `resolveConflict` to derive the clash list C2
    // renders. As first written it ran with ITS OWN `opts`, defaulted to `{}` —
    // so a caller who customised `reservedFields` or `provenanceFields` here got
    // a `conflict$` clash list that DISAGREED with what this handler actually
    // did. Concretely: a custom `provenanceFields` suppresses a clash in
    // `resolve()` and, without threading, that same field came back as a row on
    // the overwritten-answers sheet — a row for a value nothing lost.
    //
    // Exposing them is what lets the subscription thread the SAME options
    // through (`client.js` `startHQReplication`), so the sheet reports what the
    // handler decided rather than what the defaults would have decided. Pinned
    // by `tests/sync-rxdb-conflict.spec.js` with a customised field.
    mergeOpts,

    // Delegated verbatim when supplied. The fallback is only for the headless
    // tests and for a caller that has no engine to borrow from.
    isEqual: isEqual || ((a, b) => valuesEqual(a, b)),

    resolve(input) {
      const { document, clashes, baseline } = resolveConflict(input, mergeOpts);

      if (onClash && clashes.length > 0) {
        const id = documentId(input && input.newDocumentState, mergeOpts.primaryPath);
        for (const c of clashes) {
          // 🛑 The notice is a courtesy. Losing it must never wedge the sync
          // loop — a throwing subscriber would otherwise reject the resolve
          // promise and stall the upstream queue for every document behind it.
          try {
            onClash(Object.assign({ id, baseline }, c));
          } catch (err) {
            if (typeof console !== 'undefined' && console.warn) {
              console.warn('[hq-sync] conflict notice handler threw', err);
            }
          }
        }
      }

      return document;
    },
  };
}

// ---------------------------------------------------------------------------
// `conflict$` — what C2 subscribes to.
//
// The replication state's `conflict$` emits `{input, output}` once PER
// DOCUMENT (verified in the shipped bundle: the emission sits inside
// `Object.entries(conflictsById).map(...)` — `upstream.js:333`). The input
// carries all three states, so the discarded values are derivable from the
// emission alone; no side channel is needed and none is provided.
// ---------------------------------------------------------------------------

function documentId(doc, primaryPath) {
  if (!doc) return undefined;
  return doc[primaryPath || 'id'];
}

/**
 * The merge options a handler was CONSTRUCTED with, for a caller that has the
 * handler but not the options object (which is every caller: the handler is
 * handed to `addCollections` and read back off the collection).
 *
 * Threading these into `describeConflict` is the whole of the G6 correction —
 * without it the sheet and the handler can disagree. Returns `{}` for anything
 * that is not one of ours, which reproduces the previous behaviour exactly for
 * a default-constructed handler.
 */
export function conflictOptsOf(handler) {
  return handler && handler.mergeOpts && typeof handler.mergeOpts === 'object'
    ? handler.mergeOpts
    : {};
}

/**
 * Turn one `conflict$` emission into the row set C2 renders.
 *
 * @returns {{id, baseline, clashes, winnerActor, winnerAt, master, fork, assumed}}
 *
 * `winnerActor`/`winnerAt` are decision 79's who-and-when read off the state
 * that WON, which is what lets the sheet print "Dana M., 6:12 PM" rather than
 * "someone else". They may be null — a server-side or migration touch has no
 * human actor, and a null is what makes the UI honestly say "someone else"
 * instead of inventing a name.
 */
export function describeConflict(emission, opts = {}) {
  const input = (emission && emission.input) || {};
  const master = input.realMasterState || {};
  const { clashes, baseline } = resolveConflict(input, opts);

  let winnerActor = null;
  let winnerAt = null;
  for (const [actorKey, atKey] of PROVENANCE_PAIRS) {
    if (Object.prototype.hasOwnProperty.call(master, actorKey)) {
      winnerActor = master[actorKey] === undefined ? null : master[actorKey];
      winnerAt = master[atKey] === undefined ? null : master[atKey];
      break;
    }
  }

  return {
    id: documentId(input.newDocumentState, opts.primaryPath)
      || documentId(master, opts.primaryPath),
    baseline,
    clashes,
    winnerActor,
    winnerAt,
    assumed: input.assumedMasterState,
    fork: input.newDocumentState,
    master,
  };
}
