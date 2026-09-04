package redemption

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ─── Stubs ──────────────────────────────────────────────────────────────────
//
// Every stub implements Redeemer and nothing else — the interface has no
// read/check method on purpose (§18 edge-case 1), so a check-then-act cannot
// be expressed against the real seam. The analog at the bottom exists to
// prove the race harness would catch one if it were ever reintroduced.

// stubRedeemer returns a fixed status/error. Instrumented for call counts,
// first-call-fails (retry leg) and block-until-cancelled (ctx auto-cancel leg).
type stubRedeemer struct {
	status    string
	err       error
	calls     atomic.Int32
	failFirst bool          // first call errors, later calls return status
	blockCtx  bool          // block until ctx cancelled, then report ctx.Err
	cancelled chan struct{} // closed when a blocked call observes ctx.Done
}

func (s *stubRedeemer) Redeem(ctx context.Context, tokenHash, deviceID, authorizedBy string) (string, error) {
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

// atomicStubArbiter models the real redeem() RPC's guarantee: check and write
// in ONE atomic step (here: one mutex hold), so exactly one caller ever wins.
type atomicStubArbiter struct {
	mu         sync.Mutex
	redeemedBy string
	calls      atomic.Int32
}

func (s *atomicStubArbiter) Redeem(ctx context.Context, tokenHash, deviceID, authorizedBy string) (string, error) {
	s.calls.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.redeemedBy == "" {
		s.redeemedBy = deviceID
		return OutcomeRedeemed, nil
	}
	return OutcomeAlreadyUsed, nil
}

// checkThenActAnalog is the §18 edge-case-1 DEFECT, modeled at the arbiter
// seam: it reads redemption state, leaves the lock (the race window), then
// acts on the STALE read. Two concurrent scans both read clean and both win —
// two discounts go out. It exists so the race harness's teeth are provable
// (the redeem-rpc-race-proof precedent: red-first against the naive
// check-then-update analog).
type checkThenActAnalog struct {
	mu         sync.Mutex // protects each individual access, NOT the sequence
	redeemedBy string
}

func (s *checkThenActAnalog) Redeem(ctx context.Context, tokenHash, deviceID, authorizedBy string) (string, error) {
	// CHECK…
	s.mu.Lock()
	taken := s.redeemedBy != ""
	s.mu.Unlock()
	if taken {
		return OutcomeAlreadyUsed, nil
	}
	// …the window in which every concurrent caller also reads clean…
	time.Sleep(25 * time.Millisecond)
	// …then ACT on the stale read.
	s.mu.Lock()
	s.redeemedBy = deviceID
	s.mu.Unlock()
	return OutcomeRedeemed, nil
}

// ─── Harness ────────────────────────────────────────────────────────────────

// testArbitrator builds an Arbitrator with test-speed budgets.
func testArbitrator(r Redeemer, sink RaceLostSink) *Arbitrator {
	return NewArbitrator(r, sink, Config{
		RetryDelay: 100 * time.Millisecond,
		MaxRetries: 1,
		Timeout:    2 * time.Second,
	})
}

// raceWinners drives n concurrent arbitrations of ONE code through r and
// returns wire-result counts. Retries are disabled so each attempt burns
// exactly once.
func raceWinners(t *testing.T, r Redeemer, n int) map[string]int {
	t.Helper()
	arb := NewArbitrator(r, nil, Config{
		RetryDelay: 50 * time.Millisecond,
		MaxRetries: 0,
		Timeout:    5 * time.Second,
	})
	var wg sync.WaitGroup
	results := make(chan string, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			res, err := arb.Arbitrate(context.Background(), Attempt{
				TokenHash: "one-code",
				DeviceID:  fmt.Sprintf("device-%d", i),
			})
			if err != nil {
				results <- "arbitrate-error: " + err.Error()
				return
			}
			results <- res.Result
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	counts := map[string]int{}
	for r := range results {
		counts[r]++
	}
	return counts
}

// ─── Machine legs (§18) ─────────────────────────────────────────────────────

func TestHappyPathRedeemed(t *testing.T) {
	db := &stubRedeemer{status: OutcomeRedeemed}
	res, err := testArbitrator(db, nil).Arbitrate(context.Background(), Attempt{TokenHash: "h", DeviceID: "device-a"})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}
	if res.Terminal != Redeemed || res.Result != ResultRedeemed {
		t.Fatalf("terminal=%s result=%s, want redeemed/redeemed", res.Terminal, res.Result)
	}
	if got := db.calls.Load(); got != 1 {
		t.Fatalf("Redeemer invoked %d times, want 1", got)
	}
}

func TestOutcomeRouting(t *testing.T) {
	for _, tc := range []struct {
		status       string
		wantTerminal State
		wantResult   string
	}{
		{OutcomeAlreadyUsed, AlreadyUsed, ResultAlreadyUsed},
		{OutcomeExpired, Expired, ResultExpired},
	} {
		db := &stubRedeemer{status: tc.status}
		res, err := testArbitrator(db, nil).Arbitrate(context.Background(), Attempt{TokenHash: "h", DeviceID: "device-a"})
		if err != nil {
			t.Fatalf("status %q: arbitrate: %v", tc.status, err)
		}
		if res.Terminal != tc.wantTerminal || res.Result != tc.wantResult {
			t.Fatalf("status %q: terminal=%s result=%s, want %s/%s",
				tc.status, res.Terminal, res.Result, tc.wantTerminal, tc.wantResult)
		}
	}
}

// not_found is a WIRE result, not a machine terminal: the RPC's definitive
// not_found verdict terminates failed (§18 taxonomy holds — no new terminal)
// and the wire derives not_found from the recorded outcome (GAP-1 taxonomy).
// It must never retry: definitive verdicts don't consume retry budget.
func TestNotFoundIsFailedTerminalWithNotFoundWireResult(t *testing.T) {
	db := &stubRedeemer{status: OutcomeNotFound}
	res, err := testArbitrator(db, nil).Arbitrate(context.Background(), Attempt{TokenHash: "h", DeviceID: "device-a"})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}
	if res.Terminal != Failed {
		t.Fatalf("terminal=%s, want failed (not_found is not a machine terminal)", res.Terminal)
	}
	if res.Result != ResultNotFound {
		t.Fatalf("wire result=%s, want not_found", res.Result)
	}
	if got := db.calls.Load(); got != 1 {
		t.Fatalf("Redeemer invoked %d times, want 1 (definitive verdicts never retry)", got)
	}
}

