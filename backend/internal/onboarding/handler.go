package onboarding

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
	"github.com/yumyums/hq/internal/photos"
)

// writeJSON sets Content-Type and encodes v as JSON.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v) //nolint:errcheck
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

// isAdmin returns true if the user has admin or superadmin privileges.
func isAdmin(user *auth.User) bool {
	return slices.Contains(user.Roles, "admin") || user.IsSuperadmin
}

// isManagerOrAdmin returns true if the user has manager, admin, or superadmin privileges.
func isManagerOrAdmin(user *auth.User) bool {
	return slices.Contains(user.Roles, "manager") || isAdmin(user)
}

// ListTemplatesHandler handles GET /api/v1/onboarding/templates.
// Returns all onboarding templates. Requires admin/manager.
func ListTemplatesHandler(pool *pgxpool.Pool) http.HandlerFunc {
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

		templates, err := GetTemplates(r.Context(), pool)
		if err != nil {
			log.Printf("GetTemplates error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if templates == nil {
			templates = []Template{}
		}
		writeJSON(w, http.StatusOK, templates)
	}
}

// GetTemplateHandler handles GET /api/v1/onboarding/templates/{id}.
// Returns a single template with full structure. Requires authentication.
func GetTemplateHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		id := chi.URLParam(r, "id")
		tmpl, err := GetTemplate(r.Context(), pool, id)
		if err != nil {
			log.Printf("GetTemplate error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, tmpl)
	}
}

// MyTrainingsHandler handles GET /api/v1/onboarding/myTrainings.
// Returns the current user's assigned templates with progress.
func MyTrainingsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		trainings, err := GetMyTrainings(r.Context(), pool, user.ID)
		if err != nil {
			log.Printf("GetMyTrainings error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if trainings == nil {
			trainings = []AssignedTemplate{}
		}
		writeJSON(w, http.StatusOK, trainings)
	}
}

// HireTrainingHandler handles GET /api/v1/onboarding/hireTraining/{hireId}.
// Returns full training detail with section states for a specific hire+template combo.
// Query param: templateId (required).
func HireTrainingHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		hireID := chi.URLParam(r, "hireId")
		templateID := r.URL.Query().Get("templateId")
		if templateID == "" {
			writeError(w, http.StatusBadRequest, "templateId_required")
			return
		}

		training, err := GetHireTraining(r.Context(), pool, hireID, templateID)
		if err != nil {
			log.Printf("GetHireTraining error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, training)
	}
}

// ManagerHiresHandler handles GET /api/v1/onboarding/managerHires.
// Returns list of hires with progress summaries. Requires admin/manager.
func ManagerHiresHandler(pool *pgxpool.Pool) http.HandlerFunc {
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

		hires, err := GetManagerHires(r.Context(), pool)
		if err != nil {
			log.Printf("GetManagerHires error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if hires == nil {
			hires = []HireOverview{}
		}
		writeJSON(w, http.StatusOK, hires)
	}
}

