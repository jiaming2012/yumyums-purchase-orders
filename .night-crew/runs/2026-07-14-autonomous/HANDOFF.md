# HANDOFF — overnight-20260714 (for the morning of 2026-07-14)

> **Run branch:** `overnight-20260714` (cut from `dev`; never pushed, `main` untouched).
> **Slate:** `.night-crew/knowledge/reference/slate-20260714.md` (batch-signed 2026-07-13).
> **Scope:** Activity 4 — the **first APP-CODE + red-first + E2E slate**. Wave-0 infra
> standardization + the 4 confirmed-BROKEN fix-cards, **serial**, riskiest (Onboarding) last.
> Then both budget-gated **stretch** cards (all 5 committed cards landed clean with budget in hand).
> **Result:** **7/7 cards complete, every card G6-PASS, 0 parked, 0 REVISE, 0 Docker crashes.**
> 7 atomic commits. Wall-clock ≈ **2h26m** of the 6h budget. Reader = the operator; resolve via
> `/nc-morning-triage`.

## TL;DR (what changed)

- **4 confirmed-BROKEN flows fixed + guarded by a red-first regression test** (Eng KR-1: 4 → 0):
  Ops **FR-4** (yes/no "No" corrective gate), Ops **NFR-3** (photo-required gate), Purchasing
  **FR-18** (History tab built), Onboarding **NFR-5** (video-led reopen/reject). Each: test written
  FIRST, captured FAILING against the unfixed build, then fixed, then green — **no test passed
  without its fix.**
- **Wave-0 infra standardized:** local Docker DB → `postgres:16` (matches prod + the ephemeral env);
  `task test:all`/`bdd` repointed off the remote Windows box to local Docker. All 70 migrations apply
  cleanly on pg16.
- **2 stretch cards (QA KR-1):** Purchasing **FR-7** proved WORKING (real render assertions replace a
  vacuous tautology); Users **stale-E2E** repaired (2 dead Access-tab tests repointed `#t3/#s3` →
  `#t2/#s2`; `users.spec.js` 17/2 → **19/0**).
- **1 follow-up surfaced (NOT a fork — no decision blocks triage):** Ops NFR-3's **backend** resubmit
  `require_photo` gate was deferred (the submit input struct carries no rejection context); the
  frontend covers it, but a direct-API resubmit can still bypass. Logged to `BACKLOG.md`. See
  `DECISIONS-NEEDED.md`.

## Per-card outcome table

| # | Card | Verdict | Commit | Red→green evidence (on the ephemeral pg16 stack) |
|---|---|---|---|---|
| 0 | `hq-infra-docker-standardize` | **G6 PASS** | `1817474` | Acceptance: `db-start` came up **pg 16.14**, `db-test` created `hq_test`, smoke `go test ./internal/recipes/` green, 70 goose migrations clean on pg16, build/vet clean. Footprint = 2 Taskfiles. |
| 1 | `ops-fr4-no-enforcement` | **G6 PASS** | `2287947` | RED: "No" + empty note → submit **succeeded** (no severity toast). GREEN after front+back gate (both submit entrypoints). Paired positive test proves no over-blocking. |
| 2 | `ops-nfr3-photo-required` | **G6 PASS** | `ad105f7` | RED: required-photo submit with no photo → submission **created**. GREEN: `photo_required` 400 front+back. Backend resubmit-case DEFERRED (follow-up). |
| 3 | `purchasing-fr18-history` | **G6 PASS** | `4cb57b7` | RED: History assertions time out on the stub (`#history-content`/`.history-card` absent). GREEN: `renderHistory` wired to `GET /shopping/history`; 5 History tests pass. Fixture via existing API (no SQL/migration). |
| 4 | `onboarding-nfr5-video-reopen` | **G6 PASS** | `5d73b96` | RED: reopen **and** reject of the seed's video-led Equipment Training § left it `complete`/`signed_off`. GREEN: `ReopenSection` resolves the first video part; covers FR-9 + FR-15. |
| 5 | `purchasing-fr7-retest` (stretch) | **G6 PASS** | `958a176` | Proved WORKING: real empty (exact stub copy) + populated (vendor sections/checks/thumbs/locations) assertions replace the tautology. **Note:** old test was baseline-RED (targeted a nonexistent `#shopping-content`), not merely vacuous. |
| 6 | `users-stale-e2e-repair` (stretch) | **G6 PASS** | `d32830d` | RED: 2 Access-tab tests timed out on dead `#t3`. GREEN after `#t3/#s3`→`#t2/#s2` repoint + behavior-neutral `renderAccess` var rename. `users.spec.js` 17/2 → **19/0**. |

