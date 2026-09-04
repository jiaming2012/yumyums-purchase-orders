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
