#!/usr/bin/env bash
# demo-sync.sh — THE MILESTONE CLOSE-BAR DEMO. Night-crew card `demo-sync-target`
# (Activity 5, run 20260809). Delivered as `task demo:sync`.
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE. And "could not
#    run" is a DISTINCT outcome from "ran and failed" — that distinction is the
#    entire reason this milestone exists.
#
#   exit 0   GREEN. One field written through HQ's REAL write path
#            (POST /api/v1/workflow/saveResponse — real session cookie, real
#            auth middleware, real grant gate, real repository SQL, against a
#            Postgres carrying HQ's REAL migrations) surfaced in an RxDB-served
#            READ on ONE real checklist, within the stated bound. Round trip
#            proven.
#
#   exit 1   RAN AND THE ROUND TRIP FAILED. The field was written and landed in
#            HQ's Postgres, but never surfaced in the RxDB-served read within the
#            bound (relay absent, value never carried). 🛑 THIS IS A FIRST-CLASS
#            RECORDABLE OUTCOME — a FINDING, not a park. The tri-state exit exists
#            precisely so "ran and failed" is distinguishable from a silent no-op.
#
#   exit 2   COULD NOT RUN. A setup/infrastructure/precondition failure — Docker
#            down, the Spike A stack would not come up, the scratch Postgres never
#            became healthy, HQ's migrator failed, login failed, a missing
#            coordinate. 🛑 NO VERDICT IS RENDERED. It says nothing about whether
#            the round trip closes, and must NEVER be reported as red. A silent
#            no-op is the exact class this whole milestone exists to retire; an
#            honest exit-2 is its opposite.
#
#   exit 3   A verdict was reached, but Spike A's SHARED substrate could not be
#            restored (rows left in hq_sync_checklists would red
#            backend/internal/sync TestJWTBridgeRLS). Repair before trusting it.
#
#   exit 64  usage error.
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 GATE ON THIS SCRIPT DIRECTLY, NEVER ON `task demo:sync`.
#    go-task returns its OWN status (201) when a wrapped command fails, so a red
#    or a could-not-run from this script becomes an indistinguishable "201" at
#    the `task` boundary. That is B-163's lesson, applied from the start. The
#    Taskfile's `demo:sync` target is a THIN WRAPPER whose only job is to invoke
#    this file; the verdict is THIS file's exit code and it comes from exactly
#    one place.
#
# ═══════════════════════════════════════════════════════════════════════════
# WHAT THIS DEMO IS, AND WHY IT IS A RE-EXPORT RATHER THAN A FORK
#
# The close bar names a round trip that Spike C ALREADY BUILT AND PROVED:
#
#     POST /api/v1/workflow/saveResponse  (real write path)
#       -> HQ Postgres
#       -> NOTIFY spike_c_relay  (trigger, no write-path edit)
#       -> Go relay              (backend/cmd/spikec-relay, LISTEN/NOTIFY)
#       -> PostgREST             (service identity, spike B's proven lane)
#       -> hq_sync_checklists    (Spike A's substrate)
#       -> a RUNNING RxDB client (rxdb/spike-c-read.js)   ... at 248 ms.
#
# spike-c-roundtrip.sh implements that round trip AND the tri-state exit contract
# above, byte-for-byte. It brings up Spike A's stack via env-up.sh in RECONCILE
# mode (idempotent, destroys nothing), stands up a FRESH scratch HQ Postgres on a
# Docker-assigned EPHEMERAL port (never :5432/:5433/:5434 — :5433 is the
# PRODUCTION cluster), applies HQ's REAL migrations with HQ's OWN binary, logs in
# for real, writes one field through the real handler, and measures whether it
# surfaces in the RxDB read. It restores Spike A's shared tables afterwards and
# VERIFIES the restore on every path.
#
# The demo's job is to NAME that round trip as the milestone's deliverable and
# make it reproducible as `task demo:sync`, NOT to re-derive it. Forking ~630
# lines of thrice-debugged harness (orphan-server refusal, ephemeral-port
# isolation, substrate-restore verification — each fix cost a leg to learn) would
# create a SECOND copy of the verdict that could drift from the first. So this
# file delegates: the verdict comes from spike-c-roundtrip.sh, from exactly one
# place, and this file's exit code IS its exit code, passed through unchanged.
#
# ───────────────────────────────────────────────────────────────────────────
# READ-SURFACE DECISION (engineer-level, in-card choice — see the card's
# fallback clause; this is NOT a park):
#
# The card's PRIMARY ask is C3's real browser fill-view RxDB read; the DOCUMENTED
# FALLBACK is C2's `#sync-one-row`. This demo uses the RxDB READ CLIENT
# (rxdb/spike-c-read.js) as its read surface — the C2 fallback in spirit.
#
#   * C2/C3's browser read surfaces are driven by the vendored supabase-js client,
#     and in EVERY existing test (tests/sync-one-row.spec.js,
#     tests/sync-fill-view.spec.js) the substrate is a `page.route` STUB. No test
#     ever points the BROWSER read path at the REAL Spike A substrate:
#     HQ_SYNC_REST_URL / HQ_SYNC_REALTIME_URL are unset in every environment
#     (sync-rxdb/bootstrap.js).
#   * Driving the REAL browser fill-view against the REAL substrate for the demo
#     would need novel, unproven integration (inject substrate URLs into a served
#     page, a Playwright Chromium, a rewriting proxy, plus the live relay) —
#     exactly the "too heavy for a clean demo" the fallback clause anticipates.
#   * rxdb/spike-c-read.js uses the IDENTICAL `replicateSupabase` RxDB plugin the
#     browser's `startHQReplication` uses, pointed at the REAL Spike A PostgREST +
#     Realtime — a genuine RxDB-served read on one real checklist, against the
#     real substrate, proven green. It satisfies the close-bar letter: "one real
#     checklist", "RxDB-served read", "real /saveResponse write path".
#
# DECISIONS CARRIED VERBATIM (the call site's contract, not this file's to
# reopen):
#   * Decision 126 — RxDB serves READS; /saveResponse + /submitChecklist keep
#     owning ALL writes. This demo touches no write path.
#   * Decision 105 — scoped read, never whole.
#   * Spike E condition T-42 — no polling, no business-watermark resync; the relay
#     stays trigger/NOTIFY-driven (the reused mechanism is exactly this).
#
# ═══════════════════════════════════════════════════════════════════════════
# ⚠ NEVER `task spike:down`. It is the ONLY data-destroying spike target. This
#   demo consumes Spike A's stack in RECONCILE mode via env-up.sh and never
#   destroys it. The scratch HQ Postgres it stands up IS created and destroyed
#   inside one run (that is spike-c-roundtrip.sh's own scratch container, project
#   `spike-c-hq`), so the demo is re-runnable from nothing.
#
# USAGE
#   .night-crew/qa/spike-supabase/demo-sync.sh                   # the demo (green expected)
#   .night-crew/qa/spike-supabase/demo-sync.sh --break-roundtrip # RED-FIRST: relay deliberately
#                                                                 #   absent; MUST exit 1 (ran and
#                                                                 #   failed), not 0, not 2.
#   .night-crew/qa/spike-supabase/demo-sync.sh --keep            # leave the scratch pg + server up
#   .night-crew/qa/spike-supabase/demo-sync.sh --fresh-substrate # also rebuild Spike A from nothing
#
#   or via the repo Taskfile:  task demo:sync   /   task demo:sync:red
#   🛑 but GRADE on THIS script's exit code, not on `task`'s (see the B-163 note above).

