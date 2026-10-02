#!/usr/bin/env bash
# 01-concurrent-landing-inserts-overcount.sh — spike for card I4 (decision 195, T-62):
# the shipped landing dedupe (backend/internal/marketing/landing.go logScan,
# INSERT … SELECT … WHERE NOT EXISTS over a 10-minute window) is a read-then-write
# under READ COMMITTED, so 12 concurrent hits with one (short, ip_hash) should leave
# MORE than one qr_scans row. Triage saw 3/7/8/8/8. Five runs of 12 clients on a
# fresh, repo-migrated database hq_test_spike_i4_20261003 on :5434; dropped at exit.
# 🛑 exit 0 = over-count reproduced (max count > 1); exit 1 = all five runs gave 1
# (did not reproduce under this load — not looped until it does); exit 2 = could not run.
set -euo pipefail
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/_lib.sh"
spike_db_up

# The shipped statement, verbatim shape, parameters inlined ($5 = '10 minutes').
SHIPPED="INSERT INTO qr_scans (short, ua_family, referrer, ip_hash)
SELECT '$SHORT', 'ios', NULL, 'abc'
WHERE 'abc'::text IS NULL OR NOT EXISTS (
  SELECT 1 FROM qr_scans s
  WHERE s.short = '$SHORT' AND s.ip_hash = 'abc'
    AND s.scanned_at > now() - '10 minutes'::interval
);"

counts=(); max=0
for run in 1 2 3 4 5; do
  c="$(blast "$SHIPPED" 12)"
  counts+=("$c"); [ "$c" -gt "$max" ] && max="$c"
  echo "# run $run: 12 concurrent shipped inserts, same (short, ip_hash) → $c rows"
done
echo "# counts: ${counts[*]}  (wanted 1 each)"
[ "$max" -gt 1 ] || fail "all five runs left exactly 1 row — the race did not reproduce under 12 concurrent psql clients"
echo "✅ GREEN — the shipped WHERE NOT EXISTS dedupe over-counts under concurrency (max $max rows for one scan); the premise of I4 holds"
