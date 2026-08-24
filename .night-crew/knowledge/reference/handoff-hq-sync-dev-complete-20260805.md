# Handoff — finish the sync milestone end-to-end, dev complete

> **From:** the night-crew development clone, attended session 2026-08-05, at the operator's
> direction. Direction clone → target, for **`yumyums/hq`'s roadmap/milestone planning round**.
> **For:** a fresh session with no access to the conversation that produced this. Everything
> needed is in this file or at a path named in it.
> **The one-sentence brief:** the previous milestone built a tested sync library that nothing
> calls; this milestone exists to make the operator able to **see sync working, without bugs,
> using RxDB, in the dev environment** — and it is not done until they have.
> **Marks:** ✅ = re-verified by execution on 2026-08-05 · 📄 = cited from a run record, not
> re-run. Preserve the distinction if you carry claims forward.

---

## 1. The ask, in the operator's own terms

> As the operator who spent over a week of attended evenings and mornings on the "Sync
> foundation" cycle and could not use anything it built, I want this milestone to end **dev
> complete** — the sync capability running in my dev environment, demonstrated by a script I
> run myself — so that "everything is built" can never again mean something I cannot actually
> use without authoring yet another follow-on milestone.

Context the planning session should hold: this is the **second** time a milestone has had to
be created to cover a predecessor that was not dev complete, and the operator's confidence in
unattended overnight development is shaken. The aggravation dissolves on one observable
outcome: **sync working in dev.** That is the acceptance criterion. Everything in this
handoff serves it.

## 2. Where the last milestone actually ended — verified state

**What exists and is real** (do not rebuild it):

- Substrate schema, RLS and write policies proven against a real Postgres — 59
  `TestRowVisibilityRLS` subtests green; an unresolvable substrate is a gate FAIL unless the
  opt-out is typed (📄 `spikestack_gate_test.go`).
- RxDB database creation and conflict-handler wiring proven against a real engine in a real
  browser (📄 `sync-rxdb-client.spec.js:1423`). Roughly 11,200 lines of spec across the layer.
- The JWT bridge and the interlock: `HQ_SYNC_REST_URL` unset keeps everything inert, and that
  interlock has worked exactly as designed.

**What does not exist** (✅ all re-verified by execution 2026-08-05):

- `createHQSyncDatabase()` and `startHQReplication()` have **zero production call sites** —
  six repo hits, all comments, the import list, and `bootstrap.js`'s two deferred re-exports.
- `window.HQSync.db` is never assigned, so `workflows.html:3590`'s
  `if (db && db.conflict_records)` is dead by construction (📄).
- The 495 KB RxDB bundle ships precached to every crew phone and does nothing (📄).

**The false premise, so no card re-inherits it:** decision 49 claimed RxDB replicates rows
straight from HQ's Postgres with "no API boundary left to translate at." Measured: RxDB
replicates from a **second, different** Postgres (the Supabase substrate), and **nothing
carries a row from the substrate back into HQ's Postgres** — the FDW bridge is one-directional
and carries *permissions*, not data (📄 hq roadmap, park record under `sync-hard-cutover`).
Retiring `/saveResponse` would have silently detached answers from submission. The park that
stopped it was correct.

## 3. The governing decisions — settled; cite them, do not re-litigate

- **Decision 126** (triage 2026-08-02): `sync-hard-cutover` is **retired, not re-slated**.
  The replacement shape: **RxDB serves reads — the two list views and the fill view — while
  `/saveResponse` and `/submit` keep owning writes.** The operator waived P-KR3's parallel-run
  prohibition **for this shape only**; a build WO may not propose the split itself — the
  successor card must cite decision 126.
- **Decision 127**: F-2 resolved with it; B-61 closes (📄).
- The successor card named there — `sync-live-fill-view` — **was never authored.** That
  absence, unnoticed by every close artifact, is why the milestone "finished" undelivered
  (filed in the night-crew clone as B-340/B-341). Authoring it properly is this milestone's
  center.

## 4. The milestone shape — sequencing round, then spikes, then cards, then the demo bar

The operator has designed a new pre-build discipline for this round (captured in the
night-crew clone as **B-345**); this milestone is its first application:

1. **After planning and OKRs: a sequencing round, run by the architect.** Order the legs from
   milestone start to dev-complete finish; name every load-bearing unknown on the path.
2. **Plan a series of spikes that together prove, reasonably, the path can be walked** —
   before any build card is cut.
3. **Hard rule: each spike maps one-to-one onto a small-to-medium script whose entire purpose
   is to flush out gray areas and implementation blockers.** The script *is* the spike's
   verdict: "proven" means the script runs green, never a prose claim. Each script must
   render **"could not run" as a distinct outcome from "ran and failed"** — a spike that
   silently no-ops is the defect class the last milestone died of.
4. **Build cards follow only the legs the spikes proved**, walking-skeleton first: the first
   build card threads one row end-to-end in dev behind a flag, and the demo grows from it.
