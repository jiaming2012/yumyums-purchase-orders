# HANDOFF — run `overnight-20260725`

**Branch:** `overnight-20260725` (never pushed; `main` and `dev` untouched)
**Slate:** `.night-crew/knowledge/reference/slate-20260725.md` — batch sign-off 2026-07-25
**Conflict log:** `.night-crew/knowledge/reference/conflicts-20260725.md` (one entry per merge)
**Open forks:** `.night-crew/runs/2026-07-25-autonomous/DECISIONS-NEEDED.md` — **four, two of them substantive**

> **This run was resumed by hand.** The original session started F1 at 09:17 and ended after
> `F1_IMPL_START` without merging. An attended session folded F1's server half at 09:xx–10:29, then
> handed control to this session at 10:34, which ran F1's owed subset leg and dispatched W1 and W2.
> Setup was skipped by operator instruction — no branch was cut, and `overnight-20260725` was
> already correct.

---

## 🛑 THE ONE THING TO DO FIRST

**W1's go/no-go verdict must be recorded in `.night-crew/knowledge/ledger.md` before the three
downstream Activity-1 cards may dispatch.** Those cards are
`sync-rxdb-schema-and-replication`, `sync-jwt-bridge-endpoint`, and `sync-hard-cutover`.

**Product KR1 and Delivery KR1 both measure the `ledger.md` timestamp against `.night-crew/runs/`
dispatch timestamps.** `ledger.md` is an *attended* artifact — every entry is a `T-nn` triage record
carrying an operator decision — so **this run could not write it, by construction.** Dispatching
those cards before the verdict lands in the ledger breaches both KRs.

**The verdict to record: W1 = GO. W2 = GO on RxDB, with one signed assumption disproven (see FORK 3).**

Recording that is the single highest-value thing you do this morning. The next slate
(`sync-rxdb-schema-and-replication` + `sync-jwt-bridge-endpoint`, disjoint footprints, genuinely
parallelizable) is **one triage away, not one night away** — *except* that FORK 3 now gates the
conflict-policy half of the schema card. Read it before sizing.

---

## Per-card outcomes

| Card | Verdict | Merged | G6 | Roadmap |
|---|---|---|---|---|
| **F1** `workflow-submission-status-default` | ⚠️ **server half merged, CLIENT HALF MISSING** | `53e921d` (attended fold) | n/a — folded attended, pre-dates this session | **PLANNED** (flip to DONE reverted — see FORK 1) |
| **W1** `sync-spike-stack-and-jwt-bridge` | ✅ **GO** | `51d0c02` | PASS-WITH-FINDINGS, 8 non-blocking | **DONE** |
| **W2** `sync-spike-rxdb-replication` | ✅ **GO on RxDB**, one signed assumption disproven | *(see merge 3 in the conflict log)* | PASS-WITH-FINDINGS → 1 blocking, revised, re-checked | **DONE** |

**Zero cards parked for lack of progress.** Every card reached an evidenced verdict. The two open
forks are *findings*, which is what a spike is for — not failures.

---

## What each card actually established

### F1 — the fix is right, but it is half a change

Server-side normalization landed and its Go gates are green: `submitChecklist` now writes
`status='completed'` for `requires_approval:false` (a value the 0011 CHECK already permitted and
nothing used — no migration, no new lifecycle row), and `pendingApprovals` gates on the submission's
own frozen snapshot rather than the live template flag. That closes the real approvals leak.

**But it regressed two E2E tests, and I attributed that by measurement rather than argument:**

| Tree | F1 present? | Result |
|---|---|---|
| `dev` @ `d37fb10` | no | **2 passed** (2.6 m), load 2.84 → 3.28 |
| run branch @ `c14cbce` | yes | **2 failed**, load 3.01 → 2.48 |

`tests/repro-cut-task.spec.js:153` (AC-6b) and `tests/sync.spec.js:1581` (Convergence matrix W-3).
`workflows.html` recognises only `submitted` / `pending_approval` / `pending` / `approved`
(`:2093-2095`); `'completed'` matches none, so `.submit-confirm` never renders.

