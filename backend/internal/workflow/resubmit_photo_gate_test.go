package workflow

import (
	"context"
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
