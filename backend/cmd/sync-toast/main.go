// sync-toast pulls Toast ItemSelectionDetails.csv for [--from, --to] and
// upserts into menu_items + daily_menu_sales. Reuses internal/toast.RunIngest.
//
// Usage:
//   go run ./cmd/sync-toast/ --from 2026-05-01 --to 2026-05-31
//
// Env (shares LoadConfigFromEnv with cmd/server):
//   DB_URL                  (required)
//   TOAST_SFTP_KEY_PATH     (required, fail-fast — see D-12)
//   TOAST_SFTP_USER         (default YumYumsExportUser)
//   TOAST_SFTP_HOST         (default s-9b0f88558b264dfda...:22)
//   TOAST_EXPORT_ID         (default 113866)
//
// Exit codes:
//   0  ingest cycle completed (per-day errors are logged + skipped per
//      RunIngest contract — they don't cause non-zero exit)
//   1  fatal: bad flags, env, DB connect, key load, or SFTP dial (after 3 retries)
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/toast"
)

func main() {
	fromStr := flag.String("from", "", "Start date YYYY-MM-DD (required)")
	toStr := flag.String("to", "", "End date YYYY-MM-DD (required)")
	flag.Parse()

	if *fromStr == "" || *toStr == "" {
		flag.Usage()
		log.Fatal("--from and --to are required")
	}
	fromDate, err := time.Parse("2006-01-02", *fromStr)
	if err != nil {
		log.Fatalf("--from %q invalid (want YYYY-MM-DD): %v", *fromStr, err)
	}
	toDate, err := time.Parse("2006-01-02", *toStr)
	if err != nil {
		log.Fatalf("--to %q invalid (want YYYY-MM-DD): %v", *toStr, err)
	}
	if toDate.Before(fromDate) {
		log.Fatalf("--to (%s) is before --from (%s)", *toStr, *fromStr)
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is required")
	}

	cfg, err := toast.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("toast config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	cfg.Pool = pool

	result, err := toast.RunIngest(ctx, pool, cfg, fromDate, toDate)
	if err != nil {
		log.Fatalf("ingest: %v", err)
	}

	log.Printf("done. dates=[%s..%s] items_upserted=%d sales_rows_upserted=%d duration=%s",
		fromDate.Format("20060102"), toDate.Format("20060102"),
		result.ItemsUpserted, result.SalesRowsUpserted, result.Duration)
}
