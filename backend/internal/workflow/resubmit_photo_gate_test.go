package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yumyums/hq/internal/auth"
)

// newUUID returns a fresh UUID string from the DB, for a valid idempotency key.
func newUUID(t *testing.T) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(t.Context(), `SELECT gen_random_uuid()::text`).Scan(&id); err != nil {
		t.Fatalf("gen_random_uuid: %v", err)
	}
	return id
}

// seedRejectedSubmission inserts a prior submission by userID for templateID and
// rejects fieldID with the given require_photo flag — mirroring an approver who
// bounced the field back demanding a photo.
func seedRejectedSubmission(t *testing.T, templateID, fieldID, userID string, requirePhoto bool) string {
	t.Helper()
	var subID string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO checklist_submissions (template_id, template_snapshot, submitted_by, status)
		 VALUES ($1, '{}'::jsonb, $2, 'rejected')
		 RETURNING id::text`,
		templateID, userID).Scan(&subID); err != nil {
		t.Fatalf("seed rejected submission: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_submissions WHERE id=$1`, subID)
	})
	rejecter := ensureUser(t, "approver-resubmit@yumyums.test")
	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO submission_rejections (submission_id, field_id, comment, require_photo, rejected_by)
		 VALUES ($1, $2, 'Redo with a photo', $3, $4)`,
		subID, fieldID, requirePhoto, rejecter); err != nil {
		t.Fatalf("seed rejection: %v", err)
	}
	return subID
}

// driveSubmit posts a resubmit for fieldID with the given raw response value and
// returns the recorder — the direct-API path a raw client controls.
func driveSubmit(t *testing.T, userID, templateID, fieldID string, rawValue string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"template_id":%q,"idempotency_key":%q,"responses":[{"field_id":%q,"value":%s}]}`,
		templateID, newUUID(t), fieldID, rawValue)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/submitChecklist", strings.NewReader(body))
	ctx := context.WithValue(req.Context(), auth.CtxKeyUser, &auth.User{ID: userID, DisplayName: "Crew", Roles: []string{"member"}})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	SubmitChecklistHandler(testPool).ServeHTTP(rec, req)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM checklist_submissions WHERE template_id=$1 AND submitted_by=$2 AND status='pending'`,
			templateID, userID)
	})
	return rec
}

// TestResubmit_RejectedRequirePhoto_BlockedWithoutPhoto is the RED-first case:
// a direct-API resubmit of a field the approver rejected with require_photo=true,
// carrying no photo (value `true`), must be blocked server-side. Pre-fix the
// handler only runs validateFailNotes — which does not fire for a plain checkbox —
// so the resubmit SUCCEEDS (201). Post-fix the DB-resolved gate returns 400
// resubmit_photo_required.
func TestResubmit_RejectedRequirePhoto_BlockedWithoutPhoto(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "crew-resubmit-block@yumyums.test")
	tmplID, fieldID := seedTemplateWithField(t, userID)
	seedRejectedSubmission(t, tmplID, fieldID, userID, true)

	rec := driveSubmit(t, userID, tmplID, fieldID, `true`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 blocking resubmit-without-photo, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "resubmit_photo_required") {
		t.Fatalf("expected resubmit_photo_required error, got body=%s", rec.Body.String())
	}
}

// TestResubmit_RejectedRequirePhoto_SucceedsWithPhoto is the positive control:
// the same rejected-with-require_photo field resubmitted WITH an https:// photo
// URL must pass — no over-blocking.
func TestResubmit_RejectedRequirePhoto_SucceedsWithPhoto(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "crew-resubmit-ok@yumyums.test")
	tmplID, fieldID := seedTemplateWithField(t, userID)
	seedRejectedSubmission(t, tmplID, fieldID, userID, true)

	rec := driveSubmit(t, userID, tmplID, fieldID, `"https://cdn.example.com/resubmit.jpg"`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for resubmit WITH photo, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestResubmit_RejectedRequirePhoto_SucceedsWithCorrectionPhoto proves a
// NON-photo field (a checkbox, value `true`) satisfies the require_photo gate
// via the dedicated `_correction_photo` bundle the fill UI attaches, since the
// field's own value can't be a photo URL. RED before hasResubmitPhoto (the gate
// only accepted the value itself being an https URL → 400); GREEN after.
func TestResubmit_RejectedRequirePhoto_SucceedsWithCorrectionPhoto(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "crew-resubmit-correction@yumyums.test")
	tmplID, fieldID := seedTemplateWithField(t, userID)
	seedRejectedSubmission(t, tmplID, fieldID, userID, true)

	// The fill UI JSON-stringifies the bundle, so the response value is a JSON
	// string literal whose content is {"_v":true,"_correction_photo":"https://…"}.
	inner := `{"_v":true,"_correction_photo":"https://cdn.example.com/correction.jpg"}`
	b, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	rec := driveSubmit(t, userID, tmplID, fieldID, string(b))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for resubmit WITH correction photo, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// seedTemplateWithSubStep creates a checkbox parent ("Cut the check") with one
// sub-step ("Do B") and returns the template id, the top-level field id, and the
// sub-step field id.
func seedTemplateWithSubStep(t *testing.T, userID string) (templateID, topFieldID, subStepID string) {
	t.Helper()
	ctx := t.Context()
	tmplID, err := insertTemplate(ctx, testPool, TemplateInput{
		Name: "Substep Photo Checklist",
		Sections: []SectionInput{{
			Title: "Close", Order: 0,
			Fields: []FieldInput{{
				Type: "checkbox", Label: "Cut the check", Order: 0,
				SubSteps: []FieldInput{{Type: "checkbox", Label: "Do B", Order: 0}},
			}},
		}},
	}, userID)
	if err != nil {
		t.Fatalf("insertTemplate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, tmplID)
	})
	tmpl, err := getTemplateByID(ctx, testPool, tmplID)
	if err != nil || tmpl == nil {
		t.Fatalf("getTemplateByID: %v", err)
	}
	parent := fieldByLabel(t, tmpl, "Cut the check")
	if len(parent.SubSteps) == 0 {
		t.Fatalf("expected a sub-step under 'Cut the check'")
	}
	return tmplID, parent.ID, parent.SubSteps[0].ID
}

// TestResubmit_SubStepRequirePhoto_NotBlocked proves a require_photo rejection on
// a SUB-STEP does not hard-block resubmit. A sub-step is never sent as a
// top-level response, so gating on it is unsatisfiable; the gate excludes fields
// with a parent (advisory-only for sub-steps). RED before the parent_field_id
// exclusion (the sub-step id matched the gate query → unsatisfiable 400); GREEN
// after.
func TestResubmit_SubStepRequirePhoto_NotBlocked(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "crew-substep-photo@yumyums.test")
	tmplID, topFieldID, subStepID := seedTemplateWithSubStep(t, userID)
	seedRejectedSubmission(t, tmplID, subStepID, userID, true)

	// Resubmit the TOP-LEVEL field with no photo; the sub-step's require_photo
	// must not block.
	rec := driveSubmit(t, userID, tmplID, topFieldID, `true`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 (sub-step require_photo is advisory, must not block), got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestResubmit_NeverRejectedField_Unaffected is the positive control that a
// normal submit of a field never rejected with require_photo is not gated.
func TestResubmit_NeverRejectedField_Unaffected(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "crew-neverrejected@yumyums.test")
	tmplID, fieldID := seedTemplateWithField(t, userID)
	// No prior rejection seeded.

	rec := driveSubmit(t, userID, tmplID, fieldID, `true`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for never-rejected field, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestResubmit_RejectedWithoutRequirePhoto_NotGated proves the gate keys off
// require_photo specifically: a field rejected WITHOUT require_photo is not
// forced to carry a photo on resubmit.
func TestResubmit_RejectedWithoutRequirePhoto_NotGated(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "crew-reject-nophoto@yumyums.test")
	tmplID, fieldID := seedTemplateWithField(t, userID)
	seedRejectedSubmission(t, tmplID, fieldID, userID, false)

	rec := driveSubmit(t, userID, tmplID, fieldID, `true`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for reject-without-require_photo, got %d body=%s", rec.Code, rec.Body.String())
	}
}
