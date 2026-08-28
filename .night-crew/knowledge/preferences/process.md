# Preferences — process

> How work is planned, reviewed, and shipped.
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

<!-- Ported from the flat `preferences.md` at morning triage 2026-07-25 (ledger T-22).
     The flat file is no longer read by any tooling; these entries are the live ones. -->

## P-1 · Run on a clean DB with fixtures

- **Preference:** every suite entry point, and any gate that makes a suite-green claim, runs against a freshly reset database seeded from fixtures. A suite entry point that does not reset is a defect in the entry point, not a quirk to work around in the tests; a test that needs accumulated state seeds it as a fixture rather than inheriting it from history.
- **Why (operator):** *"It's almost always my preference to run on a clean DB with fixtures in order to preserve determinism (unless there is a specific reason not to do so)."* Stated during `overnight-20260720c` triage after `tests/sync.spec.js:525` (`FLD-LIVE-02`) was promoted to a flake-fix card and turned out to be neither flaky nor order-dependent: the `ops` table is an append-only sync journal that never shrinks, `task test` drops and recreates `hq_test` but a bare `npx playwright test` does not, so the journal accumulates across runs and the test passes at 98 ops and fails at 614+. The failure was a function of how many times the suite had run since the last clean.
- **Weight:** strong — a default that requires a stated reason to depart from, not an absolute rule.
- **Evidence:** decided the `cycle-gate` card's DB precondition; resolved DECISIONS-NEEDED §0b of that run.
- **Recorded:** 2026-07-21

## P-2 · Separate schemas; never cross-contaminate

- **Preference:** use separate schemas per environment, and extend the same separation to any shared mutable resource across environments, runs, suites, or sessions — databases and schemas first, then append-only tables that survive resets, client-side persistence (IndexedDB, localStorage, service-worker caches, persisted Lamport clocks), ports and containers, and the working tree itself. Investigate any surface where cross-contamination is possible rather than waiting for it to show up as a flake.
- **Why (operator):** *"It's almost always my preference to use separate schemas so as to not cross contaminate (e.g. separate schemas for dev, qa, etc); this should be applied everywhere — investigate any surfaces where cross contamination is possible."* The working-tree surface was confirmed live on 2026-07-21: five Claude Code processes held `cwd=/home/jcole/projects/hq`, one alive five days, and one wrote to the tree believing it was on `dev` when `overnight-20260720c` was actually checked out. One checkout, one git index, many writers — and worktrees additionally share `refs/stash`, which is why `git stash` is prohibited in a worktree here.
- **Weight:** strong — a default that requires a stated reason to depart from, not an absolute rule.
- **Evidence:** full audit at `reference/cross-contamination-surfaces-20260721.md`; the `:8199` port latch recurred across at least three runs.
- **Recorded:** 2026-07-21

## P-3 · Unreproducible means resolved, until it recurs

- **Preference:** any flake, intermittent failure, or defect report that survives a genuine reproduction attempt without reproducing is closed — and reopened without embarrassment if it recurs. Closing it records what was attempted, under what conditions, and what failure rate the attempt bounds. Distinguish *"not flaky"* (no mechanism, no reproduction — close it) from *"rare, mechanism understood, bounded at N%"* (a real mechanism firing rarely), and never launder the second into the first.
- **Why (operator):** *"If an issue cannot be reproduced, then mark it as resolved until / unless it appears again."* Two reports had consumed real capacity on the opposite basis: `sync.spec.js:1198` was promoted into a slate as "proven flaky — red 1-of-2 legs" and then produced ~20 consecutive greens across two independent parties, and `workflows.spec.js:2223` (`RUN-10`) was reported as a gate blocker and did not reproduce at all. Carrying an unreproducible claim forward as established fact is the process failure; reopening on recurrence is not.
- **Weight:** strong — a default that requires a stated reason to depart from, not an absolute rule.
- **Recorded:** 2026-07-21

## P-4 · A reproduction attempt must sample the RIGHT condition, or it proves nothing

