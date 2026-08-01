package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/yumyums/hq/internal/auth"
)

// ───────────────────────────────────────────────────────────────────────────
// LIVE proofs against the local Supabase spike stack.
//
// These are the complement to the hermetic tests in proxy_test.go. The
// hermetic upgrade test proves this proxy performs a real protocol switch and
// passes bytes both ways against a local echo server; it CANNOT prove that
// self-hosted Realtime — a Phoenix app with multi-tenant Host routing and its
// own HS256 verification — accepts what comes out the other side. That is what
// these tests are for.
//
// They are GATED ON AN EXPLICIT ENV FLAG, and the gate is deliberately
// asymmetric:
//
//	unset, or =0 / false / no / off  → SKIP. Nobody asked for a live run.
//	=1 (or any other value), port up   → run for real.
//	=1 (or any other value), port down → FAIL. Loudly. Never skip.
//
// The falsy spellings are handled by spikeLiveRequested and are not decoration:
// a bare `!= ""` made `HQ_SYNC_SPIKE_LIVE=0` opt IN (G6 F-4).
//
// 🛑 THE THIRD LINE IS THE POINT (G6 finding R4). The first version of this
// file skipped on an unreachable port with an excellent explanatory message —
// which `go test` prints only under `-v`. Without it, a run whose containers
// had quietly died printed `ok  github.com/yumyums/hq/internal/sync  1.513s`,
// indistinguishable from a run that proved the live upgrade. An intended live
// run must not be able to silently degrade into hermetic-only coverage. Same
// class as the B-09 suite-honesty rule: a green that omits work you believe it
// did is worse than a red.
//
//	HQ_SYNC_SPIKE_LIVE=1 go test ./internal/sync/ -run TestProxyLive
//
// Bring the stack up with:
//
//	docker compose -p spike-supabase -f docker-compose.supabase.yml up -d
//
// The published host ports are EPHEMERAL, so they are RESOLVED with
// `docker compose port` (spikeAddr below -> spikestack_gate_test.go) rather than
// remembered. The constants that used to sit here said 46355/46233 and were
// last correct on 2026-07-29 — the same staleness that made the RLS suites skip
// silently (finding F1, run overnight-20260801). Override with
// HQ_SYNC_SPIKE_REALTIME_ADDR / HQ_SYNC_SPIKE_REST_ADDR.
//
// 🛑 A SKIP IS NOT A PASS. When these skip, the only upgrade evidence in the
// tree is the hermetic one. Say so rather than implying otherwise.
// ───────────────────────────────────────────────────────────────────────────

// spikeJWTSecret is the throwaway secret committed literally in
// docker-compose.supabase.yml (see the banner in that file — generated for the
// spike, public in git on purpose, safe only because nothing real is behind
// it). It is simultaneously PGRST_JWT_SECRET and Realtime's API_JWT_SECRET.
const spikeJWTSecret = "2508c659af3c4316b0a163a00725d33a9bc4eae75aa35ac9be6a007cacb8251c"

// spikeRealtimeVHost's FIRST dot-separated label must equal the tenant
// external_id (SELF_HOST_TENANT_NAME: realtime-dev). Get it wrong and Realtime
// answers a bare 403 with no explanation — which is precisely the failure the
// proxy's Host override exists to prevent.
const spikeRealtimeVHost = "realtime-dev.localhost"

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// spikeLiveEnv opts a run in to the live proofs. Its absence is a skip; its
// presence with a dead port is a FAILURE, not a skip.
const spikeLiveEnv = "HQ_SYNC_SPIKE_LIVE"

// spikeLiveRequested reads the opt-in flag the way a human means it.
//
// 🛑 A BARE `os.Getenv(...) != ""` MADE `HQ_SYNC_SPIKE_LIVE=0` OPT **IN**
// (G6 finding F-4) — measured: `=0` with a dead port FAILED the suite. Anyone
// writing `=0` to mean "off" got the exact opposite of what they asked for,
// which is the worst possible direction for a flag whose entire job is to make
// an intended live run honest. Falsy spellings are off.
func spikeLiveRequested() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(spikeLiveEnv))) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// requireSpikeService gates the live proofs. See the asymmetry in the file
// header — this function is where it is enforced.
func requireSpikeService(t *testing.T, addr, what string) {
	t.Helper()
	if !spikeLiveRequested() {
		t.Skipf("live proofs not requested — set %s=1 to run them against the spike stack. "+
			"SKIPPED IS NOT PASSED: with this off, the only WebSocket-upgrade evidence "+
			"in this package is the hermetic test.", spikeLiveEnv)
	}
	c, err := net.DialTimeout("tcp", addr, 750*time.Millisecond)
	if err != nil {
		t.Fatalf("%s=%q was set, so a LIVE run was intended, but spike %s is not reachable "+
			"at %s: %v\n\nThis is a FAILURE and not a skip on purpose. A skip here prints "+
			"nothing without -v, so an intended live run would silently degrade to hermetic "+
			"coverage and still report `ok`. Bring the stack up with `docker compose -p "+
			"spike-supabase -f docker-compose.supabase.yml up -d`, or set %s=0.",
			spikeLiveEnv, os.Getenv(spikeLiveEnv), what, addr, err, spikeLiveEnv)
	}
	_ = c.Close()
}

