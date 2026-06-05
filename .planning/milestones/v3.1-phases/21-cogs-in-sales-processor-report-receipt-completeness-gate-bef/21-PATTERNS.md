# Phase 21: COGS in sales-processor report + receipt completeness gate before payroll - Pattern Map

**Mapped:** 2026-06-02
**Files analyzed:** 8 (5 HQ-side with analogs, 3 sales-processor-side flagged NO ANALOG)
**Analogs found:** 5 / 8

## File Classification

| New/Modified File | Repo | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|------|-----------|----------------|---------------|
| `backend/internal/inventory/handler.go` (ADD `PeriodSummaryHandler`) | hq | http handler (factory) | request-response, DB read-aggregate | `backend/internal/inventory/handler.go::ListPendingPurchasesHandler` (same file, lines 595-629) | exact |
| `backend/internal/inventory/types.go` (ADD `PeriodSummary`, `CompletenessBlock`) | hq | response type definition | n/a (data shape) | `backend/internal/inventory/types.go::StockItem` (same file, lines 103-116) | exact |
| `backend/internal/auth/service_token.go` (NEW file) | hq | http middleware (factory) | request-response | `backend/internal/auth/middleware.go::Middleware` (lines 24-42) | role-match (cookie vs token) |
| `backend/cmd/server/main.go` (MODIFY: env load + new route group) | hq | server bootstrap / route wiring | config + DI | `backend/cmd/server/main.go` lines 244-271 (DO Spaces optional env), lines 288-292 (chi sub-group with middleware) | exact |
| `backend/internal/inventory/period_summary_test.go` (NEW) | hq | integration test | DB read-aggregate | `backend/internal/inventory/stock_test.go` (table-driven unit) | partial — no integration analog exists |
| `backend/internal/auth/service_token_test.go` (NEW) | hq | unit test (httptest) | request-response | none in repo | NO ANALOG |
| `sales-processor/service/external/hq.go` (NEW) | sales-processor | http client | request-response | n/a | NO ANALOG - different repo |
| `sales-processor/.../weekly_summary.go` (MODIFY `WeeklySummary` + `Show()`) | sales-processor | report renderer | data presentation | n/a | NO ANALOG - different repo |
| `sales-processor/cmd/.../weekly.go` (ADD `--force-payroll` flag + gate) | sales-processor | CLI command | control flow | n/a | NO ANALOG - different repo |
| `CLAUDE.md` (UPDATE receipt pipeline section) | hq | docs | n/a | existing prose at line 34 | exact |
| `sales-processor/README.md` (UPDATE env vars + flag docs) | sales-processor | docs | n/a | NO ANALOG - different repo |

## Pattern Assignments

### `backend/internal/inventory/handler.go` - ADD `PeriodSummaryHandler` (http handler, request-response, DB read-aggregate)

**Analog:** `backend/internal/inventory/handler.go::ListPendingPurchasesHandler` lines 595-629 (same file - exact match for the factory-closure-over-pool + pgx-query + writeJSON shape).

**Imports already in file** (lines 1-17):
```go
package inventory

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/yumyums/hq/internal/auth"
    "golang.org/x/text/cases"
    "golang.org/x/text/language"
)
```
No new imports needed - `time`, `net/http`, `log`, `pgxpool` already present.

**Package-local helpers to reuse** (lines 24-32):
```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
    writeJSON(w, status, map[string]string{"error": msg})
}
```
**Use these. Do NOT introduce new response helpers.**

