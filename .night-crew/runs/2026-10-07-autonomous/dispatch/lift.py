#!/usr/bin/env python3
"""Lift slate-20261007's four cards into the card-list JSON `night-crew dispatch` consumes.
Intent / Spec / Gates / Context are taken VERBATIM from the slate for Cards 1-3; the run's
standing rules are appended to Context and the Gates text is copied into Context (B-492: the
build session sees only Intent, Spec and Context). Card 4 is NARROWED by the operator's
decision 214 (triage receipt 20261006, ledger T-71): only the throwing-policy refusal gate
(B-482) is built; the photo-pick feedback half (B-483, [PS-03], #scan-note) is withdrawn."""
import json, re, pathlib
root = pathlib.Path(__file__).resolve().parents[4]
slate = (root / ".night-crew/knowledge/reference/slate-20261007.md").read_text()
here = pathlib.Path(__file__).parent
rules = (here / "context-rules.md").read_text()

LOOP_GATES = """The loop itself executes only the fenced lines below (compile and vet). Every test clause above is proven by the session's recorded red/green evidence, re-run by the orchestrator's separate adversarial review, and by the orchestrator's suites on the merged tree under the lock.

```
go build -C backend ./...
go vet -C backend ./...
```"""

K4_INTENT = ("The refusal the phone shows when its campaign policy cannot be read is proven through the "
             "page, not only through the predicate. (Narrowed by operator decision 214: the photo-pick "
             "feedback line is NOT built — \"Scan from photo\" is being removed, decision 213.)")
K4_SPEC = ("(B-482 ONLY — operator decision 214, triage receipt `reference/triage-20261006.md`; the B-483 "
           "half of the signed card is WITHDRAWN: do NOT add `SCAN_STATE.note`, `#scan-note`, the "
           "\"Finish checking this code first\" copy or `[PS-03]`, and do not touch `onFilePicked`.) "
           "In `marketing/scan-page.js`, where scan-page creates the policy source "
           "(`createCampaignPolicySource(cols.campaigns)` at `:312`), read `window.__MARKETING_POLICY_SOURCE__` "
           "first: if present and it has a `policyFor` function, use it (and its `unresolved()` if present); "
           "document in the file header that it is a TEST-ONLY boot-time override set by "
           "`page.addInitScript`, never by product code. `submit-flow.js` / `submit-machine.js` untouched "
           "(a diff there is scope drift, stated). No new machine state, event or (state, event) pair — "
           "strictness 460 declared pairs; adding one is a park. `tests/marketing.spec.js`: `[SV-12]` "
           "`addInitScript` installs a throwing `policyFor`, seed a low-value code locally, go offline, scan it "
           "→ the fail-closed refusal copy renders and no `#ms-order`. `[SV-12]` must drive the scan through "
           "a path that does not depend on the \"Scan from photo\" control surviving (it is being removed) "
           "wherever the existing `[SV-*]` specs allow; if every available scan path in the spec file is the "
           "photo control, use it and say so in the report. If the refusal gate cannot be built without the "
           "withdrawn half, PARK and say so.")
K4_GATES = ("`[SV-12]` RED on the pre-change tree (the override is not read — the offer renders, the spike's "
            "leg b); then green; `[PS-01]`, `[PS-02]`, `[SV-01]`–`[SV-11]`, the Scanner polish and Camera "
            "scanner describes untouched and green — the whole of `tests/marketing.spec.js` `--retries=0`; "
            "no `#scan-note` and no `note` field anywhere in the diff; `sw.js` 51 (orchestrator); G1; the "
            "full Playwright suite on the complete tree = the FINAL suite under the lock (orchestrator). The "
            "review runs `[SV-12]` itself with the override removed and greps the diff for any change under "
            "`marketing/submit-*.js`.")

K3_EXTRA = ("TRIAGE LAUNCH NOTE (B-493, triage receipt 20261006 — this card covers BOTH directories): a second "
            "states spec, `tests/states-marketing-subscribers.spec.js` (its directory constant near line 51), "
            "rewrites eight committed PNGs under `.night-crew/runs/2026-10-02-autonomous/logs/h5/states/` the "
            "same way. Apply the SAME change to it: default to the ignored `test-screenshots/` (its own "
            "subdirectory), honour `STATES_SHOT_DIR`, same header-comment treatment, the same kind of static "
            "untracked-path assertion, and prove `git diff --stat -- .night-crew/runs/2026-10-02-autonomous/logs/h5/states/` "
            "is empty after running it. The committed h5 set, like h4, is NOT rewritten, moved or re-captured. "
            "Also grep `tests/states-*.spec.js` for any other spec writing under a tracked directory and "
            "report what you find (fix it the same way only if it is the same one-constant shape; otherwise "
            "report it and leave it).")

def card(n, slug, title, db, port, override=None, extra=""):
    start = slate.index(f"### Card {n} · `{slug}`")
    nxt = re.search(r"\n(### Card |## )", slate[start + 10:])
    body = slate[start: start + 10 + nxt.start()]
    def sec(name, until):
        m = re.search(rf"\*\*{name}:\*\*(.*?)(?=\n\n\*\*{until})", body, re.S)
        assert m, (slug, name)
        return m.group(1).strip()
    lead = body.split("\n\n")[1].strip()
    cls = re.search(r"\*\*complexity:\*\* ([a-z-]+)", body).group(1)
    park = re.search(r"\*\*PARK note \(narrow, operator-only\):\*\*(.*)", body, re.S).group(1).strip()
    intent, spec, gates = sec("Intent", "Spec"), sec("Spec", "Gates"), sec("Gates", "Context")
    if override:
        intent, spec, gates = override
    ctx = (sec("Context", "PARK note") + ("\n\n" + extra if extra else "")
           + "\n\n" + "DONE-WHEN (the card's Gates section — these are the clauses you must prove and report on; references to G6, the full Playwright suite, the 10×/5× measurement leg and `sw.js` are the orchestrator's legs): " + gates
           + "\n\n" + "Slate lead (mechanism-of-record pointers): " + lead
           + "\n\n" + "PARK note (narrow, operator-only — if you meet one of these, STOP and report it as a park; never decide it): " + park
           + "\n\n" + rules.replace("<card-slug>", slug).replace("<card>", db).replace("<port>", str(port)))
    return {"slug": slug, "title": title, "team": "engineering", "complexity": cls,
            "intent": intent, "spec": spec, "gates": gates + "\n\n" + LOOP_GATES, "context": ctx}

cards = [
    card(1, "inventory-setup-races", "what a manager types or adds in Setup stays on screen", "k1", 8281),
    card(2, "dish-merge-shapes-and-backstop-tests", "wrong merge shapes fixed, the lock and the deploy backstop pinned", "k2", 8282),
    card(3, "states-screenshots-out-of-tree", "the states specs stop dirtying the tree", "k3", 8283, extra=K3_EXTRA),
    card(4, "scanner-refusal-seam-and-pick-feedback", "the throwing-policy refusal gets a page-level gate (photo-pick feedback withdrawn, decision 214)", "k4", 8284,
         override=(K4_INTENT, K4_SPEC, K4_GATES)),
]
(here / "cards.json").write_text(json.dumps(cards, indent=2, ensure_ascii=False) + "\n")
for c in cards:
    print(c["slug"], c["complexity"], {k: len(v) for k, v in c.items() if k in ("intent","spec","gates","context")})
