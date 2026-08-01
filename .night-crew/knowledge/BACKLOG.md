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
  `'pending'` server-side forever. ~~Harmless today (UI derives from other fields)~~ — **that
  premise was DISPROVEN by run 20260725: the stuck `'pending'` was load-bearing.** It hit the
  `isPending` branch and rendered wrong-but-present copy, so removing it without a client change
  left the checklist editable and re-submittable. · origin: overnight-20260721 A1 impl (§C) ·
  **promoted → `workflow-submission-status-default`** (server half, DONE 2026-07-25) **+
  `workflow-submission-status-client-half`** (split out at morning triage 2026-07-25, ledger
  T-22 decision 49)
- ~~**Rejected-field hydrate quirk: new answer visually clears on reload until resubmission**~~ ·
  Answering a rejected field then reloading blanks the new answer visually
  (`workflows.html:1544` hydrate branch prefers the rejection snapshot); device-local, LOW.
  **dropped — superseded by the RxDB/Supabase migration** (roadmap Activity 1): a symptom of the
  manual-hydrate mechanism being replaced, not an independent fix. · origin: overnight-20260721
  A1 G6 (§C) · dropped, `/nc-roadmap-round` 2026-07-25
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
- ~~**Replay fetch-storm class is NOT fully closed**~~ ·
  S1 gated `SUBMIT_CHECKLIST`, but G6's enumeration of every branch in `applyOp` found
  `loadPendingApprovals()` and `loadTemplates()` still fire an **ungated per-op re-fetch** —
  a catch-up with N APPROVE ops still storms the approvals queue, N SAVE_TEMPLATE ops still
  storm the Builder list. Same root cause, same one-line fix pattern, deliberate-by-omission
  (the in-code comments say "always refreshes"). **dropped — superseded by the RxDB/Supabase
  migration** (roadmap Activity 1): this is the same fetch-storm class the migration retires.
  · origin: overnight-20260722 S1 G6 · dropped, `/nc-roadmap-round` 2026-07-25
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
- ~~**`sync.js` catch-up fetch-storm gate**~~ · `applyOp`'s SAVE_TEMPLATE branch re-fetches
  `myChecklists` per replayed op whenever a runner is open (`sync.js` ~491) — a device catching
  up on a large journal fires an un-awaited fetch per op (the storm behind every FLD-LIVE-02
  red). A one-line gate/debounce fixes the app-level behavior, but it is a production `sync.js`
  change and RE-ARMS the attended two-device convergence check — schedule it with that cost
  priced in. **dropped — superseded by the RxDB/Supabase migration** (roadmap Activity 1):
  `sync.js` is retired by `sync-hard-cutover`, not patched. · origin: overnight-20260724 S1
  observation (reported, not fixed) · dropped, `/nc-roadmap-round` 2026-07-25
- **`onboarding.spec.js` second-run carried-DB failures** · Two tests fail when the full suite
  runs twice against the same un-reset DB (carried hire/training state). Fine under the
  clean-DB-per-leg rule, but the suite is non-idempotent — either reset state in `beforeAll` or
  document clean-DB as a hard precondition. · origin: overnight-20260724 S1 (pre-existing, out
  of scope) · new
- **Gated-tab grants: granular-overrides-umbrella semantics** · Reverse the T-18 umbrella
  rider for slug-bearing tabs, per the operator's play-test ruling (T-21a decision 45,
  verbatim): "If there is a granular permission for a tab and it does not exist, the tab
  should not be visible. If no granular permission exists, then the tab should be visible by
  default." Concretely: `inventory-trends`/`inventory-cost` require their explicit grant —
  the `inventory` app grant no longer implies them; un-slugged tabs stay app-grant-covered.
  Red-first regression test (Jim B scenario: app grant, no tab grant → tab absent + endpoint
  403) BEFORE the fix; then drop the umbrella arg from the two RequirePermission mounts, drop
  the `'inventory'` disjunct in `hasTabGrant` (inventory.html), flip the umbrella-direction
  tests (incl. G6's grant-parity pairs). Ships as a patch release. NOT urgent — no crew
  accounts hold prod grants today. · origin: operator play-test 2026-07-24, T-21a decision
  45 · new
- **Cross-user checklist hydration divergence (approved-vs-rejected ghost state)** · Two users
  viewing the SAME checklist render different states as a function of their own last
  submission: a rejected submission resurrects as the viewer's current 2/2 state while the
  other user (whose copy was approved) sees fresh 0/2 with clicks silently no-oping (no POST,
  no toggle, no feedback). Reproduced headlessly in fresh contexts — deterministic hydration
  logic (`MY_SUBMISSIONS`-driven), NOT network/cache/gating/the 07-22 sync.js change; server
  state verified byte-identical for both users. The E2E convergence matrix misses the cell:
  it never seeds an asymmetric approved-for-A/rejected-for-B history before reopening.
  Needs a product ruling first (what SHOULD each user see on a new cycle after a split
  approve/reject?), then a red-first cell + fix. Full evidence + repro:
  `reference/sync-crossuser-hydration-20260724.md`. · origin: operator play-test 2026-07-24,
  reproduced at triage · new
- **Cost tab 0%-food-cost anomaly investigation** · F2's open note (T-19 decision 33 residue,
  disposition T-21f decision 48): a menu item can show 0% food cost. Lead hypothesis: sales
  without any recipe allocation → ingredient_cost_total = 0 → a technically-correct but
  misleading 0% — wants an explicit "unallocated" marker distinct from genuine 0%, not a
  formula fix (the margin math is fixture-proven). Investigate once prod sales sync lands and
  Cost carries real rows. · origin: F2 open note, routed T-21f · new
- **`workflows.html` sync: migrate to RxDB + self-hosted Supabase, unify autosave + live
  broadcast into one store** · The hand-rolled WebSocket + Postgres LISTEN/NOTIFY +
  Lamport-clock op log (`sync.js` + `backend/internal/sync/`) is the repeat offender behind
  this cycle's own fragility findings above (`sync.js` catch-up fetch-storm gate,
  cross-user checklist hydration divergence, the retired-but-recurring `sync.spec.js`
  flakiness) — a "fetch storm" bug class (T-18) where reconnecting clients replay their full
  missed-op history and naive per-op refetches (`loadPendingApprovals`, `loadTemplates`)
  fire redundant requests and clobber optimistic UI mid-edit, plus a standing bug where
  manager rejections never reload into client state on refresh. Decision (operator explore
  session 2026-07-24): replace BOTH write paths in `workflows.html` — the field-level
  `autoSaveField`→`POST /saveResponse` draft path AND the live ops/broadcast layer — with a
  single RxDB client store replicated against a **self-hosted** Supabase stack (Realtime +
  PostgREST) running alongside the existing Postgres on the Windows box. Last-write-wins
  conflict resolution (checklist edit conflicts are rare; no custom conflict handler needed).
  Auth: bridge the existing bearer-token/session auth rather than adopt Supabase Auth/GoTrue —
  the Go backend mints its own HS256 JWTs (`role: authenticated`, `sub`, `exp`) signed with
  Supabase's configured `JWT_SECRET`, which self-hosted PostgREST/Realtime accept for RLS
  without GoTrue. Cutover: hard swap, no parallel run — `sync.js`, `backend/internal/sync/`,
  and `/saveResponse` retire entirely once live. Feasibility confirmed by research
  (2026-07-24): RxDB's Supabase replication plugin is Cloud-agnostic (talks only to
  `@supabase/supabase-js`, i.e. PostgREST + Realtime, both in the self-hosted Docker stack);
  self-hosted-specific gotcha is each synced table must be manually added to the
  `supabase_realtime` publication (no dashboard toggle) and needs a text PK + `_deleted` +
  `_modified` trigger + RLS enabled. Scope is `workflows.html` only — inventory, purchasing,
  onboarding, and users stay on the existing Go+Postgres REST API. Promotion should start
  with a feasibility spike (stand up self-hosted Supabase in Docker alongside the existing
  Postgres, prove the Go-minted-JWT → RLS bridge end-to-end) before sizing the full
  migration. · origin: operator explore session 2026-07-24 · **promoted → `sync-rxdb-feasibility-spike`,
  `sync-rxdb-schema-and-replication`, `sync-jwt-bridge-endpoint`, `sync-hard-cutover`** (roadmap
  Activity 1 "Sync foundation", `/nc-roadmap-round` 2026-07-25)

