# Scorecard — "Prove & surface" cycle · computable legs only

> **Run:** overnight-20260724 (stretch card ST — `cycle-gate` computable legs).
> **Attestation tree:** branch `overnight-20260724` @ `83f2607` (both slate cards merged:
> G1 `grant-enforcement-parity` @ `4bb8649`, S1 `syncspec-deflake` @ `5eb4331`).
> **Date:** 2026-07-23 (night of, for the morning of 2026-07-24).
> **What this is:** the read-only computable legs of the `cycle-gate` card — suite-green
> attestation, per-KR scorecard, Delivery-KR3 median, Delivery-KR2 parity observation.
> **What this is NOT:** the milestone close. The boundary decision — grading PARTIALs,
> ratifying PENDINGs, the prod ship — is attended and belongs to morning triage /
> `/nc-milestone-close`. Nothing here closes anything.

---

## §1 Attestation — full suite, NO-RETRY hard gate (newly eligible via S1)

- **Tree:** detached throwaway worktree at `83f2607` (`git worktree add --detach overnight-20260724`),
  outside the repo, removed after the leg.
- **Stack:** isolated ephemeral compose project `nc-st` (`docker-compose.nc.yml` + a
  scratch override exposing postgres) — postgres:16 on Docker-assigned host port **57390**;
  fresh `hq_test_e2e` created via psql DROP/CREATE (the `task test` shape); Playwright
  spawned its own Go server on **TEST_PORT=8691** (`reuseExistingServer` is hard-false;
  `CI=1` set per gate run-mechanics). No shared stack, no `:5432`/`:5433` touch.
- **Load before the leg (uptime, 08:09:21):** `load average: 2.38, 2.25, 2.55` — NOT a
  quiet box (recorded honestly; this leg is a no-retry suite-green attestation, not a
  quiet-window flake-streak proof — S1 already banked that class of evidence).
- **Command (verbatim):**
  `CI=1 DB_HOST=localhost DB_PORT=57390 TEST_DB_NAME=hq_test_e2e TEST_PORT=8691 npx playwright test --retries=0 --reporter=line`
