# Extraction — inventory-setup-races

Outcome: confirmed for B-478; corrected for B-459 (one agent-reached correction, signed)

Approach used: a throwaway worktree of `dev` on a spike-owned e2e stack, a three-spec
Playwright file through the shipped `inventory.html` with only network timing altered via
`page.route` — groups delayed 600 ms (B-478 premise), a control typed after network idle,
and the Setup tab's first `GET /items` held until after a nickname add then released (the
filed B-459 timing). Tool-recorded runs: exit 1 (control typed too early), exit 1 and
exit 1 (the B-459 premise corrected by observation, both named), exit 0. Candidate input
for the card, not an adoption.

Confirmed: (a) B-478 exactly as filed — the late groups response empties a name typed
right after opening Setup and the create click sends no POST; a name typed after the
page settles survives. (b) B-459's SYMPTOM — the typed nickname missing from the chips —
but NOT its diagnosed mechanism: across three runs under the filed timing the view never
disagreed with the server; in two of three the add never reached the server (no POST
persisted), in one it reached both.

Learned: (1) the Inventory Setup flake is a dropped add during the first items fetch,
not a late overwrite of a kept one — the fix target is the add path, not only the
writers' ordering; (2) the outcome is nondeterministic run to run on an idle box, so the
card's proof is a repeated scenario (`[IS-04]` 5× in a row) plus the 10×/5× measurement;
(3) the add bar's re-render on groups landing is the deterministic half (B-478) and is
fixed by rendering once and refreshing only the select's options; (4) a spike control
must wait for network idle on a loaded box.

Plan change: the card's B-459 half is re-scoped from "sequence the writers" to "diagnose
and fix the dropped add, sequence the writers as hardening, retire the red only by
measurement"; `[IS-04]` added. The B-478 half and `[IS-01]`/`[IS-03]` unchanged.
