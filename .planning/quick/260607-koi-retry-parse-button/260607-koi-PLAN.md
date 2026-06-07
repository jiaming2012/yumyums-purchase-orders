---
phase: quick-260607-koi
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - backend/internal/inventory/handler.go
  - backend/cmd/server/main.go
  - backend/internal/inventory/period_summary_test.go
  - inventory.html
  - sw.js
  - tests/inventory.spec.js
autonomous: true
requirements:
  - KOI-BE-RETRY-PARSE-HANDLER
  - KOI-BE-RETRY-PARSE-ROUTE
  - KOI-BE-RETRY-PARSE-TESTS
  - KOI-FE-RETRY-PARSE-BUTTON
  - KOI-FE-RETRY-PARSE-HANDLER
  - KOI-FE-RETRY-PARSE-TESTS

must_haves:
  truths:
    - "POST /api/v1/inventory/purchases/pending/{id}/retry-parse returns 200 and sets parse_error=NULL when the row is pending and has a parse_error"
    - "Same endpoint returns 404 when the id does not exist"
    - "Same endpoint returns 422 {error:row_not_pending} when the row is already confirmed or discarded"
    - "Same endpoint returns 422 {error:nothing_to_retry} when parse_error is already NULL"
    - "A pending card with non-empty parse_error renders a Retry parse button below the Parser error line"
    - "Clicking Retry parse on a success response clears parse_error from the in-memory PENDING_PURCHASES row, hides the Parser error line + button, and shows the toast 'Marked for retry. Click Sync Receipts to run now.'"
    - "A pending card whose parse_error is empty does NOT render the Retry parse button"
  artifacts:
    - path: "backend/internal/inventory/handler.go"
      provides: "RetryParsePendingPurchaseHandler exported func"
      contains: "func RetryParsePendingPurchaseHandler"
    - path: "backend/cmd/server/main.go"
      provides: "POST /purchases/pending/{id}/retry-parse route registered inside the auth-gated inventory route group"
      contains: "RetryParsePendingPurchaseHandler"
    - path: "backend/internal/inventory/period_summary_test.go"
      provides: "4 new TestRetryParse_* tests"
      contains: "TestRetryParse_ClearsParseError"
    - path: "inventory.html"
      provides: ".retry-parse-btn CSS, data-action=retry-parse button in pending card, retry-parse case in the history-list click handler"
      contains: "retry-parse-btn"
    - path: "sw.js"
      provides: "regenerated precache manifest including the updated inventory.html content hash"
    - path: "tests/inventory.spec.js"
      provides: "3 new Playwright tests inside a new describe block for the Retry parse button"
      contains: "Retry parse button"
  key_links:
    - from: "inventory.html (history-list click delegate, case 'retry-parse')"
      to: "POST /api/v1/inventory/purchases/pending/{id}/retry-parse"
      via: "fetch with credentials:'include'"
      pattern: "purchases/pending/.*/retry-parse"
    - from: "RetryParsePendingPurchaseHandler"
      to: "pending_purchases.parse_error column"
      via: "UPDATE ... SET parse_error = NULL WHERE id = $1"
      pattern: "UPDATE pending_purchases SET parse_error"
    - from: "renderPendingCard parse_error branch"
      to: ".retry-parse-btn button rendered immediately after the Parser error line"
      via: "string concatenation gated on p.parse_error truthiness"
      pattern: "data-action=\"retry-parse\""
---

<objective>
Add a Retry parse button on pending review cards that have a parse_error set, plus the backend endpoint it calls, so a stuck row can be re-armed for the next Sync Receipts cycle without DB access. The endpoint just nulls parse_error on the row; the worker's 260607-fxl parseFailedRetry branch (parse_error IS NULL AND items empty AND attachments present) then picks it up on the next sync.

