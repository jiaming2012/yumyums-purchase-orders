#!/usr/bin/env bash
# 02-orderdetails-upsert-idempotent.sh — spike: the toast_orders table shape and
# its upsert key. Parses the repo-adjacent REAL sample (~/projects/yumyums/
# OrderDetails.csv), ENUMERATES duplicate "Order #" within a business date (a
# set, B-216 — the key's whole premise), then loads the rows TWICE into a
# throwaway database on the :5434 test cluster with
# ON CONFLICT (business_date, order_number) DO UPDATE and asserts the row count
# equals the distinct-key count. Reads Discount Amount / Amount into cents.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  idempotent; count == distinct keys; cents parse clean.
#   exit 1  a duplicate key within a date, or counts differ → key is wrong.
#   exit 2  could not run (:5434 container down, sample missing).
# Target: container yumyums-test-pg (:5434, role hqtest) — NEVER :5433. A
# throwaway database spike_h3_<stamp> is created and dropped on exit.
set -euo pipefail
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
SAMPLE="${ORDERDETAILS_CSV:-$HOME/projects/yumyums/OrderDetails.csv}"
[ -f "$SAMPLE" ] || cannot_run "sample OrderDetails.csv not found at $SAMPLE"
docker ps --format '{{.Names}}' | grep -qx yumyums-test-pg || cannot_run "yumyums-test-pg (:5434) not running — task test:db:up"
DB="spike_h3_$(date +%s)"
pq() { docker exec -i yumyums-test-pg psql -U hqtest -d "$1" -v ON_ERROR_STOP=1 -qtA; }
trap 'echo "DROP DATABASE IF EXISTS $DB" | pq postgres >/dev/null 2>&1 || true' EXIT
echo "# target coordinates: docker exec yumyums-test-pg (host :5434), role hqtest, throwaway db $DB"
echo "CREATE DATABASE $DB" | pq postgres >/dev/null
python3 - "$SAMPLE" > /tmp/spike-h3-rows.$$.sql <<'PY'
import csv, sys, re
from datetime import datetime
from collections import Counter
rows=list(csv.DictReader(open(sys.argv[1], newline='', encoding='utf-8-sig')))
def cents(v):
    v=(v or '0').replace('$','').replace(',','').strip()
    return int(round(float(v)*100)) if v else 0
def ts(v):
    for f in ('%m/%d/%y %I:%M %p','%m/%d/%Y %I:%M %p','%m/%d/%y %H:%M','%m/%d/%Y %H:%M'):
        try: return datetime.strptime(v.strip(), f)
        except Exception: pass
    raise SystemExit('UNPARSEABLE OPENED: '+repr(v))
keys=Counter()
out=[]
for r in rows:
    o=ts(r['Opened']); bd=o.date().isoformat()
    k=(bd, r['Order #'].strip()); keys[k]+=1
    out.append((bd, r['Order #'].strip(), r['Order Id'].strip(), o.isoformat(), cents(r['Amount']), cents(r['Discount Amount']), cents(r['Total']), (r['Voided'].strip().lower()=='true'), r.get('Order Source','').strip()))
dups=[k for k,c in keys.items() if c>1]
sys.stderr.write(f"# rows={len(rows)} distinct_keys={len(keys)} duplicate_keys={dups}\n")
sys.stderr.write(f"# order_number formats seen (set): {sorted(set(re.sub(r'[0-9]','9',k[1]) for k in keys))}\n")
sys.stderr.write(f"# business dates (set): {sorted(set(k[0] for k in keys))}\n")
if dups: sys.exit(10)
print(f"-- distinct={len(keys)}")
for t in out:
    esc=lambda s: s.replace("'","''")
    print(f"INSERT INTO toast_orders VALUES ('{t[0]}','{esc(t[1])}','{esc(t[2])}','{t[3]}',{t[4]},{t[5]},{t[6]},{str(t[7]).lower()},'{esc(t[8])}') ON CONFLICT (business_date, order_number) DO UPDATE SET amount_cents=EXCLUDED.amount_cents, discount_cents=EXCLUDED.discount_cents, total_cents=EXCLUDED.total_cents, voided=EXCLUDED.voided;")
PY
RC=$?; [ $RC = 0 ] || { [ $RC = 10 ] && fail "duplicate (business_date, order_number) keys in the real sample — the upsert key is wrong"; cannot_run "parse failed rc=$RC"; }
DISTINCT="$(sed -n 's/^-- distinct=//p' /tmp/spike-h3-rows.$$.sql)"
pq "$DB" <<'SQL' >/dev/null
CREATE TABLE toast_orders (business_date date not null, order_number text not null, order_id text not null, opened_at timestamptz not null,
 amount_cents integer not null, discount_cents integer not null default 0, total_cents integer not null, voided boolean not null default false,
 order_source text null, ingested_at timestamptz not null default now(), primary key (business_date, order_number));
SQL
grep -v '^--' /tmp/spike-h3-rows.$$.sql | pq "$DB" >/dev/null
grep -v '^--' /tmp/spike-h3-rows.$$.sql | pq "$DB" >/dev/null   # second arrival of the same report
N="$(echo 'select count(*) from toast_orders' | pq "$DB")"
SUMD="$(echo 'select coalesce(sum(discount_cents),0), coalesce(sum(amount_cents),0) from toast_orders' | pq "$DB")"
echo "# after loading the same file twice: rows=$N distinct_keys=$DISTINCT  (sum discount_cents|amount_cents = $SUMD)"
rm -f /tmp/spike-h3-rows.$$.sql
[ "$N" = "$DISTINCT" ] || fail "row count $N != distinct keys $DISTINCT after a double load — upsert is not idempotent"
echo "✅ GREEN — (business_date, order_number) is unique in real data and the double load leaves one row per order"
