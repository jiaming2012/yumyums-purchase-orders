package redemption

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yumyums/hq/internal/auth"
)

func submitReq(t *testing.T, body string, user *auth.User) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketing/redeem", strings.NewReader(body))
	if user != nil {
		req = req.WithContext(context.WithValue(req.Context(), auth.CtxKeyUser, user))
	}
	return req
}

var crewUser = &auth.User{ID: "u1", Email: "crew@yumyums.kitchen", DisplayName: "Crew"}

// The endpoint contract Card 6 builds against (merge-intent §2): a verdict is
// a 200 whose result maps onto §19.3's SRV_* events, and staff identity comes
// from the session, never the payload.
func TestSubmitHandlerContract(t *testing.T) {
	stub := &stubRedeemer{status: OutcomeRedeemed}
	h := SubmitHandler(testArbitrator(stub, nil))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, submitReq(t, `{
		"token_hash": "abc123",
		"device_id":  "device-a",
		"order_number": "4021",
		"scanned_at": "2026-09-04T19:42:00Z",
		"value": 2.5
	}`, crewUser))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body %s", rec.Code, rec.Body.String())
	}
	var resp submitResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Result != ResultRedeemed {
		t.Fatalf("result=%q, want redeemed", resp.Result)
	}
	if resp.RaceLostReconciled {
		t.Fatal("race_lost_reconciled true on a clean redeem")
	}
	if resp.Error != "" {
		t.Fatalf("error=%q, want empty", resp.Error)
	}
}

// A not_found verdict is a 200 with result "not_found" (SRV_NOT_FOUND) —
// the machine terminal is failed, the wire result is not.
func TestSubmitHandlerNotFoundVerdictIs200(t *testing.T) {
	stub := &stubRedeemer{status: OutcomeNotFound}
	h := SubmitHandler(testArbitrator(stub, nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, submitReq(t, `{"token_hash":"zzz","device_id":"device-a"}`, crewUser))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	var resp submitResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Result != ResultNotFound {
		t.Fatalf("result=%q, want not_found", resp.Result)
	}
}

// Fail-closed: no Redeemer configured → 503, never open-arbitration.
func TestSubmitHandlerFailClosed503(t *testing.T) {
	h := SubmitHandler(NewArbitrator(nil, nil, Config{}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, submitReq(t, `{"token_hash":"abc","device_id":"device-a"}`, crewUser))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "redemption_not_configured") {
		t.Fatalf("body %q lacks redemption_not_configured", rec.Body.String())
	}
}

func TestSubmitHandlerValidation(t *testing.T) {
	h := SubmitHandler(testArbitrator(&stubRedeemer{status: OutcomeRedeemed}, nil))
	for name, tc := range map[string]struct {
		body    string
		wantErr string
	}{
		"invalid json":       {`{`, "invalid_json"},
		"missing token_hash": {`{"device_id":"d"}`, "missing_token_hash"},
		"missing device_id":  {`{"token_hash":"h"}`, "missing_device_id"},
		"bad scanned_at":     {`{"token_hash":"h","device_id":"d","scanned_at":"yesterday"}`, "invalid_scanned_at"},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, submitReq(t, tc.body, crewUser))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status %d, want 400", name, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tc.wantErr) {
			t.Fatalf("%s: body %q lacks %q", name, rec.Body.String(), tc.wantErr)
		}
	}
}

// G6-flagged missing coverage, endpoint level: sink failure after a race-lost
// verdict → 500 race_lost_notification_failed (the merge-intent contract's
// loud + retryable arm).
func TestSubmitHandler500OnNotificationFailure(t *testing.T) {
	stub := &atomicStubArbiter{}
	arb := testArbitrator(stub, &failingSink{})
	h := SubmitHandler(arb)

	// Attempt 1 wins.
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, submitReq(t, `{"token_hash":"h500","device_id":"device-a"}`, crewUser))
	if rec1.Code != http.StatusOK {
		t.Fatalf("attempt 1 status %d, want 200", rec1.Code)
	}

	// Attempt 2: synced offline override loses → F4 write fails → 500.
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, submitReq(t, `{"token_hash":"h500","device_id":"device-b","offline_override":true}`, crewUser))
	if rec2.Code != http.StatusInternalServerError {
		t.Fatalf("attempt 2 status %d, want 500; body %s", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), "race_lost_notification_failed") {
		t.Fatalf("body %q lacks race_lost_notification_failed", rec2.Body.String())
	}
}

func TestSubmitHandlerNoSessionUser401(t *testing.T) {
	h := SubmitHandler(testArbitrator(&stubRedeemer{status: OutcomeRedeemed}, nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, submitReq(t, `{"token_hash":"h","device_id":"d"}`, nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", rec.Code)
	}
}