**Handler-factory pattern** (from `ListPendingPurchasesHandler`, lines 595-629):
```go
// ListPendingPurchasesHandler returns pending purchases that have not been confirmed or discarded.
func ListPendingPurchasesHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := pool.Query(r.Context(), `
            SELECT id, bank_tx_id, bank_total, vendor, event_date::text,
                   tax, total, total_units, total_cases, receipt_url,
                   reason, items, confirmed_at, confirmed_by, discarded_at, created_at
            FROM pending_purchases
            WHERE confirmed_at IS NULL AND discarded_at IS NULL
            ORDER BY created_at DESC`,
        )
        if err != nil {
            log.Printf("ListPendingPurchases query: %v", err)
            writeError(w, http.StatusInternalServerError, "internal_error")
            return
        }
        defer rows.Close()

        pending := []PendingPurchase{}
        for rows.Next() {
            var p PendingPurchase
            if err := rows.Scan( /* ... */ ); err != nil {
                log.Printf("ListPendingPurchases scan: %v", err)
                writeError(w, http.StatusInternalServerError, "internal_error")
                return
            }
            pending = append(pending, p)
        }
        writeJSON(w, http.StatusOK, pending)
    }
}
```
Copy this exact shape. For `PeriodSummaryHandler` specifically:
- `pool.QueryRow(r.Context(), ...).Scan(&cogsExcl, &cogsInclTax, &eventCount)` for the single-row aggregate (analog: `ConfirmPendingPurchaseHandler` line 669-672 uses the same single-row `QueryRow().Scan()`).
- `pool.Query(r.Context(), ...)` + `defer rows.Close()` + `for rows.Next()` for the pending_review_ids and unlinked_line_item_ids list queries (exactly the ListPendingPurchases shape).

**SQL aggregation precedent** (from `GetStockHandler`, lines 373-403): aggregation is done in SQL using `SUM(pli.quantity * pli.price)` joined `purchase_line_items pli` to `purchase_events pe`. Phase 21 follows the same join shape. Excerpt:
```sql
FROM purchase_line_items pli
JOIN purchase_events pe ON pe.id = pli.purchase_event_id
LEFT JOIN purchase_items pi ON pi.id = pli.purchase_item_id
```

**Input validation precedent** (from `ConfirmPendingPurchaseHandler`, lines 636-644):
```go
if input.ID == "" || input.VendorName == "" || input.EventDate == "" {
    writeError(w, http.StatusBadRequest, "missing_required_fields")
    return
}
```
For `PeriodSummary`, validate `from` and `to` query params with `time.Parse("2006-01-02", s)` and return `writeError(w, http.StatusBadRequest, "from must be YYYY-MM-DD")` on failure (same status + error-string convention).

**Timezone precedent in the repo:** `backend/internal/purchasing/repurchase.go:71` establishes `America/Chicago` as the project timezone. The new SQL must use:
```sql
(pp.created_at AT TIME ZONE 'America/Chicago')::date BETWEEN $1 AND $2
```
for `pending_purchases.created_at` (TIMESTAMPTZ) but plain `pe.event_date BETWEEN $1 AND $2` for `purchase_events.event_date` (DATE — no TZ).

---

### `backend/internal/inventory/types.go` - ADD `PeriodSummary`, `CompletenessBlock` (response types)

**Analog:** `backend/internal/inventory/types.go::StockItem` (same file, lines 103-116).

**Convention to copy** (entire file is the analog - flat struct with `json` tags, optional fields as `*T` with `,omitempty`):
```go
// StockItem is an aggregated stock level for one purchase item description.
type StockItem struct {
    Description      string           `json:"description"`
    GroupName        *string          `json:"group_name,omitempty"`
    TotalQuantity    int              `json:"total_quantity"`
    TotalSpend       float64          `json:"total_spend"`
    AvgPrice         float64          `json:"avg_price"`
    LastPurchaseDate string           `json:"last_purchase_date"` // YYYY-MM-DD
    LowThreshold     int              `json:"low_threshold"`
    HighThreshold    int              `json:"high_threshold"`
    Level            string           `json:"level"`
    NeedsReorder     bool             `json:"needs_reorder"`
    RepurchaseBadge  *RepurchaseBadge `json:"repurchase_badge,omitempty"`
}
```
Note the inline comment for date format (`// YYYY-MM-DD`) - apply same convention to `From`/`To` fields.

**Nested struct pattern** also used in `PurchaseEvent` (lines 63-74) where `LineItems []LineItem`. Apply same pattern: `Completeness CompletenessBlock` as a non-pointer nested struct (always present).