<!-- Handled entries start here. The 63 entries above predate the B-NN requirement and are why
     `night-crew backlog check --file` currently rejects this document (220 issues). New entries
     carry handles from 2026-07-25 forward; migrating the historical ones is B-02. -->

- **B-01 · Repair the W1 Supabase runbook's fabricated presentation** — `.night-crew/qa/spike-supabase/README.md` asserts twice (`:32-34`, `:724-727`) that its output blocks are "real captured output, not a reconstruction"; six blocks are hand-composed. All underlying facts re-verify and W1's GO stands, so this is a document-integrity repair, not a verdict change. · _morning triage 2026-07-25_ · new · lead: ten `HTTP nnn` annotations sit on `curl` calls carrying no `-w`/`-i`/`-D -`; `rtwatch` RECV lines are stripped of the `topic=` column `main.go:144` always prints; the `(5 more alice rows)` elision at `:821` implies 6 where the DB held 8 — re-run each block with real capture flags and paste what comes back.

- **B-02 · Migrate `BACKLOG.md` to B-NN handles so the validator does something** — 63 historical entries carry no handle, no one-line description and no lead, so `night-crew backlog check --file .night-crew/knowledge/BACKLOG.md` fails with 220 issues and gives us no signal at all. · _morning triage 2026-07-25_ · new · lead: run the checker, take its per-entry complaints as the worklist, and assign handles in file order from B-03 up; the "unrecognized status" complaints are mostly rich promoted-→ strings the grammar does not accept, so decide whether to simplify them or widen the grammar before bulk-editing.

- **B-03 · Route the run's gray areas through the decisions resolver** — `night-crew decisions audit --repo . --run 20260725` reports "no gray areas routed through the resolver yet", so preference coverage is undefined rather than low; the run parked its forks straight into `DECISIONS-NEEDED.md`. · _morning triage 2026-07-25_ · new · lead: the four forks this run parked are exactly the material the resolver wants — wire the park path to route through it so the coverage number has a denominator, otherwise the audit stays decorative every cycle.

- **B-04 · Watch `tests/purchasing.spec.js:1407` (FR-13)** — one flaky observation during run 20260725: died on `waitForLoadState('networkidle')` at the 30 s timeout, passed on retry, measured load 1.58 → 4.20. Deliberately not attributed — one observation is not an attribution. · _morning triage 2026-07-25_ · new · lead: if it reds again, re-run it in isolation on a quiet box before attributing; `networkidle` under a loaded box is the obvious suspect and is a different fix from a real product race.

- **B-05 · Humanize the History view's raw status token** — `workflows.html:2503` renders `escapeHtml(s.status || '')`, so a no-approval submission's history row shows the literal lowercase `completed` as user-facing copy. Cosmetic, and an improvement on the pre-card `pending`, but it is the eighth status-reading call site where the slate named seven. · _morning triage 2026-07-26 (T-23 decision 63)_ · new · lead: a small status→label map, folded into whichever card next touches `workflows.html`; the card's own framing "teach the client the DB's vocabulary" already covers it.

- **B-06 · Fix the stale comment at `tests/sync.spec.js:1584`** — reads *"requires_approval false → submit yields `'submitted'`"*; the server yields `'completed'`. It sits inside one of the status card's two acceptance specs. · _morning triage 2026-07-26 (T-23 decision 66)_ · promoted → `sync-rxdb-schema-and-replication` · lead: one line; fold into the next card that touches `sync.spec.js` rather than opening a card for it — assigned as that card's obligation 8 at morning triage 2026-07-27, after Card B did not touch the file and the item spent a second night waiting for "the next card".

- **B-07 · Re-measure leg 4 handover with `performance.now()`** — `browser/specs/leg4-leader-election.spec.js` uses `Date.now()` deltas on a box that steps its clock; G6's own run printed **−1545 ms** for the same measurement, so the quoted 47/65/87 ms are not measurements. The qualitative findings (one leader, follower silent, survivor replicates) are ordering facts and are unaffected. · _morning triage 2026-07-26 (T-23 decision 64)_ · new · lead: strike the three figures from card C's verdict and say "sub-second"; switch the spec to `performance.now()` before anyone quotes a number from it again.

- **B-08 · Merge-intents must disclose deviations at the time, not at triage** — card C's merge-intent promised `browser/` would carry its own `package.json`, `node_modules/` and lockfile; as built, `browser/` has zero dependencies and no lockfile and a new `vendor/package.json` + lockfile was created instead. The as-built design is better and is accepted (T-23 decision 65) — the defect is that the divergence was undisclosed, which devalues every future merge-intent. · _morning triage 2026-07-26_ · new · lead: process, not code — add a deviation line to the merge-intent template so "as-built differs from intent, here is why" is a field someone must fill rather than remember.

