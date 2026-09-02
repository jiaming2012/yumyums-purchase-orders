# Preferences — ux

> How the product should feel to the people using it.
>
> Weighted opinions, not binding rules: a session may deviate, but must say why.
> Add entries as `P-1`, `P-2`, … in the shape of the commented example below —
> `night-crew preferences validate` checks that shape, never your judgment.

<!--
## P-1 · A short title naming the leaning

- **Preference:** what to do, stated so a session can act on it.
- **Why (operator):** the reasoning, in your own words — this is what gets cited back to you.
- **Weight:** strong — and then whatever qualifier you want (start with strong, moderate, or weak).
- **Evidence:** optional link to a research note or a past decision.
- **Recorded:** 2026-01-01
-->

## Pending — proposed, not adopted

> Candidates offered back from your own answers, recorded with your consent.
> They are **not preferences yet**: nothing cites them, nothing validates them, and no
> command promotes them. To adopt one, renumber it to the next free `P-n`, move it up
> above this section, and delete what you don't want. To drop one, delete it.

## C-1 · A removed question keeps its label, struck through and read-only

- **Preference:** When a crew member answered a template field that no longer exists — deleted while they were offline, archived, or renamed away — the UI shows the question's original label **struck through, with the row read-only**, rather than a raw field id, an invented label, or a hidden row. The label is read from the submission's own frozen `template_snapshot`, which carries Sections → Fields → Label (`backend/internal/workflow/model.go:44-57`, marshalled whole at `repository.go:695`).
- **Why (operator):** "show the deleted question crossed out and read only so that the user isnt confused"
- **Weight:** strong
- **Evidence:** morning triage 2026-07-29, open decision (i) of the `sync-rxdb-conflict-notice` mockup — the plates had drawn the raw field id `fld_prep_sink_temp` in muted monospace on the grounds that "the template no longer holds a label for it", which is true of the template and false of the submission. Recorded as ledger T-28 decision 95.
- **Recorded:** 2026-07-29
- **Offered at:** morning triage 2026-07-29
- **Consent:** recorded on the operator's explicit yes to this item.

## C-2 · Blank is never a valid render for user-facing text

- **Preference:** When data for a user-visible slot is missing or undefined, the UI renders a loud placeholder or a visible error — never a silently empty element. Escape/format helpers must not be handed `undefined` to smooth into `''`; a row whose primary text region is empty is a defect by definition, not a styling state.
- **Why (operator):** requested after the 2026-08-28 walkthrough — the FAQ section rendered three designed-looking blank bars because a wrong field name coerced to empty string with no error anywhere; wants blankness treated as failure in future UI decisions.
- **Weight:** strong
- **Evidence:** FAQ label-contract fix, commit 2e078d3 (2026-08-28); docs/ui-design-rules.md UI-R3.
- **Recorded:** 2026-08-28
- **Offered at:** an attended session, 2026-08-28
- **Consent:** recorded on the operator's explicit yes to this item.

## C-3 · Failures are loud, retryable, and overridable

- **Preference:** Every async failure a crew member can hit (media load, data fetch, upload) has three parts: a visible error state naming what happened in crew language, a Retry affordance, and an authority override path (e.g. manager marks a training step watched) that unblocks the flow when retry cannot. Nobody gets silently stranded mid-task on a phone in a truck.
- **Why (operator):** requested after the 2026-08-28 walkthrough — onboarding video failures were silent (the player just never played) until the loud-error + retry + manager-override work; wants this three-part shape as the default for future async failure UI.
- **Weight:** strong
- **Evidence:** commit 7d9efcc video recovery / loud player errors / manager override (2026-08-28 cycle); docs/ui-design-rules.md UI-R6.
- **Recorded:** 2026-08-28
- **Offered at:** an attended session, 2026-08-28
- **Consent:** recorded on the operator's explicit yes to this item.