set -euo pipefail

DEMO_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROUNDTRIP="$DEMO_DIR/spike-c-roundtrip.sh"

# ---------------------------------------------------------------------------
# Argument translation. The demo's public vocabulary is close-bar English; it
# maps onto the harness's proven flags. `--break-roundtrip` is the demo's name
# for the red-first capture — the same mechanism-absent run spike-c-roundtrip.sh
# calls `--no-relay`. Everything else passes through.
#
# 🛑 We do NOT re-implement the tri-state gate here. The harness owns it, and a
# second copy could disagree with the first. This block only RENAMES a flag.
# ---------------------------------------------------------------------------
FORWARD=()
BREAK=0
for arg in "$@"; do
  case "$arg" in
    --break-roundtrip) BREAK=1; FORWARD+=(--no-relay) ;;
    --no-relay)        BREAK=1; FORWARD+=(--no-relay) ;;   # accept the harness spelling too
    --keep)            FORWARD+=(--keep) ;;
    --fresh-substrate) FORWARD+=(--fresh-substrate) ;;
    -h|--help)
      sed -n '1,120p' "$0" | grep -E '^# ' | sed 's/^# \{0,1\}//'
      exit 64 ;;
    *)
      echo "usage: $(basename "$0") [--break-roundtrip] [--keep] [--fresh-substrate]" >&2
      exit 64 ;;
  esac
