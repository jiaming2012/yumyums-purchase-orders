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

## Merge 3 — Card 10 `counterparty-notice-prep` → `overnight-20260901`

- **Merge commit:** `4c8f431` (`--no-ff`). **Card branch:** `wo-counterparty-notice-prep` (tip `54b4db4`).
- **Cards involved:** Card 10 only. Docs card — RF `n/a — no code change`.
- **Files / hunks:** 5 files, +233/−1, docs/reference/roadmap only: `docs/contracts/inventory-period-summary.md` (+8, owner line + §0 Card-9-visibility addendum), `docs/contracts/inventory-menu-cogs.md` (+10, same), NEW `reference/counterparty-notice-20260901-draft.md` (+138), `roadmap.md` (+15, nested prep-done bullet under `counterparty-combined-notice`, card stays PLANNED), merge-intent.
- **Shared surface:** `docs/contracts/*` — no other card touched them (Card 9 added log lines only, no doc edit). `roadmap.md` — additive nested bullet, own section. **No conflict.**
- **Conflict?** **NONE — clean** (ort).
- **G6 verdict:** **PASS** (af5d0742) — every notice claim + both doc addenda independently verified against live handlers (period-summary blocking clauses, menu-cogs field names + no `AT TIME ZONE`, B-29 commits, 0072, Card-9 log keys, B-177 framing). One nice-to-have (§1 prose simplifies the COALESCE cast; §4 states it precisely). No must-fix. Only docs changed — no production code.
- **Gate:** G1/G2/G4/Playwright all **N/A-by-footprint** (no code/asset change). Card correctly left `counterparty-combined-notice` PLANNED — closes on operator SEND (P-KR3).
- **P-KR3 reminder:** the draft at `reference/counterparty-notice-20260901-draft.md` is for the operator to review + SEND **before** the deploy.

## Merge 4 — Card 2 `toast-sync-fail-loud` → `overnight-20260901`

- **Merge commit:** `d2120c8` (`--no-ff`). **Card branch:** `wo-toast-sync-fail-loud` (tip `ca05894`).
- **Cards involved:** Card 2. Run branch had advanced (Merges 1–3 + logs).
- **Files / hunks:** 9 files, +590/−21: NEW `internal/toast/syncstatus.go` + 2 test files + `failloud_test.go`; `internal/toast/{worker.go,types.go,sync.go}` (+ErrSFTPUnavailable, SyncStatus, routing); `cmd/server/main.go` (health handler — `toast_sync` field, map widened to `map[string]any`); `roadmap.md` (pipeline-fail-loud DONE); merge-intent.
- **Shared surface:** `cmd/server/main.go` (health handler) — no other landed card touched it; **clean**. `roadmap.md` — Card 2 flips `pipeline-fail-loud` DONE vs run branch's Card 10 counterparty bullet (different sections); **ort auto-merged clean**.
- **Conflict?** **NONE — clean** (ort).
- **G6 verdict:** **PASS** (aaebed2f) — both fail-loud halves reproduced red-first; false-alarm check EXPLICIT PASS (genuine date-not-found `ErrSFTPMiss` stays silent; only dead-transport `ErrSFTPUnavailable` is loud); `-race` clean (mutex-guarded SyncStatus); alert fires once per cycle; map-widening safe; boot-order honest (`unknown` when worker disabled). No must-fix.
- **Gate after merge (control-loop re-gate on `d2120c8`):**
  - **G1:** `go build ./...` 0; `go vet ./...` 0.
  - **G2(Go):** footprint `internal/toast` **ok** 0.54s, `internal/alerts` **ok** 0.48s (isolated `hq_test_go_regate2`, `HQ_SYNC_*` unset). `internal/sync` red unchanged = B-178 (environmental).
  - **G4 / G2(Playwright):** N/A-by-footprint.
- **Roadmap:** `pipeline-fail-loud` **DONE** — both halves (a `toast-sync-fail-loud` + b `period-summary-visibility`) now landed. New `/health` field `toast_sync` documented for Card 3 (its mechanism, once the key is placed post-deploy, flips `toast_sync`→ok = the kill-drill proof).

## Merge 5 — Card 4 `client-guard-coverage` → `overnight-20260901`

