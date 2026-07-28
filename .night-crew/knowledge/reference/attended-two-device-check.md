# Attended two-device convergence check — operator runbook

> The standing flag that re-arms whenever production `sync.js`, `sw.js`/`build-sw.js`, or the
> live submit path changes. This document is the procedure; it is reusable every time the flag
> re-arms. Written 2026-07-27 after the automated half was re-measured green (below), so that
> what you do by hand is only the part no harness can reach.
>
> **Clearing the flag means naming the `git_sha` you tested.** "I checked it on my phone" with no
> sha recorded does not clear it — the flag is per-build, not per-feeling.

## What is already automated (do NOT re-do this by hand)

Measured 2026-07-27 on `dev` @ `937543a`, `--retries=0`, fresh `hq_test_e2e` per leg:

| Leg | Result |
|---|---|
| `tests/sync.spec.js` — 58 tests, 29 `browser.newContext()` pairs | 58 passed / 0 failed, 6.4m |
| `tests/broadcast-rerender.spec.js` — 6 tests | 6 passed / 0 failed, 2.7m |
| `tests/workflows.spec.js -g "offline\|DBL"` — GATE-07/08 + DBL-01..04 | 6 passed / 0 failed, 2.6m |

That covers, across two independent contexts: live WS convergence + catch-up-on-reopen for all 7
persisted field types, submit/unsubmit transitions, list progress, the systematic op-type × editor
× derived-view matrix (incl. `ESC-1` non-assignee admin second device), the W-6/W-6b LWW conflict
cells with the loser's `WebSocket` stubbed dead, and the offline submit queue's idempotency.

## What only two physical devices can prove

1. **A running service worker.** `playwright.config.js:60` sets `serviceWorkers: 'block'` repo-wide
   and no spec overrides it (`test.use(` appears in zero test files) — and the roadmap's
   `sync-rxdb-feasibility-spike` explicitly says *never change the repo-wide setting*. So SW
   install, SW update → `controllerchange` → `ptr.js` reload, and offline shell serving are
   **entirely unproven by the suite**. `tests/sw-manifest.spec.js` asserts the precache manifest as
   a *file*; it never installs one.
2. **Real socket death.** `context.setOffline(true)` flips a flag. A phone that sleeps, loses
   cellular, or hands off wifi tears the socket down and re-establishes it — the reconnect/
   catch-up path at its real trigger.
3. **iOS standalone + the prod transport.** Home-screen PWA in standalone mode, over the
   Cloudflare Tunnel, not headless Chromium on localhost.

## Step 0 — decide WHICH BUILD you are testing (do this first)

    task version              # local source / dev server / prod side-by-side
    curl -s https://hq.yumyums.kitchen/api/v1/health

As of 2026-07-27 prod serves `32afb39` (backend 0.2.2 / frontend 1.1.0, built 2026-07-24) and `dev`
is **109 commits ahead**. Both cards that armed this flag — `pwa-cache-and-build-hygiene` and
`workflow-offline-double-submit` (merge `15d9153`) — are on `dev` only.

**Therefore: testing prod today exercises neither card and does not clear the flag.** To discharge
it you must first promote `dev → main` and deploy — `task prod:deploy` does
`git reset --hard origin/main` (B-13), so it ships `main`, not `dev`. Promotion is `/nc-release`,
an attended decision of its own.

If you only want a smoke test of the *current* prod, that is fine — run blocks 1–3 below and record
it as a prod smoke, explicitly NOT as the flag's discharge.

## Setup

- Two phones. Ideally different OSes (one iOS, one Android) and **different networks** (one on
  cellular, one on wifi) — same-LAN testing hides tunnel problems.
- Two accounts: your admin/superadmin, and a crew member. Block 2's `ESC-1` case wants the *same*
  admin on both devices, so have both logins available on each phone.
- On each phone: open `https://hq.yumyums.kitchen` in Safari/Chrome → Share → **Add to Home
  Screen** → launch from the home screen icon (standalone, not a browser tab). If it opens with a
  URL bar you are not testing the PWA.

---

## Block 1 — service worker install, update, offline shell

