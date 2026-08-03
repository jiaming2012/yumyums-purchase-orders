# Merge intent — A6 · `app-version-badge`

- **Run:** `overnight-20260804` (stretch card)
- **Branch:** `card/a6-app-version-badge`
- **Base commit:** `0dcd8b4` — every diff below is measured against this, never against
  run-branch HEAD (T-32 decision 130a).
- **Closes:** D-KR2b's evidence method. Ledger T-33 **decision 133**.

---

## What this card does, in one line

`index.html` grows a discreet footer version line whose value is read from the **precached
`version.json`**, with the server's `/api/v1/health` `frontend_version` shown beside it **only as a
comparison** — so the app diagnoses its own staleness.

🛑 **Two qualifications on that sentence, both added in the fix round rather than left implicit.**

1. **"Diagnoses its own staleness" is stronger than the comparison half actually delivers.**
   `build-sw.js:441-447` registers `/\/api\//` as **NetworkFirst** with a 10s timeout and an
   `api-cache`, and `/api/v1/health` matches it. On an offline or flaky phone — routine on a food
   truck — the health response can be served **from that cache** once the timeout expires, so
   `#version-server` can show a **stale server number**, and a stale one equal to the device's
   yields a false `data-state="current"`. **The primary value is unaffected**, which is why this is
   minor and why the handler was deliberately **not** changed: the badge's own value comes from the
   precached `version.json` and never lies about *this device*, which is the defect the line exists
   to catch. But `current` is not proof of freshness. A comment now says so at the code site
   (`build-sw.js`, above `runtimeCaching`).
2. **Deviation from the slate: "beside" is conditional, not always.** The slate said show the
   cached value **beside** the server's. The implementation shows the server value **only on
   mismatch** — `#version-server` is empty when the two agree (`data-state="current"`), and carries
   `server vX.Y.Z — close and reopen to update` when they do not. That is defensible (a line that
   prints two identical numbers on every load is noise, and the line's job is to speak up when
   something is wrong) and it is **test-pinned** — test 4 asserts `#version-server` is empty at
   `current`, test 3 asserts it carries the server value at `stale`. Recording it as a **deviation**
   rather than leaving it implicit, because it was not what the slate asked for.

🛑 **The source is the design.** `version.json` ships *inside the bundle* and is served cache-first
by Workbox, so it reports what **this device** actually has — the only value capable of being stale.
`/api/v1/health`'s `frontend_version` is always current; a badge fed from it would print the right
number on a phone frozen on last week's bundle and **hide the defect the line exists to reveal**.
The health value never becomes the badge's own value, and there is **no API fallback**: if
`version.json` cannot be read the line reports `v—` / `data-state="unknown"` rather than lying.

---

## Red-first

**Named test:** `tests/version-badge.spec.js` — 5 tests, headline
`version line shows the value from version.json, NOT the value from /api/v1/health`.

**Captured RED against:** **`ba8719b`** — base `0dcd8b4` plus `tests/version-badge.spec.js` and
this note, with **no `index.html` change yet**. (The SHA is back-filled here; the section and the
red output were present in that commit as written.)

🛑 **The spec was CHANGED AFTER this capture, and the change shipped in the fix commit itself.**
`f5cd518` — the `index.html` commit — also added **19 lines to `tests/version-badge.spec.js`**
(`git show --stat f5cd518`: `index.html | 60 +`, `tests/version-badge.spec.js | 19 +`). Those 19
lines are the `GET /version.json` precondition and test 5's `data-state="current"` assertion, both
added in response to the 4/5 wrinkle recorded below. So the red quoted here was produced by a
**strictly earlier** version of the spec than the green quoted below it. The substance is already
in the "honest wrinkle" paragraph; it is stated **here**, inside the Red-first section, because a
red-first claim whose spec moved between the two captures is weaker than one whose spec did not,
and a reader should learn that from the section that makes the claim rather than three paragraphs
later. Neither addition weakens any assertion — both add coverage — and the four pre-existing
tests are byte-identical across the two captures.

**Red output (verbatim tail):**

