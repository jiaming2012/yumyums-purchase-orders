// Spike: gstate-invoke-semantics — proves github.com/floodfx/gstate's REAL API
// carries the §18 redemption machine's assumptions. Legs (ledger
// gstate-arbitration-machine.md): compile against the real API; happy-path
// routing; already_used/expired routing; unknown burn result → failed (never a
// silent expired); Invoke ctx auto-cancel on state exit; no-token guard never
// invokes the Redeemer; Failed's After-retry composes with Invoke.
//
// The stub Redeemer is the §18 interface — the DB stays the arbiter; nothing
// here re-checks redemption state (edge-case 1).
package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/floodfx/gstate"
)

type State string
type Event string

const (
	Validating   State = "validating"
	Burning      State = "burning"
	RouteOutcome State = "route_outcome"
	Redeemed     State = "redeemed"
	AlreadyUsed  State = "already_used"
	Expired      State = "expired"
	Failed       State = "failed"
)

const EvAbort Event = "ABORT" // exits Burning mid-invoke (the hung-hotspot exit)

type Attempt struct {
	TokenHash string
	DeviceID  string
	Outcome   string
	Err       string
}

func (a Attempt) Clone() Attempt { return a }

type Redeemer interface {
	Redeem(ctx context.Context, tokenHash, deviceID string) (string, error)
}

type stubRedeemer struct {
	status    string
	err       error
	calls     atomic.Int32
	blockCtx  bool          // block until ctx cancelled, then report on cancelled
	cancelled chan struct{} // closed when a blocked call observes ctx.Done
	failFirst bool          // first call errors, later calls return status
}

func (s *stubRedeemer) Redeem(ctx context.Context, tokenHash, deviceID string) (string, error) {
	n := s.calls.Add(1)
	if s.blockCtx {
		<-ctx.Done()
		close(s.cancelled)
		return "", ctx.Err()
	}
	if s.failFirst && n == 1 {
		return "", errors.New("transient: hotspot dropped")
	}
	return s.status, s.err
}

// machine builds the §18-shaped statechart against the real gstate API.
// retry: whether Failed carries the §18 After-backoff back into Burning.
func machine(db Redeemer, retry bool, retryDelay time.Duration) *gstate.Machine[State, Event, Attempt] {
	return gstate.New[State, Event, Attempt]("redemption").
		Initial(Validating).
		State(Validating, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.Always().Guard(func(a Attempt) bool { return a.TokenHash != "" }).GoTo(Burning)
			s.Always().GoTo(Failed)
		}).
		State(Burning, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.On(EvAbort).GoTo(Failed)
			s.Invoke(func(ctx context.Context, a Attempt, mutate func(func(Attempt) Attempt)) error {
				status, err := db.Redeem(ctx, a.TokenHash, a.DeviceID)
				if err != nil {
					mutate(func(a Attempt) Attempt { a.Err = err.Error(); return a })
					return err
				}
				mutate(func(a Attempt) Attempt { a.Outcome = status; return a })
				return nil
			}, RouteOutcome, Failed)
		}).
		State(RouteOutcome, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.Always().Guard(func(a Attempt) bool { return a.Outcome == "redeemed" }).GoTo(Redeemed)
			s.Always().Guard(func(a Attempt) bool { return a.Outcome == "already_used" }).GoTo(AlreadyUsed)
			s.Always().Guard(func(a Attempt) bool { return a.Outcome == "expired" }).GoTo(Expired)
			s.Always().GoTo(Failed) // unknown/empty → error, NOT a silent "expired" (§18 #3)
		}).
		State(Redeemed, func(s *gstate.StateBuilder[State, Event, Attempt]) { s.Type(gstate.Final) }).
		State(AlreadyUsed, func(s *gstate.StateBuilder[State, Event, Attempt]) { s.Type(gstate.Final) }).
		State(Expired, func(s *gstate.StateBuilder[State, Event, Attempt]) { s.Type(gstate.Final) }).
		State(Failed, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			if retry {
				s.After(retryDelay).GoTo(Burning)
			} else {
				s.Type(gstate.Final)
			}
		}).
		Build()
}

func awaitState[S ~string](t *testing.T, get func() S, want S, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if get() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("state never reached %q within %v (last: %q)", want, within, get())
}