- **Retries:** `--retries=0` (overrides the config's `retries: 1`) — the hard gate.
- **Wait discipline:** run detached; process exit verified by own probes (PID liveness +
  exit-code file); tallies read only from the real log after genuine exit. (Fabricated
  mid-run completion notifications appeared earlier this run — none trusted, none acted on.)
- **Result: FAIL at the no-retry bar — 540 passed / 1 failed / 6 skipped in 20.9m, exit code 1.**
  (Start epoch 1784808561 → end 1784809773, wall ≈ 20m12s to process exit; log tallies at
  `suite.log:6044-6045`, read after verified exit.)
- **The 1 red:** `tests/sync.spec.js:446 › Cross-device: regressions › list page progress
  decrements when another device unchecks a field [LST-17]` — `toContainText("0/1")` timed
  out at 12s; the row still read `1/1 items` (the cross-device decrement never rendered).
  **This is exactly one of the two tests S1 pre-labeled HARDENED-not-killed** ("not
  reproduced in 3 targeted contention legs each, labeled honestly"). Tonight it reproduced.
- **Isolation re-run (gate precedent, 07-16/07-19):** fresh `hq_test_e2e` DROP/CREATE, same
  stack, same `--retries=0`, `-g` targeted → **1 passed (2.4m), exit 0.** Categorization:
  full-suite-order/load-sensitive, not a deterministic regression. Load ran 2.38 → **4.37**
  over the leg (a second workload was active on the box) — the exact condition S1's quiet-box
  legs (loads 1.59–1.93) did not cover.
- **Attestation verdict, stated plainly: the suite-green NO-RETRY attestation is NOT earned
  tonight.** 546/547 non-skipped outcomes green at 0 retries; the single red is a known,
  honestly pre-labeled hardened-not-killed cell that greens in isolation. Whether that
  constitutes an acceptable gate pass (waive-with-reason) or requires a quiet-box re-leg is
  an **attended call** — not made here, not laundered.

## §2 Per-KR scorecard (16 KRs, by team)

> Statuses: MET / PARTIAL / NOT MET / PENDING-<reason> / N/A-<reason>. A KR whose evidence
> is not on file is PENDING or NOT MET — nothing graded generously, nothing fabricated.

### Product

| KR (abbrev.) | Status | Evidence |
|---|---|---|
| P1 · Cycle PRD as blocking gate; 100% requirements traced | **MET** | `prd-prove-and-surface` DONE, signed 2026-07-19 (`runs/2026-07-19-attended/sign-off.md`); roadmap Activity 1; no build WO predates it (ledger T-18 order) |
| P2 · 3 escaped defects each map to a would-have-caught matrix cell | **MET** | `convergence-matrix-systematic` DONE (merged `8249209`): 32-cell matrix, 3 escaped defects each carry a cell that reddens on the pre-fix build |
| P3 · Gating decision recorded, ratified, observable rule encoded | **MET** | Design signed 2026-07-20, ledger T-18 (`designs/prove-surface-gating-and-endpoints.md` §8: A4 = Option (i) two per-tab slugs, umbrella semantics); enforced by F5 (merged `c1a2393`) |
| P4 · 12/12 cycle-open `· new` items routed; 0 markers remain | **PARTIAL** | 12/12 cycle-open (2026-07-19) items routed — roadmap "Backlog routing record" table + PRD §Routing. BUT the literal auditable metric fails: `grep -c '· new' BACKLOG.md` = **15**, all mid-cycle accretions (origins overnight-20260721 →  T-20). Cohort-intent met; letter-of-KR not. Attended interpretation call. |

### Delivery

| KR (abbrev.) | Status | Evidence |
|---|---|---|
| D1 · Design signed BEFORE any Feature build WO | **MET** | Design signed 2026-07-20 (ledger T-18); Feature-track cards (F1/F3/F5…) dispatched overnight-20260720c and later — 0 build WOs ahead of the signature (ledger timestamps) |
| D2 · Both tabs live on PROD, 0 version drift, 2/2 screenshot-verified | **PENDING-deploy** | `task version` observed 2026-07-23 (§5 verbatim): local = dev = prod at backend 0.1.3 / frontend 1.0.3, prod `git_sha: unknown`. The matching semvers are NOT parity evidence — the bump is deferred to `/save-project` at deploy, and prod predates this cycle's merges. The ship is attended. |
| D3 · Per-card wall-clock 100% of build cards + median vs T-14 baseline | **MET** | 4/4 build runs this cycle carry measured per-card tables (20260720c, 20260721, 20260722 in `reference/card-actuals.md`; tonight in `timings.log`). Median computed in §4: cycle median **94m** (N=12) vs baseline **22m28s** (N=23). |
| D4 · Prod-alert-dup: 0 duplicate Cliq alerts observed OR one side disabled | **PENDING-attended-confirmation** | Card `prod-alert-dup-guard` still PLANNED. Ledger: "FR-12 Cliq-dup watch continues" (T-19/T-20 standing flag). No duplicate incident is recorded — but "0 duplicates over the cycle" needs the operator's cycle-end confirmation of the watched channel; not computable from files. |

### Engineering

| KR (abbrev.) | Status | Evidence |
|---|---|---|
| E1 · Trends aggregation correctness (red-first Go test) | **MET** | `trends-spend-by-group-endpoint` DONE; `backend/internal/inventory/trends_test.go` asserts every week×group cell vs SUM + the reconciliation identity against `PeriodSummaryHandler` itself (never a constant), on a fixture carrying all five G6 breakers |
| E2 · Cost margin/food-cost-% to the cent + movers ordering | **MET** (open note) | `cost-margin-endpoint` DONE 2026-07-22 (triaged T-19, decision 33): red-first Go test vs hand-computed fixture. Open note: 0%-food-cost anomaly left as an open investigation (not a correctness red on the fixture). |
| E3 · Gate server-enforced: no-grant → 403 + no tab; 0 bypass paths | **MET** | F5 (merged `c1a2393`): 403 `missing_grant`, umbrella semantics, 13 attack variants, fail-closed; + G1 (merged `4bb8649`, G6 APPROVE): parity across 11/11 slugs (7 enforced, 4 placeholders N/A-with-reason), `tests/grant-enforcement-parity.spec.js`. Riders open at triage: G1-a `/photos/*`, G1-b items cross-app READ (DECISIONS-NEEDED.md). |
| E4 · Convergence matrix systematic; 0 cells red at cycle end | **MET** | `convergence-matrix-systematic` DONE (merged `8249209`): 32 cells = 24 covered / 8 N/A-with-reason, 13 new `MTX-*`, 65/65 ×3 fresh-DB `--retries=0` + independent G6 re-run |
| E5 · Literal `task test` exit-0; waiver #1 formally retired | **PARTIAL** (downgraded by §3) | Waiver #1 retirement stands (`waiver1-isolation-fix` merged `24358f8`, exit-0 473·0·6; S1 merged `5eb4331`, quiet-box 541/0/6 at `--retries=0`). But §1's leg tonight went 540/1/6 at 0 retries under load — LST-17 red, isolation-green. See §3. |

### QA

| KR (abbrev.) | Status | Evidence |
|---|---|---|
| Q1 · `sync` pkg 0 → covered; escaped defect has red-first unit vs pre-fix code | **MET** | `sync-pkg-unit-coverage` (merged `38f2060`): `backend/internal/sync/access_test.go`, 10-combo cartesian + dedup/negative coverage; ESC-1 regression proven red on pre-fix code; superadmin N/A per `users_roles_check` |
| Q2 · 100% of fix-WOs carry red-run evidence | **MET** (labeled caveat) | Every fix-classified WO this cycle records a pre-fix red: G1 red commit `535a37c` (7 failed) independently re-verified by G6 at the red commit; S1 `:1198` + `:525` proven red captures; prior fix cards per `card-actuals.md`/run HANDOFFs. Caveat kept honest: S1's LST-17/GATE-04 are HARDENED (not reproduced in 3 targeted contention legs each) — labeled hardening, not fixes, so they don't dilute the fix-WO denominator. |
| Q3 · Trends/Cost states specs, all rows screenshot-verified | **MET** | `tests/states-trends.spec.js` + `tests/states-cost.spec.js` shipped (F3/F4); PNGs read back per card records — F3's read-back caught 3 defects invisible in code (mid-number wrap at 390px, flattening axis, `$0.00` on empty weeks) |
| Q4 · Down-migrations proven; pre-deploy backup per migration deploy | **N/A-no-schema-migrations** | Signed design chose Option (i): per-tab slugs via `SeedHQApps` — "NO migration, QA-KR4 down-migration clause is N/A" (roadmap F5 record). G1's change is route-group wiring in `backend/cmd/server/main.go`, not schema. 0 prod deploys occurred this cycle (see D2). |

**Tally (as reconciled by §3): 11 MET · 2 PARTIAL (P4, E5) · 2 PENDING (D2, D4) · 1 N/A (Q4).**

## §3 — §1 result reconciliation (downgrade in the open)

Per the 07-19 gate pattern ("let §1's real result reconcile the pre-computed scorecard"):
§1 was NOT green at 0 retries, so **E5 is downgraded MET → PARTIAL** here, in the open.

What stands: waiver #1's formal retirement (A2: literal `task test` exit-0, 473·0·6,
independently reproduced) and S1's quiet-box determinism evidence (12/12 contention legs +
5/5 measured-quiet legs at loads 1.59–1.93 + fresh-DB full-suite `--retries=0` 541/0/6).
What tonight contradicts: determinism **under load** — LST-17 went red once at 0 retries on
a box at load 2.4→4.4, then greened in isolation. Per the standing rule, "rare, mechanism
known" is not laundered into "not flaky": LST-17 remains a load-sensitive cell, exactly as
S1's HARDENED-not-killed label said. Whether E5's "deterministic stack" reading is satisfied
by the quiet-box proof (S1's) or requires green-under-any-load is the operator's call.

**Reconciled tally: 11 MET · 2 PARTIAL (P4, E5) · 2 PENDING (D2, D4) · 1 N/A (Q4).**

## §4 Delivery KR3 — median WO cycle vs the T-14 baseline

**Tonight's per-card actuals** (epoch-stamped `timings.log`, impl + G6 + merge):

| Card | Impl | G6 | Merge | Card cycle |
|---|---|---|---|---|
| G1 `grant-enforcement-parity` | 71m54s | 22m11s | 1m05s | **95m10s** |
| S1 `syncspec-deflake` | 236m04s | 36m00s | 0m48s | **272m52s** |

**Tonight's N = 2** — far too small for a median on its own (the "median" of two values is
their midpoint, 184m01s, and means little). Stated, not extrapolated.

