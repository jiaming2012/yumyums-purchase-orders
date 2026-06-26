package toast

import (
	"fmt"
	"os"
	"time"
)

// LoadConfigFromEnv reads all TOAST_* env vars and returns a populated Config
// (Pool is NOT set — the caller injects pgxpool.Pool after construction).
//
// Fails fast (D-12) if TOAST_SFTP_KEY_PATH is unset or points at an unreadable
// path WHEN THE WORKER IS ENABLED. When TOAST_SYNC_INTERVAL=0, the in-process
// worker is disabled and the key path check is skipped — disabling shouldn't
// require credentials.
//
// Defaults (per "Claude's Discretion" in 22-CONTEXT.md):
//
//	TOAST_SFTP_USER       = "YumYumsExportUser"
//	TOAST_SFTP_HOST       = "s-9b0f88558b264dfda.server.transfer.us-east-1.amazonaws.com:22"
//	TOAST_EXPORT_ID       = "113866"
//	TOAST_SYNC_INTERVAL   = "12h"  (time.ParseDuration; "0" disables the worker)
//	SyncWindowDays        = 7      (re-pull last 7 days per tick — D-04)
//	BackfillDays          = 90     (cold-start backfill — D-01)
func LoadConfigFromEnv() (Config, error) {
	interval := 12 * time.Hour
	if s := os.Getenv("TOAST_SYNC_INTERVAL"); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return Config{}, fmt.Errorf("TOAST_SYNC_INTERVAL %q: %w", s, err)
		}
		interval = d
	}

	// Worker disabled: skip key path validation. Caller is expected to
	// branch on cfg.Interval == 0 (see cmd/server/main.go).
	if interval == 0 {
		return Config{Interval: 0}, nil
	}

	keyPath := os.Getenv("TOAST_SFTP_KEY_PATH")
	if keyPath == "" {
		return Config{}, fmt.Errorf("TOAST_SFTP_KEY_PATH is required (no default — see D-12)")
	}
	if _, err := os.Stat(keyPath); err != nil {
		return Config{}, fmt.Errorf("TOAST_SFTP_KEY_PATH=%q is not readable: %w", keyPath, err)
	}

	cfg := Config{
		SFTPHost:       envOr("TOAST_SFTP_HOST", "s-9b0f88558b264dfda.server.transfer.us-east-1.amazonaws.com:22"),
		SFTPUser:       envOr("TOAST_SFTP_USER", "YumYumsExportUser"),
		SFTPKeyPath:    keyPath,
		ExportID:       envOr("TOAST_EXPORT_ID", "113866"),
		Interval:       interval,
		SyncWindowDays: 7,
		BackfillDays:   90,
	}
	return cfg, nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
