package recipes

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/alerts"
)

// timeNow is the package-level clock. Tests override via:
//
//	timeNow = func() time.Time { return fixed }
//	defer func() { timeNow = time.Now }()
//
// Tests that exercise runDriftWeek directly (passing a YYYY-MM-DD literal)
// do NOT need to touch timeNow.
var timeNow = time.Now

// StartDriftScheduler launches the weekly drift-check goroutine. The ticker
// fires every 15 minutes; runDriftTick gates on Monday 09:00–09:14
// America/Chicago so the tick only matches once per week.
//
// Must be called AFTER SetAlertQueue if Cliq delivery is desired (nil-safe
// otherwise — banner still renders on the next /drift fetch).
func StartDriftScheduler(ctx context.Context, pool *pgxpool.Pool) {
	log.Println("recipes drift scheduler: starting (15m tick, fires Monday 09:00 America/Chicago)")
	go func() {
		// Boot-time tick — recovers a missed Monday if the server happened to be
		// down at 09:00 (the tick is idempotent so a re-fire is a no-op).
		runDriftTick(ctx, pool)
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("recipes drift scheduler: shutting down")
				return
			case <-ticker.C:
				runDriftTick(ctx, pool)
			}
		}
	}()
}

// runDriftTick gates on the Monday 09:00 Chicago window and delegates to
// runDriftWeek. Outside that window, returns silently.
func runDriftTick(ctx context.Context, pool *pgxpool.Pool) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		log.Printf("recipes drift: TZ load: %v", err)
		return
	}
	now := timeNow().In(loc)
	if now.Weekday() != time.Monday || now.Hour() != 9 || now.Minute() >= 15 {
		return
	}
	weekStartStr := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Format("2006-01-02")
	runDriftWeek(ctx, pool, weekStartStr)
}

// runDriftWeek runs the drift computation + write + alert for the given
// week_start (YYYY-MM-DD literal in America/Chicago calendar). Tests call this
// directly to bypass the time gate; production reaches it through runDriftTick.
//
// Signature note: weekStart is a string literal rather than time.Time because
// the only consumer (`week_start.Format("2006-01-02")` previously) always
// converted to string anyway. Removing the time.Time eliminates the
// production-vs-test TZ ambiguity.
func runDriftWeek(ctx context.Context, pool *pgxpool.Pool, weekStart string) {
	// Idempotency check — exit early when this week's row already exists.
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM drift_check_results WHERE week_start = $1)`,
		weekStart,
	).Scan(&exists)
	if err != nil {
		log.Printf("recipes drift: idempotency check: %v", err)
		return
	}
	if exists {
		return
	}

	// Compute drift against the PRIOR week. Parse weekStart back to a time
	// for date math, then re-format both endpoints as YYYY-MM-DD strings.
	ws, err := time.Parse("2006-01-02", weekStart)
	if err != nil {
		log.Printf("recipes drift: weekStart parse %q: %v", weekStart, err)
		return
	}
	priorFrom := ws.AddDate(0, 0, -7).Format("2006-01-02")
	priorTo := ws.AddDate(0, 0, -1).Format("2006-01-02")

	// Ingest-stale guard (Pitfall 3): if fewer than 5 of the prior 7 days
	// have daily_menu_sales rows, skip — Toast ingest is presumed stalled.
	// We do NOT insert a drift row in this case (no banner, no Cliq).
	var distinctDays int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT business_date) FROM daily_menu_sales
		 WHERE business_date BETWEEN $1 AND $2`,
		priorFrom, priorTo,
	).Scan(&distinctDays)
	if err != nil {
		log.Printf("recipes drift: ingest-stale check: %v", err)
		return
	}
	if distinctDays < 5 {
		log.Printf("recipes drift: ingest stale (only %d of 7 days have sales data) — skipping week %s",
			distinctDays, weekStart)
		return
	}

	result, err := computeDrift(ctx, pool, priorFrom, priorTo)
	if err != nil {
		log.Printf("recipes drift: compute: %v", err)
		return
	}
	result.WeekStart = weekStart

	payload, err := json.Marshal(result)
	if err != nil {
		log.Printf("recipes drift: marshal: %v", err)
		return
	}

	// INSERT preserves idempotency even under concurrent ticks (server restart
	// straddling 09:00) via the week_start PRIMARY KEY + ON CONFLICT DO NOTHING.
	_, err = pool.Exec(ctx,
		`INSERT INTO drift_check_results (week_start, payload, created_at)
		 VALUES ($1, $2, now()) ON CONFLICT (week_start) DO NOTHING`,
		weekStart, payload,
	)
	if err != nil {
		log.Printf("recipes drift: insert: %v", err)
		return
	}

	if result.HasDrift() {
		// Dispatch through alertSink (not alertQueue) — Plan 01's indirection
		// makes the path test-overridable. nil-check covers the case where
		// SetAlertQueue was never called (e.g., dev box without Cliq creds).
		if alertSink != nil {
			baseURL := os.Getenv("BASE_URL")
			if baseURL == "" {
				baseURL = "https://hq.yumyums.kitchen"
			}
			msg := formatCliqMessage(result, weekStart, baseURL)
			alertSink.Enqueue(alerts.Alert{
				Channel: alerts.ChannelZohoCliq,
				Subject: "Recipes drift",
				Message: msg,
			})
		} else {
			log.Println("recipes drift scheduler: alertSink not configured — banner only")
		}
	}
	log.Printf("recipes drift scheduler: week %s — %d flagged", weekStart, result.TotalFlagged())
}