**New types to add (specified in RESEARCH.md lines 222-237):**
```go
type PeriodSummary struct {
    From               string             `json:"from"`             // YYYY-MM-DD
    To                 string             `json:"to"`               // YYYY-MM-DD
    COGSExclTax        float64            `json:"cogs_excl_tax"`
    COGSInclTax        float64            `json:"cogs_incl_tax"`
    PurchaseEventCount int                `json:"purchase_event_count"`
    Completeness       CompletenessBlock  `json:"completeness"`
}

type CompletenessBlock struct {
    Ready               bool     `json:"ready"`
    PendingReviewIDs    []string `json:"pending_review_ids"`
    UnlinkedLineItemIDs []string `json:"unlinked_line_item_ids"`
}
```
Initialize slices as `[]string{}` (not nil) so JSON renders `[]` not `null` - same convention as `pending := []PendingPurchase{}` at line 613.

---

### `backend/internal/auth/service_token.go` - NEW file (http middleware, request-response)

**Analog:** `backend/internal/auth/middleware.go::Middleware` (lines 24-42, same package).

**Full analog source** (`backend/internal/auth/middleware.go`):
```go
package auth

import (
    "context"
    "net/http"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/yumyums/hq/internal/config"
)

type contextKey string

const CtxKeyUser contextKey = "user"

// Middleware validates the hq_session cookie, looks up the session in DB,
// and attaches the User to request context. Returns 401 if invalid.
func Middleware(pool *pgxpool.Pool, superadmins map[string]config.SuperadminEntry) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("hq_session")
            if err != nil {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }
            tokenHash := HashToken(cookie.Value)
            user, err := LookupSession(r.Context(), pool, tokenHash, superadmins)
            if err != nil || user == nil {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), CtxKeyUser, user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Key pieces to mirror:**
1. Factory signature: `func ServiceTokenMiddleware(expectedToken string) func(http.Handler) http.Handler`
2. `http.Error(w, '{"error":"unauthorized"}', http.StatusUnauthorized)` — the exact JSON-string-as-text trick (note: `http.Error` adds `Content-Type: text/plain` but writes the body as-is; sales-processor will parse it as a status check, not JSON decode the error body).
3. Wrap inner handler with `http.HandlerFunc(func(w, r) {...})` then return as `http.Handler`.

**Bearer-token parsing precedent in this repo:** `backend/internal/receipt/mercury.go:30` shows the established Bearer-token header-format convention (outbound). For inbound parsing (no existing analog), use stdlib:
```go
authHeader := r.Header.Get("Authorization")
const prefix = "Bearer "
if !strings.HasPrefix(authHeader, prefix) {
    http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
    return
}
provided := strings.TrimPrefix(authHeader, prefix)
```

**Constant-time compare:** No analog in repo — establish it new from stdlib:
```go
import "crypto/subtle"

