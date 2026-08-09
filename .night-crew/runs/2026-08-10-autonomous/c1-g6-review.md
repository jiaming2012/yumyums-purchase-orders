# G6 adversarial review — Card 1 `sync-live-in-dev-substrate` (run 20260810)

**Verdict: PASS-WITH-ISSUES → MERGE-AFTER-FIX** (one trivial must-fix applied before merge).

Fresh-context reviewer; inputs were the card slate entry, the diff, the evidence claims,
and the gate ladder — not the implementer's reasoning.

## done_when (independently reproduced by G6)
- **Item 1 (door 503→200): VERIFIED-REAL.** Ran `sync-dev-proof.sh` → exit 0. `GET /sync/rest/`
  vars-UNSET → **503** (`sync_proxy_not_configured`); vars-SET → **200** (real PostgREST swagger).
  Both through the real running HQ server after a real login. Script asserts 503 first and refuses
  if the door is already open — non-vacuous.
- **Item 2 (relay carries write): VERIFIED-REAL.** `/saveResponse` → 204; row projected into the
  **substrate** carrying the sentinel in 226–269ms across G6's two runs (claim 267ms). Substrate
  restored byte-identical; scratch HQ torn down.

## Gates (independently re-run)
- **G1 PASS** — `go build`/`go vet` from `backend/` exit 0. The pgx BrokenImport LSP flag is a
  gopls-workspace artifact, not a real build failure.
- **G2(Go) PASS** — 9 pkgs `ok`, zero FAIL; counts real (`internal/workflow`=35,
  `internal/sync`=59 executed subtests, `TestSubstrateGate_ExitCodeAsymmetry` non-vacuous);
  `HQ_SYNC_SUBSTRATE_OPTIONAL` + `HQ_SYNC_GATE_CHILD` explicitly UNSET.
- **G2(PW) N/A-by-footprint** (no `.html`/`tests/*.spec.js`; no seam key matched).
- **G4 N/A-by-footprint** (no HTML/JS; `spikec_relay.go` change is doc-comment only; precache 31).
- **RF PASS** — merge-intent `## Red-first` carries the 503→200 capture; G6 reproduced the red.

## Guards
- **proxy.go:78 ACTIVATION-ORDER: VERIFIED** — 4 vars in the 5 dev targets only; `HQ_SYNC` absent
  from `docker-compose.prod.yml` / `prod`/`build` task env.
- **B-164: VERIFIED** — no live target at `:5433`; scratch HQ on an ephemeral port; substrate
  reconciled (never `spike:down`).

## Issues
- **must-fix (applied `167bc7e`):** `persistent-dev-fdw-pointing.sql:41` documented the override as
  `HQ_FDW_ALLOW_5433`; the guard reads `HQ_SYNC_DEV_ALLOW_5433`. Comment-only; fails safe (wrong
  name → guard still refuses) but misleads at the B-164 danger point. Code was already correct.
- **nits:** throwaway dev creds against ephemeral scratch (pre-existing posture); the `:5433` guard
  keys on the parsed port only (matches B-164's stated design + this box's topology).

## Roadmap flip: ACCURATE — Card 1 → DEV-COMPLETE (legs 1+2 + FDW persistence); Card 2 stays
PLANNED with the correct dependency; no leg-3 overclaim.