```
  5 failed
    [chromium] › tests/version-badge.spec.js:70:1 › version line shows the value from version.json, NOT the value from /api/v1/health
    [chromium] › tests/version-badge.spec.js:103:1 › version line reports UNKNOWN rather than falling back to /api/v1/health when version.json is unavailable
    [chromium] › tests/version-badge.spec.js:123:1 › version line flags this device as STALE when the cached bundle differs from the server
    [chromium] › tests/version-badge.spec.js:145:1 › version line reads CURRENT when the cached bundle matches the server
    [chromium] › tests/version-badge.spec.js:164:1 › unmocked, the version line shows the real version.json value and the precache manifest carries that file
```

Every failure is `Error: element(s) not found` on `locator('#version-line')` /
`locator('#version-cached')` — i.e. the tests fail for the right reason (no such surface), not on a
harness fault. Note that test 5's `sw.js`-manifest assertion **already passed** at red; the missing
half was purely the display.

**Green after:** `f5cd518` (the `index.html` commit). All 5 green in the **full-suite** G2 leg:

```
  ✓  701 [chromium] › tests/version-badge.spec.js:70:1 › version line shows the value from version.json, NOT the value from /api/v1/health (545ms)
  ✓  702 [chromium] › tests/version-badge.spec.js:103:1 › version line reports UNKNOWN rather than falling back to /api/v1/health when version.json is unavailable (480ms)
  ✓  703 [chromium] › tests/version-badge.spec.js:123:1 › version line flags this device as STALE when the cached bundle differs from the server (508ms)
  ✓  704 [chromium] › tests/version-badge.spec.js:145:1 › version line reads CURRENT when the cached bundle matches the server (505ms)
  ✓  705 [chromium] › tests/version-badge.spec.js:164:1 › unmocked, the version line shows the real version.json value and the precache manifest carries that file (422ms)
```

🛑 **One honest wrinkle between red and green, recorded rather than smoothed over.** The first
post-implementation run was **4/5**, not 5/5: the unmocked test reported
`Expected: "v1.4.0" Received: "v—"` because `version.json` did not exist in this worktree. It is a
git-ignored build artifact and the bare `npx playwright test` the gate runs never generates it. The
app was behaving **correctly** (unknown reported as unknown, no API fallback); the *stack* was
missing a file. Filed as **B-92** and mitigated in-card with an explicit `GET /version.json`
precondition assertion whose failure message names the generator, so the next reader gets a setup
error instead of an apparent code defect.

🛑 **UPDATE — the fix round FIXED this rather than leaving it filed. B-92 is resolved (`e215ef3`).**
Mitigation was not enough: as it stood, this card handed **every future card leg** a red that had
nothing to do with its own change, on the exact `npx playwright test` path `night-crew.toml:33-34`
uses — partially undoing what A1 landed the same night. See the **Fix round** section at the end.

---

## How the test fails if someone reroutes the badge to `/api/v1/health`

This is the card's central guarantee, so it is worth stating the mechanism rather than asserting it.

The spec never lets the two sources agree. It serves them **deliberately different** values:

| source | stubbed value |
|---|---|
| `version.json` | `9.9.9-from-file` |
| `/api/v1/health` `frontend_version` | `0.0.1-from-api` |

- **Test 1** pins `#version-cached` to `v9.9.9-from-file` *by equality*, and separately asserts
  `#version-cached` does **not** contain `0.0.1-from-api`. A reroute makes the badge print the API
  string → equality fails.
- **Test 2** is the one that cannot be sneaked past: it **aborts** `version.json` while
  `/api/v1/health` still answers. There is then no legitimate version to display, so *any*
  implementation carrying an API fallback prints `0.0.1-from-api` and reds. The assertions are
  `data-state="unknown"`, `data-version` absent, and the line's full text not containing the API
  string.
- **Test 3** asserts the two numbers land in **different** elements (`#version-cached` = device,
  `#version-server` = comparison), so collapsing them into one API-fed value reds too.

A test that only asserted "a version renders" would pass on the forbidden implementation. These do
not.

## `serviceWorkers: 'block'` — how it was handled

`playwright.config.js` blocks service workers repo-wide (B-15) and **that line was not touched**.
The consequence is honest and stated in the spec's header: **no test in this file can watch Workbox
serve `version.json` from the precache** — under test the file is fetched over plain HTTP. So the
guarantee is split, and both halves are asserted:

- **(a)** this spec pins **the URL the page reads its version from**, and pins the **absence of an
  API fallback** when that URL is unavailable;
- **(b)** test 5 asserts `version.json` is present in the **committed `sw.js` precache manifest** —
  which is what makes (a)'s URL cache-first, device-local and therefore staleable at runtime.

