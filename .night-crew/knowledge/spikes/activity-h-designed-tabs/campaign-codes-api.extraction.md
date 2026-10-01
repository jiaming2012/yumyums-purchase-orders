# Extraction — campaign-codes-api

Outcome: confirmed

Approach used: campaign admin as HQ Go + Postgres with a Supabase `campaigns`
projection written over PostgREST by a service-role JWT (`Prefer:
resolution=merge-duplicates` upsert of the four tablet columns), the campaign
QR as a `skip2/go-qrcode` PNG of the 35-character short URL, and the manager
tier read inside the handler from `auth.UserFromContext(ctx).Roles`. Three
spikes, all exit 0 on the first run (hand-run 2026-10-01 against the LOCAL
spike-supabase substrate; never :5433, never hosted). A candidate design
input for the card, not an adoption (NFR-6).

Confirmed: (1) the short-URL QR is version 3 (29 modules a side) at Medium
error correction and decodes to the identical string with the vendored
`html5-qrcode` scanner the tablets ship; (2) the projection path exists —
service-role upsert answered 200 with the representation, read-back matched,
fixture restored; (3) the handler can derive the caller's tier with no new
middleware or DB read.

Learned: version 3 is the ceiling for this URL at Medium correction — a
7-character key or a longer host tips to version 4 (33 modules). Keep
`hq.yumyums.kitchen/q/<6 chars>`; do not grow the payload.

Plan change: none — decision 187 stands; the card's design.md should cite
this record if it adopts the PostgREST projection route.
