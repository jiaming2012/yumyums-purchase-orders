# Spikes — sync-live-in-dev

Activity: Activity 5 — Dev complete (the close bar)

> This target repo has no `usm/roadmap.txt` story map (that is the dogfood-branch
> layout the `night-crew spikes gate/run` verbs read), so those verbs cannot drive
> here. The artifacts are authored to the skill's paths anyway — this ledger, and a
> runnable script that IS the verdict (B-345) — and the script lives alongside
> spikes A–E under `.night-crew/qa/spike-supabase/` (the established repo
> convention) rather than under `.night-crew/spikes/…`, so it can reuse env-up.sh,
> the scratch-HQ compose, the vendored RxDB and the repo-root Playwright. The
> `- script:` line below points at that location.

## The goal, and which legs need a spike

`sync-live-in-dev` (roadmap Activity 5, decision 161) makes the RxDB sync capability
**run in the operator's persistent dev environment and be usable in the app** — not
only inside `demo:sync`'s throwaway stack. Three legs:

1. **Persistent substrate + relay** — bring up the Spike A substrate + the
   LISTEN/NOTIFY relay as a persistent dev service.
2. **Config wiring** — set `HQ_SYNC_REST_URL` / `HQ_SYNC_REALTIME_URL` (and, this spike
   establishes, `HQ_SYNC_JWT_SECRET` + `HQ_SYNC_REALTIME_HOST`) in the dev env so the
   in-server `/sync/*` proxy door opens (today it answers 503 everywhere).
3. **App-surface proof** — drive the **real** browser (`workflows.html`, `hq_sync_read`
   ON) against the substrate so one field entered in the app surfaces via RxDB **in the
   app** — replacing the demo's Node RxDB read client with the actual app surface.

The roadmap's own spike gate on this card names the disposition:

> 🛑 leg 3 (real browser against the real substrate) is the one integration the
> milestone never spiked — C3's fill-view stubs the substrate, and the demo used a Node
> client precisely because browser-against-real-substrate was "unproven novel
> integration." Run `/nc-spike` on that leg before this card is slated; the other two
> legs (persistent substrate/relay, config wiring) are proven mechanics and need no
> spike.

- **Leg 1 (persistent substrate + relay) — No spike needed:** proven mechanics. The
  substrate stands up unattended (Spike A, GREEN) and the LISTEN/NOTIFY → PostgREST relay
  closed the round trip GREEN in spikes C, D and E. Productionizing it as a persistent
  compose service is build work, not a falsifiable premise.
- **Leg 2 (config wiring) — No spike needed as its own premise, but its live effect is a
  precondition of leg 3 and is asserted inside leg 3's script** (the "leg-2 door check":
  with `HQ_SYNC_*` set, `GET /sync/rest/` returns 200, not 503).
- **Leg 3 (app-surface proof) — spiked below.**

## Spike: browser-live

- proves: With HQ's `/sync/*` proxy wired to a real Supabase substrate (leg-2 config:
  `HQ_SYNC_REST_URL` / `HQ_SYNC_REALTIME_URL` / `HQ_SYNC_JWT_SECRET` /
  `HQ_SYNC_REALTIME_HOST`), the **production** `workflows.html` page — flag
  `hq_sync_read=on`, **no `page.route` stub** — drives its same-origin RxDB replication
  of the `responses` collection (substrate table `submission_responses`) through the
  proxy, under the **real per-user RLS** policy `submission_responses_select` →
  `hq_can_see_field(field_id)` resolved through the substrate's FDW to HQ's live source
  views, and one field written through the real `POST /api/v1/workflow/saveResponse` path
  surfaces in the app's own dev surface (`#sync-one-row` → `data-state="served"`) carrying
  that field's value, within a bounded time. Red-first: with the carrier that lands the
  row in the substrate withheld, the same real page must NOT reach `served`. This is the
  browser-against-real-substrate integration C3 stubbed and the demo dodged; the carrier
  itself (HQ → substrate) is a **stand-in for leg 1's persistent relay — proven mechanics
  (spikes C/D/E)** — and the spike says so, keeping the spotlight on the read.

