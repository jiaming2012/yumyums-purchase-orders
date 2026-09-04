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
