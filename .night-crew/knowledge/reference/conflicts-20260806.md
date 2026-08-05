# Conflict log — run `overnight-20260806`

> Required by §15ad.66. **Every merge to the run branch gets an entry — clean or conflicted.**
> A clean merge gets a one-line entry, so an empty log can never read as "no conflicts"
> when it means "the logging never ran".
>
> Fields per entry: the cards involved, the files and hunks, the merge-intent notes read,
> the resolution taken, and the gate result after it.

**Run:** `overnight-20260806` · **Slate:** `slate-20260806.md` · **Dispatch:** concurrent, 3 tracks
**Global Playwright suite lock in force** — queue order C1 → Track A → Track B.

---

## Merges

### 1 · `w0-repo-hygiene` → `overnight-20260806` · **CLEAN** · merge `6f91863`

**Cards involved:** W0 alone. Wave 0 runs first and alone, so no other card's work was in
flight and no collision was possible by construction.

**Files:** `night-crew.toml`, `sw.js`, `sync-rxdb/bootstrap.js`, `sync-rxdb/client.js`,
`tests/repo-hygiene.spec.js` (new), `.night-crew/knowledge/BACKLOG.md` (B-140 filed),
merge-intent note. 7 files, +461/−16.

**Intents read:** `merge-intents/w0-repo-hygiene.md` — sole side, nothing to reconcile against.

**Resolution:** none required — clean `--no-ff` merge, no conflicting hunks.

**Gate after merge:** G4 re-run **at the merge commit** (B-37 — `build-sw.js` reads git HEAD,
not the working tree): `EXIT=0`, **31 files precached**, reachability 18 parsed / 30 resolved /
0 outside, and `git status` clean on the second run ⇒ byte-idempotent. The committed `sw.js`
is correct at HEAD, so the change ships (B-13).

**Note carried:** G6 returned MERGE WITH NOTE. The residual stale activation gate at four
further sites was deliberately **not** fixed here and is filed as **B-140** — `workflows.html`
is a precached shipped asset, so retiring it forces an `sw.js` regen plus a full-suite gate.
That is a card, not a fix round; destination named as `sync-hard-cutover`.

---

### 2 · `c1-spike-env-up` → `overnight-20260806` · **CLEAN** · merge `76dc12b`

**Cards involved:** C1 alone. Track C shares no file with any other card in this slate —
that disjointness is why it got its own track — so no collision was possible.

**Files:** `Taskfile.yml`, `docker-compose.supabase.yml`, `.night-crew/qa/spike-supabase/`
(`env-up.sh`, `README.md`, `rxdb/healthcheck.js`, two captures), plus a gate log and the
merge-intent note. 9 files.

**Intents read:** `merge-intents/c1-spike-env-up.md` — sole side, nothing to reconcile.
Its original "driven, not edited" disclaimer for `docker-compose.supabase.yml` was
**retracted in the fix round**, because F1's durable fix required editing that file. The
retraction is explicit in the note under its own heading. §15ad.65: editing outside the
declared footprint is not a breach — failing to declare it is, and it is declared.

**Resolution:** none required — clean `--no-ff` merge, no conflicting hunks.

**Gate after merge:** no G4 re-run needed — **C1 touches no precached asset** (verified:
its diff contains no `.html`/`.js`/`.css`/`.json` outside `.night-crew/`, and
`rxdb/healthcheck.js` lives under `.night-crew/qa/`, which `build-sw.js` does not precache).
Precache stays 31 from merge 1.

**Note carried:** G6 returned MERGE WITH NOTE and found a **destructive** defect the card
had asserted the opposite of — `task spike:up` could wipe the spike database, via a missing
PGDATA volume plus a path-sensitive compose config hash. Both fixed and proven in the fix
round before this merge. G6's strongest observation stands on the record: this mechanism is
very likely the cause of the card's own headline finding (three healthy containers, five
days, no schema) — the card built a detector for the symptom and never diagnosed the cause,
and the cause was its own command.

---

### 3 · `a1-rls-count-assert` → `overnight-20260806` · **CLEAN** · merge `9b63958`

**Cards involved:** A1 alone at merge time. A1 shares `backend/internal/sync/` with **A3**,
which was dispatched only after this merge landed, precisely so A3 inherits A1's constant.
A1 left `rowvisibility_rls_test.go` and `jwtbridge_rls_test.go` **zero bytes changed**
(verified by me before merging), so A3's file arrives clean.

**Files:** `backend/internal/sync/spikestack_gate_test.go` (+804), `BACKLOG.md` (B-36 closure),
merge-intent note. 3 files, +1097/−4.

**Intents read:** `merge-intents/a1-rls-count-assert.md` — sole side. `git merge-tree` against
the merge base reported **0 conflict markers** before the merge was taken.

**Resolution:** none required — clean `--no-ff` merge.

**Gate after merge:** G4 not owed (Go test files + markdown only; nothing precached).
G2(Playwright) was **discarded and re-run** — see below — and the re-run is the cited figure.

**🛑 A discarded gate, recorded so the reasoning is auditable.** A2's G6 disclosed that its
first, *unlocked* end-to-end `verify-test-harness.sh` run — a `go test` over 7 packages,
i.e. a Go suite — overlapped A1's full Playwright suite. The slate forbids a Go suite
alongside a Playwright one and requires the overlapped run be **"discarded and re-run, not
reasoned about."** A1's original result showed zero failures, so a conditional reading would
have let it stand; the rule's wording exists to forbid exactly that reasoning. Discarded.
The clean re-run, taken alone under the mutex: `EXIT=0`, **791 passed + 6 skipped = 797**
across 29 files, **exactly one summary block** counted over the complete log, 23.5m.

**Root cause is the orchestrator's, not the card's.** The unlocked-probe carve-out was mine,
written for A2's *cheap per-package probes* (seconds each, nonexistent DB, no ports). A2's G6
reasonably extended it to the eleven-minute end-to-end harness, which is a Go suite, not a
probe. The exemption was drawn too broadly.

**Note carried:** G6 returned MERGE WITH NOTE on 9 findings. The sharpest: the card closing
*"a gate can print `ok` having run nothing"* had itself shipped a gate that prints `ok`
having run nothing — `HQ_SYNC_GATE_CHILD=1` disarmed both new guards while `internal/sync`
reported `ok` in 0.008s with `EXIT=0`. Fixed here as a parent-minted token. One finding (F7,
the ladder's stale eyeball-the-count instruction) was routed to **A4**, sole owner of that
file. One (G3's contradictory definition) is an operator question and is in DECISIONS-NEEDED.
