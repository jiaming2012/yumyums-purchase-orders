package workflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yumyums/hq/internal/auth"
)

// TestSaveResponse_SingleRowWrite_CDCTrigger proves B-157 (E-KR2): one
// /saveResponse (one user save) produces exactly ONE row write to
// submission_responses, so a row-level CDC trigger fires ONCE — not twice.
//
// A per-row AFTER INSERT OR UPDATE trigger on submission_responses records each
// fire into a counting table. The test drives the full production
// SaveResponseHandler path (the save write + the async field_response op
// emission) for ONE field, then asserts the trigger fired exactly once.
//
// RED on the pre-change tree (count == 2): saveResponse's INSERT is write #1,
// and EmitOp's `UPDATE submission_responses SET lamport_ts` (the async LWW
// stamp) is write #2 — a row trigger cannot tell them apart. GREEN after the
// change (count == 1): saveResponse folds the lamport_ts stamp into the same
// INSERT/upsert, and the handler emits the op row WITHOUT re-updating the
// response row.
func TestSaveResponse_SingleRowWrite_CDCTrigger(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	ctx := t.Context()
	userID := ensureUser(t, "cdc-single-fire@yumyums.kitchen")

	// A template with one field to save against.
	tmplID, err := insertTemplate(ctx, testPool, TemplateInput{
		Name: "CDC single-fire checklist",
		Sections: []SectionInput{{
			Title: "S", Order: 0,
			Fields: []FieldInput{{Type: "checkbox", Label: "Prep", Order: 0}},
		}},
		Schedules: []ScheduleInput{{ActiveDays: []int{0, 1, 2, 3, 4, 5, 6}}},
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
	prep := fieldByLabel(t, tmpl, "Prep")

	// Install a counting CDC trigger on submission_responses scoped to THIS field
	// (so it counts only this test's writes, not concurrent noise from -p 1 peers).
	// A per-statement teardown drops everything on cleanup.
	setup := `
		CREATE TABLE IF NOT EXISTS cdc_fire_log (
		  field_id uuid NOT NULL,
		  op       text NOT NULL,
		  fired_at timestamptz NOT NULL DEFAULT clock_timestamp()
		);
		CREATE OR REPLACE FUNCTION cdc_fire_log_fn() RETURNS trigger AS $$
		BEGIN
		  INSERT INTO cdc_fire_log(field_id, op) VALUES (NEW.field_id, TG_OP);
		  RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
		DROP TRIGGER IF EXISTS cdc_fire_log_trg ON submission_responses;
		CREATE TRIGGER cdc_fire_log_trg
		  AFTER INSERT OR UPDATE ON submission_responses
		  FOR EACH ROW EXECUTE FUNCTION cdc_fire_log_fn();
	`
	if _, err := testPool.Exec(ctx, setup); err != nil {
		t.Fatalf("install cdc trigger: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = testPool.Exec(bg, `DROP TRIGGER IF EXISTS cdc_fire_log_trg ON submission_responses`)
		_, _ = testPool.Exec(bg, `DROP FUNCTION IF EXISTS cdc_fire_log_fn()`)
		_, _ = testPool.Exec(bg, `DROP TABLE IF EXISTS cdc_fire_log`)
	})

	// Clean slate for this field's fire log.
	if _, err := testPool.Exec(ctx, `DELETE FROM cdc_fire_log WHERE field_id=$1`, prep.ID); err != nil {
		t.Fatalf("clear fire log: %v", err)
	}

	// Drive the FULL production path: POST /saveResponse. This runs saveResponse
	// (the save write) AND the async field_response op emission (the LWW stamp).
	body, _ := json.Marshal(SaveResponseInput{FieldID: prep.ID, Value: json.RawMessage(`true`)})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/saveResponse", strings.NewReader(string(body)))
	reqCtx := context.WithValue(req.Context(), auth.CtxKeyUser, &auth.User{ID: userID, DisplayName: "Test User", Roles: []string{"admin"}})
	req = req.WithContext(reqCtx)
	rec := httptest.NewRecorder()
	SaveResponseHandler(testPool).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("want HTTP 204, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	// The op emission is a fire-and-forget goroutine. Wait until the op row for
	// this field has landed — that is the moment the (old) second write would have
	// happened too. Poll rather than sleep a fixed interval.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var opCount int
		if err := testPool.QueryRow(ctx,
			`SELECT count(*) FROM ops WHERE entity_id=$1 AND entity_type='field_response' AND op_type='SET_FIELD'`,
			prep.ID).Scan(&opCount); err != nil {
			t.Fatalf("poll op row: %v", err)
		}
		if opCount >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("field_response op row never landed within 5s (emission path did not run)")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Small settle so any trailing entity-table write in the emission path (the
	// bug) has definitely executed before we read the count.
	time.Sleep(100 * time.Millisecond)

	var fires int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM cdc_fire_log WHERE field_id=$1`, prep.ID).Scan(&fires); err != nil {
		t.Fatalf("read fire count: %v", err)
	}

	// Sanity: the response row exists with a winning (non-zero) lamport_ts, and
	// the op row's lamport_ts matches it — LWW ordering preserved.
	var rowTS, opTS int64
	if err := testPool.QueryRow(ctx,
		`SELECT lamport_ts FROM submission_responses WHERE field_id=$1 AND submission_id IS NULL`,
		prep.ID).Scan(&rowTS); err != nil {
		t.Fatalf("read response lamport_ts: %v", err)
	}
	if err := testPool.QueryRow(ctx,
		`SELECT lamport_ts FROM ops WHERE entity_id=$1 AND entity_type='field_response' ORDER BY server_ts DESC LIMIT 1`,
		prep.ID).Scan(&opTS); err != nil {
		t.Fatalf("read op lamport_ts: %v", err)
	}
	if rowTS < 1 {
		t.Fatalf("response lamport_ts not stamped: want >=1, got %d", rowTS)
	}
	if opTS != rowTS {
		t.Fatalf("op/response lamport_ts diverge (LWW ordering broken): op=%d response=%d", opTS, rowTS)
	}

	if fires != 1 {
		t.Fatalf("B-157: one /saveResponse fired the row CDC trigger %d times, want exactly 1 "+
			"(pre-change tree fires twice: INSERT save + UPDATE lamport stamp)", fires)
	}
}
