package workflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/yumyums/hq/internal/auth"
)

// TestUpdateTemplateHandler_OpEmittedTransactionally proves FR-5/INV-1: the
// SAVE_TEMPLATE op emitted for a Builder save commits in the SAME transaction as
// the template write, so there is no window in which an accepted write exists
// without its op durably queued for other devices.
//
// RED on the pre-change build: emission is a fire-and-forget `go func(){…}()`
// (opsync.EmitOp) that runs AFTER the handler has already returned 200. The
// goroutine's first act is a DB round trip, so the moment ServeHTTP returns the
// ops table still has zero SAVE_TEMPLATE rows for this fresh template — the write
// is accepted but its op is not yet durably queued. GREEN after the change:
// updateTemplateAndEmit inserts the op inside the write's own transaction, so it
// is queryable the instant the handler returns (timing-independent).
func TestUpdateTemplateHandler_OpEmittedTransactionally(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	ctx := t.Context()
	userID := ensureUser(t, "txnemit@yumyums.kitchen")

	tmplID, err := insertTemplate(ctx, testPool, TemplateInput{
		Name:             "Txn Emit Checklist",
		RequiresApproval: false,
		Sections: []SectionInput{{
			Title: "Open", Order: 0,
			Fields: []FieldInput{{Type: "checkbox", Label: "Prep", Order: 0}},
		}},
		Schedules: []ScheduleInput{{ActiveDays: []int{5}}},
	}, userID)
	if err != nil {
		t.Fatalf("insertTemplate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, tmplID)
	})

	before, err := getTemplateByID(ctx, testPool, tmplID)
	if err != nil || before == nil {
		t.Fatalf("getTemplateByID (before): %v", err)
	}
	prep := fieldByLabel(t, before, "Prep")

	// Baseline: no SAVE_TEMPLATE op exists for this template yet.
	var baseOps int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM ops WHERE entity_id=$1 AND op_type='SAVE_TEMPLATE'`, tmplID).Scan(&baseOps); err != nil {
		t.Fatalf("count baseline ops: %v", err)
	}

	// Drive the REST Builder-save path (PUT /updateTemplate/{id}).
	body, _ := json.Marshal(TemplateInput{
		Name:             "Txn Emit Checklist",
		RequiresApproval: false,
		Sections: []SectionInput{{
			Title: "Open", Order: 0,
			Fields: []FieldInput{
				{ID: prep.ID, Type: "checkbox", Label: "Prep", Order: 0},
				{Type: "checkbox", Label: "Restock", Order: 1},
			},
		}},
		Schedules: []ScheduleInput{{ActiveDays: []int{5}}},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workflow/updateTemplate/"+tmplID, strings.NewReader(string(body)))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", tmplID)
	reqCtx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	reqCtx = context.WithValue(reqCtx, auth.CtxKeyUser, &auth.User{ID: userID, DisplayName: "Test User", Roles: []string{"admin"}})
	req = req.WithContext(reqCtx)
	rec := httptest.NewRecorder()

	UpdateTemplateHandler(testPool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want HTTP 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	// The instant the handler returns, the SAVE_TEMPLATE op MUST already be
	// durably queued — no sleep, no retry. If emission were fire-and-forget the
	// goroutine would still be mid-round-trip here and this count would be 0.
	var afterOps int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM ops WHERE entity_id=$1 AND op_type='SAVE_TEMPLATE'`, tmplID).Scan(&afterOps); err != nil {
		t.Fatalf("count post-write ops: %v", err)
	}
	if afterOps != baseOps+1 {
		t.Fatalf("SAVE_TEMPLATE op not durably queued in the write txn: want %d, got %d "+
			"(accepted write with no queued op)", baseOps+1, afterOps)
	}

	// And it commits atomically with the write: the op's lamport_ts matches the
	// template row's lamport_ts bumped in the same transaction.
	var opTS, tmplTS int64
	if err := testPool.QueryRow(ctx,
		`SELECT lamport_ts FROM ops WHERE entity_id=$1 AND op_type='SAVE_TEMPLATE' ORDER BY server_ts DESC LIMIT 1`,
		tmplID).Scan(&opTS); err != nil {
		t.Fatalf("read op lamport_ts: %v", err)
	}
	if err := testPool.QueryRow(ctx,
		`SELECT lamport_ts FROM checklist_templates WHERE id=$1`, tmplID).Scan(&tmplTS); err != nil {
		t.Fatalf("read template lamport_ts: %v", err)
	}
	if opTS != tmplTS {
		t.Fatalf("op/template lamport_ts diverge (not same-txn bump): op=%d template=%d", opTS, tmplTS)
	}
}
