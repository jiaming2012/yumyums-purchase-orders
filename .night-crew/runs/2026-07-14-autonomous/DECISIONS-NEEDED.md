# DECISIONS-NEEDED — overnight-20260714

> **RESOLVED 2026-07-14 — recorded as `ledger.md` T-9 / T-10.** No fork blocked triage. All 7 cards
> signed DRAFTING → DONE (T-9; Eng KR-1 4 → 0, QA KR-1 +3 hardened tests). F-1 (Ops NFR-3 backend
> resubmit `require_photo` gate) stays in `BACKLOG.md`; its scheduling is **delegated to the planning
> agents** (PjM/PM/eng) at slate-planning, not hand-picked at triage — operator rider, recorded as a
> rule in T-10. F-2 (orphaned `users.html:122` div) stays backlogged, folds into a future Users card.
> File kept below as the analysis record.

> Operator forks and follow-ups surfaced by the run. The run **executes**, it does not decide —
> anything requiring a judgment call is parked here for `/nc-morning-triage`.

## Open forks requiring an operator decision

**None.** All 7 cards landed clean (G6-PASS), 0 parked. No card hit a footprint breach, an unplanned
migration, a new field type, or an operator-only question. The one pre-existing fork (D-5, the
Onboarding FR-16/NFR-4 video-pipeline waiver) was already resolved at triage 2026-07-13 (`ledger.md`
T-8) and no card in this slate touched it.

## Follow-ups surfaced (for awareness — do not block triage)

### F-1 · Ops NFR-3 — backend resubmit `require_photo` gate (deferred)
- **What:** The photo-required gate is enforced front+back for a **field-level** required photo field.
  The **rejection-driven resubmit** case (a manager rejects a field with `require_photo=true`; the
  crew must attach a new photo before resubmitting) is enforced **frontend-only** — a direct-API
  resubmit can still bypass it.
- **Why it was deferred (in-footprint decision, not a fork):** `SubmitChecklistInput`
  (`backend/internal/workflow/model.go`) carries only `TemplateID`/`IdempotencyKey`/`Responses`/
  `FailNotes` — no `submission_id` and no rejection context. `validateFailNotes` therefore cannot
  know a prior rejection flagged `require_photo` for a field. Enforcing it server-side needs a new
  `submission_rejections` join — beyond the ops-nfr3 validation seam, which the card's PARK trigger
  explicitly scoped out. G6 independently confirmed the deferral is real and acceptable per the
  card's allowance.
- **Disposition:** logged to `BACKLOG.md` as a scoped fix-card. Operator may accept as backlog or
  schedule it as a small Activity-4 fix-card next slate. **No decision blocks this run's triage.**

### F-2 · Orphaned `users.html:122` `<div id="s3">` (cosmetic cleanup)
- The `users-stale-e2e-repair` card repointed the tests and renamed the misleading `s3` var, but left
  the now-fully-orphaned dead `<div id="s3">` at `users.html:122` (leftover from the 3-tab→2-tab
  refactor). Out of that card's repoint+rename scope. G6 flagged it as optional future cleanup.
  Logged to `BACKLOG.md`. Trivial; fold into any future Users card.
