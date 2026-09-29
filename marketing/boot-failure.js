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
 * True for an address that is a development host rather than the deployment:
 * localhost, a private LAN IP (192.168/x, 10.x, 172.16–31.x), or a tailnet
 * address (100.64–127.x, the CGNAT range Tailscale uses). Someone here typed
 * this host deliberately; the fix for them is https on it, not a different one.
 */
export function isLocalOrigin(hostname) {
  const h = String(hostname || '');
  if (h === 'localhost' || h === '127.0.0.1' || h === '::1' || h.endsWith('.local')) return true;
  const m = /^(\d{1,3})\.(\d{1,3})\./.exec(h);
  if (!m) return false;
  const a = Number(m[1]), b = Number(m[2]);
  return a === 10
    || (a === 192 && b === 168)
    || (a === 172 && b >= 16 && b <= 31)
    || (a === 100 && b >= 64 && b <= 127);
}

/**
 * Render (or re-render) the failure block for one boot stage. Each stage owns
 * its own block, so a second stage failing for its OWN reason reads alongside
 * the first instead of overwriting it; a stage failing BECAUSE an earlier one
 * did should not call this at all (see submit-flow.js's `upstream` flag).
 * @param {'scanner'|'submit'} kind
 * @param {unknown} err
 */
/**
 * Known causes, said in words. The raw text still ships to the console and to
 * /api/v1/logs; this is only what a crew member reads when they open Details.
 * An unrecognised cause falls through verbatim — a wrong plain sentence is
 * worse than a technical true one.
 */
function explainCause(cause) {
  const c = String(cause);
  if (/SubtleCrypto|crypto\.subtle/i.test(c)) {
    return 'This browser only offers the scanner\u2019s security features over https.';
  }
  if (/indexedDB|IDBFactory|Dexie/i.test(c)) {
    return 'This browser would not open local storage for the offline code list. '
      + 'Private browsing usually causes that.';
  }
  return c;
}

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
  //
  // Which way out depends on WHERE you are. Telling someone on 192.168.8.176
  // to "open hq.yumyums.kitchen" reads as "abandon the address you deliberately
  // typed" — they are on the dev box on purpose, and the answer there is https
  // on that same host, not a different deployment. Only a browser that is
  // neither on the LAN nor on localhost is actually being pointed home.
  const insecureHint = window.isSecureContext === false
    ? ' The scanner needs a secure (https) connection and this page was opened over http.'
      + (isLocalOrigin(window.location && window.location.hostname)
        ? ' Reopen this same address over https.'
        : ' Open <a href="' + SECURE_ADDRESS + '/marketing.html">hq.yumyums.kitchen</a> instead.')
    : '';
  block.innerHTML = '<strong>' + esc(HEADLINE[kind] || 'Something didn\'t start.') + '</strong>'
    + ' Reload to try again.' + insecureHint
    + '<details><summary>Details</summary><code>' + esc(explainCause(cause)) + '</code></details>';
}
