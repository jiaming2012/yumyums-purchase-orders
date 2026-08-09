# Merge intent — Card 1 `sync-live-in-dev-substrate` (run 20260810)

Branch: `wo-sync-live-in-dev-substrate` off `overnight-20260810` @ 03c3e06.
Card: Activity 5, roadmap legs 1+2 + FDW persistence — make the RxDB data plane
run persistently in the operator's dev environment and open the `/sync/*` proxy door.

## Shared files touched (outside this card's own footprint)

- **`backend/Taskfile.yml`** — the 4× `HQ_SYNC_*` dev-env wiring goes into the
  `dev`, `dev:tailscale`, `dev:lan` targets (and their `:log` siblings). DEV
  TARGETS ONLY. Never `docker-compose.prod.yml`, never `prod` / `build`. This is
  named in the card's OWNS footprint ("root/`backend` Taskfile `task` target + the
  4× `HQ_SYNC_*` dev-env wiring").
- **`Taskfile.yml`** (root) — new `sync:dev:*` target family (the persistent-substrate
  `task` target). Named in the card's OWNS footprint. Additive stanzas only; no
  existing target edited. Card 2 states in its own merge-intent what of these must
  survive (it consumes the persistent substrate read-only).

Everything else is inside this card's OWNS footprint:
- `docker-compose.supabase.yml` — NOT edited (already persistent-posture; PGDATA on a
  named volume, reconcile bring-up). The persistent-service layer is built ON it.
- `.night-crew/qa/spike-supabase/sync-dev-up.sh` — NEW: the persistent dev bring-up
  (substrate reconcile + relay-as-service + FDW pointing helper).
- `.night-crew/qa/spike-supabase/sync-dev-proof.sh` — NEW: the done_when proof
  (spike-f scratch-HQ model; 503→200 door + relay-carries-write).
- `.night-crew/qa/spike-supabase/sql/persistent-dev-fdw-pointing.sql` — NEW: the
  HQ-side + substrate-side FDW/role SQL for the persistent pointing.
- `backend/internal/sync/spikec_relay.go` — persistent-service wiring (a stable
  `Ready`-line contract + a doc note that it is now also the persistent dev relay).

## proxy.go

NOT touched. Read the `proxy.go:78` ACTIVATION-ORDER guard: it forbids setting
`HQ_SYNC_REST_URL` in any deploy that is not RLS-ready. The env plumbing did NOT
need a proxy.go edit — `LoadProxyConfig()` already reads all three URL/host vars
from the environment, and `poolMinter` reads `HQ_SYNC_JWT_SECRET`. The 4 vars are
supplied by the dev Taskfile targets; the guard is honored by keeping them out of
`docker-compose.prod.yml` entirely.

## What must survive any merge

1. The persistent-substrate compose posture (`docker-compose.supabase.yml` unchanged,
   never-destroy-on-restart — `sync:dev:up` calls `env-up.sh` reconcile, never
   `spike:down`).
2. The root Taskfile `sync:dev:*` target family.
3. The 4-var dev wiring in `backend/Taskfile.yml` dev / dev:tailscale / dev:lan
   (+ their `:log` variants) — DEV TARGETS ONLY.
4. The relay-as-persistent-service wiring (`spikec_relay.go` + `cmd/spikec-relay`
   driven by `sync:dev:relay`).
5. The FDW/role SQL for the persistent pointing
   (`sql/persistent-dev-fdw-pointing.sql`) + its `sync:dev:fdw` applier.

## What is safe to drop

- Nothing in the proof scripts is load-bearing for the shipped capability — they are
  the evidence, not the deliverable. If a proof script conflicts on merge, the proof
  can be re-derived; the Taskfile/SQL/relay wiring cannot.

## Red-first

done_when item 1 is the green (door → 200). The SAME proxy path with the 4 vars
UNSET returns 503 — that asymmetry is the red-first. Captured by
`sync-dev-proof.sh` (spike-f model, ephemeral scratch HQ, NO `:5433`), which runs
the door check BOTH ways in one invocation. Captured on run p20260809150620 (exit 0):

- **RED (vars unset):** `GET /sync/rest/` → **HTTP 503**
  body `{"error":"sync_proxy_not_configured"}`
- **GREEN (4 vars set):** `GET /sync/rest/` → **HTTP 200**
  body `{"swagger":"2.0","info":{...,"title":"standard public schema","version":"12.2.12"},...}`

The asymmetry is not vacuous: the script asserts the 503 first (RED) and refuses
to continue if the door is already open, then wires the 4 vars and asserts the 200.
done_when 2 (relay carries a real write): `POST /saveResponse` → 204, the field
arrived in the substrate (`spikec-<respid>` in `hq_sync_checklists`, carrying the
sentinel) in **267 ms**.

Nothing here else — no other shared file, no proxy.go edit, no prod-compose touch.
