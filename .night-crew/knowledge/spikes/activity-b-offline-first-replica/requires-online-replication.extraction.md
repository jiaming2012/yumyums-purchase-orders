# Extraction — requires-online-replication

Outcome: learned

Approach used: a **dedicated `campaigns` pull replica** — the same shipped
`marketing/sync/pull-replication.js` mechanism the codes replica uses
(`replicateRxCollection` + `makePullHandler`, GAP-1 keyset checkpoint
unchanged), with `buildPullUrl()`'s `expires_at` bound made **optional** so it
can serve a table that has no such column; the policy lookup handed to
`setCampaignPolicy` reads that replica rather than a literal. For F-2, the
**distinct landing path** guard — `scan_attempts.code_id` nullable + a
`token_hash` column + a check constraint, with the guard placed **before** the
`redeem()` call so an unverified attempt skips redeem and lands directly,
carrying `offline_override=t, unverified_code=t`. Both are candidates the
card's design.md adopts or not (NFR-6).

Confirmed: the done_when's first half, and the premise under the whole card.
The SHIPPED `marketing/submit-machine.js` on the SHIPPED vendored
`lib/xstate.umd.min.js` (sha256 `e7f04e1f…`, `mode: 'throw'`) refuses the
override for the HIGH code (…0005 → campaign …0002, `requires_online=true`)
while offline **with** `canOverride: true` → `blockedOffline`, and offers it
for the LOW code → `overrideConfirm`. The negative leg makes it a proof rather
than a restatement of Card 6 conformance seq 10: with today's shipped policy
source (`CAMPAIGN_POLICY = null`), **both** codes are overridable — the $40
catering-credit code behaves exactly like the $2 one, so the refusal is
demonstrably unreachable today. Feeding `requiresOnline` from a replica added
**zero undeclared (state,event) pairs** across all four runs with the actor
alive, so Card 6's 460-pair strictness proof still holds. Also confirmed:
`authenticated` holds SELECT on `campaigns` with `using (true)` — no RLS work
owed — and F-2 is real, not inferred.

Learned: four facts that changed the design, none of which the card carried in.
(1) **The flip decides the mechanism, and it rejects the roadmap's second
candidate.** Both mechanisms land the flag on initial sync, so that leg decides
nothing; after a stamped flip the campaigns replica delivered `true` on the next
RESYNC while the codes-embed replica still read `false`, because
`max(codes.updated_at)` was byte-identical either side of the campaign write
(`2026-09-05 09:24:41.989173+00`). An embedded flag re-reads only when the CODE
row moves, so a campaign downgraded while its codes sit still never
re-delivers — arming the refusal *wrong* is worse than today's honest unknown.
(2) `buildPullUrl()` appends `expires_at=gt.<windowIso>` unconditionally and
`campaigns` has no such column, so the shipped handler answers **HTTP 400** on
that table — the module owes an optional bound, and only that. (3) **`campaigns`
has no touch trigger and is not in the `supabase_realtime` publication** — both
enumerated. An unstamped write is invisible to *every* checkpointed replica
(not mechanism-specific), and without publication membership a pull replica
does not poll: 3s after a stamped write with no RESYNC the replica still read
the old value. (4) **`/rpc/redeem` refuses first** — HTTP 400, Postgres `22P02`
on `p_code uuid` — not the landing insert, so the guard belongs before the
redeem call. Blast radius measured, not argued: 12 redeem attempts over ~12
retry cycles produced **0 landing requests** and the legitimate attempt behind
the poison row never left `pending`. Of the two guards run, *skip-until-
arbitration* was **rejected** — it drains the queue but the audit-flagged
attempt never reaches the server, which retroactively falsifies decision 166's
own reasoning for ratifying unknown→false.

Plan change: the card builds a campaigns pull replica (the codes-embed
alternative is closed, with evidence). It owes four concrete things the ledger
did not previously name: an **optional expiry bound** in `buildPullUrl()`;
**publication membership for `campaigns`** or a fan-out of the codes channel's
RESYNC into the campaigns replica; the **F-2 guard placed before `redeem()`**
via the distinct landing path, plus the ONE validation run re-executing spike 03
against the shipped guard (GAP-1); and a harness that **mints a fresh live code
per leg** — a code is single use, and the shipped push handler's GAP-1 belt 2
correctly blocks rather than guessing when `redeem()` answers `already_used`
with no codes replica to name the winner (build-fact 5, correction 2 — that
blocking is not a product defect). Two residuals are recorded rather than
fixed here: GAP-2, the unstamped campaign write that is silently invisible to
every replica, arms the moment the §16 provisioning surface lands and belongs
to whichever card builds the campaign write path; GAP-3, the embedded-offer
path (`submit-flow.js:304-309` maps `embeddedOffer` → `unknownCode`, carrying
no `campaign_id`), stays policy-unknown by construction even after this card —
it is Activity E's `identity-code-and-qr` payload question, and must not be
mistaken for coverage when Q-KR1 is attested. Left open for the card, not
settled here: the server-side `status` taxonomy for an unverified override —
the spike landed it `accepted` + `offline_override` + `unverified_code`, but
§9/§19 should be re-read before building on it.
