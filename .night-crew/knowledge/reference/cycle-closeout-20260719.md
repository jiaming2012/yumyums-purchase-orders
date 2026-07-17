# Cycle closeout — "Nothing silently lost" (Activity 8, `cycle-gate`)

> **Run branch:** `overnight-20260719` (cut from `dev`; never pushed, `main` untouched).
> **Slate:** `.night-crew/knowledge/reference/slate-20260719.md` (batch-signed 2026-07-17).
> **Scope:** Activity 8 — the OKR cycle gate for the "Nothing silently lost" data-integrity
> cycle, fanned into 3 read-only closeout cards (suite baseline · attestation · scorecard) +
> orchestrator-assembled closeout. **Serial dispatch, one isolated pg16 env.**
> **Resolved fork (operator 2026-07-17, "Gate now, prod KRs pending"):** the read-only gate runs
> tonight and attests the dev-side deterministic stack; the 2 prod-dependent KRs are scored
> **PENDING** with exact verify commands; **Activity 7 (prod ops) runs attended AFTER** this gate
> as the milestone ship step (attest-green-before-ship).
> **Milestone boundary:** morning move is `/nc-morning-triage` (review + merge the run branch),
> then the attended **Activity 7** ship step to flip the 2 PENDING prod KRs, then `/nc-okr-session`
> to open the next cycle and consume the carried backlog.

This document is the cycle-gate evidence of record, assembled by the orchestrator in card-landing
order: Card 1 wrote the suite-baseline evidence (§1), Card 2 the attestation (§2), Card 3 the
metric + scorecard (§3). Each read-only subagent returned its section as a report; the orchestrator
ran the live suite (Card 1), reconciled Card 3's E5 against §1's actual result, and wrote the doc.
**Gate verdict: §4.**

**Headline — 11 PASS · 2 PARTIAL · 2 PENDING · 1 N/A (16 KRs).** The theme "every silent-loss mode
is enumerated, fixed, and made structurally impossible" is delivered on the dev-side evidence this
gate can see: stable field identity, loud rejection, transactional broadcast, and a convergence
matrix proven zero-flake under `--retries=0`. **Waiver #2 (vacuous 18→0) is formally retired**
(`3fd4d3f`). **Waiver #1 (`task test` exit-0) is substantially retired — the suite went 38 reds
(07-16 gate) → 1 — but NOT formally retired:** one documented cross-test-pollution red (isolation-
confirmed green) keeps literal `task test` from exiting 0, so waiver #1 carries forward, reduced.
The 2 prod KRs (parity, ghost-item) are honestly PENDING for the attended Activity-7 ship step.

---
## §1 — Suite baseline evidence (Card 1: `cycle-gate-suite-baseline`)

**Verdict: ATTEST — 0 uncategorized reds · PARK trigger did NOT fire. Suite went 38 reds (07-16 gate) → 1 red, and that red is isolation-confirmed cross-test DB-pollution, not a code defect.**

### Run mechanics (methodology — read before the numbers)

Deterministic stack run **once** on an isolated ephemeral pg16, read-only, torn down after. Go toolchain `/usr/local/go/bin/go` (go1.25.8).

- **DB isolation:** a standalone `postgres:16` container on a **Docker-assigned host port (`127.0.0.1:57606`)**, bound to loopback. Host `:5432` is held by an unrelated always-on Postgres and was **never touched** — HQ's real DB was never in the path (footprint rule honored). Migrated by a throwaway app boot (goose embedded migrations → **version 70, 48 tables**), exactly as `docker-compose.nc.yml`'s app service would.
- **Ordering:** Go units first (`go test -count=1 -p 1 ./...` against the isolated pg16), then the full Playwright suite via Playwright's **own** freshly-spawned webServer (`go run ./cmd/server`, `CI=1` so `reuseExistingServer:false` → fresh boot + auto-migrate + torn down after). `-count=1` forces a genuine run (no stale build cache).
- **Documented-baseline shell:** no `ANTHROPIC_API_KEY` / no DO-Spaces creds in the environment — matches the ledger's documented baseline. `MERCURY_API_KEY=` / `TOAST_SYNC_INTERVAL=0` / `E2E_DISABLE_SCHEDULERS=1` as the webServer config sets them.
- **No `task sw`:** read-only closeout, zero asset diff — Playwright driven directly via `npx`, not `task test` (which would rebuild the SW as a dep). Working tree stayed clean throughout.

### Two invalid attempts first (carried, not hidden)

Honesty note for the record — the first two suite attempts were **methodologically invalid and discarded**; only the third (above) is attested. Both are preserved under `suite-logs/attempts/`:

