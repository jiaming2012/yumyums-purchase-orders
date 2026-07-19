# PRD — Checklist data integrity (frozen-at-submit edit propagation)

> **Cycle:** "Nothing silently lost" — checklist data integrity (opened 2026-07-16).
> **Role:** the cycle's **blocking gate** (roadmap Activity 1, `prd-data-integrity`):
> no build WO dispatches before this PRD is signed.
> **Trigger:** operator-reported, same-day-reproduced P0 — editing a template while
> crew devices have the checklist open silently discards their work
> (`tests/repro-cut-task.spec.js`; root cause `replaceTemplate` field-ID churn +
> FK-dropped silent dead-id writes, migrations 0051/0053/0054).
> **Semantic decision (2026-07-16 grill-back, operator-delegated → PM-chosen,
> ratified at sign-off):** **frozen-at-submit.** An unsubmitted checklist always shows
> the current template on every device; submit freezes the record forever; rejection
> reopens it live. Weighed head-to-head against run-pinned immutable versions
> (Lumiform-style); chosen because the operator is the editor and wants corrections
> live, the crew is 1–5 people in one kitchen, and this shape delivers the theme with
> no schema migration. The versioning schema is demoted to backlog as a possible
> future evolution. The operator's hard delegation bar: **multi-device sync is the
> product** — all field types, sub-steps, submit/unsubmit, and derived progress bars
> converge across devices, always (FR-7).
> **Enumeration provenance (§15t two-pass):** pass 1 walked the runner/approvals/
> builder UI write paths (`workflows.html`, `sync.js`); pass 2 swept the no-UI
> surfaces — API-only endpoints, the `SAVE_TEMPLATE` op flow, `wsCatchUp` replay,
> fire-and-forget goroutines (`EmitOp`, `cleanupOldDrafts`), FK-drop migrations,
> `template_snapshot`, approval/rejection backend paths. Pass 2 found 6 loss modes
> pass 1 did not; each is routed in §Routing below.
> **Operator Brief:** `.night-crew/runs/2026-07-16-attended/intake/operator-brief.md`
> (CONFIRMED; see its Amendments section for both grill-back reversals).

## Objective

Make silent loss of crew-entered checklist work **impossible, loud, or an explicit
operator action** — nothing in between — under the frozen-at-submit semantic: fields
keep one permanent identity for life, so multi-device writes always land on the same
real field; edits propagate live to every open device with surviving answers intact;
writes the server cannot attach to a live field are rejected loudly; submit freezes
the record forever; and every accepted write is durably queued for other devices in
the same transaction. Prove the whole surface with a multi-device convergence matrix
(all 7 field types, sub-steps, submit/unsubmit, progress indicators). Ship the two
engine-trust fixes the semantic does not subsume (vanishing approval feedback,
stale-render conflict losers). Advances the **Product objective**: KR-1 (this PRD as
blocking gate, 100% requirement trace — §Trace table), KR-2 (the edit semantic
recorded + its ≥2 acceptance tests), KR-3 (routing record — carried in the roadmap,
extended by §Routing). Its requirements are the build denominator the **Delivery**,
**Engineering**, and **QA** objectives grade against.

## Operators & users

- **Crew member (team_member)** — fills checklist fields on a phone mid-shift; the
  victim of every loss mode here: their checkmarks looked saved and were not.
  (No crew is live in prod today — the protection is for the go-live day.)
- **Owner/admin** — edits templates in the Builder, sometimes mid-shift; under the
  chosen semantic their corrections appear live on crew devices, and the Builder
  warns them before a save discards anyone's unsubmitted answers.
- **Manager/approver** — approves/rejects submissions with comments; FR-8's victim
  (feedback comment can vanish behind a success message); rejection reopens the
  checklist live (FR-6).
- **The overnight crew (indirect)** — builds against this PRD's requirements; the
  PjM slates the cards within the signed design-before-build sequence.

## Named invariants

Every requirement below traces to the reproduced failure (`REPRO` —
`tests/repro-cut-task.spec.js`) or to one of these named invariants:

