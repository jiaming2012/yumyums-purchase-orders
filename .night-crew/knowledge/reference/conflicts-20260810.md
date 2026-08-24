# Conflict log — run 20260810

One entry per merge (clean or conflicted), per §15ad.66. A clean merge gets a
one-line "no conflict" entry so an empty log never reads as "no conflicts" when
it means "the logging never ran". Morning triage audits this file.

---

## Merge 1 — Card 1 `sync-live-in-dev-substrate` → `overnight-20260810`

- **Merge commit:** `bd03059` (`--no-ff`)
- **Card branch:** `wo-sync-live-in-dev-substrate` (tip `167bc7e`)
- **Cards involved:** Card 1 only — first card on the run branch; run branch had not
  moved since the card was cut (base `03c3e06`).
- **Files / hunks:** 9 files, **+1041 / −28**, all NEW or purely additive:
  - `Taskfile.yml` (+49) — `sync:dev:*` task family (new targets, no existing target edited)
  - `backend/Taskfile.yml` (+74) — 4× `HQ_SYNC_*` added to the 5 **dev** targets only
  - `.night-crew/qa/spike-supabase/sql/persistent-dev-fdw-pointing.sql` (new, +154)
  - `.night-crew/qa/spike-supabase/sync-dev-up.sh` (new, +254)
  - `.night-crew/qa/spike-supabase/sync-dev-proof.sh` (new, +381)
  - `backend/internal/sync/spikec_relay.go` (+16/−4, **doc-comment only**)
  - `.night-crew/knowledge/roadmap.md` (roadmap flip → DEV-COMPLETE)
  - `.gitignore` (+4, relay rundir), `merge-intent.md` (relocated to run artifacts)
- **Intents read:** only Card 1's `merge-intent.md` (no other card has landed). Shared
  surfaces it claimed: root `Taskfile.yml` + `backend/Taskfile.yml` — both additive, no
  existing target edited. **What Card 2 must preserve:** the `sync:dev:*` task family and
  the 4× `HQ_SYNC_*` dev-target wiring (Card 2 drives the live substrate they stand up).
- **Conflict?** **NONE — clean merge** (ort strategy, no textual conflict; base unmoved).
- **Resolution:** n/a (clean).
- **G6 verdict:** PASS-WITH-ISSUES → MERGE-AFTER-FIX. Both done_when items independently
  reproduced GREEN by G6 (door 503→200; relay write 226–269ms). One must-fix (override-var
  name in the FDW SQL comment `HQ_FDW_ALLOW_5433` → `HQ_SYNC_DEV_ALLOW_5433`) applied as
  `167bc7e` before merge; fails safe, comment-only. proxy.go:78 guard + B-164 honored.
- **Gate after merge (control-loop re-gate on `bd03059`):**
  - **G1:** `go build ./...` exit 0; `go vet ./...` exit 0 (0 warnings), from `backend/`.
  - **G2(Go):** `go test -p 1 -count=1 ./...` on `:5434`/`hq_test_go` → exit 0, **9 packages
    `ok`, zero FAIL**; DB-coupled tests ran (`internal/workflow` 1.36s, `internal/sync`
    23.6s); `HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` attested `<unset>`.
  - **G2(Playwright):** N/A-by-footprint (no `tests/*.spec.js`, no seam key matched).
  - **G4:** N/A-by-footprint (no HTML/JS; precache count unchanged at 31).

---

## Merge 2 — Card 2 `sync-live-in-dev-app-proof` → `overnight-20260810`

- **Merge commit:** `489145e` (`--no-ff`)
- **Card branch:** `wo-sync-live-in-dev-app-proof` (tip `d2fa3d2`)
- **Cards involved:** Card 2, built on top of Card 1. Card 2's base was `694bdd7` (the
  run-branch tip AFTER Card 1 merged), and the run branch had not moved since — so this
  merge had **no divergence to reconcile**.
- **Files / hunks:** 7 files, **+896 / −22**, all NEW or additive:
  - `.night-crew/qa/spike-supabase/sync-app-proof.sh` (new, +484) — the promoted red-first harness
  - `.night-crew/qa/spike-supabase/app-proof/workflows-live.spec.js` + `playwright.app-proof.config.js` (new)
  - `Taskfile.yml` (+49) — additive `sync:app-proof` wrapper (no existing target edited)
  - `night-crew.toml` (+23) — **comment-only** footprint NOTE (no key/token)
  - `.night-crew/knowledge/roadmap.md` (leg-3 flip → DEV-COMPLETE)
  - `merge-intent.md` (relocated to run artifacts)
- **Shared surfaces with Card 1:** `Taskfile.yml` (Card 2's `sync:app-proof` is additive
  to Card 1's `sync:dev:*` — no overlap) and `.night-crew/knowledge/roadmap.md` (Card 2's
  leg-3 flip sits on top of Card 1's legs-1–2 flip). Because Card 2 branched from the
  post-Card-1 tip, both landed on Card 2's base already — **no textual conflict**.
- **Intents read:** Card 2's `merge-intent.md` (gate-harness decision = standalone form (a);
  "what of Card 1 must survive" = the `sync:dev:*` task family + 4× `HQ_SYNC_*` dev wiring —
  both preserved intact) and Card 1's merge-intent for the shared-Taskfile check.
- **Conflict?** **NONE — clean merge** (ort strategy, no textual conflict; base unmoved).
- **Resolution:** n/a (clean).
- **G6 verdict:** PASS → MERGE. G6 **independently re-ran** the harness (fresh run, exit 0):
  red exit 1 (carrier down, app stays `waiting` the full 20.5s) / armed exit 0 (carrier up,
  `served` in 454ms carrying the `/saveResponse` sentinel) — non-vacuous. Footprint honest
  (no `workflows.html`/`sync-rxdb/*`/Go edit — spike prediction held), `night-crew.toml`
  comment-only (no park), B-164 restore byte-identical, roadmap flip accurate. Two cosmetic
  nits, non-blocking, no fix round.
- **Gate after merge (control-loop FINAL re-gate on `489145e`):**
  - **G1:** `go build ./...` exit 0; `go vet ./...` exit 0 (0 warnings), from `backend/`.
  - **G2(Go):** `go test -p 1 -count=1 ./...` on `:5434`/`hq_test_go` → exit 0, **9 packages
    `ok`, 0 FAIL**; `HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` attested `<unset>`.
  - **RF:** the card's headline gate — GREEN, re-verified by G6 (relay-up pass / relay-down fail).
  - **G2(Playwright):** N/A for the `:5434` suite (standalone harness is the verdict); **G4** N/A
    (no HTML/JS; precache 31); **G1/G2(Go)** unchanged by Card 2 (no Go touched).

---

## Summary

**2 merges, both clean, zero conflicts.** Both cards landed on `overnight-20260810`. Final
merged tree `489145e`: G1 + G2(Go) green, RF green for both code-changing cards. No parks.
