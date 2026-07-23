# DECISIONS NEEDED — overnight-20260724

> **RESOLVED 2026-07-23 — recorded as ledger.md T-21 (decisions 42–43).** G1-a: `/photos/*`
> stays authenticated-only as the documented exception; the durable key-binding fix is
> backlogged (decision 42; union gate rejected as cosmetic). G1-b: the
> `GET /inventory/items` (inventory ∨ purchasing) READ is **RATIFIED** (decision 43).

> Open forks routed to the operator at morning triage. Status: **OPEN** until triaged.

## G1 `grant-enforcement-parity`

### G1-a · `/photos/*` grant mapping — PARKED (card park trigger i)

The photo presign/upload endpoints (`/api/v1/photos/*`) are a cross-app utility called by
workflows.html, purchasing.html, inventory.html AND onboarding.html. No single grant governs
them; a union gate would make the `missing_grant` error field misleading. They remain
**authenticated-only** (the pre-card status quo), recorded as a documented exception in
`tests/grant-enforcement-parity.spec.js`'s header. G6 confirmed: it is the sole ungated
authenticated route, and unauthenticated requests are rejected (401).

**Question:** should `/photos/*` be gated behind the union of the four app grants, split
per-app (separate presign routes per calling app), or stay authenticated-only? Note
`GET /photos/presign` returns stored photo URLs — today any logged-in user with a key can
read any photo, regardless of grants.

### G1-b · `GET /inventory/items` opened to (inventory ∨ purchasing) READ — judgment call, ratify or revert

purchasing.html's `init()` builds the weekly order form from the item catalog inside an
un-caught `Promise.all`; a purchasing-granted, inventory-ungranted user had a broken
Purchasing view. The implementer opened catalog READ to the purchasing grant; item WRITES
remain inventory-only. G6 ruled it acceptable — the payload (`id, description, group_id,
group name, store_location, location_in_store, photo_url`) carries **no cost/price/COGS
fields**, which live behind inventory-gated endpoints.

**Question:** ratify the cross-app READ, or prefer catch-and-degrade in purchasing.html
(one-group revert in `backend/cmd/server/main.go`)?
