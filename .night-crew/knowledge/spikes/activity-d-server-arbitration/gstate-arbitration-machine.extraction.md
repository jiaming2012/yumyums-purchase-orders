# Extraction — gstate-arbitration-machine

Outcome: learned

Approach used: `github.com/floodfx/gstate` **v0.3.1** carrying the §18
statechart essentially as sketched — `New[S,E,D](id).Initial().State(…)` with
`Always().Guard().GoTo()` ordered fallback, `Invoke(func(ctx, snap, mutate)
error, onDone, onError)` wrapping the Redeemer, `After(d).GoTo()` retry,
`Start(m, data)` / `actor.Send` / `actor.Stop`, terminals as
`Type(gstate.Final)`. Proven in a spike-local Go module (backend/go.mod
untouched) with the Redeemer stubbed at the §18 interface — the DB stays the
arbiter. A candidate the card's design.md adopts or not (NFR-6).

Confirmed: the load-bearing premise — the library is real, fetches, and the
imagined API the §18 appendix was written against is faithful to the actual
one. All legs held: happy path → `redeemed`; `already_used`/`expired` route to
their terminals; unknown AND garbage burn results terminate `failed`, never a
silent `expired` (ordered fallback); a Redeemer blocked on `ctx.Done()`
observed cancellation when an event exited `burning` (the hung-hotspot
auto-cancel is real, not doc-lore); a token-less attempt reached `failed`
without invoking the Redeemer; `failed → After → burning` retry recovered a
transient error. The qmuntal/stateless fallback fork is dead — not needed.

Learned: the adoption price the sketch did not carry. (1) **gstate v0.3.1
requires Go ≥ 1.26.2 while `backend/go.mod` pins `go 1.25.5`** — adopting it
means a backend toolchain bump reaching the Dockerfile and the `-ldflags`
build path. (2) One transitive dep (`github.com/jaevor/go-nanoid`).
(3) Awaiting a terminal from a request/response handler is poll-or-observer —
gstate ships observers (`Start(m, data, opts...)`), which is also where F4's
`RaceLostReconciled` emission naturally hangs.

Plan change: Activity D's card is buildable as designed on gstate, but if its
design.md adopts it, the card must price and state the Go 1.25.5 → ≥1.26.2
toolchain bump (Dockerfile + build path included) as in-scope work — the
estimate the roadmap sized did not include it. The observer mechanism is the
named candidate for both the endpoint's await and the F4 emission.
