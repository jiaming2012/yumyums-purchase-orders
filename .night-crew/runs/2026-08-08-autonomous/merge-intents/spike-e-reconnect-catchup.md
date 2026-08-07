# Merge intent — card E · `spike-e-reconnect-catchup` (run `20260808`)

Branch `card/spike-e-reconnect-catchup`, cut from `overnight-20260808`. The only card of
the slate. Activity 2, spike E — added post-triage 2026-08-07 on the operator's ask
(**B-161**, "nothing has ever severed a replicating client").

**What the card is.** Spike C proved the round trip (one write, real path, relay,
substrate, a RUNNING RxDB client) and spike D proved the live Realtime filter. Neither
ever *severed* a client. The disconnect → dark-window-writes → reconnect → catch-up cycle
is completely unmeasured, and the build cards in Activity 3 assume it works. This card
severs a replicating RxDB client, writes N changes through HQ's REAL write path while it
is dark — **at least one INSERT and at least one UPDATE to a row the client already
holds** — reconnects it, and measures whether checkpoint pull recovers **everything**.

🛑 **A RED verdict is the deliverable, not a park and not a failure.** A red means the
build cards need an explicit resync step, and finding that out costs one night now versus
a crew member's phone sleeping through a write in production. A GREEN that never
exercised the UPDATE case is **vacuous** — the assertion set must include it.

---

## Shared files touched

| File | Why |
|---|---|
| `Taskfile.yml` | Adds exactly TWO new targets, `spike:reconnect` and `spike:reconnect:red`, appended to the existing `spike:` block beside `spike:up` / `spike:migration` / `spike:roundtrip` / `spike:realtime` (cards A–D, all merged). 🛑 The `prod:backup` stanza and every `test:*` block are **untouched** — verify with `git diff Taskfile.yml`. Resolution against any other card is additive: keep every `spike:*` target. |
| `.night-crew/knowledge/roadmap.md` | The card's own entry (Activity 2) flips from `PLANNED` to `DONE` with the verdict colour and a one-line note, in the same change set as the final work. No other card's entry is touched. |

That is the **entire** list of files outside this card's own additions.

**`backend/**` — nothing here.** The relay this card needs already exists
(`backend/internal/sync/spikec_relay.go` + `backend/cmd/spikec-relay`, both spike-marked)
and is launched exactly as `spike-c-roundtrip.sh` launches it. No Go source in HQ's module
is modified, no `go.mod` / `go.sum` change, and **`RunSpikeCRelay` stays unreferenced from
`cmd/server`** (`grep -rn RunSpikeCRelay backend/cmd/server` → no matches). Were a relay
change to prove necessary mid-run this note would be amended to declare it and that
invariant would still hold.

**No production frontend asset changes**, so `node build-sw.js` must stay at precache
**31** and the three version numbers stay at parity (no bump — this card ships no
user-visible change).

## What this card ADDS (its own footprint)

- `.night-crew/qa/spike-supabase/spike-e-reconnect.sh` — the spike, and the verdict.
- `.night-crew/qa/spike-supabase/rxdb/spike-e-reconnect.js` — the RxDB client leg
  (subscribe → observe → sever → dark → reconnect → assert).
- `.night-crew/runs/2026-08-08-autonomous/**` — this note and the captured gate + spike logs.

## Spikes A–D are consumed READ-ONLY

`env-up.sh`, `spike-b-migration.sh`, `spike-c-roundtrip.sh`, `spike-d-realtime.sh`,
`rtwatch/`, `rtprobe/`, `mintjwt/`, `rxdb/**` (every existing file), `sql/**`,
`docker-compose.supabase.yml`, `docker-compose.hq-real.yml`, `docker-compose.hq-source.yml`
— **not one of them is edited**, so all four GREEN verdicts keep reproducing byte-for-byte.
Where this card needs a variant it adds a **sibling** (`spike-e-*`), the same
addition-not-edit rule spike D applied to `rtwatch/` → `rtprobe/`.