**This cycle's running set** (all measured, merged, G6-gated build cards since cycle open
2026-07-19, from `reference/card-actuals.md` + tonight):

| Card (run) | Cycle |
|---|---|
| B1 sync-pkg-unit-coverage (20260721) | ~13m |
| C1 prove-surface-design-draft (20260721) | ~13m |
| F1 trends-endpoint (20260720c) | ~29m † |
| F2 cost-margin-endpoint (20260722) | ~34m |
| F3 trends-tab-frontend (20260720c) | ~73m |
| S1 replay-fetchstorm-gate (20260722) | ~93m |
| G1 grant-enforcement-parity (tonight) | ~95m |
| A1 convergence-matrix-systematic (20260721) | ~111m |
| F5 inventory-tab-gating (20260720c) | ~124m |
| F4 cost-tab-frontend (20260722) | ~129m |
| A2 waiver1-isolation-fix (20260721) | ~146m |
| S1 syncspec-deflake (tonight) | ~273m |

**Cycle median = (93 + 95)/2 ≈ 94m, N = 12** — vs **T-14 baseline 22m28s, N = 23**.

- † F1's ledger "Cycle" figure (29m) is internally inconsistent with its leg sum
  (~58m incl. revision); the ledger column is used as recorded. The median is insensitive
  either way (swapping 29→58 leaves the two middle values 93/95 unchanged).
