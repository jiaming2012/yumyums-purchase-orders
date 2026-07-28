package sync

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// ───────────────────────────────────────────────────────────────────────────
// The same-origin door.
//
// DECISION 69 (morning triage 2026-07-27, ledger T-25) closed the origin-shape
// question — obligation 2 of `sync-rxdb-schema-and-replication` — in favour of
// SAME-ORIGIN, proxied by this backend. The sync substrate (PostgREST at
// :3000, Realtime at :4000) is reached at `/sync/*` on HQ's own origin. There
// is no second hostname in the Cloudflare Tunnel, no CORS preflight, no second
// origin for the service worker to reason about, and — the reason it was
// chosen over a second origin — the backend sits ON the path, which is where a
// row-visibility predicate has to be enforceable from.
//
// Three things about this file are not decoration:
//
//  1. THE CLIENT NEVER HOLDS THE SUBSTRATE CREDENTIAL. A caller's own
//     `Authorization` header and `apikey` parameter are DISCARDED, and a token
//     minted for the CONTEXT user is substituted. `TokenHandler`'s
//     impersonation invariant — identity comes only from the session, never
//     from the request — is the same invariant, applied at the door. A proxy
//     that forwarded a caller-supplied bearer token would be a
//     bring-your-own-token relay into the substrate, and RLS would be
//     evaluating whoever the caller claimed to be.
//
//  2. REALTIME ROUTES TENANTS BY THE Host HEADER. Self-hosted Realtime is
//     multi-tenant even with a single tenant, and takes the FIRST
//     dot-separated label of the HTTP Host header as the tenant's
//     external_id. Forward the browser's Host (`hq.yumyums.kitchen`) and the
//     tenant lookup resolves `hq`, which does not exist, and you get a bare
//     403 with no explanation and nothing in the logs to explain it. This cost
//     the 2026-07-25 spike a bring-up attempt; see the header comment of
//     `.night-crew/qa/spike-supabase/rtwatch/main.go`. Hence RealtimeHost.
//
//  3. A BROWSER CANNOT SET A HEADER ON A WEBSOCKET HANDSHAKE. The WebSocket
//     API takes a URL and a subprotocol list and nothing else. So on the
//     Realtime path the token goes in the `apikey` QUERY PARAMETER, which is
//     where Phoenix/Realtime reads it from anyway. On the REST path it must
//     NOT: PostgREST reads unrecognised query parameters as column filters and
//     answers `column "apikey" does not exist`.
//
// NOT IN SCOPE, deliberately: the RLS predicates themselves (obligation 1),
// any RxDB client code, and `workflows.html`. This card builds the door; the
// client knocks on it later.
// ───────────────────────────────────────────────────────────────────────────

// Env vars naming the upstreams. Both unset is the normal state of a deploy
// that has not adopted the sync substrate yet, and it fails CLOSED: the room
// answers 503 sync_proxy_not_configured rather than guessing where to send
// authenticated traffic. Same precedent as ErrSecretNotConfigured and
// auth.ServiceTokenMiddleware.
const (
	// ProxyRESTURLEnv is PostgREST's base URL, e.g. http://rest:3000.
	ProxyRESTURLEnv = "HQ_SYNC_REST_URL"
	// ProxyRealtimeURLEnv is Realtime's base URL, e.g. http://realtime:4000.
	ProxyRealtimeURLEnv = "HQ_SYNC_REALTIME_URL"
	// ProxyRealtimeHostEnv overrides the Host header presented to Realtime.
	// Its FIRST dot-separated label must equal the tenant's external_id. See
	// point 2 in the header comment — this is not cosmetic.
	ProxyRealtimeHostEnv = "HQ_SYNC_REALTIME_HOST"
)

// Path prefixes. The door has exactly two rooms; anything else is a 404 from
// HQ rather than a wildcard forward to a guessed upstream.
const (
	ProxyPrefix         = "/sync"
	ProxyRESTPrefix     = "/sync/rest"
	ProxyRealtimePrefix = "/sync/realtime"
)

// ProxyConfig is the door's whole configuration.
type ProxyConfig struct {
	RESTURL      string // PostgREST base URL; empty disables /sync/rest
	RealtimeURL  string // Realtime base URL; empty disables /sync/realtime
	RealtimeHost string // Host header override for Realtime's tenant routing
}

// LoadProxyConfig reads the config from the environment.
func LoadProxyConfig() ProxyConfig {
	return ProxyConfig{
		RESTURL:      os.Getenv(ProxyRESTURLEnv),
		RealtimeURL:  os.Getenv(ProxyRealtimeURLEnv),
		RealtimeHost: os.Getenv(ProxyRealtimeHostEnv),
	}
}

// TokenMinter mints the substrate credential for one request's user. It is an
// injection seam so the proxy's own behaviour can be tested without a
// database; production always uses poolMinter, which is MintForUser.
type TokenMinter func(ctx context.Context, user *auth.User, sid string) (string, error)

