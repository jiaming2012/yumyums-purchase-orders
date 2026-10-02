# Merge intent — Card H2 · `campaigns-tab-ui`

Run `20261002` · branch `card/h2-campaigns-tab-ui` off `overnight-20261002` at `94ad3b3`
Wave 1, track B. Builds against the REAL endpoints card H1 merged at `40b5846`.

## Shared files touched

| File | Why |
|---|---|
| `marketing.html` | **Section `#s2` ONLY** (see below). Its `<style>` and its `<script src="marketing/campaigns.js" defer>` live INSIDE the `#s2` wrapper, exactly so a card owning another section of this file cannot conflict with them. |
| `sw.js` | Committed artifact, regenerated after the implementation commit (B-13 — `build-sw.js` reads git HEAD). Precache **48 → 49**: the one new module `marketing/campaigns.js`. No `globPatterns` change was needed — `marketing/*.js` already matches it, and `backend/Dockerfile` already copies `marketing/`. |
| `night-crew.toml` | **Roll-call comment only** in the MARKETING seam block, naming the two new specs the existing `marketing` token now selects. **No new key, no new token** (a new key/token is a PARK, operator-only). `tests/repo-hygiene.spec.js`'s machine-readable roll-call guard covers the OPERATIONS tokens, not this block, and neither new spec name contains an Operations token — its `expect(actual.length).toBe(11)` is unmoved. |
| `.night-crew/knowledge/roadmap.md` | H2's card line flips PLANNED → LANDED. Every card edits this file; expect a conflict and take both sides. |
| `.night-crew/runs/2026-10-02-autonomous/merge-intents/h2-campaigns-tab-ui.md` | This note. |
| `.night-crew/runs/2026-10-02-autonomous/logs/h2/*` | This card's gate logs. Nothing else under the run directory is written. |

New files, owned outright by this card, no merge risk: `marketing/campaigns.js`,
`tests/marketing-campaigns.spec.js`, `tests/states-marketing-campaigns.spec.js`.

**Nothing else.** No backend file, no migration, no `main.go`, no `go.mod`, no
`build-sw.js`, no `backend/Dockerfile`, no other page.

## What must survive any merge

- 🛑 **This card owns `marketing.html` section `#s2` and ONLY `#s2`.** Card 5
  (`stats-tab-ui`, H4) owns `#s4` and Card 6 (`subscribers-tab`, H5) owns `#s3` of the
  same file tonight, and **they must not lose theirs**. At merge: take every side's own
  section. The shared head, the `<style>` block in `<head>`, the `.tabs` bar, the inline
  `show()`, the auth probe, `#s1` (Scan) and the trailing `<script>` tags are **untouched
  by this card** — a diff of this branch that shows a change to any of them is a merge
  artifact, not this card's intent.
- **`#mc-root`'s contract**, which both new specs and the states screenshots read:
  `data-view` ∈ `list|create|ready|detail|code` and `data-status` ∈
  `idle|loading|ok|empty|error|offline|locked` on `#mc-root`; the five view shells
  `#mc-view-{list,create,ready,detail,code}`; `#mc-banner`; and the ids
  `#mc-new`, `#mc-refresh`, `#mc-period`, `#mc-list`, `#mc-empty`, `#mc-locked`,
  `#mc-chips`, `#mc-preview`, `#mc-submit`, `#mc-ready-head`, `#mc-ready-list`,
  `#mc-money`, `#mc-codes`, `#mc-qr`, `#mc-payload`, `#mc-share`, `#mc-save`,
  `#mc-copy`, `#mc-print`, `#mc-repoint`, `#mc-pause-code`, `#mc-pause-campaign`.
- **Every class this card introduces is `mc-`-prefixed and every selector it reads is
  under `#mc-root`.** That is what makes `#s2`'s `<style>` safe to live inside a `<div>`
  next to two other cards' sections.
- **ONE delegated `click` + ONE delegated `input` listener on `#mc-root`**, routed by
  `data-action` / `data-field` (the repo convention). No inline `onclick` anywhere in
  `#s2`.
- **The `money` block is RENDERED, never computed.** H1 ships the zero shape with every
  key present; `per_dollar` and `avg_order_cents_*` are `null`-not-absent on purpose and
  render as `—`. When card H3b lands the arithmetic the strip must not need a change.
- **`48 → 49`** as the precache count, and `sw.js` committed in the same change set as
  the `marketing.html` edit.

## What is safe to drop

- The **copy** on any Locked / offline / empty / error line, and the crew-language map in
  `createErrorText()`. If a later card re-words these, the later card wins — the specs
  assert the strings that are contracts (`managers_only`, `Managers only`,
  `Last synced`, `N codes ready`, `Per $1 —`), not the prose around them.
