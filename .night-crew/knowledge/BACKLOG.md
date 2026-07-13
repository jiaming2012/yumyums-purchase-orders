# Backlog — advisory items that ride future cards

> Durable parking lot for triage-surfaced items that aren't roadmap cards yet but must
> survive run-to-run (HANDOFF is rewritten each run). Format: `title · description · origin ·
> status`. Promote to a roadmap card with `promoted → <card>`; drop with `dropped — reason`
> (struck through, kept as record).

- **Users stale-E2E repair** · `tests/users.spec.js` has two Access-tab tests navigating dead
  `#t3`/`#s3` DOM (removed in the 3-tab→2-tab refactor; Access now renders into `#s2`).
  Features work; tests can't run — marked UNPROVEN (stale-test), not BROKEN. Repoint
  `#t3`/`#s3` → `#t2`/`#s2`. Folds into the **Users Activity-4 prove-UNPROVEN WO** (low effort).
  **Also (overnight-20260712 users-confirm-absence, G6-confirmed):** while repairing, rename the
  misleading `s3` local var in `renderAccess` (`users.html:466`) — it fetches the live `#s2` node
  but its name is a 3-tab holdover that likely seeded this bug. · origin: triage 2026-07-10 (D-4)
  · updated 2026-07-11

## Activity-4 fix-cards (from Activity-2 confirm-absence graduations — code-fix + regression-test WOs)

> Distinct from test-only prove-UNPROVEN WOs: these are **confirmed-BROKEN** flows where a cited
> line proves the behavior absent. Each = code fix (front+back) + red-first regression test.

- **Operations FR-4 — yes/no "No" corrective-action enforcement** · A "No" answer never blocks
  submit: `evaluateFailTrigger` handles only `out_of_range` (`workflows.html:1656-1668`), yes/no
  fields carry no `fail_trigger` (`workflows.html:558,724`), submit validation short-circuits on
  `!f.fail_trigger` (`workflows.html:2398-2405`), server `validateFailNotes` checks only
  `out_of_range` (`handler.go:80,101`). The "No" fail card (`workflows.html:2068`) is cosmetic.
  Fix: make a failing "No" require a corrective fail note front+back (mirror the temperature path)
  + red-first AC-3 test. · origin: overnight-20260712 ops-confirm-absence (G6-passed) · new
- **Operations NFR-3 — photo-required-at-submit enforcement** · No photo gate on submit/resubmit:
  frontend checks only note+severity (`workflows.html:2397-2419`), the `fld-photo-required` toggle
  + reject `require_photo` feed a banner only (`workflows.html:2024-2025`), backend
  `validateFailNotes` has no photo check (`handler.go:54-88`), submit runs one validation with no
  photo gate (`handler.go:458`). `PhotoURL` is storable (`model.go:92`) but never required.
  Fix: block submit/resubmit until a required photo is attached, front+back, + red-first test.
  · origin: overnight-20260712 ops-confirm-absence (G6-passed) · new
- **Onboarding NFR-5 — reopen/reject of a video-led section is a silent no-op** · A section
  whose first `ob_items` row is a `video_series` never reverts to active on reopen or reject:
  `ReopenSection` selects the first item with no type filter (`db.go:1015-1017`) and deletes progress
  by the parent `ob_items.id` (`db.go:1040`), but video progress is keyed by `ob_video_parts.id`
  (`db.go:970-978`, required for completeness at `db.go:645-651`), so the DELETE matches zero rows and
  `isSectionComplete` stays true; handler returns `{"ok":"true"}` masking it. FAQ-led sections are
  safe. Shared defect — hits BOTH `/reopenSection` (FR-9, crew) and `/rejectSection` (FR-15, manager).
  Fix: resolve the first *checkable unit* by item type (video_part / faq / sub_item / item) and delete
  its progress; red-first test on the seed's video-led Equipment Training §.
  · origin: overnight-20260712 onboarding-confirm-absence (G6-passed) · new
- **Purchasing FR-18 — History tab is a static stub (frontend build WO)** · `#s4` is a hardcoded
  stub (`purchasing.html:156` "Past shopping runs will appear here"); `show(n)` (`:776-782`) only
  toggles display; there is no `renderHistory` and no `GET /shopping/history` call anywhere in the
  frontend (grep = 0 hits). Backend `GetShoppingListHistory` (FR-17, `service.go:396-458`) works but
  is never wired to the UI. **Confirmed-BROKEN at Activity-1 PRD-drafting; re-confirmed by the
  overnight-20260712 sweep.** NOT waived (unlike Inventory Trends/Cost D-3) — the backend exists, so
  the absent UI is a real gap in a shipped feature. Fix: implement `renderHistory` + wire
  `/shopping/history` into the History tab, + rewrite the 4 dead History tests to drive the new UI.
  · origin: PRD-purchasing-hardening Activity 1 (re-confirmed overnight-20260712) · new

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
- **Purchasing FR-7 vacuous shopping-tab test (dropped WORKING→UNPROVEN)** · `Shopping tab shows
  stub when no active list exists` ends in `expect(text.trim().length).toBeGreaterThan(0)`
  (`tests/purchasing.spec.js:127`) — a generic-content tautology (passes for stub, populated list,
  or error string). Neither the empty-state stub text nor the grouped vendor-section render FR-7 names
  is asserted anywhere. Activity-4 test-only WO: seed an active shopping list and assert (a) the
  empty-state stub text with no list, and (b) grouped vendor sections + per-item check buttons +
  thumbnails + aisle locations when populated. · origin: overnight-20260712 purchasing-test-audit
  (G6-passed) · new
- **Inventory NFR-1 double name-normalization gap** · Two surfaces persist un-normalized text while
  the rest title-case: (1) `UpdateItemHandler` (`backend/internal/inventory/handler.go:1129-1131`)
  writes `input.Description` raw — item *edit* skips `normalizeItemName` (create/confirm/vendor-create
  all normalize); (2) `ConfirmPendingPurchaseHandler` (`:660-664`) upserts the **vendor** raw while
  receipt line-items ARE normalized (`:762`) — so FR-4's "vendor upserted title-cased" text is
  inaccurate. Neither is a G3 BROKEN (behavior present, output just un-normalized). WO: add
  `normalizeItemName` to both paths + assert title-cased output; the NFR-1 test-only WO should cover
  the 3 named surfaces AND these two gaps. Also correct FR-4's PRD text re vendor normalization.
  · origin: overnight-20260712 inventory-confirm-absence (G6-passed) · new
- **Inventory ~40 data-dependent test guards (cleanup)** · `tests/inventory.spec.js` carries ~40
  `if (await X.count() > 0) return;` / `if(count>0){…}` guards that silently pass when a seed is a
  no-op (PRD §Verification). None drops a WORKING flow (each guarded assertion is backstopped by a
  guaranteed real seed or an unguarded sibling), but they're QA-KR-1 cleanup: convert to unguarded
  (or self-seed) so a seed miss reddens. Representative: FR-2 tax/grand-total at `:1039-1042,
  1058-1061`. Rides the Inventory Activity-4 test-hardening WO. · origin: overnight-20260712
  inventory-test-audit (G6-passed) · new
