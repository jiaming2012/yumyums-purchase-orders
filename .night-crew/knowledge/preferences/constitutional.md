# Preferences — constitutional (always escalate)

> Decisions that come to you no matter how many preferences an agent can cite.
>
> This list is yours to edit — add, remove, or retitle entries freely. Each carries a
> **Threshold**: `always`, or a bound above which it binds, so small routine variances
> stay out of your queue while the same subject above the bound does not.
>
> Seeded with a starting five. Tighten the thresholds to match what you actually want
> defended before the first unattended night.

## P-1 · Amending an artifact you signed

- **Preference:** changing the content of a signed slate, PRD, or sign-off table after you approved it escalates rather than being amended in place.
- **Why (operator):** your yes was scoped to what you read; silently widening it makes the signature meaningless.
- **Weight:** strong — this is the ceiling, not a leaning.
- **Threshold:** always
- **Recorded:** 2026-01-01

## P-2 · Spend and budget changes

- **Preference:** raising a budget cap, or spending past one, escalates.
- **Why (operator):** spend is yours to authorize; small routine variance is not worth waking you for.
- **Weight:** strong — above the threshold; below it, resolve and record.
- **Threshold:** above 10% of the run budget — edit this bound to what you actually want defended.
- **Recorded:** 2026-01-01

## P-3 · Deleting or rewriting landed work

- **Preference:** reverting merged work, rewriting history, or deleting a file that carries landed work escalates.
- **Why (operator):** landed work is the record; undoing it is not a repair, it is a decision.
- **Weight:** strong — this is the ceiling, not a leaning.
- **Threshold:** always
- **Recorded:** 2026-01-01

## P-4 · Pushes and promotions

- **Preference:** pushing to a remote, promoting a run branch, or publishing a release escalates.
- **Why (operator):** publication is irreversible from the outside; it is always your act.
- **Weight:** strong — this is the ceiling, not a leaning.
- **Threshold:** always
- **Recorded:** 2026-01-01

## P-5 · Changing the escalation rules themselves

- **Preference:** editing this constitutional list, or changing how severity is decided, escalates.
- **Why (operator):** a ceiling an agent can lower is not a ceiling.
- **Weight:** strong — this is the ceiling, not a leaning.
- **Threshold:** always
- **Recorded:** 2026-01-01


<!-- P-6 ported from the flat `preferences.md` §"Artifact naming" at morning triage
     2026-07-25 (ledger T-22). It was carried there as BINDING; the folder rubric accepts
     only strong/moderate/weak as the comparable keyword, so it is recorded HERE with
     `Threshold: always` — the store's actual mechanism for a rule that always binds —
     rather than being flattened to an ordinary strong leaning. Operator-approved at that
     triage. -->

## P-6 · Artifact naming — real authoring date plus a numeric collision suffix

- **Preference:** name run artifacts `slate-YYYYMMDD.md` / `overnight-YYYYMMDD` / `runs/YYYY-MM-DD-autonomous`, where `YYYYMMDD` is the real calendar date the slate is authored, taken from the system clock — never "tomorrow," never inferred from the previous artifact's name. The first run of a real date carries NO suffix; the second is `-2`, the third `-3`, and so on, a numeric collision counter for that date and nothing more. Sort run ids with `sort -V`, never plain `sort`, which orders `-10` before `-2` and puts an unsuffixed run after its own `-2`. Every date appearing in prose is the real calendar date, always — sign-off dates, triage dates, decision dates. Labels live in filenames and branch names only.
- **Why (operator):** adopted 2026-07-22 as the fleet standard, superseding the 2026-07-20 cycle-letter rule. The letter rule was an hq-local invention the fleet matcher never recognised — `^overnight-[0-9]{8}(-[0-9]+)?$` — which is precisely why `overnight-20260720c` was silently skipped by `/nc-status`. hq conforms so the fleet tooling reads hq's runs instead of stepping over them.
- **Weight:** strong — binding, which is why it sits on this list rather than in `process.md`: it is not a leaning to be outweighed by other preferences.
- **Threshold:** always
- **Evidence:** night-crew's `fix-overnight-ergonomics` change and the matchers in the `nc-status`, `nc-morning-triage`, `nc-slate-plan` and `nc-run` skills. The drift this replaced, and the legacy-artifact reading rule that goes with it — *treat any pre-2026-07-22 label as an opaque identifier, not a date* — are retained in full in the appendix of `.night-crew/knowledge/preferences.md`.
- **Recorded:** 2026-07-22

<!-- ⚠ DEPLOYMENT CAVEAT retained from the flat file, verified 2026-07-22 and NOT yet
     re-verified: night-crew `main`'s `nc-status` and the installed
     `~/.claude/skills/nc-status` may still carry the OLD matcher `^overnight-[0-9]+$`,
     which skips a `-N` suffix just as it skipped a letter (the `-` breaks `[0-9]+$`).
     Per [[nc-tooling-tracks-main]], hq rituals track main — so P-6 is the go-forward
     convention regardless, and the tooling upgrade is the separate, blocking half. Do not
     expect `/nc-status` to see numeric hq runs until that change reaches main AND the
     user-level skills are re-synced. -->
