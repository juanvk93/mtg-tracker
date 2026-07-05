# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Tracker de partidas de Magic: The Gathering (Draft mensual) para un grupo de amigos. Backend Go (solo stdlib `net/http`), HTML renderizado en servidor con `html/template`, CSS puro + Mana Font, PostgreSQL en Supabase, deploy en Render vía Docker. Versión actual: ver `internal/version`.

## Comandos

```bash
# go.sum NO está commiteado y el Dockerfile lo exige (COPY go.mod go.sum).
# Ejecútalo antes del primer build y ANTES de cualquier push/deploy o el build de Docker/Render falla:
go mod tidy

go build ./...
go vet ./...
go test ./...                 # no hay tests todavía

# Ejecutar en local (necesita una BD PostgreSQL accesible):
DATABASE_URL="postgresql://localhost:5432/mtg_tracker?sslmode=disable" \
ADMIN_PASSWORD=admin \
go run ./cmd/server           # sirve en :8080 (PORT lo cambia)

docker build -t mtg-tracker . # multistage, binario estático (CGO_ENABLED=0), imagen Alpine final
```

`DATABASE_URL` es obligatoria (la app hace `log.Fatal` sin ella). `ADMIN_PASSWORD` por defecto es `admin`. El esquema se crea solo al arrancar; no hay migraciones.

## Arquitectura

Servidor web clásico renderizado en servidor. Sin ORM, sin router externo, sin framework JS.

- **`cmd/server/main.go`** — entrada. Registra rutas con el router de `net/http` de Go 1.22 (patrones `"GET /sesion/{id}"`, se leen con `r.PathValue("id")`). Aquí vive `cargarTemplates()` con **todas las funciones de template** (`formatFecha`, `nombreColor`, `emojiColor`, `renderAvatar`, `iconosJugador`, `mulPct`, `colorHex`, `seq`, `add1`, etc.). Si una plantilla necesita una función nueva, se añade aquí.
- **`internal/handlers/`** — `publico.go` (vistas sin auth), `admin.go` (auth + CRUD + operaciones destructivas), `backup_handler.go` (export/import JSON). Todo cuelga de `AppHandlers{DB, Templates}`; se renderiza con el helper `renderizar()`, que ejecuta a un buffer antes de escribir (evita HTML a medias si el template falla).
- **`internal/db/`** — única capa que toca SQL. `db.go` (conexión pgx + esquema), `queries.go` (CRUD, ranking, rachas, ganador de sesión, KPIs de color), `stats.go` (estadísticas avanzadas por jugador y resúmenes de temporada), `backup.go` (serialización JSON completa).
- **`internal/models/`** — structs de dominio + helpers de Magic (`NombreColor`, `EmojiColor`, `IconosJugador`).
- **`internal/version/`** — `Version` (const) y `Changelog()` (lista de cambios que alimenta `/changelog`).

### Sistema de plantillas (importante)

**Cada página es una plantilla independiente con su HTML completo.** No hay layout base que envuelva a las páginas: cada archivo es `{{define "inicio.html"}}<!DOCTYPE html>…</html>{{end}}` e incluye la barra de navegación con `{{template "nav" .}}`. `base.html` **solo** define el partial `nav`. Todo se carga en un único `*template.Template` con tres `ParseGlob` (base primero, para registrar `nav`), y `renderizar(w, "inicio.html", datos)` ejecuta la plantilla por su nombre de archivo. No hay colisión de bloques porque las páginas no comparten `{{define}}` con el mismo nombre (cada página ES su propio template). Al crear una página nueva: `{{define "archivo.html"}}` + incluir `{{template "nav" .}}` + añadir su ruta y handler.

- **Mana Font**: los símbolos de color y los avatares son iconos de la fuente Mana Font (`<i class="ms ms-...">`), no emojis. El CSS se carga por CDN **dentro del partial `nav`**. `models.EmojiColor` devuelve `template.HTML` con el `<i>`; `renderAvatar` (en `main.go`) hace fallback a emoji para avatares antiguos que no empiezan por `ms-`.
- **La versión aparece en DOS sitios**: `internal/version/version.go` (`Version` + changelog) y **hardcodeada** en el texto del enlace de `templates/base.html` (`v1.4.0`). Al publicar una versión hay que actualizar ambos.

### Modelo de datos y reglas de dominio

Jerarquía: `temporadas` (un año) → `sesiones` (un draft, con fecha) → `resultados` (un jugador en una sesión) → `victorias` + `colores_jugados` (hijos de un resultado, borrado en cascada).

- **Solo puede haber UNA temporada activa.** `CrearTemporada` y `ReabrirTemporada` cierran cualquier otra activa dentro de una transacción.
- **Las derrotas NO se almacenan.** Solo se guarda a quién venció cada jugador (`victorias.rival_id`). Una "derrota" de X se deriva contando las victorias de otros *sobre X en la misma sesión*. Todo cálculo de win/loss sigue esta lógica (ver `calcularFilaRanking`, `ObtenerDetalleSesion`); no inventes una tabla de derrotas.
- **Win rate por color = victorias/(victorias+derrotas)** (normalizado, nunca supera 100%). Ver `ObtenerEstadisticasColores` / `ObtenerColorMasWinRate`.
- **Ranking y rachas se calculan en Go**, no en SQL: `ObtenerRanking` recorre jugadores y `calcularFilaRanking` hace varias consultas por jugador (patrón N+1; los grupos son pequeños). Orden: victorias desc, luego win rate desc.
- Colores fijos `W U B R G`. `db.ColorHex` da el hex (para lógica/estilos) y Mana Font el símbolo (para mostrar).
- Al guardar resultados (`AdminGuardarResultados`), se borran **solo** los resultados de jugadores retirados de la sesión; el resto se actualiza (cada uno en su transacción vía `GuardarResultadoSesion`). El formulario también actualiza la descripción de la sesión.

### Auth admin

No hay sesiones ni hashing: `MiddlewareAdmin` compara la cookie `admin_auth` directamente con `ADMIN_PASSWORD` (la contraseña en claro es el valor de la cookie). Es deliberado para un tracker privado tras el HTTPS de Render. Las operaciones destructivas piden re-confirmar la contraseña: `AdminBorrarTemporada` (borra una temporada y sus datos) y `AdminFarewell` (borra **todo** vía `TRUNCATE ... RESTART IDENTITY CASCADE`; exige además el texto literal `FAREWELL`).

## Gotchas importantes

- **`go.sum` no está en el repo** pero el Dockerfile lo copia. Un `docker build` / deploy en Render falla si no ejecutas `go mod tidy` y lo commiteas antes.
- **pgx en modo Simple Protocol**: `db.go` fija `cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol`. Es obligatorio con el pooler de Supabase (modo Transaction): sin ello aparece `prepared statement already exists` (SQLSTATE 42P05). No lo quites.
- **pgx no admite múltiples sentencias por `Exec`**: por eso `aplicarEsquema` ejecuta cada `CREATE TABLE`/`INDEX` por separado. Mantén una sentencia por string.
- Pool limitado a propósito (`MaxOpenConns(5)`) por el free tier de Supabase.
- Deploy (`render.yaml`): servicio Docker en Render. Variables a configurar a mano: `DATABASE_URL` y `ADMIN_PASSWORD` (`PORT` viene por defecto a 8080).
