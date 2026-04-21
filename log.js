(function() {
  var origError = console.error;
  var origWarn = console.warn;

  function shipLog(level, args) {
    try {
      var msg = Array.from(args).map(function(a) {
        if (a instanceof Error) return a.message + (a.stack ? '\n' + a.stack : '');
        if (typeof a === 'object') try { var s = JSON.stringify(a); return s === '{}' ? String(a) : s; } catch(e) { return String(a); }
        return String(a);
      }).join(' ');
      navigator.sendBeacon('/api/v1/logs', JSON.stringify({
        level: level,
        message: msg,
        url: location.pathname,
        ua: navigator.userAgent.slice(0, 120)
      }));
    } catch(e) {}
  }

  console.error = function() {
    origError.apply(console, arguments);
    shipLog('error', arguments);
  };

  console.warn = function() {
    origWarn.apply(console, arguments);
    shipLog('warn', arguments);
  };

  window.addEventListener('error', function(e) {
    shipLog('error', [e.message + ' at ' + e.filename + ':' + e.lineno]);
  });

  window.addEventListener('unhandledrejection', function(e) {
    shipLog('error', ['Unhandled rejection: ' + (e.reason && e.reason.message || e.reason)]);
  });
})();