- **Preference:** before closing anything under P-3, state the condition you sampled — load, concurrency, data volume, journal depth — and whether it is the condition the original report described. *"Could not reproduce on a quiet box"* is a finding; *"could not reproduce"* is not. Absence of reproduction in N legs bounds a rate conditional on the sampled condition only.
- **Why (operator):** added the same day as P-3 because P-3 failed immediately and instructively. `sync.spec.js:1198` was declared "decisively refuted" on ~20 consecutive greens from two independent parties; re-run under a concurrent Playwright suite it produced 4 reds in 25 legs (16%, 20% under contention). Both earlier parties had sampled the quiet condition, and ~20 greens off-condition is entirely consistent with a 20% on-condition rate (p ≈ 1.2%). The failed reasoning is worth naming because it looks rigorous: a p-value was computed against a 50% unconditional rate nobody had claimed, using samples drawn from a condition where the bug does not fire. The arithmetic was correct and the conclusion was wrong.
- **Weight:** strong — this is the corollary that makes P-3 safe to apply; applying P-3 without it is how a live bug gets closed.
- **Evidence:** 5/5 green at low load bounds nothing about high load — p(0 red in 5 | 20%) ≈ 0.33.
- **Recorded:** 2026-07-21

## P-5 · Never cite an identifier without the thing it identifies

- **Preference:** A decision number, backlog handle, ledger section, card ID, key-result name, gate label or commit SHA carries no meaning on its own. Attach what it identifies, in a few words, on first use in any document or message — "decision 126 (reads move to RxDB, writes stay on REST)", not "decision 126". Keep the identifier; never let it travel alone. This binds the written record too: the ledger, handoffs and commit bodies are where bare numbers accumulate and are hardest to decode later, because nobody reads a ledger linearly. A short gloss beats a perfect one.
- **Why (operator):** "can i write a rule to stop writing, e.g. decision XYZ 126 without a reference, as i dont remember what the number represents." — operator, 2026-08-03. Second occurrence of the same class: the July plain-language feedback carved out an exception for the written record, and that exception is what let bare numbers accumulate there.
- **Weight:** strong
- **Operator:** jac475@cornell.edu
- **Recorded:** 2026-08-03
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.
- **Adopted:** 2026-08-06 — confirmed at the terminal by the operator.

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

## C-1 · Amend a stale sequencing decision rather than honour it into a worse outcome

- **Preference:** When new evidence materially changes the size or nature of a correction owed to an external counterparty, amend the earlier sequencing decision and send one complete correction — do not honour a stale plan because it was already decided. Prefer one honest message over a drip of partial apologies. Fold any older owed-but-undrafted notice into the combined one rather than sending it separately, and never send a notice still carrying an error the audit itself found.
- **Why (operator):** Chose "one combined notice" at morning triage 2026-08-02, amending ledger decision 106, after the audit grew from one drifted row to 111 rows audited and 45 wrong — 22 of which were never true at all. (Selected from offered options; no additional reason stated.)
- **Weight:** moderate
- **Evidence:** run 20260803, card P6 period-summary-contract-notice. Decision 106 had ruled two notices, June drift first and alone. P6's audit found A1's own notice carries an error (:31/A10, attributing a timezone claim to /menu-cogs, which has no AT TIME ZONE), and that A1's notice was never drafted — confirmed by searching all of git history: exactly one notice file has ever existed.
- **Operator:** jac475@cornell.edu
- **Recorded:** 2026-08-02
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.

## C-2 · E2E assertions on user-visible text assert the text, not the container

- **Preference:** A spec guarding user-visible text asserts the expected content (`toContainText` / `toHaveValue`, with the expectation sourced from the API or seed data), never just element visibility. A container with decoration — icons, disclosure triangles, borders — passes `toBeVisible()` while its content is blank, so a visibility assertion on a text element guards nothing.
- **Why (operator):** requested after the 2026-08-28 walkthrough — the FAQ blank-question bug lived undetected behind a green `expect(.faq-q).toBeVisible()` that the disclosure triangle satisfied; wants documentation of the walkthrough's UI fixes to feed future review tooling so this class doesn't recur.
- **Weight:** strong
- **Evidence:** FAQ label-contract fix, commit 2e078d3 (2026-08-28) — the strengthened text assertion went red instantly on the unfixed code; docs/ui-design-rules.md UI-R5.
- **Recorded:** 2026-08-28
- **Offered at:** an attended session, 2026-08-28
- **Consent:** recorded on the operator's explicit yes to this item.
