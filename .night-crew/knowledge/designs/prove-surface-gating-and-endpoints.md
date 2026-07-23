# Design — Prove & surface: per-tab gating, Trends/Cost endpoints, convergence coverage contract (`prove-surface-design-draft`)

> **Cycle:** "Prove & surface" — trust the sync, surface the numbers (opened 2026-07-19).
> **Role:** the roadmap **Activity-2 design gate** artifact, PRD **FR-0** — drafted by the
> overnight run, **signed by the operator**. **STATUS: ✅ SIGNED 2026-07-20 (§8; ledger
> T-18) — A4 = Option (i), D2 = Ungrouped, rider (b) rewritten to umbrella semantics,
> B5 authz gate folded into the gating WO. Activity 4 is UNBLOCKED.** The Trust
> track (FR-7/8/9/10) has no dependency on this gate (PRD NFR-5) and is already landing.
> **Traces to:** PRD `.night-crew/knowledge/prds/PRD-prove-and-surface.md` — FR-0/FR-1/FR-3/
> FR-5/FR-6, AC-1/AC-3/AC-5/AC-6, INV-A/B/C, Assumptions A1–A4/A7.
> **One decision is deliberately left open:** the `app_permissions` per-tab representation
> (PRD FR-6a / Assumption A4) is presented as two fully-worked options in §1.4/§1.5 and
> **decided by the operator in §5 — this draft does not choose.** The observable per-tab rule
> (§1.2, PRD FR-5) is fixed regardless of which representation is picked.
> **Format note (mirrors `designs/editprop-frozen-at-submit.md`):** OpenSpec is not
> initialized in this repo (`openspec/` does not exist); the prior cycle's design gate shipped
> as exactly this kind of markdown design doc. The PRD/roadmap wording "OpenSpec design
> change" would require `openspec init` — a structural repo decision left to an attended
> session (§7). This durable doc is the signed gate artifact in the meantime.

---

## 1. The gating model (FR-5, FR-6a, INV-C)

### 1.1 What exists today — the gap this design closes

- **The only auth middleware is logged-in-vs-not.** `auth.Middleware`
  (`backend/internal/auth/middleware.go:24-42`) validates the `hq_session` cookie and
  attaches the `User` to context; it never consults `app_permissions`. Every cookie-auth
  `/api/v1/inventory/*` route (`backend/cmd/server/main.go:513-556`) is reachable by **any**
  logged-in user.
- **Admin checks are ad-hoc per package**, e.g. `isAdmin()` in
  `backend/internal/purchasing/service.go:42-44` (superadmin OR `"admin"` role) and its
  duplicate guarding the users permission handlers
  (`backend/internal/users/handler.go:431-437`, `:453-460`). There is **no**
  `RequirePermission`-style middleware anywhere.
- **The permission data model exists and is healthy.** `hq_apps`
  (`backend/internal/db/migrations/0004_hq_apps.sql`: `slug/name/icon/enabled`) ×
  `app_permissions` (`0005_app_permissions.sql`: `app_id` + exactly-one-of `role`/`user_id`,
  with `role CHECK (role IN ('admin','manager','team_member'))` and unique
  `(app_id, role)` / `(app_id, user_id)` partial indexes).
- **App visibility is already grant-driven.** `MeAppsHandler`
  (`backend/internal/me/handler.go:45-78`) returns all enabled apps for superadmins
  (`queryAllApps`, `:80-100`) and role-or-user-granted apps otherwise (`queryUserApps`,
  `:102-125`). `index.html:121-130` (`filterTilesByPermissions`) hides/shows the **static**
  launcher tiles against that list — slugs with no static tile are simply ignored.
- **The Users admin UI is the grant surface.** `users.html:218` loads
  `GET /api/v1/apps/permissions`; toggles mutate `role_grants`/`user_grants`
  (`users.html:624-644`) and save via `PUT /api/v1/apps/{slug}/permissions`
  (`users.html:501`), backed by `GetAppPermissions`/`SetAppPermissions`
  (`backend/internal/users/db.go:431-459`, `:463-506` — transactional full replace per slug).
- **The `inventory` slug is already registered** — by migration
  (`0024_inventory.sql:58-60`) *and* by the startup upsert `SeedHQApps`
  (`backend/internal/db/db.go:52-64`, `ON CONFLICT (slug) DO NOTHING`). See §4 flag 1.
- **The tabs to gate are stubs.** `inventory.html` `#s5`/`#s6` (`:272-285`) render
  "coming soon" cards via `renderTrends`/`renderCost` (`:993-998`); tab switching is
  `show(n)` (`:524-539`) driving `render()` (`:541-552`). The stubs carry stale
  `TODO: gate … to manager+ via backend roles (INTG-01)` comments (`:273`, `:280`) —
  superseded by the per-tab-grant decision (§4 flag 2).

### 1.2 The observable per-tab rule — fixed regardless of A4 (PRD FR-5, INV-C)

Whichever representation §5 picks, the acceptance contract is identical:

1. A session user **without** the Trends grant gets a distinct **403** from
   `GET /api/v1/inventory/trends` AND the Trends tab button + panel do not render.
2. The Cost grant gates `GET /api/v1/inventory/cost` + tab `#s6` **independently** —
   a user may hold one grant and not the other (§1.6).
3. **0 logged-in-only bypass paths**: hiding a tab client-side without denying its endpoint
   is a violation. The 403 body is distinct from the 401 `{"error":"unauthorized"}`:
   `{"error":"forbidden","missing_grant":"<per-tab grant identifier>"}`.
