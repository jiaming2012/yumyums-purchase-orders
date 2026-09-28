# HANDOFF — overnight run 20260905

Slate: `.night-crew/knowledge/reference/slate-20260905.md` (signed 2026-09-04).
Branch: `overnight-20260905` off `dev` @ `0670798`. Dispatch: concurrent 2-track as
signed (Wave 0 → Track B→C ∥ Track D). Conflict log:
`.night-crew/knowledge/reference/conflicts-20260905.md` (6 merge entries + the
no-Merge-7 note). Decisions journal: `.night-crew/knowledge/decisions/20260905.jsonl`
(1 record). **6 of 7 cards merged; Card 6 complete-but-PARKED on an operator fork —
see DECISIONS-NEEDED.md D-1.**

## Per-card outcomes

| # | Card | Outcome | Gates (final tree) | G6 |
|---|------|---------|--------------------|-----|
| 1 | marketing-tile-and-page | ✅ merged `6d7a8f5` | RF 3 Go + 6 PW reds→greens; Go suite (sole red = base-proven TestJWTBridgeRLS); full e2e ×2 831/6/6 (all 6 pre-existing/flaky, base-proven); G4 precache 31→32 | PASS-WITH-NOTES |
| 2 | rxdb-pull-replica | ✅ merged `6bd7e13` | harness EXIT=0 (impl + G6 re-runs); GAP-1 id-tiebreak validated + ledger line; full e2e 828/9/6 (4 known + 5 flakes proven by targeted re-run); G4 at 32 | PASS-WITH-NOTES |
| 3 | scan-attempts-push-conflict | ✅ merged `9b9c669` | harness EXIT=0 ×2 incl. psql server-side enumeration; GAP-1 two-belt fix red→green + ledger line; 403 write-only held | PASS-WITH-NOTES |
| 4 | clock-offset-on-sync | ✅ merged `9a77f90` | harness EXIT=0 ×2, both skew signs red→green; sibling regressions c2+c3 EXIT=0; E-KR4 held | PASS-WITH-NOTES |
| 5 | camera-scanner-decode | ✅ merged `e20972c` | RF 14 reds→greens; full e2e 850/1/6 (lone red = base DBL-05); G4 precache 32→39; vendored html5-qrcode **registry-verified** pristine 2.3.8 | PASS-WITH-NOTES |
| 7 | gstate-arbitration-machine | ✅ merged `5fefd56` | RF ×3 (E-KR2/F4/no-TOCTOU) red→green; full Go suite on go 1.26.2 (counts checked, sole red base-proven); parity+marketing specs 20/0 after fix round | **FAIL → fix `98e189e` → re-verify PASS** |
| 6 | redemption-submit-flow | 🅿️ **PARKED, complete** — branch `wo-redemption-submit-flow` + worktree `wt-20260905-c6` preserved | conformance 18/18; strictness 9/9; fuzz 40k walks, liveness armed (can-fail proven); full e2e ×2 856/5/6, 859/2/6 zero unexplained; G4 precache 43 | PASS-WITH-NOTES, merge gated on the D-1 ruling |

Velocity: 6/7 landed. Zero regressions known at closeout. First-pass (no rework leg
among landed): 4/6 — Card 1 had one in-run B-140 fix, Card 7 one G6-driven fix round.

## What landed, functionally

