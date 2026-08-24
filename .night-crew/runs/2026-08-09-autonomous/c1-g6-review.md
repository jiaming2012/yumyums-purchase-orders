# G6 adversarial review — C1 `demo-sync-target` (run 20260809)

Fresh-context adversarial reviewer. Inputs limited to the slate entry, the diff, and independently
reproduced evidence — not the implementer's reasoning. It re-ran the demo and all three tri-state
paths itself.

## Verdict: PASS-WITH-FINDINGS  ·  no fix round required

The deliverable is real and honest: all three tri-state exits reproduce distinctly and correctly, the
GREEN is a genuine non-vacuous round-trip through the real `/saveResponse` path, the footprint is clean,
and substrate safety holds. The reviewer tried to break the GREEN (vacuous read, planted value, fake
write), the tri-state honesty (infra→1 or red→2 confusion, wrapper masking), the footprint
(workflows.html edit, precache drift), and substrate safety (residue, orphan containers) — and could not.

## Per-criterion

| Criterion | Verdict | Evidence |
|---|---|---|
| Real `/saveResponse` write | PASS | HTTP 204 to real handler on real HQ binary; scratch PG w/ 75 goose migrations, 52 tables; real login (200) + real hq_session; HQ Postgres independently held 1 sentinel row |
| RxDB-served read on one real checklist | PASS (letter) / see Finding | Row arrived in a running RxDB client via the identical `replicateSupabase` plugin the browser uses, against real Spike A PostgREST+Realtime; owner_id/field_id/value asserted. **Node client — no browser surface** |
| Tri-state 0/1/2 distinct & correct | PASS | 0/1/2 all reproduced; exit-2 renders "NO VERDICT," distinct from red |
| Gate-on-script-not-wrapper | PASS | `task demo:sync:red` returned 201 while the script's true exit was 1 — B-163 confirmed live; documented in Taskfile stanza + script header |
| Footprint clean (no workflows.html edit, precache 31) | PASS | 4 files changed, all in-footprint; workflows.html untouched; `build-sw.js` → 31 precached, 0 outside |
| T-42 / 105 / 126 adherence | PASS | Relay is `pgxlisten` LISTEN/NOTIFY (no polling/watermark); write path byte-identical with/without trigger; RLS-scoped read (owner_id+app_slug), never whole-table |

## Independently-observed exit codes
- GREEN: `./demo-sync.sh` → EXIT=0. Round trip CLOSED in **117 ms** (bound 20000 ms). Real HTTP 204; row
  carried real HQ uuid, exact field_id, exact sentinel; HQ Postgres held 1 sentinel row; substrate restore
  VERIFIED byte-identical.
- RED: `./demo-sync.sh --break-roundtrip` → EXIT=1. Write landed (204, 1 row), relay absent, RxDB empty for
  full 20101 ms → genuine mechanism-red (harness routes to exit 2 if the write *hadn't* landed, so exit 1
  proves the write really happened).
- COULD-NOT-RUN: EXIT=2 by two paths — demo's own missing-harness precondition, and harness passthrough
  (`DOCKER_HOST=tcp://127.0.0.1:1` → `docker info` fails → exit 2, no fabricated verdict).

## Finding 1 — read surface (MEDIUM, operator-awareness, NOT a defect)
The demo reads via a **Node.js RxDB client** (`rxdb/spike-c-read.js`). Confirmed by grep: no browser
launched, no Playwright, no `page.goto`, no `#sync-one-row` DOM — every browser mention in the demo path is
a comment. It satisfies the close-bar **letter** (one real checklist ✓, RxDB-served read ✓ via the identical
`replicateSupabase` plugin against the real substrate — not a `page.route` stub, real write ✓). But it proves
the **data-plane round trip** and exercises **no browser/app UI**. The C3 fill-view and C2 `#sync-one-row`
surfaces are never pointed at the real substrate here. Rationale (browser-against-real-substrate needs
unproven novel integration) is sound and is exactly the "too heavy for a clean demo" the fallback clause
anticipates. **G6's judgment: clears the letter, legitimate in-card engineer decision — but if the operator's
mental model of "sync running in my dev environment" includes seeing it in the app, this demo does not show
that. Flag for the operator before the attestation run.**

## Finding 2 — first-class deliverable vs alias (LOW, acceptable)
`task demo:sync` → `demo-sync.sh` → `spike-c-roundtrip.sh` (which also backs `task spike:roundtrip`). Verdict
logic is 100% inherited. `demo-sync.sh` adds: milestone close-bar framing/banner, close-bar-English flag
vocabulary, its own exit-2 precondition for a missing harness, and 0/1/2/3 verdict prose. A deliberate,
well-documented re-export rather than a from-scratch deliverable — reasonable (forking ~630 lines of
thrice-debugged harness would create a second copy of the verdict that could drift). Operator should know the
substance was proven by Spike C; this card packages and names it.

## Minor notes (not defects)
- Exit-2-via-bogus-Docker path prints "teardown ... did not complete cleanly" (because `docker compose down`
  also fails against the bogus DOCKER_HOST) — does not alter the exit code (still 2), no container created.
- merge-intent line ~75 "drives the running app's read path" is slightly imprecise — it drives the RxDB
  replication path, not the running app. Script header itself is precise.
- Independent substrate cross-check: G6 minted its own service_role token, queried the shared table after all
  4 runs → zero `spikec-` rows, only baseline fixtures; `TestJWTBridgeRLS` control set intact; no orphan
  `spike-c-hq` scratch containers.
- sw.js rebuild on this box is not byte-identical to committed (precache-array ordering + workbox chunk hash),
  but all 31 `{url,revision}` entries are content-identical — a toolchain/reproducibility artifact, not
  attributable to the card (which touched no served asset).
