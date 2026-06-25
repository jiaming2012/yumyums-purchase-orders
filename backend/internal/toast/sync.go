package toast

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ErrSFTPMiss signals that the date is not present on the Toast SFTP server
// (or 3 dial attempts failed). This is GRACEFUL — D-05 says SFTP misses are
// expected past Toast's ~14-day retention horizon. Callers should log INFO,
// not increment failure counters.
var ErrSFTPMiss = errors.New("toast sync: date not in SFTP")

// SyncDate fetches one business date's ItemSelectionDetails.csv from SFTP and
// writes it atomically to both the local forensic cache (D-12/D-13) and DO
// Spaces (D-01). The cache file is the durable archive's mirror, not a read
// fall-through — ingest.RunIngest never reads from cache (D-13).
//
// Returns:
//   - (true, nil)            — CSV written to cache + Spaces successfully (sidecar best-effort)
//   - (false, ErrSFTPMiss)   — date not present on SFTP, or 3 dial retries failed (D-05/D-06 graceful)
//   - (false, err)           — Spaces upload failed, or non-SFTP-miss error; caller treats as systemic
//
// Atomicity (D-14): cache file is written FIRST. If the Spaces PutObject then
// fails, the cache file is removed so `cache ⊆ Spaces` invariant holds.
func SyncDate(ctx context.Context, cfg Config, date time.Time) (bool, error) {
	if cfg.SpacesClient == nil || cfg.SpacesBucket == "" {
		return false, fmt.Errorf("toast sync: SpacesClient or SpacesBucket not configured")
	}
	if cfg.CacheDir == "" {
		return false, fmt.Errorf("toast sync: CacheDir not configured")
	}

	dateDir := date.Format("20060102")

	// 1. SFTP fetch with the D-10 5s/15s/30s backoff schedule.
	pkBytes, err := os.ReadFile(cfg.SFTPKeyPath)
	if err != nil {
		return false, fmt.Errorf("read key %q: %w", cfg.SFTPKeyPath, err)
	}
	client, err := dialWithRetry(cfg, string(pkBytes))
	if err != nil {
		// Treat dial failure same as a SFTP miss (D-06 final bullet — graceful skip after retries).
		slog.Warn("toast sync: skip (sftp dial failed)", "date", dateDir, "error", err)
		return false, ErrSFTPMiss
	}
	defer client.Close()

	remotePath := fmt.Sprintf("/%s/%s/ItemSelectionDetails.csv", cfg.ExportID, dateDir)
	r, err := client.Download(remotePath)
	if err != nil {
		// SFTP miss — D-05 graceful skip. Caller logs the INFO line.
		return false, ErrSFTPMiss
	}
	csvBytes, err := io.ReadAll(r)
	r.Close()
	if err != nil {
		return false, fmt.Errorf("toast sync: read %s: %w", remotePath, err)
	}

	// 2. Write cache FIRST (D-14). Mkdir then atomic-ish file write.
	cachePath := CacheCSVPath(cfg.CacheDir, dateDir)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return false, fmt.Errorf("toast sync: mkdir %s: %w", filepath.Dir(cachePath), err)
	}
	if err := os.WriteFile(cachePath, csvBytes, 0o644); err != nil {
		return false, fmt.Errorf("toast sync: write cache %s: %w", cachePath, err)
	}

	// 3. Upload CSV to Spaces. On failure, REMOVE cache to preserve cache ⊆ Spaces.
	csvKey := SpacesCSVKey(dateDir)
	if _, err := cfg.SpacesClient.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.SpacesBucket),
		Key:         aws.String(csvKey),
		Body:        bytes.NewReader(csvBytes),
		ContentType: aws.String("text/csv"),
	}); err != nil {
		_ = os.Remove(cachePath)
		return false, fmt.Errorf("toast sync: put %s: %w", csvKey, err)
	}

	// 4. Sidecar (.meta.json) — best-effort to both cache and Spaces.
	//    Original filename from Toast is "113866_HQ_YYYYMMDD_ItemSelectionDetails.csv"
	//    but we only have the canonical remotePath; reconstruct the SFTP-side name.
	origName := fmt.Sprintf("%s_HQ_%s_ItemSelectionDetails.csv", cfg.ExportID, dateDir)
	meta := NewMetaSidecar(origName, MetaSourceSFTP)
	metaBytes, mErr := meta.Bytes()
	if mErr != nil {
		slog.Warn("toast sync: sidecar marshal failed", "date", dateDir, "error", mErr)
	} else {
		// Write sidecar to cache (best-effort)
		metaCachePath := CacheMetaPath(cfg.CacheDir, dateDir)
		if err := os.WriteFile(metaCachePath, metaBytes, 0o644); err != nil {
			slog.Warn("toast sync: sidecar cache write failed", "date", dateDir, "error", err)
		}
		// Upload sidecar to Spaces (best-effort)
		metaKey := SpacesMetaKey(dateDir)
		if _, err := cfg.SpacesClient.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(cfg.SpacesBucket),
			Key:         aws.String(metaKey),
			Body:        bytes.NewReader(metaBytes),
			ContentType: aws.String("application/json"),
		}); err != nil {
			slog.Warn("toast sync: sidecar put failed", "date", dateDir, "error", err)
		}
	}

	slog.Info("toast sync: wrote", "date", dateDir, "csv_bytes", len(csvBytes), "key", csvKey)
	return true, nil
}

// dialWithRetry — moved verbatim from ingest.go (Phase 22 D-10 5s/15s/30s schedule).
// sync.go is now the only caller; ingest.go reads from Spaces and no longer dials SFTP.
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
			slog.Warn("toast sync: SFTP dial attempt failed", "attempt", i+1, "error", err, "retry_in", wait)
			time.Sleep(wait)
		}
	}
	return nil, fmt.Errorf("sftp dial failed after %d attempts: %w", len(backoffs), lastErr)
}
