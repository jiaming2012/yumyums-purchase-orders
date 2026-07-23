# Backlog — advisory items that ride future cards

> Durable parking lot for triage-surfaced items that aren't roadmap cards yet but must
> survive run-to-run (HANDOFF is rewritten each run). Format: `title · description · origin ·
> status`. Promote to a roadmap card with `promoted → <card>`; drop with `dropped — reason`
> (struck through, kept as record).

> **🏁 Milestone boundary — HQ hardening cycle closed 2026-07-16 (cycle-gate signed off, ledger
> T-14).** The `new` items below are now the **feedstock for the next cycle's `/nc-okr-session`**.
> Nothing here was resolved by the read-only closeout run (it touched no code). The cycle's carried
> residue in one place: the ~37–41 flaky/data-dependent + SW-blocked suite pool (declined "Stabilize
> the suite," 2026-07-15) and the ~18 vacuous-test remainder are captured by the test-hardening notes
> below; the 3 harness WOs, F-1, F-2, and the video-pipeline fixture are itemized as their own entries.
> Per T-10/T-12, promotion/scheduling of any item is a planner call, not an operator hand-pick.

- **Onboarding video-pipeline E2E fixture (prove FR-16 + NFR-4)** · Stand up a DO Spaces bucket +
  an `ffmpeg` binary on the E2E host so the video presign→PUT→FFmpeg transcode/thumbnail path
  (FR-16, `handler.go:540-640`, `video.go:22-206`) and the `503 video_storage_not_configured`
  fallback (NFR-4) can be exercised. Both are fully implemented and **waived** from the Eng-KR
  denominator at triage 2026-07-13 (D-5) as env-gated — this fixture is the preserved prove-path:
  when built, it flips FR-16/NFR-4 UNPROVEN→WORKING (or reddens them). Needs operator-supplied
  Spaces creds. ~4–5h test-infra build. · origin: triage 2026-07-13 (D-5, waive-now-but-preserve)
  · deferred 2026-07-16 — off-theme for the "nothing silently lost" cycle; needs operator creds; revisit next cycle (OKR-session routing)

- ~~**Users stale-E2E repair** · `tests/users.spec.js` two Access-tab tests navigating dead
  `#t3`/`#s3` + rename the misleading `s3` var in `renderAccess`.~~ **DONE — promoted →
  `users-stale-e2e-repair`, landed overnight-20260714 (`d32830d`, G6-PASS):** `#t3/#s3` → `#t2/#s2`
  repoint + `s3`→`accessEl` rename; `users.spec.js` 17/2 → 19/0. · origin: triage 2026-07-10 (D-4)
  · closed 2026-07-14

- **Ops NFR-3 — backend resubmit `require_photo` gate** · The field-level required-photo gate ships
  front+back (overnight-20260714, `ad105f7`), but the **rejection-driven resubmit** photo requirement
  is **frontend-only** — `SubmitChecklistInput` (`backend/internal/workflow/model.go`) carries no
  `submission_id`/rejection context, so `validateFailNotes` can't know a prior rejection set
  `require_photo` on a field. A direct-API resubmit can bypass it. Fix: plumb rejection context into
  the submit validation (a `submission_rejections` join keyed on the resubmitted submission) + a
  red-first test that a rejected-with-require_photo field blocks resubmit server-side. Small fix-card.
  **Scheduling delegated to the planning agents** (PjM `/nc-slate-plan` / PM `/nc-pm-session` / eng) —
  triage 2026-07-14 (ledger T-10) declined to hand-pick backlog-vs-slate placement; queue placement is
  a planner call, not an operator triage pick. Stays a ready candidate here until they promote it.
  · origin: overnight-20260714 ops-nfr3 (in-footprint deferral, G6-confirmed) · promoted →
  `ops-nfr3-resubmit-photo-gate` (roadmap Activity 2, OKR session 2026-07-16)

- **Users `users.html:122` orphaned `<div id="s3">` cleanup** · After `users-stale-e2e-repair`
  repointed the Access tests to `#s2` and renamed the `renderAccess` var, the dead `<div id="s3">`
  at `users.html:122` (3-tab→2-tab refactor leftover) is now fully orphaned — nothing references it.
  Trivial removal; fold into any future Users card. · origin: overnight-20260714 users-stale G6
  (optional cleanup) · promoted → `users-s3-orphan-cleanup` (roadmap Activity 2, OKR session 2026-07-16)

- **`/nc-status` non-determinism — same repo, two machines, two reports** · Running `/nc-status`
  on separate machines yields diverging output, which defeats the skill's whole purpose (a single
  authoritative "you are here"). Root cause: the skill's §1 gather-commands hardcode paths that no
  longer match this repo's layout, so each machine's agent silently improvises a *different*
  substitute. Confirmed drift: (a) `grep -oE '"bg":"#..."' usm/roadmap.txt` — **no `usm/` dir
  exists**; the roadmap is `.night-crew/knowledge/roadmap.md` (markdown, not TextUSM), so the
  card-color signal returns empty and each agent re-derives progress differently. (b) `ls
  reference/slate-*.md` — actual path is `.night-crew/knowledge/reference/slate-*.md`. (c) root
  `BACKLOG.md` / `DECISIONS-NEEDED.md` don't exist — backlog is `.night-crew/knowledge/BACKLOG.md`
  and DECISIONS-NEEDED files nest per-run under `.night-crew/runs/<date>/`, so one machine reported
  "no DECISIONS-NEEDED in tree" while the resolved fork actually lives at
  `.night-crew/runs/2026-07-15-autonomous/DECISIONS-NEEDED.md`. (d) the DONE count read 28 on one
  machine vs 27 on the other because `grep -oE '\bDONE\b' roadmap.md` counts the **status-legend
  line** (`- **Status:** \`DONE\` · …`) as a card. Fix: repoint the §1 gather block to the real
  `.night-crew/` layout; make the color/progress signal read `roadmap.md`'s status tokens by
  **card row, not word token** (exclude the legend line); and resolve DECISIONS-NEEDED by globbing
  `.night-crew/runs/*/DECISIONS-NEEDED.md` and honoring the inline `> RESOLVED` marker (so a
  triaged fork reads "present-and-resolved," never "absent"). Symmetry check: apply the same
  layout truth to `/nc-help` and any `nc-progress` variant so all status surfaces agree. The skill
  file lives outside this repo (`~/.claude/skills/nc-status/`), so this is a **framework/tooling
  WO**, not a product card — but it must land or every future `/nc-status` cross-check is unreliable.
  · origin: 2026-07-15 cross-machine `/nc-status` diff (operator-observed) · deferred 2026-07-16 —
  framework/tooling outside this repo; stays in backlog until a night-crew dogfood pass picks it up
  (operator decision, OKR-session routing)

