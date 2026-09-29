// health-banner.js — the server's bad news, said out loud, on the page where
// it bites.
//
// WHY THIS EXISTS
// ---------------
// index.html used to carry a single inline storage banner. It had three
// problems, all of which an operator hit on 2026-09-29:
//
//   1. It ended in `catch(e){}`. A server that could not be reached AT ALL
//      produced no banner and no message — the one failure most worth saying.
//   2. It read /api/v1/health through the service worker, whose NetworkFirst
//      rule (build-sw.js) serves `api-cache` after a 10s timeout. On a flaky
//      phone that means a STALE `storage: ok` and no banner, while the same
//      account on a desktop showed one. This module fetches with
//      cache:'no-store' plus a cache-buster so it can never match that rule.
//   3. It ran once, at load. A device that opened outside an outage stayed
//      clean until someone reloaded it.
//
// WHAT COUNTS AS A WARNING
// ------------------------
// Only states a person can act on. "unconfigured" is a deliberate dev/test
// state and stays silent: shouting about it on every load is how a crew learns
// to ignore the banner that matters.

(function (global) {
  'use strict';

  // Each check reads the /api/v1/health body and returns a sentence, or null.
  // `key` is what the diagnostic block prints so an administrator is told WHAT
  // broke rather than "the app is weird".
  var CHECKS = {
    server: {
      key: 'server',
      // h === null means the fetch itself failed — see refresh().
      test: function (h) { return h === null; },
      text: "Can't reach the HQ server — you're seeing saved data, and changes won't be saved.",
      value: function () { return 'unreachable'; },
    },
    storage: {
      key: 'storage',
      test: function (h) { return h && h.storage === 'unreachable'; },
      text: 'Photo storage unreachable — photo & video uploads will fail and saved photos won’t display.',
      value: function (h) { return h.storage; },
    },
    toast: {
      key: 'toast_sync',
      test: function (h) { return h && h.toast_sync && h.toast_sync.status === 'failing'; },
      text: 'Toast sales sync is failing — sales figures and COGS reports are out of date.',
      value: function (h) { return h.toast_sync.status; },
    },
    substrate: {
      key: 'sync_substrate',
      test: function (h) { return h && h.sync_substrate === 'unreachable'; },
      text: 'Scan & redeem is offline — customer codes can’t be verified or redeemed right now.',
      value: function (h) { return h.sync_substrate; },
    },
  };

  function esc(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }

  function render(el, warnings, host) {
    if (!warnings.length) { el.innerHTML = ''; el.hidden = true; return; }
    var now = new Date();
    var stamp = String(now.getHours()).padStart(2, '0') + ':' + String(now.getMinutes()).padStart(2, '0');
    var lines = warnings.map(function (w) {
      return '<div class="warnbanner">⚠️ ' + esc(w.text) + '</div>';
    }).join('');
    // One prompt, however many warnings — repeating "tell an administrator"
    // under each is noise. The diagnostic is copyable so the report that
    // reaches the administrator names the failing check instead of a vibe.
    var diag = warnings.map(function (w) { return w.key + ': ' + w.value; }).join('\n') +
      '\n' + stamp + ' · ' + host;
    lines += '<div class="warnbanner-admin">Something is wrong on the server side — the crew can’t fix this. ' +
      'Let an administrator know:<pre class="warnbanner-diag">' + esc(diag) + '</pre></div>';
    el.innerHTML = lines;
    el.hidden = false;
  }

  async function refresh(el, names) {
    var h = null;
    try {
      // cache:'no-store' AND a unique query string: the first tells the HTTP
      // cache, the second keeps the URL from matching the service worker's
      // /api/ NetworkFirst rule at all. Either alone has been observed to be
      // insufficient on iOS.
      var r = await fetch('/api/v1/health?_=' + Date.now(), { cache: 'no-store', credentials: 'include' });
      h = r.ok ? await r.json() : null;
    } catch (e) {
      h = null; // unreachable — a warning in its own right, not a silence
    }
    var warnings = [];
    names.forEach(function (n) {
      var c = CHECKS[n];
      if (c && c.test(h)) {
        warnings.push({ key: c.key, text: c.text, value: h === null ? c.value() : c.value(h) });
      }
    });
    render(el, warnings, global.location ? global.location.host : '');
  }

  // names: which checks this page cares about. A warning shown where it cannot
  // be acted on is the same noise as no warning at all — Onboarding does not
  // need to hear about Toast.
  function initHealthBanners(elId, names) {
    var el = document.getElementById(elId);
    if (!el) return;
    var run = function () { refresh(el, names); };
    run();
    // Re-check so a banner appears and clears without a reload; on focus too,
    // which is the pattern a phone actually follows (backgrounded, reopened).
    setInterval(run, 60000);
    document.addEventListener('visibilitychange', function () { if (!document.hidden) run(); });
  }

  global.initHealthBanners = initHealthBanners;
  global.__healthChecks = CHECKS; // test seam
})(window);
