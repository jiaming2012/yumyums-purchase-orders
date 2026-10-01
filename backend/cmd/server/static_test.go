package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// The embedded frontend is precached by the Workbox service worker, which
// keys every file by content hash — freshness is already handled on the
// device. Cloudflare, however, caches .js/.css/.json by extension for 4h when
// the origin sends no Cache-Control, so after a deploy the edge kept serving
// the pre-deploy marketing/scan-page.js while /api/v1/health reported the new
// SHA (2026-10-01). A phone that picked up the NEW sw.js in that window
// precached the OLD script under the new revision and stayed stuck. no-cache
// tells the edge not to store these; the worker does the caching.
func TestStaticHandlerSendsNoCache(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":             {Data: []byte("<!doctype html><title>hq</title>")},
		"sw.js":                  {Data: []byte("self.addEventListener('install',()=>{});")},
		"marketing/scan-page.js": {Data: []byte("export const x = 1;")},
		"version.json":           {Data: []byte(`{"frontend":"1.10.2"}`)},
	}
	h := staticHandler(fsys)

	// (/index.html is omitted: http.FileServer has always 301-redirected it to /.)
	for _, path := range []string{"/", "/sw.js", "/marketing/scan-page.js", "/version.json"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d, want 200", path, rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
			t.Errorf("%s: Cache-Control = %q, want %q", path, got, "no-cache")
		}
		if body, _ := io.ReadAll(rec.Body); len(body) == 0 {
			t.Errorf("%s: empty body — the header must not replace the file", path)
		}
	}
}

// A miss stays a miss: the header is about the edge not storing what we serve,
// not about turning 404s into something else.
func TestStaticHandlerStillServes404(t *testing.T) {
	h := staticHandler(fstest.MapFS{"index.html": {Data: []byte("x")}})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope.js", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}
