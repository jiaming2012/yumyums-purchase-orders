package redemption

import (
	"context"
	"errors"
	"time"

	"github.com/floodfx/gstate"
)

// Config tunes one arbitration. Zero values fall back to the defaults below;
// tests inject short budgets.
type Config struct {
	RetryDelay time.Duration // backoff before a bounded burn retry (§18's "After 2s")
	MaxRetries int           // burn-ERROR retries beyond the first attempt (definitive verdicts never retry)
	Timeout    time.Duration // await budget for a terminal
}

const (
	defaultRetryDelay = 2 * time.Second // §18 sketch
	defaultTimeout    = 15 * time.Second
	// defaultMaxRetries is deliberately 1: one §18 After-backoff re-burn for a
	// transient error, then the failed terminal. The wire result is an ERROR
	// the client may resubmit; unbounded in-machine retry would hang the
	// scanner's synchronous submit.
	defaultMaxRetries = 1
)

// ErrNotConfigured is returned when no Redeemer is wired (arbiter backend env
// unset). The endpoint maps it to 503 — fail-closed, same doctrine as the
// sync proxy and HQ_INVENTORY_SERVICE_TOKEN.
var ErrNotConfigured = errors.New("redemption: arbiter backend not configured")

// ErrTimeout is returned when the machine reaches no terminal inside the
// budget. The endpoint maps it to 504.
var ErrTimeout = errors.New("redemption: arbitration reached no terminal within budget")

// ErrNotificationFailed wraps a RaceLostSink persistence failure AFTER a
// race-lost verdict. Loud + retryable (endpoint: 500): re-arbitrating an
// already_used attempt is stable, so the client sync may simply retry.
var ErrNotificationFailed = errors.New("redemption: race-lost notification write failed")

// RaceLostSink persists the F4 RaceLostReconciled domain event as the
// Shift-Manager read-model entry (race_lost_notifications).
type RaceLostSink interface {
	Emit(ctx context.Context, ev RaceLostReconciled) error
}

// Arbitrator drives one §18 machine per attempt and owns the actor lifecycle.
type Arbitrator struct {
	redeemer Redeemer
	sink     RaceLostSink
	cfg      Config
}

// NewArbitrator wires the atomic arbiter and the F4 sink. redeemer may be nil
// (unconfigured deploy) — every Arbitrate then fails closed with
// ErrNotConfigured.
func NewArbitrator(redeemer Redeemer, sink RaceLostSink, cfg Config) *Arbitrator {
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = defaultRetryDelay
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = defaultMaxRetries
	}
	return &Arbitrator{redeemer: redeemer, sink: sink, cfg: cfg}
}

// Result is one arbitration's outcome, ready for the wire.
type Result struct {
	Terminal           State  // machine terminal (§18 taxonomy)
	Result             string // wire result (§19.3 SRV_*): redeemed|already_used|expired|not_found|error
	RaceLostReconciled bool   // F4: already_used terminal on a synced offline_override
	Err                string // machine-recorded error detail (result "error" only)
}

// Arbitrate runs one attempt through the machine and awaits a terminal.
// It reacts ONLY to the atomic redeem()'s verdict — no check-then-act (§18 #1).
func (arb *Arbitrator) Arbitrate(ctx context.Context, a Attempt) (Result, error) {
	if arb.redeemer == nil {
		return Result{}, ErrNotConfigured
	}
	if a.ScannedAt.IsZero() {
		a.ScannedAt = time.Now().UTC()
	}

	obs := newTerminalObserver()
	m := Machine(arb.redeemer, arb.cfg.RetryDelay, arb.cfg.MaxRetries)
	actor := gstate.Start(m, a, m.WithObservers(obs))
	defer actor.Stop()

	timer := time.NewTimer(arb.cfg.Timeout)
	defer timer.Stop()

	select {
	case s := <-obs.done:
		res := Result{
			Terminal: s.state,
			Result:   wireResult(s.state, s.attempt),
		}
		if res.Result == ResultError {
			res.Err = s.attempt.Err
		}
		return res, nil
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-timer.C:
		return Result{}, ErrTimeout
	}
}
