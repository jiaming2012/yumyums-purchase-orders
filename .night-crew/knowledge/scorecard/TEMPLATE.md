# Per-run scorecard record — TEMPLATE (hand-run nights)

The closeout of every hand-run overnight emits ONE file
`.night-crew/knowledge/scorecard/<run-id>.jsonl` (e.g. `20260904.jsonl`) built from
the block below. The installed CLI (`night-crew scorecard`, ≥ v3.4.0+1) already reads
these committed per-run files — no CLI step, no `team.toml` roster required
(spike `team-records-from-hand-runs`, passed 2026-09-03).

🛑 This file is deliberately **`.md`, not `.jsonl`** — any `*.jsonl` in this
directory (other than `milestones.jsonl`) is read by the scorecard's union read as a
real run. Never commit a template, example, or validation record as `*.jsonl` here.
A `<run-id>.jsonl` is a CLOSING artifact in the run-evidence oracle: emit it exactly
once per run, at closeout, never earlier, never for a run that didn't happen.

## The record — 5 JSONL lines (1 run-scorecard + 1 team line per role)

Copy this block verbatim into `<run-id>.jsonl`, then substitute every field per the
table below. One JSON object per line, no blank lines, no trailing commas.

```jsonl
{"schema":1,"kind":"run-scorecard","run_id":"YYYYMMDD","ts":"YYYY-MM-DDTHH:MM:SSZ","work_orders":0,"points_committed":0,"points_completed":0,"velocity":0,"first_pass_rate":0,"merges_standing":0,"regressions":0}
{"schema":1,"kind":"team","run_id":"YYYYMMDD","ts":"YYYY-MM-DDTHH:MM:SSZ","team":"product","rating":0,"points_completed":0,"value_per_token":0.01}
{"schema":1,"kind":"team","run_id":"YYYYMMDD","ts":"YYYY-MM-DDTHH:MM:SSZ","team":"delivery","rating":0,"points_completed":0,"value_per_token":0.01}
{"schema":1,"kind":"team","run_id":"YYYYMMDD","ts":"YYYY-MM-DDTHH:MM:SSZ","team":"engineering","rating":0,"points_completed":0,"value_per_token":0.01}
{"schema":1,"kind":"team","run_id":"YYYYMMDD","ts":"YYYY-MM-DDTHH:MM:SSZ","team":"qa","rating":0,"points_completed":0,"value_per_token":0.01}
```

## Field fill rules

| Field | Fill with |
|---|---|
| `run_id` (all 5 lines) | The run's id, e.g. `20260904` — must equal the filename stem |
| `ts` (all 5 lines) | Closeout time, UTC, RFC 3339 (`date -u +%Y-%m-%dT%H:%M:%SZ`) |
| `work_orders` | Cards on the signed slate for this run |
| `points_committed` | Sum of the slate's card point estimates (slate doc; if the slate carries no points, use 2 per card) |
| `points_completed` | Points for cards that landed green on the run branch |
| `velocity` | `points_completed / points_committed`, 2 decimals (1 = everything landed) |
| `first_pass_rate` | Fraction of landed cards that needed no rework leg, 2 decimals |
| `merges_standing` | Cards merged to the run branch and still standing at closeout |
| `regressions` | Known regressions introduced by this run at closeout time (normally 0; a triage-found escape is triage's to record, not closeout's) |
| `team` | Exactly one line each: `product`, `delivery`, `engineering`, `qa` — the four rostered roles |
| `rating` | Whole number 0–100, the role's night grade: product = slate/PRD fit of what shipped; delivery = dispatch/budget/merge orchestration; engineering = implementation legs; qa = gates, red-first discipline, evidence quality |
| `points_completed` (team lines) | The run's `points_completed` split across the four roles by where the night's effort landed; the four must sum to the run line's value |
| `value_per_token` | `points_completed / estimated tokens spent by that role`; when no token accounting exists for a hand-run night, use the nominal `0.01` |

## Verify before the closing commit

```
night-crew scorecard --repo .
```

Must exit 0, render the run id, and show all four roles as record-backed rows —
no "No runs to show.". Fix a failed render before committing; never skip
(a skipped ritual step reads exactly like a clean one — B-133).
