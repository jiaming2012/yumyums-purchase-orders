// Repo-hygiene guards — Wave 0 of run `overnight-20260806`,
// card `repo-hygiene-preconditions`.
//
// Three facts that are each one line of source and each corrupt EVIDENCE rather
// than behaviour. Nothing here touches the app; every assertion is against files
// on disk. That is the point: these are the properties the rest of the milestone's
// `done_when:` rows are allowed to depend on.
//
// 🛑 Every read here goes through `fs.readFileSync(..., 'utf8')`, never through
// `grep`. B-70 is precisely a case where `grep` reported nothing on a file that
// had matches, so a guard against B-70 that used `grep` could not fail.
const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const REPO = path.join(__dirname, '..');

function read(rel) {
  return fs.readFileSync(path.join(REPO, rel), 'utf8');
}

// ---------------------------------------------------------------------------
// (a) B-70 — a raw NUL byte puts every `grep` on the file into binary mode.
// ---------------------------------------------------------------------------
//
// `sync-rxdb/client.js` carried exactly one `U+0000` at byte offset 50850, as a
// deliberate delimiter inside the `scopeFingerprint` template literal. The
// technique is sound; the side effect was not. `file(1)` reported the file as
// `data` and GNU grep switched to binary mode, so
//
//     grep -n 'export' sync-rxdb/client.js
//
// printed NOTHING and exited 1 — on a file containing 29 occurrences of `export`.
// A gate grep that returns nothing because of a NUL byte is indistinguishable
// from a gate grep that returns nothing because the work is done, which makes
// every `done_when: "grep returns nothing"` row unreliable IN THE PASSING
// DIRECTION. The roadmap banned that criterion shape until this landed.
//
// The fix is the escape sequence `\0` in source: the same byte at runtime, so no
// fingerprint changes, but the file stays 7-bit clean.
test('no source file under sync-rxdb/ contains a NUL byte', () => {
  const dir = path.join(REPO, 'sync-rxdb');
  const files = fs
    .readdirSync(dir)
    .filter((f) => f.endsWith('.js'))
    .sort();

  expect(files.length).toBeGreaterThan(0);

  const offenders = [];
  for (const f of files) {
    const buf = fs.readFileSync(path.join(dir, f));
    const offsets = [];
    for (let i = 0; i < buf.length; i++) {
      if (buf[i] === 0) offsets.push(i);
    }
    if (offsets.length) offenders.push(`sync-rxdb/${f}: ${offsets.length} NUL at ${offsets.join(', ')}`);
  }

  expect(offenders, 'a NUL byte makes grep report nothing on this file — B-70').toEqual([]);
});

// The delimiter must still BE a NUL at runtime. Replacing it with any printable
// character would change every fingerprint the function has ever produced and
// break the "cannot occur in either operand" property the technique rests on.
test('scopeFingerprint still joins its operands with a real NUL at runtime', () => {
  const src = read('sync-rxdb/client.js');
  const m = src.match(/fingerprint: scopeFingerprint\(`\$\{scopeIdentity\(s\)\}(.*?)\$\{serialized\}`\)/);
  expect(m, 'the scopeFingerprint call site moved — re-verify the delimiter').not.toBeNull();
  // The source spelling must be the two ASCII characters backslash + zero, which
  // is the JS escape for U+0000 inside a template literal: the same byte at
  // runtime as the raw NUL it replaced, so no fingerprint changes value.
  expect(m[1], 'the delimiter must stay a NUL escape, not a printable character').toBe('\\0');
  // Belt and braces: that two-character escape really denotes one zero byte.
  expect(m[1].length).toBe(2);
  expect(String.fromCharCode(0).charCodeAt(0)).toBe(0);
});

// ---------------------------------------------------------------------------
// (c) The stale activation gate — now FACT-shaped, not spelling-shaped.
// ---------------------------------------------------------------------------
//
// `sync-rxdb-row-visibility-rls` merged in run `overnight-20260801` (`bbbfc64`,
// merged to the run branch at `bec06f6`; the roadmap flipped it DONE at
// `914536c`). Comment blocks across the tree gated activation on it "landing",
// which reads to the next author as an open precondition when it is a closed
// one, and hides the preconditions that ARE still open (the cutover).
//
// The ORIGINAL version of this test (card `repo-hygiene-preconditions`, run
// 20260806) scanned ONLY `sync-rxdb/*.js` for the literal string
// `sync-rxdb-row-visibility-rls`. That is spelling-shaped: blind to every
// paraphrase and to the whole tree outside that one directory. B-140 proved the
// hole — four live gates stood, three of them in files this test never opened,
// and it passed; those gates are now corrected. (This block deliberately avoids
// writing an example paraphrase gate: doing so would trip the very scan below.)
//
// Broadened (card `sync-doc-honesty`, run 20260901) to assert on the CLAIM, not
// the spelling: a live comment that names ANY DONE roadmap card — or the
// "row-visibility RLS" milestone — as a FUTURE/unlanded precondition. The
// roadmap files already carry the card statuses; this test reads them.

