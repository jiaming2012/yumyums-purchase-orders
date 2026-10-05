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
| `[SV-10]` non-200 | **leg 1:** the lookup is passed through (`route.continue()`) to the REAL door, which has no substrate behind it in this stack and answered **HTTP 503** by itself (status read from the response, asserted non-200). **leg 2:** 401 / 500 / 404 / 503 fulfilled by `page.route`, with bodies chosen to mislead a reader that ignored the status (`[]`, a live row). Control: `200 []` is the plain unknown-code result. | leg 1 **un-stubbed**; leg 2 is a **stub of the status**, said so in the spec. The mutation reds on leg 1. |
| `[SV-11]` throwing policy source | the SHIPPED `policyFor` export (same module instance the page booted) is handed a function that throws; its answer drives a SHIPPED `createSubmitMachine` (mode `throw`) to the offline-override question | **No seam stubbed** — `setCampaignPolicy` is never called and `MarketingScan.campaignPolicy` is never replaced. See the limit below. |

## Closing section (facts as built)

**Files changed** (vs `overnight-20261005` @ `3bd6a9b`): `marketing/scan-page.js`,
`tests/marketing.spec.js`, `sw.js`, `BACKLOG.md`, `roadmap.md`, this note, the evidence logs.
**Not touched:** `marketing/scanner.js` (four mutations applied and reverted in the worktree —
each log ends with the clean `git status`), `marketing/submit-flow.js`,
`marketing/submit-machine.js`, `night-crew.toml`, every `backend/` file. No scope drift.

**`[SV-11]` — what is real and what could not be forced.** The real policy source
(`createCampaignPolicySource().policyFor`) is a `Map` lookup and cannot be made to throw, and
`submit-flow.js` captures it once at boot — so the only way to put a throwing source behind the
PAGE's own scan → refusal render is the injection seam (`setCampaignPolicy`), which is the stub
the slate forbids. That page-level render is therefore **not driven by any test**. What `[SV-11]`
does instead, un-stubbed: it calls the shipped `policyFor` with a throwing source (the source is
that function's argument, so this is its input, not a seam) and feeds the answer to a shipped
submit machine — `requiresOnline: true`, `overrideAvailable: false`, `blockedOffline` for a code
that names a campaign; `false` for a code that names none (decision 166); a healthy source is
believed (control). Under the named mutation (`catch` → `return false`) it reds: the machine
reaches `overrideConfirm`.

**`campaigns-run.sh` does NOT gate this arm.** The slate's fallback was to name it as the arm's
only gate. Read, not run: `campaigns-harness.mjs` leg 3 calls `policyFor` with the healthy
replica source and with `null` only — never a throwing one — so the `catch` mutation would leave
it green. It was not run tonight: it does `reset_bare` on the local spike-supabase substrate,
which the box rules say stays untouched. **`[SV-11]` is the catch arm's only gate.**

**Guard placement, and why `scanText` is in it.** With the flag on the photo path alone, the
spike's own recipe (A started through `MarketingScan.scanText`) stays stuck — `scanText` was bare
`doScan` and never set `decodeBusy`. Log `06-mutation-ps-scantext-red.log` shows exactly that:
`[PS-01]` and `[PS-02]` red, `[PS-01b]` (photo then photo) green. No product code calls
`scanText`; the behaviour change is that a second `scanText` issued while a scan is still
resolving returns `null` instead of reaching the F6 gate. All 63 pre-existing specs in the file
pass with it.

**Test-authoring note.** The first draft of `[PS-01]` used polling expects during the wait; on
the pre-change tree they outlasted the lookup's 3.5 s budget, so the red it produced was a
timed-out lookup, not the stuck state. The mid-wait checks are single DOM snapshots now, and
`01-ps-prechange-red.log` is the run of the committed spec.

**Evidence** (`.night-crew/runs/2026-10-05-autonomous/logs/photo-scan-guard-and-lookup-arms/`):

| Log | What it shows |
|---|---|
| `01-ps-prechange-red.log` | pre-change page: 3 failed — `#ms-order` 0, `#scan-prompt` 1, machine `resolving` |
| `02-mutation-sv08-red.log` | 1 failed (`[SV-08]`: `offerReady` / `source: server` instead of `expiredLocally`) / 6 passed |
| `03-mutation-sv09-red.log` | 1 failed (`[SV-09]`: `offerReady` instead of `expiredLocally`) / 6 passed |
| `04-mutation-sv10-red.log` | 1 failed (`[SV-10]`, real door HTTP 503: `verified: false` missing) / 6 passed |
| `05-mutation-sv11-red.log` | 1 failed (`[SV-11]`: `overrideConfirm`, `requiresOnline: false`) / 6 passed |
| `06-mutation-ps-scantext-red.log` | `scanText` un-guarded: 2 failed (`[PS-01]`, `[PS-02]`) / 5 passed |
| `07-mutation-ps-photo-red.log` | photo path un-guarded on the fixed tree: 3 failed (`[PS-*]`) / 4 passed |
| `08-sw-regen.log` | 51 `revision` entries; only `marketing/scan-page.js`'s revision moved; second run identical |
| `09-marketing-whole-file-green.log` | whole `tests/marketing.spec.js`, `--retries=0`, fresh database: 70 passed, 0 failed, 0 skipped |
| `10-g1.log` | `go build ./...` exit 0, `go vet ./...` exit 0 |

Mutation runs 02–07 used a hand-provisioned server on `:8221` (same env as
`playwright.config.js`, database `hq_test_e2e_j1_20261005` on `:5434`) and ran the new describe
only; the whole-file run used Playwright's own `webServer` with a fresh reset.

**Not verified here:** the full Playwright suite and the full Go suite (the orchestrator's, under
the lock); a real camera decode (headless has none — the camera path's own guard is unchanged);
`campaigns-run.sh` (not run, see above).
