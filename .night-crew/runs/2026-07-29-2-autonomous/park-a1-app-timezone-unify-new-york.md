# PARK — Card A1 `app-timezone-unify-new-york`

> **Parked by the orchestrator at merge time, not by the implementer.** The card reported DONE.
> Its G6 adversarial review returned **REJECT**, and the orchestrator verified the decisive
> evidence first-hand before acting on it. **The branch is NOT merged.**
>
> Branch `card/a1-app-timezone-unify-new-york` @ `8da3ded`, worktree preserved at
> `/home/jcole/projects/hq-worktrees/a1-app-timezone-unify-new-york`. Nothing deleted, reset, or
> renamed. The work is intact and good; what it lacks is authority, not quality.

## Why this parked

The slate's PARK trigger, verbatim:

> **PARK trigger:** if any site's zone turns out to be **deliberately** Chicago for a reason not in
> the roadmap card — an external contract, a vendor's cutoff, **sales-processor expecting
> Chicago** — that is a **product question, not a refactor. Park with the evidence.**

It fired. The evidence, verified by the orchestrator directly rather than inherited from the
review:

| Where | What it says |
|---|---|
| `21-SALES-PROCESSOR-CONTRACT.md:27` | "Both are interpreted in `America/Chicago` (the food-truck operating timezone)." |
| `21-SALES-PROCESSOR-CONTRACT.md:67` | `completeness.pending_review_ids` is published to the consumer as `(created_at AT TIME ZONE 'America/Chicago')::date BETWEEN from AND to` — **the exact expression this card replaced** |
| `21-SALES-PROCESSOR-CONTRACT.md:319` | **"A5: `America/Chicago` is the correct operating timezone** — this matches `backend/internal/purchasing/repurchase.go:71` in HQ. **If the food truck moves to a different TZ, both repos must update.**" |
| `999.2-SALES-PROCESSOR-CONTRACT.md:30` | Same pin, for the menu-COGS endpoint. |

**A5 is a named cross-repo coordination assumption.** Ledger decision 83 — the operator ruling that
the app's timezone is New York — was read in full by the reviewer and **never addresses the
published contract or A5**. It names sales-processor only as a downstream consumer.

Pointedly, A5's anchor site is `repurchase.go:71`, which is one of the two "sites found beyond the
slate" this card changed. So the card altered HQ's half of a two-repo agreement, updated neither
contract document, and recorded in `roadmap.md:571` — *"Nothing parked — no site turned out to be
deliberately Chicago"* — a claim that is **false against the repo's own artifacts**.

**The consequence is concrete, not procedural.** If sales-processor computes its Monday–Sunday
payroll window in Chicago while HQ now evaluates the completeness gate in New York, the two
disagree for one hour at each period edge, on rows with no extracted `event_date`. The card did not
remove the two-boundary bug on that path — **it moved the boundary across a repo line, where
nothing in this repository can detect the disagreement.**

## Why the good half was not salvaged

The slate forbids it, in the card's own text:

> 🛑 **One card, all sites.** Piecemeal leaves two boundaries disagreeing, which is exactly today's
> bug. **If the card cannot cover every site, park it rather than land half.**

Landing the checklist/queue half (the D-1 fold) and holding the money paths is precisely the
piecemeal outcome that clause prohibits. And the unresolved question is not cosmetic: it decides
**whether the money-path sites should move at all**. Fixing everything else first would be building
on an answer nobody has given.

## The second blocking finding — a missed site inside a file the card touched

`inventory.html:2713` (form at `:2681`):

```js
var userTz = Intl.DateTimeFormat().resolvedOptions().timeZone || 'America/New_York';
api('/api/v1/purchasing/repurchase-reset/config', {method:'PUT', … timezone: userTz})
```

The Setup-tab **Badge Reset** form writes the **browser's** timezone into
`repurchase_reset_config.timezone` on every save. This is the structural twin of
`purchasing.html:295` — the site the card correctly identified as *"the client was the reason the
cutoff boundary stayed on Chicago"* — for the **same table** `0072` UPDATEs and the same table
`repurchase.go:71` was changed for.

Because that upsert is DELETE+INSERT with an explicit zone, **the next badge-reset save overwrites
`0072`'s UPDATE.** `inventory.html` is a file this card edited, and the merge-intent note's
"deliberately left alone inside touched files" section does not mention these lines.

