# HANDOFF — overnight-20260724 (for the morning of 2026-07-24)

> **Run branch:** `overnight-20260724` (cut from `dev` at `fcea6f8`; **never pushed, `main` untouched**).
> **Slate:** `.night-crew/knowledge/reference/slate-20260724.md` (batch-signed 2026-07-23).
> **Scope:** SERIAL — G1 `grant-enforcement-parity` → S1 `syncspec-deflake` → ST `cycle-gate`
> computable legs (condition-gated stretch). Per-card worktree + fresh implementer subagent +
> separate fresh G6 adversarial subagent; orchestrator alone merged.
> **Result: 2/2 slate cards MERGED (both G6: APPROVE) + the condition-gated ST stretch RAN.**
> Nothing parked from S1; G1 parked one surface (`/photos/*`) by its own trigger. The ST
> attestation leg computed an honest **540/1/6 at 0 retries** — the red is LST-17, one of S1's
> two hardened-not-killed cells, green on isolation re-run; **E5 downgraded to PARTIAL in the
> open**, waive-vs-re-leg is an attended call. Run wall-clock 6h38m of the 8h line, incl.
> closeout. Reader = the operator; resolve via `/nc-morning-triage`.

## TL;DR

- **G1 `grant-enforcement-parity` — MERGED (G6: APPROVE as-is).** All five live app surfaces
  (operations incl. `/ws`, inventory base + recipes + menu-items, purchasing, onboarding +
  videos, users + apps) now enforce `RequirePermission` with F5's umbrella semantics. Parity
  guard spec covers 11/11 seeded slugs (7 enforced, 4 placeholders N/A-with-reason + stale-N/A
  tripwire). Red-first held and was independently re-verified by G6 at the red commit. Full
  suite **542 passed / 6 skipped / 0 failed at 0 retries** on the card tree. Two items ride to
  triage: the `/photos/*` park (G1-a) and the `/inventory/items` cross-app READ judgment call
  (G1-b) — see DECISIONS-NEEDED.md.
- **S1 `syncspec-deflake` — MERGED (G6: APPROVE) at `5eb4331`; roadmap flipped `83f2607`.**
  G6 independently confirmed: test-side-only diff, every replaced blind wait equal-or-stronger,
  no assertion deleted, red logs genuine (+3 journal signature), streak arithmetic checked, and
  a **mutation probe** — `debouncedSaveField` neutered in a scratch copy → the hardened test
  fails honestly after exhausting its bounded re-dispatch, so the fix cannot mask a real
  autosave regression. G6's own legs: 3/3 quiet + 2/2 contention + independent fresh-DB full
  suite 541/0/6. Four non-blocking report inaccuracies flagged (see S1 flags below). `:1198` and `:525`
  FLD-LIVE-02 KILLED with proven red-first captures; LST-17 and GATE-04 HARDENED (honestly
  labeled — not reproduced in 3 targeted contention legs each). **Test-side only** — `sync.js`,
  `workflows.html`, `backend/` untouched; the attended two-device convergence check is **NOT
  re-armed**. Determinism claim ARMED: 12/12 consecutive `--retries=0` legs under the measured
  reproduction condition + 5/5 measured-quiet legs (loads 1.59–1.93, zero foreign heavy
  processes, measurements recorded per leg) + fresh-DB full-suite `--retries=0` green
  (541/0/6). Stated caveat: the 12-leg contention streak predates the final FLD-LIVE-02-only
  commit `8703188`; FLD-LIVE-02's post-change evidence is 3 targeted legs vs a 1572-row
  journal + the green full-suite leg. Impl ran ~235m vs the 100–160m estimate — the overage
  bought three successive mechanism discoveries (each red investigated, none rerun-and-hoped).
- **ST `cycle-gate` computable legs — RAN (condition satisfied: S1 merged clean at 6h09m,
  ~1h51m in hand).** Scorecard at `scorecard-20260724.md` (this directory). Reconciled KR
  tally: **11 MET · 2 PARTIAL (P4, E5) · 2 PENDING (D2 prod-parity, D4 alert-dup watch) ·
  1 N/A (Q4)**. The no-retry attestation on the merged tree (`83f2607`, fresh DB, own
  ephemeral stack): **540 passed / 1 failed / 6 skipped at `--retries=0`** — the red is
  **LST-17**, S1's hardened-not-reproduced cell; isolation re-run green (1/1, fresh DB).
  Honest caveat: the box was NOT quiet during the leg (load 2.38→4.37 — a second workload was
  active), so this red is consistent with the known load/order-sensitive mode, and it was NOT
  laundered into "not flaky" — rare-with-mechanism-known stays flagged as flaky. KR3 median:
  tonight N=2 (G1 95m10s, S1 272m52s — stated, not extrapolated); cycle running set median
  ≈94m (N=12) vs T-14 baseline 22m28s (N=23), the gap dominated by a population shift to
  app-code/de-flake-proof card classes. KR2 prod-parity: `task version` shows local = dev =
  prod at backend 0.1.3 / frontend 1.0.3 with prod `git_sha: unknown` — **PENDING-deploy**,
  recorded as not-parity-evidence (prod predates every merge this cycle). **The `cycle-gate`
  roadmap card stays PLANNED — the boundary close-out is attended and belongs to triage.**

