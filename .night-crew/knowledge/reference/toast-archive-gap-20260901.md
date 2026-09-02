# Toast archive gap — enumeration & recovery (as of 2026-08-31)

**Run:** `20260901` · **Card:** `toast-ingest-resurrection` (Track A, Card 3) ·
**Closes:** B-146 (mechanism half) · **Author:** implementer subagent c03

This document has two parts:
1. The **archive gap enumerated day-by-day** with a recoverable-vs-aged-out
   disposition (P-KR2), dated against tonight (2026-08-31) and Toast's ~27-day
   SFTP retention window.
2. The **attended-steps note** — exactly what the operator does, attended, on the
   prod box after this change merges to `main` and deploys.

---

## Part 1 — The gap, day by day

### Established facts (measured, from B-146 / B-145 Phase 0, 2026-08-06)

| Fact | Value |
|---|---|
| Last date-directory written to the Spaces/B2 Toast archive | `toast/20260724` |
| That directory's last write timestamp | 2026-07-26 |
| Prod container rebuilt (sync silently died here) | 2026-07-28 |
| Cause | SFTP key never shipped: `TOAST_SFTP_KEY_PATH=./id_rsa` → `/app/id_rsa`, no such file; image doesn't COPY it; `git reset --hard` never delivers the git-ignored key; `docker inspect` showed zero mounts |
| Toast SFTP retained window (measured 2026-08-06) | `20260710`–`20260805` (~27 days rolling) |
| Spaces archive coverage at that measurement | `20260305`–`20260724` |

### The gap window

- **Last good archived day:** `20260724` (2026-07-24).
- **Gap begins:** `20260725` (2026-07-25) — the first day never archived.
- **Gap ends (today):** `20260831` (2026-08-31).
- **Total gap length:** **38 days** (2026-07-25 through 2026-08-31, inclusive).

### Retention window as of tonight (2026-08-31)

Toast's SFTP side retains a **~27-day rolling window**. Applied to tonight, the
oldest day still fetchable from SFTP is approximately **2026-08-04**
(2026-08-31 − 27 days). Everything older has rolled off the SFTP side and exists
nowhere we can reach — the Spaces archive stops at `20260724`, and SFTP no longer
holds `20260725`–`20260803`. Those days are **permanently lost**.

> Retention is a *rolling* window and Toast's exact horizon is approximate
> (measured 27 days once, on 2026-08-06). Treat the 2026-08-04 boundary as the
> best current estimate; a day sitting right on the boundary (2026-08-03 /
> 2026-08-04) may fall either side by the time a sync actually runs. Fetch the
> whole recoverable window and let SFTP-miss (a silent INFO per `worker.go`
> D-05) tell you which oldest days have already aged out.

### Day-by-day disposition

**AGED-OUT — permanently lost (10 days, older than the ~27-day window):**

| Date-dir | Calendar date | Disposition |
|---|---|---|
| `20260725` | 2026-07-25 | AGED-OUT — rolled off SFTP; not in Spaces; unrecoverable |
| `20260726` | 2026-07-26 | AGED-OUT — unrecoverable |
| `20260727` | 2026-07-27 | AGED-OUT — unrecoverable |
| `20260728` | 2026-07-28 | AGED-OUT — unrecoverable (day of the fatal rebuild) |
| `20260729` | 2026-07-29 | AGED-OUT — unrecoverable |
| `20260730` | 2026-07-30 | AGED-OUT — unrecoverable |
| `20260731` | 2026-07-31 | AGED-OUT — unrecoverable |
| `20260801` | 2026-08-01 | AGED-OUT — unrecoverable |
| `20260802` | 2026-08-02 | AGED-OUT — unrecoverable |
| `20260803` | 2026-08-03 | AGED-OUT — unrecoverable (last aged-out day; borderline — verify against live SFTP miss) |

**RECOVERABLE — still fetchable on the next sync (28 days, within the window):**

| Date-dir range | Calendar range | Disposition |
|---|---|---|
| `20260804`–`20260831` | 2026-08-04 → 2026-08-31 | RECOVERABLE — within Toast's ~27-day SFTP retention as of tonight; the next sync (or a targeted backfill) will fetch these SFTP→Spaces→DB. **28 days.** |

Individual recoverable days: `20260804, 20260805, 20260806, 20260807, 20260808,
20260809, 20260810, 20260811, 20260812, 20260813, 20260814, 20260815, 20260816,
20260817, 20260818, 20260819, 20260820, 20260821, 20260822, 20260823, 20260824,
20260825, 20260826, 20260827, 20260828, 20260829, 20260830, 20260831`.

### ⚠️ Recovery of the recoverable window is NOT automatic — and there is no SFTP range-backfill CLI

The in-process worker re-pulls only a **rolling window** per tick:
`SyncWindowDays = 7` normally, `BackfillDays = 90` **only on cold start** (empty
DB). Prod's DB is NOT cold — it holds Toast data through `20260724` — so the
first successful tick after the key ships will fetch SFTP→Spaces→DB for just the
**last 7 days** (≈ `20260825`–`20260831`). It will **not** reach back to
`20260804` on its own, and the days it does fetch are the only ones that heal
automatically.

**There is no CLI that pulls a *date range* from SFTP.** The two Toast CLIs are
narrower than that (verified by reading their headers):
- `cmd/sync-toast --from --to` **reads from Spaces**, not SFTP — it re-ingests
  Spaces→DB. It can only replay days already archived in Spaces; it cannot fetch
  the gap from SFTP. Useless for closing the gap.
