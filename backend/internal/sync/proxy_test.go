package sync

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/yumyums/hq/internal/auth"
)

// ───────────────────────────────────────────────────────────────────────────
// Test fixtures
//
// These tests are HERMETIC. They stand up their own upstreams with httptest
// and inject a stub token minter, so nothing here needs Postgres, Docker, or a
// live Supabase container. The complementary proof against the REAL Realtime
// container lives in proxy_live_test.go and skips when it is not running.
// ───────────────────────────────────────────────────────────────────────────

const stubToken = "stub.minted.token"

// stubMinter stands in for the DB-backed bridge minter. Returning a fixed
// string is enough: every assertion below is about WHERE the token ends up and
// whether a caller-supplied one can survive, not about its claims — those are
// jwtbridge_test.go's subject.
func stubMinter(context.Context, *auth.User, string) (string, error) { return stubToken, nil }

// recordedRequest is what an upstream saw. Captured by value because the
// *http.Request is invalid once its handler returns (and, on the upgrade path,
// its connection has been hijacked).
type recordedRequest struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Host   string
}

type recorder struct {
	mu   sync.Mutex
	reqs []recordedRequest
}

func (rec *recorder) record(r *http.Request) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.reqs = append(rec.reqs, recordedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
		Header: r.Header.Clone(),
		Host:   r.Host,
	})
}

func (rec *recorder) last(t *testing.T) recordedRequest {
	t.Helper()
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.reqs) == 0 {
		t.Fatal("upstream received no request at all — the proxy never reached it")
	}
	return rec.reqs[len(rec.reqs)-1]
}

// testUser is the identity auth.Middleware would have attached.
var testUser = &auth.User{
	ID: "11111111-1111-1111-1111-111111111111", Email: "crew@yumyums.kitchen",
	Roles: []string{"team_member"},
}