- **Ops P0 — template-edit data loss: field IDs churn on every `updateTemplate` (REPRODUCED)** ·
  Editing a template (e.g. cutting a task from the Friday checklist) while crew devices have the
  checklist open silently discards all their subsequent work. Mechanism, confirmed by deterministic
  E2E repro: `replaceTemplate` (`backend/internal/workflow/repository.go:99-220`) deletes and
  re-inserts every field with **new IDs** on any edit (drafts are remapped old→new at edit time,
  but only once); open devices keep rendering the old IDs, and every later check/uncheck writes a
  draft under a dead field id — accepted **silently** because the field_id FK was dropped
  (migrations 0051/0053/0054). Optimistic UI shows the check; reload loses it; a device that opened
  after the edit never sees the other's ops. Repro spec: `tests/repro-cut-task.spec.js` (untracked
  on dev — asserts the DESIRED behavior, so it fails until fixed and then becomes the regression
  test; baseline pre-edit sync passes, post-edit reload-persistence fails on both devices). Fix
  stage 1 of 3 (operator-ratified direction 2026-07-16): (a) preserve field IDs on update — honor
  the ids the Builder already sends (`toApiTemplate` includes them) via diff/upsert instead of
  delete-reinsert; (b) make dead-id saves loud — `saveResponse` rejects unknown field ids via an
  app-level existence check scoped to drafts (`submission_id IS NULL`; do NOT restore the FK —
  submitted responses reference snapshot ids by design). Red-first: flip the repro spec in.
  · origin: operator report 2026-07-16 (Friday checklist, two devices), reproduced + root-caused
  same session · promoted → ~~`stage1-*`~~ → **final (2026-07-16 grill-back, frozen-at-submit):**
  `editprop-stable-field-identity` (roadmap Activity 5) — the stage-1 work as the *permanent*
  architecture, not an interim

- **Ops — template-updated broadcast: open devices re-render on edit** · Stage 2 of the
  template-edit robustness roadmap. `SAVE_TEMPLATE` ops are already emitted on template edits and
  already flow through both live WS and `wsCatchUp` replay — clients simply ignore them. Handle
  them in `applyOp` (sync.js): re-fetch the template, re-render the open checklist with the new
  shape/ids, preserve any in-progress input. Closes the staleness gap stage 1 leaves (devices
  still rendering a cut field until reload; mixed old/new-device live sync) and covers offline
  devices on reconnect for free via catch-up. Depends on stage 1 (stable ids make the re-render
  a remap instead of a reset). Mind the silent-replay rule from `42eeb39`: a template re-render
  triggered by catch-up must not toast. · origin: fix-direction session 2026-07-16 (stage 2 of 3)
  · promoted → ~~`stage2-template-updated-broadcast`~~ → **final (2026-07-16 grill-back,
  frozen-at-submit):** `editprop-broadcast-rerender` (roadmap Activity 5) — live re-render on
  edit is the chosen *permanent* semantic, not interim relief (briefly demoted mid-grill when
  versioning was the plan; revived when frozen-at-submit won the head-to-head)

- **Ops architecture — immutable template versions, run-pinned (stable field identity)** · Stage 3
  end-state that deletes the stage-1/2 compensations: fields get client-generated UUIDs honored by
  the server forever (identity as fact, not row artifact); every edit creates an immutable template
  version with "the template" a head pointer; a checklist RUN pins the version current when the run
  started (crews finish the run they started — mid-shift edits take effect next run, resolving
  "what does cutting a task mean for a half-done checklist" by rule); responses key on
  (run, field-uuid) so the existing op-log/Lamport sync layer can never target a dead id. Extends
  the existing submit-time `template_snapshot` (LC-02) upstream to edit-time. Phase-sized (schema
  migration + workflow backend rework + runner load path); slots naturally ahead of sync Phase 11 /
  the reactive-store direction. · origin: fix-direction session 2026-07-16 (stage 3 of 3,
  operator asked for the robust-architecture answer) · ~~promoted → `versioning-*`~~ **demoted
  back to backlog at the 2026-07-16 evening grill-back:** weighed head-to-head against
  frozen-at-submit (edits live on unsubmitted checklists; submit freezes) — the operator delegated
  the semantic to the PM with a multi-device-sync bar, and frozen-at-submit won: the operator is
  the editor and wants corrections live; a 1–5 person single-kitchen crew doesn't need
  fleet-auditor run-pinning; and the chosen shape needs no schema migration. Kept as the future
  evolution if a fleet-style crew ever materializes (ledger G-2). · deferred 2026-07-16

- **Inventory — prod ghost catalog item check + cleanup** · The receipt pipeline auto-created a
  `''`-description `purchase_items` row when Claude parsed a receipt with unnamed line items
  (description is UNIQUE → every future unnamed line merges into the one ghost item; it renders as
  a blank first row in the review picker). All creation paths are guarded as of `42eeb39`
  (validate.go Check 0 routes unnamed receipts to review; worker.go never auto-creates from an
  empty name), but the guard shipped AFTER real receipts flowed through prod — check prod:
  `SELECT count(*) FROM purchase_items WHERE trim(description)='';` plus its linked
  `purchase_line_items`, then rename or unlink+delete. **Prod data mutation — needs operator
  sign-off on the handling, not just the WO.** Small card. · origin: 2026-07-16 test-failure
  root-cause (empty item found in `hq_test_ui` from live Mercury ingest; prod exposure inferred)
  · promoted → `prod-ghost-item-rename` (roadmap Activity 7, OKR session 2026-07-16 — operator
  chose rename-keep-links handling)

## Activity-4 fix-cards (from Activity-2 confirm-absence graduations — code-fix + regression-test WOs)

> Distinct from test-only prove-UNPROVEN WOs: these are **confirmed-BROKEN** flows where a cited
> line proves the behavior absent. Each = code fix (front+back) + red-first regression test.

- ~~**Operations FR-4 — yes/no "No" corrective-action enforcement** · A "No" answer never blocked
  submit (`evaluateFailTrigger` handled only `out_of_range`; submit short-circuited on
  `!f.fail_trigger`; server `validateFailNotes` checked only `out_of_range`).~~ **DONE — promoted →
  `ops-fr4-no-enforcement`, landed overnight-20260714 (`2287947`, G6-PASS, red-first):** a failing
  "No" now requires a corrective fail note front+back (`isYesNoNo` + trigger in `validateFailNotes`,
  mirroring the temperature path). · origin: overnight-20260712 ops-confirm-absence · closed 2026-07-14
