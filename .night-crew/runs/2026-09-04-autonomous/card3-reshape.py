#!/usr/bin/env python3
"""card3-reshape.py — Card 3 `backlog-machine-migration` (run 20260904).

Mechanically reshapes .night-crew/knowledge/BACKLOG.md's legacy-shape entries to
the canonical form the night-crew backlog checker reads:

    - [~~]**B-NN · Title**[~~] — note · origin · status · lead: plain line

Every edit is declared in the tables below (reviewable), applied by class, and
guarded twice: a PER-EDIT token-multiset containment assert (every alphanumeric
token of the original text survives in the replacement) and a WHOLE-DOCUMENT
containment assert at the end. Fold-don't-delete throughout: no token occurrence
is ever removed — unrecognized status prose is folded verbatim into the status
payload or the lead, parenthesized (spike-proven mechanism, see
.night-crew/knowledge/spikes/activity-g-planning-surface-honest/backlog-machine-migration.extraction.md).

Defect classes treated (designed by this card):
  1. legacy prose entries (63 + 2 machine-invisible bullets): rebuilt to one
     canonical line each; new handles B-350+ assigned in document order.
  2. "extra segment" — `**destination: …**` in the status slot (44 entries):
     folded into the lead, parenthesized (the spike-proven fold, B-90 sample).
  3. bold-wrapped canonical statuses: de-bold the head so the checker's prefix
     match sees `promoted → `/`done — `/`dropped — `.
  4. non-canonical status heads: prefix the truthful canonical head, carry the
     original status text verbatim as payload (done — resolved …) or fold to
     the lead (new + parenthetical) when the entry stays open.
  5. missing plain-language lead: fold-created from the entry's own displaced
     prose where it exists; a short lead written from the entry's own words
     where nothing is foldable.
  6. missing origin (entries that never recorded one): origin derived from the
     entry's own citations / its section header, labeled as derived.
  7. tail-out-of-fold-reach (B-45): physical lines joined to one logical line.
  8. nested bold in title (B-54): inner `(**31**)` de-bolded to `(31)`.
  9. duplicate handle B-77: the DIAGNOSIS-ADDED addendum folded verbatim into
     the original B-77 entry's lead; its own bullet removed (content moved,
     never deleted).

Usage: python3 card3-reshape.py <input.md> <output.md>
"""
import re
import sys
from collections import Counter

TOK = re.compile(r"[A-Za-z0-9]+")


def toks(s):
    return Counter(TOK.findall(s))


def assert_contained(before, after, label):
    missing = toks(before) - toks(after)
    if missing:
        raise AssertionError(f"{label}: tokens lost: {dict(list(missing.items())[:10])}")


def debold_head(field):
    """Strip the FIRST bold pair (the wrapper around a status head)."""
    i = field.find("**")
    if i < 0:
        return field
    j = field.find("**", i + 2)
    if j < 0:
        return field
    out = field[:i] + field[i + 2:j] + field[j + 2:]
    assert_contained(field, out, "debold")
    return out


