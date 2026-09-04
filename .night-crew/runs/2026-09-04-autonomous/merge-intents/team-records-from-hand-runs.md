# Merge intent — Card 4 · `team-records-from-hand-runs` (run 20260904)

Branch: `wo-team-records-from-hand-runs` (cut from `overnight-20260904` @ 846e572, after
Cards 1–3 merged). Card authority: slate-20260904 Card 4. Q-KR4's producer, carried two
cycles. DOCS/RITUAL card — no app code, no database. The spike
(`.night-crew/knowledge/spikes/activity-g-planning-surface-honest/team-records-from-hand-runs.md`,
passed 2026-09-03, 0 corrections) settled the OR-arm: the installed CLI (v3.4.0+1)
ALREADY READS hand-authored per-run records — pure target-side ritual work. The gate IS
the CLI verb's exit codes plus the validation render recorded below.

## Shared files touched

- `.night-crew/knowledge/COMMANDS.md` — the CLOSEOUT scorecard-record step, added as a
  stanza under step **5** (the overnight run / launch-prompt step) — the closeout
  moment, where the control loop that ends the night actually reads. ⚠️ Card 3 (merged
  before this branch was cut) added a **§4.5 gate stanza to step 6 (TRIAGE)** — a
  DIFFERENT ritual moment. **On any merge conflict: keep BOTH additions** — Card 3's
  §4.5 backlog gate under step 6 AND this card's closeout stanza under step 5; they do
  not overlap semantically. Log any merge in the conflict log.
- `.night-crew/knowledge/scorecard/TEMPLATE.md` — the per-run record template
  (**`.md`, deliberately NOT `.jsonl`** — design call below). New file; no conflict
  surface expected.
- `.night-crew/knowledge/roadmap.md` — one-line status flip of the
  `team-records-from-hand-runs` card (PLANNED → DRAFTING), matching Cards 1–3's flip
  convention. No other roadmap line moves.
- `.night-crew/runs/2026-09-04-autonomous/merge-intents/team-records-from-hand-runs.md`
  — this file (amended as evidence lands).
- `.night-crew/runs/2026-09-04-autonomous/card4-*.log` — committed whole logs, each
  ending in its `EXIT=` line: before-red, template-inertness proof, validation render.
- `night-crew.toml` — **nothing here** (planning docs match no `[e2e.seams]` key; no
  spec subset is owed by this footprint).
- App code, `sw.js`, `package.json`, `backend/`, `tests/`, `supabase/` — **nothing
  here** (untouched).
