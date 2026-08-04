# Handoff — run this in the sales-processor repo

**Task:** account for two weekly periods that fall outside the evidence window used to close §1 of HQ's contract notice.
**Handle:** HQ backlog **B-138** · ledger **T-35 decision 142**
**Raised:** 2026-08-04, by HQ reading the consumer reply — *not* by the consumer.
**Size:** ~10 minutes if the artifacts are there. Longer only if they aren't.
**Blocking:** nothing hard. But do it **before** the HQ timezone deploy, because if the answer is bad you want to know while there is still one boundary to reason about instead of two.

---

## Why this exists

HQ's notice §1 reported that the payroll completeness gate changed twice on 2026-06-06, in opposite directions, and that neither change was published. The open question was whether that ever produced a wrong answer in practice.

The reply closed it with a good argument:

> our completeness gate hard-fails before we write any report or dispatch any transfer, so a spurious `ready:false` would show up as a missing week on disk. We have an unbroken weekly run — reports and transfer ledgers both — for every period from 2026-05-31 through 2026-07-19, with no gaps.

**The argument is sound. The window is short.** It ends **2026-07-19**; today is **2026-08-04**. Two weekly periods sit outside it:

| Period end (Sunday) | Inferred range (Mon–Sun) | Days since period closed | Status |
|---|---|---|---|
| **2026-07-26** | 2026-07-20 → 2026-07-26 | 9 | **Unambiguously overdue** |
| **2026-08-02** | 2026-07-27 → 2026-08-02 | 2 | **May not be due yet** — depends on your run schedule |

By the reply's own detection method, these two weeks are exactly where a spurious block would sit undetected: the test is right, it just hasn't been run over the newest data.

⚠️ **Two inferences of mine to confirm, not trust.** (1) Your period identifiers are **end** dates and your week runs **Monday→Sunday** — inferred from every date you listed being a Sunday, and from HQ's purchasing week hanging off a Monday. (2) The 08-02 period should have run by now — that depends entirely on what day of the week your job fires. If it runs Thursdays, 08-02 isn't late at all and only 07-26 matters.

---

## The check

### Step 1 — do the artifacts exist?

The same two things you cited as evidence: **the report** and **the transfer ledger**, for each of the two periods.

```
# Fill in your own paths — I don't have visibility into this repo's layout.
ls -la <reports-dir>  | grep -E '2026-07-26|2026-08-02'
ls -la <ledgers-dir>  | grep -E '2026-07-26|2026-08-02'
```

Also worth a glance, because it distinguishes "never ran" from "ran and failed":

```
# whatever your scheduler/job log is
grep -E '2026-07-2[0-9]|2026-08-0[0-4]' <run-log>
```

### Step 2 — what does HQ say about those periods *right now*?

Read-only. Run from anywhere that has the service token (HQ's is in `.env.prod` beside the prod compose file; **do not paste the token into a file or a chat**).

```
TOKEN='<HQ_INVENTORY_SERVICE_TOKEN>'

for range in "2026-07-20 2026-07-26" "2026-07-27 2026-08-02"; do
  set -- $range
  echo "=== period $1 -> $2 ==="
  curl -sS -H "Authorization: Bearer $TOKEN" \
    "https://hq.yumyums.kitchen/api/v1/inventory/period-summary?from=$1&to=$2" \
  | python3 -m json.tool
done
```

Look at `completeness.ready`, `completeness.pending_review_ids`, and `completeness.pending_review_details` — the last one carries vendor, date, amount and reason, so it tells you *why* a period is blocked without a second call.

🛑 **A `ready: true` today does NOT prove it was true then.** The gate reads current data. If a period blocked in July on an unreviewed receipt and that receipt has since been reviewed and confirmed, HQ now answers `ready: true` for the same period. So this step can produce a false all-clear and cannot exonerate the past on its own.

**The reverse is strong evidence.** A `ready: false` today means the period is *still* blocked — which would both explain a missing run and make the problem live rather than historical.

### Step 3 — reconcile

| Artifacts present? | HQ says today | Reading |
|---|---|---|
| **Both periods present** | either | ✅ **§1 fully closed.** The window just wasn't extended. Say so and the contract audit's §1 note should be widened to cover it. |
| **Missing** | `ready: false` | 🛑 **Live blocked run, cause visible.** Read `pending_review_details` — it names the receipts. This is the symptom §1 was hunting, and it's current, not archaeology. Clear the review queue, re-run, then decide whether the block was *correct* (a genuine unreviewed receipt) or *spurious* (the 06-06 narrowing catching something it shouldn't). |
| **Missing** | `ready: true` | ⚠️ **Ambiguous, and the most annoying outcome.** Either the job didn't fire for an unrelated reason (scheduler, credentials, disk), or it blocked at the time and the cause has since been resolved. Step 1's run log is what separates those — "no entry" vs "entry, gate failed". |
| **07-26 present, 08-02 missing** | either | Probably just not due yet. Confirm against your run schedule and stop. |

---

## What to record when you're done

Back in HQ (`/home/jcole/projects/hq`), one of these:

- **All clear** → close **B-138** in `.night-crew/knowledge/BACKLOG.md`, and widen the §1 evidence window in `docs/contracts/inventory-period-summary.md` from *"through 2026-07-19"* to whatever it actually now covers. The window being explicit is the whole point — a closed question with a stale window is what produced this handoff.
- **Anything else** → leave B-138 open with what you found, and note whether the block was correct or spurious. A *correct* block that nobody noticed for nine days is a different (and arguably worse) finding than a spurious one: it means the blocked-payroll signal isn't reaching anyone.

Either way: **HQ still owes this repo the timezone changeover date.** That's the only thing the reply said it was waiting on, and it is unrelated to this check.

---

## What I could not fill in

Stated plainly rather than guessed, because a handoff with invented paths is worse than one with blanks:

- **Report and ledger directory paths** — this repo isn't on the box HQ runs on (`/home/jcole/projects/` holds `hq`, `night-crew`, `slack-trading`, `infra`, `ui-jury`, `claude-skills` and nothing else).
- **The run log's name, format, or whether one exists.**
- **What day of the week the weekly job fires** — which is what decides whether the 08-02 period is late or simply not due.
- **Whether the gate's hard-fail leaves any trace at all when it trips.** If it exits before writing anything, absence of a report is the *only* signal, and that is itself worth fixing: a gate whose failure is indistinguishable from "never scheduled" can't be monitored. If that's the case, note it — HQ has a whole cluster of backlog items on exactly this shape (a check that can't tell you what it actually did).
