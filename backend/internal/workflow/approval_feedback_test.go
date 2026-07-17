package workflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yumyums/hq/internal/auth"
)

// seedPendingSubmission inserts a bare pending submission and returns its id.
func seedPendingSubmission(t *testing.T, templateID, userID string) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO checklist_submissions (template_id, template_snapshot, submitted_by)
		 VALUES ($1, '{}'::jsonb, $2)
		 RETURNING id::text`,
		templateID, userID).Scan(&id); err != nil {
		t.Fatalf("seed submission: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_submissions WHERE id=$1`, id)
	})
	return id
}

func seedTemplateWithField(t *testing.T, userID string) (templateID string, fieldID string) {
	t.Helper()
	ctx := t.Context()
	tmplID, err := insertTemplate(ctx, testPool, TemplateInput{
		Name: "Approval Feedback Checklist",
		Sections: []SectionInput{{
			Title: "Close", Order: 0,
			Fields: []FieldInput{{Type: "checkbox", Label: "Lock the truck", Order: 0}},
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
	f := fieldByLabel(t, tmpl, "Lock the truck")
	return tmplID, f.ID
}

func driveApprove(t *testing.T, userID string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/approveSubmission", strings.NewReader(string(body)))
	ctx := context.WithValue(req.Context(), auth.CtxKeyUser, &auth.User{ID: userID, DisplayName: "Approver", Roles: []string{"admin"}})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	ApproveSubmissionHandler(testPool).ServeHTTP(rec, req)
	return rec
}

func countRejections(t *testing.T, submissionID string) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(t.Context(),
		`SELECT count(*) FROM submission_rejections WHERE submission_id=$1`, submissionID).Scan(&n); err != nil {
		t.Fatalf("count rejections: %v", err)
	}
	return n
}

// TestApproveSubmissionHandler_FeedbackPersistFailureIsLoud proves FR-8/INV-1:
// an approval-with-feedback may report success ONLY if the feedback comment is
// durably stored. The submission_rejections INSERT is forced to fail (the
// field_id carries a syntactically invalid UUID, so the NOT NULL uuid column
// rejects the write). The comment does not persist.
//
// RED on the pre-fix handler: the failed INSERT is swallowed (slog.Error, then
// the loop continues) and the approver still receives 200 {"ok":true} — a false
// "Approved" for a comment that was never stored. GREEN after the fix: the failed
// persist surfaces as a 5xx error envelope, so no false success is reported.
func TestApproveSubmissionHandler_FeedbackPersistFailureIsLoud(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "approve-loud@yumyums.kitchen")
	tmplID, _ := seedTemplateWithField(t, userID)
	subID := seedPendingSubmission(t, tmplID, userID)

	// Feedback whose field_id is not a valid UUID → the rejection INSERT fails.
	body, _ := json.Marshal(map[string]any{
		"submission_id": subID,
		"feedback": []map[string]any{
			{"field_id": "not-a-valid-uuid", "comment": "Please redo the closing steps", "require_photo": false},
		},
	})

	rec := driveApprove(t, userID, body)

	// The comment was never durably stored.
	if got := countRejections(t, subID); got != 0 {
		t.Fatalf("precondition: expected the feedback INSERT to fail (0 rows), got %d", got)
	}

	// INV-1: success must NOT be reported when the comment did not persist.
	if rec.Code == http.StatusOK {
		t.Fatalf("LOUD FAILURE VIOLATED: handler returned 200 %q for feedback that was never stored — false 'Approved'", rec.Body.String())
	}
	if rec.Code < 500 {
		t.Fatalf("want a 5xx error envelope, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	var env map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env["error"] == "" {
		t.Fatalf("want an {\"error\":...} envelope, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestApproveSubmissionHandler_FeedbackPersistedReturnsOK is the positive control:
// when the feedback comment DOES persist, the handler still returns 200 {"ok":true}
// and the comment is queryable. Guards against the loud-failure fix breaking the
// happy path.
func TestApproveSubmissionHandler_FeedbackPersistedReturnsOK(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "approve-ok@yumyums.kitchen")
	tmplID, fieldID := seedTemplateWithField(t, userID)
	subID := seedPendingSubmission(t, tmplID, userID)

	body, _ := json.Marshal(map[string]any{
		"submission_id": subID,
		"feedback": []map[string]any{
			{"field_id": fieldID, "comment": "Nice work — one nit", "require_photo": true},
		},
	})

	rec := driveApprove(t, userID, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("happy path: want HTTP 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	if got := countRejections(t, subID); got != 1 {
		t.Fatalf("happy path: feedback comment not durably stored: want 1 row, got %d", got)
	}
	var status string
	if err := testPool.QueryRow(t.Context(),
		`SELECT status FROM checklist_submissions WHERE id=$1`, subID).Scan(&status); err != nil {
		t.Fatalf("read submission status: %v", err)
	}
	if status != "approved" {
		t.Fatalf("happy path: submission not approved: got %q", status)
	}
}
