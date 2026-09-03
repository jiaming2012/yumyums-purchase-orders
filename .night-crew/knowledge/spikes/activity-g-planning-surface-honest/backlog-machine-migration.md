# Spikes — backlog-machine-migration

Activity: Activity G — Planning surface honest (carried QA debt)

> No `usm/roadmap.txt` here (see the Activity A ledgers for the convention);
> the scripts are the verdict. Both scripts are READ-ONLY on the real
> `BACKLOG.md` — every write goes to a scratch copy, and both scripts
> sha-verify the real document untouched before exiting green. Instrument
> binary at run time: `night-crew v3.4.0+1`.

## The goal, and which legs need a spike

The card (roadmap Activity G, carried from last cycle — Q-KR3's producer):
reshape the ~193 legacy-shape entries to canonical `B-NN` form until
`night-crew backlog check` exits 0, with content preservation proven, then arm
the triage §4.5 gate. Two premises worth falsifying before a night is spent:
that the red is real and countable with the instruments the done_when names,
and that the reshape→check loop actually works on a scratch copy (isolation,
a real legacy entry greening, and a mechanical content-preservation proof).

## Spike: check-red-baseline

- proves: the migration's measuring instruments work and the red is real,
  enumerated, and pinned: `backlog check` exits non-zero on the real document
  with a parseable "N issue(s) across M entries" line (at authoring: 297/207);
  `backlog list` exits 0 and emits a countable set (the done_when comparator's
  left-hand side); both counts print side by side; and the document's sha256
  is identical before and after. If `check` unexpectedly PASSES, the spike
  fails — the premise "there is a migration to do" would be dead and the card
  moot.
- plan: run both verbs with `-repo`, parse the CLI's own enumeration (never a
  hand count), print the baseline, sha-verify.
- script: .night-crew/spikes/activity-g-planning-surface-honest/backlog-machine-migration/01-check-red-baseline.sh

## Spike: reshape-sample-greens

- proves: the card's working loop is viable end to end on a scratch copy:
  (1) **isolation** — `backlog check --file <copy>` demonstrably reads the
  copy (an appended entry moves the copy's counts; the real sha never moves);
  (2) **reshape** — the sample legacy entry **B-90** (an extra
  `**destination: …**` segment where the rubric wants `origin · status ·
  lead`) is mechanically reshaped by folding the destination text into the
  lead, and its issues disappear while the total strictly drops;
  (3) **preservation** — token-multiset containment: every alphanumeric token
  of the original entry survives the reshape (words may move, never vanish) —
  the same proof shape the card owes at scale;
  (4) the real document is byte-untouched end to end.
  One entry by design: the spike proves the LOOP; the card does the other
  ~200 entries with it.
- plan: copy → check --file → append-probe → reset copy → python3 surgery on
  the B-90 bullet → re-check → token-multiset diff → sha-verify.
- script: .night-crew/spikes/activity-g-planning-surface-honest/backlog-machine-migration/02-reshape-sample-greens.sh

## Verdict (run 2026-09-03, hand-run per the no-story-map convention)

- **check-red-baseline: passed** — exit 0. `backlog check` exits 1 with
  `backlog invalid: 297 issue(s) across 207 entries` (parsed: issues=297,
  entries=207); `backlog list` exits 0 emitting 208 lines / 144 handle-bearing;
  document sha256 `342c96cd…e31e5f0b` identical before and after. The card's
  baseline is pinned: 297 issues to retire, and the done_when comparator's two
  sides currently read 208 (list) vs 207 (check-entries) — the card should
  anchor "document entry count" on the checker's own parse, not a hand grep.
- **reshape-sample-greens: passed** — exit 0, first run after the correction
  below was applied pre-run. Isolation: appending a probe entry moved the
  COPY's entry count 207→208 while the real document's sha never moved — the
  `--file` loop is safe to iterate. Reshape: B-90's
  `**destination: …**` segment folded into its lead greens B-90 under the real
  checker and drops the total 297→296. Preservation: all 401 alphanumeric
  tokens of the original entry survive (multiset containment) — the proof
  shape the card owes at scale works.

**Conclusion:** the migration card's loop — reshape a scratch copy, re-check
with `--file`, prove preservation mechanically — is proven viable with the
installed CLI. Build-facts the card inherits: (1) fold-don't-delete is a
sufficient reshape for the "extra segment" defect class; (2) use the checker's
"across M entries" as the entry-count side of done_when; (3) beware `head`
after pipes carrying this document's multi-KB lines — see Corrections.

## Corrections

- **SIGPIPE in the spike's own display plumbing (both scripts), fixed and
  re-run green.** First execution of `01-check-red-baseline.sh` died exit 141:
  `echo "$LIST_OUT" | head -3` — the document's multi-kilobyte single-line
  entries mean `echo` is still writing when `head -3` exits, `pipefail`
  surfaces the SIGPIPE as 141, and `set -e` kills the script mid-leg. Fixed by
  replacing early-exit consumers with full-readers (`sed -n '1,3p' | cut
  -c1-160`) in 01, and the same latent shape in 02 (`grep | head -3`) before
  its first run. The premises and assertions are unchanged — this was the
  harness talking over itself, not a finding about the backlog. Carried
  forward as build-fact (3) above because the CARD's own tooling will pipe
  this same document.

## Review

- signed: operator, 2026-09-03 — covers 1 correction(s) (batch sitting with
  Activities A/G; "Sign off all three").
