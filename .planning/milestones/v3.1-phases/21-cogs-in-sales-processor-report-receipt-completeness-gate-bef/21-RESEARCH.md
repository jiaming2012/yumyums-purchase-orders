# Phase 21: COGS in sales-processor report + receipt completeness gate before payroll — Research

**Researched:** 2026-06-02
**Domain:** HQ backend Go API (chi router, pgx, postgres) + cross-service HTTP contract with `sales-processor`
**Confidence:** HIGH for HQ-side (this repo) / MEDIUM for sales-processor (separate repo, no in-repo evidence)

## Summary

This phase has two halves that are joined only by a single new HTTP endpoint. The **HQ side** (this repo) is small and well-scoped: add one new `GET /api/v1/inventory/period-summary?from=&to=` handler in the existing `internal/inventory/handler.go` package, register it on the chi router in `backend/cmd/server/main.go`, and introduce a *new* service-token auth middleware (no precedent for inbound bearer-token auth exists in the repo today — only the cookie session `auth.Middleware`). The schema is already in place: `purchase_events`, `purchase_line_items`, and `pending_purchases` were shipped in migrations `0024_inventory.sql` and `0025_pending_purchases.sql` and have not changed since.

The **sales-processor side** is in a separate repo and the planner will need to coordinate with that codebase. There is **zero evidence of `sales-processor` in this repo** — no docs, no env vars, no scripts. All sales-processor specifics in the ROADMAP scope (`service/external/hq.go`, `WeeklySummary.Show()`, `--force-payroll` flag) are `[ASSUMED]` based on the ROADMAP description alone and need user confirmation before locking. The HQ-side endpoint contract is what this research can authoritatively pin down.

**Primary recommendation:** Build the HQ endpoint as a self-contained slice in `internal/inventory`. Add a NEW auth middleware variant (`auth.ServiceTokenMiddleware`) that accepts `Authorization: Bearer <token>` and compares to `HQ_INVENTORY_SERVICE_TOKEN` env var using `crypto/subtle.ConstantTimeCompare`. Register the route in its own chi sub-group with this middleware (not under the cookie-auth group). Compute date filtering with explicit timezone semantics — see Pitfall 1.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| COGS aggregation SQL | API / Backend (hq) | Database | Single source of truth for purchase data lives in hq Postgres |
| Completeness check (pending + unlinked) | API / Backend (hq) | Database | Same data domain — gate must read the same DB that owns the receipts |
| Service-to-service auth | API / Backend (hq) | — | Token validation must happen before SQL runs; no client-side enforcement is possible across processes |
| HTTP client to HQ | API / Backend (sales-processor) | — | Caller-side concern: timeouts, retries, error mapping |
| COGS rendering on PDF/CSV | API / Backend (sales-processor) | — | Owns the `WeeklySummary` data model and report output |
| Payroll gate decision | API / Backend (sales-processor) | — | Owns the payroll flow control; `--force-payroll` is a CLI flag in the sales-processor binary |
| Per-menu-item attribution | OUT OF SCOPE | — | Deferred to Phase 999.2 by roadmap |

## Standard Stack

### Core (already in repo)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/go-chi/chi/v5` | (in go.sum) | HTTP router | Already used in `backend/cmd/server/main.go` for all `/api/v1/*` routes — [VERIFIED: backend/cmd/server/main.go:18,278] |
| `github.com/jackc/pgx/v5` | v5.x | Postgres driver | Used in every existing `internal/*/handler.go` — [VERIFIED: backend/internal/inventory/handler.go:12-13] |
| `github.com/jackc/pgx/v5/pgxpool` | v5.x | Connection pool | Shared `*pgxpool.Pool` passed to every handler factory — [VERIFIED: backend/cmd/server/main.go:20] |

### Supporting (already in repo)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `crypto/subtle` | stdlib | Constant-time string compare | For comparing the service token to env var to prevent timing attacks — [CITED: pkg.go.dev/crypto/subtle#ConstantTimeCompare] |
| `encoding/json` | stdlib | Response encoding | Used in existing `writeJSON` helper — [VERIFIED: backend/internal/inventory/handler.go:24-28] |
| `net/http` | stdlib | Handler signatures | Existing handlers return `http.HandlerFunc` — [VERIFIED: backend/internal/inventory/handler.go:35] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Cookie session reuse | Service-token bearer header | Bearer is correct — sales-processor has no user session, can't pretend to be a logged-in user. [VERIFIED: sales-processor scope explicitly says "call without a user session"] |
| HMAC-signed requests | Static shared-secret token | HMAC is correct for high-security, but token-in-env is the pragmatic norm for internal services on the same private box and matches the constraint's simplicity. [ASSUMED — user should confirm threat model]  |
| New `internal/reports` package | Add to existing `internal/inventory` | Endpoint reads inventory data only; cohesion is better in the existing package. The new auth middleware belongs in `internal/auth`. |

