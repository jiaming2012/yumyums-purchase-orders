# Extraction — scanner-refusal-seam-and-pick-feedback

Outcome: confirmed, no corrections

Approach used: a throwaway worktree of `dev` on a spike-owned e2e stack, a three-spec
Playwright file through the shipped `marketing.html` with the sync door mocked at the
network layer: a held lookup with a mid-wait photo pick; an offline scan of a held
low-value code as the control; the same scan after monkey-patching `policyFor` to throw
post-boot. Tool-recorded run: exit 0 first time. Candidate input for the card, not an
adoption.

Confirmed: (a) B-483 — the refused pick leaves the result text byte-identical, no prompt,
no feedback copy anywhere; (b) B-482 — a post-boot patch of
`MarketingScan.campaignPolicy.policyFor` changes nothing (`offerReady` both ways): the
function is captured once at boot, so the fail-closed arm is unreachable from a test
without a boot-time override.

Learned: (1) the seam has to be read where the policy source is created in
`scan-page.js`, before `submit-flow` captures it; `page.addInitScript` is how a test sets
it; (2) the feedback line is a render concern, not a machine concern — no new (state,
event) pair.

Plan change: none — the card ships the one-line note and the documented test-only
override; red-first recipes are this spike's legs (a) and (b).