// Leg 2 — happy path: validating → burning → route_outcome → redeemed.
func TestHappyPathRedeemed(t *testing.T) {
	db := &stubRedeemer{status: "redeemed"}
	a := gstate.Start(machine(db, false, 0), Attempt{TokenHash: "h", DeviceID: "device-a"})
	defer a.Stop()
	awaitState(t, a.State, Redeemed, 2*time.Second)
	if got := db.calls.Load(); got != 1 {
		t.Fatalf("Redeemer invoked %d times, want 1", got)
	}
	t.Logf("terminal=%s outcome=%s", a.State(), a.Data().Outcome)
}

// Leg 3 — already_used and expired route to their own terminals.
func TestOutcomeRouting(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   State
	}{
		{"already_used", AlreadyUsed},
		{"expired", Expired},
	} {
		db := &stubRedeemer{status: tc.status}
		a := gstate.Start(machine(db, false, 0), Attempt{TokenHash: "h", DeviceID: "device-a"})
		awaitState(t, a.State, tc.want, 2*time.Second)
		t.Logf("status=%q → terminal=%s", tc.status, a.State())
		a.Stop()
	}
}

// Leg 4 — §18 edge-case 3: an unknown/empty burn result terminates failed,
// never a silent expired.
func TestUnknownOutcomeIsFailedNotExpired(t *testing.T) {
	for _, status := range []string{"", "garbage"} {
		db := &stubRedeemer{status: status}
		a := gstate.Start(machine(db, false, 0), Attempt{TokenHash: "h", DeviceID: "device-a"})
		awaitState(t, a.State, Failed, 2*time.Second)
		if a.State() == Expired {
			t.Fatalf("unknown status %q masqueraded as expired", status)
		}
		t.Logf("status=%q → terminal=%s (not expired)", status, a.State())
		a.Stop()
	}
}

// Leg 5 — §18 edge-case 4: exiting the invoking state auto-cancels the Invoke
// ctx, so a hung burn on a dropped hotspot cannot wedge the machine.
func TestInvokeCtxCancelledOnStateExit(t *testing.T) {
	db := &stubRedeemer{blockCtx: true, cancelled: make(chan struct{})}
	a := gstate.Start(machine(db, false, 0), Attempt{TokenHash: "h", DeviceID: "device-a"})
	defer a.Stop()

	awaitState(t, a.State, Burning, 2*time.Second)
	// give the invoke goroutine a beat to actually enter Redeem and block
	deadline := time.Now().Add(time.Second)
	for db.calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if db.calls.Load() == 0 {
		t.Fatal("Redeemer never entered — cannot test cancellation")
	}

	a.Send(EvAbort) // exit Burning while the burn hangs

	select {
	case <-db.cancelled:
		t.Log("hung Redeem observed ctx.Done after state exit — auto-cancel holds")
	case <-time.After(2 * time.Second):
		t.Fatal("Invoke ctx was NOT cancelled on state exit — a hung call would wedge")
	}
	awaitState(t, a.State, Failed, 2*time.Second)
}

// Leg 6 — no token: the Always-guard fallback routes to failed without ever
// invoking the Redeemer.
func TestNoTokenNeverInvokes(t *testing.T) {
	db := &stubRedeemer{status: "redeemed"}
	a := gstate.Start(machine(db, false, 0), Attempt{TokenHash: "", DeviceID: "device-a"})
	defer a.Stop()
	awaitState(t, a.State, Failed, 2*time.Second)
	if got := db.calls.Load(); got != 0 {
		t.Fatalf("Redeemer invoked %d times on a token-less attempt, want 0", got)
	}
}

// Leg 7 — the §18 Failed→After→Burning retry composes with Invoke: a transient
// first-burn error retries and the second burn lands redeemed.
func TestAfterRetryRecoversTransientError(t *testing.T) {
	db := &stubRedeemer{status: "redeemed", failFirst: true}
	a := gstate.Start(machine(db, true, 150*time.Millisecond), Attempt{TokenHash: "h", DeviceID: "device-a"})
	defer a.Stop()
	awaitState(t, a.State, Redeemed, 4*time.Second)
	if got := db.calls.Load(); got != 2 {
		t.Fatalf("Redeemer invoked %d times, want 2 (fail then succeed)", got)
	}
	t.Logf("transient error retried after backoff; calls=%d terminal=%s", db.calls.Load(), a.State())
}
