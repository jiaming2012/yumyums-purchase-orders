package workflow

import (
	"context"
	"fmt"
	"testing"
)

// ── checklist_submissions.status is never set for a no-approval template ────
//
// `checklist_submissions.status` is TEXT NOT NULL DEFAULT 'pending' with
// CHECK (status IN ('pending','approved','rejected','completed'))
// (migrations/0011_checklist_submissions.sql). submitChecklist's INSERT never
// names the column, so every row is born 'pending'. Only approveItem and
// rejectItem move it — and neither runs for a template with
// requires_approval=false. Those rows read 'pending' server-side forever.
//
// Two failures follow, and this file pins both:
//
//  1. INVARIANT — a submission against a requires_approval:false template is
//     terminal the moment it is written. Nothing will ever review it. Leaving it
//     'pending' is a lie the DB tells every reader.
//
//  2. LEAK — pendingApprovals (repository.go) filters on s.status = 'pending'
//     and does NOT filter on requires_approval at all. It is saved today only by
//     its ta.assignment_role = 'approver' join. But nothing forbids an approver
//     assignment on a requires_approval:false template: the create/update gate is
//     deliberately narrow (see requires_approver_test.go
//     TestTemplateWritesStillAcceptedWhenValid — that combination "must still be
//     accepted"). Where such a template exists, its submissions land in a real
//     user's approvals queue and can never leave it.
//
// The fix: normalize status at insert to 'completed' (already permitted by the
// existing CHECK, currently unused — no migration, no new lifecycle value), and
// gate pendingApprovals on the submission's OWN snapshot
// ((s.template_snapshot->>'requires_approval')::boolean IS NOT FALSE) rather
// than the live template flag, consistent with the repo's frozen-at-submit
// semantics. IS NOT FALSE, not IS TRUE: a snapshot missing the key (older rows,
// and the '{}' snapshots seeded by approval_feedback_test.go) must stay visible.

var subStatusSeq int

func nextSubStatusSeq() int { subStatusSeq++; return subStatusSeq }

// UUIDs come from resubmit_photo_gate_test.go's newUUID (Postgres
// gen_random_uuid) — the repo carries no uuid dependency and this card adds none.

// seedNoApprovalTemplate builds a requires_approval:false template. When
// approverUserID is non-empty it ALSO attaches an approver assignment naming
// that user — the combination the create gate deliberately still accepts.
// Assignment is by USER, not by role: roles are constrained to
// admin/manager/team_member (0023_multi_role.sql), which every other fixture in
// this package also uses, so a role-scoped queue would carry cross-test bleed.
func seedNoApprovalTemplate(t *testing.T, userID, approverUserID string) string {
	t.Helper()
	in := TemplateInput{
		Name:             fmt.Sprintf("SS NoApproval %d", nextSubStatusSeq()),
		RequiresApproval: false,
		Sections: []SectionInput{{
			Title: "Close", Order: 0,
			Fields: []FieldInput{{Type: "checkbox", Label: "Lock the truck", Order: 0}},
		}},
		Schedules: []ScheduleInput{{ActiveDays: []int{0, 1, 2, 3, 4, 5, 6}}},
		Assignments: []AssignmentInput{
			{AssigneeType: "role", AssigneeID: "team_member", AssignmentRole: "assignee"},
		},
	}
	if approverUserID != "" {
		in.Assignments = append(in.Assignments,
			AssignmentInput{AssigneeType: "user", AssigneeID: approverUserID, AssignmentRole: "approver"})
	}
	tmplID, err := insertTemplate(t.Context(), testPool, in, userID)
	if err != nil {
		t.Fatalf("insertTemplate(requires_approval=false): %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, tmplID)
	})
	return tmplID
}

func submitAgainst(t *testing.T, templateID, userID string) string {
	t.Helper()
	subID, err := submitChecklist(t.Context(), testPool, SubmitChecklistInput{
		TemplateID:     templateID,
		IdempotencyKey: newUUID(t),
	}, userID)
	if err != nil {
		t.Fatalf("submitChecklist: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_submissions WHERE id=$1`, subID)
	})
	return subID
}

// TestSubmitNoApprovalTemplate_StatusNotLeftPending is the INVARIANT leg.
//
// A submission against a template that requires no approval is terminal at
// write time. It must not be persisted in the reviewable 'pending' state that
// nothing will ever move it out of.
func TestSubmitNoApprovalTemplate_StatusNotLeftPending(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	author := ensureUser(t, "substatus-author@yumyums.kitchen")
	tmplID := seedNoApprovalTemplate(t, author, "")
	subID := submitAgainst(t, tmplID, author)

	got := submissionStatus(t, subID)
	if got == "pending" {
		t.Errorf("submission against a requires_approval=false template persisted status=%q; "+
			"nothing will ever move it — approveItem/rejectItem do not run for this template. "+
			"want a terminal status (%q)", got, "completed")
	}
	if got != "completed" {
		t.Errorf("submission status = %q, want %q (the value already permitted by the "+
			"0011 CHECK constraint and currently unused — no migration needed)", got, "completed")
	}
}

