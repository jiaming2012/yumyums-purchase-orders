# DECISIONS-NEEDED — overnight run `20261006`

**Two cards parked on ONE open decision.** Card 1 `identity-code-and-qr` (mint each new customer
one permanent code and its QR image) was built and proven as specified, then parked by its own
build session on a measurement. Card 2 `mms-send-on-signup` (text that image to the customer)
parked with it, as the slate prescribes — nothing of Card 2 was built.

## Decision 1 — what to do about customer codes the tablet cannot read from a photo

**As the owner, I want every customer who is sent a code to be readable at the window, so that
nobody holding a valid code is turned away.**

### What a crew member would see today, if Card 1 landed as built

About **one customer code in seven** is not read by the tablet's "Scan from photo" — the screen
shows a scan error instead of the offer. It is the **same** customers every time: retrying the
same picture never helps, and a new picture of the same code fails the same way. No scan ever
showed a different customer's offer.

That is measured on the fallback reader the page uses when the device has no built-in barcode
detector. **Whether the truck's actual tablet has a built-in detector was not measured** — if it
does, its real rate could be much better. The live-camera scan was also not measured.

### What was measured, and by whom

| | Card's own code | Same code, stronger error-correction (the "should not read" control) | Bare code, no offer inside |
|---|---|---|---|
| Build session (250 / 250 / ~400 codes) | 85% read | 83% read | ~94% read |
| Independent re-check, fresh codes, through the real "Scan from photo" control | 87% (139/160) | reads as often | 95% (143/150) |

Both agree. The independent re-check then established **why**, which changes the options:

- **Every failing picture is a valid QR code.** A second decoder read 300 of 300.
- **The fault is in the tablet page's reader**, not in what the code carries: on some codes it
  mistakes a patch of data for one of the three corner squares (37 of 37 failures; 0 of 263
  successes).
- **Two things you signed on 2026-10-05 have no measurable effect:** the lighter error-correction
  level and widening the scan area from 320 to 640. The spike behind them drew one sample per
  variant; replayed, its one "bad" sample reads when turned a quarter-turn.
- **Turning the picture and retrying fixed it:** in the shipped page, trying the picture as drawn
  and then at each quarter-turn read **600 of 600** codes, no wrong customer accepted. This was a
  measurement on a scratch copy — nothing was landed.

### Options — what you and the crew would see

- **C′ — teach the tablet's photo scan to turn the picture and retry, then land both cards
  (recommended).** Every code read in the measurement; a small change to the scanner page, no
  change to the customer's code. Costs one more night: a scanner card first, then Card 1 with its
  two disproved gates restated, then Card 2. Recommended because it is the only option measured to
  remove the failure rather than live with it.
- **A — land Card 1 as built; photo scan is best-effort.** Codes go out sooner, but roughly one
  customer in seven has a code the photo scan will never read (on a tablet without a built-in
  detector), and the crew's only fallback is the live camera.
- **B — put less in the code.** Does not fix it: even the bare code fails about 1 in 20, and the
  tablet loses the offline link between an offer and its campaign.
- **D — have the server check each code is readable before issuing it.** Would need a second
  reader on the server that fails on exactly the same codes as the tablet's; none exists and none
  was measured. The largest build of the four.

**Before any of them, one attended check is worth more than another night of measurement:** open
the scanner on the truck's actual tablet and photo-scan a handful of the failing sample codes. If
the tablet reads them, the problem is confined to devices without a built-in detector and the
urgency drops.

### What is preserved

- Card 1's whole diff (25 files, +2837/−45): commit `fe45768`, ref
  `refs/night-crew/preserve/20261006/identity-code-and-qr/attempt-1/try-1/worktree`, and the same
  as a patch at `.night-crew/runs/20261006/identity-code-and-qr/changes.patch`. It is on **no
  branch** and is not merged anywhere.
- Evidence: the session's logs inside that commit under
  `.night-crew/runs/2026-10-06-autonomous/logs/identity-code-and-qr/` (`density-probe.log` is the
  measurement); the re-check's scripts and raw output at
  `.night-crew/runs/2026-10-06-autonomous/logs/verify-identity-read-rate/`.
- Both full result reports: `.night-crew/runs/20261006/summary.json`.

## Decision 2 — follows from Decision 1, no separate answer needed tonight

Card 2's session asked whether to re-dispatch it unchanged on top of whatever form of Card 1 lands
(its recommendation), or build the send now against Card 1 as it stands. Under every option above
except A the image may change, so **re-dispatch unchanged after Card 1 lands** is the default this
closeout assumes.

## Engineer-level calls made tonight, listed so they can be overturned at triage

Made by the orchestrator:
- **Ran `20261006`, not the newest saved prompt `20261007`** — that prompt says it launches only
  after `20261006` has run and been triaged.
- **The two long-stranded card branches** (`card/a3-rls-fixture-own`, `card/s2-demo-sync-target`)
  were not re-asked at launch — already ruled "leave in place" (B-442, re-confirmed at triage
  2026-10-05).
- **The loop's own gate was compile + vet only.** The slate's gates are prose and the loop runs
  only fenced commands; test clauses were left to the session's recorded evidence, the review and
  the orchestrator's suites. A slate written for the loop should carry runnable gate lines.
- **Sessions were told not to commit, not to regenerate `sw.js`, and not to run the full browser
  suite** — the loop commits a card as one commit, and `sw.js` is built from committed state.
- **Pinned run width 1 and an 8 h deadline** (the launch prompt named neither; the loop's default
  deadline is 2 h).
- **An independent re-check of Card 1's measurement** was dispatched after the park (analysis only,
  scratch tree) so this document carries a second opinion.

Made by Card 1's session (in its merge-intent note inside `fe45768`), all un-landed: a long offer
line is shortened inside the code so the symbol stays readable-sized; the image URL is a relative
path; the code's expiry is the UTC date; a second warning `identity_not_minted`; projections stop
after the first failure in one import; `playwright.config.js` gains the fixture path variable.
Known gaps it named: a code minted but not projected is never re-projected, and customers who
signed up before the card get no code.