// spikeAddr resolves a service's published host address, preferring an explicit
// override. A resolution failure is deliberately NOT a skip: this helper is only
// reached after spikeLiveRequested() said a live run was intended, and the whole
// point of requireSpikeService is that an intended live run never degrades
// quietly.
func spikeAddr(t *testing.T, overrideEnv, service, containerPort string) string {
	t.Helper()
	if v := os.Getenv(overrideEnv); v != "" {
		return v
	}
	port, err := spikeComposePort(service, containerPort)
	if err != nil {
		if !spikeLiveRequested() {
			// Nothing was requested; hand back an address that will not answer, so
			// requireSpikeService takes its own skip arm with its own message.
			return "127.0.0.1:1"
		}
		t.Fatalf("%s=%q was set, so a LIVE run was intended, but the published port for "+
			"%s/%s could not be resolved: %v\n\nBring the stack up with `docker compose -p "+
			"%s -f docker-compose.supabase.yml up -d`, set %s explicitly, or set %s=0.",
			spikeLiveEnv, os.Getenv(spikeLiveEnv), service, containerPort, err,
			spikeComposeProject, overrideEnv, spikeLiveEnv)
	}
	return "127.0.0.1:" + port
}

// TestSpikeLiveRequested pins the flag's truthiness table. It is a two-line
// function guarding a foot-gun that already fired once, and the falsy cases are
// the whole point.
func TestSpikeLiveRequested(t *testing.T) {
	for _, tc := range []struct {
		val  string
		want bool
	}{
		{"", false}, {"0", false}, {"false", false}, {"FALSE", false},
		{"no", false}, {"off", false}, {" 0 ", false},
		{"1", true}, {"true", true}, {"yes", true},
	} {
		t.Setenv(spikeLiveEnv, tc.val)
		if got := spikeLiveRequested(); got != tc.want {
			t.Errorf("%s=%q → %v, want %v", spikeLiveEnv, tc.val, got, tc.want)
		}
	}
}

// liveMinter returns a minter producing a REAL bridge token signed with the
// spike stack's shared secret — the same shape MintForUser emits, minus the
// grant-projection DB round trip.
func liveMinter(t *testing.T) TokenMinter {
	t.Helper()
	now := time.Now()
	claims := Claims{
		Sub:      testUser.ID,
		Role:     SupabaseRole,
		Exp:      now.Add(DefaultTokenTTL).Unix(),
		Iat:      now.Unix(),
		Email:    testUser.Email,
		HQRoles:  testUser.Roles,
		HQGrants: []string{},
	}
	tok, err := Sign(claims, envOr("HQ_SYNC_SPIKE_JWT_SECRET", spikeJWTSecret))
	if err != nil {
		t.Fatalf("sign live bridge token: %v", err)
	}
	return func(context.Context, *auth.User, string) (string, error) { return tok, nil }
}

