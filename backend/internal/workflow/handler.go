package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
	opsync "github.com/yumyums/hq/internal/sync"
)

// validateFailNotes checks that every response with a triggered fail condition
// has a corresponding fail note with a non-empty description and severity.
func validateFailNotes(ctx context.Context, pool *pgxpool.Pool, input SubmitChecklistInput) error {
	tmpl, err := getTemplateByID(ctx, pool, input.TemplateID)
	if err != nil || tmpl == nil {
		return nil // template validation handled elsewhere
	}

	// Build field map from template
	fieldMap := map[string]Field{}
	for _, sec := range tmpl.Sections {
		for _, f := range sec.Fields {
			fieldMap[f.ID] = f
		}
	}

	// Build fail note map from input
	failNoteMap := map[string]bool{}
	for _, fn := range input.FailNotes {
		if fn.Note != "" && fn.Severity != nil && *fn.Severity != "" {
			failNoteMap[fn.FieldID] = true
		}
	}

	// Check each response: if the field has a fail_trigger and the value triggers it,
	// there must be a fail note
	for _, resp := range input.Responses {
		f, ok := fieldMap[resp.FieldID]
		if !ok || len(f.FailTrigger) == 0 || string(f.FailTrigger) == "null" {
			continue
		}

		if evaluateFailTrigger(f.FailTrigger, resp.Value) && !failNoteMap[resp.FieldID] {
			return fmt.Errorf("corrective_action_required")
		}
	}
	return nil
}

// evaluateFailTrigger checks if a value triggers a fail condition.
func evaluateFailTrigger(trigger json.RawMessage, value json.RawMessage) bool {
	var ft struct {
		Type string   `json:"type"`
		Min  *float64 `json:"min"`
		Max  *float64 `json:"max"`
	}
	if err := json.Unmarshal(trigger, &ft); err != nil {
		return false
	}
	if ft.Type != "out_of_range" {
		return false
	}

	var num float64
	if err := json.Unmarshal(value, &num); err != nil {
		return false
	}

	if ft.Min != nil && ft.Max != nil {
		return num < *ft.Min || num > *ft.Max
	}
	if ft.Min != nil {
		return num < *ft.Min
	}
	if ft.Max != nil {
		return num > *ft.Max
	}
	return false
}

// isAdmin returns true if the user has admin or superadmin privileges (D-11).
func isAdmin(user *auth.User) bool {
	return user.Role == "admin" || user.IsSuperadmin
}

// hasApprover returns true if at least one assignment has role "approver".
func hasApprover(assignments []AssignmentInput) bool {
	for _, a := range assignments {
		if a.AssignmentRole == "approver" {
			return true
		}
	}
	return false
}

// writeJSON sets Content-Type and encodes v as JSON.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ListTemplatesHandler handles GET /api/v1/workflow/templates.
// Returns all non-archived templates as a JSON array.
func ListTemplatesHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		templates, err := listTemplates(r.Context(), pool)
		if err != nil {
			log.Printf("listTemplates error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if templates == nil {
			templates = []Template{}
		}
		writeJSON(w, http.StatusOK, templates)
	}
}

