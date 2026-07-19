# HANDOFF — overnight-20260718 (for the morning of 2026-07-18)

> **Run branch:** `overnight-20260718` (cut from `dev`; **never pushed, `main` untouched**).
> **Slate:** `.night-crew/knowledge/reference/slate-20260718.md` (batch-signed 2026-07-17).
> **Scope:** single card — `editprop-convergence-cell-hardening` (Activity 6, test-debt
> retirement, final card). Serial by definition (one card, one env).
> **Result:** **1/1 card DONE, G6-verified, 0 footprint breaches.** Half 1 (de-flake) LANDED
> test-only; Half 2 (conflict coverage) LANDED 4 of 6 types, **2 fail-note types PARKED as
> footprint-blocked** (see DECISIONS-NEEDED.md — a bounded coverage residual, not a fork).
> **`sync.js` UNTOUCHED** → no `task sw` was needed; no production behavior changed.
> Reader = the operator; resolve via `/nc-morning-triage`.

## TL;DR

The last un-built card in Activity 6 landed. It retires the operator rider *"no card may lean on
this suite as a no-retry hard gate until this lands"* — the two-device convergence suite is now
demonstrably **zero-flake under no-retry** (proof below), which unblocks `cycle-gate` (Activity 8,
attended) from adopting `task test` exit 0 on the deterministic stack as a hard gate.

- **Half 1 — zero-flake de-flake (LANDED, test-only).** The `text answer converges` and
  `temperature answer converges` two-device cells (in the W-3 "surviving answers converge" describe)
  flaked ~3/6 under `--retries=0`. **Root cause found empirically was NOT the card's assumed
  live-convergence race** — it was a stray WS-catch-up `loadMyChecklists` re-render clobbering the
  freshly-typed, **not-yet-persisted** text/temperature input to empty at the *baseline* assert
  (observed `Received ""`), with nothing re-issuing a fetch to restore it. Fixed entirely inside the
  `survivalCell` helper with deterministic waits (NOT widened timeouts):
  1. gate the edit on the autosave `POST /ops` **2xx** — a race-free durability signal
     (`SaveResponseFunc` commits before the 200), replacing a myChecklists poll that itself timed
     out under load;
  2. reopen once at baseline so it hydrates the **committed** draft (immune to the optimistic clobber);
  3. wait on the post-cut myChecklists GET (the `rerenderOpenChecklistAfterSave` re-fetch) as the
     "SAVE_TEMPLATE applied" signal before asserting.
  The authoritative gate stays the LIVE (no-reload) `.fill-field 'Decoy'` → count 0 assertion plus
  the CATCH-UP (reopen) assertion — **both preserved and unweakened** (G6-confirmed).

