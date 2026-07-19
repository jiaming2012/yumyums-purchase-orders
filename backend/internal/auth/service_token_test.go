package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServiceTokenMiddleware(t *testing.T) {
	const expected = "valid-token-xyz"

	tests := []struct {
		name           string
		token          string // value to send as expectedToken to the factory
		authHeader     string // value of Authorization header, "" means do not set
		wantStatus     int
		wantBodyHas    string
		wantNextCalled bool
	}{
		{
			name: "missing Authorization header → 401",
			token: expected, authHeader: "",
			wantStatus:     http.StatusUnauthorized,
			wantBodyHas:    "unauthorized",
			wantNextCalled: false,
		},
		{
			name: "header without Bearer prefix → 401",
			token: expected, authHeader: "valid-token-xyz", // no "Bearer " prefix
			wantStatus:     http.StatusUnauthorized,
			wantBodyHas:    "unauthorized",
			wantNextCalled: false,
		},
		{
			name: "Bearer prefix with wrong token → 401",
			token: expected, authHeader: "Bearer wrong-token",
			wantStatus:     http.StatusUnauthorized,
			wantBodyHas:    "unauthorized",
			wantNextCalled: false,
		},
		{
			name: "Bearer prefix with correct token → 200, next called",
			token: expected, authHeader: "Bearer " + expected,
			wantStatus:     http.StatusOK,
			wantBodyHas:    "ok",
			wantNextCalled: true,
		},
		{
			name: "empty expectedToken (env unset) → 503, next NOT called",
			token: "", authHeader: "Bearer anything",
			wantStatus:     http.StatusServiceUnavailable,
			wantBodyHas:    "service_token_not_configured",
			wantNextCalled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/period-summary?from=2026-01-01&to=2026-01-07", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			ServiceTokenMiddleware(tc.token)(next).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBodyHas) {
				t.Errorf("body = %q, want contains %q", rec.Body.String(), tc.wantBodyHas)
			}
			if nextCalled != tc.wantNextCalled {
				t.Errorf("nextCalled = %v, want %v", nextCalled, tc.wantNextCalled)
			}
		})
	}
}
