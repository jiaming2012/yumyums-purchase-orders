#!/usr/bin/env python3
"""Lift slate-20261006's two cards into the card-list JSON `night-crew dispatch` consumes.
Intent / Spec / Gates / Context are taken VERBATIM from the slate; the run's standing
rules (launch-20261006.md) are appended to Context, as the launch prompt requires."""
import json, re, pathlib
root = pathlib.Path(__file__).resolve().parents[4]
slate = (root / ".night-crew/knowledge/reference/slate-20261006.md").read_text()
here = pathlib.Path(__file__).parent
rules = (here / "context-rules.md").read_text()

LOOP_GATES = """The loop itself executes only the fenced lines below (compile and vet, at baseline, at VERIFY and at the run-branch tip after merge). Every test clause above is proven by the session's recorded red/green evidence, re-run by the orchestrator's G6 review, and by the orchestrator's full suites on the merged tree under the lock.

```
go build -C backend ./...
go vet -C backend ./...
```"""

def card(n, slug, title, db):
    start = slate.index(f"### Card {n} · `{slug}`")
    nxt = re.search(r"\n(### Card |## )", slate[start + 10:])
    body = slate[start: start + 10 + nxt.start()]
    def sec(name, until):
        m = re.search(rf"\*\*{name}:\*\*(.*?)(?=\n\n\*\*{until})", body, re.S)
        assert m, (slug, name)
        return m.group(1).strip()
    lead = body.split("\n\n")[1].strip()
    park = re.search(r"\*\*PARK note \(narrow, operator-only\):\*\*(.*)", body, re.S).group(1).strip()
    gates_prose = sec("Gates", "Context")
    ctx = (sec("Context", "PARK note") + "\n\n" + "DONE-WHEN (the slate's Gates section, verbatim — these are the clauses you must prove and report on; references to G6 and the full Playwright suite are the orchestrator's legs): " + gates_prose + "\n\n" + "Slate lead (mechanism of record pointers): " + lead
           + "\n\n" + "PARK note (narrow, operator-only — if you meet one of these, STOP and report it as a park; never decide it): " + park
           + "\n\n" + rules.replace("<card>", db).replace("<card-slug>", slug))
    return {"slug": slug, "title": title, "team": "engineering", "complexity": "new-mechanism",
            "intent": sec("Intent", "Spec"), "spec": sec("Spec", "Gates"),
            "gates": gates_prose + "\n\n" + LOOP_GATES, "context": ctx}

cards = [
    card(1, "identity-code-and-qr", "one permanent identity code per customer, as a QR the shipped scanner reads offline and online", "e1"),
    card(2, "mms-send-on-signup", "the code reaches the customer as an MMS image, lawfully, through a stub tonight", "e2"),
]
(here / "cards.json").write_text(json.dumps(cards, indent=2, ensure_ascii=False) + "\n")
for c in cards:
    print(c["slug"], {k: len(v) for k, v in c.items()})