Neither half alone is the property; together they are. This is the most likely place for this card
to prove less than it appears to, so it is written down rather than implied.

---

## Shared files touched — and why each

| file | why | what must survive any merge | safe to drop |
|---|---|---|---|
| `index.html` | the version line itself: `.verline` CSS block, the `#version-line` footer element after `.app`, and one IIFE at the end of the inline script. | **The IIFE's source of truth: `fetch('version.json')`, and the rule that `/api/v1/health` is read only *after* the cached value is already on screen and only into `#version-server`.** The 🛑 comment block above the IIFE — it is the whole reason the card exists. | the `.verline` CSS cosmetics (font size, colour, spacing) and the exact wording of the stale note. |
| `sw.js` | regenerated with `node build-sw.js` **after** the content commits, because `index.html` is precached and `build-sw.js` reads **git HEAD**, not the working tree (B-37). | the regenerated `index.html` revision hash. | nothing else — re-running `build-sw.js` after the merge is idempotent and expected, not a conflict. |
| `.night-crew/knowledge/roadmap.md` | the card's status flip, in the same change set (universal mechanic). | the `app-version-badge` bullet. | nothing here. |
| `tests/version-badge.spec.js` | **new file**, no merge surface. | all 5 tests, and specifically the differing-stub design described above — weakening the stubs to agree would silently retire the guarantee. Plus (fix round) test 5's `precachedUrls()` **manifest parse** — reverting it to a string match re-opens G6 finding 3. | nothing here. |
| `playwright.config.js` | **fix round.** `node scripts/write-version-json.js` added as the **second** link of `webServer.command` (B-92). 🛑 **gate-critical, shared by every card this run.** | the link itself, and its position **after** `node scripts/reset-e2e-db.js` — the reset is A1/B-76's fix and must stay first. Repo-wide `serviceWorkers: 'block'` (B-15) is untouched and must stay untouched. | the comment block's wording, not its content. |
| `build-sw.js` | **fix round.** `writeVersionJson()` extracted to `scripts/write-version-json.js` and now `require`d from there; plus a comment above `runtimeCaching` recording the NetworkFirst caveat. | the `require` — two generators, **one** payload definition. | the comment wording. |
| `scripts/write-version-json.js` | **new file**, no merge surface. Module + CLI, same shape as `scripts/reset-e2e-db.js`. | the repo-root path resolution (`__dirname/..`, not `cwd`) — it is called from two different working directories. The `console.error` banner (B-81: `webServer.stdout` defaults to `'ignore'`). | nothing here. |

**Precache count: 31 before, 31 after.** `version.json` and `index.html` were both already in the
manifest; this card adds **no asset**. A count that moves is the B-37 silent drop returning, and is
a finding, not an expectation.

**Version parity untouched.** `version.go` `Frontend` ≡ `package.json` ≡ `version.json` = **1.4.0**
at base and at HEAD. This card *displays* the version; it does not change it. No bump.

## Files outside the footprint

**`.night-crew/knowledge/BACKLOG.md`**, which carries **B-92** — filed at build time, **re-headlined
and marked RESOLVED** in the fix round. And, from the fix round, the three files the B-92 fix itself
needed: **`playwright.config.js`**, **`build-sw.js`**, **`scripts/write-version-json.js`** (new).

The full diff against `0dcd8b4` names exactly `index.html`, `sw.js`, `tests/version-badge.spec.js`,
`playwright.config.js`, `build-sw.js`, `scripts/write-version-json.js`,
`.night-crew/knowledge/roadmap.md`, `.night-crew/knowledge/BACKLOG.md` and this note.

No `backend/**`. No other `*.html`. No `night-crew.toml` change — `index.html` stays **undeclared in
`[e2e.seams]`**, which de-confines the card to the full suite by construction; that is the correct
behaviour and was paid, not worked around.

---

## Fix round — G6 `APPROVE-WITH-NOTES`, 2026-08-04

**Fix 1 — B-92 closed in-card (`e215ef3`).** The one that mattered: this card, as it stood,
introduced a regression *into the gate*. `version.json` is git-ignored (`.gitignore:13`);
`night-crew.toml:33-34` runs `npx playwright test` directly and `playwright.config.js`'s
`webServer` serves the bare worktree, so `GET /version.json` 404'd in any worktree where
`node build-sw.js` had never run — a red on every future card leg, unrelated to that leg's change,
on the exact path A1 landed to make trustworthy.