- `cmd/migrate-toast-archive --source <dir>` uploads a **local filesystem**
  archive (sales-processor's `output/toast_reports/`) → Spaces. It never touches
  SFTP either.

So the SFTP→Spaces fetch for days older than the worker's 7-day window has **no
turnkey path**. Options for the recoverable `20260804`–`20260824` slice (21 days
beyond the auto-healed 7):

1. **sales-processor archive → `migrate-toast-archive`** (preferred). sales-processor
   is a peer that pulls the SAME Toast SFTP export on its own schedule. If its
   local `output/toast_reports/` holds `20260804`–`20260824`, run
   `migrate-toast-archive --source <that dir>` to seed Spaces, then let the next
   worker tick (or `sync-toast --from 2026-08-04 --to 2026-08-31`) ingest Spaces→DB.
   **Check sales-processor first** — it may already have the whole recoverable
   window on disk, which would make this the complete fix.
2. **Temporarily widen the worker window.** A short-lived deploy with
   `SyncWindowDays` bumped (code change, not env-exposed today) or repeated ticks
   would eventually pull more days SFTP→Spaces, but the value isn't an env knob —
   treat this as a last resort requiring a code tweak.

Every day older than ~2026-08-04 will SFTP-miss (INFO log, silent per D-05) —
that is expected and confirms the aged-out boundary empirically.

**Retention is burning:** every day the sync stays down, one more recoverable day
slides into aged-out. Placing the key AND recovering the >7-day slice (via
sales-processor's archive) is time-sensitive.

---

## Part 2 — Attended steps for the operator (post-merge, on the prod box)

These run **attended**, on the prod box, **after** this branch merges to `main`
and `task prod:deploy` runs. **No prod credential entered the night-crew run** —
the key is placed here, by you, by hand.

### Step 1 — Place the SFTP key beside the compose file

The prod clone lives at `PROD_REPO` (default
`/mnt/c/Users/jcole/projects/yumyums-purchase-orders`). The compose file is
`docker-compose.prod.yml` at its root. Put the Toast SFTP **private** key there,
named exactly `id_rsa`:

```
cp /path/to/your/toast/id_rsa  $PROD_REPO/id_rsa
chmod 600 $PROD_REPO/id_rsa
# ensure it's owned by the user that runs `docker compose`
```

- **Filename must be `id_rsa`** — the bind-mount is `./id_rsa:/app/id_rsa:ro`.
- **Permissions: `600`** (owner read/write only). SSH/SFTP clients reject
  world-readable private keys; `600` is required.
- `id_rsa` is git-ignored (`.gitignore:10`), so `git reset --hard origin/main`
  during `task prod:deploy` will **not** delete it — same durability the
  `.env.prod` secrets already have.
- This is the SAME key the dev repo uses at `backend/id_rsa`. Copy that one.

### Step 2 — Deploy (or redeploy) so the mount takes effect

```
task prod:deploy
```

The updated `docker-compose.prod.yml` now (a) bind-mounts `id_rsa` → `/app/id_rsa`
read-only, (b) pins `TOAST_SFTP_KEY_PATH=/app/id_rsa`, and (c) sets
`TOAST_SYNC_INTERVAL=12h` to enable the worker. If the key file is **absent** when
the container boots, the server now **fails fast** (config.go `os.Stat` guard) —
a loud crash on boot, not a silent dead sync. That is intended: a missing key is
now impossible to miss.

### Step 3 — Verify the sync is alive

1. **Health field flips to `ok`:**
   ```
   task health:prod        # raw /api/v1/health JSON from prod
   ```
   Confirm the `toast_sync` field (Card 2, B-146 fail-loud) reports **`ok`** —
   not `failing` / `stale` / `unknown`. The worker runs a cycle immediately on
   start, so this should update within a minute of a healthy boot. A `failing`
   here means SFTP dial/auth failed — recheck the key file (path, name,
   permissions, and that it's the *private* key).

2. **A current date-directory appears in the archive:** after the first
   scheduled sync lands, a **new** `toast/YYYYMMDD/` directory for a current
   date (e.g. tonight's or tomorrow's) should appear in the Spaces/B2 archive —
   the first fresh write since `20260724`. This is the **prod proof** the roadmap
   flip defers to; the dev-provable half is done in the run, this is the
   attended confirmation.

### Step 4 — Recover the >7-day recoverable slice (time-sensitive)

The worker auto-heals only the last 7 days (≈ `20260825`–`20260831`). The
remaining recoverable days `20260804`–`20260824` (21 days) do **not** heal on
their own and there is **no SFTP range-backfill CLI** (see Part 1's warning).
Preferred path:

1. Check whether **sales-processor** (the peer that pulls the same Toast SFTP
   export) still has `20260804`–`20260824` in its local
   `output/toast_reports/`. It very likely does.
2. If so: `go run ./cmd/migrate-toast-archive/ --source <that dir>` to seed those
   days into Spaces, then `go run ./cmd/sync-toast/ --from 2026-08-04 --to 2026-08-31`
   to ingest Spaces→DB.
3. Aged-out days (`20260725`–`20260803`) will simply be absent from
   sales-processor too if it was also affected — but sales-processor pulls on its
   own schedule, so it may hold days HQ lost. Worth checking; anything it has is
   recoverable regardless of HQ's own retention math.

Do this promptly: every elapsed day loses one more recoverable day to Toast's
rolling retention.

### What is permanently lost

The 10 aged-out days `20260725`–`20260803` cannot be recovered from any source —
they rolled off Toast's SFTP retention while the sync was dead and were never
archived to Spaces. Downstream COGS/sales attribution for that window will have
a hole; no action closes it.
