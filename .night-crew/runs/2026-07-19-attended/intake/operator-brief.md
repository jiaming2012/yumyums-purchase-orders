# Operator Brief — 2026-07-19 (DRAFT — awaiting operator confirmation)

> Evening PM session for the "Prove & surface" cycle (opened this morning by
> `/nc-okr-session`). Drafted by the PM from the decisions signed at today's OKR
> session; no inbox items arrived this evening (`.night-crew/inbox/` empty) and no
> separate operator brief was written, so this brief synthesizes the signed OKRs +
> roadmap into the seven-field shape for confirmation — same pattern as 2026-07-16.
> New-find routing is delegated to PM judgment, argued in the PRD, stress-tested at
> grill-back.

## Outcome

Tomorrow we have a written, agreed plan for two things at once:

1. **Two new numbers screens.** A **Trends** tab that shows how much you spent each
   week, broken out by the kind of thing you bought (proteins, beverages, and so on),
   over the last couple of months as a simple bar chart plus a table — and a **Cost**
   tab that shows, for each menu item, how many you sold, what it brought in, what the
   ingredients cost, and the profit left over, with the best and worst performers
   called out. Both are locked so only people you've given access to can open them.
   Where the sales numbers aren't there yet, the screen says so plainly instead of
   looking broken.

2. **A test net for the sync bug we keep catching by hand.** Three times last cycle,
   while you were using the app, you found a case where a change on one phone didn't
   show up on your other phone — and each time we fixed it after the fact. The plan
   writes down every combination that has to stay in sync (who changed it, who's
   watching, what kind of change, what they should see) and turns each into a test, so
   that whole family of bugs gets caught by the test suite before it ever reaches you.

Nothing gets built until the plan is signed — it's the gate.

## Why / OKR

This is a two-half cycle. The **surface** half exists because the numbers you're
already collecting (receipts, recipes, menu sales) aren't shown anywhere useful yet —
Trends and Cost are the two screens that turn that data into decisions. The **trust**
half exists because the same class of sync bug escaped to your own hands three times
last cycle; a written coverage plan makes that class stop escaping. The plan is the
cycle's first commitment (the **Product** objective): both halves enumerated and
specified, every requirement tracing to a key result, before build starts. It advances
Product KR1 (the traced requirement list as the blocking gate), KR2 (each of the 3
escaped bugs mapped to a test that would have caught it), KR3 (the access rule for the
new tabs recorded as one signed decision), and KR4 (all 12 queued backlog items routed
through a door).

## Hard constraints

- Every requirement traces to either **a bug we reproduced** or **a named outcome you
  want** — anything else is out.
- The two new screens are **locked behind access you grant** in the Users app, and the
  lock is enforced by the server, not just hidden in the page — someone without access
  gets turned away even if they go around the screen.
- The numbers must be **right**: every weekly-spend cell equals the sum of the receipts
  behind it, and every menu item's profit equals its revenue minus its ingredient cost,
  checked against a worked example down to the cent.
- The Cost screen may ship to production with **thin sales data** — it must render
  honestly with an empty or low-data state where the sales aren't there. Correctness is
  proven on seeded test data, not on live sales.
- **No new charting library or framework** — the bars are drawn with plain built-in
  web tech, keeping the house style (static HTML + vanilla JS front, Go + Postgres
  back).
- Every fix and every new endpoint **proves itself first**: the test fails before the
  work, recorded, then flips to passing.
- Any database change this cycle is **reversible** (a proven undo) and preceded by a
  backup; any production data change happens only attended or with your explicit
  go-ahead.
- Both tabs must actually reach **production** (`hq.yumyums.kitchen`), not just the dev
  server, with versions matching.

## Decisions made vs delegated

**Decided by you (today's OKR session):**
- The cycle is **mixed**: feature (Trends + Cost) **and** trust (sync coverage).
- **Trends** = weekly spend by item-group over ~8–12 weeks (chart + table).
- **Cost** = per-menu-item food-cost table (units, revenue, ingredient cost, margin,
  food-cost %) **plus** a best/worst-performers highlight.
- Charts are **inline drawn** — no new dependency.
- The two tabs are **gated through the Users app** access model.
- **Cost in production accepts thin data** — it renders honestly where sales are
  absent; correctness is proven on seeded dev fixtures.
- **Retire the last test waiver** (the one isolation-polluted test) so `task test`
  finally exits clean.
- **Route all 12 queued backlog items** through a door this cycle.

**Left to PM/planner judgment (delegated):**
- **How the access lock is modeled** — whether the two tabs share one grant or get
  their own finer-grained permission. Recorded in the plan as a delegated decision, to
  be **settled at the design sign-off**, not pre-decided tonight.
- **The rule for receipt lines not yet linked to a catalog item** in the weekly-spend
  totals (exclude them, or show them as an "unlinked" bucket) — proposed in the plan,
  ratified at grill-back.
- Granularity of the requirements, which requirement rides which overnight card, how
  many cards per night, and build order within the signed sequence — the planner's call
  against the night budget and quality bar.
- Whether extra surfaces found during tonight's enumeration become requirements now or
  queue for later — PM judgment, argued in the plan.

## Known unknowns

- **How much live sales data production actually has.** Toast sync is inert in prod
  (`TOAST_SYNC_INTERVAL=0`), so the Cost screen may be mostly empty there — hence the
  "accept thin, render honest" acceptance bar. Correctness is proven on seeded data.
- **The unlinked-receipt-line question** — how many receipt lines in the real data
  have no catalog item attached, and therefore whether the exclude-vs-bucket rule
  changes the totals materially.
- **Whether the sync enumeration turns up a fourth escape** beyond the three we already
  caught — tonight's two-pass sweep (including the surfaces with no screen) answers this.
- **Whether the duplicate-alert risk is real** now that receipts, alerts, and Cliq run
  in production against the same accounts as dev — watched over the cycle, one side
  disabled if needed.

## References

- `.night-crew/knowledge/okrs.md` — the key results every requirement traces to
  (signed this morning).
- `.night-crew/knowledge/roadmap.md` — the cycle's activity cards, the two gates, and
  the parallel-track sequencing rule.
- `.night-crew/knowledge/reference/qa-gap-20260717-live-sync-access.md` — the three
  escaped sync defects, with the matrix cells that would have caught them.
- `.night-crew/knowledge/BACKLOG.md` — the 12 `· new` items to route, with root-cause
  notes.
- `inventory.html:993-998` — the current Trends/Cost "coming soon" stubs the tabs
  replace; `:272-285` the inline stub cards.
- `CLAUDE.md` → "Definition of Done" (State Enumeration + self-verification ritual) and
  the Menu-COGS / period-summary endpoint contracts the new endpoints extend.

## Out of scope

- **Turning on Toast sales sync in production** — the Cost tab accepts thin prod data;
  enabling Toast is not a dependency this cycle.
- **Pre-deciding the fine-grained permission model** — the decision is *recorded* in
  the plan tonight, but *settled* at the design sign-off (Activity 2).
- **The small editprop follow-ups** (transactional op emission for Create/Archive,
  atomic approval+feedback, fail-note conflict live-render) — they stay backlog
  tidy-ups, not requirements this cycle.
- **The deferred harness/fixture work** — photo-S3 harness, offline-IndexedDB harness,
  onboarding video fixture — queued, not this cycle.
- **Net-new crew-facing features** and the other apps beyond the carried cards — this
  cycle surfaces numbers and hardens sync; it does not add elsewhere.
