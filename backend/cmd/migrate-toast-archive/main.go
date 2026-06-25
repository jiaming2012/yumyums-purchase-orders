// migrate-toast-archive uploads the sales-processor local Toast archive to DO Spaces.
//
// This is a ONE-SHOT tool. The historical layer (typically the last 90+ days) lives
// in sales-processor's filesystem at /Users/jamal/projects/yumyums/sales-processor/output/toast_reports/.
// Phase 22.1 moves HQ to a Spaces-first ingest model — this CLI seeds the bucket so
// the cold-start worker tick has data to ingest.
//
// Usage:
//
//	go run ./cmd/migrate-toast-archive/ --source /Users/jamal/projects/yumyums/sales-processor/output/toast_reports
//	go run ./cmd/migrate-toast-archive/ --source /path/to/archive --overwrite
//
// Env (no defaults; all DO_SPACES_* required):
//
//	DO_SPACES_KEY       (required)
//	DO_SPACES_SECRET    (required)
//	DO_SPACES_ENDPOINT  (required, e.g. https://nyc3.digitaloceanspaces.com)
//	DO_SPACES_REGION    (required, e.g. nyc3)
//	DO_SPACES_BUCKET    (required, e.g. hq.yumyums)
//
// Per Phase 22.1 D-10: NO DB writes. The next worker tick parses the seeded
// CSVs into menu_items + daily_menu_sales via the same RunIngest path used
// for live data. Single ingest code path.
//
// Exit codes:
//
//	0  walk completed (per-date skips are not errors)
//	1  bad flags, missing env, or systemic Spaces failure (HeadObject or PutObject error other than NotFound)
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/yumyums/hq/internal/photos"
	"github.com/yumyums/hq/internal/toast"
)

var dateDirRe = regexp.MustCompile(`^\d{8}$`)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	sourceFlag := flag.String("source", "", "Path to sales-processor archive root (required, no default)")
	overwriteFlag := flag.Bool("overwrite", false, "Re-upload even if the Spaces key already exists (D-09)")
	flag.Parse()

	if *sourceFlag == "" {
		flag.Usage()
		slog.Error("--source is required (no default -- see D-08)")
		os.Exit(1)
	}

	// Resolve to an absolute path and verify it's a directory.
	sourceRoot, err := filepath.Abs(*sourceFlag)
	if err != nil {
		slog.Error("cannot resolve --source", "source", *sourceFlag, "error", err)
		os.Exit(1)
	}
	st, err := os.Stat(sourceRoot)
	if err != nil {
		slog.Error("stat --source failed", "source", sourceRoot, "error", err)
		os.Exit(1)
	}
	if !st.IsDir() {
		slog.Error("--source is not a directory", "source", sourceRoot)
		os.Exit(1)
	}

	// Spaces config — all five env vars required, no defaults.
	key := os.Getenv("DO_SPACES_KEY")
	secret := os.Getenv("DO_SPACES_SECRET")
	endpoint := os.Getenv("DO_SPACES_ENDPOINT")
	region := os.Getenv("DO_SPACES_REGION")
	bucket := os.Getenv("DO_SPACES_BUCKET")
	if key == "" || secret == "" || endpoint == "" || region == "" || bucket == "" {
		slog.Error("DO_SPACES_KEY, DO_SPACES_SECRET, DO_SPACES_ENDPOINT, DO_SPACES_REGION, DO_SPACES_BUCKET must all be set")
		os.Exit(1)
	}

	// Reuse the existing client factory (includes UsePathStyle: true — mandatory for hq.yumyums).
	client := photos.NewSpacesClient(photos.SpacesConfig{
		AccessKey: key,
		SecretKey: secret,
		Endpoint:  endpoint,
		Region:    region,
		Bucket:    bucket,
	})

	ctx := context.Background()

	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		slog.Error("read source dir failed", "dir", sourceRoot, "error", err)
		os.Exit(1)
	}

	uploaded, skipped, missing, warned := 0, 0, 0, 0

	for _, e := range entries {
		name := e.Name()

		if !e.IsDir() {
			// D-11: non-directory entries inside the archive root are unexpected.
			slog.Warn("skipping non-directory entry at archive root", "name", name)
			warned++
			continue
		}

		if !dateDirRe.MatchString(name) {
			// D-11: directories whose name is not YYYYMMDD.
			slog.Warn("skipping non-date directory (expected YYYYMMDD)", "name", name)
			warned++
			continue
		}

		dateDir := name
		csvPath := filepath.Join(sourceRoot, dateDir, "ItemSelectionDetails.csv")

		if _, err := os.Stat(csvPath); err != nil {
			// D-11: empty dates skip SILENTLY (no WARN).
			missing++
			continue
		}

		csvKey := toast.SpacesCSVKey(dateDir)

		if !*overwriteFlag {
			exists, err := toast.SpacesKeyExists(ctx, client, bucket, csvKey)
			if err != nil {
				slog.Error("HeadObject failed", "key", csvKey, "error", err)
				os.Exit(1)
			}
			if exists {
				slog.Info("skip date (already in Spaces)", "date", dateDir)
				skipped++
				continue
			}
		}

		data, err := os.ReadFile(csvPath)
		if err != nil {
			slog.Warn("read CSV failed, skipping", "path", csvPath, "error", err)
			warned++
			continue
		}

		// Upload CSV.
		if _, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(csvKey),
			Body:        bytes.NewReader(data),
			ContentType: aws.String("text/csv"),
		}); err != nil {
			slog.Error("PutObject failed", "key", csvKey, "error", err)
			os.Exit(1)
		}

		// Sidecar (D-03 / D-10 — source: "migration"). Original filename is
		// not preserved on disk in the sales-processor archive; record the
		// canonical name + source so operators can still distinguish live
		// vs migrated objects after the fact.
		meta := toast.NewMetaSidecar("ItemSelectionDetails.csv", toast.MetaSourceMigration)
		metaBytes, mErr := meta.Bytes()
		if mErr != nil {
			slog.Warn("marshal sidecar failed, continuing", "date", dateDir, "error", mErr)
		} else {
			metaKey := toast.SpacesMetaKey(dateDir)
			if _, err := client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:      aws.String(bucket),
				Key:         aws.String(metaKey),
				Body:        bytes.NewReader(metaBytes),
				ContentType: aws.String("application/json"),
			}); err != nil {
				slog.Warn("PutObject sidecar failed (CSV uploaded; best-effort)", "key", metaKey, "error", err)
			}
		}

		uploaded++
		slog.Info("uploaded CSV", "source", csvPath, "key", csvKey, "bytes", len(data))
	}

	slog.Info("done", "uploaded", uploaded, "skipped", skipped, "missing", missing, "warned", warned)
	fmt.Println("ok")
}
