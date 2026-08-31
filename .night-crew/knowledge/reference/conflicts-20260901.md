# Conflict log — run 20260901

One entry per merge (clean or conflicted), per §15ad.66. A clean merge gets a
one-line "no conflict" entry so an empty log never reads as "no conflicts" when
it means "the logging never ran". Morning triage audits this file.

**Run:** `overnight-20260901` (off `dev` @ `55aa6f8`). 11 cards, 3 concurrent tracks
(A: cards 1–3 · B: cards 4–8 · C: cards 9–11). Orchestrator alone merges to the run
branch, in landing order. `sw.js` regeneration rule: every card touching a precached
file runs `task sw` + commits `sw.js` in its own change set; after the LAST of Cards 7
and 11 lands, re-run `task sw` on the merged tree and commit the regeneration with the
merge (precache count 31 — an unexplained move is B-37).

---

## Merge 1 — Card 9 `period-summary-visibility` → `overnight-20260901`

- **Merge commit:** `4c387cb` (`--no-ff`). **Card branch:** `wo-period-summary-visibility` (tip `ab9dbb0`).
- **Cards involved:** Card 9 only — first card to land. Run branch had advanced from the card's base `0bfd1af` only by the D-KR2 stamp commit `9e9c858` (card-actuals.md), which the card does not touch.
- **Files / hunks:** 6 files, +388/−8, all NEW or additive: `backend/internal/inventory/handler.go` (+17, one `slog.Info`), `backend/internal/recipes/handler.go` (+11, one `slog.Info` — `/menu-cogs` lives here, not inventory), two NEW `*_visibility_test.go`, `roadmap.md` (split-card note, NOT a DONE flip), merge-intent.
- **Intents read:** only Card 9's. Shared surfaces claimed: none outside its two handler packages. What must survive: the two `slog.Info` visibility lines.
- **Conflict?** **NONE — clean** (ort; base effectively unmoved for the card's files).
- **G6 verdict:** **PASS** (a814c70) — both log lines reproduced red-first, success-end placement, counts-not-slices, `ready` real verdict, blocked-week case covered, no PII. No must-fix.
- **Gate after merge (control-loop re-gate on `4c387cb`, serial, no concurrent Go leg):**
  - **G1:** `go build ./...` 0; `go vet ./...` 0 (from `backend/`).
  - **G2(Go):** `go test -p 1 -count=1 ./...`, `DB_TEST_URL=hq_test_go`, `HQ_SYNC_SUBSTRATE_OPTIONAL`+`HQ_SYNC_GATE_CHILD` **unset**. Footprint `internal/inventory` **ok** 7.25s, `internal/recipes` **ok** 7.34s; all other packages **ok** EXCEPT **`internal/sync` FAIL — `TestJWTBridgeRLS`, environmental (B-178): 13 `spikec-*` relay rows contaminate the RLS fixture.** Reproduced on base, zero-concurrency; NOT this card (touches zero sync files). Disposition: known-environmental red for run 20260901.
  - **G4 / G2(Playwright):** N/A-by-footprint (no HTML/JS/specs).
- **Roadmap:** `pipeline-fail-loud` stays un-flipped — this is only half (b); half (a) `toast-sync-fail-loud` (Track A) must land first.

## Merge 2 — Card 1 `receipt-worker-correctness` → `overnight-20260901`

- **Merge commit:** `0e988ce` (`--no-ff`). **Card branch:** `wo-receipt-worker-correctness` (tip `1e4bc23`).
- **Cards involved:** Card 1; run branch had advanced (Merge 1 `4c387cb`, stamps `9e9c858`, backlog `be62f48`).
- **Files / hunks:** 4 files, +281/−16: `backend/internal/receipt/worker.go` (+81), `worker_test.go` (+174), `roadmap.md` (PLANNED→DONE flip), merge-intent.
- **Shared surface:** `.night-crew/knowledge/roadmap.md` — ALSO edited by Card 9 (Merge 1). **ort auto-merged clean** (different lines: Card 1 flips `receipt-worker-correctness` DONE; Card 9's note sits on `pipeline-fail-loud`). Both intents read; neither claimed the other's line.
- **Conflict?** **NONE — clean** (ort auto-merge of roadmap.md, no textual conflict).
- **G6 verdict:** **PASS-WITH-ISSUES** (a538ca48) — B-28 fallback + B-175 both reproduced red-first and verified. One must-fix-or-waive: the *parseable* path is not zone-converted (boundary drift). **WAIVED by the orchestrator** → filed **B-177**: out of B-28's defined scope (which is the unparseable fallback), pre-existing, and entangled with the documented Chicago→NY changeover (migration 0072). Card's defined deliverables complete.
- **Gate after merge (control-loop re-gate on `0e988ce`):**
  - **G1:** `go build ./...` 0; `go vet ./...` 0.
  - **G2(Go):** footprint `internal/receipt` **ok** 7.96s (isolated `hq_test_go_regate1`, `HQ_SYNC_*` unset). `internal/sync` red unchanged = B-178 (environmental, not this card). Full-suite shared-package coverage was established on Merge 1's re-gate `4c387cb` (all ok bar sync); Card 1 adds only `internal/receipt` changes atop it.
  - **G4 / G2(Playwright):** N/A-by-footprint.
- **Roadmap:** `receipt-worker-correctness` flipped PLANNED→DONE.
