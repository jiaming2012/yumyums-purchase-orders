# Merge-intent — `sync-doc-honesty` (Card 7, Track B, run 20260901)

Closes **B-140**, **B-18**, **B-167**. Comments-and-test only. Every code path is
byte-untouched; the sole executable change is `tests/repo-hygiene.spec.js` case 3,
broadened from slug-spelling to fact-shape (red-first). One precached asset
(`workflows.html`) is edited, which forces an `sw.js` regeneration (B-13) — that is
the whole reason this is a card and not a triage tidy-up.

The card's thesis: **a comment that names a DONE roadmap card as a still-open
precondition is a lie that costs the next author a wait or a redo.** The card
`sync-rxdb-row-visibility-rls` (B2) MERGED **2026-08-01** (`bbbfc64`; roadmap flipped
DONE at `914536c`). The genuinely-open precondition is the **cutover** — no page
calls `startHQReplication`, so setting `HQ_SYNC_REST_URL` today would start a
replication nothing reads. Where a retired gate's underlying safety point still
holds, it is **restated against the cutover / substrate-RLS-must-be-present**, not
merely deleted.

## Shared files touched (E-KR3 — full site list, so "0 remaining" is checked)

### `workflows.html` (precached → forces `sw.js` regen) — TWO sites
| Line (pre) | Was | Now |
|---|---|---|
| **~329** (in the `bootstrap.js`-mount head comment) | *"…the /sync door answers 503 until HQ_SYNC_REST_URL is set, which must not happen **until row-visibility RLS lands**."* | Restated: the door answers 503 until `HQ_SYNC_REST_URL` is set; that must not happen until the **cutover** wires a caller AND the deploy's substrate carries the RLS policies (that card merged 2026-08-01; the open precondition is the cutover, which no page has triggered). |
| **~3946** (OVERWRITTEN-ANSWERS mount block; **spells the slug**) | *"…the /sync door answers 503 by design until `sync-rxdb-row-visibility-rls` lands. So in production today there are zero records…"* | Restated: 503 by design because `HQ_SYNC_REST_URL` is unset in every environment and nothing calls `startHQReplication` yet; `sync-hard-cutover` is the card that switches the producer on. (Slug removed from the gate; the RLS card is noted as already merged.) |

### `sync-rxdb/conflict-notice-ui.js` — ONE site
| Line (pre) | Was | Now |
|---|---|---|
| **~26** | *"…the /sync door answers 503 by design **until row-visibility RLS lands**."* | Restated against the cutover: 503 because `HQ_SYNC_REST_URL` is unset everywhere and no page starts replication; the RLS card merged 2026-08-01. |

### `tests/states-sync-rxdb-conflict-notice.spec.js` — ONE site
| Line (pre) | Was | Now |
|---|---|---|
| **~30** (header comment explaining why the store is seeded) | *"…the /sync door answers 503 **until row-visibility RLS lands**."* | Restated: 503 because `HQ_SYNC_REST_URL` is unset everywhere and replication is deliberately not started; `sync-hard-cutover` switches the producer on. |

### `sync-schema/sql/0001_sync_tables.sql` — ONE site (**fifth site, found beyond the four B-140 named** — same defect, slate-ID phrasing)
| Line (pre) | Was | Now |
|---|---|---|
| **~31-38** | *"`HQ_SYNC_REST_URL` must not be set in any deploy **until B2 lands**."* (B2 = the RLS card, which merged) | Restated: must not be set until the cutover wires a caller AND the deploy's substrate carries the RLS policies; the deny-all-until-policies point is preserved, the milestone corrected. Named beyond the four because it gates on the same merged card via its slate ID "B2", not the slug/paraphrase the broadened test keys on. `tests/sync-schema.spec.js:296` asserts only `not create policy` on this file — the comment text is unguarded, so this edit breaks nothing. |

### `backend/internal/sync/proxy.go` — B-18, comment-only (no code change)
Three honesty corrections, all in comments:
1. **ACTIVATION-ORDER banner (~78-92):** *"DO NOT SET HQ_SYNC_REST_URL … UNTIL ROW-VISIBILITY RLS LANDS"* → the RLS **card** landed; the true deploy-time safety is that the **substrate must carry RLS policies AND a caller must exist (cutover)** before the door opens. The concrete-consequence paragraph (a set var on a substrate with no policies = every crew member read+write) is kept — it is still true and is the reason the substrate-RLS precondition matters.
2. **B-18(a) — RawPath comment (~258-264):** verified against the code. The comment describes `URL.EscapedPath()` re-deriving the wire path from the decoded `Path`; the code does `out.URL.Path = path` and never assigns `out.URL.RawPath`. `r.Clone` copies the inbound RawPath, so the comment's "RawPath is cleared" is **not** what the code does — corrected to state the truth (RawPath is left as the clone's copy and `EscapedPath()` falls back to re-escaping the decoded Path because the stale RawPath does not decode to it), removing the false "cleared" claim that B-18(a) flagged as the refactor hazard.
3. **B-18(b) — rejection log (~204-205):** logs the rejected path with `r.URL.EscapedPath()`, the one function this card proved launders `%2f`. Added `r.URL.RequestURI()` alongside (per B-18(b) lead: "log `r.RequestURI` … alongside") so the raw pre-decode bytes survive in the log; the traversal check itself is unchanged.

