---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - backend/internal/inventory/stock.go
  - backend/internal/inventory/stock_test.go
  - backend/internal/inventory/handler.go
  - backend/internal/inventory/types.go
  - backend/internal/purchasing/service.go
  - inventory.html
autonomous: true
requirements: [STOCK-CLASSIFY]

must_haves:
  truths:
    - "Stock API returns level and needs_reorder for each item"
    - "Frontend renders stock levels from API, not computing them locally"
    - "PO suggestions use the same classification logic as the stock endpoint"
    - "No divergence between inventory stock levels and PO suggestion filtering"
  artifacts:
    - path: "backend/internal/inventory/stock.go"
      provides: "Shared ClassifyStockLevel function"
      exports: ["ClassifyStockLevel"]
    - path: "backend/internal/inventory/stock_test.go"
      provides: "Unit tests for classification logic"
    - path: "backend/internal/inventory/types.go"
      provides: "Updated StockItem struct with level fields"
      contains: "NeedsReorder"
  key_links:
    - from: "backend/internal/inventory/handler.go"
      to: "backend/internal/inventory/stock.go"
      via: "ClassifyStockLevel call in GetStockHandler"
      pattern: "ClassifyStockLevel"
    - from: "backend/internal/purchasing/service.go"
      to: "backend/internal/inventory/stock.go"
      via: "import inventory.ClassifyStockLevel in GetSuggestions"
      pattern: "inventory\\.ClassifyStockLevel"
    - from: "inventory.html"
      to: "/api/v1/inventory/stock"
      via: "reads s.level from API response instead of computing"
      pattern: "s\\.level"
---

<objective>
Move stock level classification from frontend JS into the Go backend so both GetStockHandler and GetSuggestions use one shared function. Eliminates the divergence that caused the 4-vs-3 PO suggestions count bug.

Purpose: Single source of truth for stock level classification (low/medium/high/unknown) and needs_reorder logic.
Output: Shared Go function, enriched stock API response, simplified frontend.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@backend/internal/inventory/handler.go (GetStockHandler — lines 362-408)
@backend/internal/inventory/types.go (StockItem struct — lines 93-101)
@backend/internal/purchasing/service.go (GetSuggestions — lines 574-640)
@inventory.html (renderStock — lines 522-549)
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Create shared ClassifyStockLevel function with tests</name>
  <files>backend/internal/inventory/stock.go, backend/internal/inventory/stock_test.go, backend/internal/inventory/types.go</files>
  <behavior>
    - ClassifyStockLevel(qty=0, lowT=3, highT=10) returns ("unknown", false)
    - ClassifyStockLevel(qty=2, lowT=3, highT=10) returns ("low", true) — qty <= lowT
    - ClassifyStockLevel(qty=3, lowT=3, highT=10) returns ("low", true) — qty == lowT is low
    - ClassifyStockLevel(qty=5, lowT=3, highT=10) returns ("medium", true) — lowT < qty < highT
    - ClassifyStockLevel(qty=10, lowT=3, highT=10) returns ("high", false) — qty >= highT
    - ClassifyStockLevel(qty=15, lowT=3, highT=10) returns ("high", false)
    - ClassifyStockLevel(qty=9, lowT=3, highT=10) returns ("medium", true) — just below highT
  </behavior>
  <action>
    1. Update StockItem in types.go — add four new JSON fields:
       - LowThreshold int `json:"low_threshold"`
       - HighThreshold int `json:"high_threshold"`
       - Level string `json:"level"`
       - NeedsReorder bool `json:"needs_reorder"`

    2. Create stock.go with:
       ```go
       package inventory

       // ClassifyStockLevel determines stock level and reorder need.
       // Matches the logic: qty==0 -> unknown; qty>=highT -> high; qty>lowT -> medium; else low.
       // NeedsReorder is true for low and medium levels.
       func ClassifyStockLevel(qty, lowT, highT int) (level string, needsReorder bool) {
           switch {
           case qty == 0:
               return "unknown", false
           case qty >= highT:
               return "high", false
           case qty > lowT:
               return "medium", true
           default:
               return "low", true
           }
       }
       ```

    3. Create stock_test.go with table-driven tests covering all behavior cases above.
  </action>
  <verify>
    <automated>cd /Users/jamal/projects/yumyums/hq/backend && go test ./internal/inventory/ -run TestClassifyStockLevel -v</automated>
  </verify>
  <done>ClassifyStockLevel function exists, all test cases pass, StockItem struct has level/threshold/needs_reorder fields</done>
</task>

