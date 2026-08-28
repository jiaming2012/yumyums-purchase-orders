package onboarding

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── manager override: the escape valve for broken training media ────────────
//
// A video part whose media is dead (stranded URL, failed upload) hard-blocked
// the hire: the completeness SQL demands a video_part progress row and D-09
// demands 95% watched, with no other way to produce the row (operator,
// 2026-08-26). The override writes that same row FOR the hire — manager/admin
// only, reason required, attribution stored in the row's value column and
// surfaced on the part as `override`.

func overrideReq(t *testing.T, userID, name, role, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/overrideVideoWatched",
		strings.NewReader(body)).WithContext(obUserContext(userID, name, role))
	rec := httptest.NewRecorder()
	OverrideVideoWatchedHandler(testPool)(rec, req)
	return rec
}

func TestOverrideVideoWatchedRequiresManager(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	crew := obEnsureUser(t, "ov-crew@yumyums.kitchen", []string{"team_member"})
	hire := obEnsureUser(t, "ov-hire@yumyums.kitchen", []string{"team_member"})
	partID := seedVideoPart(t, "OV Authz", "Pre-heat", "https://dead.example/v.mov")

	rec := overrideReq(t, crew, "Crew", "team_member",
		`{"hire_id":"`+hire+`","part_id":"`+partID+`","reason":"broken"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("crew caller = %d; want 403", rec.Code)
	}
	var n int
	_ = testPool.QueryRow(t.Context(),
		`SELECT count(*) FROM ob_progress WHERE item_id = $1`, partID).Scan(&n)
	if n != 0 {
		t.Errorf("forbidden call wrote %d progress row(s); want 0", n)
	}
}

func TestOverrideVideoWatchedRequiresReason(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	mgr := obEnsureUser(t, "ov-mgr@yumyums.kitchen", []string{"manager"})
	hire := obEnsureUser(t, "ov-hire2@yumyums.kitchen", []string{"team_member"})
	partID := seedVideoPart(t, "OV Reason", "Pre-heat", "https://dead.example/v.mov")

	rec := overrideReq(t, mgr, "Mgr", "manager",
		`{"hire_id":"`+hire+`","part_id":"`+partID+`","reason":"   "}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("blank reason = %d; want 400", rec.Code)
	}
}

func TestOverrideVideoWatchedWritesAttributedRow(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	mgr := obEnsureUser(t, "ov-mgr2@yumyums.kitchen", []string{"manager"})
	hire := obEnsureUser(t, "ov-hire3@yumyums.kitchen", []string{"team_member"})
	partID := seedVideoPart(t, "OV Write", "Pre-heat", "https://dead.example/v.mov")

	rec := overrideReq(t, mgr, "Manager M.", "manager",
		`{"hire_id":"`+hire+`","part_id":"`+partID+`","reason":"Video file broken — B2 cutover"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("override = %d body=%s; want 200", rec.Code, rec.Body.String())
	}

	// The row is the SAME shape the 95%-watched path writes, so the
	// completeness SQL needs no special case — plus the attribution JSON.
	var progressType, value string
	err := testPool.QueryRow(t.Context(),
		`SELECT progress_type, COALESCE(value, '') FROM ob_progress
		 WHERE hire_id = $1 AND item_id = $2`, hire, partID).Scan(&progressType, &value)
	if err != nil {
		t.Fatalf("read back progress row: %v", err)
	}
	if progressType != "video_part" {
		t.Errorf("progress_type = %q; want video_part", progressType)
	}
	var attr struct {
		Override bool   `json:"override"`
		ByID     string `json:"by_id"`
		ByName   string `json:"by_name"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(value), &attr); err != nil {
		t.Fatalf("value is not attribution JSON: %q (%v)", value, err)
	}
	if !attr.Override || attr.ByID != mgr || attr.ByName != "Manager M." || attr.Reason != "Video file broken — B2 cutover" {
		t.Errorf("attribution = %+v; want override by the manager with the given reason", attr)
	}

	// Idempotent re-override updates in place (ON CONFLICT), no duplicate row.
	rec = overrideReq(t, mgr, "Manager M.", "manager",
		`{"hire_id":"`+hire+`","part_id":"`+partID+`","reason":"second reason"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("re-override = %d; want 200", rec.Code)
	}
	var n int
	_ = testPool.QueryRow(t.Context(),
		`SELECT count(*) FROM ob_progress WHERE hire_id = $1 AND item_id = $2`, hire, partID).Scan(&n)
	if n != 1 {
		t.Errorf("progress rows = %d; want exactly 1", n)
	}
}