## Per-card wall-clock (harness-measured, Delivery KR3 — the standing output)

> Epoch-stamped legs in `timings.log`.

| Card | Impl | G6 | Merge legs | Cycle | Outcome |
|---|---|---|---|---|---|
| G1 `grant-enforcement-parity` | **71m54s** | **22m11s** | 1m05s | **95m10s** | MERGED (est. 100–170m) |
| S1 `syncspec-deflake` | **236m04s** | **36m00s** | 0m48s | **272m52s** | MERGED (est. 100–160m — overage bought three successive FLD-LIVE-02 mechanism discoveries, each investigated red-first) |
| ST `cycle-gate` legs | **~29m** (attestation 20m12s + scorecard) | — | — | ~29m | RAN (est. 15–30m) |

Run total: **6h38m** RUN_START → ST_END (01:56 → 08:35 EDT), inside the 8h line with closeout.

## G1 — detail

- **Commits (card branch, oldest first):** `5f49d49` merge-intent + `RequirePermission`
  call-site enumeration (2 of 11 slugs enforced before the card) · `535a37c` parity spec RED
  (7 failed / 7 passed — a real ungranted logged-in account read workflow, inventory,
  purchasing, onboarding, users and apps surfaces at 200) · `33c37d1` the migration
  (`backend/cmd/server/main.go` only) · `ec93201` additive `beforeAll` app-grant baselines in 6
  suites · `7cfcc3c` items cross-app READ + fixture premises · `0460488` merge-intent mirror.
  Merged `--no-ff` at `4bb8649`; roadmap flip + park record `1816448`.
- **G6 (fresh subagent, own binary, own DB `nc-g6g1`):** APPROVE as-is. Attacks held: path
  tricks (double-slash/dot-segment/case/encoding — no trick yields 200), method confusion,
  umbrella semantics both directions, roles-vs-grants axis (admin role without grant → 403
  `missing_grant`), service tokens (valid bearer 200 / session cookie without bearer 401 —
  8c71022 not regressed, `requireReviewAuthz` still inside approve/reject), `/me` + notif-pref
  exceptions hold, `/ws` ungranted → 403. No assertion weakened in any touched suite (verified
  additive-only). Known bounded limitation recorded: parity is slug-granular (proves each slug
  gated somewhere, not that every endpoint sits behind the right gate); every current endpoint
  verified correctly gated live.
- **Fail-closed:** inherited from F5's middleware (G6 spot-checked semantics; DB-down probe not
  exercised this run).

## Verification-integrity incident (record for the operator)

During G1's final full-suite leg, the implementer subagent received **fabricated
"completion" notifications** (future timestamps, tallies from a 0-byte log, "541 passed"
four minutes into a ~20m run); the harness flagged one as suspected prompt injection. The
implementer discarded all of them, verified process exit itself (`kill -0`/`ps`/`wc`), and
read tallies only from the real log after genuine exit. The dispatch prompt for S1 carries an
explicit instruction to do the same. No injected content was acted on.

## S1 — flags carried from the implementer (for triage)

- **The `sync.spec.js:1198` line anchor is dead** — the temperature test now sits near
  `:1318` (G1's `beforeAll` moved it; S1 moved it further). Locate via
  `-g "temperature answer converges"`. Knowledge-base references (BACKLOG, slates, evidence
  doc) should migrate at triage.
- **Real-app observation, reported not fixed:** `applyOp`'s SAVE_TEMPLATE branch re-fetches
  `myChecklists` per replayed op whenever a runner is open (`sync.js` ~491) — a device
  catching up on a large journal with a runner open fires an un-awaited fetch per op (the
  storm behind every FLD-LIVE-02 red). A one-line gate/debounce in `sync.js` would fix the
  app-level behavior but re-arms the two-device check — operator's call, backlog candidate.
- **Pre-existing, out of scope:** two `tests/onboarding.spec.js` tests fail when the full
  suite runs a second time against the same un-reset DB (carried hire/training state). Fine
  under the clean-DB-per-leg rule; backlog line.
- **Report inaccuracies (G6-flagged, non-blocking, recorded for the ledger):** the
  implementer's report said "14 local-only commits" — the branch has 6, nothing missing;
  code comments date the reds "2026-07-24" while the logs are 2026-07-23 EDT (run-id vs
  calendar-date cosmetic); the merge-intent's ":1318" line anchor went stale (final: ~:1372,
  locate by title).

