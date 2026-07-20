# HANDOFF — overnight-20260721 (for the morning of 2026-07-21)

> **Run branch:** `overnight-20260721` (cut from `dev` at `7de74ff`; **never pushed, `main` untouched**).
> **Slate:** `.night-crew/knowledge/reference/slate-20260721.md` (batch-signed 2026-07-20).
> **Scope:** Activity 3 — Trust track (3 build cards) + the draft-only Activity-2 design card.
> Serial dispatch A1 → B1 → C1 → A2, per-card worktree + fresh implementer subagent + separate
> fresh G6 adversarial subagent, one ephemeral pg16 env at a time.
> **Result: 4/4 cards DONE, 4/4 G6 PASS, 0 cards PARKed, 0 cells PARKed, 0 footprint breaches.**
> **Waiver #1 formally retired** (literal `task test` exit-0, achieved + independently reproduced).
> Reader = the operator; resolve via `/nc-morning-triage`.

## TL;DR

- **A1 `convergence-matrix-systematic`** — the convergence E2E matrix is now systematic: 32 nominal
  cells → **24 covered (11 pre-existing + 13 new `MTX-*` tests) + 8 N/A with source-verified
  reasons** (the N/A table is embedded as the matrix header comment in `tests/sync.spec.js`).
  ESC-1/2a/2b/3 each map to a named would-have-caught cell (A6 historical red-first). Determinism:
  65/65 ×3 fresh-DB `--retries=0` (impl) + 65/65 independent G6 re-run. Test-only — **no
  determinism seam was needed**, so no `task sw` all night.
- **B1 `sync-pkg-unit-coverage`** — `ResolveEntityAccess` asserted across the full
  {role}×{assignment} cartesian (10 combos, exact-set equality, per-subtest truncated DB, negative
  asserts). `superadmin` rows are honestly N/A: `users_roles_check` forbids the value in the DB;
  config superadmins upsert as `'admin'`. ESC-1 regression byte-identical/unweakened. G6 ran an
  adversarial mutation to prove the assertions bite. One contract question surfaced (approvers in
  the fan-out — see DECISIONS-NEEDED §B4).
- **C1 `prove-surface-design-draft`** — the Activity-2 design draft landed at
  `.night-crew/knowledge/designs/prove-surface-gating-and-endpoints.md` (561 lines, editprop-
  precedent register). **A4 is OPEN** — both representations worked end-to-end, advisory
  recommendation = Option (i) two slugs, **the sign-off decides**. The draft's §4 found 6 real
  PRD-vs-code gaps, including a new sub-decision **D2** (linked-but-groupless sentinel lines).
  G6 verified 14 code citations line-exact and cell-for-cell fidelity to the landed matrix.
  **The design sign-off is still the attended gate — Activity 4 stays blocked until you sign.**
- **A2 `waiver1-isolation-fix`** — LST-08 root-caused: **not** a leftover approval row but the
  shared-DB **ops-journal replay fetch storm** (fresh context → Lamport 0 → `wsCatchUp` replays the
  whole journal; the `SUBMIT_CHECKLIST` branch fires an **ungated** `loadMyChecklists()` per op,
  `sync.js:443`; a stale snapshot clobbers an optimistic checkbox mid-fill → `confirm()`
  auto-dismissed → no toast). Fixed test-side (LST-08 + same-mechanism siblings RUN-07/APR-09;
  assertions strengthened, none weakened). **Literal `task test` exit-0: impl 473·0·6 exit `0`;
  G6 independent run exit `0` verbatim (473·0·6, 19.1m).** Waiver #1 (the LST-08 pollution red) is
  formally retired — Eng KR5 PARTIAL → PASS. **Honesty caveat:** G6's first full run hit a
  *successor* intermittent (`sync.spec.js:1198`, pre-existing class, out of footprint) — exit-0 is
  **achieved-and-reproduced, not asserted deterministic**. Fork in DECISIONS-NEEDED §B2.

## Per-card wall-clock (harness-measured, FR-11 / Delivery KR3 — the standing output)

| Card | Impl | G6 | Merge+flip | Card total |
|---|---|---|---|---|
| A1 `convergence-matrix-systematic` (XL) | **101m15s** ⚠ | **9m15s** | ~1m | **~111m** |
| B1 `sync-pkg-unit-coverage` (S–M) | **8m11s** | **4m27s** | <1m | **~13m** |
| C1 `prove-surface-design-draft` (M) | **7m30s** | **4m49s** | <1m | **~13m** |
| A2 `waiver1-isolation-fix` (S + exit-0 proof) | **88m00s** | **57m34s** | <1m | **~146m** |

Timestamps: `timings.log` (this directory), epoch-stamped at each leg boundary.
- ⚠ A1's impl leg includes **~25–30m of orchestration stall**: the implementer twice suspended
  itself by backgrounding its suite runs and had to be resumed (a run-mechanics defect, fixed in
  every later brief — "never background; foreground legs ≤10m, detach+`tail --pid` for longer").
  Its productive wall was ~70–75m — inside the slate's 70–90m estimate.
- A2's G6 is the deliberate outlier: the exit-0 headline got the full independent treatment
  (2 complete `task test` runs, 18.8m + 19.1m, plus both red-pair legs). That spend bought the
  successor-intermittent discovery (§B2) — worth it.
- Slate estimate vs actual: serial estimate ~175m + 30m closeout; actual card time ~283m,
  dominated by A1's stall overhead and A2's double-suite G6. Run total 08:27 → 13:10 (~4h43m).

