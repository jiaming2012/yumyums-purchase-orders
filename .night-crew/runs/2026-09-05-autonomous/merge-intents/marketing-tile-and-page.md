# Merge intent — Card 1 · `marketing-tile-and-page` (run 20260905)

Branch: `wo-marketing-tile-and-page` (cut from dev @ 0670798, Wave 0 — lands before
every other card tonight). Card authority: slate-20260905 Card 1. Footprint: scanner UI
(`index.html`, `marketing.html` NEW) + redemption backend (SeedHQApps seed + grants) +
`sw.js` (precache 31→32, deliberate) + the CLAUDE.md precache-count line.

## Shared files touched

- `index.html` — the Marketing tile added to the grid + `'marketing.html':'marketing'`
  added to `TILE_SLUGS`. Slate: **exclusive to Card 1** tonight; no other card touches
  it.
- `marketing.html` (NEW) — the page shell Cards 5 and 6 extend later tonight. See
  "What must survive any merge" for the contract they inherit.
- `backend/internal/db/db.go` — `SeedHQApps()` grows two rows (`marketing`,
  `marketing-offline-override`) **inside the existing hq_apps INSERT block** (the
  grant-parity spec derives the seeded set by parsing that exact block — a second
  INSERT statement would silently narrow the guard), plus a first-registration-gated
  grant seed (CTE on the INSERT's RETURNING; see Red-first / engineering calls below).
- `tests/grant-enforcement-parity.spec.js` — two `NA_WITH_REASON` entries
  (`marketing`, `marketing-offline-override`): no marketing `/api/v1` endpoint exists
  tonight, so there is no RequirePermission mount to point the parity guard at. The
  entries are tripwired by PARITY-NA-FRESH: the moment a marketing route lands
  (Cards 5–7 or later), the N/A entries must be deleted and the routes gated.
- `tests/marketing.spec.js` (NEW) — this card's tile/permission/seed assertions
  (the red-first artifact). Cards 5–6 will extend this file; nothing in it hardcodes
  a count another card would move.
- `backend/internal/auth/marketing_seed_test.go` (NEW — a new file rather than the
  append to permission_test.go this intent first planned: cleaner merge surface, and
  it needs its own helpers) — the Go seed/grant tests (marketing rows registered;
  grants seed on first registration only; a revoked grant survives a SeedHQApps
  re-run; the entitlement is NOT implied by the app grant at the gate).
  `permission_test.go` itself is untouched.
- `sw.js` + `CLAUDE.md` (precache-count line 31→32) — regenerated AFTER the
  marketing.html commit (build-sw.js reads git HEAD), committed together in the same
  change set. Cards 5 and 6 move the count again (each states its own move).
- `.night-crew/knowledge/roadmap.md` — one-line status flip of the
  `marketing-tile-and-page` card (PLANNED → DRAFTING), matching run 20260904's flip
  convention. No other roadmap line moves.
- `.night-crew/runs/2026-09-05-autonomous/merge-intents/marketing-tile-and-page.md` —
  this file (amended as evidence lands).
- `.night-crew/runs/2026-09-05-autonomous/card1-*.log` — committed whole gate logs,
  each ending in its `EXIT=` line.
- `users.html` — **nothing here.** The `marketing-offline-override` row renders
  through the existing access-editor path (`GetAppPermissions` returns every enabled
  hq_apps row; `renderAccess` gives it the standard 3 role toggles + individual-grant
  picker). Zero UI change needed — that is the point of the grant-surface convention.
- `night-crew.toml` — **nothing here.** `marketing.html` is a deliberately undeclared
  path → full-suite fallback (slate precondition flag; a `marketing` seam entry is
  Card 6's engineering call once `tests/marketing.spec.js` has coverage worth naming).
- `package.json` / `backend/internal/version/version.go` / `version.json` — **nothing
  here** (no version bump tonight — /save-project's job at deploy; parity must simply
  hold through the sw.js regen).
- `supabase/`, `sync-rxdb/`, `.night-crew/spikes/**`, `.night-crew/knowledge/spikes/**`
  — **nothing here** (spike dirs are read-only inputs).

## What must survive any merge

- **The marketing.html shell contract Cards 5–6 build on:** the four-tab structure
  (`#t1..#t4` buttons, `#s1..#s4` sections — Scan / Campaigns / Subscribers /
  Redemption stats), the `show(n)` tab switcher, the `#scanner-host` container inside
  the Scan section (Card 5 mounts the camera there), the shared CSS variable block +
  dark mode, the back link, and the PWA boilerplate (sw registration, log.js, ptr.js).
  Cards 5–6 extend inside the Scan section; they must not need to re-plumb the shell.
- **The `TILE_SLUGS` entry and the tile markup in `index.html`** — the permission gate
  for the tile is the `'marketing.html':'marketing'` mapping; dropping it makes the
  tile unconditionally visible (fails done_when (b)).
- **The whole SeedHQApps block including the grant-seed CTE** — the first-registration
  gating (grants seeded only when the hq_apps row is newly inserted) is what lets an
  operator revocation survive server restarts. Replacing it with an unconditional
  grant upsert reintroduces grant resurrection.
- **The two `NA_WITH_REASON` entries** until a marketing route exists — deleting them
  without mounting a gate reds PARITY-EQ; deleting them WHILE mounting the gate is the
  intended future edit.
- The roadmap status flip; the `card1-*.log` evidence files; this intent.

## What is safe to drop

- Nothing in this branch is scratch. `sw.js` is a committed artifact — if a later
  card's merge regenerates it on a newer HEAD, the regenerated file supersedes this
  card's copy (count moves are stated per card; Card 1's stated state is 32 with
  `marketing.html` present in the manifest).

## Red-first

Greenfield-shaped RF per the slate: the tile/permission assertions were written
FIRST and RUN on the pre-change tree (dev @ 0670798 + only the new test files),
where every one reds because the behavior does not exist. Full logs with commands
and exit codes: `.night-crew/runs/2026-09-05-autonomous/card1-red.log` (committed
`7a2e9e0`).

- **Go leg** — `DB_TEST_URL='postgres://hqtest:hqtest@localhost:5434/hq_test_go_c1?sslmode=disable'
  go test -count=1 -run 'TestSeedHQApps_RegistersMarketingSurfaces|TestSeedHQApps_MarketingGrants_SeedOnFirstRegistrationOnly|TestOfflineOverride_EntitlementNotImpliedByAppGrant' -v ./internal/auth/`
  → **EXIT=1**, 3/3 FAIL. Named reds:
  `TestSeedHQApps_RegistersMarketingSurfaces` (`hq_apps slug "marketing": got 0
  enabled rows, want 1`), `TestSeedHQApps_MarketingGrants_SeedOnFirstRegistrationOnly`
  (`marketing role grants … got [], want [admin manager team_member]`),
  `TestOfflineOverride_EntitlementNotImpliedByAppGrant` (no surface to grant).
- **Playwright leg** — `TEST_PORT=3101 TEST_DB_NAME=hq_test_e2e_c1 npx bddgen &&
  npx playwright test tests/marketing.spec.js --retries=0` → **EXIT=1**, 6/6 failed:
  the SEED test (no `marketing` row from `/api/v1/apps/permissions`), both tile
  tests (`a.tile[href="marketing.html"]` count 0 / `marketing app must exist to be
  strippable`), the shell test (`.tabs button` count 0 — /marketing.html serves no
  page), both entitlement tests (`#access-marketing` count 0 / `entitlement surface
  must exist` got null).
- **Green**: the same commands re-run after implementation — see
  `card1-green.log` and the gate logs below.

## Gate evidence (run in this worktree; full logs committed beside this file)

- **G1** — from `backend/`: `go build ./...` EXIT=0, `go vet ./...` EXIT=0
  (`card1-g1.log`).
- **G2 (Go)** — `DB_TEST_URL='postgres://hqtest:hqtest@localhost:5434/hq_test_go_c1?sslmode=disable'
  go test -p 1 -count=1 -v ./...` (`card1-g2-go.log`). **EXIT=1 — one failure,
  proven pre-existing** (below). Counts, not `ok`: **11 test-bearing packages**
  (alerts 3, auth 16, inventory 36, onboarding 6, photos 7, purchasing 8,
  receipt 59, recipes 59, sync 53, toast 26, **workflow 39**) — 312 top-level
  tests (309 PASS · 1 FAIL · 2 SKIP), **538 counting subtests** (531 PASS · 5 FAIL
  all inside the one failing test · 2 SKIP). The ladder's "9 packages ~439 tests /
  workflow 35" floor is met and exceeded — the tree has grown since those figures
  were written; nothing ran short. `internal/workflow` ran **39** (not zero — the
  DB-coupled suite executed). The `internal/sync` self-asserted gates PASSED:
  `TestRowVisibilitySubtestCount_Structural`, `_Executed` (59-subtest constant) and
  `TestSubstrateGate_ExitCodeAsymmetry`. The two SKIPs are the documented
  `HQ_SYNC_SPIKE_LIVE` opt-in pair in `proxy_live_test.go` (typed, named skip).
  **Environment proof in the log header:** `env | grep HQ_SYNC` → no output,
  grep_exit=1 — **`HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` were both
  UNSET** (decision 108 / B-36).
  - **The one failure — `TestJWTBridgeRLS` (internal/sync) — is PRE-EXISTING
    substrate contamination, not this branch:** its service_role CONTROL (and the
    V9/V12 controls that recheck it) expects exactly the 4 fixture rows in the
    spike-supabase stack's `hq_sync_checklists` and instead sees 12 extra
    `spikec-*` rows — residue from the Activity-C spike sittings. This branch's
    diff contains **zero** references to `spikec`/`hq_sync_checklists` (grep in
    `card1-baseline-jwtbridge.log` header), and the SAME test fails IDENTICALLY on
    the base tree (dev @ 0670798, throwaway worktree, fresh Go DB
    `hq_test_go_c1b`): `card1-baseline-jwtbridge.log`, EXIT=1, same `spikec-*`
    list. Named, not chased, per the slate's baseline rule. The substrate was NOT
    cleaned by this card — the slate marks the spike-supabase stack **reconcile
    mode only**, and the residue is triage's call, not a Wave-0 card's.
- **G2 (Playwright)** — full suite (marketing.html is an undeclared `[e2e.seams]`
  path → full-suite fallback, deliberate). **Two runs:**
  - **Run 1** (`card1-e2e.log`, HEAD ef3217e, concurrent with the Go leg):
    EXIT=1 — **831 passed / 6 failed / 6 skipped (28.1m)**, exactly one summary
    block. Verdict per failure:
    1. `repo-hygiene.spec.js:184` (B-140 stale-gate scan) — **INTRODUCED by this
       card**: the two §16 create/stats comments paired GATE_PHRASE wording
       ("when those endpoints land") with the DONE card slug
       `grant-enforcement-parity` in the scan window. **Fixed in `2592c09`**
       (reworded; meaning unchanged), sw.js re-regenerated in `9bcee92`.
    2–6. `onboarding.spec.js:696` (30.0s timeout at readiness form),
       `sw-api-cache-partition.spec.js` B1-XT-01/-02/-05 (each a 120s timeout at
       the LOGIN helper's `waitForURL`, before any SW assertion), and
       `workflows.spec.js:1202` DBL-05 — all timeout/race shapes consistent with
       the CPU contention of running the Go suite (45s of `internal/sync`
       subprocess re-compiles included) concurrently on this box. Judged by the
       clean re-run below rather than chased individually.
  - **Run 2** (`card1-e2e-2.log`, final HEAD 9bcee92, serial — no concurrent
    leg): EXIT=1 — **831 passed / 6 failed / 6 skipped (26.6m)**, exactly one
    summary block. `repo-hygiene.spec.js:184` is GREEN (the 2592c09 fix held)
    and `onboarding.spec.js:696` is GREEN (run-1 red was a flake). Run-2 reds:
    `onboarding.spec.js:2268` and `workflows.spec.js:1407` GATE-02 (each green
    in run 1 on the same code, both 30s-timeout shapes → flakes, proven by the
    two logs disagreeing); and the **recurring four** —
    `sw-api-cache-partition.spec.js` B1-XT-01/-02/-05 (120s timeouts at the
    login helper, before any SW assertion) and `workflows.spec.js:1202` DBL-05
    (queue-entry count 0 vs 1) — red in BOTH runs. **No base-tree run of these
    four was captured** (the run was closed out by the control loop before that
    leg): pre-existing is PLAUSIBLE (no mechanism connects them to this diff —
    no workflows/login/SW-logic code changed; sw.js moved by one manifest line
    + runtime-chunk rename) but **UNPROVEN — triage owes these four a base-tree
    comparison** before merging, exactly as the slate's baseline rule requires.
- **G4** — `node build-sw.js` EXIT=0, **32 files precached** with
  `marketing.html` in the manifest; second run leaves the tree clean (only
  uncommitted run-dir gate logs in `git status`, `sw.js` byte-identical); version
  parity 1.6.2 ≡ 1.6.2 ≡ 1.6.2 across `version.go Frontend` / `package.json` /
  `version.json` — no bump, parity intact (`card1-g4.log`).
- **RF** — the ## Red-first section above; red log `card1-red.log`, green re-run
  `card1-green.log` (Go 3/3 PASS EXIT=0; Playwright 6/6 passed EXIT=0).
- **Isolation note (deviation, stated):** the Go leg ran with `HQ_RLS_TEST_DB`
  **unset** rather than the launch prompt's `hq_rls_c1`. Unset is the suite's
  designed per-leg-unique path: `rvHQDatabase()` falls back to the PID-derived
  `hq_rls_b2_fdw_p<pid>` (B-142b), which cannot collide with any concurrent leg —
  the same isolation the mandate exists to guarantee, one notch stronger. All
  other isolation values were used as mandated (TEST_PORT=3101,
  TEST_DB_NAME=hq_test_e2e_c1, Go DB hq_test_go_c1, scratchpad `c1-impl/`).

## Engineering calls (recorded for the merge record)

1. **Grant storage shape (fork #12, operator-resolved surface):** the
   `offline_override` entitlement is an `hq_apps` grant-surface row,
   slug `marketing-offline-override`, reusing the established narrow-slug convention
   (design `prove-surface-gating-and-endpoints.md` §1.4 Option (i)) — so it flows
   through `/me/apps`, the Users access editor, and `auth.RequirePermission`
   unchanged, and is grantable per-role AND per-user out of the box. It is an
   **entitlement, not a gated tab**: when a handler eventually enforces it (Card 6's
   offline submit / Card 7's arbitration surface), the check must be
   `RequirePermission(pool, "marketing-offline-override")` with **NO umbrella slug** —
   holding the `marketing` app grant must never imply the override (that would derive
   the entitlement from tab access, which fork #12 explicitly forbids). Recorded in
   the SeedHQApps comment at the seed site.
2. **Seed mechanics:** grants seed only on first registration of the app row (CTE
   over the hq_apps INSERT's `RETURNING`), so `SeedHQApps` running on every startup
   cannot resurrect grants an operator revoked. Seeded set: `marketing` →
   roles admin, manager, team_member (§16 table — scan/redeem min role is
   team_member); `marketing-offline-override` → role admin only ("seeded true for
   admin users"; managers get it by explicit grant, per the entitlement-not-role
   semantic).
3. **Create/stats gates "inside the handler":** no marketing `/api/v1` handler exists
   tonight (Campaigns / Subscribers / Stats are labeled placeholders; the scanner's
   backend is Cards 5–7), so there is nothing to mount yet. The obligation is carried
   forward as the two tripwired `NA_WITH_REASON` entries in the parity spec — the
   suite reds the moment a marketing route lands ungated — plus the seed-site comment
   naming the §16 gates (view/create campaigns & stats = manager-tier, enforced
   in-handler when those endpoints land).
4. **users.html untouched:** the entitlement row inherits the standard gated-tab
   nudge ("Grant Marketing too…"). Its substance is correct for the entitlement — an
   offline-override grant without the marketing app grant is unreachable — so the
   card accepts the shared copy rather than special-casing the renderer.
5. **Placeholder copy** (mine per the PARK note): each placeholder section is a card
   with a "Soon" badge naming what it will hold, mirroring index.html's placeholder
   convention.