# ---------------------------------------------------------------------------
# Legacy blocks: keyed by the ORIGINAL bullet line number (1-based).
# rule: KEEP  — last field is already a canonical status; write the lead.
#       NEWF  — status := new; fold ALL trailing fields verbatim into the
#               lead's leading parenthetical.
#       LAND  — status := landed → `card` (<original status text>).
#       DROPC — status 'dropped, X' → 'dropped — X'.
#       DEBOLD— strip the bold wrapper off an already-canonical status head.
#       EXPL  — per-entry explicit assembly (documented inline).
# ---------------------------------------------------------------------------
LEGACY = {
    16: dict(h="B-350", rule="NEWF", title_override="🔺 [PRIORITY] Resume GSD Phase 23 — wage-history / employee-fields / scheduling foundation (preserved worktree)",
             lead="operator + planner scope the pickup (rebase vs reimplement); worktree and branch preserved — do NOT discard."),
    31: dict(h="B-351", rule="NEWF", lead="build the Spaces+ffmpeg fixture to flip FR-16/NFR-4 UNPROVEN→WORKING."),
    40: dict(h="B-352", rule="LAND", card="users-stale-e2e-repair", lead="landed overnight-20260714 (d32830d, G6-PASS); kept as record."),
    46: dict(h="B-353", rule="KEEP", lead="plumb rejection context into the submit validation so a direct-API resubmit cannot bypass the required photo."),
    59: dict(h="B-354", rule="KEEP", lead="trivial removal of the orphaned div; fold into any future Users card."),
    65: dict(h="B-355", rule="NEWF", lead="repoint the /nc-status gather commands at the real .night-crew/ layout so every machine reports the same state."),
    90: dict(h="B-356", rule="KEEP", lead="preserve field IDs on template edit and make dead-id saves loud — the P0 template-edit data-loss fix, stage 1 as permanent architecture."),
    111: dict(h="B-357", rule="KEEP", lead="handle SAVE_TEMPLATE ops in applyOp so open devices re-render live on template edits (stage 2)."),
    125: dict(h="B-358", rule="EXPL", lead="stage-3 versioning stays parked unless a fleet-style crew ever materializes."),
    143: dict(h="B-359", rule="KEEP", lead="check prod for the ghost item and rename-keep-links per the operator choice; prod data mutation needs operator sign-off."),
    161: dict(h="B-360", rule="LAND", card="ops-fr4-no-enforcement", lead="landed overnight-20260714 (2287947, G6-PASS, red-first); kept as record."),
    167: dict(h="B-361", rule="LAND", card="ops-nfr3-photo-required", lead="field-level gate landed overnight-20260714 (ad105f7); the resubmit residual is tracked by the backend resubmit require_photo gate entry."),
    174: dict(h="B-362", rule="LAND", card="onboarding-nfr5-video-reopen", lead="landed overnight-20260714 (5d73b96, G6-PASS, red-first); kept as record."),
    181: dict(h="B-363", rule="LAND", card="purchasing-fr18-history", lead="landed overnight-20260714 (4cb57b7, G6-PASS, red-first); kept as record."),
    190: dict(h="B-364", rule="KEEP", lead="add builder-UI Add-Field coverage for all six field types including photo."),
    196: dict(h="B-365", rule="KEEP", lead="replace the assertion-free reject test with a real FR-12 assertion."),
    201: dict(h="B-366", rule="KEEP", lead="replace the 6 conditional-skip guards with self-seeded fixtures so a shape mismatch reddens."),
    209: dict(h="B-367", rule="LAND", card="purchasing-fr7-retest", lead="landed overnight-20260714 (958a176, G6-PASS); kept as record."),
    217: dict(h="B-368", rule="LAND", card="inventory-nfr1-normalize-fix", lead="landed overnight-20260715 (748463c, G6-PASS, red-first); kept as record."),
    226: dict(h="B-369", rule="KEEP", lead="convert the ~40 data-dependent guards to unguarded or self-seeded so a seed miss reddens."),
    241: dict(h="B-370", rule="KEEP", lead="folded into PRD-data-integrity FR-5 transactional op emission; rides editprop-broadcast-rerender; kept as record."),
    249: dict(h="B-371", rule="NEWF", lead="clear the partial saved value when a photo upload fails — stale-state hygiene, small frontend fix plus persistence test."),
    256: dict(h="B-372", rule="NEWF", lead="untestable until the offline-IndexedDB harness exists; rides WO-offline-indexeddb-harness."),
    262: dict(h="B-373", rule="NEWF", lead="same harness dependency; rides WO-offline-indexeddb-harness."),
    267: dict(h="B-374", rule="KEEP", lead="upgraded to an FR-7 convergence-matrix cell at the grill-back; kept as record."),
    281: dict(h="B-375", rule="KEEP", lead="add a now-time seam to the four cron check funcs, then real cron-decision unit tests; unblocks P-6."),
    289: dict(h="B-376", rule="NEWF", lead="one harness unblocks the P-1/P-2/P-3 photo flows."),
    296: dict(h="B-377", rule="NEWF", lead="unblocks the P-4/P-5 offline persistence proofs."),
    311: dict(h="B-378", rule="NEWF", lead="convert CreateTemplate and ArchiveTemplate to EmitOpTx for INV-1 parity."),
    317: dict(h="B-379", rule="NEWF", lead="extend applyOp SET_FIELD to unpack _fail_note on the incoming-op path, then land the two parked W-6b cells."),
    329: dict(h="B-380", rule="NEWF", lead="thread a tx through approveSubmission so status and feedback commit atomically."),
    336: dict(h="B-381", rule="NEWF", lead="switch the two fixed flush waits to waitForResponse; low priority."),
    345: dict(h="B-382", rule="EXPL", lead="fix the one cross-test DB-pollution red so literal task test exits 0, formally retiring waiver #1."),
    361: dict(h="B-383", rule="KEEP", lead="make per-card wall-clock timing a standing build-run output so the next gate computes a real Delivery median."),
    367: dict(h="B-384", rule="EXPL", lead="bake CI=1 and explicit pre-migration into gate run-mechanics; doc/skill fix, not code."),
    376: dict(h="B-385", rule="KEEP", lead="cross-user access matrix plus sync-package unit coverage for ResolveEntityAccess across role and assignment combos."),
    388: dict(h="B-386", rule="KEEP", lead="the convergence matrix must vary op type across submit, approve and reject, and assert live convergence of derived views."),
    406: dict(h="B-387", rule="EXPL", lead="both halves fixed red-first the same day (sub-step banners plus the correction-photo slot); presign+PUT camera plumbing stays parked-by-convention; kept as record."),
    433: dict(h="B-388", rule="DEBOLD", lead="server half done 2026-07-25; the client half is its own card split out at T-22 decision 49."),
    442: dict(h="B-389", rule="DROPC", lead="superseded by the RxDB/Supabase migration; kept as record."),
    448: dict(h="B-390", rule="EXPL", lead="the flake is real (~16-20% under contention); fix is test-side re-render safety, not timeouts; scope includes :525."),
    479: dict(h="B-391", rule="DROPC", lead="superseded by the RxDB/Supabase migration; architectural argument at designs/fetchstorm-replay-class-superseded.md; kept as record."),
    491: dict(h="B-392", rule="KEEP", lead="needs re-scoping as its own card with workflows.html in footprint."),
    497: dict(h="B-393", rule="KEEP", lead="one-char fix — drop the trailing slash so a node_modules symlink cannot enter the index."),
    502: dict(h="B-394", rule="KEEP", lead="kill the listener PID rather than the go run parent; assign distinct TEST_PORTs up front."),
    506: dict(h="B-395", rule="KEEP", lead="write -p 1 into standing run mechanics; default parallelism reddens four packages on the shared DB."),
    511: dict(h="B-396", rule="KEEP", lead="cents-as-int or a decimal string is the repo-wide correct fix."),
    515: dict(h="B-397", rule="KEEP", lead="consider a mechanical guard refusing stash when the git common dir differs from the git dir."),
    520: dict(h="B-398", rule="KEEP", lead="repoint the standing-rules inherit at the real files; the named one does not exist."),
    526: dict(h="B-399", rule="KEEP", lead="belongs in a PM session — long-term average with direction of travel, and discount-aware margin needs Toast fields first."),
    538: dict(h="B-400", rule="NEWF", lead="a genuinely separate cluster, or at minimum separate roles with REVOKE — not more search_path discipline."),
    551: dict(h="B-401", rule="NEWF", lead="invert the default so alerts require ALERTS_ENABLED=1 and a forgotten flag fails safe; live prod creds must leave the universal dotenv."),
    562: dict(h="B-402", rule="KEEP", lead="delete the vestigial target or point it at a genuinely separate port."),
    566: dict(h="B-403", rule="KEEP", lead="a general reaper and port-ownership convention is the durable fix."),
    570: dict(h="B-404", rule="KEEP", lead="confirm ownership before anything touches the shared stash slot."),
    574: dict(h="B-405", rule="KEEP", lead="deduplicate the DB_PORT default and audit for other duplicated defaults."),
    579: dict(h="B-406", rule="EXPL", lead="shipped; line anchors are dead — locate the four tests by title; kept as record."),
    584: dict(h="B-407", rule="EXPL", lead="promoted; the confidentiality answer and true gap size are recorded; kept as record."),
    586: dict(h="B-408", rule="KEEP", lead="bind photo keys to their owning app or record so per-app grants actually gate reads; needs a small design pass."),
    593: dict(h="B-409", rule="DROPC", lead="superseded — sync.js is retired by sync-hard-cutover, not patched; argument at designs/fetchstorm-catchup-gate-superseded.md; kept as record."),
    606: dict(h="B-410", rule="KEEP", lead="either reset state in beforeAll or document clean-DB as a hard precondition."),
    611: dict(h="B-411", rule="KEEP", lead="red-first Jim B regression test, then drop the umbrella arg and the inventory disjunct; ships as a patch release, not urgent."),
    623: dict(h="B-412", rule="EXPL", lead="the ruling defines the target state; the red-first cell and fix are still unwritten."),
    650: dict(h="B-413", rule="KEEP", lead="wants an explicit unallocated marker distinct from genuine 0%; investigate once Cost carries real prod rows."),
    656: dict(h="B-414", rule="DEBOLD", lead="the four-card sync foundation program; feasibility confirmed; kept as the umbrella record."),
}

