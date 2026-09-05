# Merge intent — Card 5 · `camera-scanner-decode` (run 20260905)

Branch: `wo-camera-scanner-decode` (cut from the concurrent-track base @ 9a77f90 — includes
Card 1's marketing shell + permission seed, Card 7's POST /api/v1/marketing/redeem, Card 2's
pull replicas + widened vendor bundle, Card 3's scan_attempts push module, Card 4's sync
clock). Card authority: slate-20260905 Card 5. Footprint: scanner UI (`marketing.html` +
`marketing/`) + `sw.js` — plus the two mechanically-forced companions stated below
(build-sw.js globs and backend/Dockerfile copies move WITH a precache addition, decision 59;
tests/marketing.spec.js is the slate-named home for this card's assertions).
Spike authority: `.night-crew/knowledge/spikes/activity-c-scanner-screen/camera-scanner-decode.md`
(decode → extract → WebCrypto → replica-key chain proven in real Chromium; vendored
html5-qrcode single file, 375,364 B). Design authority: docs/qr-offline-redemption-handoff.md
§12/§4/§10, F3/F5/F6, D-KR3.

## Shared files touched

- `marketing.html` (EDIT — Card 1's shell) — **what of Card 1's page must survive:** the
  four-tab structure (`#t1..#t4` / `#s1..#s4`, `show(n)`), the `soon-card` placeholders for
  Campaigns/Subscribers/Stats, the auth probe, the back link, and the `#scanner-host` id
  (Card 1's stated mount contract — kept, now populated). This card replaces
  `#scanner-host`'s placeholder CONTENT with the scanner UI (DOM contract below), adds
  ok/bad CSS tokens (mirroring workflows.html's palette) + scanner styles to the inline
  `<style>`, and appends two script tags before `log.js`/`ptr.js`:
  `<script src="lib/html5-qrcode.min.js" defer>` (classic single-file vendored lib — the
  spike's proven pattern) and `<script type="module" src="marketing/scan-page.js">`.
- `lib/html5-qrcode.min.js` (NEW, vendored) — the spike's exact file (375,364 bytes),
  copied from the spike dir on this box, NO network fetch. 🛑 Recorded deviation: the
  slate says "the spike's committed file", but the spike's own `.gitignore` lists
  `web/html5-qrcode.min.js` — it was never committed anywhere; it exists only in the main
  checkout's working tree at
  `hq/.night-crew/spikes/activity-c-scanner-screen/camera-scanner-decode/web/`. This card
  copies THAT file (byte size matches the spike record exactly: 375,364) and commits it —
  the first committed copy in any tree. Placed in `lib/` (the single-file classic-script
  vendored-lib precedent, `lib/chart.umd.min.js`), NOT `vendor/` — `vendor/` is the
  build-vendor.sh bundle pipeline and `tests/sw-manifest.spec.js:82` pins the vendor/
  precache set to exactly `['vendor/rxdb.bundle.js']`; `lib/` keeps that pin untouched and
  needs zero new Dockerfile lines (`COPY lib ./lib` + `cp -r ../lib` already stage it).
- `marketing/scanner.js` (NEW) — pure scan logic, dependency-injected like the
  `marketing/sync/` family (zero imports): `extractToken` (#10 reader), `parseEmbeddedOffer`
  (D-KR3 descriptor reader), `createTokenHasher` (WebCrypto SHA-256 → lowercase hex, memoized),
  `createScanResolver` (the resolution-order + F3 state machine), `makeSerializedEnqueue`
  (the per-code enqueue serializer Card 6 must call). Full surface under "What must survive".
- `marketing/scan-page.js` (NEW) — the browser wiring: imports the vendored bundle
  (`../vendor/rxdb.bundle.js`), `./scanner.js`, `./sync/replicas.js`,
  `./sync/push-replication.js`, `./sync/clock.js`; creates the Dexie RxDatabase +
  collections, the one sync clock (persist/initialState round-trip via localStorage),
  the camera/file-scan UI wiring (state-first render, ONE delegated click listener), and
  exposes `window.MarketingScan` (surface below — Card 6's entry point AND the test seam).
- `build-sw.js` (EDIT) — three globPatterns added: `'lib/html5-qrcode.min.js'`,
  `'marketing/*.js'`, `'marketing/sync/*.js'` (single-level on purpose — never sweeps
  `marketing/sync/harness/` or `marketing/package.json`). No transform, canary, or
  runtimeCaching change.
- `backend/Dockerfile` (EDIT — outside the nominal footprint, mechanically forced by
  decision 59: a glob without the matching copy bricks the SW install, and
  tests/sw-manifest.spec.js's Obligation-5 guard reds on the mismatch) — builder-stage
  `COPY marketing/*.js ./marketing/` + `COPY marketing/sync/*.js ./marketing/sync/`, and
  staging `cp -r ../marketing cmd/server/public/`. No Go source touched; no Go gates owed.
- `sw.js` (REGENERATED, after the page/module/lib commits — build-sw.js reads git HEAD)
  — **precache count moves 32 → 39, all seven deliberate additions enumerated:**
  `lib/html5-qrcode.min.js`, `marketing/scanner.js`, `marketing/scan-page.js`,
  `marketing/sync/replicas.js`, `marketing/sync/pull-replication.js`,
  `marketing/sync/push-replication.js`, `marketing/sync/clock.js`. Regenerated with THIS
  worktree's lockfile-true `npm ci` env (Card 2's toolchain finding — main checkout's stale
  workbox produces a spurious reshape).
- `CLAUDE.md` (EDIT) — the precache-count line, 32 → 39, same change set as sw.js.
- `tests/marketing.spec.js` (EDIT — Card 1's file, per the slate) — a new
  `describe('Camera scanner decode …')` appended; Card 1's tests untouched.
- `tests/fixtures/qr-*.png` (NEW) — committed QR fixtures generated offline with the
  spike's own `qrcode` node module (no network): fixture-1 (seed token, replica path +
  hash-equals-seed-literal), fixture-4 (seed REDEEMED token, F3), embedded (un-synced token
  + `#o=` descriptor).
- `.night-crew/knowledge/roadmap.md` — one-line flip `camera-scanner-decode`
  PLANNED → DRAFTING (overnight-20260905), the run's convention. No other line moves.
- `.night-crew/runs/2026-09-05-autonomous/merge-intents/camera-scanner-decode.md` — this
  file (amended as evidence lands); `card5-*.log` — committed whole gate logs.
- `index.html`, `supabase/`, `backend/**/*.go`, `night-crew.toml`, `package.json`,
  `vendor/`, `sync-rxdb/`, other `tests/*.spec.js` — **nothing here.** (`night-crew.toml`:
  marketing.html/sw.js stay undeclared seams → full-suite fallback, deliberate; a
  `marketing` seam entry stays Card 6's call.)

## What must survive any merge — the surface Card 6 builds against

**DOM contract (inside `#scanner-host`, ids stable — Card 6 extends the SAME Scan section):**

- `#scan-status` — one-line sync/connectivity status. Card 6's reachability probe (#13)
  OWNS this line's truth; tonight it never claims liveness (see Engineering calls).
- `#scan-camera-view` — the html5-qrcode render target (camera video / file-scan work area).
- `#scan-controls` — `[data-action="start-camera"]` button + the `#scan-file` input
  (`type=file`, the library's file-scan path — ALSO the headless test drive).
- `#scan-result` — the result card. Attributes are the machine-readable outcome:
  `data-kind` ∈ `offerReady | embeddedOffer | unknownCode | spentLocally | deferToServer |
  expiredLocally | invalidPayload | decodeError`, `data-token-hash` = the on-device
  SHA-256 hex (replica key), `data-source` ∈ `replica | embedded` when offers show.
  Inside: `#scan-offer-list` (one `.offer-row[data-code-id]` per offer — DISPLAY ONLY, no
  pick affordance, F5), `.result-note` (the F5 "apply in Toast by hand" line / embedded
  trust note), and **`#scan-submit-slot` — an EMPTY div. Card 6 mounts the order-number
  entry + submit flow HERE** (rendered for every kind except the hard offline rejects
  `spentLocally` / `expiredLocally` / `invalidPayload` / `decodeError`).
  `[data-action="scan-again"]` resets to idle (F6's session dedupe lives behind it).
- ONE delegated click listener on `#scanner-host` routing via `data-action` (repo
  convention) — Card 6 adds actions to the SAME listener's router, not new listeners.

**`window.MarketingScan` (set by scan-page.js; `ready` resolves after boot):**

- `ready` (Promise), `db`, `collections` (`codes`, `offers`, `scan_attempts`), `clock`
  (Card 4's instance — persisted to localStorage `hq_marketing_clock_v1` on every capture,
  rebooted from it via `initialState`), `resolver` (`createScanResolver` instance),
  `scanText(payload)` (decode-less entry — resolve + render a payload string),
  `setOnlineProbe(fn)` — **Card 6's plug-in point for the real reachability signal**;
  tonight's default is `() => false` (never claim liveness you can't see),
  `enqueue(fields, opts)` — the SERIALIZED enqueueAttempt wrapper (one in-flight enqueue
  per code_id; Card 6 must submit through THIS, never raw `enqueueAttempt` — the landed
  find-then-insert dedupe is not atomic), `startSync({restUrl, bearer})` + `resync()` —
  replica wiring (clock threaded into deps; Card 6/later provisioning supplies coordinates).
- `marketing/scanner.js` exports (`extractToken`, `parseEmbeddedOffer`,
  `createTokenHasher`, `createScanResolver`, `makeSerializedEnqueue`) and the result-kind
  taxonomy above — renaming or re-shaping breaks Card 6's submit flow and this card's tests.
- The `#scanner-host` id (Card 1's contract), Card 1's tests, the four-tab shell.
- The seven-entry precache addition + the build-sw.js globs + the Dockerfile marketing/
  copies — dropping any one bricks the SW install for returning clients (decision 59).
- The roadmap flip; the `card5-*.log` evidence; this intent.

## What is safe to drop

- Nothing in this branch is scratch. (Test fixtures are committed inputs to the specs.)

## Red-first

Captured to `card5-red.log` (committed `d6c2d30`) BEFORE the production files existed in
the tree (they land in the NEXT commit, `a47ad41` — git history is the chronology, the
family pattern). Greenfield: the full describe — decode-from-image + hash-equals-seed-
literal, resolution order (replica-first / embedded fallback / unknownCode), F3
offline-reject vs online-defer, offset-clock expiry + persist/initialState round-trip,
hash caching, serialized enqueue, F6 re-scan, camera error state — ran against Card 1's
shell and redded **14/14, EXIT=1**, every failure the same behavior-absent shape
(`window.MarketingScan` never appears; 30s waitForFunction timeout — verified: 14
identical error lines, zero harness errors).

🛑 Process deviation, disclosed in-log and in the commit: a FIRST red attempt was
invalidated and re-run. The implementer wrote production files into the working tree
while that attempt's webServer (`STATIC_DIR=../` — serves the LIVE tree) was running, so
its last 6 tests raced green against mid-run files. The production files were stashed,
the red leg re-run start-to-finish against the untouched pre-change tree, and THAT clean
capture is the committed log. Lesson recorded for future cards: never modify the worktree
while a webServer-based Playwright leg is live.

## Gate evidence (run in this worktree; full logs committed beside this file)

- **RF** — `card5-red.log` (above): 14/14 red on pre-change tree, EXIT=1. Green: the same
  14, targeted run on the implemented tree — **14 passed (12.0s), first attempt** (also
  visible inside the full-suite log).
- **G2 (Playwright, FULL suite — marketing.html + sw.js are undeclared seams → full
  fallback, deliberate)** — `card5-e2e.log`, HEAD `8505041`: `npx bddgen && TEST_PORT=3105
  TEST_DB_NAME=hq_test_e2e_c5 npx playwright test --retries=0` → **850 passed / 1 failed /
  6 skipped (18.9m), EXIT=1, exactly one summary block** (grep-verified: one `passed (`
  line). The 1 red, judged against the slate's armed baseline: `workflows.spec.js:1202`
  **DBL-05** — a slate-NAMED pre-existing base red ("pre-existing on base — … workflows
  DBL-05"). The other named baseline reds (sw-api-cache-partition B1-XT-01/-02/-05) were
  GREEN this run — they flap, same shape Card 2 recorded. **No failure in this run is
  attributable to this diff**; all 14 new scanner tests green inside the full run.
- **G4** — `card5-g4.log`, HEAD `3272a18`: `node build-sw.js` twice — both EXIT=0, **39
  files precached (2682.0 KB)**, tree clean after pass 2 (committed sw.js reproduced
  byte-identical — lockfile-true npm ci env, workbox-build 7.4.1 per Card 2's toolchain
  finding). Reachability: 26 files parsed, 41 refs resolved, 0 outside. Version parity
  **1.6.2 ≡ 1.6.2 ≡ 1.6.2** (version.go Frontend / package.json / version.json) — no
  bumps tonight.
- **G1 / G2 (Go)** — n/a: no `.go` file touched (backend/Dockerfile only — asset staging
  lines, mechanically forced by decision 59; verified by the diff).
- **Harness leg (optional per the slate)** — not run, stated: the slate marks it
  "Optional if your Playwright tests already prove it against fixtures", and they do —
  resolution order (replica-first, embedded fallback, unknownCode) and both F3 branches
  are pinned by the committed Playwright tests against local-replica fixtures mirroring
  the seed literals; the replica MECHANISM itself is Card 2/3/4's substrate-harness-proven
  surface, untouched by this card (their modules are wired, not edited — zero diff under
  `marketing/sync/`).