// 🔴 RED-FIRST · E-KR2 — §18 edge-case 3: an unknown/EMPTY burn result must
// terminate failed (wire "error"), NEVER a silent expired. Red while the
// route_outcome ordered fallback is absent: the machine parks in
// route_outcome and the arbitration times out instead of failing loudly.
func TestEKR2_UnknownOrEmptyBurnResultIsFailedNeverExpired(t *testing.T) {
	for _, status := range []string{"", "garbage"} {
		db := &stubRedeemer{status: status}
		res, err := testArbitrator(db, nil).Arbitrate(context.Background(), Attempt{TokenHash: "h", DeviceID: "device-a"})
		if err != nil {
			t.Fatalf("E-KR2 (status %q): arbitration errored (%v) instead of terminating failed — the fallthrough is absent", status, err)
		}
		if res.Terminal == Expired || res.Result == ResultExpired {
			t.Fatalf("E-KR2 (status %q): masqueraded as expired — the silent-expired defect §18 #3 names", status)
		}
		if res.Terminal != Failed || res.Result != ResultError {
			t.Fatalf("E-KR2 (status %q): terminal=%s result=%s, want failed/error", status, res.Terminal, res.Result)
		}
	}
}

func TestMissingTokenFailsWithoutInvoke(t *testing.T) {
	db := &stubRedeemer{status: OutcomeRedeemed}
	res, err := testArbitrator(db, nil).Arbitrate(context.Background(), Attempt{TokenHash: "", DeviceID: "device-a"})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}
	if res.Terminal != Failed || res.Result != ResultError {
		t.Fatalf("terminal=%s result=%s, want failed/error", res.Terminal, res.Result)
	}
	if got := db.calls.Load(); got != 0 {
		t.Fatalf("Redeemer invoked %d times on a token-less attempt, want 0", got)
	}
}

// §18's failed→After→burning retry, bounded: a transient first-burn error
// retries once and the second burn lands redeemed.
func TestBoundedRetryRecoversTransientError(t *testing.T) {
	db := &stubRedeemer{status: OutcomeRedeemed, failFirst: true}
	res, err := testArbitrator(db, nil).Arbitrate(context.Background(), Attempt{TokenHash: "h", DeviceID: "device-a"})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}
	if res.Terminal != Redeemed {
		t.Fatalf("terminal=%s, want redeemed after one retry", res.Terminal)
	}
	if got := db.calls.Load(); got != 2 {
		t.Fatalf("Redeemer invoked %d times, want 2 (fail then succeed)", got)
	}
}

