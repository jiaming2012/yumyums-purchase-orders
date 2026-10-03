# Spikes — atomic-scan-dedupe

Activity: Activity I — Honest gates, then verify before the till (triage 20261002 follow-ups)

> Hand-run convention (see README.md in this directory). Runs on `:5434` in a spike-owned
> database `hq_test_spike_i4_20261003` migrated by the repo's own goose runner; dropped at the
> end. Never `:5433`.

## The goal, and which legs need a spike

The card (I4 — decision 195, ledger T-62): the public landing's 10-minute scan dedupe
(`backend/internal/marketing/landing.go` `INSERT … SELECT … WHERE NOT EXISTS`) is a
read-then-write under READ COMMITTED; 12 concurrent hits from one IP produced 3/7/8/8/8 rows
where one was wanted, on an unauthenticated route, so the orphan rate's denominator is
inflatable from outside. Decided (engineer-level, handed back by the operator): a tumbling
10-minute bucket, a unique index on `(short, ip_hash, bucket)`, `ON CONFLICT DO NOTHING`.
Behaviour: double-tapping a QR link counts once; a return visit after ten minutes still
counts; an anonymous scan with no IP hash still counts every time (the stated honest
over-count).

Two premises a script can settle now: (1) the over-count reproduces with plain concurrent
inserts of the shipped statement; (2) the decided shape yields exactly one row under the same
load, and a bucket expression Postgres accepts for a stored generated column (or, failing
that, a bucket computed in the insert) exists on this server version.

## Spike: concurrent-landing-inserts-overcount

- proves: 12 concurrent executions of the shipped statement with one `(short, ip_hash)`
  leave more than one `qr_scans` row in at least one of five runs.
- plan: create + migrate the spike DB, seed a user/campaign/code, fire 12 psql clients in
  parallel five times, count rows after each, print the counts.
- script: .night-crew/spikes/activity-i-honest-gates-then-verify-before-the-till-triage-20261002-follow-ups/atomic-scan-dedupe/01-concurrent-landing-inserts-overcount.sh

### Runs

- 2026-10-03T13:37:48Z · exit 0 · passed

## Spike: bucket-unique-index-makes-it-one

- proves: with a 10-minute bucket column (generated if the server allows an immutable
  expression, else written by the insert), a partial unique index on
  `(short, ip_hash, bucket) WHERE ip_hash IS NOT NULL`, and `INSERT … ON CONFLICT DO NOTHING`,
  the same 12×5 load leaves exactly one row every run; two scans eleven minutes apart leave
  two; a NULL `ip_hash` inserts every time.
- plan: same DB, apply the candidate DDL (the card's migration draft), re-run the load and the
  two edge cases, print the counts and the expression that worked.
- script: .night-crew/spikes/activity-i-honest-gates-then-verify-before-the-till-triage-20261002-follow-ups/atomic-scan-dedupe/02-bucket-unique-index-makes-it-one.sh

### Runs

- 2026-10-03T13:38:04Z · exit 0 · passed

## Verdict (hand-run 2026-10-02)

- **concurrent-landing-inserts-overcount: passed** — exit 0, ~16 s: 12 clients on a shared
  start line (a `pg_sleep`-until barrier in `_lib.sh blast()`) × 5 runs against the shipped
  statement left **12 / 12 / 12 / 11 / 12** rows where one was wanted. The over-count
  reproduces; the premise was understated, not wrong (triage saw 3–8 with HTTP-level stagger).
- **bucket-unique-index-makes-it-one: passed** — exit 0, ~16 s: with the candidate DDL the same
  12 × 5 load left **1 / 1 / 1 / 1 / 1**; two scans eleven minutes apart → 2; two `NULL`
  `ip_hash` scans → 2. **PostgreSQL 16.13 accepts the STORED generated column**
  `date_bin('10 minutes', scanned_at, timestamptz '2000-01-01 00:00:00+00')` — `date_bin` over
  `timestamptz` with a constant origin is immutable — so no fallback shape is needed. The DDL
  and the insert that held are in the extraction record (migration 0087 draft).

## Corrections

- **Agent-reached correction 1 — the regression test must race from already-open connections.**
  Plain `xargs -P 12` process spawn staggers the clients enough to mask the race (first attempt
  1 1 2 1 1); only a shared start line reproduces it. The card's
  `TestLandingDedupeIsAtomicUnderConcurrency` opens its connections first and releases 12
  goroutines on one barrier, or it proves nothing.
- **Agent-reached correction 2 — a tumbling bucket is not a sliding window.** Two taps that
  straddle a bucket edge (`:09:59` and `:10:00`) count twice, where the shipped `scanned_at >
  now() - interval '10 minutes'` would have counted once. Decision 195 chose the tumbling bucket
  knowingly (the alternative — an advisory lock on an unauthenticated route — was rejected), so
  this is the accepted trade-off, **stated for the operator's review** and carried into the
  card's test names (`TestLandingCountsAgainAfterWindow` asserts the bucket semantics, not the
  sliding one).
- Precision: `users` seed shape as in the sibling ledger (0017 / 0023).

## Comebacks

- none recorded — the harness gotcha (`(cd … && server &)` backgrounds a wrapper shell, so `$!`
  is not the server; fixed with `exec` + TERM→KILL escalation) is a spike-script lesson,
  recorded in `_lib.sh`, not a product gap.

## Review

- signed: operator, 2026-10-02 — covers 2 correction(s) (reviewed at the slate sitting of 2026-10-02, §4 batch sign-off, with the sitting's call for each in view)
