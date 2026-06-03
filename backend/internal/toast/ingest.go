package toast

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunIngest executes one Toast ingest cycle over [fromDate, toDate] inclusive.
// Idempotent by construction (ON CONFLICT DO UPDATE on daily_menu_sales, last
// row wins per D-05). Per-day errors are logged + skipped so a single bad day
// doesn't kill the cycle.
//
// Callers: cmd/sync-toast (CLI, Plan 04) and worker.runCycle (Plan 04).
func RunIngest(ctx context.Context, pool *pgxpool.Pool, cfg Config, fromDate, toDate time.Time) (*IngestResult, error) {
	start := time.Now()

	pkBytes, err := os.ReadFile(cfg.SFTPKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read key %q: %w", cfg.SFTPKeyPath, err)
	}

	client, err := dialWithRetry(cfg, string(pkBytes))
	if err != nil {
		return nil, fmt.Errorf("sftp dial: %w", err)
	}
	defer client.Close()

	result := &IngestResult{}

	for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
		dateDir := d.Format("20060102")
		remotePath := fmt.Sprintf("/%s/%s/ItemSelectionDetails.csv", cfg.ExportID, dateDir)

		f, err := client.Download(remotePath)
		if err != nil {
			// Closed-day directories don't exist on the SFTP server — that's
			// expected; log at debug-ish level and continue.
			log.Printf("toast ingest: skip %s (download: %v)", dateDir, err)
			continue
		}

		rows, parseErr := parseItemSelectionDetails(f, d.Format("2006-01-02"))
		f.Close()
		if parseErr != nil {
			log.Printf("toast ingest: skip %s (parse: %v)", dateDir, parseErr)
			continue
		}

		items, sales, err := upsertDayInTx(ctx, pool, rows)
		if err != nil {
			log.Printf("toast ingest: skip %s (db: %v)", dateDir, err)
			continue
		}

		result.Dates = append(result.Dates, dateDir)
		result.ItemsUpserted += items
		result.SalesRowsUpserted += sales
	}

	result.Duration = time.Since(start)
	return result, nil
}

// dialWithRetry wraps toast.New with the D-10 backoff schedule (5s / 15s / 30s).
// Each attempt uses a 30s connect timeout (mirrors sales-processor).
func dialWithRetry(cfg Config, pemKey string) (*Client, error) {
	backoffs := []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}
	var lastErr error
	for i, wait := range backoffs {
		client, err := New(SFTPConfig{
			Username:   cfg.SFTPUser,
			PrivateKey: pemKey,
			Server:     cfg.SFTPHost,
			Timeout:    30 * time.Second,
		})
		if err == nil {
			return client, nil
		}
		lastErr = err
		if i < len(backoffs)-1 {
			log.Printf("toast ingest: SFTP dial attempt %d failed: %v — retrying in %s", i+1, err, wait)
			time.Sleep(wait)
		}
	}
	return nil, fmt.Errorf("sftp dial failed after %d attempts: %w", len(backoffs), lastErr)
}

// upsertDayInTx writes one day's AggregatedRows in a single DB transaction.
// Returns (menu_items_upserted, daily_menu_sales_upserted) counts.
//
// menu_items: INSERT ... ON CONFLICT (master_id) DO UPDATE SET name/menu/menu_group/menu_subgroup
//
//	plus last_seen = GREATEST(menu_items.last_seen, EXCLUDED.last_seen).
//
// daily_menu_sales: INSERT ... ON CONFLICT (menu_item_id, business_date) DO UPDATE SET
//
//	units_sold/gross_amount/updated_at — last-pull wins (D-05).
func upsertDayInTx(ctx context.Context, pool *pgxpool.Pool, rows []AggregatedRow) (int, int, error) {
	if len(rows) == 0 {
		return 0, 0, nil
	}
	dbTx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin: %w", err)
	}
	defer dbTx.Rollback(ctx) //nolint:errcheck

	var itemsUpserted, salesUpserted int
	for _, r := range rows {
		// last_seen bump only happens via GREATEST() in the SQL — see CONTEXT D-07
		// "Claude's Discretion". r.UnitsSold is always > 0 here because the parser
		// excludes voided rows entirely (D-06).
		lastSeenArg := r.BusinessDate

		var menuItemID string
		err := dbTx.QueryRow(ctx, `
			INSERT INTO menu_items (master_id, name, menu, menu_group, menu_subgroup, last_seen)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (master_id) DO UPDATE SET
				name          = EXCLUDED.name,
				menu          = EXCLUDED.menu,
				menu_group    = EXCLUDED.menu_group,
				menu_subgroup = EXCLUDED.menu_subgroup,
				last_seen     = GREATEST(menu_items.last_seen, EXCLUDED.last_seen)
			RETURNING id`,
			r.MasterID, r.Name, r.Menu, r.MenuGroup, r.MenuSubgroup, lastSeenArg,
		).Scan(&menuItemID)
		if err != nil {
			return 0, 0, fmt.Errorf("upsert menu_item %q: %w", r.MasterID, err)
		}
		itemsUpserted++

		_, err = dbTx.Exec(ctx, `
			INSERT INTO daily_menu_sales (menu_item_id, business_date, units_sold, gross_amount, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (menu_item_id, business_date) DO UPDATE SET
				units_sold   = EXCLUDED.units_sold,
				gross_amount = EXCLUDED.gross_amount,
				updated_at   = now()`,
			menuItemID, r.BusinessDate, r.UnitsSold, r.GrossAmount,
		)
		if err != nil {
			return 0, 0, fmt.Errorf("upsert daily_menu_sales for %q on %s: %w", r.MasterID, r.BusinessDate, err)
		}
		salesUpserted++
	}

	if err := dbTx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}
	return itemsUpserted, salesUpserted, nil
}

// isColdStart detects whether this is the first ever ingest by checking
// daily_menu_sales emptiness (D-02). No bespoke flag or env var — self-healing
// if someone TRUNCATEs the table.
func isColdStart(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM daily_menu_sales`).Scan(&n); err != nil {
		return false, fmt.Errorf("isColdStart: %w", err)
	}
	return n == 0, nil
}
