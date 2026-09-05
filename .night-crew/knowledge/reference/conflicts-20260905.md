# Conflict log — run 20260905

Every merge to `overnight-20260905` gets an entry, clean or conflicted, so an empty log
never reads as "no conflicts" when it means "the logging never ran" (§15ad.66).

## Merge 1 — `wo-marketing-tile-and-page` (Card 1, Wave 0)

- **Cards involved:** Card 1 only (first merge of the night; run branch at the slate
  sign-off commit `0670798`).
- **Files/hunks:** clean merge, no conflicts — `index.html` (tile + TILE_SLUGS),
  `marketing.html` (NEW shell, 4 sub-sections), `backend/internal/db/db.go` (SeedHQApps
  seed + grants via RETURNING CTE), `backend/internal/auth/marketing_seed_test.go` (NEW),
  `tests/marketing.spec.js` (NEW), `tests/grant-enforcement-parity.spec.js` (2
  NA_WITH_REASON entries + PARITY-NA-FRESH tripwire), `sw.js` + `workbox-cdd33147.js`
  (precache 31→32, deliberate, stated), `CLAUDE.md` precache line, roadmap flip
  (PLANNED→DRAFTING), merge-intent + 8 `card1-*.log` evidence files.
- **Intents read:** Card 1's merge-intent only (no other side exists on a first merge).
- **Resolution:** none needed.
- **Gate result after merge:** tree content identical to the reviewed card branch —
  G1/G2(Go)/G2(Playwright ×2 full)/G4/RF as committed in `card1-*.log`; G6
  **PASS-WITH-NOTES** (independent base-tree leg `card1-g6-basereds.log`: all 4 recurring
  e2e reds pre-existing — B1-XT-01/-02/-05 fail on base identically; DBL-05 fails
  alone-vs-alone on BOTH trees, order-sensitive, real dev bug → triage). Sole Go red
  `TestJWTBridgeRLS` pre-existing (`card1-baseline-jwtbridge.log`). Notes to triage:
  §16:436 (`offline_override` = manager grant) now disagrees with shipped fork-#12
  resolution (entitlement, seeded admin-only) — annotate the doc; PARITY-NA-FRESH
  tripwire matches route paths containing the slug only (redemption-mounted routes would
  not trip it — Cards 5–7 reviewers told); DBL-05 backlog entry owed.

## Merge 2 — `wo-gstate-arbitration-machine` (Card 7, Track D)

- **Cards involved:** Card 7 onto the Wave-0 tree (run branch at `73c36bc`; no other
  merge since — clean by construction).
- **Files/hunks:** clean merge, no conflicts — `backend/internal/redemption/` (NEW: 7
  source + 4 test files, 28 tests), `backend/internal/db/migrations/0077_race_lost_notifications.sql`
  (NEW numbered file; supabase/ untouched), `backend/go.mod`+`go.sum` (go 1.26.2 +
  gstate v0.3.1 + go-nanoid), `backend/Dockerfile` (golang:1.26-alpine, -ldflags path
  intact), `backend/cmd/server/main.go` (route wiring only — Card 1's seed untouched),
  `tests/grant-enforcement-parity.spec.js` (stale `marketing` NA entry deleted in the
  G6 fix round — pre-authorized by its own reason text), roadmap flip, merge-intent +
  card7-*.log evidence.
- **Intents read:** Card 7's merge-intent (endpoint contract for Card 6 recorded there:
  `POST /api/v1/marketing/redeem`, session auth + RequirePermission("marketing"),
  fail-closed 503 without arbiter env; F4 read-model home = HQ Postgres, migration
  0077 — engineering calls recorded). Card 1's consulted for the parity-spec surface —
  the fix round edits the spec Card 1 created; the NA deletion is the interaction the
  tripwire was built to force.
- **Resolution:** none needed.
- **Gate result after merge:** tree content identical to the reviewed branch. G6
  round 1 **FAIL** (cross-card PARITY-EQ/PARITY-NA-FRESH fired on the marketing route,
  `card7-g6-parity.log`; MaxRetries=0 in prod construction) → fix round `98e189e`
  (red-first regression on the literal Config{} path; MaxRetriesNone sentinel; 2 new
  notification-failure tests; parity+marketing specs 20/0 EXIT=0,
  `card7-fix-parity.log`) → G6 re-verify **PASS** (independent re-runs: build/vet 0,
  redemption 28/28, parity 20/0 on reviewer isolation). Full Go suite on bumped
  toolchain: sole red = base-proven `TestJWTBridgeRLS` (card7-g2-go.log). Notes to
  triage: no dedupe on `race_lost_notifications` (replay can duplicate a manager
  notification); F4's scan_attempts-status bullet owned by NO card tonight; server
  trusts client `offline_override` flag (F4 flagging only); next full-suite count
  expectation 566 with-subtest results (28 redemption).

## Merge 3 — `wo-rxdb-pull-replica` (Card 2, Track B)