**Installation:** No new Go dependencies required. All needed libraries are already in `go.sum`.

**Version verification:** Verified against `backend/go.mod` (Go 1.25.5). No new deps needed.

## Architecture Patterns

### System Architecture Diagram

```
                              SALES-PROCESSOR (separate repo, separate DB)
                              ─────────────────────────────────────────────
                              CLI: ./sales-processor weekly [--force-payroll]
                                  │
                                  ▼
                              service/external/hq.go (NEW)
                                  HTTPClient + GetPeriodSummary(from, to)
                                  │
                                  ▼ HTTPS, Bearer HQ_INVENTORY_SERVICE_TOKEN
                              ─────────────────────────────────────────────
                              │
                              │     hq.yumyums.kitchen
                              ▼
HQ BACKEND (this repo)
─────────────────────────────────────────────────────────────────────────────
chi router (backend/cmd/server/main.go)
  │
  ├── /api/v1/auth/*           (no auth — login/invite)
  ├── /api/v1/* (cookie auth)  (existing — auth.Middleware)
  │
  └── /api/v1/inventory/period-summary  (NEW — auth.ServiceTokenMiddleware)
          │
          ▼
      internal/inventory/handler.go::PeriodSummaryHandler  (NEW)
          │
          ├─→ SQL 1: COGS aggregate ────► purchase_line_items
          │                                JOIN purchase_events
          │                                WHERE event_date BETWEEN $from AND $to
          │
          ├─→ SQL 2: pending_review ────► pending_purchases
          │                                WHERE created_at::date BETWEEN $from AND $to
          │                                AND confirmed_at IS NULL
          │                                AND discarded_at IS NULL
          │
          └─→ SQL 3: unlinked_lines ────► purchase_line_items pli
                                           JOIN purchase_events pe
                                           WHERE pe.event_date BETWEEN $from AND $to
                                           AND pli.purchase_item_id IS NULL

      Postgres (hq DB)
─────────────────────────────────────────────────────────────────────────────
```

### Recommended File Layout
```
backend/
├── cmd/server/main.go               # ADD: new chi route registration
└── internal/
    ├── auth/
    │   ├── middleware.go            # MODIFY: add ServiceTokenMiddleware (or new file)
    │   └── service_token.go         # OR: new file for clean separation
    └── inventory/
        ├── handler.go               # ADD: PeriodSummaryHandler
        ├── types.go                 # ADD: PeriodSummary, CompletenessBlock types
        └── period_summary_test.go   # NEW: integration test against hq_test DB
```

### Pattern 1: Handler Factory Function
**What:** All handlers in this codebase are factory functions that close over the pool.
**When to use:** Every new endpoint.
**Example:** [VERIFIED: backend/internal/inventory/handler.go:595-629]
```go
func ListPendingPurchasesHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := pool.Query(r.Context(), `SELECT ...`)
        if err != nil {
            log.Printf("ListPendingPurchases query: %v", err)
            writeError(w, http.StatusInternalServerError, "internal_error")
            return
        }
        defer rows.Close()
        // scan rows
        writeJSON(w, http.StatusOK, pending)
    }
}
```

### Pattern 2: Middleware via `func(http.Handler) http.Handler`
**What:** Auth middleware is a factory returning the standard wrapper signature.
**When to use:** The new `ServiceTokenMiddleware`.
**Example:** [VERIFIED: backend/internal/auth/middleware.go:24-42]
```go
func Middleware(pool *pgxpool.Pool, superadmins map[string]config.SuperadminEntry) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("hq_session")
            if err != nil {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }
            // ... lookup user, attach to ctx, call next
        })
    }
}
```

**The new middleware** should follow this exact shape and live in `internal/auth/`:
```go
// service_token.go (NEW)
package auth

import (
    "crypto/subtle"
    "net/http"
    "strings"
)

// ServiceTokenMiddleware authenticates internal service-to-service callers via
// a static bearer token loaded from env. Returns 401 if the header is missing,
// malformed, or does not match expectedToken. The empty token DISABLES the endpoint
// (returns 503) to prevent accidental open-access in misconfigured environments.
func ServiceTokenMiddleware(expectedToken string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if expectedToken == "" {
                http.Error(w, `{"error":"service_token_not_configured"}`, http.StatusServiceUnavailable)
                return
            }
            authHeader := r.Header.Get("Authorization")
            const prefix = "Bearer "
            if !strings.HasPrefix(authHeader, prefix) {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }
            provided := strings.TrimPrefix(authHeader, prefix)
            if subtle.ConstantTimeCompare([]byte(provided), []byte(expectedToken)) != 1 {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Pattern 3: Chi sub-group with middleware
**What:** Group routes that share middleware with `r.Group(func(r chi.Router) { r.Use(...); ... })`.
**Example:** [VERIFIED: backend/cmd/server/main.go:288-292,328-330]
```go
// In main.go after loading env:
serviceToken := os.Getenv("HQ_INVENTORY_SERVICE_TOKEN")

