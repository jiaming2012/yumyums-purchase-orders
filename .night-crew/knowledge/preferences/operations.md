# Preferences — operations

> The production posture: what serves the business, what may touch it, and how it survives mistakes.
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

## P-1 · Tests never share a cluster with production

- **Preference:** Test suites run against their own Postgres container, never the cluster serving production. A test mistake must be structurally unable to touch operating data — no test file holds admin credentials to production.
- **Why (operator):** Operator chose cluster separation at morning triage 2026-08-05 (D-2b, run 20260806), after a G6 review probe executed DROP DATABASE against production via the shared cluster (B-141). No further reason stated; the recommended option was chosen.
- **Weight:** moderate
- **Operator:** jac475@cornell.edu
- **Recorded:** 2026-08-05
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.
- **Adopted:** 2026-08-06 — confirmed at the terminal by the operator.

## Pending — proposed, not adopted

> Candidates offered back from your own answers, recorded with your consent.
> They are **not preferences yet**: nothing cites them, nothing validates them, and no
> command promotes them. To adopt one, renumber it to the next free `P-n`, move it up
> above this section, and delete what you don't want. To drop one, delete it.

## C-2 · Production data is always restorable

- **Preference:** Production carries both a nightly pg_dump written outside the Docker volume and WAL-based point-in-time recovery. An image rollback without a data restore path is not a recovery posture.
- **Why (operator):** Operator chose nightly dump plus PITR at morning triage 2026-08-05 (D-2a, run 20260806), after B-143 — no backup of any kind — made the production drop unrecoverable. No further reason stated.
- **Weight:** moderate
- **Operator:** jac475@cornell.edu
- **Recorded:** 2026-08-05
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.

## C-3 · Overnight runs git-operate only from a dedicated run worktree

- **Preference:** An overnight run performs every git operation — merges, commits, branch switches — from a dedicated run worktree (e.g. hq-worktrees/run-<runid>), never the repo's main checkout. The main checkout belongs to attended sessions.
- **Why (operator):** Chosen at morning triage 2026-08-06 so attended sessions can't collide with a run's merges again: during run 20260807 a concurrent attended session moved the main checkout mid-card and the run's first merge landed on dev (recovered in full, conflict log §1). The recommended candidate-rule option was chosen; no further reason stated.
- **Weight:** moderate
- **Operator:** jac475@cornell.edu
- **Recorded:** 2026-08-06
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.
