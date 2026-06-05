# Phase 22: HQ Toast ingest — SFTP fetcher + menu_items + daily sales aggregate - Context

**Gathered:** 2026-06-03
**Status:** Ready for planning

<domain>
## Phase Boundary

HQ becomes a self-contained, peer Toast SFTP consumer. The phase delivers:

- An SFTP client + CSV parser that pulls `ItemSelectionDetails.csv` from Toast's file-sharing server on its own schedule
- Two new HQ tables (`menu_items`, `daily_menu_sales`) populated by aggregate-at-ingest (no raw sale rows stored)
- A background sync goroutine in `cmd/server` (12h cadence) + a one-shot `cmd/sync-toast` binary
- One read endpoint (`GET /api/v1/inventory/menu-items?since=YYYY-MM-DD`) for the new Menu tab
- A minimal Menu view in `inventory.html` as a new top-level tab between Stock and Setup

What this phase explicitly does NOT include: recipes, per-ingredient % attribution, COGS computation, menu-cogs endpoint, sales-processor-side rendering. All of that is Phase 999.2.

</domain>

<decisions>
## Implementation Decisions

### Cold-start backfill

- **D-01:** First-ever sync pulls **last 90 days** of Toast reports. ~90 SFTP downloads on cold start; enough for weekly comparisons and Phase 999.2 backtesting without making the first run an hour+.
- **D-02:** Cold start is detected by **`SELECT COUNT(*) FROM daily_menu_sales = 0`**. No bespoke flag, no env var for "is this cold start." Self-healing if someone TRUNCATEs the table.
- **D-03:** **Crash recovery: always restart the 90-day window from the start of the run.** Lean on `ON CONFLICT DO UPDATE` idempotency — re-pulling already-processed dates is cheap and correct. No `sync_state` bookkeeping table.

### Per-tick sync window + late corrections

- **D-04:** Each 12h tick **re-pulls the last 7 days**. Catches Toast's typical late-correction window (voids, comps, manager adjustments settle within 3–5 days).
- **D-05:** **Last-pull wins** — `daily_menu_sales` upserts via `ON CONFLICT (menu_item_id, business_date) DO UPDATE SET units_sold = EXCLUDED.units_sold, gross_amount = EXCLUDED.gross_amount, updated_at = now()`. Weekly reports always show Toast's most recent truth.

### Data shape (CSV → DB)

- **D-06:** **Voided lines (`Void? = true`) are excluded entirely** from `units_sold` and `gross_amount`. Phase 999.2's COGS attribution treats "units sold" as revenue-generating only.
- **D-07:** **Store the full 3-level menu hierarchy:** `menu` (TEXT NOT NULL), `menu_group` (TEXT NOT NULL), `menu_subgroup` (TEXT NULL). Subgroup is blank on most rows today but may become populated as the Toast menu grows; the cost of an extra nullable column is low.

### Menu UI placement

- **D-08:** **New top-level tab "Menu"** in `inventory.html`, placed **between Stock and Setup**. Tab order becomes: Purchases / Stock / **Menu** / Trends / Cost / Setup. Sets up Phase 999.2's Recipes tab to land either as a sub-section here or as a sibling tab.
- **D-09:** **Menu card content per item:** `name` + `menu_group` + `last_seen` + `this-week units sold`. Sort by `last_seen DESC`. Compact, scannable, surfaces freshness at a glance.

### SFTP operational behavior

- **D-10:** **SFTP unreachable handling:** Retry up to **3x with exponential backoff** (5s / 15s / 30s) on the same tick. If all 3 fail, log a single ERROR with context (host, last error), skip the tick, retry on the next 12h cycle. No email alerts in this phase.
- **D-11:** **Both trigger paths ship:**
  - `toast.StartWorker(ctx, cfg)` invoked from `cmd/server/main.go` (mirrors `receipt.StartWorker` + `purchasing.StartScheduler`). 12h ticker; runs immediately on startup, then on each tick.
  - `cmd/sync-toast` standalone binary for one-off backfills, manual reruns, or future external cron. Reuses the same `toast.RunIngest(ctx, pool, cfg, fromDate, toDate)` function the worker uses.
- **D-12:** **Missing/unreadable SSH key is fail-fast:** if `TOAST_SFTP_KEY_PATH` is unset OR the file at that path doesn't exist / isn't readable, the server logs ERROR and exits. **No dev escape.** User confirmed `creds/id_rsa` is portable to the dev laptop. Deviates from `receipt.StartWorker`'s graceful-skip pattern intentionally — Toast ingest is core enough to Phase 999.2 that running HQ without it is misleading.
- **D-13:** **Per-cycle logging:** one INFO summary line per ingest cycle in the form `toast ingest: dates=[YYYYMMDD..YYYYMMDD] items_upserted=N sales_rows_upserted=M duration=Xs`. Errors logged separately with the date that failed.

