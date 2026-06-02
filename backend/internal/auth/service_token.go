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