**The roadmap's premise was backwards.** It called the stuck `'pending'` *"harmless today."* It was
**load-bearing** — it hit the `isPending` branch and rendered *"Submitted for approval… Waiting for
manager review,"* wrong copy for a no-approval template but a rendered element, so the suite stayed
green. F1 removed the value the client was accidentally relying on and put nothing in its place.

**→ FORK 1.** Three options laid out, none chosen. This is F1's own park trigger (ii) — *"an existing
test depends on no-approval submissions reading `'pending'` → that is a contract question, park it
rather than editing the expectation."* I did not edit a test, revert the fix, or pick a repair.

**This also settles the question you left open:** commit `0b53d46` on
`card/f1-workflow-submission-status-default` (red-first Playwright test, no client half) is **NOT
redundant** with the Go-layer proof. It is the test for the missing half, now proven necessary by two
independent specs its author was not looking at. Recommend keeping it and pairing it with whichever
option you choose.

**🛑 And the seam map is wrong.** F1 was slated seam-confined, so it paid only the
`workflows|persistence` subset — which I ran, and which was **green (102 passed)**. *Neither failing
spec is in that subset.* Both `tests/sync.spec.js` and `tests/repro-cut-task.spec.js` exercise the
submit seam F1 changed. **Tonight's evidence is that a green subset bought false confidence.**
Recommend extending `[e2e.seams]` so a `submitChecklist` change de-confines to those specs too.

### W1 — GO. The stack works, and it is smaller than expected

Self-hosted Supabase — `supabase/postgres` + `postgrest/postgrest` + `supabase/realtime`, and
**Kong, Studio and GoTrue all proved unnecessary** — accepts an HS256 token minted by a ~10-line
stdlib-only Go program, on **both** PostgREST and Realtime.

**RLS demonstrably discriminates**, which was the card's sharpest requirement (*"a 200 alone proves
nothing"*). Authorized reads return disjoint per-owner rows on a byte-identical URL; anon is refused
`42501`/401; a forged `owner_id` write is refused **403 `new row violates row-level security
policy`**; expired and wrong-secret tokens are refused. Critically, a `service_role` **BYPASSRLS
control** returns all rows on the same endpoint — ruling out "the table was empty," which is how this
proof usually fools itself. **G6 reproduced the whole discrimination pair independently** and got
byte-identical output.

**Sharpest edge, as predicted:** self-hosted Realtime tenancy. Resolved via `SEED_SELF_HOST=true`
rather than hand-writing the AES-128-encrypted `jwt_secret` row. The real trap was **Host-header
tenant routing** — the first dot-label of the Host header is the `external_id`, and a wrong host
gives a bare 403 with no body.

**Per-table contract cost (this sizes `sync-rxdb-schema-and-replication`):** three mechanically
repeated DDL statements plus one policy set per table. **None required per-table judgement** — it is
generatable from a table list.

**A guess of ours corrected by measurement:** forgetting `ALTER PUBLICATION` is **not** silent, but
`phx_join` still replies `{"status":"ok"}` with a subscription id and the error arrives in a *later*
`system` frame. Any HQ wrapper that resolves "subscribed" on the join reply will believe it is
subscribed to a table that never fires.

**Bonus finding:** Realtime enforces RLS **per subscriber** — a subscriber saw only its own rows.

**Landmine for `sync-jwt-bridge-endpoint`:** `auth.jwt()` does not exist here and **`auth.uid()` is
wrong for HQ** — without GoTrue's migrations the `auth` schema ships only `email`/`role`/`uid`, and
`auth.uid()` reads the *legacy singular* GUC (off in this stack) and casts to `uuid`. **Every
copy-pasted hosted-Supabase policy will fail, non-obviously.**

### W2 — GO on RxDB, and it disproved a decision you signed

`rxdb@17.4.0`'s `replicateSupabase` replicates **both directions** over W1's stack:
- **push** — local `insert()` → row in Postgres, verified over an *independent* request;
- **pull** — remote INSERT / UPDATE / soft-delete converge in **~90–130 ms** with the replication
  instance started once and **never restarted, cancelled, or `reSync()`-ed**. Against a 5000 ms
  `retryTime` default, that is two orders of magnitude under — so it arrived over Realtime, not a
  retry. G6 verified the no-restart claim by reading the harness, not the prose.

