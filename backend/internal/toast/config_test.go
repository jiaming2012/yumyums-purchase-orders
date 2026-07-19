package toast

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfigFromEnv_IntervalZero_DoesNotRequireKeyPath proves the bug
// fix for "backend hard-exits on missing TOAST_SFTP_KEY_PATH even when
// TOAST_SYNC_INTERVAL=0". Setting INTERVAL=0 is the documented kill switch
// for the in-process worker; it should not require an SFTP key file to be
// present on disk, because the worker won't dial SFTP at all.
func TestLoadConfigFromEnv_IntervalZero_DoesNotRequireKeyPath(t *testing.T) {
	t.Setenv("TOAST_SFTP_KEY_PATH", "")
	t.Setenv("TOAST_SYNC_INTERVAL", "0")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("expected no error with INTERVAL=0 and empty key path, got: %v", err)
	}
	if cfg.Interval != 0 {
		t.Fatalf("expected Interval=0, got %v", cfg.Interval)
	}
}

// TestLoadConfigFromEnv_IntervalNonZero_RequiresKeyPath preserves the
// fail-fast contract (D-12) for the worker-enabled case.
func TestLoadConfigFromEnv_IntervalNonZero_RequiresKeyPath(t *testing.T) {
	t.Setenv("TOAST_SFTP_KEY_PATH", "")
	t.Setenv("TOAST_SYNC_INTERVAL", "12h")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected fail-fast error when key path missing and worker enabled, got nil")
	}
}

// TestLoadConfigFromEnv_KeyPathReadable_ReturnsFullConfig is the happy path:
// worker enabled (default interval), key file exists and is readable.
func TestLoadConfigFromEnv_KeyPathReadable_ReturnsFullConfig(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "id_rsa")
	if err := os.WriteFile(keyFile, []byte("stub-key"), 0600); err != nil {
		t.Fatalf("write stub key: %v", err)
	}
	t.Setenv("TOAST_SFTP_KEY_PATH", keyFile)
	t.Setenv("TOAST_SYNC_INTERVAL", "")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.SFTPKeyPath != keyFile {
		t.Fatalf("expected key path %q, got %q", keyFile, cfg.SFTPKeyPath)
	}
	if cfg.Interval == 0 {
		t.Fatalf("expected non-zero default Interval, got 0")
	}
}
