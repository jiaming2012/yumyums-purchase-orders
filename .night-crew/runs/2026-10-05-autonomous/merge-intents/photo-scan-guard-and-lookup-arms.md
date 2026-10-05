# Merge intent — card `photo-scan-guard-and-lookup-arms` (Card 2, roadmap J1, run `20261005`)

Branch `card/j1-photo-scan-guard-and-lookup-arms`, cut from the run branch `overnight-20261005`
at `3bd6a9b` (Card 1's merge is already in it). Track A, alone. BACKLOG **B-475** (a photo
picked while the scanner waits on the server strands the current customer's offer with no
submit control) and **B-476** (four scan-time-check breakages leave every test green); ledger
T-64 decision 205.

Contract: `.night-crew/knowledge/reference/slate-20261005.md` §"Card 2". Goal ledger:
`.night-crew/knowledge/spikes/activity-j-scanner-and-backend-guards-triage-20261003-follow-ups/photo-scan-guard-and-lookup-arms.md`
(2/2 passed; build-facts binding).

**What a person sees differ.** Today: the scanner is waiting on the server for customer A's
code, someone picks a photo of customer B's code, and when the server answers, A's offer is on
screen with no order-number field, no submit button, and a "Finish the current customer
first" sheet over it. After this card: the photo picked mid-wait is simply not read — the
screen keeps saying "Checking with the server…", A's offer arrives with its order-number field
and submit button, and no sheet appears. A photo picked when nothing is in flight decodes
exactly as before.

---

## Shared files touched, and why each

| File | In footprint? | Why |
|---|---|---|
| `marketing/scan-page.js` | yes (solo owner) | `onFilePicked` takes the camera path's `decodeBusy` guard around decode + `doScan`; the input is still reset in `finally` (also when the pick is refused, so the same photo can be picked again once the wait ends). **Placement decision (the night's, stated):** the programmatic entry `window.MarketingScan.scanText` — which was bare `doScan` — goes through the same flag, because the spike's recipe (and therefore the re-homed `[PS-01]`, and G6's drive) starts code A's scan through `scanText`; with the flag on the photo path alone, a `scanText`-started wait never sets `decodeBusy` and the photo would still strand it. One flag, three entry points (camera decode, photo, `scanText`). `doScan`, the F6 gate, the resolver wiring and every render are untouched. No new copy, no new page state, no new machine (state, event) pair. |
| `tests/marketing.spec.js` | yes (solo owner) | Appended: `[PS-01]`, `[PS-01b]`, `[PS-02]`, `[SV-08]`–`[SV-11]`. No existing test or helper edited; `[SV-01]`–`[SV-07]`, Scanner polish and Camera scanner describes untouched. |
| `sw.js` | yes | Regenerated after `scan-page.js` is committed (it reads git HEAD). Revision of `marketing/scan-page.js` changes; nothing added or removed — count stays 51. The orchestrator regenerates at merged HEAD; never hand-merge. |
| `.night-crew/knowledge/BACKLOG.md`, `roadmap.md` | bookkeeping | B-475 / B-476 → `landed → photo-scan-guard-and-lookup-arms`; roadmap J1 line `PLANNED` → `LANDED (run 20261005, branch card/j1-photo-scan-guard-and-lookup-arms)`. |
| `.night-crew/runs/2026-10-05-autonomous/logs/photo-scan-guard-and-lookup-arms/` | evidence | Pre-change red, four mutation reds, green whole-file run, G1, `sw.js` regen/count. |

Expected NOT touched: `marketing/scanner.js` (every mutation is applied in the worktree and
reverted, never committed), `marketing/submit-flow.js`, `marketing/submit-machine.js`,
`night-crew.toml`, any `backend/` file. Any change to this list is stated in the closing
section below.

## What must survive any merge

1. The `decodeBusy` check-and-set at the top of `onFilePicked` and its release in `finally`.
2. The `input.value = ''` reset on BOTH the refused and the completed pick.
3. `scanText` going through the same flag (drop it and `[PS-01]` reds; `[PS-01b]` — photo then
   photo — still holds).
4. The seven new specs, each with the assertions its mutation reds on (listed below).

## What is safe to drop

Nothing here. (The evidence logs are evidence, not code, but the done_when cites them.)

## Per done_when clause — what rides a stub or fixture

The Playwright stack has no sync substrate, so — exactly as the existing "Scan-time verify"
describe — `mockSyncTransports` serves the mint envelope and the pull replicas at the network
layer. That is the stack, not the feature under test.

| Clause | What answers | Stub? |
|---|---|---|
| `[PS-01]` / `[PS-01b]` held lookup | `page.route` on the door's lookup path HOLDS the request; the test later fulfils it with one live row | The HOLD is un-stubbed (nothing answers while the photo is picked). The released ROW is the one permitted stub. The photo is `tests/fixtures/qr-fixture-1.png` through the real `#scan-file` input and the real html5-qrcode decode. Code B is seeded into the local replicas (fixture). |
| `[PS-02]` | no lookup involved for the photo (code held locally — fixture rows seeded) | fixture only |
| `[SV-08]` held + locally expired, zero lookups | counting route that WOULD serve a live row if reached | un-stubbed count (zero requests sent); local rows are seeded fixtures |
| `[SV-09]` expired server row | `page.route` fulfils one expired row | the permitted stub (the row) |
| `[SV-10]` non-200 | to be stated at close (real door vs fulfilled status) | see closing section |
| `[SV-11]` throwing policy source | to be stated at close | see closing section |

## Closing section (facts as built)

_To be completed at the end of the card._
