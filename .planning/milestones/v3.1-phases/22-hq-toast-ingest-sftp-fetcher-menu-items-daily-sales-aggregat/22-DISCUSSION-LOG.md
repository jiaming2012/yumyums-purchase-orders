# Phase 22: HQ Toast ingest — Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `22-CONTEXT.md` — this log preserves the alternatives considered.

**Date:** 2026-06-03
**Phase:** 22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat
**Areas discussed:** Cold-start backfill range, Per-tick date strategy + late-corrections, Voids + Menu hierarchy, Menu UI placement, SFTP operational concerns

---

## Cold-start backfill range

### Q1: How far back should the first ingest pull?

| Option | Description | Selected |
|--------|-------------|----------|
| Last 90 days | Heavy enough for weekly comparisons + a quarter of trend data. First sync pulls ~90 dirs (~10 min). | ✓ |
| Last 365 days | Full year of seasonality. First sync pulls ~365 dirs (~40 min). | |
| All available (~510 days) | Everything Toast has retained. First sync pulls all ~510 dirs (~hour+). | |
| Yesterday-forward only | First sync only pulls 1 day; data accumulates going forward. | |

**Captured as:** D-01

### Q2: How should HQ detect that it's a cold start vs ongoing sync?

| Option | Description | Selected |
|--------|-------------|----------|
| Empty `daily_menu_sales` = cold start | If the table has zero rows, treat as cold start and backfill 90 days. | ✓ |
| `TOAST_BACKFILL_DAYS` env var with default 90 | Env-driven. Pull last N days every server start. | |
| Explicit flag on sync-toast binary | `cmd/sync-toast --backfill=90` for one-off cold-start runs. | |

**Captured as:** D-02

### Q3: If the cold-start ingest crashes mid-backfill, what should restart behavior be?

| Option | Description | Selected |
|--------|-------------|----------|
| Resume from highest business_date in daily_menu_sales | On startup, look at MAX(business_date) and pull from there. | |
| Always restart from N days ago | Simpler logic, idempotent upserts make it safe. | ✓ |
| Persist progress in a sync_state table | Track each date as pending/done/failed in DB. | |

**Captured as:** D-03

---

## Per-tick date strategy + late corrections

### Q4: On each 12h scheduler tick, which date range should be pulled?

| Option | Description | Selected |
|--------|-------------|----------|
| Last 7 days | Re-pull last week's reports. Catches typical late corrections. | ✓ |
| Last 3 days | Smaller window, faster tick. Misses corrections older than 3 days. | |
| Last 30 days | Maximum safety net for late corrections. ~30 downloads per tick. | |
| Yesterday + today only | Minimal. Doesn't handle late corrections at all. | |

**Captured as:** D-04

### Q5: When a re-pulled CSV shows different numbers than what's stored, what wins?

| Option | Description | Selected |
|--------|-------------|----------|
| Last pull wins — ON CONFLICT DO UPDATE | Trust Toast's most recent report. | ✓ |
| First pull wins — ON CONFLICT DO NOTHING | Stable historical record but ignores Toast corrections. | |
| Track both with pulled_at + previous_value columns | Audit trail. Heavier schema. | |

**Captured as:** D-05

---

## Voids + Menu hierarchy

### Q6: Should voided line items count toward units_sold / gross_amount?