5. **The close bar is dev complete:** the milestone ships a demo script (e.g. a
   `task demo:sync` target) as a first-class deliverable, and **the milestone may not close
   until the operator has personally run it in dev and seen sync working.** No KR grade, card
   count, or closeout substitutes for that run.

## 5. The spike series — proposed; the sequencing round owns the final list

The first two are the operator's own, verbatim in intent; C and D are what the evidence says
the path additionally needs. Each names the blocker it exists to flush out.

| # | Spike | Script proves | Blocker it flushes |
|---|---|---|---|
| A | **Stand up RxDB + Supabase** from scratch, scripted | a clean environment reaches "both up, schema applied, healthy" unattended | environment/config gray areas; anything hand-configured and undocumented |
| B | **Migration rehearsal** — create one Postgres whose schema mimics HQ's with a small subset of fields; add a data fixture; stand up fresh Supabase + RxDB instances; migrate the fixtured data across | HQ-shaped data actually lands in the substrate and surfaces in RxDB | the schema-mapping and data-path gray areas; the leg nine nights were built on top of without testing |
| C | **The round-trip** — one row written through the real write path (`/saveResponse`) appears in an RxDB-served read within bounded time | the HQ-Postgres → substrate → RxDB read path exists at all | the exact premise that killed `sync-hard-cutover`; this is the load-bearing spike — if C cannot go green, stop and re-plan before any card is cut |
| D | **Realtime filter against a live server** | the replication filter behaves against real infrastructure, not an injected fake | **B-62**, explicitly still open; `HQ_SYNC_REST_URL` unset is the interlock working, not evidence of correctness |

Spike scripts are not throwaway: A seeds the dev environment target, C seeds the walking
skeleton, and together they grow into the `task demo:sync` close-bar script.

## 6. Riders and known defects the cards must carry — "without bugs" is part of the criterion

From the retired card's own record, still riding the successor (📄): **B-62, B-63, B-64,
B-66–B-69, B-79** (a must-survive refusal pinned by no test — `if (false)` leaves the suite
green). Plus, all ✅ re-verified 2026-08-05:

- **One NUL byte in `sync-rxdb/client.js`** makes `grep` report nothing on strings present
  three times (`tr -dc '\000' < sync-rxdb/client.js | wc -c` → 1). Any `done_when:` of the
  form "grep returns nothing" is currently satisfiable by the file being unreadable — fix the
  byte early, and reject that `done_when:` shape in this milestone's cards.
- **`night-crew.toml`'s `sync` token selects 6 spec files while claiming 4**
  (`ls tests/ | grep -c sync` → 6) — a false claim in the file deciding which tests a card
  must run.
- **`bootstrap.js:22` still gates activation "until `sync-rxdb-row-visibility-rls` lands"** —
  that card is merged; the banner is stale in the very file the activation cards will edit.
- 📄 **No gate leg has ever reset the e2e database** (hq B-76 / decision 131) — a green can be
  an artifact; the demo script should run against a scripted-fresh environment (spike A gives
  you one).

## 7. OKR guidance for this round — so the close can see delivery this time

Last cycle's twelve KRs were all honestly gradable and **none measured delivery** — process
scores fully compatible with zero delivery. For this milestone:

- **At least one KR per delivery objective measures the objective's claim directly**, and the
  natural one is free here: *"the operator has run `task demo:sync` in dev and the round-trip
  passes — measured by: the demo script's output at close."*
- **Every KR declares its `measured by:` artifact at authoring, and the grade is dry-run in
  the authoring sitting** — "could not be graded" must be impossible to discover at close
  (night-crew clone B-343 carries the mechanism; apply the discipline by hand this round).
- Spike scripts make honest `measured by:` targets for the de-risking phase ("spikes A–D
  green — measured by: each script's exit status and logged outcome").

## 8. Re-verification — the load-bearing claims, in under a minute

From the hq checkout:

```sh
# §2 — still no call sites (expect 6 hits: comments, imports, deferred re-exports; no calls)
grep -rn "createHQSyncDatabase\|startHQReplication" --include=*.js --include=*.html . | grep -v tests

# §6 — the NUL byte, the test-count claim, the stale banner
tr -dc '\000' < sync-rxdb/client.js | wc -c        # 1
ls -1 tests/ | grep -c sync                         # 6, toml claims 4
sed -n '20,24p' sync-rxdb/bootstrap.js              # "until sync-rxdb-row-visibility-rls lands"

# §3 — the governing record (decision 126 and the park evidence)
grep -n "sync-hard-cutover" .night-crew/knowledge/roadmap.md | head
```

## 9. What this handoff does not claim

- **Not that the last cycle was wasted.** The library layers are real and tested, and the
  cycle disproved a data-losing architecture before it shipped. This milestone is the missing
  last leg, not a rebuild.
- **Not that the spike list in §5 is final.** A/B are the operator's design; C/D are proposed
  from evidence. The sequencing round owns the final series — the hard rule it must keep is
  spike ↔ script, one-to-one.
- **Not that any 📄 row was re-executed on 2026-08-05.** Only the ✅ rows were.
- **Not a slate.** Card sizing, night budgets, and sign-off belong to hq's planning session
  and the operator.
