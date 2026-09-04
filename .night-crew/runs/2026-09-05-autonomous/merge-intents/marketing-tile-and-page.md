# Merge intent — Card 1 · `marketing-tile-and-page` (run 20260905)

Branch: `wo-marketing-tile-and-page` (cut from dev @ 0670798, Wave 0 — lands before
every other card tonight). Card authority: slate-20260905 Card 1. Footprint: scanner UI
(`index.html`, `marketing.html` NEW) + redemption backend (SeedHQApps seed + grants) +
`sw.js` (precache 31→32, deliberate) + the CLAUDE.md precache-count line.

## Shared files touched

- `index.html` — the Marketing tile added to the grid + `'marketing.html':'marketing'`
  added to `TILE_SLUGS`. Slate: **exclusive to Card 1** tonight; no other card touches
  it.
- `marketing.html` (NEW) — the page shell Cards 5 and 6 extend later tonight. See
  "What must survive any merge" for the contract they inherit.
- `backend/internal/db/db.go` — `SeedHQApps()` grows two rows (`marketing`,
  `marketing-offline-override`) **inside the existing hq_apps INSERT block** (the
  grant-parity spec derives the seeded set by parsing that exact block — a second
  INSERT statement would silently narrow the guard), plus a first-registration-gated
  grant seed (CTE on the INSERT's RETURNING; see Red-first / engineering calls below).
- `tests/grant-enforcement-parity.spec.js` — two `NA_WITH_REASON` entries
  (`marketing`, `marketing-offline-override`): no marketing `/api/v1` endpoint exists
  tonight, so there is no RequirePermission mount to point the parity guard at. The
  entries are tripwired by PARITY-NA-FRESH: the moment a marketing route lands
  (Cards 5–7 or later), the N/A entries must be deleted and the routes gated.
- `tests/marketing.spec.js` (NEW) — this card's tile/permission/seed assertions
  (the red-first artifact). Cards 5–6 will extend this file; nothing in it hardcodes
  a count another card would move.
- `backend/internal/auth/permission_test.go` — one appended Go test block for the
  seed + grants (marketing rows registered; grants seeded; re-running SeedHQApps does
  NOT resurrect a revoked grant). Appended only; no existing test edited.
- `sw.js` + `CLAUDE.md` (precache-count line 31→32) — regenerated AFTER the
  marketing.html commit (build-sw.js reads git HEAD), committed together in the same
  change set. Cards 5 and 6 move the count again (each states its own move).
- `.night-crew/knowledge/roadmap.md` — one-line status flip of the
  `marketing-tile-and-page` card (PLANNED → DRAFTING), matching run 20260904's flip
  convention. No other roadmap line moves.
- `.night-crew/runs/2026-09-05-autonomous/merge-intents/marketing-tile-and-page.md` —
  this file (amended as evidence lands).
- `.night-crew/runs/2026-09-05-autonomous/card1-*.log` — committed whole gate logs,
  each ending in its `EXIT=` line.
- `users.html` — **nothing here.** The `marketing-offline-override` row renders
  through the existing access-editor path (`GetAppPermissions` returns every enabled
  hq_apps row; `renderAccess` gives it the standard 3 role toggles + individual-grant
  picker). Zero UI change needed — that is the point of the grant-surface convention.
- `night-crew.toml` — **nothing here.** `marketing.html` is a deliberately undeclared
  path → full-suite fallback (slate precondition flag; a `marketing` seam entry is
  Card 6's engineering call once `tests/marketing.spec.js` has coverage worth naming).
- `package.json` / `backend/internal/version/version.go` / `version.json` — **nothing
  here** (no version bump tonight — /save-project's job at deploy; parity must simply
  hold through the sw.js regen).
- `supabase/`, `sync-rxdb/`, `.night-crew/spikes/**`, `.night-crew/knowledge/spikes/**`
  — **nothing here** (spike dirs are read-only inputs).

## What must survive any merge

- **The marketing.html shell contract Cards 5–6 build on:** the four-tab structure
  (`#t1..#t4` buttons, `#s1..#s4` sections — Scan / Campaigns / Subscribers /
  Redemption stats), the `show(n)` tab switcher, the `#scanner-host` container inside
  the Scan section (Card 5 mounts the camera there), the shared CSS variable block +
  dark mode, the back link, and the PWA boilerplate (sw registration, log.js, ptr.js).
  Cards 5–6 extend inside the Scan section; they must not need to re-plumb the shell.
- **The `TILE_SLUGS` entry and the tile markup in `index.html`** — the permission gate
  for the tile is the `'marketing.html':'marketing'` mapping; dropping it makes the
  tile unconditionally visible (fails done_when (b)).
- **The whole SeedHQApps block including the grant-seed CTE** — the first-registration
  gating (grants seeded only when the hq_apps row is newly inserted) is what lets an
  operator revocation survive server restarts. Replacing it with an unconditional
  grant upsert reintroduces grant resurrection.
- **The two `NA_WITH_REASON` entries** until a marketing route exists — deleting them
  without mounting a gate reds PARITY-EQ; deleting them WHILE mounting the gate is the
  intended future edit.
- The roadmap status flip; the `card1-*.log` evidence files; this intent.

## What is safe to drop

- Nothing in this branch is scratch. `sw.js` is a committed artifact — if a later
  card's merge regenerates it on a newer HEAD, the regenerated file supersedes this
  card's copy (count moves are stated per card; Card 1's stated state is 32 with
  `marketing.html` present in the manifest).

## Red-first

(To be filled as evidence lands — greenfield-shaped RF per the slate: the
tile/permission assertions red on the pre-change tree because the behavior does not
exist yet.)

- Playwright: `tests/marketing.spec.js` — nothing here yet.
- Go: seed/grant test in `backend/internal/auth/permission_test.go` — nothing here
  yet.

## Engineering calls (recorded for the merge record)

1. **Grant storage shape (fork #12, operator-resolved surface):** the
   `offline_override` entitlement is an `hq_apps` grant-surface row,
   slug `marketing-offline-override`, reusing the established narrow-slug convention
   (design `prove-surface-gating-and-endpoints.md` §1.4 Option (i)) — so it flows
   through `/me/apps`, the Users access editor, and `auth.RequirePermission`
   unchanged, and is grantable per-role AND per-user out of the box. It is an
   **entitlement, not a gated tab**: when a handler eventually enforces it (Card 6's
   offline submit / Card 7's arbitration surface), the check must be
   `RequirePermission(pool, "marketing-offline-override")` with **NO umbrella slug** —
   holding the `marketing` app grant must never imply the override (that would derive
   the entitlement from tab access, which fork #12 explicitly forbids). Recorded in
   the SeedHQApps comment at the seed site.
2. **Seed mechanics:** grants seed only on first registration of the app row (CTE
   over the hq_apps INSERT's `RETURNING`), so `SeedHQApps` running on every startup
   cannot resurrect grants an operator revoked. Seeded set: `marketing` →
   roles admin, manager, team_member (§16 table — scan/redeem min role is
   team_member); `marketing-offline-override` → role admin only ("seeded true for
   admin users"; managers get it by explicit grant, per the entitlement-not-role
   semantic).
3. **Create/stats gates "inside the handler":** no marketing `/api/v1` handler exists
   tonight (Campaigns / Subscribers / Stats are labeled placeholders; the scanner's
   backend is Cards 5–7), so there is nothing to mount yet. The obligation is carried
   forward as the two tripwired `NA_WITH_REASON` entries in the parity spec — the
   suite reds the moment a marketing route lands ungated — plus the seed-site comment
   naming the §16 gates (view/create campaigns & stats = manager-tier, enforced
   in-handler when those endpoints land).
4. **users.html untouched:** the entitlement row inherits the standard gated-tab
   nudge ("Grant Marketing too…"). Its substance is correct for the entitlement — an
   offline-override grant without the marketing app grant is unreachable — so the
   card accepts the shared copy rather than special-casing the renderer.
5. **Placeholder copy** (mine per the PARK note): each placeholder section is a card
   with a "Soon" badge naming what it will hold, mirroring index.html's placeholder
   convention.
