# Cross-user checklist divergence — reproduced 2026-07-24 (operator play-test finding)

**Report:** operator observed the operations checklist
`workflows.html#checklist=eaa4ec71-4c00-46bb-a0c4-741473df7633` ("Friday checklist") out of
sync between Jamal C (admin) and Jim B (team_member) on dev.

**Verdict: REPRODUCED headlessly in fresh browser contexts. Deterministic client-side
per-user hydration divergence — NOT device/network/cache, NOT the 07-22 `sync.js` change,
NOT G1's gating.** The two-device-check-shaped explanations were all ruled out.

## What was ruled out (evidence)

- **Server state is identical for both users.** `GET /workflow/myChecklists` returned
  byte-identical payloads (4930 bytes) for both sessions; `GET /workflow/ops/since?since=0`
  returned the same 119-op journal to both. Diffed, not assumed.
- **Grants/WS:** Jim B holds `operations` via the `team_member` role grant — `/ws` and all
  `/workflow` REST pass for him. G1's gate is not involved.
- **Not load/latency:** reproduced with two fresh Playwright contexts against a local scratch
  server (`:8485`, real dev DB) — no Tailscale, no phones, no service worker, no journal
  catch-up storm needed.

## The reproduction (fresh contexts, same server, same DB)

Script: scratchpad `sync-repro.js` pattern — cookie-inject two sessions, open
`workflows.html`, click the checklist row, read `.check-btn` state.

    rowA (Jamal): "Friday checklist 1 section · 0/2 items"   first .check-btn: unchecked
    rowB (Jim B): "Friday checklist 1 section · 2/2 items"   first .check-btn: CHECKED
    Jamal clicks the checkbox → NO visual toggle, NO POST fired, no console error
    → CONVERGED: false

## Mechanism (from DB evidence)

The runner hydrates field state from **the viewing user's own submission history**
(`hydrateFieldState` ← `MY_SUBMISSIONS.responses`; there are ZERO draft rows —
`submission_responses.submission_id IS NULL` returns nothing — so hydration is entirely
submission-history-driven here):

- Jim B's latest submission of this template: **rejected** (`bfcf855a`, 07-19) → his client
  resurrects his 2/2 rejected answers as current state.
- Jamal's latest: **approved** (`6de34a39`, 07-19) → his client renders fresh 0/2, and his
  checkbox clicks are silently ignored (no POST, no toggle — consistent with an
  approved-cycle lock, but with no visible "Approved" badge on the row and no feedback on
  tap).

So two users viewing the SAME checklist see different states as a pure function of whose
submission was approved vs rejected last cycle — and neither state is the shared live state
the sync layer is supposed to converge on. The last ops/responses in dev date to 07-19;
nothing either user did today reached the server (Jamal's clicks no-op; Jim's state is a
ghost of his rejected copy).

## Why the E2E matrix missed it

The 32-cell convergence matrix covers {viewer}×{editor}×{op-type}×{derived-view} on a LIVE
shared cycle. This cell is **cross-user × cross-cycle-state**: user A's submission approved,
user B's rejected, THEN both reopen on a later day. No matrix row seeds that asymmetric
submission history first.

## Open product question (for the operator, next planning)

What should each user see when reopening a checklist whose last cycle ended
approved-for-one/rejected-for-the-other? Presumably: a new scheduled day starts a fresh
shared 0/2 cycle for everyone, and rejected-resurrection only applies within the same open
cycle. Needs an explicit ruling before the fix is specced.

## Status

Backlogged (operator: play-test findings ride the backlog). The repro is cheap to re-run:
scratch server on the dev DB + two cookie sessions. Sessions minted for the repro were
deleted; scratch server torn down.