- **INV-1 (no silent loss)** — a value the UI shows as saved is either durably
  stored and visible after reload on every device, or the user is told it failed.
  Loss is only ever: impossible, loud, or an explicit operator action.
- **INV-2 (stable field identity)** — a template edit never changes the identity of
  a field that survives the edit; the server honors Builder-generated IDs forever.
- **INV-3 (edit semantic — operator-delegated, PM-chosen, sign-off-ratified
  2026-07-16)** — *an unsubmitted checklist always shows the current template on
  every device; submit freezes the record forever; rejection reopens it live.*
  This replaces the morning's "crews finish the run they started" semantic —
  reversal recorded in the ledger (G-2) and the Brief amendments.
- **INV-4 (loud rejection)** — the server never silently accepts a write it cannot
  attach to a field in the current template.
- **INV-5 (day-boundary draft expiry is a rule, not an accident)** — unsubmitted
  drafts expire at the day boundary **by design** (`cleanupOldDrafts`,
  `repository.go:977-987`). Named so the expiry is a documented rule; any future
  change to it is a spec change, not a drive-by.
- **INV-6 (explicit discard)** — cutting a field discards its unsubmitted answers;
  that is an operator action, and the Builder says so before it happens.

## Requirements

Every requirement is falsifiable and carries its KR trace inline; the failure/
invariant trace is in §Trace table.

### Design gate

- **FR-1** — An OpenSpec design change for frozen-at-submit edit propagation is
  written and **operator-signed before any build card dispatches** (auditable from
  ledger timestamps). The design must answer: stable field IDs honored forever;
  the re-render contract (surviving answers intact; what "surviving" means per
  field type incl. sub-steps); the rejection rules (frozen record · live redo
  carrying answers · moot flags on cut fields dissolve visibly); the cut-field
  discard warning (INV-6); the convergence contract per surface (fields, progress
  bars, list view); day-boundary schedule-change behavior (the C5 question —
  §Routing); and race handling in the edit→broadcast window. — traces to Delivery
  KR "edit-propagation design signed before build".

### Edit propagation (the structural fix)

- **FR-2** — `updateTemplate` preserves field identity: the server upserts by the
  field IDs the Builder already sends (`toApiTemplate` includes them) — update kept
  fields, insert genuinely new ones, delete removed ones; condition remap applies
  to new fields only. `replaceTemplate`'s delete-and-reinsert
  (`repository.go:99-219`) is replaced by this diff/upsert. *(POST
  `/updateTemplate`)* — traces to Engineering KR "stable identity".
- **FR-3** — A write naming a field absent from the current template is rejected
  with a distinct error envelope (`{"error":"unknown_field"}`, 422) via an
  app-level existence check — **not** a restored FK (submitted responses reference
  `template_snapshot` IDs by design) — and the runner surfaces the rejection: no
  optimistic checkmark survives a rejected save. *(POST `/saveResponse` /
  `SET_FIELD` op; workflows.html error path)* — traces to Engineering KR "loud
  rejection", INV-4, INV-1.
- **FR-4** — Clients handle `SAVE_TEMPLATE` ops in `applyOp` (they already arrive
  via live WS and `wsCatchUp` replay and are currently ignored — `sync.js:401-449`):
  re-fetch the template, re-render the open checklist to the new shape with all
  surviving answers intact, dissolve moot rejection flags visibly, and stay silent
  on catch-up replay (the `42eeb39` no-toast rule). The Builder warns before a save
  that discards today's unsubmitted answers on cut fields (INV-6). *(sync.js +
  workflows.html Builder)* — traces to Engineering KR "edit propagation", INV-3,
  INV-6.
- **FR-5** — Op emission is transactional with the write it describes: the op row
  commits in the same transaction as the business write, replacing the
  fire-and-forget goroutine (`EmitOp`, `sync/ops.go:245-264`) — 0 accepted writes
  whose op is not durably queued for other devices. *(backend sync layer)* — traces
  to Engineering KR "edit propagation", INV-1.