// The narrow guarantee the original test made is KEPT as a subset assertion, so
// broadening never loses the specific property `w0-repo-hygiene` established.
test('no sync-rxdb source spells the merged row-visibility-rls slug (narrow, subset)', () => {
  const dir = path.join(REPO, 'sync-rxdb');
  const hits = [];
  for (const f of fs.readdirSync(dir).filter((x) => x.endsWith('.js')).sort()) {
    read(`sync-rxdb/${f}`)
      .split('\n')
      .forEach((line, i) => {
        // A LIVE gate spells the slug in future-precondition phrasing. A
        // retrospective quote ("this used to read ... that card MERGED") is
        // honest and must not red — the same resolved-marker rule the
        // whole-tree scan below uses.
        if (line.includes('sync-rxdb-row-visibility-rls') && GATE_PHRASE.test(line)) {
          hits.push(`sync-rxdb/${f}:${i + 1}: ${line.trim()}`);
        }
      });
  }
  expect(hits, 'that card merged 2026-08-01 — a gate on it is stale').toEqual([]);
});

// Future-precondition phrasing: "until|before|once|when ... lands|ships|merges"
// (and inflections). This is the shape of a claim that something has not
// happened yet.
const GATE_PHRASE =
  /\b(?:until|before|once|when)\b[^.\n]{0,140}\b(?:land(?:s|ed|ing)?|ship(?:s|ped|ping)?|merg(?:e|es|ed|ing))\b/i;
// The "row-visibility RLS" milestone by name, in any spelling seen in the tree.
const ROW_VIS_MILESTONE = /row[- ]visibility[- ]rls|row[- ]visibility\s+rls/i;
// A block that ACKNOWLEDGES the card is done (merged/landed/used-to/corrected)
// is an honest retrospective, not a live gate — it must not red.
const RESOLVED_MARKER =
  /\bmerged\b|\blanded\b|\bshipped\b|used to|earlier version|corrected|\balready\b|\bDONE\b|no longer/i;

// Every DONE card slug the roadmaps carry — the CURRENT roadmap plus the
// archived ones (the 08-05 sync-foundation roadmap is where the RLS card holds
// its DONE status; the current roadmap does not list it as a card at all).
function doneCardSlugs() {
  const roadmapFiles = [path.join(REPO, '.night-crew/knowledge/roadmap.md')];
  const refDir = path.join(REPO, '.night-crew/knowledge/reference');
  for (const f of fs.readdirSync(refDir)) {
    if (/^roadmap-.*\.md$/.test(f)) roadmapFiles.push(path.join(refDir, f));
  }
  const slugs = new Set();
  // `- **`slug`** · **DONE...`  — the roadmap card status line format.
  const re = /^\s*-\s*\*\*`([a-z0-9][a-z0-9-]*)`\*\*\s*·\s*\*\*DONE/gm;
  for (const file of roadmapFiles) {
    const txt = fs.readFileSync(file, 'utf8');
    let m;
    while ((m = re.exec(txt)) !== null) slugs.add(m[1]);
  }
  return slugs;
}

// The LIVE source tree — code, assets, tests. Deliberately EXCLUDES `.night-crew/`
// (frozen run artifacts, QA captures, and the knowledge ledger/backlog/roadmap
// records all correctly QUOTE the defect — retiring those would erase the
// history), `node_modules`, and vendored code (`vendor/`, `workbox-*.js`,
// generated `sw.js`).
function liveSourceFiles() {
  const out = [];
  for (const f of fs.readdirSync(REPO)) {
    if (!/\.(?:js|html)$/.test(f)) continue;
    if (f.startsWith('workbox-') || f === 'sw.js') continue;
    out.push(f);
  }
  const dirs = ['backend', 'sync-rxdb', 'sync-schema', 'tests', 'scripts', 'lib'];
  const walk = (rel) => {
    const abs = path.join(REPO, rel);
    if (!fs.existsSync(abs)) return;
    for (const e of fs.readdirSync(abs, { withFileTypes: true })) {
      if (e.isDirectory()) {
        if (e.name === 'node_modules' || e.name === 'vendor') continue;
        walk(path.join(rel, e.name));
      } else if (/\.(?:js|go|html|sql)$/.test(e.name)) {
        out.push(path.join(rel, e.name));
      }
    }
  };
  for (const d of dirs) walk(d);
  return out;
}