// TestSubmitApprovalTemplate_StillPending is the anti-overshoot leg: a template
// that DOES require approval must still write 'pending', or the whole approvals
// queue empties.
func TestSubmitApprovalTemplate_StillPending(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	author := ensureUser(t, "substatus-approval-author@yumyums.kitchen")
	tmplID, err := insertTemplate(t.Context(), testPool, TemplateInput{
		Name:             fmt.Sprintf("SS Approval %d", nextSubStatusSeq()),
		RequiresApproval: true,
		Sections: []SectionInput{{
			Title: "Close", Order: 0,
			Fields: []FieldInput{{Type: "checkbox", Label: "Lock the truck", Order: 0}},
		}},
		Schedules: []ScheduleInput{{ActiveDays: []int{0, 1, 2, 3, 4, 5, 6}}},
		Assignments: []AssignmentInput{
			{AssigneeType: "role", AssigneeID: "admin", AssignmentRole: "approver"},
		},
	}, author)
	if err != nil {
		t.Fatalf("insertTemplate(requires_approval=true): %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, tmplID)
	})

	subID := submitAgainst(t, tmplID, author)
	if got := submissionStatus(t, subID); got != "pending" {
		t.Errorf("submission against a requires_approval=true template = %q, want %q — "+
			"the approvals queue depends on this", got, "pending")
	}
}

// TestPendingApprovals_NoApprovalSubmissionDoesNotLeak is the LEAK leg.
//
// A requires_approval:false template WITH an approver assignment is an accepted
// combination (requires_approver_test.go pins that on purpose). Its submissions
// must never surface in that approver's queue: nobody is meant to review them,
// and no in-app action can clear them.
func TestPendingApprovals_NoApprovalSubmissionDoesNotLeak(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	author := ensureUser(t, "substatus-leak-author@yumyums.kitchen")
	// A dedicated approver, assigned by user id, so this queue contains nothing
	// but what this test puts there — no cross-test bleed from role fixtures.
	approver := mkAuthzUser(t, "substatus-leak-approver@yumyums.kitchen", []string{"manager"})

	tmplID := seedNoApprovalTemplate(t, author, approver)
	subID := submitAgainst(t, tmplID, author)

	queue, err := pendingApprovals(t.Context(), testPool, approver)
	if err != nil {
		t.Fatalf("pendingApprovals: %v", err)
	}
	for _, s := range queue {
		if s.ID == subID {
			t.Fatalf("submission %s against a requires_approval=false template LEAKED into the "+
				"approver's pending queue (status=%q, queue size %d). Nothing can ever clear it: "+
				"approving it would forge an approval nobody asked for.", subID, s.Status, len(queue))
		}
	}
}

// TestPendingApprovals_RealPendingStillVisible is the anti-overshoot twin of the
// leak leg: a genuine requires_approval=true submission must still reach its
// approver, and a legacy row whose snapshot carries no requires_approval key at
// all must stay visible too (hence IS NOT FALSE, not IS TRUE).
func TestPendingApprovals_RealPendingStillVisible(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	author := ensureUser(t, "substatus-real-author@yumyums.kitchen")
	approver := mkAuthzUser(t, "substatus-real-approver@yumyums.kitchen", []string{"manager"})

	tmplID, err := insertTemplate(t.Context(), testPool, TemplateInput{
		Name:             fmt.Sprintf("SS RealPending %d", nextSubStatusSeq()),
		RequiresApproval: true,
		Sections: []SectionInput{{
			Title: "Close", Order: 0,
			Fields: []FieldInput{{Type: "checkbox", Label: "Lock the truck", Order: 0}},
		}},
		Schedules: []ScheduleInput{{ActiveDays: []int{0, 1, 2, 3, 4, 5, 6}}},
		Assignments: []AssignmentInput{
			{AssigneeType: "user", AssigneeID: approver, AssignmentRole: "approver"},
		},
	}, author)
	if err != nil {
		t.Fatalf("insertTemplate(requires_approval=true): %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, tmplID)
	})

	realSub := submitAgainst(t, tmplID, author)
	// A legacy-shaped row: snapshot with no requires_approval key at all. This is
	// exactly what approval_feedback_test.go's seedPendingSubmission writes and
	// what every pre-fix production row looks like.
	legacySub := seedPendingSubmission(t, tmplID, author)

	queue, err := pendingApprovals(t.Context(), testPool, approver)
	if err != nil {
		t.Fatalf("pendingApprovals: %v", err)
	}
	seen := map[string]bool{}
	for _, s := range queue {
		seen[s.ID] = true
	}
	if !seen[realSub] {
		t.Errorf("a genuine requires_approval=true submission vanished from its approver's queue")
	}
	if !seen[legacySub] {
		t.Errorf("a legacy submission whose snapshot carries no requires_approval key vanished " +
			"from its approver's queue — the gate must be IS NOT FALSE, not IS TRUE")
	}
}

// TestTemplateSnapshotCarriesRequiresApproval proves the premise the
// pendingApprovals gate rests on: the snapshot written at submit time actually
// contains the requires_approval key, for both values. If this fails, the gate
// must fall back to the live t.requires_approval column.
func TestTemplateSnapshotCarriesRequiresApproval(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	author := ensureUser(t, "substatus-snapshot-author@yumyums.kitchen")
	tmplID := seedNoApprovalTemplate(t, author, "")
	subID := submitAgainst(t, tmplID, author)

	var raw *string
	var keys []string
	if err := testPool.QueryRow(t.Context(),
		`SELECT template_snapshot->>'requires_approval',
		        ARRAY(SELECT jsonb_object_keys(template_snapshot) ORDER BY 1)
		 FROM checklist_submissions WHERE id=$1`,
		subID).Scan(&raw, &keys); err != nil {
		t.Fatalf("read snapshot key: %v", err)
	}
	shown := "<nil>"
	if raw != nil {
		shown = *raw
	}
	t.Logf("real row %s: template_snapshot->>'requires_approval' = %s; snapshot keys = %v",
		subID, shown, keys)
	if raw == nil {
		t.Fatalf("template_snapshot carries NO requires_approval key — the pendingApprovals " +
			"gate cannot rest on the snapshot; fall back to t.requires_approval")
	}
	if *raw != "false" {
		t.Errorf("template_snapshot->>'requires_approval' = %q, want %q", *raw, "false")
	}
}