- ~~**Operations NFR-3 — photo-required-at-submit enforcement** · No photo gate on submit/resubmit;
  `PhotoURL` storable (`model.go:92`) but never required.~~ **DONE (field-level) — promoted →
  `ops-nfr3-photo-required`, landed overnight-20260714 (`ad105f7`, G6-PASS, red-first):** submit now
  blocks until a required photo field carries a valid `https://` value, front+back (`isHTTPSPhotoValue`
  iterates template fields, not just responses). **Residual → F-1 below:** the rejection-driven
  *resubmit* photo requirement stays frontend-only (no `submission_id`/rejection context in the submit
  input). · origin: overnight-20260712 ops-confirm-absence · partially closed 2026-07-14
- ~~**Onboarding NFR-5 — reopen/reject of a video-led section is a silent no-op** · Video progress
  keyed by `ob_video_parts.id`, but `ReopenSection` deleted by parent `ob_items.id` → zero rows,
  section stayed complete.~~ **DONE — promoted → `onboarding-nfr5-video-reopen`, landed
  overnight-20260714 (`5d73b96`, G6-PASS, red-first):** `ReopenSection` resolves the first video part
  for `video_series` items and deletes THAT progress, reverting the section to active. Covers FR-9
  (crew reopen) + FR-15 (manager reject). · origin: overnight-20260712 onboarding-confirm-absence ·
  closed 2026-07-14
- ~~**Purchasing FR-18 — History tab is a static stub (frontend build WO)** · `#s4` hardcoded stub;
  no `renderHistory`, no `GET /shopping/history` call; backend `GetShoppingListHistory` never wired
  to the UI.~~ **DONE — promoted → `purchasing-fr18-history`, landed overnight-20260714 (`4cb57b7`,
  G6-PASS, red-first):** `renderHistory` built + wired to `GET /shopping/history`; 5 History tests
  pass (fixture via existing API, no SQL/migration). · origin: PRD-purchasing-hardening Activity 1 ·
  closed 2026-07-14

## Test-hardening notes (from Activity-3 test-audits — ride the downstream prove-UNPROVEN/test WOs; not reclassifications)

- **Operations FR-15 builder-UI coverage gap** · The six builder field types are proven only via
  `POST /createTemplate` API calls, not the builder Add-Field UI; only 4/6 types covered (photo = 0,
  sub-steps thin). Flow stays WORKING (real assertions exist; not statically vacuous) but is
  under-proven vs. its claim ("add each type *via the builder*"). Add builder-UI Add-Field coverage
  for all six types incl. photo. · origin: overnight-20260712 ops-test-audit (G6-passed) · promoted →
  rides `vacuous-tests-18-to-0` (roadmap Activity 6, OKR session 2026-07-16)
- **Operations FR-10/FR-12 vacuous reject test** · `reject item with comment`
  (`tests/workflows.spec.js:485-508`) wraps its whole body in `if (await flagBtn.isVisible())` with
  no `expect` — genuinely vacuous. Maps to FR-10/FR-12 (already UNPROVEN). Use as the starting point
  for the FR-12 test-only WO (the assertion to add). · origin: overnight-20260712 ops-test-audit ·
  promoted → rides `vacuous-tests-18-to-0` (roadmap Activity 6, OKR session 2026-07-16)
- **Onboarding 6 conditional-skip guard sites** · `tests/onboarding.spec.js` has 6 guard/`return`
  sites that skip an assertion when the seed shape differs: `:991` (FR-13, PRD-flagged), `:148-151`
  (FR-3), `:250-259`+`:826-841` (FR-5), `:304-306` (FR-2), `:700-745` (FR-13 UI), `:2104-2107`+
  `:2136-2139` (FR-15). All flows STAY WORKING (each has an unconditional sibling assertion) — these
  are hardening flags, not drops. Replace each guard with a self-seeded fixture so a shape mismatch
  reddens instead of silently skipping. Rides the Onboarding Activity-4 test-hardening WO.
  · origin: overnight-20260712 onboarding-test-audit (G6-passed) · promoted → rides
  `vacuous-tests-18-to-0` (roadmap Activity 6, OKR session 2026-07-16)
- ~~**Purchasing FR-7 vacuous shopping-tab test (dropped WORKING→UNPROVEN)** · `Shopping tab shows
  stub when no active list exists` ended in `expect(text.trim().length).toBeGreaterThan(0)` — a
  generic-content tautology.~~ **DONE — promoted → `purchasing-fr7-retest`, landed overnight-20260714
  (`958a176`, G6-PASS):** real assertions replace the tautology — (a) exact empty-state stub copy with
  no list, (b) grouped vendor sections + per-item check buttons + thumbnails + aisle locations when
  populated. (Note: the old test was baseline-RED — it targeted a nonexistent `#shopping-content` —
  not merely vacuous.) FR-7 UNPROVEN → WORKING. · origin: overnight-20260712 purchasing-test-audit ·
  closed 2026-07-14
- ~~**Inventory NFR-1 double name-normalization gap** · Two surfaces persist un-normalized text while
  the rest title-case: (1) `UpdateItemHandler` writes `input.Description` raw — item *edit* skips
  `normalizeItemName`; (2) `ConfirmPendingPurchaseHandler` upserts the **vendor** raw while receipt
  line-items ARE normalized.~~ **DONE — promoted → `inventory-nfr1-normalize-fix`, landed
  overnight-20260715 (`748463c`, G6-PASS, red-first):** the E4 prove card committed the NFR-1 RED
  first, then the fix added `normalizeItemName` to BOTH surfaces (`handler.go:660` vendor + `:1130`
  description) — RED→GREEN on pristine-vs-fixed binaries. Both named gaps closed in one 2-line fix;
  FR-4's "vendor upserted title-cased" PRD text is now accurate (no correction needed). Eng KR-1 +1 → 0.
  · origin: overnight-20260712 inventory-confirm-absence · closed 2026-07-15