- **Cards involved:** Card 2 (base `73c36bc`) onto the run branch that already carries
  Card 7 (Track D merged first under concurrent dispatch).
- **Files/hunks:** clean merge — Card 7 intersection is `.night-crew/knowledge/roadmap.md`
  only, different card lines (~256 vs ~352), auto-merged. New `marketing/sync/`
  (pull-replication.js, replicas.js, harness), `marketing/package.json`,
  `vendor/src/rxdb-hq-entry.mjs` + regenerated `vendor/rxdb.bundle.js` (+717 B, rxdb@17.4.0
  pin unchanged, surface widened for Cards 5/6), `sw.js` (count 32, bundle revision +
  disclosed lockfile-true workbox reshape), spike-ledger GAP-1 `validated:` line,
  roadmap flip, merge-intent + 5 gate logs. 🛑 **Control-loop error, disclosed:** G6's
  independent-harness log (`card2-g6-harness.log`, EXIT=0, all legs held, 105 ms
  propagation) was left untracked in the card worktree and lost when the worktree was
  removed before copying it out. The G6 report text in HANDOFF is the surviving record
  of that re-run; the implementer's own `card2-harness.log` (same legs, EXIT=0) is
  committed.
- **Intents read:** Card 2's merge-intent (unwired-tonight call; entry points + vendor
  surface named for Cards 5/6; sw.js stale-toolchain hazard). Card 7's consulted —
  no overlap beyond roadmap.
- **Resolution:** none needed. 🛑 **Deliberate: no sw.js regen in the main checkout
  after this merge** — the main checkout's node_modules carries stale workbox 7.3.0
  (Card 2's triage finding); a regen here would revert the lockfile-true output with a
  spurious whole-file reshape. G4 idempotency was proven in the card worktree on
  lockfile-true deps (card2-g4.log + G6's own regen, tree clean).
- **Gate result after merge:** tree content identical to reviewed branch. Harness
  EXIT=0 (implementer first attempt + G6 independent re-run, 105 ms propagation);
  e2e 828/9/6 fully accounted (4 known + 5 flakes proven by targeted same-tree re-run
  5/5); G4 idempotent at count 32; RF reds proven to predate the fix in history.
  G6 **PASS-WITH-NOTES**. Notes to triage: (1) **commit-order skew** — `updated_at`
  is txn-START time, so a long transaction can commit rows behind an advanced
  checkpoint, permanently invisible to replicas (RESYNC never rewinds) — backlog
  candidate (checkpoint rewind on RESYNC or commit-ordered cursor); (2) CHANNEL_ERROR/
  JWT-expiry dead-nudge-stream modeling is OWED BY CARDS 5/6 (no fallback poll);
  (3) out-of-window rows never `_deleted` locally — unbounded growth, no owner yet;
  (4) merge-intent omits `marketingCollectionSpec()` from its API list (wiring recipe
  in replicas.js:26-28 covers it).

## Merge 4 — `wo-scan-attempts-push-conflict` (Card 3, Track B)

- **Cards involved:** Card 3 (merge-base `786be13`) onto the run branch. Intersection
  with landed cards: `roadmap.md` only; Card 2's modules byte-untouched (new sibling
  files only).
- **Files/hunks:** clean merge — NEW `marketing/sync/push-replication.js`, NEW push
  harness (`push-run.sh`, `push-harness.mjs`), spike-ledger GAP-1 `validated:` line,
  roadmap flip, merge-intent + card3-red.log + card3-harness.log (+ G6's
  `card3-g6-harness.log`, copied out and committed with this entry). No page/sw.js/
  backend/tests/night-crew.toml → Playwright/Go/G4 legitimately n/a.
- **Intents read:** Card 3's merge-intent (API surface for Cards 5/6: SCAN_ATTEMPTS_SCHEMA,
  scanAttemptsCollectionSpec, enqueueAttempt, makePushHandler, startScanAttemptsReplica;
  queue shape §4 + 5 local-only fields, stripped from the landing body — verified).
- **Resolution:** none needed.
- **Gate result after merge:** tree identical to reviewed branch. Harness EXIT=0
  (implementer first attempt + G6 independent re-run incl. psql server-side
  enumeration); RF red proven pre-fix in history (module absent at red commit).
  G6 **PASS-WITH-NOTES**. Findings to triage + Cards 5/6:
  (1) **BELT-2 UNBOUNDED ACROSS SESSIONS (demonstrated)** — after local-store loss
  (iOS PWA eviction), a device re-scanning its own earlier-redeemed code gets the new
  attempt arbitrated ACCEPTED: 2 accepted server rows for one code, UI shows
  "redeemed ✓" where F3 says "already used" — double-serve + §9 audit corruption.
  Slate's own prescribed belt shape → design gap for TRIAGE, not implementer error.
  Candidate fix: accept only when `redeemed_at >= scanned_at` (skew slack), else
  rejected with winner = own device. (2) `enqueueAttempt` dedupe not atomic
  (find-then-insert) — two rapid same-code scans can both insert and both be accepted;
  Card 5's wiring should serialize enqueue per code. (3) A stuck attempt
  head-of-line-blocks the batch (RxDB push contract; retries resumable, no data loss).
  (4) merge-intent doesn't say what Card 6 renders for `deduped:true`; ties into (1).
  (5) `push-run.sh` header documents the red-mode exit contract inverted vs the
  implementation (exit 1 IS the demonstrated red) — comment fix on a follow-up branch.