done

# 🛑 The one precondition this wrapper asserts itself, because a MISSING harness
# is a could-not-run (exit 2) and must not surface as go-task's 201 or as a
# confusing `bash: ... : No such file` three lines down. Everything ELSE that
# could fail to run — Docker down, stack won't come up, a missing coordinate —
# is the harness's own exit-2 territory, reported with its cause, and passed
# through unchanged below.
if [ ! -x "$ROUNDTRIP" ]; then
  printf '\n🛑 COULD NOT RUN (not a verdict) — the round-trip harness is missing or not executable: %s\n' "$ROUNDTRIP" >&2
  exit 2
fi

printf '════════════════════════════════════════════════════════════════════\n'
printf '  task demo:sync — the milestone close-bar demo (card demo-sync-target)\n'
printf '  run 20260809 · Activity 5\n'
printf '\n'
printf '  ONE field, written through the REAL write path\n'
printf '    (POST /api/v1/workflow/saveResponse), surfacing in an RxDB-served READ\n'
printf '    on ONE real checklist, round-trip. Reuses Spike C'"'"'s proven harness.\n'
printf '\n'
printf '  Carries: decision 126 (RxDB serves READS; writes keep owning writes),\n'
printf '           decision 105 (scoped read, never whole),\n'
printf '           spike E T-42 (trigger/NOTIFY-driven; no polling).\n'
printf '  Read surface: rxdb/spike-c-read.js (C2 fallback in spirit — see header).\n'
printf '\n'
if [ "$BREAK" = "1" ]; then
  printf '  MODE: --break-roundtrip (RED-FIRST). The relay is deliberately absent.\n'
  printf '        Both sides are real; nothing bridges them. This MUST exit 1\n'
  printf '        (ran and the round trip failed), never 0 and never 2.\n'
else
  printf '  MODE: demo (green expected).\n'
fi
printf '\n'
printf '  🛑 GRADE ON THIS SCRIPT'"'"'S EXIT CODE, NOT ON `task` (B-163: go-task\n'
printf '     returns 201 on a failing command). 0=green 1=ran-and-failed 2=could-not-run.\n'
printf '════════════════════════════════════════════════════════════════════\n'

# ---------------------------------------------------------------------------
# Delegate. The harness's exit status IS the demo's verdict — 0/1/2/3 map
# straight through. We do NOT wrap this in a subshell that could swallow the
# status, and we do NOT `|| true` it (that would turn a red or a could-not-run
# into a false green — the silent no-op this milestone exists to retire).
# `set -e` would exit on a non-zero here before we can echo the mapping, so we
# capture the code explicitly and re-exit with it.
# ---------------------------------------------------------------------------
set +e
"$ROUNDTRIP" "${FORWARD[@]}"
RC=$?
set -e

printf '\n════════════════════════════════════════════════════════════════════\n'
case "$RC" in
  0) printf '  ✅ demo:sync VERDICT: GREEN (exit 0) — the round trip closes.\n' ;;
  1) printf '  🛑 demo:sync VERDICT: RED (exit 1) — RAN AND THE ROUND TRIP FAILED.\n'
     printf '     This is a first-class recorded outcome (a FINDING), not a park and\n'
     printf '     not a could-not-run. The write landed in HQ Postgres and did not\n'
     printf '     surface in the RxDB-served read within the bound.\n' ;;
  2) printf '  🛑 demo:sync: COULD NOT RUN (exit 2) — NO VERDICT. Infra/precondition\n'
     printf '     failure; see the harness output above for the named leg. This says\n'
     printf '     NOTHING about whether the round trip closes.\n' ;;
  3) printf '  🛑 demo:sync: exit 3 — a verdict was reached but Spike A'"'"'s shared\n'
     printf '     substrate could not be restored. Repair before trusting anything.\n' ;;
  *) printf '  🛑 demo:sync: the harness exited %s, outside its documented 0/1/2/3\n' "$RC"
     printf '     contract. Treating as could-not-run.\n' ;;
esac
printf '════════════════════════════════════════════════════════════════════\n'

exit "$RC"
