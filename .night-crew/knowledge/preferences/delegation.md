# Preferences — delegation

> What agents may decide without asking, and where each grant stops.
>
> Weighted opinions, not binding rules: a session may deviate, but must say why.
> Add entries as `P-1`, `P-2`, … in the shape of the commented example below —
> `night-crew preferences validate` checks that shape, never your judgment.
>
> 🛑 Entries here **widen autonomy**, which is what makes this category different from
> the others. Every entry carries a **Bound** naming where the grant stops, and that
> sentence is the operator's own words — never drafted for them.
>
> Category file created 2026-08-03 at morning triage, on the operator's explicit
> permission, to hold the rider recorded in ledger §T-34.

<!--
## P-1 · A short title naming the leaning

- **Preference:** what agents may decide, stated so a session can act on it.
- **Bound:** where the grant stops — the operator's own sentence.
- **Why:** the operator's reason, in their words.
- **Weight:** moderate | strong
- **Since:** YYYY-MM-DD
-->

## P-1 · Agents decide implementation details

- **Preference:** When a ritual or skill routes a mechanism-level question to the operator — where the gate ladder lives, what the launch prompt carries, which per-change discipline the repo adopts, whether a validator's divergence justifies a mass rewrite — decide it at role level and state the decision with its reasoning, rather than asking the operator to pick among options. Genuine product forks still escalate.
- **Why (operator):** "agents decide implementation details" — operator, morning triage 2026-08-03, in answer to a question offering B-26 / B-80 / B-105 / the BACKLOG.md validator divergence for selection.
- **Weight:** moderate
- **Bound:** Stops at genuine product forks — questions about what the product should do. Everything else, including deploys and releases, agents decide and state.
- **Operator:** jac475@cornell.edu
- **Recorded:** 2026-08-03
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.
- **Adopted:** 2026-08-03 — confirmed at the terminal by the operator.

## Pending — proposed, not adopted

> Candidates offered back from your own answers, recorded with your consent.
> They are **not preferences yet**: nothing cites them and nothing validates them until
> you adopt one. To adopt: run `night-crew preferences adopt <category>/<C-n>`, which
> shows you the exact entry and asks — your yes is read from a terminal, so no pipe, flag,
> redirect or environment variable can answer for you, while a caller that deliberately
> allocates a terminal of its own satisfies that check: it makes adopting a deliberate act,
> not a defence against something setting out to defeat it. Or do it by hand as before:
> renumber the candidate to the next free `P-n` and move it above this section.
> To drop one, delete it.

## C-1 · Preference-covered questions proceed under citation; park only what no preference answers

- **Preference:** When a card or session hits a question the operator has an adopted preference for, it decides under that preference, cites it, and surfaces the decision for end-of-run review — reviewable and revertible beats parked. Parking is reserved for questions no adopted preference answers, and nothing on the always-escalate ceiling (safety/escalation/constitutional) is ever decided this way.
- **Why (operator):** Stated at the 2026-08-28 retro, refining T-43(b): "if Claude can reference a preference of mine to answer it itself, then I can just review it at the end / revert the change if necessary instead of parking."
- **Weight:** strong
- **Bound:** only decisions that are cheaply revertible at review — each cited and surfaced in the run's end-of-run review artifact; anything irreversible or outward-facing (sends, deploys, deletions), and anything on the always-escalate ceiling, stays the operator's regardless of what a preference says
- **Operator:** jamal@Jamals-MacBook-Pro.local
- **Recorded:** 2026-08-28
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.

## C-2 · Race-proven body ships; disclosed envelope + stamps are in-remit

- **Preference:** When a card's deployed function preserves the signed text's enforcement rule, atomicity, and result vocabulary, the night may ship it without parking even when it adds standard safe-pattern execution wiring (owner-privilege execution, pinned search path, explicit grants) and/or bookkeeping stamps on rows the function already updates — provided every addition is disclosed in the shipped artifact itself and named for ratification in HANDOFF.
- **Why (operator):** Operator, morning triage 2026-09-04, on Card 2's redeem() delta (updated_at stamp + SECURITY DEFINER wiring): chose 'Stands as shipped — the version that was actually race-proven is the version that ships.' Bound sentence adopted by the operator from the offered wording, envelope-plus-stamps scope.
- **Weight:** moderate
- **Bound:** Only the execution envelope and bookkeeping stamps on rows the function already updates; the enforcement rule, the result vocabulary, and which rows get touched must match what I signed.
- **Operator:** jamal@Jamals-MacBook-Pro.local
- **Recorded:** 2026-09-04
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.