// mountProxy wires the handler behind a stand-in for auth.Middleware and serves
// it from a REAL http.Server. A real server is mandatory for the upgrade tests:
// httptest.NewRecorder does not implement http.Hijacker, so a 101 can never be
// observed through it.
func mountProxy(t *testing.T, cfg ProxyConfig, user *auth.User, minter TokenMinter) *httptest.Server {
	t.Helper()
	h := newProxyHandler(minter, cfg)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user != nil {
			r = r.WithContext(context.WithValue(r.Context(), auth.CtxKeyUser, user))
		}
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// echoWSUpstream is a WebSocket server that accepts the upgrade and echoes
// every frame back with an "echo:" prefix. This is the hermetic stand-in for
// Realtime: it proves the proxy performs a genuine protocol switch and passes
// bytes in BOTH directions, without asserting anything about Phoenix.
func echoWSUpstream(t *testing.T, rec *recorder) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("upstream websocket.Accept failed: %v", err)
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for {
			typ, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			if err := c.Write(ctx, typ, append([]byte("echo:"), data...)); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func jsonUpstream(t *testing.T, rec *recorder, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "postgrest/12.0.2")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// wsURL rewrites an http:// httptest base URL for websocket.Dial.
func wsURL(base, path string) string { return "ws" + strings.TrimPrefix(base, "http") + path }

// ───────────────────────────────────────────────────────────────────────────
// 🔴 RED #1 — the PLAIN HTTP proxy request.
// ───────────────────────────────────────────────────────────────────────────

// TestProxy_PlainHTTPRequestReachesRESTUpstream is the first of the two
// red-first tests the card requires. It asserts the whole plain-HTTP contract
// at once: the request arrives, the /sync/rest prefix is stripped, the query
// survives, the Host is the UPSTREAM's (not the browser-facing one — the trap a
// naive Director walks into), and the caller travels with a server-minted
// bearer token rather than whatever it sent.
func TestProxy_PlainHTTPRequestReachesRESTUpstream(t *testing.T) {
	rec := &recorder{}
	up := jsonUpstream(t, rec, `[{"id":"n-1"}]`)

	proxy := mountProxy(t, ProxyConfig{RESTURL: up.URL}, testUser, stubMinter)

	resp, err := http.Get(proxy.URL + "/sync/rest/spike_notes?select=id&order=id.desc")
	if err != nil {
		t.Fatalf("GET through proxy: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", resp.StatusCode, body)
	}
	if string(body) != `[{"id":"n-1"}]` {
		t.Errorf("body = %s, want the upstream's body verbatim", body)
	}

	got := rec.last(t)
	if got.Path != "/spike_notes" {
		t.Errorf("upstream path = %q, want %q — the /sync/rest prefix must be stripped, "+
			"otherwise PostgREST looks up a table called `sync`", got.Path, "/spike_notes")
	}
	if got.Query.Get("select") != "id" || got.Query.Get("order") != "id.desc" {
		t.Errorf("upstream query = %v, want select=id&order=id.desc preserved", got.Query)
	}
	upHost := mustHost(t, up.URL)
	if got.Host != upHost {
		t.Errorf("upstream saw Host = %q, want %q. A Director that leaves the inbound Host "+
			"alone sends the BROWSER's host upstream — which is exactly how Realtime's "+
			"tenant lookup ends up resolving `localhost` and 403ing.", got.Host, upHost)
	}
	if auth := got.Header.Get("Authorization"); auth != "Bearer "+stubToken {
		t.Errorf("upstream Authorization = %q, want %q — the bridge token the backend "+
			"mints is what the proxied services accept", auth, "Bearer "+stubToken)
	}
	// PostgREST reads unknown query parameters as column filters, so an
	// `apikey` here would 400 the request with "column apikey does not exist".
	if got.Query.Has("apikey") {
		t.Errorf("upstream query carried apikey=%q; PostgREST reads unknown query params as "+
			"column filters and will reject it", got.Query.Get("apikey"))
	}
}

// ───────────────────────────────────────────────────────────────────────────
// 🔴 RED #2 — the WEBSOCKET UPGRADE request.
// ───────────────────────────────────────────────────────────────────────────

// TestProxy_WebSocketUpgradeSwitchesProtocolAndPassesBytesBothWays is the
// second red-first test, and the one the card says is worth having: it proves
// the proxy performs a real protocol switch and that bytes flow in BOTH
// directions afterwards — not merely that a 101 came back.
//
// It is HERMETIC. The upstream is a local coder/websocket echo server, so what
// it proves is that this proxy handles `Connection: Upgrade`, the
// `Sec-WebSocket-*` handshake, and post-101 bidirectional framing. It proves
// nothing about Phoenix, tenants, or Realtime's own JWT verification — see
// proxy_live_test.go for that half.
func TestProxy_WebSocketUpgradeSwitchesProtocolAndPassesBytesBothWays(t *testing.T) {
	rec := &recorder{}
	up := echoWSUpstream(t, rec)

	proxy := mountProxy(t, ProxyConfig{
		RealtimeURL:  up.URL,
		RealtimeHost: "realtime-dev.localhost",
	}, testUser, stubMinter)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx,
		wsURL(proxy.URL, "/sync/realtime/socket/websocket?vsn=1.0.0"), nil)
	if err != nil {
		t.Fatalf("websocket.Dial through the proxy failed: %v — the upgrade never completed", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("handshake status = %d, want 101 Switching Protocols", resp.StatusCode)
	}

	// → client to upstream
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"event":"phx_join"}`)); err != nil {
		t.Fatalf("write through the proxy: %v", err)
	}
	// ← upstream to client
	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read back through the proxy: %v — the 101 succeeded but no bytes returned, "+
			"which is the failure mode a status-code-only assertion misses", err)
	}
	if typ != websocket.MessageText {
		t.Errorf("frame type = %v, want text", typ)
	}
	if want := `echo:{"event":"phx_join"}`; string(data) != want {
		t.Errorf("echo = %q, want %q", data, want)
	}

	got := rec.last(t)
	if got.Path != "/socket/websocket" {
		t.Errorf("upstream path = %q, want /socket/websocket (prefix stripped)", got.Path)
	}
	if !strings.EqualFold(got.Header.Get("Upgrade"), "websocket") {
		t.Errorf("upstream Upgrade header = %q, want websocket — a proxy that strips it as a "+
			"hop-by-hop header turns the upgrade into a plain GET", got.Header.Get("Upgrade"))
	}
	if !headerHasToken(got.Header, "Connection", "Upgrade") {
		t.Errorf("upstream Connection header = %q, want it to contain the Upgrade token",
			got.Header.Get("Connection"))
	}
	if got.Header.Get("Sec-Websocket-Key") == "" {
		t.Error("upstream saw no Sec-WebSocket-Key — the handshake headers must be forwarded verbatim")
	}
	if got.Query.Get("vsn") != "1.0.0" {
		t.Errorf("upstream vsn = %q, want 1.0.0 preserved", got.Query.Get("vsn"))
	}
	// Phoenix/Realtime reads the token from the apikey query parameter — a
	// browser WebSocket cannot set an Authorization header, so this is the
	// ONLY channel that works for the upgrade.
	if got.Query.Get("apikey") != stubToken {
		t.Errorf("upstream apikey = %q, want the minted %q. A browser cannot set an "+
			"Authorization header on a WebSocket handshake, so the proxy must place the "+
			"token in the query string or Realtime never sees it.",
			got.Query.Get("apikey"), stubToken)
	}
	if got.Host != "realtime-dev.localhost" {
		t.Errorf("upstream saw Host = %q, want realtime-dev.localhost. Self-hosted Realtime "+
			"routes tenants by the FIRST dot-separated label of the Host header; get it "+
			"wrong and you get a bare 403 with no explanation.", got.Host)
	}
}

// ───────────────────────────────────────────────────────────────────────────
// Credential handling
// ───────────────────────────────────────────────────────────────────────────

// TestProxy_ReplacesCallerSuppliedCredentials is the impersonation guard at the
// door. The proxy mints for the CONTEXT user; nothing a caller puts on the wire
// may survive to the upstream, or the same-origin door becomes a
// bring-your-own-token relay into the sync substrate.
func TestProxy_ReplacesCallerSuppliedCredentials(t *testing.T) {
	rec := &recorder{}
	up := jsonUpstream(t, rec, `[]`)
	proxy := mountProxy(t, ProxyConfig{RESTURL: up.URL}, testUser, stubMinter)

	req, err := http.NewRequest("GET", proxy.URL+"/sync/rest/spike_notes?apikey=forged-in-query", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer forged-in-header")
	req.Header.Set("apikey", "forged-in-apikey-header")
	req.Header.Set("Cookie", "hq_session=super-secret-session-value")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request through proxy: %v", err)
	}
	resp.Body.Close()

	got := rec.last(t)
	if got.Header.Get("Authorization") != "Bearer "+stubToken {
		t.Errorf("Authorization = %q, want the SERVER-minted %q — a caller-supplied bearer "+
			"token must never reach the substrate", got.Header.Get("Authorization"), "Bearer "+stubToken)
	}
	if v := got.Header.Get("apikey"); v != "" && v != stubToken {
		t.Errorf("apikey header = %q, want it dropped or replaced", v)
	}
	if got.Query.Has("apikey") {
		t.Errorf("apikey query param survived as %q; it must be dropped on the REST path",
			got.Query.Get("apikey"))
	}
	if c := got.Header.Get("Cookie"); c != "" {
		t.Errorf("Cookie = %q reached the upstream. HQ's session cookie is the credential "+
			"for HQ, not for the sync substrate — forwarding it widens its blast radius "+
			"for no benefit.", c)
	}
}

// TestProxy_HopByHopHeadersAreNotForwarded — RFC 9110 §7.6.1 hop-by-hop
// headers terminate at this proxy. `Upgrade` is the deliberate exception and
// is covered by the upgrade test above.
func TestProxy_HopByHopHeadersAreNotForwarded(t *testing.T) {
	rec := &recorder{}
	up := jsonUpstream(t, rec, `[]`)
	proxy := mountProxy(t, ProxyConfig{RESTURL: up.URL}, testUser, stubMinter)

	req, _ := http.NewRequest("GET", proxy.URL+"/sync/rest/spike_notes", nil)
	req.Header.Set("Keep-Alive", "timeout=5")
	req.Header.Set("Proxy-Connection", "keep-alive")
	req.Header.Set("Te", "trailers")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request through proxy: %v", err)
	}
	resp.Body.Close()

	got := rec.last(t)
	for _, h := range []string{"Keep-Alive", "Proxy-Connection", "Te"} {
		if v := got.Header.Get(h); v != "" {
			t.Errorf("hop-by-hop header %s = %q reached the upstream", h, v)
		}
	}
}

// ───────────────────────────────────────────────────────────────────────────
// Fail-closed behaviour and non-leakage
// ───────────────────────────────────────────────────────────────────────────

// TestProxy_UnconfiguredUpstreamFailsClosed — an unset upstream URL is a
// misconfigured deploy. Mirrors sync_bridge_not_configured and
// auth.ServiceTokenMiddleware: 503, never a guess at where to send the traffic.
func TestProxy_UnconfiguredUpstreamFailsClosed(t *testing.T) {
	proxy := mountProxy(t, ProxyConfig{}, testUser, stubMinter)

	for _, path := range []string{"/sync/rest/spike_notes", "/sync/realtime/socket/websocket"} {
		resp, err := http.Get(proxy.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("%s status = %d, want 503; body = %s", path, resp.StatusCode, body)
		}
		if !strings.Contains(string(body), "sync_proxy_not_configured") {
			t.Errorf("%s body = %s, want the sync_proxy_not_configured envelope", path, body)
		}
	}
}

// TestProxy_BridgeSecretUnsetFailsClosed — the minter's 503, surfaced at the
// door with the same envelope the /sync/token endpoint uses.
func TestProxy_BridgeSecretUnsetFailsClosed(t *testing.T) {
	rec := &recorder{}
	up := jsonUpstream(t, rec, `[]`)
	failing := func(context.Context, *auth.User, string) (string, error) {
		return "", ErrSecretNotConfigured
	}
	proxy := mountProxy(t, ProxyConfig{RESTURL: up.URL}, testUser, failing)

	resp, err := http.Get(proxy.URL + "/sync/rest/spike_notes")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body = %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "sync_bridge_not_configured") {
		t.Errorf("body = %s, want the sync_bridge_not_configured envelope", body)
	}
	if len(rec.reqs) != 0 {
		t.Error("the upstream was contacted despite an unmintable token — the door must " +
			"fail closed BEFORE it proxies anything")
	}
}

// TestProxy_AnonymousIsRejected — defence in depth. Only reachable if the route
// were ever mounted outside the session middleware group; it must never fall
// through to an unauthenticated proxy hop.
func TestProxy_AnonymousIsRejected(t *testing.T) {
	rec := &recorder{}
	up := jsonUpstream(t, rec, `[]`)
	proxy := mountProxy(t, ProxyConfig{RESTURL: up.URL}, nil, stubMinter)

	resp, err := http.Get(proxy.URL + "/sync/rest/spike_notes")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if len(rec.reqs) != 0 {
		t.Error("an anonymous request reached the upstream")
	}
}

// TestProxy_UnknownSubPathIs404 — the door has exactly two rooms. Anything else
// is a 404 from HQ, not a wildcard forward to a guessed upstream.
func TestProxy_UnknownSubPathIs404(t *testing.T) {
	proxy := mountProxy(t, ProxyConfig{RESTURL: "http://127.0.0.1:1"}, testUser, stubMinter)

	for _, path := range []string{"/sync", "/sync/", "/sync/storage/objects", "/sync/restaurant"} {
		resp, err := http.Get(proxy.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404; body = %s", path, resp.StatusCode, body)
		}
	}
}

// TestProxy_UpstreamFailureDoesNotLeakInternals — G4. When the substrate is
// down the client learns that it is down and nothing else: not the internal
// host, not the port, not the token that was about to be presented.
func TestProxy_UpstreamFailureDoesNotLeakInternals(t *testing.T) {
	// 127.0.0.1:1 is reserved-and-closed on every platform CI runs on.
	proxy := mountProxy(t, ProxyConfig{RESTURL: "http://internal-rest.sync.local:1"}, testUser, stubMinter)

	resp, err := http.Get(proxy.URL + "/sync/rest/spike_notes")
	if err != nil {
		t.Fatalf("GET through proxy: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body = %s", resp.StatusCode, body)
	}
	for _, secret := range []string{"internal-rest.sync.local", ":1", stubToken} {
		if strings.Contains(string(body), secret) {
			t.Errorf("error body leaked %q to the client: %s", secret, body)
		}
	}
	if !strings.Contains(string(body), "sync_upstream_unavailable") {
		t.Errorf("body = %s, want the sync_upstream_unavailable envelope", body)
	}
}

// TestProxy_UpstreamServerHeaderIsStripped — the client is told it is talking
// to HQ. Which flavour and version of PostgREST sits behind the door is a
// free reconnaissance hint the browser has no use for.
func TestProxy_UpstreamServerHeaderIsStripped(t *testing.T) {
	rec := &recorder{}
	up := jsonUpstream(t, rec, `[]`) // sets Server: postgrest/12.0.2
	proxy := mountProxy(t, ProxyConfig{RESTURL: up.URL}, testUser, stubMinter)

	resp, err := http.Get(proxy.URL + "/sync/rest/spike_notes")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if s := resp.Header.Get("Server"); strings.Contains(strings.ToLower(s), "postgrest") {
		t.Errorf("response Server header = %q — it names the internal service", s)
	}
}

// ───────────────────────────────────────────────────────────────────────────
// helpers
// ───────────────────────────────────────────────────────────────────────────

func mustHost(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u.Host
}

func headerHasToken(h http.Header, key, token string) bool {
	for _, v := range h.Values(key) {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}
