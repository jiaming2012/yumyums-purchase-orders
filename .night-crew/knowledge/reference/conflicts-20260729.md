# Merge conflict log — `overnight-20260729`

> DESIGN §15ad.66. **Every** merge to the run branch gets an entry here, clean or conflicted.
> Clean merges get a one-line entry on purpose — so an empty log can never be misread as
> "no conflicts" when what it actually means is "the logging never ran".
>
> Run: `overnight-20260729` · Slate: `slate-20260729.md` (signed 2026-07-28, 4 cards)
> Merge order: A (Wave 0, alone) → then Track A (B) ∥ Track B (C → D), merged in slate order.
> Merges are performed by the orchestrator only. Conflicts are resolved by reading BOTH sides'
> merge-intent notes under `.night-crew/runs/2026-07-29-autonomous/` and resolving against
> **intent, not text** — then re-running G1+G2.

| # | Card merged | Files / hunks | Verdict | Gate result after merge |
|---|---|---|---|---|
| 1 | A · `precache-manifest-from-head` | 7 files, 0 conflicted hunks | **CLEAN** | G1 green, `sw.js` rebuild idempotent |
| 2 | C · `sync-proxy-endpoint` | 7 files, 0 conflicted hunks | **CLEAN** | G1 green, Go suite green |
| 3 | B · `workflow-queue-period-and-failnote-upsert` | 10 files, **1 conflicted hunk** (`version.go`) | **RESOLVED** | G1 green, 9/9 Go packages green, `sw.js` idempotent |

---

## 1 — Card A `precache-manifest-from-head` → `overnight-20260729`

**Merge type:** CLEAN. Wave 0, merged alone, first. No other card had landed, so there was
nothing to conflict *with* — this entry exists precisely so that fact is on the record rather
than inferred from an empty file.

**Files carried in (7):**

| File | Note |
|---|---|
| `build-sw.js` | the fix — `git ls-files` (index) → `git ls-tree -r --name-only -z HEAD` (commit) |
| `tests/sw-manifest.spec.js` | test 1 co-moved off `ls-files`; new red-first staged-probe test |
| `sw.js` | regenerated through the **fixed** globber |
| `package.json`, `backend/internal/version/version.go` | Frontend 1.2.0 → 1.2.1, mirrored |
| `.night-crew/knowledge/roadmap.md` | card flipped to DONE |
| `.night-crew/runs/2026-07-29-autonomous/merge-intent-a-…md` | the card's merge-intent note |

**Intents read:** only Card A's. Its note flags three shared surfaces for the cards still to come,
and they are recorded here because later merges will need them:

1. **`build-sw.js`** is shared with the already-landed `pwa-cache-and-build-hygiene`, which
   authored `trackedFiles` / `GENERATED_BUT_SHIPPED`. Card A renamed
   `trackedFiles`→`committedFiles` and `trackedOnlyTransform`→`committedOnlyTransform`. **Any
   future conflict here is a rename collision, not a logic collision.** The property that must
   survive is *"nothing outside HEAD ∪ allowlist reaches the manifest"* — not the spelling.
   `-r` and `-z` are both load-bearing.
2. **`backend/internal/version/version.go` + `package.json`** are shared with every card tonight.
   On conflict: take the **higher** Frontend semver and **re-mirror into both** — CLAUDE.md
   forbids them diverging.
3. **`sw.js`** is generated. Never hand-merge it — take either side, then re-run `node build-sw.js`
   and commit the result. Cards B and C regenerate it *through the globber this card just fixed*,
   which is why A was Wave 0 and landed alone.

**Resolution taken:** none required.

**Gate result after merge:** `go build ./...` + `go vet ./...` green on the merged tree;
`node build-sw.js` at the merge commit leaves the tree clean (22 files / 1463.6 KB, unchanged),
confirming the committed `sw.js` is what the fixed globber produces.

**G6:** APPROVE-WITH-NITS (no blocking findings). Nits 1 and 2 are carried to the closeout as
backlog candidates, not fixed in-run — see HANDOFF.md.

---
## 2 — Card C `sync-proxy-endpoint` → `overnight-20260729`

**Merge type:** CLEAN. Card C was cut from `25fbc16` (i.e. *after* Card A landed), so it already
contained Card A's work and the two shared surfaces Card A flagged never came into tension:

- **`backend/internal/version/version.go`** — Card C bumped `Backend` 0.2.2 → 0.3.0 and left
  `Frontend` at Card A's 1.2.1, so the mirror rule was satisfied without a resolution. Card C's
  note asks that a future conflict here take **the higher semver per constant independently** — a
  `Backend` conflict must not drag `Frontend` backwards.
