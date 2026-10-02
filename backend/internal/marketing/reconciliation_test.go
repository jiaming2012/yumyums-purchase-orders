package marketing

import (
	"net/http"
	"testing"
	"time"
)

// ── card H3b · the queue, the decisions, the orphan rate ──

func TestQueueOrdersOverridesThenOrphansThenUnmatched(t *testing.T) {
	pool := setupReconDB(t)
	mux := mountedMux(testDeps(pool))
	manager := seedUser(t, pool, "manager")
	campaign := seedCampaignRow(t, pool, manager, "queue", "Queue Campaign", 300, nil)
	code := seedCodeRow(t, pool, manager, campaign, "QQQQQ2", "truck_sign", nil)

	base := time.Now().Add(-3 * time.Hour)
	// Seeded in the WRONG order on purpose: the queue's order must come from
	// the bucket ladder, not from insertion or scanned_at.
	unmatched := seedAttempt(t, pool, attemptFixture{
		CodeID: &code, CampaignID: &campaign, ScannedAt: base.Add(10 * time.Minute),
		OrderNumber: strptr("4821"),
	})
	orphan := seedAttempt(t, pool, attemptFixture{
		CodeID: &code, CampaignID: &campaign, ScannedAt: base.Add(20 * time.Minute),
	})
	override := seedAttempt(t, pool, attemptFixture{
		CodeID: &code, CampaignID: &campaign, ScannedAt: base.Add(30 * time.Minute),
		Override: true,
	})

	rec := do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/reconciliation/queue", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /reconciliation/queue = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var out ReconQueueResponse
	decode(t, rec, &out)

	if len(out.Queue) != 3 {
		t.Fatalf("queue has %d rows, want 3 (overrides+orphans+unmatched)\nbody: %s", len(out.Queue), rec.Body.String())
	}
	wantBuckets := []string{"override", "orphan", "unmatched"}
	wantIDs := []string{override, orphan, unmatched}
	for i := range wantBuckets {
		if out.Queue[i].Bucket != wantBuckets[i] {
			t.Errorf("queue[%d].bucket = %q, want %q", i, out.Queue[i].Bucket, wantBuckets[i])
		}
		if out.Queue[i].ID != wantIDs[i] {
			t.Errorf("queue[%d].id = %q, want the %s attempt %q", i, out.Queue[i].ID, wantBuckets[i], wantIDs[i])
		}
	}
	if len(out.Overrides) != 1 || len(out.Orphans) != 1 || len(out.Unmatched) != 1 {
		t.Errorf("per-bucket arrays = %d/%d/%d, want 1/1/1",
			len(out.Overrides), len(out.Orphans), len(out.Unmatched))
	}
}

// An unmatched attempt carries the nearest Toast order within ±30 min of
// scanned_at as a suggestion; one outside the window does not.
func TestUnmatchedCarriesNearestOrderSuggestionWithinThirtyMinutes(t *testing.T) {
	pool := setupReconDB(t)
	mux := mountedMux(testDeps(pool))
	manager := seedUser(t, pool, "manager")
	campaign := seedCampaignRow(t, pool, manager, "sugg", "Suggestion", 300, nil)
	code := seedCodeRow(t, pool, manager, campaign, "SSSSS2", "flyer", nil)

	scannedAt := time.Now().Add(-4 * time.Hour)
	attempt := seedAttempt(t, pool, attemptFixture{
		CodeID: &code, CampaignID: &campaign, ScannedAt: scannedAt,
		OrderNumber: strptr("9999"),
	})
	// 12 minutes before the scan → inside the window; 90 minutes after → outside.
	seedToastOrder(t, pool, scannedAt, "77", 1800, 300, scannedAt.Add(-12*time.Minute))
	seedToastOrder(t, pool, scannedAt, "78", 2500, 0, scannedAt.Add(90*time.Minute))

	rec := do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/reconciliation/queue", nil)
	var out ReconQueueResponse
	decode(t, rec, &out)
	if len(out.Unmatched) != 1 {
		t.Fatalf("unmatched bucket = %d rows, want 1\nbody: %s", len(out.Unmatched), rec.Body.String())
	}
	row := out.Unmatched[0]
	if row.ID != attempt {
		t.Fatalf("unmatched[0].id = %q, want %q", row.ID, attempt)
	}
	if row.Suggestion == nil {
		t.Fatalf("unmatched[0].suggestion is null; want order 77 (12 min before the scan)")
	}
	if row.Suggestion.OrderNumber != "77" {
		t.Errorf("suggestion.order_number = %q, want %q (the nearest inside ±30 min)", row.Suggestion.OrderNumber, "77")
	}
	if row.Suggestion.AmountCents != 1800 {
		t.Errorf("suggestion.amount_cents = %d, want 1800", row.Suggestion.AmountCents)
	}
}