r.Route("/api/v1", func(r chi.Router) {
    // ... existing routes ...

    // Service-to-service (no cookie session): inventory period summary
    r.Group(func(r chi.Router) {
        r.Use(auth.ServiceTokenMiddleware(serviceToken))
        r.Get("/inventory/period-summary", inventory.PeriodSummaryHandler(pool))
    })
})
```

### Pattern 4: writeJSON / writeError helpers
**What:** Every inventory handler uses package-local `writeJSON(w, status, value)` and `writeError(w, status, msg)`.
**Source:** [VERIFIED: backend/internal/inventory/handler.go:24-32]

### Pattern 5: Response type definition lives in `types.go`
**What:** Existing types like `PurchaseEvent`, `PendingPurchase` live in `internal/inventory/types.go` with `json` struct tags.
**Source:** [VERIFIED: backend/internal/inventory/types.go]

**Recommended new types:**
```go
// In internal/inventory/types.go
type PeriodSummary struct {
    From               string             `json:"from"`             // YYYY-MM-DD
    To                 string             `json:"to"`               // YYYY-MM-DD
    COGSExclTax        float64            `json:"cogs_excl_tax"`
    COGSInclTax        float64            `json:"cogs_incl_tax"`
    PurchaseEventCount int                `json:"purchase_event_count"`
    Completeness       CompletenessBlock  `json:"completeness"`
}

