---
phase: 21-cogs-in-sales-processor-report-receipt-completeness-gate-bef
plan: 02
subsystem: backend/auth
tags: [go, http-middleware, auth, bearer-token, asvs-l1, security]
dependency_graph:
  requires:
    - backend/internal/auth (existing package)
    - net/http, crypto/subtle, strings (stdlib)
  provides:
    - auth.ServiceTokenMiddleware factory for bearer-token auth on service-to-service routes
  affects:
    - None at runtime — Plan 03 wires the middleware onto the chi router; this plan only adds the building block.
tech_stack:
  added: []
  patterns:
    - Constant-time token comparison via crypto/subtle.ConstantTimeCompare (timing-attack mitigation)
    - Fail-closed empty-secret guard (503 before reading the Authorization header)
    - Sentinel-only error bodies — never echo provided/expected token
    - Table-driven white-box subtests via net/http/httptest (matches inventory/stock_test.go convention)
key_files:
  created:
    - backend/internal/auth/service_token.go
    - backend/internal/auth/service_token_test.go
  modified: []
decisions:
  - Middleware lives in a NEW file alongside (not under) the cookie-session auth.Middleware so future service-to-service endpoints can mount it as a peer chi sub-group without colliding with user-session paths.
  - Empty expectedToken returns 503 BEFORE any header inspection — a misconfigured deploy fails loudly closed rather than silently open.
  - No log calls in the middleware file — token material must never enter the log stream.
metrics:
  duration_min: 2
  completed_date: "2026-06-02"
  tasks_completed: 2
  files_changed: 2
---

# Phase 21 Plan 02: ServiceTokenMiddleware Summary

Adds a stdlib-only `ServiceTokenMiddleware` factory in `backend/internal/auth/` that authenticates inbound service-to-service callers (e.g. the sales-processor in a separate repo) via a static bearer token, using `crypto/subtle.ConstantTimeCompare` and fail-closed empty-secret semantics; ships with a 5-case table-driven test that exercises every code path.

## Objective

Establish a new inbound bearer-token auth path that lives alongside (not under) the existing cookie-session `auth.Middleware`. This is the first inbound bearer auth in the repo — the pattern lives in a new file so future service-to-service endpoints can mount it as a peer chi sub-group.

Purpose: sales-processor (separate repo, separate DB, separate host) needs to call `GET /api/v1/inventory/period-summary` without a user session. A static shared secret in an env var is the pragmatic choice for a private-network deployment behind Cloudflare Tunnel.

## Tasks Completed

| # | Task                                       | Commit    | Files                                              |
| - | ------------------------------------------ | --------- | -------------------------------------------------- |
| 1 | Create ServiceTokenMiddleware              | `b4250fa` | backend/internal/auth/service_token.go             |
| 2 | Unit-test all 5 middleware paths           | `fc52f9f` | backend/internal/auth/service_token_test.go        |

## Final File Contents

### `backend/internal/auth/service_token.go`

```go
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// ServiceTokenMiddleware authenticates internal service-to-service callers via
// a static bearer token loaded from env (HQ_INVENTORY_SERVICE_TOKEN). The
// caller must send `Authorization: Bearer <token>` matching expectedToken.
//
// Behavior:
//   - If expectedToken is empty (env var unset), every request is rejected
//     with 503 Service Unavailable. This is fail-closed — a misconfigured
//     deploy must NOT silently become open-access.
//   - Otherwise: missing header, missing/wrong "Bearer " prefix, or wrong
//     token → 401 Unauthorized.
//   - On match, calls next.ServeHTTP(w, r) with no context modification —
//     this is a sessionless service caller, so no User is attached.
//
// Token comparison uses crypto/subtle.ConstantTimeCompare to prevent timing
// attacks (V6 ASVS L1). The middleware never logs or echoes either token.
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

## Test Output (`go test -v`)

```
=== RUN   TestServiceTokenMiddleware
=== RUN   TestServiceTokenMiddleware/missing_Authorization_header_→_401
=== RUN   TestServiceTokenMiddleware/header_without_Bearer_prefix_→_401
=== RUN   TestServiceTokenMiddleware/Bearer_prefix_with_wrong_token_→_401
=== RUN   TestServiceTokenMiddleware/Bearer_prefix_with_correct_token_→_200,_next_called
=== RUN   TestServiceTokenMiddleware/empty_expectedToken_(env_unset)_→_503,_next_NOT_called
--- PASS: TestServiceTokenMiddleware (0.00s)
    --- PASS: TestServiceTokenMiddleware/missing_Authorization_header_→_401 (0.00s)
    --- PASS: TestServiceTokenMiddleware/header_without_Bearer_prefix_→_401 (0.00s)
    --- PASS: TestServiceTokenMiddleware/Bearer_prefix_with_wrong_token_→_401 (0.00s)
    --- PASS: TestServiceTokenMiddleware/Bearer_prefix_with_correct_token_→_200,_next_called (0.00s)
    --- PASS: TestServiceTokenMiddleware/empty_expectedToken_(env_unset)_→_503,_next_NOT_called (0.00s)
