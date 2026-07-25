# Merge intent — W1 `sync-spike-stack-and-jwt-bridge`

Branch: `card/w1-sync-spike-stack-and-jwt-bridge` (cut from `overnight-20260725` at `c14cbce`)
Written BEFORE implementation, as the card's first commit.

## Card in one line

Wave-0 feasibility gate for the RxDB/Supabase sync cycle. Stand up a **local, throwaway**
self-hosted Supabase stack (Postgres + PostgREST + Realtime, no Kong/Studio/GoTrue) in a
NEW `docker-compose.supabase.yml`, mint an HS256 JWT from **stdlib Go only**, and prove the
stack accepts it — a discriminating RLS read/write through PostgREST and a Realtime change
event over WebSocket. Deliverable is a **written GO/NO-GO verdict plus an operator-runnable
runbook**, not working sync code.

## Shared files touched

- `.night-crew/runs/2026-07-25-autonomous/merge-intents/w1-sync-spike-stack-and-jwt-bridge.md`
  — this note. New file, unique to this card. No conflict surface.
- `.night-crew/runs/2026-07-25-autonomous/timings.log` — append-only per-leg timing lines,
  prefixed `W1_`. Every card appends its own prefixed lines; conflicts, if any, are
  append-order only and **both sides should be kept**.
- `.night-crew/runs/2026-07-25-autonomous/DECISIONS-NEEDED.md` — **only if this card parks.**
  If written, it is appended as its own `## W1 …` section; keep both sides on conflict.

Everything else this card writes is new and exclusively its own:

- `docker-compose.supabase.yml` (new, repo root — a NEW file, **not** an edit of
  `docker-compose.nc.yml`)
- `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md` (new)
- `.night-crew/qa/spike-supabase/**` (new directory — Go mint program, SQL fixture,
  README runbook half 1)

No product code is touched. `backend/internal/**`, `*.html`, `sync.js`, `tests/**` are
**read-only** for this card. Any file added to this list during implementation is appended
below under "Late additions" with the evidence that forced it.

### Late additions

_(to be filled in only if implementation forces a file outside the list above; "nothing
here" if it stays clean)_

## What must survive any merge

1. **The verdict line.** `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md`
   opens with **GO** or **NO-GO** (or an explicit PARKED) on its first line. Three
   downstream roadmap cards gate on that single line. A merge that reformats the file must
   not bury or soften it.
2. **The evidence blocks.** Every claim in the verdict and the runbook is backed by a
   command that was actually run and its **real captured output**. The output blocks are
   the artifact, not decoration — dropping them for brevity destroys the card's value.
3. **`docker-compose.supabase.yml` and the Go mint program are kept and committed.** They
   are the reproduction path for the operator, who asked to run this spike themselves. They
   are not scratch files to be cleaned up after the proof.
4. **The throwaway-secret banner.** The inline JWT secret / demo keys in
   `docker-compose.supabase.yml` sit under a loud banner naming them as throwaway, and the
   README carries the matching one-line warning never to reuse them anywhere real. This is
   a deliberate, signed reproducibility decision (Supabase's own self-hosted docs ship demo
   keys the same way) — not an oversight to be "fixed" at merge by moving them to `.env`.
5. **The local-only banner** at the top of the README: local Docker, never production,
   never a hosted Supabase project, no real HQ data.
6. **The per-table contract cost paragraph** — text PK, `_deleted`, `_modified` trigger,
   RLS enabled, manual `ALTER PUBLICATION supabase_realtime ADD TABLE`. That count is the
   input that sizes `sync-rxdb-schema-and-replication`.
7. **Runbook half 1 stands alone.** It must remain a complete, runnable document even if a
   later card (W2, RxDB) never runs. The seam where half 2 gets appended is marked; half 1
   is append-only from W2's side.

## What is safe to drop

- Prose wording, section ordering, and headings anywhere in the README or the design doc,
  so long as items 1–7 above survive intact.
- Image tag pins in `docker-compose.supabase.yml`, if a later card has evidence a
  different tag is needed — but the tag actually used must then be re-verified, not
  assumed.
- The `timings.log` lines. They are a record, not a behaviour.
- Anything in this note itself.

## Not done, deliberately

- **No RLS *policy design*.** Which claims map to which grants belongs to the later card
  `sync-jwt-bridge-endpoint` and to the operator. This spike only proves the stack
  *accepts* a Go-minted token and that a policy *can* discriminate. Any policy-semantics
  question found is **recorded as a question**, not answered.
- **No Kong, Studio, or GoTrue** unless bring-up proves one is required — and if it does,
  that fact is written up as a sharp edge rather than quietly absorbed.
- **No wrapper task targets.** The operator-runnable path is a README of real commands,
  deliberately not `Taskfile.yml` targets. Decided; not to be "improved".
- **No sync code, no schema migration, no product wiring.** This card is a gate, not an
  implementation.
- **The stack is left running at card end.** Teardown is documented as its own README step
  so the operator chooses when it goes away.

## Four-HARD-constraints attestation

Each of these, if broken, changes the build or test environment for **every other card in
the cycle**. All four are untouched by this card; the orchestrator can verify with
`git diff --stat overnight-20260725..HEAD`.

1. `backend/go.mod` — **UNTOUCHED**. The JWT mint is stdlib only (`crypto/hmac`,
   `crypto/sha256`, `encoding/json`, `encoding/base64`). **No JWT module added.** The
   Realtime client uses `github.com/coder/websocket`, verified as an existing **direct**
   dependency before use (`backend/go.mod:11 — github.com/coder/websocket v1.8.14`).
2. Root `package.json` — **UNTOUCHED**. It is the Playwright environment for every card in
   the repo; nothing here needs a JS dependency.
3. `docker-compose.nc.yml` — **UNTOUCHED**. Supabase goes in a NEW, separate
   `docker-compose.supabase.yml`, brought up under the explicit distinct project name
   `spike-supabase` so it can never collide with the nc env. Extending the nc env would
   boot Supabase for every night-crew run in this repo forever.
4. Root `Taskfile.yml` — **UNTOUCHED**. See "Not done, deliberately" — the README of real
   commands is the decided operator path.

Merge-gate suites were run against the **already-running** ephemeral pg16
(`nc-f1-postgres-1`, host port 46413), reusing it rather than bringing up another nc env,
with a distinct database (`hq_test_e2e_w1`) and port (`8299`) so nothing collides.
