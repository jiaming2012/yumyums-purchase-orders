// backfill-receipts runs ONE Mercury receipt ingest cycle with an explicit
// lookback, without starting the server. Built for the B-145 recovery
// (decision 156: backfill to 2026-03-01) and kept because "replay the receipt
// ingest for a window" is the same recurring need cmd/sync-toast serves for
// Toast.
//
// Unlike cmd/server it does NOT migrate the database, seed fixtures, or start
// any background worker — one cycle, then exit. Safe to point at production
// from a working tree whose migrations are ahead of what prod runs.
//
// Usage:
//
//	go run ./cmd/backfill-receipts/ --lookback-days 160
//
// Env (same names cmd/server reads):
//
//	DB_URL              (required — include search_path for the target schema)
//	MERCURY_API_KEY     (required)
//	ANTHROPIC_API_KEY   (required — receipt parsing)
//	STORAGE_KEY, STORAGE_SECRET, STORAGE_BUCKET, STORAGE_REGION, STORAGE_ENDPOINT
//	                    (required — receipt files are uploaded to object storage)
//
// Exit codes: 0 cycle completed; 1 bad flags/env or cycle error.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/photos"
	"github.com/yumyums/hq/internal/receipt"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	lookback := flag.Int("lookback-days", 0, "Days back from now to ingest (required, > 0)")
	flag.Parse()
	if *lookback <= 0 {
		flag.Usage()
		slog.Error("--lookback-days is required and must be > 0")
		os.Exit(1)
	}

	dbURL := os.Getenv("DB_URL")
	mercuryKey := os.Getenv("MERCURY_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if dbURL == "" || mercuryKey == "" || anthropicKey == "" {
		slog.Error("DB_URL, MERCURY_API_KEY and ANTHROPIC_API_KEY are all required")
		os.Exit(1)
	}

	spacesRegion := os.Getenv("STORAGE_REGION")
	spacesEndpoint := os.Getenv("STORAGE_ENDPOINT")
	spacesBucket := os.Getenv("STORAGE_BUCKET")
	spacesKey := os.Getenv("STORAGE_KEY")
	spacesSecret := os.Getenv("STORAGE_SECRET")
	if spacesKey == "" || spacesSecret == "" || spacesBucket == "" || spacesEndpoint == "" {
		slog.Error("STORAGE_KEY, STORAGE_SECRET, STORAGE_BUCKET and STORAGE_ENDPOINT are required")
		os.Exit(1)
	}
	presigner, err := photos.NewSpacesPresigner(photos.SpacesConfig{
		AccessKey: spacesKey,
		SecretKey: spacesSecret,
		Endpoint:  spacesEndpoint,
		Region:    spacesRegion,
		Bucket:    spacesBucket,
	})
	if err != nil {
		slog.Error("spaces presigner init failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	result, err := receipt.RunIngestCycle(ctx, receipt.WorkerConfig{
		MercuryAPIKey:   mercuryKey,
		AnthropicAPIKey: anthropicKey,
		Pool:            pool,
		SpacesPresigner: presigner,
		SpacesEndpoint:  spacesEndpoint,
		SpacesBucket:    spacesBucket,
		LookbackDays:    *lookback,
	})
	if err != nil {
		slog.Error("ingest cycle failed", "error", err)
		os.Exit(1)
	}

	slog.Info("done",
		"lookback_days", *lookback,
		"processed", result.Processed,
		"auto_created", result.AutoCreated,
		"pending_review", result.PendingReview,
		"cached", result.Cached)
}