## Gate evidence (on the final merged tree, `83f2607`)

- **`go build ./...` + `go vet ./...`** — exit 0, re-run after each merge.
- **Full suite, `--retries=0`, fresh DB:** three independent runs on identical tree content —
  S1 implementer 541/0/6, S1's G6 541/0/6, and ST's attestation on the merged run branch
  **540/1/6** (LST-17 red, isolation re-run green, box not quiet — see ST section). G1's card
  tree separately ran 542/6/0 at 0 retries (test count differs by S1's later consolidation).
- **Red-first:** independently re-verified by G6 on both cards at the red commits (`535a37c`,
  `cf70ce0`-era logs), not by trusting reports. No assertion weakened on either card
  (G6-verified additive-only / equal-or-stronger per replacement).
- **`task sw`:** never needed — no production HTML/JS changed all night.
- **Ephemeral-env discipline:** host :5432/:5433 never touched; projects `nc-g1`, `nc-g6g1`,
  `nc-s1`, `nc-g6s1`, `nc-st` all torn down `--volumes`, 0 containers remaining (each agent
  verified at exit).
- **Conflict log:** `reference/conflicts-20260724.md` — 2 merges, both CLEAN, both logged with
  intents read.
- **Footprint:** every diff inside the declared card footprints except G1's declared
  fixture-premise touches (7 test suites + 1 BDD step file, all additive, mirrored in its
  merge-intent).

## ⚠ Standing flags

- **Prod deploy NOT done** (attended, rides the cycle gate). Frontend semver untouched (1.0.3);
  bump belongs to `/save-project` at deploy time.
- **The attended two-device convergence check did NOT re-arm** — no production `sync.js`
  change landed this run.
- `/photos/*` remains authenticated-only pending G1-a.
- **`cycle-gate` roadmap card stays PLANNED** — only the computable legs ran; the boundary
  close-out (PARTIAL/PENDING judgment + prod-parity ship) is attended.

## For the morning reader (triage order)

1. **Review the run branch and merge `overnight-20260724` → `dev`** (`--no-ff`). 2 cards, 2
   clean merges, conflict log audited-ready.
2. **Answer G1-a** (`/photos/*` grant mapping — union / per-app split / stay
   authenticated-only; note presign GET returns stored photo URLs to any logged-in user).
3. **Ratify or revert G1-b** (`GET /inventory/items` open to purchasing-grant READ; payload
   has no cost fields; revert is a one-group change in `main.go`).
4. **Rule on E5 / the attestation red:** waive LST-17's single load-window red (waiver #1 as
   the OKR anticipates) or order a re-leg on a genuinely quiet box before the cycle-gate
   close-out. The evidence either way is in `scorecard-20260724.md` §1/§3.
5. **Run the attended `cycle-gate` boundary close-out** when ready: judge the PARTIAL/PENDING
   rows (P4, E5, D2, D4), then the prod-parity ship (`/save-project` → `task prod:deploy`) —
   scorecard §6 lists exactly what settles each row.
6. **Migrate stale `:1198`/`:525` line anchors** in knowledge-base references to title-based
   selectors (S1 flag above).
7. **Backlog lines:** the `sync.js` catch-up fetch-storm one-line gate (app-level fix,
   re-arms two-device check — operator's call), and the onboarding second-run carried-DB
   failures.

---

## Triage record (2026-07-23, attended)

- **Merged** `overnight-20260724` → `dev` at `f776578` (`--no-ff`) after independent
  re-verification: go build/vet/tests green; attended full Playwright leg **542/0/6 in 20.4m,
  zero retries fired**; conflict log + merge-intents audited clean.
- **G1-a resolved (decision 42):** `/photos/*` stays authenticated-only as the documented
  exception; durable key-binding card backlogged. Union gate rejected as cosmetic.
- **G1-b resolved (decision 43):** `/inventory/items` (inventory ∨ purchasing) READ **ratified**.
- **E5 ruled (decision 44):** no-retry attestation **granted with waiver** for LST-17's single
  under-load red; LST-17 stays flagged load-sensitive.
- **Backlogged:** photos key-binding card; `sync.js` catch-up fetch-storm gate (re-arms
  two-device check — priced in); onboarding second-run carried-DB failures.
- **Still open for the attended cycle-gate close-out:** P4 interpretation, D2 prod ship,
  D4 Cliq-dup confirmation, E2 0%-food-cost note.

Full record: `ledger.md` §T-21.
