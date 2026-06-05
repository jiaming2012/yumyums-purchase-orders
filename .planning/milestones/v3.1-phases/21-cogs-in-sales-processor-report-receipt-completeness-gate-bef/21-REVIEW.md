---
phase: 21-cogs-in-sales-processor-report-receipt-completeness-gate-bef
reviewed: 2026-06-02T00:00:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - CLAUDE.md
  - backend/cmd/server/main.go
  - backend/internal/auth/service_token.go
  - backend/internal/auth/service_token_test.go
  - backend/internal/inventory/handler.go
  - backend/internal/inventory/period_summary_test.go
  - backend/internal/inventory/types.go
findings:
  critical: 0
  warning: 3
  info: 6
  total: 9
status: issues_found
---

# Phase 21: Code Review Report

**Reviewed:** 2026-06-02T00:00:00Z
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Phase 21 adds a new service-to-service HTTP endpoint (`GET /api/v1/inventory/period-summary`) gated by a bearer-token middleware, plus a Postgres-backed integration test. The core security posture is solid:

- The middleware uses `crypto/subtle.ConstantTimeCompare` (timing-attack mitigation).
- Fail-closed behavior is correct: empty `expectedToken` returns 503 BEFORE any header inspection, so a misconfigured deploy cannot silently become open-access.
- All SQL is parameterized — no string interpolation of user input.
- Error responses to clients are constant strings; full driver errors go to `log.Printf` only.
- The new `r.Group` is correctly placed as a PEER of (not a child of) the cookie-auth group, so a valid session cookie cannot bypass the bearer-token gate.
- The middleware contains zero `log.` calls — token material cannot leak via logs.
- The handler initializes ID slices as `[]string{}` so JSON renders `[]` not `null`.
- The integration test correctly uses `db.Migrate(pool)` to share schema with production and `TRUNCATE ... RESTART IDENTITY CASCADE` between subtests.

No Critical issues found. Three Warnings worth fixing before/soon after merge, and six Info items that are quality / maintainability suggestions rather than bugs.

## Warnings

### WR-01: COGS query silently coerces NULL to NaN-like state on empty range (subtle DB-driver risk)

**File:** `backend/internal/inventory/handler.go:1088-1103`
**Issue:** The CTE structure relies on `COALESCE(SUM(...), 0)` inside the `lines` CTE — good. But the outer `SELECT (SELECT total FROM lines) + COALESCE(SUM(tax), 0)` happens against the `events` table. When `events` is empty:
- `events` has 0 rows.
- `lines.total` is `0` (because of `COALESCE` on the `SUM`).
- The outer `SELECT ... FROM events` produces ONE row (aggregate without GROUP BY), with `COALESCE(SUM(tax), 0) = 0` and `COUNT(*) = 0`.

This is correct, but the construction is fragile: changing the outer `SELECT` to a non-aggregate form (e.g. adding a join) would suddenly produce zero rows on empty input and `QueryRow().Scan()` would return `pgx.ErrNoRows`, which the handler maps to a generic 500. There is no test that exercises the "empty range with no fixtures" path through the actual SQL — the existing tests always insert at least one event before calling the handler in the success cases.

**Fix:** Add an explicit empty-range subtest to `period_summary_test.go` that asserts `200` + `cogs_excl_tax: 0`, `cogs_incl_tax: 0`, `purchase_event_count: 0`, `ready: true`, both ID lists `[]`:
```go
t.Run("empty range returns zeros and ready=true", func(t *testing.T) {
    resetFixtures(t)
    code, got := callHandler(t, from, to)
    if code != http.StatusOK { t.Fatalf("status=%d", code) }
    if got.COGSExclTax != 0 || got.COGSInclTax != 0 || got.PurchaseEventCount != 0 {
        t.Errorf("expected all-zero aggregate, got %+v", got)
    }
    if !got.Completeness.Ready { t.Errorf("Ready=false, want true on empty range") }
})
```
This locks in the current (correct) behavior and surfaces a regression if the SQL is ever restructured.

### WR-02: `http.Error` sets `Content-Type: text/plain` while body is JSON