| Option | Description | Selected |
|--------|-------------|----------|
| Exclude voids entirely | Skip rows where Void? = true. units_sold reflects what was actually paid for. | ✓ |
| Include voids in a separate voided_units column | Track but don't count. | |
| Include voids in units_sold (Toast's raw count) | Total activity, not net. | |

**Captured as:** D-06

### Q7: How much of the Menu → Menu Group → Menu Subgroup hierarchy to store?

| Option | Description | Selected |
|--------|-------------|----------|
| Menu + Menu Group only | Skip Subgroup — it's blank on most rows seen. | |
| All three (Menu, Menu Group, Menu Subgroup) | Full fidelity. Subgroup nullable. | ✓ |
| Menu Group only (flat) | Drop Menu too. Simplest schema. | |

**Captured as:** D-07

---

## Menu UI placement

### Q8: Where should the Menu view live in inventory.html?

| Option | Description | Selected |
|--------|-------------|----------|
| New top-level tab "Menu" between Stock and Setup | Most discoverable. Sets up Phase 999.2. | ✓ |
| New top-level tab "Menu" after Setup | End of tab bar. Less prominent. | |
| Subsection under Setup | Hides menu items behind Setup. | |
| Standalone modal triggered from Setup | Lightest UI footprint, worst discoverability. | |

**Captured as:** D-08

### Q9: What should the Menu view show per item?

| Option | Description | Selected |
|--------|-------------|----------|
| Name + Menu Group + last_seen + this-week units sold | Four fields. Compact card. Sorted by last_seen DESC. | ✓ |
| Above + total revenue this week | Adds gross_amount. Useful for top sellers. | |
| Above + sparkline of last-30-day units | Visual trend per item. Heavier render. | |
| Minimal — just name + last_seen | Bare list. | |

**Captured as:** D-09

---

## SFTP operational concerns

### Q10: If the Toast SFTP server is unreachable on a tick, what should happen?

| Option | Description | Selected |
|--------|-------------|----------|
| Log error, skip tick, retry on next 12h cycle | Graceful-skip, same as receipt.StartWorker. | (combined) |
| Log error + send email alert via Zoho SMTP | Active alerting via existing email setup. | |
| Retry 3x with backoff on the same tick, then skip | More resilient to transient blips. | ✓ |

**User's choice:** Both first and third combined — "Retry 3x with backoff on the same tick. Then Log error, skip tick, retry on next 12h cycle." Captured as D-10 with explicit 5s/15s/30s backoff and no email alerts.

### Q11: Background worker, separate binary, or both?

| Option | Description | Selected |
|--------|-------------|----------|
| Both — worker for routine sync, binary for one-off backfills | Best flexibility. | ✓ |
| In-process goroutine only | Simpler. Requires server restart to force re-sync. | |
| Binary + external cron only | No goroutine. Doesn't depend on server uptime. | |

**Captured as:** D-11

### Q12: When TOAST_SFTP_KEY_PATH is missing/unreadable, what should the server do?

| Option | Description | Selected |
|--------|-------------|----------|
| Log WARNING, skip Toast worker, server continues | Matches receipt.StartWorker behavior. | |
| Log ERROR and exit | Fail-fast — forces ops to fix the config. | (initial) |
| Log WARNING, expose endpoint returning 503 | Mirrors Phase 21's fail-closed pattern. | |

**User's choice:** "Log ERROR and exit." User noted: "I have run this on my laptop before, so should be able to port over creds/id_rsa." This locked the fail-fast approach for D-12.

### Q13 (follow-up): Should empty TOAST_SFTP_KEY_PATH be a dev escape?

| Option | Description | Selected |
|--------|-------------|----------|
| Empty TOAST_SFTP_KEY_PATH disables ingest | Sentinel value for "this env doesn't run ingest." | |
| TOAST_INGEST_ENABLED bool env, separate from key path | Two env vars. More explicit. | |
| Always log ERROR and exit on key issue — no dev escape | Production-grade parity. | ✓ |

**Captured as:** D-12 final form — no dev escape; production/dev parity enforced.

### Q14: What should each ingest cycle log?

| Option | Description | Selected |
|--------|-------------|----------|
| Summary only — one INFO line per cycle | Matches receipt.StartWorker style. | ✓ |
| Per-date detail — one INFO line per date processed | Verbose. 7 lines per tick (90 on cold start). | |
| Summary + structured event for each new menu item discovered | Loud signal when menu changes. | |

**Captured as:** D-13

---

## Claude's Discretion

The user explicitly accepted Claude's defaults for:
- Exact env var names (`TOAST_SFTP_KEY_PATH`, `TOAST_SFTP_USER`, `TOAST_SFTP_HOST`, `TOAST_EXPORT_ID`, `TOAST_SYNC_INTERVAL`)
- Retry backoff parameters (5s / 15s / 30s)
- Ticker startup behavior (tick immediately on startup, then NewTicker)
- Migration column ordering and `menu_subgroup` exact type (TEXT NULLABLE)
- Pagination on `/menu-items` endpoint (none — table is small)
- UI rendering approach (vanilla JS, no new deps)

## Deferred Ideas

- Email alerts on SFTP failures (rejected from Area 5 — log-only is enough for now)
- Voided units tracked in a separate column (rejected from Area 3 — schema can add later)
- 30-day sparkline trend per item (deferred to Phase 999.2 or later)
- `sync_state` table for fine-grained crash-resume bookkeeping (rejected from Area 1 — idempotent restarts cover it)
- Storing raw Toast sale rows in HQ (locked-out architectural decision per `project_hq_toast_ingest.md`)
