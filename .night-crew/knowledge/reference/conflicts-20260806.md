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
