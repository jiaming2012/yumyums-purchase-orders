# Extraction — scan-time-verify

Outcome: confirmed

Approach used: a device-JWT GET against the local PostgREST for one `public.codes`
row by `token_hash` (the shape of scanner-polish spike 02), three hashes — seeded
live, random, seeded redeemed — each timed. One spike, exit 0 on the first run.
Candidate input for the card, not an adoption (NFR-6).

Confirmed: decision 200's direction holds as stated — an online phone can look up
the hash it has already computed on the connection it already has, in ~10–80 ms
locally, with the five columns the resolver needs; the device role reads redeemed
and expired rows too, so "already used" can render at scan time; an unknown hash is
an empty list, not an error. No new backend surface is needed.

Learned: PostgREST serialises timestamps with `+00:00`, not `Z`; a body-less
`HEAD … Prefer: count=exact` existence check is available if the card wants it.

Plan change: none — the card is wired as the optional `serverLookup` dep on
`createScanResolver`, called only when online and the token is in neither replica,
with the probe's timeout and today's behaviour as the fallback.
