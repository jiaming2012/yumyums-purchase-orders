# Cycle closeout — HQ hardening cycle (Activity 5, `cycle-gate`)

> **Run branch:** `overnight-20260716` (cut from `dev`; never pushed, `main` untouched).
> **Slate:** `.night-crew/knowledge/reference/slate-20260716.md` (batch-signed 2026-07-15).
> **Scope:** Activity 5 — the OKR cycle gate, fanned into 3 read-only closeout cards
> (suite baseline · attestation · scorecard). **Serial dispatch.**
> **Gate posture (operator 2026-07-15, "Attest & waive"):** 3 criteria attested, 2 formally
> waived against documented reality and carried to the next cycle's roadmap.
> **Milestone boundary:** on a clean gate the next move is `/nc-okr-session` (open the next
> cycle, consume the carried-forward backlog) — not `/nc-slate-plan`. `/nc-morning-triage` first.

This document is the cycle-gate evidence of record. It is assembled by the orchestrator in
card-landing order: Card 1 wrote the suite-baseline evidence block, Card 2 the attestation,
Card 3 the metric + scorecard. Each subagent returned its section as a report; the orchestrator
wrote the doc. **Gate verdict: §4.**

---

## §1 — Suite baseline evidence (Card 1: `cycle-gate-suite-baseline`)

**Verdict: ATTEST PASS (0 uncategorized reds · PARK trigger did not fire).**

### Run mechanics (methodology — read before using the numbers)
Full E2E suite run **once** on an isolated ephemeral stack, read-only, torn down after:
- **DB isolation:** a standalone `postgres:16` on host port **5455**. `localhost:5432` is held by an
  unrelated `infra-postgres-1` (postgres:13) container and was **not touched** — HQ's real DB was
  never in the path (footprint rule honored). This deviates from `task test:all`'s hardcoded
  `localhost:5432` only in the port; semantics are identical.
- **Replicated `task test:all` ordering:** Go units first (against the isolated pg16), then the full
  Playwright suite via Playwright's own self-spawned webServer (`go run ./cmd/server`, fresh app boot
  → superadmins self-seed from `config/superadmins.yaml` after the Go phase, exactly as `test:all`
  sequences it). Go toolchain: `/usr/local/go/bin/go` (go1.25.8).
