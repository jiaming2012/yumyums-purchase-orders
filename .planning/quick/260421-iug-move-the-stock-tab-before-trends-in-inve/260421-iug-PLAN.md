---
phase: quick
plan: 260421-iug
type: execute
wave: 1
depends_on: []
files_modified: [inventory.html, tests/inventory.spec.js, CLAUDE.md]
autonomous: true
requirements: []
must_haves:
  truths:
    - "Stock tab appears as the 2nd tab (after Purchases, before Trends)"
    - "Clicking Stock tab loads stock data and renders stock list"
    - "Clicking Trends tab shows coming soon content"
    - "All other tabs (Purchases, Cost, Setup) are unaffected"
  artifacts:
    - path: "inventory.html"
      provides: "Tab order: Purchases, Stock, Trends, Cost, Setup"
    - path: "tests/inventory.spec.js"
      provides: "Updated tab assertions matching new order"
    - path: "CLAUDE.md"
      provides: "Updated 5-tab layout description"
  key_links:
    - from: "inventory.html button#t2"
      to: "show(2) -> loadStock()"
      via: "onclick handler and show() function"
    - from: "inventory.html button#t3"
      to: "show(3) -> renderTrends()"
      via: "onclick handler and show() function"
---

<objective>
Move the Stock tab before Trends in inventory.html tab order.

Purpose: Stock is used frequently; Trends is a coming-soon placeholder. Putting Stock earlier improves daily usability.
Output: Tab order changes from Purchases/Trends/Stock/Cost/Setup to Purchases/Stock/Trends/Cost/Setup.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@inventory.html
@tests/inventory.spec.js
@CLAUDE.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Swap Stock and Trends tab positions in inventory.html</name>
  <files>inventory.html</files>
  <action>
Swap the tab button labels and section content so Stock is tab 2 and Trends is tab 3. Keep all element IDs (t2, t3, s2, s3) in place — only swap what they contain:

1. **Tab buttons (lines 174-175):** Change t2 button text from "Trends" to "Stock" and t3 button text from "Stock" to "Trends". The onclick handlers stay as show(2) and show(3) respectively.

2. **Section divs:** Swap the innerHTML of s2 and s3:
   - s2 (was trends coming-soon) becomes stock content: `<div id="reorder-section"></div><div id="stock-list"></div>`
   - s3 (was stock content) becomes trends content: `<!-- TODO: gate Trends... --><div id="trends-container" class="coming-soon">...</div>`

3. **show() function (line 262):** Change `if(n===3){loadStock();}` to `if(n===2){loadStock();}` — Stock is now tab 2.

4. **render() function (lines 269-270):** Swap the tab number checks:
   - `if(ACTIVE_TAB===2)renderStock();` (was renderTrends)
   - `if(ACTIVE_TAB===3)renderTrends();` (was renderStock)

No other JS changes needed — show() iterates [1,2,3,4,5] generically, and no code calls show(2) or show(3) programmatically.
  </action>
  <verify>
    <automated>cd /Users/jamal/projects/yumyums/hq && grep -n 'Stock\|Trends' inventory.html | head -20</automated>
  </verify>
  <done>Tab order in HTML is Purchases, Stock, Trends, Cost, Setup. JS show()/render() logic matches new positions.</done>
</task>

<task type="auto">
  <name>Task 2: Update test assertions and CLAUDE.md for new tab order</name>
  <files>tests/inventory.spec.js, CLAUDE.md</files>
  <action>
**tests/inventory.spec.js:**

1. **Tab label assertions (lines 114-115):** Swap expected text:
   - `#t2` should now `toContainText('Stock')` (was 'Trends')
   - `#t3` should now `toContainText('Trends')` (was 'Stock')

2. **Tab name in test title (line 112):** Update to `'shows 4 tabs: Purchases, Stock, Trends, Cost'`

3. **Stock tab clicks:** All `page.click('#t3')` and `page.locator('#t3').click()` for Stock become `#t2`:
   - Lines 236, 246, 257, 277, 291, 303 — `page.click('#t3')` -> `page.click('#t2')`
   - Lines 1461, 1473, 1501, 1538, 1557, 1643, 1689 — `page.locator('#t3').click()` -> `page.locator('#t2').click()`

4. **Trends tab click (line 318):** `page.click('#t2')` -> `page.click('#t3')`

5. **Trends section assertions (lines 319-320):** `#s2` -> `#s3`:
   - `await expect(page.locator('#s3')).toBeVisible();`
   - `await expect(page.locator('#s3')).toContainText('Spending Trends');`

**CLAUDE.md:**

Update both occurrences of the 5-tab layout description (lines 33 and 48) from "Purchases / Trends / Stock" to "Purchases / Stock / Trends":
- Line 33: `5-tab layout (Purchases / Stock / Trends / Cost / Setup)`
- Line 48: `Purchases (purchase events + pending review), Stock (levels + reorder suggestions), Trends (coming soon), Cost (coming soon), Setup (items + vendors management)`
  </action>
  <verify>
    <automated>cd /Users/jamal/projects/yumyums/hq && npx playwright test tests/inventory.spec.js -g "shows 4 tabs" --reporter=line 2>&1 | tail -5</automated>
  </verify>
  <done>All inventory tests pass with new tab order. CLAUDE.md reflects Purchases/Stock/Trends/Cost/Setup order.</done>
</task>

</tasks>

<verification>
Run the full inventory test suite to confirm nothing is broken:
```bash
cd /Users/jamal/projects/yumyums/hq && npx playwright test tests/inventory.spec.js --reporter=line
```
</verification>

<success_criteria>
- Stock tab is the 2nd tab (position 2) in inventory.html
- Trends tab is the 3rd tab (position 3) in inventory.html
- Clicking Stock loads stock data, clicking Trends shows coming-soon
- All inventory Playwright tests pass
- CLAUDE.md documentation matches new order
</success_criteria>

<output>
After completion, create `.planning/quick/260421-iug-move-the-stock-tab-before-trends-in-inve/260421-iug-SUMMARY.md`
</output>