// TestProxyLive_RealtimeUpgrade is the proof the card asks for: a WebSocket
// upgrade travelling through THIS proxy into the real self-hosted Realtime
// container, completing the handshake and completing a Phoenix channel join.
//
// What a green here proves that the hermetic test cannot:
//   - Realtime's tenant lookup resolved, i.e. the proxy's Host override reached
//     it intact (a wrong Host is a bare 403 before any 101).
//   - Realtime VERIFIED the HS256 token the proxy injected into ?apikey= (a bad
//     signature is also a 403 at the handshake, not an error frame).
//   - A `phx_reply` came back with status ok, so frames flowed in both
//     directions over the switched protocol against a real Phoenix endpoint.
//
// What it still does not prove: that a row change is DELIVERED end to end —
// that needs a replicating table and belongs to the parent card, not the door.
func TestProxyLive_RealtimeUpgrade(t *testing.T) {
	addr := spikeAddr(t, "HQ_SYNC_SPIKE_REALTIME_ADDR", "realtime", "4000")
	requireSpikeService(t, addr, "realtime")

	proxy := mountProxy(t, ProxyConfig{
		RealtimeURL:  "http://" + addr,
		RealtimeHost: spikeRealtimeVHost,
	}, testUser, liveMinter(t))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx,
		wsURL(proxy.URL, "/sync/realtime/socket/websocket?vsn=1.0.0"), nil)
	if err != nil {
		status := "no response"
		if resp != nil {
			status = resp.Status
		}
		t.Fatalf("upgrade into LIVE Realtime through the proxy failed: %v (HTTP %s). "+
			"A 403 here means either the tenant Host never arrived or the token did not verify.",
			err, status)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("handshake status = %d, want 101", resp.StatusCode)
	}
	t.Logf("LIVE: 101 Switching Protocols from realtime at %s via the proxy", addr)

	join := map[string]any{
		"topic": "realtime:public:spike_notes",
		"event": "phx_join",
		"ref":   "1",
		"payload": json.RawMessage(`{"config":{"broadcast":{"ack":false,"self":false},` +
			`"presence":{"key":""},"private":false,` +
			`"postgres_changes":[{"event":"*","schema":"public","table":"spike_notes"}]}}`),
	}
	b, err := json.Marshal(join)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
		t.Fatalf("write phx_join through the proxy: %v", err)
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read from LIVE Realtime through the proxy: %v — the 101 succeeded "+
				"but no frame came back", err)
		}
		var m struct {
			Event   string          `json:"event"`
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(data, &m); err != nil {
			t.Logf("LIVE recv (unparsed): %s", data)
			continue
		}
		t.Logf("LIVE recv event=%s topic=%s payload=%s", m.Event, m.Topic, m.Payload)
		if m.Event != "phx_reply" {
			continue
		}
		var reply struct {
			Status   string          `json:"status"`
			Response json.RawMessage `json:"response"`
		}
		if err := json.Unmarshal(m.Payload, &reply); err != nil {
			t.Fatalf("decode phx_reply payload %s: %v", m.Payload, err)
		}
		if reply.Status != "ok" {
			t.Fatalf("phx_reply status = %q (response %s), want ok. Realtime received the "+
				"join through the proxy but refused it.", reply.Status, reply.Response)
		}
		return // proven
	}
	t.Fatal("no phx_reply within the deadline — Realtime accepted the upgrade but never " +
		"answered the join")
}

// TestProxyLive_RESTRequest is the plain-HTTP half against the real PostgREST
// container. It asserts only that PostgREST answered THIS proxy's request as
// an authenticated caller — not what rows came back, which depends on spike
// fixtures the door does not own.
func TestProxyLive_RESTRequest(t *testing.T) {
	addr := spikeAddr(t, "HQ_SYNC_SPIKE_REST_ADDR", spikeRESTService, spikeRESTPort)
	requireSpikeService(t, addr, "rest")

	proxy := mountProxy(t, ProxyConfig{RESTURL: "http://" + addr}, testUser, liveMinter(t))

	resp, err := http.Get(proxy.URL + "/sync/rest/")
	if err != nil {
		t.Fatalf("GET LIVE PostgREST through the proxy: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	t.Logf("LIVE PostgREST via proxy: HTTP %d, %d bytes", resp.StatusCode, len(body))

	// PIN THE 200 (G6 finding R5b). The first version asserted only
	// `< 500 && != 401`, which would have gone green on a 404 or a 400 — i.e.
	// on a proxy that reached PostgREST but mangled the path, which is exactly
	// the class of bug R1 turned out to be. 200 is what a correctly-proxied
	// authenticated root request actually returns (the OpenAPI description),
	// and it is what this asserts.
	//
	// A signature or role problem would be PostgREST's 401 with a PGRST3xx
	// code; it is called out separately only because it is the one failure that
	// means the door forwarded a token PostgREST will not take.
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatalf("PostgREST rejected the proxied bridge token: %s", truncate(body, 400))
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 from the proxied PostgREST root; body = %s",
			resp.StatusCode, truncate(body, 400))
	}
	if len(body) == 0 {
		t.Error("PostgREST answered 200 with an empty body — the root should carry its " +
			"OpenAPI description, so an empty one suggests the response was not relayed")
	}
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return fmt.Sprintf("%s… (%d bytes total)", b[:n], len(b))
}
