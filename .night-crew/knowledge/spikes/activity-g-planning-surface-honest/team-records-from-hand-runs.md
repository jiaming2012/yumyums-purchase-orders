# Spikes — team-records-from-hand-runs

Activity: Activity G — Planning surface honest (carried QA debt)

> No `usm/roadmap.txt` here (see the Activity A ledgers for the convention);
> the script is the verdict. Read-only on the real repo — the hand-authored
> record goes to a scratch repo, and the real scorecard state is verified
> untouched. Instrument binary at run time: `night-crew v3.4.0+1`.

## The goal, and which legs need a spike

The card (roadmap Activity G, carried — Q-KR4's producer): the scorecard sees
no rostered role on this hand-run target. Scope: emit the per-run scorecard
files the CLI **already reads** from the hand-run slate/closeout ritual; if
that provably requires CLI changes, record the finding clone-side and close
target-side. The one premise worth falsifying first is the "already reads"
claim itself — discovered from the CLI source
(`cmd/nightcrew/scorecard.go`: the §15y.1 union of committed
`.night-crew/knowledge/scorecard/<run-id>.jsonl` files and local
`runs/<run-id>/metrics.jsonl`) and then proven against the INSTALLED binary,
because the binary, not the source, is what closes run.

## Spike: scorecard-reads-hand-run-records

- proves: a hand-authored `<run-id>.jsonl` in the committed location — one
  `{"kind":"run-scorecard"}` line plus four `{"kind":"team"}` lines (product /
  delivery / engineering / qa, the four roles the roadmap names) — makes the
  installed `night-crew scorecard` render all four team rows record-backed,
  with no CLI change. Three legs: (1) the Q-KR4 red is pinned on the real repo
  ("No runs to show." + an enumerated committed dir holding only
  `milestones.jsonl`); (2) the scratch repo with the one hand-written file
  renders all four roles and the run id; (3) the real repo's scorecard state
  is byte-untouched. A leg-2 failure IS the card's OR-arm finding (the CLI
  can't read any target-side file) and would be recorded clone-side — the
  spike makes that fork decidable before the card is slated.
- plan: baseline the real repo read-only; write the record in a scratch repo
  (shape mirrored from the CLI's own test fixture, `scorecard_test.go`); run
  `scorecard -repo <scratch>`; grep the four team rows + run id; verify the
  real dir unchanged. Roster note: `team.toml` does not exist on this target;
  the spike deliberately proves RECORDS render without one (the roster only
  adds unmeasured-team visibility) — whether the card also seeds a roster is
  build-time detail, shape documented at `internal/team/team.go`.
- script: .night-crew/spikes/activity-g-planning-surface-honest/team-records-from-hand-runs/01-scorecard-reads-hand-run-records.sh

## Verdict (run 2026-09-03, hand-run per the no-story-map convention)

- **scorecard-reads-hand-run-records: passed** — exit 0 on the first run.
  Leg 1: the Q-KR4 red pinned on the real repo — committed dir holds only
  `milestones.jsonl` (0 per-run files), `scorecard` renders "No runs to show."
  Leg 2: one hand-authored
  `.night-crew/knowledge/scorecard/hq-20260903-hand.jsonl` (1 run-scorecard +
  4 team records) in a scratch repo renders a full table — all four roles
  (product / delivery / engineering / qa) as record-backed rows with rating,
  value-per-token and points columns, plus the run-summary line ("4 of 4 work
  orders merged and standing; velocity 1.00"). Leg 3: real repo byte-untouched.

**Conclusion:** the card's "files the CLI already reads" premise is TRUE for
the installed binary (v3.4.0+1) — the OR-arm (CLI changes needed) is settled
NO for rendering; the card is pure target-side ritual work: emit one such
file per hand-run night from the slate/closeout ritual. Build-facts the card
inherits: the minimal sufficient shape is one `kind:"run-scorecard"` line plus
one `kind:"team"` line per role (fields as in the spike's fixture); no
`team.toml` is required for record-backed rows (a roster only adds
unmeasured-team visibility and is optional build detail).

## Corrections

- none — no agent-reached corrections. The script passed on its first run.

## Review

- n/a — no corrections, so no batch review is required for this goal (§4
  fires only when the ledger holds agent-reached corrections).
