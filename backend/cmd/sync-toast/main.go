// sync-toast pulls Toast ItemSelectionDetails.csv for [--from, --to] from
// the object-storage archive (Backblaze B2) and upserts into menu_items + daily_menu_sales. Reuses
// internal/toast.RunIngest.
//
// Phase 22.1 change: this CLI used to dial SFTP directly. It now reads from
// object storage (the durable archive). Live SFTP→Spaces sync is the in-process
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
//	STORAGE_KEY           (required — Phase 22.1, RunIngest reads from Spaces)
//	STORAGE_SECRET        (required)
//	STORAGE_ENDPOINT      (required)
//	STORAGE_REGION        (required)
//	STORAGE_BUCKET        (required)
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
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/photos"
	"github.com/yumyums/hq/internal/toast"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	fromStr := flag.String("from", "", "Start date YYYY-MM-DD (required)")
	toStr := flag.String("to", "", "End date YYYY-MM-DD (required)")
	flag.Parse()

	if *fromStr == "" || *toStr == "" {
		flag.Usage()
		slog.Error("--from and --to are required")
		os.Exit(1)
	}
	fromDate, err := time.Parse("2006-01-02", *fromStr)
	if err != nil {
		slog.Error("invalid --from date (want YYYY-MM-DD)", "from", *fromStr, "error", err)
		os.Exit(1)
	}
	toDate, err := time.Parse("2006-01-02", *toStr)
	if err != nil {
		slog.Error("invalid --to date (want YYYY-MM-DD)", "to", *toStr, "error", err)
		os.Exit(1)
	}
	if toDate.Before(fromDate) {
		slog.Error("--to is before --from", "to", *toStr, "from", *fromStr)
		os.Exit(1)
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		slog.Error("DB_URL is required")
		os.Exit(1)
	}

	cfg, err := toast.LoadConfigFromEnv()
	if err != nil {
		slog.Error("toast config failed", "error", err)
		os.Exit(1)
	}

	// Phase 22.1: RunIngest reads from Spaces. Build the Spaces client here
	// so the CLI doesn't depend on cmd/server wiring.
	spacesKey := os.Getenv("STORAGE_KEY")
	spacesSecret := os.Getenv("STORAGE_SECRET")
	spacesEndpoint := os.Getenv("STORAGE_ENDPOINT")
	spacesRegion := os.Getenv("STORAGE_REGION")
	spacesBucket := os.Getenv("STORAGE_BUCKET")
	if spacesKey == "" || spacesSecret == "" || spacesEndpoint == "" || spacesRegion == "" || spacesBucket == "" {
		slog.Error("STORAGE_KEY, STORAGE_SECRET, STORAGE_ENDPOINT, STORAGE_REGION, STORAGE_BUCKET must all be set")
		os.Exit(1)
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
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	cfg.Pool = pool

	result, err := toast.RunIngest(ctx, pool, cfg, fromDate, toDate)
	if err != nil {
		slog.Error("ingest failed", "error", err)
		os.Exit(1)
	}

	slog.Info("done",
		"from", fromDate.Format("20060102"),
		"to", toDate.Format("20060102"),
		"items_upserted", result.ItemsUpserted,
		"sales_rows_upserted", result.SalesRowsUpserted,
		"duration", result.Duration)
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