if subtle.ConstantTimeCompare([]byte(provided), []byte(expectedToken)) != 1 {
    http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
    return
}
```

**Empty-token guard:** No analog (this is a new pattern Phase 21 establishes). When `expectedToken == ""` return 503 to fail-closed:
```go
if expectedToken == "" {
    http.Error(w, `{"error":"service_token_not_configured"}`, http.StatusServiceUnavailable)
    return
}
```
This mirrors the 503-when-misconfigured pattern logged for DO Spaces at `main.go:270` (which only logs and continues — but Phase 21's middleware enforces server-side).

---

### `backend/cmd/server/main.go` - MODIFY (env load + new chi sub-group)

**Analog 1 - Env-var optional load with WARNING log:** lines 244-271 (DO Spaces presigner init):
```go
spacesEndpoint := os.Getenv("DO_SPACES_ENDPOINT")
spacesBucket := os.Getenv("DO_SPACES_BUCKET")
spacesRegion := os.Getenv("DO_SPACES_REGION")
// ...
if os.Getenv("DO_SPACES_KEY") != "" && os.Getenv("DO_SPACES_SECRET") != "" && spacesBucket != "" && spacesEndpoint != "" {
    // init
} else {
    log.Println("WARNING: DO Spaces env vars not set (DO_SPACES_KEY, ...) — photo and video upload endpoints will return 503")
}
```
For Phase 21:
```go
serviceToken := os.Getenv("HQ_INVENTORY_SERVICE_TOKEN")
if serviceToken == "" {
    log.Println("WARNING: HQ_INVENTORY_SERVICE_TOKEN not set — /api/v1/inventory/period-summary will return 503")
}
```

**Analog 2 - chi.Group with middleware (peer to cookie-auth group):** lines 288-292:
```go
// WebSocket endpoint at /ws — behind auth middleware, outside /api/v1 prefix
r.Group(func(r chi.Router) {
    r.Use(auth.Middleware(pool, superadmins))
    r.Get("/ws", opsync.WsHandler(hub, pool))
})
```

And the larger cookie-auth group at line 328-330:
```go
// Protected — auth middleware applied to this group
r.Group(func(r chi.Router) {
    r.Use(auth.Middleware(pool, superadmins))
    // ... many routes ...
})
```

**Apply this pattern for Phase 21:** add a NEW peer group inside `r.Route("/api/v1", func(r chi.Router) {...})` that does NOT use `auth.Middleware` (cookie auth) but instead uses the new `auth.ServiceTokenMiddleware`. Place it OUTSIDE the existing cookie-auth `r.Group` (which starts at line 328):
```go
r.Route("/api/v1", func(r chi.Router) {
    // ... unauthenticated routes (health, logs, /auth/login, /auth/invite-info, /auth/accept-invite) ...

    // Service-to-service (no cookie session): inventory period summary
    r.Group(func(r chi.Router) {
        r.Use(auth.ServiceTokenMiddleware(serviceToken))
        r.Get("/inventory/period-summary", inventory.PeriodSummaryHandler(pool))
    })

    // Protected — auth middleware applied to this group (existing, unchanged)
    r.Group(func(r chi.Router) {
        r.Use(auth.Middleware(pool, superadmins))
        // ... existing routes ...
    })
})
```

**Why NOT inside the existing `r.Route("/inventory", ...)` block at line 385:** that block lives inside the cookie-auth `r.Group` — adding the new route there would force sales-processor to send a session cookie. Service-token routes must be a peer group, not a child of the cookie-auth group.

---

### `backend/internal/inventory/period_summary_test.go` - NEW (integration test)

**Closest analog in repo:** `backend/internal/inventory/stock_test.go` — table-driven unit test, NOT integration. There is **no existing pgxpool integration test pattern in this repo**, so Phase 21 establishes one.

**What stock_test.go gives us (lines 1-64):**
```go
package inventory

import "testing"

func TestClassifyStockLevel(t *testing.T) {
    tests := []struct {
        name        string
        qty         int
        // ... fields ...
        wantLevel   string
        wantReorder bool
    }{
        { name: "zero quantity is unknown", qty: 0, lowT: 3, highT: 10,
          wantLevel: "unknown", wantReorder: false },
        // ... more cases ...
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            level, needsReorder := ClassifyStockLevel(tc.qty, tc.lowT, tc.highT)
            if level != tc.wantLevel {
                t.Errorf("ClassifyStockLevel(%d, %d, %d) level = %q, want %q",
                    tc.qty, tc.lowT, tc.highT, level, tc.wantLevel)
            }
        })
    }
}
```
**Reusable conventions:** package = `inventory` (same package, white-box access to types and helpers), table-driven subtests with `t.Run(tc.name, ...)`, descriptive case names.

**For Phase 21 integration:** must add fixtures (DB pool setup, migrations, truncate-between-subtests). Use `DB_TEST_URL` from Taskfile.yml line 11 (defaulted to `postgres://...@.../hq_test?sslmode=disable`). Migration runner is `db.Migrate(pool)` — same call as `main.go:202`.

**Fixture setup must touch:**
- `vendors` (INSERT one row, get id) — schema at `0024_inventory.sql:4-8`
- `purchase_events` (INSERT with `event_date DATE`, `tax NUMERIC(10,2)`, `total NUMERIC(10,2)`) — schema at `0024_inventory.sql:33-42`
- `purchase_line_items` (INSERT with `quantity INTEGER`, `price NUMERIC(10,4)`, nullable `purchase_item_id`) — schema at `0024_inventory.sql:44-52`
- `pending_purchases` (INSERT with `created_at TIMESTAMPTZ` set to known value, nullable `confirmed_at`, `discarded_at`) — schema at `0025_pending_purchases.sql:4-21`

