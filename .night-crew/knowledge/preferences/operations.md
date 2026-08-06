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

## Pending — proposed, not adopted

> Candidates offered back from your own answers, recorded with your consent.
> They are **not preferences yet**: nothing cites them, nothing validates them, and no
> command promotes them. To adopt one, renumber it to the next free `P-n`, move it up
> above this section, and delete what you don't want. To drop one, delete it.

## C-1 · Tests never share a cluster with production

- **Preference:** Test suites run against their own Postgres container, never the cluster serving production. A test mistake must be structurally unable to touch operating data — no test file holds admin credentials to production.
- **Why (operator):** Operator chose cluster separation at morning triage 2026-08-05 (D-2b, run 20260806), after a G6 review probe executed DROP DATABASE against production via the shared cluster (B-141). No further reason stated; the recommended option was chosen.
- **Weight:** moderate
- **Operator:** jac475@cornell.edu
- **Recorded:** 2026-08-05
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.

## C-2 · Production data is always restorable

- **Preference:** Production carries both a nightly pg_dump written outside the Docker volume and WAL-based point-in-time recovery. An image rollback without a data restore path is not a recovery posture.
- **Why (operator):** Operator chose nightly dump plus PITR at morning triage 2026-08-05 (D-2a, run 20260806), after B-143 — no backup of any kind — made the production drop unrecoverable. No further reason stated.
- **Weight:** moderate
- **Operator:** jac475@cornell.edu
- **Recorded:** 2026-08-05
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.
