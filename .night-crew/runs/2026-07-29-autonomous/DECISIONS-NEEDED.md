# Decisions needed — `overnight-20260729`

> **RESOLVED 2026-07-28 — recorded as `.night-crew/knowledge/ledger.md` T-26, decisions 82–88.**
> All four forks walked with the operator at morning triage. Kept as the analysis record; the
> ledger carries what was decided and why.
>
> - **D-1** → decision **83**. Answered with a rule, not an answer: *"the apps time zone should be
>   NY time."* Checking it against the tree turned a four-site fix into a cross-cutting one — the
>   codebase runs **two conflicting timezone regimes**, with `America/Chicago` hardcoded in the COGS
>   / purchasing / recipes money paths while the user-facing default is New York. New card
>   `app-timezone-unify-new-york`: one card, all sites, **fix forward only**.
> - **D-2** → decision **84**. Documented constraint is sufficient; no marker gate.
> - **D-3** → decision **82**, and it did not go as this file anticipated. A concurrent attended
>   session had already recorded a sign-off (decision 80). Walking the plates at triage surfaced a
>   defect that sign-off could not have accounted for, so decision 80 is **superseded in part**,
>   amendments **A-1** and **A-2** are required, and the UI card returns to **ATTENDED-BLOCKED**.
> - **D-4** → decision **85**. Both filed to `BACKLOG.md` as **B-19** and **B-20**.
>
> Three findings the run did **not** report were added by the adversarial reproduction and resolved
> as decisions **86** (unparseable trailers → B-21), **87** (traversal claim travels as its scope
> statement) and **88** (two roadmap record defects corrected).


> Run: `overnight-20260729` · Slate: `slate-20260729.md` (signed 2026-07-28, 4 cards)
> **No card parked.** All four landed. Every item below is a fork the run **refused to decide**,
> not a blocker it hit — surfaced for morning triage rather than improvised at 3am.

---

## D-1 · The app's day boundary is 19:00 America/Chicago, and Card B just made it load-bearing

**Status:** OPEN — product decision, deliberately outside Card B's footprint.
**Found by:** G6 adversarial review of Card B (finding A, non-blocking).
**Sites:** `sync.js:565`, `workflows.html:1758-1762`, plus three pre-existing comparisons at
`workflows.html:2274`, `:2308`, `:2674`.

`currentSubmitPeriod()` is `new Date().toISOString().slice(0,10)` — **UTC**. In CDT that means the
app's "today" rolls over at **19:00 Chicago** (18:00 in CST), i.e. in the middle of dinner service.

**The failure it now enables.** Crew is offline. At 18:58 Chicago they press Submit on the closing
checklist → the queue entry is stamped `period = D`. Server still unreachable. At 19:02 they press
Submit again → `currentSubmitPeriod()` now returns `D+1` → the entry is no longer "current" → a
fresh `idempotency_key` is minted → on drain, **two submission rows for one operational evening**.
Before this card, that same double-press produced one row.

**Why the run did not fix it.** The card is *self-consistent* with the app's existing day model: all
three pre-existing "already submitted today" comparisons use the identical UTC expression, so at
19:02 the checklist list already shows the checklist as not-submitted-today. Changing the definition
means moving **all four sites at once**, and deciding what "today" means for a food truck is a
product call, not a bug fix. `users.timezone` already exists in the schema, so the correct fix is
available whenever you want it.

**Not affected** (checked, not assumed): DST (UTC dates are monotone — no skipped or duplicated
day); a Monday-queued submission draining Thursday (`drainQueue` has no period filter); a wrong
device clock (degrades to pre-decision-60 behaviour, not data loss).

**The question for you:** does "today" mean the UTC date, or the truck's local operating day? If the
latter, this wants a follow-up card moving all four sites to `America/Chicago` together.

---

## D-2 · When may the `/sync/*` door actually be configured?

**Status:** OPEN — sequencing decision. **The run recorded the constraint but cannot enforce it.**
**Found by:** G6 adversarial review of Card C (finding N2).

