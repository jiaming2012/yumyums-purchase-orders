// sync-schema/collections.js — the RxDB collection definitions for HQ's sync
// layer. SCHEMA ONLY.
//
// Card `sync-rxdb-collections-and-table-contract` (overnight-20260729-2, B1),
// fanned out of the dissolved `sync-rxdb-schema-and-replication` (ledger
// decision 81). This file wires no replication, writes no policy, constructs no
// client and imports nothing. It is plain data, on purpose:
//
//   * it can be proven by `tests/sync-schema.spec.js` with no RxDB runtime,
//     which is what keeps this card genuinely "schema only";
//   * it adds no npm dependency. The `@supabase/supabase-js` pin belongs to
//     `sync-rxdb-replication-and-conflict-handler`, and `rxdb` reaches the
//     browser through the committed `vendor/rxdb.bundle.js` (ESM), which this
//     module deliberately does not import;
//   * ESM, because the eventual consumer is a `<script type="module">` in the
//     PWA sitting next to that same bundle.
//
// ===========================================================================
// THE THREE DECLARATIONS THIS FILE OWNS. Other cards READ them and must not
// re-litigate them.
// ===========================================================================
//
// (a) `_modified` is NOT declared — ledger decision 78.
//     Leaving it out keeps it a pure, server-stamped pull cursor. DECLARING it
//     pulls `_modified` into `addDocEqualityToQuery`'s compare-and-swap, so ANY
//     server-side touch becomes a conflict — including ones where no answer
//     changed. Those land in the conflict-notice UI as the "a change we couldn't
//     identify" row: the one row in the whole design from which NOTHING can be
//     recovered (no Restore, only Open checklist and Dismiss). The tightened
//     detection buys little, because the field-level three-way merge on
//     `sync-rxdb-replication-and-conflict-handler` does that work deliberately
//     and with better information. Revisit ONLY if that merge rule proves unable
//     to tell a real same-field clash from a stale fork without it.
//
//     `_deleted` is not declared either, for a different reason: RxDB owns that
//     field. The Supabase replication plugin maps the Postgres column onto
//     RxDB's internal deleted flag; declaring it collides with the plugin.
//
//     Both are asserted as NEGATIVE tests in tests/sync-schema.spec.js so a
//     later card cannot "helpfully" add either back without reddening the suite.
//
// (b) Replicated rows CARRY who-and-when — ledger decision 79.
//     Without it the conflict sheet's "Dana M., 6:12 PM" degrades to "someone
//     else". The product's stated core value is accountability — who checked
//     what. Two of the four collections already had the columns in Postgres
//     (`answered_by`/`answered_at`, `rejected_by`/`rejected_at`); the cost of
//     this decision is therefore THREE new columns in total —
//     `templates.updated_by`, `checklists.updated_by`, `checklists.updated_at`.
//
// (c) The conflict record is a LOCAL collection — ledger T-27 decision 89.
//     Personal undo, per device. NO server table, NO endpoint, NO replication of
//     this collection — which keeps the signed mockup's contract ("no new sync
//     plumbing … no server endpoint") literally true rather than quietly
//     widening it. Its SHAPE is replication-ready, so promoting it to a
//     cross-device audit trail later is adding a table and a policy, not a
//     redesign. The accepted, unmitigated consequences: per-device (a manager
//     cannot see that a crew member's food-safety reading was overwritten),
//     evictable under iOS storage pressure, and lost on reinstall.
//
// ===========================================================================
// WHAT IS MIRRORED, AND WHAT IS DELIBERATELY NOT
// ===========================================================================
//
// The four collections mirror four HQ tables one-for-one:
//
//     templates  → checklist_templates      (backend/internal/db/migrations/0006)
//     checklists → checklist_submissions    (0011)
//     responses  → submission_responses     (0012)
//     approvals  → submission_rejections    (0014)
//
// `approvals` maps to `submission_rejections` because that is the only
// standalone reviewer-decision table HQ has: an approval is recorded as the
// ABSENCE of rejection rows plus `checklist_submissions.status`, while a
// rejection is a per-field row carrying the reviewer's comment. Per-field is
// also the granularity the conflict sheet works at, so the mirror is faithful in
// both directions.
//
// NOT mirrored here, and that is a boundary rather than an omission:
// `checklist_sections`, `checklist_fields`, `checklist_schedules`,
// `template_assignments`, `submission_fail_notes`. The card names four
// collections and these are not among them. The reason it holds up in practice
// is `checklists.template_snapshot`: a submission carries a frozen copy of the
// structure it was filled against, so a crew member filling a checklist offline
// needs no section/field replication at all. `template_assignments` is a
// separate matter — `sync-rxdb-row-visibility-rls` (B2) projects it into the
// sync DB as an RLS input, not as a replicated collection.
//
// NOT carried across, deliberately: `lamport_ts` (0016). It is substrate of the
// op-log/Lamport-clock layer this cycle REPLACES. Under RxDB the pull cursor is
// `_modified` (server-stamped, undeclared) and ordering is the replication
// protocol's business.
//
// ===========================================================================
// THE PARK TRIGGER DID NOT FIRE — recorded so the next reader does not re-ask.
// ===========================================================================
//
// The card's PARK trigger is "a field whose RxDB type has no honest equivalent
// (the float64 money path is the obvious candidate)". None of the four mirrored
// tables carries money — no price, no cost, no currency. HQ's money lives in
// `purchase_*`/`recipes`/`menu_items`, none of which is in this footprint. The
// one place a number crosses this schema is a temperature reading inside
// `responses.value`, which is a physical measurement with no currency
// semantics and round-trips honestly as a JSON number. Nothing was parked.

