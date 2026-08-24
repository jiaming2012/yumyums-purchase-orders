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
