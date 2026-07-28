package workflow

import (
	"context"
	"fmt"
	"testing"
)

// ── submission_fail_notes had no unique constraint and a bare INSERT ────────
//
// `submitChecklist` upserts on two of the three tables it writes:
//
//	checklist_submissions  ON CONFLICT (idempotency_key) DO UPDATE ... RETURNING id
//	submission_responses   ON CONFLICT (submission_id, field_id) DO UPDATE
//	submission_fail_notes  INSERT ... VALUES (...)          <-- bare
//
// That was harmless while one checklist could only ever produce one POST. It
// stopped being harmless when `workflow-offline-double-submit` (ledger T-23
// decision 60) made the client REUSE the queued idempotency_key on a second
// press: two POSTs now deliberately land on the SAME submission row, and the
// comment in workflows.html that justifies the design says "the server upserts
// only the fields present in each payload". It did — for responses. For fail
// notes it appended, so the approver read the same note twice.
//
// Measured at morning triage 2026-07-27, not reasoned: one payload POSTed twice
// under one idempotency_key returned 201/201 with an IDENTICAL submission id and
// left `submission_rows=1 response_rows=1 fail_note_rows=2`.
//
// The fix is the matching pair — a unique index on (submission_id, field_id)
// (migration 0071) AND `ON CONFLICT ... DO UPDATE` on the insert. Neither half
// works alone: the index without the upsert turns a silent duplicate into a hard
// 500 on the second POST, which is worse than the bug. This file pins both legs.

var failNoteSeq int

func nextFailNoteSeq() int { failNoteSeq++; return failNoteSeq }

// seedFailNoteTemplate builds a one-checkbox template with a name unique to this
// run — checklist_templates carries a unique index on the active name, so the
// fixed-name helpers in the other test files cannot be reused here.
func seedFailNoteTemplate(t *testing.T, userID string) (templateID, fieldID string) {
	t.Helper()
	ctx := t.Context()
	tmplID, err := insertTemplate(ctx, testPool, TemplateInput{
		Name: fmt.Sprintf("FN Upsert %d", nextFailNoteSeq()),
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
	return tmplID, fieldByLabel(t, tmpl, "Lock the truck").ID
}

func countFailNotes(t *testing.T, submissionID string) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(t.Context(),
		`SELECT count(*) FROM submission_fail_notes WHERE submission_id=$1`, submissionID).Scan(&n); err != nil {
		t.Fatalf("count fail notes: %v", err)
	}
	return n
}

func readFailNote(t *testing.T, submissionID, fieldID string) (note string, severity *string) {
	t.Helper()
	if err := testPool.QueryRow(t.Context(),
		`SELECT note, severity FROM submission_fail_notes WHERE submission_id=$1 AND field_id=$2`,
		submissionID, fieldID).Scan(&note, &severity); err != nil {
		t.Fatalf("read fail note: %v", err)
	}
	return note, severity
}

// TestSubmitChecklist_RepeatedKey_FailNoteNotDuplicated is the RED-first leg:
// the exact triage reproduction. Same payload, same idempotency_key, twice.
func TestSubmitChecklist_RepeatedKey_FailNoteNotDuplicated(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "failnote-dupe@yumyums.test")
	tmplID, fieldID := seedFailNoteTemplate(t, userID)
	sev := "major"
	key := newUUID(t)
	in := SubmitChecklistInput{
		TemplateID:     tmplID,
		IdempotencyKey: key,
		FailNotes:      []FailNoteInput{{FieldID: fieldID, Note: "Walk-in reading 45F", Severity: &sev}},
	}

	firstID, err := submitChecklist(t.Context(), testPool, in, userID)
	if err != nil {
		t.Fatalf("submitChecklist (first): %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_submissions WHERE id=$1`, firstID)
	})

	secondID, err := submitChecklist(t.Context(), testPool, in, userID)
	if err != nil {
		t.Fatalf("submitChecklist (second, same idempotency_key): %v — the second POST must "+
			"SUCCEED, not 500. A unique index without a matching ON CONFLICT would fail here", err)
	}
	if secondID != firstID {
		t.Fatalf("idempotency_key %q produced two submissions (%s, %s); the "+
			"ON CONFLICT (idempotency_key) upsert is the premise of this test", key, firstID, secondID)
	}

	if got := countFailNotes(t, firstID); got != 1 {
		t.Errorf("submission %s holds %d fail-note rows for one field, want 1 — "+
			"the approver sees the same note %d times. submission_fail_notes needs the same "+
			"ON CONFLICT (submission_id, field_id) DO UPDATE the responses insert directly "+
			"above it already carries", firstID, got, got)
	}
}