- `syncSubmitLabel()` — the one deliberate exception to state-first rendering (it only
  toggles `#mc-submit`'s `disabled` so a keystroke does not re-render the create sheet
  and move the caret). A later refactor that renders without losing the caret should
  delete it.
- The `#tab=2` deep link. It exists because the states spec wants a one-hop entry and
  because `#tab=N` is the repo's idiom; it calls the page's own `show()` and changes
  nothing. If `marketing.html` later adopts `tab.js` properly, drop these three lines.
- The keyboard (`Enter`/`Space`) parity listener on `role="button"` rows.

## Red-first

The red is **structural and observed**, not asserted in prose: `#s2` is still the "Soon"
placeholder on the base tree and `marketing/campaigns.js` does not exist, so every
done_when row times out waiting for `#mc-root`. Captured BEFORE any implementation file
was committed, at
`.night-crew/runs/2026-10-02-autonomous/logs/h2/RF-red-first.log`.

Observed, at `HEAD=94ad3b30fc64b6df6b16dd9907774f00b45d3ac5`, with
`TEST_PORT=8521 TEST_DB_NAME=hq_test_e2e_h2_20261002 DB_PORT=5434`:

```
$ grep -c mc-root marketing.html ; ls marketing/campaigns.js
0
ls: cannot access 'marketing/campaigns.js': No such file or directory

$ npx bddgen && npx playwright test tests/marketing-campaigns.spec.js --retries=0
  6 failed
    [chromium] › tests/marketing-campaigns.spec.js › [MC-01] … [MC-05]
    Error: expect(locator).toBeVisible() failed
    Locator: locator('#mc-root')
    Expected: visible
    Received: <element(s) not found>
      - waiting for locator('#mc-root')
TEST_EXIT=1
```

The server log in that same run is the other half of the honesty: the REAL
`POST /api/v1/marketing/campaigns` answered `201` throughout (`marketing: campaign
projection failed; projected_at left NULL … projection not configured`), so the red is
the missing UI and nothing else. Green-after: all six pass — `logs/h2/G2-playwright-subset.log`.

## State Enumeration rows — which ride a FIXTURE and which hit the REAL endpoint

Binding for G6. `tests/states-marketing-campaigns.spec.js`, 8 rows:

| Row | Forced by | REAL endpoint? |
|---|---|---|
| **empty** | `page.route` → `{"campaigns":[]}` | **FIXTURE.** The e2e database is shared across the suite and reset once at `webServer` start; `tests/marketing-campaigns.spec.js` sorts before this file and creates campaigns, so "no campaigns exist" is not a condition this spec can produce on demand. Stated rather than faked green. |
| **loading** | `page.route` with a 1.5 s delay in front of the real body | **FIXTURE.** A real endpoint cannot be made slow on demand. |
| **error** | `page.route` → `500 {"error":"boom"}` | **FIXTURE.** Same reason. |
| **success** | real `POST /campaigns` then the real `GET /campaigns` | **REAL.** |
| **locked (403 managers_only)** | a real invited `team_member`, logged in | **REAL.** The spec reads the envelope from that session and asserts `403` / `error == "managers_only"` before it asserts the pixel. |
| **offline (last synced + disabled create)** | real success, then `context.setOffline(true)` and a real refresh | **REAL data, real browser offline.** No route mocked — the fetch is refused the way a truck with no bars refuses it. |
| **not projected (pill)** | a real create | **REAL.** `HQ_SYNC_REST_URL` is unset on the test stack, so H1's handler genuinely returns `projected_at: null` + `warnings:["not_projected"]` (the run log above shows the server saying so). This row needed no fixture at all. |
| **long content** | a real create with a 94-char `offer_text` | **REAL.** |

So **5 of 8 rows hit the real endpoint**; 3 (empty, loading, error) ride `page.route`
fixtures because the condition is not producible on demand. The functional spec
`tests/marketing-campaigns.spec.js` mocks **no** marketing route at all.

## The `navigator.share` stub — the one permitted stub, named

`[MC-03a]` installs `navigator.share` and `navigator.canShare` via
`page.addInitScript` + `Object.defineProperty`, because **headless Chromium exposes
neither** (spike `web-share-files-enumerated`, chromium row: `share: undefined,
canShare: undefined`) and installing them is the only way to OBSERVE the call. The stub
records what it was handed; the test asserts `files.length === 1`,
`files[0].type === "image/png"`, `files[0].size > 0` and that the filename carries the
code's own 6-char short — i.e. that the real PNG bytes from
`GET /codes/{id}.png` reached `navigator.share`.

