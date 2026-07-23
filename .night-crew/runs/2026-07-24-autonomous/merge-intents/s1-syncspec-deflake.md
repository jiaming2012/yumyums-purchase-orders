# Merge intent — S1 `syncspec-deflake` (card/s1-syncspec-deflake)

## Shared files touched

- `tests/sync.spec.js` — de-flake `:1198` (survivalCell settle + POST-observed
  re-dispatch), `:525` FLD-LIVE-02 order-dependence fix, LST-17 (both variants)
  hardening. S1 owns this file per the slate footprint.
- `tests/workflows.spec.js` — GATE-04 hardening only. S1 owns GATE-04 per the
  slate footprint; G1's additive `beforeAll` app-grant baselines are already in
  the tree S1 branched from.
- Production files (`sync.js`, `workflows.html`): **nothing here** — no
  production edit planned. If a determinism seam becomes unavoidable, this
  section will be updated in the same commit as the touch.
- `backend/`: **nothing here** — forbidden for this card.

## What must survive any merge

- G1's `beforeAll` app-grant baselines in `tests/sync.spec.js` and
  `tests/workflows.spec.js` (this branch was cut AFTER G1 merged and builds on
  them verbatim).
- The `:1198` line anchor: the temperature-convergence test's declaration is
  kept on line 1198 of `tests/sync.spec.js` (backlog/slates/cycle-gate reference
  it as `sync.spec.js:1198`); edits above it are line-balanced to preserve this.
- The survivalCell settle/re-dispatch helpers and per-test hardening added by
  this card.

## Safe to drop

- Nothing here — all changes are test-side de-flake work; dropping any of it
  re-opens the named flakes.
