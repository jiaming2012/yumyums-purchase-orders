# Merge intent — Wave 0 · `repo-hygiene-preconditions`

- **Run:** `overnight-20260806`
- **Branch:** `card/w0-repo-hygiene`
- **Base commit:** `14e2a01` — every diff claim in this note is measured against `14e2a01`,
  never against run-branch HEAD (T-32 decision 130a).
- **Wave:** 0. Runs first and alone; every other card in the night rebases onto this.
- **Closes:** B-70 (the NUL byte). Corrects two false written claims that are not filed as bugs.

Written **before** implementing (§15ad.65). Updated in place only for facts that changed.

---

## Shared files touched — each line says why

| File | Why this card touches it |
|---|---|
| `sync-rxdb/client.js` | **Owned.** Defect (a): the single raw `U+0000` at byte offset 50850, inside the `scopeFingerprint` template literal, is replaced by the escape sequence `\0`. Same byte at runtime; the source becomes 7-bit clean so `file(1)` stops calling it `data` and GNU grep stops switching to binary mode. **Also** defect (c)'s second site — `:1108-1109` carries the same stale "until `sync-rxdb-row-visibility-rls` lands" gating claim as `bootstrap.js:22`, and it is corrected in the same change set (see "beyond the stated footprint" below). |
| `sync-rxdb/bootstrap.js` | **Owned.** Defect (c): the `:22` banner gating activation on `sync-rxdb-row-visibility-rls` — a card that merged at `bbbfc64` / `bec06f6` (run `overnight-20260801`; roadmap flipped it DONE at `914536c`). The banner is restated against the preconditions that are *actually* still open. |
| `night-crew.toml` | **Owned.** Defect (b): the comment at `:50-58` claims the four Operations tokens select "exactly workflows / persistence / sync / repro-cut-task .spec.js and nothing else — re-verified at landing". They select **9** files. Comment corrected; the token **values** are unchanged. |
| `tests/repo-hygiene.spec.js` | **New. The red-first test.** Beyond the stated footprint — see below. |
| `sw.js` | **Consequence, not a choice.** Both edited `sync-rxdb/*.js` files are precached assets, so their content hashes moved and the committed manifest had to be regenerated (`node build-sw.js`, commit `672fc01`, run **after** the three fix commits because `build-sw.js` reads git HEAD, not the working tree). B-13: an `sw.js` you did not commit does not deploy. Exactly **2** revisions change — `sync-rxdb/client.js` `9e37a8c7…`→`65174ce3…` and `sync-rxdb/bootstrap.js` `23997981…`→`6aaded00…`; **0 entries added, 0 removed, count 31 → 31**, so this is not B-37's silent drop. |
| `.night-crew/runs/2026-08-06-autonomous/merge-intents/w0-repo-hygiene.md` | This note. |

**Beyond the slate's stated footprint** (§15ad.65 — planning information, not a fence):

1. **`tests/repo-hygiene.spec.js` (new).** B-70's own lead says the guarantee "should not depend
   on remembering", and the roadmap's ban on `done_when: "grep returns nothing"` is only liftable
   if something keeps the file readable. All three defects are one-line facts a spec can assert,
   so one spec guards all three. Its name shares no substring with any `[e2e.seams]` token
   (`workflows` / `persistence` / `sync` / `repro-cut-task` / `inventory` / `recipes` /
   `onboarding` / `users` / `purchasing`), so it does not perturb defect (b)'s arithmetic.
   🛑 **It raises the suite's test count.** Judge the full-suite figure against baseline + the new
   cases, not against 785 flat.
2. **`sync-rxdb/client.js:1108-1109`.** The card names `bootstrap.js:22` as the site of defect (c).
   There is a **second** site, in a file this card already owns, carrying the same false claim.
   Fixing one and leaving the other would leave the acceptance criterion satisfiable while the
   defect survives. Recorded here rather than parked: it is the same correction, in-footprint.