- **Inventory ~40 data-dependent test guards (cleanup)** · `tests/inventory.spec.js` carries ~40
  `if (await X.count() > 0) return;` / `if(count>0){…}` guards that silently pass when a seed is a
  no-op (PRD §Verification). None drops a WORKING flow (each guarded assertion is backstopped by a
  guaranteed real seed or an unguarded sibling), but they're QA-KR-1 cleanup: convert to unguarded
  (or self-seed) so a seed miss reddens. Representative: FR-2 tax/grand-total at `:1039-1042,
  1058-1061`. Rides the Inventory Activity-4 test-hardening WO. · origin: overnight-20260712
  inventory-test-audit (G6-passed) · promoted → rides `vacuous-tests-18-to-0` (roadmap Activity 6,
  OKR session 2026-07-16)

## Pass-2 enumeration finds (from the 2026-07-16 evening PM session — `PRD-data-integrity` §Routing)

> The data-integrity PRD's two-pass surface sweep found 10 loss-mode candidates; 5 folded into
> the PRD (FR-9/FR-10/FR-11 + 2 subsumed), these 5 routed here. Full dispositions in
> `prds/PRD-data-integrity.md` §Routing.

- ~~**Sync op-log durability — `EmitOp` is fire-and-forget**~~ · After a successful business
  write, the op row that tells other devices about it is inserted in a goroutine with errors
  logged only (`backend/internal/sync/ops.go:245-264`). **PROMOTED same evening at the grill-back:**
  under the operator's delegated UX bar ("devices are always in sync"), delayed propagation IS a
  loss — folded into `PRD-data-integrity` FR-5 (transactional op emission), rides
  `editprop-broadcast-rerender`. · origin: pm-session 2026-07-16 pass-2 sweep · promoted →
  `editprop-broadcast-rerender` (grill-back 2026-07-16)

- **Runner — failed photo upload leaves a partial saved value** · The photo-upload `.catch` blocks
  (`workflows.html:1564-1572`, `:1635-1647`) render a retry UI but don't clear the partial value
  from `FIELD_RESPONSES`/`DRAFT_RESPONSES`; a stale value can linger under the field. No durable
  crew work is lost (the photo never existed server-side; submit validation blocks required-photo),
  so this is stale-state hygiene, not a loss mode. Small frontend fix + persistence test.
  · origin: pm-session 2026-07-16 pass-2 sweep · deferred 2026-07-19 (PM-session routing — off-theme / harness-dependent; reason per item in PRD-prove-and-surface §Routing items 10 & 12)

- **Offline submit idempotency under IndexedDB failure (suspected, unverified)** · If the offline
  submit queue's `idempotency_key` is lost to an IndexedDB write failure, reconnect drain
  (`sync.js:466-494`) could duplicate a submission. Untestable without the offline harness — rides
  `WO-offline-indexeddb-harness` (deferred this cycle) when it's built. · origin: pm-session
  2026-07-16 pass-2 sweep · deferred 2026-07-19 (PM-session routing — needs the offline-IndexedDB harness, not built this cycle; PRD-prove-and-surface §Routing item 11)

- **Lamport clock corruption → catch-up gap (suspected, unverified)** · If the stored Lamport
  clock in IndexedDB is cleared/corrupted, the next `wsCatchUp` (`sync.js:303-315`) may skip or
  refetch ops, staling the UI. Same harness dependency — rides `WO-offline-indexeddb-harness`.
  · origin: pm-session 2026-07-16 pass-2 sweep · deferred 2026-07-19 (PM-session routing — off-theme / harness-dependent; reason per item in PRD-prove-and-surface §Routing items 10 & 12)

- ~~**Unsubmit → fast resubmit fail-note staleness (suspected, unverified)**~~ · If a user
  unsubmits (`workflows.html:2372-2380`) and resubmits before the re-render, stale `FAIL_NOTES`
  state may duplicate or drop fail notes. **UPGRADED same evening at the grill-back:** the operator
  named submit/unsubmit explicitly in the convergence bar — becomes an FR-7 matrix cell
  (`editprop-convergence-matrix`) instead of a queued suspect. · origin: pm-session 2026-07-16
  pass-2 sweep · promoted → `editprop-convergence-matrix` (grill-back 2026-07-16)

## Prove-sweep PARK fix-WOs (from overnight-20260715 — production refactor / new harness, beyond same-footprint test-only)

> Surfaced by the prove-UNPROVEN sweep as the *fixes* the parked flows imply. Each needs a
> production seam or a new test harness, so per the slate they were NOT graduated the same night.
> Scheduling is the planners' call (queue placement per T-10). Unblocks the §A parks in
> `runs/2026-07-15-autonomous/DECISIONS-NEEDED.md`.

- **WO-cron-clock-seam** · Add a `now time.Time` (or package `nowFn`) seam to the 4 `run*Check`
  funcs — `runReminderCheck`/`runCutoffCheck`/`runLowStockCheck` (`internal/purchasing/scheduler.go:54/167/247`)
  + `runRepurchaseResetCheck` (`repurchase.go:129`), each currently reads `now := time.Now().In(loc)`
  inline — then add real cron-decision unit tests (seed config + past-cutoff → assert transition, no
  15-min wait). Adjacent pure logic already proven GREEN in `scheduler_prove_test.go`. **Unblocks P-6
  (Purchasing FR-19/20/21/22).** · origin: overnight-20260715 purchasing-prove-state-auth-scheduler
  (PARK) · promoted → `carried-fix-wos-sweep` (roadmap Activity 6, OKR session 2026-07-16 — pure
  code seam, no creds/harness dependency)
- **WO-photo-s3-harness** · A way to exercise photo presign→PUT→public-URL in E2E (mock S3 or a test
  DO-Spaces bucket); the ephemeral stack sets no `SPACES_*` so `UploadHandler` returns 503. **Unblocks
  P-1 (Onboarding FR-18), P-2 (Inventory FR-27), P-3 (Operations NFR-2 PUT leg)** — 3 flows in one
  harness. Related to (but distinct from) the Onboarding video-pipeline fixture above (video vs item/
  receipt photo). · origin: overnight-20260715 (A4/E3/D2 PARKs) · deferred 2026-07-16 — needs new
  harness infrastructure (mock S3 / test bucket + creds), same class as the deferred video fixture;
  revisit next cycle (OKR-session routing)
- **WO-offline-indexeddb-harness** · A dedicated Playwright project with the service worker +
  IndexedDB enabled (`playwright.config.js:29` blocks SWs today) to test offline sync/queue/conflict
  + draft-persist-across-redirect. **Unblocks P-4 (Operations NFR-5), P-5 (Operations NFR-7 draft
  leg).** · origin: overnight-20260715 ops-prove-cross (PARK) · deferred 2026-07-16 — needs a new
  SW+IndexedDB Playwright project (structural harness build), off-theme for this cycle; revisit
  next cycle (OKR-session routing)

