# DECISIONS-NEEDED — overnight-20260722

> Open forks from the autonomous run of 2026-07-22. Reader = the operator.
> Resolve via `/nc-morning-triage`. The run did not decide any of these.

---

## F1 — `trends-spend-by-group-endpoint` — **PARKED** (card's own PARK trigger fired)

**Status:** implementation complete and technically sound, **not merged**. Worktree preserved at
`scratchpad/wt-f1`, branch `card/f1-trends-endpoint` @ `88cab9d` (2 commits: red-first test
`fc0f08d`, implementation `88cab9d`). Nothing from F1 is on `overnight-20260722`.

**Why parked:** the card's PARK trigger reads — *"if proration can't reconcile with
`period-summary` on the shared fixture (AC-6 identity breaks) → PARK with the failing
reconciliation case; do not ship a number that disagrees with payroll's."* The identity held on
the fixture as authored, but the G6 adversarial reviewer demonstrated that **the fixture is rigged
on every axis that matters** (one line item per event, `Σlines == total − tax` by construction,
every event hardcoded `mercury_category='COGS'`, zero pending rows). With realistic fixtures the
identity breaks **five independent ways**.

**This is not an implementer error.** The implementer followed the SIGNED design §2.2 verbatim and
self-flagged the `mercury_category` divergence rather than silently patching it — the procedurally
correct call. **The design is what's defective.** The fix is a design amendment, which is
operator-only; the run correctly refused to improvise it.

### The failing reconciliation cases (G6-measured, Trends vs `period_summary.cogs_incl_tax`)

| Probe | Scenario | Trends | period-summary | Δ |
|---|---|---|---|---|
| **B1** | Partial line coverage — event total 110.00, tax 10.00, only 90.00 itemized (a delivery fee on the receipt, not a catalog line) | 99.00 | 100.00 | **−1.00** |
| **B2** | One reviewable `pending_purchases` row in window | 110.00 | 185.00 | **−75.00** |
| **B3** | One `mercury_category='Software'` event (500.00) beside a COGS event | 610.00 | 110.00 | **+500.00** |
| **B4** | 20 cells, 4-dp prices, 8.25% tax — per-bucket `ROUND` vs whole-window `ROUND` | 220.0000 | 220.0300 | **−0.03** |
| **B5** | Event with `total == tax` (25.00/25.00) — `NULLIF` guard ⇒ factor 1, lines are 0 | 0.00 | 25.00 | **−25.00** |

**B1 alone is disqualifying** — it is the *normal* case on real receipts (any unitemized fee), not
an exotic edge.

**Root cause:** Trends and `period-summary` are structurally different computations.
`period-summary` = `Σ(qty×price) + Σ(tax) + Σ|pending.bank_total|`; Trends =
`Σ(qty×price × total/(total−tax))`. These are algebraically equal **only** when every event's lines
sum exactly to `total − tax` **and** there are no pending rows. Real receipts satisfy neither.

### The fork — three operator decisions, all design amendments to §2.2

1. **`mercury_category` filtering.** `period-summary` filters `mercury_category = ANY($allowlist)`
   at three CTEs (`handler.go:1335`, `:1350`, `:1406`). Design §2.2's SQL sketch has **no such
   filter**, so Trends has none. On production data the gap is **unbounded** — it is the entire
   non-COGS half of the Mercury feed (software, equipment, fuel, insurance, rent), landing on a tab
   labelled "spend by group" that sits next to payroll's COGS number.
   **G6 recommendation:** add `AND pe.mercury_category = ANY($3)` with the same allowlist
   `period-summary` uses, threaded through `TrendsHandler(pool, cogsAllowlist)`.
   *Magnitude on real data is unmeasured — G6 had no production access. The direction
   (Trends over-reports) is certain; the size is not.*
2. **`pending_purchases`.** Trends never reads them; `period-summary` counts unreviewed reviewable
   rows as COGS. Does Trends include them, or carry a completeness note (as it already does for
   `unlinked`, which is a different gap)?
3. **What "reconciles with `period-summary`" MEANS when `Σlines ≠ total − tax`** — which on real
   receipts is the common case. Until this is stated, AC-6 is not a testable contract. §2.1 also
   claims Trends reconciles with `menu-cogs` (a *third* distinct number, since `menu-cogs` applies
   `AND pli.purchase_item_id IS NOT NULL` plus recipe allocation) — **untested by this card and
   unexamined by G6.**

### Knock-on: **F3 `trends-tab-frontend` could not run tonight**

Per the slate's own dependency note — *"if F1 PARKs, F3 cannot run (no API) — skip to F2/F4."*
F3 was not dispatched. Track F ran F2 → F4 → F5 instead.

### To un-park

Amend design §2.2 on the three points above, then re-dispatch F1 as a card whose red-first test
uses an **unrigged** fixture — G6's B1/B2/B3/B5 are ready-made cases to seed it with. The existing
implementation is a good base; the SQL needs the category filter, a pending-rows decision, and a
rounding-site change.

---

## Standing bugs surfaced (not forks — backlog candidates)

These need no decision tonight but should be routed at triage.

1. **HIGH — `NULLIF(total−tax,0)` silently zeroes money** (F1 probe B5). An event with
   `total == tax`, or lines unparsed while tax is recorded, contributes 0.00 to Trends and its full
   total to `period-summary`. Design §2.1 explicitly promises *"money never silently dropped."*
   Applies to the parked F1 code; check whether the same guard pattern exists elsewhere.
2. **MED — money is `float64` end-to-end** across the inventory package (`period-summary` too, so
   this is a pre-existing repo convention, not an F1 regression). JSON can emit
   `23.099999999999998`; it compounds the B4 per-bucket rounding drift. Cents-as-int or a decimal
   string is the repo-wide correct fix.
3. **MED — parallel-suite cross-package DB interference is a standing hazard.** G6 verified
   independently at the **base** commit (F1 absent): `go test ./...` under default parallel `-p` is
   broadly red — `internal/inventory` (6), `internal/purchasing` (4), `internal/receipt` (9),
   `internal/recipes` (5+). Multiple packages `TRUNCATE` a shared DB concurrently. `-p 1` is green.
   **This means `-p 1` is load-bearing for every card's G1/G2 signal** and should be written into
   the standing run mechanics, not rediscovered per card.
4. **LOW — `time.Now()` is server-local, not `America/Chicago`.** `trendsWindow` uses the
   container's local zone while `period-summary` is explicit about Chicago (`handler.go:1346`,
   `:1412`, `:1496`). On a UTC container, 18:00–24:00 Chicago computes the window a day ahead; when
   that crosses a Monday it shifts `from` by a **full week bucket**.

---

## Documentation defect — the inherited-rules pointer is dangling

Every slate from 07-15 through 07-22 inherits its standing rules and gates G1–G6 "unchanged by
reference from `reference/overnight-run-plan-20260707.md`". **That file does not exist** and does
not appear anywhere in the repo. The gates' real origin is
`.night-crew/runs/2026-07-09-attended/slate-20260710.md` § "Run mechanics" (G1–G4, G6), with the
app-code adaptation restated inline in `slate-20260714.md` — which is the form applicable to code
cards. This run executed against that reconstruction.

**Fix at triage:** either create `reference/overnight-run-plan-20260707.md` as the real standing-rules
document, or correct the pointer in the slate template. A dangling inherit in the one document that
defines the run's gates is a latent single point of failure.
