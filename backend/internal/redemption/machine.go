package redemption

import (
	"context"
	"time"

	"github.com/floodfx/gstate"
)

// Machine builds the §18 statechart:
//
//	validating  ─[token≠""]→ burning                ─[else]→ failed
//	burning     ─Invoke(redeem)─ done → route_outcome · error → route_failure
//	route_outcome ─[redeemed]→ redeemed ─[already_used]→ already_used
//	              ─[expired]→ expired ─[else, ordered last]→ failed   (E-KR2)
//	route_failure ─[retries<max]→(retries++) retry_wait ─[else]→ failed
//	retry_wait  ─After(retryDelay)→ burning
//	redeemed / already_used / expired / failed : Final
//
// Retry is bounded BEFORE the wait, via Always-guarded routing, because
// gstate v0.3.1's delayed (After) transitions drop Guard/Assign at fire time
// (executeInternalTransition rebuilds a bare TransitionDef{Target}) — a
// guarded After is silently unguarded, so §18's sketch of the bound living on
// the failed state cannot bound anything. RetryWait's After is deliberately
// unguarded: the budget was already spent entering it. Only burn ERRORS
// (transient: network, DB down) consume retry budget; definitive verdicts —
// including not_found and unknown/garbage outcomes — route through
// route_outcome and never retry.
//
// The machine trusts the Redeemer's verdict and nothing else. No transition
// reads redemption state outside the atomic burn (§18 edge-case 1).
func Machine(db Redeemer, retryDelay time.Duration, maxRetries int) *gstate.Machine[State, Event, Attempt] {
	return gstate.New[State, Event, Attempt]("redemption").
		Initial(Validating).
		State(Validating, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.Always().Guard(func(a Attempt) bool { return a.TokenHash != "" }).GoTo(Burning)
			s.Always().
				Assign(func(a Attempt) Attempt { a.Err = "missing token"; return a }).
				GoTo(Failed)
		}).
		State(Burning, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			// gstate spawns a goroutine and AUTO-CANCELS this ctx on state
			// exit — the flaky-LTE protection (§18 edge-case 4). db.Redeem
			// must honor ctx.
			s.Invoke(func(ctx context.Context, a Attempt, mutate func(func(Attempt) Attempt)) error {
				status, err := db.Redeem(ctx, a.TokenHash, a.DeviceID, a.AuthorizedBy)
				if err != nil {
					mutate(func(a Attempt) Attempt { a.Err = err.Error(); return a })
					return err // → route_failure
				}
				mutate(func(a Attempt) Attempt { a.Outcome = status; return a })
				return nil // → route_outcome
			}, RouteOutcome, RouteFailure)
		}).
		State(RouteOutcome, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			// Eventless fan-out on the burn verdict (ordered).
			s.Always().Guard(func(a Attempt) bool { return a.Outcome == OutcomeRedeemed }).GoTo(Redeemed)
			s.Always().Guard(func(a Attempt) bool { return a.Outcome == OutcomeAlreadyUsed }).GoTo(AlreadyUsed)
			s.Always().Guard(func(a Attempt) bool { return a.Outcome == OutcomeExpired }).GoTo(Expired)
		}).
		State(RouteFailure, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.Always().
				Guard(func(a Attempt) bool { return a.Retries < maxRetries }).
				Assign(func(a Attempt) Attempt { a.Retries++; return a }).
				GoTo(RetryWait)
			s.Always().GoTo(Failed)
		}).
		State(RetryWait, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.After(retryDelay).GoTo(Burning)
		}).
		State(Redeemed, func(s *gstate.StateBuilder[State, Event, Attempt]) { s.Type(gstate.Final) }).
		State(AlreadyUsed, func(s *gstate.StateBuilder[State, Event, Attempt]) { s.Type(gstate.Final) }).
		State(Expired, func(s *gstate.StateBuilder[State, Event, Attempt]) { s.Type(gstate.Final) }).
		State(Failed, func(s *gstate.StateBuilder[State, Event, Attempt]) { s.Type(gstate.Final) }).
		Build()
}
