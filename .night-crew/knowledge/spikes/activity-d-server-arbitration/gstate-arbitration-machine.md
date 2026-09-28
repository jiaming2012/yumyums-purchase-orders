# Spikes — gstate-arbitration-machine

Activity: Activity D — The server arbitration machine (gstate)

> No `usm/roadmap.txt` on this target — hand-run convention (full preamble in
> `../activity-b-offline-first-replica/rxdb-pull-replica.md`). No substrate
> needed: the machine orchestrates; the DB stays the arbiter (§18 edge-case 1),
> so the spike stubs the Redeemer at the interface §18 defines and proves the
> LIBRARY's semantics, which is the unproven half.

## The goal, and which legs need a spike

The card (roadmap Activity D): `backend/internal/redemption` — the §18
statechart (`validating → burning → route_outcome → {redeemed | already_used |
expired | failed}`) wrapping the atomic `redeem()` via `Invoke`, plus the HQ
endpoint, with F4's `RaceLostReconciled` on reconciled overrides.

The load-bearing unproven premise: **the §18 appendix was written against
`github.com/floodfx/gstate`'s IMAGINED API — the library is not in
`backend/go.mod` and has never been imported in this tree.** Everything
Activity D is sized on assumes it exists, fetches, and behaves as sketched:
generic `Machine[State, Event, Data]`, eventless `Always().Guard().GoTo()`
chains with ordered fallback, `Invoke` running async with **ctx auto-cancel on
state exit** (edge-case 4 — the flaky-hotspot protection), and `After` retry.
If the real API differs, the card's design and estimate change; if the library
doesn't hold up, §18 itself names the fallback (qmuntal/stateless, with the
async inverted). That fork should cost an afternoon here, not a night mid-run.

Failure taxonomy, decided at authoring: **network unreachable = could-not-run;
module-not-found / API-cannot-express-§18 = failed** (a falsified premise, not
an environmental refusal).

## Spike: gstate-invoke-semantics

- proves: against the REAL fetched library, in a spike-local Go module
  (backend/go.mod untouched): (1) `github.com/floodfx/gstate` resolves and the
  §18-shaped machine COMPILES against its actual API (drift from the sketch is
  recorded as a correction, not silently absorbed); (2) happy path — a stub
  Redeemer returning `redeemed` drives `validating → burning → route_outcome →
  redeemed`; (3) `already_used` and `expired` route to their terminals;
  (4) edge-case 3 — an UNKNOWN/empty burn result terminates `failed`, never a
  silent `expired` (the ordered-fallback premise); (5) edge-case 4 — a
  Redeemer that blocks until `ctx.Done()` is observably CANCELLED when the
  machine leaves/stops the invoking state (the auto-cancel premise §18 calls
  "the library's job"); (6) missing token → `failed` without ever invoking the
  Redeemer. Enumerated: the resolved module version, and every terminal state
  per leg.
- plan: `go/` dir with its own `go.mod`; `go get github.com/floodfx/gstate`
  (`GOFLAGS=-mod=mod`, network → could-not-run only on unreachable proxy);
  `machine_test.go` implements the six legs with a stub Redeemer
  (channel-instrumented for the cancellation leg); the .sh wraps
  `go test -v ./...` with the exit taxonomy above and prints the resolved
  version from `go list -m`.
- script: .night-crew/spikes/activity-d-server-arbitration/gstate-arbitration-machine/01-gstate-invoke-semantics.sh

## Verdict (run 2026-09-04, hand-run per the no-story-map convention)

- **gstate-invoke-semantics: passed** — exit 0, first run. The library
  resolved as **github.com/floodfx/gstate v0.3.1** and the §18-shaped machine
  compiled against its actual API essentially as sketched —
  `New[S,E,D](id).Initial().State(…)` with `Always().Guard().GoTo()`,
  `Invoke(func(ctx, snap, mutate) error, onDone, onError)`, `After(d).GoTo()`,
  `Start(m, data)` / `actor.Send` / `actor.Stop`. All seven legs held: happy
  path → `redeemed`; `already_used`/`expired` route to their terminals;
  unknown AND garbage outcomes terminate `failed` (never a silent `expired`);
  a Redeemer blocked on `ctx.Done()` observed cancellation when an event
  exited `burning` (the hung-hotspot protection is real, not doc-lore); a
  token-less attempt reached `failed` with the Redeemer never invoked; and the
  `failed → After → burning` retry recovered a transient error (2 calls,
  terminal `redeemed`).

**Conclusion:** Activity D is buildable as designed on gstate; the §18 sketch
is faithful to the real API. Build-facts the card inherits: (1) **gstate
v0.3.1 requires Go ≥ 1.26.2, and `backend/go.mod` pins `go 1.25.5`** —
adopting it means a backend Go toolchain bump (Dockerfile + `-ldflags` build
path included), which the card must price and state; (2) one transitive dep
(`github.com/jaevor/go-nanoid`); (3) terminal states are `Type(gstate.Final)`;
awaiting a terminal from a request/response handler is poll-or-observer —
gstate ships observers (`Start(m, data, opts...)`), which is also where F4's
`RaceLostReconciled` emission naturally hangs.

## Corrections

- none — no agent-reached corrections. The script passed on its first run
  (the test was authored against the read source of the real library rather
  than compiled blind from the sketch, so API drift had no chance to surface
  as a red; none existed that mattered — the sketch's builder shape is the
  real shape).

## Review

- n/a — no corrections, so no batch review is owed for this goal (§4 fires
  only when the ledger holds agent-reached corrections).
