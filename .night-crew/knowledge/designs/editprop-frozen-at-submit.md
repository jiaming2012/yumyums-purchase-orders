# Design — Frozen-at-submit edit propagation (`editprop-openspec-design`)

> **Cycle:** "Nothing silently lost" — checklist data integrity (opened 2026-07-16).
> **Role:** the roadmap **Activity 4 design gate** and PRD **FR-1**. This is the second
> attended gate — the PRD sign-off (2026-07-16 evening) did NOT pre-sign it. **0 Activity-5
> build cards dispatch before this design is operator-signed** (auditable from ledger
> timestamps; the slate that dispatches them carries a later sign-off).
> **Traces to:** PRD `.night-crew/knowledge/prds/PRD-data-integrity.md` (FR-1…FR-7, the
> engine-trust FR-8/FR-9 are separate Activity-2 cards, not gated by this design).
> **Semantic (INV-3, ratified at PRD sign-off):** **frozen-at-submit.** An unsubmitted
> checklist always shows the current template on every device; submit freezes the record
> forever; rejection reopens it live.
> **Format note:** OpenSpec is not initialized in this repo and no prior night-crew HQ cycle
> used it; this durable design doc is the signed gate artifact. Activity-5 cards implement
> against §7's decomposition with atomic commits + roadmap flips (the HQ run convention), not
> an `openspec/changes/` tree. If OpenSpec is later adopted here, this doc is the source to
> formalize from.

---

## 1. The model — three record states

