# HANDOFF — run `20260906-2`

**Branch:** `overnight-20260906-2` (cut from `dev` at `1f7d384`) · **Merge:** `afc9e97`
**Slate:** `reference/slate-20260906-2.md`, signed 2026-09-05 · **1 card, Activity B**
**Landed: 1 of 1. Nothing parked. No operator fork was hit.**

---

## Per-card outcomes

| Card | Outcome | Merge | G6 |
|---|---|---|---|
| 1 · `refusal-holds-before-sync` | ✅ **LANDED** | `afc9e97` (clean) | **APPROVE-WITH-FINDINGS** |

### What it does

B-432 is closed. The §8 `requires_online = true` refusal now holds **before** the
campaigns replica has delivered. The predicate is fail-closed at the policy seam
(`marketing/sync/replicas.js:320-325`):

```js
if (campaignId === null || campaignId === undefined || campaignId === '') return null;
if (byId.has(campaignId)) return { requiresOnline: byId.get(campaignId), unresolved: false };
return { requiresOnline: true, unresolved: true };   // fail closed — B-432
```

🛑 **State it as it is: the refusal is conditional, and the condition is Map membership.**
It keys on whether the campaign is resolved, **never** on replica readiness — `unresolved()`
exists but the fail-closed arm never consults it. That is what closes the codes-arrive-first
window a bare readiness latch leaves open. Decision 166's ratified F2 affordance survives by
construction: a known code always names a campaign, so a genuinely-unknown code
(`campaignId` null) never enters the fail-closed arm and keeps its override + unverified
warning.

*(The overclaim warning in the slate was heeded — the merge-intent leads with a predicate
table and explicitly refuses the unconditional framing. That framing is how B-432 hid last
run.)*

### done_when — both halves, answered separately

**(a) The refusal holds before sync — MET.** Branch-3 e2e with only the `campaigns:` seed
removed: **red EXIT=1** (`Expected "requires-online-unresolved" / Received "override"`,
9 × `<div id="ms-gate" data-branch="override">`) → **green EXIT=0**. G6 independently
confirmed via `git diff 82c4e3d..HEAD -- tests/marketing.spec.js` that the test body was
**not touched after the red commit**, and that commit order is test → migration → vendor →
fix. The red is on unmutated production code. RF gate genuine.

**(b) Attempt-record distinguishability — MET.** Measured on the live arbiter through the
shipped `enqueueAttempt` and the shipped push handler, post-migration:

```
aaaaaaaa… | accepted | override=t | unverified=t | policy_unresolved=t   ← replica failure
bbbbbbbb… | accepted | override=t | unverified=t | policy_unresolved=f   ← genuinely-unknown campaign
distinct statuses on the arbiter: accepted     ← no new terminal status (§9/§19 unchanged)
redeem calls on the unverified rows: 0         ← F-2 guard still diverts
```

G6 traced the field through all eight layers that the spike had measured as *dropping* it
(capture → override write → serialized enqueue → destructure whitelist → RxDB schema v1 →
land-unverified body → normal landing body → server column) and found it carried at every
one.

⚠️ **Caveat, stated because G6 raised it and it is real:** the proof is **stitched from two
runs, not one.** The harness leg drives shipped-source → enqueue → push → arbiter with
literal `true`/`false`; the browser test drives submit-flow → local doc with an *injected*
policy via `setCampaignPolicy`. The chain is continuous **by code**, but no single execution
traverses it whole. The page starts no replica yet (sync provisioning is a later card).

**Codes-arrive-first sub-case: COVERED, not carried.** Built for real in `refusal-run.sh`
leg 6 — new campaign + code inserted server-side, codes replica resynced, campaigns replica
deliberately not. `READY + codes-arrive-first | requiresOnline=true | overrideAvailable=false`.
The leg asserts the campaign is absent from the Map *before* judging, so a leg that cannot
pose its question reds rather than passing quietly.

### Gate evidence on the final tree

| Gate | Result |
|---|---|
| **G1** | `go build ./...` EXIT=0, `go vet ./...` EXIT=0 — **zero `.go` files in the diff** |
| **G2 (Go)** | **NOT RUN** — `backend/` untouched (`git diff --stat -- backend/` empty). Correctly skipped, stated rather than implied |
| **G2 (Playwright)** | `npx bddgen` EXIT=0; full suite EXIT=1 — **860 passed / 4 failed / 6 skipped**, 870 tests, 27.6m, **exactly one summary block** counted over the complete log. De-confined to the full suite (`supabase/`, `vendor/`, `sw.js` undeclared in `night-crew.toml`) |
| **G3** | N/A — `openspec: absent` (preflight re-run at launch, exit 0). No scaffolding created |
| **G4** | Re-run at **merged HEAD** — EXIT=0, **43 files precached** (unchanged), 0 references outside the precache, tree **clean on second run** |
| **RF** | Satisfied twice — the e2e red above, and `refusal-run.sh red-preserved` EXIT=1 with 5 disagreements, proving the harness assertions catch the defect class rather than passing vacuously |