// CreateTemplateHandler handles POST /api/v1/workflow/createTemplate.
// Admin-only (D-11). Creates a new template and returns its ID.
func CreateTemplateHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		var input TemplateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if input.RequiresApproval && !hasApprover(input.Assignments) {
			writeError(w, http.StatusBadRequest, "requires_approver")
			return
		}

		id, err := insertTemplate(r.Context(), pool, input, user.ID)
		if err != nil {
			log.Printf("insertTemplate error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if payload, merr := json.Marshal(map[string]any{"template_id": id, "name": input.Name}); merr == nil {
			opsync.EmitOp(pool, opsync.OpInput{
				DeviceID:   "server",
				UserID:     user.ID,
				EntityID:   id,
				EntityType: "template",
				OpType:     opsync.OpSaveTemplate,
				Payload:    json.RawMessage(payload),
				LamportTS:  0,
			})
		} else {
			log.Printf("CreateTemplateHandler: failed to marshal op payload: %v", merr)
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// UpdateTemplateHandler handles PUT /api/v1/workflow/updateTemplate/{id}.
// Admin-only (D-11). Full replace of template content (D-09).
func UpdateTemplateHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		templateID := chi.URLParam(r, "id")
		if templateID == "" {
			writeError(w, http.StatusBadRequest, "missing_id")
			return
		}

		var input TemplateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if input.RequiresApproval && !hasApprover(input.Assignments) {
			writeError(w, http.StatusBadRequest, "requires_approver")
			return
		}

		if err := replaceTemplate(r.Context(), pool, templateID, input); err != nil {
			log.Printf("replaceTemplate error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if payload, merr := json.Marshal(map[string]any{"template_id": templateID}); merr == nil {
			opsync.EmitOp(pool, opsync.OpInput{
				DeviceID:   "server",
				UserID:     user.ID,
				EntityID:   templateID,
				EntityType: "template",
				OpType:     opsync.OpSaveTemplate,
				Payload:    json.RawMessage(payload),
				LamportTS:  0,
			})
		} else {
			log.Printf("UpdateTemplateHandler: failed to marshal op payload: %v", merr)
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// ArchiveTemplateHandler handles DELETE /api/v1/workflow/archiveTemplate/{id}.
// Admin-only (D-11). Soft-deletes a template by setting archived_at (D-07).
func ArchiveTemplateHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		templateID := chi.URLParam(r, "id")
		if templateID == "" {
			writeError(w, http.StatusBadRequest, "missing_id")
			return
		}

		if err := archiveTemplate(r.Context(), pool, templateID); err != nil {
			log.Printf("archiveTemplate error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if payload, merr := json.Marshal(map[string]any{"template_id": templateID}); merr == nil {
			opsync.EmitOp(pool, opsync.OpInput{
				DeviceID:   "server",
				UserID:     user.ID,
				EntityID:   templateID,
				EntityType: "template",
				OpType:     opsync.OpArchiveTemplate,
				Payload:    json.RawMessage(payload),
				LamportTS:  0,
			})
		} else {
			log.Printf("ArchiveTemplateHandler: failed to marshal op payload: %v", merr)
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// MyChecklistsHandler handles GET /api/v1/workflow/myChecklists.
// Returns today's assigned templates and submissions for the authenticated user.
// Also runs draft cleanup as a fire-and-forget side effect (D-22).
func MyChecklistsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// Fire-and-forget draft cleanup
		go func() {
			if err := cleanupOldDrafts(r.Context(), pool); err != nil {
				log.Printf("cleanupOldDrafts error: %v", err)
			}
		}()

		// Accept optional ?dow= from client to handle timezone differences
		var clientDOW *int
		if dowStr := r.URL.Query().Get("dow"); dowStr != "" {
			if v, err := strconv.Atoi(dowStr); err == nil && v >= 0 && v <= 6 {
				clientDOW = &v
			}
		}
		templates, submissions, err := myChecklists(r.Context(), pool, user.ID, clientDOW)
		if err != nil {
			log.Printf("myChecklists error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		drafts, err := myDrafts(r.Context(), pool, user.ID)
		if err != nil {
			log.Printf("myDrafts error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if templates == nil {
			templates = []Template{}
		}
		if submissions == nil {
			submissions = []Submission{}
		}
		if drafts == nil {
			drafts = []FieldResponse{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"templates":   templates,
			"submissions": submissions,
			"drafts":      drafts,
		})
	}
}

// MyHistoryHandler handles GET /api/v1/workflow/myHistory.
// Returns the authenticated user's last 50 submissions.
func MyHistoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		submissions, err := myHistory(r.Context(), pool, user.ID)
		if err != nil {
			log.Printf("myHistory error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if submissions == nil {
			submissions = []Submission{}
		}
		writeJSON(w, http.StatusOK, submissions)
	}
}

// SaveResponseHandler handles POST /api/v1/workflow/saveResponse.
// Upserts a draft field response for auto-save (D-21). Returns 204 No Content.
func SaveResponseHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var input SaveResponseInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		if err := saveResponse(r.Context(), pool, input.FieldID, input.Value, user.ID); err != nil {
			log.Printf("saveResponse error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if payload, merr := json.Marshal(map[string]any{"field_id": input.FieldID, "value": input.Value}); merr == nil {
			opsync.EmitOp(pool, opsync.OpInput{
				DeviceID:   "server",
				UserID:     user.ID,
				EntityID:   input.FieldID,
				EntityType: "field_response",
				OpType:     opsync.OpSetField,
				Payload:    json.RawMessage(payload),
				LamportTS:  0,
			})
		} else {
			log.Printf("SaveResponseHandler: failed to marshal op payload: %v", merr)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// SubmitChecklistHandler handles POST /api/v1/workflow/submitChecklist.
// Creates a submission with idempotency key protection (D-15).
// Returns 409 if the template is archived (D-14).
func SubmitChecklistHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var input SubmitChecklistInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		// Validate: fields with triggered fail conditions must have a corrective action
		if err := validateFailNotes(r.Context(), pool, input); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		id, err := submitChecklist(r.Context(), pool, input, user.ID)
		if err != nil {
			if err == ErrTemplateArchived {
				writeError(w, http.StatusConflict, "template_archived")
				return
			}
			log.Printf("submitChecklist error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if payload, merr := json.Marshal(map[string]any{"submission_id": id, "template_id": input.TemplateID}); merr == nil {
			opsync.EmitOp(pool, opsync.OpInput{
				DeviceID:   "server",
				UserID:     user.ID,
				EntityID:   id,
				EntityType: "submission",
				OpType:     opsync.OpSubmitChecklist,
				Payload:    json.RawMessage(payload),
				LamportTS:  0,
			})
		} else {
			log.Printf("SubmitChecklistHandler: failed to marshal op payload: %v", merr)
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// PendingApprovalsHandler handles GET /api/v1/workflow/pendingApprovals.
// Returns submissions pending approval where the user is assigned as approver (D-23).
func PendingApprovalsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		submissions, err := pendingApprovals(r.Context(), pool, user.ID)
		if err != nil {
			log.Printf("pendingApprovals error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if submissions == nil {
			submissions = []Submission{}
		}
		writeJSON(w, http.StatusOK, submissions)
	}
}

// ApproveSubmissionHandler handles POST /api/v1/workflow/approveSubmission.
// Marks a pending submission as approved.
func ApproveSubmissionHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var body struct {
			SubmissionID string `json:"submission_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SubmissionID == "" {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		if err := approveSubmission(r.Context(), pool, body.SubmissionID, user.ID); err != nil {
			log.Printf("approveSubmission error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if payload, merr := json.Marshal(map[string]any{"submission_id": body.SubmissionID}); merr == nil {
			opsync.EmitOp(pool, opsync.OpInput{
				DeviceID:   "server",
				UserID:     user.ID,
				EntityID:   body.SubmissionID,
				EntityType: "submission",
				OpType:     opsync.OpApproveItem,
				Payload:    json.RawMessage(payload),
				LamportTS:  0,
			})
		} else {
			log.Printf("ApproveSubmissionHandler: failed to marshal op payload: %v", merr)
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// RejectItemHandler handles POST /api/v1/workflow/rejectItem.
// Inserts a rejection record and marks the submission as rejected (D-06).
func RejectItemHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var input RejectItemInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		if err := rejectItem(r.Context(), pool, input, user.ID); err != nil {
			log.Printf("rejectItem error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if payload, merr := json.Marshal(map[string]any{"submission_id": input.SubmissionID, "field_id": input.FieldID, "note": input.Comment}); merr == nil {
			opsync.EmitOp(pool, opsync.OpInput{
				DeviceID:   "server",
				UserID:     user.ID,
				EntityID:   input.SubmissionID,
				EntityType: "submission",
				OpType:     opsync.OpRejectItem,
				Payload:    json.RawMessage(payload),
				LamportTS:  0,
			})
		} else {
			log.Printf("RejectItemHandler: failed to marshal op payload: %v", merr)
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}