- **B-09 · `task test:` omits the `bdd:gen` dep, so every worktree silently runs 19 of 20 spec files** — reproduced at morning triage: a fresh worktree lists `Total: 559 tests in 19 files`, the main repo `560 in 20`; running `npx bddgen` regenerates `.features-gen/features/user-invite-onboarding.feature.spec.js` **byte-identical** and the count goes to 560/20. `task bdd:gen` already exists and `task bdd` / `task test:all` already depend on it — `task test:` (`Taskfile.yml:28-30`) is the one target that does not, so every card leg reported a full suite in good faith while a whole project contributed zero tests, with no error and no skip line. · _morning triage 2026-07-27 (T-25 decision 73)_ · **promoted → `test-harness-fail-loud`** (slate-planning session 2026-07-28, ledger T-27 decision 90 — §15k architecture-blocking bar: `sync-rxdb-row-visibility-rls`'s security gate cannot be trusted under a harness with this property). **✅ LANDED 2026-07-28 on `overnight-20260729-2`, merged to `dev` 2026-07-29 as `f35fa56`** — `bdd:gen` is in `test:`'s deps and the suite now resolves **21 spec files / 597 tests**, re-verified by adversarial re-execution at triage. 🛑 **Residual: the detection half is NOT closed — see B-23.** The floor that was supposed to announce the next silent-green is a hand-typed 20 and the tree is now at 21, so an empty Playwright project still reports PASS. The dep is fixed; the guard against its recurrence is not. · lead: one-line Taskfile dep, plus make an empty Playwright project a hard error so the next silent-green announces itself.

- **B-10 · The `await` on `clearApiCache()` has zero coverage** — dropping the `await` (keeping the call) leaves `tests/index.spec.js -g "logout deletes api-cache" --repeat-each=3` at 3 passed; deleting the call entirely reds 1. So the suite proves the call happens, not that it completes before `window.location.href` tears the page down — and both merge-intent A item 1 and the code comment name the await as "the entire disclosure". · _morning triage 2026-07-27 (T-25, adversarial finding F-1)_ · new · lead: a test that resolves `caches.delete` slowly and asserts the redirect has not fired yet; fold into whichever card next touches `index.html` — likely `sync-rxdb-schema-and-replication` via decision 70.

- **B-11 · Merge-intent notes must be re-read WHOLE after a repair, not appended to** — Card B's note disclosed its `sync.js` edit correctly in *Late additions*, then left "**`sync.js` was NOT edited**" standing two lines below it and "**No change to `sync.js`** — not `drainQueue`, not the `duplicate_submission` arms" standing under *Not done, deliberately* — enumerating precisely the two things the repair changed. A merger resolving a `sync.js` conflict against that section would have dropped the `queuedAt` sort the note's own *Late additions* says to keep. This is B-08's lesson one notch down: the deviation WAS disclosed at the time; the contradicting claims elsewhere in the same note were not retracted. · _morning triage 2026-07-27 (T-25 decision 76)_ · new · lead: make the repair leg re-read the whole note and strike contradicted lines rather than only appending — a note that contradicts itself is worse than no note, because the merger trusts it unattended at 3am.

- **B-12 · hq's BACKLOG.md fails `night-crew backlog check` — 220 issues across 71 entries** — only B-01…B-08 use the validating shape (`**B-NN · Title** — description · _origin_ · status · lead:`); roughly 63 legacy entries use the retired `title · description · origin · new` shorthand, with missing handles, missing leads, and free-text statuses the validator rejects. Pre-existing and not caused by any one run, but it means the store cannot be machine-read and new items are being appended to a document that does not validate. · _morning triage 2026-07-27_ · new · lead: a mechanical migration card, not a triage edit — rewrite the legacy entries into the current shape, assign handles in file order, and gate it on `night-crew backlog check` exiting clean.

- **B-13 · CLAUDE.md says `task prod:deploy` runs `task sw`; the Taskfile does not** — `Taskfile.yml:174-210` does a `git reset --hard origin/main` then `docker compose build`, so the **committed** `sw.js` is what ships and nothing regenerates it on the box. Harmless while `sw.js` is always committed with its source, and load-bearing for B-09's sibling hazard: it is why a staged-but-uncommitted file baked into `sw.js` reaches production. · _morning triage 2026-07-27 (T-25 decision 67, aside)_ · promoted → P1 `build-deploy-manifest-integrity` (slate-20260802) · lead: fix the doc to match the Taskfile, or add the dep — but decide which is intended before editing either, since `precache-manifest-from-head` changes what "correct" means here.

- **B-15 · Build the SW-enabled two-context harness that shrinks the attended two-device check** — the attended check is carried on every run because nothing automated can see a *running* service worker: `playwright.config.js:60` sets `serviceWorkers: 'block'` repo-wide, no spec overrides it (`test.use(` appears in zero test files), and `sync-rxdb-feasibility-spike`'s leg 3 already rules that the repo-wide setting must **never** change. So SW install, SW update → `controllerchange` → `ptr.js` reload, and offline-shell serving are proven by exactly nothing — `tests/sw-manifest.spec.js` asserts the precache manifest as a *file* and never installs one. Meanwhile the app-layer half is strong and green: re-measured 2026-07-27 on `dev` @ `937543a` at `--retries=0`, fresh DB per leg — `sync.spec.js` **58/0** (6.4m, 29 two-context pairs), `broadcast-rerender.spec.js` **6/0** (2.7m), offline queue `-g "offline|DBL"` **6/0** (2.6m), both armed flakes (`:446` LST-17, `:1198`) green on a quiet box. The gap is transport-layer, not app-layer. · _operator session 2026-07-27, after re-measuring the automated half against the standing flag_ · new · lead: a **separate** Playwright project/config (own `serviceWorkers: 'allow'`, own port + DB, repo-wide block untouched) driving two contexts against a real `sw.js` — assert install, assert a rebuilt `sw.js` drives `controllerchange` → reload, assert the offline shell serves, then run the convergence + offline-drain cells through it. Two scheduling notes: (a) `sync-rxdb-feasibility-spike` **needs this same harness** for its leg 3 (a network-first API fallback answering a *replication* request with cached JSON), so build it once, there, rather than twice; (b) frame it transport-agnostically — the cycle's hard cutover retires `sync.js`, and a harness written against today's op-log transport would be half-invalidated by it, whereas an SW+convergence harness is exactly the regression net you want *across* the swap. Residual after this lands: iOS-standalone-on-prod only — procedure in `reference/attended-two-device-check.md`.

- **B-14 · The morning-triage G4 discipline greps are VACUOUS in hq and read as clean** — the ritual greps `internal/journal` and `internal/workorder`; hq's Go tree is `backend/{cmd,internal}` and **neither package exists**, so the greps return empty because there is nothing to find. Any run or triage reporting them "clean" reports a vacuum — the same silent-green class as B-09, one layer up in the tooling. **Reconfirmed at morning triage 2026-07-29 and reported as N/A rather than PASS** — the adversarial reviewer was briefed to say so explicitly and did; the triage merge commit records them as a vacuum. Third consecutive triage carrying this. · _morning triage 2026-07-27 (T-25 decision 75)_ · new · lead: this binds the night-crew clone, not hq — carry it there; either make the greps assert the target packages exist first, or make them repo-conditional so an inapplicable gate reports N/A rather than PASS.

- **B-16 · A reviewer dropping a card's test DB reads as a passing suite, not a broken one** — on `overnight-20260729` Card C's first G6 reviewer ran `DROP DATABASE hq_test_go_c` during its cleanup while the implementer was still using it; the implementer caught it only by questioning a suspiciously fast green. Mechanism, characterised by a later reviewer: `internal/sync/access_test.go:29,33` and `jwtbridge_test.go:169,173` `t.Skipf` on both connect *and* ping failure, `pgxpool.New` is lazy so a missing database surfaces at `Ping` — as a **skip** — and non-verbose output is a bare `ok ... 2.948s`, indistinguishable from a full run. Destroying the environment is therefore indistinguishable from passing it. This is B-09's silent-green class one layer down, in the harness rather than the task runner. Secondary: the host now carries ~30 `hq_test_*` databases from successive runs' reviewers, none cleaned up. · _overnight-20260729 closeout (incident during Card C review)_ · **(b) promoted → `test-harness-fail-loud`** (slate-planning session 2026-07-28, ledger T-27 decision 90). **(a) is NOT in the card** — it is standing G6 dispatch text, written directly into the `overnight-20260729-2` launch prompt, which is where this entry's own lead says it belongs. **(a) remains open as a standing-prompt obligation until it lands in the durable G6 dispatch text rather than one night's prompt.** Note: the skip-on-unreachable defect is **repo-wide**, broader than this entry states — measured 2026-07-28 across `recipes`, `workflow`, `inventory`, `receipt` and `sync`. **✅ (b) LANDED and independently re-verified at triage 2026-07-29:** all seven converted packages exit 1 under a set-but-unreachable `DB_TEST_URL` in three separate shapes (dead port, unroutable host, live server / missing DB), and skip-on-**unset** is preserved at exit 0 — so destroying the environment is no longer indistinguishable from passing it. Two things worth carrying: the admitted semantics change is real and complete (`internal/workflow` runs 35 tests live and **0** set-but-unreachable — a broken DB now yields *no* hermetic signal from those packages rather than partial signal), and *"zero DB-backed skips remaining"* is true **behaviourally** while **93 DB-gated `t.Skip` sites remain in source across 19 files** by design for the unset case — the phrase reads like deletion and is not. 🛑 **The guard over (b) is weak — see B-22.** **(a) remains OPEN.** · lead: two parts, and the second is the real one — (a) reviewer prompts must forbid dropping any database the reviewer did not create, which this run applied by hand from the second reviewer onward and which belongs in the standing G6 dispatch text rather than in each night's prompt; (b) make the Go suite **fail** rather than skip when `DB_TEST_URL` is set but unreachable, keeping skip only for *unset* — exactly the asymmetric gate Card C built for `HQ_SYNC_SPIKE_LIVE` in the same run (`proxy_live_test.go`, "flag set + port dead ⇒ FAIL"), so the pattern is already in-tree to copy.

- **B-17 · `build-sw.js` justifies a load-bearing flag with a claim that is empirically false** — `build-sw.js:39` (mirrored verbatim into `.night-crew/knowledge/roadmap.md:383`, so the false claim now lives in two places) says `--name-only` C-quotes any path containing "a space or a non-ASCII byte". G6 measured it in a throwaway repo: `git ls-tree -r --name-only HEAD` returns `a file with spaces.html` **unquoted** and only C-quotes the non-ASCII case — `core.quotePath` escapes non-ASCII and control bytes, not spaces. The `-z` flag is still correct and still necessary (for the non-ASCII case); only the stated reason is half wrong. Cosmetic today, but it is the comment a future reader will reason from when deciding whether `-z` can be dropped. Related and worth recording alongside it: `ls-tree HEAD` reads **local** HEAD while the image builds from `origin/main`, so committing a file locally, regenerating `sw.js`, and pushing only the `sw.js` commit still reproduces the 404 class — strictly tighter than the `git ls-files` it replaced, correctly out of `precache-manifest-from-head`'s scope, but the DONE record should not read as if the class is fully closed. · _overnight-20260729, G6 review of Card A (nits 1 and 2, non-blocking)_ · new · lead: correct both copies of the sentence to name the real reason (`core.quotePath` escapes non-ASCII/control bytes, which is why `-z` is required), and add one line to the DONE card acknowledging the HEAD-vs-`origin/main` residual so it is not rediscovered as a regression.

- **B-18 · Two comments in `sync/proxy.go` describe code that does not exist or launder the evidence they record** — both surfaced by adversarial review of `sync-proxy-endpoint` after two repair rounds had already been spent, so they were carried out rather than fixed in-run. (a) `proxy.go:257-264` explains an `out.URL.RawPath = ""` assignment that **is not in the file** (`grep -n RawPath proxy.go` shows only reads). Harmless today because the stale RawPath disagrees with the prefix-trimmed Path, so `EscapedPath()` always falls back to re-escaping — but the hazard is the *obvious* fix: a refactor that trims RawPath in parallel with Path makes them agree again, `EscapedPath()` starts returning the caller's spliced bytes, and the wire path becomes attacker-controlled with the traversal check upstream none the wiser. (b) `proxy.go:203-205` logs the rejection with `r.URL.EscapedPath()` — the one function this card proved launders `%2f` — so an attacker sending `/sync/rest%2fadmin{` is recorded as `path=/sync/rest/admin%7B` and `reason=encoded_slash` is the only surviving signal. (c) Adjacent, deliberately deferred to protect that night's merge surface: `backend/cmd/server/main.go:436-438` reads as an all-clear ("inert until a deploy configures it") inches from the env-var names, with no "and do not configure it yet" — while setting them before row-visibility RLS lands grants every logged-in crew member read *and* write on the exposed schema. · _overnight-20260729, G6 final verification of Card C (nits N1-a, N1-b, F-3)_ · new · lead: (a) delete the comment or restore the assignment — do not leave a comment describing absent code in the one function whose false durable claim already cost a repair round; (b) log `r.RequestURI` or `r.URL.RawPath` alongside; (c) fold the `main.go` warning into whichever card next touches route registration.

- **B-19 · `unsubmitChecklist` orphans fail notes and nothing re-attaches them** — `repository.go:1281` detaches fail notes to `submission_id = NULL` on unsubmit; nothing ever re-attaches them and they carry no `answered_by`, so they leak permanently. Migration `0071`'s new unique index does **not** collide with them — Postgres treats NULLs as distinct — so the upsert work landed in `overnight-20260729` neither fixes nor worsens this. Confirmed genuinely pre-existing against `25fbc16` by adversarial review, i.e. not damage that run caused and reclassified. · _morning triage 2026-07-28 (T-26 decision 85, from Card B's disclosure)_ · promoted → P2 `workflow-unsubmit-failnote-reattach` (slate-20260802) · lead: decide the product question first — on unsubmit, should a corrective-action note survive to be re-attached on the next submit, or be discarded with the submission? The code currently does neither, which is the actual defect; fold into whichever card next touches `internal/workflow`'s submit/unsubmit path.

