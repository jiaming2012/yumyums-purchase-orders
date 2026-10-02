// Package marketing is the campaign-admin half of HQ's Marketing app: the
// manager-facing campaign record, the QR codes minted from it, the PNG a
// manager shares from their phone, and the public landing a customer's camera
// lands on.
//
// # Where the data lives (decision 187)
//
// The twenty-column manager record is HQ Postgres (migration 0083:
// campaigns_admin, qr_codes, qr_scans). The four columns a tablet needs
// offline — id, name, face_value, requires_online — are PROJECTED into the
// Supabase `campaigns` replica after the local transaction commits. One
// writer, one projection, fail loud: a projection that is unconfigured or
// broken leaves projected_at NULL and returns warnings:["not_projected"], so
// the UI can say "not on tablets yet" instead of lying about a success.
//
// # Authorization, in two layers
//
// The surface gate is auth.RequirePermission(pool, "marketing") in main.go —
// the `marketing` app grant, which also opens the scanner. The MANAGER TIER is
// enforced inside every handler this package mounts (handoff §16): a
// team_member gets 403 {"error":"managers_only"}, which the Campaigns tab
// renders as the designed Locked state. It reads auth.UserFromContext, so it
// costs no middleware and no second DB read (spike
// manager-tier-derivable-in-handler, exit 0).
//
// The public landing is the deliberate exception: GET /q/{short} is mounted by
// MountPublic on the ROOT router, outside /api/v1 and outside every gate,
// because a customer pointing a phone camera at a truck sign has no HQ session.
package marketing

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// Env vars this package reads.
const (
	// QRBaseURLEnv overrides the host the QR payload encodes. The default is
	// the production host, and it is 35 characters with a 6-char short — which
	// spike 01 measured as QR version 3, the ceiling for a truck-sign scan.
	// Lengthening it tips the symbol to version 4 (33 modules).
	QRBaseURLEnv = "HQ_QR_BASE_URL"
	// LandingBaseURLEnv overrides the website the landing forwards to.
	LandingBaseURLEnv = "HQ_MARKETING_LANDING_BASE_URL"
	// IPSaltEnv rotates the qr_scans.ip_hash salt. The hash is scoped to the
	// 10-minute dedupe (§4's own comment), so this is a rotation knob rather
	// than a secret.
	IPSaltEnv = "HQ_QR_IP_SALT"
)

// Defaults. DefaultQRBaseURL is deliberately a constant and not a config
// default with a wider type: the payload length is a signed build-fact.
const (
	DefaultQRBaseURL      = "https://hq.yumyums.kitchen"
	DefaultLandingBaseURL = "https://yumyums.kitchen"
	defaultIPSalt         = "hq-qr"
)

// WarningNotProjected is the one warning string this card emits. The UI keys
// its "not on tablets yet" pill off this exact value.
const WarningNotProjected = "not_projected"

// Deps is everything the handlers need. Later cards in Activity H add fields
// here rather than growing main.go's call sites.
type Deps struct {
	Pool           *pgxpool.Pool
	Projection     ProjectionConfig
	QRBaseURL      string
	LandingBaseURL string
	IPSalt         string
}

// NewDeps reads the environment and applies the defaults. main.go calls this
// once and hands the result to both Mount and MountPublic.
func NewDeps(pool *pgxpool.Pool) Deps {
	d := Deps{
		Pool:           pool,
		Projection:     LoadProjectionConfig(),
		QRBaseURL:      strings.TrimRight(os.Getenv(QRBaseURLEnv), "/"),
		LandingBaseURL: strings.TrimRight(os.Getenv(LandingBaseURLEnv), "/"),
		IPSalt:         os.Getenv(IPSaltEnv),
	}
	if d.QRBaseURL == "" {
		d.QRBaseURL = DefaultQRBaseURL
	}
	if d.LandingBaseURL == "" {
		d.LandingBaseURL = DefaultLandingBaseURL
	}
	if d.IPSalt == "" {
		d.IPSalt = defaultIPSalt
	}
	if !d.Projection.Configured() {
		slog.Info("marketing campaign projection not configured; campaigns will save with projected_at NULL and warnings:[\"not_projected\"]",
			"missing", RESTURLEnv+" and/or "+ServiceKeyEnv)
	}
	// ── card H3b (run 20261002): record the Deps for MountReports ──
	//
	// main.go calls NewDeps once (line ~467) and then MountReports(r) INSIDE the
	// `bi` group, with H1's one-argument signature. H3b fills that seam from
	// inside this package, which is the whole reason main.go stays untouched by
	// this card (decision 192) — so the pool has to travel some way other than
	// the signature. NewDeps runs before both mounts, so this pointer is always
	// set by the time MountReports reads it; if it somehow is not, the report
	// handlers answer 503 reports_unavailable rather than nil-panic.
	reportDeps.Store(&d)
	return d
}

// reportDeps is how MountReports reaches the pool without changing its
// signature. Written once at startup by NewDeps, read once at route
// registration; atomic because a test may call NewDeps while a server goroutine
// is up.
var reportDeps atomic.Pointer[Deps]

