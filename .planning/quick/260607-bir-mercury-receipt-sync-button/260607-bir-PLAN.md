---
phase: 260607-bir
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - backend/internal/db/migrations/0067_receipt_sync_runs.sql
  - backend/internal/receipt/worker.go
  - backend/internal/receipt/worker_test.go
  - backend/internal/inventory/sync_receipts.go
  - backend/internal/inventory/sync_receipts_test.go
  - backend/cmd/server/main.go
  - inventory.html
  - tests/inventory.spec.js
autonomous: true
requirements:
  - SYNC-01
  - SYNC-02
  - SYNC-03

must_haves:
  truths:
    - "User taps 'Sync Receipts' button in inventory.html Purchases tab and an ingest cycle begins"
    - "Button is disabled and shows 'Syncing…' while a sync is in flight"
    - "Status survives a full page reload — reopening the Purchases tab still shows 'Syncing…' if the sync is still running"
    - "When the sync completes, a dismissable chip shows 'Last synced Xm ago — N processed, M pending review'"
    - "A second POST while a run is in flight is rejected with 409 (single-flight)"
    - "Polling pauses when the document is hidden and resumes on visibilitychange visible"
    - "A panic inside runIngestCycle still updates the receipt_sync_runs row to status=failed (no orphan running rows)"
  artifacts:
    - path: "backend/internal/db/migrations/0067_receipt_sync_runs.sql"
      provides: "receipt_sync_runs table with status enum and counts"
      contains: "CREATE TABLE receipt_sync_runs"
    - path: "backend/internal/receipt/worker.go"
      provides: "runIngestCycle refactored to return IngestResult struct + exported RunIngestCycle wrapper"
      contains: "type IngestResult struct"
    - path: "backend/internal/inventory/sync_receipts.go"
      provides: "POST /sync-receipts + GET /sync-receipts/status handlers with single-flight guard + panic-safe goroutine"
      exports: ["SyncReceiptsHandler", "SyncReceiptsStatusHandler"]
    - path: "inventory.html"
      provides: "Sync button + status chip above vendor filter in #s1, with visibility-aware polling"
      contains: "id=\"sync-receipts-btn\""
  key_links:
    - from: "backend/cmd/server/main.go"
      to: "inventory.SyncReceiptsHandler / SyncReceiptsStatusHandler"
      via: "chi route registration in /inventory group ~line 418"
      pattern: "sync-receipts"
    - from: "inventory.html (loadHistory / show(1))"
      to: "/api/v1/inventory/sync-receipts/status"
      via: "GET on Purchases tab activation + 3s poll while running"
      pattern: "sync-receipts/status"
    - from: "backend/internal/inventory/sync_receipts.go (goroutine)"
      to: "receipt.RunIngestCycle + receipt_sync_runs UPDATE"
      via: "defer recover() → UPDATE … status=failed on panic; UPDATE … status=done with counts on success"
      pattern: "defer func.*recover"
---

<objective>
Add a "Sync Receipts" button to the inventory.html Purchases tab that triggers the existing
Mercury receipt worker on demand, with durable status that survives page switches, full
reload, and PWA close/reopen.

Purpose: Today the receipt worker only runs on a 6h ticker. The owner needs to pull new
Mercury receipts immediately after a card swipe without waiting or restarting the server.

Output:
- New migration: receipt_sync_runs table
- Refactored runIngestCycle that returns processed/auto_created/pending_review/cached counts
- Two new endpoints: POST /api/v1/inventory/sync-receipts, GET /api/v1/inventory/sync-receipts/status
- Inventory Purchases tab UI: button + status chip with visibility-aware polling
- Backend Go tests + Playwright tests covering single-flight, panic recovery, and reload survival
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@./CLAUDE.md
@./backend/internal/receipt/worker.go
@./backend/internal/receipt/types.go
@./backend/internal/receipt/worker_test.go
@./backend/internal/inventory/handler.go
@./backend/cmd/server/main.go
@./inventory.html

<interfaces>
<!-- Key types and contracts already in the codebase. Use these directly — no exploration needed. -->