**Every red→green claim was independently re-verified by a separate fresh G6 subagent** against the
diff + evidence only (not the implementer's reasoning). G6 re-confirmed each new test genuinely went
red pre-fix and each fix stayed in its declared footprint.

## Gate results on the final merged tree

- **Integration build/vet (orchestrator, all 7 merged):** `go build ./...` + `go vet ./...` **clean** —
  the two Ops cards' serial `handler.go` edits and the onboarding `db.go` edit integrate coherently.
- **`task sw`** regenerated once per HTML-touching merge (fr4, nfr3, fr18, users-stale) — `sw.js` on the
  branch reflects the final `workflows.html`/`purchasing.html`/`users.html`. Frontend semver unchanged
  (1.0.2 — bump belongs to `/save-project` at deploy, not overnight).
- **No new reds introduced:** every card's affected-seam subset was diffed against its own baseline on
  the ephemeral stack; all flips outside a card's new test were confirmed pre-existing/flaky in isolation
  (offline-sync, tab-persistence, cross-test DB-pollution — HQ's known ~37–41-red pool).
- **Footprint:** every card touched only its declared files (verified per merge). Zero scope breaches.

## Cycle status-tally movement (the KR denominators)

| App | Before (post-overnight-20260712) | After this run | Δ |
|---|---|---|---|
| Operations | 10 W / 15 U / **2 B** | 10 W / 15 U / **0 B** (2 fixed) | FR-4, NFR-3 BROKEN → fixed+guarded |
| Onboarding | 23 W / 10 U / **1 B** | 23 W / 10 U / **0 B** (1 fixed) | NFR-5 BROKEN → fixed+guarded (FR-9 + FR-15) |
| Purchasing | 6 W / 19 U / **1 B** | ~7 W / ~18 U / **0 B** | FR-18 BROKEN → built+guarded; FR-7 U→W (proved) |
| Users | 10 W / 16 U / 0 B | ~11 W / ~15 U / 0 B | 2 stale-tests repaired → Access flow proven |
| Inventory | 19 W / 19 U / 2 B | unchanged | FR-24/25 waived (D-3); not in this slate |

- **Eng KR-1 (0 known-broken built flows):** the confirmed-BROKEN denominator moves **4 → 0** this run
  (Ops FR-4, Ops NFR-3, Onboarding NFR-5, Purchasing FR-18 — all fixed/built + red-first guarded).
  Remaining waived-unbuilt: Inventory FR-24/25 (D-3), Onboarding FR-16/NFR-4 (D-5). **Subject to the
  operator confirming DRAFTING → DONE at triage.**
- **QA KR-1 (vacuous/stale tests → 0):** 1 vacuous WORKING test hardened (Purchasing FR-7) + 2 stale
  UNPROVEN tests repaired (Users Access) — 3 tests moved from non-asserting to genuine guards.

## Roadmap state

All 7 rows flipped **PLANNED → DRAFTING** in `roadmap.md` (Wave-0 + the 4 fix-cards + the 2 stretch
cards). Per the slate, **DRAFTING → DONE happens at morning-triage sign-off, not overnight** — the
operator flips them after reviewing this HANDOFF.

## Decisions needed (operator, at triage)

**No open forks block triage.** One follow-up is surfaced for awareness (see `DECISIONS-NEEDED.md`):

- **Ops NFR-3 backend resubmit `require_photo` gate (deferred, follow-up — not a fork):** the
  field-level required-photo gate is enforced front+back; the *rejection-driven* resubmit photo
  requirement is frontend-only because `SubmitChecklistInput` carries no `submission_id`/rejection
  context. A direct-API resubmit could bypass it. Logged to `BACKLOG.md` as a scoped follow-up
  (needs a `submission_rejections` join). The run made this scope call inside the card's explicit
  allowance — no decision required, but flagging it so it's on the record.

## Suggested triage order

1. Spot-check the **4 BROKEN→fixed** cards' red→green evidence (Ops FR-4/NFR-3, Onboarding NFR-5,
   Purchasing FR-18) — the load-bearing Eng-KR-1 movement. The commits carry the RED/GREEN detail;
   G6 already re-verified each at the diff.
2. Note the **Ops NFR-3 deferred backend resubmit gate** (above) — accept as a `BACKLOG` follow-up or
   schedule it as a small fix-card.
3. Sign off the **7 DRAFTING rows → DONE** (or hold any you want to re-read).
4. Merge `overnight-20260714` → `dev` `--no-ff`, then `dev` is ready for a normal `task prod:deploy`
   (this run left the frontend semver at 1.0.2 — `/save-project` bumps it at deploy time).
5. Record triage resolutions in `ledger.md`; the next slate = the **~79-flow test-only prove-UNPROVEN
   bulk** (per-app, per-tab for Inventory) — the remaining Activity-4 volume.

## Notes for the next slate (sizing signal)

- The first app-code actuals are in `reference/card-actuals.md` (§ overnight-20260714): app-fix cards
  ≈ **15–27 min implement** wall-clock (2 Docker builds dominate the floor), G6 ≈ 1–3 min, ~2min/card
  orchestrator merge. Serial, one ephemeral env at a time held with **zero Docker crashes**.
- **Fixture availability is the biggest time lever** — cards whose fixture pre-exists in the seed
  (onboarding-nfr5) are cheapest; cards that must author a fixture (purchasing-fr18) cost more than
  their logic implies. Check seed coverage when card-splitting the prove-UNPROVEN bulk.
- The ephemeral Docker env (`docker-compose.nc.yml`, pg16) is validated faithful and is now the
  canonical local DB path (Wave-0 made pg16 the standard). No host/brew Postgres was used or needed.
