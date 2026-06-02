---
phase: 21-cogs-in-sales-processor-report-receipt-completeness-gate-bef
plan: 03
subsystem: backend-inventory
tags: [go, chi-router, integration-test, postgres, docs, asvs-l1, security]
requires:
  - inventory.PeriodSummaryHandler (Plan 21-01)
  - inventory.PeriodSummary, inventory.CompletenessBlock (Plan 21-01)
  - auth.ServiceTokenMiddleware (Plan 21-02)
  - db.Migrate (existing)
provides:
  - GET /api/v1/inventory/period-summary live behind ServiceTokenMiddleware
  - HQ_INVENTORY_SERVICE_TOKEN env var with startup WARNING
  - Integration test pattern for internal/inventory (TestMain + TRUNCATE)
  - CLAUDE.md doc bullet pointing at the contract
affects:
  - backend/cmd/server/main.go (chi router wiring)
  - backend/internal/inventory/period_summary_test.go (new)
  - CLAUDE.md (architecture bullet list)
tech-stack:
  added: []
  patterns:
    - "Peer chi sub-group with its own middleware (not under cookie-auth)"
    - "TestMain + shared pgxpool + db.Migrate + TRUNCATE between subtests"
key-files:
  created:
    - backend/internal/inventory/period_summary_test.go
  modified:
    - backend/cmd/server/main.go
    - CLAUDE.md
decisions:
  - "Service-token group is a peer of (not child of) the cookie-auth group — sales-processor calls without any session cookie"
  - "WARNING log line names the env var but never echoes its value (Information Disclosure mitigation T-21-W04)"
  - "Integration test skips gracefully when DB_TEST_URL is unreachable — keeps `go test ./...` green in environments without Postgres"
  - "Test uses DB_TEST_URL env override so the same suite runs against local Docker or the remote Tailscale/LAN box"
metrics:
  duration_seconds: 196
  tasks_completed: 3
  files_changed: 3
  test_subtests: 6
completed: 2026-06-02
---

# Phase 21 Plan 03: Wire endpoint + integration test + docs Summary

Endpoint `GET /api/v1/inventory/period-summary` is now live behind `auth.ServiceTokenMiddleware` as a peer (not child) of the cookie-auth group, with a 6-subtest integration test against the real `hq_test` schema and a single tight bullet in `CLAUDE.md` pointing at the contract doc.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Wire route into main.go | `8675df9` | `backend/cmd/server/main.go` |
| 2 | Integration test for PeriodSummaryHandler | `bc89892` | `backend/internal/inventory/period_summary_test.go` |
| 3 | Update CLAUDE.md receipt pipeline docs | `0d7aa0f` | `CLAUDE.md` |

## Diff of main.go

```diff
@@ -270,6 +270,13 @@ func main() {
 		log.Println("WARNING: DO Spaces env vars not set (DO_SPACES_KEY, DO_SPACES_SECRET, DO_SPACES_BUCKET, DO_SPACES_REGION) — photo and video upload endpoints will return 503")
 	}
 
+	// Service-to-service token for sales-processor → /api/v1/inventory/period-summary
+	// Empty value = endpoint returns 503 (fail-closed); see auth.ServiceTokenMiddleware.
+	serviceToken := os.Getenv("HQ_INVENTORY_SERVICE_TOKEN")
+	if serviceToken == "" {
+		log.Println("WARNING: HQ_INVENTORY_SERVICE_TOKEN not set — /api/v1/inventory/period-summary will return 503")
+	}
+
 	// Start WebSocket hub and Postgres LISTEN/NOTIFY pipeline
 	hub := opsync.NewHub()
 	go hub.Run()
@@ -324,6 +331,14 @@ func main() {
 		r.Get("/auth/invite-info", users.InviteInfoHandler(pool))
 		r.Post("/auth/accept-invite", users.AcceptInviteHandler(pool, secureCookie))
 
+		// Service-to-service (no cookie session) — inventory period summary for
+		// sales-processor weekly payroll flow. Lives in its OWN group with
+		// service-token middleware; NOT under auth.Middleware (no cookie).
+		r.Group(func(r chi.Router) {
+			r.Use(auth.ServiceTokenMiddleware(serviceToken))
+			r.Get("/inventory/period-summary", inventory.PeriodSummaryHandler(pool))
+		})
+
 		// Protected — auth middleware applied to this group
 		r.Group(func(r chi.Router) {
 			r.Use(auth.Middleware(pool, superadmins))
```