**Licensing: no paid dependency required.** `rxdb@17.4.0` is Apache-2.0 (verified in the installed
package, not the marketing page). **Dexie storage = free, IndexedDB storage = premium** — your
reading was right. Premium buys speed, not capability, so the "paid dependency" park trigger did
**not** fire. The Supabase replication is a **real shipped plugin** (247 lines, full types), not an
example we would maintain — but **size it as young**: introduced as *beta* in 16.19.0 (Sept 2025),
with fixes since and no entry rescinding the label.

**🛑 THE FINDING — replication is NOT last-write-wins. → FORK 3.**

The explore session chose LWW with no custom conflict handler. Constructed case: agreed doc → client
offline → **Postgres edited first (T1)** → **RxDB edited second (T2 > T1, with a deliberate 1500 ms
gap)** → reconnect. **The strictly-later local write was discarded; the earlier server write
survived.**

**No clock participates.** The mechanism is optimistic concurrency — a compare-and-swap against
assumed master state — and RxDB's `defaultConflictHandler` returns `realMasterState` on every
conflict (its own source comment: *"will always drop the fork state and use the master state
instead"*). `_modified` is only the pull cursor and is excluded from the CAS. **Clock skew is
irrelevant. This is master-wins, not LWW.**

Concretely for HQ: *a crew member completes a checklist in the truck with no signal; a manager edits
the same submission from the office; the phone reconnects — and the crew member's work is dropped.*
For a product whose stated core value is **accountability**, whether that is acceptable is a product
call, and HQ's rows are frequently multi-actor (submitter *and* approver), so "the owner wins" does
not express it either.

Reproduced **three times** — twice by the card, once independently by G6, which also verified every
mechanism claim verbatim against the shipped RxDB source.

**G6 corrected one thing here, and it mattered.** The card originally wrote that the loss is silent
*and that no signal exists to subscribe to*. The second half is false: `RxReplicationState` exposes
**`conflict$`**, which emits the discarded document. G6 proved it with a probe. That error inflated
the cost of DECISIONS-NEEDED option 4 ("surface the conflict to the user") — making a
`conflict$.subscribe(...)` plus UI look like new plumbing — **in a document written to help you
choose between options.** Sent back for revision and corrected; the loss is **silent by default**,
which is still the finding, but it is observable.

**What a Node-side proof does NOT establish** — stated plainly by the card, and worth carrying:
browser storage behaviour (it ran on memory storage — no Dexie/IndexedDB, no quota, no Safari
eviction), persistence across reloads, **service-worker interaction** (HQ's network-first Workbox
`sw.js` vs. RxDB fetches and a long-lived WebSocket is untested and a real risk), PWA offline
semantics (the "offline" was a cooperative `cancel()`), multi-tab leader election, and token refresh
across an offline period.

---

## Gate evidence — final tree

| Gate | Result |
|---|---|
| **G1** `go build ./...` | exit 0 — re-run on the merged tree after conflict resolution |
| **G2** `go vet ./...` | exit 0 — same |
| **G4 Go** `go test ./... -count=1 -p 1` | **9 green packages** (alerts, auth, inventory, purchasing, receipt, recipes, sync, toast, workflow) |
| **G4 Playwright (full)** | **538 passed / 2 failed / 1 flaky / 6 skipped, 30.5 m** — load 1.58 → 4.20 |
| **G4 Playwright (subset, F1)** | **102 passed / 1 skipped / 0 failed, 6 m 18 s** — load 1.83 → 3.60 |

**The 2 failed are exactly the two known F1 reds** (FORK 1). **No new reds from either spike** —
both W1 and W2 touch zero product and zero test bytes, verified by `git diff`, not by attestation.

**The 1 flaky** is `tests/purchasing.spec.js:1407` (FR-13) — died on `waitForLoadState('networkidle')`
at the 30 s timeout, **passed on retry**. It is **not** `sync.spec.js:1198`. Reported with measured
load (1.58 → 4.20) and **deliberately not attributed**: one observation is not an attribution, and
nobody re-ran it in isolation. Worth a watch, not a claim.

**`tests/sync.spec.js:1198` — the known ~16–20 % flake — did not red at any point tonight**, across
three full-suite legs and one subset leg.

**Blast-radius attestation — checked with `git diff`, not taken on trust.** Across the whole run
(`dev..HEAD`), `backend/go.mod`, root `package.json`, `package-lock.json`, `docker-compose.nc.yml`
and root `Taskfile.yml` are **all untouched**, and `docker-compose.supabase.yml` was modified only by
W1 which owns it. The only product delta in the entire run is F1's `repository.go` plus its two test
files.

---

## Per-card timings (Delivery KR3) — measured, not estimated

Full epoch-stamped record in `timings.log`.

| Leg | Wall clock | Slate estimate | |
|---|---|---|---|
| **F1** subset Playwright | **6 m 18 s** | ~8–12 m | **~half the low end** |
| **W1** impl (incl. its own G1–G4) | **53 m 05 s** | 150–300 m | 🔻 **~1/3 of the low end** |
| W1 · image pull (cold, 3.3 GB) | 6 m 36 s | 5–20 m | in band |
| W1 · bring-up (3 attempts) | 3 m 22 s | **30–90 m** | 🔻 **~1/10 of the low end** |
| W1 · JWT mint + PostgREST proof | 48 s | 20–40 m | 🔻 dramatically under |
| W1 · Realtime proof | 2 m 38 s | 20–50 m | 🔻 dramatically under |
| W1 · G6 | 5 m 14 s | 15–45 m | under |
| W1 · merge (incl. conflict) | ~11 m | — | — |
| **W2** impl (incl. its own G1–G4) | **49 m 31 s** | 105–210 m | 🔻 **under half the low end** |
| W2 · `npm i` (cold) | 45 s | 15–30 m | 🔻 dramatically under |
| W2 · push + pull proofs | 2 m 19 s | 60–120 m | 🔻 dramatically under |
| W2 · LWW observation | 2 m 03 s | 15–30 m | 🔻 under |
| W2 · G6 | 7 m 02 s | 15–45 m | under |
| **Orchestrator** · F1 attribution check | 21 m 43 s | *(unbudgeted)* | the run's own investigation |
| Full Playwright legs (×3) | 22.0 m / 8.3 m / 30.5 m | ~20 m | at/over |

**Ledger signal, and it is the biggest one of the night:** *both* first-of-kind cards came in at
roughly **a third of their low estimate**, and the leg the slate called "the sharpest edge"
(Realtime tenant bring-up, priced 30–90 m) took **3 m 22 s**. The slate priced them wide *because*
the ledger had no signal — correct discipline, and now the ledger has one. **The dominant cost of
both spike cards was not the spike; it was the ~20–30 m full Playwright suite each had to pay**, and
the orchestrator's own 21 m attribution investigation. Price future spike cards on suite time, not
infra time.

**Serial critical path predicted 6–12 h. Actual, from resume to closeout: ~3 h.**

---

## State left behind — deliberate

- **The Supabase stack is LEFT RUNNING** (`docker compose -p spike-supabase -f
  docker-compose.supabase.yml`), by requirement, so you can run the spike yourself. **Teardown is
  documented as its own step** in the runbook and was **not** executed by anyone — not the card, not
  either G6 reviewer.
- **The runbook is ready to run:** `.night-crew/qa/spike-supabase/README.md`. Both halves. Every
  command in it was actually executed and its real output captured, and **both G6 reviewers ran
  commands from it verbatim and got matching output** — it works for someone who did not write it.
  Start at Step 1 (port resolution); nothing is hardcoded.
- **The verdicts:** `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md` — GO on the first
  line, W1's half then W2's.
- Throwaway credentials are inline and loudly labelled, per the signed decision. **Never reuse them
  anywhere real.**
- Rows left in the throwaway `spike_notes` table by the cards and both reviewers — harmless by the
  harness's per-run-id design.
- Worktrees preserved: `hq-worktrees/w1-…`, `hq-worktrees/w2-…`, plus the pre-existing
  `f1-…`, `g1-…`, `s1-…`.
- A scratch `dev` worktree at the session scratchpad, used for the F1 attribution check — disposable.

---

## Next actions, in order

1. **Record W1's GO verdict in `ledger.md`.** Unblocks three cards. Nothing else competes for first
   place. *(§"THE ONE THING TO DO FIRST")*
2. **Decide FORK 1** — F1's missing client half. This is a live defect on the run branch: two E2E
   tests are red, and merging this branch to `dev` moves them there. Decide before merging, or merge
   and fix immediately.
3. **Decide FORK 3** — conflict policy. It gates the conflict half of
   `sync-rxdb-schema-and-replication`, so it wants deciding *before* that card is slated. Note the
   corrected pricing: surfacing a discarded write is `conflict$.subscribe(...)` plus UI.
4. **Dispose of `0b53d46`** — the evidence says keep it and pair it with FORK 1's chosen option.
5. **Consider the seam-map fix** — `[e2e.seams]` for `backend/internal/workflow` submission-status
   changes should include `sync.spec.js` and `repro-cut-task.spec.js`. Tonight is the evidence.
6. **FORK 4 — Kong, or a permanent client shim?** Lower stakes but should be *decided* rather than
   inherited by accident. `@supabase/supabase-js` freezes `<baseUrl>/rest/v1` and
   `<baseUrl>/realtime/v1` in its constructor — it assumes **one origin behind Kong**, and W1
   deliberately did not deploy Kong, so a stock `createClient()` cannot reach either service.
   W2 bridged it in ~25 lines using extension points supabase-js already exposes
   (`global.fetch`, `realtime.transport`) and it worked first try — so this is not a blocker, but the
   migration must choose: **run Kong** (one more service to run, configure and secure) or **stay
   gateway-less with a small permanent client-construction helper in HQ** (one fewer service, but
   standing HQ code coupled to a supabase-js internal that could move in a minor release). Likely
   home: `sync-jwt-bridge-endpoint`.
7. **FORK 2** — an untracked 225 KB `backlog-round.html` appeared in the main checkout at 11:26,
   mid-run. Never tracked in any branch; no card wrote it (both worked in isolated worktrees, and
   both reviewers left theirs byte-clean). `uptime` showed two logged-in users, so the likely author
   is a concurrent session of yours. **I left it exactly as found** — not committed, not deleted. It
   is outside every card's footprint and not mine to dispose of.
8. **Watch `purchasing.spec.js:1407`** (FR-13) next full-suite leg. One flaky observation, not
   attributed.

---

## Run-mechanics notes worth keeping

- **`export PATH="/usr/local/go/bin:$PATH"` is required before any Playwright or Go leg in this
  environment.** The non-interactive shell does not carry Go, and Playwright's `webServer` dies with
  `go: not found` / exit 127 — which looks like a test failure and is not. Cost two aborted legs
  (~4 min, no tests run) before it was diagnosed. **Also: this box's PATH contains spaces**
  (`/mnt/c/Program Files/…`), so `env PATH=$PATH …` splits and fails; always quote.
- **`-p 1` remains load-bearing** for the Go suite. Without it the workflow package reds with
  `checklist_templates_created_by_fkey` FK violations — parallel packages share `hq_test_go` and
  their TestMains truncate `users` cross-package. Documented cross-contamination surface #3.
- **The box was never idle.** ~13 unrelated containers ran throughout (other projects' Postgres,
  Temporal, observability). Every suite result above carries its measured 1-min load; none was
  sampled on a quiet box. A green here bounds the *loaded-ish* condition.
- Naive `$!` after a `nohup … &` in this harness yields a wrapper pid that exits immediately —
  `tail --pid` on it returns instantly and looks like a finished run. Find the real `node`/`go` pid
  with `pgrep -af` first. Cost one false "done" reading.
- No `:8199` latch recurred tonight; separate `TEST_PORT`/`TEST_DB_NAME` per leg (8199/8299/8399/
  8499/8599) kept all four Playwright legs isolated with zero collisions.
