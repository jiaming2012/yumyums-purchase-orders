---
phase: 21-cogs-in-sales-processor-report-receipt-completeness-gate-bef
verified: 2026-06-02T18:00:00Z
status: human_needed
score: 18/18 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Live curl 503 path against running server with HQ_INVENTORY_SERVICE_TOKEN unset"
    expected: "Start backend with env var unset, then curl http://localhost:8080/api/v1/inventory/period-summary?from=2026-05-25&to=2026-05-31 returns 503 + {\"error\":\"service_token_not_configured\"}, server log shows the WARNING line at startup"
    why_human: "Plan 21-03 <verification> block explicitly listed this as a checkpoint step; not executed live by the executor (SUMMARY documents this as a deferred human-verify step). Middleware unit tests cover the 503 code path in isolation, but the env-load + WARNING-log path through main.go has not been exercised on a running binary."
  - test: "Live curl 401 path against running server"
    expected: "Start backend with HQ_INVENTORY_SERVICE_TOKEN=test-token, then curl WITHOUT Authorization header returns 401 + {\"error\":\"unauthorized\"}. Same again with a wrong bearer token also returns 401."
    why_human: "Same as above — chi router + middleware integration is unit-tested but not live-tested end-to-end through a real server process."
  - test: "Live curl 200 path against running server"
    expected: "Start backend with HQ_INVENTORY_SERVICE_TOKEN=test-token, populate hq DB with at least one purchase_event, then curl with Authorization: Bearer test-token returns 200 + JSON matching PeriodSummary contract (from/to/cogs_excl_tax/cogs_incl_tax/purchase_event_count/completeness)."
    why_human: "Confirms the chi route mount, middleware chain, and handler all work end-to-end on a running server. Plan 21-03 SUMMARY explicitly defers this to phase-level approval."
  - test: "(Optional) Re-run integration tests against hq_test DB"
    expected: "DB_TEST_URL='postgres://yumyums:yumyums@192.168.8.164:5433/hq_test?sslmode=disable' go test ./internal/inventory -run TestPeriodSummary -v shows 6 PASS subtests."
    why_human: "Cached PASS result confirmed in this verifier's run, but the DB lives on a remote Tailscale/LAN box (Windows dev Postgres). Re-execution requires network reachability that may not persist between sessions."
---

# Phase 21: COGS in sales-processor report + receipt completeness gate Verification Report

**Phase Goal:** Display tax-excluded COGS on the sales-processor weekly PDF/CSV report, and hard-fail payroll generation when HQ receipts for the period are not fully ingested + reviewed + catalog-linked.

**HQ-side scope (this repo only):** HQ endpoint `GET /api/v1/inventory/period-summary?from=&to=` returning cogs_excl_tax, cogs_incl_tax, purchase_event_count, and completeness block (ready bool + pending_review IDs + unlinked line_item IDs), behind a service-token middleware. Sales-processor consumer is OUT OF SCOPE for this repo (handed off).

**Verified:** 2026-06-02T18:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification.

## Goal Achievement

### Observable Truths (18 across 3 plans)