### Claude's Discretion

The following weren't asked because the answer is either codebase-convention or low-impact:

- **Env var names:** `TOAST_SFTP_KEY_PATH` (no default), `TOAST_SFTP_USER` (default `YumYumsExportUser`), `TOAST_SFTP_HOST` (default `s-9b0f88558b264dfda.server.transfer.us-east-1.amazonaws.com:22`), `TOAST_EXPORT_ID` (default `113866`), `TOAST_SYNC_INTERVAL` (default `12h`, parsed by `time.ParseDuration`; `0` disables the worker).
- **Retry backoff:** 5s / 15s / 30s. Cap each attempt with a 30s connect timeout (mirrors sales-processor's `Timeout: 30 * time.Second`).
- **Ticker behavior:** tick immediately on startup, then `time.NewTicker(interval)`. Mirrors `purchasing.StartScheduler`.
- **Migration numbering:** `0060_menu_items.sql` and `0061_daily_menu_sales.sql`. Down migrations included per convention.
- **`menu_items.last_seen` update rule:** bump to `MAX(business_date)` for that item every time a CSV row appears with positive `units_sold` for it. (Voided-only rows don't bump.)
- **Schema details:** UUIDs via `gen_random_uuid()`; `daily_menu_sales` PK is composite `(menu_item_id, business_date)`; `daily_menu_sales.updated_at` TIMESTAMPTZ defaults `now()` and updates on upsert; `gross_amount` is `NUMERIC(10,2)` (rounded at ingest, same convention as `purchase_events.total`).
- **Endpoint pagination:** `GET /menu-items?since=` returns all rows newer than `since` ordered by `last_seen DESC`. No pagination — the table is small (low hundreds of items max).
- **UI rendering:** plain vanilla JS following the existing inventory.html idioms. New tab button, new section, no new external deps.

### Folded Todos

None — `todo.match-phase 22` returned zero matches.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and contract
- `.planning/ROADMAP.md` Phase 22 entry — full scope statement, acceptance criteria
- `.planning/phases/21-cogs-in-sales-processor-report-receipt-completeness-gate-bef/21-SALES-PROCESSOR-CONTRACT.md` — service-token mounting precedent (Phase 22 endpoint is cookie-auth, NOT service-token — but the chi.Group pattern is the same)
- `~/.claude/projects/-Users-jamal-projects-yumyums-hq/memory/project_hq_toast_ingest.md` — architectural rationale (why HQ owns its Toast fetch, why same SSH key)

### Code to port (highest priority)
- `/Users/jamal/projects/yumyums/sales-processor/sftp/default.go` — **port verbatim** to `backend/internal/toast/sftp.go`. Reuse the `Config` struct, `New(config Config)`, `Download(filePath)`, and `Close()` methods.
- `/Users/jamal/projects/yumyums/sales-processor/main.go` lines 210–280 (`fetchToastCSVReports`) — reference for the per-date download loop. HQ's version replaces the file-write with a stream parse + DB upsert.

### Codebase analogs (mirror these patterns)
- `backend/internal/receipt/worker.go` — analog for `toast.StartWorker(ctx, cfg)` factory shape, graceful-skip-on-empty-config (DEVIATION: Phase 22 fails fast on missing key, see D-12)
- `backend/internal/purchasing/scheduler.go` — analog for goroutine pattern (immediate tick + `time.NewTicker`)
- `backend/cmd/import-notion/main.go` — CSV parse precedent using stdlib `encoding/csv`
- `backend/cmd/seed/` — one-shot binary structure for `cmd/sync-toast`

### Schema convention
- `backend/internal/db/migrations/0024_inventory.sql` — BEGIN/COMMIT, IF NOT EXISTS, UUID PKs, ON DELETE CASCADE conventions
- `backend/internal/inventory/types.go` lines 62–116 — handler-side struct + json-tag convention (the read endpoint will mirror this)

### HTTP wiring
- `backend/cmd/server/main.go` lines 270–340 — env-load + chi route group structure. The new `menu-items` endpoint mounts inside the existing cookie-auth group (it's UI-facing, not service-to-service).

### Environment / SFTP credentials
- `/Users/jamal/projects/yumyums/sales-processor/creds/id_rsa` — same key file HQ uses. Operator copies/symlinks into HQ's `backend/creds/id_rsa` (or wherever `TOAST_SFTP_KEY_PATH` points).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pgxpool.Pool` already threaded through every handler factory and worker.
- `writeJSON` / `writeError` helpers in `backend/internal/inventory/handler.go`.
- `db.Migrate(pool)` runs migrations on startup — Phase 22's new SQL files plug in automatically.
- `golang.org/x/crypto` already in `go.mod` as an indirect dep — SSH support is one direct import away.

### Established Patterns
- **Worker factory:** `func StartWorker(ctx context.Context, cfg WorkerConfig)` — package-owned config struct, graceful-skip-on-empty-required-fields, immediate-tick-then-ticker. Phase 22 mirrors this with `toast.StartWorker(ctx, toast.Config{...})`, deviating only on the missing-key handling (fail-fast vs graceful skip).
- **CSV ingest:** Stdlib `encoding/csv` only (see `import-notion/main.go`). No third-party CSV library.
- **Migration pattern:** Sequential numbered SQL files with `BEGIN; ... COMMIT;` and matching Down inside the same file. Next numbers: 0060, 0061.

### Integration Points
- **New env load + worker start:** in `cmd/server/main.go`, after the existing `purchasing.StartScheduler(ctx, pool)` block. Load `TOAST_*` env vars, validate `TOAST_SFTP_KEY_PATH` exists/readable (fail-fast per D-12), construct `toast.Config{}`, call `toast.StartWorker(ctx, cfg)`.
- **New endpoint:** in `r.Group(func(r chi.Router){ r.Use(auth.Middleware(...)) })` (the existing cookie-auth group), add `r.Get("/inventory/menu-items", inventory.ListMenuItemsHandler(pool))`. **Not** under the Phase 21 service-token group — this endpoint is for the HQ UI, not for sales-processor.
- **New tab in inventory.html:** insert "Menu" between Stock and Setup in the tab bar; new content section with `id="s-menu"`; fetch from `/api/v1/inventory/menu-items?since=<7-days-ago>` on tab open.

### New Dependencies
- `github.com/pkg/sftp` — needs `go get` + addition to `go.mod` (sales-processor already uses it; same version).
- `golang.org/x/crypto/ssh` — promote from indirect to direct import.

</code_context>

<specifics>
## Specific Ideas

- **Toast SFTP data shape:** 510 daily directories exist on disk at `/Users/jamal/projects/yumyums/sales-processor/output/toast_reports/{YYYYMMDD}/` (2023-08-25 → 2026-05-31). Each contains 7 reports; Phase 22 only reads `ItemSelectionDetails.csv`. The 5/31/26 directory has 20 distinct menu items.
- **CSV columns Phase 22 actually uses (subset of 32):** `Master Id`, `Menu Item` (name), `Menu`, `Menu Group`, `Menu Subgroup(s)`, `Sent Date` (parse for business_date), `Qty` (sum into `units_sold`), `Gross Price` (sum into `gross_amount`), `Void?` (filter out true rows).
- **Toast SFTP coordinates:** host `s-9b0f88558b264dfda.server.transfer.us-east-1.amazonaws.com:22`, user `YumYumsExportUser`, exportId path prefix `/113866/{YYYYMMDD}/`. Same as sales-processor's `main.go` constants.
- **User confirmed:** SSH key portable to dev laptop — Phase 22 can fail-fast on missing key without breaking dev flow.

</specifics>

<deferred>
## Deferred Ideas

These came up but belong in other phases or got explicitly rejected for Phase 22:

- **Email alerts on SFTP failures** (Area 5) — Out of scope; log-only is enough for now. Revisit if Toast outages become a real operational pain point.
- **Voided units tracked in a separate `voided_units` column** (Area 3) — Future enhancement if waste/comp pattern analysis becomes interesting. Schema can add the column later without breaking existing readers.
- **30-day sparkline trend per item in the Menu view** (Area 4) — Heavier render. Could land in Phase 999.2 alongside the Recipes UX when there's already a rich Menu surface.
- **`sync_state` bookkeeping table for fine-grained resume** (Area 1) — Add only if cold-start failures become recurring AND idempotent restarts prove too slow. Premature for v1.
- **Storing raw Toast sale rows in HQ** — Rejected by the architectural decision in `project_hq_toast_ingest.md`. Sales-processor remains the source of truth for per-order analytics; HQ stays aggregate-only.

### Reviewed Todos (not folded)

None — no relevant todos surfaced by `todo.match-phase 22`.

</deferred>

---

*Phase: 22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat*
*Context gathered: 2026-06-03*