1. **Foreign-server reuse.** Playwright's `reuseExistingServer: !process.env.CI` (CI unset) silently **reused a leaked `hqserver` orphaned by a different session** (pid 34423, session `af72b7da…`, pointing at a long-dead pg on `:5439`). Every test failed on the login page — the run tested nothing about our code. Fixed by killing the orphan and setting `CI=1` (forces a fresh own-webServer + teardown).
2. **Unmigrated DB.** Go units first ran against an isolated pg that had **not been migrated** (the standalone container publishes no app service to auto-migrate it) — all 42 `internal/recipes` tests `Fatalf`'d on `relation "drift_check_results" does not exist`, while other packages returned stale `(cached)` results masking it. Fixed by migrating the DB via a throwaway app boot **before** the go phase and forcing `-count=1`.

Neither invalid attempt is a product signal; both are harness-provisioning defects in this orchestrator's first passes, corrected and documented.

### Counts (authoritative — Playwright JSON reporter + `go test`)

| Suite | passed | failed (unexpected) | flaky | skipped | Exit |
|---|---|---|---|---|---|
| **Playwright** (chromium + bdd) | **450** | **1** | **0** | **6** | 1 |
| **Go units** (`go test -count=1 -p 1 ./...`) | 7 pkgs `ok` | **0** | — | — | **0** |

**`0 flaky` is load-bearing:** with `retries:1`, the single Playwright failure failed on **both** attempts → deterministic-within-run, not transient. Full-suite wall-clock ≈ 16.3 min.

### Go units — fully green (cleaner than the 07-16 gate)
All 7 test-bearing packages `ok`, fresh (`-count=1`, no cache): `auth`, `inventory` (20.0s), `purchasing` (6.9s), **`receipt` (20.7s)**, `recipes` (15.9s), `toast`, `workflow`. **Exit 0.** Notably `internal/receipt` **passed without `ANTHROPIC_API_KEY`** — the env-gated red the 07-16 gate carried (T-8/T-11) is **not present this cycle**; the Go half of the deterministic stack is unconditionally green.

### The single Playwright red — categorized: cross-test DB-pollution (documented; isolation-confirmed)
`tests/workflows.spec.js › approved checklist shows Approved badge and cannot be resubmitted [LST-08 RUN-08]` — `expect(locator('#toast')).toBeVisible()` got `hidden` (both attempts). This is the **same test the 07-16 gate categorized as cross-test DB-pollution** (`cycle-closeout-20260716.md` §1, Category 3 — "passes alone, fails in-suite").

**Isolation re-run (the discipline: a result is not a proof).** Re-ran this one test alone on a **fresh single-test pg16** under `--retries=0`: **`1 passed (2.5m)` → GREEN.** The failure is therefore a **cross-test ordering/pollution artifact** (shared `hq_test` across the 457-test suite), **not** a product defect and **not** a new red. → **0 uncategorized reds.**

### Convergence-suite zero-flake proof (Card 1 item 3 — the no-retry hard gate)
`tests/sync.spec.js` (the two-device convergence matrix — 39 tests) run **3 consecutive times on fresh DBs under `--retries=0`**: **39 passed / 39 / 39 — 3-for-3, ~4.6m each, exit 0 all three.** The Eng "convergence matrix — 0 cells red" and Delivery "convergence proof (no-retry hard gate)" KRs rest on **demonstrated determinism**, not one pass. This confirms the operator's 2026-07-18 rider-retirement holds at the gate.

### 6 skipped = documented expected-skips (not reds)
Same set as the 07-16 gate: 2 S3-parks carried as `test.skip`+reason (`inventory FR-27 photo upload` — needs live DO Spaces; `onboarding FR-18 custom-thumbnail` — needs S3 PUT) + 4 conditional expected-skips (`persistence recipe usage_pct round-trip`; `purchasing vendor-section completion persists`; `purchasing toast without photo`; `purchasing PO approve-button-for-admin-when-locked`).

### The PARK check — cleared
PARK trigger = *"any red not attributable to a documented baseline cause."* The **one** Playwright red maps to the documented cross-test-pollution category and **greens in isolation**; the Go suite is exit-0. **0 uncategorized reds → no PARK.** `DECISIONS-NEEDED.md §A` stays empty.

### Consequence for waiver #1 (Eng KR5) — honest, not inflated
The gate's new criterion this cycle is **`task test` exits 0 on the deterministic stack** (would *formally* retire waiver #1). The full Playwright suite returned **exit 1** — one documented cross-test-pollution red short of literal exit-0. Per the slate's explicit clause ("if exit-0 over the full defined deterministic stack proves unreachable, attest the substitute and mark the Eng waiver-#1 KR **PARTIAL, not PASS**, never silently"):

- **Substitute criterion "0 new uncategorized reds vs the documented baseline" — MET** (1 red, categorized, isolation-green; Go exit-0).
- **Literal "`task test` exit 0 / 0 pre-existing reds" — NOT met** (1 pre-existing pollution red remains).
- → **Eng KR5 = PARTIAL. Waiver #1 is SUBSTANTIALLY retired (38 reds → 1) but NOT formally retired** — it carries forward, reduced to a single test-isolation defect. **Carried backlog item:** fix the `approved checklist … [LST-08 RUN-08]` cross-test isolation (some earlier spec leaves `#toast`/approval state in shared `hq_test`) so next cycle reaches literal `task test` exit-0 and formally retires waiver #1.