- **`.night-crew/knowledge/roadmap.md`** — Card C flipped its own card and annotated the parent
  card's obligation 6. Disjoint from Card A's flip.

**The collision that did NOT happen, and is still pending:** `backend/cmd/server/main.go` is the
one file Cards B and C were both expected to touch. **Card C merged first**, adding +26 lines at
`:418-444` — a new **root-level** group between `/ws` and `r.Route("/api/v1", …)`. Card B's
workflow routes live inside `/api/v1/workflow` (~`:526-540`), a different region with unrelated
semantics. **When Card B merges, keep both hunks.** Three properties of Card C's registration are
load-bearing and must survive any resolution:

1. **Root-level, NOT under `/api/v1`** — `/api/v1/sync/token` already occupies that prefix and a
   wildcard there would shadow it.
2. **`auth.Middleware` and nothing else** — deliberately outside `RequirePermission`, same
   reasoning as `/sync/token`. Picking a grant would invent a permission concept, which is the
   parent card's park trigger.
3. **`r.Handle`, not `r.Mount`.**

**Intents read:** Card C's note (re-read in full three times per B-11; five lines struck across two
repair rounds). Card A's note, for the version-mirror rule. Card B's note was read but not needed —
no file in this merge is one Card B has committed to.

**Resolution taken:** none required.

**Gate result after merge:** `go build ./...` + `go vet ./...` green on the merged tree; Go suite
green (see below). No frontend file changed, so `sw.js` was not regenerated and needed no rebuild.

**G6 history — three passes, unusually deep for one card:**

| Pass | Verdict | What it established |
|---|---|---|
| Full review | APPROVE-WITH-NITS | 6/6 planted mutants killed; live test cannot be made to lie; **confirmed the signed card's premise was wrong** — a naive `ReverseProxy` handles the 101 and the byte pump unaided; it fails at *tenant lookup* via the Host header, not at the protocol switch |
| Repair-1 delta | APPROVE-WITH-NITS | 38-vector attack matrix; traversal rejection holds incl. on WebSocket upgrade (rejected before the hijack); rule **not** over-broad against live PostgREST vocabulary; found the `%2f` check itself bypassable via Go's `EscapedPath()` fallback |
| Repair-2 delta | APPROVE-WITH-NITS | Bypass **closed** — 31 attack vectors all 400 with 0 upstream hits; **not over-broad** — 31 legitimate vectors through live PostgREST + Realtime, 0 rejected by the door |

**Two repair rounds were spent** (the policy maximum). Remaining nits are carried to the closeout
as follow-ups, NOT fixed in-run:

- `proxy.go:257-264` — a comment describes an `out.URL.RawPath = ""` assignment **that does not
  exist**. Harmless today (the stale RawPath disagrees with the trimmed Path, so `EscapedPath()`
  always falls back to re-escaping). The hazard is the obvious "fix": a refactor that trims RawPath
  in parallel with Path makes them agree, `EscapedPath()` starts returning the caller's spliced
  bytes, and the wire path becomes attacker-controlled.
- `proxy.go:203-205` — the rejection WARN logs `r.URL.EscapedPath()`, i.e. the **laundered** path
  for exactly the class it just rejected. `reason=encoded_slash` is the only surviving signal.
- `main.go:436-438` — its comment reads as an all-clear ("inert until a deploy configures it")
  inches from the env-var names, with no "and do not configure it yet". **Deliberately left alone
  to protect this merge's collision surface with Card B.**

**🛑 Operator-facing, carried to HANDOFF:** the door is inert in every current deploy
(`HQ_SYNC_REST_URL` / `HQ_SYNC_REALTIME_URL` unset → 503), and **must stay that way until
row-visibility RLS lands** — it forwards every method to PostgREST with a `role: authenticated`
JWT and empty grants. The constraint is now recorded in `proxy.go`, the parent card's obligation 6,
and the DONE card.

---
## 3 — Card B `workflow-queue-period-and-failnote-upsert` → `overnight-20260729`

**Merge type:** **CONFLICTED — one hunk**, in `backend/internal/version/version.go`. This is the
collision Card A's merge-intent note predicted on the night's first merge, and it resolved exactly
by the rule that note laid down.

### 🛑 Merge order deviation, stated plainly

The launch prompt says merges happen **in slate order** (A → B → C → D). **Actual order was
A → C → B → D.** Card B's implementer ran ~105 minutes (three full Playwright suites, two of them
correctly discarded — see below), while Card C finished implementation in ~25 minutes and then went
through two repair rounds. Holding Card C's merge until Card B finished would have serialised the
two tracks the same prompt told me to run concurrently. I took concurrency over merge ordering and
am recording the deviation rather than leaving it to be inferred from timestamps.