- **Half 2 — conflict-branch coverage (LANDED, 4 new types).** A new `W-6b` describe parameterizes
  the LWW-409 / `applyOp` render path (previously proven only for text) over **yes_no, temperature,
  sub-step, checkbox** — each a red→green cell: WS stubbed dead + Lamport pinned stale on the loser,
  winner seeded via `POST ops` SET_FIELD at lamport 5000000, loser writes a distinct value,
  `waitLoser409` hard-gates on a real 409, and the final assertion reads the **rendered winner**
  (never the loser's rejected value). Checkbox needed a distinct winner user + strict write order
  (per-field shared entity lamport; a null uncheck DELETEs the row before the LWW guard) — handled.
  **2 types PARKED as footprint-blocked** (fail-note text+severity, fail-note photo-URL): the
  `{_v,_fail_note}` bundle is unpacked ONLY by `hydrateFieldState` (workflows.html), NOT by the
  `applyOp` SET_FIELD branch the 409 handler drives (`sync.js:405` — grep confirms no `_fail_note`
  unpack there), so covering them requires an out-of-footprint production change. See
  DECISIONS-NEEDED.md. Half 2 is independent of Half 1 and landed regardless, per the slate.

## Gate evidence (on the final merged tree)

- **Footprint:** clean. `git diff dev..overnight-20260718` touches only `tests/sync.spec.js`
  (+303 / −21) and `.night-crew/knowledge/roadmap.md` (card flip). `sync.js` diff is **empty**
  (verified on both the impl commit and the merged tree) → **no `task sw`**.
- **Half 1 no-retry streak (the load-bearing de-flake proof, `--retries=0`):**
  - Implementer: **10/10** isolated target-cell runs, **+ 0 target-cell (text/temperature)
    failures across 8 whole-describe runs under full-suite CPU load** — the exact condition where
    they previously failed ~3/6 (baseline flake reproduced before the fix).
  - **G6 independent re-run** (own ephemeral pg16, `--retries=0`): **5/5 text, 6/6 temperature,
    11/11 full W-3 describe.** (A single temp failure appeared only under G6's abnormal repeated
    single-cell reruns against a *non-reset* DB — commit-gate timing out on an autosave that never
    fired due to accumulated draft pollution; vanished on a proper reset, i.e. how the real harness
    runs. Not a genuine cell flake.)
- **Half 2:** implementer **12/12 green ×3** under `--retries=0`; **G6 independent 4/4 twice**.
  Each cell verified red→green real (would fail if the 409/applyOp render path were broken).
- **G6 verdict: PASS** on impl `14a36e8`. No new reds, no weakened assertions, no vacuous cells,
  no footprint breach. The 2 parks independently confirmed legitimate.
- **Known baseline untouched:** HQ still carries its ~37–41 pre-existing E2E reds. The pre-existing
  bespoke `unsubmit transition` cell flake (W-3 list-progress describe; a ~2min WS-broadcast hang,
  does NOT use `survivalCell`) is part of that baseline and was left alone per the card.

## Per-card wall-clock (harness-measured — fixes the 07-17 instrumentation-gap note)

| Card | Impl (measured) | G6 (measured) | Merge | Notes |
|---|---|---|---|---|
| `editprop-convergence-cell-hardening` | **~73 min** (4,387,969 ms) | **~16 min** (964,145 ms) | ~2 min | Impl wall dominated by the no-retry streak + under-load repro runs (this suite's "own size class" — proving zero-flake costs repeated Playwright wall-clock, not one red→green). G6 wall included ~fighting a stale foreign server on :8199 that `reuseExistingServer` latched onto — resolved by running against its own ephemeral env; the code under review was never the problem. |

> `card-actuals.md` basis for the convergence-hardening size class: **~73m impl / ~16m G6** measured
> here vs the slate's ~60–90m impl estimate — inside the band. The de-flake *proof* (repeat runs) is
> the non-compressible cost, as the slate predicted.

## Commits on `overnight-20260718`

- `14a36e8` — `test(sync): harden two-device convergence cells + extend W-6 conflict coverage`
  (impl; includes the roadmap card flip to DONE).
- `9f4b84d` — `merge: overnight-20260718 — editprop-convergence-cell-hardening (G6 PASS)`
  (`--no-ff`; G6 SHA + evidence appended to the card).
- (this closeout) — `docs(night-crew)`: HANDOFF + DECISIONS-NEEDED + card G6-evidence line.

## For the morning reader (triage)

1. Review this HANDOFF + `git diff dev..overnight-20260718 -- tests/sync.spec.js` (303 lines,
   one file of test logic). `sync.js` is untouched — nothing to re-verify on the production path.
2. Merge `overnight-20260718 → dev --no-ff` if satisfied. No `task sw` needed (no HTML/JS asset
   changed). Optionally cold-re-run the two target cells under `--retries=0` for your own streak.
3. **One open item in DECISIONS-NEEDED.md:** the 2 parked fail-note conflict-coverage types —
   NOT a blocker for this card (it's DONE and the rider is retired), but a bounded coverage residual
   that needs an operator call on whether it's worth an out-of-footprint follow-up. Details there.
4. The operator rider is retired → `cycle-gate` (Activity 8) is unblocked to adopt the no-retry
   hard gate. Activity 7 (prod ops) remains operator-gated/attended as before.

---

## Triage disposition (2026-07-18) — merged, forks resolved

> Resolved via `/nc-morning-triage`. Recorded in `ledger.md` §"Morning-triage resolutions
> (2026-07-18)". Merged `overnight-20260718 → dev --no-ff` (`6291ef2`); merged tree re-verified
> (spec parse + `go build`/`vet`/`test` green; `sync.js`/`workflows.html` diffs empty).

**Standing flags after triage:**
- **Rider CLEARED** — the no-retry hard-gate bar on the two-device convergence suite is discharged
  (the card landed; suite is zero-flake under `--retries=0`). `cycle-gate` (Activity 8) may now
  adopt `task test` exit-0 on the deterministic stack as a hard gate.
- **Activity 6 COMPLETE** — all test-debt-retirement cards DONE.
- Attended two-device convergence / `task sandbox:e2e` gate stays satisfied (this run touched no
  production/verify path); re-arms whenever the verify/merge path changes underneath it.
- DB flag satisfied (Docker pg16 canonical). Frontend semver unchanged (bump belongs to
  `/save-project` at deploy). `dev` pushed to `origin/dev`; `dev → main` promotion stays separate.

**D-1 (2 parked fail-note conflict types):** operator chose **accept + track in BACKLOG** (advisory
"Fail-note conflict live-render on the `applyOp`/409 path", bundle candidate with F-B) over
graduating a roadmap card now. Not a blocker; the card is DONE and the rider retired.