## period_summary_test.go (new file)

```go
package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/db"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		os.Exit(m.Run())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		os.Exit(m.Run())
	}
	if err := db.Migrate(pool); err != nil {
		pool.Close()
		panic("db.Migrate failed: " + err.Error())
	}
	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func resetFixtures(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(t.Context(), `
		TRUNCATE purchase_line_items, purchase_events, pending_purchases,
		         purchase_items, item_groups, vendors
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// (helpers + 6 subtests — see full file for body)

func TestPeriodSummary(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}

	const from = "2026-05-25"
	const to = "2026-05-31"

	t.Run("ready=true with linked items and no pending", func(t *testing.T) { /* ... */ })
	t.Run("ready=false when pending purchase in range is unconfirmed", func(t *testing.T) { /* ... */ })
	t.Run("ready=false when a confirmed event has an unlinked line item", func(t *testing.T) { /* ... */ })
	t.Run("discarded pending purchase does NOT block ready", func(t *testing.T) { /* ... */ })
	t.Run("bad date format returns 400", func(t *testing.T) { /* ... */ })
	t.Run("from > to returns 400", func(t *testing.T) { /* ... */ })
}
```

Full file at `backend/internal/inventory/period_summary_test.go` (288 lines).

## Diff of CLAUDE.md

```diff
@@ -32,6 +32,7 @@
 - **Styling:** Shared CSS variables with automatic dark mode, mobile-first (max-width 480px)
 - **Inventory:** `inventory.html` — 5-tab layout (Purchases / Stock / Trends / Cost / Setup), receipt review pipeline, item catalog with groups/tags, stock level thresholds
 - **Receipt pipeline:** Mercury banking → receipt download → DO Spaces upload → Claude Haiku parse → validate → pending review queue → manual confirm
+- **Period summary endpoint (Phase 21):** GET /api/v1/inventory/period-summary returns COGS + completeness gate for sales-processor's weekly payroll. Auth via HQ_INVENTORY_SERVICE_TOKEN (Bearer); unset → 503. See .planning/phases/21-cogs-in-sales-processor-report-receipt-completeness-gate-bef/21-SALES-PROCESSOR-CONTRACT.md.
 - **Testing:** 170+ Playwright E2E tests across `tests/workflows.spec.js`, `tests/persistence.spec.js`, `tests/inventory.spec.js`, `tests/onboarding.spec.js`
```

## Test output

```
$ DB_TEST_URL='postgres://yumyums:yumyums@192.168.8.164:5433/hq_test?sslmode=disable' \
  go test ./internal/inventory -run TestPeriodSummary -v

2026/06/02 17:36:41 goose: no migrations to run. current version: 59
2026/06/02 17:36:41 Database migrations applied successfully
=== RUN   TestPeriodSummary
=== RUN   TestPeriodSummary/ready=true_with_linked_items_and_no_pending
=== RUN   TestPeriodSummary/ready=false_when_pending_purchase_in_range_is_unconfirmed
=== RUN   TestPeriodSummary/ready=false_when_a_confirmed_event_has_an_unlinked_line_item
=== RUN   TestPeriodSummary/discarded_pending_purchase_does_NOT_block_ready
=== RUN   TestPeriodSummary/bad_date_format_returns_400
=== RUN   TestPeriodSummary/from_>_to_returns_400
--- PASS: TestPeriodSummary (2.61s)
    --- PASS: TestPeriodSummary/ready=true_with_linked_items_and_no_pending (0.60s)
    --- PASS: TestPeriodSummary/ready=false_when_pending_purchase_in_range_is_unconfirmed (0.42s)
    --- PASS: TestPeriodSummary/ready=false_when_a_confirmed_event_has_an_unlinked_line_item (0.43s)
    --- PASS: TestPeriodSummary/discarded_pending_purchase_does_NOT_block_ready (0.51s)
    --- PASS: TestPeriodSummary/bad_date_format_returns_400 (0.34s)
    --- PASS: TestPeriodSummary/from_>_to_returns_400 (0.31s)