#### Plan 21-01 — Types + Handler

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1.1 | PeriodSummary and CompletenessBlock JSON types exist and serialize as documented | VERIFIED | `backend/internal/inventory/types.go:153-175` — both structs exported, all 9 snake_case json tags present (`from`, `to`, `cogs_excl_tax`, `cogs_incl_tax`, `purchase_event_count`, `completeness`, `ready`, `pending_review_ids`, `unlinked_line_item_ids`); `Completeness` is non-pointer nested struct as required |
| 1.2 | PeriodSummaryHandler returns COGS aggregates and completeness data for a date range | VERIFIED | `backend/internal/inventory/handler.go:1064-1189` — handler factory closes over `*pgxpool.Pool`, parses from/to, runs three SQL queries, assembles `PeriodSummary{}` and writes via `writeJSON` |
| 1.3 | COGS uses purchase_events.event_date (DATE); completeness uses pending_purchases.created_at AT TIME ZONE 'America/Chicago' | VERIFIED | handler.go:1092 `WHERE event_date BETWEEN $1 AND $2` (plain DATE); handler.go:1118 `WHERE (created_at AT TIME ZONE 'America/Chicago')::date BETWEEN $1 AND $2`; handler.go:1151 `WHERE pe.event_date BETWEEN $1 AND $2` |
| 1.4 | Discarded pending_purchases (discarded_at IS NOT NULL) are excluded from pending_review_ids | VERIFIED | handler.go:1120 `AND discarded_at IS NULL`; integration test subtest 4 ("discarded pending purchase does NOT block ready") passes against hq_test DB |
| 1.5 | ready = true only when both pending_review_ids and unlinked_line_item_ids are empty | VERIFIED | handler.go:1182 `Ready: len(pendingIDs) == 0 && len(unlinkedIDs) == 0` |
| 1.6 | ROUND(..., 2) applied to NUMERIC(10,4) price aggregates | VERIFIED | handler.go:1095 `ROUND(COALESCE(SUM(pli.quantity * pli.price), 0)::numeric, 2)`; integration subtest 1 asserts exact 43.00 / 47.00 values (no rounding drift) |

#### Plan 21-02 — ServiceTokenMiddleware

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 2.1 | Rejects requests missing Authorization header with 401 | VERIFIED | service_token.go:33-35; `TestServiceTokenMiddleware/missing_Authorization_header_→_401` PASS |
| 2.2 | Rejects malformed (non-'Bearer ') headers with 401 | VERIFIED | service_token.go:33 `strings.HasPrefix(authHeader, prefix)`; `TestServiceTokenMiddleware/header_without_Bearer_prefix_→_401` PASS |
| 2.3 | Rejects wrong tokens with 401 using crypto/subtle.ConstantTimeCompare | VERIFIED | service_token.go:38 `subtle.ConstantTimeCompare([]byte(provided), []byte(expectedToken)) != 1`; import `"crypto/subtle"` on line 4; `TestServiceTokenMiddleware/Bearer_prefix_with_wrong_token_→_401` PASS |
| 2.4 | Accepts the correct token and calls the next handler | VERIFIED | service_token.go:42 `next.ServeHTTP(w, r)`; `TestServiceTokenMiddleware/Bearer_prefix_with_correct_token_→_200,_next_called` PASS (nextCalled bool asserted true) |
| 2.5 | Returns 503 (fail-closed) when expectedToken is empty | VERIFIED | service_token.go:27-30 — 503 branch evaluated BEFORE reading Authorization header; `TestServiceTokenMiddleware/empty_expectedToken_(env_unset)_→_503,_next_NOT_called` PASS |
| 2.6 | Middleware error bodies never echo the provided or expected token; never log token material | VERIFIED | Only sentinel strings in `http.Error` calls (`{"error":"unauthorized"}`, `{"error":"service_token_not_configured"}`); `grep -n "log\." service_token.go` returns ZERO matches |

