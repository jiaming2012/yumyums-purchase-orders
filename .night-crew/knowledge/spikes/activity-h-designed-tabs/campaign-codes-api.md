# Spikes — campaign-codes-api

Activity: Activity H — Campaign admin, subscribers, stats (the designed tabs)

> No `usm/roadmap.txt` on this target — the `night-crew spikes gate/run` verbs
> cannot resolve a product-mode activity by name (re-confirmed this sitting:
> `spikes gate --activity "Activity H"` → "no roadmap activity named … found",
> and the same for "Activity B"), so this ledger follows the hand-run
> convention every Activity A–G ledger records. Scripts ARE the verdict
> (B-345); verdicts below are the scripts' exit codes, hand-run 2026-10-01.
>
> **Substrate:** the committed LOCAL `spike-supabase` compose project only,
> brought to GREEN by `env-up.sh` in reconcile mode this sitting (exit 0,
> finished 2026-10-01T15:36:47Z). Never :5433, never a hosted project.

## The goal, and which legs need a spike

The card (roadmap Activity H, H1): campaign admin in HQ Go + Postgres —
`campaigns_admin` / `qr_codes` / `qr_scans`, one QR per channel minted on
create, a Supabase `campaigns` **projection** (decision 187), the public
`GET /q/{short}` landing, a PNG endpoint, manager tier enforced in the
handler. Three premises are falsifiable before a line of the card is written:
the QR is scannable and small, the projection write path exists, and the
handler can read the caller's tier without new plumbing.

## Spike: qr-png-decodes-with-vendored-scanner

- proves: a `skip2/go-qrcode` PNG of the 35-character short URL is version ≤ 3
  (≤ 29 modules a side — sign-scannable) AND decodes to the identical string
  with the exact scanner library the tablets ship (`lib/html5-qrcode.min.js`,
  `scanFile`), so the card's two QR claims (small, decodable) hold with the
  real libraries, not a different encoder/decoder pair.
- plan: throwaway Go module under TMPDIR generates the PNG and reports the
  module count; a Playwright Chromium page loads the vendored scanner and
  decodes the file; assert version ≤ 3 and decoded == input.
- script: .night-crew/spikes/activity-h-designed-tabs/campaign-codes-api/01-qr-png-decodes-with-vendored-scanner.sh

## Spike: projection-upsert-over-postgrest

- proves: an HQ-side service-role JWT can UPSERT the four tablet columns of a
  campaign into Supabase `campaigns` over PostgREST
  (`Prefer: resolution=merge-duplicates`) and read it back — the write path
  decision 187 rests on, against the real RLS (service role bypasses; the
  device role has SELECT only).
- plan: mint a service_role token with the committed throwaway secret
  (`mintjwt`), POST the TEST fixture campaign a0…0001 with a new name, GET it
  back, then PATCH the original name back (reconcile, never destroy).
- script: .night-crew/spikes/activity-h-designed-tabs/campaign-codes-api/02-projection-upsert-over-postgrest.sh

## Spike: manager-tier-derivable-in-handler

- proves: the §16 rule "enforce the manager tier inside the handler" has
  something to read — `auth.UserFromContext(ctx) *User` exists and
  `User.Roles []string` is on it — so `403 managers_only` needs no new
  middleware and no extra DB read. Minimal runnable check for a premise that
  is otherwise code-reading (FR-13a: no "no spike needed" escape).
- plan: grep both symbols in `backend/internal/auth`, then `go build` the
  package.
- script: .night-crew/spikes/activity-h-designed-tabs/campaign-codes-api/03-manager-tier-derivable-in-handler.sh

## Verdict (hand-run 2026-10-01)

- **qr-png-decodes-with-vendored-scanner: passed** — exit 0, first run.
  `modules=29` (version 3) for the 35-char URL; `html5-qrcode.scanFile`
  decoded `https://hq.yumyums.kitchen/q/7KQ2M3` exactly. Note for the card:
  version 3 is the ceiling at Medium error correction — a longer host or a
  7-char key tips to version 4 (33 modules); keep the short URL short.
- **projection-upsert-over-postgrest: passed** — exit 0, first run. Upsert
  answered HTTP 200 with the representation; read-back showed the projected
  name; fixture name restored.
- **manager-tier-derivable-in-handler: passed** — exit 0, first run.

## Corrections

- none — no agent-reached corrections

## Comebacks

- none recorded — no gap surfaced

## Review

- signed: operator, 2026-10-01 — covers 0 correction(s) (no agent-reached corrections; reviewed in the same batch sitting)
