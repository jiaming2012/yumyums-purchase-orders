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

// --- test doubles ---

type fakeAlertSink struct {
	alerts []alerts.Alert
}

func newFakeAlertSink() *fakeAlertSink { return &fakeAlertSink{} }

func (f *fakeAlertSink) enqueue(a alerts.Alert) { f.alerts = append(f.alerts, a) }