**Teardown between subtests:** `TRUNCATE purchase_line_items, purchase_events, pending_purchases, vendors RESTART IDENTITY CASCADE` (no precedent — establish this in Phase 21).

---

### `backend/internal/auth/service_token_test.go` - NEW (unit/handler test)

**NO ANALOG in repo** — there are no existing tests in `backend/internal/auth/`. Use Go stdlib `net/http/httptest`:
- `httptest.NewRequest("GET", "/", nil)` to build a request
- `req.Header.Set("Authorization", "Bearer test-token")`
- `w := httptest.NewRecorder()` for the response
- Wrap a no-op `http.HandlerFunc` with `auth.ServiceTokenMiddleware("test-token")(noop).ServeHTTP(w, req)`
- Assert `w.Code == 401` or `200`

**Test cases required (from RESEARCH.md Test Map lines 512-514):**
- Missing `Authorization` header → 401
- Malformed (no `Bearer ` prefix) → 401
- Wrong token → 401
- Correct token → 200 (next handler called)
- Empty `expectedToken` parameter → 503

---

## NO ANALOG (different repo - sales-processor)

The following files live in the `sales-processor` repo (separate codebase, separate DB). **The researcher flagged 9 unknown assumptions about sales-processor** (RESEARCH.md Assumptions Log A1–A9). The planner must coordinate with the sales-processor repo directly — no in-repo analog exists, do not fabricate one.

| File | Why NO ANALOG |
|------|---------------|
| `sales-processor/service/external/hq.go` | New HTTP client; sales-processor codebase not in this repo. CLI framework, project layout, and existing HTTP-client conventions unknown. [ASSUMED A1] |
| `sales-processor/.../weekly_summary.go` (`WeeklySummary` struct + `Show()` method) | Struct name, file path, render technology (PDF lib, CSV writer) unknown. [ASSUMED A2] |
| `sales-processor/cmd/.../weekly.go` (`--force-payroll` flag) | CLI framework unknown (stdlib flag vs cobra vs urfave/cli). [ASSUMED A3] |
| `sales-processor/README.md` | Documentation conventions of the other repo unknown. |

**Planner recommendation (per RESEARCH.md Open Question 1, lines 463-466):** Split Phase 21 into two plans — "HQ side" (this repo, fully patterned here) and "sales-processor side" (other repo, requires user clarification on repo path, CLI framework, and existing HTTPClient patterns before tasks can be authored).

---

## Shared Patterns (HQ side)

### Pattern A: Handler factory closing over `*pgxpool.Pool`
**Source:** `backend/internal/inventory/handler.go` — every handler in this file (`ListVendorsHandler` line 35, `ListPendingPurchasesHandler` line 596, `ConfirmPendingPurchaseHandler` line 632, ~20 more).
**Apply to:** `PeriodSummaryHandler` (the new handler).
**Excerpt** (canonical shape from line 35-37):
```go
func ListVendorsHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := pool.Query(r.Context(), `SELECT ...`)
        // ...
    }
}
```

### Pattern B: Package-local `writeJSON` / `writeError`
**Source:** `backend/internal/inventory/handler.go:24-32`.
**Apply to:** All responses from `PeriodSummaryHandler`.
**Rule:** Never write JSON to the response by hand — always `writeJSON(w, status, value)` and `writeError(w, status, msg)`.

### Pattern C: Parameterized SQL with `$1`, `$2` (pgx convention)
**Source:** `backend/internal/inventory/handler.go:441-446` (`GetStockHandler` repurchase query), and dozens more.
**Apply to:** Date-range query in `PeriodSummaryHandler`.
**Rule:** Never string-interpolate date values; always pass `fromStr`, `toStr` as `$1`, `$2`.

### Pattern D: `log.Printf("HandlerName operation: %v", err)` then `writeError(... "internal_error")`
**Source:** Every error branch in `handler.go` — e.g. lines 41-43, 607-610, 621-624.
**Apply to:** All error branches in `PeriodSummaryHandler` and the new middleware.
**Excerpt:**
```go
if err != nil {
    log.Printf("ListPendingPurchases query: %v", err)
    writeError(w, http.StatusInternalServerError, "internal_error")
    return
}
```
**Rule:** Log the full error; return `"internal_error"` (no detail leak) to the client.