| # | Do | Pass |
|---|---|---|
| 1.1 | Launch the PWA from the home screen on both devices | Launcher grid renders, no URL bar |
| 1.2 | Visit `/api/v1/health` in a browser tab on each device | `git_sha` matches the build you deployed |
| 1.3 | With the PWA backgrounded (not force-quit), deploy the next build, then reopen the PWA | App reloads **itself** once within a few seconds and shows the new version — no force-quit, no reinstall |
| 1.4 | Airplane mode ON, force-quit the PWA, cold-launch from the home screen | App shell renders (launcher grid), NOT the browser offline page |
| 1.5 | Still offline, tap into Operations | Page loads from cache; API-dependent lists may be empty, but the shell must not be a dead page |

1.3 is the failure class the precache guards exist for: a 404 in the precache manifest fails the
whole SW install silently, and the crew's symptom is "the PWA stopped updating" with no error
anyone can see.

## Block 2 — live convergence, two devices, one checklist

Device **A** = editor, device **B** = observer with the same checklist open. Do not reload B.

| # | Do on A | Pass on B (no reload, few seconds) |
|---|---|---|
| 2.1 | Check a checkbox | Shows checked |
| 2.2 | Answer a yes/no | Shows the same answer |
| 2.3 | Type a text answer | Shows the text |
| 2.4 | Enter a temperature | Shows the value |
| 2.5 | Tick a sub-step | Shows ticked |
| 2.6 | Fail a field + write a fail note + set severity | Note text and severity both appear |
| 2.7 | As admin in Builder: cut a field from the template | The cut field disappears from B's **open runner**; already-entered answers on surviving fields are kept; a newly added field appears empty |
| 2.8 | Submit the checklist on A | B's list shows the transition (progress/status), not a stale fillable runner |
| 2.9 | Same admin account logged in on **both** devices, edit as a non-assignee admin | The edit reaches the admin's own second device (`ESC-1`) |

## Block 3 — suspend, resume, network handoff (not reachable headless)

| # | Do | Pass |
|---|---|---|
| 3.1 | Lock B's screen ~2 min while A makes 3 edits | On unlock, B shows **all 3** within a few seconds |
| 3.2 | Watch B's toasts during 3.1's catch-up | Catch-up replay is **silent** — no toast storm. Only a *live* edit while B is awake surfaces one |
| 3.3 | On B: wifi off (falls to cellular) while A edits, then wifi on | Edits arrive; no duplicate rows; no stuck spinner |
| 3.4 | Force-quit B entirely, relaunch, open the same checklist | Every answer from A is present (catch-up on reopen) |

## Block 4 — offline submit queue (why the flag re-armed — Card B)

Run this **only against a build that contains `15d9153`**. On prod's current `32afb39` it tests the
old path and proves nothing about the card.

| # | Do on B | Pass |
|---|---|---|
| 4.1 | Open a checklist, answer every field | — |
| 4.2 | Airplane mode ON | — |
| 4.3 | Press Submit | No destructive error; the checklist stays editable/queued, not silently "gone" |
| 4.4 | Press Submit **again**, still offline | Still one queued submission — this is the double-submit case |
| 4.5 | Enter one more answer, still offline | Answer accepted locally |
| 4.6 | Airplane mode OFF, wait for drain | — |
| 4.7 | On A (admin → Approvals) | **Exactly ONE** submission for that checklist — not two |
| 4.8 | Open that submission on A | The answer entered in 4.5 is present, not dropped |
| 4.9 | Reopen the checklist on B | Not stuck; not offering a phantom re-submit |

## Block 5 — cross-tenant cache (Card A)

| # | Do on one device | Pass |
|---|---|---|
| 5.1 | Log out, log in as the other user | The previous user's checklists/approvals never appear — not even for a flash before the new data lands |

---

## Recording the result

Write into the current run's `HANDOFF.md` (or the triage note if there is no open run), and into
`ledger.md` at the next triage:

- the `git_sha` from `/api/v1/health` you tested (**required** — no sha, no discharge),
- both device models + OS versions, and which network each was on,
- per-block pass/fail, with the observed symptom for any fail (not "it seemed slow"),
- whether this was the flag's **discharge** or a **prod smoke** (per Step 0).

A block that could not be run (e.g. block 4 on a build without the card) is reported as **not run**,
not as a pass. The flag stays armed until every block has a result against one sha.
