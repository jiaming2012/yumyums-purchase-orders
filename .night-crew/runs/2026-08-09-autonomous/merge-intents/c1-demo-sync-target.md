# Merge intent — C1 · `demo-sync-target`

Run `20260809` · branch `card/c1-demo-sync-target` · Activity 5, the milestone close-bar deliverable.

---

## Scope

Ship `task demo:sync` as a first-class deliverable: a scripted-fresh environment (Spike A's
stack via `env-up.sh`, reconcile mode), ONE field written through the REAL write path
(`POST /api/v1/workflow/saveResponse`), surfacing in an RxDB-served read on ONE real checklist,
round-trip — with a **tri-state exit contract** (0 green / 1 ran-and-failed / 2 could-not-run)
that is the milestone's whole reason to exist.

The new script `demo-sync.sh` is a **thin re-export of Spike C's proven round-trip harness**
(`spike-c-roundtrip.sh`), which already implements the exact round trip the close bar names
(real `/saveResponse` → HQ Postgres → LISTEN/NOTIFY relay → PostgREST service-identity write →
RxDB-served read, proven at 248ms) AND the exact tri-state contract the close bar requires
(exit 0/1/2, with exit 2 as "could not run" distinct from exit 1 "ran and failed"). Rather than
fork/duplicate ~630 lines of hard-won, thrice-debugged harness (orphan-server refusal, ephemeral
port isolation, substrate restore verification), `demo-sync.sh` is the milestone's named entry
point that delegates to it — so the verdict comes from exactly ONE place and cannot drift from
the spike it reuses. It adds the demo framing (banner, the close-bar decisions it carries) and
maps the harness's `--no-relay` red-first mode through as `--break-roundtrip`.

## Read-surface decision (engineer-level, in-card choice — NOT a park)

The card's PRIMARY ask is the C3 real fill-view RxDB read; the DOCUMENTED FALLBACK is C2's
`#sync-one-row`. **I use the Spike C round-trip harness's RxDB Node read client
(`rxdb/spike-c-read.js`) as the read surface** — the C2 fallback in spirit. Reasoning:

- C2/C3's browser read surfaces (`#sync-one-row`, the fill view) are driven by the vendored
  supabase-js client and in EVERY existing test (`tests/sync-one-row.spec.js`,
  `tests/sync-fill-view.spec.js`) the substrate is a `page.route` STUB — no test ever points the
  browser read path at the REAL Spike A substrate. `HQ_SYNC_REST_URL`/`HQ_SYNC_REALTIME_URL` are
  unset in every environment (`sync-rxdb/bootstrap.js`).
- Driving the REAL browser fill-view against the REAL substrate end-to-end for the demo would
  require novel, unproven integration (inject substrate URLs into a served page, a Playwright
  Chromium, a rewriting proxy, plus the live relay) — exactly the "too heavy for a clean demo"
  the card's fallback clause anticipates.
- `rxdb/spike-c-read.js` uses the IDENTICAL `replicateSupabase` RxDB plugin that the browser's
  `startHQReplication` uses, pointed at the REAL Spike A PostgREST + Realtime — a genuine
  RxDB-served read on one real checklist, against the real substrate, proven green. It satisfies
  the close-bar letter ("one real checklist", "RxDB-served read", "real /saveResponse write path").

This is an in-card engineering choice, stated per the card's instruction; it is NOT the OPEN
My-Checklists read path (T-43(b)) and needs no scope widening — so no PARK trigger is hit.

## Decisions carried verbatim at the call site

- **Decision 126** — RxDB serves READS; `/saveResponse` + `/submitChecklist` keep owning ALL
  writes. `demo-sync.sh` touches no write path (the harness applies a trigger + external relay,
  never edits `saveResponse`).
- **Decision 105** — scoped read, never whole. The RxDB client reads only the scoped rows.
- **Spike E condition T-42** — no polling, no business-watermark resync; the relay stays
  trigger/NOTIFY-driven (the reused harness is exactly this LISTEN/NOTIFY mechanism).

---

## Shared files touched

| File | Why | What must survive a merge | Safe to drop |
|---|---|---|---|
| `Taskfile.yml` | I ADD a new `demo:sync` + `demo:sync:red` stanza (thin wrappers → `demo-sync.sh`). There is no `demo:` namespace yet — I create it. Card 2 (B-163 companion) later adds a `spike:reconnect:red` note to a DIFFERENT stanza in the same file — disjoint, no overlap. | The `demo:sync` / `demo:sync:red` targets and their thin-wrapper convention (all logic in `demo-sync.sh`; gate on the SCRIPT not `task`). | Nothing — this is additive; no existing target is modified. |
| `.night-crew/knowledge/roadmap.md` | I flip the `demo-sync-target` card (line ~481) PLANNED → DONE. | The card's description text and the DONE verdict line. | Nothing else on that line changes. |

New files OWNED by this card (no merge risk — they did not exist before):
`.night-crew/qa/spike-supabase/demo-sync.sh` and `.night-crew/runs/2026-08-09-autonomous/*`.

