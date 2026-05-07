# 🃏 MTG Draft Tracker

Tracker de partidas de Magic: The Gathering para grupos de amigos que juegan Draft mensualmente.

## Stack

- **Backend**: Go (net/http stdlib)
- **Frontend**: HTML/CSS puro (sin frameworks JS)
- **Base de datos**: PostgreSQL en Supabase (free tier)
- **Deploy**: Render (free tier)

---

## Deploy completo (Render + Supabase)

### Paso 1: Crear la base de datos en Supabase

1. Ve a [supabase.com](https://supabase.com) y crea una cuenta
2. Crea un **nuevo proyecto**:
   - Nombre: `mtg-tracker` (o el que quieras)
   - Contraseña de la BD: **apúntala, la necesitarás**
   - Región: lo más cercano a tus amigos (ej: West EU para España)
3. Una vez creado, ve a **Settings → Database**
4. En la sección **Connection string → URI**, copia la URL. Tendrá este formato:
   ```
   postgresql://postgres.[ID]:[PASSWORD]@aws-0-eu-west-1.pooler.supabase.com:6543/postgres
   ```
5. Sustituye `[PASSWORD]` por la contraseña que elegiste

> **No necesitas crear tablas manualmente.** La app las crea automáticamente al arrancar.

### Paso 2: Subir el código a GitHub

1. Crea un repositorio en GitHub (puede ser privado)
2. Sube todo el contenido de esta carpeta:
   ```bash
   cd mtg-tracker
   git init
   git add .
   git commit -m "MTG Draft Tracker inicial"
   git remote add origin https://github.com/TU_USUARIO/mtg-tracker.git
   git push -u origin main
   ```

### Paso 3: Deploy en Render

1. Ve a [render.com](https://render.com) y crea una cuenta (usa "Sign in with GitHub")
2. Click en **New → Web Service**
3. Conecta tu repositorio de GitHub
4. Configuración:
   - **Name**: `mtg-tracker` (o el que quieras)
   - **Region**: Frankfurt o Amsterdam (cercano a España)
   - **Runtime**: Docker
   - **Instance Type**: Free
5. Añade las **variables de entorno** (Environment → Add Environment Variable):

   | Key              | Value                                           |
   |------------------|-------------------------------------------------|
   | `DATABASE_URL`   | La URL de conexión de Supabase (paso 1.4)       |
   | `ADMIN_PASSWORD` | La contraseña que quieras para el panel de admin |
   | `PORT`           | `8080`                                          |

6. Click en **Deploy**

La primera compilación tarda ~2 minutos. Después, la app estará en:
```
https://mtg-tracker-XXXX.onrender.com
```

### Notas sobre el free tier de Render

- La app **se duerme** tras 15 minutos sin tráfico
- La primera visita tras dormirse tarda **~30 segundos** en arrancar (cold start)
- No afecta a los datos: están en Supabase, no en Render
- Para una app que usáis una vez al mes es más que suficiente

---

## Desarrollo local

### Requisitos

- Go 1.22+
- Una base de datos PostgreSQL (puede ser la de Supabase o una local)

### Con la BD de Supabase (lo más fácil)

```bash
# Clonar y entrar al proyecto
cd mtg-tracker

# Instalar dependencias
go mod download

# Arrancar con la URL de Supabase
DATABASE_URL="postgresql://postgres.[ID]:[PASS]@aws-0-eu-west-1.pooler.supabase.com:6543/postgres" \
ADMIN_PASSWORD=admin \
go run ./cmd/server
```

### Con PostgreSQL local (opcional)

```bash
# Crear base de datos
createdb mtg_tracker

# Arrancar
DATABASE_URL="postgresql://localhost:5432/mtg_tracker?sslmode=disable" \
ADMIN_PASSWORD=admin \
go run ./cmd/server
```

Abre `http://localhost:8080`

### Variables de entorno

| Variable         | Descripción                         | Obligatoria |
|-----------------|-------------------------------------|-------------|
| `DATABASE_URL`  | URL de conexión PostgreSQL           | Sí          |
| `ADMIN_PASSWORD`| Contraseña del panel admin           | No (default: `admin`) |
| `PORT`          | Puerto del servidor                  | No (default: `8080`)  |

---

## Uso

### Primer uso

1. Accede a `/admin` e introduce la contraseña configurada
2. Crea una **temporada** con el año actual (ej: 2026)
3. Añade los **jugadores** del grupo (nombre + emoji + color)
4. Cuando tengáis una sesión de draft, crea una **sesión**
5. Registra los **resultados**: quién jugó, a quién venció cada uno, qué colores usó

### Vista pública (para los amigos)

Comparte la URL directamente, no necesitan login:

- `/` — **Ranking** de la temporada activa con win rates y rachas
- `/historial` — Todas las sesiones jugadas
- `/sesion/{id}` — Detalle de cada sesión con resultados
- `/estadisticas` — Colores por jugador y head-to-head

---

## Estructura del proyecto

```
mtg-tracker/
├── cmd/server/
│   └── main.go              # Punto de entrada, rutas HTTP, funciones de template
├── internal/
│   ├── db/
│   │   ├── db.go            # Conexión PostgreSQL y esquema
│   │   └── queries.go       # Consultas: ranking, head-to-head, stats, rachas
│   ├── handlers/
│   │   ├── publico.go       # Vistas públicas (ranking, historial, estadísticas)
│   │   └── admin.go         # Panel de administración (auth, CRUD)
│   └── models/
│       └── models.go        # Estructuras de datos y helpers de colores
├── templates/
│   ├── base.html            # Layout base con navegación
│   ├── public/              # Páginas públicas (ranking, historial, sesión, stats)
│   └── admin/               # Páginas de admin (login, panel, resultados, edición)
├── static/
│   ├── css/main.css         # Estilos (tema oscuro, Cinzel + Inter, colores de maná)
│   └── js/main.js           # Interactividad de formularios
├── Dockerfile               # Multistage: Go alpine → alpine (imagen ~15MB)
├── render.yaml              # Blueprint de Render
├── go.mod                   # Dependencias (pgx para PostgreSQL)
└── README.md
```

---

## Backup de datos

Desde el dashboard de Supabase puedes:

- **Table Editor**: ver y editar datos directamente
- **SQL Editor**: ejecutar consultas SQL
- **Backups**: Supabase hace backups diarios automáticos (en plan free, retención de 7 días)

Para un backup manual:

```bash
# Exportar toda la BD (necesitas pg_dump instalado)
pg_dump "TU_DATABASE_URL" > backup.sql

# Restaurar
psql "TU_DATABASE_URL" < backup.sql
```

---

## FAQ

**¿Puedo usar otro hosting en lugar de Render?**
Sí, cualquier plataforma que soporte Docker o binarios Go: Railway, Fly.io, un VPS, etc. Solo necesitas configurar `DATABASE_URL` y `ADMIN_PASSWORD`.

**¿Y si quiero cambiar de Supabase?**
Cualquier PostgreSQL funciona. Cambia `DATABASE_URL` y listo.

**¿Es segura la autenticación?**
Es una cookie HTTP-only con la contraseña. Suficiente para un tracker privado entre amigos. Render fuerza HTTPS así que la cookie va cifrada en tránsito.

**¿Cuánto cuesta?**
Cero. Supabase free tier (500MB) y Render free tier (con cold start) son más que suficientes para un grupo de amigos que juega una vez al mes.