---

## §2 — Attestation (Card 2: cycle-gate-attestation)

**All 4 audit areas verified.** Every citation below was opened at the line/commit/test before it was written. Two honest carries surface and are marked, not hidden: (a) the Engineering-KR **commit SHAs handed to this card are dangling pre-squash worktree objects, not branch-reachable, and the slate's SHA→claim mapping for "loud rejection" and "transactional emission" is scrambled** — the *work* all exists and is verified; the attestation cites the **actual landed squashes** and corrects the mapping; (b) the red-first pairs are **documented but not git-reconstructable** because the night-crew merge protocol squashes each card's worktree into one landing commit (the 07-16 §2 caveat, per T-14).

### Reachability note (load-bearing — read before the tables)

The five editprop SHAs the slate cites (`6a483d1`, `0d49f27`, `1c7c73c`, `72fffba`, `6c3aafb`) **resolve as git objects** (`git show` succeeds) but `git merge-base --is-ancestor 6a483d1 HEAD` → **NO**, and `git branch --contains` returns nothing: they are **dangling pre-squash worktree commits**. The branch-reachable **landed** commits are the squashes, all ancestors of `HEAD` under merge `22cb7dd` (`overnight-20260717 → dev`):

| Pre-squash (cited, dangling) | Landed squash (branch-reachable) | Card |
|---|---|---|
| `6a483d1` | **`86bd09c`** | `editprop-stable-field-identity` |
| `0d49f27` + `1c7c73c` | **`186e14c`** | `editprop-broadcast-rerender` |
| `72fffba` + `6c3aafb` | **`3e5b921`** | `editprop-convergence-matrix` |

Verified the squashes carry identical diffs to the worktree objects (e.g. `86bd09c` and `6a483d1` both add `ErrUnknownField`/422 in `handler.go`+`repository.go`; `186e14c` and `0d49f27` both add `EmitOpTx` in `sync/ops.go`). Attestation cites the **landed** SHA with the worktree SHA in parentheses.

### (1) Product KR1 — 100% of PRD requirements trace to a reproduced failure or a named invariant

**Verdict: PASS (12/12 requirements trace).** Audited the PRD trace table (`PRD-data-integrity.md:170-185`), all 12 rows:

| Rows | Trace anchor | Type | PRD lines |
|---|---|---|---|
| FR-2, FR-3, FR-4, FR-7 | **REPRO** (`tests/repro-cut-task.spec.js` — the reproduced P0) | reproduced failure | `:174-177,:180` |
| FR-1, FR-5, FR-6, FR-8, FR-9 | **INV-1 / INV-3 / INV-6** (named invariants, `:64-83`) | named invariant | `:173,:178-182` |
| NFR-1, NFR-2, NFR-3 | QA discipline · Brief hard constraint · operator-signed design gate | process anchor | `:183-185` |

Every functional requirement (FR-1…FR-9, 9/9) traces to REPRO or an INV-1…6 named invariant; the three NFRs trace to named process anchors. **0 untraced rows.** Honest nuance: the 3 NFR anchors are process constraints, not members of the INV-1…6 set — still named, still auditable, consistent with the PRD's own §Requirements preamble (`:62-63`).

### (2) Product KR2 / Delivery repro-pair — 1 ratified decision + ≥2 passing acceptance tests + 1 RED/1 GREEN in the WO record

**Verdict: PASS.**

**The edit semantic as exactly 1 sign-off-ratified decision (INV-3, frozen-at-submit):**

| Artifact | Citation | Content |
|---|---|---|
| PRD invariant | `PRD-data-integrity.md:70-73` | INV-3 — "operator-delegated, PM-chosen, sign-off-ratified 2026-07-16" |
| Ledger decision record | `ledger.md:329-353` (G-2) + commit `5e7c161` "evening PM session + grill-back — frozen-at-submit PRD signed (G-1/G-2)" | one decision: FROZEN-AT-SUBMIT chosen head-to-head over run-pinned versioning |
| OKR | `okrs.md:30` | KR-2 names it as "1 operator-delegated, sign-off-ratified decision" |

Exactly **1** decision, ratified at sign-off. ✓

**≥2 passing acceptance tests (opened `tests/repro-cut-task.spec.js`, 237 lines, in the current tree):**

| Test | file:line | Asserts (claimed semantic) |
|---|---|---|
| **Test A** — AC-6a "cut a task mid-run → surviving checkbox stays checked on the observing device (live + catch-up)" | `repro-cut-task.spec.js:87-148` | Device B has the checklist open; admin (A) cuts a field via `updateTemplate`; B **live** shows the cut field gone (count 0, `:133`) AND the surviving `Wipe counters` checkbox **still checked** (`:135`), and again after reload/catch-up (`:143-144`). Mid-run edit re-renders open devices with surviving answers intact. ✓ |
| **Test B** — AC-6b "a later edit (rename + add + cut) does not change the submitted record's rendered review" | `repro-cut-task.spec.js:152-236` | After submit freezes the record, a maximally disruptive edit (rename + add + cut) is applied to the **live** template; the submitted record's rendered review is **byte-identical** (`afterFp` `toEqual(before)`, `:228`) and the frozen server `template_snapshot` string is unchanged (`snapAfter === snapBefore`, `:235`). A submitted checklist is unaffected by later edits. ✓ |

