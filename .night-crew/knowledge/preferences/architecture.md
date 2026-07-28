# Preferences — architecture

> Paradigms, service boundaries, storage and messaging choices.
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

## C-1 · The app's timezone is America/New_York

- **Preference:** Every date and time boundary the app computes — submission "today", the purchasing week, the COGS/payroll period, recipe cost weeks, scheduled jobs — resolves in America/New_York. Never UTC, never a hardcoded America/Chicago, never the device's local zone. Where a stored default exists it mirrors users.DefaultTimezone rather than restating a literal.
- **Why (operator):** The apps time zone should be NY time.
- **Weight:** strong
- **Evidence:** ledger T-26 decision 83; card app-timezone-unify-new-york. Found because the codebase was running two conflicting regimes — New York in the user-facing defaults, America/Chicago in the COGS completeness gate, CurrentWeekStart, the recipe cost week, and two migration column defaults.
- **Recorded:** 2026-07-28
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.