Spike A's substrate is consumed in **RECONCILE MODE**. `--fresh` / `--fresh-substrate` are
**never passed, not once, not even to test the refusal** (B-159 names this exact footgun).
This card's script therefore exposes **no** `--fresh-substrate` flag at all — the flag spike
C carries is deliberately absent here so it cannot be typed by accident.

Spike C's scratch-Postgres discipline is inherited verbatim: own compose project
(`spike-e-hq`), Docker-assigned ephemeral host port, **runtime refusal** if the assigned
port resolves to 5432 / 5433 / 5434, `env … binary &` rather than `( … ) &` so `$!` is the
server's pid and not a subshell's, and a pre-flight refusal to attach to a server this
script did not start.

## What must survive any merge

1. **`spike-e-reconnect.sh`'s exit-code contract.** `0` = GREEN (full recovery observed),
   `1` = **RED — ran, and catch-up demonstrably misses dark-window changes** (a successful
   spike), `2` = **COULD NOT RUN** (infra/setup — **NOT a verdict**), `3` = a verdict was
   reached but spike A's shared substrate could not be restored, `64` = usage. Merging a
   version that collapses `1` and `2` destroys the only thing the card is for.
2. **The UPDATE case, and its mandatory status.** The dark window must contain at least one
   INSERT (a substrate row id the client has never held) **and** at least one UPDATE to a
   row the client already holds. An INSERT can be recovered by a full re-read; only the
   UPDATE exercises the checkpoint path against a document already in the local store. A
   green that skipped it is vacuous and the script must refuse to report one.