test('no live comment gates activation on a DONE roadmap card (fact-shaped, whole tree)', () => {
  const done = doneCardSlugs();
  // Sanity: the roadmaps must actually yield slugs, or this guard is vacuous —
  // the repo's characteristic bug class is a check whose subject set went empty.
  expect(done.size, 'no DONE card slugs parsed from the roadmaps — guard would be vacuous').toBeGreaterThan(5);
  expect(done.has('sync-rxdb-row-visibility-rls'), 'the RLS card must be a known DONE slug').toBe(true);

  const slugs = [...done].sort((a, b) => b.length - a.length);
  const WIN = 4; // lines of context on each side of the gate line
  const hits = [];

  for (const rel of liveSourceFiles()) {
    const lines = read(rel).split('\n');
    for (let i = 0; i < lines.length; i++) {
      if (!GATE_PHRASE.test(lines[i])) continue;
      const block = lines.slice(Math.max(0, i - WIN), i + WIN + 1).join(' ');
      const named = slugs.filter((s) => block.includes(s));
      const milestone = ROW_VIS_MILESTONE.test(block);
      if (!named.length && !milestone) continue; // gate phrasing about something else
      if (RESOLVED_MARKER.test(block)) continue; // honest retrospective, not a live gate
      hits.push(`${rel}:${i + 1}: ${lines[i].trim()} [names: ${named.join(', ') || (milestone ? 'row-visibility-rls milestone' : '')}]`);
    }
  }

  expect(
    hits,
    'a live comment names a DONE roadmap card as a future/unlanded precondition — that gate is stale (B-140)'
  ).toEqual([]);
});

// ---------------------------------------------------------------------------
// (b) night-crew.toml's [e2e.seams] comment must agree with tests/.
// ---------------------------------------------------------------------------
//
// The comment claimed the four Operations tokens "select exactly workflows /
// persistence / sync / repro-cut-task .spec.js and nothing else — re-verified at
// landing". They select NINE files: `sync` alone matches six.
//
// 🛑 This is an OVER-selection, not a coverage hole. Per B-87 the Playwright CLI
// filters are OR'd, so a card confined by this seam OVER-RUNS; it never runs the
// wrong specs. What is wrong is the file that decides gate cost, and therefore
// the cost estimate every slate takes from it.
//
// The comment now carries a machine-readable roll-call line so the claim cannot
// drift again silently: adding a spec whose name contains a seam token reds this
// test rather than quietly inflating a "confined" gate.
const SELECTS_MARKER = '#   selects:';

function seamTokens(toml) {
  // The Operations rows — the ones the corrected comment describes.
  const line = toml.split('\n').find((l) => l.startsWith('"backend/internal/workflow"'));
  expect(line, '[e2e.seams] no longer has a backend/internal/workflow row').toBeTruthy();
  return line
    .slice(line.indexOf('['))
    .replace(/[[\]"]/g, '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);
}

test('night-crew.toml records the spec files its Operations tokens really select', () => {
  const toml = read('night-crew.toml');
  const tokens = seamTokens(toml);
  expect(tokens.sort()).toEqual(['persistence', 'repro-cut-task', 'sync', 'workflows']);

  // What the tokens ACTUALLY select, computed the way night-crew computes it:
  // union the touched seams' tags, join with `|`, hand to Playwright as a
  // positional path regex.
  const re = new RegExp(tokens.join('|'));
  const actual = fs
    .readdirSync(path.join(REPO, 'tests'))
    .filter((f) => f.endsWith('.spec.js') && re.test(f))
    .sort();

  // What the file SAYS it selects.
  const marker = toml.split('\n').find((l) => l.trim().startsWith(SELECTS_MARKER.trim()));
  expect(
    marker,
    `night-crew.toml's [e2e.seams] comment must carry a "${SELECTS_MARKER}" roll-call of the spec files the Operations tokens select`
  ).toBeTruthy();
  const claimed = marker
    .slice(marker.indexOf(':') + 1)
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
    .sort();

  expect(claimed, 'the comment and `ls tests/` disagree — B-87 mis-costing').toEqual(actual);
  // Stated so a future reader meets the real number before the prose.
  //
  // 9 → 10 on card `skeleton-one-row-end-to-end` (run 20260808-2, C2), which
  // added tests/sync-one-row.spec.js. 10 → 11 on card `activate-fill-view-reads`
  // (same run, C3), which added tests/sync-fill-view.spec.js. Both specs drive
  // workflows.html and sync-rxdb/*, so both genuinely belong to the Operations
  // seam; naming either to dodge the `sync` token would have hidden a spec the
  // seam exists to select.
  // Bumping this number is only ever correct alongside the roll-call line in
  // night-crew.toml — the assertion above is what couples them.
  expect(actual.length).toBe(11);
});