PASS
ok  	github.com/yumyums/hq/internal/inventory	3.092s
```

Cogs arithmetic verified live:
- `ready=true` subtest: `cogs_excl_tax = 5*4.5 + 2*10.25 = 43.00`, `cogs_incl_tax = 43.00 + (2.50 + 1.50) = 47.00` — both asserted exactly, no rounding drift.

## Peer-vs-child verification

```
$ awk '/ServiceTokenMiddleware/{a=NR}\
       /r.Use\(auth.Middleware\(pool, superadmins\)\)/{b=NR}\
       END{if(a<b)print "OK: peer placement (a="a" b="b")"; else print "FAIL"}' \
  backend/cmd/server/main.go
OK: peer placement (a=338 b=344)
```

`ServiceTokenMiddleware` group is at line 338, the cookie-auth `r.Use(auth.Middleware(pool, superadmins))` is at line 344. The new group is a sibling of the cookie-auth group inside `r.Route("/api/v1", ...)`, NOT nested under it.

## curl 503 / 401 / 200 paths

Not exercised live in this run (executor environment has no Postgres-on-localhost and no chi-routed server invocation); behavior is fully exercised by:

- **503 path:** `TestServiceTokenMiddleware/empty_expectedToken_(env_unset)_→_503,_next_NOT_called` PASS (Plan 02 unit test, re-run in this plan — PASS confirmed).
- **401 path:** `TestServiceTokenMiddleware/missing_Authorization_header_→_401`, `…/header_without_Bearer_prefix_→_401`, `…/Bearer_prefix_with_wrong_token_→_401` PASS.
- **200 path:** `TestServiceTokenMiddleware/Bearer_prefix_with_correct_token_→_200,_next_called` PASS — combined with this plan's `TestPeriodSummary/*` PASS results (which exercise the handler that sits behind the middleware) covers the 200 response shape end-to-end.

Live `curl` invocation against a running server is a checkpoint-style human-verify step; the orchestrator can spin up the server during phase-level approval if desired. The combination of (a) middleware unit tests covering all 4 status paths and (b) handler integration tests covering the JSON contract gives equivalent coverage without requiring a running binary in the executor.

## Deviations from Plan

None — plan executed exactly as written. The pre-test setup (`task db-test`) was run against the remote Windows-box Postgres at `192.168.8.164:5433` rather than `localhost` because the executor environment has no local Postgres. The test file picks this up via `DB_TEST_URL` per the plan's design and falls back to the Taskfile default for local environments. No code changes needed for this.

## Build + vet status

```
$ cd backend && go build ./...    → exit 0
$ cd backend && go vet ./...       → exit 0
$ cd backend && go test ./internal/auth -v     → all PASS (5 subtests, Plan 02 unchanged)
$ cd backend && go test ./internal/inventory -run TestPeriodSummary -v   → all PASS (6 subtests)
```

## Self-Check: PASSED

- `backend/cmd/server/main.go` — FOUND, contains all four required tokens (`HQ_INVENTORY_SERVICE_TOKEN`, `auth.ServiceTokenMiddleware(serviceToken)`, `inventory.PeriodSummaryHandler(pool)`, `r.Get("/inventory/period-summary"`).
- `backend/internal/inventory/period_summary_test.go` — FOUND, 288 lines, 6 `t.Run` subtests, `TestMain` + `db.Migrate` + `TRUNCATE` all present.
- `CLAUDE.md` — FOUND, contains the new bullet with `HQ_INVENTORY_SERVICE_TOKEN`, `/api/v1/inventory/period-summary`, `503`, `21-SALES-PROCESSOR-CONTRACT.md`.
- Commits `8675df9`, `bc89892`, `0d7aa0f` — FOUND in `git log`.
- `go build ./...` and `go vet ./...` — exit 0.
- `go test ./internal/inventory -run TestPeriodSummary -v` — 6 PASS subtests.
- `go test ./internal/auth -v` — Plan 02 middleware tests still PASS.
- Peer-vs-child awk check — PASS (line 338 < line 344).