A checklist record (one crew member's run of one template for one day) is in exactly one state:

| State | Shape shown | Edits propagate? | Writes accepted? |
|---|---|---|---|
| **Unsubmitted (live)** | the **current** template, always, on every open device | **Yes** — live re-render (§4) | Yes, against the current template (§3) |
| **Submitted (frozen)** | the `template_snapshot` captured at submit — byte-stable forever | **No** — later edits never touch it | No |
| **Rejected (reopened, live)** | back to the current template, prior answers carried, rejection flags shown | **Yes** — same as unsubmitted | Yes |

Transitions: `unsubmitted → submit → frozen`; `frozen → reject → unsubmitted(live, flagged)`;
`unsubmitted → resubmit → frozen` (re-freezes a fresh snapshot). There is no fourth state and
no versioning entity — the versioning schema was weighed head-to-head and demoted to backlog
(PRD §Out of scope).

## 2. Stable field identity (FR-2, INV-2)

- `updateTemplate` **upserts by the field IDs the Builder already sends** (`toApiTemplate`
  includes them): **update** kept fields in place, **insert** genuinely new ones, **delete**
  removed ones. Condition remap applies to **new** fields only.
- The `replaceTemplate` delete-and-reinsert path (`repository.go:99-219`) is **deleted from the
  codebase** — not left dormant.
- Consequence: a field that survives an edit keeps one permanent `checklist_fields.id` for life,
  so multi-device writes always land on the same real field. Field-ID churn — the Friday P0 root
  cause — becomes structurally impossible.

## 3. Loud rejection (FR-3, INV-4, INV-1)

- A write (`/saveResponse` / `SET_FIELD` op) naming a field **absent from the current template**
  is rejected with a **distinct envelope `{"error":"unknown_field"}`, HTTP 422**, via an
  **app-level existence check** — **not** a restored FK (submitted responses reference
  `template_snapshot` IDs by design; an FK would break them).
- The runner **surfaces** the rejection: **no optimistic checkmark survives a rejected save.**
  The field visibly returns to not-saved; the crew member is told it failed.

## 4. Edit propagation & the re-render contract (FR-4, INV-3, INV-6)

- Clients **handle `SAVE_TEMPLATE` ops in `applyOp`** (they already arrive via live WS and
  `wsCatchUp` replay and are currently **ignored** — `sync.js:401-449`): **re-fetch** the
  template, **re-render** the open unsubmitted checklist to the new shape.
- **"Surviving answer" per field type** — a field that keeps its ID keeps its rendered answer
  across the re-render, for **all 7 persisted types + sub-steps**: checkbox, yes/no, text,
  temperature, sub-steps (each sub-item's checked state), fail-note text, fail severity, and the
  photo-URL value. A cut field's answer is discarded (§4 warning). A new field renders empty.
- **Silent on catch-up replay** — a re-render driven by `wsCatchUp` bulk replay shows **no
  toast** (the `42eeb39` no-toast rule); only genuinely live teammate edits may surface UI.
- **Builder discard warning (INV-6)** — before a save that **cuts a field** (or a schedule change
  that drops today — §5) **while today's unsubmitted answers exist on it**, the Builder warns the
  admin **naming the count** ("N crew have unsubmitted answers on fields you're removing — saving
  discards them"). Proceeding is an **explicit, warned operator action** — the INV-1 third branch.

## 5. Schedule-change contract (C5 — operator decision 2026-07-16: **warned live removal**)

A schedule edit is **just another live edit** — no day-scoped special case.

- Scenario: a checklist scheduled Mon–Fri; it's Tuesday mid-shift; a crew member has it open,
  half-filled, unsubmitted. The owner edits the schedule to drop Tuesday and saves.
- **Decision:** dropping today from the schedule is treated **exactly like cutting a field**. If
  any crew member has the checklist **open today with unsubmitted answers**, the Builder **warns
  the admin first**, naming the count ("N crew have this open today — saving removes it from their
  devices and discards their unsubmitted answers"). On proceed, the checklist is **removed live**
  from those devices (a `SAVE_TEMPLATE`/schedule op the runner honors in `applyOp`).
- **Rationale:** consistent with the live-edit model and INV-1 — loss is only ever an explicit,
  warned operator action; the crew is never silently yanked without the admin being told the cost.
- **Corollary:** a schedule change that does **not** drop today (e.g. adds Saturday) needs no
  warning and simply propagates.

## 6. Transactional op emission, lifecycle, convergence, races

- **Transactional op emission (FR-5, INV-1):** the op row **commits in the same transaction** as
  the business write it describes — replacing the fire-and-forget goroutine (`EmitOp`,
  `sync/ops.go:245-264`). **0 accepted writes whose op is not durably queued** for other devices.
- **Frozen-at-submit lifecycle (FR-6, INV-3):** submit freezes the existing `template_snapshot`
  (kept, proven by LC-02); a submitted record's rendered review is **byte-identical** to what was
  submitted regardless of later edits; a **rejection reopens live** against the current template,
  prior answers carried, rejection flags shown; **flags on since-cut fields dissolve visibly**
  ("1 flagged item was removed"); **resubmit re-freezes** a fresh snapshot.
- **Convergence contract (FR-7, A-5):** "in sync" = **converged within one op round-trip**,
  asserted **on the observing (second) device**, never on the writer's optimism. Surfaces that
  must converge: every field's state, sub-steps, submit/unsubmit transitions, and the **list-view
  progress indicator**. Both **live** and **catch-up** (reconnect) paths.
- **Race contract (edit ↔ write window):** the DB **serializes** — the edit is a single upsert
  transaction (§2), field writes are ops. The server **always judges a write against the current
  committed template:**
  - write commits **after** the edit that cut its field → **422 `unknown_field`** (§3, loud);
  - write commits **before** the edit → it lands, then the edit discards it under the **INV-6
    warned-discard** rule (§4);
  - FR-5's transactionality guarantees **no torn state** (no accepted write without its op).
  No new operator decision — the outcomes are already covered by FR-3 + INV-6 + FR-5.
- **Day boundary (INV-5, unchanged):** unsubmitted drafts expire at the day boundary **by
  design** (`cleanupOldDrafts`, `repository.go:977-987`). Named, kept, not changed here.

## 7. Build decomposition — the three Activity-5 cards this design authorizes

Serialized (all touch the workflow-engine / sync surfaces); land in this order:

1. **`editprop-stable-field-identity`** — §2 + §3. `updateTemplate` diff/upsert by Builder IDs;
   delete the `replaceTemplate` reinsert path; app-level existence check → 422 `unknown_field`;
   runner surfaces the rejection (no optimistic check survives). Red-first: the rewritten
   `repro-cut-task.spec.js` post-edit identity checks + a Go test for the 422.
2. **`editprop-broadcast-rerender`** — §4 + §5 + §6 (FR-5 transactional emission). `SAVE_TEMPLATE`
   handled in `applyOp` (re-fetch + re-render, surviving answers intact, silent on catch-up);
   Builder discard warning for cut fields **and** the schedule-drops-today case (§5); op emission
   moved into the write transaction. Red-first per sub-behavior.
3. **`editprop-convergence-matrix`** — §6 convergence contract. The red-first two-device E2E
   matrix (7 field types + sub-steps + submit/unsubmit + list-view progress, live + catch-up) via
   the rewritten `repro-cut-task.spec.js`, including the ≥2 semantic acceptance tests (AC-6a
   mid-run edit re-renders open devices answers-intact; AC-6b submitted record unaffected).

Traceability: card 1 → Eng "stable identity" + "loud rejection"; card 2 → Eng "edit
propagation"; card 3 → Eng "convergence matrix", Product KR-2, Delivery "repro red→green pair".

## 8. Explicitly out of scope (unchanged from PRD)

- Run-pinned immutable template versions (weighed, demoted to backlog). No schema migration.
- Recovering/auditing historical stranded prod rows (no active users; FR-2/FR-3 stop new
  stranding).
- The engine-trust fixes FR-8 (`engine-approval-feedback-loud`) and FR-9
  (`engine-conflict-refetch`) — real fixes, but **not gated by this design**; they are Activity-2
  cards and may run in parallel.

## 9. Sign-off

**What you get:** one clear rule for editing a live checklist — the crew always sees your latest
version until they submit; once they submit, their record is frozen exactly as they sent it;
if you reject it, it reopens on the current version with their answers kept. Every way the old
engine could quietly drop a checkmark is now either impossible, a loud error, or a change you're
warned about before it happens.

**What changes day-to-day:** when you edit a template mid-shift, open crew phones refresh to the
new shape and keep everything they'd already filled in. If your edit removes a field — or drops
today from the schedule — while someone's mid-run, you get a warning naming how many people it
affects before it saves.

**Trade-offs:** a schedule change that drops today *can* remove an in-progress checklist from a
crew phone — but only after you're warned and choose to proceed (never silently). No immutable
version history this cycle (chosen against; backlogged).

**Not covered yet:** run-pinned versioning; the two engine-trust fixes ride their own cards.

**Assumptions carried:** A-1 (frozen-at-submit), A-2 (rejection lifecycle), A-3 (cut-field
discard warning), A-5 (converged-within-one-round-trip), plus the **C5 = warned-live-removal**
decision recorded in §5. Race handling stated as a contract (§6), no open fork.

**Status:** ✅ **SIGNED OFF by operator 2026-07-16 evening** ("sign off, then finish the
slate"). This timestamp is the Activity-4 gate — it precedes every Activity-5 build dispatch
(the overnight run's cards carry a later sign-off; auditable from ledger/commit timestamps per
FR-1 / AC-1). The C5 fork resolved to **warned live removal** (§5); no forks remain open.