**File:** `backend/internal/auth/service_token.go:28, 34, 39`
**Issue:** `http.Error()` from `net/http` unconditionally sets `Content-Type: text/plain; charset=utf-8` (and `X-Content-Type-Options: nosniff`), but the body shipped is a JSON document: `{"error":"unauthorized"}` / `{"error":"service_token_not_configured"}`. A strict client (e.g. sales-processor doing `Content-Type` sniffing or using a JSON-only HTTP client) may fail to parse the error body or surface a content-type mismatch warning.

The 200/JSON success path (via `writeJSON`) DOES set `application/json`. The mismatch between 4xx/5xx and 2xx response media types is an interop hazard for the cross-repo caller documented in `21-SALES-PROCESSOR-CONTRACT.md`.

**Fix:** Use the same response style as `inventory.writeError` (which is local to that package, so define a private helper in `auth`):
```go
func writeJSONError(w http.ResponseWriter, status int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.WriteHeader(status)
    _, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
```
Then replace the three `http.Error(...)` calls. Bonus: this also drops the trailing `\n` that `http.Error` appends, which is irrelevant for JSON consumers but cleaner on the wire.

### WR-03: Integration test will fail with `dup key` if run twice without `task db-test` resetting the database

**File:** `backend/internal/inventory/period_summary_test.go:101-130` (`insertEventAndLine`)
**Issue:** `bank_tx_id` is `UNIQUE NOT NULL` on `purchase_events` (per `0024_inventory.sql`). The test helper builds it from `"tx-"+eventDate+"-"+strconv.Itoa(int(price*10000))`. Inside one subtest this is collision-free because subtests call `resetFixtures` first (TRUNCATE). But if a test panics mid-subtest (before `t.Cleanup` registered, before `resetFixtures` ran in the NEXT subtest), the next subtest's `resetFixtures` still recovers via TRUNCATE — so the production code is safe.

The actual hazard is co-existence: if a developer is debugging by running a single subtest with `-run TestPeriodSummary/ready=true_with` against a database that already has rows (because TestMain skips TRUNCATE on its initial setup), the first `insertEventAndLine` call would fail with `duplicate key value violates unique constraint "purchase_events_bank_tx_id_key"` and the failure message would be opaque.

Additionally, `vendors.name` has a `UNIQUE` constraint (per `CreateVendorHandler` using `ON CONFLICT (name)`), and `purchase_items.description` has one too (per `CreateItemHandler` using `ON CONFLICT (description)`). Since every subtest calls `insertVendor(t, "Acme")` and `insertPurchaseItem(t, "Salmon")`, the test relies on `resetFixtures` running successfully BEFORE each insert.

**Fix:** Add a one-shot truncate at the start of `TestPeriodSummary` itself (before the subtest loop), so the very first subtest is also guaranteed a clean slate even when DB state pre-dates the test run:
```go
func TestPeriodSummary(t *testing.T) {
    if testPool == nil {
        t.Skip("DB_TEST_URL not reachable; skipping integration test")
    }
    resetFixtures(t)  // ensure clean state even if prior test run left rows

    const from = "2026-05-25"
    ...
}
```
Each subtest's own `resetFixtures(t)` call covers the inter-subtest case. The top-level call covers the "first subtest, DB had leftover rows" case.

## Info

### IN-01: `pool.Close()` is skipped if `m.Run()` panics