> **B-18(c)** (`backend/cmd/server/main.go:436-438` all-clear) is **out of this card's
> footprint** (cmd/server, not sync). Its own lead says "fold into whichever card next
> touches route registration" — this card touches no route registration. Left for that
> card; noted here so it is not lost.

### `tests/repo-hygiene.spec.js` — case 3 broadened (the only executable change; RED-FIRST)
Was: scans `sync-rxdb/*.js` only, for the literal string `sync-rxdb-row-visibility-rls`.
Now: builds the set of DONE card slugs from **all** roadmaps (current `roadmap.md` +
archived `reference/roadmap-*.md` — the archived 08-05 roadmap is where the RLS card
carries its DONE status), scans the **whole live source tree** (top-level `*.html`/`*.js`
minus `workbox-*`/`sw.js`, plus `backend/ sync-rxdb/ sync-schema/ tests/ scripts/ lib/`,
excluding `node_modules`/`vendor`/frozen `.night-crew/` artifacts), and asserts NO live
comment names a DONE slug **or** the "row-visibility RLS" milestone as a future
precondition (future-precondition phrasing `until|before|once|when … lands|ships|merges`
co-occurring within a small line window with a DONE slug or the milestone name). The old
literal-string check is kept as a **subset** assertion so the narrow guarantee is not lost.

### `sw.js` — REGENERATED by `task sw`
Only the `workflows.html` precache **revision** changes (its comment edit changes its
content hash). Precache count stays **31**. The workbox runtime chunk hash is UNCHANGED
(node_modules carries `workbox-build@7.3.0`, matching the committed sw.js). 🛑 Orchestrator
reconciles `sw.js` at closeout after its post-Card-7 `task sw`; Card 11
(`deploy-hygiene-honesty`) touched `build-sw.js` but NOT `sw.js`.

## What must survive
- Every code path byte-identical (proxy.go traversal check, the rewrite, the log's
  `reason`/`user_id`/existing `path` field; the four SQL tables' deny-all state; the
  conflict-notice render seam).
- `tests/repo-hygiene.spec.js` case 3's NARROW guarantee (no `sync-rxdb/*.js` spells the
  slug) — kept as a subset assertion inside the broadened test.
- The three tests already green on this tree: the case-3 file's other two cases (NUL byte,
  night-crew.toml roll-call) untouched.

## What is safe to drop / not carried
- The literal-slug-only scope of case 3 (superseded by the fact-shaped scan, which strictly
  contains it).
- B-18(c) main.go warning — explicitly deferred (out of footprint; see above).
- The stale "until … lands" milestone in all five retired sites — the caveat's safety point
  is preserved via the cutover/substrate-RLS restatement wherever it still holds.

## Gate posture
- **G1** (backend `go build ./... && go vet ./...`): green — proxy.go change is comment-only.
- **G4**: `sw.js` regenerated, precache **31** before/after, workbox chunk UNCHANGED, version
  parity holds. `workflows.html` committed BEFORE regen (build-sw.js reads git HEAD).
- **G2** (confined Playwright subset per footprint + sync-rxdb rider): `workflows`,
  `persistence`, `repo-hygiene`, `sw-manifest`, `states-sync-rxdb-conflict-notice`,
  `sync-rxdb-conflict-notice`. Baseline reds on this tree: **B-174** (sw-api-cache-partition)
  + **B-176** (workflows DBL-05).
- **RF**: comment retirements are `n/a — no code change`. Case-3 broadening is red-first:
  re-insert a paraphrased gate naming a DONE card (no slug) into a tree file → broadened test
  CATCHES it (RED); remove → GREEN. Proves the guard is fact-shaped, not spelling-shaped.

## B-167 disposition (each of a–f)
Discharged/converted as checklist items — stated in the closeout/return. None is standalone
work; (a)(c)(e)(f) are precision/coverage notes folded forward, (b)(d) are naming/precision
notes. Full per-item disposition in the return.
