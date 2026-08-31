package toast

import (
	"testing"
	"time"
)

// TestSyncStatus_UnknownBeforeAnyCycle: a freshly-constructed tracker reports
// "unknown" — the worker has not run. Health must not claim "ok" before proof.
func TestSyncStatus_UnknownBeforeAnyCycle(t *testing.T) {
	s := NewSyncStatus(0)
	if got := s.Snapshot().Status; got != SyncUnknown {
		t.Fatalf("fresh tracker: got %q, want %q", got, SyncUnknown)
	}
}

// TestSyncStatus_FailingAfterSFTPFailure is the CORE fail-loud contract: once a
// cycle records an SFTP-unavailable failure, health reports "failing" and
// carries a last_error timestamp + summary. This is the signal that was
// missing for the entire B-146 silent-death window.
func TestSyncStatus_FailingAfterSFTPFailure(t *testing.T) {
	s := NewSyncStatus(0)
	s.RecordFailure("sftp dial/auth failed: connection refused")

	v := s.Snapshot()
	if v.Status != SyncFailing {
		t.Fatalf("after failure: got status %q, want %q", v.Status, SyncFailing)
	}
	if v.LastError == "" {
		t.Fatalf("after failure: last_error timestamp is empty, want RFC3339")
	}
	if v.LastErrorSummary == "" {
		t.Fatalf("after failure: last_error_summary is empty, want the reason")
	}
}

// TestSyncStatus_OKAfterSuccess: a successful cycle clears the failing state.
func TestSyncStatus_OKAfterSuccess(t *testing.T) {
	s := NewSyncStatus(0)
	s.RecordSuccess()
	if got := s.Snapshot().Status; got != SyncOK {
		t.Fatalf("after success: got %q, want %q", got, SyncOK)
	}
}

// TestSyncStatus_FailureThenRecovery: failure surfaces as failing, then a later
// success flips back to ok (the recovery must be visible in health).
func TestSyncStatus_FailureThenRecovery(t *testing.T) {
	s := NewSyncStatus(0)
	s.RecordFailure("boom")
	if got := s.Snapshot().Status; got != SyncFailing {
		t.Fatalf("after failure: got %q, want %q", got, SyncFailing)
	}
	s.RecordSuccess()
	if got := s.Snapshot().Status; got != SyncOK {
		t.Fatalf("after recovery: got %q, want %q", got, SyncOK)
	}
}

// TestSyncStatus_StaleWhenLastSuccessOld: with a staleness window, a success
// older than the window reports "stale" — the loop may be up but no fresh data.
func TestSyncStatus_StaleWhenLastSuccessOld(t *testing.T) {
	s := NewSyncStatus(time.Hour)
	s.RecordSuccess()
	// Force the recorded success to be well in the past.
	s.mu.Lock()
	s.lastSuccess = time.Now().Add(-2 * time.Hour)
	s.mu.Unlock()

	if got := s.Snapshot().Status; got != SyncStale {
		t.Fatalf("old success with 1h window: got %q, want %q", got, SyncStale)
	}
}

// TestSyncStatus_NilSafe: a nil tracker Snapshots as unknown and its recorders
// are no-ops (mirrors photos.StorageHealth nil-receiver safety).
func TestSyncStatus_NilSafe(t *testing.T) {
	var s *SyncStatus
	if got := s.Snapshot().Status; got != SyncUnknown {
		t.Fatalf("nil tracker: got %q, want %q", got, SyncUnknown)
	}
	s.RecordSuccess() // must not panic
	s.RecordFailure("x")
}
