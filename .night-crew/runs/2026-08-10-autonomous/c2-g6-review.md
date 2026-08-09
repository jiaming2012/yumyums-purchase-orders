# G6 adversarial review — Card 2 `sync-live-in-dev-app-proof` (run 20260810)

**Verdict: PASS → MERGE.** G6 independently re-ran the harness (the gold standard) and
reproduced the exact red-first asymmetry the card claims.

Fresh-context reviewer; inputs were the card slate entry, the diff, the evidence claims,
and the gate ladder — not the implementer's reasoning.

## RF non-vacuity (THE critical claim) — VERIFIED-REAL, re-run by G6
Independent run `ap20260809154717`, `SCRIPT_EXIT=0`, both legs in one invocation:
- **RED (carrier DOWN): spec exit 1.** App opened replication (all four collections `[sync 200]`
  through the proxy, exact draft filter), sat at `data-state="waiting"` the **full 20.5s** (never
  `served`). The red waits the real bound — not "too fast."
- **ARMED (carrier UP): spec exit 0** in 454ms, `served` carrying the exact `/saveResponse`
  sentinel, read from `db.responses`.
- **Non-vacuous by construction:** the `served` transition is driven by a live RxDB subscription
  (`handle.db.responses.findOne({field_id}).$.subscribe(...)` in `workflows.html:4102-4108`) that
  only fires on a real replicated doc; the spec asserts the on-screen value contains the sentinel.
  No fixture, no pre-seeded row (baseline appproof-count asserted 0). The harness's own vacuity
  trap (`RED_RC -eq 0 → cannot_run`) treats a red-that-passes as could-not-run, never green.

## done_when: MET. Real `workflows.html` (`hq_sync_read=on`, no stub) against the live substrate,
one field via real `/saveResponse` (204), surfaces in-app, SAME spec fails carrier-down. Gated on
the SCRIPT exit, not `task` (B-163).

## Gates
- **RF GREEN** (reproduced). **G2(PW) N/A-by-footprint** (no seam key; standalone harness exit IS
  the verdict). **G1/G2(Go)/G4 N/A-by-footprint** (no Go, no HTML/JS, precache 31).

## Guards & footprint
- **night-crew.toml comment-only** — every added line `#`-prefixed (footprint NOTE, no key/token →
  no park). `tests/repo-hygiene.spec.js` untouched, count stays 11.
- **Footprint honest** — exactly 7 files; zero `workflows.html`/`sync-rxdb/*`/Go/`tests/*.spec.js`
  edits (spike prediction held).
- **B-164** — scratch HQ on an ephemeral port (refused 5432/5433/5434 before+after); substrate
  reconciled (never `spike:down`); restore byte-identical (0 appproof rows, FDW back to
  `:5434/hq_test_b2_fdw`, no scratch-HQ leak).
- **Roadmap flip ACCURATE** (leg 3 → DEV-COMPLETE; Card 1's entry not corrupted).

## Issues (both nits, non-blocking)
- merge-intent undersells the promotion as a "rename" — it is actually a strengthening (spike-f
  keyed red on a *different* assertion `absent`; the card collapsed to ONE always-`served` test
  achieving the red purely by withholding the carrier, which is what done_when demands).
- throwaway scratch-container dev creds (banner-documented; not a real secret leak).

## Merge recommendation: MERGE — contract fully met, RF reproduced fresh, substrate byte-identical.