- `.night-crew/knowledge/spikes/**`, `.night-crew/spikes/**` — **nothing here**
  (read-only inputs: the spike ledger + extraction are this card's build facts).
- `.night-crew/knowledge/BACKLOG.md` — **nothing here** (Card 3's exclusive subject;
  untouched by this branch).

## Design calls (stated per the card)

1. **Template location + extension:** the template lives INSIDE
   `.night-crew/knowledge/scorecard/` (maximally discoverable — beside where the
   closeout writes) but as **`TEMPLATE.md`**, not `*.jsonl`. The CLI's §15y.1 union
   read enumerates per-run `*.jsonl` files in that dir; ANY `.jsonl` there — even a
   placeholder-id one — risks rendering as a bogus run row forever. A `.md` file is
   inert to the union read (proven empirically: `card4-green-template-inert.log`
   shows `scorecard --repo .` still renders "No runs to show." with the template
   present). The JSONL shape lives in a fenced block inside the template, with a
   placeholder run id (`YYYYMMDD`) that the closeout substitutes.
2. **Roster:** no `team.toml` is seeded — spike-proven optional (records render
   record-backed without one; a roster only adds unmeasured-team visibility). Skipped
   per the card's explicit permission.
3. **No real-run-id record committed:** `.night-crew/knowledge/scorecard/<runid>.jsonl`
   is a CLOSING artifact in the run-evidence oracle. This branch commits **no**
   `*.jsonl` under the scorecard dir (`milestones.jsonl` pre-exists, untouched). The
   validation record (`00000000-validation.jsonl`) existed only transiently in the
   working tree during the validation render and was deleted before the landing
   commit — `git ls-files .night-crew/knowledge/scorecard/` proof recorded below.
   Tonight's `20260904.jsonl` is emitted by the RUN'S CLOSEOUT (the control loop),
   following the COMMANDS.md step verbatim — not by this card.

## What must survive any merge

- BOTH COMMANDS.md ritual stanzas — Card 3's §4.5 triage gate (step 6) AND this
  card's closeout scorecard-record stanza (step 5). Different ritual moments; keep
  both, never either/or.
- `.night-crew/knowledge/scorecard/TEMPLATE.md` whole — the closeout step references
  it by exact path; the control loop follows it verbatim tonight.
- The roadmap status flip; the `card4-*.log` evidence files; this intent.
- The ABSENCE of any committed `scorecard/*.jsonl` besides the pre-existing
  `milestones.jsonl` — if a merge somehow introduces a `20260904.jsonl` from another
  source before closeout, that is the control loop's closing artifact, not this
  card's; do not attribute it here.

## What is safe to drop

- Nothing committed on this branch is scratch. The validation record was never
  committed (deleted pre-landing; see Design call 3). Run-dir logs are append-only
  evidence — in a conflict keep both sides.

## Red-first

**n/a — no code change** (docs/ritual card form). The Q-KR4 red is pinned by the
spike (run 2026-09-03, leg 1): on the real repo the committed scorecard dir holds only
`milestones.jsonl` (0 per-run files) and `night-crew scorecard` renders
**"No runs to show."** — no team table at all, so all four roles (product / delivery /
engineering / qa) are trivially NOT record-backed.

Re-observed on THIS worktree before any change (`card4-red-scorecard-before.log`):

- `night-crew scorecard --repo .` → **EXIT=0**, output verbatim:
  `# Team scorecard` / `No runs to show.`
- Committed dir enumerated in the log: `milestones.jsonl` only.
- Binary: `night-crew v3.4.0+1` — same instrument the spike pinned.

## Validation evidence (mechanism proven WITHOUT a closing artifact)

- **Template inertness:** with `TEMPLATE.md` present in the scorecard dir,
  `night-crew scorecard --repo .` → EXIT=0, still "No runs to show." — the template
  cannot be mistaken for a run record (`card4-green-template-inert.log`).
- **Validation render:** a transient `00000000-validation.jsonl` (fake run id, built
  by following the TEMPLATE.md instructions verbatim) placed in the scorecard dir →
  `night-crew scorecard --repo .` → EXIT=0, full table renders: run `00000000` +
  all four roles (product / delivery / engineering / qa) as record-backed rows
  (`card4-green-scorecard-validation.log`). File deleted immediately after capture;
  `git ls-files .night-crew/knowledge/scorecard/` at landing lists exactly
  `TEMPLATE.md` + `milestones.jsonl`.

**Result (landed):** both greens hold as described. The validation render's Teams
table, verbatim rows: `product / delivery / engineering / qa`, each `Runs 1 ·
Rating 80 · Value per token 0.01 · Points done 2 · Regressions 0`, plus the
run-summary line `Latest run 00000000: 4 of 4 work orders merged and standing;
velocity 1.00; first-pass rate 1.00` — EXIT=0. The record was built by following
`TEMPLATE.md`'s copy-and-substitute instructions literally, so the render also
proves the template is followable verbatim (what the control loop does tonight).
Cleanup proof at landing: working tree + index hold no `scorecard/*.jsonl` besides
the pre-existing `milestones.jsonl`.

## Commit plan (stated per mechanics rule 2)

1. This merge-intent + the before-red log (FIRST commit, before any change).
2. The template (`scorecard/TEMPLATE.md`) + the COMMANDS.md closeout stanza.
3. Validation evidence logs + roadmap flip + this intent's evidence sections filled.

All commits carry trailer `Night-Crew-Run: 20260904` and name
`team-records-from-hand-runs` in the body.