- Excluded, with reasons: 20260720c Wave 0 (1m — XS direct commit, no G6 leg, not a WO
  cycle); 20260720c follow-up sweep (84m — attended, post-closeout); parked cards
  F1-20260722 (~21m) and D1-20260720c (~181m) — parked work is not a completed WO cycle.
- **Honest comparison:** the ~4.2× jump over baseline is dominated by a population shift,
  not like-for-like slowdown. The T-14 N=23 baseline was the prior cycle's docs/audit/prove
  population (median-class cards 8–33m); this cycle's population is app-code, authz-parity,
  XL test-matrix and de-flake-proof classes, priced separately in `card-actuals.md`
  (G6 repriced 15–45m; de-flake proof ~60–90m+). The number is computed and real; what it
  *means* for the KR is an attended judgment.

## §5 Delivery KR2 — prod parity (observed, PENDING-deploy)

`task version` output (2026-07-23, read-only; dev server on :8080 WAS running):

```
── Local source ──────────────────────────────────────
  Frontend (package.json):  1.0.3
  Frontend (version.json):  1.0.3
  Backend  (version.go):    0.1.3

── Dev server (localhost:8080) ─────────────────────────
{ "backend_version": "0.1.3", "built_at": "unknown", "frontend_version": "1.0.3", "git_sha": "unknown", "status": "ok" }

── Production (https://hq.yumyums.kitchen) ─────────────────────────
{ "backend_version": "0.1.3", "built_at": "unknown", "frontend_version": "1.0.3", "git_sha": "unknown", "status": "ok" }
```

All three columns match at backend 0.1.3 / frontend 1.0.3 — **this is NOT parity evidence
for KR D2.** The semver constants are deliberately untouched until `/save-project` runs at
deploy time (HANDOFF standing flag: "Prod deploy NOT done… Frontend semver untouched"), so
prod's 1.0.3 build predates every merge of this cycle; `git_sha: unknown` means the running
prod commit cannot be attested from here. The expected post-ship state is local AHEAD of
prod after the bump, then re-converged by `task prod:deploy`. **Status: PENDING-deploy**
(the ship is attended). 2/2 prod tab screenshots are likewise pending the ship.

## §6 What remains for the attended close-out

| Open row | What would settle it |
|---|---|
| **P4** (PARTIAL) | Operator interpretation: cycle-open-cohort reading (met) vs literal-grep reading (15 markers remain — all mid-cycle accretions that are next `/nc-okr-session` feedstock). Either grade it or route the 15 at the boundary round. |
| **D2** (PENDING-deploy) | `/save-project` (semver bump) → `task prod:deploy` → `task version` shows prod == local → 2/2 tab screenshots behind the gate on `https://hq.yumyums.kitchen`. |
| **D4** (PENDING-attended-confirmation) | Operator confirms the Cliq channel showed 0 duplicate alerts over the cycle (or disables one side and records it); ledger entry closes FR-12 watch. |
| **E2 open note** | 0%-food-cost anomaly — open investigation, not gating; carry or route. |
| **G1-a / G1-b** (riders, not KRs) | DECISIONS-NEEDED.md: `/photos/*` grant mapping; ratify-or-revert `/inventory/items` cross-app READ. |
| **§1 attestation / E5** (PARTIAL) | The no-retry gate read 540/1/6; the red is LST-17 (S1's HARDENED-not-killed cell), isolation-green, on a loaded box. Options: waive-with-reason (known cell, honestly labeled) or re-leg on a measured-quiet box before granting the suite-green attestation. Attended. |
| **The milestone close itself** | Attended: merge `overnight-20260724` → `dev`, grade this scorecard, the prod ship, `/nc-milestone-close` boundary marks. This document decides none of it. |
