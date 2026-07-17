# Roadmap — "Nothing silently lost" cycle (checklist data integrity)

> **Cycle:** Nothing silently lost — every way the checklist engine can silently lose
> crew-entered work is enumerated, fixed, and made structurally impossible.
> **Traces to:** `.night-crew/knowledge/okrs.md` (Product / Delivery / Engineering / QA,
> signed 2026-07-16). **Produced:** 2026-07-16 attended `/nc-okr-session` (the roadmap's
> guaranteed producer, DESIGN §15u). Previous cycle's roadmap archived at
> `reference/roadmap-2026-07-09-hq-hardening.md`.
> **Trigger:** operator-reported, same-day-reproduced P0 — template edits silently discard
> open-device crew work (repro: `tests/repro-cut-task.spec.js`, untracked until stage 1
> flips it green; root cause: `replaceTemplate` field-ID churn + FK-dropped silent dead-id
> writes; backlog entry has full cites).

## How this roadmap works

- **Activity-level cards.** Each card is WO-sized-ish work the PjM/`nc-slate-plan` sizes
  to a night. Cards carry a **module footprint** (for parallel tracks) and a **KR trace**.
- **Status:** `DONE` · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED`.
- **Cadence is the PjM's, not the operator's.** Cards-per-night is the planner's call
  against the night budget + quality bar (budget is a floor, not a ceiling).
- **Sequencing rule (RE-AMENDED at the evening PM grill-back, operator 2026-07-16):**
  the semantic was revisited head-to-head; the operator **delegated the choice to the PM**
  with a hard UX bar (multi-device sync always convergent). PM chose **frozen-at-submit**:
  an unsubmitted checklist always shows the current template on every device; submit
  freezes the record forever; rejection reopens it live. The versioning schema is deleted —
  no migration this cycle. The build is stable field identity + loud rejection + edit
  broadcast (old stages 1–2 revived as the permanent architecture) + transactional op
  emission + a device-convergence matrix. The only gate stays the signed design
  (Activity 4) before any build card.
- **Red-first is mandatory on every fix card** (QA KR2): the test fails before the fix,
  recorded in the WO record.

## Module footprints (independent → parallelizable)

| Track | Frontend | Backend | Tests |
|---|---|---|---|
| Workflow engine | `workflows.html`, `sync.js` | `backend/internal/workflow` | `workflows.spec.js`, `sync.spec.js`, `persistence.spec.js`, `repro-cut-task.spec.js` |
| Users | `users.html` | `backend/internal/users` | `users.spec.js` |
| Inventory (prod ops) | — | `backend/internal/inventory` | `inventory.spec.js` |
| Test-debt | — | — | all spec files (audit-scoped) |

---

## Activity 1 — PRD gate · *blocking, first*

> The stages 1–3 data-integrity PRD. No build WO dispatches before this lands.
> Produced by the evening `/nc-pm-session` + `/nc-pm-grill-back`.

- **`prd-data-integrity`** · **DONE** ✅ signed 2026-07-16 (evening PM session + grill-back; frozen-at-submit, 9 FR + 3 NFR, `prd validate` green) · The PRD enumerating every silent-loss mode with a
  requirement→(reproduced failure | named invariant) trace table; the operator-signed
  mid-run edit semantic ("crews finish the run they started; edits take effect next run");
  the 8-item backlog routing record. → Product KR1, KR2, KR3. *(attended evening session)*

## Activity 2 — Engine-trust fixes + carried small fixes · *after Activity 1; parallel tracks*

> ~~`stage1-field-id-preservation`~~ · ~~`stage1-dead-id-reject`~~ — **REVIVED under new
> names (2026-07-16 grill-back, frozen-at-submit decision):** the stage-1 work returned as
> the *permanent* architecture — see Activity 5's `editprop-stable-field-identity`
> (upsert + loud rejection). These tombstones stay so the morning's card names resolve.

- **`engine-approval-feedback-loud`** · DONE ✅ overnight-20260717 (G6 PASS `f1cf912`; red→green re-verified — failed submission_rejections persist now returns 500 feedback_persist_failed instead of false "Approved"; ON CONFLICT DO NOTHING removed, proven behavior-neutral. Follow-up logged: approval+feedback atomicity via tx through approveSubmission) · An approval with a feedback comment only
  reports success if the comment is durably stored — today the `submission_rejections`
  insert swallows failure (`handler.go:614-622`, `ON CONFLICT DO NOTHING`, error logged
  not surfaced) while the approver sees "Approved". Red-first Go test forcing the failed
  insert. Footprint: workflow engine (backend). → QA KR2, PRD FR-6 (INV-1).
- **`engine-conflict-refetch`** · PLANNED · A device whose field write loses LWW (409)
  re-fetches and renders the winning value instead of keeping the stale render
  (`sync.js` conflict path). Red-first E2E. Footprint: workflow engine (`sync.js`).
  → QA KR2, PRD FR-7 (INV-1).
- **`ops-nfr3-resubmit-photo-gate`** · DONE ✅ overnight-20260717 (G6 PASS `01c8f7e`; red→green re-verified — direct-API resubmit of a require_photo-rejected field: 201 bypass → 400 resubmit_photo_required; server-side gate resolved from submission_rejections on the authenticated submitter's most-recent prior submission, both direct-API submit paths; no client-controllable escape) · Carried fix-card: plumb rejection context
  into submit validation so a rejected-with-`require_photo` field blocks direct-API
  resubmit server-side; red-first. Footprint: workflow engine (backend). → QA KR2
  (red-first denominator), carried from hardening cycle (ledger T-10).
- **`users-s3-orphan-cleanup`** · PLANNED · Trivial carried card: remove dead
  `<div id="s3">` at `users.html:122`. Footprint: Users (zero contention — free
  parallelism). → hygiene; no KR.

## Activity 3 — ~~Stage 2: template-updated broadcast~~ · *REVIVED under a new name*

> ~~`stage2-template-updated-broadcast`~~ — **REVIVED (2026-07-16 grill-back,
> frozen-at-submit decision):** under the chosen semantic, live re-render on edit is the
> *permanent* behavior, not interim relief — see Activity 5's
> `editprop-broadcast-rerender`. This tombstone stays so the morning's card name resolves.

## Activity 4 — Edit-propagation design gate · *attended; blocks Activity 5*

- **`editprop-openspec-design`** · **DONE** ✅ signed 2026-07-16 (attended design gate; frozen-at-submit design, **C5 = warned-live-removal**; `designs/editprop-frozen-at-submit.md`) · The OpenSpec change for frozen-at-submit
  edit propagation: stable field IDs honored forever; edits re-render open devices with
  surviving answers intact; submit freezes the record (existing `template_snapshot`);
  rejection rules (frozen record · live redo carrying answers · moot flags on cut fields
  dissolve visibly); cut-field discard rule + Builder warning when today's unsubmitted
  answers exist; the convergence contract (what "in sync" means per surface); day-boundary
  schedule-change behavior (the C5 question); race handling in the edit→broadcast window.
  **Operator sign-off on the design is the gate — 0 build WOs dispatch before it**
  (auditable from ledger timestamps). → Delivery KR "edit-propagation design signed
  before build".

## Activity 5 — Edit-propagation build · *serialized after Activity 4 sign-off; no schema migration*

- **`editprop-stable-field-identity`** · DONE ✅ overnight-20260717 (G6 PASS `6a483d1`; red→green re-verified independently — Go 422 + cross-device E2E identity; `replaceTemplate` reinsert path deleted; app-level existence check, no restored FK) · `updateTemplate` upserts by the field
  IDs the Builder already sends (update kept / insert new / delete removed; conditions
  remap for new fields only) instead of delete+reinsert; a write naming a field absent
  from the current template → distinct 422 envelope, surfaced in the runner (no optimistic
  checkmark survives a rejected save). Revives stages 1a+1b as permanent. Footprint:
  workflow engine (backend + runner error path). → Eng KRs "stable identity" + "loud
  rejection".
- **`editprop-broadcast-rerender`** · DONE ✅ overnight-20260717 (G6 PASS `0d49f27`+`1c7c73c`; all 5 sub-behaviors red→green re-verified — SAVE_TEMPLATE re-render surviving-answers, silent-on-catch-up, C5 warned live removal, transactional op emission in-txn, INV-6 discard warning naming the crew count + orphaned-draft delete scoped to unsubmitted) · Handle `SAVE_TEMPLATE` ops in `applyOp`
  (they already flow through live WS + `wsCatchUp` — clients just ignore them): re-fetch
  template, re-render the open checklist with surviving answers intact, dissolve moot
  rejection flags visibly, stay silent on catch-up replay (the `42eeb39` no-toast rule);
  Builder warns before a save that discards today's unsubmitted answers on cut fields.
  Plus transactional op emission: the op row commits in the same transaction as the write
  it describes (closes the `EmitOp` fire-and-forget gap, `sync/ops.go:245-264`). Footprint:
  workflow engine (`sync.js` + `workflows.html` + backend sync). → Eng KR "edit
  propagation".
- **`editprop-convergence-matrix`** · DONE ✅ overnight-20260717 (G6 FAIL-REVISE → revised → PASS `72fffba`+`6c3aafb`; full two-device matrix green — all 7 types + sub-steps + photo + submit/unsubmit + list-view progress + denominator, live + catch-up; AC-6a bug-guard + AC-6b frozen-snapshot lock red→green; surfaced+fixed unsubmit-broadcast gap in-footprint; suite reliably green under combined load, orchestrator re-verified 36/36) · The red-first multi-device E2E matrix
  (the operator's delegated UX bar): all 7 field types + sub-steps + submit/unsubmit
  transitions + list-view progress indicators converge across ≥ 2 devices; includes the
  ≥ 2 semantic acceptance tests (mid-run edit re-renders open devices, surviving answers
  intact; a submitted checklist is unaffected by later edits) via the rewritten
  `repro-cut-task.spec.js`. → Eng KR "convergence matrix", Product KR "edit semantic",
  Delivery KR "repro red→green pair".

## Activity 6 — Test-debt retirement · *independent parallel track (any time)*

- **`vacuous-tests-18-to-0`** · PLANNED · Each remaining conditional `test.skip()` / silent
  guard-return becomes a real seeded assertion or is deleted (denominator = the audit that
  produced the 18). Footprint: test-debt (audit-scoped). → QA KR1, retires carried
  waiver #2.
- **`carried-fix-wos-sweep`** · PLANNED · The carried prove-sweep fix-WO with no harness
  dependency: `WO-cron-clock-seam` (a `now` seam in the 4 `run*Check` funcs + real
  cron-decision unit tests — unblocks P-6, Purchasing FR-19/20/21/22). The other two
  prove-sweep PARKs (photo-S3 harness, offline-IndexedDB harness) are deferred this cycle
  — harness-infrastructure class, see routing record. Red-first. → QA KR2.

## Activity 7 — Prod ops · *operator-gated*

- **`prod-ghost-item-rename`** · PLANNED · Operator-chosen handling (2026-07-16): rename
  `''` → `(Unnamed — needs review)`, KEEP line-item links. Verify: empty-description count
  = 0 in prod AND previously-linked `purchase_line_items` count unchanged. **Prod data
  mutation — runs attended or with explicit operator go.** Footprint: Inventory (prod DB).
  → QA KR3.
- **`prod-deploy-parity`** · PLANNED · Operator runs `task prod:deploy` (never automated);
  card verifies `task version` shows prod == local `version.go` constants (includes
  `42eeb39`). → Delivery KR "prod parity".

## Activity 8 — Cycle gate · *last, serialized*

- **`cycle-gate`** · PLANNED · Suite-green attestation on the deterministic stack
  (`task test` exit 0 — formally retires carried waiver #1); median WO cycle time vs the
  recorded baseline (first cycle with a pass/fail target); per-KR scorecard; closeout doc.
  → Eng KR "task test exits 0", Delivery KR "median WO cycle".

---

## Backlog routing record (Product KR3 — 15/15 `new` items routed 2026-07-16)

> **Amendment (2026-07-16 evening grill-back, final):** the semantic decision landed on
> **frozen-at-submit** (operator-delegated, PM-chosen), so the destinations moved twice
> tonight and settle as: stage-1 work → `editprop-stable-field-identity`; stage-2 work →
> `editprop-broadcast-rerender` (both now the *permanent* architecture); the stage-3
> versioning schema is **demoted to BACKLOG** (weighed head-to-head and not chosen; kept
> as a future evolution if a fleet-style crew ever materializes). The rows record what the
> morning session decided; this note records what changed and why.

| Backlog item | Door | Destination |
|---|---|---|
| Ops P0 template-edit data loss (stage 1) | promoted | ~~`stage1-*`~~ → `editprop-stable-field-identity` |
| Ops stage 2 broadcast | promoted | ~~`stage2-template-updated-broadcast`~~ → `editprop-broadcast-rerender` |
| Ops stage 3 immutable versions | promoted | ~~Activities 4–5 (`versioning-*`)~~ → demoted to BACKLOG (frozen-at-submit chosen instead) |
| Inventory prod ghost item | promoted | `prod-ghost-item-rename` |
| Ops NFR-3 resubmit photo gate | promoted | `ops-nfr3-resubmit-photo-gate` |
| Users `#s3` orphan cleanup | promoted | `users-s3-orphan-cleanup` |
| Onboarding video-pipeline fixture | deferred | off-theme for this cycle; needs operator Spaces creds; revisit next cycle (operator 2026-07-16) |
| `/nc-status` non-determinism | deferred | framework/tooling outside this repo; stays in backlog (operator 2026-07-16) |
| Ops FR-15 builder-UI coverage gap | promoted | rides `vacuous-tests-18-to-0` |
| Ops FR-10/FR-12 vacuous reject test | promoted | rides `vacuous-tests-18-to-0` |
| Onboarding 6 conditional-skip guards | promoted | rides `vacuous-tests-18-to-0` |
| Inventory ~40 data-dependent guards | promoted | rides `vacuous-tests-18-to-0` |
| WO-cron-clock-seam | promoted | `carried-fix-wos-sweep` (pure code seam, no harness dependency) |
| WO-photo-s3-harness | deferred | needs mock-S3/test-bucket harness + creds — same class as the video fixture; revisit next cycle |
| WO-offline-indexeddb-harness | deferred | needs a new SW+IndexedDB Playwright project; off-theme this cycle |
