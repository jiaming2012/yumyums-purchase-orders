package purchasing

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/alerts"
	"github.com/yumyums/hq/internal/db"
)

// ─────────────────────────────────────────────────────────────────────────────
// Night-crew fix WO — carried-fix-wos-sweep (T-1, the WO-cron-clock-seam).
//
// This file lands the REAL cron-DECISION unit tests that scheduler_prove_test.go
// PARKED (FR-19 / FR-20 / FR-21 / FR-22) for lack of an injectable clock.
//
// The fix: each run*Check func now takes a `now func() time.Time` seam
// (scheduler.go / repurchase.go). Production still passes time.Now (via
// runSchedulerTick); these tests inject a FROZEN clock and assert the cron
// DECISION at controlled instants — fires at the configured day-of-week+time,
// does NOT fire the minute before, and is idempotent within the week.
//
// DB-coupled legs seed the config + fixtures and assert the DB side-effect the
// decision produces (auto-lock, alert_log row, last_reset_at bump). Pure legs
// need no DB. TestMain (below) connects via DB_TEST_URL and skips gracefully
// when unreachable, mirroring inventory/period_summary_test.go.
// ─────────────────────────────────────────────────────────────────────────────

var cronTestPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		// Cannot construct pool — leave cronTestPool nil so DB tests skip.
		os.Exit(m.Run())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		os.Exit(m.Run())
	}
	if err := db.Migrate(pool); err != nil {
		pool.Close()
		panic("db.Migrate failed: " + err.Error())
	}
	cronTestPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// requireDB skips the calling test when the test DB is unreachable.
func requireDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if cronTestPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping DB-coupled cron-decision test")
	}
	return cronTestPool
}

// frozenClock returns a now func() time.Time that always yields t — the mock
// clock the seam makes injectable.
func frozenClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// chicago loads America/Chicago or fails the test (fixtures are DST-stable in
// January, the month all fixtures below use).
func chicago(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("LoadLocation(America/Chicago): %v", err)
	}
	return loc
}

// ensureAlertQueue makes alertQueue non-nil so the reminder/low-stock checks do
// not early-return. The queue is never Start()ed, so Enqueue merely buffers
// (non-blocking, 100-slot) and nothing is actually delivered — the tests assert
// on the DB idempotency-log side-effect, not on delivery. Restores prior value.
func ensureAlertQueue(t *testing.T) {
	t.Helper()
	prev := alertQueue
	alertQueue = alerts.NewQueue(alerts.Config{})
	t.Cleanup(func() { alertQueue = prev })
}

// ── Fixture helpers ──────────────────────────────────────────────────────────

func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE cutoff_config, repurchase_reset_config, alert_log, low_stock_alert_log,
		         purchase_orders, purchase_line_items, purchase_events,
		         purchase_items, item_groups, stock_count_overrides, vendors
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func seedCutoffConfig(t *testing.T, pool *pgxpool.Pool, dow int, cutoff, tz string) {
	t.Helper()
	if _, err := UpsertCutoffConfig(context.Background(), pool, dow, cutoff, tz); err != nil {
		t.Fatalf("seed cutoff config: %v", err)
	}
}

func seedDraftPO(t *testing.T, pool *pgxpool.Pool, weekStart string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO purchase_orders (week_start, status) VALUES ($1, 'draft') RETURNING id::text
	`, weekStart).Scan(&id)
	if err != nil {
		t.Fatalf("seed draft PO: %v", err)
	}
	return id
}

func poStatus(t *testing.T, pool *pgxpool.Pool, id string) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM purchase_orders WHERE id = $1`, id).Scan(&s); err != nil {
		t.Fatalf("read PO status: %v", err)
	}
	return s
}

func countAlertLog(t *testing.T, pool *pgxpool.Pool, alertType, weekStart string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM alert_log WHERE alert_type = $1 AND week_start = $2`, alertType, weekStart).Scan(&n); err != nil {
		t.Fatalf("count alert_log: %v", err)
	}
	return n
}

func countLowStockLog(t *testing.T, pool *pgxpool.Pool, weekStart string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM low_stock_alert_log WHERE week_start = $1`, weekStart).Scan(&n); err != nil {
		t.Fatalf("count low_stock_alert_log: %v", err)
	}
	return n
}