4. Superadmins implicitly hold every grant (consistent with `queryAllApps` and the
   `isAdmin` convention).
5. The service-token endpoints (`period-summary`, `menu-cogs`,
   `backend/cmd/server/main.go:449-451`) are **not** touched by any of this — they sit
   under `auth.ServiceTokenMiddleware`, not the cookie group.

### 1.3 The common enforcement spine (shared by both options)

End-to-end, five stations — each option below only changes *how the grant is stored and
queried*, never the stations themselves:

1. **Middleware** — a net-new `auth.RequirePermission(pool, grantID)` chi middleware,
   mounted *inside* the existing cookie group (i.e. after `auth.Middleware`,
   `main.go:456`), wrapping only the two new data routes:

   ```go
   r.Group(func(r chi.Router) {
       r.Use(auth.RequirePermission(pool, "inventory-trends"))
       r.Get("/inventory/trends", inventory.TrendsHandler(pool))
   })
   r.Group(func(r chi.Router) {
       r.Use(auth.RequirePermission(pool, "inventory-cost"))
       r.Get("/inventory/cost", inventory.CostHandler(pool))
   })
   ```

   Logic: superadmin → pass; else one EXISTS query against `app_permissions`
   (shape differs per option, §1.4/§1.5); miss → the §1.2 403 envelope.
2. **403 shape** — `{"error":"forbidden","missing_grant":"inventory-trends"}` (resp.
   `inventory-cost`), HTTP 403. Distinct from 401 so the frontend can tell "log in again"
   from "you lack this grant".
3. **`/me` visibility field** — the client needs to know *before rendering tabs* which
   grants it holds. Option (i) rides the existing `GET /api/v1/me/apps` unchanged; option
   (ii) extends its row shape (details per option).
4. **Users admin-UI grant surface** — one toggle set **per tab** (FR-5: "one toggle per
   tab"). Option (i) gets this for free (two new rows appear in the existing per-app
   Access list); option (ii) needs a nested sub-toggle UI.
5. **`inventory.html` tab render guard** — on load, fetch the grant set once; ungated →
   remove/hide `#t5`/`#s5` (resp. `#t6`/`#s6`) and never call the endpoint; also guard the
   `location.hash='tab=5/6'` deep-link path in `show(n)` (`inventory.html:524-531`) so a
   pasted URL cannot render an ungated tab shell. The client guard is UX only — the
   server 403 is the gate (INV-C).

### 1.4 OPTION (i) — two dedicated per-tab slugs (`inventory-trends`, `inventory-cost`)

**Representation.** Two new `hq_apps` rows; a grant is an ordinary `app_permissions` row
against them. **No migration** — `SeedHQApps` (`db.go:52-64`) is an every-startup upsert;
adding the two slugs there registers them in dev and prod on next deploy.

- **Middleware query** — trivial, identical to `queryUserApps`'s shape
  (`me/handler.go:103-110`):

  ```sql
  SELECT EXISTS (
    SELECT 1 FROM app_permissions p
    JOIN hq_apps a ON a.id = p.app_id
    WHERE a.slug = $1 AND a.enabled = true
      AND (p.role = ANY($2) OR p.user_id = $3))
  ```
- **`/me` resolver** — **zero backend change.** `/me/apps` already returns the two slugs
  when granted; `inventory.html` checks the list for `inventory-trends`/`inventory-cost`.
- **Users UI** — **zero backend change.** `GetAppPermissions` (`db.go:431-459`) returns
  every enabled app, so the two rows appear in the Access list automatically with the
  standard role/user toggles; `SetAppPermissions` full-replace-per-slug works unchanged.
- **Launcher side effect** — none in practice: `filterTilesByPermissions`
  (`index.html:121-130`) only toggles *existing static tiles*; there is no tile for the
  new slugs, so nothing renders. (If a tile were ever added by mistake it would show — a
  naming convention note in `SeedHQApps` is the guard.)
- **Trade-offs.**
  - `+` No migration → NFR-3/INV-E reversibility cost is **zero** (no down-migration, no
    up→down→up proof, no pre-deploy backup obligation on this axis).
  - `+` Every station reuses code that exists today; the diff is: seed 2 slugs + 1 new
    middleware + client checks.
  - `+` Cosmetic naming (`Inventory · Trends` / `Inventory · Cost` as the row `name`)
    keeps the Users Access list legible.
  - `−` `hq_apps` semantically becomes "apps *and* gated tabs" — the table name lies
    slightly; the `parent-tab` slug convention (`inventory-*`) is implicit, not schema-
    enforced.
  - `−` Future-tab scaling adds one `hq_apps` row per gated tab — fine at this scale
    (per-tab is the go-forward convention, so expect more rows over time), but the
    Access list grows linearly and has no grouping.
- **Reversal cost if later regretted:** insert-only data; deleting the two rows cascades
  the grants (`ON DELETE CASCADE`, `0005_app_permissions.sql`).

### 1.5 OPTION (ii) — a per-tab sub-permission column on `app_permissions`

**Representation.** A migration (next free number ≥ `0069`) adds a nullable `tab TEXT`
column to `app_permissions`; a per-tab grant is a row `(app_id=inventory, role|user_id,
tab='trends'|'cost')`; `tab IS NULL` keeps meaning today's whole-app grant. The two
partial unique indexes must be rebuilt to include `tab` (else a role can't hold both tab
grants). **This is a schema migration → NFR-3/INV-E applies in full:** a proven
up→down→up down-migration in the WO record + 1 pre-deploy DB backup artifact.