// §5: `400 note_required` when reason="other" and the note is empty. Every other
// reason posts without one.
func TestDeclineOtherRequiresNote(t *testing.T) {
	pool := setupReconDB(t)
	mux := mountedMux(testDeps(pool))
	manager := seedUser(t, pool, "manager")
	campaign := seedCampaignRow(t, pool, manager, "decl", "Decline", 250, nil)
	code := seedCodeRow(t, pool, manager, campaign, "DDDDD2", "flyer", nil)
	attempt := seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign})

	path := "/api/v1/marketing/reconciliation/" + attempt + "/decline"

	// reason=other, no note → 400 note_required
	rec := do(t, mux, userCtx(manager, "manager"), http.MethodPost, path,
		map[string]any{"reason": "other"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("decline other without a note = %d, want 400\nbody: %s", rec.Code, rec.Body.String())
	}
	var envelope map[string]any
	decode(t, rec, &envelope)
	if envelope["error"] != "note_required" {
		t.Errorf("error = %v, want note_required", envelope["error"])
	}

	// reason=other, whitespace-only note → still 400: a space is not a note.
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodPost, path,
		map[string]any{"reason": "other", "note": "   "})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("decline other with a whitespace note = %d, want 400\nbody: %s", rec.Code, rec.Body.String())
	}

	// reason=other WITH a note → 200
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodPost, path,
		map[string]any{"reason": "other", "note": "staff scanned the wrong screen"})
	if rec.Code != http.StatusOK {
		t.Fatalf("decline other with a note = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}

	// a non-`other` reason needs no note.
	attempt2 := seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign})
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodPost,
		"/api/v1/marketing/reconciliation/"+attempt2+"/decline",
		map[string]any{"reason": "customer_left"})
	if rec.Code != http.StatusOK {
		t.Fatalf("decline customer_left without a note = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}

	// the declined bucket shows reason, note, who and when (§5 row /declined).
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/reconciliation/declined", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /reconciliation/declined = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var declined ReconDeclinedResponse
	decode(t, rec, &declined)
	if len(declined.Declined) != 2 {
		t.Fatalf("declined bucket = %d rows, want 2\nbody: %s", len(declined.Declined), rec.Body.String())
	}
	found := false
	for _, row := range declined.Declined {
		if row.ID == attempt {
			found = true
			if row.Reason == nil || *row.Reason != "other" {
				t.Errorf("declined row reason = %v, want other", row.Reason)
			}
			if row.Note == nil || *row.Note != "staff scanned the wrong screen" {
				t.Errorf("declined row note = %v, want the posted note", row.Note)
			}
			if row.DecidedBy == "" {
				t.Errorf("declined row decided_by is empty; the bucket must say who")
			}
		}
	}
	if !found {
		t.Errorf("declined bucket does not carry attempt %q", attempt)
	}
}

// §5: match against an order number that is not in toast_orders is
// `409 order_not_found`; one that is lands a decision and moves the attempt out
// of the queue.
func TestMatchUnknownOrderIs409AndKnownOrderResolves(t *testing.T) {
	pool := setupReconDB(t)
	mux := mountedMux(testDeps(pool))
	manager := seedUser(t, pool, "manager")
	campaign := seedCampaignRow(t, pool, manager, "match", "Match", 300, nil)
	code := seedCodeRow(t, pool, manager, campaign, "MMMMM2", "truck_sign", nil)
	scannedAt := time.Now().Add(-5 * time.Hour)
	attempt := seedAttempt(t, pool, attemptFixture{
		CodeID: &code, CampaignID: &campaign, ScannedAt: scannedAt,
	})
	path := "/api/v1/marketing/reconciliation/" + attempt + "/match"

	rec := do(t, mux, userCtx(manager, "manager"), http.MethodPost, path,
		map[string]any{"order_number": "4040"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("match an absent order = %d, want 409\nbody: %s", rec.Code, rec.Body.String())
	}
	var envelope map[string]any
	decode(t, rec, &envelope)
	if envelope["error"] != "order_not_found" {
		t.Errorf("error = %v, want order_not_found", envelope["error"])
	}

	seedToastOrder(t, pool, scannedAt, "4040", 2200, 300, scannedAt.Add(-5*time.Minute))
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodPost, path,
		map[string]any{"order_number": "4040"})
	if rec.Code != http.StatusOK {
		t.Fatalf("match a present order = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}

	// resolved → out of the queue, into matched_count.
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/reconciliation/queue", nil)
	var queue ReconQueueResponse
	decode(t, rec, &queue)
	if len(queue.Queue) != 0 {
		t.Errorf("queue still has %d rows after a match\nbody: %s", len(queue.Queue), rec.Body.String())
	}
	if queue.MatchedCount != 1 {
		t.Errorf("matched_count = %d, want 1", queue.MatchedCount)
	}
}

// A declined attempt can be reopened, which puts it back in the queue.
func TestReopenReturnsAttemptToTheQueue(t *testing.T) {
	pool := setupReconDB(t)
	mux := mountedMux(testDeps(pool))
	manager := seedUser(t, pool, "manager")
	campaign := seedCampaignRow(t, pool, manager, "reopen", "Reopen", 300, nil)
	code := seedCodeRow(t, pool, manager, campaign, "RRRRR2", "flyer", nil)
	attempt := seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign})

	if rec := do(t, mux, userCtx(manager, "manager"), http.MethodPost,
		"/api/v1/marketing/reconciliation/"+attempt+"/decline",
		map[string]any{"reason": "comped"}); rec.Code != http.StatusOK {
		t.Fatalf("decline = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	rec := do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/reconciliation/queue", nil)
	var queue ReconQueueResponse
	decode(t, rec, &queue)
	if len(queue.Queue) != 0 || queue.DeclinedCount != 1 {
		t.Fatalf("after decline: queue=%d declined_count=%d, want 0/1", len(queue.Queue), queue.DeclinedCount)
	}

	if rec := do(t, mux, userCtx(manager, "manager"), http.MethodPost,
		"/api/v1/marketing/reconciliation/"+attempt+"/reopen", nil); rec.Code != http.StatusOK {
		t.Fatalf("reopen = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/reconciliation/queue", nil)
	decode(t, rec, &queue)
	if len(queue.Queue) != 1 || queue.DeclinedCount != 0 {
		t.Errorf("after reopen: queue=%d declined_count=%d, want 1/0\nbody: %s",
			len(queue.Queue), queue.DeclinedCount, rec.Body.String())
	}
}

// An offline override is verified or rejected, not matched/declined — and until
// it is, it sits first in the queue (Q-KR2).
func TestVerifyAndRejectResolveAnOverride(t *testing.T) {
	pool := setupReconDB(t)
	mux := mountedMux(testDeps(pool))
	manager := seedUser(t, pool, "manager")
	campaign := seedCampaignRow(t, pool, manager, "ovr", "Override", 4000, nil)
	code := seedCodeRow(t, pool, manager, campaign, "VVVVV2", "sms", nil)
	a1 := seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign, Override: true})
	a2 := seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign, Override: true})

	for path, attempt := range map[string]string{"verify": a1, "reject": a2} {
		rec := do(t, mux, userCtx(manager, "manager"), http.MethodPost,
			"/api/v1/marketing/reconciliation/"+attempt+"/"+path,
			map[string]any{"note": "checked the tablet log"})
		if rec.Code != http.StatusOK {
			t.Fatalf("%s = %d, want 200\nbody: %s", path, rec.Code, rec.Body.String())
		}
	}
	rec := do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/reconciliation/queue", nil)
	var queue ReconQueueResponse
	decode(t, rec, &queue)
	if len(queue.Overrides) != 0 {
		t.Errorf("overrides bucket = %d after verify+reject, want 0\nbody: %s", len(queue.Overrides), rec.Body.String())
	}
}

// The queue is a WRITE surface and stays in Marketing behind the manager tier
// (decision 192).
func TestReconciliationQueueIsManagerOnly(t *testing.T) {
	pool := setupReconDB(t)
	mux := mountedMux(testDeps(pool))
	member := seedUser(t, pool, "team_member")
	for _, target := range []string{
		"/api/v1/marketing/reconciliation/queue",
		"/api/v1/marketing/reconciliation/declined",
	} {
		rec := do(t, mux, userCtx(member, "team_member"), http.MethodGet, target, nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("GET %s as team_member = %d, want 403\nbody: %s", target, rec.Code, rec.Body.String())
		}
		var envelope map[string]any
		decode(t, rec, &envelope)
		if envelope["error"] != "managers_only" {
			t.Errorf("GET %s as team_member error = %v, want managers_only", target, envelope["error"])
		}
	}
}
