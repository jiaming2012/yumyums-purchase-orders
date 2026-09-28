// marketing/boot-failure.js — the ONE way the Scan section reports that it
// could not start.
//
// Before this, both boot() catch blocks (scan-page.js, submit-flow.js) wrote
// `'… failed to start — reload to retry. (' + e.message + ')'` straight into
// #scan-status. An RxDB failure's message is a paragraph of library text
// ("Error-Code: UT8. Hint: Error messages are not included in RxDB core to
// reduce build size … Parameters: args: { "typeof_crypto_subtle": …") — that
// paragraph was the whole screen a crew member saw on a phone.
//
// The contract now: one plain sentence + what to do, the cause collapsed
// behind "Details" for whoever the phone gets handed to, and console.error
// (which log.js ships to /api/v1/logs) for the rest of us. UI-R3/R6 still
// hold — loud, named, retryable — the wording is just for people.

const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({
  '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
}[c]));

const HEADLINE = {
  scanner: "Scanner didn't start.",
  submit: "Submit flow didn't start.",
};

export const SECURE_ADDRESS = 'https://hq.yumyums.kitchen';

/**
 * Render (or re-render) the failure block for one boot stage. Each stage owns
 * its own block, so a second stage failing for its OWN reason reads alongside
 * the first instead of overwriting it; a stage failing BECAUSE an earlier one
 * did should not call this at all (see submit-flow.js's `upstream` flag).
 * @param {'scanner'|'submit'} kind
 * @param {unknown} err
 */
export function reportBootFailure(kind, err) {
  const cause = err && err.message ? err.message : String(err);
  console.error('[marketing] ' + kind + ' boot failed: ' + cause);
  const status = document.getElementById('scan-status');
  if (!status) return;
  let block = status.querySelector('[data-boot-fail="' + kind + '"]');
  if (!block) {
    block = document.createElement('div');
    block.setAttribute('data-boot-fail', kind);
    status.appendChild(block);
  }
  // Plain http (the dev box's LAN IP, say) is the one environment we know
  // boots differently — name the way out rather than leave the crew guessing.
  const insecureHint = window.isSecureContext === false
    ? ' This page was opened over a plain http address; if it keeps happening, open '
      + '<a href="' + SECURE_ADDRESS + '/marketing.html">hq.yumyums.kitchen</a> instead.'
    : '';
  block.innerHTML = '<strong>' + esc(HEADLINE[kind] || 'Something didn\'t start.') + '</strong>'
    + ' Reload to try again.' + insecureHint
    + '<details><summary>Details</summary><code>' + esc(cause) + '</code></details>';
}