- **Middleware query** — the §1.4 EXISTS plus `AND p.tab = $4`. Semantics decision baked
  in here: a whole-app `inventory` grant (`tab IS NULL`) does **not** imply tab grants —
  per-tab means explicitly granted (the operator's go-forward convention; anything else
  reintroduces a bundled grant by the back door).
- **`/me` resolver** — real change: `queryUserApps`/`queryAllApps`
  (`me/handler.go:80-125`) must aggregate `tab` values into a new per-row field
  (e.g. `"tabs":["trends","cost"]`), or a sibling endpoint must exist. Either way the
  `/me/apps` response shape changes — `index.html`'s cached-shape comparison
  (`index.html:158-166`) tolerates it, but it is a contract change to an endpoint three
  pages consume.
- **Users UI** — real change: the Access list needs a nested per-tab toggle block under
  the Inventory row (a new UI pattern in `users.html`), and `SetPermInput`/
  `SetAppPermissions` (`db.go:83-87`, `:463-506`) must become tab-aware — the current
  full-replace-per-app DELETE (`db.go:480`) would silently wipe tab grants on every
  whole-app save unless the input carries them. This is the option's sharpest edge.
- **Trade-offs.**
  - `+` `hq_apps` stays "one row = one launcher app"; tabs are modeled where they
    conceptually live.
  - `+` Scales to many gated tabs without growing the app list; grouping in the Users UI
    is natural (tabs nested under their app).
  - `−` Migration + down-migration + up→down→up proof + pre-deploy backup (NFR-3).
  - `−` Touches every station: schema, middleware, `/me` shape, Users UI + its save path,
    seed. Largest diff, most places for the FR-5 red-first pairs to catch a hole.
  - `−` The `SetAppPermissions` wipe hazard above must be caught by a dedicated test or
    it becomes the cycle's next escaped defect.

### 1.6 The mixed-grant case (Trends-only user) — both options

