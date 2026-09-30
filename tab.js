// tab.js — Shared tab persistence via URL hash.
// Include right after tab section divs. Synchronous — runs before paint.
// Usage: <script src="tab.js" data-tabs="3"></script>
//   data-tabs: number of tabs (default 3)
//   data-home: the slot to activate with no (or an invalid) hash; default 1.
//     inventory.html sets 0 so its hub (#s0) paints before boot instead of
//     Receipts flashing first (B-455 / WO-2a). A page without #s0 is unaffected.
//   Reads #tab=N from URL hash, activates matching tab button (t1..tN)
//   and section (s1..sN). Falls back to the home slot if no hash.
(function() {
  var script = document.currentScript;
  var count = +(script && script.dataset.tabs) || 3;
  var home = script && script.dataset.home != null ? +script.dataset.home : 1;
  var m = location.hash.match(/tab=(\d+)/);
  var active = m ? +m[1] : home;
  if (active < 0 || active > count) active = home;
  for (var i = 0; i <= count; i++) {
    var btn = document.getElementById('t' + i);
    // Toggle, don't assign: the Inventory hub rows carry classes of their own
    // (hub-row, dim) that an assignment would strip.
    if (btn) btn.classList.toggle('on', i === active);
    var sec = document.getElementById('s' + i);
    if (sec) sec.style.display = i === active ? '' : 'none';
  }
})();
