// sync-toast pulls Toast ItemSelectionDetails.csv for [--from, --to] from
// DO Spaces and upserts into menu_items + daily_menu_sales. Reuses
// internal/toast.RunIngest.
//
// Phase 22.1 change: this CLI used to dial SFTP directly. It now reads from
// DO Spaces (the durable archive). Live SFTP→Spaces sync is the in-process
// worker's job; historical archive seeding is cmd/migrate-toast-archive's job.
// This CLI's role is "replay ingest for a date range" — useful after a parser
// bug fix when you want to re-upsert without waiting for the next worker tick.
//
// Usage:
//
//	go run ./cmd/sync-toast/ --from 2026-05-01 --to 2026-05-31
//
// Env (shares LoadConfigFromEnv with cmd/server):
//
//	DB_URL                  (required)
//	TOAST_SFTP_KEY_PATH     (required, fail-fast — D-12 preserved)
//	TOAST_SFTP_USER         (default YumYumsExportUser)
//	TOAST_SFTP_HOST         (default s-9b0f88558b264dfda...:22)
//	TOAST_EXPORT_ID         (default 113866)
//	DO_SPACES_KEY           (required — Phase 22.1, RunIngest reads from Spaces)
//	DO_SPACES_SECRET        (required)
//	DO_SPACES_ENDPOINT      (required)
//	DO_SPACES_REGION        (required)
//	DO_SPACES_BUCKET        (required)
//
// Exit codes:
//
//	0  ingest cycle completed (per-day errors are logged + skipped per
//	   RunIngest contract — they don't cause non-zero exit)
//	1  fatal: bad flags, env, DB connect, or systemic Spaces failure
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/photos"
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

	// Phase 22.1: RunIngest reads from Spaces. Build the Spaces client here
	// so the CLI doesn't depend on cmd/server wiring.
	spacesKey := os.Getenv("DO_SPACES_KEY")
	spacesSecret := os.Getenv("DO_SPACES_SECRET")
	spacesEndpoint := os.Getenv("DO_SPACES_ENDPOINT")
	spacesRegion := os.Getenv("DO_SPACES_REGION")
	spacesBucket := os.Getenv("DO_SPACES_BUCKET")
	if spacesKey == "" || spacesSecret == "" || spacesEndpoint == "" || spacesRegion == "" || spacesBucket == "" {
		log.Fatal("DO_SPACES_KEY, DO_SPACES_SECRET, DO_SPACES_ENDPOINT, DO_SPACES_REGION, DO_SPACES_BUCKET must all be set")
	}
	cfg.SpacesClient = photos.NewSpacesClient(photos.SpacesConfig{
		AccessKey: spacesKey,
		SecretKey: spacesSecret,
		Endpoint:  spacesEndpoint,
		Region:    spacesRegion,
		Bucket:    spacesBucket,
	})
	cfg.SpacesBucket = spacesBucket
	cfg.SpacesEndpoint = spacesEndpoint
	cfg.CacheDir = envOr("TOAST_CACHE_DIR", "backend/cache/toast")

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

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