- **B-20 · `renderSyncBanner` paints the "Queued" badge onto Builder tab rows** — `sync.js:671` selects `[data-template-id]` document-wide, and the Builder tab renders those attributes too, so a template row in Builder gets a sync badge describing a queued *submission* it has nothing to do with. Reproduced in a browser by adversarial review (`builder=1 list=1 docwide=2`); byte-identical to base, so pre-existing. It is also what forced `[VOC-01]`'s locator to be scoped to `#checklist-list` during `overnight-20260729` — the test was fixed, the underlying selector was not. · _morning triage 2026-07-28 (T-26 decision 85, from Card B's disclosure)_ · promoted → P3 `sync-banner-builder-tab-scope` (slate-20260802) · lead: scope the selector to the checklist list rather than the document, then widen `[VOC-01]` back to document-wide so the test would catch a recurrence instead of being blind to it by construction.

- **B-21 · 14 of 33 commits on `overnight-20260729` carry a `Night-Crew-Card:` trailer git cannot parse** — a **blank line** between the trailer and `Co-Authored-By:` splits the trailer block, so `git interpret-trailers --parse` sees only the last paragraph. Affected: **all of Card C's commits**, **all four merge commits**, and the closeout — so `sync-proxy-endpoint` is entirely invisible to any trailer-parsing tooling, while all four merge-intent notes assert the trailer as a landed convention (textually true, mechanically false). Cards A and B each rewrote history mid-run specifically to fix this and Card C never did; the orchestrator's own merge commits then reproduced it. Triage declined to rewrite history — correcting four merge commits risks the record being preserved, for a gain that is cosmetic to tooling and zero to behaviour (T-26 decision 86). **Evidence 2026-07-29: the practice held for one night.** `overnight-20260729-2` has **0 of 32** commits whose trailer `git interpret-trailers --parse` cannot read, **all five merge commits included** — measured at triage, not by eye. 🛑 **Kept OPEN deliberately:** one clean night proves three agents got it right once, not that the emitter or the standing prompt text was fixed. The lead below is still unactioned. · _morning triage 2026-07-28 (adversarial finding F-1, undisclosed by the run)_ · new · lead: fix at the **emitter**, not the history — the trailer block must be emitted as one adjacent paragraph, and the standing card/G6/closeout prompt text should say so explicitly since three separate agents got it wrong the same way; optionally add a cheap post-merge check that every commit's trailer actually parses, because "the text is present" is what everyone verified and it is not the property that matters.

- **B-22 · `verify-test-harness.sh` Check B aggregates with OR, so six of seven packages can report `ok` on a dropped database and the gate still prints PASS** — the check runs one aggregate `cd backend && go test`, which is non-zero if *any* package fails, so reverting fail-loud in six of the seven converted packages (or in just `internal/inventory`) leaves `CHECK_B_AGGREGATE_EXIT=1` and the script reports harness OK with `ok github.com/yumyums/hq/internal/inventory 0.030s` sitting in a log it never reads. Only a 7-of-7 revert reds it. This is a stronger instance of the defect the card's own HANDOFF says it fixed in Check B2 — B2's unset direction genuinely aggregates AND across all seven. **The production fix is sound**: all seven packages exit 1 individually under three unreachable-DB shapes (dead port 110.7 s, unroutable host 109.7 s, live server / missing DB 0.02 s), so fail-loud does not depend on hanging. It is the guard that is weak. · _morning triage 2026-07-29 (T-28 decision 103a, adversarial finding F-1)_ · new · lead: make Check B per-package — `for p in $DB_PKGS; do go test ./$p || fail; done` — and assert the package count it iterated, so a shrinking `DB_PKGS` list announces itself; fold into whichever card next touches `scripts/verify-test-harness.sh`.

- **B-23 · Check A2's spec-file floor of 20 lost its bite during the very run that added it** — the floor is a hand-typed constant with a comment reading "19 static under ./tests + 1 generated", and `dev` did have 19 static specs. B1 added `tests/sync-schema.spec.js` on the same run, making 20 static + 1 generated = **21**, and nobody ratcheted the floor. Measured: move `features/` away so `bddgen` exits 0 emitting nothing → `Total: 596 tests in 20 files` → **A2 PASS, exit 0**; delete one real spec → `588 tests in 20 files` → **PASS**. The check's own comment says it exists to catch "a moved features dir, a renamed glob, a generator that exits 0 on an empty input set" — it now catches none of them. B-09's detection gap is re-opened, one file wide. Check A (the mechanism check) is genuinely falsifiable: removing `bdd:gen` from `Taskfile.yml` deps reds it, verified. · _morning triage 2026-07-29 (T-28 decision 103b, adversarial finding F-2)_ · new · lead: derive the floor instead of typing it — `$(ls tests/*.spec.js | wc -l) + 1` — so the next card that adds a spec cannot silently spend the margin; a hand-typed floor is the same class of defect as the thing it guards.

- **B-24 · `shoot.mjs` measurement 6 walks `.plate` with no population floor, and the file's own header claims it has one** — `shoot.mjs:50-61` asserts "Six of the seven measurements now pin their population; m1 is not a population walk and has nothing to pin." False: m6 — the A-1 arithmetic check **added in the repair round** specifically to close the "value present but not right" hole — walks `document.querySelectorAll('.plate')`, prints `checked`, and gates only on `b6.bad.length`. Renaming `class="plate"` to `class="plateX"` in `mockup.html` yields `[light] A-1 arithmetic: 0 counting plates reconciled, 0 disagreeing -> PASS`, `self-verification PASS RAW_EXIT=0`. The other three repaired checks **are** falsifiable, each confirmed by mutation: deleting every `Undo` → `58 measured (expected >=62) -> FAIL` exit 1; demoting the error plate's `Retry` → `61 measured -> FAIL` exit 1; renaming a `Restore mine` into the non-destructive allowlist → `12 destructive controls (11 row/batch) -> FAIL` exit 1. · _morning triage 2026-07-29 (T-28 decision 103c, adversarial finding F-3)_ · new · lead: add an `EXPECTED_COUNTING_PLATES` pin to m6 and correct the "six of seven" sentence in the header — a false claim about which checks are pinned is worse than an unpinned check, because it stops anyone looking; rides the card that redraws the plates for A-3.

