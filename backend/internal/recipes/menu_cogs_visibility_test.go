package recipes

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// captureLog is a slog.Handler that records emitted records so a test can
// assert a specific line was written.
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

func withCapturedLog(fn func()) *captureLog {
	cap := &captureLog{}
	prev := slog.Default()
	slog.SetDefault(slog.New(cap))
	defer slog.SetDefault(prev)
	fn()
	return cap
}

// callMenuCogsCaptured runs the handler with the default logger swapped for a
// capturer, returning both the recorder and the capture.
func callMenuCogsCaptured(t *testing.T, pool *pgxpool.Pool, query string) (*httptest.ResponseRecorder, *captureLog) {
	t.Helper()
	rec := httptest.NewRecorder()
	cap := withCapturedLog(func() {
		req := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
		MenuCogsHandler(pool).ServeHTTP(rec, req)
	})
	return rec, cap
}

// TestMenuCogs_EmitsVisibilityLog is the RED-FIRST test for the /menu-cogs half
// of B-139. The endpoint had no success reader at all; its first-ever caller
// should be visible in prod logs. Asserts one slog.Info "menu-cogs served" line
// at the end of a successful response carrying period bounds, the menu-item
// count, and the query mode.
//
// RED on the current tree (no such log line), GREEN after the fix.
func TestMenuCogs_EmitsVisibilityLog(t *testing.T) {
	pool := setupTestDB(t)

	// Seed one menu item with a recipe + sales so the summary returns a row.
	vendorID := seedVendor(t, pool, "Vis Vendor")
	miID := seedMenuItem(t, pool, "Salmon Plate")
	piID := seedPurchaseItem(t, pool, "Salmon Fillet")
	eventID := seedPurchaseEvent(t, pool, vendorID, "2026-05-27", 10.00, 110.00)
	seedPurchaseLineItem(t, pool, eventID, piID, "Salmon Fillet", 2, 10.00)
	seedRecipe(t, pool, miID, piID, 50.0)
	seedDailyMenuSales(t, pool, miID, "2026-05-27", 4, 80.00)

	rec, cap := callMenuCogsCaptured(t, pool, "from=2026-05-25&to=2026-05-31")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	attrs, ok := cap.find("menu-cogs served")
	if !ok {
		t.Fatalf(`no slog.Info "menu-cogs served" line was emitted — /menu-cogs has no success reader (B-139)`)
	}
	if got := attrs["from"].String(); got != "2026-05-25" {
		t.Errorf(`log key "from" = %q, want "2026-05-25"`, got)
	}
	if got := attrs["to"].String(); got != "2026-05-31" {
		t.Errorf(`log key "to" = %q, want "2026-05-31"`, got)
	}
	if _, has := attrs["menu_item_count"]; !has {
		t.Errorf(`log is missing key "menu_item_count"`)
	} else if attrs["menu_item_count"].Int64() != 1 {
		t.Errorf(`log key "menu_item_count" = %d, want 1`, attrs["menu_item_count"].Int64())
	}
	if _, has := attrs["breakdown"]; !has {
		t.Errorf(`log is missing key "breakdown"`)
	} else if attrs["breakdown"].Bool() {
		t.Errorf(`log key "breakdown" = true, want false (default summary mode)`)
	}
}

// TestMenuCogs_VisibilityLogBreakdownMode asserts the breakdown flag is
// reflected in the log line when ?breakdown=true is requested.
func TestMenuCogs_VisibilityLogBreakdownMode(t *testing.T) {
	pool := setupTestDB(t)

	vendorID := seedVendor(t, pool, "Vis Vendor BD")
	miID := seedMenuItem(t, pool, "Salmon Plate BD")
	piID := seedPurchaseItem(t, pool, "Salmon Fillet BD")
	eventID := seedPurchaseEvent(t, pool, vendorID, "2026-05-27", 10.00, 110.00)
	seedPurchaseLineItem(t, pool, eventID, piID, "Salmon Fillet BD", 2, 10.00)
	seedRecipe(t, pool, miID, piID, 50.0)
	seedDailyMenuSales(t, pool, miID, "2026-05-27", 4, 80.00)

	rec, cap := callMenuCogsCaptured(t, pool, "from=2026-05-25&to=2026-05-31&breakdown=true")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	attrs, ok := cap.find("menu-cogs served")
	if !ok {
		t.Fatalf(`no slog.Info "menu-cogs served" line was emitted in breakdown mode`)
	}
	if !attrs["breakdown"].Bool() {
		t.Errorf(`log key "breakdown" = false, want true`)
	}
}
