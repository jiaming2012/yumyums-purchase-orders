# HANDOFF — overnight-20260712 (for the morning of 2026-07-12)

> **Run branch:** `overnight-20260712` (cut from `dev`; never pushed, `main` untouched).
> **Slate:** `.night-crew/knowledge/reference/slate-20260712.md` (batch-signed 2026-07-11).
> **Scope:** Activity 2 (confirm-absence sweeps) + Activity 3 (test-audits) — the 5 apps ×
> 2 cards = **10 cards, serial**, riskiest (Inventory) last. **Inspection + status
> reclassification only** — no app code, no fixes, no tests written, no DB, no E2E run.
> **Result:** 10/10 cards complete, **every card G6-passed**, 0 cards parked. **10 commits,
> 100% docs-only** (all under `.night-crew/knowledge/`; footprint verified clean — no app
> source/test/DB touched). Reader = the operator; resolve via `/nc-morning-triage`.

## TL;DR (what changed)

- **3 new confirmed-BROKEN graduations** (each cited + G6-re-verified at the line):
  Operations **FR-4** & **NFR-3**, Onboarding **NFR-5**. All 3 → Activity-4 **code-fix +
  regression-test** cards.
- **1 WORKING→UNPROVEN drop** (measured vacuous test): Purchasing **FR-7** (generic-content
  tautology). First hard QA-KR-1 data point.
- **1 priority-UNPROVEN cleared** (confirm-absence NEGATIVE): Operations **FR-12** (reject
  handler is complete) → stays UNPROVEN, drops "priority".
- **1 doc-consistency fix:** Purchasing **FR-13** inline mark reconciled WORKING→UNPROVEN to
  match its authoritative tally (no count change).
- **1 operator fork surfaced** (not decided by the run): **D-5** — waive Onboarding FR-16 +
  NFR-4 (env-gated video pipeline)? See `DECISIONS-NEEDED.md`.
- **0 graduations** on Users, Purchasing, Inventory confirm-absence; **0 drops** on Ops,
  Users, Onboarding, Inventory test-audits. Pre-existing waived-BROKEN unchanged: Purchasing
  FR-18 (re-confirmed), Inventory FR-24/25.

## Per-card reclassification table

