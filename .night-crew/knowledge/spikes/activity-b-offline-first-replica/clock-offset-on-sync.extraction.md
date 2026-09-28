# Extraction — clock-offset-on-sync

Outcome: confirmed

Approach used: the HTTP `Date` header on the PostgREST pull response the sync
already makes as the serverNow source — no new endpoint. Capture
`offset = serverNow − deviceNow` on every successful pull, store it beside
the checkpoint, and run every offline `expires_at` comparison as
`deviceNow + offset < expires_at`. A candidate the card's design.md adopts or
not (NFR-6).

Confirmed: both premises. (a) The `Date` header is present and usable — under
an injected 2-days-slow device clock the computed offset recovered the skew to
**196 ms** error (rtt 85 ms; whole-second header resolution is plenty for
expiry windows measured in hours). (b) The §5.1 defect class is real and the
adjustment closes it: the red analog (`deviceNow < expires_at`) ACCEPTED a
code that expired a day earlier under the slow clock; the offset-adjusted
check REJECTED the same code under the same skew. The comparison is pure local
arithmetic — exactly what the offline path runs.

Learned: (nothing new — the first-named candidate source held; the `now()` RPC
fallback the spike named for a missing header is not needed.)

Plan change: none — approach offered as a candidate only. Sign convention for
the card if adopted: `offset = serverNow − deviceNow`.
