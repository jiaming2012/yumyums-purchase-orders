package onboarding

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// OverrideVideoWatchedHandler handles POST /api/v1/onboarding/overrideVideoWatched.
//
// The escape valve when a training video's media is broken (dead URL, failed
// upload): a manager marks the part watched FOR the hire, with attribution and
// a required reason, instead of the hire being permanently unable to complete
// the checklist (operator ruling, 2026-08-26). Requires admin/manager.
//
// Body: { "hire_id": "...", "part_id": "...", "reason": "..." }
//
// Writes the SAME ob_progress row the 95%-watched path writes — the
// completeness SQL and sign-off gates need no special case — with the
// attribution JSON in the row's existing value column:
//
//	{"override": true, "by_id": "...", "by_name": "...", "reason": "...", "at": "..."}
//
// HireTraining surfaces that JSON verbatim as the part's `override` field, so
// the UI can render "marked watched by <name>" instead of implying the hire
// watched it.
func OverrideVideoWatchedHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isManagerOrAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		var body struct {
			HireID string `json:"hire_id"`
			PartID string `json:"part_id"`
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		body.Reason = strings.TrimSpace(body.Reason)
		if body.HireID == "" {
			writeError(w, http.StatusBadRequest, "hire_id_required")
			return
		}
		if body.PartID == "" {
			writeError(w, http.StatusBadRequest, "part_id_required")
			return
		}
		if body.Reason == "" {
			writeError(w, http.StatusBadRequest, "reason_required")
			return
		}

		var exists bool
		if err := pool.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM ob_video_parts WHERE id = $1)`, body.PartID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusNotFound, "part_not_found")
			return
		}
		if err := pool.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, body.HireID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusNotFound, "hire_not_found")
			return
		}

		attribution, _ := json.Marshal(map[string]any{
			"override": true,
			"by_id":    user.ID,
			"by_name":  user.DisplayName,
			"reason":   body.Reason,
			"at":       time.Now().UTC().Format(time.RFC3339),
		})

		_, err := pool.Exec(r.Context(),
			`INSERT INTO ob_progress (hire_id, item_id, progress_type, value)
			 VALUES ($1, $2, 'video_part', $3)
			 ON CONFLICT (hire_id, item_id, progress_type)
			 DO UPDATE SET value = EXCLUDED.value, checked_at = now()`,
			body.HireID, body.PartID, string(attribution))
		if err != nil {
			slog.Error("OverrideVideoWatched insert failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