**The canShare-absent fallback is asserted UN-STUBBED beside it**, in `[MC-03b]` of the
same spec: nothing is installed, the test first asserts
`typeof navigator.canShare === "undefined"` (so it cannot pass by accident on a browser
that grew the API), then asserts `#mc-share` has **count 0** and that `#mc-save`
("Save PNG"), `#mc-copy` ("Copy link") and `#mc-print` ("Print") are rendered, with the
payload URL readable on the sheet. That is the branch every desktop browser and this
CI actually take. `GAP-H2-1` (webkit, the iOS proxy, unmeasured on this machine) is
untouched by this card: a different matrix changes the copy, not the structure.

No other stub, mock or fake exists in either spec.

## Decisions this card made that the slate left to it

- **Share detection is per-payload and at sheet-open time**, not at boot:
  `typeof navigator.share === 'function' && typeof navigator.canShare === 'function' &&
  navigator.canShare({files:[<real 4-byte image/png File>]}) === true`. `canShare`'s
  contract is about the payload, and a browser can expose `share()` while refusing
  files. Probing at open time is also what lets a test install the API before the
  sheet renders.
- **When the probe says no, there is no Share button at all** — `Save PNG` becomes the
  primary action and the sheet says "This browser cannot share a file." A button that
  cannot work is worse than its absence, and it makes `[MC-03b]`'s assertion
  unambiguous.
- **Offline is told from a 500 by the ABSENCE of an HTTP status**, not by
  `navigator.onLine` (which lies on captive portals). `fetch` rejecting with no
  `err.status` is offline; any status is the error row. The last-synced list is cached
  in `localStorage` **per period** (`hq.mc.list.v1:<period>`), because a 7-day list
  standing in for a 90-day one would be a quiet lie.
- **Create is disabled offline**, with a title saying why. A campaign mints codes
  server-side in one transaction; there is nothing honest to queue.
- **PNG size:** the sheet renders `?size=512` (one phone screen) and Share / Save fetch
  `?size=1024` (the print-usable default). Both are on H1's ladder `{256,512,1024,2048}`
  — an off-ladder size is a `400 bad_size`, not a silent snap, so the ladder is read as
  a contract.
- **"Re-point" is `PATCH /codes/{id}` with `landing` + `placement`** — §5's
  "re-point without reprint". The sheet says so in those words.
- **The Item field and the `inventory` grant.** The dish catalog lives behind
  `GET /api/v1/inventory/menu-items` (`toast.ListMenuItemsHandler`, `inventory` grant),
  and the design shows Item on the create sheet. A manager with `marketing` but not
  `inventory` is refused there, so the field stays VISIBLE and disabled with the line
  "Picking a dish needs Inventory access — this offer will count against any item."
  Not a silently missing control (UI-R3), and not an invented endpoint. `?since=2000-01-01`
  is passed because the handler defaults to the last 7 days and a dish that has not sold
  this week is still a dish you can run an offer on. **Noted as a deviation**, below.
- **Activation is OBSERVED, not hooked.** A `MutationObserver` on `#s2`'s `style`
  attribute boots the module the first time the section becomes visible and re-reads the
  list on every return. The inline `show()` is NOT wrapped, patched or edited — this
  card does not own it.

## The surface that makes D-1 reachable

`DECISIONS-NEEDED.md` **D-1** — migration 0083's `menu_items` FKs carry no `ON DELETE`,
so a campaign referencing a dish makes the Inventory › Recipes **dish merge** fail with
an opaque 500. **This card is the surface that makes it reachable**: picking an Item on
the create sheet is now a routine two-tap action, where before D-1 required a hand-rolled
API call. Not fixed and not designed around here (it is parked at top severity, the
operator's) — Item selection is built exactly as the design shows. Flagging the
connection so triage sees it: the fix lands wherever D-1 is resolved, and this card's
`#mc-f-item` is the thing that will start producing the referencing rows.

## Deviations from the design of record

1. **The *Current Campaigns 1–8* canvases themselves were not reachable from this
   environment** (the Claude Design project is a hosted URL; this worktree has no
   network path to it). Built from the binding in-repo decomposition of those pages — the
   slate's Card 2 scope paragraph, roadmap H2, and handoff §2/§5/§6 — which enumerate the
   five screens and every control by name. Anything the canvases show that those three
   do not name is therefore unbuilt rather than deliberately omitted.
2. **The Item field degrades to "Any item" + "needs Inventory access"** for a manager
   without the `inventory` grant (above). The design shows Item unconditionally; serving
   it unconditionally would need either a second catalog read inside the marketing
   package (backend, another card's footprint and another card's merge) or an invented
   endpoint. Noted, not invented.

Nothing else. No new sheet, no new state, no new lifecycle row, no `night-crew.toml`
key or token, no app grant, nothing that sends.
