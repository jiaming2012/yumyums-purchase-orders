# merge-intent — P6 `period-summary-contract-notice`

Run: `20260803-autonomous` · Branch: `card/p6-period-summary-contract-notice` (off `overnight-20260803`)
Card: Track B / stretch, promoted from B-29.

## Shared files touched

| File | Why |
|---|---|
| `docs/contracts/inventory-period-summary.md` | The Phase 21 contract of record (`21-SALES-PROCESSOR-CONTRACT.md`). Card part 1 corrects the `pending_review_ids` drift statement; card part 2 adds the row-by-row drift audit. Card A1 (`67b2c9f`, merged) already rewrote `:27`, `:28`, `:68` and A5 for the timezone; **this card edits the same rows again**, so a merge that resolves in favour of A1's text silently drops this card's findings. |
| `docs/contracts/inventory-menu-cogs.md` | The Phase 999.2 sibling contract (`999.2-SALES-PROCESSOR-CONTRACT.md`). Card part 2 audits every row here too. A1 added `:31` and A10 (`:453`); this card marks both **inaccurate for this endpoint** (menu-cogs has no timezone cast at all), so again this card's text must win over A1's. |
| `docs/contracts/NOTICE-sales-processor-2026-08-03-UNSENT.md` | New file, no conflict surface. Card part 3 — the drafted outbound notice. **UNSENT.** |
| `.night-crew/knowledge/BACKLOG.md` | Discoveries filed as B-71..B-74 (allocated to this card). Append-only at the tail of the open-items list; conflicts with S1a/S1b are positional only (they used B-61..B-70). |
| `.night-crew/knowledge/roadmap.md` | Required status flip for the P6 card in the same change set. |

## What must survive any merge

1. **The `mercury_category` + `reason` clauses on `completeness.pending_review_ids`** (`inventory-period-summary.md`). A1's amendment corrected the `COALESCE` half of this row and **left a second, later behaviour change unstated** (`d41faef`, 2026-06-06). If a merge restores A1's version of that row the contract goes back to overstating what blocks payroll. This is the single most load-bearing line in the change set.
2. **The `HQ_COGS_CATEGORY_ALLOWLIST` env-var row** added to §2 of `inventory-period-summary.md`. It is a wholly undocumented input that changes COGS; it has never appeared in either contract.
3. **The Drift Audit sections** (`§0`) added to both documents. They are the card's `done_when:` deliverable — every `:NN` row marked CONFIRMED or DRIFTED-with-commit. Dropping the section fails the card even if the inline corrections survive.
4. **The five menu-cogs wire-shape corrections** (`name` → `menu_item_name`, `menu` absent, `menu_subgroup` omitted not null, undocumented `toast_master_id`, `units_sold` is a JSON float). These were observed by marshalling the real structs, not read off the code. They are decode-breaking for the client struct the contract itself publishes.
5. **The `UNSENT` marker and the "nothing was sent" statement** in the notice file. The notice is the operator's act; the file must never merge in a state that reads as delivered.

## What is safe to drop

- The prose ordering inside the `§0` Drift Audit tables — rows may be re-sorted freely as long as no row is lost.
- Any wording overlap with A1's timezone text where the two say the same thing (`:28`, A5). If A1's phrasing wins on the **timezone-only** sentences, nothing is lost; only the `pending_review_ids` gate clauses and the menu-cogs "shares date semantics" claim must resolve in this card's favour.
- The backlog entries B-71..B-74 may be renumbered if another card lands in that range first; their content must survive, their numbers need not.

## Explicitly not here

- **No production code changed.** The defect is documentary. `backend/internal/inventory` and `backend/internal/recipes` are untouched — the footprint permitted a code fix "only if warranted" and it was not. Nothing here can conflict with a code-carrying card.
- **No test files changed** — nothing here.
- **No `sw.js` / frontend / build artifact** — nothing here. G4 is a no-op parity check.