The generator now sits in `webServer.command`, the same place and for the same reason as A1's
database reset: it runs once, in the parent, before any test and before the server exists, and no
CLI argument can skip it, so the `subset` leg gets it too.

```
node scripts/reset-e2e-db.js && node scripts/write-version-json.js && cd backend && …
```

🛑 **Not `node build-sw.js`** — that reads git HEAD and rewrites `sw.js`, which would dirty the tree
mid-gate (B-37) and produce a spurious `sw.js` diff on every run. `writeVersionJson()` was
**extracted** from `build-sw.js` into `scripts/write-version-json.js` instead, so the payload keeps
**one** definition. Extraction was chosen over inlining a `node -e` in the config **on blast radius
grounds that cut the other way than raw diff size**: the `node -e` touches one file, but it creates
a *third*, differently-worded definition of a shipping artifact's payload — and B-92's own body
warns that the two existing generators already disagree on source (`package.json` vs `version.go`).
`build-sw.js` produces byte-identical output after the extraction, which is the check that the
larger diff bought nothing but tidiness.

**Fix 2 — test 5's precache assertion strengthened (`c56024b`).** `toMatch(/["']version\.json["']/)`
→ `expect(precachedUrls(sw)).toContain('version.json')`, parsing `url:"…"` per
`tests/sw-manifest.spec.js:17-22`. Measured against a synthetic `sw.js` where `version.json` appears
only in a `runtimeCaching` matcher and is **not** precached: the old assertion **PASSES** (guarantee
silently retired), the new one **FAILS**. This is the half that covers the gap `serviceWorkers:
'block'` leaves, so it should be the strong half.

**Fix 3 — B-92 re-headlined + resolved.** The old headline said "the Playwright gate path" — broader
than the body: the ephemeral env (`night-crew.toml:12-16` → `backend/Dockerfile:58-64`) **does**
generate the file, and in that mode `playwright.config.js:65` makes `webServer` undefined entirely.
Narrowed to the **in-worktree `npx playwright test`** path and marked resolved with `e215ef3`.

**Fix 4 — documentation.** The NetworkFirst caveat and the "beside vs only-on-mismatch" deviation,
both above; the Red-first disclosure, in that section.

### Evidence

| check | result |
|---|---|
| **fresh `git archive` of the branch, no `version.json`, pre-fix** | `1 failed, 4 passed` — `Error: GET /version.json returned 404` |
| **fresh `git archive` of the branch, no `version.json`, post-fix** | **`5 passed (14.5s)`** |
| A1's reset still fires | output line 1: `[WebServer] ── reset hq_test_e2e_a6fix on localhost:5433 ──`; line 3: `[WebServer] ── wrote version.json frontend=1.4.0 ──` — reset **first** |
| shared-harness check, second spec | `tests/db-isolation.spec.js` (A1's own) → `1 passed`, both banners present |
| `node build-sw.js` ×2 | **31** precached both times; `sw.js` byte-unchanged, tree clean |
| three-way parity | `version.go` `Frontend` ≡ `package.json` ≡ `version.json` = **1.4.0**, untouched |

Isolation for every leg above: `TEST_PORT=8226`, `TEST_DB_NAME=hq_test_e2e_a6fix`,
`HQ_RLS_TEST_DB=hq_rls_a6fix_0804` — never the bare default. Nothing ran alongside.

### Not fixed, and named

**B-92's source disagreement stands.** `scripts/write-version-json.js` reads `package.json`;
`backend/Dockerfile:61` reads `version.go` `Frontend`. Harmless only while the three-way parity
holds, and it would surface as dev and prod showing different numbers the moment it does not. That
half of B-92's body is left standing in the entry as next-milestone work; this round closed the
*artifact-missing* half, which is the half that was breaking the gate.

## Deliberately left undone

- **No `/api/v1/health` change.** The endpoint already returns `frontend_version`; nothing was
  needed.
- **The line is on `index.html` only.** The launcher is the page every user passes through, which
  is the card's stated scope. Putting it on all seven precached pages was not attempted.
- **D-KR2a / D-KR2b's actual evidence collection is attended and not this card's** — it is the
  ~15-minute `task prod:deploy` → `task version` → read-the-line-on-a-returning-client ritual after
  this merges.