3. **Positive-arrival before the severing** (spike D's rule). A dead subscription and a
   recovered one must be distinguishable on **both** sides of the gap: the client is
   observed replicating a live row before the sever, and observed holding nothing new
   during the dark window, before any recovery claim is made.
4. **The dark-window silence assertion.** The collection is re-read at the end of the dark
   window and must be UNCHANGED. Without it, "the client was severed" is a claim rather
   than a measurement, and a client that never actually went dark would make the recovery
   trivially green.
5. **The `--no-pull` flag.** It is how the red-first capture is *reproduced on demand*
   rather than merely reported (C's `--no-relay` and D's `--no-filter` precedent). Do not
   "tidy" it away as dead code.
6. **The teardown's substrate restore, and its self-verification, on BOTH paths.** B-148's
   standard: snapshot the exact id set of `hq_sync_checklists` and the exact pair set of
   `hq_grant_projection` BEFORE anything is written, remove what was added, and **assert
   the sets are byte-identical again** — exercised on the red run too, not only the green
   one. A failed restore turns any verdict into exit `3`, because rows left behind red
   `backend/internal/sync`'s `TestJWTBridgeRLS`, whose `service_role` control asserts an
   EXACT full-table row set.
7. **Isolation as executable checks, not comments.** Ephemeral scratch Postgres, refusal on
   5432/5433/5434. :5433 is the PRODUCTION *and* dev cluster — a probe there destroyed the
   prod database on 2026-08-06 (B-141/B-143, ledger decision 155).
8. **The `submitted_at` / `answered_at` watermark finding**, recorded whatever the colour.
   The card names it a FINDING, not a blocker: measure what the real write path does.
9. **The verdict itself**, in the roadmap card and the run's HANDOFF, whatever its colour.

## What is safe to drop

- The scratch `spike-e-hq` Postgres and everything in it — created and destroyed every run
  by construction. `--keep` suspends the destroy for manual inspection only.
- The captured gate logs under `.night-crew/runs/2026-08-08-autonomous/` may be pruned once
  the run is triaged and the verdict is in the roadmap.
- `rxdb/spike-e-reconnect.js` is a **spike artefact with a defined end of life**: the
  Activity 3 card `skeleton-one-row-end-to-end` either adopts a real resync step (if this
  reds) or the verdict retires the question. It is not a proposal for HQ's production shape.

## Conflicts expected with the run's other cards

**Nothing here** — this is a one-card slate, run serially. The only cross-card surface that
would ever matter is `Taskfile.yml`'s `spike:` block, and the resolution there is additive.

---

## Red-first

Gate RF. The natural red for this card is **realtime-only recovery**: the identical
sequence — same substrate, same relay, same real write path, same client, same sever, same
dark-window INSERT + UPDATE — with the **checkpoint pull leg disabled on reconnect**. The
client comes back on a live Realtime subscription only.

That MUST miss the dark-window rows. Realtime replays nothing that happened before the
subscription existed, so a realtime-only reconnect can only ever see the future. If the
script cannot tell that apart from a working catch-up, the green run's assertion set is
vacuous and the card has measured nothing.

Reproducible on demand at any later date with:

```
.night-crew/qa/spike-supabase/spike-e-reconnect.sh --no-pull
```

or `task spike:reconnect:red`.

**Captured red (mechanism absent):**

- command: `.night-crew/qa/spike-supabase/spike-e-reconnect.sh --no-pull`
- log: `.night-crew/runs/2026-08-08-autonomous/card-e-rf-red.log`
- exit code: **1** (ran, mechanism disproven — distinct from `2` "could not run")
- failing leg: step 8, `CATCH-UP`. With the checkpoint pull absent, **all three**
  dark-window changes were MISSED — `insertA`, `updateB`, `insertC`. The substrate held 3
  rows; the reconnected client held 1, still carrying the **pre-sever** body.
- proof the red is *the mechanism* and not a broken harness — every other leg passes in the
  same log:
  - substrate reconciled (spike A GREEN, reconcile mode, no `--fresh`);
  - HQ's own migrator applied the real schema to the scratch `spike-e-hq` Postgres
    (Docker-assigned ephemeral port, refusal armed);
  - a real `POST /api/v1/auth/login` returned HTTP 200 and a real `hq_session`;
  - `/saveResponse` returned **HTTP 204** for every one of the four writes;
  - the **positive-arrival** leg was green: `B1` was OBSERVED reaching the live client as
    `spikec-b91a6243-…` before the sever, so the client is provably replicating;
  - **dark-window silence VERIFIED**: substrate 3 rows · dark client 1 doc, byte-identical
    to the pre-sever fingerprint — the gap is measured, not assumed;
  - the UPDATE landed **in place** (same substrate primary key, new body) and HQ's own
    Postgres shows field B with exactly **1** draft row;
  - 🛑 the **liveness control** ARRIVED: a post-reconnect write reached the realtime-only
    client within the bound (applier accepted `spikec-ffcc5bf9-…`). The reconnected client
    is therefore provably alive, which is what makes the miss attributable to the **absent
    checkpoint pull** and not to a dead socket. `checkpoints_handed_to_the_pull_handler`
    reads `[]` — no pull was issued, by construction.
- teardown on the red path **VERIFIED** `hq_sync_checklists` and `hq_grant_projection`
  byte-identical to the pre-run baseline — the B-148 recovery path was rehearsed on the red
  run, not only the green one.

**Green run:** the same command without `--no-pull` —
`.night-crew/runs/2026-08-08-autonomous/card-e-spike-verdict.log`, exit **0**. Same
substrate, same relay, same real write path, same sever, same three dark-window changes,
same assertion function. The only difference between the two runs is whether the checkpoint
pull leg is armed on reconnect: `MISSED / MISSED / MISSED` becomes
`RECOVERED / RECOVERED / RECOVERED`.

Both logs are committed.

## The finding

**GREEN — a severed RxDB client recovers EVERYTHING on reconnect via checkpoint pull,
including the in-place UPDATE.** Exit **0**.

| dark-window change | red (`--no-pull`) | green (pull armed) |
|---|---|---|
| INSERT, field A — a substrate row the client never held | MISSED | **RECOVERED** |
| **UPDATE, field B — the row the client ALREADY HELD, re-written in place** | MISSED | **RECOVERED** |
| INSERT, field C — a second new substrate row | MISSED | **RECOVERED** |
| liveness control (post-reconnect write) | ARRIVED | ARRIVED |

All three landed by the time `awaitInitialReplication()` resolved (elapsed **1 ms** into a
20 000 ms bound — the pull had already delivered them before the poll began).

**🛑 The UPDATE case was EXERCISED, not skipped, and it is corroborated at three levels:**
the substrate row kept the **same primary key** and gained the dark-window body; the RxDB
document under that same key carries the dark-window value after reconnect; and HQ's own
Postgres shows field B holding exactly **1** draft row (the shell asserts this and reds a
green that fails it). So this green is not vacuous.

**It was a CHECKPOINT pull, not a full re-read** — observed, not assumed. The supabase
plugin hands `queryBuilder` the checkpoint it is about to resume from; the first
post-reconnect pull was handed exactly the checkpoint captured at the sever
(`{id: spikec-4cc42568…, modified: 2026-08-07T13:09:31.989749+00:00}`), the second carried
the advanced one. The script refuses a green whose first post-reconnect pull had no
checkpoint, because a full re-read would recover everything for a reason that has nothing
to do with catch-up and would not survive a real dataset.

### The `submitted_at` semantics finding (the card's named FINDING)

Measured this run against the schema HQ's **own migrator** built, plus the row-level
timestamps of rows this run actually wrote:

1. **`checklist_submissions.submitted_at` NEVER advances after INSERT.** Column default
   `now()`, **0** user triggers on the table, and the only UPDATEs HQ issues against it
   (`repository.go:1186` approve, `:1232` reject) set `status` / `reviewed_by` /
   `reviewed_at` and leave `submitted_at` alone. It is a **creation** timestamp wearing a
   watermark's name. An advancing-watermark catch-up keyed on it would silently miss every
   approval and every rejection — the same unreliable-watermark shape that disqualified
   `answered_at` for spike C's polling-relay candidate.
2. **`submission_responses.answered_at` DOES advance on the update branch** — measured
   end-to-end on the real path: `13:09:31.954691Z` → `13:09:36.146254Z` across the dark
   UPDATE, stamped explicitly by `repository.go:829`
   (`ON CONFLICT … DO UPDATE SET value = EXCLUDED.value, answered_at = now()`).
3. **🛑 Neither of them is what the pull actually checkpoints on, and that is why the green
   is green.** The RxDB supabase plugin checkpoints on the **substrate's** `_modified`
   column, stamped by `hq_sync_checklists_set_modified` (a BEFORE INSERT OR UPDATE trigger,
   `sql/hq-bridge-fixture.sql:79-90`) on every projection, and resumes with a strict
   `_modified > m OR (_modified = m AND id > id)` — a `gt` with an id tie-breaker, **not** a
   `gte`. The watermark is therefore minted at the projection boundary by the substrate
   itself and is completely independent of whatever HQ did or did not do to its business
   timestamps.

**What Activity 3 inherits from that.** The green is *conditional on the relay
re-projecting the row on every HQ change*. It holds here because the carrier is a row-level
`AFTER INSERT OR UPDATE` trigger, which fires whatever moved. **Any future design that
polls HQ on a business watermark instead of NOTIFY reintroduces the miss exactly** — and
`submitted_at`, demonstrably, does not advance. Spike C's candidate (2), the polling relay,
was already ranked strictly weaker; this measurement is the concrete cost of picking it.
The equivalent live-side clause `sync-rxdb/client.js:874` emits
(`submitted_at=gte.<since>`) is a fixed **floor**, not an advancing cursor, so it does not
carry the miss shape — but the red leg shows the realtime stream alone recovers nothing
across a gap regardless, so the floor is not load-bearing for catch-up either way.

**B-161's disposition: answered.** The disconnect/reconnect/catch-up cycle closes, the
build cards do **not** need an explicit resync step *for this carrier*, and the condition
under which they would need one is now written down and reproducible on demand.

Night-Crew-Run: 20260808
