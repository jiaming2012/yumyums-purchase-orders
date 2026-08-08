# Conflict log — run 20260809

One entry per merge into `overnight-20260809`, clean or conflicted (launch prompt §merge, §15ad.66).
An empty log must never be mistakable for "no conflicts" — every merge is logged, clean merges as a
one-line entry. Morning triage audits this file.

Serial run: `demo-sync-target` (C1) → [stretch-gate] `spike-exit-code-honesty` B-163 (C2) → closeout.

---

## Merge 1 — `card/c1-demo-sync-target` → `overnight-20260809`  ·  CLEAN

- **Card:** C1 `demo-sync-target` (Activity 5, milestone close-bar deliverable).
- **Merge:** `git merge --no-ff card/c1-demo-sync-target`. **Automatic merge went well — zero conflicts.**
  First card of the run; base `overnight-20260809` had not moved from `bdf9f5a`, so nothing to collide with.
- **Files brought in (4):** `.night-crew/qa/spike-supabase/demo-sync.sh` (new, 223 lines),
  `Taskfile.yml` (+43 — new `demo:sync` + `demo:sync:red` stanzas, additive), `.night-crew/knowledge/roadmap.md`
  (card flip PLANNED→DONE), `.night-crew/runs/2026-08-09-autonomous/merge-intents/c1-demo-sync-target.md`.
- **Merge-intent read (c1-demo-sync-target.md):** shared files declared — `Taskfile.yml` (additive, new
  stanza; card 2's future `spike:reconnect:red` note is a disjoint stanza — no overlap expected) and
  `roadmap.md` (card flip only). `workflows.html` READ-only, no diff. No Go, no new specs, no migrations.
  Nothing to resolve against intent — clean.
- **Gate result after merge:**
  - G1 / G2(Go): **N/A-by-footprint** — no `.go` file in the diff.
  - G2(Playwright): **N/A-by-footprint** — no `[e2e.seams]` key matches the changed paths; `workflows.html`
    read-only (no edit → seam does not fire). The demo's own tri-state run is the verdict.
  - G3: **N/A** (openspec absent).
  - G4: **precache 31, unchanged by construction** — the card touches no precached/served asset (`sw.js`
    not in the diff; `demo-sync.sh` and `Taskfile.yml` are not served). Confirmed post-commit (below).
  - G4 discipline greps: **N/A-VACUOUS** — neither `internal/journal` nor `internal/workorder` exists in
    this repo (B-14).
  - RF: tri-state exit captured red-first — exit 2 (could-not-run), exit 1 (ran-and-failed), exit 0 (green),
    all distinct; independently reproduced by G6.
  - G6: **PASS-WITH-FINDINGS** (no fix round). GREEN reproduced (round-trip ~117 ms through real
    `/saveResponse` → NOTIFY relay → PostgREST → running RxDB client); tri-state 0/1/2 reproduced distinct;
    wrapper trap confirmed (`task demo:sync:red`→201 vs script's true 1); substrate cleanliness
    independently cross-checked (G6 minted its own service_role token — zero residue, control set intact).
    **Operator-awareness finding (MEDIUM):** the read surface is a Node RxDB replication client, NOT the
    browser app UI — clears the close-bar *letter* but exercises no app surface; carried to
    DECISIONS-NEEDED / HANDOFF for the operator's attestation run. Deliverable-vs-alias (LOW): `demo-sync.sh`
    is a deliberate, well-documented re-export of `spike-c-roundtrip.sh` — packages+names the proven Spike C
    round trip as the milestone demo; acceptable.
- **Resolution taken:** none needed (clean). Verdict GREEN; card merged as landed on the run branch.

---

## Merge 2 — `card/c2-spike-exit-code-honesty` → `overnight-20260809`  ·  CLEAN

- **Card:** C2 `spike-exit-code-honesty` (B-163). Fixes 3 exit-code conflations so infra failures exit 2, not 1.
- **Merge:** `git merge --no-ff card/c2-spike-exit-code-honesty`. **Automatic merge went well — zero conflicts.**
  Card 2 branched off `overnight-20260809` @ `0fade6b` (POST card-1 merge), so its `Taskfile.yml` change sits
  ON TOP of card 1's `demo:sync` stanza — no divergence, nothing to resolve.
- **Files brought in (5):** `.night-crew/qa/spike-supabase/spike-e-reconnect.sh` (+41/-9, seven `srcpsql`
  guards + vacuous-green→2), `.night-crew/qa/spike-supabase/rxdb/spike-e-reconnect.js` (+23, uncaught→2
  handlers), `Taskfile.yml` (+10, comment-only 201-trap note on `spike:reconnect`), `.night-crew/knowledge/BACKLOG.md`
  (B-163 → RESOLVED, 1 line), merge-intent.
- **Merge-intent read (c2-spike-exit-code-honesty.md):** shared files declared — `Taskfile.yml` (comment-only
  note on the `spike:reconnect` block, disjoint from card 1's `demo:sync`; nothing card 1 wrote touched) and
  `BACKLOG.md` (B-163 flip only, B-168 untouched). Confirmed against the tree: the Taskfile hunk is additive
  comment lines on a different stanza than card 1's — no textual or semantic collision. Clean.
- **Gate result after merge:**
  - G1 / G2(Go): **N/A-by-footprint** — no `.go` file in the diff.
  - G2(Playwright): **N/A-by-footprint** — spike-only; no `[e2e.seams]` key matches the changed paths; no
    spec exercises the spike scripts/targets. No seam fires.
  - G3: **N/A** (openspec absent).
  - G4: **precache 31, unchanged by construction** — no served/precached asset touched (shell + node scripts
    under `.night-crew/`, Taskfile note, BACKLOG). Confirmed post-commit (below).
  - G4 discipline greps: **N/A-VACUOUS** (B-14).
  - RF: three exit-code conflations each proven red-first — infra failure 1→2, vacuous-green 1→2, uncaught JS
    exception 1→2 (both sync-throw and rejected-await shapes). Independently reproduced by G6 (BEFORE + AFTER).
  - G6: **PASS** (no fix round). All three transitions reproduced; **no over-correction** — `die(RED)` still
    exits 1 (confirmed three ways), green still 0, `B_ROWS=2` reaches the vacuous eval not a mask; completeness
    confirmed at seven guards; contract preserved 0/1/2/3.
- **Resolution taken:** none needed (clean). B-163 fixed and flipped RESOLVED; card merged as landed.
