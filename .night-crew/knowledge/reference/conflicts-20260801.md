# Conflict log — run 20260801

Per §15ad.66: an entry for **every** merge to `overnight-20260801`, clean or conflicted, so an
empty log never reads as "no conflicts" when it means "the logging never ran."

Each entry records: the cards involved, the files and hunks, the merge-intent notes read, the
resolution taken, and the gate result after it.

Pre-resolved by the slate, not findings:
- `sw.js` is GENERATED — never merge the artifact; take either side, regenerate with `task sw`,
  re-run G4 (idempotence + version parity).
- `version.go` — resolve **per-constant**, not per-file (precedent `79fa7cd`, 07-29).
- Migration numbers are ASSIGNED: A1 = `0072`, B2 = `0073`. A card finding its number taken
  **stops and reports**; it does not renumber.

---

## Entries

### Merge 1 — `card/b2-sync-rxdb-row-visibility-rls` → `overnight-20260801`

**Result: CLEAN — zero conflicts.** Logged because every merge gets an entry, so an empty log
cannot be mistaken for "no conflicts" when it means "the logging never ran."

- **Cards involved:** B2 only. No other card had landed at this point.
- **Files:** 12 changed, 3238 insertions / 32 deletions. New: migration `0073_sync_fdw_views.sql`,
  `sync-schema/sql/0002_hq_fdw.sql`, `0003_rls_policies.sql`,
  `backend/internal/sync/{rowvisibility_rls_test.go,spikestack_gate_test.go}`, two QA captures,
  the card's merge-intent and closeout notes. Modified: `proxy_live_test.go`,
  `jwtbridge_rls_test.go`, `.night-crew/knowledge/roadmap.md` (this card's own bullet only).
- **Intents read:** B2's merge-intent note plus its F2/F1/F4 addendum. It declared
  `backend/internal/sync` as its own, no `sw.js` / `version.go` / `workflows.html` /
  `tests/sync.spec.js` change, and `roadmap.md` as its only shared file (one bullet).
  Nothing on the run branch contradicted that.
- **Why no conflict was possible:** B2 had already fast-forwarded `33bd8c0..01107fc`; the run
  branch's only later commits (`2a9d052`, `469e775`) are `DECISIONS-NEEDED.md`, a file B2 never
  touched. The three pre-resolved collision surfaces the slate warned about — `sw.js`,
  `version.go`, migration numbering — were all untouched by this card. Migration `0073` was
  claimed as assigned; `0072` remains free for A1.
- **Resolution taken:** none required. `--no-ff` so the card's history stays identifiable.
- **Gate after merge:** see below.

**Carried forward from this card's review, not resolved by the merge:**
- Fork 1 in `DECISIONS-NEEDED.md` — the ~23 ms/row cost (G6 F3). Operator call at triage.
- `tests/purchasing.spec.js:1792` (FR-23 repurchase-reset) failed once during the fix pass with a
  640 ms backwards jump between two server-side `now()` reads, then **passed in isolation**.
  Newly-observed red, distinct from B-27, not retired by the non-reproduction.
- The merge-intent's "safe to drop … port defaults" bullet is now **false** — those constants were
  the F1 bug. Reintroducing a port constant under `backend/internal/sync/` reintroduces it.
