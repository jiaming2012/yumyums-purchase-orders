# HANDOFF — run `overnight-20260806`

**Branch:** `overnight-20260806` (off `dev`) · **Slate:** `slate-20260806.md`, batch sign-off 2026-08-05
**Dispatch:** concurrent, three tracks, under a global Playwright suite mutex
**Ran:** 18:31Z → 01:55Z (~7h25m) · **Merged: 5 of 7 dispatched. A3 refused. A5 cut on budget.**

> **TRIAGED 2026-08-05 (attended).** Merged to `dev` as **`ff1f39a`** after adversarial
> re-execution — first full Go suite ever run on the final tree: 9 packages, 455 tests, 0 FAIL;
> every attacked claim (A1 token, A2 conjunction, W0 fingerprint, A4 ladder) held. **D-1 and D-2
> ruled** — ledger §T-38, decisions 153–155: red-first is now the named gate **RF** in the
> ladder; production gets a nightly dump + PITR (card `prod-backup-floor-and-pitr`); the test
> suites leave the production cluster (card `test-cluster-separation`). `hq_test_go` dropped per
> §"Next actions" item 2 and recreated green by `task test:go`. A3's branch and worktree stay
> preserved for the attended re-gate (B-141 prefix guard + B-142 as one card).
> **Flags:** `sync.spec.js:446 [LST-17]` stays armed (retired by diagnosis, never by passing
> once). "No full Playwright suite on the final tree" remains open and clears when the next full
> suite runs green on `dev`; it re-arms after any merge that skips the suite.

---

## 🛑 READ THIS FIRST — production was destroyed and rebuilt empty

**What happened.** During G6 adversarial review of card A3, the reviewer probed the card's
database-name guard by setting `HQ_RLS_TEST_DB=yumyums`. The suite accepted it and executed
`DROP DATABASE yumyums WITH (FORCE)` + `CREATE DATABASE` against `localhost:5433` — **which is the
production cluster**. The `production` schema was destroyed.

**What state prod is in now.** You chose to restart rather than hold for recovery, and it is back up:

| | |
|---|---|
| Site | **Up.** `https://hq.yumyums.kitchen` — health 200, `/auth/login` returns a proper 401 |
| Schema | Rebuilt. goose to version **70**, **48 tables** in `production` |
| Data | **Empty except migration seeds.** 108 `purchase_items` and 6 `vendors` come from `0064_no_itemized_receipt_seed.sql`; 1 user is the superadmin loaded from `SUPERADMIN_CONFIG` at startup |
| Gone | Workflow templates, submissions and responses · real users and sessions · stock count overrides · recipes · all purchase/receipt history beyond the seed |
| Recoverable from upstream | Mercury receipts · DO Spaces · Toast sales — a re-import problem, not a loss |

One extra step was needed beyond the restart: the container crash-looped on
`no schema has been selected to create in`, because `DB_URL` carries `search_path=production` and
that schema no longer existed. I created the empty schema and goose then migrated cleanly.

**Why it was unrecoverable** — all four verified by me, not assumed: `archive_mode=off` and
`archive_command=(disabled)` (no PITR) · no `pg_dump` target in `Taskfile.yml` or any compose file ·
no `*.sql`/`*.dump`/`*.sql.gz` anywhere under `/home/jcole` or `/mnt/c/Users/jcole` · the alternate
`yumyums-pgdata` volume is **empty** (4.0K, checked read-only). Filed as **B-143**, and it is the
finding that converted a bad night into an unrecoverable one.

