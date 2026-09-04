package redemption

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/yumyums/hq/internal/auth"
)

// submitRequest is the wire shape Card 6's scanner posts (contract of record:
// this card's merge-intent, §"What must survive any merge" #2).
type submitRequest struct {
	TokenHash       string   `json:"token_hash"`
	DeviceID        string   `json:"device_id"`
	OrderNumber     string   `json:"order_number"`
	OfflineOverride bool     `json:"offline_override"`
	UnverifiedCode  bool     `json:"unverified_code"`
	ScannedAt       string   `json:"scanned_at"` // RFC3339; empty → server now
	Value           *float64 `json:"value"`      // offer face value ($); absent → unknown
}

type submitResponse struct {
	Result             string `json:"result"`
	RaceLostReconciled bool   `json:"race_lost_reconciled"`
	Error              string `json:"error,omitempty"`
}

// SubmitHandler is POST /api/v1/marketing/redeem — the HQ endpoint the
// scanner's online submit posts to (R2) and the arbitration door for synced
// offline_override attempts. Mounted behind auth.Middleware +
// RequirePermission("marketing"); the submitting session user is recorded as
// staff server-side (never client-supplied).
//
// Every arbitration VERDICT is a 200 — the verdict is data, mapping 1:1 onto
// §19.3's SRV_* boundary events. Non-200s are transport/infrastructure:
// 400 malformed, 503 not configured (fail-closed), 504 no terminal within
// budget, 500 the F4 notification write failed after a race-lost verdict
// (loud + retryable — re-arbitrating an already_used attempt is stable).
func SubmitHandler(arb *Arbitrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			// Defence in depth — auth.Middleware already gates the mount.
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		var req submitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		if req.TokenHash == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing_token_hash"})
			return
		}
		if req.DeviceID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing_device_id"})
			return
		}
		var scannedAt time.Time
		if req.ScannedAt != "" {
			var err error
			scannedAt, err = time.Parse(time.RFC3339, req.ScannedAt)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_scanned_at"})
				return
			}
		}

		attempt := Attempt{
			TokenHash:       req.TokenHash,
			DeviceID:        req.DeviceID,
			OrderNumber:     req.OrderNumber,
			AuthorizedBy:    user.Email, // staff — server-trusted identity
			OfflineOverride: req.OfflineOverride,
			UnverifiedCode:  req.UnverifiedCode,
			ScannedAt:       scannedAt,
		}
		if req.Value != nil {
			attempt.Value = *req.Value
			attempt.ValueKnown = true
		}

		res, err := arb.Arbitrate(r.Context(), attempt)
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, submitResponse{
				Result:             res.Result,
				RaceLostReconciled: res.RaceLostReconciled,
				Error:              res.Err,
			})
		case errors.Is(err, ErrNotConfigured):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "redemption_not_configured"})
		case errors.Is(err, ErrNotificationFailed):
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "race_lost_notification_failed"})
		case errors.Is(err, ErrTimeout), errors.Is(err, context.DeadlineExceeded):
			writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "arbitration_timeout"})
		default:
			// Includes a client that went away mid-arbitration; the write is
			// then moot but harmless.
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "arbitration_failed"})
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
