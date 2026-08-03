// scripts/write-version-json.js — the ONE place `version.json` is written.
//
// `version.json` is a git-IGNORED build artifact (.gitignore:13) that
// nonetheless SHIPS: `sw.js` precaches it (`GENERATED_BUT_SHIPPED`, both in
// build-sw.js and tests/sw-manifest.spec.js), and `index.html`'s version line
// reads it as the device-local, staleable source of truth for the frontend
// version. A Workbox precache entry that 404s fails the ENTIRE service-worker
// install, so the file existing is not optional wherever the tree is served.
//
// It has two other generators, and this module is the shared body of one of
// them so the payload shape has a single definition:
//   * build-sw.js — requires writeVersionJson() below (local dev, `task sw`,
//     and `task test` via its `sw` dep).
//   * backend/Dockerfile:57-64 — regenerates it INSIDE the image from the
//     authoritative `version.go` `Frontend` constant. 🛑 Note the two
//     generators read DIFFERENT sources (package.json here, version.go there);
//     that is harmless only while the standing three-way parity holds, and
//     would surface as dev and prod showing different numbers the moment it
//     does not. See B-92.
//
// 🛑 WHY THIS IS A SEPARATE SCRIPT AND NOT `node build-sw.js`.
// `playwright.config.js`'s `webServer.command` needs this file to exist before
// the E2E server serves `STATIC_DIR=../`. It must NOT get there by running
// build-sw.js: that reads **git HEAD**, rewrites `sw.js`, and would dirty the
// tree mid-gate — which is B-37's whole hazard, and would make every gate run
// produce a spurious `sw.js` diff. This script writes the one small file and
// touches nothing else.
//
// Paths resolve against the REPO ROOT, not `process.cwd()`, so the same call
// works from build-sw.js at the root and from a `webServer.command` chain.
const fs = require('fs');
const path = require('path');

const ROOT = path.join(__dirname, '..');

// Semver only — dynamic fields (git_sha, built_at) live in /api/v1/health.
// package.json "version" mirrors the Frontend constant in
// backend/internal/version/version.go.
function writeVersionJson() {
  const pkg = JSON.parse(fs.readFileSync(path.join(ROOT, 'package.json'), 'utf8'));
  const payload = { frontend: pkg.version };
  fs.writeFileSync(path.join(ROOT, 'version.json'), JSON.stringify(payload) + '\n');
  return payload;
}

module.exports = { writeVersionJson };

if (require.main === module) {
  try {
    const payload = writeVersionJson();
    // 🛑 console.ERROR, not console.log — same reason as scripts/reset-e2e-db.js
    // (B-81). This runs inside `webServer.command`, and Playwright's webServer
    // plugin pipes the child's stdout only when `webServer.stdout === 'pipe'`
    // (it defaults to 'ignore'); stderr IS piped by default. A generator whose
    // output is swallowed leaves a gate log with no evidence it ran.
    console.error(`── wrote version.json frontend=${payload.frontend} ──`);
  } catch (err) {
    console.error(`\n[write-version-json] ${err.message}\n`);
    process.exit(1);
  }
}
