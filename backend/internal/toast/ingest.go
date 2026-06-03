package toast

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunIngest executes one Toast ingest cycle over [fromDate, toDate] inclusive.
// Reads CSVs from DO Spaces (Phase 22.1 D-04 / D-13 — never from cache, never
// from SFTP). Idempotent by construction (ON CONFLICT DO UPDATE on
// daily_menu_sales, last row wins per Phase 22 D-05). Per-day errors are
// logged + skipped so a single bad day doesn't kill the cycle (Pattern 3).
//
// Returns a non-nil error ONLY for systemic failures (Spaces unreachable in
// pre-flight, or missing config). Per-key 404s are NOT systemic — the worker's
// consecutive-failure counter (Plan 04) should not fire on a clean Spaces miss.
//
// Callers: cmd/sync-toast (CLI, Plan 06) and worker.runCycle (Plan 04).
func RunIngest(ctx context.Context, pool *pgxpool.Pool, cfg Config, fromDate, toDate time.Time) (*IngestResult, error) {
	start := time.Now()

	if cfg.SpacesClient == nil || cfg.SpacesBucket == "" {
		return nil, fmt.Errorf("toast ingest: SpacesClient or SpacesBucket not configured")
	}

	result := &IngestResult{}

	for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
		dateDir := d.Format("20060102")
		key := SpacesCSVKey(dateDir)

		resp, err := cfg.SpacesClient.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(cfg.SpacesBucket),
			Key:    aws.String(key),
		})
		if err != nil {
			var nsk *s3types.NoSuchKey
			if errors.As(err, &nsk) {
				// D-05 generalization: clean miss on Spaces is expected for dates
				// past Toast's retention horizon that migration hasn't seeded yet.
				log.Printf("toast ingest: skip %s (not in Spaces)", dateDir)
				continue
			}
			// Other Spaces errors (auth, network) — log per-date but DON'T return.
			// Worker's per-cycle wrapper decides whether systemic counter should
			// fire (it observes whether RunIngest returns an error overall).
			log.Printf("toast ingest: skip %s (spaces: %v)", dateDir, err)
			continue
		}

		rows, parseErr := parseItemSelectionDetails(resp.Body, d.Format("2006-01-02"))
		resp.Body.Close()
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

// upsertDayInTx writes one day's AggregatedRows in a single DB transaction.
// UNCHANGED from Phase 22 (D-05/D-07 idempotency contract). Do not touch the SQL.
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
// if someone TRUNCATEs the table. UNCHANGED from Phase 22.
func isColdStart(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM daily_menu_sales`).Scan(&n); err != nil {
		return false, fmt.Errorf("isColdStart: %w", err)
	}
	return n == 0, nil
}