From backend/internal/receipt/types.go (line 60-70):
```go
type WorkerConfig struct {
    MercuryAPIKey    string
    AnthropicAPIKey  string
    Pool             *pgxpool.Pool
    SpacesPresigner  *s3.PresignClient
    SpacesEndpoint   string
    SpacesBucket     string
    Interval         time.Duration
    LookbackDays     int
}
```

From backend/internal/receipt/worker.go (line 58-239) — current signature:
```go
func runIngestCycle(ctx context.Context, cfg WorkerConfig) error
```
Already computes `autoCreated, pendingReview, skippedCached` locals and `len(txns)` as
`processed`. Plan 02 refactors this to return an `IngestResult` struct holding those four
counts plus the error.

Migration runner: backend/internal/db/migrations/*.sql, embedded via `embed.FS` in
backend/internal/db/db.go and run by `db.Migrate(pool)`. Migrations use goose Up/Down
format with `BEGIN; ... COMMIT;`. Latest migration is `0066_mercury_category_on_pending_purchases.sql`,
so the next is `0067_*`.

From backend/internal/inventory/handler.go (line 24-32):
```go
func writeJSON(w http.ResponseWriter, status int, v any)
func writeError(w http.ResponseWriter, status int, msg string)
```

From backend/cmd/server/main.go ~line 418 — the inventory route group is auth-gated
under `r.Route("/inventory", func(r chi.Router) { ... })`. New routes go inside this
group so they pick up the same Bearer/cookie auth middleware that already wraps it.

Worker wiring in backend/cmd/server/main.go ~line 535:
```go
receiptCfg := receipt.WorkerConfig{
    MercuryAPIKey:   os.Getenv("MERCURY_API_KEY"),
    ...
}
```
The same `receiptCfg` value is what the new POST handler will close over.

From inventory.html (line 230-235):
```html
<div id="s1" style="display:none">
  <select id="vendor-filter" class="filter-select">
    <option value="">All Vendors</option>
  </select>
  <div id="history-list"></div>
</div>
```
Sync button + chip go ABOVE the `<select id="vendor-filter">`.

From inventory.html (line 332) — tab activation hook:
```js
if(n===1){loadHistory();}
```
Plan 03 adds `refreshSyncStatus()` next to `loadHistory()`.

Playwright stub pattern (tests/inventory.spec.js line 519+):
```js
await page.route('**/api/v1/inventory/menu-items*', async route => {
  await route.fulfill({ status: 200, contentType: 'application/json', body: '...' });
});
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Migration + receipt worker refactor — durable run table and IngestResult</name>
  <files>
    backend/internal/db/migrations/0067_receipt_sync_runs.sql,
    backend/internal/receipt/worker.go,
    backend/internal/receipt/worker_test.go
  </files>
  <action>
    A. Create migration `backend/internal/db/migrations/0067_receipt_sync_runs.sql` with goose Up/Down
    blocks wrapped in BEGIN/COMMIT (mirror the style of 0066). Up:

    ```sql
    -- +goose Up
    BEGIN;
    CREATE TABLE receipt_sync_runs (
      id              BIGSERIAL PRIMARY KEY,
      started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
      finished_at     TIMESTAMPTZ,
      status          TEXT NOT NULL CHECK (status IN ('running','done','failed')),
      processed       INTEGER NOT NULL DEFAULT 0,
      auto_created    INTEGER NOT NULL DEFAULT 0,
      pending_review  INTEGER NOT NULL DEFAULT 0,
      cached          INTEGER NOT NULL DEFAULT 0,
      error           TEXT,
      triggered_by    TEXT NOT NULL DEFAULT 'manual'
    );
    -- Single-flight guard: partial unique index — at most one running row.
    CREATE UNIQUE INDEX receipt_sync_runs_single_running
      ON receipt_sync_runs ((1)) WHERE status = 'running';
    -- Latest-row lookup.
    CREATE INDEX receipt_sync_runs_started_at_desc
      ON receipt_sync_runs (started_at DESC);
    COMMIT;

    -- +goose Down
    BEGIN;
    DROP TABLE receipt_sync_runs;
    COMMIT;
    ```

    B. In `backend/internal/receipt/worker.go`:
      1. Add a new exported type ABOVE `runIngestCycle`:
         ```go
         // IngestResult captures the counts produced by one ingest cycle.
         type IngestResult struct {
             Processed     int
             AutoCreated   int
             PendingReview int
             Cached        int
         }
         ```
      2. Change `runIngestCycle` signature to `func runIngestCycle(ctx context.Context, cfg WorkerConfig) (IngestResult, error)`.
         At every existing `return` path return an IngestResult — zero-value on early
         error/empty paths, fully populated at the final success path. Do NOT change the
         log line at line 236-237; keep it as a side-effect log.
      3. Add an exported wrapper for the inventory handler to call:
         ```go
         // RunIngestCycle runs one Mercury ingest cycle and returns the result counts.
         // Used by the on-demand sync endpoint; the background worker calls runIngestCycle directly.
         func RunIngestCycle(ctx context.Context, cfg WorkerConfig) (IngestResult, error) {
             return runIngestCycle(ctx, cfg)
         }
         ```
      4. Update the two `StartWorker` callers of `runIngestCycle` (lines 36 and 49) to
         discard the result: `if _, err := runIngestCycle(ctx, cfg); err != nil { ... }`.

    C. In `backend/internal/receipt/worker_test.go`, add ONE focused test
    `TestRunIngestCycle_NoTransactions_ReturnsZeroResult` that:
      - Skips if `testPool == nil`
      - Calls `resetReceiptFixtures(t)`
      - Builds a `WorkerConfig` with `MercuryAPIKey: ""` so the real path is avoided —
        OR (preferred) calls `runIngestCycle` with a config whose pool is set but Mercury
        key triggers FetchTransactions to error early; assert err is non-nil and the
        returned IngestResult is the zero value.

      The point of this test is to prove the new signature compiles and the zero-value
      contract holds. Heavier ingest-pipeline behavior is already covered by existing
      worker integration tests — do NOT expand the test surface here.

    Do NOT add any other behavior. The migration column set and goroutine logic are
    consumed in Task 2.
  </action>
  <verify>
    <automated>cd backend &amp;&amp; go build ./... &amp;&amp; go test ./internal/receipt/ -run TestRunIngestCycle_NoTransactions_ReturnsZeroResult -count=1</automated>
  </verify>
  <done>
    Migration file 0067 exists with goose markers. `runIngestCycle` returns `(IngestResult, error)`.
    Exported `RunIngestCycle` wrapper exists. `StartWorker` still compiles with discarded result.
    New test passes (or skips if DB unavailable). `go build ./...` clean.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Backend handlers — POST /sync-receipts (single-flight + panic-safe) and GET /sync-receipts/status</name>
  <files>
    backend/internal/inventory/sync_receipts.go,
    backend/internal/inventory/sync_receipts_test.go,
    backend/cmd/server/main.go
  </files>
  <behavior>
    - POST /sync-receipts when no row with status=running exists → 200 with
      `{id, status:"running", started_at}` and spawns a goroutine.
    - POST /sync-receipts when a row with status=running exists → 409 with
      `{"error":"sync_already_running"}` and does NOT spawn a goroutine.
    - Goroutine on success → UPDATE receipt_sync_runs SET status='done',
      finished_at=now(), processed=..., auto_created=..., pending_review=..., cached=...
      WHERE id=$1.
    - Goroutine on returned error → UPDATE status='failed', finished_at=now(),
      error=err.Error() WHERE id=$1.
    - Goroutine on panic → recover() and UPDATE status='failed', finished_at=now(),
      error="panic: <recovered value>" WHERE id=$1. NO running row left behind.
    - GET /sync-receipts/status → returns latest row JSON, or `null` body with 200 if
      table is empty.
  </behavior>
  <action>
    A. Create `backend/internal/inventory/sync_receipts.go` with:

    ```go
    package inventory

    import (
        "context"
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "time"

        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgxpool"
        "github.com/yumyums/hq/internal/receipt"
    )

    type syncRunRow struct {
        ID            int64      `json:"id"`
        StartedAt     time.Time  `json:"started_at"`
        FinishedAt    *time.Time `json:"finished_at"`
        Status        string     `json:"status"`
        Processed     int        `json:"processed"`
        AutoCreated   int        `json:"auto_created"`
        PendingReview int        `json:"pending_review"`
        Cached        int        `json:"cached"`
        Error         *string    `json:"error"`
        TriggeredBy   string     `json:"triggered_by"`
    }

    // IngestRunner is the function the handler calls. Real impl is receipt.RunIngestCycle.
    // Tests inject a stub.
    type IngestRunner func(ctx context.Context) (receipt.IngestResult, error)

    func SyncReceiptsHandler(pool *pgxpool.Pool, runner IngestRunner) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // Insert with status='running'. The partial unique index on status='running'
            // raises a unique violation if one already exists — that's our single-flight guard.
            var id int64
            var startedAt time.Time
            err := pool.QueryRow(r.Context(),
                `INSERT INTO receipt_sync_runs (status, triggered_by)
                 VALUES ('running', 'manual')
                 RETURNING id, started_at`,
            ).Scan(&id, &startedAt)
            if err != nil {
                // pgx wraps unique-violation as a *pgconn.PgError with Code "23505".
                // Cheap check: substring match avoids importing pgconn just for this.
                if isUniqueViolation(err) {
                    writeError(w, http.StatusConflict, "sync_already_running")
                    return
                }
                log.Printf("SyncReceipts insert: %v", err)
                writeError(w, http.StatusInternalServerError, "internal_error")
                return
            }

            // Detach the request context — the goroutine outlives the HTTP request.
            go runSyncGoroutine(pool, runner, id)

            writeJSON(w, http.StatusOK, map[string]any{
                "id":         id,
                "status":     "running",
                "started_at": startedAt,
            })
        }
    }

    func runSyncGoroutine(pool *pgxpool.Pool, runner IngestRunner, id int64) {
        ctx := context.Background()
        defer func() {
            if rec := recover(); rec != nil {
                msg := fmt.Sprintf("panic: %v", rec)
                _, _ = pool.Exec(ctx,
                    `UPDATE receipt_sync_runs
                     SET status='failed', finished_at=now(), error=$1
                     WHERE id=$2`, msg, id)
                log.Printf("SyncReceipts goroutine panic for run %d: %v", id, rec)
            }
        }()

        result, err := runner(ctx)
        if err != nil {
            _, updErr := pool.Exec(ctx,
                `UPDATE receipt_sync_runs
                 SET status='failed', finished_at=now(), error=$1
                 WHERE id=$2`, err.Error(), id)
            if updErr != nil {
                log.Printf("SyncReceipts failed-update for run %d: %v", id, updErr)
            }
            return
        }
        _, updErr := pool.Exec(ctx,
            `UPDATE receipt_sync_runs
             SET status='done', finished_at=now(),
                 processed=$1, auto_created=$2, pending_review=$3, cached=$4
             WHERE id=$5`,
            result.Processed, result.AutoCreated, result.PendingReview, result.Cached, id)
        if updErr != nil {
            log.Printf("SyncReceipts done-update for run %d: %v", id, updErr)
        }
    }

    func SyncReceiptsStatusHandler(pool *pgxpool.Pool) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            var row syncRunRow
            err := pool.QueryRow(r.Context(),
                `SELECT id, started_at, finished_at, status, processed,
                        auto_created, pending_review, cached, error, triggered_by
                 FROM receipt_sync_runs
                 ORDER BY started_at DESC
                 LIMIT 1`,
            ).Scan(&row.ID, &row.StartedAt, &row.FinishedAt, &row.Status,
                &row.Processed, &row.AutoCreated, &row.PendingReview, &row.Cached,
                &row.Error, &row.TriggeredBy)
            if err == pgx.ErrNoRows {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                w.Write([]byte("null"))
                return
            }
            if err != nil {
                log.Printf("SyncReceiptsStatus query: %v", err)
                writeError(w, http.StatusInternalServerError, "internal_error")
                return
            }
            writeJSON(w, http.StatusOK, row)
        }
    }

    func isUniqueViolation(err error) bool {
        // Cheap detection without importing pgconn — pgx error strings always
        // include the SQLSTATE "23505" for unique violations.
        if err == nil { return false }
        return contains(err.Error(), "23505")
    }

    func contains(haystack, needle string) bool {
        // small helper to keep imports lean; safe for ASCII tokens like "23505"
        return len(haystack) >= len(needle) && (haystack == needle || (len(haystack) > 0 && (indexOf(haystack, needle) >= 0)))
    }
    func indexOf(s, sub string) int {
        for i := 0; i+len(sub) <= len(s); i++ {
            if s[i:i+len(sub)] == sub { return i }
        }
        return -1
    }
    ```

    Note on isUniqueViolation: if reviewing while writing reveals that the codebase
    already imports `github.com/jackc/pgx/v5/pgconn` elsewhere, prefer the typed check:
    ```go
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) && pgErr.Code == "23505" { return true }
    ```
    Either is acceptable; use the typed version if the import is already common in the
    package. Do NOT add a new module dependency.

    B. Create `backend/internal/inventory/sync_receipts_test.go` with three tests that
    DO NOT depend on the real ingest pipeline — they use an injected `IngestRunner` stub:

    1. `TestSyncReceipts_SingleFlight_Returns409` — skip if testPool nil; reset table;
       call POST handler once via `httptest.NewRecorder` + chi handler → expect 200;
       call again immediately → expect 409 with `{"error":"sync_already_running"}`.
       After, manually UPDATE the row to done and assert a third POST succeeds.

    2. `TestSyncReceipts_Goroutine_UpdatesRowToDone` — inject a runner that returns
       `IngestResult{Processed: 7, AutoCreated: 2, PendingReview: 1, Cached: 4}, nil`.
       POST handler → poll the row by id with a 2s deadline until status != 'running'.
       Assert status='done', processed=7, auto_created=2, pending_review=1, cached=4,
       finished_at IS NOT NULL.

    3. `TestSyncReceipts_Goroutine_RecoversFromPanic` — inject a runner that does
       `panic("boom")`. POST → poll → assert status='failed', error LIKE 'panic: boom',
       finished_at IS NOT NULL.

    Use `runSyncGoroutine` directly (skip the HTTP layer) for the panic test if it's
    simpler — both approaches verify the same code path.

    Reset table at start of each test:
    ```go
    _, _ = testPool.Exec(ctx, `TRUNCATE receipt_sync_runs RESTART IDENTITY`)
    ```

    Add a package-level `var testPool *pgxpool.Pool` + TestMain mirroring
    `backend/internal/receipt/worker_test.go` (run `db.Migrate(pool)`).

    C. In `backend/cmd/server/main.go`, find the `r.Route("/inventory", ...)` block
    (~line 418-442). Add INSIDE the closure, immediately after the existing
    `r.Get("/purchases/pending", ...)` line so it lands with the other purchases
    routes:

    ```go
    r.Post("/sync-receipts", inventory.SyncReceiptsHandler(pool, func(ctx context.Context) (receipt.IngestResult, error) {
        return receipt.RunIngestCycle(ctx, receiptCfg)
    }))
    r.Get("/sync-receipts/status", inventory.SyncReceiptsStatusHandler(pool))
    ```

    The handler closes over `receiptCfg` declared at ~line 535. Verify the variable is
    in scope at line 418 — if `receiptCfg` is constructed AFTER the route block, move
    the route registration into a follow-up `r.Route("/inventory", ...)` extension
    block placed after `receiptCfg` is built, OR hoist `receiptCfg` construction above
    the route block. Whichever is the smaller diff. Add `"github.com/yumyums/hq/internal/receipt"`
    to imports if not already present (StartWorker is already called from main, so it
    almost certainly is).
  </action>
  <verify>
    <automated>cd backend &amp;&amp; go build ./... &amp;&amp; go test ./internal/inventory/ -run TestSyncReceipts -count=1 -v</automated>
  </verify>
  <done>
    All three sync_receipts tests pass (or skip if DB unavailable). `go build ./...` clean.
    `curl -X POST http://localhost:8080/api/v1/inventory/sync-receipts` (with auth) returns
    `{id,status:"running",started_at}`. Second POST returns 409 if first still running.
    `GET /api/v1/inventory/sync-receipts/status` returns the latest row or `null`.
  </done>
