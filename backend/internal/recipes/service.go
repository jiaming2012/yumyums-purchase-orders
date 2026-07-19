package recipes

import "github.com/yumyums/hq/internal/alerts"

// alertEnqueuer abstracts alerts.Queue.Enqueue so tests can install a fake.
// The concrete *alerts.Queue satisfies this interface.
type alertEnqueuer interface {
	Enqueue(alerts.Alert)
}

// alertQueue is the package-level async alert dispatcher (concrete handle).
// Set via SetAlertQueue at server boot. Nil-safe — if unset, the drift scheduler
// logs the failure but doesn't dispatch (mirrors purchasing.SetAlertQueue posture
// at internal/purchasing/service.go:18-25 and toast.SetAlertQueue at
// internal/toast/worker.go:13-23). Plan 04 reads alertSink (NOT alertQueue) inside
// the scheduler so tests can substitute a fake.
var alertQueue *alerts.Queue

// alertSink is the indirection point the drift scheduler dispatches through.
// In production it points at the same *alerts.Queue. In tests it can be assigned
// to a fakeAlertQueue. Plan 04 must dispatch via `alertSink.Enqueue(...)` with a
// nil-check.
var alertSink alertEnqueuer

// SetAlertQueue wires the Cliq alert delivery queue into the recipes drift scheduler.
// Call once at startup BEFORE StartDriftScheduler, from cmd/server/main.go.
// Installs BOTH the concrete alertQueue var AND the alertSink interface.
func SetAlertQueue(q *alerts.Queue) {
	alertQueue = q
	if q != nil {
		alertSink = q
	}
}
