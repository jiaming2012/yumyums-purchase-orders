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

// ErrSFTPMiss signals that the date is not present on the Toast SFTP server.
// This is GRACEFUL — D-05 says SFTP misses are expected past Toast's ~14-day
// retention horizon. Callers should log INFO, not increment failure counters.
//
// B-146: this used to ALSO cover "3 dial attempts failed", which made a dead
// transport indistinguishable from an absent date. Dial/auth failure now
// returns ErrSFTPUnavailable so the worker can fail loud.
var ErrSFTPMiss = errors.New("toast sync: date not in SFTP")

// ErrSFTPUnavailable signals that the SFTP endpoint could not be opened or
// authenticated (dial failed after retries, or auth was rejected). This is the
// B-146 silent-death class: unlike ErrSFTPMiss it is NOT graceful — the whole
// pipeline is dead, so the worker must surface it in /api/v1/health and fire a
// Cliq alert immediately, never log-and-continue.
var ErrSFTPUnavailable = errors.New("toast sync: sftp dial/auth failed")

// SyncDate fetches one business date's ItemSelectionDetails.csv from SFTP and
// writes it atomically to both the local forensic cache (D-12/D-13) and DO
// Spaces (D-01). The cache file is the durable archive's mirror, not a read
// fall-through — ingest.RunIngest never reads from cache (D-13).
//
// Returns:
//   - (true, nil)                  — CSV written to cache + Spaces successfully (sidecar best-effort)
//   - (false, ErrSFTPMiss)         — date not present on SFTP (D-05 graceful skip)
//   - (false, ErrSFTPUnavailable)  — SFTP dial/auth failed (B-146 loud path — health + alert)
//   - (false, err)                 — Spaces upload failed, or other error; caller treats as systemic
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
		// A key that cannot be read is a DEAD transport — no data can ever land,
		// the same class as a dial/auth failure. Classify it as ErrSFTPUnavailable
		// so the worker fails loud (health failing + Cliq alert) instead of the
		// plain error that runCycle's default branch mistakes for a live-transport
		// Spaces hiccup (reachedSFTP=true → RecordSuccess). This was the 2026-09-02
		// silent death: /app/id_rsa bind-mounted as an empty directory, every read
		// "is a directory", yet /health stayed toast_sync:ok.
		slog.Error("toast sync: SFTP key unreadable", "path", cfg.SFTPKeyPath, "error", err)
		return false, fmt.Errorf("%w: read key %q: %v", ErrSFTPUnavailable, cfg.SFTPKeyPath, err)
	}
	dial := cfg.Dialer
	if dial == nil {
		dial = dialWithRetry
	}
	client, err := dial(cfg, string(pkBytes))
	if err != nil {
		// B-146: dial/auth failure is NOT a graceful miss — the transport is
		// dead. Return ErrSFTPUnavailable so the worker fails loud (health +
		// Cliq alert). Previously this was downgraded to ErrSFTPMiss and went
		// silent, which is exactly how prod's sync died invisibly for weeks.
		slog.Error("toast sync: SFTP unavailable (dial/auth failed)", "date", dateDir, "error", err)
		return false, fmt.Errorf("%w: %v", ErrSFTPUnavailable, err)
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
