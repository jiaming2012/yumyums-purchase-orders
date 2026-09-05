# DECISIONS-NEEDED — run 20260905

One fork, parked by the decisions machinery (verdict: park, top severity by the
three-role vote; recorded in `.night-crew/knowledge/decisions/20260905.jsonl`).

## D-1 · Card 6 parked: the `requires_online` unknown-policy default (§8)

**The question (operator's to answer):** which §8 behavior applies when a campaign's
`requires_online` flag is not client-readable? Tonight no campaigns replica exists and
the codes replica lacks the flag, so on real data EVERY campaign resolves to
policy-unknown.

**What the card built:** the §8 refusal is fully implemented and machine-guarded
wherever the flag IS known (`overrideAvailable = canOverride && !requiresOnline`,
pinned by the branch-3 e2e test via an injectable `setCampaignPolicy` seam). The
shipped default is **unknown→false**: the force-submit override is available behind
the `marketing-offline-override` entitlement + the §13 confirmation, and every such
attempt lands audit-flagged `offline_override=true` (plus `unverified_code=true` on
F2). Consequence tonight: the refusal cannot fire on any real code until the flag is
replicated.

**Why it parked:** the signed slate's Card-6 park note — "any weakening of the
`requires_online=true` refusal → park" — is a hard stopping condition; G6 ruled the
default park-class; no adopted delegation covers a §8 policy fork (P-1's bound stops
at product forks); the three-role vote rated it top. The alternative default
(unknown→true) was also a policy call: it silently deletes F2/the offline path.

**The proposal on the table (G6 + control loop, expected to survive unchanged):**
ratify unknown→false as shipped (audit-flagged, entitlement-gated, revertible by one
constant) **plus a named follow-up card** that replicates `requires_online` to
devices (or embeds it in the pull) and arms the refusal on real data — required
BEFORE any real campaign is provisioned. Close-bar leg 3 / Q-KR1 cannot be attested
until that lands, so the milestone cannot close over the gap.

**State of the card:** COMPLETE and unmerged. Branch `wo-redemption-submit-flow`
(10 commits + the F-2 merge-intent amendment), worktree preserved at
`/Users/jamal/projects/yumyums/wt-20260905-c6`. All gates green: conformance 18/18,
strictness 9/9, fuzz 40,000 walks with the per-step liveness assertion armed
(G6 proved the assertion can fail: deaths=254 on a neutered scratch copy), full
suite ×2 (856/5/6, 859/2/6 — zero unexplained reds), G4 idempotent at precache 43,
G6 PASS-WITH-NOTES. If ratified as-is, the merge is one command:
`git merge --no-ff wo-redemption-submit-flow` (+ conflict-log entry).

**Also folded into this fork's record (G6-c6 findings that ride the same card):**
- **F-2 (latent, cross-card):** the F2 unknown-code write puts `code_id = token_hash`
  (64 hex) but the substrate column is `uuid not null` — when provisioning arms sync,
  Card 3's push handler (no unverified_code branch) would 400 on uuid coercion and
  retry-poison the device queue. Unreachable tonight; merge-intent build call 7
  amended to record the truth; the follow-up card must give unverified attempts a
  distinct landing path or a skip-until-arbitration guard.
- **F-4 (hardening candidate):** the order-number guard lives in the page dispatch
  layer; machine-level SUBMIT and Card 7's handler would accept an empty order_number
  via a seam bypass. Cheap fixes: a machine context guard or a server-side 400.