Marketing tile + permission-gated page shell with the `offline_override` entitlement
surface (#12 as resolved); two checkpointed pull replicas with the GAP-1 keyset
tie-break; the device-owned push with race-losing UI flip and the two-belt GAP-1 fix;
tamper-safe offline expiry off the PostgREST Date header; the full camera→decode→
hash→resolve scanner (F3/F5) at precache 39; and the §18 server arbitration machine on
gstate v0.3.1 with the Go 1.25.5→1.26.2 toolchain bump, `POST /api/v1/marketing/redeem`,
and the F4 read-model (HQ migration 0077). Parked but built: the strict XState submit
flow (confirm-then-burn, three offline branches, P-KR4 auto-resume, F6) — one operator
yes away from merging.

## Next actions (operator, morning)

1. **`/nc-morning-triage`** — review the run branch, merge to dev.
2. **D-1 ruling (DECISIONS-NEEDED.md)** — ratify or override unknown→false for
   `requires_online`; on ratify, merge `wo-redemption-submit-flow` (+ conflict-log
   entry) and name the follow-up card (replicate the flag + fix F-2's landing path).
3. **ATTENDED: live-camera check (Card 5)** — headless proved decode-from-image, hash
   chain, and the camera-denied path only. On a real phone over HTTPS: HQ → Marketing
   → Scan → Start camera → grant → point at a printed `tests/fixtures/qr-fixture-1.png`
   (token card1-test-code-fixture-1) → confirm live decode → result card; confirm
   camera pause on decode + resume via Scan next.
4. **Backlog candidates from tonight's G6s** (details in the conflict log entries):
   - DBL-05 deterministically red on dev base in single-test isolation (real bug or
     time-dependent test defect).
   - Replica **commit-order skew**: `updated_at` = txn-START time; a long transaction
     can commit rows behind an advanced checkpoint — permanently invisible to devices
     (RESYNC never rewinds). Candidate: checkpoint rewind on RESYNC / commit-ordered cursor.
   - **Belt-2 cross-session double-serve** (demonstrated): after local-store loss, a
     device re-scanning its own redeemed code gets a second `accepted`. Candidate:
     accept only when `redeemed_at >= scanned_at` (skew slack).
   - `race_lost_notifications` has no dedupe (replay can duplicate manager pings);
     F4's scan_attempts-status acceptance bullet is owned by NO card.
   - `enqueueAttempt` dedupe not atomic (Card 6 serializes; raw callers race).
   - Record the two vendored libs' sha256 in the spike/extraction records
     (html5-qrcode 660b1243…, xstate e7f04e1f…38fa28 — both registry-verified tonight).
   - F6 was partial after Card 5; full semantics live in parked Card 6.
   - Out-of-window replica rows never `_deleted` locally (unbounded growth, no owner).
   - `push-run.sh` header documents its red-mode exit contract inverted (comment fix).
   - §16:436 says `offline_override` = manager grant — annotate with the shipped
     fork-#12 resolution (entitlement, admin-seeded) before someone "fixes" the seed.
5. **Toolchain notes:** prod image builder now `golang:1.26-alpine` (first deploy
   pulls it). The MAIN checkout's node_modules carries stale workbox 7.3.0 — never
   regen sw.js there (lockfile-true is 7.4.1; card worktrees npm-ci'd); consider
   `npm ci` in the main checkout at triage.

## Run integrity notes (disclosed)

- **Rate-limit stall**: session limit cut both track implementers mid-card
  (~mid-afternoon); resumed on reset with no lost or repeated work; a second window
  was crossed during Card 6 harmlessly.
- **Control-loop error**: Card 2's G6 independent-harness log was lost at worktree
  removal before copy-out (disclosed in conflict-log Merge 3 entry); the G6 report
  text is the surviving record. Cards 1/3/4/5/6/7 G6 evidence logs are committed.
- **Decisions journal hygiene**: two malformed entries (an empty shape-probe and an
  invalid-severity vote) were removed before the clean record was written; the final
  journal holds exactly one record, the D-1 park.
- **G4 discipline greps**: `N/A-VACUOUS — neither package exists in this repo (B-14)`.
- **Workers check at closeout** (2026-09-05T03:00:50Z): `night-crew workers check`
  exit 0 — no pollers on `night-crew` / `night-crew-env`. (Poller-TTL caveat: this
  reports the check's own moment.)
- **`night-crew.toml`**: Card 6's branch carries the new `marketing` seam entry —
  it lands only if D-1 ratifies; until then marketing-path changes still de-confine
  to the full suite.
- The operator attended briefly mid-run (Slack Stop-hook change on this machine —
  unrelated to the run's file surfaces).

## Triage disposition (2026-09-05, attended — ledger T-53)

- **Merged to dev:** run branch at `ba5efba`; **card 6 at `f2f832a`** (D-1 ratified —
  unknown→false stands as shipped; follow-up card `requires-online-replication` named,
  required before any real campaign). Independent adversarial re-execution reproduced
  every executable gate claim; merged-tree Go suite re-run green vs the armed baseline
  (sole red = base-proven TestJWTBridgeRLS); card 6 conformance 18/18 + strictness
  re-proven at triage. Pushed to origin/dev.
- **Standing flags:**
  - **ATTENDED live-camera check (Card 5) — STILL ARMED.** Headless proved decode-from-image
    only. Re-arms whenever `marketing/scanner.js` / `marketing/scan-page.js` / the vendored
    lib change. Steps in "Next actions" #3 above.
  - **No real campaign may be provisioned** until `requires-online-replication` lands
    (D-1 rider; close-bar leg 3 / Q-KR1 unattestable until then).
  - Playwright armed reds B-174 ×3 + B-176 ×1 + B-178 unchanged (no card owed them);
    DBL-05 now filed as **B-421**.
- **Toolchain note amended (adversarial Finding 2):** plain `npm ci` dies mid-reify in a
  fresh clone on this box (`Exit handler never called!`, `.bin` never linked) — the working
  sequence is **`npm ci && npm rebuild`**. Applied to the main checkout at triage (stale
  workbox 7.3.0 trap cleared).
- **Backlog graduated:** B-421..B-431 (see BACKLOG.md — includes the belt-2 double-serve,
  commit-order skew, the push-harness HTTP-failure injection gap the triage mutation probe
  M3 exposed, and the F-4 order-number seam bypass).