Also regenerated: `sw.js` (G4). It is a committed artifact; `build-sw.js` reads git HEAD.
`demo-sync.sh` and `Taskfile.yml` are NOT precached assets, so precache count stays **31**.

## Files READ but NOT edited

- `workflows.html` — READ ONLY, per the footprint. The demo drives the running app's read path
  via the RxDB client; NO source edit. Confirmed no diff to it on any commit of this branch.
- `sync-rxdb/bootstrap.js`, `tests/sync-one-row.spec.js`, `tests/sync-fill-view.spec.js`,
  `spike-c-roundtrip.sh`, `rxdb/spike-c-read.js`, `env-up.sh`, `sql/spike-c-relay-trigger.sql` —
  read to understand the reused wiring; unchanged.

Nothing else. No backend Go file, no other HTML tool page, no `night-crew.toml` entry, no
`docker-compose*.yml` edit, no version bump (not this card's job).

## Empty sections

- **Backend Go**: nothing here. No `.go` file changed (G1/G2-Go N/A-by-footprint).
- **New Playwright specs**: nothing here. No seam-mapped app source touched → G2(Playwright) N/A.
- **Migrations / schema**: nothing here.
- **G4 discipline greps**: N/A-VACUOUS — neither `internal/journal` nor `internal/workorder`
  exists in this repo (B-14).

---

## Red-first

The tri-state exit contract IS the deliverable, so "the test is the exit code". Each wrong-exit
path was forced deliberately and captured with the literal command + `echo "EXIT=$?"`. All three
are DISTINCT, exactly as the contract requires.

### exit 2 — COULD NOT RUN (no verdict)

Forced by pointing the wrapper at a deliberately-absent harness coordinate: a byte-identical copy
of `demo-sync.sh` run from a scratch directory where `spike-c-roundtrip.sh` does not exist, so the
wrapper's own missing-harness precondition fires.

```
$ /…/scratchpad/demo-sync-probe.sh
🛑 COULD NOT RUN (not a verdict) — the round-trip harness is missing or not executable: /…/scratchpad/spike-c-roundtrip.sh
EXIT=2
```

Result: **EXIT=2**, NOT 1 and NOT 0. ✅

### exit 1 — RAN AND THE ROUND TRIP FAILED (a finding)

Forced via `--break-roundtrip` (the demo's name for the harness's proven `--no-relay` red-first
mode): the real Spike A stack up, HQ's real 75 migrations applied, the real field written through
real `/saveResponse` — and the relay DELIBERATELY ABSENT, so nothing bridges HQ Postgres to the
substrate.

```
$ ./.night-crew/qa/spike-supabase/demo-sync.sh --break-roundtrip
  … health: {"backend_version":"0.3.0",…,"status":"ok"}  ·  goose versions applied: 75 · public tables: 52
  🛑 --no-relay: the trigger is NOT applied and the relay is NOT started.
  🛑 ROUND TRIP RED: the row written through /saveResponse did NOT reach the RxDB-served read within 20000 ms.
  HQ Postgres holds 1 row(s) carrying the sentinel (the write path itself)   ← proves it RAN, not a harness failure
  VERIFIED: hq_sync_checklists and hq_grant_projection are byte-identical to the pre-run baseline.
  🛑 demo:sync VERDICT: RED (exit 1) — RAN AND THE ROUND TRIP FAILED.
EXIT=1
```

Result: **EXIT=1**, NOT 2 and NOT 0. The write really landed in HQ Postgres (1 sentinel row) —
that is what makes this a MECHANISM red rather than a could-not-run. ✅

### exit 0 — GREEN (the round trip closes)

The full demo, relay armed.

```
$ ./.night-crew/qa/spike-supabase/demo-sync.sh
  trigger present: spike_c_relay_notify AFTER INSERT OR UPDATE ON submission_responses
  relay READY (LISTEN on spike_c_relay established, pid …)
  ARRIVED in 115 ms -> spikec-7d06f41c-…
  PASS  a row written through /saveResponse reached a RUNNING RxDB client
  PASS  it carries HQ's real user uuid in owner_id (identity axis intact through the projection)
  PASS  it carries the exact field_id and value the write path was given
  ROUND TRIP CLOSED in 115 ms (bound 20000 ms)
  MECHANISM PROVEN: LISTEN/NOTIFY relay.
  VERIFIED: hq_sync_checklists and hq_grant_projection are byte-identical to the pre-run baseline.
  hq postgres: project spike-c-hq, container 8b7d1c5a1ced, host port 63396 (ephemeral)
  ✅ demo:sync VERDICT: GREEN (exit 0) — the round trip closes.
EXIT=0
```

Result: **EXIT=0**. One field written through the REAL `/saveResponse` surfaced in an RxDB-served
read on one real checklist, round-trip, in **115 ms** (faster than Spike C's cited 248ms; well
under the 20s bound). Ephemeral port 63396 — never :5432/:5433/:5434. ✅

### Wrong-then-right

Shown at the wrapper's precondition seam: the SAME `demo-sync.sh`, invoked from a location where
the harness IS present, produces a real verdict (green/red above); invoked from a location where
the harness is ABSENT, produces exit 2. The exit code tracks the precondition, not the prose —
which is the whole point of the tri-state contract.