- **Merge commit:** `5d96274` (`--no-ff`). **Card branch:** `wo-client-guard-coverage` (tip `37423c9`).
- **Cards involved:** Card 4 (FIRST in Track B — lands the B-154 `[e2e.seams]` rider). Run branch had advanced (Cards 1,2,9,10 + logs).
- **Files / hunks:** 5 files, +192/−1, ZERO runtime code: `night-crew.toml` (the `[e2e.seams] "sync-rxdb"` rider), NEW `tests/sync-rxdb-client.spec.js` + `tests/index.spec.js` (guard tests), `roadmap.md` (DONE), merge-intent.
- **Shared surface:** `night-crew.toml` — no other landed card touched it. `roadmap.md` — own section. **No conflict.**
- **Conflict?** **NONE — clean** (ort).
- **G6 verdict:** **PASS** (a17bc838) — `[e2e.seams]` row selects exactly the 7 sync-rxdb specs, correctly EXCLUDES load-sensitive `sync.spec.js` (zero sync-rxdb refs); both guards red on their exact mutations (B-149 uid clause → +6 surfaces leak; B-10 drop await → redirect before purge); repo-hygiene roll-call intact; ZERO runtime code. No must-fix. nice-to-have: workbox-*.js artifacts not gitignored.
- **Gate — Playwright (authoritative for this sync-footprint card):**
  - Card 4's own worktree ran the FULL suite: **822 passed / 6 skipped / 7 failed = exactly ONE summary block**. Both new guard tests pass. The 7 failures decompose to **B-174 ×3** (`sw-api-cache-partition` B1-XT-01/-02/-05) + **B-176** (`workflows.spec.js` DBL-05), both documented deterministic pre-existing reds, + **3 flakes** (`sync.spec.js:2976`, `workflows.spec.js:412`, `workflows.spec.js:1110`) proven GREEN on isolated rerun. Zero attributable to Card 4 (no runtime code).
  - 🛑 **Baseline correction (carry forward):** the slate's NAMED armed reds (B-27 inventory:883, LST-17 sync:446, B-162 receipt-carousel:123) ALL PASSED this run — they are flaky-named, not deterministic. The real deterministic Playwright baseline on this tree is **B-174 + B-176**. Cards 7/11 Playwright judging uses this.
  - **Merged-tree health-shape check (orchestrator):** because Card 4's suite ran on its own base (before Card 2's `/health` change), the orchestrator ran the 3 health-consuming specs (`storage-banner`, `version-badge`, `grant-enforcement-parity`) on the merged tree `5d96274` → **24 passed** (incl. "banner stays hidden when health omits storage" + "live /api/v1/health carries the storage field"). Card 2's `toast_sync`/`map[string]any` widening is safe against the frontend. G1/G2-Go/G4 N/A-by-footprint.

## Merge 6 — Card 11 `deploy-hygiene-honesty` → `overnight-20260901`

- **Merge commit:** `8dcf506` (`--no-ff`). **Card branch:** `wo-deploy-hygiene-honesty` (tip `2561ded`).
- **Cards involved:** Card 11 (LAST in Track C — completes Track C).
- **Files / hunks:** 6 files, +238/−9: `backend/Dockerfile` (printf gains `\n`), `build-sw.js` (comment), `scripts/write-version-json.js` (comment), NEW `tests/version-json-parity.spec.js`, `roadmap.md` (DONE), merge-intent.
- **Shared surface:** `sw.js`/precache — Card 11 correctly did NOT touch sw.js (no precached byte changed). Card 7 `sync-doc-honesty` WILL touch sw.js; orchestrator's post-Card-7 `task sw` reconciles. **No conflict.**
- **Conflict?** **NONE — clean** (ort).
- **G6 verdict:** **PASS** (afc24b6e) — version.json generators byte-identical (21 bytes, md5 `226015…` = committed sw.js revision), red-first byte-diff reproduced (34≠33); B-17 claim factually verified (git C-quotes non-ASCII/control, spaces bare, `-z` raw) in a throwaway repo; live roadmap honest (false wording only in frozen artifacts, correctly untouched); precache 31; workbox chunk-hash env-noise confirmed benign (all 31 app-asset revisions byte-identical). No must-fix.
- **Gate:** G1 build+vet exit 0. **G4:** build-sw.js exit 0, precache **31**, version parity 1.5.0 (verified by G6 in worktree; no landed card changed the precache set, so it holds on the merged tree; authoritative final regen runs at closeout after Card 7). **Full Playwright: DEFERRED** — card changes no frontend/precached asset; the closeout full-suite on the final tree covers it. Roadmap `deploy-hygiene-honesty` DONE.
- **🛑 Closeout note — workbox pin (B-179 candidate):** `node_modules` has `workbox-build@7.3.0` while lockfile pins `7.4.1`; a `build-sw.js` regen produces a spurious runtime chunk-hash delta (`workbox-0225851e`→`d4a0f5c1`, count still 31, app assets identical). Before the final closeout regen, either `npm ci` to match the lockfile, or commit only if the delta is chunk-hash-only and documented.

## Merge 7 — Card 3 `toast-ingest-resurrection` → `overnight-20260901`