type CompletenessBlock struct {
    Ready            bool     `json:"ready"`
    PendingReviewIDs []string `json:"pending_review_ids"`
    UnlinkedLineItemIDs []string `json:"unlinked_line_item_ids"`
}
```

### Anti-Patterns to Avoid
- **Don't reuse `auth.Middleware`:** It requires a `hq_session` cookie and a DB session row. Sales-processor has neither. Build a separate middleware.
- **Don't compare tokens with `==`:** Timing-attack vulnerability. Use `subtle.ConstantTimeCompare`.
- **Don't put the new endpoint inside the existing cookie-auth `r.Group`:** It would force sales-processor to fake a cookie. Put it in its own group with the service-token middleware.
- **Don't compute COGS in Go:** SQL aggregation against indexed columns (`event_date`, `purchase_event_id`) is faster and the existing `GetStockHandler` already follows this convention. [VERIFIED: backend/internal/inventory/handler.go:373-403]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bearer token parsing | Custom regex / manual split | `strings.HasPrefix` + `strings.TrimPrefix` | Existing Mercury caller uses this exact pattern — [VERIFIED: backend/internal/receipt/mercury.go:30] |
| Token comparison | `provided == expected` | `subtle.ConstantTimeCompare` | Timing-attack mitigation — [CITED: pkg.go.dev/crypto/subtle] |
| Date range parsing | Custom regex | `time.Parse("2006-01-02", s)` | stdlib YYYY-MM-DD parser handles all edge cases |
| JSON aggregation | Loop in Go | `SUM(qty*price)` in SQL | Already the established pattern in `GetStockHandler` |
| HTTP client (sales-processor side) | Raw `net/http` from scratch | `net/http.Client` with explicit `Timeout` field set | Avoid the global default client (no timeout). [ASSUMED — sales-processor repo conventions not verified] |

**Key insight:** Nothing exotic is needed. This phase composes existing patterns from the same repo (`writeJSON`, handler factories, chi groups, pgx queries) — the only net-new abstraction is the service-token middleware, which is ~20 lines of stdlib code.

## Runtime State Inventory

> Phase 21 is greenfield (new endpoint + new env var + new caller). No rename or migration. This section is omitted intentionally — see scope.

## Common Pitfalls

### Pitfall 1: Date-range query with `TIMESTAMPTZ::date` truncates in the *server session* timezone
**What goes wrong:** The constraint says `pending_purchases.created_at::date BETWEEN from AND to`. Because `created_at` is `TIMESTAMPTZ`, the `::date` cast uses Postgres's `TimeZone` GUC, which defaults to the server's `TZ` env (UTC in most container setups). A receipt ingested at `2026-06-01 23:30:00-05:00` (Chicago, June 1) is stored as `2026-06-02 04:30:00 UTC`. With `TimeZone=UTC` the cast yields `2026-06-02`, so it falls *outside* a Chicago-week range that ends June 1.
**Why it happens:** Server clock and DB clock are UTC but the food truck operates in `America/Chicago`. The repo already encodes this convention for `repurchase_reset_config` ([VERIFIED: backend/internal/purchasing/repurchase.go:71]).
**How to avoid:**
- Either set the connection-time TZ: `SET TIME ZONE 'America/Chicago'` per query, OR
- Use explicit cast: `(created_at AT TIME ZONE 'America/Chicago')::date BETWEEN $1 AND $2`
- Document the convention in the endpoint's contract — `from` and `to` are interpreted as `America/Chicago` calendar dates.
**Warning signs:** Off-by-one purchase counts at week boundaries; receipts "missing" from a report.

### Pitfall 2: `purchase_events.event_date` is `DATE`, not timestamp
**What goes wrong:** [VERIFIED: 0024_inventory.sql:37] `event_date DATE NOT NULL`. The constraint says "weekly range" for COGS, which is unambiguous on `event_date` (no TZ). BUT it specifies `pending_purchases.created_at` for the gate — two *different* date fields for two different aggregates in the same endpoint.
**Why it happens:** `purchase_events.event_date` is set by the receipt parser (the date printed on the receipt); `pending_purchases.created_at` is set by the worker at ingestion time. Confusing them produces silent wrong answers.
**How to avoid:** In the handler, deliberately keep the two SQL queries separate with explicit comments. Use `pe.event_date BETWEEN $1 AND $2` for COGS; use `pp.created_at AT TIME ZONE 'America/Chicago' ::date BETWEEN $1 AND $2` for the gate.

### Pitfall 3: `price` column is `NUMERIC(10,4)`, not `(10,2)`
**What goes wrong:** [VERIFIED: 0024_inventory.sql:50] `price NUMERIC(10,4) NOT NULL`. `quantity` is `INTEGER`. `qty*price` therefore returns NUMERIC. Scanning into `float64` works but the existing `LineItem.Price` is `float64` ([VERIFIED: types.go]) so precision compounds across many rows.
**Why it happens:** Receipts have 4-decimal unit prices; the table stores them faithfully but Go uses float.
**How to avoid:** Round the final aggregate to 2 decimals in SQL or in Go: `ROUND(SUM(pli.quantity * pli.price), 2)::numeric(12,2)` — and scan into `float64`. The existing `GetStockHandler` ignores this and works fine for stock totals, but COGS will appear on a customer-facing report — be precise.

### Pitfall 4: "Unlinked line items" definition is ambiguous in scope
**What goes wrong:** The scope says "unlinked line_item IDs" but doesn't say which table. Two candidates:
- (a) `purchase_line_items.purchase_item_id IS NULL` — confirmed purchase event with no catalog link.
- (b) Items inside `pending_purchases.items` JSONB where `purchase_item_id` is null — pending review.
**Verified from the codebase:** [VERIFIED: backend/internal/inventory/handler.go:706-718] When `ConfirmPendingPurchase` runs, it INSERTs into `purchase_line_items` with `purchase_item_id` from the input — and the frontend forces the user to link every item before allowing confirm ([VERIFIED: inventory.html:789] `var unmapped=state.line_items.filter(function(li){return !li.purchase_item_id;});`). However, [VERIFIED: backend/internal/receipt/worker.go:279] When the receipt worker auto-creates a purchase event (validation passed path), it DOES insert `purchase_item_id` via `DerivePurchaseItemID` — but if that returns nothing, the column is nullable. So (a) is the right definition: count `purchase_line_items.purchase_item_id IS NULL` for confirmed events in the date range.
**How to avoid:** Define the completeness predicate explicitly in the handler godoc: "An event is unlinked when at least one of its `purchase_line_items.purchase_item_id` is NULL." Return the offending `purchase_line_items.id` UUIDs.

### Pitfall 5: Token in env vs. token in DB
**What goes wrong:** If `HQ_INVENTORY_SERVICE_TOKEN` is unset, an empty string compares equal to an empty `Authorization: Bearer ` header → open access.
**How to avoid:** [Pattern shown in middleware example above] — when `expectedToken == ""`, return 503, not 401. Log a startup WARNING from main.go if the env var is unset.

### Pitfall 6: `r.Body` and request limits
**What goes wrong:** Not applicable — this is GET, no body. Skip body-size limiting.

### Pitfall 7: Inclusive vs exclusive end of range
**What goes wrong:** SQL `BETWEEN` is inclusive on both ends. If sales-processor sends `from=2026-05-25&to=2026-05-31` for "week of May 25-31", that's correct. If it sends `from=2026-05-25&to=2026-06-01` for "exclusive Sunday-to-Sunday", off-by-one.
**How to avoid:** Document `from` and `to` as inclusive calendar dates in `America/Chicago`. Reject non-`YYYY-MM-DD` formats with 400.

## Code Examples

### Period-Summary Handler — full SQL shape
```go
// File: backend/internal/inventory/handler.go (append)
// Source: synthesized from existing patterns in same file