// poolMinter is the production minter: the CALLER'S OWN bridge token, minted
// the same way POST /api/v1/sync/token mints it, with the same fail-closed
// behaviour on an unset secret.
func poolMinter(pool *pgxpool.Pool) TokenMinter {
	return func(ctx context.Context, user *auth.User, sid string) (string, error) {
		secret := os.Getenv(SyncJWTSecretEnv)
		if secret == "" {
			return "", ErrSecretNotConfigured
		}
		tok, _, err := MintForUser(ctx, pool, user, sid, secret, DefaultTokenTTL)
		return tok, err
	}
}

// ProxyHandler is the handler mounted at /sync/*.
//
// MOUNT IT INSIDE A GROUP CARRYING auth.Middleware. It reuses HQ's existing
// session middleware and invents no second auth path; the nil-user check below
// is defence in depth, only reachable if that mounting were ever undone.
//
// It is deliberately NOT behind auth.RequirePermission, for the same reason
// /api/v1/sync/token is not: this is access-resolution plumbing, and the real
// per-row authorization is RLS inside the proxied services reading the live
// grant projection. Choosing an app grant to gate the substrate door behind
// would be inventing a permission concept — the parent card's park trigger.
func ProxyHandler(pool *pgxpool.Pool, cfg ProxyConfig) http.Handler {
	return newProxyHandler(poolMinter(pool), cfg)
}

// upstream is one configured room behind the door.
type upstream struct {
	name     string
	target   *url.URL
	hostOver string // Host header override, "" = use target's host
	apikeyQP bool   // put the token in ?apikey= (Realtime) or strip it (REST)
	proxy    *httputil.ReverseProxy
}

func newProxyHandler(mint TokenMinter, cfg ProxyConfig) http.Handler {
	rest := newUpstream("rest", cfg.RESTURL, "", false)
	realtime := newUpstream("realtime", cfg.RealtimeURL, cfg.RealtimeHost, true)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Identity. Only from the context the session middleware attached;
		//    never from the request.
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		// 2. Path safety, BEFORE the room is chosen — because the room is
		//    chosen from the decoded path, and an encoded separator lets a
		//    caller forge that decision. See unsafeRequestPath.
		if reason := unsafeRequestPath(r.URL); reason != "" {
			slog.Warn("sync proxy rejected an unsafe request path", "user_id", user.ID,
				"reason", reason, "path", r.URL.EscapedPath())
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sync_path_rejected"})
			return
		}

		// 3. Which room. Resolved before anything is minted, so an unknown
		//    path costs no work and reveals nothing about what exists.
		var up *upstream
		var prefix string
		switch {
		case underPrefix(r.URL.Path, ProxyRESTPrefix):
			up, prefix = rest, ProxyRESTPrefix
		case underPrefix(r.URL.Path, ProxyRealtimePrefix):
			up, prefix = realtime, ProxyRealtimePrefix
		default:
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sync_upstream_not_found"})
			return
		}
		if up.proxy == nil {
			writeJSON(w, http.StatusServiceUnavailable,
				map[string]string{"error": "sync_proxy_not_configured"})
			return
		}

		// 4. Mint. BEFORE proxying, so an unmintable token never results in an
		//    unauthenticated hop into the substrate.
		sid := ""
		if c, err := r.Cookie("hq_session"); err == nil {
			sid = auth.HashToken(c.Value)
		}
		tok, err := mint(r.Context(), user, sid)
		if err != nil {
			if errors.Is(err, ErrSecretNotConfigured) {
				writeJSON(w, http.StatusServiceUnavailable,
					map[string]string{"error": "sync_bridge_not_configured"})
				return
			}
			// The secret is never logged or echoed, here or anywhere.
			slog.Error("sync proxy mint failed", "error", err, "user_id", user.ID,
				"upstream", up.name)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		// 5. Rewrite. Work on a clone so the inbound request — which the
		//    server still owns — is not mutated underneath it.
		out := r.Clone(r.Context())

		path := strings.TrimPrefix(r.URL.Path, prefix)
		if path == "" || path[0] != '/' {
			path = "/" + path
		}
		out.URL.Path = path
		// RawPath is cleared so the wire path is re-derived from the decoded
		// Path by URL.EscapedPath(). That is only safe because
		// unsafeRequestPath already rejected every encoded separator — Go's
		// path escaper does NOT re-escape "/", so a surviving %2f would become
		// a real separator here. Everything else (spaces, non-ASCII, "%") is
		// re-escaped correctly, which is why re-deriving beats splicing the
		// caller's raw bytes.

		q := out.URL.Query()
		if up.apikeyQP {
			// Realtime: the ONLY channel a browser WebSocket can carry a
			// credential on. Set, never append — a caller's value is replaced.
			q.Set("apikey", tok)
		} else {
			// PostgREST reads unknown query params as column filters.
			q.Del("apikey")
		}
		out.URL.RawQuery = q.Encode()

		// Caller-supplied credentials do not survive the door.
		out.Header.Del("apikey")
		out.Header.Set("Authorization", "Bearer "+tok)
		// HQ's session cookie authenticates HQ, not the substrate. Forwarding
		// it would widen its blast radius for no benefit.
		out.Header.Del("Cookie")

		up.proxy.ServeHTTP(w, out)
	})
}

