# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Tracker de partidas de Magic: The Gathering (Draft mensual) para un grupo de amigos. Backend Go (solo stdlib `net/http`), HTML renderizado en servidor con `html/template`, CSS puro + Mana Font, PostgreSQL en Supabase, deploy en Render vía Docker. Versión actual: ver `internal/version`.

## Comandos

```bash
# go.sum YA está commiteado (el Dockerfile lo copia y lo exige). Si cambias dependencias,
# ejecuta go mod tidy y commitea go.mod + go.sum antes de push/deploy, o el build de Render falla.
go mod tidy

go build ./...
go vet ./...
go test ./...                 # tests de funciones puras (calcularRacha, desviación, gráfica de evolución)

# Ejecutar en local (necesita una BD PostgreSQL accesible):
DATABASE_URL="postgresql://localhost:5432/mtg_tracker?sslmode=disable" \
ADMIN_PASSWORD=admin \
go run ./cmd/server           # sirve en :8080 (PORT lo cambia)

docker build -t mtg-tracker . # multistage, binario estático (CGO_ENABLED=0), imagen Alpine final
```

`DATABASE_URL` es obligatoria (la app hace `log.Fatal` sin ella). `ADMIN_PASSWORD` por defecto es `admin`. El esquema se crea/actualiza solo al arrancar (`aplicarEsquema`): `CREATE TABLE IF NOT EXISTS` + `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` idempotentes. Para añadir una columna, agrega un `ALTER ... IF NOT EXISTS` a esa lista (no hay sistema de migraciones versionadas).

## Arquitectura

Servidor web clásico renderizado en servidor. Sin ORM, sin router externo, sin framework JS.

- **`cmd/server/main.go`** — entrada. Registra rutas con el router de `net/http` de Go 1.22 (patrones `"GET /sesion/{id}"`, se leen con `r.PathValue("id")`). Aquí vive `cargarTemplates()` con **todas las funciones de template** (`formatFecha`, `nombreColor`, `emojiColor`, `renderAvatar`, `iconosJugador`, `mulPct`, `colorHex`, `seq`, `add1`, etc.). Si una plantilla necesita una función nueva, se añade aquí.
- **`internal/handlers/`** — `publico.go` (vistas sin auth: inicio/ranking, historial, detalle de sesión, estadísticas, **perfil de jugador** `/jugador/{id}`, **modo TV** `/tv`, changelog, resumen de temporada; incluye el helper `resolverTemporada`), `admin.go` (auth + CRUD + activar/desactivar jugadores + operaciones destructivas), `graficas.go` (geometría SVG de las gráficas de evolución y win rate, y los premios de temporada, todo precalculado en Go), `backup_handler.go` (export/import JSON). Todo cuelga de `AppHandlers{DB, Templates}`; se renderiza con el helper `renderizar()`, que ejecuta a un buffer antes de escribir (evita HTML a medias si el template falla).
- **`internal/db/`** — única capa que toca SQL. `db.go` (conexión pgx + esquema/migraciones), `queries.go` (CRUD, ranking, rachas, ganador de sesión, KPIs de color), `stats.go` (estadísticas avanzadas por jugador, historial de drafts, evolución por sesión, matriz head-to-head, meta de colores y resúmenes de temporada), `backup.go` (serialización JSON completa).
- **`internal/models/`** — structs de dominio + helpers de Magic (`NombreColor`, `EmojiColor`, `IconosJugador`).
- **`internal/version/`** — `Version` (const) y `Changelog()` (lista de cambios que alimenta `/changelog`).

### Sistema de plantillas (importante)

**Cada página es una plantilla independiente con su HTML completo.** No hay layout base que envuelva a las páginas: cada archivo es `{{define "inicio.html"}}<!DOCTYPE html>…</html>{{end}}` e incluye la barra de navegación con `{{template "nav" .}}`. `base.html` **solo** define el partial `nav`. Todo se carga en un único `*template.Template` con tres `ParseGlob` (base primero, para registrar `nav`), y `renderizar(w, "inicio.html", datos)` ejecuta la plantilla por su nombre de archivo. No hay colisión de bloques porque las páginas no comparten `{{define}}` con el mismo nombre (cada página ES su propio template). Al crear una página nueva: `{{define "archivo.html"}}` + incluir `{{template "nav" .}}` + añadir su ruta y handler.

