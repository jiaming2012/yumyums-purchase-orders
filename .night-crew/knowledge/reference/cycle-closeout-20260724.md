# Cycle closeout — "Prove & surface" (`cycle-gate`, attended boundary close)

> **Cycle:** Prove & surface (opened 2026-07-19, closed **2026-07-24**).
> **Runs in window:** overnight-20260720c · overnight-20260721 · overnight-20260722 ·
> overnight-20260724 (all merged to `dev`, all triaged — T-18/T-19/T-20/T-21 series).
> **Computable legs:** `runs/2026-07-24-autonomous/scorecard-20260724.md` (ST stretch card).
> **Attended settlements:** ledger T-21 → T-21f (decisions 42–48).
> **Shipped to prod:** backend 0.2.2 / frontend 1.1.0 at `32afb39`
> (releases 4219fa4 → a206daa → 32afb39, all 2026-07-23/24), parity verified, 2/2 tabs
> screenshot-verified by the operator on a returning client.
> **Close performed hand-authored:** the milestone CLI verbs (grade/audit/export/mark) cannot
> read hand-run-slate history (no metrics.jsonl) — filed as a night-crew backlog item at the
> transfer. Boundary marker written CLI-schema-compatible by hand.

## Final grades — 15 MET · 1 N/A (16 KRs)

| Team | KR | Grade | Deciding evidence |
|---|---|---|---|
| Product | P1 PRD gate | **MET** | signed 2026-07-19 before any build WO (ledger timestamps) |
| Product | P2 escaped-defect cells | **MET** | 3/3 defects red on pre-fix build (merged `8249209`) |
| Product | P3 gating decision | **MET** | design §8 signed; enforced by F5+G1 |
| Product | P4 backlog routing | **MET** (decision 47) | 12/12 cycle-open items routed; 15 accretions → next round |
| Delivery | D1 design-before-build | **MET** | T-18 signature precedes all feature dispatches |
| Delivery | D2 prod parity + tabs | **MET** (T-21e) | `task version` parity at `32afb39`; 2/2 operator screenshots |
| Delivery | D3 per-card timing | **MET** | 100% measured; median 94m (N=12) vs baseline 22m28s (N=23), population shift stated |
| Delivery | D4 alert-dup | **MET** (decision 46) | last dup 2026-07-21 (fixture-named, root-caused); ALERTS_ENABLED gate live in prod |
| Eng | E1 trends correctness | **MET** | red-first fixture test + reconciliation identity |
| Eng | E2 cost margin | **MET** | fixture-proven to the cent; 0% note routed (decision 48) |
| Eng | E3 gate enforced | **MET** | 403+missing_grant, fail-closed, G6 attack runs held, 11/11-slug parity spec |
| Eng | E4 convergence matrix | **MET** | 32/32 enumerated cells green; **T-21c: a 33rd un-enumerated cell (cross-user × cross-cycle-state) found by operator play-test on close day — backlogged with a product question; input to the next matrix, not a red cell of this one** |
| Eng | E5 no-retry exit-0 | **MET with waiver** (decision 44) | 4× zero-retry green (473/0/6, 2× 541/0/6, attended 542/0/6) vs 1× 540/1/6 under measured foreign load; LST-17 remains flagged load-sensitive — not laundered |
| QA | Q1 sync pkg coverage | **MET** | ESC-1 red on pre-fix code; cartesian coverage |
| QA | Q2 fix-WO red-runs | **MET** | 100%, G6 re-verified at the red commits |
| QA | Q3 states specs | **MET** | 2/2 specs; PNG read-backs caught 3 real rendering defects |
| QA | Q4 down-migrations | **N/A** | signed design chose no-migration; 0 migration deploys |

Four grades rest on operator rulings (P4 interpretation, E5 waiver, D4 channel watch, E4
new-cell disposition) — each recorded with its reasoning in the T-21 series; the rest are
computed or artifact-backed.

## What the boundary week also produced (close-day findings, all recorded)

- **T-21a / decision 45:** gated-tab umbrella semantics reversed by play-test (granular
  overrides umbrella) — backlogged, not urgent.
- **T-21b / decision 46:** alert delivery made opt-in (`ALERTS_ENABLED=1`, prod-only) after
  the dup root-cause; red-first tested, deployed, `delivery_enabled=true` confirmed in prod logs.
- **T-21c:** cross-user hydration divergence reproduced and evidenced
  (`reference/sync-crossuser-hydration-20260724.md`); product ruling queued.
- **T-21d:** prod SW-update pipeline defect (`version.json` 404 aborted every service-worker
  install since 2026-07-05) found by the operator's returning-client check, fixed (0.2.2),
  verified end-to-end. Standing lesson: prod-parity evidence requires a returning-client check.
- **README rewritten** for the current system and methodology.

## Carried forward (the next planning round's inputs)

- BACKLOG.md `· new` items (≈20): incl. umbrella-semantics reversal, photos key-binding gate,
  cross-user hydration cell (+ product ruling), sync.js fetch-storm gate (re-arms two-device
  check), 0%-food-cost investigation, onboarding second-run idempotence.
- **Standing flags:** attended two-device convergence check ARMED (carried from 07-22;
  unexercised — the operator's prod check covered tabs, not two-device edit convergence);
  cross-contamination surface #4 (one Postgres cluster, one credential pair) OPEN;
  `stash@{0}` unattributed WIP.
- OKR-authoring lesson (from P4): write KR metrics that cannot be failed by desirable
  behavior (capturing new backlog items should never redden a routing KR).

**Boundary marked:** `scorecard/milestones.jsonl` `hq-20260724` (last_run 20260724).
**Next step:** `/nc-roadmap-round` consumes this document, the backlog, and the T-21 series.