PASS
ok  	github.com/yumyums/hq/internal/auth	0.381s
```

All 5 subtests pass against the real `ServiceTokenMiddleware` implementation.

## Verification

| Check                                                                       | Result |
| --------------------------------------------------------------------------- | ------ |
| `cd backend && go build ./...`                                              | PASS   |
| `cd backend && go build ./internal/auth/...`                                | PASS   |
| `cd backend && go vet ./internal/auth/...`                                  | PASS   |
| `cd backend && go test ./internal/auth/ -run TestServiceTokenMiddleware -v` | PASS (5/5 subtests) |
| `grep -q "func ServiceTokenMiddleware(expectedToken string)..." service_token.go` | PASS |
| `grep -q '"crypto/subtle"' service_token.go`                                | PASS   |
| `grep -q "subtle.ConstantTimeCompare" service_token.go`                     | PASS   |
| `grep -q "service_token_not_configured" service_token.go`                   | PASS   |
| `grep -q "StatusServiceUnavailable" service_token.go`                       | PASS   |
| `grep -q '"Bearer "' service_token.go`                                      | PASS   |
| `! grep -q "log\." service_token.go` (no log calls)                         | PASS   |
| `grep -c "wantStatus" service_token_test.go` ≥ 5                            | PASS (8 hits, one per case row × multiple fields) |

`grep "log\." backend/internal/auth/service_token.go` returns NO matches — token material is never logged or echoed.

## Success Criteria

1. `auth.ServiceTokenMiddleware` exported from the `auth` package with the documented signature. ✓
2. All 5 unit-test cases pass: missing header (401), no Bearer prefix (401), wrong token (401), correct token (200, next called), empty expected token (503, next NOT called). ✓
3. Token comparison uses `crypto/subtle.ConstantTimeCompare`. ✓
4. No `log.` calls in the middleware file (verified by grep). ✓
5. Middleware is NOT YET mounted on any chi route — Plan 03 wires it. ✓ (no changes to `cmd/server/main.go` in this plan)

## Deviations from Plan

None — plan executed exactly as written. Both `<action>` blocks were applied verbatim; all acceptance criteria, behaviors, and verification gates passed on first run.

## Threat Model — Mitigation Coverage

| Threat ID | Category | Mitigated In This Plan? | How                                                     |
| --------- | -------- | ----------------------- | ------------------------------------------------------- |
| T-21-A01  | Spoofing | Yes                     | Bearer token compared via subtle.ConstantTimeCompare; mismatches → 401 |
| T-21-A02  | Info Disclosure (timing)  | Yes                     | `crypto/subtle.ConstantTimeCompare([]byte, []byte) == 1` used; no `==` on strings |
| T-21-A03  | Info Disclosure (logs)    | Yes                     | Middleware contains zero `log.` calls; verified by grep |
| T-21-A04  | Info Disclosure (errors)  | Yes                     | Error bodies are constant sentinel strings; neither provided nor expected token is reflected |
| T-21-A05  | Spoofing (empty secret)   | Yes                     | `expectedToken == ""` short-circuits to 503 **before** reading the Authorization header |
| T-21-A06  | Tampering (wrong route)   | Deferred to Plan 03     | This plan only adds the building block; Plan 03 mounts it as a peer chi sub-group |
| T-21-A07  | Spoofing (rotation)       | Accepted (deferred)     | Out of scope — documented |
| T-21-A08  | Info Disclosure (chi log) | Accepted                | chi's default logger does not log request headers |

## Threat Flags

No new security-relevant surface introduced beyond the plan's `<threat_model>`. The middleware is defined but not yet mounted — surface change occurs in Plan 03.

## Notes for Plan 03

- Middleware compiles standalone; no chi route is mounted yet.
- Plan 03 should:
  - Load `HQ_INVENTORY_SERVICE_TOKEN` from env in `cmd/server/main.go`.
  - Create a peer chi sub-group (NOT under the cookie-session group) that applies `auth.ServiceTokenMiddleware(token)`.
  - Mount `GET /api/v1/inventory/period-summary` (or wherever the service-facing route lives) on that sub-group only.
  - Add an integration test that exercises 401 / 503 / 200 end-to-end through the chi router.

## Self-Check: PASSED

- File `backend/internal/auth/service_token.go` exists in worktree.
- File `backend/internal/auth/service_token_test.go` exists in worktree.
- Commit `b4250fa` exists in `git log` (`feat(21-02): add ServiceTokenMiddleware...`).
- Commit `fc52f9f` exists in `git log` (`test(21-02): cover all 5 ServiceTokenMiddleware paths`).
- All 5 subtests pass against the real implementation (captured above).
