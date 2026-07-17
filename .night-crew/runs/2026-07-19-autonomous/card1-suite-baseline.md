## §1 — Suite baseline evidence (Card 1: `cycle-gate-suite-baseline`)

**Verdict: ATTEST — 0 uncategorized reds · PARK trigger did NOT fire. Suite went 38 reds (07-16 gate) → 1 red, and that red is isolation-confirmed cross-test DB-pollution, not a code defect.**

### Run mechanics (methodology — read before the numbers)

Deterministic stack run **once** on an isolated ephemeral pg16, read-only, torn down after. Go toolchain `/usr/local/go/bin/go` (go1.25.8).

- **DB isolation:** a standalone `postgres:16` container on a **Docker-assigned host port (`127.0.0.1:57606`)**, bound to loopback. Host `:5432` is held by an unrelated always-on Postgres and was **never touched** — HQ's real DB was never in the path (footprint rule honored). Migrated by a throwaway app boot (goose embedded migrations → **version 70, 48 tables**), exactly as `docker-compose.nc.yml`'s app service would.
- **Ordering:** Go units first (`go test -count=1 -p 1 ./...` against the isolated pg16), then the full Playwright suite via Playwright's **own** freshly-spawned webServer (`go run ./cmd/server`, `CI=1` so `reuseExistingServer:false` → fresh boot + auto-migrate + torn down after). `-count=1` forces a genuine run (no stale build cache).
- **Documented-baseline shell:** no `ANTHROPIC_API_KEY` / no DO-Spaces creds in the environment — matches the ledger's documented baseline. `MERCURY_API_KEY=` / `TOAST_SYNC_INTERVAL=0` / `E2E_DISABLE_SCHEDULERS=1` as the webServer config sets them.
- **No `task sw`:** read-only closeout, zero asset diff — Playwright driven directly via `npx`, not `task test` (which would rebuild the SW as a dep). Working tree stayed clean throughout.

### Two invalid attempts first (carried, not hidden)

Honesty note for the record — the first two suite attempts were **methodologically invalid and discarded**; only the third (above) is attested. Both are preserved under `suite-logs/attempts/`:

1. **Foreign-server reuse.** Playwright's `reuseExistingServer: !process.env.CI` (CI unset) silently **reused a leaked `hqserver` orphaned by a different session** (pid 34423, session `af72b7da…`, pointing at a long-dead pg on `:5439`). Every test failed on the login page — the run tested nothing about our code. Fixed by killing the orphan and setting `CI=1` (forces a fresh own-webServer + teardown).
2. **Unmigrated DB.** Go units first ran against an isolated pg that had **not been migrated** (the standalone container publishes no app service to auto-migrate it) — all 42 `internal/recipes` tests `Fatalf`'d on `relation "drift_check_results" does not exist`, while other packages returned stale `(cached)` results masking it. Fixed by migrating the DB via a throwaway app boot **before** the go phase and forcing `-count=1`.

Neither invalid attempt is a product signal; both are harness-provisioning defects in this orchestrator's first passes, corrected and documented.

### Counts (authoritative — Playwright JSON reporter + `go test`)

| Suite | passed | failed (unexpected) | flaky | skipped | Exit |
|---|---|---|---|---|---|
| **Playwright** (chromium + bdd) | **450** | **1** | **0** | **6** | 1 |
| **Go units** (`go test -count=1 -p 1 ./...`) | 7 pkgs `ok` | **0** | — | — | **0** |

**`0 flaky` is load-bearing:** with `retries:1`, the single Playwright failure failed on **both** attempts → deterministic-within-run, not transient. Full-suite wall-clock ≈ 16.3 min.

### Go units — fully green (cleaner than the 07-16 gate)
All 7 test-bearing packages `ok`, fresh (`-count=1`, no cache): `auth`, `inventory` (20.0s), `purchasing` (6.9s), **`receipt` (20.7s)**, `recipes` (15.9s), `toast`, `workflow`. **Exit 0.** Notably `internal/receipt` **passed without `ANTHROPIC_API_KEY`** — the env-gated red the 07-16 gate carried (T-8/T-11) is **not present this cycle**; the Go half of the deterministic stack is unconditionally green.

### The single Playwright red — categorized: cross-test DB-pollution (documented; isolation-confirmed)
`tests/workflows.spec.js › approved checklist shows Approved badge and cannot be resubmitted [LST-08 RUN-08]` — `expect(locator('#toast')).toBeVisible()` got `hidden` (both attempts). This is the **same test the 07-16 gate categorized as cross-test DB-pollution** (`cycle-closeout-20260716.md` §1, Category 3 — "passes alone, fails in-suite").

**Isolation re-run (the discipline: a result is not a proof).** Re-ran this one test alone on a **fresh single-test pg16** under `--retries=0`: **`1 passed (2.5m)` → GREEN.** The failure is therefore a **cross-test ordering/pollution artifact** (shared `hq_test` across the 457-test suite), **not** a product defect and **not** a new red. → **0 uncategorized reds.**

### Convergence-suite zero-flake proof (Card 1 item 3 — the no-retry hard gate)
`tests/sync.spec.js` (the two-device convergence matrix — 39 tests) run **3 consecutive times on fresh DBs under `--retries=0`**: **39 passed / 39 / 39 — 3-for-3, ~4.6m each, exit 0 all three.** The Eng "convergence matrix — 0 cells red" and Delivery "convergence proof (no-retry hard gate)" KRs rest on **demonstrated determinism**, not one pass. This confirms the operator's 2026-07-18 rider-retirement holds at the gate.

### 6 skipped = documented expected-skips (not reds)
Same set as the 07-16 gate: 2 S3-parks carried as `test.skip`+reason (`inventory FR-27 photo upload` — needs live DO Spaces; `onboarding FR-18 custom-thumbnail` — needs S3 PUT) + 4 conditional expected-skips (`persistence recipe usage_pct round-trip`; `purchasing vendor-section completion persists`; `purchasing toast without photo`; `purchasing PO approve-button-for-admin-when-locked`).

### The PARK check — cleared
PARK trigger = *"any red not attributable to a documented baseline cause."* The **one** Playwright red maps to the documented cross-test-pollution category and **greens in isolation**; the Go suite is exit-0. **0 uncategorized reds → no PARK.** `DECISIONS-NEEDED.md §A` stays empty.

### Consequence for waiver #1 (Eng KR5) — honest, not inflated
The gate's new criterion this cycle is **`task test` exits 0 on the deterministic stack** (would *formally* retire waiver #1). The full Playwright suite returned **exit 1** — one documented cross-test-pollution red short of literal exit-0. Per the slate's explicit clause ("if exit-0 over the full defined deterministic stack proves unreachable, attest the substitute and mark the Eng waiver-#1 KR **PARTIAL, not PASS**, never silently"):

- **Substitute criterion "0 new uncategorized reds vs the documented baseline" — MET** (1 red, categorized, isolation-green; Go exit-0).
- **Literal "`task test` exit 0 / 0 pre-existing reds" — NOT met** (1 pre-existing pollution red remains).
- → **Eng KR5 = PARTIAL. Waiver #1 is SUBSTANTIALLY retired (38 reds → 1) but NOT formally retired** — it carries forward, reduced to a single test-isolation defect. **Carried backlog item:** fix the `approved checklist … [LST-08 RUN-08]` cross-test isolation (some earlier spec leaves `#toast`/approval state in shared `hq_test`) so next cycle reaches literal `task test` exit-0 and formally retires waiver #1.