## Merge 5 — `wo-clock-offset-on-sync` (Card 4, Track B)

- **Cards involved:** Card 4 (merge-base `37aa1a6`) onto the run branch.
- **Files/hunks:** ONE CONFLICT — `timings.log`, append-append (control loop's card-3
  line vs the card's own timing line), resolved by UNION (both lines kept, chronology
  preserved). All else clean: NEW `marketing/sync/clock.js` + clock harness siblings;
  EDITs confined to `makePullHandler` (capture after HTTP 200, before body parse) and
  `startReplica` (clock threading, explicit-now precedence); push-replication.js
  zero-diff (audited: no expiry decisions); roadmap flip; merge-intent + 5 logs
  (+ G6's `card4-g6-harness.log`, copied out and committed here).
- **Intents read:** Card 4's (offset beside — not inside — Card 2's checkpoint;
  Cards 5/6 API: `clock.isExpired()`/`clock.now()`, persist/initialState round-trip
  binding on their wiring). Cards 2/3's consulted — their harnesses byte-untouched and
  re-run green on this tree (card4-regression-c2/c3.log, both EXIT=0).
- **Resolution:** union of the two timings appends; no code hunk conflicted, tree code
  content identical to the G6-reviewed branch → G1/G2 re-run not owed (conflicted path
  is a run-artifact text file outside every gate's subject; stated deliberately).
- **Gate result after merge:** harness EXIT=0 (implementer + G6 independent re-run);
  RF both-skew-signs red proven pre-fix in history. G6 **PASS-WITH-NOTES**. Notes to
  triage: (a) ms-level offset figures are second-boundary artifacts of the whole-second
  Date header (G6 re-run: 675/708 ms) — never read as sub-second capability; harmless
  at the 2-day threat model; (b) the slate's done_when literal alone is
  non-discriminating (naive-fast also rejects expired) — the harness's
  valid-code-still-resolves assertion is load-bearing and must survive into E-KR4's
  pinning test; (c) marketing/sync files carry no [e2e.seams] entry — Playwright-n/a
  rests on the slate's explicit gate assignment (precedent stated, not drifted);
  (d) pre-capture isExpired degrades to naive until first successful pull — residual
  named here since the merge-intent names only the window-bound half.

## Merge 6 — `wo-camera-scanner-decode` (Card 5, Track C)

- **Cards involved:** Card 5 (merge-base `9a77f90`) onto the run branch.
- **Files/hunks:** clean merge — `marketing.html` (scanner UI in `#scanner-host`; Card 1's
  four-tab shell survives, its 6 tests green in G6's re-run), NEW `marketing/scanner.js` +
  `marketing/scan-page.js`, NEW `lib/html5-qrcode.min.js` (375,364 B, sha256
  660b12437b1d747e3e68b8be0685c08cb728140110ad213f167b14b66f8b1d8e — G6
  registry-verified byte-identical to npm html5-qrcode@2.3.8, matching the spike
  lockfile's sha512; content-scanned clean), `build-sw.js` narrow globs,
  `backend/Dockerfile` COPY pairing (decision 59; Card 7's builder bump intact),
  `tests/marketing.spec.js` +14, 3 QR fixtures, `sw.js` + CLAUDE.md **32→39** (same
  commit), roadmap flip, merge-intent + card5-red/e2e/g4 logs (+ G6's `card5-g6.log`,
  copied out, committed here).
- **Intents read:** Card 5's (Card 6 contract: `#scan-submit-slot` in
  `#scan-result[data-kind|data-token-hash|data-source]`, `window.MarketingScan` API,
  serialized-enqueue MUST). Cards 1/2/3/4 intents consulted; `marketing/sync/`
  zero-diff.
- **Resolution:** none needed.
- **Gate result after merge:** tree identical to reviewed branch. Full suite 850/1/6
  (lone red = pre-existing DBL-05; B1-XT trio flapped green); G6 targeted re-run
  20/20; G4 exit 0 ×2 at count 39, byte-identical second pass; RF 14/14
  behavior-absent reds proven pre-fix (an invalidated first red attempt disclosed and
  redone clean). G6 **PASS-WITH-NOTES**. Notes to triage: (1) record the vendored
  lib's sha256 in the spike/extraction record; (2) **F6 is partial after Card 5**
  (re-show only) — Card 6 owns full session semantics, told in its prompt; (3) the
  attended live-camera check stands (HANDOFF next-actions); (4) micro-note: token
  regex is first-match — a hypothetical double-`/r/` payload would pick a different
  token than the spike's end-anchored regex (not a contract shape).
