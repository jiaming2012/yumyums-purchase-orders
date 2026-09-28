# Extraction — team-records-from-hand-runs

Outcome: confirmed

Approach used: a hand-authored committed per-run scorecard file at
`.night-crew/knowledge/scorecard/<run-id>.jsonl` — one
`{"schema":1,"kind":"run-scorecard",…}` line plus one
`{"schema":1,"kind":"team",…}` line per rostered role (product / delivery /
engineering / qa), shape mirrored from the CLI's own test fixture — read by
the installed binary's §15y.1 union read. Candidate shape for the ritual's
template, not an adoption (NFR-6).

Confirmed: the card's central premise — "emit the per-run scorecard files the
CLI **already reads**" — is TRUE for the installed binary (v3.4.0+1): the one
hand-written file renders a full table with all four roles as record-backed
rows (rating, value-per-token, points) plus the run-summary line; the Q-KR4
red is pinned on the real repo ("No runs to show.", committed dir holding only
`milestones.jsonl`); the real repo stayed byte-untouched.

Learned: no `team.toml` roster is required for record-backed rows — a roster
only adds unmeasured-team visibility (a rostered team with no record renders
as unmeasured instead of vanishing). Whether the card also seeds a roster is
optional build detail, shape documented at the clone's `internal/team/team.go`.

Plan change: the card's OR-arm ("if that provably requires CLI changes,
record the finding clone-side") is settled NOT NEEDED for rendering — the
card scopes to pure target-side ritual work: a template plus a closeout-ritual
step that emits one such `<run-id>.jsonl` per hand-run night, with the spike's
fixture as the candidate template. Q-KR4's "at least 1 record-backed row per
rostered team" is achievable from the ritual alone.