#### Plan 21-03 — Route wiring + integration test + docs

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 3.1 | GET /api/v1/inventory/period-summary is registered behind ServiceTokenMiddleware as a PEER (not child) of the cookie-auth chi group | VERIFIED | main.go:337-340 — new `r.Group(func(r chi.Router) { r.Use(auth.ServiceTokenMiddleware(serviceToken)); r.Get("/inventory/period-summary", ...) })` is at line 337-340 INSIDE `r.Route("/api/v1", ...)` (opens at line 301); the cookie-auth `r.Group(... auth.Middleware(pool, superadmins) ...)` opens at line 343-344 — sibling, not nested. AWK check: `OK: peer placement inside /api/v1 (ServiceTokenMiddleware line 338 < cookie-auth line 344)` |
| 3.2 | Server startup logs WARNING when HQ_INVENTORY_SERVICE_TOKEN is unset, and the endpoint returns 503 in that case | VERIFIED (code path) / human-verify (live curl) | main.go:273-278 `serviceToken := os.Getenv("HQ_INVENTORY_SERVICE_TOKEN"); if serviceToken == "" { log.Println("WARNING: HQ_INVENTORY_SERVICE_TOKEN not set ...") }`; 503 response is covered by Plan 21-02 unit test. Live curl on a running server deferred to human checkpoint. |
| 3.3 | Integration test exercises three scenarios against the real hq_test DB: ready=true, ready=false (pending), ready=false (unlinked) | VERIFIED | period_summary_test.go:170-250 — 3 named subtests; `go test ./internal/inventory -run TestPeriodSummary -v` shows all PASS (cached against `postgres://yumyums:yumyums@192.168.8.164:5433/hq_test`) |
| 3.4 | Integration test uses DB_TEST_URL from env and calls db.Migrate(pool) so the same schema as production is exercised | VERIFIED | period_summary_test.go:19 `dbURL := os.Getenv("DB_TEST_URL")`; line 33 `if err := db.Migrate(pool); err != nil { ... }` in TestMain |
| 3.5 | Discarded pending_purchases (discarded_at NOT NULL) inside the period do NOT block ready | VERIFIED | period_summary_test.go:252-271 "discarded pending purchase does NOT block ready" subtest PASS — proves SQL filter `AND discarded_at IS NULL` is the live exclusion clause |
| 3.6 | CLAUDE.md receipt pipeline section documents the new endpoint + env var with a pointer to the contract doc | VERIFIED | CLAUDE.md:35 — single tight bullet containing `/api/v1/inventory/period-summary`, `HQ_INVENTORY_SERVICE_TOKEN`, `503`, and `21-SALES-PROCESSOR-CONTRACT.md` — exactly as plan specified |

**Score:** 18/18 truths verified.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `backend/internal/inventory/types.go` | PeriodSummary + CompletenessBlock struct definitions with json tags | VERIFIED | exists, +24 lines (lines 153-175), all 9 JSON tags present, types exported |
| `backend/internal/inventory/handler.go` | PeriodSummaryHandler factory closing over *pgxpool.Pool | VERIFIED | exists, function at line 1064, all three SQL queries present with correct semantics, uses `writeJSON`/`writeError` helpers |
| `backend/internal/auth/service_token.go` | ServiceTokenMiddleware factory: (expectedToken string) -> func(http.Handler) http.Handler | VERIFIED | exists, 46 lines, exact signature on line 24, uses `crypto/subtle.ConstantTimeCompare`, zero `log.` calls |
| `backend/internal/auth/service_token_test.go` | Unit tests for all 5 paths | VERIFIED | exists, table-driven `TestServiceTokenMiddleware` with 5 cases, all PASS via `go test -v` |
| `backend/cmd/server/main.go` | HQ_INVENTORY_SERVICE_TOKEN env load + startup WARNING + new peer chi sub-group | VERIFIED | env load at lines 273-278, peer `r.Group` at lines 337-340 (before cookie-auth group at 343-344) |
| `backend/internal/inventory/period_summary_test.go` | Integration test with TestMain + 6 subtests + TRUNCATE + db.Migrate | VERIFIED | exists, 289 lines, `TestMain`, `resetFixtures` with `TRUNCATE ... CASCADE`, 6 `t.Run` subtests covering ready/pending/unlinked/discarded/bad-date/from-gt-to |
| `CLAUDE.md` | Period summary endpoint bullet with HQ_INVENTORY_SERVICE_TOKEN + contract pointer | VERIFIED | line 35 contains the documented bullet, all 4 required tokens present, existing Receipt pipeline bullet preserved on line 34 |

### Key Link Verification

