# Extraction — atomic-scan-dedupe

Outcome: confirmed, with two corrections

Approach used: a spike-owned database on `:5434` migrated by booting the built
server once; 12 psql clients released on a shared `pg_sleep`-until start line,
five runs each against the shipped statement and then against the candidate DDL.
Two spikes, both exit 0 on the first run. Candidate input for the card, not an
adoption (NFR-6).

Confirmed: the shipped `INSERT … WHERE NOT EXISTS` lets 11–12 of 12 aligned
concurrent hits through; the decided shape leaves exactly one row under the same
load, still counts a return visit after the window, and still counts every
anonymous (`NULL ip_hash`) scan. PostgreSQL 16.13 accepts the generated column.
The migration 0087 draft and the insert, verbatim from the run:

    ALTER TABLE qr_scans ADD COLUMN bucket timestamptz GENERATED ALWAYS AS (date_bin('10 minutes', scanned_at, timestamptz '2000-01-01 00:00:00+00')) STORED;
    CREATE UNIQUE INDEX qr_scans_dedupe_idx ON qr_scans (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL;

    INSERT INTO qr_scans (short, ua_family, referrer, ip_hash)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL DO NOTHING;

Learned: (1) the race only reproduces from already-open connections released
together — a spawn-staggered test passes against the broken code, so the card's
concurrency test must use a barrier; (2) a tumbling bucket counts two taps that
straddle a bucket edge twice, which the sliding window would not — decision 195's
accepted trade-off, now stated; (3) the `WHERE ip_hash IS NOT NULL` clause must
appear in the conflict target to match the partial index, and the `$5` interval
parameter disappears from the handler.

Plan change: none to scope — the card ships the printed DDL + insert; its
concurrency test is specified as barrier-released goroutines on open connections,
and its window test asserts bucket semantics.