// ---------------------------------------------------------------------------
// Shared field shapes.
// ---------------------------------------------------------------------------

// HQ ids are Postgres UUIDs (`gen_random_uuid()`), 36 characters as text. The
// sync tables use `id text primary key` rather than `uuid` because RxDB
// documents carry client-generated string ids — a device offline on the truck
// invents the id before any server sees the row. RxDB requires an explicit
// `maxLength` on a string primary key (it sizes the index), and 36 both fits a
// UUID exactly and refuses anything that is not one.
const UUID = { type: 'string', maxLength: 36 };
const NULLABLE_UUID = { type: ['string', 'null'], maxLength: 36 };
const TIMESTAMP = { type: 'string', format: 'date-time' };
const NULLABLE_TIMESTAMP = { type: ['string', 'null'], format: 'date-time' };

// `submission_responses.value` is JSONB and genuinely polymorphic — the fill UI
// writes a boolean (checkbox), a string ('yes'/'no', free text), a number
// (temperature), or an object when a value is bundled with metadata
// (`{_v, _fail_note}`, `{_v, _correction_photo}`, `{value, sub_steps}`).
// JSON Schema expresses that natively with a type union, so no information is
// lost and nothing is coerced. Not indexed, so the union costs nothing.
const JSON_VALUE = { type: ['boolean', 'number', 'string', 'object', 'null'] };

// ---------------------------------------------------------------------------
// The four REPLICATED collections.
//
// `table` is declared rather than inferred: the Supabase replication plugin is
// handed `tableName`, and B2's RLS work needs to name the same table without
// guessing which product-facing collection it belongs to.
// ---------------------------------------------------------------------------

