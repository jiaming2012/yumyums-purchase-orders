# Spikes — scanner-refusal-seam-and-pick-feedback

Activity: Activity K — Review leftovers and Inventory Setup races (triage 20261005 follow-ups)

> Tool-run (`night-crew spikes run`). One Playwright spike in a THROWAWAY worktree off `dev`
> (`../hq-worktrees/spike-k4-20261007`) on a spike-owned e2e stack
> (`hq_test_spike_k4_20261007`, `TEST_PORT=8343`, `:5434`). The sync door is mocked at the
> network layer the way `tests/marketing.spec.js` does it. Never `:5433`.

## The goal, and which legs need a spike

The card (K4 — B-482, B-483; adversarial review of run `20261005`, triage T-66): a photo picked
while the scanner is checking a code is refused by the J1 guard with no feedback of its own
(B-483); and the refusal screen under a throwing campaign-policy source has no page-level gate
because `submit-flow.js` captures `policyFor` once at boot — the only way in is a seam that does
not exist yet (B-482). Two premises a script can settle now, through the SHIPPED page: (1) with
code A's server lookup held, a photo of locally-held code B picked mid-wait leaves the result
area reading exactly what it read before the pick, with no new line anywhere — silent; (2)
patching `window.MarketingScan.campaignPolicy.policyFor` to throw AFTER boot changes nothing
about an offline scan of a held low-value code (the result kind equals the unpatched control) —
the capture-once design means a boot-time override is the only seam a test can use.

## Spike: refused-pick-is-silent-and-post-boot-policy-patch-is-inert

- proves: (a) B-483 — provisioned scanner, lookup for never-seen code A held, scan A
  (`checkingServer`), pick a photo of held code B: no `#scan-prompt`, the `#scan-result` text is
  byte-identical before and after the pick, and no element on the page contains "pick again" or
  "Finish checking"; (b) B-482 — provisioned scanner, code B held locally with a low-value
  campaign, `policyFor` monkey-patched to throw after boot, go offline, scan B's payload: the
  result kind equals a control run with no patch (the throw never reaches the page), so the
  fail-closed refusal is unreachable without a boot-time override.
- plan: worktree, warm build, copy a three-spec Playwright file (premise a, premise b, control
  b) into the worktree's `tests/`, run on the spike stack, read outcomes and the logged kinds.
- script: .night-crew/spikes/activity-k-review-leftovers-and-inventory-setup-races-triage-20261005-follow-ups/scanner-refusal-seam-and-pick-feedback/01-refused-pick-is-silent-and-post-boot-policy-patch-is-inert.sh

### Runs

- 2026-10-05T15:07:40Z · exit 0 · passed

## Verdict (tool-run 2026-10-05)

- **refused-pick-is-silent-and-post-boot-policy-patch-is-inert: passed** on its first run (exit
  0, 2.6 m): (a) with code A's lookup held and `#scan-result` at `checkingServer`, a photo of held
  code B picked mid-wait → `{"kind":"checkingServer","prompt":0,"textUnchanged":true,
  "pickAgain":false,"finishChecking":false,"note":0}` — the J1 guard refuses silently (B-483);
  (c) control: offline scan of held low-value code B, unpatched → `offerReady`, no refusal copy;
  (b) `window.MarketingScan.campaignPolicy.policyFor` patched to throw AFTER boot, same offline
  scan → `offerReady`, no refusal copy — identical to the control: the throw never reaches the
  page, because `submit-flow.js` captured the function once at boot (B-482's premise holds; a
  boot-time override is the only seam a test can use).

## Corrections

- none agent-reached. Precision carried into the card: the override must be read where the
  policy source is CREATED (`scan-page.js`, before `submit-flow` boots), not patched onto the
  object afterwards; `page.addInitScript` is the test's door.

## Comebacks

- none.

## Review

- signed: operator, 2026-10-05 — covers 0 correction(s) (reviewed at the slate sitting of
  2026-10-05, slate-20261007 §4 batch sign-off; no corrections to review).
