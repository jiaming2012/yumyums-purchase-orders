package marketing

// zz_spike_e2_test.go — THROWAWAY spike test for E2-02 (mms-send-on-signup, D-KR2). Copied by
// the spike script into the worktree beside subscribers_test.go and run there; never part of
// the tree. It uses the package's own helpers (setupTestDB, ffImporter, mountedMux, do) and
// records the red-first BASELINE for the card's two compliance tests:
//   (a) a subscriber with a phone and NO sms consent exists    — the refusal subject
//   (b) a subscriber with a phone and sms consent exists       — the send subject
//   (c) zero code_sent events                                   — nothing has ever been sent
//   (d) POST /api/v1/marketing/sms/inbound answers 404          — no STOP door exists
//   (e) the refusal subject's opted_out_at is NULL
// PASS means the gaps are real and the fixture carries both subjects.

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestSpikeE2ConsentAndStopBaseline(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()
	if _, err := ImportSubscribers(ctx, pool, ffImporter(t)); err != nil {
		t.Fatalf("import: %v", err)
	}

	type subj struct {
		id      string
		consent bool
		optedAt *time.Time
	}
	pick := func(phone string) subj {
		var s subj
		if err := pool.QueryRow(ctx,
			`SELECT id::text, sms_consent, opted_out_at FROM subscribers WHERE phone_e164 = $1`, phone).
			Scan(&s.id, &s.consent, &s.optedAt); err != nil {
			t.Fatalf("no subscriber with phone %s: %v", phone, err)
		}
		return s
	}
	refusal := pick("+17735550117") // fixture 1183 — consent ["Email"] only
	send := pick("+17735559930")    // fixture 1182 — consent ["Phone"]
	if refusal.consent {
		t.Errorf("(a) refusal subject +17735550117 has sms_consent=true; want false")
	}
	if !send.consent {
		t.Errorf("(b) send subject +17735559930 has sms_consent=false; want true")
	}
	var sent, total, withPhone, consenting int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM subscriber_events WHERE kind='code_sent'`).Scan(&sent)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM subscribers`).Scan(&total)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM subscribers WHERE phone_e164 IS NOT NULL`).Scan(&withPhone)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM subscribers WHERE phone_e164 IS NOT NULL AND sms_consent`).Scan(&consenting)
	if sent != 0 {
		t.Errorf("(c) code_sent events = %d, want 0", sent)
	}
	if refusal.optedAt != nil {
		t.Errorf("(e) refusal subject opted_out_at = %v, want NULL", refusal.optedAt)
	}

	// (d) the would-be carrier webhook: unauthenticated, form-encoded, as a carrier posts it.
	mux := mountedMux(testDeps(pool))
	rec := do(t, mux, context.Background(), http.MethodPost, "/api/v1/marketing/sms/inbound",
		map[string]any{"From": "+17735550117", "Body": "STOP"})
	if rec.Code != http.StatusNotFound {
		t.Errorf("(d) POST /sms/inbound = %d, want 404 (no STOP door exists today)", rec.Code)
	}

	t.Logf("SPIKE-E2-2: subscribers=%d with_phone=%d sms_consenting=%d code_sent=%d inbound_status=%d refusal_subject=%s send_subject=%s",
		total, withPhone, consenting, sent, rec.Code, refusal.id, send.id)
}
