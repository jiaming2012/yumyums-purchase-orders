package workflow

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// ── requires_approval → hasApprover, enforced for EVERY caller ──────────────
//
// POST /api/v1/workflow/ops with op_type SAVE_TEMPLATE dispatches through
// workflowOpRouter (backend/cmd/server/main.go) to CreateTemplateFunc /
// UpdateTemplateFunc — the exported aliases for the bare repository functions
// insertTemplate / updateTemplate.
//
// Both REST twins refuse a template that requires approval but names no
// approver (handler.go, `input.RequiresApproval && !hasApprover(...)` →
// 400 requires_approver). The /ops branch carried NO such check, so an
// unprivileged caller could author, through the side door, a template with
// requires_approval=true and zero approver assignments. Submissions against it
// are accepted and then sit `pending` in NOBODY's queue — pendingApprovals
// joins template_assignments with assignment_role='approver', so a template
// with none matches no reviewer. Silent, persistent, unresolvable in-app.
//
// These tests drive the REPOSITORY entry points the router calls — not the
// handlers — because that is where the class is closed. A check bolted onto the
// router would leave the next caller of insertTemplate/updateTemplate free to
// re-open the same hole; a check inside the write cannot be routed around.
//
// This is a VALIDATION gate (400), NOT an authorization gate (403). The
// separate question of whether crew should be able to mutate templates at all
// over /ops is an open product decision (run 2026-07-20c DECISIONS-NEEDED
// §1-B) and is deliberately untouched here — see the "still accepted" tests.

func approverless() TemplateInput {
	return TemplateInput{
		Name:             fmt.Sprintf("RA Forge %d", nextRASeq()),
		RequiresApproval: true,
		Sections: []SectionInput{{
			Title: "S1", Order: 0,
			Fields: []FieldInput{{Type: "checkbox", Label: "A", Order: 0}},
		}},
		Schedules: []ScheduleInput{{ActiveDays: []int{0, 1, 2, 3, 4, 5, 6}}},
		// Assignees, but no approver — exactly the payload the /ops side door accepted.
		Assignments: []AssignmentInput{
			{AssigneeType: "role", AssigneeID: "team_member", AssignmentRole: "assignee"},
		},
	}
}

var raSeq int

func nextRASeq() int { raSeq++; return raSeq }

func countTemplatesNamed(t *testing.T, name string) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM checklist_templates WHERE name = $1`, name).Scan(&n); err != nil {
		t.Fatalf("count templates %q: %v", name, err)
	}
	return n
}

func approverCount(t *testing.T, templateID string) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM template_assignments WHERE template_id = $1 AND assignment_role = 'approver'`,
		templateID).Scan(&n); err != nil {
		t.Fatalf("count approvers: %v", err)
	}
	return n
}