</task>

<task type="auto">
  <name>Task 3: Frontend — sync button, status chip, visibility-aware polling in inventory.html</name>
  <files>
    inventory.html,
    tests/inventory.spec.js
  </files>
  <action>
    A. CSS additions (append to the existing `<style>` block near the other Purchases tab styles, around the `.filter-select` rule at line 39):

    ```css
    .sync-bar{display:flex;flex-direction:column;gap:8px;margin-bottom:12px}
    .sync-btn{width:100%;padding:11px 14px;background:var(--info-bg);color:var(--info-tx);border:0.5px solid var(--brd);border-radius:8px;font-size:14px;font-weight:600;cursor:pointer;font-family:inherit;min-height:44px}
    .sync-btn:disabled{opacity:0.6;cursor:default}
    .sync-chip{display:flex;align-items:center;justify-content:space-between;gap:8px;padding:9px 12px;background:var(--card);border:0.5px solid var(--brd);border-radius:8px;font-size:13px;color:var(--mut)}
    .sync-chip.ok{color:var(--info-tx);background:var(--info-bg)}
    .sync-chip.err{color:var(--warn-tx);background:var(--warn-bg)}
    .sync-chip-dismiss{background:none;border:none;color:inherit;cursor:pointer;font-size:18px;line-height:1;padding:0 4px;min-width:24px;min-height:24px}
    ```

    B. HTML — replace lines 230-235 (the existing `#s1` block) with:

    ```html
    <div id="s1" style="display:none">
      <div class="sync-bar">
        <button id="sync-receipts-btn" class="sync-btn" type="button">Sync Receipts</button>
        <div id="sync-receipts-chip" style="display:none"></div>
      </div>
      <select id="vendor-filter" class="filter-select">
        <option value="">All Vendors</option>
      </select>
      <div id="history-list"></div>
    </div>
    ```

    C. JS — add a new section right above the `show()` function (~line 320). Use
    SCREAMING_SNAKE_CASE for constants, camelCase for functions, match the file's
    inline-compact style:

    ```js
    // ─── Receipt sync ──────────────────────────────────────────────────────
    var SYNC_STATE=null;      // latest row from GET /sync-receipts/status, or null
    var SYNC_POLL_TIMER=null;
    var SYNC_CHIP_DISMISSED_ID=null; // remembers which run's chip the user dismissed
    var SYNC_POLL_MS=3000;

    function renderSyncUI(){
      var btn=document.getElementById('sync-receipts-btn');
      var chip=document.getElementById('sync-receipts-chip');
      if(!btn||!chip)return;
      var s=SYNC_STATE;
      var running=s&&s.status==='running';
      btn.disabled=!!running;
      btn.textContent=running?'Syncing…':'Sync Receipts';
      // Chip rules:
      //  - running: hide chip (button label carries the state)
      //  - failed: red chip, not dismissable until user retries
      //  - done: green chip with summary + ×; hide if user already dismissed this run id
      if(!s||running){chip.style.display='none';chip.className='';chip.innerHTML='';return;}
      if(s.status==='failed'){
        chip.style.display='';
        chip.className='sync-chip err';
        chip.innerHTML='<span>Sync failed: '+escapeHTML(s.error||'unknown error')+'</span>';
        return;
      }
      if(s.status==='done'){
        if(SYNC_CHIP_DISMISSED_ID===s.id){chip.style.display='none';return;}
        var mins=Math.max(0,Math.floor((Date.now()-new Date(s.finished_at||s.started_at).getTime())/60000));
        var when=mins===0?'just now':(mins+'m ago');
        chip.style.display='';
        chip.className='sync-chip ok';
        chip.innerHTML='<span>Last synced '+when+' — '+s.processed+' processed, '+s.pending_review+' pending review</span>'+
                      '<button class="sync-chip-dismiss" data-action="dismiss-sync-chip" aria-label="Dismiss">×</button>';
      }
    }

    function escapeHTML(str){
      return String(str).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c];});
    }

    async function refreshSyncStatus(){
      try{
        var res=await fetch('/api/v1/inventory/sync-receipts/status',{credentials:'include'});
        if(!res.ok){return;}
        var body=await res.json();
        SYNC_STATE=body; // server returns the row object or null
        renderSyncUI();
        if(SYNC_STATE&&SYNC_STATE.status==='running'){startSyncPoll();}
        else{stopSyncPoll(); if(SYNC_STATE&&SYNC_STATE.status==='done'){loadHistory();}}
      }catch(e){/* offline / transient; no UI noise */}
    }

    function startSyncPoll(){
      if(SYNC_POLL_TIMER)return;
      if(document.hidden)return; // visibilitychange listener will start it later
      SYNC_POLL_TIMER=setInterval(refreshSyncStatus,SYNC_POLL_MS);
    }
    function stopSyncPoll(){
      if(SYNC_POLL_TIMER){clearInterval(SYNC_POLL_TIMER);SYNC_POLL_TIMER=null;}
    }

    document.addEventListener('visibilitychange',function(){
      if(document.hidden){stopSyncPoll();}
      else if(SYNC_STATE&&SYNC_STATE.status==='running'){startSyncPoll();refreshSyncStatus();}
    });

    async function triggerSync(){
      var btn=document.getElementById('sync-receipts-btn');
      if(btn)btn.disabled=true;
      try{
        var res=await fetch('/api/v1/inventory/sync-receipts',{method:'POST',credentials:'include'});
        if(res.status===409){
          // another tab already started a run — pick up its state
          await refreshSyncStatus();
          return;
        }
        if(!res.ok){throw new Error('HTTP '+res.status);}
        var body=await res.json();
        SYNC_STATE={id:body.id,status:'running',started_at:body.started_at,processed:0,auto_created:0,pending_review:0,cached:0};
        SYNC_CHIP_DISMISSED_ID=null;
        renderSyncUI();
        startSyncPoll();
      }catch(e){
        if(btn)btn.disabled=false;
        alert('Could not start sync: '+e.message);
      }
    }

    // Event delegation for #s1 — keep the existing wiring scheme.
    document.addEventListener('click',function(e){
      var t=e.target;
      if(!t)return;
      if(t.id==='sync-receipts-btn'){triggerSync();return;}
      if(t.getAttribute&&t.getAttribute('data-action')==='dismiss-sync-chip'){
        if(SYNC_STATE)SYNC_CHIP_DISMISSED_ID=SYNC_STATE.id;
        renderSyncUI();
        return;
      }
    });
    ```

    D. Wire into `show(n)` — line 332 currently reads `if(n===1){loadHistory();}`.
    Change to:
    ```js
    if(n===1){loadHistory();refreshSyncStatus();}
    ```

    E. Also call `refreshSyncStatus()` once on initial page load — find the existing
    bootstrap section (search for the first `loadHistory()` call at line 1514 or the
    DOMContentLoaded/init block) and add `refreshSyncStatus();` immediately after the
    initial `loadHistory()` call. This handles the case where the user reloads while
    the Purchases tab is already the default-active tab.

    F. Run `node build-sw.js` after the HTML edit. The plan executor MUST run this
    before any verify step — per CLAUDE.md, "Run `task sw` or `node build-sw.js` after
    changing HTML/JS files." A stale service worker will mask the sync button.

    G. Add Playwright tests at the END of `tests/inventory.spec.js`, in a new describe
    block. Use the `page.route` stub pattern already in the file (line 519):

    ```js
    test.describe('Receipt sync button', () => {
      test.beforeEach(async ({ page }) => {
        await login(page);
      });

      test('clicking Sync Receipts disables button and shows Syncing…', async ({ page }) => {
        // Status endpoint returns null on first load
        await page.route('**/api/v1/inventory/sync-receipts/status', async route => {
          await route.fulfill({ status: 200, contentType: 'application/json', body: 'null' });
        });
        // POST returns 200 with a running run
        await page.route('**/api/v1/inventory/sync-receipts', async route => {
          if (route.request().method() === 'POST') {
            await route.fulfill({
              status: 200, contentType: 'application/json',
              body: JSON.stringify({ id: 1, status: 'running', started_at: new Date().toISOString() })
            });
          } else { await route.continue(); }
        });
        await page.goto('/inventory.html');
        await page.waitForLoadState('networkidle');
        const btn = page.locator('#sync-receipts-btn');
        await expect(btn).toHaveText('Sync Receipts');
        await btn.click();
        await expect(btn).toHaveText(/Syncing/);
        await expect(btn).toBeDisabled();
      });

      test('reload mid-run shows Syncing… (state survives via GET /status)', async ({ page }) => {
        // Status endpoint returns a running row
        await page.route('**/api/v1/inventory/sync-receipts/status', async route => {
          await route.fulfill({
            status: 200, contentType: 'application/json',
            body: JSON.stringify({
              id: 42, status: 'running',
              started_at: new Date().toISOString(), finished_at: null,
              processed: 0, auto_created: 0, pending_review: 0, cached: 0,
              error: null, triggered_by: 'manual'
            })
          });
        });
        await page.goto('/inventory.html');
        await page.waitForLoadState('networkidle');
        // Purchases is the default tab — sync button should mount on load
        const btn = page.locator('#sync-receipts-btn');
        await expect(btn).toBeVisible();
        await expect(btn).toHaveText(/Syncing/);
        await expect(btn).toBeDisabled();
      });

      test('completed run shows summary chip with counts', async ({ page }) => {
        await page.route('**/api/v1/inventory/sync-receipts/status', async route => {
          await route.fulfill({
            status: 200, contentType: 'application/json',
            body: JSON.stringify({
              id: 7, status: 'done',
              started_at: new Date(Date.now() - 60000).toISOString(),
              finished_at: new Date().toISOString(),
              processed: 5, auto_created: 3, pending_review: 2, cached: 0,
              error: null, triggered_by: 'manual'
            })
          });
        });
        await page.goto('/inventory.html');
        await page.waitForLoadState('networkidle');
        const chip = page.locator('#sync-receipts-chip');
        await expect(chip).toBeVisible();
        await expect(chip).toContainText('5 processed');
        await expect(chip).toContainText('2 pending review');
        // Dismiss × hides the chip
        await chip.locator('[data-action="dismiss-sync-chip"]').click();
        await expect(chip).not.toBeVisible();
      });
    });
    ```

    Place this block at the end of the file (after the last `test.describe` closes) so
    it does not disrupt existing test ordering.
  </action>
  <verify>
    <automated>node build-sw.js &amp;&amp; npx playwright test tests/inventory.spec.js -g "Receipt sync button" --reporter=line</automated>
  </verify>
  <done>
    Service worker regenerated (sw.js modified). Purchases tab #s1 renders the
    sync button above the vendor filter. All three Playwright "Receipt sync button"
    tests pass. Manual smoke: click button → "Syncing…", reload → still "Syncing…",
    completion shows chip with counts, × dismisses it.
  </done>
