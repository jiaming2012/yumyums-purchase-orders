package main

import (
	"io/fs"
	"net/http"
)

// staticHandler serves the embedded frontend (or STATIC_DIR in dev) with
// Cache-Control: no-cache on every response.
//
// Freshness for these files is already handled on the device: the Workbox
// service worker precaches them keyed by content hash and swaps the set when
// sw.js changes. What no-cache governs is the Cloudflare edge in front of the
// tunnel. With no Cache-Control from the origin, Cloudflare caches .js/.css/
// .json by extension for 4h — so after `task prod:deploy` on 2026-10-01 the
// edge kept serving the PRE-deploy marketing/scan-page.js while /api/v1/health
// already reported the new SHA. Worse, Workbox fetches precache entries by
// their plain URL (the revision lives only in the cache key), so a phone that
// picked up the new sw.js inside that window stored the stale script under the
// new revision and stayed stuck until the file's hash changed again.
// Cloudflare honours origin no-cache by not storing the asset at all; the cost
// is one tunnel hop per file on the post-deploy refresh, which a 1–5 person
// crew's PWA never notices.
func staticHandler(fsys fs.FS) http.Handler {
	files := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})
}
