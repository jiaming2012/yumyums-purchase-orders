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

// TestLoadConfigFromEnv_KeyPathSetButMissing_FailsFast pins the EXACT B-146
// prod failure: TOAST_SFTP_KEY_PATH is SET (non-empty) but points at a file
// that does not exist on disk. In prod, TOAST_SFTP_KEY_PATH=./id_rsa resolved
// to /app/id_rsa inside the container, but nothing ever COPYed or bind-mounted
// a key there, so os.Stat failed and the worker-enabled server hard-exited (or,
// pre fail-loud, silently never landed data). This is DISTINCT from the
// empty-string case (TestLoadConfigFromEnv_IntervalNonZero_RequiresKeyPath):
// it exercises the os.Stat guard, not the empty-check guard. The
// toast-ingest-resurrection fix supplies a real file at this path via a
// docker-compose bind-mount so this branch stops firing in prod.
func TestLoadConfigFromEnv_KeyPathSetButMissing_FailsFast(t *testing.T) {
	dir := t.TempDir()
	absent := filepath.Join(dir, "id_rsa") // deliberately never created
	t.Setenv("TOAST_SFTP_KEY_PATH", absent)
	t.Setenv("TOAST_SYNC_INTERVAL", "12h") // worker enabled → key must be readable

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected fail-fast error when key path is set but the file is absent (B-146 prod condition), got nil")
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

// TestLoadConfigFromEnv_KeyPathIsDirectory_FailsFast pins the EXACT 2026-09-02
// prod failure that the "absent file" test above does NOT cover: the key path
// EXISTS but is a DIRECTORY. docker-compose bind-mounts `./id_rsa:/app/id_rsa:ro`;
// when no `id_rsa` FILE is placed beside the compose, Docker creates an empty
// DIRECTORY at the source and mounts it. os.Stat SUCCEEDS on a directory, so the
// old guard passed, the worker started, and every SyncDate then died at runtime
// ("read /app/id_rsa: is a directory") while /health still reported toast_sync:ok.
// The boot guard must reject a non-regular key and fail fast.
func TestLoadConfigFromEnv_KeyPathIsDirectory_FailsFast(t *testing.T) {
	dir := t.TempDir()
	keyAsDir := filepath.Join(dir, "id_rsa")
	if err := os.Mkdir(keyAsDir, 0o755); err != nil {
		t.Fatalf("mkdir key-as-dir: %v", err)
	}
	t.Setenv("TOAST_SFTP_KEY_PATH", keyAsDir)
	t.Setenv("TOAST_SYNC_INTERVAL", "12h") // worker enabled → key must be a readable file

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected fail-fast error when key path is a directory (2026-09-02 prod condition), got nil")
	}
}

// TestLoadConfigFromEnv_KeyPathEmpty_FailsFast: a regular file that is empty is
// not a usable key either — os.Stat + IsRegular both pass, so the guard must
// also reject a zero-byte key rather than let the worker start and fail per-cycle.
func TestLoadConfigFromEnv_KeyPathEmpty_FailsFast(t *testing.T) {
	dir := t.TempDir()
	emptyKey := filepath.Join(dir, "id_rsa")
	if err := os.WriteFile(emptyKey, nil, 0o600); err != nil {
		t.Fatalf("write empty key: %v", err)
	}
	t.Setenv("TOAST_SFTP_KEY_PATH", emptyKey)
	t.Setenv("TOAST_SYNC_INTERVAL", "12h")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected fail-fast error when key file is empty, got nil")
	}
}