| From | To  | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| PeriodSummaryHandler | purchase_events / purchase_line_items / pending_purchases | three pgx.Pool.Query calls | WIRED | One QueryRow + two Query loops in handler.go:1088-1170; all three tables hit via parameterized SQL ($1, $2) |
| ServiceTokenMiddleware | crypto/subtle.ConstantTimeCompare | direct stdlib call inside inner handler | WIRED | service_token.go:38 `subtle.ConstantTimeCompare([]byte(provided), []byte(expectedToken)) != 1` |
| main.go | auth.ServiceTokenMiddleware + inventory.PeriodSummaryHandler | chi.Group with r.Use + r.Get inside r.Route('/api/v1') | WIRED | main.go:337-340 — `r.Use(auth.ServiceTokenMiddleware(serviceToken))` + `r.Get("/inventory/period-summary", inventory.PeriodSummaryHandler(pool))` |
| period_summary_test.go | DB_TEST_URL + db.Migrate | pgxpool.New + db.Migrate(pool) helper in TestMain | WIRED | period_summary_test.go:19-37 — env override read, pool created, `db.Migrate(pool)` invoked before subtests run |
| main.go peer group placement | Cookie-auth group | r.Route("/api/v1") containing TWO sibling r.Group blocks | WIRED (peer) | ServiceTokenMiddleware group at line 337-340 precedes cookie-auth group at line 343-344, both as siblings inside the /api/v1 route opened at line 301. AWK verification: `OK: peer placement inside /api/v1 (ServiceTokenMiddleware line 338 < cookie-auth line 344)` |

### Data-Flow Trace (Level 4)

The handler is a pass-through aggregation: data flows from Postgres into the JSON response via parameterized SQL. All three queries hit real tables (no static returns).

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| PeriodSummary.COGSExclTax | cogsExcl (float64) | `pool.QueryRow(... SUM(quantity*price) ... FROM purchase_line_items WHERE purchase_event_id IN (events))` | Yes — real SUM aggregate on live table | FLOWING |
| PeriodSummary.COGSInclTax | cogsIncl (float64) | Same query: `(SELECT total FROM lines) + COALESCE(SUM(tax), 0) FROM events` | Yes | FLOWING |
| PeriodSummary.PurchaseEventCount | eventCount (int) | Same query: `COUNT(*) FROM events` | Yes | FLOWING |
| CompletenessBlock.PendingReviewIDs | pendingIDs ([]string{}) | `pool.Query("SELECT id::text FROM pending_purchases WHERE (created_at AT TIME ZONE 'America/Chicago')::date BETWEEN $1 AND $2 AND confirmed_at IS NULL AND discarded_at IS NULL")` | Yes — real query against live table | FLOWING |
| CompletenessBlock.UnlinkedLineItemIDs | unlinkedIDs ([]string{}) | `pool.Query("SELECT pli.id::text FROM purchase_line_items pli JOIN purchase_events pe WHERE pe.event_date BETWEEN $1 AND $2 AND pli.purchase_item_id IS NULL")` | Yes | FLOWING |
| CompletenessBlock.Ready | computed in Go | `len(pendingIDs) == 0 && len(unlinkedIDs) == 0` | Yes — derived from the two live lists | FLOWING |

The integration test (subtest 1 — ready=true) asserts exact arithmetic match `5*4.5 + 2*10.25 = 43.00` and `43.00 + (2.50 + 1.50) = 47.00` against the real DB, proving data flow end to end with no static-fallback paths.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Go build cleanly | `cd backend && go build ./...` | exit 0, no output | PASS |
| Go vet cleanly | `cd backend && go vet ./...` | exit 0, no output | PASS |
| ServiceTokenMiddleware tests (all 5 paths) | `cd backend && go test ./internal/auth/ -run TestServiceTokenMiddleware -v` | 5/5 subtests PASS in 0.378s | PASS |
| PeriodSummary integration tests (all 6 paths) | `cd backend && go test ./internal/inventory -run TestPeriodSummary -v` | 6/6 subtests PASS in 2.61s against `192.168.8.164:5433/hq_test` (cached) | PASS |
| Peer-vs-child placement of new chi.Group inside /api/v1 | awk scan scoped to `r.Route("/api/v1"` | OK: ServiceTokenMiddleware line 338 < cookie-auth line 344 | PASS |
| Zero `log.` calls in service_token.go (token-leak guard) | `grep -n "log\." backend/internal/auth/service_token.go` | empty (no matches) | PASS |
| All 7 task commits present in git log | `git log --oneline | grep -E "10d9682\|88de46d\|b4250fa\|fc52f9f\|8675df9\|bc89892\|0d7aa0f"` | 7 commits found | PASS |