// PeriodSummaryHandler returns COGS and receipt-completeness data for sales-processor.
// Date range is INCLUSIVE on both ends, interpreted as America/Chicago calendar dates.
// `event_date` is used for COGS (purchase_events have explicit receipt dates);
// `created_at` (timezone-cast) is used for the completeness gate (ingestion time).
func PeriodSummaryHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        fromStr := r.URL.Query().Get("from")
        toStr := r.URL.Query().Get("to")
        if _, err := time.Parse("2006-01-02", fromStr); err != nil {
            writeError(w, http.StatusBadRequest, "from must be YYYY-MM-DD")
            return
        }
        if _, err := time.Parse("2006-01-02", toStr); err != nil {
            writeError(w, http.StatusBadRequest, "to must be YYYY-MM-DD")
            return
        }
        if fromStr > toStr {
            writeError(w, http.StatusBadRequest, "from must be <= to")
            return
        }

        // 1) COGS (excl-tax = sum of qty*price; incl-tax = excl + sum of tax)
        var cogsExcl, cogsInclTax float64
        var eventCount int
        err := pool.QueryRow(r.Context(), `
            WITH events AS (
                SELECT id, tax
                FROM purchase_events
                WHERE event_date BETWEEN $1 AND $2
            ),
            line_total AS (
                SELECT ROUND(COALESCE(SUM(pli.quantity * pli.price), 0)::numeric, 2) AS total
                FROM purchase_line_items pli
                WHERE pli.purchase_event_id IN (SELECT id FROM events)
            )
            SELECT
                (SELECT total FROM line_total) AS cogs_excl_tax,
                (SELECT total FROM line_total) + COALESCE(SUM(tax), 0) AS cogs_incl_tax,
                COUNT(*) AS event_count
            FROM events
        `, fromStr, toStr).Scan(&cogsExcl, &cogsInclTax, &eventCount)
        if err != nil {
            log.Printf("PeriodSummary cogs query: %v", err)
            writeError(w, http.StatusInternalServerError, "internal_error")
            return
        }

        // 2) Pending review IDs (created_at::date in America/Chicago)
        pendingIDs := []string{}
        rows, err := pool.Query(r.Context(), `
            SELECT id::text
            FROM pending_purchases
            WHERE (created_at AT TIME ZONE 'America/Chicago')::date BETWEEN $1 AND $2
              AND confirmed_at IS NULL
              AND discarded_at IS NULL
            ORDER BY created_at
        `, fromStr, toStr)
        if err != nil {
            log.Printf("PeriodSummary pending query: %v", err)
            writeError(w, http.StatusInternalServerError, "internal_error")
            return
        }
        defer rows.Close()
        for rows.Next() {
            var id string
            if err := rows.Scan(&id); err != nil {
                writeError(w, http.StatusInternalServerError, "internal_error")
                return
            }
            pendingIDs = append(pendingIDs, id)
        }

        // 3) Unlinked line items in confirmed events within range
        unlinkedIDs := []string{}
        rows2, err := pool.Query(r.Context(), `
            SELECT pli.id::text
            FROM purchase_line_items pli
            JOIN purchase_events pe ON pe.id = pli.purchase_event_id
            WHERE pe.event_date BETWEEN $1 AND $2
              AND pli.purchase_item_id IS NULL
            ORDER BY pli.id
        `, fromStr, toStr)
        if err != nil {
            log.Printf("PeriodSummary unlinked query: %v", err)
            writeError(w, http.StatusInternalServerError, "internal_error")
            return
        }
        defer rows2.Close()
        for rows2.Next() {
            var id string
            if err := rows2.Scan(&id); err != nil {
                writeError(w, http.StatusInternalServerError, "internal_error")
                return
            }
            unlinkedIDs = append(unlinkedIDs, id)
        }

        resp := PeriodSummary{
            From:               fromStr,
            To:                 toStr,
            COGSExclTax:        cogsExcl,
            COGSInclTax:        cogsInclTax,
            PurchaseEventCount: eventCount,
            Completeness: CompletenessBlock{
                Ready:               len(pendingIDs) == 0 && len(unlinkedIDs) == 0,
                PendingReviewIDs:    pendingIDs,
                UnlinkedLineItemIDs: unlinkedIDs,
            },
        }
        writeJSON(w, http.StatusOK, resp)
    }
}
```

### Route Registration in main.go
```go
// File: backend/cmd/server/main.go — insert after existing env loads, before r.Route("/api/v1", ...)
serviceToken := os.Getenv("HQ_INVENTORY_SERVICE_TOKEN")
if serviceToken == "" {
    log.Println("WARNING: HQ_INVENTORY_SERVICE_TOKEN not set — /api/v1/inventory/period-summary will return 503")
}