// unsafeRequestPath names the reason a request path must not be proxied, or
// returns "" when it is safe. It runs BEFORE the room is chosen, because the
// room is chosen from the DECODED path and an encoded separator lets a caller
// forge that decision (`/sync/rest%2f..%2f..%2fadmin` decodes into the REST
// room and then walks straight out of it).
//
// 🛑 REJECT, DO NOT NORMALISE. `path.Clean` would silently turn the caller's
// request into a different one and proxy that; a 400 says what happened
// instead. Nothing legitimate is lost: the entire vocabulary the sync clients
// speak is `/socket/websocket`, `/<table>` and `/rpc/<fn>`, none of which
// contains a dot segment or an encoded separator.
//
// Why it matters even though both upstreams are path-less today: the standard
// self-hosted Supabase shape puts a gateway with a PATH PREFIX in front
// (`http://kong:8000/rest/v1`). The moment HQ_SYNC_REST_URL looks like that,
// `/sync/rest/../auth/v1/admin/users` escapes the intended prefix and reaches
// a SIBLING SERVICE carrying HQ's minted bearer token.
func unsafeRequestPath(u *url.URL) string {
	// The ESCAPED form first. An encoded separator is invisible in u.Path —
	// by the time you are looking at the decoded path it has already become a
	// real "/" and there is nothing left to detect.
	esc := strings.ToLower(u.EscapedPath())
	if strings.Contains(esc, "%2f") {
		return "encoded_slash"
	}
	// A backslash is not a URL separator, but enough upstreams and gateways
	// treat it as one that forwarding it is a needless bet.
	if strings.Contains(esc, "%5c") || strings.Contains(u.Path, `\`) {
		return "encoded_backslash"
	}
	// Dot SEGMENTS only. A dot inside a segment is ordinary — table and
	// function names contain periods — so `/rest/schema.table` is fine and
	// `/rest/..` is not.
	for _, seg := range strings.Split(u.Path, "/") {
		if seg == "." || seg == ".." {
			return "dot_segment"
		}
	}
	return ""
}

// underPrefix reports whether p is prefix itself or a path below it. Plain
// strings.HasPrefix would route "/sync/restaurant" to the REST upstream.
func underPrefix(p, prefix string) bool {
	return p == prefix || strings.HasPrefix(p, prefix+"/")
}

func newUpstream(name, raw, hostOver string, apikeyQP bool) *upstream {
	up := &upstream{name: name, hostOver: hostOver, apikeyQP: apikeyQP}
	if raw == "" {
		return up // proxy stays nil → 503
	}
	target, err := url.Parse(raw)
	if err != nil || target.Scheme == "" || target.Host == "" {
		slog.Error("sync proxy upstream URL is unusable, that room will 503",
			"upstream", name, "error", err)
		return up
	}
	up.target = target

	up.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// SetURL joins the target's path, copies scheme+host, and — the
			// part that matters — sets Out.Host to "" so the outbound Host
			// becomes the TARGET's, not the browser's. Go's older Director
			// API leaves the inbound Host in place, which is exactly how
			// Realtime's tenant lookup ends up resolving the public hostname.
			pr.SetURL(target)
			pr.SetXForwarded()
			if hostOver != "" {
				pr.Out.Host = hostOver
			}
			// SetXForwarded sets X-Forwarded-For/Proto/Host. X-Forwarded-Host
			// would tell the upstream HQ's public hostname; nothing behind
			// this door needs it and Realtime would only be confused by it.
			pr.Out.Header.Del("X-Forwarded-Host")
		},

		// ReverseProxy handles the 101 protocol switch and the post-upgrade
		// byte pump itself, provided the ResponseWriter implements
		// http.Hijacker and the inbound request is HTTP/1.1. Both hold here:
		// chi's middleware.Logger wrapper implements Hijacker, and the same
		// path already carries the /ws endpoint in production.
		//
		// FlushInterval -1 disables response buffering. Realtime's long-poll
		// fallback transport and any streaming PostgREST response would
		// otherwise sit in a buffer until it filled.
		FlushInterval: -1,

		ModifyResponse: func(resp *http.Response) error {
			// The client is talking to HQ. Which flavour and version of
			// PostgREST or Phoenix sits behind the door is free
			// reconnaissance the browser has no use for.
			resp.Header.Del("Server")
			return nil
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			// The target host:port is logged, never echoed. A client learns
			// the substrate is unavailable and nothing else.
			slog.Error("sync proxy upstream error", "upstream", name,
				"target", target.Host, "path", r.URL.Path, "error", err)
			writeJSON(w, http.StatusBadGateway,
				map[string]string{"error": "sync_upstream_unavailable"})
		},
	}
	return up
}