**Post-merge on the final tree:** `tests/marketing.spec.js` — **33 passed, EXIT=0**, one
summary block (`c1-postmerge-marketing.log`).
**Scorecard render:** `night-crew scorecard --repo .` — **EXIT=0**, run `20260906-2`
rendered, all four teams record-backed.
**Nothing left polling:** `night-crew workers check` re-run at 19:14 EDT — queues
`night-crew`, `night-crew-env` clear, no pollers. (Poller TTL caveat: this is what the
check reported at that time.)

**Regression legs:** `f2-run.sh`, `campaigns-run.sh`, `push-run.sh` all EXIT=0.
**Machine surface:** zero new (state,event) pairs — `git diff --stat` on
`marketing/submit-machine.js` and `tests/machine/` is **empty**. Card 6's 460-pair
strictness proof untouched.

### 🛑 The no-new-reds baseline did NOT match — read this before trusting the count

The suite's four reds are **not** the B-433 corrected four-red set. One went green and a
different one went red; the totals are identical, which is exactly how such a swap hides:

| Test | In baseline? | This run |
|---|---|---|
| `sync.spec.js:2976 [SYNC-FC-01]` | yes | failed — expected |
| `sync.spec.js:3062 [SYNC-RF-01]` | yes | failed — expected |
| `sync.spec.js:3136 [SYNC-RF-02]` | yes | failed — expected |
| `sw-api-cache-partition.spec.js:92 [B1-XT-01]` | yes | **PASSED** |
| `inventory.spec.js:1404` | **no** | **FAILED** → **B-437** |

**Characterization of `inventory.spec.js:1404`** (`pending.count()` Received 0, at
whole-suite position 112):

- Green **in isolation on the card tree** — EXIT=0 (implementer).
- Green **in isolation on the base tree `1f7d384`** — EXIT=0 (orchestrator; the implementer
  explicitly reported it had *not* run this leg, so it was run here).
- `tests/inventory.spec.js` is **byte-identical** between base and card branch.
- The diff contains **zero inventory files and zero Go files**. The only shared artifacts
  are `sw.js` — which no test observes, since `serviceWorkers: 'block'` is repo-wide
  (B-15) — and `vendor/rxdb.bundle.js`, which inventory does not load.

**Verdict: non-attributable to this card**, consistent with B-369's data-dependent seeds
meeting B-32's load sensitivity. **Stated honestly: non-attribution rests on isolation runs
plus diff contents, not on a second full-suite run of the base tree.** That stronger check
was not performed. `B1-XT-01` flipping green is the reciprocal instability and says the
baseline set itself is not stable — worth a triage look.

---

## Findings G6 raised that the card did not report

G6's verdict was **APPROVE-WITH-FINDINGS**. Nothing blocked the merge. Five findings:

- **F1 — the one with substance. `lastError` is latched but never re-cleared.**
  `replicas.js:313` sets `lastError = null` exactly once, at the `ready = true` transition;
  `:307` re-latches on every later `error$` emission and nothing clears it again for the
  life of the page. So one routine campaigns pull failure mid-shift makes `unresolved()`
  return `true` for the rest of the session, and every **genuinely-unknown-code** override
  taken afterwards is recorded `policy_unresolved = true` — filing into the bucket the
  migration's own column comment reserves for "the campaigns replica had not delivered."
  **The refusal is unaffected** (known codes key on the Map, never on `unresolved()`) and
  the direction is conservative, so this is not a weakening of the `requires_online`
  refusal and therefore not a park. But the column stops meaning what its DDL says it
  means, and no comment, test, or merge-intent acknowledges it. **Needs a card.**
- **F2 — `SCAN_ATTEMPTS_MIGRATION_STRATEGIES` is executed by no test.** Every harness and
  e2e builds a fresh database, so the "crew phone holding v0 `pending` attempts" case the
  strategy exists to protect is unexercised — on the first schema migration in the tree,
  over the most consequential data in the app. G6 ran it by hand (memory storage, shipped
  spec + shipped plugin): migration succeeded. The **Dexie** path remains unproven and
  there is no regression guard.
- **F3 — the F-2 divert condition was not tightened to match rider (a)'s constraint.**
  `push-replication.js:303` diverts on `doc.unverified_code` alone while the new
  `scan_attempts_names_a_code` also demands `offline_override`. A row with
  `unverified_code=true, offline_override=false` — constructible via `enqueueAttempt`'s
  optional params, no production caller does it — would divert, take a 400 check violation
  and **throw**, i.e. the F-2 head-of-line poison class. Latent, not live.
- **F4 — the merge-intent misstates its own diff.** It says "`P-KR4` was left alone"; the
  diff adds `campaigns: [campaignLowRow()]` to P-KR4 at `tests/marketing.spec.js:907-913`.
  Harmless in effect; it is the class of self-description this review slot exists to catch.
- **F5 (minor) — `attach()` swallows an `error$` subscribe failure with an empty catch**
  (`replicas.js:310`), silently reducing the latch to `!ready`. Deliberate per its comment,
  but untested and unlogged.

