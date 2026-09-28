package receipt

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// ---------------------------------------------------------------------------
// Transient Anthropic failures must not park a receipt.
//
// RED-FIRST. On 2026-09-28 the Anthropic account ran out of credits; every
// parse failed twice with `400 … Your credit balance is too low` and the
// worker did what it does on a double failure — stored the error on the row.
// That guard exists so a receipt the models genuinely cannot read is not
// re-parsed forever, but a billing wall is not that: the receipt is fine and
// the next poll after a top-up would succeed. Six receipts sat parked, the
// card showed a truncated "400 Bad Request", and "Reprocess All Pending" was
// the only way out.
// ---------------------------------------------------------------------------

const creditBalanceErr = `ParseReceipt: API call failed: POST "https://api.anthropic.com/v1/messages": 400 Bad Request (Request-ID: req_011CfWESjusDYaqeb2s9L67T) {"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits."},"request_id":"req_011CfWESjusDYaqeb2s9L67T"}`

func TestIsTransientAPIError_Classifies(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantTrans  bool
		wantReason string
	}{
		{"credit balance 400 (wrapped, as the worker sees it)", fmt.Errorf("sonnet: %w", errors.New(creditBalanceErr)), true, "Anthropic account out of credits"},
		{"typed 400 with credit-balance body", &anthropic.Error{StatusCode: 400, RequestID: "req_x"}, false, ""}, // a plain 400 with no billing text is a real request error
		{"typed 429 rate limit", &anthropic.Error{StatusCode: 429}, true, "Anthropic rate limit hit"},
		{"typed 529 overloaded", &anthropic.Error{StatusCode: 529}, true, "Anthropic API overloaded"},
		{"typed 500", &anthropic.Error{StatusCode: 500}, true, "Anthropic API error"},
		{"typed 401 bad key", &anthropic.Error{StatusCode: 401}, false, ""},
		{"genuine parse failure", errors.New("failed to unmarshal: invalid character '<' looking for beginning of value"), false, ""},
		{"nil", nil, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, reason := isTransientAPIError(c.err)
			if got != c.wantTrans {
				t.Fatalf("transient = %v, want %v", got, c.wantTrans)
			}
			if reason != c.wantReason {
				t.Errorf("reason = %q, want %q", reason, c.wantReason)
			}
		})
	}
}

// A billing failure on both attempts lands the row in the queue with a
// `transient:` marker (so the card can say why), and the NEXT poll re-parses
// it — unlike a genuine double parse failure, which stays parked.
func TestRunIngestCycle_TransientFailureIsRetriedOnNextPoll(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	tx := MercuryTransaction{
		ID: "T-transient", Amount: -42.50, CreatedAt: "2026-09-27T10:00:00Z",
		Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
	}
	cfg := WorkerConfig{MercuryAPIKey: "stub", AnthropicAPIKey: "stub", Pool: testPool, LookbackDays: 14}

	// Poll 1: credits are gone — both attempts fail the same way.
	first := &workerStubs{
		txns:      []MercuryTransaction{tx},
		parseErr:  errors.New(creditBalanceErr),
		sonnetErr: errors.New(creditBalanceErr),
	}
	installWorkerStubs(t, first)
	res, err := runIngestCycle(t.Context(), cfg)
	if err != nil {
		t.Fatalf("poll 1: %v", err)
	}
	if res.PendingReview != 1 {
		t.Fatalf("poll 1 PendingReview = %d, want 1 (the swipe still lands in the queue)", res.PendingReview)
	}
	var reason, parseErr sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT reason, parse_error FROM pending_purchases WHERE bank_tx_id = $1`, tx.ID).Scan(&reason, &parseErr); err != nil {
		t.Fatalf("select after poll 1: %v", err)
	}
	if reason.String != "Receipt could not be parsed automatically" {
		t.Errorf("reason = %q", reason.String)
	}
	if parseErr.String != "transient: Anthropic account out of credits" {
		t.Errorf("parse_error = %q, want the transient marker with the human reason", parseErr.String)
	}

	// Poll 2: credits are back — the row must be re-parsed and promoted.
	second := &workerStubs{
		txns:         []MercuryTransaction{tx},
		parseItems:   []ReceiptItem{{Name: "Salmon", Quantity: 1, Price: 42.50}},
		parseSummary: ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 42.50, TotalUnits: 1},
	}
	installWorkerStubs(t, second)
	res, err = runIngestCycle(t.Context(), cfg)
	if err != nil {
		t.Fatalf("poll 2: %v", err)
	}
	if second.parseCallCount == 0 {
		t.Fatalf("poll 2 never called the parser — the transient row was treated as permanently failed")
	}
	if res.AutoCreated != 1 {
		t.Errorf("poll 2 AutoCreated = %d, want 1", res.AutoCreated)
	}
	var pendingCount, eventCount int
	_ = testPool.QueryRow(t.Context(), `SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id = $1 AND confirmed_at IS NULL AND discarded_at IS NULL`, tx.ID).Scan(&pendingCount)
	_ = testPool.QueryRow(t.Context(), `SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id = $1`, tx.ID).Scan(&eventCount)
	if pendingCount != 0 || eventCount != 1 {
		t.Errorf("after poll 2: pending=%d event=%d, want 0/1", pendingCount, eventCount)
	}
}

// The existing guard still holds: a genuine double parse failure (a stored
// error that is NOT transient) is never re-parsed by a later poll.
func TestRunIngestCycle_GenuineParseFailureStaysParked(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	tx := MercuryTransaction{
		ID: "T-genuine", Amount: -42.50, CreatedAt: "2026-09-27T10:00:00Z",
		Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
	}
	cfg := WorkerConfig{MercuryAPIKey: "stub", AnthropicAPIKey: "stub", Pool: testPool, LookbackDays: 14}
	first := &workerStubs{
		txns:      []MercuryTransaction{tx},
		parseErr:  errors.New("failed to unmarshal: invalid character '<'"),
		sonnetErr: errors.New("failed to unmarshal: invalid character '<'"),
	}
	installWorkerStubs(t, first)
	if _, err := runIngestCycle(t.Context(), cfg); err != nil {
		t.Fatalf("poll 1: %v", err)
	}
	var parseErr sql.NullString
	_ = testPool.QueryRow(t.Context(), `SELECT parse_error FROM pending_purchases WHERE bank_tx_id = $1`, tx.ID).Scan(&parseErr)
	if !parseErr.Valid || len(parseErr.String) < 10 || parseErr.String[:10] == "transient:" {
		t.Fatalf("parse_error = %+v, want the raw concatenated error (not a transient marker)", parseErr)
	}

	second := &workerStubs{txns: []MercuryTransaction{tx}, parseItems: []ReceiptItem{{Name: "Salmon", Quantity: 1, Price: 42.50}}, parseSummary: ReceiptSummary{Total: 42.50, TotalUnits: 1}}
	installWorkerStubs(t, second)
	res, err := runIngestCycle(t.Context(), cfg)
	if err != nil {
		t.Fatalf("poll 2: %v", err)
	}
	if second.parseCallCount != 0 || res.Cached != 1 {
		t.Errorf("poll 2 re-parsed a genuinely failed row: parseCalls=%d cached=%d", second.parseCallCount, res.Cached)
	}
}
