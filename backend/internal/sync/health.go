package sync

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Substrate reachability for /api/v1/health.
//
// The proxy already fails closed — with HQ_SYNC_REST_URL unset every /sync/*
// request answers 503 — but that is only visible to whoever happens to tap
// Scan. The crew member holding the phone needs to know BEFORE they point it
// at a customer's code, which means the launcher has to be able to ask.
//
// Three states, deliberately mirroring photos.StorageHealth so the frontend
// can treat both the same way:
//
//	"unconfigured" — no REST URL set. The normal state outside a sync deploy,
//	                 and NOT a warning: nothing is expected to work, and
//	                 shouting about it on every launcher load would train the
//	                 crew to ignore the banner that matters.
//	"ok"           — the substrate answered.
//	"unreachable"  — configured but not answering. Docker down on the box, the
//	                 container stopped, the host gone.
const (
	SubstrateUnconfigured = "unconfigured"
	SubstrateOK           = "ok"
	SubstrateUnreachable  = "unreachable"
)

// SubstrateHealth answers reachability with a short TTL so a launcher polling
// every 60s (and every page load besides) cannot turn into a probe storm
// against the substrate.
type SubstrateHealth struct {
	restURL string
	client  *http.Client
	ttl     time.Duration

	mu       sync.Mutex
	cached   string
	cachedAt time.Time
}

func NewSubstrateHealth(restURL string, ttl time.Duration) *SubstrateHealth {
	return &SubstrateHealth{
		restURL: restURL,
		// Short timeout on purpose: this runs inside a health request the
		// launcher is waiting on. A substrate that takes 10s to answer is
		// unreachable as far as a crew member scanning a code is concerned.
		client: &http.Client{Timeout: 3 * time.Second},
		ttl:    ttl,
	}
}

func (s *SubstrateHealth) Status(ctx context.Context) string {
	if s == nil || s.restURL == "" {
		return SubstrateUnconfigured
	}
	s.mu.Lock()
	if time.Since(s.cachedAt) < s.ttl && s.cached != "" {
		cached := s.cached
		s.mu.Unlock()
		return cached
	}
	s.mu.Unlock()

	status := SubstrateUnreachable
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.restURL, nil)
	if err == nil {
		resp, doErr := s.client.Do(req)
		if doErr == nil {
			resp.Body.Close()
			// Any HTTP answer means the process is up and listening. A 401 from
			// PostgREST without a key is a REACHABLE substrate, not a broken
			// one — this probe is deliberately unauthenticated.
			status = SubstrateOK
		}
	}

	s.mu.Lock()
	s.cached, s.cachedAt = status, time.Now()
	s.mu.Unlock()
	return status
}