Given a non-superadmin holding the Trends grant and not the Cost grant:
`/me` grant set contains Trends only → `#t5`/`#s5` render, `#t6`/`#s6` do not;
`GET /inventory/trends` → 200; `GET /inventory/cost` → 403 (AC-5's mixed case).
Two edges both options share:

- **Tab grant without the base `inventory` app grant.** The launcher tile stays hidden
  (tile visibility keys off the `inventory` slug), but `inventory.html` is a static asset
  any logged-in user can open by URL — the Trends tab would render and its endpoint would
  200. This is *consistent* with FR-5 (the per-tab grant is the gate; the app grant is
  launcher UX), but the Users UI should nudge admins to grant `inventory` alongside a tab
  grant. Recorded as expected behavior, not a bug.
- **Deep link** (`inventory.html#tab=5`) — the `show(n)` guard in §1.3 station 5 covers it.

### 1.7 Test obligations (FR-5 — red-first, with/without-grant pairs)

Per PRD FR-5/AC-5, **each tab** ships a red-first **with-grant / without-grant pair**:
Go (endpoint: 403 for ungated, 200 for granted, distinct envelope asserted) + E2E (tab
hidden for ungated, rendered for granted), **plus** the mixed Trends-only/Cost-hidden
case, **plus** a direct-endpoint-call negative proving no client-only bypass. Option (ii)
additionally owes the `SetAppPermissions` no-wipe regression (§1.5).

## 2. The two aggregation endpoints (FR-1, FR-3, INV-B, A1/A2/A3)

### 2.1 Shared rules

- **Both are cookie-auth, net-new, gated by §1** (Assumption A3). The service-token
  `period-summary` (`backend/internal/inventory/handler.go`, mounted `main.go:450`) and
  `menu-cogs` (`backend/internal/recipes/handler.go:44`, mounted `main.go:451`) are
  **byte-unchanged** (AC-1) — the build cards must carry a regression assertion on both
  existing contracts.
- **Window (Assumption A1):** fixed 12 weeks ending today, computed server-side; v1 takes
  **no query params** (the delegated default; a `?weeks=` knob is a later, cheap change).
  Weeks bucket by `date_trunc('week', …)` (ISO weeks, Monday start — consistent with the
  Monday drift-check convention).
- **Tax-proration** — copied verbatim from the `window_spend` CTE at
  `backend/internal/recipes/handler.go:74-77`:

  ```sql
  SUM(
    (pli.quantity * pli.price) *
    COALESCE(pe.total / NULLIF(pe.total - pe.tax, 0), 1)
  ) AS spend_incl_tax
  ```

  Trends uses this same expression per week×group cell so Trends spend and `menu-cogs`
  COGS reconcile (INV-B); Cost inherits it transitively via the menu-cogs allocation CTE
  it extends.
- **Money never silently dropped:** per-group totals + the Unlinked total = the window's
  confirmed spend (AC-6 reconciliation identity) — a fixture test asserts the identity.

### 2.2 `GET /api/v1/inventory/trends` (FR-1, FR-6b)

> ## ⚠ AMENDED 2026-07-20 — decisions 29, 30, 31 (`ledger.md` §T-19)
>
> **This section as originally signed (2026-07-20) is DEFECTIVE and must not be implemented
> verbatim.** F1 was dispatched faithfully against the text below on overnight-20260722 and
> **PARKED at G6**, which broke its AC-6 reconciliation five ways with realistic fixtures. The
> implementer was correct to follow the sketch and flag rather than silently patch — the defect
> was in this design. The amendments below are the operator decisions taken at triage; they are
> what un-park F1. Where they conflict with the original text, **the amendments win.**
>
> ### Amendment 1 (Decision 29) — filter to the COGS allowlist
>
> The signed SQL has **no `mercury_category` filter**, so a chart titled "spend by group" would
> render rent, insurance, software and fuel as groups and over-report against payroll by an
> unbounded amount (G6 measured +500.00 on a two-event synthetic). Trends MUST filter to the
> **same allowlist `period-summary` is constructed with**:
>
> ```sql
> AND pe.mercury_category = ANY($3)   -- $3 = cogsAllowlist
> ```
>
> - **Source of truth:** `cogsAllowlist`, built in `cmd/server/main.go:438-445` from
>   `HQ_COGS_CATEGORY_ALLOWLIST` (comma-separated, default `"COGS"`). Do **not** re-derive or
>   hardcode it.
> - **Integration note (saves the implementer a discovery):** `cogsAllowlist` is currently
>   constructed *inside* the service-token router group (`main.go:430-451`). Trends is a
>   **cookie-auth** endpoint in a different group — hoist the construction above both groups and
>   pass the same slice to each. Do not build a second copy.
> - **Consequence, stated explicitly:** `mercury_category = ANY($3)` does **not** match SQL NULL,
>   so NULL-category events are **excluded** — same as `period-summary`. This is intended. Note
>   that `menu-cogs`/Cost do **not** filter this way (`recipes/cost.go:29-30`), so Trends and Cost
>   legitimately disagree on this axis; that divergence is documented, not a bug to reconcile.
>
> ### Amendment 2 (Decision 30) — unreviewed receipts are a note, never a bar
>
> Unreviewed receipts have **no linked line items** — linking is precisely what review does — so
> they cannot be bucketed into a week×group cell at all. They are **excluded from `cells`** and
> surfaced as a **completeness figure**, mirroring the existing `unlinked` idiom rather than
> inventing a second one. An "unreviewed" pseudo-group bar was considered and declined.
>
> The eligible-pending population is defined by `period-summary` and MUST be matched exactly
> (`handler.go:1345-1351`): `confirmed_at IS NULL` · `discarded_at IS NULL` ·
> `mercury_category = ANY($3)` · `reason != 'no_attachment_on_bank_tx'` · dated by
> `COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date)`. Amount is
> `SUM(ABS(bank_total))`.
>
> ### Amendment 3 (Decision 31) — attributed spend; NO proration
>
> **Delete the tax-proration factor entirely.** The signed sketch scales every line by
> `COALESCE(pe.total / NULLIF(pe.total - pe.tax, 0), 1)`. `period-summary` does no such thing —
> `cogs_excl_tax` sums line items at **face value** and accounts tax as a separate term
> (`handler.go:1356-1359`). The proration was the mechanism that inflated every food line to
> swallow an unitemized remainder, silently overstating per-group numbers with nothing on screen
> indicating by how much. G6's minimal breaker was **a receipt with an unitemized delivery fee —
> the normal case, not an edge case.**
>
> Cell spend is therefore simply `SUM(pli.quantity * pli.price)`.
>
> Unattributed money is **surfaced, not smeared**. The per-event **unitemized remainder** —
> `(pe.total - pe.tax) - Σ(that event's line items)` — is reported as its own completeness figure.
> It is deliberately **not** an addend to the payroll identity, because `period-summary` does not
> count it either; it exists to explain receipt-total vs line-item coverage to the operator.
>
> ### Amended response shape
>
> The signed envelope below carries no home for the Amendment 2/3 figures. It gains a
> `completeness` object; everything else is unchanged:
>
> ```json
> {
>   "window": { "from": "2026-04-27", "to": "2026-07-19", "weeks": 12 },
>   "groups": [ { "id": "…uuid…", "name": "Proteins" } ],
>   "cells":   [ { "week_start": "2026-07-13", "group_id": "…uuid…", "spend": 412.87 } ],
>   "unlinked":[ { "week_start": "2026-07-13", "spend": 63.10 } ],
>   "unlinked_total": 63.10,
>   "completeness": {
>     "pending_total": 240.00,
>     "pending_count": 3,
>     "unitemized_remainder": 18.45,
>     "reconciles_to_cogs_excl_tax": 4102.55
>   }
> }
> ```
>
> - `pending_total` / `pending_count` — Amendment 2 (unreviewed receipts).
> - `unitemized_remainder` — Amendment 3, window-summed. **Not** an addend to the identity.
> - `reconciles_to_cogs_excl_tax` — the endpoint's own computed
>   `Σcells + Σunlinked + pending_total`, published so the client can display the identity's
>   left side without re-summing, and so a mismatch is visible in the response itself.
> - The **D2 "Ungrouped" pseudo-group** (signed 2026-07-20) is unchanged by these amendments:
>   linked-but-groupless items get an explicit Ungrouped bucket in `groups`/`cells`, and are
>   **not** folded into `unlinked` (which means `purchase_item_id IS NULL` only).
>
> ### The reconciliation identity — exact, and what AC-6 must now assert
>
> Because `cells + unlinked` is exactly `period-summary`'s `lines` term once Amendments 1–3 hold:
>
> ```
> Σcells + Σunlinked + pending  ==  period_summary.cogs_excl_tax
> ```
>
> This is an **exact equality on the same window**, not an approximation — the old identity
> could not hold on messy real receipts, and this one does. Tax is excluded from both sides;
> reconcile against `cogs_excl_tax`, **not** `cogs_incl_tax`.
>
> ### Test obligation, hardened (supersedes the AC-1/AC-6 bullet below)
>
> The parked card's red-first fixture was **rigged on every axis simultaneously** — the identity
> held on the authored fixture and broke on honest ones. The re-dispatch MUST:
>
> 1. Assert the identity by **calling `period-summary` itself** in the same test against the same
>    window and comparing responses — **never** against a hand-computed constant. A constant is
>    what let the rig survive.
> 2. Include, as distinct fixture cases, G6's five ready-made breakers — at minimum: a receipt with
>    an **unitemized delivery fee** (B1, the minimal breaker), a **non-COGS-category** event, a
>    **NULL-category** event, an **eligible pending** row, and a **linked-but-groupless** item
>    (the D2 "Ungrouped" pseudo-group, signed).
> 3. Keep the `period-summary` response **byte-unchanged** (regression guard).
>
> *(Original signed text follows, retained for provenance. Read it through the amendments above.)*

**Response shape:**

```json
{
  "window": { "from": "2026-04-27", "to": "2026-07-19", "weeks": 12 },
  "groups": [ { "id": "…uuid…", "name": "Proteins" } ],
  "cells": [
    { "week_start": "2026-07-13", "group_id": "…uuid…", "spend": 412.87 }
  ],
  "unlinked": [ { "week_start": "2026-07-13", "spend": 63.10 } ],
  "unlinked_total": 63.10
}
```

- `cells` is sparse (no zero rows); the client renders missing cells as 0.
- **Unlinked rule (FR-6b, signed at grill-back):** lines with `purchase_item_id IS NULL`
  are **excluded from group buckets** and returned per-week in `unlinked` plus the window
  `unlinked_total`, rendered as the "Unlinked $X" completeness note. This is consistent
  with the two existing endpoints: `period-summary` *reports* unlinked line IDs
  (`inventory/handler.go:1575-1601`, readiness gate `:1615`) and `menu-cogs` *excludes*
  them from allocation (`recipes/handler.go:81` — `AND pli.purchase_item_id IS NOT NULL`).
  Per-week unlinked (not just a window total) is what lets each weekly total reconcile to
  actual receipts (Glossary "Unlinked spend").
- **SQL sketch:**

  ```sql
  SELECT date_trunc('week', pe.event_date)::date AS week_start,
         pi.group_id,                       -- NULL for unlinked AND for groupless items (§4 flag 3)
         SUM((pli.quantity * pli.price) *
             COALESCE(pe.total / NULLIF(pe.total - pe.tax, 0), 1)) AS spend
  FROM purchase_line_items pli
  JOIN purchase_events pe ON pe.id = pli.purchase_event_id
  LEFT JOIN purchase_items pi ON pi.id = pli.purchase_item_id
  WHERE pe.event_date BETWEEN $1 AND $2
  GROUP BY 1, 2
  ORDER BY 1, 2;
  ```

  The handler splits `group_id IS NULL` rows into the `unlinked` array by whether
  `purchase_item_id` was NULL — see §4 flag 3 for the linked-but-groupless wrinkle
  (`purchase_items.group_id` is nullable, `0024_inventory.sql:30`).
- **Red-first Go test (AC-1):** ≥8-week × ≥2-group seeded fixture; every returned cell =
  hand-computed tax-prorated SUM; unlinked excluded from buckets but present in
  `unlinked`; reconciliation identity holds; `period-summary` response byte-unchanged.

### 2.3 `GET /api/v1/inventory/cost` (FR-3, A2)

**Response shape:**

```json
{
  "window": { "from": "2026-04-27", "to": "2026-07-19", "weeks": 12 },
  "rows": [
    {
      "menu_item_id": "…uuid…",
      "menu_item_name": "Salmon Bowl",
      "menu_group": "Bowls",
      "units_sold": 143,
      "revenue": 1287.00,
      "ingredient_cost_total": 402.11,
      "margin": 884.89,
      "food_cost_pct": 31.24,
      "unallocated": null
    },
    {
      "menu_item_id": "…uuid…",
      "menu_item_name": "Comped Special",
      "units_sold": 4,
      "revenue": 0,
      "ingredient_cost_total": 11.20,
      "margin": -11.20,
      "food_cost_pct": null
    }
  ],
  "movers": {
    "by_food_cost_pct": { "best": ["…ids…"], "worst": ["…ids…"] },
    "by_margin":        { "best": ["…ids…"], "worst": ["…ids…"] }
  }
}
```

- **Revenue is net-new:** `menu-cogs` exposes `units_sold` but not `gross_amount`
  (its `menu_units` CTE, `recipes/handler.go:84-89`, sums only `units_sold`). The Cost
  endpoint widens that CTE against `daily_menu_sales.gross_amount`
  (`0061_daily_menu_sales.sql` — `gross_amount NUMERIC(10,2) NOT NULL`):

  ```sql
  menu_sales AS (
    SELECT dms.menu_item_id,
           SUM(dms.units_sold)   AS units_sold,
           SUM(dms.gross_amount) AS revenue
    FROM daily_menu_sales dms
    WHERE dms.business_date BETWEEN $1 AND $2
    GROUP BY dms.menu_item_id
  )
  ```

  joined in place of `menu_units` atop the same `window_spend`/`alloc` CTEs
  (`recipes/handler.go:70-98`), then per row:

  ```sql
  ROUND((revenue - ingredient_cost_total)::numeric, 2)            AS margin,
  CASE WHEN revenue = 0 THEN NULL
       ELSE ROUND((ingredient_cost_total / revenue * 100)::numeric, 2)
  END                                                             AS food_cost_pct
  ```
- **Zero-revenue rule (A2, signed):** `revenue = 0` → `food_cost_pct` is JSON `null`
  (rendered "—"), never a divide-by-zero or ∞; the row keeps `units_sold` +
  `ingredient_cost_total` (and its negative margin) so INV-B never shows a false number.
- **Both movers orderings (grill-back resolution, Glossary "Mover"):**
  `by_food_cost_pct` — best = lowest %, worst = highest %, `null`-% rows excluded from
  this ranking (no % to rank on); `by_margin` — best = highest margin dollars, worst =
  lowest (comped rows *do* participate — a negative margin is a real worst-mover).
  Server-computed so the two strips are deterministic and testable in Go.
- **No-recipe / partial-allocation rows:** mirror `menu-cogs`'s unallocated reason
  strings (the State-Enumeration "sales but no recipe" row) rather than inventing new
  vocabulary — revenue + units shown, cost/margin marked, never a silent 0.
- **Sparse prod (A7, accept-sparse-prod):** prod `TOAST_SYNC_INTERVAL=0` may leave
  `daily_menu_sales` empty → `rows: []` and the tab's honest low-data card is the
  accepted prod PASS.
- **Red-first Go test (AC-3):** margins/food-cost-% match a hand-computed fixture to the
  cent; both movers orderings asserted; zero-revenue row returns `null`; `menu-cogs`
  response byte-unchanged.

## 3. Convergence coverage contract (FR-8/FR-9, INV-D) — codifying the landed matrix

The Trust track has already landed the systematic matrix in `tests/sync.spec.js`
(header comment at `tests/sync.spec.js:1880-1966`). This section promotes that
implementation into the **contract**: the matrix below is the coverage denominator; the
named tests are the proof.

### 3.1 Axes

`{op-type} × {editor} × {derived-view}` — asserted on the **second device**, live and
after catch-up:

- **op-type** ∈ `SET_FIELD` (FLD), `SUBMIT_CHECKLIST` (SUB), `APPROVE_ITEM` (APR),
  `REJECT_ITEM` (RJT)
- **editor** ∈ assignee (ASG), non-assignee admin (ADM — reaches the checklist only via
  the admin view-all clause; receives ops only via the ESC-1 admins-union +
  author-inclusion fan-out, `backend/internal/sync/listener.go:59-72`)
- **derived-view** ∈ field-value (VAL), correction-banner (BAN), edit-vs-readonly (ERO),
  list-progress-count (CNT)

### 3.2 The 32-cell table (24 covered / 8 N/A) — from `tests/sync.spec.js:1905-1953`

| Cell | Status | Proof / N/A reason |
|---|---|---|
| FLD-ASG-VAL | covered (pre-existing) | SYN-03 + FLD-LIVE-01/02 + W-3 per-type matrix |
| FLD-ASG-BAN | **N/A** | a SET_FIELD op carries no rejection-state change; banners derive solely from `submission.rejections`, which only REJECT_ITEM creates |
| FLD-ASG-ERO | **N/A** | `fillState.readonly` derives only from submission status; SET_FIELD never changes it |
| FLD-ASG-CNT | covered (pre-existing) | LST-17 + 'MX Progress' + 'MX Denom' |
| SUB-ASG-VAL | covered (new) | MTX-SUB-ASG-VAL/ERO |
| SUB-ASG-BAN | covered (new) | MTX-RJT-ASG cycle (resubmit leg clears the banner) |
| SUB-ASG-ERO | covered (pre-existing) | 'MX Submit' + 'MX Unsubmit' (re-proven in MTX-SUB-ASG-VAL/ERO) |
| SUB-ASG-CNT | covered (new) | MTX-SUB-ASG-CNT |
| APR-ASG-VAL | **N/A** | APPROVE_ITEM mutates no field_response rows |
| APR-ASG-BAN | **N/A** | the ⚠ banner renders only for status 'rejected'; approval feedback is FEEDBACK_NOTES, a distinct non-matrix view |
| APR-ASG-ERO | covered (pre-existing) | RJT-LIVE-03 (Approved flips live on device B) |
| APR-ASG-CNT | covered (new) | MTX-APR-ASG-CNT |
| RJT-ASG-VAL | covered (new) | MTX-RJT-ASG cycle |
| RJT-ASG-BAN | covered (pre-existing + new) | RJT-LIVE-01 **[ESC-2a]** + MTX-RJT-ASG-BAN-SUBSTEP **[ESC-3]** |
| RJT-ASG-ERO | covered (new) | MTX-RJT-ASG cycle **[ESC-2a]** |
| RJT-ASG-CNT | covered (pre-existing) | RJT-LIVE-02 **[ESC-2b]** |
| FLD-ADM-VAL | covered (new) | MTX-FLD-ADM-VAL **[ESC-1]** |
| FLD-ADM-BAN | **N/A** | same reason as FLD-ASG-BAN |
| FLD-ADM-ERO | **N/A** | same reason as FLD-ASG-ERO |
| FLD-ADM-CNT | covered (new) | MTX-FLD-ADM-CNT |
| SUB-ADM-VAL | covered (new) | MTX-SUB-ADM-VAL/ERO |
| SUB-ADM-BAN | covered (new) | MTX-RJT-ADM cycle |
| SUB-ADM-ERO | covered (new) | MTX-SUB-ADM-VAL/ERO |
| SUB-ADM-CNT | covered (new) | MTX-SUB-ADM-CNT |
| APR-ADM-VAL | **N/A** | same reason as APR-ASG-VAL |
| APR-ADM-BAN | **N/A** | same reason as APR-ASG-BAN |
| APR-ADM-ERO | covered (new) | MTX-APR-ADM-ERO |
| APR-ADM-CNT | covered (new) | MTX-APR-ADM-CNT |
| RJT-ADM-VAL | covered (new) | MTX-RJT-ADM cycle |
| RJT-ADM-BAN | covered (new) | MTX-RJT-ADM cycle |
| RJT-ADM-ERO | covered (new) | MTX-RJT-ADM cycle |
| RJT-ADM-CNT | covered (new) | MTX-RJT-ADM-CNT |

`SAVE_TEMPLATE`/`ARCHIVE_TEMPLATE` sit deliberately outside this 4-op matrix — covered by
the W-3 blocks plus `repro-cut-task.spec.js` / `broadcast-rerender.spec.js` (the prior
cycle's editprop contract).

### 3.3 Escaped-defect → cell mapping (FR-9, AC-9)

| Escape | Would-have-caught cell(s) | Red-first evidence (A6 — recorded historical runs) |
|---|---|---|
| **ESC-1** cross-user access | FLD-ADM-VAL (MTX-FLD-ADM-VAL) | `sync/access_test.go` `TestResolveEntityAccess_AdminReceivesLiveOps` — red on pre-fix `ResolveEntityAccess` |
| **ESC-2a** rejection reason not live | RJT-ASG-BAN + RJT-ASG-ERO | RJT-LIVE-01 |
| **ESC-2b** observer count frozen | RJT-ASG-CNT | RJT-LIVE-02 |
| **ESC-3** sub-step rejection dead-end | RJT-ASG-BAN sub-step variant (MTX-RJT-ASG-BAN-SUBSTEP) | APR-SUBSTEP-0718 |

3/3 escapes carry ≥1 cell; 0 lack one (Product KR2).

### 3.4 Standing extension rule

The matrix is a **living denominator**: introducing a new op type, a new editor class, or
a new derived view **obliges extending the table** — every new cell is either covered by
a named red-first test or carries a written N/A reason in the `sync.spec.js` header
comment (the canonical registry this section codifies). A cell with neither is a
violation of this contract, auditable at any cycle gate.

## 4. Where the PRD and the code disagree — flags found while grounding this draft

1. **"`inventory` slug registration in `hq_apps`" (FR-0/FR-5) is already done** —
   migration `0024_inventory.sql:58-60` and the startup upsert `db.go:52-64` both
   register it. The only registration Feature cards owe is the **new per-tab identifiers**
   (two seed lines under option i; nothing under option ii).
2. **Stale gating intent in the stubs:** `inventory.html:273`/`:280` carry
   `TODO: gate … to manager+ via backend roles (INTG-01)` — a role-based plan superseded
   by the signed per-tab-grant convention. The build card should delete these comments.
3. **Linked-but-groupless lines are a hole in FR-6b's dichotomy.** The Glossary defines
   unlinked as "no `purchase_item_id`, hence no `group_id`" — but a *linked* item can also
   have `group_id IS NULL` (`purchase_items.group_id` is nullable with
   `ON DELETE SET NULL`, `0024_inventory.sql:30`; and this is not hypothetical — the
   `'(no itemized receipt)'` sentinel item is **seeded with `group_id NULL`** in every
   database, `0064_no_itemized_receipt_seed.sql:16-22`, plus the SET NULL path fires
   whenever a group is deleted). Folding these into "Unlinked $X"
   would misstate the note's meaning; dropping them breaks the AC-6 reconciliation
   identity. **Recommended rule (not signed — see §5 D2):** bucket them as an explicit
   `"Ungrouped"` pseudo-group so the identity holds and the note stays truthful.
4. **Role-vocabulary mismatch (pre-existing, observed):** `app_permissions.role` is
   CHECK-constrained to `('admin','manager','team_member')` (`0005_app_permissions.sql`)
   while `GetUsersForAlerts` matches users against `'crew'/'manager'/'admin'` roles
   (`users/db.go:198-211`). Role-grants for a "crew" population must use `team_member`
   rows. No action this cycle; recorded so the Users-UI card doesn't trip on it.
5. **Superadmin implicit visibility:** `queryAllApps` (`me/handler.go:80-100`) gives
   superadmins every enabled app regardless of grants — §1.2 rule 4 makes
   `RequirePermission` mirror this deliberately (superadmin bypasses the grant check),
   otherwise a superadmin would see a tab (via `/me/apps`) whose endpoint 403s.
6. **Tab-grant-without-app-grant** (§1.6) — consistent with FR-5 but worth the operator's
   eyes: a Trends grant alone leaves the launcher tile hidden while the direct URL works.

## 5. OPEN DECISION A4 — the operator's call (PRD FR-6a; this draft does NOT choose)

The observable per-tab rule (§1.2) and the FR-5 test obligations (§1.7) are identical
under both options; only the storage/query/UI mechanics differ. **Check exactly one:**

- [x] **Option (i) — two dedicated per-tab slugs** (`inventory-trends`,
  `inventory-cost` in `hq_apps`; no migration; §1.4) ← **SIGNED 2026-07-20**
- [ ] **Option (ii) — per-tab sub-permission column** (`app_permissions.tab`;
  migration + down-migration + up→down→up proof + pre-deploy backup per NFR-3/INV-E; §1.5)

**Recommendation (draft's analysis — advisory only, the sign-off decides):**
**Option (i).** It reaches the identical observable contract with zero schema risk
(NFR-3 cost zero), reuses every existing station (`/me/apps`, the Users Access list, the
`queryUserApps` query shape) unchanged, and its feared launcher side effect turns out not
to exist (`index.html:121-130` ignores slugs without static tiles — verified §1.4). Option
(ii) is the "purer" model but pays a migration, a `/me` contract change, a new nested
Users-UI pattern, and the `SetAppPermissions` wipe hazard (§1.5) for benefits that only
materialize at a tab-count this app is unlikely to reach soon. If the Access-list growth
under (i) ever grates, migrating to (ii) later is a mechanical data move.

**D2 (small, surfaced by §4 flag 3) — linked-but-groupless lines in Trends:**

- [x] Bucket as an explicit `"Ungrouped"` pseudo-group (recommended — keeps the AC-6
  identity and the "Unlinked $X" note truthful) ← **SIGNED 2026-07-20**
- [ ] Other: ____________

## 6. What signing this authorizes (and what it does not)

Signing §5 + §8 unblocks **the Feature-track build WOs only** (Activity 4), implementing
against this design with atomic commits + red-first per PRD NFR-1:

1. **`inventory-tab-gating`** — §1 end-to-end for the chosen A4 option:
   `RequirePermission`, 403 envelope, `/me` visibility, Users grant surface, tab render
   guards, the FR-5 test pairs incl. mixed-grant (lands first; the endpoints mount
   inside its groups).
2. **`trends-spend-by-group-endpoint`** — §2.2 (FR-1/FR-6b/AC-1 + D2 rule).
3. **`trends-tab-frontend`** — FR-2: `#s5` inline SVG/CSS chart + table +
   `tests/states-trends.spec.js` forcing every Trends State-Enumeration row.
4. **`cost-margin-endpoint`** — §2.3 (FR-3/A2/AC-3, both movers orderings).
5. **`cost-tab-frontend`** — FR-4: `#s6` sortable table + two movers strips +
   `tests/states-cost.spec.js`, accept-sparse-prod honest empty state.

It does **not** authorize: any change to the service-token endpoints (out of scope, held
invariant), enabling Toast sync in prod, or the Trust-track cards (already ungated per
NFR-5). Until the operator signs, **Activity 4 stays blocked** — this document landing on
a run branch changes nothing by itself.

## 7. OpenSpec formalization flag

The roadmap/PRD word this gate as "an OpenSpec design change", but this repo has **no
`openspec/` directory** and the prior cycle's design gate shipped as a markdown design
doc (`designs/editprop-frozen-at-submit.md`) with an explicit note to the same effect.
Formalizing this document as a real OpenSpec change would require `openspec init` — a
structural repo decision (new tooling, new CLAUDE.md discipline section) that belongs to
an attended session, not an overnight run. If OpenSpec is later adopted, this doc is the
source to formalize from.

## 8. Sign-off

**What you get:** the two money tabs — where the spend goes week by week (Trends) and
which menu items earn or lose (Cost) — each behind its own on/off switch per person in
the Users app, enforced by the server, not just hidden. Every number on them is a sum you
can trace to receipt lines or sales rows; anything unlinked shows up as an honest
"Unlinked $X" instead of vanishing. And the live-sync test matrix that caught this
month's three escapes becomes a standing contract, not a one-off.

**What you decide this morning:** §5 — how a per-tab switch is stored (two extra rows in
the apps table with no schema change, or a new column with a migration). The switch
*behaves* identically either way. Plus the small D2 rule for items whose group was
deleted.

**Trade-offs:** option (i) slightly bends "one row = one app"; option (ii) buys purity
with a migration and more moving parts. Cost may ship to prod as an honest empty screen
(accept-sparse-prod — already accepted at the OKR session).

**Assumptions carried:** A1 (12-week window), A2 (zero-revenue → "—"), A3 (net-new
cookie-auth endpoints; service-token contracts untouched), A7 (accept-sparse-prod);
A4 is the open decision above.

**Status:** ✅ **SIGNED — operator, 2026-07-20 morning triage (ledger T-18).** §5 boxes
checked: **A4 = Option (i)** (two per-tab slugs) · **D2 = "Ungrouped" pseudo-group**.
Activity 4 (the 5 Feature build WOs, §6) is unblocked. Signature amendments — these
supersede the corresponding draft text above:

1. **Rider (b) REWRITTEN — umbrella grant semantics (operator rider, verbatim: "App
   grant = All tabs granted. They should not be considered separate objects.").** A
   whole-app grant includes every gated tab of that app automatically; per-tab grants
   exist to give narrower, tab-only access. `RequirePermission` therefore passes on
   **(tab slug ∨ whole-app `inventory` slug ∨ superadmin)** — the §1.4 middleware EXISTS
   query checks both slugs, not the tab slug alone. This replaces the draft's strict
   "whole-app grant does NOT imply tab grants" reading wherever it appears (§1.5
   trade-off text included).
2. **Rider (a) KEPT** — per-week `unlinked` array stands as drafted (§2.2).
3. **Rider (c) SIGNED as expected behavior** — tab grant without app grant: tile hidden,
   direct URL works (§1.6); the Users UI nudge stands.
4. **B5 fold-in — the `inventory-tab-gating` WO (§6 card 1) additionally gates the
   approve/reject endpoints** (`ApproveSubmissionHandler`/`RejectItemHandler`,
   `backend/internal/workflow/handler.go:728-753, 793+` — today ungated beyond login).
   Role rule specified at slate time (expected: approvers + admins/superadmins).

Per PRD AC-0, auditable from ledger timestamps: draft landed overnight-20260721
(`08e81e1`, merged `3d5fc17`), signed at the 2026-07-20 morning triage.
