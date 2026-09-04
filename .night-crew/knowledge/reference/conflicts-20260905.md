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