Both tests exist and assert the exact claimed semantics.

**Exactly 1 RED + 1 GREEN in the WO record:** `421ceee` committed the repro spec **skip-guarded** (baseline RED preserved, message: "Skipped so task test stays green until the editprop build card un-skips it… and records the red→green pair in its WO record"). The 07-17 HANDOFF W-1/W-2/W-3 rows (`.night-crew/runs/2026-07-17-autonomous/HANDOFF.md:70-72`) record the pair: "AC-6a … is a real red→green" (`:51`), assertions un-skipped and extended to frozen-at-submit, **red on pristine (churn build), green on the fix**. **Documented 1-RED/1-GREEN; squash-caveated** (test+fix land in one squash commit, not git-bisectable).

### (3) Engineering KRs — each landed commit verified against its claim

**Verdict: behaviors PASS; provided citations corrected (2 of 4 SHA mappings were misattributed).**

| Eng KR | Slate's cited SHA | Correct landed commit | `git show --stat` verified content |
|---|---|---|---|
| **Stable identity** — `updateTemplate` diff-upsert, old delete-and-reinsert path deleted | `6a483d1` ✓ (worktree) | **`86bd09c`** (`6a483d1`) | `repository.go` +272: `updateTemplate` diff-upserts by Builder-sent IDs (kept UPDATE, new INSERT, removed DELETE); **old `replaceTemplate` delete-and-reinsert path deleted** (`func replaceTemplate` removed). `stable_identity_test.go` +265 asserts surviving-ID guarantee. **Correction:** the slate's phrase "replaceTemplate reinserts deleted" is inverted — the delete-and-reinsert `replaceTemplate` was the *bug* and is **deleted**; `updateTemplate` replaces it. |
| **Loud rejection** — distinct 422 envelope, 0 dead-id 200s | ~~`0d49f27`/`1c7c73c`~~ **WRONG** | **`86bd09c`** (`6a483d1`) | The 422 `{"error":"unknown_field"}` envelope + `var ErrUnknownField` + `writeError(w, http.StatusUnprocessableEntity, "unknown_field")` are in the **stable-identity** commit (`handler.go`/`repository.go` in `6a483d1`), **not** `0d49f27`/`1c7c73c`. `0d49f27` is broadcast+transactional; `1c7c73c` is the INV-6 discard warning. `stable_identity_test.go` covers the 422 contract at function + handler level. |
| **Transactional op emission** — op commits in the write's txn, 0 writes with unqueued op | ~~`72fffba`/`6c3aafb`~~ **WRONG** | **`186e14c`** (`0d49f27`) | `EmitOpTx(ctx, tx, op)` — records op (lamport bump + row INSERT + `pg_notify`) **inside the caller's transaction, no self-commit** — is in `sync/ops.go` +64 in `0d49f27`/`186e14c`, with `broadcast_emit_test.go` +118 (RED on the goroutine path). **Not** `72fffba`/`6c3aafb` (those are the matrix). |
| **Convergence matrix — 0 cells red** | (find artifact) | **`3e5b921`** (`72fffba`+`6c3aafb`) → de-flaked `6291ef2` | `tests/sync.spec.js` (1534 L, in tree): two "Convergence matrix (W-3)" describes (`:832`, `:1023`) = **~11 cells** — checkbox/yes-no/text/temperature/sub-step/fail-note text+severity/fail-note photo-URL (7 field-type cells) + submit/unsubmit transitions + list-progress + list-denominator-on-cut — plus the W-6/W-6b LWW-conflict cells (`:1211,:1414`). Zero-flake under `--retries=0` proven by the 07-18 hardening (`14a36e8`, merged `6291ef2`; ledger `:441-450`). |

**Net:** all four Engineering behaviors are landed and verified in the tree; **the "loud rejection" and "transactional emission" SHA citations the card was given are misattributed** and are corrected above. **Convergence-matrix "0 cells red" rests on the landed spec + the recorded 07-18 de-flake; the live re-run is Card 1's job.**

### (4) QA KR2 — 100% of this cycle's fix-WOs carry red-run evidence

**Verdict: PASS (documented), with the squash caveat carried.** Enumerated this cycle's fix-classified WOs from the 07-17 + 07-18 HANDOFFs; each carries a documented red-run in its WO record (`.night-crew/runs/2026-07-17-autonomous/HANDOFF.md:68-78`):