# Canonical-status prefixes for the KEEP sanity check (mirrors backlog.go).
STATUS_PREFIXES = ("promoted → ", "dropped — ", "landed → ", "done — ")


def known_status(s):
    if s == "new":
        return True
    return any(s.startswith(p) and s[len(p):].strip() for p in STATUS_PREFIXES)


def rebuild_legacy(joined, cfg):
    """Rebuild one legacy block (already joined to a single string) to the
    canonical one-line form. Returns the new line. Asserts token containment."""
    m = re.match(r"^- (?:🔺 )?(~~)?\*\*(.*?)\*\*(~~)?\s*(.*)$", joined, re.S)
    if not m:
        raise AssertionError(f"legacy block did not match: {joined[:80]}")
    struck = bool(m.group(1))
    title = cfg.get("title_override") or m.group(2)
    if "**" in title:
        raise AssertionError(f"title carries **: {title[:60]}")
    rest = m.group(4)
    rest = re.sub(r"^[·—]\s*", "", rest)  # drop the legacy separator marker

    sep = " · origin: "
    if sep in rest:
        cut = rest.rfind(sep)
        note = rest[:cut].strip()
        suffix = "origin: " + rest[cut + len(sep):].strip()
        fields = suffix.split(" · ")
        origin, tail = fields[0], fields[1:]
    else:
        note, origin, tail = rest.strip(), None, []

    # A strike that spanned title+description leaves one unmatched `~~` in the
    # note; the canonical strike wraps the title only, so drop that lone marker
    # (punctuation only — the token assert guards the words).
    if struck and note.count("~~") % 2 == 1:
        note = note.replace("~~", "", 1).strip()

    rule = cfg["rule"]
    lead = cfg["lead"]
    if rule == "KEEP":
        assert len(tail) == 1, f"{cfg['h']}: KEEP expects 1 trailing field, got {len(tail)}: {tail}"
        status = tail[0]
        assert known_status(status), f"{cfg['h']}: KEEP status not canonical: {status[:80]}"
    elif rule == "NEWF":
        assert tail, f"{cfg['h']}: NEWF expects trailing field(s)"
        status = "new"
        lead = "(" + " · ".join(tail) + ") " + lead
    elif rule == "LAND":
        assert len(tail) == 1, f"{cfg['h']}: LAND expects 1 trailing field, got {tail}"
        status = f"landed → `{cfg['card']}` ({tail[0]})"
    elif rule == "DROPC":
        assert len(tail) == 1 and tail[0].startswith("dropped, "), f"{cfg['h']}: DROPC shape: {tail}"
        status = "dropped — " + tail[0][len("dropped, "):]
    elif rule == "DEBOLD":
        assert len(tail) == 1, f"{cfg['h']}: DEBOLD expects 1 trailing field, got {len(tail)}"
        status = debold_head(tail[0])
        assert known_status(status), f"{cfg['h']}: DEBOLD status not canonical: {status[:80]}"
    elif rule == "EXPL":
        origin, status, lead, note = explicit(cfg["h"], note, origin, tail, lead)
    else:
        raise AssertionError(f"unknown rule {rule}")

    if origin is None:
        raise AssertionError(f"{cfg['h']}: no origin resolved")
    assert " · " not in origin, f"{cfg['h']}: origin carries the field separator"
    assert " · " not in status or rule == "EXPL", f"{cfg['h']}: status carries the field separator"

    s = "~~" if struck else ""
    line = f"- {s}**{cfg['h']} · {title}**{s} — {note} · {origin} · {status} · lead: {lead}"
    assert_contained(joined, line, cfg["h"])
    return line


