# merge-intent — S1a `sync-cutover-list-scope`

Run `overnight-20260803`, Track A, first card. Branch `card/s1a-sync-cutover-list-scope`,
cut off `overnight-20260803`.

Written BEFORE implementing (§15ad.65). Updated in place only for facts that changed.

---

## Shared files touched — each line says why

| File | Why this card touches it |
|---|---|
| `sync-rxdb/client.js` | **The card's own file.** `normalizeScope` / `scopeFilterFor` gain a second scope MODE (list) beside the fill scope, plus `realtimeFilterFor` and the `client.channel` shim that carries B-42 option (i). |
| `sync-schema/sql/0004_write_policies.sql` | **COMMENT-ONLY.** B-58: three comments assert `W11` guards `submission_rejections_update`'s `USING` clause; measurement says it does not. No policy, function, grant or `create`/`drop` statement is changed — `git diff` on this file is comments and nothing else. |
| `backend/internal/sync/rowvisibility_rls_test.go` | B-58's discriminating variant (**W17**) plus the list-scope discrimination subtests the card's second `done_when:` row needs, plus the corrected comment at the old line 1922. Additive: no existing variant is deleted or renamed. |
| `tests/sync-rxdb-client.spec.js` | The list-scope unit tests, including the "no date floor ⇒ throw" row. The scope harness (`fakeQuery` / `parseClause`) gains a `gte` operator it did not have, because the list scope's floor uses one. **Additive to the harness; every existing operator behaves as before.** |
| `.night-crew/knowledge/roadmap.md` | The card's status flip, required in the same change set. |
| `.night-crew/knowledge/BACKLOG.md` | Discoveries filed under the standing scope freeze, numbers **B-61..B-64** (allocated to this card up front). |
| `.night-crew/runs/2026-08-03-autonomous/merge-intents/` | This note. |

**NOT touched, and that is a `done_when:`-level constraint of this card, not a footprint hint:**
`sync-rxdb/bootstrap.js`, `workflows.html` — S1b owns the wiring.
`sync.js`, `autoSaveField`, `/saveResponse` — no write path moves on this card.
`sync-schema/collections.js` — byte-unchanged; no schema key is added (that is the PARK trigger).
`sync-schema/sql/0001..0003` — byte-unchanged.

---

## What must survive any merge

1. **`normalizeScope` still THROWS for a scope it cannot recognise.** The fill scope's
   refusals (`checklistId` required, `templateId` required — F-5, the whitelist — F-4) are
   unchanged and must stay unchanged. A merge that makes an unrecognised scope fall through
   to "pull whole" undoes the whole of A1.
2. **The list scope's DATE FLOOR is MANDATORY.** `normalizeScope` throws, naming the floor,
   when a list scope omits it. This is the one thing keeping the widening bounded — B-42
   already recorded that nothing evicts, so an unbounded list scope re-opens the per-phone
   storage bound the whole widening was granted on condition of not re-opening.
3. **The Realtime `filter` is present on exactly THREE collections and ABSENT on `responses`,**
   with the reason at the call site. A merge that "completes the set" by filtering `responses`
   on a single clause silently drops the two-branch offline-draft case.
4. **`submission_rejections_update`'s `USING` clause stays `hq_can_approve_field`.** W17 is now
   the only subtest that reds when it is swapped for `hq_can_see_field`; before this card
   nothing did. A merge that drops W17 restores the unpinned clause B-58 filed.
5. **The corrected comments stay corrected.** 0004 §5d(2), 0004:483 and the W11 banner named
   W11 as the guard on the `USING` clause. Measurement says it is not. A merge that restores
   the old wording restores a file that overstates its own guard — the exact defect class
   card A2 was chartered to remove.
6. **The scope fingerprint distinguishes two USERS on one phone.** A shared truck phone that
   switches crew member must not resume the previous user's checkpoint. The identity term in
   `scopePlanFor`'s fingerprint input is what delivers that.

---

## What is safe to drop

- The **prose volume** of the new comment blocks in `client.js`. Compress freely; the
  behaviour is pinned by tests, not by the comments.
- The exact **wording** of the new throw messages, provided the list-scope floor message
  still contains a token a test can match on (`date floor`).
- The `gte` addition to `tests/sync-rxdb-client.spec.js`'s `fakeQuery` harness **if and only
  if** the list-scope pull tests go with it. Half of that pair is worse than neither.
- The BACKLOG entries B-61..B-64 if another leg of this run filed the same finding first —
  keep one copy, keep the lower number, keep the destination.

---

## Nothing here (stated explicitly, not omitted)

- **Migrations:** nothing here. No `0005_*.sql` was written — the list scope needed no policy
  it does not already have.
- **Schema / collections:** nothing here. `sync-schema/collections.js` is byte-unchanged and
  no new column, view or queryable key was introduced. (Needing one is this card's PARK
  trigger; it did not fire — see the note in `client.js`'s LIST SCOPE block.)
- **Write path:** nothing here.
- **Conflicts expected with other legs of this run:** nothing known. S1b (`sync-hard-cutover`)
  is cut AFTER this card merges and owns `bootstrap.js` / `workflows.html`, which this card
  does not open.
