const CACHE = 'yumyums-v51';
const ASSETS = ['./', './purchasing.html', './users.html', './login.html', './workflows.html', './onboarding.html', './ptr.js', './manifest.json'];

self.addEventListener('install', e => {
  e.waitUntil(caches.open(CACHE).then(c => c.addAll(ASSETS)));
  self.skipWaiting();
});

self.addEventListener('activate', e => {
  e.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.filter(k => k !== CACHE).map(k => caches.delete(k)))
    )
  );
  self.clients.claim();
});

self.addEventListener('fetch', e => {
  const url = new URL(e.request.url);
  if (url.pathname.startsWith('/api/')) {
    // Network-first: API calls always go to network; no caching
    e.respondWith(fetch(e.request).catch(() => new Response('{"error":"offline"}', {
      status: 503,
      headers: {'Content-Type':'application/json'}
    })));
    return;
  }
  // Cache-first: static assets served from cache; fallback to network
  e.respondWith(
    caches.match(e.request).then(r => r || fetch(e.request).catch(() => caches.match('./index.html')))
  );
});