def explicit(h, note, origin, tail, lead):
    """Per-entry explicit assemblies (each documented)."""
    if h == "B-358":  # immutable template versions: demoted + deferred → open; fold both fields
        assert len(tail) == 2, tail
        return origin, "new", "(" + " · ".join(tail) + ") " + lead, note
    if h == "B-382":  # waiver #1: merge the operator-choice provenance field into origin
        assert len(tail) == 2, tail
        return origin + " — " + tail[0], tail[1], lead, note
    if h == "B-384":  # 'folded → rides …' → promoted with the fold text as payload
        assert len(tail) == 1, tail
        return origin, "promoted → `percard-timing-instrumentation` (" + tail[0] + ")", lead, note
    if h == "B-387":  # Rejection feedback on SUB-STEPS: 'resolved <date>' → done —
        assert len(tail) == 1, tail
        return origin, "done — " + tail[0], lead, note
    if h == "B-390":  # sync.spec de-flake investigation: extra evidence field → origin; de-bold status
        assert len(tail) == 2, tail
        status = debold_head(tail[1])
        assert known_status(status), status[:80]
        return origin + " — " + tail[0], status, lead, note
    if h == "B-406":  # shipped de-flake record: no origin was ever captured
        origin = ("T-20/T-21 triage era (origin derived from the entry's own citations at the "
                  "2026-09-04 B-NN migration; none was recorded at capture)")
        status = "landed → `syncspec-deflake` (shipped overnight-20260724, S1)"
        return origin, status, lead, note
    if h == "B-407":  # Cost-tab gate: promoted record, no origin ever captured
        origin = ("T-20 decision 36 (origin derived from the entry's own citation at the "
                  "2026-09-04 B-NN migration; none was recorded at capture)")
        # note currently holds '**promoted → `grant-enforcement-parity`** (T-20 decision 36). Answer was …'
        mm = re.match(r"^\*\*promoted → (`[^`]+`)\*\* \((.*?)\)\.\s*(.*)$", note, re.S)
        assert mm, note[:80]
        status = f"promoted → {mm.group(1)} ({mm.group(2)})"
        return origin, status, lead, mm.group(3)
    if h == "B-412":  # cross-user hydration: ruling → promoted to card P4, ruling folded to lead
        assert len(tail) == 1, tail
        status = "promoted → Night B card P4 (`reference/slate-20260802.md`)"
        return origin, status, "(" + tail[0] + ") " + lead, note
    raise AssertionError(f"no explicit rule for {h}")