### Requirements Coverage

ROADMAP Phase 21 line 113 explicitly states `Requirements: TBD (no formal REQ-IDs — see roadmap Acceptance bullets as the source of truth)`. All three plans declare `requirements: []` (empty) in frontmatter. There are no REQ-IDs to cross-reference against REQUIREMENTS.md, and `grep "Phase 21\|REQ-21" REQUIREMENTS.md` returns no matches.

The roadmap **Acceptance bullets** serve as the requirements proxy. They span both repos (HQ + sales-processor); the HQ portion is in scope for this verification:

| Acceptance Bullet (HQ-scope only) | Source | Status | Evidence |
| --------------------------------- | ------ | ------ | -------- |
| HQ endpoint returns cogs_excl_tax, cogs_incl_tax, purchase_event_count, and completeness block | ROADMAP Scope 1 | SATISFIED | PeriodSummary contract verified above (truths 1.1, 1.2) |
| HQ bearer-token check via HQ_INVENTORY_SERVICE_TOKEN | ROADMAP Scope 2 | SATISFIED | ServiceTokenMiddleware + env load + 503 fail-closed verified (truths 2.x, 3.2) |
| HQ integration test on period-summary (ready=true + each not-ready path) | ROADMAP Scope 6 (HQ portion) | SATISFIED | 6 subtests PASS (truth 3.3, 3.5) |
| hq/CLAUDE.md receipt pipeline section docs the new endpoint | ROADMAP Scope 7 (HQ portion) | SATISFIED | CLAUDE.md:35 bullet present (truth 3.6) |
| Discarded pending_purchases count as resolved | ROADMAP Constraints | SATISFIED | SQL filter + subtest 4 verified (truths 1.4, 3.5) |

Sales-processor-side acceptance bullets (1.HQ-only portion of Scope 3, all of Scopes 4, 5; and the three top-level Acceptance scenarios that involve `--force-payroll` / PDF / exit codes) are OUT OF SCOPE per the user's instructions and per ROADMAP line 95-97 ("HANDED OFF to sales-processor team").

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| backend/internal/auth/service_token.go | 28, 34, 39 | http.Error sets Content-Type: text/plain while body is JSON | Info (already flagged in 21-REVIEW.md as WR-02) | Strict JSON-only clients may surface a content-type mismatch on 401/503. Not a goal-blocker; documented as known follow-up. |
| backend/internal/inventory/handler.go | 1127, 1159 | Double `defer rows.Close()` in same scope | Info (flagged in 21-REVIEW.md as IN-04) | Minor — first rows handle held until function exit instead of being explicitly closed before second query. No correctness impact at expected traffic (1 call/week). |
| backend/internal/inventory/period_summary_test.go | 155-160 | `boolByte()` helper is indirect | Info (flagged in 21-REVIEW.md as IN-02) | Style only — no correctness impact. |
| backend/internal/inventory/handler.go | (entire PeriodSummaryHandler) | No empty-range subtest (could mask future regression to pgx.ErrNoRows on empty events) | Info (flagged in 21-REVIEW.md as WR-01) | Current behavior correct; lock-in test recommended but not blocking. |
| (none) | (none) | TODO/FIXME/PLACEHOLDER in new files | None | grep on new files returns zero TODO/FIXME/HACK/PLACEHOLDER markers |
| (none) | (none) | Empty return / stub returns | None | grep on new files returns zero `return null|return \{\}|return \[\]|=> \{\}` patterns; all handler/middleware code paths return real data or sentinel error bodies |
| (none) | (none) | console.log / debug-only handlers | None | Go side; no debug-only console output found |

The 4 Info-level patterns above all match findings already documented in `21-REVIEW.md`. None of them are goal-blockers — they are quality/maintainability follow-ups. The 3 Warnings flagged in `21-REVIEW.md` (WR-01, WR-02, WR-03) are similarly non-blocking quality items per the reviewer's own classification.

### Human Verification Required

The four items listed in YAML frontmatter under `human_verification`. In summary:

