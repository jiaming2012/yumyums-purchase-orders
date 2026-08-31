package toast

import (
	"sync"
	"time"
)

// Toast sync status values surfaced as the "toast_sync".status field of
// /api/v1/health.
//
// B-146: prod's Toast sync was silently dead for weeks because an SFTP
// dial/auth failure was downgraded to an "expected miss" — no health signal,
// no alert, log-and-continue. These states make a dead transport LOUD.
const (
	// SyncOK — the most recent cycle reached SFTP (dial+auth succeeded).
	SyncOK = "ok"
	// SyncFailing — the most recent cycle could not open/authenticate SFTP.
	SyncFailing = "failing"
	// SyncStale — SFTP last succeeded longer ago than the staleness window.
	// Distinct from failing: the loop may be up but no fresh data is landing.
	SyncStale = "stale"
	// SyncUnknown — the worker has not completed a cycle yet (or is disabled).
	SyncUnknown = "unknown"
)

// SyncStatus is a concurrency-safe last-sync tracker. The Toast worker writes
// it once per cycle (RecordSuccess / RecordFailure); the /api/v1/health handler
// reads it via Snapshot. Shape mirrors photos.StorageHealth: a small struct the
// health handler surfaces, so a broken sync is explicit in the health payload —
// not discovered one missing payroll week at a time.
type SyncStatus struct {
	// staleAfter bounds how old lastSuccess may be before a healthy-looking
	// status is reported as "stale". Zero disables staleness reporting.
	staleAfter time.Duration

	mu               sync.Mutex
	lastSuccess      time.Time
	lastError        time.Time
	lastErrorSummary string
	// everFailed is set once RecordFailure runs; it distinguishes "unknown"
	// (never run) from a recovered "ok" state.
	everRan bool
}

// NewSyncStatus creates a tracker. staleAfter is the window past which a
// last-success that is otherwise fine is reported as SyncStale (e.g. 2 sync
// intervals). Zero disables the stale check.
func NewSyncStatus(staleAfter time.Duration) *SyncStatus {
	return &SyncStatus{staleAfter: staleAfter}
}

// RecordSuccess marks that a cycle reached SFTP successfully. Clears the
// failing state; the last-error fields are retained for forensics but no longer
// drive the status.
func (s *SyncStatus) RecordSuccess() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSuccess = time.Now()
	s.everRan = true
}

// RecordFailure marks that a cycle could not open/authenticate SFTP. summary is
// a short human-readable reason surfaced in health + logs.
func (s *SyncStatus) RecordFailure(summary string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = time.Now()
	s.lastErrorSummary = summary
	s.everRan = true
}

// SyncStatusView is the JSON projection surfaced at /api/v1/health under the
// "toast_sync" key. Timestamps are RFC3339; empty (zero) timestamps are omitted.
type SyncStatusView struct {
	Status           string `json:"status"`
	LastSuccess      string `json:"last_success,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	LastErrorSummary string `json:"last_error_summary,omitempty"`
}

// Snapshot computes the current status and returns the health-facing view.
// Status precedence:
//   - unknown  — never ran a cycle
//   - failing  — the last recorded event was a failure (lastError after lastSuccess)
//   - stale    — last success is older than staleAfter (window enabled)
//   - ok       — last success is recent and is the latest event
func (s *SyncStatus) Snapshot() SyncStatusView {
	if s == nil {
		return SyncStatusView{Status: SyncUnknown}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	v := SyncStatusView{}
	if !s.lastSuccess.IsZero() {
		v.LastSuccess = s.lastSuccess.UTC().Format(time.RFC3339)
	}
	if !s.lastError.IsZero() {
		v.LastError = s.lastError.UTC().Format(time.RFC3339)
		v.LastErrorSummary = s.lastErrorSummary
	}

	switch {
	case !s.everRan:
		v.Status = SyncUnknown
	case s.lastError.After(s.lastSuccess):
		// Most recent event was a failure (covers the never-succeeded case too,
		// where lastSuccess is the zero time).
		v.Status = SyncFailing
	case s.staleAfter > 0 && time.Since(s.lastSuccess) > s.staleAfter:
		v.Status = SyncStale
	default:
		v.Status = SyncOK
	}
	return v
}