🛑 **And this may itself be a second product question.** `tests/inventory.spec.js:2022` is named
*"badge reset saves with browser timezone, not hardcoded value"* and asserts
`body.timezone === 'America/Chicago'` from a Central-set browser context — **it passes at the
card's HEAD**. Someone once decided badge reset follows the operator's device. No ledger or roadmap
entry records that decision. Whether it should now follow the app zone instead is a call for the
operator, not for a 3am refactor.

## A third finding, high but not blocking

`backend/internal/inventory/trends.go:89-98` — `trendsWindow` computes the Trends tab's 12-week
COGS window with a bare `time.Now()` and **no `LoadLocation`**, so it runs in the server's local
zone (UTC in the container). It sits **eleven lines above `trends.go:240`, which the card did
edit**, and it is the direct analogue of `recipes/cost.go:costWindow` and `CurrentWeekStart`, both
of which the card correctly zoned.

Left as-is, the app would carry **two 12-week COGS windows on two different zones** — Recipes/Cost
on New York, Trends on server-local. That is the "two boundaries disagreeing" state this card
exists to end. Undisclosed in the merge-intent note. It belongs in the resuming card's scope.

## What is good in this branch, and should be kept

The review was emphatic that this is well-built work that landed a decision it lacked authority to
land. Verified by the reviewer, independently:

- **`0072` is clean on fix-forward.** It rewrites exactly two columns (`timezone`, `updated_at`) on
  two single-row config tables, both predicated on `WHERE timezone = 'America/Chicago'`. It touches
  no COGS table, no `purchase_events`, no `pending_purchases`, no `daily_menu_sales`, and
  deliberately not even `repurchase_reset_config.last_reset_at`. **No already-reported figure can
  move.** The `Down` migration restores DEFAULTs only and explicitly refuses to revert row values.
- **The UPDATE is necessary**, and the reviewer tested the claim rather than accepting it: both
  tables are written by DELETE+INSERT with an explicit `timezone` parameter, so the column DEFAULT
  never fires once a row exists. Re-pointing DEFAULTs alone would leave the live Chicago row
  forever. The value is load-bearing — `purchasing/scheduler.go:257-274` reads it to compute
  `weekStart` for alert idempotency and the PO cutoff lock.
- **The red-first work is genuine and reproduced independently**, including the honest disclosure
  that the red commit also carries the `weekStartNow` clock seam (with the zone still Chicago) —
  because otherwise the Go test would not compile, and a compile error is not a red. The reviewer
  confirmed the seam alone does not fix the bug.
- **DBL-05's fix is not a tautology**, proven by injection: the reviewer replaced `appDateString`'s
  body with the old UTC expression and the test failed as designed. The card had found a test that
  was *asserting the defect*.
- **`cost_test.go`'s fixture flip is a real behaviour change surfaced by an existing test**, not a
  test bent to fit code, and the added 22:30 NY sibling genuinely brackets the boundary.
- Mechanical gates all hold: 7/7 trailers parse, `sw.js` byte-identical to a fresh rebuild with no
  manual cache bump, versions unmoved, and the persistence-rule claim verified (the edits are
  read-side date comparisons and queue metadata — nothing crosses
  `autoSaveField → DRAFT_RESPONSES → hydrateFieldState`, so no back-and-reopen test is owed).

## Two nits the resuming card should absorb

- **`APP_TIMEZONE` in `sync.js:555` is a hand-copied literal.** Nothing mechanical keeps it equal to
  `users.DefaultTimezone` — no test asserts it, and `/api/v1/health` exposes no app-timezone field
  the frontend could read. The 🛑 comment is a convention, **which is the same mechanism that
  produced the bug this card is fixing.** A one-line spec asserting the two match would close it.
- **`appDateString()`'s UTC fallback is silent.** It fires if `Intl.DateTimeFormat` throws or omits
  parts — realistically a small-ICU runtime or an embedded WebView without full tzdata. Unlikely on
  iOS/Android PWAs, but it would **silently restore the exact UTC boundary the card removed**, with
  no warning.
- The money path has **no boundary test**. `pendingPeriodDateExpr` — the COGS window and the
  completeness gate feeding payroll, the card's headline site — is covered by no red.
  `period_summary_test.go:1188` uses a timestamp that resolves identically in both zones, so it
  cannot distinguish them, and six comments in that file still assert Chicago semantics.
- **`receipt/worker.go:1056`** — `parseEventDate` falls back to server-local `time.Now()` when
  Mercury's `CreatedAt` is unparseable, and that value becomes `pending_purchases.event_date`, i.e.
  a COGS period assignment. Outside this card's footprint; file it.