### Pattern E: Initialize slice literals as `[]T{}` not `nil`
**Source:** `handler.go:613` (`pending := []PendingPurchase{}`), `handler.go:47` (`vendors := []Vendor{}`).
**Apply to:** `pendingIDs := []string{}` and `unlinkedIDs := []string{}` in `PeriodSummaryHandler`.
**Rule:** Ensures JSON serializes as `[]` (empty array), not `null`.

### Pattern F: `r.Context()` threaded into every pool call
**Source:** Every `pool.Query(r.Context(), ...)` / `pool.QueryRow(r.Context(), ...)` / `pool.Exec(r.Context(), ...)` in `handler.go`.
**Apply to:** All three SQL calls in `PeriodSummaryHandler`.
**Rule:** Never pass `context.Background()` or `context.TODO()` inside a handler — always `r.Context()` so request cancellation cascades.

### Pattern G: Optional/configurable feature with WARNING-log fallback in `main.go`
**Source:** `main.go:244-271` (DO Spaces) — env vars are checked, if missing a WARNING is logged and the endpoint returns 503 at request time (rather than failing startup).
**Apply to:** `HQ_INVENTORY_SERVICE_TOKEN` env-var load + the 503-when-empty branch inside `ServiceTokenMiddleware`.

### Pattern H: Documentation comment style
**Source:** Every exported handler in `handler.go` has a single-line doc comment above the function (e.g. line 595: `// ListPendingPurchasesHandler returns pending purchases that have not been confirmed or discarded.`).
**Apply to:** `PeriodSummaryHandler` doc must state: inclusive date range, `America/Chicago` interpretation, that `event_date` is used for COGS and `created_at` (TZ-cast) for the completeness gate.

---

## File-by-File Pattern Reference Quick-Index

| Task | Read these lines first |
|------|------------------------|
| Add `PeriodSummaryHandler` to handler.go | handler.go:24-32 (helpers), 595-629 (list-pattern), 632-740 (multi-step pattern), 373-403 (SQL aggregate join) |
| Add `PeriodSummary`/`CompletenessBlock` to types.go | types.go:62-74 (nested struct), 103-116 (StockItem), 76-94 (PendingPurchase optional fields) |
| Create `auth/service_token.go` | auth/middleware.go:24-42 (full middleware analog), receipt/mercury.go:30 (Bearer-header convention) |
| Wire route in main.go | main.go:244-271 (env+WARNING), 288-292 (chi.Group with middleware), 328-330 (peer auth group), 385-407 (existing inventory routes — DO NOT add inside this block) |
| Create integration test | stock_test.go:1-64 (table-driven shape — same package, t.Run subtests), Taskfile.yml:11,96-102 (DB_TEST_URL + `task db-test`), main.go:202 (`db.Migrate(pool)` call) |
| Create middleware unit test | NO ANALOG — use stdlib `net/http/httptest` |
| Update CLAUDE.md receipt-pipeline section | CLAUDE.md line 34 |

---

## Metadata

**Analog search scope:**
- `backend/internal/inventory/` (handler.go, types.go, stock_test.go)
- `backend/internal/auth/` (middleware.go — only file; no tests exist)
- `backend/internal/receipt/` (mercury.go — for outbound Bearer-token convention)
- `backend/internal/purchasing/` (repurchase.go — for `America/Chicago` precedent cited in RESEARCH.md)
- `backend/cmd/server/main.go` (env loading + chi route registration)
- `backend/internal/db/migrations/0024_inventory.sql`, `0025_pending_purchases.sql` (schema confirmation)
- `backend/Taskfile.yml` (test DB conventions)

**Files scanned:** 9 source files + 2 migrations + Taskfile = 12 read

**Skipped (per phase instructions):** sales-processor-side files — researcher logged 9 assumptions (A1–A9) and zero in-repo evidence; planner must consult that repo separately rather than fabricate analogs.

**Pattern extraction date:** 2026-06-02