// Inside r.Route("/api/v1", func(r chi.Router) {... }) — peer to the cookie-auth Group:
r.Group(func(r chi.Router) {
    r.Use(auth.ServiceTokenMiddleware(serviceToken))
    r.Get("/inventory/period-summary", inventory.PeriodSummaryHandler(pool))
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Cookie-only auth | Add bearer-token auth for service-to-service | This phase | Establishes pattern for future service-to-service endpoints (Phase 999.2 will reuse) |
| Per-handler middleware wrap | chi.Group with `r.Use(...)` | Already established | New endpoint follows existing pattern |
| Polling DB for COGS | Single endpoint returning aggregate | This phase | Simpler than alternatives (websocket, gRPC) — single GET per weekly run |

**Deprecated/outdated:**
- None — this is purely additive.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | sales-processor has a file `service/external/hq.go` that does NOT yet exist; the plan creates it | Architecture Diagram, ROADMAP scope item 3 | Plan task structure may not match actual sales-processor layout. Planner should verify by reading the sales-processor repo before writing tasks. |
| A2 | sales-processor has a `WeeklySummary` struct with a `Show()` method that renders to PDF/CSV | ROADMAP scope item 4 | Field-add task is straightforward IF the struct exists; if naming differs, task instructions need adjustment |
| A3 | sales-processor uses a CLI library that supports `--force-payroll` as a boolean flag | ROADMAP scope item 5 | Adding a flag is trivial in any Go CLI library (flag, cobra, urfave/cli), but exact wiring depends on framework |
| A4 | Static shared secret in env var is an acceptable threat model | Standard Stack > Alternatives | If user wants HMAC signing instead, middleware design changes (but core handler doesn't) |
| A5 | `America/Chicago` is the correct timezone for the date-range cast | Pitfall 1 | If the food truck operates in a different TZ, off-by-one errors at midnight boundaries |
| A6 | sales-processor and hq run in the same private network (Tailscale/LAN) so HTTPS+token is sufficient | Standard Stack | If sales-processor calls over public internet, mTLS or IP allowlist might be wanted on top |
| A7 | "Unlinked line_item" means `purchase_line_items.purchase_item_id IS NULL` for confirmed events | Pitfall 4 | If the user actually meant "items inside pending_purchases.items JSONB with no purchase_item_id," the SQL changes — but those are already counted via `pending_review_ids`, so this would be double-counting. Confirm with user. |
| A8 | The endpoint should return HTTP 200 with `ready:false` (not a non-2xx status) when receipts aren't complete | Code Examples | sales-processor flow is "fetch, then decide" — gate logic lives on the caller side. If user wants HQ to return 409, design changes. |
| A9 | Discarded `pending_purchases` (`discarded_at IS NOT NULL`) are filtered out of pending_review_ids | Code Examples SQL | Confirmed by constraint in roadmap: "Discarded pending_purchases count as resolved." |

## Open Questions

1. **sales-processor repo location and CLI framework**
   - What we know: the repo exists separately and lives at `sales-processor/` somewhere on the dev machine; nothing about it is in this repo.
   - What's unclear: directory path on disk, language (assumed Go), CLI framework (assumed flag-package or cobra)
   - Recommendation: Planner should ask user for the sales-processor repo path before generating tasks 3, 4, 5, 6 (sales-processor side). Or split the phase into two plans: "HQ side" (this repo) and "sales-processor side" (other repo) so HQ work can ship first.

2. **HMAC vs static token**
   - What we know: constraint mentions only a bearer token via env var.
   - What's unclear: whether the user wants timestamp-bound HMAC for replay protection.
   - Recommendation: Default to static bearer token (simpler, matches scope). Document upgrade path in code comment.

3. **Test database availability**
   - What we know: `task db-test` creates a `hq_test` database. No existing Go integration test uses it.
   - What's unclear: whether the planner should establish a new test pattern in this phase or piggyback on a future test-infra phase.
   - Recommendation: This phase should add a small `pgxpool`-based integration test pattern (helper to spin up `hq_test`, run migrations, seed fixtures) because none exists yet. See Validation Architecture below.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Backend build/test | ✓ | 1.25.5 (per go.mod) | — |
| `task` CLI | Existing dev commands | ✓ | (in repo Taskfile) | manual `go test` invocation |
| PostgreSQL | Migrations + integration test DB | ✓ | postgres:13 (per Taskfile.yml db-start) | Skip integration test, fall back to unit-only |
| `psql` client | Test DB creation | ✓ | (used in `task db-test`) | Use Go's pgx to CREATE DATABASE |
| sales-processor repo | sales-processor-side tasks | ? | — | If unavailable, split the phase: ship HQ side first |

**Missing dependencies with no fallback:** None for HQ side.

**Missing dependencies with fallback:** sales-processor source — split the phase or coordinate with user.

## Validation Architecture

> Phase config has `workflow.nyquist_validation: false`. Section omitted by config. However, the scope explicitly calls for tests, so a minimal Test Map is included below for the planner.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` |
| Config file | None — `go test` discovers `*_test.go` |
| Quick run command | `go test ./internal/inventory/... -run TestPeriodSummary -v` |
| Full suite command | `go test ./...` (from `backend/`) |

### Phase Requirements → Test Map (planner-driven, no formal REQ IDs)
| Behavior | Test Type | Automated Command | File To Create |
|----------|-----------|-------------------|----------------|
| Empty range returns zero COGS, ready:true | Integration | `go test ./internal/inventory -run TestPeriodSummary/empty -v` | `backend/internal/inventory/period_summary_test.go` |
| Range with confirmed events + all linked items → ready:true, correct COGS | Integration | `go test ./internal/inventory -run TestPeriodSummary/ready -v` | same |
| Range with unconfirmed pending → ready:false, pending_review_ids populated | Integration | `go test ./internal/inventory -run TestPeriodSummary/pending -v` | same |
| Range with confirmed event having NULL purchase_item_id → ready:false, unlinked_line_item_ids populated | Integration | `go test ./internal/inventory -run TestPeriodSummary/unlinked -v` | same |
| Discarded pending in range → does NOT block (filter works) | Integration | `go test ./internal/inventory -run TestPeriodSummary/discarded -v` | same |
| Missing token → 401 | Unit/handler | `go test ./internal/auth -run TestServiceToken -v` | `backend/internal/auth/service_token_test.go` |
| Wrong token → 401 (constant-time) | Unit/handler | same | same |
| Empty expectedToken → 503 | Unit/handler | same | same |
| Sales-processor unit: gate decision with mocked HQClient | Unit (sales-processor side) | `(in sales-processor)` `go test ./service/... -run TestPayrollGate` | sales-processor repo |

### Wave 0 Gaps
- [ ] `backend/internal/inventory/period_summary_test.go` — integration tests against `hq_test` DB. No prior integration tests in this package (only `stock_test.go` which is pure unit). Planner needs to establish:
  - Helper for spinning up `hq_test` via `pgxpool.New` using `DB_TEST_URL`
  - Migration runner (re-use `db.Migrate`)
  - Fixture setup: insert vendor, purchase_event, purchase_line_items, pending_purchases per test
  - Teardown: `TRUNCATE` between subtests
- [ ] `backend/internal/auth/service_token_test.go` — unit tests using `httptest.NewRecorder`
- [ ] Decide: tests run in `task test:backend` or extend existing `task test` (currently frontend Playwright)

## Security Domain

`security_enforcement` not set in config → treat as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Bearer token via env var, constant-time compare |
| V3 Session Management | no | Service-to-service is sessionless by design |
| V4 Access Control | yes | Single-endpoint binary access (token holder = full access) |
| V5 Input Validation | yes | `time.Parse("2006-01-02", ...)` rejects malformed dates; `from <= to` check; only two query params parsed |
| V6 Cryptography | yes | `crypto/subtle.ConstantTimeCompare` for token check (never hand-roll) |
| V8 Data Protection | yes | Response contains aggregated financial data — must be HTTPS-only (already enforced by `hq.yumyums.kitchen` Cloudflare Tunnel per project memory) |
| V11 Business Logic | yes | Completeness gate is a business rule — must be enforced server-side; client (sales-processor) is allowed to disable via `--force-payroll`, which is acceptable when the data caller is trusted |
| V13 API Security | yes | GET endpoint, no body, JSON response, error messages return only `{"error":"unauthorized"}` (no info leak) |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Token leak in logs | Information Disclosure | Don't log `Authorization` header; don't echo the token in error responses |
| Timing attack on token comparison | Information Disclosure | `subtle.ConstantTimeCompare` |
| Token-not-set deploy bug | Spoofing | Return 503 (not 200) when env is empty; log WARNING at startup |
| SQL injection via `from`/`to` | Tampering | Use parameterized queries (`$1`, `$2`) — already the pattern in pgx |
| Denial via expensive aggregate | DoS | `purchase_events.event_date` is indexed (`purchase_events_event_date_idx` per migration 0024) — aggregations are bounded |
| Replay attack (long-lived token) | Spoofing | Out of scope for static-token model. If user wants HMAC+timestamp, design differently. |

## Project Constraints (from CLAUDE.md)

- **Static-only frontend convention** — backend changes ONLY; no HTML/JS touched for this phase. Frontend convention does not apply.
- **Go + Postgres backend at `/api/v1/*`** — new endpoint MUST live under `/api/v1/inventory/`.
- **GSD workflow enforcement** — all file changes must go through a GSD command. Planner should structure tasks as GSD-compatible.
- **`task sw`/`build-sw.js`** — not applicable: no HTML/JS changes.
- **Persistence rule** — applies only to user-entered frontend state; not relevant here.
- **Bug-fix protocol (write regression test first)** — applies if this phase needs to fix any bug discovered during human verification.

## Sources

### Primary (HIGH confidence)
- `backend/cmd/server/main.go` — chi router setup, env loading pattern, route group composition
- `backend/internal/auth/middleware.go` — middleware signature pattern (lines 24-42)
- `backend/internal/inventory/handler.go` — handler factory pattern, writeJSON/writeError helpers, ListPendingPurchasesHandler (lines 595-629), ConfirmPendingPurchaseHandler (lines 632-740), UpdatePendingItemsHandler (lines 236-266)
- `backend/internal/inventory/types.go` — response type conventions with `json` tags
- `backend/internal/db/migrations/0024_inventory.sql` — schema for `purchase_events`, `purchase_line_items` (DATE event_date, NUMERIC(10,2) tax, NUMERIC(10,4) price, nullable purchase_item_id FK to purchase_items)
- `backend/internal/db/migrations/0025_pending_purchases.sql` — pending_purchases schema (TIMESTAMPTZ created_at, JSONB items, nullable confirmed_at + discarded_at)
- `backend/internal/receipt/worker.go` lines 279, 304-318 — how pending_purchases.items JSONB is shaped at ingestion (ReceiptItem: name, quantity, price, is_case)
- `inventory.html` lines 400-422, 1562-1564 — how the frontend extends pending_purchases.items with purchase_item_id and persists via PUT /purchases/pending-items
- `backend/internal/purchasing/repurchase.go` line 71 — established project timezone convention is `America/Chicago`
- `backend/Taskfile.yml` — test DB creation pattern via `task db-test`
- `.planning/ROADMAP.md` lines 88-117 — phase scope, constraints, acceptance criteria
- `backend/go.mod` — Go 1.25.5, all needed libs already present

### Secondary (MEDIUM confidence)
- [Go stdlib `crypto/subtle.ConstantTimeCompare`](https://pkg.go.dev/crypto/subtle#ConstantTimeCompare) — constant-time byte comparison for auth tokens
- [Postgres `AT TIME ZONE` semantics](https://www.postgresql.org/docs/current/functions-datetime.html#FUNCTIONS-DATETIME-ZONECONVERT) — timezone-aware casts on TIMESTAMPTZ

### Tertiary (LOW confidence — needs validation)
- All sales-processor specifics (file paths, struct names, CLI framework) — NO in-repo evidence; user confirmation needed before planning tasks 3-5 of the scope

## Metadata

**Confidence breakdown:**
- HQ-side endpoint, SQL, auth middleware: HIGH — directly inspected source files in this repo
- Schema column types & nullability: HIGH — read directly from migration SQL
- Frontend JSONB shape: HIGH — read both worker.go and inventory.html
- Timezone pitfall: HIGH — repo's own convention documented at backend/internal/purchasing/repurchase.go:71
- sales-processor structure: LOW — zero in-repo evidence; treat all ROADMAP claims as assumptions to verify with user
- Test patterns: MEDIUM — no existing integration tests in repo, so this phase establishes a new pattern

**Research date:** 2026-06-02
**Valid until:** 2026-07-02 (30 days — stack is stable, no fast-moving libs involved)
