package marketing

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

// MountReports is the decision-192 SEAM. Card H1 wrote this test as "it must
// register NOTHING until H3b fills it"; card H3b is the card that fills it, so
// the assertion is inverted here in the same change set that fills the seam.
//
// 🛑 /api/v1/bi/campaigns/* carries the `bi` grant and NO manager tier
// (decision 192). The tier asymmetry itself is asserted in stats_test.go's
// TestStatsReportsAreRegisteredTwiceWithByteIdenticalBodies; this test pins the
// route table.
func TestMountReportsRegistersTheBICampaignReports(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { MountReportsDeps(r, Deps{}) })

	got := map[string]bool{}
	if err := chi.Walk(r, func(method, route string, h http.Handler, m ...func(http.Handler) http.Handler) error {
		got[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	for _, want := range []string{
		"GET /api/v1/bi/campaigns/overview",
		"GET /api/v1/bi/campaigns/by",
	} {
		if !got[want] {
			t.Errorf("MountReports did not register %q (registered: %v)", want, got)
		}
	}
	if len(got) != 2 {
		t.Errorf("MountReports registered %d routes (%v); the BI mirror is exactly the two report reads", len(got), got)
	}
}

// MountReports(r) — H1's one-argument signature, which main.go still calls — must
// register the same table, reaching the pool through the NewDeps-recorded Deps
// rather than through a main.go edit.
func TestMountReportsKeepsCardH1sSignature(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { MountReports(r) })
	n := 0
	if err := chi.Walk(r, func(method, route string, h http.Handler, m ...func(http.Handler) http.Handler) error {
		n++
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	if n != 2 {
		t.Errorf("MountReports(r) registered %d routes, want 2", n)
	}
}

// Mount owns the gated route table. Pinning the shape here is what lets Cards
// 3/4/6 append into it without re-reading main.go.
func TestMountRegistersTheGatedRouteTable(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/api/v1/marketing", func(r chi.Router) { Mount(r, Deps{}) })

	got := map[string]bool{}
	if err := chi.Walk(r, func(method, route string, h http.Handler, m ...func(http.Handler) http.Handler) error {
		got[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	for _, want := range []string{
		"GET /api/v1/marketing/campaigns",
		"POST /api/v1/marketing/campaigns",
		"GET /api/v1/marketing/campaigns/{id}",
		"PATCH /api/v1/marketing/campaigns/{id}",
		"POST /api/v1/marketing/campaigns/{id}/codes",
		"PATCH /api/v1/marketing/codes/{id}",
		"GET /api/v1/marketing/codes/{id}.png",
		// ── card H3b's append-only block ──
		"GET /api/v1/marketing/reconciliation/queue",
		"GET /api/v1/marketing/reconciliation/declined",
		"POST /api/v1/marketing/reconciliation/{attempt_id}/match",
		"POST /api/v1/marketing/reconciliation/{attempt_id}/decline",
		"POST /api/v1/marketing/reconciliation/{attempt_id}/reopen",
		"POST /api/v1/marketing/reconciliation/{attempt_id}/verify",
		"POST /api/v1/marketing/reconciliation/{attempt_id}/reject",
		"GET /api/v1/marketing/stats/overview",
		"GET /api/v1/marketing/stats/by",
	} {
		if !got[want] {
			t.Errorf("Mount did not register %q (registered: %v)", want, got)
		}
	}
}

// MountPublic owns exactly one route and it is deliberately OUTSIDE every
// permission gate: a customer with a phone camera has no HQ session.
func TestMountPublicRegistersOnlyTheLanding(t *testing.T) {
	r := chi.NewRouter()
	MountPublic(r, Deps{})
	routes := []string{}
	if err := chi.Walk(r, func(method, route string, h http.Handler, m ...func(http.Handler) http.Handler) error {
		routes = append(routes, method+" "+route)
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	// GET and HEAD are the two methods chi reports for one r.Get + r.Head pair.
	for _, want := range []string{"GET /q/{short}", "HEAD /q/{short}"} {
		found := false
		for _, r := range routes {
			if r == want {
				found = true
			}
		}
		if !found {
			t.Errorf("MountPublic did not register %q (registered: %v)", want, routes)
		}
	}
	if len(routes) != 2 {
		t.Errorf("MountPublic registered %d routes (%v); the public surface is exactly GET+HEAD /q/{short}", len(routes), routes)
	}
}