</task>

</tasks>

<verification>
End-to-end manual smoke (after all three tasks committed):

1. Backend: `cd backend && go build ./... && go test ./internal/receipt/ ./internal/inventory/ -count=1`
2. Service worker: `node build-sw.js` (must run after HTML edits per CLAUDE.md)
3. Playwright: `task test` (or at minimum `npx playwright test tests/inventory.spec.js`)
4. Live smoke in browser:
   - Visit `/inventory.html`, confirm Purchases tab shows "Sync Receipts" button above vendor filter
   - Click → button shows "Syncing…", disabled
   - Reload page mid-run → still "Syncing…" (proves durability via DB row)
   - Wait for completion → green chip "Last synced 0m ago — N processed, M pending review"
   - Tap × → chip hides, button re-enables
   - Click sync again → new run begins
</verification>

<success_criteria>
- Migration 0067 adds receipt_sync_runs table with single-running partial unique index
- runIngestCycle returns IngestResult and an exported RunIngestCycle wrapper exists
- POST /api/v1/inventory/sync-receipts is single-flight (409 on concurrent) and panic-safe (goroutine recover updates row to failed)
- GET /api/v1/inventory/sync-receipts/status returns latest row or null
- inventory.html Purchases tab renders sync button + dismissable status chip above the vendor filter
- Polling pauses on document.hidden and resumes on visible
- Status survives full page reload (state is on the server, not in localStorage)
- All three new backend Go tests pass; all three new Playwright tests pass
- Service worker regenerated via `node build-sw.js`
- No new external dependencies; no new npm packages
</success_criteria>

<output>
After completion, create `.planning/quick/260607-bir-mercury-receipt-sync-button/260607-bir-SUMMARY.md` capturing:
- Files changed
- Migration applied
- Endpoint contracts (request/response shapes for the two new routes)
- Test command outputs (Go + Playwright)
- One-line "how to use" note for the owner
</output>
