# Extraction — test-integrity-fix

Outcome: confirmed

Approach used: a throwaway worktree off `dev` in which the shipped `failClosed`
was inverted (one line) and an `http.Post` appended to `campaigns.go`, with the
three gates run before and after against spike-owned `:5434` databases and the
local substrate; and a node run against the vendored RxDB 17.4.0 opening the
shipped codes schema three ways. Two spikes, both exit 0 from a clean state
(spike 01 on its second run after a script-side grep bug). Candidate input for
the card, not an adoption (NFR-6).

Confirmed: all three false gates reproduce by execution exactly as triage filed
them — `[SP-03b]` stays green and `campaigns-run.sh` stays `GREEN` against an
inverted predicate (the harness imports nothing from `submit-flow.js`), and
`TestNothingInThisPackageSends` passes with outbound HTTP in `campaigns.go`. The
decision-191 rider "add `campaign_id` to `required`" is a replica-schema
migration: the same-version widen is refused with `DB6` on a store that holds
rows, and v1 + a migration strategy carries the rows across; a v1 row without
`campaign_id` is refused with `VD2`, which is the invariant `[SP-03b]` claimed
to test, now a schema fact.

Learned: `failClosed` / `policyFor` must be EXPORTED from `submit-flow.js` before
the harness can import them; both `codes` and `offers` share the schema, so the
strategy goes on both in `marketingCollectionSpec()`; the Dexie migration path
remains untested (no `fake-indexeddb` vendored — B-441 stays open); a worktree
run of the harness needs the QA `rxdb/node_modules` symlink or `env-up.sh` fails
on module resolution.

Plan change: none to scope — the card's red-first recipes are the spike's three
mutations, its schema change is v1 + strategies for both collections, and its
`[SP-03b]` replacement asserts `VD2`.