- **Documented-baseline shell:** no `ANTHROPIC_API_KEY` / no DO-Spaces creds in the environment
  (matches the ledger's documented baseline, T-8/T-11) — so external-AI and S3 paths behave exactly
  as the recorded baseline expects, without live external calls.
- **Seed scope (honest caveat):** only the app's self-seeded **superadmins** were present. The
  `.night-crew/qa/test-users/` personas (only `example.toml.sample` exists on disk) and per-card
  domain fixtures (pending purchases, shopping lists, onboarding assignments) were **not** seeded.
  This is a faithful reproduction of a **bare `task test:all`** run — the run shape that carries HQ's
  documented **~37–41 flaky/pre-existing reds** (ledger T-12 standing note). Cards in prior runs
  proved individual flows GREEN on **per-card** seeded stacks; the bare full-suite run does not carry
  those fixtures, and the reds concentrate exactly there.

### Counts (authoritative — Playwright JSON reporter + `go test`)

| Suite | passed | failed (unexpected) | flaky | skipped | Exit |
|---|---|---|---|---|---|
| **Playwright** (chromium + bdd) | **387** | **38** | **0** | **6** | 1 |
| **Go units** (`go test -p 1 ./...`) | 5 pkgs `ok` | **1 pkg FAIL** | — | — | 1 |

**`0 flaky`** is load-bearing: with `retries:1`, every one of the 38 failed on **both** attempts →
these are **deterministic-within-run**, not transient flapping. Determinism is consistent with
structural (SW-blocked) and fixture-absent causes, and lets each red be categorized rather than
dismissed as noise.

### Go-unit red — categorized: env-gated (documented)
Exactly one package fails: `internal/receipt › TestRunIngestCycle_ScenarioTable` (subtests
`both_fail_with_realistic_errors`, `both_fail_decimal_qty`) — the Claude-Haiku receipt-parse
AI-matching path with **no `ANTHROPIC_API_KEY`**. This is the identical env-gated red the ledger
records at T-8 and T-11. The other 5 packages (`auth`, `inventory`, `purchasing`, `recipes`,
`toast`) are `ok`.

### Playwright reds — all 38 categorized (0 uncategorized)
A 7-test **fix-adjacent isolation re-run** (fresh pg16 → fresh app, `--retries=0`) verified the
categorization split rather than asserting it: **1 greened in isolation** (proving cross-test
DB-pollution), **6 reproduced** their failure on a clean single-test DB (proving structural SW-block
/ seed-absence, not run-order pollution). Every failure maps to a documented cause:

| # | Category | Count | Mechanism / evidence | Representative failing tests |
|---|---|---|---|---|
| 1 | **SW-blocked / offline-sync** (structural) | 2 | `playwright.config.js:29` `serviceWorkers:'block'` → offline queue/IndexedDB + realtime-toast paths can't execute. **Isolation-confirmed:** `Queued` never renders. | `workflows › submit while offline queues in IndexedDB`; `sync › no "updated by" toast for own field saves` |
| 2 | **Reload / tab-persistence** | 5 | Reload-persistence depends on SW/IndexedDB-backed client state or seeded content the bare run lacks. **Isolation-confirmed:** shopping content never loads on reload. | onboarding `checkbox`/`video part` reload; purchasing `shopping check-off`/`store location` reload; `workflows › users tab persists on reload` |
| 3 | **Cross-test DB-pollution** | 2 | Shared `hq_test` across the whole suite. **Isolation-confirmed GREEN:** `approved checklist…` passes alone, fails in-suite. | `workflows › approved checklist shows Approved badge…`; `workflows › sub-step completion attributes parent checkbox…` |
| 4 | **Data-dependent fixtures — Inventory** (documented ~40-guard pool) | 12 | Need seeded pending-purchase/receipt/review fixtures a bare run doesn't create → review form (`.review-tax`) never appears. **Isolation-confirmed** timeout. BACKLOG names FR-2 tax/grand-total (`inventory.spec.js:1039-1042,1058-1061`). | `create new item via Items tab`; `editing tax updates grand total`; mismatch-banner ×2; price-field ×2; `backend rejects confirm…`; `PO suggestions count…`; `cutoff pill admin-interactive`; `clears parse_error…`; `Confirm Receipt disabled…`; receipt-carousel multi-attachment |
| 5 | **Data-dependent / persona guards — Onboarding** (documented 6-guard pool + unseeded manager/crew/hire personas) | 13 | Personas + assigned-training fixtures not seeded → `#mgr-body` missing hire, `reject-section` not visible, self-created template not in My List. **Isolation-confirmed** on 2. | `manager sees hire with assigned training`; `crew can unsubmit…`; `manager can reject signed-off/before sign-off`; sign-off role picker ×3; save-video-for-later ×2; sub-item completion ×2; `video part play button`; `unsubmitted section proof-photo` |
| 6 | **Data-dependent — Purchasing** (shopping-list / PO / store_location seed) | 4 | Need a seeded shopping list / locked PO / store_location catalog. | `PO tab shows stub or locked PO`; `item without store_location…`; `items grouped by store_location…`; `item info navigates to Inventory Setup` |
| | **Total** | **38** | | **0 uncategorized** |

**6 skipped = expected-skip (not reds):** 2 are the documented `-0715` PARKs carried as
`test.skip`+reason — `FR-27 photo upload` (needs live S3) and `FR-18 custom-thumbnail` (needs S3
PUT); 4 are conditional expected-skips (`recipe usage_pct round-trip`, `vendor section completion
persists`, `toast when checking item without photo`, `PO approve-button-for-admin-when-locked`).

### The PARK check (the one real risk) — cleared
The PARK trigger is *"a red that maps to no documented category — a genuinely new broken product
flow."* Two independent checks clear it:
1. **All 38 map** to a documented category (table above); the count (38) sits inside the documented
   ~37–41 baseline; the Go red is the known env-gated one.
2. **No repaired flow regressed.** Every this-cycle fix's own tests are **GREEN**: ops FR-4
   (`answering No without a corrective note blocks submit`; `server rejects…`), ops NFR-3 (`submit
   blocked when a required photo is not attached`), purchasing FR-18 (`History tab renders…`;
   `history tab shows seeded completed shopping list`), inventory NFR-1 (`NFR-1: create-vendor and
   create-item title-case…`; `FR-26`/`FR-29` title-case). None appears in the 38.

**→ 0 uncategorized reds. No PARK.** `DECISIONS-NEEDED.md §A` stays empty. The gate's suite criterion
attests as **"0 new uncategorized reds vs the documented ~37–41 flaky baseline"** (criterion 2 is
formally waived from its literal "0 pre-existing reds / clean suite" reading — see §2).

---

## §2 — Attestation (Card 2: `cycle-gate-attestation`)

Read-only audit across the 5 hardening PRDs + `git log` + `BACKLOG.md`/`ledger.md`. Every citation
opened at the line/commit.

### (1) 0-known-broken-flows attestation

| Flow (ever marked BROKEN) | Disposition | Evidence |
|---|---|---|
| **Ops FR-4** — yes/no "No" corrective-action enforcement | **GREEN-fixed** | `2287947` `fix(ops): enforce corrective note on yes/no "No"…` — `workflow/handler.go` + `workflows.html`, front+back |
| **Ops NFR-3** — photo-required-at-submit | **GREEN-fixed** (field-level) | `ad105f7` `fix(ops): block submit until required photo attached` — `handler.go` + `workflows.html`. Residual **F-1** (rejection-driven *resubmit* gate) is frontend-only → backlogged, not a BROKEN flow |
| **Onboarding NFR-5** — reopen/reject of video-led section = silent no-op | **GREEN-fixed** | `5d73b96` `fix(onboarding): reopen/reject… reverts to active` — `onboarding/db.go`; covers FR-9 + FR-15 |
| **Purchasing FR-18** — History tab static stub | **GREEN-fixed** | `4cb57b7` `feat(purchasing): build History tab (renderHistory + /shopping/history)` — `purchasing.html` |
| **Inventory FR-24 (Trends)** / **FR-25 (Cost)** | **WAIVED (D-3, 2026-07-10)** — unbuilt-future | `.coming-soon` stubs confirmed present: `inventory.html:994` `renderTrends()` / `:998` `renderCost()`, no API/state |
| **Onboarding FR-16** (video presign→PUT→FFmpeg) / **NFR-4** (`503 video_storage_not_configured`) | **WAIVED (D-5, 2026-07-13)** — env-gated (marked UNPROVEN, not BROKEN) | Fully implemented, confirmed: `onboarding/handler.go:540/549/601/604`, `video.go` (206 L). Needs DO Spaces creds + ffmpeg |

**Attestation statement:** **Known-broken *built* product flows this cycle: 4 → 0.** The four
confirmed-broken built flows are each GREEN-fixed with a verified landing commit and the fix code
present in the tree; §1's suite run independently confirms all four repaired flows' tests are GREEN
and none regressed. The only items excluded from the "0 known-broken" denominator are the **4
operator waivers** (D-3 Inventory FR-24/25; D-5 Onboarding FR-16/NFR-4). No built flow remains
BROKEN. (A 5th repair — Inventory **NFR-1** normalization — landed GREEN this cycle from a graduated
UNPROVEN→RED, not from the original BROKEN set; see below.)

### (2) Red-first proofs

| Card | Fix commit | Red-first evidence | Git-reconstructable? |
|---|---|---|---|
| `ops-fr4-no-enforcement` | `2287947` | test + fix bundled; RED "submit-succeeded" captured (`card-actuals.md:142`) | No — squash bundles test+fix |
| `ops-nfr3-photo-required` | `ad105f7` | RED "submission-created" captured (`card-actuals.md:142`) | No — squash |
| `purchasing-fr18-history` | `4cb57b7` | RED captured (`card-actuals.md` Activity-4 row; backlog "red-first") | No — squash |
| `onboarding-nfr5-video-reopen` | `5d73b96` | RED "stayed-complete" captured (`card-actuals.md:142`) | No — squash |
| `inventory-nfr1-normalize-fix` | `77957c1` (fix) | **RED test `1a0265e` committed BEFORE fix** — `git merge-base --is-ancestor` confirms ordering; fix adds `normalizeItemName` at `handler.go:660`+`:1130` | **Yes — the one git-verifiable red→green pair** |

**Honest caveat (carried, not hidden):** the night-crew merge protocol squashes each app-fix card's
worktree into a **single** landing commit, so for the four Activity-4 fix cards the new spec test and
the fix arrive together — git cannot show a standalone failing-before commit. Their red-first
ordering is **documented** (`card-actuals.md:142-145`, each naming the observable break; commit
messages assert "red-first") but **not git-reconstructable**. **Only Inventory NFR-1** is an
independently git-verifiable RED→GREEN pair (`1a0265e` precedes `77957c1`). QA-KR-3 is attested on
the ledger record for four of five and on git for the fifth.

### (3) Formal waiver record — gate criteria 2 & 3

- **Criterion "full E2E suite green, 0 pre-existing reds" → WAIVED** to *"0 new uncategorized reds vs
  the documented ~37–41 flaky baseline"* (met — §1). Basis: the ledger's standing no-new-reds
  convention (`card-actuals.md:146-148`; `ledger.md` T-11/T-12); structural blocker
  `playwright.config.js:29` (`serviceWorkers:'block'`) makes a literal green suite unreachable.
- **Criterion "vacuous tests 23 → 0" → WAIVED** to *"the ~4–5 rewritten this cycle; the ~18 remainder
  are deferred test-hardening WOs still `new` in `BACKLOG.md`."* Deferred remainder: Ops FR-10/12
  vacuous reject test (`workflows.spec.js:485-508`), Ops FR-15 builder-UI gap, Onboarding 6
  conditional-skip guards, Inventory ~40 data-dependent guards.

**Carried-forward backlog (to the next cycle's roadmap, all confirmed present in `BACKLOG.md`):**
1. The **~37–41 flaky/data-dependent + SW-blocked pool** ("Stabilize the suite" — declined 2026-07-15).
2. The **~18 vacuous remainder** (the 4 test-hardening notes above).
3. The **3 harness WOs**: `WO-cron-clock-seam`, `WO-photo-s3-harness`, `WO-offline-indexeddb-harness`.
4. **F-1** — Ops NFR-3 backend resubmit `require_photo` gate.
5. **Onboarding video-pipeline E2E fixture** (D-5 preserved prove-path for FR-16/NFR-4).
6. **F-2** — `users.html:122` orphaned `<div id="s3">` (confirmed present).

---

## §3 — OKR scorecard + Delivery metric (Card 3: `cycle-gate-scorecard`)

### Delivery metric — median WO cycle time (baseline; no pass/fail target this cycle)

**Basis:** per-card implementer/agent wall-clock for the app-code + prove/test **delivery** WOs in
the two real delivery runs (`overnight-20260714` serial, `overnight-20260715` concurrent). Excludes
the Wave-0 infra card and the smaller Activity-1/2/3 doc-audit cards (reported separately in
`card-actuals.md`). Data points auditable there.

- **N = 23 WOs.**
- **Combined median = 22m28s** (12th of 23 sorted; `purchasing-prove-order`). *(Independently
  re-verified by the orchestrator: 12th-of-23 sorted = 22.47 min.)*
- **Per-run spread:** `-0714` (serial, N=6): 6m28s–27m12s, **median ≈ 19m25s**. `-0715` (concurrent
  rolling-3, N=17): 8m26s–32m44s, **median 23m24s**.
- **Honest read:** `-0715` ran rolling-3 concurrent, inflating **per-card latency ~1.5–2×** (shared
  CPU/IO), and its 17 cards dominate N — so the combined 22.5-min median leans toward the loaded
  figure. Size future **serial** slates at **~19 min/card**; size **concurrent** slates on
  total-wall = (Σ impl)/3 + G6/merge tail, per the ledger's guidance. Baseline established.

### OKR scorecard (all 9 KRs)

| KR | Verdict | Evidence | Honest gap |
|---|---|---|---|
| **Product KR-1** — 5/5 apps have a hardening PRD | **PASS** | Roadmap Activity 1 all DONE; ledger T-1 | None |
| **Product KR-2** — enumeration recall ≥90% | **PASS (two-pass)** | Single-pass 73–91% (ledger 2026-07-10), but the mandatory cross-check caught every miss; prove sweep discovered **no new shippable flows** (T-11) → post-two-pass recall ≥90% | Load-bearing on the cross-check, not first-pass foresight; misses hid in backend-only surfaces |
| **Product KR-3** — ≥90% WOs trace to a PRD flow | **PASS** | Every WO card carries a KR/flow trace; only the Wave-0 infra WO is non-flow (≥22/23 → >95%) | The one infra WO is non-flow by nature |
| **Delivery KR-1** — 100% broken/unverified flows have a shipped WO | **PASS** | 4 BROKEN → fix-WOs RED→GREEN (T-9); ~52 UNPROVEN → prove-WOs (T-11); D-3/D-5 waived by sign-off | 11 PARK + 1 UNTESTABLE remain, each a committed `test.skip`+reason; 3 future fix-WOs queued |
| **Delivery KR-2** — median WO cycle time over ≥5 WOs | **PASS** | Recorded: N=23, median 22m28s (above) | Concurrency caveat; read as serial ~19.4m / concurrent ~23.4m |
| **Engineering KR-1** — 0 known-broken flows | **PASS** | 4→0 (T-9) + the 1 graduated NFR-1 RED fixed same-night (T-12); §1 confirms no repaired flow regressed | Excludes the 4 explicit D-3/D-5 waivers |
| **Engineering KR-2** — full E2E suite green, 0 pre-existing reds | **MISS → WAIVED** | The pre-declared waiver: ~37–41 documented flaky/data-dependent reds + SW-blocked offline (§1; `playwright.config.js:29`). Attest-substitute **"0 new uncategorized reds vs baseline" met** (§1: 38 reds, all categorized, 0 new) | Literal clean-suite bar unmet by construction; waived-and-carried |
| **QA KR-1** — vacuous tests 23 → 0 | **PARTIAL** | ~4–5 rewritten this cycle (Purchasing FR-7 tautology→genuine T-9; +2 vacuous rewrites; +2 stale repaired) | ~18 remainder deferred to `BACKLOG.md` (Ops FR-10/12, Ops FR-15, Onboarding 6 guards, Inventory ~40 guards) — clear PARTIAL |
| **QA KR-2** — 100% critical flows have ≥1 real E2E test | **PARTIAL** | ~52 UNPROVEN flows given real red-first assertions (T-11); all 4 PRIORITY-risk flows proved WORKING | 11 PARK + 1 UNTESTABLE + 4 waived lack a driving test — each an honest `test.skip`+reason; unblocked by the 3 harness WOs |
| **QA KR-3** — 100% repaired/added tests carry a red-first proof | **PASS** | All fix/prove cards red-first (T-9/T-11/T-12); one git-verifiable pair (NFR-1), four on ledger record (see §2 caveat) | Red-first ordering git-reconstructable for 1 of 5 fix cards; documented for the rest |

**Cycle summary — 6 PASS · 2 PARTIAL · 1 WAIVED.** Closed cleanly: Product KR-1/2/3, Delivery
KR-1/2, Engineering KR-1, QA KR-3. The cycle's headline — **UNPROVEN ≈ untested, not broken (actual
RED = exactly 1 of ~78 forecast 34–40)** — is the strongest signal: the five apps were far sounder
than scoping feared; the work was *proving* them, not repairing them. PARTIAL-and-carried: QA KR-1
(vacuous 23→~18) and QA KR-2 (~52 flows driven, remainder honestly parked). WAIVED-and-carried:
Engineering KR-2 (clean-suite unattainable; honest substitute met). No inflation to all-green — the
residue is deliberately deferred to the planners via `BACKLOG.md`.

---

## §4 — Gate verdict

**CYCLE GATE: PASS (attested), under the operator's 2026-07-15 "Attest & waive" posture.**

| Gate criterion | Verdict | Basis |
|---|---|---|
| 0 known-broken flows | **ATTEST PASS** | 4→0 built flows GREEN-fixed (§2); +1 graduated NFR-1 fix; 4 waivers (D-3/D-5) excluded by sign-off; §1 confirms no repaired flow regressed |
| Full E2E suite green, 0 pre-existing reds | **WAIVE → attest "0 new uncategorized reds vs the documented ~37–41 baseline"** | §1: 387 pass · 38 fail · 0 flaky · 6 skip; all 38 categorized, 0 uncategorized, PARK did not fire; SW-block (`playwright.config.js:29`) makes clean-suite structurally unreachable |
| Vacuous tests 23 → 0 | **WAIVE → attest ~4–5 rewritten; ~18 deferred** | §2: remainder itemized in `BACKLOG.md` as test-hardening WOs |
| Every repaired flow red-first | **ATTEST PASS** | §2: all 5 red-first (1 git-verifiable pair, 4 on ledger record — caveat recorded) |
| Median WO cycle time ≥5 WOs (baseline) | **COMPUTED** | §3: N=23, median 22m28s (serial ~19.4m / concurrent ~23.4m) |

**Two criteria formally waived and carried to the next cycle's roadmap** (extends the D-3/D-5 waiver
precedent); three attested; the metric computed. **No card parked; `DECISIONS-NEEDED.md` empty.**

**Milestone boundary reached.** The morning move on this clean gate is `/nc-morning-triage` (review +
merge the run branch), then `/nc-okr-session` to open the next cycle and consume the carried-forward
backlog — **not** `/nc-slate-plan`. The carried-forward feedstock is enumerated in §2(3) and lives in
`BACKLOG.md`.
