# Merge intent — S · `spike-b-migration-rehearsal`

Run `20260807` · branch `card/s-spike-b-migration` · stretch card · feeds **D-KR1** (2nd of 4 spike
verdicts) · roadmap card `spike-b-migration-rehearsal`, Activity 2.

---

## Scope

Spike B from the governing handoff §5: stand up a Postgres whose schema mimics HQ's with a small
subset of fields, fixture it, and **migrate that fixtured data across** into spike A's already-proven
Supabase substrate — then prove it surfaces in a real RxDB client. This is the leg nine nights of
planning were built on without ever being executed.

**The verdict is the script's exit status, never prose** (the spike↔script rule, roadmap.md:50).
`.night-crew/qa/spike-supabase/spike-b-migration.sh` — exit 0 = HQ-shaped data landed in the
substrate and surfaced in RxDB; non-zero = it did not, with the failing leg named. **A RED verdict is
a successful spike** and is recorded, not ground against.

---

## Shared files touched

| File | Why |
|---|---|
| `Taskfile.yml` | One new, **disjoint** `spike:migration` stanza (plus its `:fresh` variant if needed), added at the END of the existing `spike:*` block so the card is runnable the same way spike A is. |
| `.night-crew/knowledge/roadmap.md` | Card status flip for `spike-b-migration-rehearsal`, per that file's convention. For a spike card the flip records **the verdict delivered — red or green — not "success"**. |
| `.night-crew/knowledge/BACKLOG.md` | Only if the spike surfaces a defect worth filing. If it surfaces none, this file is not touched. |

Everything else is inside the card's owned footprint `.night-crew/qa/spike-supabase/**` (new spike-B
script, its HQ-source compose file, its SQL, its Node migrate/verify harness) and the run's evidence
directory `.night-crew/runs/2026-08-07-autonomous/s-logs/`.

**Not touched:** no frontend HTML/JS, no backend Go, no `sw.js` / `version.json` (nothing precached
changes — G4 must show `sw.js` UNCHANGED), no `package.json` / `version.go` version bump, no
`night-crew.toml`, no `docker-compose.test.yml`, no `docker-compose.supabase.yml`, no
`docker-compose.yml`.

---

## What must survive any merge

- **W0's re-pointed `test:*` env blocks in `Taskfile.yml` must survive this card, in full.** This
  card adds a **disjoint** `spike:migration` stanza and touches **neither** those blocks nor a single
  line inside them. If a merge presents a conflict against any `test:*` target, W0's side wins
  outright — nothing this card needs lives there.
- **The attended `prod:backup` stanza in `Taskfile.yml` must survive this card, in full.** Same
  ruling: disjoint addition only, no edit, no reorder, no reindent. If a merge presents a conflict
  against `prod:backup`, the existing side wins outright.
- **Spike A's artifacts are read-only to this card.** `env-up.sh`, `docker-compose.supabase.yml`,
  `sql/spike-fixture.sql`, `sql/hq-bridge-fixture.sql`, `sql/hq-bridge-policies.sql`,
  `rxdb/spike-env.js` and `rxdb/proof-*.js` are **consumed, never modified** — spike A's GREEN
  verdict must keep reproducing byte-for-byte after this card lands. Spike B calls `env-up.sh` and
  imports `spike-env.js`; it does not edit them.
- **The scratch-container isolation rule.** The HQ-shaped Postgres is its **own** throwaway compose
  project on a **Docker-assigned ephemeral host port**. It is never `:5433` (the shared cluster —
  which is production; a probe there destroyed the prod DB on 2026-08-06), never `:5434`
  (`yumyums-test-pg`, W0's proof substrate), and never `5432`. Any later edit that pins a fixed port
  or re-points this at an existing cluster re-arms exactly that incident.
- **The exit-status contract of `spike-b-migration.sh`.** No `|| true` on an assertion, no
  "warn and continue", no advisory leg. A step that cannot decide is a FAILURE. This is spike A's
  own rule (env-up.sh:18-27) carried forward; softening it destroys the only thing the script is for.

## What is safe to drop

- The prose and comment blocks explaining *why* (informative, not load-bearing).
- The captured logs under `.night-crew/runs/2026-08-07-autonomous/s-logs/` — evidence only,
  regenerable by re-running the script.
- The fixture's specific row bodies and user names — any HQ-shaped seed with ≥2 owners × ≥2 apps
  satisfies the same assertions.

## Nothing here

- **No production code change of any kind.** No HTML, no JS shipped to the PWA, no Go, no SQL
  migration under `backend/internal/db/migrations/`, no API contract.
- No version bump, no `sw.js` regeneration expected to produce a diff.
- `main`, `dev`, and every other card branch are untouched.

---

## Red-first

**n/a — spike card; the verdict IS the captured evidence.** Same shape as run `20260806`'s card C1
(`spike-a-environment-up`): the deliverable is a runnable script plus its captured output, not a code
fix, so there is no defect to capture failing before a fix and no regression test to add to
`tests/persistence.spec.js`.

The exit-status contract that stands in its place:

| Exit | Meaning |
|---|---|
| `0` | **GREEN.** A fresh HQ-shaped Postgres came up, was fixtured, and its rows were migrated into the Supabase substrate through the real API path; RLS discriminates over the **migrated** rows on both axes (identity *and* live entitlement); and a real RxDB client pulled **exactly** the RLS-visible migrated set, bodies matching. |
| `!= 0` | **RED**, with the failing leg named and the reason printed — and that is a **successful, reportable spike outcome**, not a card failure. Recorded with the script's captured output; not ground against for green. |

Every assertion is made against **migrated** rows, never against spike A's pre-existing seed rows —
so "the substrate already worked" cannot be mistaken for "the migration worked."

**PARK conditions** (verbatim from the card): a credential, a paid account, or a decision about which
substrate topology the project standardises on → PARK with analysis. Debugging compose or the
migration script is **not** a park. A red verdict is **not** a park — it is the deliverable.