| # | Card | G6 | Reclassifications (cited) | Tally after |
|---|---|---|---|---|
| 1 | ops-confirm-absence | PASS | **FR-4 → BROKEN** (yes/no "No" corrective-action never blocks submit — `workflows.html:1656-1668,2398-2405`; `handler.go:80,101`). **NFR-3 → BROKEN** (photo-required never enforced at submit/resubmit — `workflows.html:2397-2419`; `handler.go:54-88,458`). **FR-12** confirm NEGATIVE (reject flips `status='rejected'`, `repository.go:902-909`) → stays UNPROVEN. | 10 W / 15 U / **2 B** |
| 2 | ops-test-audit | PASS | 0 drops — 10 WORKING non-vacuous. Notes: FR-15 builder-UI/photo coverage gap; `reject item with comment` test vacuous (`workflows.spec.js:485-508`) but maps to already-UNPROVEN FR-10/12. | 10 W / 15 U / 2 B |
| 3 | users-confirm-absence | PASS | 0 graduations — 16 stay UNPROVEN. FR-9 email-immutable = feature-by-absence (not broken); FR-16/17 render into live `#s2` = stale-test (not broken-render). Cosmetic: `renderAccess` var named `s3` targets `#s2`. | 10 W / 16 U / 0 B |
| 4 | users-test-audit | PASS | 0 drops — 10 WORKING non-vacuous (hollow-visibility probes FR-8/13/14 + minted-token FR-3/5/12 all require the real path). Stale-test fold-in confirmed (Access-tab tests → dead `#t3/#s3`, back no WORKING flow). | 10 W / 16 U / 0 B |
| 5 | onboarding-confirm-absence | PASS | **NFR-5 → BROKEN** (reopen/reject of a video-led section is a silent no-op — `ReopenSection` deletes progress by parent `ob_items.id` `db.go:1040`, but video progress is keyed by `ob_video_parts.id` `db.go:970-978`, required for completeness `db.go:645-651`; FAQ-led sections safe; hits BOTH `/reopenSection` FR-9 + `/rejectSection` FR-15). FR-16/NFR-4 present-but-untested → **waiver fork D-5**. | 23 W / 10 U / **1 B** |
| 6 | onboarding-test-audit | PASS | 0 drops — 23 WORKING non-vacuous. 6 conditional-skip guard sites (incl. PRD-flagged `:991`) — all flag-only (each has an unconditional sibling assertion). | 23 W / 10 U / 1 B |
| 7 | purchasing-confirm-absence | PASS | 0 graduations — 18 stay UNPROVEN; FR-18 remains only BROKEN. All 4 scheduler crons + `ApprovePO` + 5 D-1 no-UI handlers confirmed real. **FR-13 inline mark reconciled WORKING→UNPROVEN** (tally was already correct). | 7 W / 18 U / 1 B |
| 8 | purchasing-test-audit | PASS | **FR-7 → UNPROVEN** (only test ends in `expect(text.trim().length).toBeGreaterThan(0)`, `purchasing.spec.js:127` — asserts neither stub text nor vendor-section render; G6 confirmed no other asserting test). 6 kept WORKING. | **6 W** / **19 U** / 1 B |
| 9 | inventory-confirm-absence | PASS | 0 graduations — 19 stay UNPROVEN; FR-24/25 remain waived-BROKEN. NFR-1 named contract (confirm/item-create/vendor-create) normalizes; **2 latent norm-gaps flagged** for WO (UpdateItem edit `handler.go:1129-1131`; confirm-vendor raw `:660-664` — FR-4 text off). | 19 W / 19 U / 2 B |
| 10 | inventory-test-audit | PASS | 0 drops — 19 WORKING non-vacuous. Go DB-guard skip = env-not-vacuous; synthetic-DOM tests (FR-17/19) drive real handlers; FR-2 backstopped by real `/pending-seed`. ~40 data-dependent test guards → cleanup note. | 19 W / 19 U / 2 B |

**Every BROKEN citation above was independently re-verified by a separate fresh G6 subagent
at the cited line** (the NFR-5 graduation had all three causal links checked; FR-4/NFR-3 had
the enforcement-absence confirmed against a full grep of the submit path).

## Cycle status-tally movement (the KR denominators)

| App | Before (Activity 1 sign-off) | After this run | Δ |
|---|---|---|---|
| Operations | 10 W / 17 U / 0 B | 10 W / 15 U / **2 B** | +2 BROKEN (FR-4, NFR-3); FR-12 de-prioritized |
| Users | 10 W / 16 U / 0 B | 10 W / 16 U / 0 B | no change (confirmed present-but-untested) |
| Onboarding | 23 W / 11 U / 0 B | 23 W / 10 U / **1 B** | +1 BROKEN (NFR-5) |
| Purchasing | 7 W / 18 U / 1 B | **6 W / 19 U / 1 B** | FR-7 W→U (vacuous test) |
| Inventory | 19 W / 19 U / 2 B | 19 W / 19 U / 2 B | no change (FR-24/25 waived; NFR-1 gaps noted) |

- **Eng KR-1 (0 known-broken):** the confirmed-BROKEN denominator over **built** flows is now
  **exact = 4** → Ops FR-4, Ops NFR-3, Onboarding NFR-5, Purchasing FR-18. (Inventory FR-24/25
  waived-unbuilt, D-3; Onboarding FR-16/NFR-4 pending the D-5 waiver decision.)
- **QA KR-1 (vacuous tests 23 → 0):** first measured landscape. **1 hard vacuous WORKING test
  found & dropped** (Purchasing FR-7). The suites carry **very few `test.skip`** — the vacuity
  is overwhelmingly **data-dependent early-return guards** (Onboarding ×6, Inventory ~40) and
  **synthetic-state** render tests. These do NOT false-green their flows (each has a real
  sibling/seed backstop) but are the cleanup surface the "23" estimate was pointing at. The
  8-`test.skip` / 8-guard-return split in the OKR should be re-derived from these findings at
  triage — the real count is measured per-app in the sweep records now.

## Activity-4 backlog (sized — ready to card-split next slate)

