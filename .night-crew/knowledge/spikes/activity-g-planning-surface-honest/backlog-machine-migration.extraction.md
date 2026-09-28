# Extraction — backlog-machine-migration

Outcome: confirmed

Approach used: iterate on a scratch copy via `night-crew backlog check --file
<copy>` (the flag exists for exactly this), reshape mechanically with
fold-don't-delete surgery (the sample's extra `**destination: …**` segment
folded into its lead, parenthesized — no words removed), and prove
preservation with token-multiset containment (every alphanumeric token of the
original entry must survive). Candidate loop for the card, not an adoption
(NFR-6).

Confirmed: the red is real and countable — `backlog check` exits 1 with 297
issues across 207 entries, `backlog list` exits 0 with a countable set; the
instrument demonstrably reads the copy (appending a probe entry moved the
copy's entry count 207→208 while the real document's sha256 never moved); the
sample legacy entry (B-90) greens under the real checker (297→296) with all
401 of its tokens preserved; and both scripts left the real document
byte-identical.

Learned: two measurement facts the card should bake in — (1) `backlog list`
emits 208 lines against the checker's 207 entries, so the done_when's
"document entry count" must anchor on the checker's own "across M entries"
parse, not a hand grep or the list line count; (2) the document's multi-KB
single-line entries SIGPIPE any early-exit pipe consumer (`head` after a pipe)
under `pipefail` — the card's own tooling must use full-readers.

Plan change: the card's done_when is re-anchored: `check` exit 0 AND
`backlog list` count == the checker's entry count (its own parse), with the
token-multiset containment proof run over the whole document as the
content-preservation evidence in the merge-intent. The fold-don't-delete
reshape is offered as the candidate mechanism for the "extra segment" defect
class (the largest class observed); other defect classes ("missing origin",
"missing plain-language lead") still need per-class treatment the card
designs.