// Retry budget exhausted → failed terminal, wire error — the machine never
// spins unbounded (v0.3.1's After drops guards, so the bound lives in
// route_failure; this is the test that pins it).
func TestRetryExhaustionIsFailedError(t *testing.T) {
	db := &stubRedeemer{err: errors.New("db down")}
	res, err := testArbitrator(db, nil).Arbitrate(context.Background(), Attempt{TokenHash: "h", DeviceID: "device-a"})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}
	if res.Terminal != Failed || res.Result != ResultError {
		t.Fatalf("terminal=%s result=%s, want failed/error", res.Terminal, res.Result)
	}
	if res.Err == "" {
		t.Fatal("failed result carries no error detail")
	}
	if got := db.calls.Load(); got != 2 {
		t.Fatalf("Redeemer invoked %d times, want 2 (initial + MaxRetries=1)", got)
	}
}

// §18 edge-case 4: stopping the actor cancels the Invoke ctx, so a burn hung
// on a dropped hotspot cannot wedge anything. (The spike proved the state-exit
// flavor; this pins the arbitrator's own timeout→Stop path.)
func TestInvokeCtxCancelledWhenArbitrationAbandoned(t *testing.T) {
	db := &stubRedeemer{blockCtx: true, cancelled: make(chan struct{})}
	arb := NewArbitrator(db, nil, Config{RetryDelay: 50 * time.Millisecond, MaxRetries: 0, Timeout: 300 * time.Millisecond})
	_, err := arb.Arbitrate(context.Background(), Attempt{TokenHash: "h", DeviceID: "device-a"})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("arbitrate err=%v, want ErrTimeout while the burn hangs", err)
	}
	select {
	case <-db.cancelled:
		// auto-cancel held: the hung Redeem observed ctx.Done after Stop.
	case <-time.After(2 * time.Second):
		t.Fatal("Invoke ctx was NOT cancelled after abandoning arbitration — a hung call would wedge")
	}
}

func TestUnconfiguredRedeemerFailsClosed(t *testing.T) {
	arb := NewArbitrator(nil, nil, Config{})
	_, err := arb.Arbitrate(context.Background(), Attempt{TokenHash: "h", DeviceID: "device-a"})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err=%v, want ErrNotConfigured", err)
	}
}

// ─── No-TOCTOU (§18 edge-case 1) ────────────────────────────────────────────

// 🔴 RED-FIRST · no-TOCTOU: 8 concurrent arbitrations of one code against an
// ATOMIC arbiter → exactly one redeemed, seven already_used, exactly one burn
// per attempt. The machine derives every verdict solely from the atomic
// burn's answer; there is no read path in the Redeemer interface to guard on.
func TestNoTOCTOU_ConcurrentAttemptsSingleWinner(t *testing.T) {
	stub := &atomicStubArbiter{}
	counts := raceWinners(t, stub, 8)
	if counts[ResultRedeemed] != 1 || counts[ResultAlreadyUsed] != 7 {
		t.Fatalf("winner split %v, want exactly 1 redeemed / 7 already_used", counts)
	}
	if got := stub.calls.Load(); got != 8 {
		t.Fatalf("arbiter consulted %d times for 8 attempts, want exactly 8 (one atomic burn each, no pre-reads)", got)
	}
}

// 🔴 RED-CAPTURE STATE — this body currently asserts the check-then-act
// analog behaves like the atomic arbiter (single winner). It CANNOT: the
// analog acts on a stale read, so concurrent attempts double-win. The failure
// this produces is the recorded proof that the race harness catches the §18
// edge-case-1 defect class (the redeem-rpc-race-proof precedent). The green
// commit inverts this into the permanent control asserting the double-win IS
// detected.
func TestNoTOCTOU_CheckThenActAnalogDoubleWins(t *testing.T) {
	counts := raceWinners(t, &checkThenActAnalog{}, 8)
	if counts[ResultRedeemed] != 1 {
		t.Fatalf("check-then-act analog produced %d winners for one code (%v) — the TOCTOU double-win the harness must catch", counts[ResultRedeemed], counts)
	}
}
