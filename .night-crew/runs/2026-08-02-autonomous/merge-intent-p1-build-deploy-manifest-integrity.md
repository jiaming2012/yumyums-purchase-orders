# Merge intent — Card P1 `build-deploy-manifest-integrity`

Run: `overnight-20260802`
Branch: `card/p1-build-deploy-manifest-integrity`
Base: `overnight-20260802`, cut **after** A1, A2 and B1 merged (`de7d78c`).

**What this card does, in one line:** `build-sw.js` gains an **import-reachability
guard** that makes B-37's silent drop fatal — *nothing precached may import
something not precached* — and `CLAUDE.md` stops claiming `task prod:deploy` runs
`task sw`, which it has never done (B-13, **decided at slate: the Taskfile is
right and the doc is wrong**).

> 🛑 **This note is load-bearing for Night B.** `S1 sync-hard-cutover` edits
> `build-sw.js` again tomorrow, after **both** B1 (tonight, merged) and this card.
> §2 is the contract it is held to, and §2.4 names the one thing in this card that
> S1 can trip over without meaning to.

---

## 1. Shared files touched

| File | Shared with | Why this card touches it |
|---|---|---|
| `build-sw.js` | **B1 tonight (already merged)**, **S1 tomorrow** | The card. Adds the import-reachability guard + two `globPatterns` entries. **B1's two `api-cache` plugin hooks are NOT touched** — see §2.5. |
| `sw.js` | B1 (regenerated it), S1 (will) | Generated artifact, committed by contract. Regenerated AFTER the source commits, per G4. **Count moves 29 → 31 — this card is the one allowed to move it; justification in §2.3.** |
| `CLAUDE.md` | nobody tonight | B-13. The deploy block described a flow the Taskfile does not run. |
| `tests/sw-manifest.spec.js` | nobody tonight | The guard on the guard. B1 did not touch this file; neither did A1 or A2. |
| `.night-crew/knowledge/BACKLOG.md` | every card tonight | The status flip for **B-37 and B-13** (P1 is a promoted backlog item, **not** a roadmap bullet) plus discoveries B-52/B-53. |
| `.night-crew/knowledge/roadmap.md` | every card tonight | **NOT TOUCHED.** P1 has no roadmap card — it was promoted straight off the backlog. No bullet was invented for it. |

## 2. What must survive any merge

### 2.1 The invariant is *reachability*, not *completeness*

`checkImportReachability()` fails the build when a **precached** `.html`/`.js`
file references a local path that is **not in the final manifest**. It says
nothing about files nobody references.

- **A blanket "everything must be precached" rule is explicitly NOT what this is**
  and must not be turned into one. `build-sw.js` still exits 0 with unreferenced
  files skipped — that is the feature (`README.md`, `playwright.config.js`,
  `workbox-*.js`, every scratch `*.html` on a dev box). Proven by the third red
  case in §3.
- The two failure reasons are reported **distinctly**, because the fixes differ:
  `skipped (not in HEAD)` → *commit the file*; `not matched by globPatterns` →
  *add a glob **and** the matching `backend/Dockerfile` copy* (decision 59's trap,
  guarded by `tests/sw-manifest.spec.js`).

### 2.2 Bare module specifiers are NOT paths — do not "simplify" this away

`vendor/rxdb.bundle.js` contains `from "ws"`. An HTML `src="log.js"` **is** a
path; a JS specifier `"ws"` **is not**. The guard therefore accepts any
non-scheme, non-`//` HTML attribute value, but accepts a JS specifier only when it
begins `./`, `../` or `/`. **Collapsing the two rules into one makes the build fail
on `ws` and every other bundled dependency name.** This is the single most likely
"cleanup" to break the guard.

### 2.3 Precache count 29 → **31**, and why that is the fix rather than a regression

`log.js` (1.2 KB) and `tab.js` (0.8 KB) are referenced by **every** precached HTML
page — `log.js` by all 7, `tab.js` by 5 — and were in `globPatterns` **nowhere**.
They shipped into the image (`backend/Dockerfile` `COPY *.html *.js`) so they work
online and have always failed offline: on a returning client with no network,
`tab.js` never runs and **five of seven tools open with every tab section
visible at once and no tab switching**, because `tab.js` is what applies
`#tab=N` before paint. That is precisely **D-KR2's returning-client parity**, and
it is what the guard found on the merged tree with no synthetic case at all.

- **31 is the new invariant.** If it moves under any card that is not deliberately
  adding or removing a shipped asset, that is B-37 again.
- No `backend/Dockerfile` change was needed: `COPY *.html *.js` and
  `cp ../*.js cmd/server/public/` already stage both files, so
  `tests/sw-manifest.spec.js`'s obligation-5 guard stays green. **A future glob
  addition may not be so lucky** — check that test before adding one.

### 2.4 🛑 The two canaries — S1, READ THIS ONE

The guard refuses to pass on an empty parse (B-22/B-23/B-24). Anti-vacuity is
enforced three ways, and the third is a hard-coded pair:

```js
const REACHABILITY_CANARIES = [
  ['index.html',     'ptr.js'],                  // the HTML src="" path
  ['workflows.html', 'sync-rxdb/bootstrap.js'],  // the module-graph path
];
```