Card C landed a same-origin `/sync/*` reverse proxy. It is **inert in every current deploy** —
`HQ_SYNC_REST_URL` and `HQ_SYNC_REALTIME_URL` are unset, so every request answers 503.

**Setting those two environment variables today would give every logged-in crew member read *and*
write on the whole exposed schema** — the proxy forwards every method to PostgREST with a
`role: authenticated` JWT and `hq_grants: []`. Row-visibility RLS is obligation 1 of
`sync-rxdb-schema-and-replication`, which has **not** landed.

The constraint is now written into `proxy.go`'s env-var block, the parent card's obligation 6, and
the DONE card. But it is a comment, not a gate: nothing in the repo *prevents* the two variables
from being set.

**The question for you:** is a documented constraint sufficient, or should the proxy hard-refuse to
start when the upstreams are configured and an RLS-landed marker is absent? The run did not build
that gate because inventing the marker is a design decision.

---

## D-3 · Conflict-notice sign-off — the mockup is ready, and three things in it are yours to reject

**Status:** OPEN — **this is the sign-off Card D exists to make possible.**
`sync-rxdb-conflict-notice-ui` stays ATTENDED-BLOCKED until you answer.

Artifact: `.planning/phases/sync-rxdb-conflict-notice/mockup.html` + `UI-SPEC.md`, 11 plates,
22 screenshots (480px, light + dark), 10 state rows, all 20 `done_when:` rows passing, no waivers.
Two independent verifier passes under CLAUDE.md's restricted-input gate; the second says sign-off
can proceed.

**A mockup existing is not a sign-off.** The roadmap annotation says so explicitly so that no future
run can infer one.

Three decisions embedded in it, each rejectable:

1. **The counting rule — the most consequential.** The banner reports *what was overwritten in the
   window*; **it is not a to-do list**. Counts never decrement on Restore, Keep theirs, or a failed
   restore — only Dismiss or 30-day expiry removes a row. Chosen to preserve **Undo**, because a row
   removed from the sheet cannot be undone. If you want a decrementing banner, you trade away the
   only escape from a mis-tapped Restore.
2. **"Recoverable" vs the degradation ladder** (verifier residual nit). A removed-field row has **no
   Restore** — its recovery is *Copy value* — yet it is counted in the chip base as "1 answer". An
   operator reading "recoverable" as "has a Restore button" would expect `0 answers +1`. One clause
   in the rule closes it either way. **Easy to approve without noticing.**
3. **30-day retention is a placeholder** awaiting your number.

Also conditional, and owned elsewhere: attribution ("Dana M., 6:12 PM") depends on a
who-and-when schema decision inside `sync-rxdb-schema-and-replication`. If that isn't carried, those
lines degrade to "someone else".

**One question Card D parked rather than improvised:** past ~10 conflict groups the sheet needs a
cap or a date filter. Not designed. Judge it against one long dead-zone shift with an active manager.

---

## D-4 · Two pre-existing defects, disclosed but unfiled

**Status:** OPEN — needs a filing decision, not a fix decision.
**Found by:** Card B; confirmed genuinely pre-existing by G6 against `25fbc16` (finding C).

Both are real, both are outside Card B's footprint, and both are currently disclosed **only** in a
DONE card body and a merge-intent note — not in `BACKLOG.md`, which is what `/nc-roadmap-round`
consumes:

- **Orphaned fail notes.** `unsubmitChecklist` detaches fail notes to `submission_id = NULL`
  (`repository.go:1281`) and nothing re-attaches them; they carry no `answered_by`, so they leak.
  Migration `0071`'s unique index does **not** collide with them — Postgres treats NULLs as distinct.
- **Builder-row badge.** `renderSyncBanner` selects `[data-template-id]` document-wide
  (`sync.js:671`), so the "Queued" badge paints onto Builder tab rows too. Reproduced in a browser
  by G6; unchanged from base.

**The question for you:** file both into `BACKLOG.md` at triage, or fold them into whichever card
next touches those files? The run left them where the card put them rather than editing the backlog
on its own authority.