Purpose: Lets the operator recover from parser improvements (like 260607-k1n's float64 fix) without psql. Specifically unblocks the Restaurant Depot $391.96 row right now.

Output: One backend handler + route + 4 unit tests; one FE button + click handler + 3 Playwright tests; regenerated service worker.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@./CLAUDE.md
@.planning/quick/260607-fxl-retry-parse-failed-rows-and-confirm-gate/260607-fxl-SUMMARY.md

<interfaces>
<!-- Key contracts extracted from the codebase. Use these directly. -->

From backend/internal/inventory/handler.go (writeError + writeJSON helpers, error envelope conventions):
```go
func writeError(w http.ResponseWriter, status int, msg string) { /* writes {"error":msg} */ }
// 422 structured envelope (260607-fxl convention):
writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
    "error":  "empty_items_not_allowed",
    "reason": "add at least one line item or set pending reason to no_attachment_on_bank_tx",
})
// 404 short-form (existing UpdatePendingItemsHandler convention):
writeError(w, http.StatusNotFound, "pending_purchase_not_found")
```

From backend/internal/inventory/handler.go ConfirmPendingPurchaseHandler (handler signature shape — DO NOT mirror its body, just the func signature pattern):
```go
func ConfirmPendingPurchaseHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ... handler body
    }
}
```

From backend/cmd/server/main.go (route registration inside the inventory auth-gated group at lines ~455-465):
```go
r.Get("/purchases/pending", inventory.ListPendingPurchasesHandler(pool))
r.Post("/purchases/confirm", inventory.ConfirmPendingPurchaseHandler(pool))
r.Post("/purchases/discard", inventory.DiscardPendingPurchaseHandler(pool))
r.Put("/purchases/pending-items", inventory.UpdatePendingItemsHandler(pool))
```
The new route goes alongside these. NOTE: existing pending mutators take id in the JSON body, NOT URL path. Per the user's requirement (verbatim), the retry-parse route uses chi URL path param: `/purchases/pending/{id}/retry-parse`. Use `chi.URLParam(r, "id")` to extract it. Add `"github.com/go-chi/chi/v5"` to the handler.go imports if not already present.

From backend/internal/inventory/types.go PendingPurchase (response shape for 200 — confirmed present from 260607-e1c):
- Has a parse_error field already (set in 260607-e1c).
- The 200 response body is the full PendingPurchase row with parse_error now nil/empty.

From backend/internal/inventory/period_summary_test.go (test helpers):
```go
// Line 163 — minimal helper, default reason / no bank_total:
func insertPendingPurchase(t *testing.T, createdAt string, confirmed, discarded bool) string

// Line 240 — with bank_total:
func insertPendingPurchaseWithBankTotal(t *testing.T, bankTxID string, bankTotal float64, createdAt string) string

// Line 263 — with event_date + reason:
func insertPendingPurchaseWithEventDate(t *testing.T, bankTxID, eventDate, createdAt, reason string) string
```
None of these touch parse_error. After insertion the tests UPDATE parse_error directly (mirrors how TestRunIngestCycle_SkipsRealCached in worker_test.go does it). Use the existing test harness / DB setup from this file — there's a confirmPendingHelper added in 260607-fxl; reuse the same Postgres pool setup the surrounding TestConfirmPending_* tests use.

From inventory.html line 591 (pending card parse_error rendering — 260607-e1c):
```js
var parseErrHtml='';
if(p.reason==='Receipt could not be parsed automatically' && p.parse_error){
  var truncated=String(p.parse_error).slice(0,140);
  parseErrHtml='<div class="event-meta" style="font-style:italic;opacity:0.7;margin-top:2px">Parser error: '+escHtml(truncated)+'</div>';
}
return '<div class="event-card" data-action="review-pending" data-id="'+p.id+'" ...>'+...+parseErrHtml+'...</div>';
```
The Retry parse button must be appended INSIDE the same parse_error branch (so it only renders when the Parser error line renders). Because the outer card carries `data-action="review-pending"`, the button needs its own `data-action="retry-parse"` AND the click delegate uses `e.target.closest('[data-action]')` which returns the innermost — so the button's action wins over the card's. No stopPropagation needed.

From inventory.html line 73 (.view-receipt-btn CSS — the muted-blue template):
```css
.view-receipt-btn{width:100%;padding:10px;background:var(--bg);border:0.5px solid var(--brd);border-radius:8px;font-size:13px;color:var(--info-tx);font-weight:500;cursor:pointer;font-family:inherit;margin-top:8px;text-align:center}
```
New `.retry-parse-btn` CSS should match this exactly (same look-and-feel). Adjust width if it shouldn't be 100% on a pending card — keep 100% to match the View Original Receipt button width inside the review form for visual consistency. Margin-top 6 is enough since it sits right under the italic Parser error line.

From inventory.html line 892 (history-list click delegate — where the new case goes):
```js
document.getElementById('history-list').addEventListener('click',function(e){
  var el=e.target.closest('[data-action]');
  if(!el)return;
  var action=el.dataset.action;
  var id=el.dataset.id;
  if(action==='toggle-event'){...}
  else if(action==='review-pending'){...}
  // ... add new case 'retry-parse' here, alongside the existing cases
});
```
showToastMessage is defined at line 1379 of inventory.html and accepts a single string arg — used elsewhere in the file (line 1360). renderHistoryList() is the canonical re-render fn after PENDING_PURCHASES mutation (called from confirm/discard/discard etc.).

From tests/inventory.spec.js line 2909 (existing parse_error describe block for 260607-e1c — model for the new tests):
```js
test.describe('Pending card — parse_error display (260607-e1c)', () => {
  test('renders Parser error line when parse_error is set', async ({ page }) => {
    await page.route('**/api/v1/inventory/purchases/pending', route =>
      route.fulfill({ status: 200, contentType:'application/json',
        body: JSON.stringify([{ id:'p1', vendor_name:'Test', event_date:'2026-06-07',
          bank_total: -10, reason:'Receipt could not be parsed automatically',
          parse_error: "failed to unmarshal: invalid character '<' looking for beginning of value (text: <html>)",
          items: [] }]) }));
    // ... navigate, assert card contains 'Parser error:'
  });
});
```
Place the new `test.describe('Retry parse button (260607-koi)', ...)` block immediately AFTER the existing parse_error describe block (~line 2935-ish in the current file, before the PDF iframe describe at line 2939).
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Backend retry-parse endpoint + route + 4 unit tests</name>
  <files>backend/internal/inventory/handler.go, backend/cmd/server/main.go, backend/internal/inventory/period_summary_test.go</files>
  <behavior>
    Tests (write FIRST in period_summary_test.go, then make them pass):

    - TestRetryParse_ClearsParseError:
        Seed a pending row (insertPendingPurchaseWithEventDate(... reason='Receipt could not be parsed automatically' ...)).
        UPDATE pending_purchases SET parse_error = 'haiku: boom; sonnet: boom' WHERE id = $1.
        POST /api/v1/inventory/purchases/pending/{id}/retry-parse (with the test's auth cookie/middleware setup).
        Expect: status 200; response body is the PendingPurchase row with parse_error empty/null.
        SELECT parse_error FROM pending_purchases WHERE id=$1 → expect NULL.

    - TestRetryParse_404OnUnknownId:
        Generate a random valid UUID that is NOT in pending_purchases (uuid.New().String()).
        POST that id → expect status 404, response body {"error":"pending_purchase_not_found"} (matching the existing UpdatePendingItemsHandler 404 envelope, NOT a structured 422).

    - TestRetryParse_422OnConfirmedRow:
        Seed a row, UPDATE confirmed_at = NOW(), parse_error = 'haiku: boom; sonnet: boom'.
        POST → expect status 422, body {"error":"row_not_pending","reason":"row is already confirmed or discarded"}.

    - TestRetryParse_422WhenNothingToRetry:
        Seed a pending row with parse_error IS NULL (default).
        POST → expect status 422, body {"error":"nothing_to_retry","reason":"row has no parse_error to clear"}.

    All 4 tests must use the same Postgres pool / chi router / auth-context wiring as the surrounding TestConfirmPending_* tests in this file (set up by 260607-fxl's confirmPendingHelper — reuse or replicate; do NOT introduce a new harness).
  </behavior>
  <action>
    Implement after tests are red.

    1. Backend handler in backend/internal/inventory/handler.go (append at end of file or place alongside DiscardPendingPurchaseHandler — match the existing func ordering):

       ```go
       // RetryParsePendingPurchaseHandler clears parse_error on a pending row so the
       // next worker sync cycle (260607-fxl parseFailedRetry branch) will re-attempt
       // parsing through Sonnet. Used to re-arm rows stuck after a parser bug fix
       // without DB access.
       func RetryParsePendingPurchaseHandler(pool *pgxpool.Pool) http.HandlerFunc {
           return func(w http.ResponseWriter, r *http.Request) {
               id := chi.URLParam(r, "id")
               if id == "" {
                   writeError(w, http.StatusBadRequest, "id_required")
                   return
               }
               // SELECT current row state to decide 404 vs 422 disposition.
               var confirmedAt, discardedAt sql.NullTime
               var parseError sql.NullString
               err := pool.QueryRow(r.Context(),
                   `SELECT confirmed_at, discarded_at, parse_error FROM pending_purchases WHERE id = $1`,
                   id,
               ).Scan(&confirmedAt, &discardedAt, &parseError)
               if err != nil {
                   // pgx.ErrNoRows OR any other read error — match existing convention:
                   // UpdatePendingItemsHandler returns 404 pending_purchase_not_found on missing row.
                   writeError(w, http.StatusNotFound, "pending_purchase_not_found")
                   return
               }
               if confirmedAt.Valid || discardedAt.Valid {
                   writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
                       "error":  "row_not_pending",
                       "reason": "row is already confirmed or discarded",
                   })
                   return
               }
               if !parseError.Valid || parseError.String == "" {
                   writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
                       "error":  "nothing_to_retry",
                       "reason": "row has no parse_error to clear",
                   })
                   return
               }
               if _, err := pool.Exec(r.Context(),
                   `UPDATE pending_purchases SET parse_error = NULL WHERE id = $1`,
                   id,
               ); err != nil {
                   log.Printf("RetryParsePendingPurchase update: %v", err)
                   writeError(w, http.StatusInternalServerError, "internal_error")
                   return
               }
               // Return the updated row. Re-SELECT with the full PendingPurchase
               // projection — reuse the existing single-row fetch helper used by
               // ListPendingPurchasesHandler (or if no such helper, do a fresh
               // SELECT here mirroring the SELECT used in ListPendingPurchasesHandler,
               // filtered by id). Match the same field projection so the FE gets
               // the same shape it gets from /purchases/pending list.
               // If no single-row fetch helper exists, the simplest correct path
               // is to inline the SELECT used by ListPendingPurchasesHandler with
               // `WHERE id = $1` and scan into a PendingPurchase struct.
               pending, ferr := fetchPendingPurchaseByID(r.Context(), pool, id)
               if ferr != nil {
                   log.Printf("RetryParsePendingPurchase refetch: %v", ferr)
                   writeError(w, http.StatusInternalServerError, "internal_error")
                   return
               }
               writeJSON(w, http.StatusOK, pending)
           }
       }
       ```

       If `fetchPendingPurchaseByID` does not exist (likely — only the list handler does it inline), create a small unexported helper at the bottom of handler.go that runs the same SELECT used by ListPendingPurchasesHandler filtered by id and returns (PendingPurchase, error). Mirror the field projection EXACTLY so the FE response shape matches what /purchases/pending returns.

       Ensure imports include: `"database/sql"`, `"github.com/go-chi/chi/v5"`. (database/sql may already be there from 260607-fxl's ConfirmPending changes — check before adding.)

    2. Route wiring in backend/cmd/server/main.go (~line 463, in the same auth-gated inventory route group, immediately after r.Post("/purchases/discard", ...)):

       ```go
       r.Post("/purchases/pending/{id}/retry-parse", inventory.RetryParsePendingPurchaseHandler(pool))
       ```

    3. Tests in backend/internal/inventory/period_summary_test.go — place at the BOTTOM of the file, after the existing TestConfirmPending_* tests from 260607-fxl. Reuse confirmPendingHelper's pool/router setup but for POST to the new path. If confirmPendingHelper hardcodes the confirm route, factor out a retryParseHelper that mirrors it for the retry-parse path (or add a small router-builder that registers the new route alongside).

    Conventions:
    - 422 envelopes use map[string]string (mirrors empty_items_not_allowed at handler.go:700-704).
    - 404 uses writeError single-key envelope (mirrors UpdatePendingItemsHandler at handler.go:262).
    - DO NOT add a migration. parse_error already exists per 260607-e1c.
    - DO NOT add deps.

    Commit: `feat(260607-koi): backend retry-parse endpoint + tests`
  </action>
  <verify>
    <automated>cd backend &amp;&amp; DB_HOST=100.70.200.55 DB_PORT=5433 go test -count=1 -run TestRetryParse ./internal/inventory/... &amp;&amp; DB_HOST=100.70.200.55 DB_PORT=5433 go test -count=1 ./internal/inventory/... &amp;&amp; go build ./...</automated>
  </verify>
  <done>
    - `go test -count=1 -run TestRetryParse ./internal/inventory/...` passes (4 new tests green)
    - Full `go test -count=1 ./internal/inventory/...` passes (no regression on existing TestConfirmPending_*, TestPeriodSummary, etc.)
    - `go build ./...` clean (no unused imports, no missing deps)
    - Route POST /api/v1/inventory/purchases/pending/{id}/retry-parse is registered inside the auth-gated group
    - Handler returns 200 on happy path, 404 on unknown id, 422 row_not_pending on confirmed/discarded, 422 nothing_to_retry on null parse_error
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: FE Retry parse button + click handler + 3 Playwright tests + sw.js rebuild</name>
  <files>inventory.html, sw.js, tests/inventory.spec.js</files>
  <behavior>
    Tests (write FIRST in tests/inventory.spec.js, inside a new `test.describe('Retry parse button (260607-koi)', ...)` block placed immediately after the existing 260607-e1c parse_error describe at ~line 2935):

    - Test 1: 'Retry parse button is shown when pending row has parse_error':
        page.route stubs GET /api/v1/inventory/purchases/pending to return one row with
        `reason:'Receipt could not be parsed automatically'`, `parse_error:'haiku: boom; sonnet: boom'`, `items:[]`.
        Navigate to inventory.html, switch to the Purchases tab (or whichever tab renders the pending card — mirror what the 260607-e1c tests do).
        Assert: `page.locator('[data-action="retry-parse"]')` is visible AND has text 'Retry parse'.
        Assert: the same card also contains 'Parser error:' (regression sanity).

    - Test 2: 'Retry parse button is hidden when parse_error is empty':
        page.route stubs the same pending list but with `parse_error:''` (or omit the field).
        Assert: `page.locator('[data-action="retry-parse"]')` has count 0.
        Assert: the card itself still renders (regression sanity — make sure we didn't accidentally hide the whole card).

    - Test 3: 'Retry parse button clears parse_error from card on success':
        page.route stubs GET /pending with one row that has parse_error set.
        page.route stubs POST /api/v1/inventory/purchases/pending/*/retry-parse to return 200 with the same row but `parse_error:''`.
        Click the button.
        Assert: the 'Parser error:' text disappears from the card.
        Assert: the `[data-action="retry-parse"]` button disappears from the card.
        (Toast assertion optional — showToastMessage is fire-and-forget; if a stable selector exists for it, assert it too, otherwise skip.)
  </behavior>
  <action>
    Implement after tests are red.

    1. CSS (inventory.html, in the same <style> block — alongside .view-receipt-btn at line 73):
       ```css
       .retry-parse-btn{width:100%;padding:10px;background:var(--bg);border:0.5px solid var(--brd);border-radius:8px;font-size:13px;color:var(--info-tx);font-weight:500;cursor:pointer;font-family:inherit;margin-top:6px;text-align:center}
       ```

    2. Render branch (inventory.html, around line 591 — inside the existing `if(p.reason==='Receipt could not be parsed automatically' && p.parse_error){` block):
       ```js
       if(p.reason==='Receipt could not be parsed automatically' && p.parse_error){
         var truncated=String(p.parse_error).slice(0,140);
         parseErrHtml='<div class="event-meta" style="font-style:italic;opacity:0.7;margin-top:2px">Parser error: '+escHtml(truncated)+'</div>'+
           '<button class="retry-parse-btn" data-action="retry-parse" data-id="'+p.id+'">Retry parse</button>';
       }
       ```
       Keep the existing return statement — the button is appended INSIDE the event-card div via parseErrHtml concatenation, so the card's outer `data-action="review-pending"` still works for clicks outside the button (event delegation picks the innermost data-action).

    3. Click handler case (inventory.html, in the history-list click delegate at line 892, alongside the existing cases — place AFTER `discard-receipt` for grouping):
       ```js
       else if(action==='retry-parse'){
         // 260607-koi: clear parse_error server-side so the next worker sync
         // (260607-fxl parseFailedRetry branch) re-attempts the parse. The
         // operator clicks Sync Receipts themselves — this endpoint does NOT
         // trigger sync.
         var rid=id;
         fetch('/api/v1/inventory/purchases/pending/'+encodeURIComponent(rid)+'/retry-parse',{
           method:'POST',credentials:'include'
         }).then(function(resp){
           if(resp.ok){
             showToastMessage('Marked for retry. Click Sync Receipts to run now.');
             var row=PENDING_PURCHASES.find(function(pp){return pp.id===rid;});
             if(row)row.parse_error='';
             renderHistoryList();
           }else{
             alert('Could not mark for retry: HTTP '+resp.status);
           }
         }).catch(function(err){
           alert('Could not mark for retry: '+err.message);
         });
       }
       ```

    4. After ALL inventory.html edits are saved, regenerate the service worker so the precache hash for inventory.html updates:
       ```
       node build-sw.js
       ```
       This writes the new content-hashed precache manifest into sw.js. Playwright tests block service workers (per playwright config), so the SW rebuild is for production deploy correctness, not test pass — but the user-visible verification (Restaurant Depot $391.96 in the live PWA) requires it.

    5. Playwright tests in tests/inventory.spec.js — new describe block placed immediately after the existing parse_error describe (~line 2935). Mirror the page.route + navigation pattern of the 260607-e1c tests exactly (same goto path, same page.locator strategy). For Test 3 use `await page.route('**/api/v1/inventory/purchases/pending/*/retry-parse', route => route.fulfill({...}))` — the `*` glob covers the URL-encoded id.

    Commit: `feat(260607-koi): FE retry button + tests`

    NOTE: The sw.js regeneration creates a real diff (precache hash). Include sw.js in the FE commit.
  </action>
  <verify>
    <automated>node build-sw.js &amp;&amp; DB_HOST=100.70.200.55 DB_PORT=5433 TOAST_SFTP_KEY_PATH=/tmp/yumyums-test/fake-toast-key TOAST_SYNC_INTERVAL=0 npx playwright test tests/inventory.spec.js -g "Retry parse button"</automated>
  </verify>
  <done>
    - `node build-sw.js` runs clean; sw.js diff includes a fresh content hash for inventory.html
    - 3 new Playwright tests under `test.describe('Retry parse button (260607-koi)', ...)` all pass
    - Existing 260607-e1c parser-error tests and 260607-fxl confirm-disabled tests still pass (no regression — quick smoke: `npx playwright test tests/inventory.spec.js -g "Parser error line|Confirm Receipt disabled"`)
    - inventory.html contains: `.retry-parse-btn` CSS, the button in the parse_error render branch, and the `retry-parse` case in the history-list click delegate
    - On a real pending row with parse_error set, the button appears; clicking it returns 200 from the backend and the Parser error line + button disappear from the card
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Authenticated browser session → POST /api/v1/inventory/purchases/pending/{id}/retry-parse | Cookie-auth operator-only mutation that clears a column to re-arm worker behavior |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-260607-koi-01 | Tampering | RetryParsePendingPurchaseHandler | mitigate | Route is registered INSIDE the auth-gated `r.Group` (same group as /confirm and /discard) — auth middleware rejects unauthenticated requests before the handler runs |
| T-260607-koi-02 | Elevation of Privilege | RetryParsePendingPurchaseHandler | mitigate | Handler does NOT trigger sync, does NOT call Anthropic, does NOT touch billing — only nulls a single column. Worst-case abuse = trigger one extra Sonnet parse on next operator-initiated Sync Receipts click. Cost cap is operator click frequency |
| T-260607-koi-03 | Tampering | id URL param | mitigate | Postgres parameterized query (`WHERE id = $1`) — id is treated as a uuid value, not interpolated. Invalid uuid → ErrNoRows → 404, not crash |
| T-260607-koi-04 | Denial of Service | RetryParsePendingPurchaseHandler | accept | Single-row UPDATE, no joins, no external calls. Authenticated-only. Rate-limiting deferred — matches existing /confirm and /discard which also lack per-endpoint rate limits |
| T-260607-koi-05 | Information Disclosure | 200 response body (full PendingPurchase row) | accept | Response shape matches what /purchases/pending list already returns to the same authenticated operator. No new fields exposed |
| T-260607-koi-06 | Repudiation | RetryParsePendingPurchaseHandler | accept | No audit log — matches existing /pending-items PUT and /discard endpoints. Worker sync runs are already logged in receipt_sync_runs; the retry's effect surfaces there on the next sync |
</threat_model>

<verification>
End-to-end manual check (after both commits land and SW deploys):
1. Find the Restaurant Depot $391.96 pending row → expect "Retry parse" button visible below the Parser error line.
2. Click button → expect toast "Marked for retry. Click Sync Receipts to run now."; expect the Parser error line and the button to disappear from the card.
3. Click Sync Receipts → expect that row to auto-resolve via Haiku (relies on k1n's float64 fix) and move from "Needs Review" status into the normal pending review flow OR get fully classified.

Automated:
- All 4 new TestRetryParse_* backend tests green.
- All 3 new Playwright tests under 'Retry parse button (260607-koi)' green.
- No regression in TestConfirmPending_*, TestPeriodSummary, 260607-e1c parse_error display, or 260607-fxl confirm-disabled-state tests.
</verification>

<success_criteria>
- POST /api/v1/inventory/purchases/pending/{id}/retry-parse exists and behaves per the 4 specified dispositions (200 / 404 / 422 row_not_pending / 422 nothing_to_retry).
- A pending card with non-empty parse_error renders a Retry parse button with `data-action="retry-parse"`.
- Clicking the button → fetch POST → on 200 → toast + clear parse_error in PENDING_PURCHASES + re-render → card shows no Parser error line and no button.
- sw.js precache manifest updated to the new inventory.html content hash.
- 4 backend tests + 3 Playwright tests added and passing.
- Two atomic commits land in the worktree: backend, then FE.
- No DB migration. No new deps.
</success_criteria>

<output>
After completion, create `.planning/quick/260607-koi-retry-parse-button/260607-koi-SUMMARY.md` following the standard summary template — note the two commits, the 7 new tests, the route path, and the manual verification status against Restaurant Depot $391.96 (or note that it's the operator's verification step, not Claude's).
</output>