**Nothing was harmed by it**: Cards B and C are disjoint in every file except `version.go`, and the
resolution rule is order-independent (take the higher semver per constant). Had they collided in
`backend/cmd/server/main.go` as expected, merging C first would still have been safe — C's hunk is a
root-level group and B never touched that file at all.

### The conflict

    <<<<<<< HEAD (run branch, after Card C)     |  ======= (card/b-...)
        Backend  = "0.3.0"                      |      Backend  = "0.2.3"
        Frontend = "1.2.1"                      |      Frontend = "1.2.2"

Both cards branched from `25fbc16`, where `Backend` was `0.2.2` and `Frontend` was Card A's `1.2.1`.
Card C bumped `Backend` → `0.3.0` (**minor** — new `/sync/*` endpoint). Card B bumped `Backend` →
`0.2.3` (**patch** — repository + migration) *and* `Frontend` → `1.2.2` (**patch** — the vocabulary
rename in `workflows.html` / `sync.js`).

**Intents read, both sides:**

- **Card C's note:** *"take the higher semver per constant independently. A `Backend` conflict must
  not drag `Frontend` backwards if Card B moved it."*
- **Card A's note:** *"on conflict take the higher Frontend semver and re-mirror into both"* —
  `version.go` and `package.json` must never diverge (CLAUDE.md).

**Resolution taken — against intent, not text:**

    Backend  = "0.3.0"   // C's minor supersedes B's patch; BOTH backend changes are present
    Frontend = "1.2.2"   // B's patch; C touched no frontend file

Neither side's number was taken wholesale — the resolution is per-constant, which is precisely what
the two notes asked for and what a textual "ours/theirs" would have got wrong in both directions
(`--ours` loses B's frontend bump; `--theirs` regresses the backend from a shipped 0.3.0 to 0.2.3).

`package.json` merged **clean** at `"1.2.2"` and already mirrors the resolved `Frontend`. `sw.js`
was regenerated after the resolution and came back **byte-identical** to what Card B committed
(22 files / 1468.9 KB) — the frontend tree is the one Card B actually tested.

**Gate result after resolution:** `go build ./...` + `go vet ./...` green; **full Go suite green,
9/9 packages** (alerts, auth, inventory, purchasing, receipt, recipes, sync, toast, workflow) with
real DB timings — not the silent-skip trap. `node build-sw.js` idempotent at the merge commit.

### G6 verdict: APPROVE-WITH-NITS, no blocking defect

The park trigger was the night's highest-stakes question and G6 answered it independently and
read-only: it enumerated the schemas rather than trusting the two the card named (only `production`
and `public` exist), found **0 fail-note rows and 0 `(submission_id, field_id)` duplicates in both**,
and then swept **all 29 connectable databases on the host**. The only duplicates anywhere were in a
prior reviewer's own scratch DB. It further proved the migration **fails safe** on a DB that does
have duplicates — clean rollback, no index, goose stays at 70, rows intact — and that the
`CREATE UNIQUE INDEX` lock is a non-issue because migrations run before the HTTP listener binds.

Two findings carried to the closeout as follow-ups, **not fixed in-run** (both are outside this
card's footprint):

- **UTC period ⇒ the app's day boundary is 19:00 America/Chicago.** `currentSubmitPeriod()` is
  `new Date().toISOString().slice(0,10)`. A double-press straddling 19:00 while offline now mints a
  second `idempotency_key` and yields two submission rows for one operational evening. **Not a
  regression this card introduced** — the three pre-existing "already submitted today" comparisons
  use the identical UTC expression, so the card is self-consistent with the app's existing day
  model. Fixing it means moving all four sites to `America/Chicago` at once, which is a product
  decision about what "today" means. `users.timezone` already exists. → DECISIONS-NEEDED.
- **Deploy-ordering hazard.** If a duplicate reaches prod before the deploy, migration `0071` fails
  → `os.Exit(1)` → `restart: unless-stopped` ⇒ crashloop. Probability is near zero (prod has had 2
  submissions ever and 0 fail notes) and the bare index is the *instructed* behaviour under the
  card's PARK-over-improvise contract. → HANDOFF: re-run the duplicate `SELECT` immediately before
  `task prod:deploy`.

Also confirmed by G6 and worth recording: the `[VOC-01]` locator change **fixed a test without
hiding a bug** — the decisive collision assertion (`getByText('Pending sync', {exact:true})`
→ `toHaveCount(0)`) remains **document-wide**; what the scoping gave up was an accidental cardinality
check nobody designed. Both badges read "Queued", so the vocabulary contract held.

---
