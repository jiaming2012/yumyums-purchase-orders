# Merge-intent — `counterparty-notice-prep` (Card 10, Track C)

The **preparable half** of roadmap card `counterparty-combined-notice`. This card
does NOT close B-29/B-137 — those close only when the operator SENDS the notice
(P-KR3). This card drafts the notice and corrects the contract docs to match the
live tree; the SEND is the operator's attended act.

**RF: `n/a — no code change`** (docs/audit card, `openspec: absent`).

## What this card does

1. Audits both contract docs against the LIVE handlers on tonight's tree
   (includes Cards 1 + 9):
   - `docs/contracts/inventory-period-summary.md` vs `PeriodSummaryHandler`
     (`backend/internal/inventory/handler.go`).
   - `docs/contracts/inventory-menu-cogs.md` vs `MenuCogsHandler`
     (`backend/internal/recipes/handler.go`).
2. Corrects every no-longer-true statement in-place (see "corrections" below).
3. Drafts the combined notice at
   `.night-crew/knowledge/reference/counterparty-notice-20260901-draft.md`.
4. Records the roadmap card as prep-complete-awaiting-operator-send (NOT DONE).

## Shared files touched

- `docs/contracts/inventory-period-summary.md` — **shared** (external contract of
  record). Additive header owner-line + a §0 addendum noting the Card 9 success
  log and that the corrected doc still matches HEAD post-Cards-1+9. No wire-shape
  claim changed — the 2026-08-03 audit already made both docs match HEAD, and
  tonight's tree introduced no response-shape change.
- `docs/contracts/inventory-menu-cogs.md` — **shared** (external contract of
  record). Same: additive owner-line + §0 addendum for the Card 9 `menu-cogs served`
  log line.
- `.night-crew/knowledge/reference/counterparty-notice-20260901-draft.md` — **new
  file**, this card's deliverable. Not shared with any other card.
- `.night-crew/knowledge/roadmap.md` — record Card 10 prep-complete under Activity 1;
  DO NOT flip `counterparty-combined-notice` to DONE.

## What must survive any merge

- Both contract-doc corrections (owner line + Card 9 log-line addendum in each §0).
- The notice draft file at the named path, audited against tonight's tree.
- The roadmap note that Card 10 (prep) is complete and `counterparty-combined-notice`
  awaits the operator's SEND — the card stays PLANNED, not DONE.

## What is safe to drop

- Nothing load-bearing. If a merge conflicts on the contract docs against another
  card that also touched `docs/contracts/*`, KEEP the substantive 2026-08-03 audit
  body verbatim (it is the spec sales-processor is built against) and re-apply only
  this card's additive owner-line + Card 9 addendum.

## Role-level call made (stated, not escalated)

**Framing: the notice is FIRST-PERSON (operator → self), not an outward apology to
an external maintainer.** The roadmap card still reads "sending is outward-facing —
the operator's act," but B-137 (morning triage 2026-08-03) established the operator
maintains sales-processor themselves — there is no external counterparty. B-137's own
lead prescribes the fix shape: add an owner line to both contract-doc headers, and
rewrite the notice as a first-person changelog + pre-deploy checklist with the apology
and counterparty-only questions stripped. B-137 is the more recent, more specific
finding and supersedes the card's pre-B-137 "outward-facing" wording. The prior
`NOTICE-sales-processor-2026-08-03-UNSENT.md` is left in place (it is cited evidence);
the new 2026-09-01 draft supersedes it and says so.
