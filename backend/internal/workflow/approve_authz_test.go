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

// ── B5 fold-in — authz on approve/reject (design §8 amendment 4) ────────────
//
// Before this card, ApproveSubmissionHandler and RejectItemHandler were gated by
// LOGIN ONLY: any authenticated user could approve or reject any submission,
// including one they were never assigned to review. The role rule fixed at slate
// time and NOT reopened here:
//
//	allowed ⇔ approver assignment on the submission's template
//	          ∨ admin role
//	          ∨ superadmin
//
// "Approver assignment" means exactly what PendingApprovalsHandler already means
// by it (repository.go pendingApprovals): a template_assignments row with
// assignment_role='approver' matching the caller by user id or by role.

// mkAuthzUser inserts a user with explicit roles (ensureUser hardcodes admin).
func mkAuthzUser(t *testing.T, email string, roles []string) string {
	t.Helper()
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO users (email, first_name, last_name, roles, status)
		 VALUES ($1, 'Authz', 'User', $2, 'active')
		 ON CONFLICT (email) DO UPDATE SET roles = EXCLUDED.roles, status = 'active'
		 RETURNING id::text`,
		email, roles).Scan(&id)
	if err != nil {
		t.Fatalf("insert user %s: %v", email, err)
	}
	return id
}

func assignApprover(t *testing.T, templateID, assigneeType, assigneeID string) {
	t.Helper()
	_, err := testPool.Exec(t.Context(),
		`INSERT INTO template_assignments (template_id, assignee_type, assignee_id, assignment_role)
		 VALUES ($1, $2, $3, 'approver')`,
		templateID, assigneeType, assigneeID)
	if err != nil {
		t.Fatalf("assign approver: %v", err)
	}
}

// callApprove / callReject drive the handlers with an arbitrary caller.
func callApprove(t *testing.T, caller *auth.User, submissionID string) (int, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"submission_id": submissionID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/approveSubmission", strings.NewReader(string(body)))
	req = req.WithContext(context.WithValue(req.Context(), auth.CtxKeyUser, caller))
	rec := httptest.NewRecorder()
	ApproveSubmissionHandler(testPool).ServeHTTP(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String())
}

func callReject(t *testing.T, caller *auth.User, submissionID, fieldID string) (int, string) {
	t.Helper()
	body, _ := json.Marshal(RejectItemInput{
		SubmissionID: submissionID, FieldID: fieldID, Comment: "redo this",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/rejectItem", strings.NewReader(string(body)))
	req = req.WithContext(context.WithValue(req.Context(), auth.CtxKeyUser, caller))
	rec := httptest.NewRecorder()
	RejectItemHandler(testPool).ServeHTTP(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String())
}

// submissionStatus reads the persisted status so a "403" that nonetheless
// mutated cannot pass.
func submissionStatus(t *testing.T, submissionID string) string {
	t.Helper()
	var s string
	if err := testPool.QueryRow(t.Context(),
		`SELECT status FROM checklist_submissions WHERE id = $1`, submissionID).Scan(&s); err != nil {
		t.Fatalf("read submission status: %v", err)
	}
	return s
}

// setupAuthzFixture builds a template with a field, one pending submission, and
// returns (templateID, fieldID, submissionID).
func setupAuthzFixture(t *testing.T) (string, string, string) {
	t.Helper()
	author := ensureUser(t, "authz-author@yumyums.kitchen")
	tmplID, fieldID := seedTemplateWithField(t, author)
	subID := seedPendingSubmission(t, tmplID, author)
	return tmplID, fieldID, subID
}

// ── WITHOUT the role: a logged-in stranger must be refused ─────────────────

func TestApproveSubmission_NonApproverNonAdmin_403(t *testing.T) {
	if testPool == nil {
		t.Skip("no test database")
	}
	_, _, subID := setupAuthzFixture(t)
	strangerID := mkAuthzUser(t, "authz-stranger@yumyums.kitchen", []string{"team_member"})
	stranger := &auth.User{ID: strangerID, Roles: []string{"team_member"}, Status: "active"}

	code, body := callApprove(t, stranger, subID)
	if code != http.StatusForbidden {
		t.Errorf("approve by non-approver/non-admin: status = %d, want 403 (body=%s)", code, body)
	}
	if !strings.Contains(body, `"forbidden"`) {
		t.Errorf("approve by non-approver: body = %s, want forbidden envelope", body)
	}
	if got := submissionStatus(t, subID); got != "pending" {
		t.Errorf("approve by non-approver MUTATED the submission: status = %q, want %q", got, "pending")
	}
}

func TestRejectItem_NonApproverNonAdmin_403(t *testing.T) {
	if testPool == nil {
		t.Skip("no test database")
	}
	_, fieldID, subID := setupAuthzFixture(t)
	strangerID := mkAuthzUser(t, "authz-stranger2@yumyums.kitchen", []string{"team_member"})
	stranger := &auth.User{ID: strangerID, Roles: []string{"team_member"}, Status: "active"}

	code, body := callReject(t, stranger, subID, fieldID)
	if code != http.StatusForbidden {
		t.Errorf("reject by non-approver/non-admin: status = %d, want 403 (body=%s)", code, body)
	}
	if got := submissionStatus(t, subID); got != "pending" {
		t.Errorf("reject by non-approver MUTATED the submission: status = %q, want %q", got, "pending")
	}
	var n int
	if err := testPool.QueryRow(t.Context(),
		`SELECT count(*) FROM submission_rejections WHERE submission_id = $1`, subID).Scan(&n); err != nil {
		t.Fatalf("count rejections: %v", err)
	}
	if n != 0 {
		t.Errorf("reject by non-approver wrote %d rejection rows, want 0", n)
	}
}

// ── WITH the role: each of the three allowed paths ─────────────────────────

func TestApproveSubmission_UserApproverAssignment_200(t *testing.T) {
	if testPool == nil {
		t.Skip("no test database")
	}
	tmplID, _, subID := setupAuthzFixture(t)
	approverID := mkAuthzUser(t, "authz-user-approver@yumyums.kitchen", []string{"team_member"})
	assignApprover(t, tmplID, "user", approverID)
	approver := &auth.User{ID: approverID, Roles: []string{"team_member"}, Status: "active"}

	if code, body := callApprove(t, approver, subID); code != http.StatusOK {
		t.Errorf("approve by assigned user-approver: status = %d, want 200 (body=%s)", code, body)
	}
	if got := submissionStatus(t, subID); got != "approved" {
		t.Errorf("approve by assigned approver: status = %q, want %q", got, "approved")
	}
}

func TestApproveSubmission_RoleApproverAssignment_200(t *testing.T) {
	if testPool == nil {
		t.Skip("no test database")
	}
	tmplID, _, subID := setupAuthzFixture(t)
	approverID := mkAuthzUser(t, "authz-role-approver@yumyums.kitchen", []string{"manager"})
	assignApprover(t, tmplID, "role", "manager")
	approver := &auth.User{ID: approverID, Roles: []string{"manager"}, Status: "active"}

	if code, body := callApprove(t, approver, subID); code != http.StatusOK {
		t.Errorf("approve by role-assigned approver: status = %d, want 200 (body=%s)", code, body)
	}
	if got := submissionStatus(t, subID); got != "approved" {
		t.Errorf("approve by role approver: status = %q, want %q", got, "approved")
	}
}

func TestRejectItem_AdminWithoutAssignment_200(t *testing.T) {
	if testPool == nil {
		t.Skip("no test database")
	}
	_, fieldID, subID := setupAuthzFixture(t)
	adminID := mkAuthzUser(t, "authz-admin@yumyums.kitchen", []string{"admin"})
	admin := &auth.User{ID: adminID, Roles: []string{"admin"}, Status: "active"}

	if code, body := callReject(t, admin, subID, fieldID); code != http.StatusOK {
		t.Errorf("reject by admin (no assignment): status = %d, want 200 (body=%s)", code, body)
	}
	if got := submissionStatus(t, subID); got != "rejected" {
		t.Errorf("reject by admin: status = %q, want %q", got, "rejected")
	}
}

func TestApproveSubmission_SuperadminWithoutAssignment_200(t *testing.T) {
	if testPool == nil {
		t.Skip("no test database")
	}
	_, _, subID := setupAuthzFixture(t)
	superID := mkAuthzUser(t, "authz-super@yumyums.kitchen", []string{"team_member"})
	super := &auth.User{ID: superID, Roles: []string{"team_member"}, Status: "active", IsSuperadmin: true}

	if code, body := callApprove(t, super, subID); code != http.StatusOK {
		t.Errorf("approve by superadmin (no assignment): status = %d, want 200 (body=%s)", code, body)
	}
	if got := submissionStatus(t, subID); got != "approved" {
		t.Errorf("approve by superadmin: status = %q, want %q", got, "approved")
	}
}

// An approver assignment on a DIFFERENT template must not carry over — the check
// has to resolve the submission's own template, not "is an approver somewhere".
func TestApproveSubmission_ApproverOnOtherTemplate_403(t *testing.T) {
	if testPool == nil {
		t.Skip("no test database")
	}
	_, _, subID := setupAuthzFixture(t)
	otherAuthor := ensureUser(t, "authz-other-author@yumyums.kitchen")
	// A DISTINCT template — seedTemplateWithField hardcodes one name and
	// idx_checklist_templates_name_active makes a second one a unique violation.
	otherTmplID, err := insertTemplate(t.Context(), testPool, TemplateInput{
		Name: "Authz Other Template",
		Sections: []SectionInput{{
			Title: "Close", Order: 0,
			Fields: []FieldInput{{Type: "checkbox", Label: "Other field", Order: 0}},
		}},
	}, otherAuthor)
	if err != nil {
		t.Fatalf("insertTemplate (other): %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, otherTmplID)
	})

	approverID := mkAuthzUser(t, "authz-elsewhere@yumyums.kitchen", []string{"team_member"})
	assignApprover(t, otherTmplID, "user", approverID)
	approver := &auth.User{ID: approverID, Roles: []string{"team_member"}, Status: "active"}

	code, body := callApprove(t, approver, subID)
	if code != http.StatusForbidden {
		t.Errorf("approve by approver-of-another-template: status = %d, want 403 (body=%s)", code, body)
	}
	if got := submissionStatus(t, subID); got != "pending" {
		t.Errorf("cross-template approve MUTATED the submission: status = %q", got)
	}
}

// An 'assignee' (fill-out) assignment is NOT an approver assignment — the
// submitter must not be able to self-approve.
func TestApproveSubmission_AssigneeRoleIsNotApprover_403(t *testing.T) {
	if testPool == nil {
		t.Skip("no test database")
	}
	tmplID, _, subID := setupAuthzFixture(t)
	crewID := mkAuthzUser(t, "authz-crew@yumyums.kitchen", []string{"team_member"})
	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO template_assignments (template_id, assignee_type, assignee_id, assignment_role)
		 VALUES ($1, 'user', $2, 'assignee')`, tmplID, crewID); err != nil {
		t.Fatalf("assign assignee: %v", err)
	}
	crew := &auth.User{ID: crewID, Roles: []string{"team_member"}, Status: "active"}

	if code, body := callApprove(t, crew, subID); code != http.StatusForbidden {
		t.Errorf("approve by 'assignee'-assigned crew: status = %d, want 403 (body=%s)", code, body)
	}
	if got := submissionStatus(t, subID); got != "pending" {
		t.Errorf("assignee self-approve MUTATED the submission: status = %q", got)
	}
}