## Editprop follow-ups (from overnight-20260717 morning triage — same-pattern tidy-ups, planner-scheduled)

> Surfaced as out-of-footprint follow-ups on the editprop build; operator routed to BACKLOG at
> triage 2026-07-17 (no competing option — queue placement is a planner call per T-10/T-12). The
> substantive convergence-cell item (F-A) was NOT backlogged — it went straight to roadmap card
> `editprop-convergence-cell-hardening` (Activity 6). The `workbox-build` gap (F-D) was fixed at
> triage (`3b1be67`), not backlogged.

- **Transactional op emission for Create/Archive (INV-1 parity)** · W-2 moved `updateTemplate`'s
  op emission into the write txn (`EmitOpTx`), but `CreateTemplateHandler` and
  `ArchiveTemplateHandler` still use the fire-and-forget `EmitOp` goroutine. Convert both to
  `EmitOpTx` (mirrors W-2's pattern; no schema change) for full INV-1 "0 accepted writes whose op
  is not durably queued" parity. Small follow-up card. · origin: triage 2026-07-17 (F-B, W-2
  follow-up) · deferred 2026-07-19 (PM-session routing — editprop tidy-up, stays BACKLOG not a KR this cycle; PRD-prove-and-surface §Routing item 6)
- **Fail-note conflict live-render on the `applyOp`/409 path (`_fail_note` unpack)** · The W-6b
  conflict-coverage sweep (`editprop-convergence-cell-hardening`) hardened the LWW-409/`applyOp`
  render for 4 answer types (yes_no, temperature, sub-step, checkbox) but could NOT reach the 2
  fail-note types: `applyOp`'s `SET_FIELD` branch (`sync.js:405`) unpacks only `value`/`sub_steps`,
  never `_fail_note` — that bundle is unpacked solely by `hydrateFieldState` (`workflows.html:1480`)
  on load/reopen. So on a two-device conflict where a fail-note (value+note+severity, or photo URL)
  loses LWW, the losing device shows a stale/malformed fail card until the next reopen (data never
  lost server-side; a live-render staleness window on a rare concurrent path). Production card:
  extend `applyOp` `SET_FIELD` to unpack `_fail_note` on the incoming-op/409 path, then land the 2
  parked W-6b cells — own design/footprint/G6, NOT test-debt. **Bundle candidate with F-B above**
  (both touch the op-emission/apply path). · origin: triage 2026-07-18 (D-1, overnight-20260718 —
  operator chose accept + track over graduate-now) · deferred 2026-07-19 (PM-session routing — out-of-footprint editprop tidy-up, needs `_fail_note` unpack on apply path; PRD-prove-and-surface §Routing item 7)
- **Atomic approval + feedback (`approveSubmission` tx)** · `approveSubmission` commits
  `status='approved'` *before* the feedback loop, so a feedback-persist failure correctly returns
  500 `feedback_persist_failed` (W-4's goal, requirement MET) but leaves the submission already
  `approved` — a partial commit. Thread a `tx` through `approveSubmission` (repository.go) so
  status + feedback commit atomically; also removes the less-specific `internal_error` a retrying
  approver sees on the 2nd attempt (idempotent via the `status='pending'` guard). · origin: triage
  2026-07-17 (F-C, W-4 follow-up) · deferred 2026-07-19 (PM-session routing — editprop tidy-up, not a KR this cycle; PRD-prove-and-surface §Routing item 8)
- **Onboarding persistence tests: `waitForResponse` over fixed flush wait** · Two converted
  persistence tests in `tests/onboarding.spec.js` use `waitForTimeout(1500)` instead of
  `waitForResponse('/saveProgress')`. The post-reload assertion is still the load-bearing proof, so
  the guard isn't weakened — but a fixed wait is a small flake-surface. Switch to `waitForResponse`
  on the save POST in a future test-hardening pass. Low priority. · origin: triage 2026-07-17 (F-E,
  T-2 minor) · deferred 2026-07-19 (PM-session routing — low-priority test-hardening; PRD-prove-and-surface §Routing item 9)

## Waiver-#1 last mile (from overnight-20260719 cycle gate — operator chose "graduate" 2026-07-19)

- **`suite-isolation-approved-checklist` — formally retire waiver #1** · The 2026-07-19 cycle gate
  ran the deterministic stack green on an isolated pg16 (Go units exit-0; Playwright 450 pass · 1
  fail · 0 flaky · 6 skip) EXCEPT one red: `tests/workflows.spec.js › approved checklist shows
  Approved badge and cannot be resubmitted [LST-08 RUN-08]` — `expect(locator('#toast')).toBeVisible()`
  gets `hidden` **in the full suite but PASSES in isolation** (fresh single-test DB, `--retries=0` →
  `1 passed`; cycle-closeout-20260719.md §1). It is a **cross-test DB-pollution / test-isolation
  defect, not a product defect**: some earlier spec sharing `hq_test` leaves approval/`#toast` state
  that this test's assertion trips over. This single red is the ONLY thing keeping literal
  `task test` from exit-0, so waiver #1 is currently **substantially retired (38 reds → 1) but not
  formally**. **Fix:** isolate this test's state dependency (identify the polluting predecessor —
  likely an approval/submission-rejection row or a `#toast` left visible — and either scope its
  fixture teardown or make the assertion order-independent), then re-run the full suite to confirm
  **literal `task test` exit-0**, which **formally retires carried waiver #1** (Eng KR5 PASS). Small
  test-hardening WO (no production change; test-only footprint). Red-first: the full-suite red is the
  baseline; green = literal exit-0. · origin: overnight-20260719 cycle gate (Eng KR5 PARTIAL) ·
  **operator chose (a) graduate 2026-07-19** (DECISIONS-NEEDED §C) · promoted → `waiver1-isolation-fix` (Activity 3; PM-session routing 2026-07-19, PRD-prove-and-surface §Routing / FR-10)
- **Per-card wall-clock instrumentation as a standing build-run output** · The 2026-07-19 gate could
  not compute a this-cycle Delivery median (KR4 PARTIAL) because the 07-17 run's 9 build cards were
  not per-card timed — only 07-18's single card was measured. `-0718` already re-adopted the
  harness-measured table; make it the **invariant** for every build run so the ledger stays measured,
  not narrated, and the next gate can compute a real median vs the T-14 baseline (N=23/22m28s).
  · origin: overnight-20260719 cycle gate (Delivery KR4 PARTIAL, fix-forward) · promoted → `percard-timing-instrumentation` (Activity 3; PM-session routing 2026-07-19, PRD-prove-and-surface §Routing / FR-11)
