// Service worker: se sirve en /sw.js (no en /static/) para que su ámbito sea toda la web.
// Estrategia network-first: online siempre datos frescos; sin conexión (o si la red no
// responde en unos segundos) se usa la copia guardada. Solo se guardan los estáticos y
// las páginas que funcionan sin servidor (/vidas, /dado); nunca el admin.
const CACHE = 'mtg-tracker-v3';
const PRECACHE = ['/vidas', '/dado', '/static/css/main.css', '/static/js/main.js', '/static/icon.svg'];
const ESPERA_RED_MS = 4000;

function cacheable(ruta) {
  return ruta.startsWith('/static/') || ruta === '/vidas' || ruta === '/dado';
}

self.addEventListener('install', (e) => {
  // Cada recurso por separado: si uno falla, los demás se guardan igual
  e.waitUntil(caches.open(CACHE).then((c) => Promise.allSettled(PRECACHE.map((u) => c.add(u)))));
  self.skipWaiting();
});

self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
  );
  self.clients.claim();
});

self.addEventListener('fetch', (e) => {
  const req = e.request;
  if (req.method !== 'GET') return;
  const url = new URL(req.url);
  // Otros orígenes (CDN de fuentes) y admin: los gestiona el navegador sin el SW
  if (url.origin !== self.location.origin || url.pathname.startsWith('/admin')) return;

  e.respondWith((async () => {
    const red = fetch(req).then((resp) => {
      if (resp.ok && cacheable(url.pathname)) {
        const copia = resp.clone();
        caches.open(CACHE).then((c) => c.put(req, copia)).catch(() => {});
      }
      return resp;
    });
    const guardada = await caches.match(req);
    if (!guardada) return red; // sin copia: hay que esperar a la red
    const espera = new Promise((resolve) => setTimeout(() => resolve(guardada), ESPERA_RED_MS));
    return Promise.race([red, espera]).catch(() => guardada);
  })());
});
