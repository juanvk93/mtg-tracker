// Service worker mínimo: cachea el shell y sirve network-first (datos frescos online,
// funciona degradado sin conexión).
const CACHE = 'mtg-tracker-v1';
const SHELL = ['/', '/static/css/main.css', '/static/js/main.js', '/static/icon.svg'];

self.addEventListener('install', (e) => {
  e.waitUntil(caches.open(CACHE).then((c) => c.addAll(SHELL)).catch(() => {}));
  self.skipWaiting();
});

self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
  );
  self.clients.claim();
});

self.addEventListener('fetch', (e) => {
  if (e.request.method !== 'GET') return;
  e.respondWith(
    fetch(e.request)
      .then((resp) => {
        // Guardar una copia de recursos estáticos para uso offline
        if (e.request.url.includes('/static/')) {
          const clone = resp.clone();
          caches.open(CACHE).then((c) => c.put(e.request, clone)).catch(() => {});
        }
        return resp;
      })
      .catch(() => caches.match(e.request))
  );
});