- **B-25 · Armed reds must be named by title/grep handle, never by line anchor** — `tests/sync.spec.js:1198` is `await p.waitForTimeout(400)` inside a helper's loop body: it names no test, and the test it used to name is now at `:1372`. Known dead since 2026-07-24 (`runs/2026-07-24-autonomous/HANDOFF.md:102` says so, `:164` files a migration item) and **still armed in `slate-20260729-2`'s preconditions table** — so every card told to "expect `:1198`" for five nights was told to expect something unobservable, and every report saying "it passed" was unfalsifiable. Two cards hit it independently on `overnight-20260729-2`. `:446` `[LST-17]` is live and correct. · _morning triage 2026-07-29 (T-28 decision 100, fork D-5)_ · new · lead: change the slate/launch-prompt convention to carry a grep handle (`-g "[LST-17]"`) rather than `file:line`, then sweep the other armed reds for line anchors — the anchor rot is silent by construction, so the fix has to be at the convention, not at this one line.

- **B-26 · The file every slate inherits gates G1–G6 from does not exist, and G5 has never been defined** — every slate since 07-15 and every launch prompt say the gates are "unchanged from `reference/overnight-run-plan-20260707.md`". Confirmed by `find` at triage: **that file is nowhere in the repo.** The contract was recoverable from practice and the runs used it (G1 build+vet, G2 Go+Playwright, G3 red-first re-verified by G6, G4 `sw.js` idempotence + version parity, G6 adversarial review), but **G5 has no definition in any sense the runs use** — it appears only in a PRD and a QA-coverage table, in two unrelated senses — and is not practiced. So every run this month has been graded against a ladder with no written definition. Worse than `DECISIONS-NEEDED.md` D-6 filed it, which read as a documentation nit. · _morning triage 2026-07-29 (T-28 decision 101, fork D-6)_ · new · lead: write the file the prompts already point at, from practice, and record **G5 as never-defined** rather than renumbering — a retired number reads as history, a renumbered ladder hides that a gate was cited for a month and never existed.

- **B-27 · `inventory.spec.js:883` fails from cross-spec pollution, unattributed** — `item modal pre-fills search with current line item text` fails `Expected "Special Sauce", Received "Test Item"`. **Proven pre-existing by reproduction:** G6 ran the preceding specs with B1's new spec file entirely absent and got the byte-identical failure, and `inventory.spec.js` alone on a fresh database passes 150/150. Source is one of `broadcast-rerender` / `grant-enforcement-parity` / `index`. **The first-proposed mechanism is wrong** and is recorded as wrong so it does not become folklore: a `.first()` collision over a shared `eventDate` cannot be it, because the pending list is `ORDER BY created_at DESC` with no re-sort, so `.first()` is the newest row and `event_date` is not in the sort key. Likelier: `seedPendingPurchase` swallowing a failed POST (`tests/inventory.spec.js:70`). `playwright.config.js` defaults to `retries: 1`, which is why the baseline reads green — cards at `--retries=0` see it. Did not surface in either the run's or triage's `--retries=0` full suite, consistent with load/ordering sensitivity. · _morning triage 2026-07-29 (T-28 decision 102, fork D-7)_ · new · lead: fix `seedPendingPurchase` to fail loud on a non-2xx POST first — that is a one-line change that either fixes this or eliminates the leading suspect; do not attribute the pollution until it reds again on a quiet box in isolation.

- **B-28 · `receipt/worker.go`'s `parseEventDate` stamps a COGS period from server-local time, outside the app timezone** — `backend/internal/receipt/worker.go:1056` ends `return time.Now().Format("2006-01-02")`: when Mercury's `CreatedAt` matches none of the accepted layouts, the fallback is the **server's** local date (UTC in the container), and that value is written to `pending_purchases.event_date` — a **COGS period assignment**, and the one field that WINS the `COALESCE` in `pendingPeriodDateExpr`. So the single path that is exempt from card A1's zone unification is the path that produces the value the zone would otherwise decide: a receipt ingested between 20:00 New York and midnight UTC is filed to tomorrow's period, permanently, with no boundary for A1 to move. Bounded by how often Mercury emits an unparseable `CreatedAt` — which is unmeasured, and silent when it happens (no log, no `reason`). · _flagged in the `overnight-20260729-2` park note for card A1; carried out of the 2026-08-01 resume as explicitly out of footprint (`backend/internal/receipt` is not in A1's packages)_ · new · lead: two things, and the second matters more — (a) format in `users.DefaultTimezone` like every other site A1 moved; (b) the silent fallback is the real defect: an unparseable timestamp should set `reason` / log at WARN so a wrong period is *observable* rather than indistinguishable from a correct one. Fold into whichever card next touches `internal/receipt`; a one-line zone fix without (b) leaves the class intact.