export const REPLICATED_COLLECTIONS = {
  templates: {
    name: 'templates',
    table: 'checklist_templates',
    replicated: true,
    schema: {
      version: 0,
      primaryKey: 'id',
      type: 'object',
      properties: {
        id: UUID,
        name: { type: 'string', maxLength: 200 },
        requires_approval: { type: 'boolean' },
        created_by: NULLABLE_UUID,
        created_at: TIMESTAMP,
        // Decision 79. `updated_at` already existed in Postgres; `updated_by`
        // is new. Nullable because a server-side or migration touch has no
        // human actor — and a null is what makes the UI honestly say "someone
        // else" instead of inventing a name.
        updated_by: NULLABLE_UUID,
        updated_at: TIMESTAMP,
        archived_at: NULLABLE_TIMESTAMP,
      },
      required: ['id', 'name', 'requires_approval', 'created_at', 'updated_by', 'updated_at'],
    },
  },

  checklists: {
    name: 'checklists',
    table: 'checklist_submissions',
    replicated: true,
    schema: {
      version: 0,
      primaryKey: 'id',
      type: 'object',
      properties: {
        id: UUID,
        template_id: UUID,
        // Opaque by design: a frozen copy of the template structure this
        // submission was filled against, so a filled checklist is self-contained
        // offline even after the template is edited (replaceTemplate re-creates
        // fields with new UUIDs — see migrations 0051/0053/0054, which dropped
        // the field_id FKs for exactly that reason). Declared with no nested
        // `properties` on purpose: constraining it here would make this schema a
        // second, drifting definition of the builder's output.
        template_snapshot: { type: 'object' },
        submitted_by: UUID,
        submitted_at: TIMESTAMP,
        status: { type: 'string', enum: ['pending', 'approved', 'rejected', 'completed'] },
        reviewed_by: NULLABLE_UUID,
        reviewed_at: NULLABLE_TIMESTAMP,
        idempotency_key: NULLABLE_UUID,
        // Decision 79 — both new. `submitted_by`/`submitted_at` answer "who
        // submitted", which is NOT the same question as "who last changed this
        // row"; an approver's review changes the row without touching either.
        updated_by: NULLABLE_UUID,
        updated_at: TIMESTAMP,
      },
      required: [
        'id', 'template_id', 'template_snapshot', 'submitted_by', 'submitted_at',
        'status', 'updated_by', 'updated_at',
      ],
    },
  },

  responses: {
    name: 'responses',
    table: 'submission_responses',
    replicated: true,
    schema: {
      version: 0,
      primaryKey: 'id',
      type: 'object',
      properties: {
        id: UUID,
        // NULLABLE, and this is load-bearing: a draft response has no submission
        // yet (migration 0012's partial unique index on
        // `(field_id, answered_by) WHERE submission_id IS NULL` is the draft
        // path). Drafts are precisely what a crew member fills offline, so the
        // collection that must sync best is the one whose FK is absent.
        submission_id: NULLABLE_UUID,
        field_id: UUID,
        value: JSON_VALUE,
        // Decision 79 — already present in Postgres, zero new columns. These two
        // are what the conflict sheet reads for "Dana M., 6:12 PM".
        answered_by: UUID,
        answered_at: TIMESTAMP,
      },
      required: ['id', 'field_id', 'value', 'answered_by', 'answered_at'],
    },
  },

  approvals: {
    name: 'approvals',
    table: 'submission_rejections',
    replicated: true,
    schema: {
      version: 0,
      primaryKey: 'id',
      type: 'object',
      properties: {
        id: UUID,
        submission_id: UUID,
        field_id: UUID,
        comment: { type: 'string' },
        require_photo: { type: 'boolean' },
        // Decision 79 — already present in Postgres, zero new columns.
        rejected_by: UUID,
        rejected_at: TIMESTAMP,
      },
      required: [
        'id', 'submission_id', 'field_id', 'comment', 'require_photo',
        'rejected_by', 'rejected_at',
      ],
    },
  },
};

// ---------------------------------------------------------------------------
// Retention for the local conflict record.
//
// ONE named constant, deliberately. The number itself is REOPENED and belongs to
// `sync-rxdb-conflict-notice-mockup-amendments`, which must draw it as a visible
// placeholder rather than a settled fact — so it has to be changeable in exactly
// one place. `tests/sync-schema.spec.js` tokenises every numeric literal in THIS
// FILE and asserts exactly one of them evaluates to the window below — the only
// check that actually enforces that. It compares by value, not by spelling, so a
// second copy written as a float or an exponent is caught too. Write the number
// nowhere else in this file, in code OR in a comment.
//
// The sweep is LOCAL: a client-side delete of records older than this window.
// There is no server-side retention job, because there is no server table.
// ---------------------------------------------------------------------------
export const CONFLICT_RECORD_RETENTION_DAYS = 30;

// ---------------------------------------------------------------------------
// The LOCAL collection — decision 89.
//
// Not in sync-schema/sql/0001_sync_tables.sql, and that absence is the decision.
// ---------------------------------------------------------------------------