- `base.html` define TRES partials: `nav` (barra + botón de tema), `selectorTemporada` y `head` (meta theme-color, `<link rel="manifest">`, icono y script anti-parpadeo del tema). Cada página, tras su `<title>`, incluye `{{template "head" .}}`; las que tienen barra incluyen `{{template "nav" .}}`. `tv.html` es autónoma (sin nav, forzada a oscuro).
- **Mana Font**: los símbolos de color y los avatares son iconos de la fuente Mana Font (`<i class="ms ms-...">`), no emojis. El CSS se carga por CDN **dentro del partial `nav`** (y en `tv.html`, que no lo usa). `models.EmojiColor` devuelve `template.HTML` con el `<i>`; `renderAvatar` (en `main.go`) hace fallback a emoji para avatares antiguos que no empiezan por `ms-`.
- **La versión aparece en DOS sitios**: `internal/version/version.go` (`Version` + changelog) y **hardcodeada** en el texto del enlace de `templates/base.html` (p. ej. `v1.7.0`). Al publicar una versión hay que actualizar ambos y añadir una entrada al `Changelog()`.
- **Sistema de diseño (CSS)**: todo el estilo vive en `static/css/main.css` con tokens en `:root` (y sus overrides oscuros). Paleta fría con acento turquesa `--brand` (+ `--brand-bg`/`--brand-glow` para los difuminados), podio metálico `--oro`/`--plata`/`--bronce`, `--shadow`/`--shadow-hover`, y `--font-display` (Cinzel, para titulares/números; body en Inter). Convenciones: tarjeta-héroe del MVP `.mvp-hero` en el ranking; KPIs destacados con `.stat-card.kpi-hero`; fila del líder `tr.fila-lider` y posiciones `.rank-1/2/3`; win rate como píldora `.win-rate.alta/media/baja`; puntos de maná por jugador `.manas .m-{W,U,B,R,G}` (los colores los aporta `FilaRanking.Colores`). El MVP (líder de victorias y win rate) se muestra en la tarjeta-héroe cuando es el mismo jugador; si difieren, salen dos `kpi-hero` en la parrilla (lógica en `inicio.html`).
- **Selector de temporada**: las vistas públicas resuelven la temporada con `resolverTemporada` (`?temporada=ID` → activa → más reciente) y ese `<select>` reutilizable vive en `{{define "selectorTemporada"}}` de `base.html`. Al añadir una vista que dependa de temporada, usa el helper y pásale `Temporadas` + `Temporada`.
- **Tema claro/oscuro**: por defecto sigue `prefers-color-scheme`; el botón ◐ del nav (`toggleTema` en `main.js`) fija `data-theme` en `localStorage`. El CSS override vive en `:root[data-theme="dark"]` y el `@media` oscuro está guardado con `:root:not([data-theme="light"])`. El script del partial `head` lo aplica antes de pintar.
- **PWA**: `static/manifest.webmanifest` + `static/sw.js` (network-first, cachea el shell) + `static/icon.svg`; el service worker se registra en `main.js`.

### Modelo de datos y reglas de dominio

Jerarquía: `temporadas` (un año) → `sesiones` (un draft, con fecha) → `resultados` (un jugador en una sesión) → `victorias` + `colores_jugados` (hijos de un resultado, borrado en cascada).

- **Solo puede haber UNA temporada activa.** `CrearTemporada` y `ReabrirTemporada` cierran cualquier otra activa dentro de una transacción.
- **Jugadores inactivos** (`jugadores.activo`): se desactivan sin borrarlos (mantienen su histórico y siguen en ranking/estadísticas). Los formularios de resultados usan `ObtenerJugadoresActivos` **más** los que ya participaron en esa sesión (para poder editar sesiones antiguas).
- **Notas por resultado** (`resultados.notas`): arquetipo/comentario libre de cada jugador en cada draft; se guarda en `GuardarResultadoSesion` (campo `notas_<id>` del formulario) y se muestra en la sesión y en el perfil.
- **Las derrotas NO se almacenan.** Solo se guarda a quién venció cada jugador (`victorias.rival_id`). Una "derrota" de X se deriva contando las victorias de otros *sobre X en la misma sesión*. Todo cálculo de win/loss sigue esta lógica (ver `ObtenerRanking`, `ObtenerDetalleSesion`); no inventes una tabla de derrotas.
- **Win rate por color = victorias/(victorias+derrotas)** (normalizado, nunca supera 100%). `ObtenerEstadisticasColores` lo calcula en UNA consulta y devuelve `Veces` (nº de drafts con el color) y `Partidas` (V+D).
- **Ranking y rachas se calculan en Go**, no en SQL: `ObtenerRanking` trae todos los resultados de la temporada en UNA consulta y agrega en memoria; la racha se calcula con el helper puro `calcularRacha` (historial en orden fecha desc). `ObtenerDetalleSesion` usa un nº fijo de consultas y deriva las derrotas en memoria. Orden del ranking: victorias desc, luego win rate desc.
- Colores fijos `W U B R G`. `db.ColorHex` da el hex (para lógica/estilos) y Mana Font el símbolo (para mostrar).
- Al guardar resultados (`AdminGuardarResultados`), se borran **solo** los resultados de jugadores retirados de la sesión; el resto se actualiza (cada uno en su transacción vía `GuardarResultadoSesion`). El formulario también actualiza la descripción de la sesión.

### Auth admin

No hay sesiones ni hashing: `MiddlewareAdmin` compara la cookie `admin_auth` directamente con `ADMIN_PASSWORD` (la contraseña en claro es el valor de la cookie). Es deliberado para un tracker privado tras el HTTPS de Render. Las operaciones destructivas piden re-confirmar la contraseña: `AdminBorrarTemporada` (borra una temporada y sus datos) y `AdminFarewell` (borra **todo** vía `TRUNCATE ... RESTART IDENTITY CASCADE`; exige además el texto literal `FAREWELL`).

## Gotchas importantes

- **`go.sum` debe estar commiteado** (ya lo está): el Dockerfile hace `COPY go.mod go.sum`. Si cambias dependencias, `go mod tidy` + commitear `go.sum` o el build de Render falla.
- **pgx en modo Simple Protocol**: `db.go` fija `cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol`. Es obligatorio con el pooler de Supabase (modo Transaction): sin ello aparece `prepared statement already exists` (SQLSTATE 42P05). No lo quites.
- **pgx no admite múltiples sentencias por `Exec`**: por eso `aplicarEsquema` ejecuta cada `CREATE TABLE`/`ALTER`/`INDEX` por separado. Mantén una sentencia por string. Las columnas nuevas se añaden con `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` (idempotente) en esa misma lista.
- Pool limitado a propósito (`MaxOpenConns(5)`) por el free tier de Supabase.
- Deploy (`render.yaml`): servicio Docker en Render. Variables a configurar a mano: `DATABASE_URL` y `ADMIN_PASSWORD` (`PORT` viene por defecto a 8080).
