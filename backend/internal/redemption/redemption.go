// Package redemption is the server-side arbitration machine for QR code
// redemption — the §18 statechart of docs/qr-offline-redemption-handoff.md,
// built on github.com/floodfx/gstate v0.3.1 (adopted; spike
// .night-crew/knowledge/spikes/activity-d-server-arbitration/).
//
// The machine ORCHESTRATES; it is not the single-use arbiter. The atomic
// `redeem()` RPC (supabase/migrations/20260904000200_redeem_rpc.sql, §6) is
// the only thing enforcing single use, and this package reacts solely to its
// verdict. There is deliberately no way to express a check-then-act here: the
// Redeemer interface has exactly one method and no read/check path (§18
// edge-case 1 — a guard on a prior SELECT reintroduces the TOCTOU race).
//
// Card gstate-arbitration-machine, night-crew run 20260905.
package redemption

import (
	"context"
	"time"
)

// State is a machine state (§18/§19 taxonomy). The four terminals —
// Redeemed, AlreadyUsed, Expired, Failed — are exactly §18's; RouteFailure
// and RetryWait are internal wiring states (bounded retry, see machine.go),
// not terminals, and never surface on the wire.
type State string

// Event exists to satisfy gstate's generic signature; the redemption machine
// is driven entirely by Always/Invoke/After transitions and defines no events.
type Event string

const (
	Validating   State = "validating"
	Burning      State = "burning"
	RouteOutcome State = "route_outcome"
	RouteFailure State = "route_failure" // internal: burn error triage (retry budget)
	RetryWait    State = "retry_wait"    // internal: §18 After-backoff before re-burn
	Redeemed     State = "redeemed"      // terminal ✓
	AlreadyUsed  State = "already_used"  // terminal
	Expired      State = "expired"       // terminal
	Failed       State = "failed"        // terminal — transient exhaustion / unknown result / not_found
)

// Burn outcome strings — the redeem() RPC's signed taxonomy (§6 v2 body:
// ok=true ⇒ redeemed; else already_used | expired | not_found, never NULL).
const (
	OutcomeRedeemed    = "redeemed"
	OutcomeAlreadyUsed = "already_used"
	OutcomeExpired     = "expired"
	OutcomeNotFound    = "not_found"
)

// Wire results — the endpoint's response taxonomy, 1:1 with §19.3's boundary
// events SRV_REDEEMED / ALREADY_USED / EXPIRED / NOT_FOUND / ERROR. not_found
// is a WIRE result only, never a machine terminal: the RPC's not_found reason
// routes through the E-KR2 fallback to the failed terminal and the wire result
// is derived from the attempt's recorded outcome (see wireResult).
const (
	ResultRedeemed    = "redeemed"
	ResultAlreadyUsed = "already_used"
	ResultExpired     = "expired"
	ResultNotFound    = "not_found"
	ResultError       = "error"
)

// Attempt is the machine's data (gstate Cloner). Value-only fields so Clone
// is a plain copy with no shared mutable state.
type Attempt struct {
	TokenHash       string    // sha256 hex of the scanned identity token (never the raw token, §4)
	OrderNumber     string    // Toast order # if captured (§13)
	DeviceID        string    // stable device identifier
	AuthorizedBy    string    // the HQ session user (staff) — server-assigned, never client-supplied
	OfflineOverride bool      // true when this is a synced queued override attempt (§13)
	UnverifiedCode  bool      // F2: override on a code the replica could not verify
	ScannedAt       time.Time // client scan time (server now when absent)
	Value           float64   // offer face value ($) as displayed at accept time
	ValueKnown      bool      // whether Value was supplied (zero is a real value)

	// Set by the machine.
	Outcome string // burn outcome as returned by the Redeemer
	Err     string // last burn / routing error, carried into the failed terminal
	Retries int    // burn-error retries consumed (bounded by Config.MaxRetries)
}

// Clone implements gstate.Cloner.
func (a Attempt) Clone() Attempt { return a }

// Redeemer wraps the ATOMIC arbiter — the §6 redeem() RPC / conditional
// UPDATE. Implementations MUST honor ctx: gstate cancels the Invoke context on
// state exit, and a hung call on a dropped hotspot must not wedge the machine
// (§18 edge-case 4).
//
// This interface deliberately has NO read/check method. Single use is decided
// inside Redeem's atomic statement; anything shaped "check first, then act"
// is the §18 edge-case-1 TOCTOU defect and has no seam to hang on here.
type Redeemer interface {
	Redeem(ctx context.Context, tokenHash, deviceID, authorizedBy string) (status string, err error)
}

// wireResult derives the endpoint's result from a terminal state + attempt.
func wireResult(terminal State, a Attempt) string {
	switch terminal {
	case Redeemed:
		return ResultRedeemed
	case AlreadyUsed:
		return ResultAlreadyUsed
	case Expired:
		return ResultExpired
	default:
		// The failed terminal fans out on the wire: a definitive not_found
		// verdict is reported as such (§9/§19 taxonomy, GAP-1); everything
		// else — unknown/empty outcome, transient exhaustion, missing token —
		// is an ERROR, never a silent expired (E-KR2, §18 #3).
		if a.Outcome == OutcomeNotFound {
			return ResultNotFound
		}
		return ResultError
	}
}