- **FR-6** — Frozen-at-submit lifecycle: a submitted record never changes shape
  (existing `template_snapshot`, kept); a rejection reopens the checklist live
  against the current template with prior answers carried over and rejection flags
  shown; flags on since-cut fields dissolve visibly ("1 flagged item was removed");
  resubmit re-freezes. *(submit/reject/resubmit flows)* — traces to Product KR-2,
  INV-3.
- **FR-7** — Multi-device convergence (the operator's delegated bar): for all 7
  persisted field types + sub-steps + submit and unsubmit transitions + the
  list-view progress indicators, a change on one device converges on every other
  open device (live) and on reconnect (catch-up), asserted by a red-first E2E
  matrix — 0 matrix cells red at cycle end. *(two-context Playwright matrix)* —
  traces to Engineering KR "convergence matrix", INV-1.

### Engine-trust fixes (not subsumed by the semantic)

- **FR-8** — An approval submitted with feedback only reports success if the
  feedback comment is durably stored. Today the `submission_rejections` insert
  swallows failure (`ON CONFLICT DO NOTHING`, error logged not surfaced —
  `handler.go:614-622`) while the approver sees "Approved". *(POST
  `/approveSubmission`)* — traces to QA KR "red-first" (fix-WO), INV-1.
- **FR-9** — When the server rejects a field write as stale (409 LWW conflict),
  the losing device re-fetches and renders the winning value — the screen never
  keeps showing a value the database rejected. Today it flashes but keeps the stale
  render (`sync.js` conflict path). *(POST `/ops` 409 handling)* — traces to QA KR
  "red-first" (fix-WO), INV-1.

### Non-functional

- **NFR-1** — Every fix requirement above lands red-first: the test fails against
  the unfixed build, recorded in the WO record, then flips green. — traces to QA KR
  "100% red-first".
- **NFR-2** — House build unchanged: static HTML + vanilla JS frontend, Go +
  Postgres backend; no new frameworks or dependencies. **No schema migration is
  expected this cycle**; if one ships anyway, it carries the QA discipline
  (down-migration proven up→down→up + 1 pre-deploy backup artifact). — traces to
  Brief hard constraint, QA KR "0 irreversible schema changes".
- **NFR-3** — Sequencing: the signed design (FR-1) precedes every edit-propagation
  build card; the engine-trust fixes (FR-8/FR-9) are independent and may run in
  parallel at the PjM's discretion. — traces to Delivery objective.

## Trace table (Product KR-1 — 100% of requirements)

| Req | Traces to (reproduced failure \| named invariant) | OKR KR |
|---|---|---|
| FR-1 | INV-3 + INV-6 — design gate | Delivery design-before-build |
| FR-2 | REPRO — field-ID churn, `repository.go:112-183` | Eng stable-identity |
| FR-3 | REPRO — dead-id silent accept, migrations 0051/0053/0054 | Eng loud-rejection · INV-4 |
| FR-4 | REPRO — open device stays stale; `SAVE_TEMPLATE` ignored in `applyOp` | Eng edit-propagation · INV-3 · INV-6 |
| FR-5 | INV-1 — confirmed fire-and-forget at `sync/ops.go:245-264`; violates the operator's always-in-sync bar | Eng edit-propagation |
| FR-6 | INV-3 — the chosen semantic's lifecycle, encoded | Product KR-2 |
| FR-7 | REPRO (multi-device is where the P0 showed) + INV-1 | Eng convergence-matrix |
| FR-8 | INV-1 — confirmed swallow at `handler.go:614-622` | QA red-first |
| FR-9 | INV-1 — confirmed stale render on 409, `sync.js` conflict path | QA red-first |
| NFR-1 | QA discipline | QA red-first |
| NFR-2 | Brief hard constraint | QA backups (conditional) |
| NFR-3 | Operator-signed design gate | Delivery objective |

## Routing (pass-2 finds through the three doors)

No inbox items existed this evening (`.night-crew/inbox/` empty); the 15 backlog
items were routed at the morning OKR session (roadmap routing record, amended twice
at the grill-back — final destinations: stage-1/2 work revived as the permanent
`editprop-*` cards; the versioning schema demoted to backlog). The routing below
covers the 10 pass-2 enumeration finds — three doors, PM judgment per the Brief's
delegation; grill-back outcomes folded in:

| Find | Door | Disposition |
|---|---|---|
| Approval feedback silently dropped (`handler.go:614-622`) | **fold** | FR-8 — manager-entered work, on-theme, small red-first fix (`engine-approval-feedback-loud`) |
| 409-conflict loser keeps stale render | **fold** | FR-9 — the screen lies about persistence; on-theme (`engine-conflict-refetch`) |
| `EmitOp` fire-and-forget (op-log write can fail after a successful save) | **fold** | FR-5 — **promoted from backlog at grill-back**: under the operator's "devices are always in sync" bar, delayed propagation is a loss; transactional op emission |
| Stranded dead-id drafts already in prod | **drop** | was a count-and-report audit; dropped — no active prod users, FR-2/FR-3 stop new stranding, and existing orphan rows are unreachable noise the runner never reads. Reason recorded here |
| Deleted-field drafts never cleaned by remap (`repository.go:197-213`) | **fold** | subsumed: FR-2 preserves identity so only *cut* fields strand; INV-6 makes that an explicit warned discard; FR-3 rejects stragglers loudly |
| Day-boundary schedule change strands an open run (suspected, unverified) | **fold (as design input)** | FR-1 must answer it; its prove-test rides FR-7's matrix |
| Failed photo upload leaves a partial saved value | **backlog** | no durable crew work lost (photo never existed server-side; submit validation blocks); stale-state hygiene, queue |
| Offline submit idempotency under IndexedDB corruption (suspected) | **backlog** | rides the deferred `WO-offline-indexeddb-harness` — untestable without it |
| Lamport clock corruption → catch-up gap (suspected) | **backlog** | same harness dependency; queue with it |
| Unsubmit→fast resubmit fail-note staleness (suspected) | **fold (as matrix cell)** | **upgraded at grill-back**: the operator named submit/unsubmit explicitly in the convergence bar — becomes an FR-7 matrix cell instead of a queued suspect |

Also demoted to backlog at the grill-back: **the versioning schema itself**
(immutable run-pinned template versions) — weighed head-to-head against
frozen-at-submit and not chosen; kept as a future evolution if a fleet-style crew
ever materializes.

Backlog-routed items are appended to `BACKLOG.md` as structured entries this
session (origin: `pm-session 2026-07-16 pass-2 sweep`, status `new` → routed at
next OKR session or promoted by the PjM per T-10/T-12).

## Acceptance criteria

Surface-anchored, Given/When/Then.

- **AC-1** (FR-1) — Given the edit-propagation OpenSpec change is drafted, When the
  operator signs it, Then the sign-off timestamp precedes every build WO dispatch
  in the ledger (audited at cycle gate).
- **AC-2** (FR-2) — Given a template with fields A and B and crew drafts on both,
  When the admin cuts field B and saves in the Builder, Then field A's row in
  `checklist_fields` keeps its ID and A's draft still renders after reload on every
  device (rewritten `tests/repro-cut-task.spec.js` post-edit checks).
- **AC-3** (FR-3) — Given any client holding a stale field ID, When it POSTs a
  draft write naming that ID, Then the server returns 422
  `{"error":"unknown_field"}` and writes no row (red-first Go test), And the runner
  shows the field as not-saved — no green check persists (E2E).
- **AC-4** (FR-4) — Given device A has the checklist open and device B's admin cuts
  a field, When the `SAVE_TEMPLATE` op reaches device A live, Then A re-renders the
  new shape with every surviving answer intact; And Given a device offline during
  the edit, When it reconnects and `wsCatchUp` replays the op, Then the same
  re-render happens with no toast; And Given today's unsubmitted answers exist on
  the cut field, When the admin saves the edit, Then the Builder first shows a
  discard warning naming the count.
- **AC-5** (FR-5) — Given the op insert is forced to fail, When a client saves a
  field, Then the save itself fails loudly (no 200), And 0 rows exist in the
  business tables without a matching op row (red-first Go test on the forced
  failure).
- **AC-6** (FR-6, with the two Product KR-2 tests riding AC-6a/6b) — **(6a)** Given
  a checklist mid-fill on two devices, When the admin edits the template, Then both
  devices re-render the new shape with surviving answers intact and the crew
  completes the current shape; **(6b)** Given a submitted checklist, When the admin
  later edits the template, Then the submitted record's rendered review is
  byte-identical to what was submitted; And Given a rejection on a field that was
  since cut, When the crew reopens, Then the flag dissolves visibly and resubmit
  still routes to the manager.
- **AC-7** (FR-7) — Given two devices with the same checklist open, When each of
  the 7 field types, a sub-step, a submit, and an unsubmit is driven on one device,
  Then the other device's field state AND list-view progress indicator converge for
  every cell of the matrix, live and after a reconnect (red-first E2E matrix; 0 red
  cells).
- **AC-8** (FR-8) — Given an approver approving with a feedback comment, When the
  `submission_rejections` insert fails, Then the API response is an error and the
  UI does not show "Approved" (red-first Go test forcing the conflict).
- **AC-9** (FR-9) — Given two devices writing the same field, When one write loses
  LWW and gets 409, Then within one op cycle the losing device renders the winning
  value (red-first E2E asserting the rendered value equals the stored one).

## Verification plan

- **Environment:** the ephemeral Docker pg16 stack (`docker-compose.nc.yml`) — the
  canonical local DB path (ledger 2026-07-14). No prod data work is expected (no
  migration); prod deploy is the operator's `task prod:deploy` + `task version`
  parity check.
- **Baseline red:** `tests/repro-cut-task.spec.js` is rewritten to assert the
  frozen-at-submit semantic; its red run against the unfixed build is the recorded
  baseline (Delivery KR). It flips green when the editprop cards land and is
  committed to the suite.
- **Red-first protocol (NFR-1):** every fix card records the failing run before
  the fix — bug-fix protocol per CLAUDE.md; only the new tests run during
  iteration, full suite at card close.
- **Convergence matrix (FR-7):** two Playwright contexts against one DB; matrix
  rows = {checkbox, yes/no, text, temperature, sub-steps, fail-note text, fail
  severity, photo-URL value} × {live, catch-up} plus submit/unsubmit transitions
  and the list-view progress bar; every cell asserts the second device's observed
  state, not the first device's optimism.
- **Cycle-end gates:** `task test` exit 0 on the deterministic stack (retires
  carried waiver #1); `task version` shows prod == local constants (Delivery
  prod-parity KR).

## Out of scope

- **Run-pinned immutable template versions** — weighed and not chosen (2026-07-16
  grill-back); demoted to backlog as a possible future evolution. No `template_
  versions` schema, no run entity, no migration this cycle.
- Recovering or auditing historical stranded rows in prod — no active users;
  FR-2/FR-3 stop new stranding; existing orphans are unreachable noise.
- The deferred harness/fixture work: photo-S3 harness, offline-IndexedDB harness,
  onboarding video fixture (cycle routing, 2026-07-16).
- Net-new crew-facing features; the other four apps except the carried roadmap
  cards (`ops-nfr3-resubmit-photo-gate`, `users-s3-orphan-cleanup`,
  `prod-ghost-item-rename` ride their own cards, not this PRD).
- Changing INV-5 (day-boundary draft expiry) — named and kept as-is.

## Success metrics

- Rewritten `tests/repro-cut-task.spec.js` green and in the suite with its recorded
  red baseline; 0 regressions of it for the rest of the cycle.
- 100% of writes naming a field outside the current template return 422, 0 return
  200; the `replaceTemplate` delete-reinsert path is deleted from the codebase.
- FR-7 matrix: 0 red cells at cycle end — the operator's "always in sync" bar is a
  number, not a feeling.
- 0 accepted business writes without a committed op row (FR-5), measured by the
  forced-failure test plus a DB invariant query in the WO record.
- Cycle gate: `task test` exit 0; prod parity per `task version`; 100% of this
  PRD's fix-WOs carry red-run evidence.