- **B-29 · HQ changed what `/period-summary`'s completeness gate returns on 2026-06-06 and never told sales-processor — an undisclosed two-repo contract drift on the payroll gate** — Phase 21 published `completeness.pending_review_ids` as `pending_purchases.id` rows where `(created_at AT TIME ZONE 'America/Chicago')::date BETWEEN from AND to` (`21-SALES-PROCESSOR-CONTRACT.md:67`, archived in `875e26c` 2026-06-05), and that was **accurate to the code as it then stood**. On **2026-06-06** commit `cf959bd` — quick task `260606-0gh` (`.planning/quick/260606-0gh-completeness-gate-filters-pending-review/`), a task entirely separate from Phase 21 — replaced it with `COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date)` **and did not touch the contract**. That task's own PLAN.md reasons explicitly about *"a sales-processor query for May 25–31"*, so the counterparty impact was in view at the time. **It is a behaviour change, not a clarification:** under the published expression a late-discovered receipt (the PLAN's own example — a May 29 purchase ingested June 2) was filtered on `created_at`, fell outside the May window, and did **NOT** block May payroll; under the shipped code its extracted `event_date` puts it inside the window and it **does**. So sales-processor may have been receiving an undocumented `ready:false` — a **blocked payroll run** — since June 2026, with no way to reconcile it against the contract it was handed. Card A1's `:67` amendment initially compounded this by asserting *"the `COALESCE` is not new"* / "carried since Phase 21", which reads to the counterparty as *"you were always getting this"* — the opposite of the truth; that text is corrected on `card/a1-app-timezone-unify-new-york`. 🛑 **This is a SECOND notice to sales-processor, independent of A1's timezone notice.** The operator decides what is told and when; A1's contract edit states the drift on the page but does not deliver it. · _adversarial review of card A1 on run `overnight-20260801`, finding F1 — verified at source against `875e26c` and `cf959bd`_ · new · lead: three parts, and the third is the general one — (a) the operator must decide whether sales-processor is told about this separately from the timezone change, and whether any past `ready:false` needs reconciling; (b) audit the other `:NN` rows of both `*-SALES-PROCESSOR-CONTRACT.md` documents the same way, by diffing each published expression against the code at HEAD rather than trusting the prose — this one survived fourteen months of reading because nobody diffed it; (c) a published cross-repo contract has no mechanical link to the code it describes, so any quick task is free to silently invalidate it — the durable fix is a check that fails when a contract's stated SQL no longer appears in the handler it names, or failing that, a rule that touching a handler named in a `*-CONTRACT.md` requires re-reading that document.

- **B-30 · `[A1-TZ-02]` is a NEW armed red — the offline-queue period test reds under whole-suite load and greens in isolation** — `tests/sync.spec.js` *"a queued submission still lends its idempotency_key at 7:30pm CT [A1-TZ-02]"* failed once in a full `--retries=0` suite (`Error: the queued entry must STILL lend its key at 7:30pm CT`, `Received: null` at the `A1_T3` assertion) and passed everywhere else it was run: green in isolation via `-g "A1-TZ"` (5/5), and green **at the identical suite position — test 478 of 603 —** on an immediate re-run of the whole suite. Both suite runs were valid (one summary block each) on a box carrying **three concurrent Playwright stacks** (this card's, plus card C1's and B2's). **The shape of the failure names the mechanism, and it is a harness race, not a product defect:** the test stamps the queue entry's `period` at `T1`, asserts a hit at `T2`, then asserts a hit at `T3`. The `T2` assertion PASSED and the `T3` one returned `null` — which is exactly what happens if `page.clock.setFixedTime` had not yet taken effect at `T1`/`T2` (so both the stamp and the `T2` comparison used the REAL date and agreed) and had taken effect by `T3` (so the comparison used 2026-07-15 and disagreed). A period mismatch caused by the *product* would have reddened `T2` as well, and it did not; `[A1-TZ-01]`, which exercises the same three instants through `isCurrentPeriodEntry` directly, passed in both suite runs. Note `tests/sync.spec.js` is the **only** spec file in the repo that uses `page.clock` (grepped), so this is not cross-spec clock interference. 🛑 **Non-reproduction does not retire it** — that is precisely the standing rule for `[LST-17]` and B-27, and this entry exists so the next run that sees it does not rediscover it as a novel regression. Named by grep handle, not line anchor, per B-25. · _fix round of card `app-timezone-unify-new-york`, run `overnight-20260801`, G2 Playwright run 1 of 2_ · new · lead: make the clock installation observable before the fixture depends on it — assert inside the page that `new Date().toISOString()` equals `A1_T1` immediately after each `setFixedTime` and before the IndexedDB write, so a clock that has not landed FAILS AT THE SETUP LINE naming the real cause instead of surfacing three assertions later as a mysterious `null`. That is a test-only change and it converts a silent race into a loud one; do it on whichever card next touches `tests/sync.spec.js`.

- **B-31 · `index.html`'s launcher hides tools a user can actually reach — it gates tiles on literal slugs, and per-tab grants are not literal slugs** — `filterTilesByPermissions` (`index.html:121-130`) builds `allowed` as the literal set of `slug` values returned from `hq_apps` and hides any tile whose slug is not in it. Under this repo's per-tab grant convention a user can hold **only** `inventory-trends` — a legitimate grant — in which case they can reach `/api/v1/inventory/trends` and the Trends tab, and the launcher shows them **no Inventory tile at all**. That is the launcher concealing a surface the user *can* reach, which is exactly the harm decision 56 / obligation 4 names, in the direction nobody checked. `sync-rxdb-replication-and-conflict-handler` discharged obligation 4 in the other direction only (umbrella `inventory` ⇒ `inventory-trends`, `inventory-cost`, exported and tested in `sync-rxdb/client.js`) and correctly did not touch `index.html`. **Pre-existing, NOT created by that card** — the gating predicate is byte-identical to base. · _overnight-20260801 G6 review of Card C1 (obligation 4 ruled satisfied-as-worded but under-delivering its stated purpose)_ · new · lead: the launcher needs the same slug relation the client helper already exports — a tile should show when the user holds the umbrella slug **or any slug under it**, so `inventory-trends` alone reveals the Inventory tile (landing on the tab the grant actually covers). Do not re-implement the relation: `expandGrantSlugs` in `sync-rxdb/client.js` is the tested one, and its inverse is what the launcher wants. Belongs to whichever card next owns `index.html` — it is a launcher change, not a sync change, and folding it into a sync card would repeat the footprint mistake that made it invisible.

- **B-32 · A FAMILY, not two bugs — load/scale-sensitive 30 s timeouts that redden at whole-suite scale and green in isolation, on specs the reddening card never touched** — Two more members surfaced in `overnight-20260801`'s G6 review of card C2, and they are filed as ONE entry because the interesting thing is the shape they share with **B-27** and **B-30**, not either individually. **(a) `tests/workflows.spec.js` *"submitted checklist survives builder edit with assignment change [LC-02]"*** — 30 s timeout. 🛑 **NOT C2's, and the proof predates C2 by five days:** the identical test reddened the same way on **2026-07-26**, on card B's leg, on a tree where `workflows.html` and `tests/workflows.spec.js` were proven **byte-identical to base** (`.night-crew/qa/spike-supabase/captures/gate-20260726-card-b.txt`, `.night-crew/runs/2026-07-26-autonomous/HANDOFF.md:166`). It was refused attribution then. G6's own run reproduced the *class* on a different, untouched spec. **(b) `tests/inventory.spec.js:2908`** *"Receipt sync button › reload mid-run shows Syncing… (state survives via GET /status)"* — 30 s `networkidle` timeout in G6's run. **Proven not C2's by byte-identity:** `inventory.html` contains **zero** references to `sync-rxdb`/`conflict-notice` (grepped), and neither it nor `tests/inventory.spec.js` appears anywhere in the card's diff (`git diff --name-only` over the card range returns no inventory file). **The family, stated as a family:** a long-running spec at whole-suite position reds on a wall-clock 30 s timeout and greens on the same box in isolation or at `--retries=1`. The captured signatures differ and are recorded rather than smoothed over: member (a)'s call log on 2026-07-26 read *"element was detached from the DOM, retrying"* — a re-render racing a click — while member (b) hung on `networkidle`. Same ceiling, same load sensitivity, different proximate wait. B-27 (cross-spec pollution in `inventory.spec.js:883`), B-30 (`[A1-TZ-02]` clock race) and both members here all have it. Every one of them has so far been reproduced-or-not on a box carrying **two to four concurrent Playwright stacks**, which is this project's normal overnight shape and is the leading suspect for the shared cause. 🛑 **Non-reproduction does not retire any member** — the standing rule for `[LST-17]`, B-27 and B-30 applies to this entry in full, and "rare, mechanism known" must not be laundered into "not flaky". Named by grep handle, not line anchor, per B-25, except where the line is the only handle. · _fix round of card `sync-rxdb-conflict-notice-ui`, run `overnight-20260801`, filing G6's two unattributed reds_ · new · lead: stop diagnosing members one at a time and measure the family instead — record per-test wall-clock duration and the suite position for every run into a durable artefact, then check whether the reddening tests are the ones already nearest the 30 s ceiling under concurrent-stack load. If they are, the fix is a harness one (raise the ceiling for the known-slow specs, or stop running four stacks against one Postgres) and it retires four backlog entries at once; if they are not, each member is a real and separate product bug and should be split back out. Do this before any further per-member investigation.

- **B-33 · The run-evidence oracle cannot see ANY hq slate run's closing evidence — every night reads `no-run-evidence` at the next launch** — `night-crew run-evidence check --repo . --run 20260801` returns **`no-run-evidence`** on a night that landed 4 of 4 with a committed closeout record, a full conflict log, HANDOFF and DECISIONS-NEEDED. Two independent path mismatches, either of which alone is sufficient: **(1)** the oracle reads `.night-crew/runs/<runid>/{summary.json,metrics.jsonl,journal.jsonl}` — i.e. `runs/20260801/` — but the launch prompt mandates the run directory be `.night-crew/runs/2026-08-01-autonomous/`, so those three can never be found regardless of whether they are written; **(2)** it reads `reference/conflicts-<runid>.md` relative to the **repo root**, and **hq has no root `reference/`** — this repo's reference home is `.night-crew/knowledge/reference/`, which is where the conflict log, the closeout record, the slate and the launch prompt all correctly live. `night-crew launch-prompt` searches BOTH homes and resolves correctly; `run-evidence check` does not, so the two verbs disagree about where this repo keeps its artifacts. **Consequence:** §1 of `/nc-run`'s "has this slate already run?" guard is **inert in every night-crew target repo scaffolded this way**. The next launch will read a completed night as never having run and offer to execute the slate again. The §2 run-branch guard is the only thing that catches it, and only while the run branch stays unmerged — once triage merges `overnight-20260801` to `dev`, **nothing** stops a re-execution of an already-landed slate. · _overnight-20260801 closeout; the closeout record was written per `/nc-run` §3a and the oracle still could not see it_ · new · lead: this is a **night-crew clone** fix, not an hq one — teach `run-evidence check` the same two-home search `launch-prompt` already implements (scaffolded knowledge home first, then root `reference/`), and either resolve the run directory by glob (`runs/*<date>*`) or make the runid-keyed name the convention the launch prompt emits. Until it ships, treat §1's verdict in this repo as **unreliable** and lean on §2. Sibling of B-25/B-26 — a tool-side defect surfaced by a target repo.

- **B-34 · `purchasing.spec.js:1792` saw the server's clock run BACKWARDS 640 ms between two consecutive `now()` reads** — the FR-23 repurchase-reset test failed once during card B2's fix round on run `overnight-20260801`, then passed in isolation. The failure is not a timeout and not a fixture collision: two server-side `now()` reads taken in sequence returned times **640 ms apart in the wrong direction**, so a window computed from the first read excluded a row stamped by the second. 🛑 **Deliberately NOT folded into B-32.** B-32's family is wall-clock 30 s ceilings under concurrent-stack load — a *timeout* shape. This is a *monotonicity* shape, and the two have different fixes: B-32's is harness capacity, this one is either NTP slew on the host, a Postgres `now()`/`clock_timestamp()` mixup in the query under test, or a genuine ordering bug in the repurchase window. Filing them together would let the interesting one hide inside the boring one. · _morning triage 2026-07-31, run `overnight-20260801`; observed during B2's fix round, flagged by the run as "decide at triage whether it earns a number"_ · new · lead: first establish which clock moved — log both reads with `clock_timestamp()` alongside `now()` in the failing path, since `now()` is transaction-start time and two reads in **different** transactions can legitimately appear out of order under concurrent load while `clock_timestamp()` cannot go backwards. If `clock_timestamp()` is monotonic and `now()` is not, this is a transaction-boundary bug in the window computation and it is a real product defect, not a flake. Do this on whichever card next touches `internal/purchasing`.

- **B-35 · The standard gate command `go test ./...` DROPS a database it does not own — a B-16 trigger sitting inside the ladder itself** — `TestRowVisibilityRLS` drops and recreates its FDW fixture database on entry, and its default name is `hq_test_b2_fdw`. Any plain `go test ./...` in this repo therefore destroys `hq_test_b2_fdw` without asking, including a reviewer's or a concurrently-running card's. Reproduced at morning triage: the adversarial reviewer's first full `go test -p 1 -count=1 ./...` run silently dropped it before the reviewer switched to `HQ_RLS_TEST_DB=hq_adv_b2_fdw` for every subsequent probe. **This is exactly B-16's failure mode** — one agent destroying another's database, where the victim sees a broken suite rather than a stolen fixture — except that B-16 was about *reviewer cleanup* and this is baked into the **primary gate command every card runs**. Concurrent dispatch (three tracks, this project's normal overnight shape) makes collision the expected case, not the unlucky one. · _morning triage 2026-07-31; found by the adversarial re-execution of run `overnight-20260801`, disclosed by that agent against itself and not reported by the run_ · new · lead: the fixture database name must be unique per process, not a shared constant — derive it from the PID or require `HQ_RLS_TEST_DB` to be set and **fail loudly when it is not**, rather than defaulting to a name another agent may be using. Prefer failing over defaulting: a card that forgets the variable should get an error, not silently eat someone else's fixture. Sibling of B-16; fixing it there without fixing it here leaves the larger hole open.

- **B-36 · `internal/sync` prints `ok` and exits 0 while skipping B2's ENTIRE RLS attack suite — the row-visibility security gate cannot be distinguished from a gate that never ran** — `resolveSpikeConfig` locates the sync substrate by shelling out to `docker compose -p spike-supabase … port`. **Any** reason that command fails — docker absent, compose project down, ports not published, a different project name — is treated as "not configured" and becomes `t.Skip`, so the package reports `ok` and the ladder's recorded line **"G2 Go … 9 packages `ok`, 0 failed"** carries **zero** information about whether the security suite executed. Verified at triage by execution: with `PATH` stripped of docker, `go test -run TestRowVisibilityRLS ./internal/sync/` exits **0** with `ok`, and only `-v` reveals `--- SKIP` and the test's own honest line *"no sync substrate configured — skipping, and SKIPPED IS NOT PASSED"*. The test says the right thing; **nothing in the gate reads it.** 🛑 Distinct from **B-22**, which is about `verify-test-harness.sh` Check B aggregating with OR — that is the harness script; this is the suite's own skip path, and `verify-test-harness.sh` does not cover this axis at all (it grades only the `DB_TEST_URL` case). This is the repo's characteristic bug class — a check whose subject set can go empty — inside the card that exists to prove row visibility. · _morning triage 2026-07-31; reproduced by the adversarial re-execution of run `overnight-20260801`, finding F-1_ · new · lead: a security suite must not be allowed to skip silently in a gate context — gate `t.Skip` on an explicit opt-out (`HQ_SYNC_SUBSTRATE_OPTIONAL=1`) and `t.Fatal` otherwise, so an unreachable substrate reds the package instead of passing it. Until that lands, the standing triage rule recorded in ledger T-29 applies: gate evidence for `internal/sync` must include `-run TestRowVisibilityRLS -v` output showing subtests ran, because the package `ok` line provably cannot distinguish ran-and-passed from never-ran.

- **B-37 · `build-sw.js` exits 0 while silently dropping shipping assets from the precache — warnings only, no failure** — reproduced at triage: with git HEAD at a pre-bundle commit and the tip's assets present in the working tree, `node build-sw.js` printed `skipped (not in HEAD): vendor/rxdb.bundle.js` plus six `sync-rxdb/` and `sync-schema/` modules, emitted **22 files / 1481.9 KB instead of 29 / 2111.1 KB**, and **exited 0**. `workflows.html` IS precached and loads `sync-rxdb/bootstrap.js` as a module, so the shipped service worker would cache a page whose dependencies it deliberately omitted — Obligation 5's production-outage mode. It bit for real during run `overnight-20260801`'s merge 3, where regenerating mid-merge produced a 24-file manifest; the run caught it and recorded the ordering rule in the conflict log, but **the rule is procedural and nothing mechanical enforces it**. 🛑 Distinct from **B-17**, which covers the wrong `--name-only` C-quoting comment and the HEAD-vs-`origin/main` residual — neither of those is the exit code. Reading from git HEAD is by design (card `precache-manifest-from-head`); exiting 0 after dropping a file another precached file imports is not. · _morning triage 2026-07-31; reproduced by the adversarial re-execution of run `overnight-20260801`, finding F-2_ · promoted → P1 `build-deploy-manifest-integrity` (slate-20260802) · lead: make the skip fatal when the skipped file is reachable from something already in the manifest — parse the precached HTML/JS for local `import`/`src` references and exit non-zero if any resolves to a path that was skipped. A blanket "fail on any skip" is wrong (skipping genuinely uncommitted scratch files is the feature); the actionable invariant is that **nothing precached may import something not precached**. Pairs with B-13, which is why an uncommitted-but-staged file reaches production in the first place.

- **B-38 · A card the cutover hard-depends on was written down three times overnight and never became a card — HANDOFF's "What is NOT done" has no route into the roadmap** — Card B2 wrote *"RxDB push replication is REFUSED until a follow-up card writes WITH CHECK policies. That card owns…"* into `sync-schema/sql/0003_rls_policies.sql:104` during run `overnight-20260801` (commit `06283b9`). The same deferral then appeared in that run's `DECISIONS-NEEDED.md:63` as contributing context under Fork 1, and a third time in `HANDOFF.md` under **"What is NOT done"**. HANDOFF also stated it as one of *"two things triage must rule on."* **Morning triage 2026-07-31 resolved decisions 104–110 and none of them is that one** — it was neither decided nor deferred-with-a-reason, it simply fell through. The consequence was invisible for a full day: `sync-hard-cutover`'s roadmap entry said it depended on **three** cards when it actually depended on **five**, and `slate-20260801.md:292`'s milestone remainder told the operator the milestone was *"2 cards short"* and *"slatable in one further night"* — **already false at the moment it was signed**, because B2's own deliverable was concurrently writing down a third card. Found at the next evening's `/nc-slate-plan` by grepping the SQL for `WITH CHECK`, **not** by reading the roadmap; authored as `sync-rxdb-write-policies` (ledger T-30 decision 111). 🛑 **The failure is structural, not anyone's inattention:** DECISIONS-NEEDED forks get a triage round and a recorded resolution; **HANDOFF's "What is NOT done" gets read once and then the file is archived.** A deferral written by a card about its own successor has no channel that ends in a roadmap card. · _slate planning 2026-07-31 evening, `/nc-slate-plan` for `overnight-20260801-2`; the operator caught the milestone-remainder discrepancy at sign-off and the trace was run from there_ · new · lead: two parts. (a) **Ritual fix, cheap:** `/nc-morning-triage` should treat every "What is NOT done" bullet the way it treats a fork — each one exits triage as a roadmap card, a backlog entry, or an explicit "no action, because…" recorded in the ledger. Three dispositions, none of them silence. (b) **Mechanical check, the durable half:** a card that writes "a follow-up card" / "a separate card" / "until a later card" into shipped source or into its own report is asserting a dependency that does not exist yet — grep the run's diff for that phrasing at closeout and fail the closeout until each hit is either an existing roadmap card name or a newly authored one. Sibling of B-33 in kind (an oracle that cannot see what the ritual produced), but this one needs no tooling change to fix the first half.

- **B-44 · The same `'submitted'`/`'completed'` doc rot ledger decision 66 filed for `tests/sync.spec.js:1584` ALSO sits at `tests/repro-cut-task.spec.js:169`, and no obligation names it** — 🔢 _renumbered B-39 → B-44 at the B1 fix round, run `overnight-20260802`: three legs of this run each independently claimed `B-39`. The pre-step's B-39/B-40/B-41 merged first and keep those numbers; A1 took B-42/B-43; this leg's two entries move to B-44/B-45 and its new one to B-46._ — obligation 8 of `sync-cache-and-identity-hygiene` was written against one line anchor, so the fix landed on one line. `tests/repro-cut-task.spec.js:169` carries the identical claim (`requires_approval false → 'submitted', freezes the snapshot`) and the server sets `'completed'` (`backend/internal/workflow/repository.go:715-716`). Not folded in under the standing scope freeze: it reddens no key result, and `tests/repro-cut-task.spec.js` is outside B1's footprint and outside the card's obligations. · _run `overnight-20260802`, card B1, found while discharging obligation 8_ · new · lead: one word, and the same one. The reusable lesson is the shape, not the line: **an obligation written as a file:line anchor fixes an instance, not a class** — decision 66 recorded "the stale comment at `tests/sync.spec.js:1584`" when what it had found was a claim about submit status that the codebase repeats. A grep-defined obligation (`grep -rn "requires_approval false" tests/`) would have caught both and cost nothing more. Sibling of B-25 in kind (line anchors rot); this is the version where the anchor was not merely stale but too narrow from the day it was written.

- **B-45 · The full suite's single failure is drawn from a ROTATING POOL — three runs of one tree, three different failures, none of them the documented baseline** — 🔢 _renumbered B-40 → B-45 at the B1 fix round, run `overnight-20260802`, for the number collision recorded in B-44. **Strengthened at the same time with a third corroboration**, which changed the finding from "it moved once" to "the baseline is a distribution."_ — **Three** full `npx playwright test --retries=0` runs of the identical `overnight-20260802` B1 worktree produced **one failure each, and a DIFFERENT one each time**:
  1. `list page progress decrements when another device unchecks a field [LST-17]` (armed, expected) — card leg, isolated stack `TEST_PORT=8305` / `hq_n802_b1i_e2e`.
  2. `Users UI + Access hardening › FR-18/19: adding then removing an individual user grant persists across reload` (`tests/users.spec.js:1185`) — `page.waitForResponse` for `PUT /api/v1/apps/{slug}/permissions` never resolved, 30.4s timeout; **passes in 1.7s** re-run in isolation immediately afterwards. Same stack, DB dropped and recreated before the run.
  3. `tests/inventory.spec.js:2994 › Receipt sync button › manual sync chip shows Synced from {date} using lookback_days` — a 30s `networkidle` timeout; **passes in 1.4s** in isolation. Observed by B1's **G6 adversarial reviewer**, a fresh context on its own stack, which makes it an independent third sample rather than a repeat of the card's own environment.

  4. **(B1 fix round, same night.)** The same tree run **concurrently with another leg's full suite** produced **27 failures in 56.7m**; re-run **alone** on a fresh database it produced **0 failures in 24.2m** — 741 passed, 6 skipped, one summary block. **Nothing in the pool failed at all, including all three armed reds and B-27.** 24 of the 27 contended failures were timeout-class (`Test timeout exceeded`, `waitForResponse`, `toBeVisible`), concentrated in the live cross-device `sync.spec.js` convergence matrix.

  🛑 **Run 4 identifies the MECHANISM, and it is not flakiness in any test.** The 2.3x wall-clock difference on an identical tree is CPU starvation: two 750-test serial suites, each with its own `go run` server, on one box. Under that load the live-sync tests miss their propagation windows. **The pool is contention-driven, so the ritual's per-card verdict depends on what OTHER legs happen to be running** — which no card can see, control, or report. That is the actionable core of this entry.

  Each run's other two titles passed. **B-27 itself (`tests/inventory.spec.js:883`) was green in ALL THREE**, so the documented baseline red did not reproduce once. 🛑 **The point is not any of the three tests.** A 743-test serial suite on one shared `go run` server + one shared database has a ~1-failure-per-run floor drawn from a rotating pool — so **every card's G2 verdict is judged against a baseline that is not stable between runs of the same commit**, while the ritual's own rule (judge against the B-27 baseline, name every survivor by full title) assumes it is. In the reviewer's words: **the baseline is a distribution, not a list**, and the ritual currently has no way to express that. · _run `overnight-20260802`, card B1 G2 leg (runs 1–2) and the B1 G6 adversarial review (run 3)_ · new · lead: **do not chase any of the three tests first.** Establish the distribution: run the suite N≥5 on an untouched `dev` and record which titles fail each time. If the pool is real, two things follow — (a) the actionable fix is the config comment's own escape hatch, `PW_WORKERS` with per-worker `TEST_DB_NAME`+`TEST_PORT`, because all three observed failures are `waitForResponse`/`networkidle`/`toContainText` timeouts under contention on the single server rather than assertion mismatches; and (b) the RITUAL needs a second verdict shape — "one unexplained timeout-class failure, re-run in isolation and passed" must be expressible as a pass-with-note, or every honest card will keep having to argue its baseline. Sibling of B-27 in kind (a red the baseline names) but larger. Related: run 1's Playwright process **hung after printing its summary** (webServer child not reaped; it held :8305 for 35 minutes until killed by hand), and that run's failure stack frame printed a path from **another card's worktree** (`a1-sync-replication-scope-per-checklist`) while the clean re-run printed the correct one — worth confirming the shared `/tmp/playwright-transform-cache-1000` is not serving cross-worktree sourcemaps.

- **B-46 · With no token the launcher paints NOTHING — no message, no spinner, no affordance, permanently** — `index.html`'s fail-closed identity branch is correct about what it refuses (it must not paint the previous user's tiles), but it has no visual half. With no verified `/api/v1/me` and no device identity token, `readCachedApps(deviceId)` returns `null` because `deviceId` is null, `filterTilesByPermissions` is therefore never called, and `.grid` stays at the inline `visibility:hidden` it ships with. The crew member sees the wordmark and empty space. There is no "you're offline", no "sign in to load your tools", no retry — and nothing on screen distinguishes it from a broken app. It **self-heals** on the next `index.html` load that reaches the network, which is why it is not blocking and why it did not surface as a defect during the card. 🛑 It is nonetheless the ordinary first-launch-offline experience on a truck. · _run `overnight-20260802`, card B1 G6 adversarial review, NON-BLOCKING-6; filed rather than built under the run's standing scope freeze — a UI for this is a design decision, not a fix round's call_ · new · **destination: next milestone** · lead: this is a UI-SPEC-shaped item, not a one-liner — the State Enumeration Table needs an "identity unverifiable" row, and the copy has to distinguish "offline, we'll load your tools when signal returns" from "you are signed out" without claiming to know which. Pairs naturally with any card that touches the launcher's empty/loading states; the mechanism side is one `else` branch in `checkAuth()`.
