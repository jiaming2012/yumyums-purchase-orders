// Spike-local Go module. DELIBERATELY SEPARATE from backend/go.mod, which is a
// HARD-untouched file for this night-crew cycle: adding a module here cannot
// change the dependency graph of HQ's actual backend.
//
// The only requirement is github.com/coder/websocket, and it is pinned to
// v1.8.14 — the exact version backend/go.mod already lists as a DIRECT
// dependency (backend/go.mod:11). Nothing new enters the repo's supply chain;
// the module cache already had it, so this resolves offline.
//
// mintjwt has NO third-party imports at all. It is stdlib crypto/hmac.
module spike-supabase

go 1.25

require github.com/coder/websocket v1.8.14