- **Gate run-mechanics: `CI=1` + explicit pre-migration by default** · Two run-to-run wall-clock
  losses (07-18 G6, 07-19 gate) came from the same `:8199` `reuseExistingServer` foreign-server
  latch and (07-19) an unmigrated isolated DB. Bake into the gate/G6 run-mechanics: always run the
  suite with `CI=1` (forces `reuseExistingServer:false` → own webServer + teardown) AND pre-migrate
  the isolated pg16 via a throwaway app boot before Go units. Run-mechanics doc/skill fix, not a code
  change. · origin: overnight-20260719 cycle gate (card-actuals 8th-slate obs) · folded → rides `percard-timing-instrumentation` / cycle-gate run-mechanics (PM-session routing 2026-07-19, PRD-prove-and-surface §Routing / FR-11)

## Escaped defect + QA gap (found 2026-07-17 on dev, operator play)

- **Cross-user live-sync access matrix + `sync`-package unit coverage** · An escaped defect
  (live-sync ops fanned out only to a checklist's assignees, excluding non-assignee editors incl.
  admins/superadmins → the operator's own edits never reached their 2nd device; fixed red-first,
  `sync/ops.go` `ResolveEntityAccess` unions admins + `listener.go` always includes the author,
  regression test `sync/access_test.go`) revealed a QA hole: **every convergence/live-sync test
  drives the assignee editing their own checklist**, so the recipient-resolution for a *non-assignee
  editor* was never tested, AND the whole `sync` package had **zero Go tests**. WO: add a cross-user
  access matrix ({who views}×{who edits}×{who observes}×{role/assignment}, asserting access AND
  live-op propagation) + `sync`-package unit coverage for `ResolveEntityAccess` across all
  role/assignment combos. Full write-up: `reference/qa-gap-20260717-live-sync-access.md`.
  · origin: 2026-07-17 operator-found live-sync bug (dev play, post cycle-gate) · promoted → `sync-pkg-unit-coverage` + `convergence-matrix-systematic` (Activity 3; PM-session routing 2026-07-19, PRD-prove-and-surface §Routing / FR-7·FR-8·FR-9)

- **Live approval-state convergence coverage** · Two MORE escaped defects of the SAME class found
  2026-07-18 in continued operator play: an approve/reject op reached the receiving device but the
  client re-rendered from a **stale `MY_SUBMISSIONS` cache** instead of reconciling the changed
  submission status. Symptoms: (1) a manager's rejection reason never reached the submitter's other
  device live (`applyOp` `REJECT_ITEM` only refreshed the Approvals tab — a no-op for a non-approver),
  and (2) an observer's list count stayed frozen on the pre-rejection submission **snapshot**
  (`getProgress` counts `submission.responses` while status is pending/submitted/approved). Fixed
  red-first (broad refresh-on-op: `applyOp` routes `APPROVE_ITEM`/`REJECT_ITEM` through
  `loadMyChecklists` so every receiving device reconciles submission status → correction banner,
  edit-vs-readonly mode, and list count all converge live). Regression tests: `tests/sync.spec.js`
  `RJT-LIVE-01/02/03`. **QA hole this widens:** the convergence matrix tested only `SET_FIELD`
  ops — never the *submission-lifecycle* ops (submit/approve/reject) cross-device, and never asserted
  that a status change reconciles the observer's list count / an open runner. Fold into the WO above:
  the access matrix must also vary the **op type** (field edit vs submit/approve/reject) and assert
  **live convergence of derived views** (banner, readonly, progress count), not just the field value.
  Full write-up: `reference/qa-gap-20260717-live-sync-access.md` (§ 2026-07-18 addendum).
  · origin: 2026-07-18 operator-found approval-sync bugs (dev play) · promoted → `convergence-matrix-systematic` (Activity 3; PM-session routing 2026-07-19, PRD-prove-and-surface §Routing / FR-8·FR-9 ESC-2)