- plan: Reuse spike-C's harness discipline — a **fresh scratch HQ Postgres** booted by
  HQ's own binary (real migrations), a **real login → real `hq_session` cookie**, spike A's
  substrate **reconciled, never destroyed**, and a **snapshot → verify → restore** of every
  substrate table touched. Leg-3-specific steps: (a) start the HQ server with the four
  `HQ_SYNC_*` vars pointed at the reconciled substrate (opens the proxy door); (b) make the
  **production per-user RLS resolve** by repointing the substrate FDW server `hq_pg` at the
  scratch HQ — which carries migration 0073/0074's source views + the `hq_sync_fdw` role
  (given LOGIN, with the password the substrate's existing user mapping already uses) —
  snapshotting the FDW options for exact restore; (c) write one field via `/saveResponse`,
  then a **carrier** projects that exact `submission_responses` row into the substrate's
  production `submission_responses` table (a stand-in for leg 1's relay); (d) drive the real
  `workflows.html` dev surface in Chromium (Playwright, no stub, real cookie) and assert
  `#sync-one-row[data-state="served"]` with the sentinel; (e) a **red-first pass** (carrier
  withheld) must NOT reach `served`. Both passes run in one invocation. The script's exit
  status is the verdict (0 green / 1 red / 2 could-not-run / 3 restore-failed / 64 usage);
  the browser armed pass is the verdict, everything before it is could-not-run.

- script: .night-crew/qa/spike-supabase/spike-f-browser-live.sh
  (Playwright spec + config: .night-crew/qa/spike-supabase/browser-live/; Taskfile wrapper:
  `task spike:browser-live` — but GATE ON THE SCRIPT'S EXIT CODE, never on `task`, B-163.)

### Context finding carried into the card (the FDW entanglement)

Established while authoring, and load-bearing for whoever builds leg 3: the production
`submission_responses` per-user RLS resolves **through the FDW server `hq_pg`** to HQ's
live views — not through directly-seedable substrate rows like spike C's
`hq_grant_projection`. In the running substrate that server currently points at
`host.docker.internal:5434/hq_test_b2_fdw` (the Go RLS suite's transient fixture DB) and is
**dead when that DB is not up**, so the policy is inert. Making the real app read resolve
therefore requires the FDW pointed at a live HQ carrying the `hq_sync_*` source views +ROLE
— which the persistent-dev-environment card (leg 1/2) must arrange for the operator's
`dev:tailscale` HQ, not only for a scratch container. The spike does it by repoint+restore;
the card needs a persistent answer. (The test user being a **superadmin** shortcuts the
assignee arm: `hq_can_see_template`'s admin arm passes for any template, so only field→
template resolution is needed.)

## Verdict (run 2026-08-08, run id `f20260808232119`)

- **browser-live: passed** — `./.night-crew/qa/spike-supabase/spike-f-browser-live.sh` exited
  **0** on the first run. Evidence, in order:
  - Substrate reconciled GREEN (`env-up.sh`, ports rest=63264 realtime=63263 db=63239).
  - Scratch HQ `spike-f-hq` booted on ephemeral host port 60237 (real migrations); server
    healthy (`backend 0.3.0 / frontend 1.4.0`).
  - **Leg-2 door proven:** with `HQ_SYNC_REST_URL`/`HQ_SYNC_REALTIME_URL`/`HQ_SYNC_JWT_SECRET`/
    `HQ_SYNC_REALTIME_HOST` set, `GET /sync/rest/` returned **200** (not 503).
  - **FDW repoint resolved:** `hq_pg` → `host.docker.internal:60237/hq_real`;
    `hq_field_templates` = 7 rows through the bridge; RLS chain diagnostics field→template=1,
    admin-arm rows for user=1.
  - `POST /api/v1/workflow/saveResponse` → **204**; draft `submission_responses` row landed in
    HQ carrying the sentinel.
  - **Red-first (no carrier):** the real `workflows.html` opened replication — the log shows all
    four production collections fetched through the proxy at **200**
    (`/sync/rest/checklist_templates`, `/sync/rest/submission_responses` with the exact
    `or=(submission_id.eq.<cid>, and(submission_id.is.null, field_id.in.(<F>)))` draft filter,
    `/sync/rest/checklist_submissions`, `/sync/rest/submission_rejections`) — and the dev surface
    stayed at `data-state="waiting"`. Did NOT surface → the armed assertion is **not vacuous**.
  - **Armed (carrier landed the row):** `#sync-one-row` reached `data-state="served"` in **673 ms**,
    `id=spikef-f20260808232119`, `value="spikef-f20260808232119-…N"` — the sentinel written
    through the real `/saveResponse`, served out of `db.responses` in the real app.
  - **Restore VERIFIED:** `submission_responses` id-set byte-identical to the pre-run baseline;
    FDW `hq_pg` restored to `host.docker.internal:5434/hq_test_b2_fdw`; scratch HQ destroyed.

**Conclusion:** leg 3 — the real browser against the real substrate through the app — is
**proven achievable**. It is no longer "unproven novel integration"; the `sync-live-in-dev`
card can be slated. Two build-facts the card inherits from this run, neither blocking:
(1) the proxy needs **four** vars, not the two the roadmap named — `HQ_SYNC_JWT_SECRET`
(equal to the substrate JWT secret) and `HQ_SYNC_REALTIME_HOST=realtime-dev.localhost` are also
required; (2) the production per-user RLS resolves through the **FDW server `hq_pg`**, so leg 1/2
must arrange a persistent FDW→HQ pointing for the operator's `dev:tailscale` HQ (with the
`hq_sync_fdw` role given LOGIN), not only for a scratch container — see the FDW finding above.

## Corrections

- none — no agent-reached corrections. The script passed on its first run; nothing was
  investigated-and-changed, so there is nothing to batch-review.

## Review

- n/a — no corrections, so no one-sitting batch review is required (§4 fires only when the
  ledger holds agent-reached corrections). The goal is settled and slatable.