// Mount registers the gated campaign-admin route table. Call it INSIDE a group
// that already carries auth.Middleware and
// auth.RequirePermission(pool, "marketing").
//
// 🛑 THIS IS THE SEAM. Cards H3b (reconciliation + stats), H4 and H5
// (subscribers) add their routes to THIS function, not to main.go — main.go is
// touched by H1 and H3a only (decision 192).
func Mount(r chi.Router, d Deps) {
	r.Get("/campaigns", ListCampaignsHandler(d))
	r.Post("/campaigns", CreateCampaignHandler(d))
	r.Get("/campaigns/{id}", GetCampaignHandler(d))
	r.Patch("/campaigns/{id}", PatchCampaignHandler(d))
	r.Post("/campaigns/{id}/codes", CreateCodeHandler(d))
	r.Patch("/codes/{id}", PatchCodeHandler(d))
	r.Get("/codes/{id}.png", CodePNGHandler(d))

	// ───────────────────────────────────────────────────────────────────────
	// card H3b · reconciliation + stats (run 20261002). APPEND-ONLY BLOCK —
	// cards H4/H5 add their own below this one; do not interleave.
	//
	// The reconciliation QUEUE stays in Marketing behind the manager tier
	// (decision 192: it is writes, not a report). The two REPORT reads are
	// registered HERE as well as on the BI hub — same handlers, byte-identical
	// bodies, manager tier on this pair ONLY. See MountReports.
	// ───────────────────────────────────────────────────────────────────────
	r.Get("/reconciliation/queue", ReconQueueHandler(d))
	r.Get("/reconciliation/declined", ReconDeclinedHandler(d))
	r.Post("/reconciliation/{attempt_id}/match", ReconDecisionHandler(d, "matched"))
	r.Post("/reconciliation/{attempt_id}/decline", ReconDecisionHandler(d, "declined"))
	r.Post("/reconciliation/{attempt_id}/reopen", ReconDecisionHandler(d, "reopened"))
	r.Post("/reconciliation/{attempt_id}/verify", ReconDecisionHandler(d, "verified"))
	r.Post("/reconciliation/{attempt_id}/reject", ReconDecisionHandler(d, "rejected"))
	r.Get("/stats/overview", StatsOverviewHandler(d, true))
	r.Get("/stats/by", StatsByHandler(d, true))
}

// MountReports is the decision-192 seam for the BI mirror:
// GET /api/v1/bi/campaigns/overview and /api/v1/bi/campaigns/by, behind the
// `bi` grant and with NO manager tier, serving byte-identical bodies to the
// /marketing/stats/* reads.
//
// FILLED BY CARD H3b (run 20261002). H1 shipped it as a deliberate no-op
// because a seam that already answered would gate a surface no handler backs;
// H3b owns both the handlers and the r.Route below, which is why landing the
// report half cost no second edit to main.go.
//
// 🛑 NO MANAGER TIER ON THIS PAIR. Decision 192: the `bi` grant alone opens the
// campaign reports, so anyone holding BI sees campaign money — the consequence
// the operator accepted when they moved the reports off the Marketing page. The
// marketing pair in Mount passes managerTier=true; this one passes false. That
// boolean is the ONLY difference between the two registrations, which is what
// makes the bodies byte-identical.
func MountReports(r chi.Router) {
	var d Deps
	if p := reportDeps.Load(); p != nil {
		d = *p
	}
	MountReportsDeps(r, d)
}

// MountReportsDeps is MountReports with the Deps passed explicitly. It exists so
// a test can mount both halves of decision 192 against one pool, and so the
// route table is registered by exactly one function either way.
func MountReportsDeps(r chi.Router, d Deps) {
	r.Route("/bi/campaigns", func(r chi.Router) {
		r.Get("/overview", StatsOverviewHandler(d, false))
		r.Get("/by", StatsByHandler(d, false))
	})
}

// MountPublic registers the public landing on the ROOT router. It must be
// mounted OUTSIDE /api/v1, outside auth.Middleware and outside every
// RequirePermission, and BEFORE main.go's "/*" static handler.
func MountPublic(r chi.Router, d Deps) {
	h := LandingHandler(d)
	r.Get("/q/{short}", h)
	// HEAD is answered but never logged (§5): a link unfurler or a monitor must
	// not read as a customer. The handler branches on r.Method.
	r.Head("/q/{short}", h)
}

// ── manager tier (handoff §16) ──

// managerRoles is the tier that may administer campaigns. A superadmin passes
// via auth.User.IsSuperadmin, mirroring auth.RequirePermission and
// onboarding.isManagerOrAdmin — the two existing readings of "manager tier" in
// this tree. Adding a role here is a permission change, not a refactor.
var managerRoles = []string{"manager", "admin"}

// requireManager is the first statement of every handler Mount registers,
// reads included: the designed Locked state covers the whole Campaigns tab, not
// just its write buttons. It returns nil and writes the response when the
// caller is refused.
func requireManager(w http.ResponseWriter, r *http.Request) *auth.User {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		// Only reachable if a caller mounts this outside the cookie group.
		// Never fall through to the handler.
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	if user.IsSuperadmin {
		return user
	}
	for _, role := range managerRoles {
		if slices.Contains(user.Roles, role) {
			return user
		}
	}
	writeError(w, http.StatusForbidden, "managers_only")
	return nil
}

// ── envelopes ──

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeErrorWith carries extra keys beside "error" — used by the PNG size
// ladder so a 400 can name the sizes it would have accepted.
func writeErrorWith(w http.ResponseWriter, status int, msg string, extra map[string]any) {
	body := map[string]any{"error": msg}
	for k, v := range extra {
		body[k] = v
	}
	writeJSON(w, status, body)
}