// TestCreateTemplateFuncRejectsApproverlessApproval drives the EXACT alias the
// /ops SAVE_TEMPLATE create branch calls.
func TestCreateTemplateFuncRejectsApproverlessApproval(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable")
	}
	ctx := context.Background()
	userID := ensureUser(t, "ra-create@yumyums.kitchen")

	input := approverless()
	id, err := CreateTemplateFunc(ctx, testPool, input, userID)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE name=$1`, input.Name)
	})

	if !errors.Is(err, ErrRequiresApprover) {
		// Errorf, not Fatalf: the not-mutated assertion below is the real proof
		// and must be reported in the same run, red or green.
		t.Errorf("CreateTemplateFunc(requires_approval=true, no approver) = (%q, %v); want ErrRequiresApprover", id, err)
	}
	// The refusal is the claim; the absent row is the proof.
	if n := countTemplatesNamed(t, input.Name); n != 0 {
		t.Errorf("refused, but %d template row(s) named %q were written anyway — the mutation must not occur", n, input.Name)
	}
}

// TestUpdateTemplateFuncRejectsApproverlessApproval drives the alias the /ops
// SAVE_TEMPLATE update branch calls, and proves the pre-existing template is
// left untouched — no renamed header, no deleted approver assignment. The
// update path deletes assignments before re-inserting them, so a half-applied
// write here would ALSO strip the approver and produce the same orphan queue.
func TestUpdateTemplateFuncRejectsApproverlessApproval(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable")
	}
	ctx := context.Background()
	userID := ensureUser(t, "ra-update@yumyums.kitchen")

	good := approverless()
	good.Name = fmt.Sprintf("RA Good %d", nextRASeq())
	good.Assignments = append(good.Assignments,
		AssignmentInput{AssigneeType: "role", AssigneeID: "admin", AssignmentRole: "approver"})

	tmplID, err := CreateTemplateFunc(ctx, testPool, good, userID)
	if err != nil {
		t.Fatalf("seed template with an approver must be accepted: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, tmplID)
	})
	if approverCount(t, tmplID) != 1 {
		t.Fatalf("seed template should have 1 approver assignment")
	}

	// Now rewrite it through the side door, stripping the approver while keeping
	// requires_approval=true — and renaming, so a partial write is visible.
	bad := good
	bad.Name = fmt.Sprintf("RA Rewritten %d", nextRASeq())
	bad.Assignments = []AssignmentInput{
		{AssigneeType: "role", AssigneeID: "team_member", AssignmentRole: "assignee"},
	}

	err = UpdateTemplateFunc(ctx, testPool, tmplID, bad)
	if !errors.Is(err, ErrRequiresApprover) {
		t.Errorf("UpdateTemplateFunc(requires_approval=true, no approver) = %v; want ErrRequiresApprover", err)
	}

	var gotName string
	if err := testPool.QueryRow(ctx, `SELECT name FROM checklist_templates WHERE id=$1`, tmplID).Scan(&gotName); err != nil {
		t.Fatalf("reload template: %v", err)
	}
	if gotName != good.Name {
		t.Errorf("refused, but the template header was mutated: name=%q want %q", gotName, good.Name)
	}
	if n := approverCount(t, tmplID); n != 1 {
		t.Errorf("refused, but the approver assignment was destroyed: %d approver rows, want 1", n)
	}
}

// TestTemplateWritesStillAcceptedWhenValid is the anti-overshoot leg at the
// repository layer: the gate must be narrow. requires_approval=false with no
// approver, and requires_approval=true WITH an approver, both still write.
func TestTemplateWritesStillAcceptedWhenValid(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable")
	}
	ctx := context.Background()
	userID := ensureUser(t, "ra-valid@yumyums.kitchen")

	noApproval := approverless()
	noApproval.RequiresApproval = false
	noApproval.Name = fmt.Sprintf("RA NoApproval %d", nextRASeq())
	id1, err := CreateTemplateFunc(ctx, testPool, noApproval, userID)
	if err != nil {
		t.Fatalf("requires_approval=false with no approver must still be accepted: %v", err)
	}
	t.Cleanup(func() { _, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, id1) })

	withApprover := approverless()
	withApprover.Name = fmt.Sprintf("RA WithApprover %d", nextRASeq())
	withApprover.Assignments = append(withApprover.Assignments,
		AssignmentInput{AssigneeType: "role", AssigneeID: "admin", AssignmentRole: "approver"})
	id2, err := CreateTemplateFunc(ctx, testPool, withApprover, userID)
	if err != nil {
		t.Fatalf("requires_approval=true WITH an approver must still be accepted: %v", err)
	}
	t.Cleanup(func() { _, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, id2) })

	// And the update path stays open for a valid rewrite.
	withApprover.Name = fmt.Sprintf("RA WithApprover Renamed %d", nextRASeq())
	if err := UpdateTemplateFunc(ctx, testPool, id2, withApprover); err != nil {
		t.Fatalf("valid update must still be accepted: %v", err)
	}
}