<task type="auto">
  <name>Task 2: Wire classification into GetStockHandler and GetSuggestions</name>
  <files>backend/internal/inventory/handler.go, backend/internal/purchasing/service.go</files>
  <action>
    **GetStockHandler (handler.go lines 362-408):**
    1. Expand the SQL query to also JOIN item_groups and return low_threshold, high_threshold alongside existing fields. The query already JOINs item_groups via `ig.name AS group_name` but does not select thresholds. Add:
       - `COALESCE(ig.low_threshold, 3) AS low_threshold`
       - `COALESCE(ig.high_threshold, 10) AS high_threshold`
       These go in the inner subquery alongside group_name, then are passed through to the outer SELECT.

    2. Update the rows.Scan call to also scan LowThreshold and HighThreshold into StockItem.

    3. After scanning each row, call `ClassifyStockLevel(s.TotalQuantity, s.LowThreshold, s.HighThreshold)` and assign to `s.Level` and `s.NeedsReorder`.

    **GetSuggestions (purchasing/service.go lines 574-640):**
    1. Add import for the inventory package: `"github.com/yumyums/hq/internal/inventory"`
    2. Remove the inline SQL CASE expression for stock_level (lines 589-592). Instead, scan current_stock and low_threshold from the query, then after the scan call `level, _ := inventory.ClassifyStockLevel(s.CurrentStock, s.LowThreshold, highThreshold)` and set `s.StockLevel = level`.
    3. The WHERE clause `sub.current_stock < COALESCE(ig.high_threshold, 10) AND sub.current_stock > 0` stays in SQL for efficient filtering — the shared function is used for labeling only.
    4. Also scan high_threshold from the SQL query (add it to the SELECT list and add a local var to scan into, or add HighThreshold to OrderSuggestion). Use the scanned value when calling ClassifyStockLevel.
  </action>
  <verify>
    <automated>cd /Users/jamal/projects/yumyums/hq/backend && go build ./...</automated>
  </verify>
  <done>Backend compiles. GetStockHandler returns level, needs_reorder, low_threshold, high_threshold in JSON. GetSuggestions uses inventory.ClassifyStockLevel for stock_level labeling.</done>
</task>

<task type="auto">
  <name>Task 3: Update frontend to consume backend-provided levels</name>
  <files>inventory.html</files>
  <action>
    In renderStock() (inventory.html around line 522-536):

    1. Remove the entire threshold lookup and level computation block (lines 525-536):
       ```js
       // DELETE this block:
       stocks.forEach(s=>{
         if(!s.level){
           var tq=s.total_quantity||0;
           var lowT=3,highT=10;
           if(s.group_name){
             var grp=ITEM_GROUPS.find(function(g){return g.name===s.group_name;});
             if(grp){lowT=grp.low_threshold||3;highT=grp.high_threshold||10;}
           }
           s.level=tq===0?'unknown':tq>=highT?'high':tq>lowT?'medium':'low';
         }
       });
       ```
       The API now provides `s.level` directly — no frontend computation needed.

    2. Update the needsReorder filter (line 539) to use the API field:
       Change: `var needsReorder=stocks.filter(s=>s.level==='low'||s.level==='medium');`
       To: `var needsReorder=stocks.filter(s=>s.needs_reorder);`

    3. Verify the badge rendering still works — it reads `s.level` which is now API-provided (same string values: low/medium/high/unknown), so badgeHtml() needs no changes.
  </action>
  <verify>
    <automated>cd /Users/jamal/projects/yumyums/hq && npx playwright test tests/inventory.spec.js --timeout 60000 2>&1 | tail -20</automated>
  </verify>
  <done>Frontend reads level and needs_reorder from API. No local classification logic remains in inventory.html. Existing inventory E2E tests pass.</done>
</task>

</tasks>

<verification>
1. `cd backend && go test ./internal/inventory/ -run TestClassifyStockLevel -v` — all classification cases pass
2. `cd backend && go build ./...` — full backend compiles with no errors
3. `npx playwright test tests/inventory.spec.js` — existing E2E tests pass
4. Manual: Load inventory Stock tab, verify items show correct low/medium/high badges and reorder suggestions section matches
</verification>

<success_criteria>
- ClassifyStockLevel is the single source of truth for stock level classification
- GetStockHandler JSON response includes level, needs_reorder, low_threshold, high_threshold
- GetSuggestions uses inventory.ClassifyStockLevel instead of inline SQL CASE
- Frontend inventory.html has zero stock level computation logic — reads from API
- All existing tests pass
</success_criteria>

<output>
After completion, create `.planning/quick/260422-not-shared-stock-level-classification-move-f/260422-not-SUMMARY.md`
</output>