# ---------------------------------------------------------------------------
# Surgical edits on already-handled B-NN single-line entries.
# ---------------------------------------------------------------------------
DEST_FOLD = [  # class 2: the spike-proven destination fold
    "B-42", "B-46", "B-47", "B-48", "B-49", "B-51", "B-52", "B-53", "B-55",
    "B-56", "B-57", "B-58", "B-59", "B-60", "B-63", "B-64", "B-65", "B-66",
    "B-67", "B-68", "B-69", "B-70", "B-71", "B-72", "B-73", "B-74", "B-75",
    "B-76", "B-77", "B-78", "B-79", "B-80", "B-82", "B-83", "B-84", "B-85",
    "B-86", "B-87", "B-88", "B-89", "B-90", "B-91", "B-92", "B-93",
]
DEBOLD_STATUS = ["B-09", "B-61", "B-62", "B-161"]  # class 3
DONE_PREFIX = ["B-22", "B-26", "B-172"]            # class 4: resolved/RESOLVED → done —
NEW_FOLD_LEAD = ["B-14", "B-43", "B-50"]           # class 4: stays-open prose → new, prose → lead


def destination_fold(line, h):
    """` · **destination: X**TAIL [· lead: Y]` → ` · lead: (destination: X)TAIL [Y]`.

    The LAST destination segment is the status slot — an entry's note may QUOTE
    the house style (B-60 quotes `· new · **destination: <where>** · lead:`
    verbatim), and folding a quotation would rewrite what the entry says."""
    matches = list(re.finditer(r" · \*\*(destination: .*?)\*\*", line))
    assert matches, f"{h}: no destination segment"
    m = matches[-1]
    inner, start, end = m.group(1), m.start(), m.end()
    li = line.find(" · lead: ", end)
    if li >= 0:
        tail = line[end:li]
        existing = line[li + len(" · lead: "):]
        new = line[:start] + " · lead: (" + inner + ")" + tail + " " + existing
    else:
        tail = line[end:]
        new = line[:start] + " · lead: (" + inner + ")" + tail
    assert_contained(line, new, h)
    return new


