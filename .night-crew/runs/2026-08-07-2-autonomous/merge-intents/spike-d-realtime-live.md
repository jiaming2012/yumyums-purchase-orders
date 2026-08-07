# Merge intent — card D · `spike-d-realtime-live` (run `20260807-2`)

Branch `card/d-spike-d-realtime-live`, cut from the run branch after card C merged.
Activity 2, spike D — the 4th of D-KR1's four spike verdicts. Closes **B-62**.

**What the card is.** B-42 option (i) put a single `column=op.value` clause into the
plugin's `postgres_changes` binding through a shim on `client.channel`
(`sync-rxdb/client.js`). `tests/sync-rxdb-client.spec.js` [SCOPE-04] proves the clause
is correct, is one clause, reaches the binding config under the right channel name, and
is absent on `responses` — **every one of those assertions is about the object handed to
the library.** Nothing in the tree shows the *substrate* honours it. This card drives
the same clause against a **live** self-hosted Realtime server and lets the answer be
whatever it is. 🛑 **A RED verdict — the filter is ignored — is the deliverable, not a
park.** B-62 names the failure mode precisely: the filter is a NARROWING, so if it is
ignored the system behaves exactly as it did before the card, and three files say it is
closed when it is not.

---

## Shared files touched

| File | Why |
|---|---|
| `Taskfile.yml` | Adds ONE new target, `spike:realtime`, beside the existing `spike:up` / `spike:migration` / `spike:roundtrip` (card C's, merged). 🛑 The `prod:backup` stanza, every `test:*` block, and card C's `spike:roundtrip` stanza are **untouched** — verify with `git diff Taskfile.yml`. Resolution against card C is additive: keep both targets. |

That is the **only** file outside the card's own footprint. Everything else is inside
`.night-crew/qa/spike-supabase/` (the script, its fixture SQL, its new `rtprobe/` helper)
and `.night-crew/runs/2026-08-07-2-autonomous/` (gate logs + this file).

**`sync-rxdb/client.js` is NOT modified.** The card permitted touching it "only if
surfacing the channel shim for the live proof requires it". It did not: the spike calls
the **already-exported** `realtimeFilterFor()` directly from Node and feeds its literal
output to the live subscriber. That is strictly better evidence than a new seam would
have been — the string on the wire is the string HQ's production code emits, not a
re-typed copy of it. 🛑 **Consequence, and the card's D-specific requirement:
`tests/sync-rxdb-client.spec.js`'s config-level assertions stay green — untouched, byte
for byte.** No production JS/HTML asset changes, so `node build-sw.js` must stay at
precache **31**.

**Files NOT touched, deliberately:** `sync-rxdb/**`, `sync-schema/**`, `backend/**`
(no Go in HQ's module — the helper lives in the spike-local `spike-supabase` module,
which exists precisely so a spike cannot alter HQ's dependency graph), `backend/go.mod`,
`backend/go.sum`, `package.json`, `playwright.config.js`, `night-crew.toml`, and every
file under `.night-crew/qa/spike-supabase/` that spike A, B or C owns (`env-up.sh`,
`spike-b-migration.sh`, `spike-c-roundtrip.sh`, `rtwatch/main.go`, `mintjwt/main.go`,
`rxdb/**`, `sql/spike-fixture.sql`, `sql/hq-*.sql`, `docker-compose.supabase.yml`,
`docker-compose.hq-*.yml`) — all consumed read-only so spike A's, B's and C's verdicts
keep reproducing byte-for-byte.

**`rtwatch/` is extended by ADDITION, not edit.** The card allowed extending it. It is
left alone and a sibling `rtprobe/` is added instead, because `rtwatch` is cited by
spike A's README proofs (R3 in particular) and its single-binding, exit-on-first-event
shape is *load-bearing for those proofs*. `rtprobe` is the multi-binding,
filter-carrying, observe-a-window variant this card needs. Same module, same pinned
`github.com/coder/websocket v1.8.14`, no new supply-chain surface.

## What must survive any merge

1. **`.night-crew/qa/spike-supabase/spike-d-realtime.sh` and its exit-code contract.**
   The contract is the deliverable, not the prose:
   `0` = GREEN (filter honoured), `1` = **RED — ran, and the substrate does not honour
   the filter** (a successful spike), `2` = **COULD NOT RUN** (infra/setup — NOT a
   verdict), `3` = a verdict was reached but the substrate could not be restored,
   `64` = usage. Merging a version that collapses `1` and `2` destroys the only thing
   the card is for.
2. **The positive-arrival leg, and its mandatory status.** An ignored filter and a dead
   subscription look identical from the negative leg alone. The in-scope row MUST be
   observed arriving on every filtered channel or the run is vacuous and the script
   reds. Do not "simplify" the assertion set down to the suppression leg.
3. **The unfiltered controls — both of them.** (a) `responses`, filter deliberately
   absent, where **both** rows must arrive: that is what turns "we chose not to filter
   it" from a comment into a measurement. (b) A second, unfiltered subscription on the
   **same table** as the filtered ones, so the identical out-of-scope row is *observed
   reaching Realtime* on one channel and *not* on another. Without (b), "the row never
   got published" and "the filter suppressed it" are indistinguishable.
4. **The filter strings come from `realtimeFilterFor()` at run time.** They are never
   typed into the script. If HQ's clause shape changes, this spike changes with it or
   fails loudly — it can never quietly prove a string production no longer emits.
   All four shapes HQ emits are driven: `gte` (checklists/list), `eq` (checklists/fill),
   `in.(…)` (templates/list), and `null` (responses, both modes).
5. **The `--no-filter` flag.** It is how the red-first capture is *reproduced on demand*
   rather than merely reported. Do not "tidy" it away as dead code.
6. **The teardown's substrate restore, and its self-verification.** B-148's residual is
   that spike B's harness recovery path was never re-rehearsed after its fix, so this
   card does not borrow that path: it snapshots spike A's `spike_notes` row set **and
   the `supabase_realtime` publication's exact table membership** before it creates
   anything, drops its own two tables at teardown, and **asserts both snapshots are
   byte-identical afterwards**. A failed restore turns a green run into exit `3`.
   The restore is exercised on the red path too, not only the green one.
7. **Isolation assertions as executable checks, not comments.** The spike resolves
   spike A's `db` container through the `spike-supabase` compose project and refuses to
   run if the resolved published port is `5432`, `5433` or `5434`. :5433 is the
   PRODUCTION *and* dev cluster — a probe there destroyed the prod database on
   2026-08-06 (B-141/B-143, ledger decision 155).
8. **The verdict itself**, recorded in the run's HANDOFF regardless of colour, and
   B-62's disposition set from it.

## What is safe to drop

- The two fixture tables `public.spike_d_checklists` / `public.spike_d_responses` — they
  are created and dropped every run by construction; nothing persists. `--keep`
  suspends the drop for manual inspection only.
- The captured gate logs under `.night-crew/runs/2026-08-07-2-autonomous/` may be pruned
  once the run is triaged and the verdict is in the roadmap.
- `rtprobe/` is a **spike artefact with a defined end of life**: the Activity 3 card
  that first sets `HQ_SYNC_REST_URL` (`sync-hard-cutover`, B-62's stated destination)
  either replaces it with a real client assertion or the verdict retires it. It is not
  a proposal for HQ's production shape.

## Conflicts expected with the run's other cards

- **`Taskfile.yml` is the one real conflict surface.** Card C appended `spike:roundtrip`
  to the same `spike:` block; this card appends `spike:realtime` after it. Resolution is
  additive — keep both targets, in either order.
- Card C and this card both consume spike A's substrate in **reconcile mode, never
  destroy**. The slate runs them serially, so there is no interleaving. This card writes
  to **no shared table at all** — it creates its own two and drops them — so even a
  concurrent run could not disturb `hq_sync_checklists` / `hq_grant_projection` and
  therefore cannot red `backend/internal/sync/jwtbridge_rls_test.go`.

---

## Red-first

Gate RF. The natural red for a spike is **the assertion set run with the mechanism
absent** — here, the same live subscription, the same two rows, the same substrate, with
the **filter clause not attached to the binding**.

That is exactly the pre-B-42 world B-62 warns is indistinguishable from an ignored
filter, which is why it is the right red: if the script cannot tell those two apart it
is not evidence, and the red proves it can.

Reproducible on demand at any later date with:

```
.night-crew/qa/spike-supabase/spike-d-realtime.sh --no-filter
```

**Captured red (mechanism absent):**

- command: `.night-crew/qa/spike-supabase/spike-d-realtime.sh --no-filter`
- log: `.night-crew/runs/2026-08-07-2-autonomous/card-d-rf-red.log`
- exit code: **1** (ran, mechanism disproven — distinct from `2` "could not run")
- failing leg: `7. SUPPRESSION` — with no filter attached, the out-of-scope row arrived
  on all three nominally-filtered channels.
- proof the red is *the mechanism* and not a broken harness: every other leg PASSES in
  the same log — substrate reconciled, fixture applied, all five subscriptions joined
  and acked, and the **positive-arrival leg green on every channel** (the in-scope row
  arrived everywhere). The harness is demonstrably alive on both sides of the gap.

**Green run:** the same command without `--no-filter` —
`.night-crew/runs/2026-08-07-2-autonomous/card-d-spike-verdict.log`.

Both logs are committed.

Night-Crew-Run: 20260807-2
