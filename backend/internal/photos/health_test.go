package photos

// B-172 regression: a canceled/unreachable storage account must be explicit.
// These tests pin the StorageHealth probe that /api/v1/health surfaces as the
// "storage" field: ok | unreachable | unconfigured.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeS3 returns an httptest server whose HeadBucket (HEAD /{bucket}) answers
// with the status code held in code. Swap the code to simulate an account
// dying or recovering between probes.
func fakeS3(t *testing.T, code *atomic.Int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(code.Load()))
	}))
}

func healthClientFor(url string) *SpacesConfig {
	return &SpacesConfig{
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Endpoint:  url,
		Region:    "test-region",
		Bucket:    "test-bucket",
	}
}

func TestStorageHealth_Unconfigured(t *testing.T) {
	if got := NewStorageHealth(nil, "", time.Minute).Status(context.Background()); got != StorageUnconfigured {
		t.Fatalf("nil client: got %q, want %q", got, StorageUnconfigured)
	}
	var h *StorageHealth
	if got := h.Status(context.Background()); got != StorageUnconfigured {
		t.Fatalf("nil receiver: got %q, want %q", got, StorageUnconfigured)
	}
}

func TestStorageHealth_Reachable(t *testing.T) {
	var code atomic.Int64
	code.Store(http.StatusOK)
	srv := fakeS3(t, &code)
	defer srv.Close()

	cfg := healthClientFor(srv.URL)
	h := NewStorageHealth(NewSpacesClient(*cfg), cfg.Bucket, time.Minute)
	if got := h.Status(context.Background()); got != StorageOK {
		t.Fatalf("200 HeadBucket: got %q, want %q", got, StorageOK)
	}
}

func TestStorageHealth_UnreachableOn403(t *testing.T) {
	// A canceled account answers 403 to every request — the B-172 shape.
	var code atomic.Int64
	code.Store(http.StatusForbidden)
	srv := fakeS3(t, &code)
	defer srv.Close()

	cfg := healthClientFor(srv.URL)
	h := NewStorageHealth(NewSpacesClient(*cfg), cfg.Bucket, time.Minute)
	if got := h.Status(context.Background()); got != StorageUnreachable {
		t.Fatalf("403 HeadBucket: got %q, want %q", got, StorageUnreachable)
	}
}

func TestStorageHealth_UnreachableOnDeadEndpoint(t *testing.T) {
	var code atomic.Int64
	code.Store(http.StatusOK)
	srv := fakeS3(t, &code)
	srv.Close() // connection refused from here on

	cfg := healthClientFor(srv.URL)
	h := NewStorageHealth(NewSpacesClient(*cfg), cfg.Bucket, time.Minute)
	if got := h.Status(context.Background()); got != StorageUnreachable {
		t.Fatalf("dead endpoint: got %q, want %q", got, StorageUnreachable)
	}
}

func TestStorageHealth_CachesWithinTTLAndReprobesAfter(t *testing.T) {
	var code atomic.Int64
	code.Store(http.StatusOK)
	srv := fakeS3(t, &code)
	defer srv.Close()

	cfg := healthClientFor(srv.URL)
	h := NewStorageHealth(NewSpacesClient(*cfg), cfg.Bucket, time.Minute)
	if got := h.Status(context.Background()); got != StorageOK {
		t.Fatalf("initial probe: got %q, want %q", got, StorageOK)
	}

	// Account dies. Within the TTL the cached verdict is served — no re-probe.
	code.Store(http.StatusForbidden)
	if got := h.Status(context.Background()); got != StorageOK {
		t.Fatalf("within TTL: got %q, want cached %q", got, StorageOK)
	}

	// Force the TTL to lapse (same-package access; no sleeps).
	h.mu.Lock()
	h.lastCheck = time.Now().Add(-2 * time.Minute)
	h.mu.Unlock()
	if got := h.Status(context.Background()); got != StorageUnreachable {
		t.Fatalf("after TTL: got %q, want %q", got, StorageUnreachable)
	}

	// Recovery flips it back after the next lapse.
	code.Store(http.StatusOK)
	h.mu.Lock()
	h.lastCheck = time.Now().Add(-2 * time.Minute)
	h.mu.Unlock()
	if got := h.Status(context.Background()); got != StorageOK {
		t.Fatalf("after recovery: got %q, want %q", got, StorageOK)
	}
}