def surgical(line, h):
    if h in DEST_FOLD:
        line = destination_fold(line, h)
    if h in DEBOLD_STATUS:
        m = re.search(r" · \*\*(promoted → |done — |dropped — )", line)
        assert m, f"{h}: no bold canonical status head"
        i = m.start() + 3  # index of '**'
        j = line.find("**", i + 2)
        assert j > 0, h
        new = line[:i] + line[i + 2:j] + line[j + 2:]
        assert_contained(line, new, h)
        line = new
    if h in DONE_PREFIX:
        m = re.search(r" · (\*\*✅ RESOLVED|resolved 2026)", line)
        assert m, f"{h}: no resolved-status head"
        new = line[:m.start() + 3] + "done — " + line[m.start() + 3:]
        assert_contained(line, new, h)
        line = new
    if h in NEW_FOLD_LEAD:
        # status field = everything between the last ' · ' before the status
        # text and the lead (or EOL). Identify the status field per entry.
        heads = {"B-14": " · **📝 RECORDED", "B-43": " · **partial ruling", "B-50": " · 🛑 **aggravated"}
        i = line.find(heads[h])
        assert i >= 0, f"{h}: status head not found"
        li = line.find(" · lead: ", i)
        if li >= 0:
            statustext = line[i + 3:li]
            existing = line[li + len(" · lead: "):]
            new = line[:i] + " · new · lead: (" + statustext + ") " + existing
        else:
            statustext = line[i + 3:]
            new = line[:i] + " · new · lead: (" + statustext + ")"
        assert_contained(line, new, h)
        line = new
    if h == "B-16":
        old = " · **(b) promoted → `test-harness-fail-loud`**"
        new_seg = " · promoted → (b) `test-harness-fail-loud`"
        assert old in line, "B-16 head not found"
        new = line.replace(old, new_seg, 1)
        assert_contained(line, new, h)
        line = new
    if h == "B-81":
        old = " · **resolved 2026-08-04**"
        assert line.endswith(old), "B-81 tail not as expected"
        new = line[: -len(old)] + " · done — resolved 2026-08-04 · lead: resolved 2026-08-04; kept as record."
        assert_contained(line, new, h)
        line = new
    if h == "B-145":
        m = re.search(r" · \*\*per-class status:\*\* ", line)
        assert m, "B-145 status head not found"
        new = line[:m.start() + 3] + "done — " + line[m.start() + 3:] + (
            " · lead: reconstruction executed and recorded per decisions 154/157/158; "
            "recipes hand-rebuild and crew re-invites remain operator-owned; kept as record."
        )
        assert_contained(line, new, h)
        line = new
    if h == "B-54":
        assert "(**31**)" in line, "B-54 nested bold not found"
        new = line.replace("(**31**)", "(31)", 1)
        # token check: '31' survives
        assert_contained(line, new, h)
        line = destination_fold(new, h)
    return line