// seedLowStockItem creates a group (low_threshold=3), an item in it, and a
// purchase event + line item of quantity 2 → classifies as "low"
// (ClassifyStockLevel: 0<qty<=lowT → low).
func seedLowStockItem(t *testing.T, pool *pgxpool.Pool, desc string) {
	t.Helper()
	ctx := context.Background()
	var groupID, vendorID, itemID, eventID string
	if err := pool.QueryRow(ctx, `INSERT INTO item_groups (name, low_threshold, high_threshold) VALUES ('Proteins', 3, 10) RETURNING id::text`).Scan(&groupID); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO vendors (name) VALUES ('Test Vendor') RETURNING id::text`).Scan(&vendorID); err != nil {
		t.Fatalf("seed vendor: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO purchase_items (description, group_id) VALUES ($1, $2) RETURNING id::text`, desc, groupID).Scan(&itemID); err != nil {
		t.Fatalf("seed item: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, total) VALUES ($1, 'tx-1', '2026-01-01', 10) RETURNING id::text`, vendorID).Scan(&eventID); err != nil {
		t.Fatalf("seed event: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO purchase_line_items (purchase_event_id, purchase_item_id, description, quantity, price) VALUES ($1, $2, $3, 2, 5)`, eventID, itemID, desc); err != nil {
		t.Fatalf("seed line item: %v", err)
	}
}

func seedRepurchaseResetConfig(t *testing.T, pool *pgxpool.Pool, dow int, resetTime, tz string, lastReset *time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := UpsertRepurchaseResetConfig(ctx, pool, dow, resetTime, tz); err != nil {
		t.Fatalf("seed repurchase reset config: %v", err)
	}
	if lastReset != nil {
		if _, err := pool.Exec(ctx, `UPDATE repurchase_reset_config SET last_reset_at = $1`, *lastReset); err != nil {
			t.Fatalf("set last_reset_at: %v", err)
		}
	}
}

func lastResetAt(t *testing.T, pool *pgxpool.Pool) *time.Time {
	t.Helper()
	var v *time.Time
	if err := pool.QueryRow(context.Background(), `SELECT last_reset_at FROM repurchase_reset_config LIMIT 1`).Scan(&v); err != nil {
		t.Fatalf("read last_reset_at: %v", err)
	}
	return v
}

// ── FR-19: auto-lock cutoff decision (runCutoffCheck) ────────────────────────
//
// Cutoff configured Wednesday 17:00 Chicago. 2026-01-07 is a Wednesday.
func TestCronCutoffDecision(t *testing.T) {
	pool := requireDB(t)
	loc := chicago(t)

	t.Run("does NOT fire the minute before cutoff", func(t *testing.T) {
		truncateAll(t, pool)
		seedCutoffConfig(t, pool, int(time.Wednesday), "17:00", "America/Chicago")
		poID := seedDraftPO(t, pool, "2026-01-05")
		before := time.Date(2026, 1, 7, 16, 59, 0, 0, loc) // Wed 16:59
		runCutoffCheck(context.Background(), pool, frozenClock(before))
		if got := poStatus(t, pool, poID); got != "draft" {
			t.Fatalf("PO should stay draft at Wed 16:59, got %q", got)
		}
	})

	t.Run("fires the minute after cutoff → auto-locks draft PO", func(t *testing.T) {
		truncateAll(t, pool)
		seedCutoffConfig(t, pool, int(time.Wednesday), "17:00", "America/Chicago")
		poID := seedDraftPO(t, pool, "2026-01-05")
		after := time.Date(2026, 1, 7, 17, 1, 0, 0, loc) // Wed 17:01
		runCutoffCheck(context.Background(), pool, frozenClock(after))
		if got := poStatus(t, pool, poID); got != "locked" {
			t.Fatalf("PO should be locked at Wed 17:01, got %q", got)
		}
	})

	t.Run("does NOT fire on cutoff day before cutoff time (Friday config)", func(t *testing.T) {
		// A second DOW to prove the day-of-week arithmetic is config-driven, not
		// hardcoded to Wednesday: cutoff Friday 17:00, now Friday 16:59 → the most
		// recent occurrence is today's not-yet-reached 17:00 → must NOT lock.
		truncateAll(t, pool)
		seedCutoffConfig(t, pool, int(time.Friday), "17:00", "America/Chicago")
		poID := seedDraftPO(t, pool, "2026-01-05")
		friBefore := time.Date(2026, 1, 9, 16, 59, 0, 0, loc) // Fri 16:59
		runCutoffCheck(context.Background(), pool, frozenClock(friBefore))
		if got := poStatus(t, pool, poID); got != "draft" {
			t.Fatalf("PO should stay draft at Fri 16:59 (this week's Fri cutoff not yet reached), got %q", got)
		}
	})
}

// ── FR-20: cutoff reminder window decision (runReminderCheck) ─────────────────
//
// Cutoff Wednesday 17:00 Chicago → reminder window is [Tue 17:00, Tue 18:00).
func TestCronReminderDecision(t *testing.T) {
	pool := requireDB(t)
	loc := chicago(t)
	ensureAlertQueue(t)
	const weekStart = "2026-01-05" // Monday of the Tue 2026-01-06 fixture week

	t.Run("does NOT fire before the reminder window", func(t *testing.T) {
		truncateAll(t, pool)
		seedCutoffConfig(t, pool, int(time.Wednesday), "17:00", "America/Chicago")
		beforeWindow := time.Date(2026, 1, 6, 16, 0, 0, 0, loc) // Tue 16:00 (<24h boundary)
		runReminderCheck(context.Background(), pool, frozenClock(beforeWindow))
		if n := countAlertLog(t, pool, "cutoff_reminder", weekStart); n != 0 {
			t.Fatalf("no reminder should be logged before window, got %d", n)
		}
	})

	t.Run("fires inside the reminder window → logs cutoff_reminder", func(t *testing.T) {
		truncateAll(t, pool)
		seedCutoffConfig(t, pool, int(time.Wednesday), "17:00", "America/Chicago")
		inWindow := time.Date(2026, 1, 6, 17, 30, 0, 0, loc) // Tue 17:30 (23.5h before)
		runReminderCheck(context.Background(), pool, frozenClock(inWindow))
		if n := countAlertLog(t, pool, "cutoff_reminder", weekStart); n != 1 {
			t.Fatalf("reminder should be logged once inside window, got %d", n)
		}
	})

	t.Run("does NOT fire after the reminder window", func(t *testing.T) {
		truncateAll(t, pool)
		seedCutoffConfig(t, pool, int(time.Wednesday), "17:00", "America/Chicago")
		afterWindow := time.Date(2026, 1, 6, 19, 0, 0, 0, loc) // Tue 19:00 (<23h before)
		runReminderCheck(context.Background(), pool, frozenClock(afterWindow))
		if n := countAlertLog(t, pool, "cutoff_reminder", weekStart); n != 0 {
			t.Fatalf("no reminder should be logged after window, got %d", n)
		}
	})

	t.Run("is idempotent within the window (single send per week)", func(t *testing.T) {
		truncateAll(t, pool)
		seedCutoffConfig(t, pool, int(time.Wednesday), "17:00", "America/Chicago")
		inWindow := time.Date(2026, 1, 6, 17, 30, 0, 0, loc)
		runReminderCheck(context.Background(), pool, frozenClock(inWindow))
		runReminderCheck(context.Background(), pool, frozenClock(inWindow.Add(15*time.Minute)))
		if n := countAlertLog(t, pool, "cutoff_reminder", weekStart); n != 1 {
			t.Fatalf("reminder must be logged exactly once per week, got %d", n)
		}
	})
}

// ── FR-21: low-stock weekly bucketing decision (runLowStockCheck) ─────────────
//
// The time-driven decision here is the week_start idempotency bucket: an item
// alerts once per ISO week, re-alerts in a new week.
func TestCronLowStockDecision(t *testing.T) {
	pool := requireDB(t)
	loc := chicago(t)
	ensureAlertQueue(t)
	const desc = "Salmon Fillet"
	const weekA = "2026-01-05" // Monday of week containing Wed 2026-01-07
	const weekB = "2026-01-12" // Monday of the next week

	t.Run("alerts a low item once in its week", func(t *testing.T) {
		truncateAll(t, pool)
		seedLowStockItem(t, pool, desc)
		nowA := time.Date(2026, 1, 7, 9, 0, 0, 0, loc) // Wed of week A
		runLowStockCheck(context.Background(), pool, frozenClock(nowA))
		if n := countLowStockLog(t, pool, weekA); n != 1 {
			t.Fatalf("low item should be logged once in week A, got %d", n)
		}
	})

	t.Run("does NOT re-alert the same item later the same week", func(t *testing.T) {
		truncateAll(t, pool)
		seedLowStockItem(t, pool, desc)
		nowA1 := time.Date(2026, 1, 7, 9, 0, 0, 0, loc) // Wed
		nowA2 := time.Date(2026, 1, 9, 9, 0, 0, 0, loc) // Fri, same week
		runLowStockCheck(context.Background(), pool, frozenClock(nowA1))
		runLowStockCheck(context.Background(), pool, frozenClock(nowA2))
		if n := countLowStockLog(t, pool, weekA); n != 1 {
			t.Fatalf("low item must be logged once for week A across two same-week runs, got %d", n)
		}
	})

	t.Run("re-alerts the same item in the next week", func(t *testing.T) {
		truncateAll(t, pool)
		seedLowStockItem(t, pool, desc)
		nowA := time.Date(2026, 1, 7, 9, 0, 0, 0, loc)  // week A
		nowB := time.Date(2026, 1, 14, 9, 0, 0, 0, loc) // week B (next week)
		runLowStockCheck(context.Background(), pool, frozenClock(nowA))
		runLowStockCheck(context.Background(), pool, frozenClock(nowB))
		if n := countLowStockLog(t, pool, weekB); n != 1 {
			t.Fatalf("low item should re-alert (log) in week B, got %d", n)
		}
	})
}

// ── FR-22: repurchase-reset decision (runRepurchaseResetCheck) ───────────────
//
// Reset configured Monday 06:00 Chicago. 2026-01-05 is a Monday.
func TestCronRepurchaseResetDecision(t *testing.T) {
	pool := requireDB(t)
	loc := chicago(t)

	t.Run("does NOT fire before reset time", func(t *testing.T) {
		truncateAll(t, pool)
		seedRepurchaseResetConfig(t, pool, int(time.Monday), "06:00", "America/Chicago", nil)
		beforeReset := time.Date(2026, 1, 5, 5, 0, 0, 0, loc) // Mon 05:00
		runRepurchaseResetCheck(context.Background(), pool, frozenClock(beforeReset))
		if v := lastResetAt(t, pool); v != nil {
			t.Fatalf("last_reset_at should stay NULL before reset time, got %v", v)
		}
	})

	t.Run("fires after reset time with no prior reset → bumps last_reset_at", func(t *testing.T) {
		truncateAll(t, pool)
		seedRepurchaseResetConfig(t, pool, int(time.Monday), "06:00", "America/Chicago", nil)
		afterReset := time.Date(2026, 1, 5, 6, 1, 0, 0, loc) // Mon 06:01
		runRepurchaseResetCheck(context.Background(), pool, frozenClock(afterReset))
		v := lastResetAt(t, pool)
		if v == nil {
			t.Fatalf("last_reset_at should be set after reset time")
		}
		wantCandidate := time.Date(2026, 1, 5, 6, 0, 0, 0, loc) // Mon 06:00
		if !v.Equal(wantCandidate) {
			t.Fatalf("last_reset_at = %v, want candidate %v", v.In(loc), wantCandidate)
		}
	})

	t.Run("does NOT re-fire when already reset this week", func(t *testing.T) {
		truncateAll(t, pool)
		already := time.Date(2026, 1, 5, 6, 0, 30, 0, loc) // Mon 06:00:30, after the 06:00 candidate
		seedRepurchaseResetConfig(t, pool, int(time.Monday), "06:00", "America/Chicago", &already)
		later := time.Date(2026, 1, 5, 7, 0, 0, 0, loc) // Mon 07:00, still after candidate
		runRepurchaseResetCheck(context.Background(), pool, frozenClock(later))
		v := lastResetAt(t, pool)
		if v == nil || !v.Equal(already) {
			t.Fatalf("last_reset_at should be unchanged (%v) when already reset this week, got %v", already, v)
		}
	})
}
