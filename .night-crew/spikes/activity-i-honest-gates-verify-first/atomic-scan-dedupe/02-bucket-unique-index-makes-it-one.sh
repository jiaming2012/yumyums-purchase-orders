#!/usr/bin/env bash
# 02-bucket-unique-index-makes-it-one.sh — spike for card I4 (decision 195, T-62): the
# decided shape — a tumbling 10-minute `bucket` column, a partial unique index on
# (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL, and INSERT … ON CONFLICT DO NOTHING —
# makes the same 12×5 load leave exactly one row every run, keeps two scans eleven
# minutes apart as two rows, and still inserts every anonymous (NULL ip_hash) scan.
# Also settles whether PostgreSQL 16 accepts date_bin(...) over a timestamptz as a
# STORED generated column (it needs an immutable expression) or the insert must write
# the bucket itself. Prints the DDL + INSERT that worked = the card's migration 0087 draft.
# Fresh repo-migrated database hq_test_spike_i4_20261003 on :5434; dropped at exit.
# 🛑 exit 0 = all three legs hold; exit 1 = a leg failed; exit 2 = could not run.
set -euo pipefail
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/_lib.sh"
spike_db_up

EPOCH="timestamptz '2000-01-01 00:00:00+00'"
GEN_DDL="ALTER TABLE qr_scans ADD COLUMN bucket timestamptz GENERATED ALWAYS AS (date_bin('10 minutes', scanned_at, $EPOCH)) STORED;"
DEF_DDL="ALTER TABLE qr_scans ADD COLUMN bucket timestamptz NOT NULL DEFAULT date_bin('10 minutes', now(), $EPOCH);"
IDX_DDL="CREATE UNIQUE INDEX qr_scans_dedupe_idx ON qr_scans (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL;"

echo "# trying stored generated column:"; echo "  $GEN_DDL"
if err="$(q -c "$GEN_DDL" 2>&1)"; then
  SHAPE="generated"; BUCKET_DDL="$GEN_DDL"
  echo "# accepted — date_bin over a timestamptz is immutable on this server"
else
  echo "# REJECTED: $err"
  echo "# falling back to a plain column with a DEFAULT:"; echo "  $DEF_DDL"
  q -c "$DEF_DDL" || cannot_run "fallback DDL also rejected"
  SHAPE="default"; BUCKET_DDL="$DEF_DDL"
fi
q -c "$IDX_DDL"; echo "# index: $IDX_DDL"

# The new statement. With the generated shape the bucket derives from scanned_at, so the
# INSERT names no bucket; with the default shape the DEFAULT writes it (same now()).
NEW="INSERT INTO qr_scans (short, ua_family, referrer, ip_hash)
VALUES ('$SHORT', 'ios', NULL, 'abc')
ON CONFLICT (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL DO NOTHING;"

# (a) 12 concurrent × 5 runs → exactly 1 row each
counts=(); bad=0
for run in 1 2 3 4 5; do
  c="$(blast "$NEW" 12)"; counts+=("$c"); [ "$c" = "1" ] || bad=1
  echo "# leg a, run $run: 12 concurrent ON CONFLICT inserts → $c rows"
done
echo "# leg a counts: ${counts[*]}  (wanted 1 each)"
[ "$bad" = 0 ] || fail "leg a: a run left != 1 rows (${counts[*]})"

# (b) two scans eleven minutes apart → 2 rows. The second row is given an explicit
# scanned_at 11 minutes later; the generated shape derives bucket from it, the default
# shape must be given the bucket from that same timestamp (which is what a handler
# passing its own timestamp would do).
q -c "TRUNCATE qr_scans"
q -c "$NEW"
if [ "$SHAPE" = generated ]; then
  q -c "INSERT INTO qr_scans (short, ua_family, referrer, ip_hash, scanned_at) VALUES ('$SHORT','ios',NULL,'abc', now() + interval '11 minutes') ON CONFLICT (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL DO NOTHING;"
else
  q -c "INSERT INTO qr_scans (short, ua_family, referrer, ip_hash, scanned_at, bucket) VALUES ('$SHORT','ios',NULL,'abc', now() + interval '11 minutes', date_bin('10 minutes', now() + interval '11 minutes', $EPOCH)) ON CONFLICT (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL DO NOTHING;"
fi
cb="$(q -c 'select count(*) from qr_scans')"; echo "# leg b: two scans 11 minutes apart → $cb rows (wanted 2)"
[ "$cb" = 2 ] || fail "leg b: 11-minute-apart scans left $cb rows"
# same-bucket sanity: a third scan inside the first bucket is still swallowed
q -c "$NEW"; cb2="$(q -c 'select count(*) from qr_scans')"; [ "$cb2" = 2 ] || fail "leg b: a repeat in the current bucket inserted ($cb2 rows)"

# (c) two anonymous scans (NULL ip_hash) → 2 rows
q -c "TRUNCATE qr_scans"
NULLSTMT="INSERT INTO qr_scans (short, ua_family, referrer, ip_hash) VALUES ('$SHORT','ios',NULL,NULL) ON CONFLICT (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL DO NOTHING;"
q -c "$NULLSTMT"; q -c "$NULLSTMT"
cc="$(q -c 'select count(*) from qr_scans')"; echo "# leg c: two NULL-ip_hash scans → $cc rows (wanted 2)"
[ "$cc" = 2 ] || fail "leg c: anonymous scans deduped ($cc rows)"

cat <<REPORT

# ---- migration 0087 draft (shape that worked: $SHAPE) ----
$BUCKET_DDL
$IDX_DDL
# ---- landing.go statement ----
$NEW
REPORT
echo "✅ GREEN — bucket ($SHAPE) + partial unique index + ON CONFLICT DO NOTHING: 12×5 → 1 row each (${counts[*]}), 11 min apart → $cb, NULL ip_hash → $cc"
