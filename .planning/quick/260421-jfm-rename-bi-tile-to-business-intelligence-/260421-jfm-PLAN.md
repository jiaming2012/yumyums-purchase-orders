---
phase: quick
plan: 260421-jfm
type: execute
wave: 1
depends_on: []
files_modified: [index.html]
autonomous: true
requirements: []

must_haves:
  truths:
    - "Home grid shows 'Business Intelligence' instead of 'BI'"
    - "Home grid shows 'Purchase Orders' instead of 'Purchasing'"
  artifacts:
    - path: "index.html"
      provides: "Updated tile titles"
      contains: "Business Intelligence"
  key_links: []
---

<objective>
Rename two tiles on the HQ home grid in index.html:
- "BI" tile becomes "Business Intelligence"
- "Purchasing" tile becomes "Purchase Orders"

Purpose: Clearer tile labels for the crew.
Output: Updated index.html with renamed tiles.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@index.html
</context>

<tasks>

<task type="auto">
  <name>Task 1: Rename BI and Purchasing tiles</name>
  <files>index.html</files>
  <action>
In index.html, make two text changes:

1. Line 73 — Change the BI tile title from "BI" to "Business Intelligence":
   `<div class="tile-title">BI</div>` -> `<div class="tile-title">Business Intelligence</div>`

2. Line 46 — Change the Purchasing tile title from "Purchasing" to "Purchase Orders":
   `<div class="tile-title">Purchasing</div>` -> `<div class="tile-title">Purchase Orders</div>`

Do NOT change any other attributes, hrefs, descriptions, or the TILE_SLUGS mapping (purchasing slug stays as-is since it maps to the filename, not the display name).

After editing, run `node build-sw.js` to regenerate the service worker with updated content hashes.
  </action>
  <verify>
    <automated>grep -q "Business Intelligence" index.html && grep -q "Purchase Orders" index.html && ! grep -q '"tile-title">BI<' index.html && ! grep -q '"tile-title">Purchasing<' index.html && echo "PASS" || echo "FAIL"</automated>
  </verify>
  <done>Home grid shows "Business Intelligence" and "Purchase Orders" as tile titles. SW rebuilt.</done>
</task>

</tasks>

<verification>
- grep for "Business Intelligence" in index.html confirms rename
- grep for "Purchase Orders" in index.html confirms rename
- No references to old "BI" or "Purchasing" tile titles remain
- TILE_SLUGS mapping unchanged (still maps purchasing.html to 'purchasing')
</verification>

<success_criteria>
index.html tile grid displays "Business Intelligence" where "BI" was, and "Purchase Orders" where "Purchasing" was. Service worker rebuilt.
</success_criteria>

<output>
After completion, create `.planning/quick/260421-jfm-rename-bi-tile-to-business-intelligence-/260421-jfm-SUMMARY.md`
</output>