**File:** `backend/internal/inventory/period_summary_test.go:38-41`
**Issue:** `TestMain` runs `code := m.Run()` then `pool.Close()` then `os.Exit(code)`. If a subtest panics in an un-recovered way (rare in `testing` since `t.Fatal` doesn't panic, but possible from `_, _ = w.Write` etc.), `pool.Close()` may not run — though in practice the `os.Exit` will terminate the process and the OS reclaims the pool's connections.
**Fix:** Use `defer pool.Close()` before `m.Run()`. `os.Exit` skips defers, but a panic in `m.Run` does not call `os.Exit` directly, so the defer would still fire. Cleaner pattern:
```go
testPool = pool
defer pool.Close()
os.Exit(m.Run())
```
Note: `os.Exit` does skip defers, so the cleanest restructure is to extract the run logic into a helper that returns the exit code, and let the `defer` fire before the outer `os.Exit`. Minor housekeeping.

### IN-02: `boolByte()` helper is unnecessarily indirect

**File:** `backend/internal/inventory/period_summary_test.go:148, 155-160`
**Issue:** `boolByte(confirmed)` returns `0` or `1` as a `byte`, then is cast `int(boolByte(...))`. This is a clever-but-opaque way to compute a discriminator for the synthetic `bank_tx_id`. A reader has to trace through the function to understand intent.
**Fix:** Inline a clearer expression:
```go
disc := 0
if confirmed { disc += 1 }
if discarded { disc += 2 }
... "pp-tx-"+createdAt+strconv.Itoa(disc) ...
```
Or simply append a random suffix via `t.Name()` to make ID uniqueness explicit.

### IN-03: No rate limiting on bearer-token endpoint

**File:** `backend/internal/auth/service_token.go:24-45`
**Issue:** The endpoint is rate-unlimited at the application layer. With a high-entropy 32-byte secret behind Cloudflare Tunnel the brute-force surface is small, but an attacker who can reach `:8080` directly could attempt unlimited guesses. The threat model in `21-02-PLAN.md` (T-21-A07) explicitly accepts token rotation as out-of-scope; rate limiting is a related defense-in-depth gap.
**Fix:** Not required for v1 — record as a future hardening item in `21-SALES-PROCESSOR-CONTRACT.md` "Open assumptions". If addressed later, add a chi `httprate.LimitByIP(10, 1*time.Minute)` to the new sub-group only (so legit cookie traffic is unaffected).

### IN-04: `defer rows.Close()` after a sibling `defer rows2.Close()` in the same scope

**File:** `backend/internal/inventory/handler.go:1127, 1159`
**Issue:** Both `rows` and `rows2` are closed via `defer` at function exit. `rows` (the pending-purchase rows) stays open after the loop completes successfully, even though it's drained. This is harmless (pgx releases the underlying connection on the next pool acquisition for `rows2`) and idiomatic, but for very latency-sensitive paths the rows-handle holds the pgx connection until both queries are done and the function returns. Since `rows` is fully iterated and `rows.Err()` is checked, calling `rows.Close()` explicitly before issuing the second query would release the connection back to the pool sooner.
**Fix:** Replace the first `defer rows.Close()` with an explicit `rows.Close()` after the `rows.Err()` check. Pattern:
```go
// ... loop, then rows.Err() check ...
rows.Close()
// then start rows2, err := pool.Query(...)
```
Negligible for this endpoint's traffic shape (1 call per week per sales-processor run).

### IN-05: `inventory.CreatePurchaseEventHandler` calls `auth.UserFromContext` — does this matter for the new peer group?

**File:** `backend/internal/inventory/handler.go:537`
**Issue:** Not introduced by this phase, but worth flagging while reviewing peer-group correctness: several existing handlers (`CreatePurchaseEventHandler` at line 537, `ConfirmPendingPurchaseHandler` at line 634) call `auth.UserFromContext(r.Context())` and dereference the returned user. The new peer group's `ServiceTokenMiddleware` does NOT attach a user to context. If a future plan accidentally adds one of these existing handlers to the service-token group, it would `nil`-deref on the user. The new `PeriodSummaryHandler` correctly avoids this — it makes no auth.UserFromContext call.
**Fix:** Add a defensive comment near the new peer group in `main.go`:
```go
// Service-to-service (no cookie session) — handlers in this group MUST NOT
// call auth.UserFromContext (no user is attached).
r.Group(func(r chi.Router) { ... })
```
This is documentation only — no code change. Helps future maintainers avoid the trap.

### IN-06: Test helper inserts use `t.Context()` (Go 1.24+); confirm CI uses Go ≥1.24

**File:** `backend/internal/inventory/period_summary_test.go:46, 75, 89, 104, 114, 120, 148`
**Issue:** `t.Context()` was added in Go 1.24. The repo's `go.mod` specifies `go 1.25.5`, so this is fine in CI, but worth noting in `21-03-SUMMARY.md` for the sales-processor team who may be on an older Go toolchain when consuming this repo's patterns.
**Fix:** No code change. Mention the Go-version requirement in the summary doc.

---

_Reviewed: 2026-06-02T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
