# Extraction — rxdb-pull-replica

Outcome: confirmed

Approach used: `replicateRxCollection` with a custom pull handler querying
PostgREST as a device JWT — `order=updated_at.asc,id.asc` +
`updated_at=gt.<checkpoint>` (URL-encoded) + the §5.3 bound
`expires_at=gt.<now-2d>` — and `pull.stream$` emitting RESYNC on Realtime
postgres_changes / re-SUBSCRIBED (§7.3). Run against the BUILT Activity A
migrations + seed on the local `spike-supabase` substrate. A candidate the
card's design.md adopts or not (NFR-6).

Confirmed: the card's two falsifiable premises. (1) The pull is bounded AND
checkpointed at once — initial sync landed exactly the 6 in-window codes while
the out-of-window rows did NOT land (the bound held both ways), and the
post-initial requests observably carried the real `updated_at` cursor
(request log enumerated, not inferred). (2) The done_when core — a code burned
by the committed `redeem()` RPC on device A surfaced as redeemed in device B's
RUNNING replica **254 ms** later via Realtime nudge → pull tick, no restart,
no manual reSync. The offers replica is this same mechanism with a different
filter and key — no separate premise.

Learned: two build-facts inside the confirm. The checkpoint cursor MUST be
URL-encoded (`+00:00` in a raw query string decodes as a space and breaks the
timestamptz parse — the spike's handler does
`encodeURIComponent(cursor)`). And the `gt` cursor leaves a theoretical
missed-row edge: same-`updated_at` ties at a batch boundary can be skipped —
the card must keep the id tiebreak in the checkpoint or over-fetch one row.
That edge is the exact silent-miss failure class (a redemption never reaching
device B), so it is anchored as GAP-1 in this goal's ledger `## Comebacks`.

Plan change: the card builds the pull replica as spiked, with the id tiebreak
(or over-fetch) added to the checkpoint — the fixing card owes the one
validation run that exercises a same-timestamp batch boundary (GAP-1). The
prerequisite stub stands as stated in the ledger: Activity B remains gated on
Activity 0's field observation #6; if the activity collapses to a thin live
cache, this same mechanism is what the thin cache uses — the evidence prices
both shapes. `redeem()`'s `updated_at = now()` stamp (decision 163) is what
makes redemptions visible to the pull tick; verify 04 leg H already guards it.