- **Merge commit:** `4bbb7dc` (`--no-ff`). **Card branch:** `wo-toast-ingest-resurrection` (tip `a283c5a`). **Completes Track A.**
- **Cards involved:** Card 3 (LAST in Track A). Run branch had advanced (Cards 1,2,4,9,10,11 + logs).
- **Files / hunks:** 5 files, +314/−7: `docker-compose.prod.yml` (+23 — `volumes:` bind-mount `./id_rsa:/app/id_rsa:ro` + `TOAST_SFTP_KEY_PATH` + `TOAST_SYNC_INTERVAL: "12h"`), `backend/internal/toast/config_test.go` (+22 — red-first config-seam test), NEW `reference/toast-archive-gap-20260901.md` (archive gap + attended-steps note), `roadmap.md` (DONE), merge-intent.
- **Shared surface:** `docker-compose.prod.yml` — no other card touched it. `roadmap.md` — own section. **No conflict.** (Root `Taskfile.yml` NOT touched — Card 6 owns a different Taskfile section, no collision.)
- **Conflict?** **NONE — clean** (ort).
- **G6 verdict:** **PASS** (a87d827a). 🛑 **NO-KEY safety check CLEAN** — no private-key material created/committed anywhere (the load-bearing safety gate for this card). Bind-mount resolves (`docker compose config` validated), sync resurrection confirmed (`TOAST_SYNC_INTERVAL=12h` starts worker), archive-gap math independently reproduced (cutoff 2026-08-04; 10 aged-out + 28 recoverable = 38-day gap; >7-day slice needs sales-processor `migrate-toast-archive`, no SFTP range-backfill — verified against `cmd/sync-toast` + `cmd/migrate-toast-archive`), attended-steps note complete, red-first config-seam test guards the `os.Stat` branch.
  - nice-to-have (not a blocker): a forgotten key file yields an empty *directory* (`os.Stat` passes, no boot fail-fast) — but Card 2's fail-loud path catches it (first cycle → `os.ReadFile` fails → `ErrSFTPUnavailable` → `toast_sync: failing` + Cliq alert), and the attended note Step 3 verifies it. Cards 2+3 reinforce.
  - informational: `docker compose config` `env_file.0 must be a string` is a PRE-EXISTING long-form quirk (base fails identically), not card-introduced; prod's compose version handles it.
- **Gate after merge (control-loop re-gate on `4bbb7dc`):** G1 `go build ./...` 0, `go vet ./...` 0. `internal/toast` **ok** 0.55s (isolated `hq_test_go_regate3`, `HQ_SYNC_*` unset). G4/G2-Playwright N/A-by-footprint. `internal/sync` red unchanged = B-178.
- **Roadmap:** `toast-ingest-resurrection` DONE (dev-provable half). Prod proof attended post-deploy. **Track A DONE (Cards 1,2,3).**

## Merge 8 — Card 5 `cdc-single-fire` → `overnight-20260901`

- **Merge commit:** `3359dee` (`--no-ff`). **Card branch:** `wo-cdc-single-fire` (tip `f937737`).
- **Cards involved:** Card 5 (Track B, 2nd). Run branch had advanced (Cards 1,2,3,4,9,10,11).
- **Files / hunks:** 8 files, +414/−27: `backend/internal/workflow/{repository.go,handler.go}` (saveResponse folds lamport_ts into its upsert; SaveResponseHandler computes stamp), `backend/internal/sync/ops.go` (ADDS `EmitOpForStampedEntity`, `NextLamportTS`, `insertOpRowAndNotify` — no existing symbol modified), `cmd/server/main.go` (SaveResponseFunc `, 0`), NEW `cdc_single_fire_test.go`, `stable_identity_test.go` (+6), `roadmap.md` (DONE), merge-intent.
- **Shared surfaces:** `cmd/server/main.go` — ALSO edited by Card 2 (health handler). Card 5's edit is in `workflowOpRouter` (different function); **ort auto-merged clean**. `internal/sync/ops.go` — Card 8 `app-slug-association` will also touch `internal/sync`; Card 5's additions are 3 NEW symbols (projection-writer app_slug is untouched), so **low conflict risk** with Card 8 — keep both if any. `roadmap.md` — own section.
- **Conflict?** **NONE — clean** (ort).
- **G6 verdict:** **PASS** (a64ca53c) — red-first trigger-count reproduced (revert→2 fires, restore→1); `internal/workflow` 39/39; LWW ordering + draft distinction intact; **workflow-footprint Playwright subset (workflows+persistence+sync+repro-cut-task) = 187 passed / 2 failed**, both accounted: `[DBL-05]`=B-176 (documented baseline red), `[SYNC-RF-02]`=flake (passed in isolation); all 34 persistence save/draft tests GREEN (/saveResponse contract preserved). G6 ruled the **/ops-left-unchanged scoping SOUND** (folding lamport inline on /ops would advance lamport_ts before CheckLWW → false conflicts; B-157 scopes to /saveResponse). Two pre-existing nice-to-haves noted (non-atomic NextLamportTS read, field_response CheckLWW keying) — neither a regression.
- **Gate after merge (control-loop re-gate on `3359dee`):** G1 `go build ./...` 0, `go vet ./...` 0. `internal/workflow` **ok** 4.24s (isolated `hq_test_go_regate5`, `HQ_SYNC_*` unset). Playwright footprint gate = G6's subset run (authoritative; merged tree adds only Card 3 toast-infra + Card 11 build-tooling atop G6's base, neither touches the workflow frontend flows). `internal/sync` red unchanged = B-178.
- **Roadmap:** `cdc-single-fire` DONE.