// SaveProgressHandler handles POST /api/v1/onboarding/saveProgress.
// Reads {item_id, progress_type, checked}. Auto-saves per item.
func SaveProgressHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var body struct {
			ItemID         string   `json:"item_id"`
			ProgressType   string   `json:"progress_type"`
			Checked        bool     `json:"checked"`
			MaxWatchedTime *float64 `json:"max_watched_time,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if body.ItemID == "" {
			writeError(w, http.StatusBadRequest, "item_id_required")
			return
		}
		if body.ProgressType == "" {
			writeError(w, http.StatusBadRequest, "progress_type_required")
			return
		}

		// Check if section is awaiting sign-off (complete + requires_sign_off + not signed off)
		if !body.Checked {
			locked, err := IsSectionLockedForEdits(r.Context(), pool, user.ID, body.ItemID, body.ProgressType)
			if err != nil {
				log.Printf("IsSectionLockedForEdits error: %v", err)
			}
			if locked {
				writeError(w, http.StatusBadRequest, "section_awaiting_signoff")
				return
			}
		}

		if err := SaveProgress(r.Context(), pool, user.ID, body.ItemID, body.ProgressType, body.Checked, body.MaxWatchedTime); err != nil {
			log.Printf("SaveProgress error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// SignOffHandler handles POST /api/v1/onboarding/signOff.
// Reads {section_id, hire_id, notes, rating}. Requires admin/manager.
// Validates notes non-empty and rating in ('ready','needs_practice','struggling').
func SignOffHandler(pool *pgxpool.Pool) http.HandlerFunc {
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
			SectionID string `json:"section_id"`
			HireID    string `json:"hire_id"`
			Notes     string `json:"notes"`
			Rating    string `json:"rating"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if body.SectionID == "" {
			writeError(w, http.StatusBadRequest, "section_id_required")
			return
		}
		if body.HireID == "" {
			writeError(w, http.StatusBadRequest, "hire_id_required")
			return
		}
		// Notes are optional; rating is required
		validRatings := map[string]bool{"ready": true, "needs_practice": true, "struggling": true}
		if !validRatings[body.Rating] {
			writeError(w, http.StatusBadRequest, "invalid_rating")
			return
		}

		// Check sign_off_roles restriction if set on the section
		var signOffRoles []string
		_ = pool.QueryRow(r.Context(),
			`SELECT sign_off_roles FROM ob_sections WHERE id = $1`, body.SectionID,
		).Scan(&signOffRoles)
		if len(signOffRoles) > 0 {
			hasRole := false
			for _, r := range signOffRoles {
				for _, ur := range user.Roles {
					if r == ur {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}
			if !user.IsSuperadmin && !hasRole {
				writeError(w, http.StatusForbidden, "sign_off_role_required")
				return
			}
		}

		input := SignOffInput{
			ManagerID: user.ID,
			SectionID: body.SectionID,
			HireID:    body.HireID,
			Notes:     body.Notes,
			Rating:    body.Rating,
		}
		if err := SignOff(r.Context(), pool, input); err != nil {
			log.Printf("SignOff error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// CreateTemplateHandler handles POST /api/v1/onboarding/createTemplate.
// Reads full template structure. Requires admin/manager.
func CreateTemplateHandler(pool *pgxpool.Pool) http.HandlerFunc {
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

		var input CreateTemplateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if input.Name == "" {
			writeError(w, http.StatusBadRequest, "name_required")
			return
		}

		id, err := CreateTemplate(r.Context(), pool, input)
		if err != nil {
			log.Printf("CreateTemplate error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"id": id})
	}
}

// UpdateTemplateHandler handles PUT /api/v1/onboarding/updateTemplate/{id}.
// Reads full template structure and performs a full replace. Requires admin/manager.
func UpdateTemplateHandler(pool *pgxpool.Pool) http.HandlerFunc {
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

		id := chi.URLParam(r, "id")
		var input CreateTemplateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		if err := UpdateTemplate(r.Context(), pool, id, input); err != nil {
			log.Printf("UpdateTemplate error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// AssignTemplateHandler handles POST /api/v1/onboarding/assignTemplate.
// Reads {hire_id, template_id}. Requires admin/manager.
func AssignTemplateHandler(pool *pgxpool.Pool) http.HandlerFunc {
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
			HireID     string `json:"hire_id"`
			TemplateID string `json:"template_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if body.HireID == "" || body.TemplateID == "" {
			writeError(w, http.StatusBadRequest, "hire_id_and_template_id_required")
			return
		}

		if err := AssignTemplate(r.Context(), pool, body.HireID, body.TemplateID, user.ID); err != nil {
			log.Printf("AssignTemplate error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// UnassignTemplateHandler handles POST /api/v1/onboarding/unassignTemplate.
// Reads {hire_id, template_id}. Requires admin/manager.
func UnassignTemplateHandler(pool *pgxpool.Pool) http.HandlerFunc {
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
			HireID     string `json:"hire_id"`
			TemplateID string `json:"template_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if body.HireID == "" || body.TemplateID == "" {
			writeError(w, http.StatusBadRequest, "hire_id_and_template_id_required")
			return
		}

		if err := UnassignTemplate(r.Context(), pool, body.HireID, body.TemplateID); err != nil {
			log.Printf("UnassignTemplate error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// DeleteTemplateHandler handles DELETE /api/v1/onboarding/deleteTemplate/{id}.
// Soft-deletes an onboarding template by setting archived_at. Requires admin/manager.
func DeleteTemplateHandler(pool *pgxpool.Pool) http.HandlerFunc {
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

		id := chi.URLParam(r, "id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id_required")
			return
		}

		_, err := pool.Exec(r.Context(), `UPDATE ob_templates SET archived_at = now() WHERE id = $1`, id)
		if err != nil {
			log.Printf("DeleteTemplate error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// VideoPresignHandler handles POST /api/v1/videos/presign.
// Generates a presigned PUT URL for direct browser upload to DO Spaces.
// Requires admin/manager. Validates content_type against allowed video MIME types.
func VideoPresignHandler(presigner *s3.PresignClient, bucket, endpoint string) http.HandlerFunc {
	allowedContentTypes := map[string]bool{
		"video/mp4":       true,
		"video/quicktime": true,
		"video/webm":      true,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if presigner == nil {
			writeError(w, http.StatusServiceUnavailable, "video_storage_not_configured")
			return
		}

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
			TemplateID  string `json:"template_id"`
			PartID      string `json:"part_id"`
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if body.TemplateID == "" || body.PartID == "" || body.Filename == "" || body.ContentType == "" {
			writeError(w, http.StatusBadRequest, "template_id_part_id_filename_content_type_required")
			return
		}
		if !allowedContentTypes[body.ContentType] {
			writeError(w, http.StatusBadRequest, "invalid_content_type")
			return
		}

		key := "videos/onboarding/" + body.TemplateID + "/" + body.PartID + "/" + body.Filename
		putURL, err := photos.GeneratePresignedPutURL(r.Context(), presigner, bucket, key, body.ContentType, 30*time.Minute)
		if err != nil {
			log.Printf("VideoPresignHandler presign error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"url":        putURL,
			"object_key": key,
			"public_url": photos.PublicURL(endpoint, bucket, key),
		})
	}
}

// VideoProcessHandler handles POST /api/v1/videos/process.
// Triggers FFmpeg conversion and thumbnail extraction in a background goroutine.
// Returns 202 Accepted immediately. Requires admin/manager.
func VideoProcessHandler(presigner *s3.PresignClient, bucket, endpoint string, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if presigner == nil {
			writeError(w, http.StatusServiceUnavailable, "video_storage_not_configured")
			return
		}

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
			PartID    string `json:"part_id"`
			ObjectKey string `json:"object_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if body.PartID == "" || body.ObjectKey == "" {
			writeError(w, http.StatusBadRequest, "part_id_and_object_key_required")
			return
		}

		result, err := processVideo(r.Context(), presigner, bucket, endpoint, pool, body.PartID, body.ObjectKey)
		if err != nil {
			log.Printf("VideoProcessHandler error (part %s): %v", body.PartID, err)
			writeError(w, http.StatusInternalServerError, "processing_failed")
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}
