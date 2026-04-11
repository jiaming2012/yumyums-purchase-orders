# Yumyums Purchasing

Weekly collaborative purchasing workflow for Yumyums. Team members add items to a shared weekly requisition throughout the week; a Sunday cutoff locks the list and generates per-vendor purchase orders.

## Status

Mock-only PWA. Server handlers not yet built.

## Stack

- **Backend:** Go + Postgres (co-located on existing Hetzner box, separate schema)
- **Frontend:** Plain HTML + HTMX, deployed as a minimal PWA
- **Reverse proxy:** Caddy (Let's Encrypt automatic HTTPS)
- **Scheduling:** stdlib `cron` / `time.AfterFunc` — *not* Temporal (see below)
- **Auth:** TBD — shared token vs. magic link

## Build vs. buy

Evaluated Zoho Creator at ~$32–40/mo for 4–5 users (Standard, annual). Rejected:

- Duplicates capabilities already in our stack (Postgres, Go, Hetzner).
- Standard plan caps at 50 cloud-function calls per user per day — tight for scheduled Deluge.
- Recurring per-seat cost for what is fundamentally a 400–600 LOC Go service.
- Reconsider only if we adopt Zoho One for Books/Inventory bundling.

## Data model

Four tables. `week_of` is always the Monday of that week.

- **`master_items`** — catalog: name (unique), category, unit, default_vendor, last_price, par_level, active.
- **`requisitions`** — week_of, item_id, qty, requested_by, notes, status (`open` | `locked` | `ordered`). Index on `(week_of, status)`.
- **`purchase_orders`** — week_of + vendor (unique together), total_est.
- **`po_lines`** — po_id + item_id (composite PK), qty, est_price.

PO generation is one transaction with three statements: insert grouped POs by vendor, insert po_lines, flip requisitions to `ordered`.

## Service shape

Three HTTP handlers + one scheduled job:

- `GET /requisition` — renders current week's form, lists active master items grouped by category with stepper qty inputs.
- `POST /requisition` — inserts rows where qty > 0, stamps `requested_by` from session, rejects if any row for that week is `locked`.
- `GET /po/:week` — runs the grouping query, renders printable per-vendor pages.
- **Cutoff cron** — Sundays 18:00 ET, flips current week's `open` rows to `locked`, then runs PO generation in the same transaction.

Estimated ~400–600 LOC including templates.

## Frontend (PWA)

Plain HTML + HTMX, no build step. Minimum-viable PWA: `manifest.json` + shell-caching service worker. Defer offline sync (IndexedDB queue) and push notifications until the team actually hits those pain points.

**Requires HTTPS on a real subdomain** (`order.yumyums.com`) for iOS to enable the install prompt, standalone mode, and the service worker. Caddy handles this automatically.

### UX decisions

- **Stepper buttons** (− N +) over plain number inputs — better for one-thumb use and wet hands in a kitchen environment.
- **Always-open category sections** with sticky headers (not collapsible) — faster for someone who knows what they're looking for.
- **Browse-and-fill** layout (all items visible with qty next to each), not search/autocomplete. Switch to search when the master list passes ~75 items.
- **Contributor initials on the locked view** — seeing "Lemons · 3 case · MK TR" shows when multiple people independently asked for the same thing, a soft signal that par level is too low.
- **Cutoff pill** prominently in the header so the deadline is always visible.

## Temporal: deliberately *not* used here

A separate Temporal server exists for the trading stack and future scheduled AI agents. Yumyums purchasing **does not** integrate with it.

**Why:** The cutoff is a deterministic two-statement transaction. No multi-step orchestration, no flaky external APIs, no human-in-loop, no long-running state. Plain cron is the right tool. Coupling Yumyums to the trading Temporal server would create a bad blast-radius dependency between the highest-stakes system (trading) and the lowest-stakes (food truck order form).

**Where Temporal *does* fit (future):** an "ordering assistant" agent layer that runs *after* the cron creates the locked PO — fetching vendor prices, calling an LLM for substitution suggestions, drafting emails, awaiting human approval. That has real activities, real retries, and a human-in-loop step. Clean split: **cron for deterministic DB transitions, Temporal for anything touching an LLM, external API, or human.**

## Deployment

- Same Hetzner box as the trading stack.
- Separate Postgres schema (not separate database).
- Caddy subdomain `order.yumyums.com` with auto Let's Encrypt.
- Marginal cost: zero.

## Open questions

1. **Auth model** — shared token in URL (zero friction, weak attribution) vs. magic link (real attribution, slightly more friction). Blocks the Go handlers.
2. **Push to Zoho Books/Inventory?** — if we end up there for accounting, the cron's final step could `POST` the PO into Inventory instead of (or in addition to) rendering it locally.
3. **Master items seed strategy** — manual entry, CSV import, or scraped from past Square sales data?

## v2 backlog

- **Out-of-stock flag on PO lines.** Tap a line item on the PO view to mark it as not-in-stock; notify the PO owner and auto-create a follow-up requisition row for next week's form.
- Search/filter bar on the form (when master list > ~30 items).
- Per-line notes ("get the bigger size if available").
- IndexedDB offline queue for dead-zone areas.
- Push notifications for cutoff reminders.
- Par-level alerts when multiple contributors hit the same item.

## Mock PWA — local dev

```
yumyums-pwa/
  index.html      # all three screens, tab switcher
  manifest.json   # PWA metadata
  sw.js           # shell-only service worker
  icons/
    icon-192.png
    icon-512.png
```

Deploy options ranked by speed:

1. **Cloudflare Pages drag-and-drop** — 2 min, free, real HTTPS, instant `*.pages.dev` URL. Best for the mock phase.
2. **Hetzner + Caddy** — `scp` to `/var/www/yumyums-mock`, add a Caddy block, reload. Worth doing now if DNS is ready since this is the eventual production path.
3. **Local `python3 -m http.server`** — sanity check only; iOS won't enable PWA features over plain HTTP on a LAN address.

On the phone: Safari/Chrome → Share → Add to Home Screen.
