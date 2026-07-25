# Merge intent — W2 `sync-spike-rxdb-replication`

Branch: `card/w2-sync-spike-rxdb-replication` off `overnight-20260725` @ `51d0c02`
(which already carries W1's fold).

---

## Files I own outright (new, nobody else writes them)

- `.night-crew/qa/spike-supabase/rxdb/**` — the isolated Node harness: its **own**
  `package.json` + `package-lock.json` + `node_modules/` (gitignored), the RxDB
  collection definition, and the push / pull / LWW proof scripts.

Nothing else in the repo is created by this card.

---

## Shared files touched — all three are APPEND-ONLY surfaces

| File | Owner | Why I touch it | What must survive |
|---|---|---|---|
| `.night-crew/qa/spike-supabase/README.md` | W1 | Card asks for **runbook half 2**. W1 left an explicit `<!-- SEAM: HALF 2 ... -->` HTML comment at the end of the file. I append **strictly below that seam**. | **Every byte above the seam, verbatim** — including the LOCAL-ONLY banner, the throwaway-credential warning, all of W1's numbered proofs, the sharp-edges list, and the "Open question for the operator" section. If a merge shows any diff above the seam line, that is a defect in my card, not a conflict to resolve. |
| `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md` | W1 | Card asks me to **append the RxDB verdict** + sizing for `sync-rxdb-schema-and-replication`. | W1's stack/JWT verdict section in full. W1's GO is **settled and is not mine to revise**; my append sits after it as a separate, clearly-headed RxDB verdict. |
| `.night-crew/runs/2026-07-25-autonomous/timings.log` | run-shared | Leg instrumentation (`W2_*`). | F1's legs, W1's legs, **and the orchestrator's correction block** (the `>>>` lines attributing the two reds to `d1674d3`). I append at EOF only; I never rewrite or reflow the file. |

Possible fourth, only if I hit a park trigger:

- `.night-crew/runs/2026-07-25-autonomous/DECISIONS-NEEDED.md` — append a new
  `## W2 — …` section at EOF. **Never overwrite**; the orchestrator has already
  parked the F1 regression there and that content must survive intact.

---

## What must survive any merge

1. The whole `rxdb/` harness directory including its **own lockfile** — the
   harness is the artifact the operator runs; without the lockfile the versions
   in the verdict are unreproducible.
2. My README half-2 append, positioned **below** W1's seam comment.
3. The RxDB verdict append in the design note, positioned **after** W1's verdict.
4. My `W2_*` timing legs at the tail of `timings.log`.
5. The licensing/storage finding and its source URL — it is a go/no-go input for
   the operator and must not be summarized away.

## What is safe to drop

- `.night-crew/qa/spike-supabase/rxdb/node_modules/` — gitignored, never committed,
  reinstallable from my lockfile with `npm ci`.
- Any transient log/output file I write while capturing evidence. All evidence
  that matters is transcribed into the README with its observed output; scratch
  files are not load-bearing.

Explicitly: **there is nothing else safe to drop.** Every other byte in my diff
is either the harness or the evidence.

---

## FOUR HARD BLAST-RADIUS CONSTRAINTS — per-item attestation

1. `backend/go.mod` — **UNTOUCHED.** I add no Go dependency. I do not build or
   modify Go code; my only Go usage is `go run ./mintjwt` inside
   `.night-crew/qa/spike-supabase/`, which has its own separate `go.mod` (W1's)
   and is invoked read-only.
2. Root `package.json` and root `package-lock.json` — **UNTOUCHED.** `rxdb` and
   `@supabase/supabase-js` install into `.night-crew/qa/spike-supabase/rxdb/`
   under its **own** `package.json` + lockfile. I never run `npm i` from the
   repo root; every install is preceded by a `cd` into the rxdb directory. The
   root pair must remain byte-identical to `overnight-20260725` — that is the
   Playwright environment every other card in the cycle builds on.
3. `docker-compose.nc.yml` — **UNTOUCHED.** The night-crew ephemeral stack is
   unrelated to this spike; I reuse the already-running `nc-f1-postgres-1`
   (host port 46413) for the G4 Playwright leg and create only a throwaway
   database inside it (`hq_test_e2e_w2`).
4. Root `Taskfile.yml` — **UNTOUCHED.** The operator-runnable path is a README of
   real commands, deliberately not wrapper targets. I add no task.

## Fifth line — W1's stack

5. `docker-compose.supabase.yml` — **UNMODIFIED.** W1 owns it. I **consume** it
   only: `docker compose -p spike-supabase -f docker-compose.supabase.yml port …`
   to resolve host ports, against the stack W1 left running. I do not bring it
   down and I do not change a byte. If my harness had needed a compose change,
   that would be a **W1 amendment** — a park-and-declare, not a silent edit.

Also untouched, for completeness: **no product code.** Nothing under
`backend/internal/**`, no `*.html`, no `sync.js`, nothing under `tests/**`.
The orchestrator can confirm all of the above with:

```
git diff --stat overnight-20260725..HEAD -- backend/go.mod package.json package-lock.json \
  docker-compose.nc.yml Taskfile.yml docker-compose.supabase.yml \
  backend/internal tests '*.html' sync.js
```

which must print **nothing**.

---

## Merge order note

W1's fold is already in my base, so my README and design-note appends apply on
top of W1's final text. No W1/W2 textual conflict is expected in those two files
because my content lives strictly after W1's terminal seam / verdict. The one
file with a genuine concurrent-append shape is `timings.log`; a conflict there is
resolved by **keeping both sides in chronological order**, never by picking one.

---

## Revision round (post-G6, docs-only)

The card went to a fresh G6 adversarial reviewer: **PASS-WITH-FINDINGS**, with
**one blocking finding, docs-only**. No proof was re-run, no harness logic
touched, no suite re-run. The five constraints above are unchanged and still
clean; the appends are still strictly additive (README 681/0, design note 242/0
against `51d0c02`, W1's half 1 byte-identical).

**Blocking finding — corrected.** I had written that the discarded write is
silent *and* that there is no signal an application could subscribe to. The
second half was wrong. `RxReplicationState` exposes `conflict$`
(`rxdb/dist/esm/plugins/replication/index.js:44,51,287-289`) and in this exact
scenario it emits one event carrying the discarded local write. I verified this
myself before editing: read the shipped source, then ran a throwaway probe of
the `proof-lww.js` scenario with a `conflict$` subscription added
(`error$` 0 events, `conflict$` 1 event, `input.newDocumentState.body ==
"LOCAL-EDIT (written second, T2)"`, `output` the server state). The probe was
deleted, not committed; the runbook records the one-line change that reproduces
it. Corrected in all three operator-facing places (runbook step 5,
DECISIONS-NEEDED FORK 3, design-note sizing) to **silent by default** —
nothing thrown, nothing on `error$`, nothing reaching the user without code —
**but** the signal exists, so surfacing a discarded write is a subscription plus
UI. **DECISIONS-NEEDED option 4 was re-priced accordingly**; that was the point
of the fix, since its cost had been inflated into "new plumbing".

**Three non-blocking factual fixes**, all confirmed against sources before
changing:

- Changelog attributions were off by one release each. `FIX(supabase-replication)
  push.modifier is not used` is **16.21.0 (25 Nov 2025)**, not 16.20.0;
  `feat: replication-supabase querybuilder` is **16.21.1 (2 Dec 2025)**, not
  16.21.0. Verified in the shipped `node_modules/rxdb/CHANGELOG.md`. The
  "young but actively maintained" conclusion is unaffected.
- "45–130 ms" sat under a transcript showing 121/129 ms; the 45 came from the
  untranscribed earlier run `r1784996802` (59/45/121 ms, in `timings.log`).
  Both the runbook and the design note now use the transcribed numbers and name
  the second run explicitly. The design note had the same defect in two more
  places; all fixed for consistency.
- The `rx-storage.html` quote was trimmed without an ellipsis, dropping
  *"…which is a bit faster and has a smaller build size"*. Re-fetched the page
  and restored the clause in full in both documents — it supports the
  conclusion that premium buys speed, not capability.

Nothing else was relitigated and no scope was added.
