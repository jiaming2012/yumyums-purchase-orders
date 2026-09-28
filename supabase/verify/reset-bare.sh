#!/usr/bin/env bash
# reset-bare.sh — test-fixture action: drop ONLY this card's objects from the
# throwaway spike-supabase substrate, leaving everything else in the stack
# untouched. Used to stage the bare state for 01-structure.sh's fresh leg and
# for the red-first probe (`01-structure.sh --assert-only` against bare).
#
# 🛑 This is NOT a substrate teardown and it is NOT `env-up.sh --fresh`.
# It touches four tables in the LOCAL throwaway stack and nothing else.
#
# exit 0  the card's objects are gone (or were never there)
# exit 2  could not run

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

substrate_up
echo
echo "── reset to bare: dropping this card's objects only ──"
reset_bare
echo "  dropped (if present): public.scan_attempts, public.codes, public.campaigns, public.marketing_settings"
echo
echo "✅ bare state staged (throwaway substrate; nothing else touched)"
exit 0
