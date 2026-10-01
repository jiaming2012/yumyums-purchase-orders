package marketing

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

// MountReports is the decision-192 SEAM: H3b registers the BI campaign report
// reads (/bi/campaigns/overview, /bi/campaigns/by) from INSIDE this package, so
// main.go is touched tonight by H1 and H3a only. Tonight it must register
// NOTHING — a seam that already answers would gate a surface no handler backs.
func TestMountReportsIsANoOpSeamToday(t *testing.T) {
	r := chi.NewRouter()
	MountReports(r)

	n := 0
	if err := chi.Walk(r, func(method, route string, h http.Handler, m ...func(http.Handler) http.Handler) error {
		n++
		t.Logf("MountReports registered %s %s", method, route)
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	if n != 0 {
		t.Errorf("MountReports registered %d routes; it is a no-op seam until H3b fills it", n)
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
