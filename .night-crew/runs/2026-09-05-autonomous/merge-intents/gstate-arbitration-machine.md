# Merge intent — Card 7 · gstate-arbitration-machine (Track D)

Branch: `wo-gstate-arbitration-machine` · Run: 20260905
Scope: `backend/internal/redemption` (NEW) on gstate v0.3.1; the §18 statechart wrapping
the atomic `redeem()` via Invoke; the HQ endpoint the scanner's online submit posts to
(R2); the priced Go 1.25.5 → ≥1.26.2 toolchain bump; the F4 `RaceLostReconciled`
read-model + Shift-Manager notification entry.

## Shared files touched

| File | Why |
|---|---|
| `backend/go.mod` | The priced toolchain bump: `go 1.25.5` → `go 1.26.2` (gstate v0.3.1's own go.mod requires 1.26.2) + `github.com/floodfx/gstate v0.3.1` direct dep. Shared with the whole backend — every Go leg on every card builds on this directive. |
| `backend/go.sum` | gstate v0.3.1 + transitive `github.com/jaevor/go-nanoid v1.4.0` (priced by the spike). Shared for the same reason. |
| `backend/Dockerfile` | Builder image `golang:1.25-alpine` → `golang:1.26-alpine` so the image build can honor the bumped go directive. The `-ldflags` block injecting `version.GitSHA`/`version.BuiltAt` is untouched. |
| `backend/cmd/server/main.go` | Route wiring only: `POST /api/v1/marketing/redeem` mounted in the protected group behind `auth.RequirePermission(pool, "marketing")`, + fail-closed Redeemer construction from env. Card 1's landed seed changes in `backend/internal/db/db.go` are NOT touched. |
| `backend/internal/db/migrations/0077_race_lost_notifications.sql` | NEW numbered goose migration (F4 read-model home — decision below). New file, no edit to any existing migration; Activity A's `supabase/migrations/*` are read-only to this card. |
| `.night-crew/knowledge/roadmap.md` | This card's line PLANNED → DRAFTING (overnight). |

Everything else is new files under `backend/internal/redemption/` — no other card's footprint.

## What must survive any merge

1. **The toolchain bump.** `backend/go.mod` `go 1.26.2` + the Dockerfile builder
   `golang:1.26-alpine`. Reverting either while `internal/redemption` imports gstate
   breaks every Go build. go1.26.2 was already in the local module cache from the spike
   — no network toolchain fetch was needed on this box.
2. **The endpoint contract (Card 6 builds against THIS):**

   **`POST /api/v1/marketing/redeem`**
   Auth: HQ session cookie (normal `auth.Middleware`) + `RequirePermission("marketing")`.
   The submitting/syncing session user is recorded as `staff` server-side — the client
   does not send identity.

   Request (JSON):
   ```json
   {
     "token_hash": "<sha256 hex of the scanned identity token>",   // REQUIRED
     "device_id":  "<stable device identifier>",                    // REQUIRED
     "order_number": "1234",             // optional — Toast order # if captured
     "offline_override": false,          // true when this is a synced queued override attempt (§13)
     "unverified_code": false,           // F2 flag, echoed into the F4 entry when a race is lost
     "scanned_at": "2026-09-05T02:00:00Z", // optional RFC3339 client scan time; default server now
     "value": 2.50                       // optional — offer face value ($) as displayed at accept time
   }
   ```

   Response `200` (every arbitration VERDICT is a 200 — the verdict is data; maps 1:1
   onto §19.3's `SRV_*` boundary events):
   ```json
   {
     "result": "redeemed" | "already_used" | "expired" | "not_found" | "error",
     "race_lost_reconciled": false,   // true exactly on the F4 path (already_used + offline_override)
     "error": "…"                     // present only when result == "error"
   }
   ```
   Non-200s: `400` malformed JSON / missing `token_hash` or `device_id`; `401/403` from
   the auth stack; `503 {"error":"redemption_not_configured"}` when the arbiter backend
   env is unset (fail-closed, same doctrine as the sync proxy and
   HQ_INVENTORY_SERVICE_TOKEN); `500 {"error":"race_lost_notification_failed"}` if the
   F4 notification write fails after an already_used override verdict (loud + retryable:
   re-arbitrating an already_used attempt is stable, so the client sync may retry).
   `504 {"error":"arbitration_timeout"}` if the machine reaches no terminal inside the
   server-side budget.

   Config (all fail-closed): `HQ_SYNC_REST_URL` (PostgREST base — same var the sync
   proxy already uses) + `HQ_SYNC_SERVICE_KEY` (NEW: the substrate service_role key; the
   redeem RPC's grant anticipates exactly this caller). Unset ⇒ 503.

3. **The wire-vs-machine taxonomy split.** Machine terminals are exactly §18's
   `{redeemed, already_used, expired, failed}` — no new terminal (PARK boundary
   respected). `not_found` is NOT a machine state: the RPC's `not_found` reason routes
   through the E-KR2 fallback to the `failed` terminal, and the WIRE result is derived
   as `not_found` from the attempt's recorded outcome. Unknown/empty/garbage outcomes
   stay wire `error` — never `expired`, never `not_found`.
4. **Migration `0077_race_lost_notifications.sql`** and the `race_lost_notifications`
   table (F4's read-model entry: code, device, staff, time, value).
5. **The no-TOCTOU property.** The `Redeemer` interface has exactly one method —
   `Redeem` — there is deliberately NO read/check method to guard on. Token-hash → code
   resolution inside the RPC redeemer is identity resolution and does not read or gate
   on redemption state; the conditional UPDATE inside `redeem()` stays the only arbiter.

## What is safe to drop

- Retry tuning constants (`defaultRetryDelay`, `defaultMaxRetries`, the arbitration
  timeout default) — any values keep the machine correct; tests inject their own.
- The negative-control tests (`*_DoesNotEmit`, the check-then-act analog control) if a
  merge needs to thin the suite — the three named red-first tests are the gate.
- Internal state NAMES `route_failure` / `retry_wait` (machine wiring, mine to choose);
  the four terminals and `validating/burning/route_outcome` are §18's and are not.

## Red-first

Three baked edge cases, individually named, reds captured to
`.night-crew/runs/2026-09-05-autonomous/card7-red.log` (command + exit code) BEFORE the
greening code; greens in `card7-green.log`.

1. **E-KR2 fallthrough** — `TestEKR2_UnknownOrEmptyBurnResultIsFailedNeverExpired`
   (`backend/internal/redemption/machine_test.go`). Red state: the machine's
   `route_outcome` carries only the three known-outcome guards and NO ordered fallback —
   an unknown/empty burn result parks the machine and the arbitration times out instead
   of terminating `failed`. Green: the explicit `Always → failed` fallback (ordered
   last), asserted `failed` and asserted NOT `expired` for both `""` and `"garbage"`.
2. **F4 emission** — `TestF4_RaceLostReconciledEmitsNotification`
   (`backend/internal/redemption/f4_test.go`, DB-coupled). The done_when two-attempt
   reconciliation: attempt 1 (device A, online) → `redeemed`; attempt 2 (device B,
   synced `offline_override=true`) → `already_used` terminal. Red state: observer
   resolves the await but emits nothing — no `RaceLostReconciled`, no
   `race_lost_notifications` row. Green: the terminal observer flags the race-lost
   reconciliation and the arbitrator synchronously persists the entry (code, device,
   staff, time, value) before responding.
3. **No-TOCTOU (§18 #1)** — `TestNoTOCTOU_ConcurrentAttemptsSingleWinner` (8 concurrent
   arbitrations of one code against an atomically-single-winning stub arbiter: exactly
   one `redeemed`, seven `already_used`, exactly one burn call per attempt). This
   property holds by construction from the first honest machine, so its red is the
   redeem-rpc-race-proof precedent shape: at red-capture the harness is pointed at a
   naive check-then-act ANALOG (read redemption state, yield, then act on the stale
   read) and the single-winner assertion demonstrably fails — proving the harness
   catches the defect class. The green commit keeps that analog as a permanent inverted
   control (`TestNoTOCTOU_CheckThenActAnalogDoubleWins`: the analog MUST double-win
   under the harness, or the harness has lost its teeth).

## Engineering calls recorded here

**F4 read-model home: HQ Postgres** (`race_lost_notifications` via goose migration
0077), not a supabase/ migration. Rationale: (a) the entry's consumer is the Shift
Manager inside the HQ app — every HQ-facing notification/read surface lives in HQ
Postgres behind the session + grant stack, and the upcoming `reconciliation-view` card
reads from HQ; (b) the emitter runs inside HQ Go (the arbitrator), which holds the HQ
pool — the write is a plain same-process insert; (c) putting a manager-only table on the
Supabase substrate would demand new RLS policy design for a device-visible surface,
which brushes the card's PARK boundary (notification policy) for zero benefit — the
substrate carries the device-facing replication set, not HQ operator read-models.
A supabase/ migration for this card therefore does not exist; Activity A's migrations
are consumed read-only.

**Observer shape** (the card's "observers carry the endpoint's await AND the F4
emission"): one `terminalObserver` implementing gstate's `StateEnteredObserver`. It is
non-blocking per gstate's locking contract (callbacks run under the actor lock): on
entry of any Final state it captures `{terminal, attempt-data snapshot, raceLost}` —
`raceLost = (terminal == already_used && attempt.OfflineOverride)` — and signals a
buffered channel. The awaiting `Arbitrate` goroutine receives the settle, performs the
F4 persistence synchronously via `RaceLostSink` (so the done_when assertion is
deterministic, no fire-and-forget), derives the wire result, and stops the actor. The
observer decides/emits the domain event; the sink persists it; no DB work ever runs
under the actor lock.

## Gate evidence (completed post-build)

| Gate | Verdict | Log |
|---|---|---|
| G1 | PASS — `go build ./...` exit 0, `go vet ./...` exit 0, on `go1.26.2 darwin/amd64`, from `backend/` | `card7-g1.log` |
| G2 (Go) | PASS against armed-reds baseline — 12 test-bearing packages (baseline 11 + `internal/redemption` NEW), 562 with-subtest results (≈538 baseline + 24 new), `internal/workflow` 39 ≥ 35, sync's 59-subtest gates self-asserted PASS, `HQ_SYNC_SUBSTRATE_OPTIONAL`/`HQ_SYNC_GATE_CHILD` both unset. `SUITE_EXIT=1` is entirely the base-proven pre-existing `TestJWTBridgeRLS` (same 4 subtests + parent as `card1-baseline-jwtbridge.log`); zero failures outside it | `card7-g2-go.log` |
| G2 (Playwright) | N/A per signed slate — backend-only card, no page file touched (touched paths carry no `[e2e.seams]` entry) | — |
| RF | Three named reds captured with command + exit code before the greening code; greens re-run individually | `card7-red.log` / `card7-green.log` |
| G4 | Not owed — no frontend asset changed, `sw.js` untouched | — |
| Docker build | Smoke SKIPPED (optional per slate; `golang:1.26-alpine` would need a network pull). Builder-image bump verified by inspection; `-ldflags` path verified by a direct `go build -ldflags` smoke on 1.26.2 (exit 0, injected GitSHA present in binary via `strings`) | `timings.log` note |

**Machine wiring** (v0.3.1 API fact discovered building this: `After(d)` delayed
transitions DROP Guard/Assign — `executeInternalTransition` rebuilds a bare
`TransitionDef{Target}` — so §18's sketch of a guarded/bounded retry directly on
`failed` cannot bound anything): retry is bounded BEFORE the wait, via Always-guarded
routing, and `failed` becomes a true `Final` terminal. Shape:

```
validating  ─[token≠""]→ burning            ─[else]→ failed (Err=missing_token)
burning     ─Invoke(redeem)─ done → route_outcome · error → route_failure
route_outcome ─[redeemed]→ redeemed ─[already_used]→ already_used ─[expired]→ expired
              ─[else, ordered last]→ failed          (E-KR2: never a silent expired)
route_failure ─[retries<max]→(retries++) retry_wait ─[else]→ failed
retry_wait  ─After(backoff, unguarded — bound already enforced)→ burning
redeemed / already_used / expired / failed : Type(Final)
```
`route_failure`/`retry_wait` are internal wiring states, not terminals; the terminal set
is exactly §18/§19's four.
