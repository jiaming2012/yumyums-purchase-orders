// tab.js — Shared tab persistence via URL hash.
// Include right after tab section divs. Synchronous — runs before paint.
// Usage: <script src="tab.js" data-tabs="3"></script>
//   data-tabs: number of tabs (default 3)
//   Reads #tab=N from URL hash, activates matching tab button (t1..tN)
//   and section (s1..sN). Falls back to tab 1 if no hash.
(function() {
  var script = document.currentScript;
  var count = +(script && script.dataset.tabs) || 3;
  var m = location.hash.match(/tab=(\d+)/);
  var active = m ? +m[1] : 1;
  if (active < 1 || active > count) active = 1;
  for (var i = 1; i <= count; i++) {
    var btn = document.getElementById('t' + i);
    if (btn) btn.className = i === active ? 'on' : '';
    var sec = document.getElementById('s' + i);
    if (sec) sec.style.display = i === active ? '' : 'none';
  }
})();