## Gate evidence (on the final tree)

- **Footprint:** `git diff dev..overnight-20260721` touches exactly: `tests/sync.spec.js` (+635/−20),
  `backend/internal/sync/access_test.go` (+303/−0), `tests/workflows.spec.js` (+63/−25),
  `designs/prove-surface-gating-and-endpoints.md` (new, docs), `roadmap.md`, `timings.log`.
  **Zero production files. No `task sw` (no HTML/JS asset changed; A2's `task test` regenerated
  sw.js twice with zero diff — content-hash identical).**
- **G1** `go build ./... && go vet ./...` — green on the final merged tree (re-run at closeout).
- **G2/Go** — full `go test -count=1 -p 1 ./...` exit 0 on a pre-migrated ephemeral pg16 (B1 impl +
  B1 G6, code tree identical to final); sync tests skip cleanly without a DB.
- **Suite** — literal `task test` exit `0` on the settled tree: **473 passed · 0 failed · 0 flaky ·
  6 documented expected-skips** (impl run + G6 independent run 2; G6 run 1 = the §B2 intermittent).
  Convergence set (sync + repro-cut-task + broadcast-rerender): 65/65 ×4 fresh-DB `--retries=0`
  runs across impl + G6.
- **G4/replay/openspec** — N/A for HQ (per 07-19 triage precedent; no openspec/ in this repo).
- **Ephemeral-env discipline:** host `:5432` never touched; all legs on Docker-assigned loopback
  ports (compose projects `nc-a1/b1/a2`, `nc-g6a1/b1/a2`), all torn down `--volumes`.

## Commits on `overnight-20260721` (12 + closeout)

Per card: impl commit(s) → `--no-ff` merge with G6 evidence in the body → roadmap flip commit.
`c7b4ccd`/`8249209`/`4cb9559` (A1) · `0ebc81d`/`38f2060`/`2e42af5` (B1) ·
`08e81e1`/`3d5fc17`/`4a60449` (C1) · `544e68b`+`08c1bef`/`24358f8`/`96e56b2` (A2) · + this closeout.

## For the morning reader (triage order)

1. **Merge `overnight-20260721` → `dev`** (`--no-ff`). Test-only + docs; no production surface.
   `main` untouched, branch never pushed.
2. **Sign (or amend) the Activity-2 design** — `designs/prove-surface-gating-and-endpoints.md`.
   Your calls: **A4** (two slugs vs sub-permission column; draft recommends two slugs) + **D2**
   (Ungrouped pseudo-group for linked-but-groupless sentinel lines) + three LOW sign-off riders
   (DECISIONS-NEEDED §B1). Signing unblocks the 5 Activity-4 Feature WOs.
3. **Decide the successor-intermittent fork** (§B2): `sync.spec.js:1198` reddened 1-of-2 G6 full
   runs. Waiver #1 is retired; whether a new hardening card (or waiver) opens for this test is
   yours. Recommendation: graduate a card next cycle, paired with §B3's production fix — the same
   fetch-storm class likely underlies both.
4. **Route the two backlog candidates** (§B3 production `SUBMIT_CHECKLIST` replay gate — impl+G6
   both endorse, G6 urges upgrading; §B4 approver-fan-out contract; §B5 approve/reject authz gap).
5. **FR-12 Cliq-dup watch** continues over the cycle (not a tonight item; nothing observed tonight).

**Standing flags:** prod-deploy/attended-convergence flags untouched (no verify/prod/DB path
changed). Frontend semver untouched (no asset change). DB flag satisfied (ephemeral pg16 canonical).
`percard-timing-instrumentation` roadmap card: this run's table above IS the standing output —
consider flipping it DONE at triage (it's a run-mechanic, not a code card; the orchestrator left
it for your call since the card spans the whole cycle).

## Triage disposition (2026-07-20) — merged, all 5 forks resolved, design SIGNED

Merged to `dev` `--no-ff` (`e1d22ad`) after attended review: build+vet green, `go test` all
packages ok on branch + merged tree (DB-backed sync tests skip without pg; the run's 2× ephemeral
pg16 legs are the DB evidence), G4/replay checks N/A for HQ, footprint = test-only + docs, zero
production files. Recorded as `ledger.md` §T-18. `dev` pushed to `origin/dev`.

**Resolutions:** B1 design SIGNED (A4 = Option (i) · D2 = Ungrouped · rider (b) → umbrella
semantics · rider (c) expected) → **Activity 4 UNBLOCKED**. B2+B3 → `replay-fetchstorm-gate`
card promoted (Activity 3). B4 → "everyone with entity access sees live ops" ratified. B5 →
folded into `inventory-tab-gating`. `percard-timing-instrumentation` → DONE. Preference
capture/decisions audit skipped — not deployed to night-crew `main` (NF-3).

**Standing flags after triage:** prod-deploy / attended-convergence flags **satisfied** — this run
touched no verify/prod/DB path; the convergence flag **re-arms when `replay-fetchstorm-gate`
lands** (it changes `sync.js`, the live-sync path under the flag). DB flag satisfied (ephemeral
pg16 canonical). Frontend semver untouched. `main` untouched; dev → main promotion separate.
FR-12 Cliq-dup watch continues.

**Cards unblocked:** all 5 Activity-4 Feature WOs (`inventory-tab-gating` first — endpoints mount
inside its groups) + `replay-fetchstorm-gate` (Activity 3). Next move: `/nc-slate-plan` — the
queue is decided; no open forks remain.
