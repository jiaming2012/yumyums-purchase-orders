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

**Captured RED against:** the tree of **this very commit** — base `0dcd8b4` plus
`tests/version-badge.spec.js` and this note, with **no `index.html` change yet**. (SHA back-filled
in the closing docs commit of this branch.)

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

**Green after:** _(back-filled in the closing docs commit — see the card report.)_

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
| `tests/version-badge.spec.js` | **new file**, no merge surface. | all 5 tests, and specifically the differing-stub design described above — weakening the stubs to agree would silently retire the guarantee. | nothing here. |

**Precache count: 31 before, 31 after.** `version.json` and `index.html` were both already in the
manifest; this card adds **no asset**. A count that moves is the B-37 silent drop returning, and is
a finding, not an expectation.

**Version parity untouched.** `version.go` `Frontend` ≡ `package.json` ≡ `version.json` = **1.4.0**
at base and at HEAD. This card *displays* the version; it does not change it. No bump.

## Files outside the footprint

**Nothing here.** The diff against `0dcd8b4` names exactly `index.html`, `sw.js`,
`tests/version-badge.spec.js`, `.night-crew/knowledge/roadmap.md` and this note.

No `backend/**`. No other `*.html`. No `night-crew.toml` change — `index.html` stays **undeclared in
`[e2e.seams]`**, which de-confines the card to the full suite by construction; that is the correct
behaviour and was paid, not worked around.

## Deliberately left undone

- **No `/api/v1/health` change.** The endpoint already returns `frontend_version`; nothing was
  needed.
- **The line is on `index.html` only.** The launcher is the page every user passes through, which
  is the card's stated scope. Putting it on all seven precached pages was not attempted.
- **D-KR2a / D-KR2b's actual evidence collection is attended and not this card's** — it is the
  ~15-minute `task prod:deploy` → `task version` → read-the-line-on-a-returning-client ritual after
  this merges.
