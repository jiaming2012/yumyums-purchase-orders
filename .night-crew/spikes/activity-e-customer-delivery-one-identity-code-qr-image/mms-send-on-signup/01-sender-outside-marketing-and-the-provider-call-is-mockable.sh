#!/usr/bin/env bash
# 01-sender-outside-marketing-and-the-provider-call-is-mockable.sh — spike for card E2
# (mms-send-on-signup, D-KR1/D-KR2, decision 208). Four legs in a THROWAWAY worktree off `dev`:
#   (a) CONTROL   TestNothingInThisPackageSends passes on the unmutated tree
#   (b) MUTATION  a marketing file taking http.Post's address REDS the guard — a sender cannot live there
#   (c) SEAM      a throwaway internal/delivery package (Sender interface + SignalWire over LaML REST)
#                 passes its own tests against an httptest stub: path, basic auth, four form fields,
#                 sid on 201, error carrying the body on 401 — no credentials, no network
#   (d) IMPORT    a marketing file importing internal/delivery and holding a delivery.Sender passes
#                 the guard — the seam is legal because the package path carries no vendor name
# Go database hq_test_spike_e2_go on :5434 (TestMain migrates). NEVER :5433. No provider is called.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  (a) pass, (b) FAIL, (c) pass, (d) pass
#   exit 1  the guard let a sender into marketing, or the seam/import did not pass
#   exit 2  could not run
set -euo pipefail
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"
preflight
worktree_up
GUARD='TestNothingInThisPackageSends'
MKT="$WT/backend/internal/marketing"
MUT="$MKT/zz_spike_egress.go"

leg "(a) control — the guard on the unmutated worktree"
fresh_go_db
RC_A="$(go_run a ./internal/marketing "$GUARD")"; summary a
echo "#   control: exit=$RC_A pass=$(passes a) fail=$(fails a) skip=$(skips a)"
built a || cannot_run "the control did not compile — see $LOG/a.log"
[ "$RC_A" = 0 ] && [ "$(passes a)" = 1 ] && [ "$(fails a)" = 0 ] || cannot_run "control is not green (exit=$RC_A) — see $LOG/a.log"

leg "(b) mutation — http.Post's address taken inside internal/marketing"
cat >"$MUT" <<'GO'
package marketing

import "net/http"

// spike E2-01 (b): would a sender placed in this package get past the egress guard?
var spikeEgress = http.Post
GO
RC_B="$(go_run b ./internal/marketing "$GUARD")"; summary b
grep -E "outbound HTTP call|zz_spike_egress" "$LOG/b.log" | head -3 | sed 's/^/#   /' || true
echo "#   mutated: exit=$RC_B pass=$(passes b) fail=$(fails b)"
built b || cannot_run "the mutated tree did not compile — see $LOG/b.log"
rm -f "$MUT"
[ "$RC_B" != 0 ] && [ "$(fails b)" = 1 ] || fail "the guard stayed GREEN with http.Post in internal/marketing — a sender could ship there; the premise that it must live outside is wrong (see $LOG/b.log)"
grep -q "zz_spike_egress.go contains an outbound HTTP call" "$LOG/b.log" || fail "the guard went red for a different reason than the egress — see $LOG/b.log"

leg "(c) seam — internal/delivery with Sender + SignalWire, tests against an httptest stub"
mkdir -p "$WT/backend/internal/delivery"
cp "$SCRIPT_DIR/delivery/signalwire.go" "$WT/backend/internal/delivery/signalwire.go"
cp "$SCRIPT_DIR/delivery/signalwire_test.go" "$WT/backend/internal/delivery/signalwire_test.go"
( cd "$WT/backend" && go vet ./internal/delivery/ ) >"$LOG/c-vet.log" 2>&1 || { cat "$LOG/c-vet.log" >&2; fail "internal/delivery does not vet"; }
RC_C="$(go_run c ./internal/delivery "TestSpikeSignalWire")"; summary c
grep -E "SPIKE-E2-1c" "$LOG/c.log" | sed 's/^/#   /' || true
echo "#   seam: exit=$RC_C pass=$(passes c) fail=$(fails c)"
built c || cannot_run "internal/delivery did not compile — see $LOG/c.log"
[ "$RC_C" = 0 ] && [ "$(passes c)" = 2 ] && [ "$(fails c)" = 0 ] || fail "the provider call is not mockable as written — $(fails c) red (see $LOG/c.log)"

leg "(d) import — marketing holds a delivery.Sender; the guard must stay green"
cat >"$MKT/zz_spike_seam.go" <<'GO'
package marketing

import "github.com/yumyums/hq/internal/delivery"

// spike E2-01 (d): the seam the card wires — marketing imports the interface, never the vendor.
var spikeSender delivery.Sender
GO
RC_D="$(go_run d ./internal/marketing "$GUARD")"; summary d
echo "#   import: exit=$RC_D pass=$(passes d) fail=$(fails d)"
built d || cannot_run "marketing with the delivery import did not compile — see $LOG/d.log"
rm -f "$MKT/zz_spike_seam.go"
[ "$RC_D" = 0 ] && [ "$(passes d)" = 1 ] && [ "$(fails d)" = 0 ] || fail "the guard reds marketing for importing internal/delivery — the seam as designed is illegal (see $LOG/d.log)"

leg "restore"
rm -rf "$WT/backend/internal/delivery"
[ -z "$(git -C "$WT" status --porcelain)" ] || cannot_run "worktree not clean after restore: $(git -C "$WT" status --porcelain | tr '\n' ' ')"
main_tree_untouched

echo
echo "✅ GREEN on dev@$(git -C "$REPO_ROOT" rev-parse --short dev): the egress guard refuses a sender in internal/marketing (b),"
echo "   a vendor-free internal/delivery package passes its LaML-stub tests (c), and marketing may import its Sender (d)."