**`.gitattributes` is NOT touched, and that is deliberate.** B-70's lead offered two independent
fixes: (a) the `\0` escape, and (b) `-text` / `grep -a` as a belt-and-braces guard. With (a)
applied the file is 7-bit clean and a `-text` attribute would assert a property no longer at risk
while silently masking a *future* reintroduction — which is exactly what `tests/repo-hygiene.spec.js`
is there to catch loudly instead. One guard, and it fails loud.

**Nothing else.** No Go code. No `workflows.html`. No `build-sw.js`, no `package.json`, no
`version.go` — neither semver moves, and frontend stays **1.4.0**.

🛑 **`sw.js` IS in this card's diff, and an earlier revision of this note denied it.** The sentence
above read "no `build-sw.js`, no `sw.js` … this card ships no frontend asset change, so precache
stays 31", which contradicted commit `672fc01` sitting in the same branch. The **edit** was correct
and required — the card edits two precached assets, and B-13 is unambiguous that an uncommitted
`sw.js` does not deploy. The **declaration** was the defect (§15ad.65: editing outside the stated
footprint is not a breach; failing to declare it is). Now declared in the table above. The precache
count claim survives intact — it really is **31 before and 31 after**, with exactly two revisions
moved and nothing added or removed.

---

## What MUST survive any merge

1. **`sync-rxdb/client.js` contains zero NUL bytes.** This is the whole point of Wave 0. Any merge
   that reintroduces the raw `U+0000` — including a "resolve by taking theirs" on that hunk —
   silently re-arms B-70 and makes every downstream card's grep evidence unreliable *in the passing
   direction*. `tests/repo-hygiene.spec.js` will red on it; do not skip that test to land a merge.
2. **The escape must stay `\0` inside the template literal, not a rewrite of the delimiter.** The
   fingerprint input is `${scopeIdentity(s)}\0${serialized}`. Changing the delimiter to any
   printable character changes every fingerprint the function has ever produced and breaks the
   "cannot occur in either operand" property the technique rests on. The byte does not move;
   only its source spelling does.
3. **`night-crew.toml`'s corrected comment.** It is the file that decides gate cost. The next
   author who reads "exactly … and nothing else" will under-budget a night by ~20 minutes per
   confined card. Values unchanged — only the prose.
4. **No `sync-rxdb-row-visibility-rls` gating language in `sync-rxdb/`.** Both sites, not one.
5. **This note.**

## What is safe to drop

- Any wording in the corrected `bootstrap.js` / `client.js` banners, provided the *stale
  precondition* does not come back. A later card that rewrites those comment blocks wholesale is
  free to; it must not restore a gate on a merged card.
- The exact prose of the `night-crew.toml` comment, provided the file-count claim stays true.
- This card's scratch logs under `$SCRATCH`. Not committed.

## Nothing here (stated explicitly)

- **Schema / config-key changes:** nothing here. Only `[e2e.seams]` prose is edited; no key added,
  removed or renamed. (A new key or a schema change is the PARK condition and did not arise.)
- **Backend changes:** nothing here.
- **Migrations:** nothing here.
- **Contract changes** (`docs/contracts/`): nothing here.
- **Version bumps:** nothing here.
- **Parked items:** nothing here.

---

## 🛑 For A5 `shipped-bug-sweep` (the stretch card) — it rebases onto this

A5 fixes **B-89** at `sync-rxdb/bootstrap.js:66-68` (`JSON.parse(raw)` →
`Array.isArray(apps) ? … : []`, which returns `[]` on every real client because `index.html:241`
writes an **object**). That is a **different region of the same file** from this card's edit.

- **This card touches only the header comment block, `bootstrap.js:1-50`.** No executable
  statement in `bootstrap.js` changes. `cachedGrantSlugs()` and its call site at `:100` are
  untouched, so A5's diff should apply without a textual conflict — but the line numbers in A5's
  own planning notes are taken against `14e2a01` and **will shift** if the corrected banner is a
  different number of lines than the one it replaces. A5 must locate its hunk by symbol
  (`cachedGrantSlugs`), not by line number.
- **A5 must preserve:** the absence of `sync-rxdb-row-visibility-rls` gating language. If A5's
  rebase resolves a header conflict by taking `14e2a01`'s version, it silently reverts defect (c).
- **A5 may freely rewrite** anything below the header. This card claims none of it.