def main(inp, outp):
    text = open(inp, encoding="utf-8").read()
    lines = text.split("\n")
    n = len(lines)
    out = []
    i = 0

    # Pre-capture the B-77 duplicate line for the merge, then drop its bullet.
    dup_idx = None
    for k, l in enumerate(lines):
        if l.startswith("- **B-77 · DIAGNOSIS ADDED 2026-08-04"):
            dup_idx = k
            break
    assert dup_idx is not None, "B-77 duplicate not found"
    dup_text = lines[dup_idx][2:]  # strip '- '
    dup_fold = " 🛑 (" + dup_text.replace(" · lead: ", " — lead: ", 1) + ")"

    handle_re = re.compile(r"^- \*\*(B-\d+) · ")

    while i < n:
        l = lines[i]
        lineno = i + 1

        if lineno in LEGACY:
            cfg = LEGACY[lineno]
            j = i
            block = []
            while j < n and lines[j].strip() != "" and (j == i or not lines[j].startswith("- ")):
                block.append(lines[j])
                j += 1
            joined = " ".join(x.strip() for x in block)
            out.append(rebuild_legacy(joined, cfg))
            i = j
            continue

        if lineno == 795:  # B-45: join the whole multi-paragraph entry to one line
            j = i
            block = []
            while j < n and not lines[j + 1 if False else j].startswith("- **B-46"):
                if lines[j].strip():
                    block.append(lines[j].strip())
                j += 1
            joined = " ".join(block)
            assert joined.startswith("- **B-45 ·"), joined[:40]
            assert_contained("\n".join(lines[i:j]), joined, "B-45")
            out.append(joined)
            i = j
            continue

        if i == dup_idx:  # duplicate B-77 bullet: content already folded into the original
            i += 1
            # also swallow ONE following blank line so the doc doesn't gain a double gap
            if i < n and lines[i].strip() == "":
                i += 1
            continue

        m = handle_re.match(l)
        if m:
            h = m.group(1)
            new = l
            if h in DEST_FOLD or h in DEBOLD_STATUS or h in DONE_PREFIX or h in NEW_FOLD_LEAD or h in ("B-16", "B-81", "B-145", "B-54"):
                if h == "B-77":
                    new = surgical(l, h) + dup_fold
                else:
                    new = surgical(l, h)
            out.append(new)
            i += 1
            continue

        out.append(l)
        i += 1

    result = "\n".join(out)
    assert_contained(text, result, "WHOLE DOCUMENT")
    open(outp, "w", encoding="utf-8").write(result)
    print(f"reshaped: {inp} -> {outp}")
    print(f"  lines {len(lines)} -> {len(out)}")
    b, a = toks(text), toks(result)
    print(f"  tokens before: {sum(b.values())} occurrences / {len(b)} distinct")
    print(f"  tokens after : {sum(a.values())} occurrences / {len(a)} distinct")
    print(f"  lost: {sum((b - a).values())} (must be 0)")


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