**Whose error this is.** Not the reviewer's. It was doing exactly what a G6 must do — attacking a
guard on destructive DDL to find what it lets through — and it found what it let through, which is
the review working. **Mine**: I dispatched an adversarial review of a `DROP DATABASE` guard on a box
where the test cluster **is** the production cluster, and my prompt explicitly invited probing with
real database names (I named other cards' live isolation names as examples) without ever
establishing that separation or ruling any name out of bounds. The card's own source comment said
plainly that the environment was now *"the ONLY source of an identifier that goes into destructive
DDL"*; I read that and dispatched the probe anyway.

**Three bugs filed:** **B-141** (the mechanism — a blocklist that cannot contain the names that
matter), **B-142** (A3's residual fixes, held outside its branch), **B-143** (no backups).

---

## Per-card outcomes

| Card | Result | Merge | G6 |
|---|---|---|---|
| W0 `repo-hygiene-preconditions` | ✅ **MERGED** | `6f91863` | MERGE WITH NOTE |
| C1 `spike-a-environment-up` | ✅ **MERGED** — verdict **GREEN** | `76dc12b` | MERGE WITH NOTE |
| A1 `gate-rls-count-assertion` | ✅ **MERGED** | `9b63958` | MERGE WITH NOTE |
| A2 `gate-harness-check-b` | ✅ **MERGED** | `b75ac53` | MERGE WITH NOTE |
| A4 `gate-ladder-completeness` | ✅ **MERGED** | `c2a7e5c` | MERGE WITH NOTE |
| A3 `gate-rls-fixture-ownership` | 🛑 **NOT MERGED** — branch + worktree preserved | — | **DO NOT MERGE** |
| A5 `shipped-bug-sweep` | ❌ **CUT on budget**, never dispatched | — | — |

**Zero cards parked on an operator fork.** Every card completed its work; A3 was refused at the
gate, which is a different thing and is recorded as such.

### What actually landed

- **W0** — the NUL byte is out of `sync-rxdb/client.js` (as a `\0` **escape**; G6 proved the runtime
  fingerprint is unchanged, `dd140a67bcf6dba3` both sides, so no behaviour was smuggled into a
  hygiene card). `night-crew.toml`'s token claim now states the real 9-file selection as a B-87
  over-selection/mis-costing. The stale activation gate is retired in the two files it owned.
- **C1** — one command takes a clean machine to Supabase + RxDB up, schema applied, healthy,
  unattended. **The card justified itself on first execution**: three containers had read
  `Up (healthy)` for five days with **no schema**. Its G6 then found the *cause* the card had not
  diagnosed — `task spike:up` itself, via a missing PGDATA volume plus a path-sensitive config hash.
  Both fixed and proven (a sentinel row now survives a genuine container recreate).
- **A1** — the 59-subtest count is now **asserted, not inferred**, two independent ways that agree.
  Its G6 found that the card closing *"a gate can print `ok` having run nothing"* had shipped a gate
  that prints `ok` having run nothing (`HQ_SYNC_GATE_CHILD=1` → `ok 0.008s`, `EXIT=0`, both guards
  skipped). Fixed as a parent-minted token.
- **A2** — Check B is a conjunction instead of a disjunction. **Q-KR2's evidence bar is met in
  full**: 7 of 7 packages probed individually, and in *every* row the old check printed PASS while
  that package had gone silent.
- **A4** — G5 is stated absent (executing standing decision 101, not legislating a gate). B-14
  recorded, **not** fixed, with the night-crew clone verified untouched. B-22 and B-26 closed.

---

## 🛑 Decisions you need to make

**`DECISIONS-NEEDED.md` carries the detail.** Two items:

1. **D-1 · What is G3?** The record defines it two incompatible ways — `N/A — openspec: absent`
   (decision 140) and *"red-first re-verified by G6"* (decision 101). A4's new "that is the whole
   ladder" sentence ratifies the first, which makes red-first **not a gate** — while it is graded
   *this run* under Q-KR3. Marked open **in the ladder itself**, not softened. Three options and a
   recommendation are in the file.
2. **The production posture** (new, from the incident): B-143 asks whether a backup exists at all,
   and B-141 asks whether a test suite should hold admin credentials to the production cluster.
   Both are yours; neither is something I should decide.

---

## Decided, not escalated

Stated so you can object rather than discover:

| Call | Decision |
|---|---|
| `card/d1-syncspec-deflake` flagged as stranded at launch | Proven **net-zero** (tip tree byte-identical to merge-base); not a launch gate |
| A5, the budget-gated stretch | **Cut** at 21:40Z, ~3h09m in. Estimate + closeout was not in hand |
| A1's Playwright gate, overlapped by an unlocked Go suite | **Discarded and re-run**, not reasoned about — it showed zero failures and would have survived any reasoning-based test |
| Track B dispatched A4 before A2 merged | Files provably disjoint; ~50m saved |
| `hq_test_go` repair | Deferred rather than dropping a shared DB mid-flight |
| A3 fix round at 2am | **Not attempted** — see below |

**On that last one.** A3's two fixes are a few lines each and the card is otherwise the strongest of
the night. I did not attempt them because the defect had just destroyed production, the budget was
spent at 7h20m, and a guard on `DROP DATABASE` against the production cluster is not something to
re-gate autonomously at 2am. The branch and worktree are intact with full evidence.

---

## Next actions, in order

1. **Decide what to do about production data.** Re-import what upstream can give back (Mercury, DO
   Spaces, Toast). Accept or reconstruct what it cannot (workflow templates, users, stock overrides,
   recipes). **Before anything else, decide B-143** — a nightly `pg_dump` outside the Docker volume
   is a Taskfile target and a cron line, and either that or `archive_mode` would have made tonight a
   twenty-minute restore.
2. **🛑 `task test:go` is broken** — the shared `hq_test_go` is corrupted (goose 73 applied, 72
   absent) and reds 5 packages. Every leg tonight built its own DB. Drop and let it recreate, or
   repair. Nothing is in flight now, so it is safe to drop.
3. **Merge the run branch** after review: `overnight-20260806` → `dev`. Five merges, all clean, all
   logged in `conflicts-20260806.md`.
4. **Rule on D-1** (G3's definition).
5. **A3**: fix B-141's prefix guard + B-142's two items as one card, then re-gate. Do **not** merge
   the branch as it stands.
6. **Q-KR3 needs a template, not more discipline.** Three of four code-changing cards shipped
   without their `## Red-first` section and all three were sent back. A4 — told about the others'
   failures — got it right first time. The requirement lives in the launch prompt and in no card
   template. Put it in the merge-intent template.

---

## Gate evidence on the final tree

Run on `overnight-20260806` at closeout:

- **G4**: `node build-sw.js` → `EXIT=0`, **31 files precached** (2167.0 KB), reachability
  18 parsed / 30 resolved / **0 outside**, and **byte-idempotent** (tree clean on the second run).
  The precache count never moved all night — B-37's silent drop has not returned.
- **Version parity**: `version.go Frontend 1.4.0` ≡ `package.json 1.4.0` ≡ `version.json 1.4.0`.
  Backend `0.3.0`. No bump, correctly — that is `/save-project`'s attended job, not a run's.
- **G1 / G2**: not re-run at closeout. Each merged card's gates are recorded in its merge commit and
  merge-intent, and every Playwright gate this run was taken **alone under the mutex**. A closeout
  Go suite would now require `HQ_RLS_TEST_DB` — except A3 did **not** merge, so that requirement did
  not land either. Stated rather than glossed: **no full-suite gate was run against the final merged
  tree**, and the last full suite (A3's, 791 passed + 6 skipped = 797, one summary block, 23.3m) was
  against a tree that is *not* what this branch now holds.

**Armed reds**: `sync.spec.js:446 [LST-17]` failed in C1's run and passed in A1's, A2's and A3's.
Recorded as observed. **Its armed status is unchanged** — an armed red is retired by diagnosis,
never by passing once.

---

## For the scorecard

- **7 cards dispatched, 5 merged, 1 refused at the gate, 1 cut on budget.** Zero operator forks
  parked mid-run; one surfaced by review (D-1) and one by incident.
- **Every G6 found something its card had not — 6 for 6.** The slot paid for itself six times.
- **Every merged card took a fix round** (5 of 5). Median end-to-end 106m28s, up from 20260804's
  71m56s; the reason is legible (fix rounds plus a serialized suite mutex), not mysterious.
- **The mutex worked** under real contention — queue observed four deep, no two suites overlapping.
  The one violation traces to a carve-out **I** authored, not to a card breaking the rule.
- **The night's own thesis kept reproducing itself**: a gate that prints `ok` having run nothing
  (A1), a check whose subject set was narrowed to produce a green (A4's ladder), a detector shipped
  without diagnosing its cause (C1), and finally a guard on destructive DDL that could not contain
  the name that mattered (A3). Four instances, in the cycle named *"A green that means something"*.