## For every other card in the night

- Cards touching `sync-rxdb/client.js` (Activity 3 / `sync-hard-cutover` and the spikes) inherit a
  file that `grep` can finally read.

  🛑 **Line numbers in `client.js` DO move. Locate by symbol, not by line number.** An earlier
  revision of this note claimed they do not; that claim was false and is corrected here, because
  `sync-hard-cutover` edits this file and is not in tonight's slate — it will read this on a later
  night, against a tree where the shift has already happened.

  Measured, `14e2a01` → this card's tip:

  | Fact | Value |
  |---|---|
  | File length | **1217 → 1236 lines** (+19) |
  | Hunk 1 | `@@ -1051 +1051,13 @@` — the NUL fix is not a one-byte→two-byte edit; the single line 1051 becomes **13** lines — a 12-line comment block explaining why the byte is spelled `\0`, then the `fingerprint:` line itself |
  | Hunk 2 | `@@ -1106,6 +1118,13 @@` — the stale-gate correction replaces base lines 1106–1111 (6 lines) with 13 |
  | Shift, base lines **≥ 1052** | **+12** |
  | Shift, base lines **≥ 1112** | **+19** |

  Confirmed symbol moves (`grep -an`, binary-safe — plain `grep` could not read the base file,
  which is B-70 itself):

  | Symbol | `14e2a01` | tip | Δ |
  |---|---|---|---|
  | `createHQSyncDatabase` | `1077` | **`1089`** | +12 |
  | `startHQReplication` | `1132` | **`1151`** | +19 |

  **Any card quoting a `client.js` line number above 1051 from the slate is now wrong by 12 or 19.**
  Re-resolve every such citation by symbol before trusting it. Line numbers ≤ 1051 are unchanged.
- Cards quoting a confined-subset cost from `night-crew.toml` should re-read it after this lands.
  The tokens did not change; what they were *claimed* to select did.

---

## Gate results — appended after the fact, measured not inherited

| Gate | Result |
|---|---|
| **G1** | `go build ./...` **exit 0**, `go vet ./...` **exit 0**, both from `backend/`. Zero output lines from either. |
| **G2 (Go)** | `go test -p 1 -count=1 ./...` **exit 0**. **9** packages ok, **439** `=== RUN` lines, **0** FAIL, 2 SKIP (`TestProxyLive_RealtimeUpgrade`, `TestProxyLive_RESTRequest` — live substrate). `internal/workflow` ran **35**, as the ladder requires. `internal/sync` ran **142**, of which `TestRowVisibilityRLS` contributed **59** subtests; `HQ_SYNC_SUBSTRATE_OPTIONAL` was **unset**. Isolated DB `hq_test_go_w0_0806`. |
| **G2 (Playwright)** | `npx bddgen` (exit 0) then `npx playwright test --retries=0`, run **alone**. **exit 0**, **791 passed / 6 skipped / 0 failed**, **21m 43s** wall. **Exactly ONE summary block**, counted with `grep -c "passed ("` over the complete 4,626-line log — not a tail. Isolation: `TEST_PORT=3106`, `TEST_DB_NAME=hq_test_w0_0806`, `HQ_RLS_TEST_DB=hq_rls_w0_0806`. |
| **G3** | N/A — preflight `openspec: absent`; no scaffolding created. |
| **G4** | `node build-sw.js` exit 0, **31** files precached, reachability 18 parsed / 30 resolved / 0 outside. Run a second time after committing `sw.js`: **tree clean**, idempotent. Frontend 1.4.0, unchanged. |

🛑 **The slate's inherited full-suite baseline of "785 tests" is stale by 8.** `--list` reports
**797 tests in 29 files** on this branch, of which this card's `tests/repo-hygiene.spec.js`
contributes **4** — so the pre-card tree was **793 in 28 files**, not 785. The 785 figure came from
run `20260803` and the slate said plainly it was inherited rather than re-measured; it now is
re-measured. Later cards tonight should judge against **793 + their own additions**, not 785.

**Wall-clock note for the orchestrator's queue:** 21.6m, against the slate's ~24.2m estimate, on a
box running nothing else.