- **Rejection feedback on SUB-STEPS (fixed) + the require-photo dead-end (open).** Operator play
  (dev, 2026-07-18) found rejecting a **sub-step** (e.g. "Cut the check → Do B") stored the comment
  + require_photo but rendered them NOWHERE — the runner drew the correction banner only at the
  PARENT field level (`REJECTION_FLAGS[parentId]`), sub-step rows had none, and `hydrateFieldState`'s
  field-id filter (`tplFieldIds`) never even included sub-step ids. Fixed red-first: shared
  `correctionBannerFor(id, resp)` helper renders on parent AND sub-step rows, and `tplFieldIds` now
  includes sub-step ids (both open paths). Test: `tests/workflows.spec.js` `APR-SUBSTEP-0718`.
  **RESOLVED 2026-07-18 — the require-photo DEAD-END + correction-photo slot.** A non-photo field
  rejected with `require_photo=true` used to show "📷 Photo required before resubmit" with NO capture
  control, while both gates demanded the field's *value* be an `https://` URL — impossible for a
  checkbox (its value is boolean). Built a dedicated **correction-photo slot** (`CORRECTION_PHOTOS`),
  persisted by bundling `_correction_photo` into the saved value (separate from the answer, mirroring
  `_fail_note`): capture UI on any non-photo require-photo field, unpacked on hydrate (draft +
  submission) and in the manager review view, and honored by the frontend gate + a new backend
  `hasResubmitPhoto`. Also fixed a latent **hard-block**: a sub-step `require_photo` matched the
  backend resubmit gate but was unsatisfiable (sub-steps aren't sent as top-level responses) → the
  gate now excludes `parent_field_id` fields (sub-step require-photo is advisory). Tests: backend
  `TestResubmit_..._SucceedsWithCorrectionPhoto` + `..._SubStepRequirePhoto_NotBlocked`; E2E
  `APR-DEADEND-0718`; persistence `FLD-CORRECTION-PHOTO`. Parked-by-convention: the presign+PUT camera
  plumbing itself (injected in tests, like onboarding FR-18).
  · origin: 2026-07-18 operator-found sub-step rejection bug (dev play) · resolved 2026-07-18

> **Triage 2026-07-20 (overnight-20260721, ledger T-18) graduations.** §B items were routed at
> triage (B2+B3 → `replay-fetchstorm-gate` card; B4 ratified; B5 folded into
> `inventory-tab-gating`) and do NOT appear below. These are the run's §C durable observations
> that ride future cards:

- **`checklist_submissions.status` never set for `requires_approval:false` submissions** ·
  Defaults `'pending'` and `submitChecklist` never updates it, so no-approval submissions read
  `'pending'` server-side forever. Harmless today (UI derives from other fields) but a trap for
  any future server-side status consumer — normalize on submit or document the invariant. ·
  origin: overnight-20260721 A1 impl (§C) · new
- **Rejected-field hydrate quirk: new answer visually clears on reload until resubmission** ·
  Answering a rejected field then reloading blanks the new answer visually
  (`workflows.html:1544` hydrate branch prefers the rejection snapshot); device-local, LOW.
  Rides any future runner-hydration card. · origin: overnight-20260721 A1 G6 (§C) · new
- **`sync.spec.js` de-flake: `:1198` + `:525`** ·
  `:1198 temperature answer converges` is **flaky — confirmed by controlled reproduction, at
  ~16% overall (4 red / 25 `--retries=0` legs), ~20% (4/20) under a concurrent Playwright
  suite** (investigation 2026-07-21, `investigate/1198-under-load-20260721`). The flake is
  REAL; the numbers previously carried here were not. **Two corrections to the prior claim:**
  (a) "red 1-of-2 legs at load 0.84" **misattributed the load** — its own source table
  (`runs/2026-07-22-autonomous/DECISIONS-NEEDED.md` §S1 tail) shows load 0.84 on the leg that
  **PASSED**; the red leg started at load **3.96** on a rising box. (b) The implied ~50% rate is
  refuted — two parties' ~20 consecutive greens are consistent with ~16-20% *conditional on
  contention* (p(0 red in 20 | 20%) ≈ 1.2%, and they sampled the quiet condition).
  **Failure mode is NOT what the card assumes.** All 4 reds share one signature:
  `page.waitForResponse` timeout at **`sync.spec.js:1119`** — the `POST /ops` **commit** wait —
  not `CONVERGE_TIMEOUT` and not a convergence assert. The autosave debounce is **400ms**
  (`workflows.html:278`), so a 12s timeout means the op **never fired**, not that it was slow;
  ops-journal deltas confirm it (green legs +5 rows, red legs +3 — no `SET_FIELD` row).
  **Therefore raising any timeout is the wrong fix.** The likely mechanism is the same stray WS
  catch-up `loadMyChecklists` re-render the de-flake comment names: it detaches the temperature
  input between `fill()` and `dispatchEvent('change')`, so the change handler never arms
  `debouncedSaveField` and no POST is ever issued. The de-flake **relocated** that race (from
  "typed value clobbered to empty" to "save never arms") rather than closing it — gating on the
  `POST /ops` 2xx cannot be race-free against a re-render that prevents the POST existing.
  Fix direction: make answer-entry re-render-safe (settle the runner before entering, and/or
  re-assert value + re-dispatch if no POST is observed). Test-side; no production change needed.
  **Scope must also include `:525 FLD-LIVE-02`**, which G6
  found fails 3/3 in isolation *and at the pre-gate baseline* — a pre-existing
  order-dependent test. The flake surface is broader than "just `:1198`". ·
  origin: overnight-20260722 S1 PARK (b) + quiet streak ·
  **evidence re-derived 2026-07-21** — see `reference/1198-flake-reproduction-20260721.md` ·
  **promoted → `syncspec-deflake` (D1, slate-20260720c)** — cleared the §15k architecture-blocking
  bar: the `cycle-gate` card promises no-retry suite-green attestation, which cannot pass while
  `:1198` is proven-flaky and `:525` reds at baseline.
- **Replay fetch-storm class is NOT fully closed** ·
  S1 gated `SUBMIT_CHECKLIST`, but G6's enumeration of every branch in `applyOp` found
  `loadPendingApprovals()` and `loadTemplates()` still fire an **ungated per-op re-fetch** —
  a catch-up with N APPROVE ops still storms the approvals queue, N SAVE_TEMPLATE ops still
  storm the Builder list. Same root cause, same one-line fix pattern, deliberate-by-omission
  (the in-code comments say "always refreshes"). · origin: overnight-20260722 S1 G6 · new
- **`sync.js api()` fetch-abort guard (S1 sub-move (d), deliberately skipped)** ·
  Suite-teardown noise (`loadMyChecklists error: Failed to fetch`) originates from a `catch`
  in **`workflows.html:389`**, outside S1's footprint. A sync.js-only fix would require
  `api()` to never reject during unload, altering the error path of every workflow API call
  including the offline-queue fallback — poor risk/benefit unattended. Needs re-scoping as
  its own card **with `workflows.html` in footprint**. · origin: overnight-20260722 S1 (d) · new
- **`.gitignore` lets a `node_modules` symlink into the index** ·
  The line is `node_modules/` (trailing slash) — matches a directory but **not a symlink**.
  Worktrees have no `node_modules`, so symlinking the main install is the natural move and it
  slips straight past into `git add -A`. One implementer already hit it and reverted its own
  instance. One-char fix: drop the slash. · origin: overnight-20260722 S1 flag · new
- **`:8199` port latch recurred (third run)** ·
  A `go run` child survives `kill` of its parent and holds the port. Standing recipe should
  kill the **listener PID** (`ss -ltnp`), not the `go run` parent, and concurrent tracks
  should be assigned distinct `TEST_PORT`s up front. · origin: overnight-20260722 S1 flag · new
- **`-p 1` is load-bearing for Go suites — write it into standing run mechanics** ·
  Verified at base commit (no card code present): default parallel `-p` reddens four packages
  (`inventory` 6, `purchasing` 4, `receipt` 9, `recipes` 5+) via concurrent `TRUNCATE`s on a
  shared DB. `-p 1` is green. This makes every card's build/vet signal unreliable until
  discovered per-card. · origin: overnight-20260722 F1 G6 · new
- **Money is `float64` end-to-end in the inventory/recipes path** ·
  Pre-existing repo convention (`period-summary` too), not a card regression. JSON can emit
  `23.099999999999998`, and it compounds per-bucket rounding drift. Cents-as-int or a decimal
  string is the repo-wide correct fix. · origin: overnight-20260722 F1 G6 · new
- **`git stash` prohibition needs a mechanical guard** ·
  A subagent ran `git stash` in a worktree (forbidden, 07-15 hazard), self-disclosed, and
  recovered cleanly; the operator's own stash entry was verified intact. But the rule lives
  only in prose. Consider a guard refusing `stash` when `git rev-parse --git-common-dir`
  differs from `--git-dir`. · origin: overnight-20260722 F4 incident · new
- **Dangling standing-rules pointer in every slate since 07-15** ·
  Slates inherit gates G1–G6 "unchanged by reference from
  `reference/overnight-run-plan-20260707.md`" — **that file does not exist.** Real origin is
  `runs/2026-07-09-attended/slate-20260710.md` §"Run mechanics" plus the app-code adaptation
  in `slate-20260714.md`. A dangling inherit in the one document defining the run's gates is a
  latent single point of failure. · origin: overnight-20260722 orchestrator · new
- **PRODUCT THREAD: the tabs show single numbers where the operator wants comparisons** ·
  Two triage answers converged on the same shape. (1) Food cost should be a **long-term
  average with a direction of travel**, not a fixed-12-week snapshot — this dissolves the
  0%-food-cost bug rather than patching it. (2) **Margin with and without discounting** —
  blocked today: `daily_menu_sales` stores only `units_sold` + `gross_amount`, no discount or
  comp field, so it needs Toast sync to capture them first. Not cleanup; belongs in a PM
  session. · origin: overnight-20260722 triage T-19 · new

---

## Graduated / added at morning triage T-20 (2026-07-21, overnight-20260720c)

- **Prod, dev and test share ONE Postgres cluster, one role, one password** ·
  `yumyums-dev-pg` holds database `yumyums` with schema `public` (dev, 50 tables) AND schema
  `production` (prod, 48 tables), plus databases `hq_test_go` / `hq_test_e2e`. Prod is separated
  from dev by nothing but a client-supplied `search_path=production` in a connection string — not
  an enforced boundary, no privilege separation, same `yumyums` role for both. Omit the parameter
  and you land in `public`. This is the sharpest instance of audit surface #4 and was found while
  answering the operator's "should never be writing to the live db!". Correct fix is a genuinely
  separate cluster (or at minimum separate roles with `REVOKE`), not more `search_path` discipline.
  · origin: T-20 triage investigation · **new, HIGH**
- **`dotenv: ['backend/.env']` injects 21 LIVE credentials into every task from the main checkout** ·
  Root `Taskfile.yml:3`. A dev server (PID 75921) ran three days holding live Mercury *production*,
  Anthropic, Zoho Cliq and SMTP keys with `E2E_DISABLE_SCHEDULERS` unset — receipt worker, cutoff,
  drift and alert queue all armed against real external services. Triage blanked the Cliq/SMTP vars
  for the *Playwright* server (decision 40), which does NOT address `task dev` or any other target.
  Proper fix: invert the default so alerts require an explicit `ALERTS_ENABLED=1` and forgetting a
  flag fails safe. · origin: T-20 / audit surface #5 · **new, HIGH**
- **`task backend:db-start` is vestigial and now conflicts** · It creates a second HQ Postgres
  container (`yumyums-pg`, which does not exist today) and, after decision 39 templated its host
  mapping to `{{.DB_PORT}}`, would collide with `yumyums-dev-pg` on 5433. Either delete the target
  or point it at a genuinely separate port. Not redesigned at triage. · origin: T-20 · new
- **Stale server processes squat ports across sessions** · PID 83061 holds `:18484` from a worktree
  (`hq-worktrees/sync-units`) that has been **deleted**, pointed at a dead Postgres port. Same shape
  as the `:8199` latch (audit #2, fixed by decision 41) with a different number. A general reaper /
  port-ownership convention is the durable fix. · origin: T-20 · new
- **`stash@{0}` holds unattributed WIP in a slot shared by five worktrees** ·
  `WIP on dev: acd2c7f refactor: migrate all server logging from log to slog NDJSON output`.
  `refs/stash` is shared across all worktrees, which is why `git stash` is already prohibited here.
  Confirm ownership before anything touches it. Still present after triage. · origin: audit #6 · new
- **Two config files carry independent copies of the same default** · `backend/Taskfile.yml` and
  `playwright.config.js` each define their own `DB_PORT` / `dbPort` default. Changing one silently
  leaves the other — which happened at T-20 and cost a full E2E leg. Both now cross-reference each
  other in comments, but the duplication itself is the defect. Audit for other duplicated defaults.
  · origin: T-20 reviewer error · new
- ~~**`sync.spec.js` de-flake: `:1198` + `:525`**~~ · **promoted → `syncspec-deflake` (re-aimed at
  T-20)**. Scope widened to four tests: `:1198`, `:525`, LST-17, GATE-04. Fix is test-side; no
  production change; no timeout increase helps. **Shipped overnight-20260724 (S1). Line anchors
  are now dead** — G1/S1 moved the tests; locate by title (`-g "temperature answer converges"`,
  FLD-LIVE-02 by name). T-21.
- ~~**Cost-tab gate: confidentiality or tidiness?**~~ · **promoted → `grant-enforcement-parity`**
  (T-20 decision 36). Answer was confidentiality, and the gap is 9 of 11 grants, not 2 routes.
- **`/photos/*` key-binding gate (G1-a)** · The photo presign/upload endpoints are the sole
  authenticated-only routes — any logged-in user with a photo key can read any stored photo URL
  regardless of grants. A union-of-app-grants gate was rejected as cosmetic (T-21 decision 42);
  the durable fix binds photo keys to their owning app/record so per-app grants actually gate
  reads. Needs a small design pass (key namespace vs owner-record join). The documented
  exception in `tests/grant-enforcement-parity.spec.js` stands until this ships. · origin:
  overnight-20260724 G1 park, T-21 decision 42 · new
- **`sync.js` catch-up fetch-storm gate** · `applyOp`'s SAVE_TEMPLATE branch re-fetches
  `myChecklists` per replayed op whenever a runner is open (`sync.js` ~491) — a device catching
  up on a large journal fires an un-awaited fetch per op (the storm behind every FLD-LIVE-02
  red). A one-line gate/debounce fixes the app-level behavior, but it is a production `sync.js`
  change and RE-ARMS the attended two-device convergence check — schedule it with that cost
  priced in. · origin: overnight-20260724 S1 observation (reported, not fixed) · new
- **`onboarding.spec.js` second-run carried-DB failures** · Two tests fail when the full suite
  runs twice against the same un-reset DB (carried hire/training state). Fine under the
  clean-DB-per-leg rule, but the suite is non-idempotent — either reset state in `beforeAll` or
  document clean-DB as a hard precondition. · origin: overnight-20260724 S1 (pre-existing, out
  of scope) · new
