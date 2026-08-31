package inventory

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// captureLog is a slog.Handler that records emitted records so a test can
// assert a specific line was written. It is set as the default logger for the
// duration of one handler call and restored after.
type captureLog struct {
	mu      sync.Mutex
	records []slog.Record
}

func (c *captureLog) Enabled(context.Context, slog.Level) bool { return true }
func (c *captureLog) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, r.Clone())
	return nil
}
func (c *captureLog) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *captureLog) WithGroup(string) slog.Handler      { return c }

// find returns the first record whose message equals msg, and its attributes
// flattened into a map, plus whether it was found.
func (c *captureLog) find(msg string) (map[string]slog.Value, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range c.records {
		if r.Message == msg {
			attrs := map[string]slog.Value{}
			r.Attrs(func(a slog.Attr) bool {
				attrs[a.Key] = a.Value
				return true
			})
			return attrs, true
		}
	}
	return nil, false
}

// withCapturedLog swaps the default slog logger for a capturer for the duration
// of fn and returns it.
func withCapturedLog(fn func()) *captureLog {
	cap := &captureLog{}
	prev := slog.Default()
	slog.SetDefault(slog.New(cap))
	defer slog.SetDefault(prev)
	fn()
	return cap
}

// TestPeriodSummary_EmitsVisibilityLog is the RED-FIRST test for B-139.
//
// Before the fix, PeriodSummaryHandler emits ten slog.Error lines on failure
// paths and NOT ONE success line — so a blocked payroll week leaves no trace in
// prod logs, indistinguishable from a skipped week. This asserts one slog.Info
// "period-summary served" line at the end of a successful response, carrying the
// period bounds, the completeness verdict, and the two blocking-set counts.
//
// This test is RED on the current tree (no such log line) and GREEN after the
// fix. It runs against the DB fixture harness (skips if DB_TEST_URL unset).
func TestPeriodSummary_EmitsVisibilityLog(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not set — skipping DB-coupled visibility test")
	}
	resetFixtures(t)

	// Seed one confirmed COGS event with a linked line item: Ready=true,
	// no pending review, no unlinked line item.
	vendorID := insertVendor(t, "Visibility Vendor")
	piID := insertPurchaseItem(t, "Salmon Fillet")
	insertEventAndLine(t, vendorID, "2026-05-27", 5.00, 55.00, 10.00, 5, piID)

	cap := withCapturedLog(func() {
		req := httptest.NewRequest(http.MethodGet, "/?from=2026-05-25&to=2026-05-31", nil)
		rec := httptest.NewRecorder()
		PeriodSummaryHandler(testPool, []string{"COGS"}).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
	})

	attrs, ok := cap.find("period-summary served")
	if !ok {
		t.Fatalf(`no slog.Info "period-summary served" line was emitted — B-139: a successful /period-summary response leaves no trace in prod logs`)
	}

	if got := attrs["from"].String(); got != "2026-05-25" {
		t.Errorf(`log key "from" = %q, want "2026-05-25"`, got)
	}
	if got := attrs["to"].String(); got != "2026-05-31" {
		t.Errorf(`log key "to" = %q, want "2026-05-31"`, got)
	}
	if _, has := attrs["ready"]; !has {
		t.Errorf(`log is missing key "ready" (the completeness verdict)`)
	} else if attrs["ready"].Kind() != slog.KindBool {
		t.Errorf(`log key "ready" kind = %v, want Bool`, attrs["ready"].Kind())
	}
	if !attrs["ready"].Bool() {
		t.Errorf(`log key "ready" = false, want true (seeded a clean, complete period)`)
	}
	if _, has := attrs["pending_review_count"]; !has {
		t.Errorf(`log is missing key "pending_review_count"`)
	} else if attrs["pending_review_count"].Int64() != 0 {
		t.Errorf(`log key "pending_review_count" = %d, want 0`, attrs["pending_review_count"].Int64())
	}
	if _, has := attrs["unlinked_line_item_count"]; !has {
		t.Errorf(`log is missing key "unlinked_line_item_count"`)
	} else if attrs["unlinked_line_item_count"].Int64() != 0 {
		t.Errorf(`log key "unlinked_line_item_count" = %d, want 0`, attrs["unlinked_line_item_count"].Int64())
	}
}

// TestPeriodSummary_VisibilityLogReflectsBlockedWeek asserts the log line's
// ready=false / non-zero counts on a BLOCKED period — the exact case B-139 is
// about: an unlinked line item makes Ready=false, and the log must record it.
func TestPeriodSummary_VisibilityLogReflectsBlockedWeek(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not set — skipping DB-coupled visibility test")
	}
	resetFixtures(t)

	vendorID := insertVendor(t, "Blocked Vendor")
	// purchaseItemID="" → unlinked line item → Ready=false.
	insertEventAndLine(t, vendorID, "2026-05-27", 5.00, 55.00, 10.00, 5, "")

	cap := withCapturedLog(func() {
		req := httptest.NewRequest(http.MethodGet, "/?from=2026-05-25&to=2026-05-31", nil)
		rec := httptest.NewRecorder()
		PeriodSummaryHandler(testPool, []string{"COGS"}).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
	})

	attrs, ok := cap.find("period-summary served")
	if !ok {
		t.Fatalf(`no slog.Info "period-summary served" line was emitted for a blocked week`)
	}
	if attrs["ready"].Bool() {
		t.Errorf(`log key "ready" = true, want false (unlinked line item blocks the week)`)
	}
	if attrs["unlinked_line_item_count"].Int64() < 1 {
		t.Errorf(`log key "unlinked_line_item_count" = %d, want >= 1`, attrs["unlinked_line_item_count"].Int64())
	}
}

// ensure strings import is used even if helpers above change.
var _ = strings.TrimSpace