## Decisions the card made (engineering-level, decided and stated — not operator forks)

- **UI copy (build-fact 6):** new `data-branch="requires-online-unresolved"` with its own
  words. Reusing "High-value offer: online verification is **required**" asserts two facts
  the device does not have (UI-R3) and misdirects (UI-R6) — "required" reads permanent,
  "hasn't synced yet" reads transient, and it *is* transient.
- **Discriminator name:** kept the spike's `policy_unresolved` — it survived the spike's
  own read-back, so renaming buys nothing.
- **Rider (a):** **COUPLED** in a new constraint rather than documenting the decoupling.
  §19 F2's unverified path *is* the permissioned override path, so the decoupled shape has
  no reader, and leaving it open costs an unauthorised accepted redemption the first time a
  second producer exists.
- **Schema migration:** version bump + named strategy + vendored `RxDBMigrationSchemaPlugin`.
  `autoMigrate:false` was rejected: it strands v0 docs — the crew's unsent overrides —
  invisible to the push replica.
- **B-436 filed and left open, deliberately:** the `!CAMPAIGN_POLICY → false` arm still
  fails open. Outside the spike's proven scope; closing it costs the whole offline override
  on that device, and its landed negative (`campaigns-run.sh` leg 3) pins the opposite.
  Made visible in data (`policy_unresolved=true`), not silent.

## Deviations, stated

- **`TEST_DB_NAME` renamed from the dispatch value.** The slate/dispatch name
  `hq_e2e_c1_20260906_2` is **refused** by `scripts/reset-e2e-db.js`'s own guard
  (`/^hq_test(?:_[a-z0-9]+)*$/`). Ran as `hq_test_e2e_c1_20260906_2` /
  `hq_test_rls_c1_20260906_2` — still unique per leg, which is B-80's actual purpose.
  **The guard was not widened.** The orchestrator's own legs used the same corrected shape.
- **Files outside the stated footprint** (not a breach — §15ad.65, the footprint is
  planning information): `vendor/` (forced — schema v1 throws without the plugin, found by
  a crash, not by reasoning), `marketing/scan-page.js`, the `f2`/`push` harnesses (which
  would have broken identically — all three landed gates re-run green), `sw.js`,
  `.night-crew/` docs.
- **`node_modules` was symlinked** between worktrees rather than `npm ci` (offline box),
  by both the card and the orchestrator's comparison legs. The card mitigated by proof
  rather than assertion: re-running `build-vendor.sh`'s esbuild invocation against the
  unmodified entry reproduced the committed bundle **byte-identically** (`cmp`).
- **`reference/overnight-run-plan-20260707.md`, cited by the launch prompt for G1–G6, does
  not exist in this repo and never has.** That is B-26's dangling pointer, already ruled on
  at triage 2026-08-03 (decision 138): HQ's ladder is
  `.night-crew/knowledge/reference/gate-ladder.md`. Resolved to that file; nothing was
  blocked. **The slate template regressed to the dead pointer again** — see next actions.
- **`gate-ladder.md` is stale in two places** and was overridden deliberately: it states
  precache count **31** (actual **43**), and it still names Postgres **`:5433` /
  `yumyums:yumyums`** — the cluster that serves production. All legs ran on **`:5434` /
  `hqtest`** per decision 155.

## Gates the ritual owes an answer on

- **G4 discipline greps:** **N/A-VACUOUS — neither package exists in this repo (B-14).**
  Not "clean", not "PASS".

## Next actions

1. **Merge `overnight-20260906-2` → `dev`** at triage. One card, clean merge, G6 approved.
2. **Attest close-bar leg 3.** It was 🛑 BLOCKED on B-432; B-432 is now closed, so the leg
   becomes *attestable*. **Attestation is the operator's** (decision 161's class) — no
   overnight can do it.
3. **F1 needs a card** — the un-cleared `lastError` degrading the discriminator's meaning
   after any post-ready blip. This is the one finding with field consequences.
4. **F2/F3/F5** — fold into whichever card next touches the push handler or the schema.
   F3 in particular before a second `unverified_code` producer exists.
5. **B-437** (`inventory.spec.js:1404`) — and the reciprocal `B1-XT-01` flip. The
   four-red baseline is not stable; a base full-suite re-run would settle both.
6. **Carry B-26's dangling pointer back to the clone** — the slate template still emits
   `reference/overnight-run-plan-20260707.md`. Decision 138 ruled this two cycles ago and
   the template was never fixed. Same class as B-14's remedy: clone-side, not hq-side.
7. **Remaining milestone cards: 6** — Activity 0 (`redemption-unknowns-spike`,
   `external-accounts-provision`, attended by design), Activity E, Activity F (both refused
   by the per-goal spike gate — no spikes authored). **The overnight-able queue for
   Activity B is now empty.** It refills only through attended sittings.
8. **Still armed:** the ATTENDED live-camera check from run 20260905 card 5.