// TestSubmitChecklist_RepeatedKey_FailNoteEditWins is the anti-overshoot leg: an
// upsert that silently DO NOTHINGs would also make the count 1, while quietly
// dropping the crew member's edit. The second payload's note and severity must
// win, exactly as submission_responses' `SET value = EXCLUDED.value` does.
func TestSubmitChecklist_RepeatedKey_FailNoteEditWins(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "failnote-edit@yumyums.test")
	tmplID, fieldID := seedFailNoteTemplate(t, userID)
	minor, critical := "minor", "critical"
	key := newUUID(t)

	subID, err := submitChecklist(t.Context(), testPool, SubmitChecklistInput{
		TemplateID:     tmplID,
		IdempotencyKey: key,
		FailNotes:      []FailNoteInput{{FieldID: fieldID, Note: "Looked fine", Severity: &minor}},
	}, userID)
	if err != nil {
		t.Fatalf("submitChecklist (first): %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_submissions WHERE id=$1`, subID)
	})

	if _, err := submitChecklist(t.Context(), testPool, SubmitChecklistInput{
		TemplateID:     tmplID,
		IdempotencyKey: key,
		FailNotes:      []FailNoteInput{{FieldID: fieldID, Note: "Actually the seal is torn", Severity: &critical}},
	}, userID); err != nil {
		t.Fatalf("submitChecklist (second): %v", err)
	}

	if got := countFailNotes(t, subID); got != 1 {
		t.Fatalf("want exactly 1 fail-note row after the corrected re-submit, got %d", got)
	}
	note, sev := readFailNote(t, subID, fieldID)
	if note != "Actually the seal is torn" {
		t.Errorf("fail note = %q, want the LATER payload's text — an ON CONFLICT DO NOTHING "+
			"would deduplicate the row but silently discard the correction", note)
	}
	if sev == nil || *sev != "critical" {
		t.Errorf("fail severity = %v, want %q (the later payload's)", sev, "critical")
	}
}

// TestSubmitChecklist_RepeatedKey_FailNotePhotoSurvives pins the deliberate
// OMISSION in the upsert's SET list. photo_url is not in the INSERT column list,
// so `SET photo_url = EXCLUDED.photo_url` would write NULL over an attached
// photo on every re-POST. The correction photo travels on the RESPONSE value
// (workflows.html bundles it as `_correction_photo`), not on the fail note, so
// nothing repopulates it.
func TestSubmitChecklist_RepeatedKey_FailNotePhotoSurvives(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "failnote-photo@yumyums.test")
	tmplID, fieldID := seedFailNoteTemplate(t, userID)
	sev := "major"
	key := newUUID(t)
	in := SubmitChecklistInput{
		TemplateID:     tmplID,
		IdempotencyKey: key,
		FailNotes:      []FailNoteInput{{FieldID: fieldID, Note: "Seal torn", Severity: &sev}},
	}

	subID, err := submitChecklist(t.Context(), testPool, in, userID)
	if err != nil {
		t.Fatalf("submitChecklist (first): %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_submissions WHERE id=$1`, subID)
	})
	const photo = "https://cdn.example/receipts/seal.jpg"
	if _, err := testPool.Exec(t.Context(),
		`UPDATE submission_fail_notes SET photo_url=$1 WHERE submission_id=$2 AND field_id=$3`,
		photo, subID, fieldID); err != nil {
		t.Fatalf("attach photo: %v", err)
	}

	if _, err := submitChecklist(t.Context(), testPool, in, userID); err != nil {
		t.Fatalf("submitChecklist (second): %v", err)
	}

	var got *string
	if err := testPool.QueryRow(t.Context(),
		`SELECT photo_url FROM submission_fail_notes WHERE submission_id=$1 AND field_id=$2`,
		subID, fieldID).Scan(&got); err != nil {
		t.Fatalf("read photo_url: %v", err)
	}
	if got == nil || *got != photo {
		t.Errorf("photo_url = %v after a re-POST, want %q preserved — the upsert must NOT "+
			"set photo_url, which is absent from the INSERT column list and therefore NULL "+
			"in EXCLUDED", got, photo)
	}
}