export const LOCAL_COLLECTIONS = {
  conflict_records: {
    name: 'conflict_records',
    // No `table` key at all. Not `table: null`, not `table: ''` — absent, so
    // that any code that reaches for a table name to replicate against fails
    // loudly rather than replicating to a table that does not exist.
    replicated: false,
    local: true,
    schema: {
      version: 0,
      primaryKey: 'id',
      type: 'object',
      properties: {
        id: UUID,
        // Nullable for the same reason `responses.submission_id` is: an
        // overwritten DRAFT answer is exactly the case this record exists for.
        submission_id: NULLABLE_UUID,
        field_id: UUID,
        // The value that LOST. Same polymorphic shape as `responses.value`, so
        // Restore is a copy rather than a conversion.
        discarded_value: JSON_VALUE,
        // Decision 79's who-and-when, carried here too — this is the pair the
        // sheet prints as "Dana M., 6:12 PM" on the Now shows line. It names the
        // actor whose write WON, not the person holding the phone.
        overwritten_by: NULLABLE_UUID,
        overwritten_at: TIMESTAMP,

        // ─── ADDED BY `sync-rxdb-conflict-notice-ui` (overnight-20260801, C2) ──
        //
        // Everything above is B1's and is byte-unchanged, including `required`:
        // the three declarations this file's header says other cards must not
        // re-litigate — `_modified`/`_deleted` undeclared, who-and-when carried,
        // the collection LOCAL with no `table` — are all untouched. What follows
        // is the set of fields the sheet cannot be DRAWN without, each with the
        // plate that needs it.
        //
        // All optional. A record written by an older build still reads, and the
        // renderer degrades per field rather than refusing the row.

        // The group key. `conflict$` fires once per DOCUMENT and the sheet
        // groups by document — the grouping is not cosmetic, it is the shape the
        // event arrives in.
        doc_id: { type: 'string', maxLength: 64 },
        // Which replicated collection the document came from, so a future
        // non-response conflict is not silently rendered as an answer.
        collection: { type: ['string', 'null'], maxLength: 40 },

        // 🛑 THE `Now shows` VALUE. Every plate in the signed mockup draws it,
        // A-2's confirm lists it struck through, and r2's schema carried no
        // field for it — the record held only what was LOST, never what won. A
        // sheet that cannot say what the checklist reads now cannot ask anyone
        // to decide whether to replace it.
        current_value: JSON_VALUE,

        // 🛑 AMENDMENT A-3 (ledger T-28 decision 95). A removed question keeps
        // its label, struck through and read-only. The label is frozen off the
        // submission's own `template_snapshot` at the moment the record is
        // written, so the row survives the live template moving on — and so the
        // renderer is not obliged to hold a snapshot to draw a row.
        //
        // NULLABLE ON PURPOSE, and that null is A-3's own fallback: a snapshot
        // that genuinely carries no label for the id renders the raw field id
        // exactly as r2 drew it. Because NOTHING VALIDATES `template_snapshot`
        // (this file's `checklists.template_snapshot` is `{type:'object'}` with
        // no nested `properties` — B1's recorded-not-fixed item R-C, promoted to
        // a dependency by A-3), "malformed" and "carries no label" are the same
        // branch by construction, which is what makes the fallback total.
        field_label: { type: ['string', 'null'], maxLength: 400 },
        // The answer's type and unit, so a temperature reads "38 °F" and a
        // free-text note reads in quotes. Rendering only.
        field_type: { type: ['string', 'null'], maxLength: 40 },
        display_unit: { type: ['string', 'null'], maxLength: 16 },
        // True when the LIVE template no longer contains this field id — the
        // owner edited the template while the phone was offline. It is what
        // selects A-3's struck-through read-only rendering over the ordinary
        // one, and it is stored rather than derived because the live template is
        // not always in hand when the sheet is drawn.
        field_removed: { type: 'boolean' },

        // Group header copy. Read off the submission when the record is written.
        checklist_name: { type: ['string', 'null'], maxLength: 200 },
        checklist_date: { type: ['string', 'null'], maxLength: 40 },
        // The human name behind `overwritten_by`. A-2.3 requires name AND time
        // on every `Now shows` row, including the collapsed one, and a uuid is
        // not an attribution. Null degrades honestly to "someone else" — a
        // server-side or migration touch has no human actor.
        overwritten_by_name: { type: ['string', 'null'], maxLength: 120 },

        // The row's own state. Counting rule 6 reads it: `restored` and `kept`
        // are REVIEWED; `open`, `restoring` and `failed` are still to review.
        // 🛑 There is no `dismissed`: dismissing REMOVES the record, which is
        // the only way a row leaves the sheet other than expiry ((b) STANDS).
        status: { type: 'string', enum: ['open', 'restoring', 'restored', 'failed', 'kept'] },
        // A restored row that was then undone returns to `open` and carries one
        // muted line saying so, because a silent tap is worse than a loud one.
        undone: { type: 'boolean' },
        failure: { type: ['string', 'null'], enum: ['offline', 'conflict', null] },
      },
      required: ['id', 'field_id', 'discarded_value', 'overwritten_by', 'overwritten_at'],
    },
  },
};

// The two field names that must never appear in a declared schema.
//
// NOTHING IMPORTS THIS TODAY, and that is not an oversight — read before wiring
// it up. `tests/sync-schema.spec.js` deliberately keeps its OWN copy of these
// two names (`MUST_NOT_DECLARE`, spec line ~93) instead of importing this list.
// A negative test that reads the forbidden names from the module it is testing
// proves nothing: delete a name here and the test silently stops checking for
// it, still green. The duplication IS the check. Do not "DRY" the two lists
// together.
//
// What this export is for, then: a single written-down declaration that a future
// card adding a fifth collection can check its schema against at RUNTIME (a
// build step, an assertion in the eventual RxDB bootstrap) — a consumer that is
// not the test. If the replication card lands and still nothing imports it,
// delete it rather than leaving a decorative export behind.
export const RESERVED_UNDECLARED_FIELDS = ['_deleted', '_modified'];