| Fix-WO | Landed commit | Red-run evidence (documented) | Git-reconstructable? |
|---|---|---|---|
| W-1 `editprop-stable-field-identity` | `86bd09c` | 422 `unknown_field` + cross-device identity red on churn build → green; G6 re-reproduced | No — squash |
| W-2 `editprop-broadcast-rerender` | `186e14c` | 5 sub-behaviors red→green (SAVE_TEMPLATE re-render, silent catch-up, transactional emission, INV-6) | No — squash |
| W-3 `editprop-convergence-matrix` | `3e5b921` | AC-6a bug-guard + AC-6b snapshot-lock red→green; unsubmit-broadcast gap fixed | No — squash |
| W-4 `engine-approval-feedback-loud` | `f50dd32` | 200 (false "Approved") → 500 `feedback_persist_failed`; G6-reproduced | No — squash |
| W-5 `ops-nfr3-resubmit-photo-gate` | `733fa16` | direct-API resubmit 201 bypass → 400 `resubmit_photo_required`; G6-reproduced | No — squash |
| W-6 `engine-conflict-refetch` | `fc0ed6b` | LWW loser `undefined` → `WINNER` via `applyOp`; deterministic 3/3, G6-reproduced | No — squash |
| (07-18) `editprop-convergence-cell-hardening` | `14a36e8` (merged `6291ef2`) | 4 conflict types red→green + no-retry de-flake streak; 2 fail-note types parked (D-1, footprint) | No — squash |

