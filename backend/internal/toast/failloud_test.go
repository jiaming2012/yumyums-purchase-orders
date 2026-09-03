package toast

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/yumyums/hq/internal/alerts"
)

// writeStubKey drops a readable stub key so SyncDate gets past its os.ReadFile
// guard and reaches the (injected, failing) dialer.
func writeStubKey(t *testing.T, dir string) string {
	t.Helper()
	keyFile := filepath.Join(dir, "id_rsa")
	if err := os.WriteFile(keyFile, []byte("stub-key"), 0600); err != nil {
		t.Fatalf("write stub key: %v", err)
	}
	return keyFile
}

// spacesSentinel returns a non-nil *s3.Client so SyncDate's nil-check passes.
// The failing dialer trips before any Spaces call, so the client is never used.
func spacesSentinel() *s3.Client {
	return &s3.Client{}
}

// failingDialer is a fake SFTP dialer that always fails to open/authenticate —
// the exact B-146 scenario (key never shipped / auth rejected). Injected via
// Config.Dialer so SyncDate never touches a live SFTP endpoint.
func failingDialer(cfg Config, pemKey string) (*Client, error) {
	return nil, errors.New("ssh dial: unable to authenticate")
}

// TestSyncDate_DialFailure_ReturnsUnavailable proves the classification split:
// a dial/auth failure must NOT be downgraded to ErrSFTPMiss (the silent,
// "expected" miss). It must surface as ErrSFTPUnavailable so the worker can
// route it to the loud path. This is the root of the B-146 silent death.
func TestSyncDate_DialFailure_ReturnsUnavailable(t *testing.T) {
	dir := t.TempDir()
	keyFile := writeStubKey(t, dir)

	cfg := Config{
		SFTPKeyPath:  keyFile,
		SpacesBucket: "unit-bucket",
		CacheDir:     dir,
		Dialer:       failingDialer,
	}
	// SpacesClient is checked before dial; provide a non-nil sentinel so we
	// reach the dial. We never actually call it because dial fails first.
	cfg.SpacesClient = spacesSentinel()

	_, err := SyncDate(t.Context(), cfg, time.Now())
	if !errors.Is(err, ErrSFTPUnavailable) {
		t.Fatalf("dial failure: got %v, want ErrSFTPUnavailable", err)
	}
	if errors.Is(err, ErrSFTPMiss) {
		t.Fatalf("dial failure must NOT be classified as ErrSFTPMiss (silent miss)")
	}
}

// TestHandleSyncOutcome_SFTPUnavailable_FailsLoud is the fail-loud contract at
// the routing layer. Given an SFTP-unavailable outcome, the worker must:
//   (a) set the SyncStatus to failing (readable by /api/v1/health), AND
//   (b) enqueue a Cliq alert (asserted against a fake sink — no live webhook).
//
// RED on the current tree: no ErrSFTPUnavailable, no SyncStatus, no immediate
// alert path exists. GREEN after this card wires them.
func TestHandleSyncOutcome_SFTPUnavailable_FailsLoud(t *testing.T) {
	status := NewSyncStatus(0)
	sink := newFakeAlertSink()

	// Pre-condition: nothing surfaced yet.
	if got := status.Snapshot().Status; got == SyncFailing {
		t.Fatalf("precondition: status already failing before outcome")
	}

	handleSyncOutcome(syncOutcome{
		sftpUnavailable: true,
		lastErr:         ErrSFTPUnavailable,
	}, status, sink.enqueue)

	// (a) Health status reflects the failure.
	v := status.Snapshot()
	if v.Status != SyncFailing {
		t.Fatalf("after SFTP-unavailable: health status %q, want %q", v.Status, SyncFailing)
	}
	if v.LastError == "" {
		t.Fatalf("after SFTP-unavailable: last_error timestamp empty")
	}

	// (b) A Cliq alert was enqueued.
	if len(sink.alerts) != 1 {
		t.Fatalf("after SFTP-unavailable: enqueued %d alerts, want 1", len(sink.alerts))
	}
	a := sink.alerts[0]
	if a.Channel != alerts.ChannelZohoCliq {
		t.Fatalf("alert channel: got %q, want %q", a.Channel, alerts.ChannelZohoCliq)
	}
	if a.Message == "" {
		t.Fatalf("alert message is empty")
	}
}

// TestHandleSyncOutcome_Success_NoAlert: a clean cycle records success and does
// NOT fire an alert.
func TestHandleSyncOutcome_Success_NoAlert(t *testing.T) {
	status := NewSyncStatus(0)
	sink := newFakeAlertSink()

	handleSyncOutcome(syncOutcome{reachedSFTP: true}, status, sink.enqueue)

	if got := status.Snapshot().Status; got != SyncOK {
		t.Fatalf("clean cycle: health status %q, want %q", got, SyncOK)
	}
	if len(sink.alerts) != 0 {
		t.Fatalf("clean cycle: enqueued %d alerts, want 0", len(sink.alerts))
	}
}

// TestSyncDate_KeyUnreadable_ReturnsUnavailable pins the 2026-09-02 prod silent
// death. /app/id_rsa was a DIRECTORY (docker bind-mounted a missing source), so
// os.ReadFile failed "is a directory". The old code returned that as a PLAIN
// error, which runCycle's default branch files as a "systemic" error — and that
// branch sets reachedSFTP=true (it assumes a post-download Spaces hiccup, where
// the transport is alive), so handleSyncOutcome then RecordSuccess()es and
// /health reports toast_sync:ok on a 100%-dead pipeline. A key that cannot be
// read is a DEAD transport, same class as a dial/auth failure, so SyncDate must
// classify it as ErrSFTPUnavailable → the loud path (failing + alert).
func TestSyncDate_KeyUnreadable_ReturnsUnavailable(t *testing.T) {
	dir := t.TempDir()
	keyAsDir := filepath.Join(dir, "id_rsa")
	if err := os.Mkdir(keyAsDir, 0o755); err != nil { // exact prod shape: a directory, not a file
		t.Fatalf("mkdir key-as-dir: %v", err)
	}

	cfg := Config{
		SFTPKeyPath:  keyAsDir,
		SpacesBucket: "unit-bucket",
		CacheDir:     dir,
		Dialer:       failingDialer, // never reached — the key read fails first
	}
	cfg.SpacesClient = spacesSentinel()

	_, err := SyncDate(t.Context(), cfg, time.Now())
	if !errors.Is(err, ErrSFTPUnavailable) {
		t.Fatalf("unreadable key (directory): got %v, want ErrSFTPUnavailable", err)
	}
	if errors.Is(err, ErrSFTPMiss) {
		t.Fatalf("unreadable key must NOT be classified as ErrSFTPMiss (silent miss)")
	}
}

// --- test doubles ---

type fakeAlertSink struct {
	alerts []alerts.Alert
}

func newFakeAlertSink() *fakeAlertSink { return &fakeAlertSink{} }

func (f *fakeAlertSink) enqueue(a alerts.Alert) { f.alerts = append(f.alerts, a) }
