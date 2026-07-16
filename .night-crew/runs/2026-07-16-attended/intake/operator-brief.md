# Operator Brief — 2026-07-16 (CONFIRMED)

> Evening PM session for the "Nothing silently lost" cycle (opened this morning).
> Drafted by the PM from the decisions signed at today's OKR session; operator
> confirmed 2026-07-16 evening (stage-3 wording expanded to the full four-element
> architecture at operator request before confirmation). New-find routing is
> delegated to PM judgment, argued in the PRD, stress-tested at grill-back.

## Outcome

Tomorrow we have a written, agreed plan that lists **every way the checklist system
can quietly throw away a crew member's work** — not just the one we caught on Friday's
checklist — and for each one, a test that will prove it's fixed. The plan also writes
down, in one sentence you've approved, the rule for what happens when you edit a
checklist while the crew is mid-shift: **the crew finishes the run they started; your
edit takes effect the next time the checklist runs.** Nothing gets built this cycle
until this plan is signed — it's the gate.

## Why / OKR

Friday's incident proved the checklist engine can lose crew work without anyone
noticing — the crew saw their checkmarks, but the system quietly dropped them. This
cycle exists to make that impossible. This plan is the cycle's first commitment (the
Product objective): every silent-loss mode enumerated and specified, with a
falsifiable acceptance test, before build starts. It advances all three Product key
results: the traced requirement list (KR1), the signed mid-run edit rule with its two
acceptance tests (KR2), and the record that all 15 queued backlog items went through a
door (KR3 — already done at this morning's session; this plan carries the record).

## Hard constraints

- Every requirement in the plan traces to either **a failure we reproduced** or **a
  named rule the system must never break** — anything else is out.
- The mid-run edit rule is recorded verbatim: *crews finish the run they started;
  edits take effect next run.*
- The quick protective fixes ship **before** the deeper rebuild — crews get protection
  now, not after a schema migration.
- Every fix proves itself broken first: the test fails before the fix, recorded.
- No new frameworks or dependencies; the house build stays static HTML + vanilla JS
  front, Go + Postgres back.
- Any change to production data happens only attended or with your explicit go-ahead.
- Every database schema change this cycle must be reversible (proven undo path) and
  preceded by a backup.

## Decisions made vs delegated

**Decided by the operator (today's OKR session):**
- Fix in three stages: keep field identity on edit + reject writes to deleted fields
  (stage 1), tell open devices about edits so they refresh (stage 2), then the
  structural fix (stage 3, design **and** build this cycle): every field keeps one
  permanent identity for life; an edit never rewrites the checklist in place but
  creates a new saved version of it; a checklist run is tied to whichever version
  existed when the run started; and crew answers are recorded against (that run +
  that permanent field identity) — so a save can never land on a field that "no
  longer exists," by construction rather than by defensive checks.
- Stages 1–2 ship to production before the stage-3 build starts.
- The stage-3 design gets its own written proposal you sign **before** any of its
  build work is dispatched.
- The stray blank item in the production catalog is renamed to "(Unnamed — needs
  review)" keeping its links — not deleted.
- Deferred this cycle: the onboarding video test fixture (needs your storage
  credentials), the photo-upload and offline test harnesses (new infrastructure), and
  the status-tool fix (lives outside this repo).

**Left to PM/planner judgment (delegated):**
- How finely the plan splits one loss-mode from the next (granularity).
- Which requirements ride which overnight card, how many cards per night, and the
  build order within the signed sequence — the planner's call against the night
  budget and quality bar.
- Whether additional loss-modes found during tonight's enumeration become
  requirements now or queue for later — PM judgment, argued in the plan.

## Known unknowns

- Whether more silent-loss modes exist beyond the reproduced one — tonight's
  enumeration sweep (two passes, including the surfaces with no screen) answers this.
- Whether production already holds orphaned crew work from past edits — checkmarks
  written against deleted fields that no screen will ever show again — and whether any
  of it is worth recovering or just counting.
- How risky the stage-3 migration is against live data — existing templates and
  half-done checklists must come through intact.

## References

- `.night-crew/knowledge/okrs.md` — the key results every requirement traces to
  (signed this morning).
- `.night-crew/knowledge/roadmap.md` — the cycle's activity cards and sequencing rule.
- `tests/repro-cut-task.spec.js` — the reproduced Friday failure, written as the test
  that stays red until stage 1 lands.
- `.night-crew/knowledge/BACKLOG.md` — the routed backlog with full root-cause notes.
- `CLAUDE.md` → "Workflows Data Persistence Rule" — the existing persistence contract.

## Out of scope

- Any net-new crew-facing feature — this cycle fixes and hardens, it does not add.
- The other apps (Inventory, Onboarding, Users, Purchasing) except the small carried
  fix-cards already on the roadmap.
- The deferred harness/fixture work (video, photo-upload, offline) and the
  status-tool repair — queued, not this cycle.
- Recovering historical lost data (if found, it's counted and reported; recovery is
  a separate decision).

## Amendments (2026-07-16 evening grill-back)

- **Premise correction (operator):** production has **no active users** of the
  Operations checklists — the "protect crews now" urgency behind shipping quick fixes
  first was mistaken, and wiping the production checklist data is acceptable.
- **Decision (operator):** go **straight to the robust system** — the quick
  protective fixes (old stages 1 and 2) are cancelled. Their useful parts are
  absorbed into the rebuild: keeping field identity happens by construction, and the
  "reject writes to fields that no longer exist" guard ships as part of it.
- **Consequences:** the OKRs, roadmap, and plan were amended in-session and
  re-validated; the counting-old-lost-work audit was dropped (the data it would count
  is being wiped); the production wipe/reseed happens attended at the rebuild's
  deploy, with a backup taken first.

## Second amendment (same grill-back, later)

- **The rule itself was revisited (operator):** instead of "the crew finishes the
  checklist they started; edits wait for the next run," the operator proposed —
  and after viability review delegated the final call to the PM — **frozen only at
  submit**: an unfinished checklist always shows the current version on every
  device; your mid-shift edit appears immediately and the crew completes the
  updated list; once submitted, the record never changes; a manager's rejection
  reopens it against the current version.
- **PM decision (delegated):** frozen-at-submit adopted. You are the editor and
  want corrections live; the crew is small and in one kitchen; and this shape needs
  **no database migration at all** — so the wipe from the first amendment is moot.
- **The operator's quality bar (recorded verbatim in spirit):** multi-device sync
  is the product — checklists, every field type, sub-lists, submitting and
  un-submitting, and the progress bars derived from them stay in sync across all
  devices, always, with every edge case walked. This became its own requirement
  with a device-to-device test matrix.
- **Two rules attached, for your sign-off:** cutting a field discards its
  unsubmitted answers and the Builder warns you first, naming the count; a
  rejection flag on a field you've since cut dissolves visibly rather than
  blocking or vanishing.