### A. Confirmed-BROKEN → code-fix + red-first regression test (4 cards)
1. **Ops FR-4** — yes/no "No" corrective-action enforcement (front+back, mirror temp path). `BACKLOG.md`.
2. **Ops NFR-3** — photo-required-at-submit/resubmit enforcement (front+back). `BACKLOG.md`.
3. **Onboarding NFR-5** — video-led reopen/reject no-op; resolve first *checkable unit* by item
   type in `ReopenSection`; red-first test on the seed's video-led Equipment Training §. Hits
   both `/reopenSection` + `/rejectSection`. `BACKLOG.md`.
4. **Purchasing FR-18** — History-tab is a static stub; frontend build (`renderHistory` + wire
   `GET /shopping/history`) + rewrite the 4 dead History tests. Pre-confirmed; re-confirmed. NOT
   waived (backend exists). `BACKLOG.md`.

### B. Test-only WOs (write red-first assertion; graduate to fix only if red) — per app
- **Operations:** 15 UNPROVEN. · **Users:** 16 UNPROVEN (incl. 2 stale-test FR-16/17 → `#t3/#s3`
  ⇒ `#t2/#s2` repoint + rename the `s3` var). · **Onboarding:** 10 UNPROVEN (FR-16/NFR-4 pending
  D-5 waiver). · **Purchasing:** 19 UNPROVEN (incl. newly-dropped FR-7 — assert stub text +
  vendor-section render). · **Inventory:** 19 UNPROVEN.
- Ops alone = 15; total test-only surface ≈ **79 UNPROVEN flows** across 5 apps → this needs
  card-splitting (per-app, likely per-tab for Inventory) at the next slate.

### C. Test-hardening notes (ride the WOs above — not standalone cards)
- Ops FR-15 builder-UI/photo coverage gap; FR-10/12 vacuous reject test (`workflows.spec.js:485`)
  as the FR-12 WO's starting assertion.
- Onboarding 6 conditional-skip guards (incl. `:991`) → replace with self-seeded fixtures.
- Inventory **NFR-1 double normalization gap** (UpdateItem edit `:1129-1131` + confirm-vendor
  `:660-664`) — add `normalizeItemName` to both + correct FR-4 PRD text. Inventory ~40 test-guard
  cleanup (convert `if(count>0){…}` to unguarded/self-seed).

*(All of A + the test-hardening notes are itemized in `.night-crew/knowledge/BACKLOG.md` with
cited lines and origin tags.)*

## Roadmap state

All 10 Activity-2/3 rows flipped **PLANNED → DRAFTING** in `roadmap.md`. Per the slate, **DRAFTING
→ DONE happens at morning-triage sign-off, not overnight** — the operator flips them after
reviewing this HANDOFF. Activities 4 (prove-&-fix WOs) and 5 (cycle gate) remain PLANNED.

## Decisions needed (operator, at triage)

- **D-5 — waive Onboarding FR-16 + NFR-4 (video pipeline) as environment-gated?** Fully
  implemented (`video.go`), untestable without S3 creds + ffmpeg. Run recommends waive (parallels
  D-3 Trends/Cost). See `DECISIONS-NEEDED.md`. This is the **only** open fork — no cards parked.

## Suggested triage order

1. Re-verify the **3 new BROKEN citations** at their lines (FR-4, NFR-3, NFR-5) — the load-bearing
   findings (30 min; the NFR-5 causal chain is the one worth a careful read).
2. Rule **D-5** (waive FR-16/NFR-4 or schedule an env fixture).
3. Sign off the 10 DRAFTING rows → DONE (or hold any you want to re-read).
4. Merge `overnight-20260712` → `dev` `--no-ff` (docs-only, footprint clean).
5. Record triage resolutions in `ledger.md`; then the next slate = **Activity 4** (needs localhost
   Postgres + E2E armed first — see the slate's precondition flag).

## Precondition for the NEXT slate (Activity 4)

Activity 4 writes app code + runs red-first proofs against a live DB. **Arm localhost Postgres
(`brew postgresql@16`) + the E2E suite before it** — the standing HANDOFF DB flag bites there,
not here. This run deliberately touched no DB/E2E.
