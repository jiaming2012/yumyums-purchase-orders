package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
	"github.com/yumyums/hq/internal/db"
	"github.com/yumyums/hq/internal/testdb"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv(testdb.EnvVar)
	// Computed BEFORE the fallback: the fallback is the *unset* case, and the
	// unset case still skips. See internal/testdb for the asymmetry.
	requested := dbURL != ""
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		testdb.ExitIfRequested(requested, dbURL, "connect", err)
		// DB_TEST_URL unset and the local fallback is not there — leave
		// testPool nil so the DB-backed tests skip.
		os.Exit(m.Run())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		testdb.ExitIfRequested(requested, dbURL, "ping", err)
		os.Exit(m.Run())
	}
	if err := db.Migrate(pool); err != nil {
		pool.Close()
		panic("db.Migrate failed: " + err.Error())
	}
	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// ensureUser inserts (or reuses) an admin user and returns its id. users is not
// truncated between tests, so ON CONFLICT keeps it idempotent.
func ensureUser(t *testing.T, email string) string {
	t.Helper()
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO users (email, first_name, last_name, roles, status)
		 VALUES ($1, 'Test', 'User', ARRAY['admin']::text[], 'active')
		 ON CONFLICT (email) DO UPDATE SET status = EXCLUDED.status
		 RETURNING id::text`,
		email).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

// fieldByLabel returns the top-level field with the given label from a hydrated
// template, or fails the test.
func fieldByLabel(t *testing.T, tmpl *Template, label string) Field {
	t.Helper()
	for _, sec := range tmpl.Sections {
		for _, f := range sec.Fields {
			if f.Label == label {
				return f
			}
		}
	}
	t.Fatalf("field %q not found in template", label)
	return Field{}
}

func findFieldByLabel(tmpl *Template, label string) (Field, bool) {
	for _, sec := range tmpl.Sections {
		for _, f := range sec.Fields {
			if f.Label == label {
				return f, true
			}
		}
	}
	return Field{}, false
}

// TestUpdateTemplate_StableFieldIdentity proves the core FR-2/INV-2 guarantee:
// a field that survives an edit keeps ONE permanent checklist_fields.id for
// life, and its draft response persists under that same id. This goes RED on
// the delete-and-reinsert build (the surviving field is minted a fresh id, so
// the assertion that the original id still exists fails).
func TestUpdateTemplate_StableFieldIdentity(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	ctx := t.Context()
	userID := ensureUser(t, "identity@yumyums.kitchen")

	// "Friday checklist" with two tasks.
	tmplID, err := insertTemplate(ctx, testPool, TemplateInput{
		Name:             "Friday Checklist (identity)",
		RequiresApproval: false,
		Sections: []SectionInput{{
			Title: "Closing", Order: 0,
			Fields: []FieldInput{
				{Type: "checkbox", Label: "Wipe counters", Order: 0},
				{Type: "checkbox", Label: "Check fridge temps", Order: 1},
			},
		}},
		Schedules: []ScheduleInput{{ActiveDays: []int{5}}},
	}, userID)
	if err != nil {
		t.Fatalf("insertTemplate: %v", err)
	}
	t.Cleanup(func() { _, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, tmplID) })

	before, err := getTemplateByID(ctx, testPool, tmplID)
	if err != nil || before == nil {
		t.Fatalf("getTemplateByID (before): %v", err)
	}
	wipe := fieldByLabel(t, before, "Wipe counters")
	fridge := fieldByLabel(t, before, "Check fridge temps")

	// A crew member checks the surviving field — a draft response under wipe.ID.
	if _, err := saveResponse(ctx, testPool, wipe.ID, json.RawMessage(`true`), userID, 0); err != nil {
		t.Fatalf("saveResponse (draft): %v", err)
	}

	// The "cut": drop "Check fridge temps", keep "Wipe counters" (Builder sends
	// its existing id), add a genuinely new field.
	if err := updateTemplate(ctx, testPool, tmplID, TemplateInput{
		Name:             "Friday Checklist (identity)",
		RequiresApproval: false,
		Sections: []SectionInput{{
			Title: "Closing", Order: 0,
			Fields: []FieldInput{
				{ID: wipe.ID, Type: "checkbox", Label: "Wipe counters", Order: 0},
				{Type: "checkbox", Label: "Restock napkins", Order: 1},
			},
		}},
		Schedules: []ScheduleInput{{ActiveDays: []int{5}}},
	}); err != nil {
		t.Fatalf("updateTemplate: %v", err)
	}

	after, err := getTemplateByID(ctx, testPool, tmplID)
	if err != nil || after == nil {
		t.Fatalf("getTemplateByID (after): %v", err)
	}

	// (1) The surviving field keeps its EXACT id — stable identity.
	kept, ok := findFieldByLabel(after, "Wipe counters")
	if !ok {
		t.Fatalf("surviving field 'Wipe counters' vanished from template")
	}
	if kept.ID != wipe.ID {
		t.Fatalf("STABLE IDENTITY VIOLATED: surviving field id churned %s -> %s", wipe.ID, kept.ID)
	}

	// (2) The removed field is gone from checklist_fields.
	if _, ok := findFieldByLabel(after, "Check fridge temps"); ok {
		t.Fatalf("removed field 'Check fridge temps' still present")
	}
	var fridgeStillExists bool
	if err := testPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM checklist_fields WHERE id=$1)`, fridge.ID).Scan(&fridgeStillExists); err != nil {
		t.Fatalf("query fridge existence: %v", err)
	}
	if fridgeStillExists {
		t.Fatalf("removed field row %s not deleted", fridge.ID)
	}

	// (3) The new field got a fresh id (not a reused/churned one).
	fresh, ok := findFieldByLabel(after, "Restock napkins")
	if !ok {
		t.Fatalf("new field 'Restock napkins' not inserted")
	}
	if fresh.ID == wipe.ID || fresh.ID == fridge.ID {
		t.Fatalf("new field reused an existing id: %s", fresh.ID)
	}

	// (4) The draft response written under the surviving id still lands on it —
	// the persistence guarantee that field-id churn used to break.
	var draftCount int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM submission_responses WHERE field_id=$1 AND submission_id IS NULL`,
		wipe.ID).Scan(&draftCount); err != nil {
		t.Fatalf("query draft: %v", err)
	}
	if draftCount != 1 {
		t.Fatalf("draft response under surviving field id lost: want 1 row, got %d", draftCount)
	}
}

// TestSaveResponse_UnknownFieldRejected proves the FR-3/INV-4 app-level existence
// check: a write naming a field absent from the current template returns
// ErrUnknownField. RED on the pre-check build (the field_id FK was dropped, so
// the INSERT succeeds silently and saveResponse returns nil).
func TestSaveResponse_UnknownFieldRejected(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	ctx := t.Context()
	userID := ensureUser(t, "unknownfield@yumyums.kitchen")

	// A field id that does not exist in any template.
	deadFieldID := "00000000-0000-4000-8000-00000000dead"
	_, err := saveResponse(ctx, testPool, deadFieldID, json.RawMessage(`true`), userID, 0)
	if !errors.Is(err, ErrUnknownField) {
		t.Fatalf("saveResponse to absent field: want ErrUnknownField, got %v", err)
	}

	// Sanity: a live field is accepted.
	tmplID, err := insertTemplate(ctx, testPool, TemplateInput{
		Name: "Live template (unknownfield)",
		Sections: []SectionInput{{
			Title: "S", Order: 0,
			Fields: []FieldInput{{Type: "checkbox", Label: "Real task", Order: 0}},
		}},
	}, userID)
	if err != nil {
		t.Fatalf("insertTemplate: %v", err)
	}
	t.Cleanup(func() { _, _ = testPool.Exec(context.Background(), `DELETE FROM checklist_templates WHERE id=$1`, tmplID) })
	tmpl, err := getTemplateByID(ctx, testPool, tmplID)
	if err != nil || tmpl == nil {
		t.Fatalf("getTemplateByID: %v", err)
	}
	real := fieldByLabel(t, tmpl, "Real task")
	if _, err := saveResponse(ctx, testPool, real.ID, json.RawMessage(`true`), userID, 0); err != nil {
		t.Fatalf("saveResponse to live field should succeed, got %v", err)
	}
}

// TestSaveResponseHandler_UnknownField422 proves the loud-rejection HTTP
// contract: SaveResponseHandler returns 422 {"error":"unknown_field"} for a
// dead field id. RED on the pre-check build (the handler returns 204).
func TestSaveResponseHandler_UnknownField422(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	userID := ensureUser(t, "handler422@yumyums.kitchen")

	body, _ := json.Marshal(SaveResponseInput{
		FieldID: "00000000-0000-4000-8000-0000000beef0",
		Value:   json.RawMessage(`true`),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/saveResponse", strings.NewReader(string(body)))
	ctx := context.WithValue(req.Context(), auth.CtxKeyUser, &auth.User{ID: userID, DisplayName: "Test User", Roles: []string{"admin"}})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	SaveResponseHandler(testPool).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want HTTP 422, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	var env map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, rec.Body.String())
	}
	if env["error"] != "unknown_field" {
		t.Fatalf(`want {"error":"unknown_field"}, got %s`, rec.Body.String())
	}
}
