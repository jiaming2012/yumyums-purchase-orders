package redemption

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

// fakePostgREST serves /codes (identity resolution) and /rpc/redeem (the
// atomic burn) the way PostgREST does, and records what the redeemer asked.
type fakePostgREST struct {
	t          *testing.T
	codeID     string // "" → token unknown (empty array)
	ok         bool
	reason     string // "" → null reason
	rpcStatus  int    // non-0 → force this status on /rpc/redeem
	rpcCalls   atomic.Int32
	lastCodesQ url.Values
}

func (f *fakePostgREST) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /codes", func(w http.ResponseWriter, r *http.Request) {
		f.checkAuth(r)
		f.lastCodesQ = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		if f.codeID == "" {
			fmt.Fprint(w, `[]`)
			return
		}
		fmt.Fprintf(w, `[{"id":%q}]`, f.codeID)
	})
	mux.HandleFunc("POST /rpc/redeem", func(w http.ResponseWriter, r *http.Request) {
		f.checkAuth(r)
		f.rpcCalls.Add(1)
		if f.rpcStatus != 0 {
			w.WriteHeader(f.rpcStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if f.reason == "" {
			fmt.Fprintf(w, `[{"ok":%t,"reason":null}]`, f.ok)
			return
		}
		fmt.Fprintf(w, `[{"ok":%t,"reason":%q}]`, f.ok, f.reason)
	})
	return mux
}

func (f *fakePostgREST) checkAuth(r *http.Request) {
	if r.Header.Get("apikey") != "test-service-key" ||
		r.Header.Get("Authorization") != "Bearer test-service-key" {
		f.t.Errorf("request %s %s missing service credentials", r.Method, r.URL.Path)
	}
}

func newTestRedeemer(t *testing.T, f *fakePostgREST) (*RPCRedeemer, func()) {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	rd := NewRPCRedeemer(RedeemerConfig{RESTURL: srv.URL, ServiceKey: "test-service-key"})
	return rd, srv.Close
}

func TestRPCRedeemerRedeems(t *testing.T) {
	f := &fakePostgREST{t: t, codeID: "11111111-1111-1111-1111-111111111111", ok: true}
	rd, done := newTestRedeemer(t, f)
	defer done()
	status, err := rd.Redeem(context.Background(), "hash-a", "device-a", "crew@yumyums.kitchen")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if status != OutcomeRedeemed {
		t.Fatalf("status=%q, want redeemed", status)
	}
	// The identity-resolution query must carry NO redemption-state filter —
	// resolving the id is allowed; reading redeemed_at to gate on is the §18
	// edge-case-1 defect (the RPC's conditional UPDATE is the only arbiter).
	q := f.lastCodesQ
	if got := q.Get("select"); got != "id" {
		t.Fatalf("codes select=%q, want id only", got)
	}
	for param := range q {
		if param != "select" && param != "token_hash" {
			t.Fatalf("codes query carries unexpected filter %q — identity resolution must not read redemption state", param)
		}
	}
}

func TestRPCRedeemerVerdictTaxonomy(t *testing.T) {
	for _, reason := range []string{OutcomeAlreadyUsed, OutcomeExpired, OutcomeNotFound} {
		f := &fakePostgREST{t: t, codeID: "11111111-1111-1111-1111-111111111111", ok: false, reason: reason}
		rd, done := newTestRedeemer(t, f)
		status, err := rd.Redeem(context.Background(), "hash-a", "device-a", "")
		done()
		if err != nil {
			t.Fatalf("reason %q: %v", reason, err)
		}
		if status != reason {
			t.Fatalf("reason %q came back as %q", reason, status)
		}
	}
}

// An unknown token resolves to not_found WITHOUT calling the burn.
func TestRPCRedeemerUnknownTokenIsNotFoundWithoutBurn(t *testing.T) {
	f := &fakePostgREST{t: t, codeID: ""}
	rd, done := newTestRedeemer(t, f)
	defer done()
	status, err := rd.Redeem(context.Background(), "unknown-hash", "device-a", "")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if status != OutcomeNotFound {
		t.Fatalf("status=%q, want not_found", status)
	}
	if got := f.rpcCalls.Load(); got != 0 {
		t.Fatalf("burn called %d times for an unknown token, want 0", got)
	}
}

// A NULL reason on a false verdict (GAP-1 says impossible) surfaces as an
// empty status so the machine's E-KR2 fallback fails loudly — never expired.
func TestRPCRedeemerNullReasonFeedsEKR2Fallback(t *testing.T) {
	f := &fakePostgREST{t: t, codeID: "11111111-1111-1111-1111-111111111111", ok: false, reason: ""}
	rd, done := newTestRedeemer(t, f)
	defer done()
	status, err := rd.Redeem(context.Background(), "hash-a", "device-a", "")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if status != "" {
		t.Fatalf("status=%q, want empty (E-KR2 fallback input)", status)
	}
}

func TestRPCRedeemerUpstreamErrorIsError(t *testing.T) {
	f := &fakePostgREST{t: t, codeID: "11111111-1111-1111-1111-111111111111", rpcStatus: http.StatusInternalServerError}
	rd, done := newTestRedeemer(t, f)
	defer done()
	if _, err := rd.Redeem(context.Background(), "hash-a", "device-a", ""); err == nil {
		t.Fatal("upstream 500 returned no error")
	}
}

func TestRedeemerConfigConfigured(t *testing.T) {
	if (RedeemerConfig{}).Configured() {
		t.Fatal("empty config reports configured")
	}
	if (RedeemerConfig{RESTURL: "http://rest:3000"}).Configured() {
		t.Fatal("half config (URL only) reports configured")
	}
	if !(RedeemerConfig{RESTURL: "http://rest:3000", ServiceKey: "k"}).Configured() {
		t.Fatal("full config reports unconfigured")
	}
}