#### 1. Live curl 503 path (env var unset)

**Test:** With `HQ_INVENTORY_SERVICE_TOKEN` unset, start the backend (`cd backend && go run ./cmd/server`) and run:
```
curl -sv 'http://localhost:8080/api/v1/inventory/period-summary?from=2026-05-25&to=2026-05-31'
```
**Expected:** HTTP 503, body `{"error":"service_token_not_configured"}`. Server log shows `WARNING: HQ_INVENTORY_SERVICE_TOKEN not set — /api/v1/inventory/period-summary will return 503` at startup.
**Why human:** Plan 21-03 `<verification>` block explicitly listed this as a checkpoint step; not executed live by the executor (SUMMARY documents it as deferred). Middleware unit tests cover the 503 code path in isolation, but the env-load + WARNING-log path through main.go has not been exercised on a running binary.

#### 2. Live curl 401 path (missing / wrong bearer)

**Test:** Start backend with `HQ_INVENTORY_SERVICE_TOKEN=test-token go run ./cmd/server`. Then:
```
curl -sv 'http://localhost:8080/api/v1/inventory/period-summary?from=2026-05-25&to=2026-05-31'
curl -sv -H 'Authorization: Bearer wrong' 'http://localhost:8080/api/v1/inventory/period-summary?from=2026-05-25&to=2026-05-31'
```
**Expected:** Both return HTTP 401, body `{"error":"unauthorized"}`.
**Why human:** Same as above — chi router + middleware integration is unit-tested but not live-tested end-to-end through a real server process.

#### 3. Live curl 200 path (correct bearer + real DB)

**Test:** Same server. Ensure at least one row in `purchase_events` for the date range. Then:
```
curl -sv -H 'Authorization: Bearer test-token' 'http://localhost:8080/api/v1/inventory/period-summary?from=2026-05-25&to=2026-05-31'
```
**Expected:** HTTP 200, body is JSON matching the PeriodSummary contract (from/to/cogs_excl_tax/cogs_incl_tax/purchase_event_count/completeness{ready/pending_review_ids/unlinked_line_item_ids}). Content-Type is `application/json`.
**Why human:** Confirms the chi route mount, middleware chain, and handler all work end-to-end on a running server with real DB rows. Plan 21-03 SUMMARY explicitly defers this to phase-level approval.

#### 4. (Optional) Re-run integration tests against hq_test DB

**Test:**
```
DB_TEST_URL='postgres://yumyums:yumyums@192.168.8.164:5433/hq_test?sslmode=disable' \
  go test ./internal/inventory -run TestPeriodSummary -v
```
**Expected:** 6 PASS subtests (ready=true, ready=false-pending, ready=false-unlinked, discarded-not-blocking, bad-date 400, from>to 400).
**Why human:** Cached PASS result confirmed in this verifier's run (see Behavioral Spot-Checks), but the DB lives on a remote Tailscale/LAN box (Windows dev Postgres). Re-execution requires network reachability that may not persist between sessions.

### Gaps Summary

There are no implementation gaps. All 18 must-have truths verified against the codebase. All 7 documented artifact files exist with the documented content. All key links wired (handler→DB, middleware→subtle, main.go→middleware+handler, test→DB+migrate). The peer-vs-child chi placement is correct inside `/api/v1`. Build, vet, and both test suites (auth unit tests + inventory integration tests) PASS.

The phase is functionally complete on the HQ side. The four items in `human_verification` are checkpoint-style live-server confirmations of paths already covered by unit/integration tests — they are required to fully close the plan's `<verification>` block, but they are not implementation gaps and do not require any new planning.

Sales-processor consumer-side scope (ROADMAP Scopes 3, 4, 5, and the three top-level Acceptance scenarios involving `--force-payroll`, PDF, exit codes) is explicitly OUT OF SCOPE for this repo and was handed off to the sales-processor team per 21-SALES-PROCESSOR-CONTRACT.md.

---

_Verified: 2026-06-02T18:00:00Z_
_Verifier: Claude (gsd-verifier)_
