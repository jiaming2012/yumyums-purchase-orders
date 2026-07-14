# Backlog — advisory items that ride future cards

> Durable parking lot for triage-surfaced items that aren't roadmap cards yet but must
> survive run-to-run (HANDOFF is rewritten each run). Format: `title · description · origin ·
> status`. Promote to a roadmap card with `promoted → <card>`; drop with `dropped — reason`
> (struck through, kept as record).

- **Onboarding video-pipeline E2E fixture (prove FR-16 + NFR-4)** · Stand up a DO Spaces bucket +
  an `ffmpeg` binary on the E2E host so the video presign→PUT→FFmpeg transcode/thumbnail path
  (FR-16, `handler.go:540-640`, `video.go:22-206`) and the `503 video_storage_not_configured`
  fallback (NFR-4) can be exercised. Both are fully implemented and **waived** from the Eng-KR
  denominator at triage 2026-07-13 (D-5) as env-gated — this fixture is the preserved prove-path:
  when built, it flips FR-16/NFR-4 UNPROVEN→WORKING (or reddens them). Needs operator-supplied
  Spaces creds. ~4–5h test-infra build. · origin: triage 2026-07-13 (D-5, waive-now-but-preserve)
  · new

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
  · origin: overnight-20260714 ops-nfr3 (in-footprint deferral, G6-confirmed) · new

- **Users `users.html:122` orphaned `<div id="s3">` cleanup** · After `users-stale-e2e-repair`
  repointed the Access tests to `#s2` and renamed the `renderAccess` var, the dead `<div id="s3">`
  at `users.html:122` (3-tab→2-tab refactor leftover) is now fully orphaned — nothing references it.
  Trivial removal; fold into any future Users card. · origin: overnight-20260714 users-stale G6
  (optional cleanup) · new

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
  for all six types incl. photo. · origin: overnight-20260712 ops-test-audit (G6-passed) · new
- **Operations FR-10/FR-12 vacuous reject test** · `reject item with comment`
  (`tests/workflows.spec.js:485-508`) wraps its whole body in `if (await flagBtn.isVisible())` with
  no `expect` — genuinely vacuous. Maps to FR-10/FR-12 (already UNPROVEN). Use as the starting point
  for the FR-12 test-only WO (the assertion to add). · origin: overnight-20260712 ops-test-audit · new
- **Onboarding 6 conditional-skip guard sites** · `tests/onboarding.spec.js` has 6 guard/`return`
  sites that skip an assertion when the seed shape differs: `:991` (FR-13, PRD-flagged), `:148-151`
  (FR-3), `:250-259`+`:826-841` (FR-5), `:304-306` (FR-2), `:700-745` (FR-13 UI), `:2104-2107`+
  `:2136-2139` (FR-15). All flows STAY WORKING (each has an unconditional sibling assertion) — these
  are hardening flags, not drops. Replace each guard with a self-seeded fixture so a shape mismatch
  reddens instead of silently skipping. Rides the Onboarding Activity-4 test-hardening WO.
  · origin: overnight-20260712 onboarding-test-audit (G6-passed) · new
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
  inventory-test-audit (G6-passed) · new

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
  (PARK) · new
- **WO-photo-s3-harness** · A way to exercise photo presign→PUT→public-URL in E2E (mock S3 or a test
  DO-Spaces bucket); the ephemeral stack sets no `SPACES_*` so `UploadHandler` returns 503. **Unblocks
  P-1 (Onboarding FR-18), P-2 (Inventory FR-27), P-3 (Operations NFR-2 PUT leg)** — 3 flows in one
  harness. Related to (but distinct from) the Onboarding video-pipeline fixture above (video vs item/
  receipt photo). · origin: overnight-20260715 (A4/E3/D2 PARKs) · new
- **WO-offline-indexeddb-harness** · A dedicated Playwright project with the service worker +
  IndexedDB enabled (`playwright.config.js:29` blocks SWs today) to test offline sync/queue/conflict
  + draft-persist-across-redirect. **Unblocks P-4 (Operations NFR-5), P-5 (Operations NFR-7 draft
  leg).** · origin: overnight-20260715 ops-prove-cross (PARK) · new
