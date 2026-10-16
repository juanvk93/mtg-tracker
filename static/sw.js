// Service worker mínimo: cachea el shell y sirve network-first (datos frescos online,
// funciona degradado sin conexión).
const CACHE = 'mtg-tracker-v2';
// /vidas va en el shell para poder llevar las vidas sin conexión
const SHELL = ['/', '/vidas', '/static/css/main.css', '/static/js/main.js', '/static/icon.svg'];

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
        // Guardar una copia de recursos estáticos (y del contador de vidas) para uso offline
        const ruta = new URL(e.request.url).pathname;
        if (resp.ok && (ruta.startsWith('/static/') || ruta === '/vidas')) {
          const clone = resp.clone();
          caches.open(CACHE).then((c) => c.put(e.request, clone)).catch(() => {});
        }
        return resp;
      })
      .catch(() => caches.match(e.request))
  );
});
