# Conflict log — run `20260803`

Every merge onto `overnight-20260803` gets an entry, **clean or conflicted** (§15ad.66). A one-line
entry for a clean merge is the point: an empty log must never be readable as "no conflicts" when it
means "the logging never ran."

Dispatch was **SERIAL** (operator's choice at sign-off), so cards never developed concurrently and
the structural opportunity for a collision was small by construction — but that is a reason the log
is short, not a reason to skip it.

Entries are appended **after** each merge lands, never before.

---

## Merge 1 — S1a `sync-cutover-list-scope` → `overnight-20260803`

- **When:** 2026-08-02 ~11:32, after G6 PASS
- **Cards involved:** S1a only. Nothing else had been cut; `overnight-20260803` was still at the
  branch point (`60b9edb`).
- **Result:** 🟢 **CLEAN — no conflicts, no hunks resolved by hand.**
- **Files:** 9 changed, +1397/−45 — `sync-rxdb/client.js`, `sync-schema/sql/0004_write_policies.sql`
  (comments only), `backend/internal/sync/rowvisibility_rls_test.go`,
  `tests/sync-rxdb-client.spec.js`, `sw.js`, plus run/knowledge artifacts (`BACKLOG.md`,
  `roadmap.md`, the design note, the merge-intent).
- **Intents read:** `merge-intent-s1a-sync-cutover-list-scope.md` only — there was no second intent
  to weigh it against. Its "must survive" set (the list scope, the date-floor guard, the three-
  collection Realtime filter, W17) is intact in the merge commit; nothing was dropped.
- **Gate after the merge:** G1 `go build ./...` 0 · `go vet ./...` 0 (run from `backend/`, the module
  root — the first attempt was run from the repo root, where `./...` matches no module, and a pipe
  masked the error; re-run correctly). G4 `node build-sw.js` → **31 files precached**, tree clean on
  the rebuild ⇒ idempotent, version parity 1.4.0 three-way. Regenerated **after** the merge commit
  per B-37.
