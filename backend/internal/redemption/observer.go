package redemption

import (
	"context"

	"github.com/floodfx/gstate"
)

// settled is what the terminal observer hands the awaiting Arbitrate call:
// the terminal state, a snapshot of the attempt data at terminal entry, and
// the F4 race-lost determination.
type settled struct {
	state    State
	attempt  Attempt
	raceLost bool
}

// terminalObserver carries the endpoint's await AND the F4 emission decision
// (the card's observer role). It implements gstate's StateEnteredObserver.
//
// gstate locking contract (observer.go, v0.3.1): OnStateEntered runs
// synchronously on the actor's event-processing goroutine WHILE HOLDING the
// actor's internal write lock. So this callback must be non-blocking and must
// not touch the Actor — it snapshots the event data, decides, and signals a
// buffered channel with a non-blocking send. The DB write for the F4
// notification is performed by the awaiting Arbitrate goroutine after it
// receives the settle — deterministic (no fire-and-forget), and no DB work
// ever runs under the actor lock.
type terminalObserver struct {
	gstate.BaseObserver[State, Event, Attempt]
	done chan settled
}

func newTerminalObserver() *terminalObserver {
	return &terminalObserver{done: make(chan settled, 1)}
}

func (o *terminalObserver) OnStateEntered(_ context.Context, e *gstate.StateEvent[State, Event, Attempt]) {
	switch e.State {
	case Redeemed, AlreadyUsed, Expired, Failed:
		// A terminal (all four are gstate.Final) — settle the await.
	default:
		return
	}
	s := settled{state: e.State, attempt: *e.Data()}
	// F4 / E-KR3: an already_used terminal on a SYNCED offline_override is a
	// reconciled lost race — the observer carries the RaceLostReconciled
	// emission decision; the awaiting Arbitrate persists it via the sink.
	if e.State == AlreadyUsed && s.attempt.OfflineOverride {
		s.raceLost = true
	}
	select {
	case o.done <- s:
	default:
		// Already settled — a machine reaches exactly one terminal, so this
		// arm only guards against a pathological double-entry.
	}
}