If either reference legitimately goes away, **`node build-sw.js` fails with a
message naming this list and telling you to update it.** That is deliberate: a
guard whose subject set silently empties is the exact failure this repo keeps
catching. **`sync-hard-cutover` is the most likely card to remove the second
canary** (it owns the RxDB write path). If S1 drops
`<script type="module" src="sync-rxdb/bootstrap.js">` from `workflows.html`,
**replace the canary with whatever module entry point takes its place — do not
delete the row and do not delete the check.**

### 2.5 B1's contract, restated — both hooks survive this card intact

Verified after the edit, not assumed:

- **`cacheKeyWillBeUsed`** on the `api-cache` route — present, unmodified.
  Defended by **`[B1-XT-02]` alone**; a green `[B1-XT-01]` does **not** mean the
  disclosure is closed (B1 §2.2, measured by mutation M1).
- **`cacheWillUpdate`** on the same route — present, unmodified. Defended by
  **`[B1-XT-05]` alone**; it is the hook that *looks* redundant and is not
  (B1 §2.3, mutation M2 was 13/13 green before `[B1-XT-05]` existed).
- The **three literal spellings** of `hq-identity` / `/__hq_identity` —
  `build-sw.js`, `index.html`, `login.html` — all intact. **This card added no
  root `.js` file**, exactly as B1 §2.6 asked, so the collision B1 avoided did not
  happen. (It added two `globPatterns` entries for `.js` files that already
  existed.)
- `runtimeCaching` is still **one** route with **one** `cacheName`. `globIgnores`,
  `committedOnlyTransform` and `GENERATED_BUT_SHIPPED` are unchanged.

### 2.6 The guard runs as the LAST `manifestTransform`, on purpose

`manifestTransforms: [committedOnlyTransform, importReachabilityTransform]`.

- It must run **after** `committedOnlyTransform`, or it checks against a manifest
  that still contains uncommitted files and can never see a skip.
- Throwing from a transform means **`sw.js` is not written at all** on failure.
  A guard that writes a bad artifact and *then* exits non-zero leaves something a
  hurried hand can `git add`.
- `committedOnlyTransform` now records its drops in the module-level
  `lastSkipped` set so the reachability report can name *why* a target is absent.
  If a later card reorders or removes a transform, that set goes stale silently —
  keep them adjacent and in this order.

## 3. What is safe to drop

- **The per-file `refsByFile` map** is only used by the anti-vacuity canaries and
  the tests. Nothing in the build depends on its shape.
- **`HTML_MODULEPRELOAD`** — nothing in the tree uses `rel="modulepreload"` today;
  it is there so that adding one does not silently create an unguarded edge.
  Dropping it costs nothing until someone uses it.
- **The `console.log` count line** (`import reachability: N files parsed, …`) is
  evidence, not an assertion. 🛑 **The three anti-vacuity checks it accompanies are
  NOT in this category** — delete those and a guard that parses zero files reports
  PASS, which is the whole class of bug this repo keeps re-finding.
- **The CLAUDE.md wording.** Any accurate description of `Taskfile.yml:178-218`
  is fine; what may not come back is the claim that anything regenerates `sw.js`
  on the deploy box.

## 4. What is NOT in this card

- **No `task sw` dependency was added to `prod:deploy`.** The slate DECIDED the
  Taskfile is right: `build-sw.js` reads **git HEAD**, and the prod clone's HEAD
  after `git reset --hard origin/main` **is** the shipped tree, so regenerating on
  the box is redundant by construction. Flagged for ratification at triage. **Do
  not re-open it.**
- **No deploy.** `task prod:deploy` was not run and was not available.
- **No `openspec/`.** Preflight verdict is `openspec: absent`; nothing here
  creates it.
- **No armed red touched.** `[LST-17]` (both tests matching the bare tag),
  `[A1-TZ-02]` and `[LC-02]` are untouched. `HQ_SYNC_REST_URL` stays armed and
  unset.
- **No backend change.** Zero files under `backend/`.
- **Non-executable subresources are out of scope.** `<link rel="icon">`,
  stylesheets and `<img src>` are not parsed. The guard is about the *module and
  script graph*, which is where absence breaks the app rather than the favicon.
  One real finding fell out of that boundary and is filed, not built — **B-53**.

## 5. Empty fields

- **Migrations:** nothing here.
- **New dependencies:** nothing here.
- **Schema changes:** nothing here.
- **Backend files:** nothing here.
- **Operator forks raised:** nothing here — the card did not park.
- **Roadmap bullets touched:** nothing here — P1 has no roadmap card, by design.

## 6. Filed, not built

- **B-52 — the guard is blind to conditional/computed script injection.** Only
  literal `src="…"` attributes and literal module specifiers are parsed; a page
  that builds a URL at runtime is invisible to it. Nothing in the tree does this
  today. **Destination: NEXT milestone.**
- **B-53 — `favicon.ico`, `favicon-32x32.png` and `apple-touch-icon.png` are
  referenced by every page, committed, and copied into the image by NOTHING.**
  Found while bounding §4's subresource scope. `backend/Dockerfile` stages
  `*.html *.js manifest.json`, `icons`, `lib`, `vendor`, `sync-rxdb`,
  `sync-schema` — no root image files — so these 404 in prod **online**, not just
  offline. Cosmetic, and outside the module-graph invariant this card implements.
  **Destination: NEXT milestone.**