Excluded from the fix denominator (correctly, not fixes): **U-1** `users-s3-orphan-cleanup` (`a11a58f`, hygiene, zero behavior change) and **T-2** `vacuous-tests-18-to-0` (`3fd4d3f`, test-only conversion, retires waiver #2). **T-1** `carried-fix-wos-sweep` (`c5aede8`) carries a behavioral red→green (clock-seam) though it is seam/test-shaped.

**100% of fix-WOs (6/6 core + the 07-18 hardening) carry documented red-run evidence.**

**Honest caveat (carried, not hidden — per T-14):** the night-crew merge protocol squashes each app-fix card's worktree into a **single** landing commit, so the new spec test and the fix arrive together — git cannot show a standalone failing-before commit for any of them. Red-first ordering is **documented** (07-17 HANDOFF per-card table naming each observable break; commit messages assert "red-first"; independent fresh-G6 re-reproduction recorded at `:80-82`) but **not git-reconstructable** for any fix this cycle. Unlike the 07-16 cycle — which had exactly one git-verifiable RED→GREEN pair (Inventory NFR-1, `1a0265e` precedes `77957c1`) — **this cycle has zero standalone git-bisectable red→green pairs**; the repro baseline `421ceee` was committed *skip-guarded* (kept green on purpose), so even it is a documented, not executed-failing, baseline. QA-KR2 is attested **on the WO/ledger record**, corroborated by the independent per-card G6 re-reproduction.

---

## §3 — OKR scorecard + Delivery metric (Card 3: cycle-gate-scorecard)

**Tally (reconciled against §1): 11 PASS · 2 PARTIAL · 2 PENDING · 1 N/A (= 16 KRs).** No inflation — the 2 PENDING are prod-gated (flip at Activity 7), the 2 PARTIAL are (a) the median-instrumentation gap and (b) Eng KR5 waiver-#1 (§1 found literal `task test` exit-0 not reached — 1 documented pollution red — so PARTIAL-substitute per the slate clause), and the 1 N/A is the no-migration-this-cycle down-migration KR. **Waiver #1 is substantially retired (38→1 red) but NOT formally retired → carried;** waiver #2 (vacuous 18→0) IS retired at Q1 against commit `3fd4d3f`.

> **Reconciliation note:** this card pre-computed E5 as PASS with the explicit conditional "if §1 finds exit-0 unreachable, downgrades to PARTIAL-substitute per the slate PARK clause." §1's actual result (450 pass · 1 pollution red · Go exit-0; literal `task test` exit-1) triggers exactly that downgrade. E5 below is updated to PARTIAL; every other row stands as computed.

### Delivery metric — median WO cycle time this cycle vs baseline (N=23 / 22m28s, T-14)

**Verdict: NOT COMPUTABLE this cycle → Delivery median KR PARTIAL. No median fabricated.**

The baseline recorded last cycle (T-14, `cycle-closeout-20260716.md` §3) is **N=23, median 22m28s** — computed from the two *delivery* runs that carried per-card wall-clock: `overnight-20260714` (serial, N=6: 6m28s–27m12s, median ≈19m25s) and `overnight-20260715` (concurrent rolling-3, N=17: 8m26s–32m44s, median 23m24s).

This cycle ("Nothing silently lost") shipped its delivery work across **two runs**, but only one card is per-card timed:

- **`overnight-20260717` (9 build cards) — NOT per-card instrumented.** Its HANDOFF recorded verdicts + *qualitative* sizing only ("app-fix + red-first cards ~15–30m impl; the two XL editprop cards W-2/W-3 ran longest"), never per-card minutes (`card-actuals.md` §"overnight-20260717", explicit ⚠ note).
- **`overnight-20260718` (1 card, `editprop-convergence-cell-hardening`) — measured:** **~73m impl** (4,387,969 ms) / **~16m G6** (964,145 ms) / ~2m merge. This is its own *convergence-hardening* size class (~60–90m, dominated by the `--retries=0` de-flake proof streaks), not the 15–30m clean-path fix-card class — so it is not median-comparable to the baseline's WO population even standing alone.

**Because 9 of this cycle's ~10 delivery WOs are untimed, a this-cycle median is not derivable.** A single measured point (`-0718`, and one from an atypical size class) cannot yield a median; deriving one would be fabrication. → The Delivery "median WO cycle ≤ baseline" KR is **PARTIAL**: the baseline stands (T-14, unchanged), one this-cycle point is measured, the population is not.

**Fix-forward (logged):** re-adopt per-card impl/G6 wall-clock tabling as a standing run output — `card-actuals.md` notes `-0718` already re-adopted the harness-measured table; make it the invariant for every build run so the ledger stays *measured*, not narrated. Until a fully-instrumented delivery run lands, the baseline remains the only computable median.

### OKR scorecard (all 16 KRs — mirrors 07-16 §3; "attest against §1/§2" rows reconciled by the orchestrator against Cards 1–2)

| KR | Verdict | Evidence | Honest gap |
|---|---|---|---|
| **Product KR1** — data-integrity PRD as blocking gate, 100% reqs trace to a reproduced failure or named invariant (trace table, audited at gate) | **PASS** *(attest §2)* | `PRD-data-integrity` (frozen-at-submit) signed at the PM grill-back (ledger G-1/G-2, `5e7c161`); Card 2 audits the PRD trace table — 12/12 trace | Trace-table completeness is §2's audit surface (3 NFRs → named process anchors) |
| **Product KR2** — edit semantic = 1 operator-delegated sign-off decision, encoded as ≥2 passing acceptance tests | **PASS** *(attest §2)* | Semantic recorded as G-2 (operator delegated → PM chose frozen-at-submit, 2026-07-16); Test A (`repro-cut-task.spec.js:87-148`) + Test B (`:152-236`) both assert the semantic | Both acceptance cells' green state is §2's attestation |
| **Product KR3** — 15/15 cycle-open `new` backlog items routed; 0 `· new` remain | **PASS** *(scoped)* | The 15 cycle-open items (8 main + 4 test-hardening + 3 PARK fix-WOs) were routed — `grep -c '· new'` = **0** at routing commit `7bf10ad` | ⚠ Literal audit now returns **7** `· new` in `.night-crew/knowledge/BACKLOG.md` — all **post-open accretions** (PM-session 2026-07-16 pass-2 sweep + F-C/W-4 and T-2 triage follow-ups, dated ≥ cycle open), i.e. next-cycle feedstock, **not** unrouted cycle-open items. Spirit met; raw grep drifted by legitimate accretion |
| **Delivery KR1** — edit-propagation design operator-signed BEFORE any build card (ledger timestamps) | **PASS** | Design signed `41751ce` (Jul 16 **15:16:34**) precedes first build `86bd09c` (Jul 16 **16:10:15**) by ~54m | None — clean git-timestamp ordering |
| **Delivery KR2** — `repro-cut-task.spec.js` rewritten to frozen-at-submit; exactly 1 red + 1 green in WO record | **PASS** *(attest §2)* | Repro spec committed skip-guarded (`421ceee`), rewritten/greened across the editprop chain (`86bd09c`, `3e5b921`); red→green pair in the WO records | Squash-merge caveat — pre-fix red not standalone-reconstructable; documented |
| **Delivery KR3** — prod parity: `task version` prod backend/frontend == local `version.go` | **PENDING** *(prod-gated)* | Cannot attest in a read-only gate — Activity 7 (operator-run `task prod:deploy`) not yet executed | **Verify command (Activity 7):** `task version` → assert prod `Backend`/`Frontend` == local `backend/internal/version/version.go` constants. Flip on match |
| **Delivery KR4** — median WO cycle time ≤ baseline (N=23/22m28s) | **PARTIAL** | Baseline stands (T-14); this-cycle only `-0718` measured (~73m impl/~16m G6, own size class); `-0717`'s 9 cards uninstrumented | Full this-cycle median not computable (9/10 WOs untimed) — fix-forward: standing per-card timing table. **Not fabricated** |
| **Engineering KR1** — stable identity: `updateTemplate` upserts by IDs (no delete+reinsert) | **PASS** *(attest §2)* | `86bd09c` (`6a483d1`) "stable field identity via `updateTemplate` diff-upsert" — old `replaceTemplate` delete+reinsert removed | Reconciled with §2's diff attestation |
| **Engineering KR2** — loud rejection: absent-field writes rejected server-side w/ distinct envelope, 0 return 200 | **PASS** *(attest §2)* | `86bd09c` (`6a483d1`) — 422 `unknown_field` app-level existence check; red-first dead-id path (slate SHA mapping corrected in §2) | 0-dead-id-200s is §2's landed-commit attestation |
| **Engineering KR3** — edit propagation re-renders open devices w/ surviving answers, silent on catch-up; op emission transactional | **PASS** *(attest §2)* | `186e14c` (`0d49f27`) "broadcast SAVE_TEMPLATE re-render + transactional `EmitOpTx` + INV-6 discard warning" | F-B (Create/Archive still fire-and-forget) is a backlogged parity residual, not a KR breach |
| **Engineering KR4** — convergence matrix: 7 field types + sub-steps + submit/unsubmit + list progress converge across ≥2 devices; 0 cells red | **PASS** *(attest §1)* | `3e5b921` two-device matrix + `14a36e8` cell hardening — `text`/`temperature` cells zero-flake under `--retries=0` (rider RETIRED 2026-07-18); Card 1 proves the streak | 2 fail-note conflict types parked (D-1, footprint-blocked `applyOp` `_fail_note` unpack) — rare crew path, server-side never loses data. Rests on §1's `--retries=0` streak |
| **Engineering KR5** — `task test` exits 0 at cycle end on the deterministic stack (retires carried waiver #1) | **PARTIAL** *(§1: substitute met, literal exit-0 not reached)* | §1: Go units exit-0 (all 7 pkgs, `-count=1`); Playwright **450 pass · 1 fail · 0 flaky · 6 skip**; the 1 red is isolation-confirmed cross-test DB-pollution (greens alone → 0 uncategorized). Substitute "0 new uncategorized reds vs baseline" **MET**; suite reduced **38 reds → 1** vs the 07-16 gate | Literal `task test` exit-0 **not** reached (1 pre-existing pollution red → PW exit-1), so **waiver #1 is substantially but NOT formally retired → carried.** Backlog: fix the `approved checklist … [LST-08 RUN-08]` cross-test isolation to reach literal exit-0 next cycle |
| **QA KR1** — vacuous-test remainder 18 → 0 (retires carried waiver #2) | **PASS** *(retires waiver #2)* | Commit **`3fd4d3f`** `test(vacuous): convert 16 audited vacuous guards… (vacuous-tests-18-to-0)` — 18 = 16 converted + 2 Ops already hardened at base; confirmed ancestor of `overnight-20260719` | ⚠ **SHA correction:** the slate cites `3f68cc9`, a **superseded pre-merge object NOT in the run-branch ancestry**. Retirement attested on the correct landed commit `3fd4d3f` |
| **QA KR2** — 100% of this cycle's fix-WOs carry red-run evidence | **PASS** *(attest §2)* | Fix cards each red-first: `86bd09c`, `186e14c`, `3e5b921`, `f50dd32`, `733fa16`, `fc0ed6b` (+07-18 `14a36e8`) | Squash-merge caveat (bundle test+fix → pre-fix commit not standalone-reconstructable) — **carried, not hidden** (T-14) |
| **QA KR3** — prod ghost catalog item resolved: `SELECT count(*) FROM purchase_items WHERE trim(description)=''` → 0 in prod AND line-items count unchanged | **PENDING** *(prod-gated)* | Cannot attest in a read-only gate — Activity 7 (`prod-ghost-item-rename` + operator prod DB mutation) not yet run | **Verify command (Activity 7):** `SELECT count(*) FROM purchase_items WHERE trim(description)='';` → assert **0** in prod, AND assert the previously-linked `purchase_line_items` count is unchanged (rename `''` → `(Unnamed — needs review)`, links preserved). Flip on both |
| **QA KR4** — 100% of schema migrations shipped this cycle have a proven up→down→up down-migration + 1 pre-deploy DB backup | **N/A** | **No schema migration shipped this cycle** — `git log 2931adc..overnight-20260719 -- backend/internal/db/migrations/` is **empty**; highest migration `0070` unchanged. Frozen-at-submit (G-2) deleted the versioning schema; the wipe/reseed is moot | N/A by construction, not PASS — no migration exists to prove a down-path or back up |

**Cycle summary — 11 PASS · 2 PARTIAL · 2 PENDING · 1 N/A (16 total).** Closed on evidence this gate can see: Product KR1/2/3, Delivery KR1/2, Engineering KR1–4, QA KR1/2 — with **waiver #2 retired** (Q1, `3fd4d3f`). **PARTIAL-and-named:** Delivery KR4 (median not computable — 9/10 WOs uninstrumented; baseline unchanged, no median fabricated) and **Engineering KR5** (§1: substitute "0 new uncategorized reds" met, but literal `task test` exit-0 not reached — 1 documented pollution red — so **waiver #1 substantially-but-not-formally retired, carried**). **PENDING → Activity 7 (attended ship step):** Delivery KR3 (prod parity) and QA KR3 (prod ghost item), each with its exact verify command recorded. **N/A:** QA KR4 (no migration shipped). No inflation to all-PASS.

**Orchestrator corrections folded in:** (1) waiver #2 is commit **`3fd4d3f`**, not the slate's `3f68cc9` (stale pre-merge SHA, not in ancestry); (2) `BACKLOG.md` lives at `.night-crew/knowledge/BACKLOG.md`, `grep -c '· new'` currently returns 7 post-open accretions (P3's scoped 15/15 satisfied at `7bf10ad`).

---

## §4 — Gate verdict

**CYCLE GATE: PASS (attested), under the operator's 2026-07-17 "Gate now, prod KRs pending" posture. No card parked — the PARK trigger did not fire. `DECISIONS-NEEDED.md §A` is empty.**

| Gate criterion | Verdict | Basis |
|---|---|---|
| 0 known-broken flows (this cycle's fixes landed, none regressed) | **ATTEST PASS** | §2: stable identity (`86bd09c`), loud rejection (422 `unknown_field`), transactional broadcast (`186e14c`), convergence matrix (`3e5b921`), + approval-feedback / resubmit-photo / conflict-refetch fixes — each a landed squash; §1's suite confirms no repaired flow regressed |
| `task test` exit 0 on the deterministic stack (would retire waiver #1) | **PARTIAL → attest substitute "0 new uncategorized reds vs the documented baseline"** | §1: **450 pass · 1 fail · 0 flaky · 6 skip** (Playwright) + **Go units exit-0**. The 1 red is isolation-confirmed cross-test DB-pollution (`approved checklist… [LST-08 RUN-08]` — `1 passed` alone) → **0 uncategorized**. Literal `task test` exit-1 (1 pollution red) ⇒ **waiver #1 substantially (38→1) but NOT formally retired — carried** |
| Vacuous tests 18 → 0 (retire waiver #2) | **ATTEST PASS — WAIVER #2 RETIRED** | §3: `3fd4d3f` `test(vacuous): convert 16 audited vacuous guards… (vacuous-tests-18-to-0)`; 18 = 16 converted + 2 Ops hardened at base. (Slate's `3f68cc9` was a superseded pre-squash object — corrected to the landed SHA) |
| Convergence matrix — 0 cells red, no-retry hard gate | **ATTEST PASS** | §1: `sync.spec.js` **39/39 passed × 3 consecutive `--retries=0` fresh-DB runs** — demonstrated determinism, not one pass; validates the 2026-07-18 rider-retirement at the gate |
| Edit semantic = 1 ratified decision + ≥2 acceptance tests | **ATTEST PASS** | §2: INV-3 frozen-at-submit (1 sign-off-ratified decision, ledger G-2); Test A (`repro-cut-task.spec.js:87-148`) + Test B (`:152-236`), both asserting the semantic |
| Every repaired flow red-first | **ATTEST (documented)** | §2: all fix-WOs carry documented red-runs (07-17/07-18 HANDOFFs + G6 re-reproduction). **Squash caveat carried:** 0 git-bisectable red→green pairs this cycle (test+fix bundled per squash), unlike 07-16's single verifiable pair — attested on the WO/ledger record, not on git |
| Median WO cycle time vs baseline (N=23/22m28s) | **PARTIAL — not computable** | §3: 9 of ~10 delivery WOs (the 07-17 run) were not per-card timed; only `-0718`'s single card measured. Baseline stands; no median fabricated. Fix-forward: standing per-card timing table |
| Prod parity (`task version`) + prod ghost item (`trim(description)=''` → 0) | **PENDING → Activity 7** | §3: prod-gated; cannot attest in a read-only gate. Exact verify commands recorded in §3 + `DECISIONS-NEEDED.md` for the attended ship step |

**Disposition:** 4 criteria ATTEST PASS · 2 PARTIAL (waiver-#1 substitute; median not computable) · 2 PENDING (prod, Activity 7). **Waiver #2 retired; waiver #1 carried, reduced 38→1.** The gate ratifies the dev-side deterministic stack as green-modulo-one-documented-pollution-red, with the 2 prod KRs deferred to the attended ship step exactly as the resolved fork intended.

**Milestone boundary reached.** The "Nothing silently lost" cycle is closed on the evidence this gate can see. Morning move: `/nc-morning-triage` (review + merge `overnight-20260719`), then the attended **Activity 7** ship step (`prod-ghost-item-rename` + operator `task prod:deploy` → flip the 2 PENDING KRs), then `/nc-okr-session` to open the next cycle. **Carried to the next cycle's roadmap/backlog:** (1) **waiver #1's last mile** — fix the `approved checklist … [LST-08 RUN-08]` cross-test isolation so `task test` reaches literal exit-0; (2) **per-card timing instrumentation** as a standing run output (Delivery-median fix-forward); (3) the editprop follow-ups already in `BACKLOG.md` (F-B transactional Create/Archive emission, F-C atomic approval+feedback tx, F-E `waitForResponse` over fixed flush, D-1 fail-note conflict live-render on the `applyOp`/409 path).
