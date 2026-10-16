// Servidor principal del tracker de Magic: The Gathering
// Usa PostgreSQL (Supabase) como base de datos
package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"mtg-tracker/internal/db"
	"mtg-tracker/internal/handlers"
	"mtg-tracker/internal/models"
)

func main() {
	// Configuración desde variables de entorno
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("Variable de entorno DATABASE_URL es obligatoria.\n" +
			"Ejemplo: postgresql://user:pass@host:5432/dbname?sslmode=require")
	}

	// Inicializar base de datos PostgreSQL
	database, err := db.Inicializar(databaseURL)
	if err != nil {
		log.Fatal("Error al conectar con la base de datos:", err)
	}
	defer database.Close()

	// Cargar templates con funciones auxiliares
	tmpl := cargarTemplates()

	// Crear handlers
	h := handlers.NuevoAppHandlers(database, tmpl)

	// Configurar rutas
	mux := http.NewServeMux()

	// Archivos estáticos
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// ========== RUTAS PÚBLICAS ==========
	mux.HandleFunc("GET /", h.Inicio)
	mux.HandleFunc("GET /historial", h.Historial)
	mux.HandleFunc("GET /sesion/{id}", h.DetalleSesion)
	mux.HandleFunc("GET /estadisticas", h.Estadisticas)
	mux.HandleFunc("GET /jugador/{id}", h.PerfilJugador)
	mux.HandleFunc("GET /tv", h.ModoTV)
	mux.HandleFunc("GET /changelog", h.Changelog)
	mux.HandleFunc("GET /temporada/{id}/resumen", h.ResumenTemporada)
	mux.HandleFunc("GET /dado", h.Dado)
	mux.HandleFunc("GET /vidas", h.Vidas)
	mux.HandleFunc("GET /healthz", h.Healthz)

	// ========== RUTAS DE ADMINISTRACIÓN ==========
	mux.HandleFunc("GET /admin/login", h.AdminLogin)
	mux.HandleFunc("POST /admin/login", h.AdminLogin)
	mux.HandleFunc("GET /admin/logout", h.AdminLogout)

	// Panel admin (protegido)
	mux.HandleFunc("GET /admin", handlers.MiddlewareAdmin(h.AdminPanel))

	// Temporadas
	mux.HandleFunc("POST /admin/temporada/crear", handlers.MiddlewareAdmin(h.AdminCrearTemporada))
	mux.HandleFunc("POST /admin/temporada/cerrar", handlers.MiddlewareAdmin(h.AdminCerrarTemporada))
	mux.HandleFunc("POST /admin/temporada/reabrir", handlers.MiddlewareAdmin(h.AdminReabrirTemporada))
	mux.HandleFunc("POST /admin/temporada/borrar", handlers.MiddlewareAdmin(h.AdminBorrarTemporada))

	// Farewell: borra TODOS los datos (requiere doble confirmación)
	mux.HandleFunc("POST /admin/farewell", handlers.MiddlewareAdmin(h.AdminFarewell))

	// Sesiones
	mux.HandleFunc("POST /admin/sesion/crear", handlers.MiddlewareAdmin(h.AdminCrearSesion))
	mux.HandleFunc("GET /admin/sesion/{id}/resultados", handlers.MiddlewareAdmin(h.AdminFormResultados))
	mux.HandleFunc("POST /admin/sesion/{id}/resultados", handlers.MiddlewareAdmin(h.AdminGuardarResultados))

	// Jugadores
	mux.HandleFunc("POST /admin/jugador/crear", handlers.MiddlewareAdmin(h.AdminCrearJugador))
	mux.HandleFunc("POST /admin/jugador/toggle", handlers.MiddlewareAdmin(h.AdminToggleJugador))
	mux.HandleFunc("GET /admin/jugador/{id}/editar", handlers.MiddlewareAdmin(h.AdminEditarJugadorForm))
	mux.HandleFunc("POST /admin/jugador/{id}/editar", handlers.MiddlewareAdmin(h.AdminActualizarJugador))

	// Backup: exportar e importar JSON
	mux.HandleFunc("GET /admin/exportar", handlers.MiddlewareAdmin(h.AdminExportarJSON))
	mux.HandleFunc("GET /admin/importar", handlers.MiddlewareAdmin(h.AdminImportarJSONForm))
	mux.HandleFunc("POST /admin/importar", handlers.MiddlewareAdmin(h.AdminImportarJSON))

	// Keep-alive: evita que Render duerma el servicio y que Supabase pause la BD
	go keepAlive(database)

	// Iniciar servidor
	addr := ":" + puerto
	log.Printf("🃏 MTG Tracker iniciado en http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("Error al iniciar el servidor:", err)
	}
}

// intervaloKeepAlive: muy por debajo de los 15 min de inactividad tras los que Render
// (plan free) duerme el servicio.
const intervaloKeepAlive = 2 * time.Minute

// keepAlive genera actividad periódica mientras el servidor está levantado:
//   - Lanza una consulta ligera a la BD para que Supabase no pause el proyecto.
//   - Si existe RENDER_EXTERNAL_URL (Render la define sola), se hace una petición a
//     /healthz a través de la URL pública: cuenta como tráfico entrante y Render no
//     duerme el servicio. Una llamada a localhost NO serviría, no pasa por su proxy.
func keepAlive(database *sql.DB) {
	urlPublica := strings.TrimRight(os.Getenv("RENDER_EXTERNAL_URL"), "/")
	cliente := &http.Client{Timeout: 30 * time.Second}

	ticker := time.NewTicker(intervaloKeepAlive)
	defer ticker.Stop()
	for range ticker.C {
		if err := db.Ping(database); err != nil {
			log.Println("keep-alive: error en la consulta a la BD:", err)
		}
		if urlPublica == "" {
			continue
		}
		resp, err := cliente.Get(urlPublica + "/healthz")
		if err != nil {
			log.Println("keep-alive: error al llamar a /healthz:", err)
			continue
		}
		resp.Body.Close()
	}
}

// cargarTemplates carga y parsea todos los templates HTML con funciones auxiliares
func cargarTemplates() *template.Template {
	funciones := template.FuncMap{
		// Formatear fecha en español
		"formatFecha": func(t time.Time) string {
			meses := []string{"", "enero", "febrero", "marzo", "abril", "mayo", "junio",
				"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}
			return fmt.Sprintf("%d de %s de %d", t.Day(), meses[t.Month()], t.Year())
		},
		// Formatear fecha corta
		"formatFechaCorta": func(t time.Time) string {
			return t.Format("02/01/2006")
		},
		// Fecha para input type="date"
		"formatFechaInput": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		// Formatear win rate como porcentaje
		"formatPct": func(f float64) string {
			return fmt.Sprintf("%.1f%%", f)
		},
		// Redondear float
		"redondear": func(f float64) int {
			return int(math.Round(f))
		},
		// Nombre de color de Magic
		"nombreColor": models.NombreColor,
		// Emoji de color de Magic
		"emojiColor": models.EmojiColor,
		// Color hex para un símbolo
		"colorHex": db.ColorHex,
		// Unir strings
		"join": strings.Join,
		// Secuencia de números
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i + 1
			}
			return s
		},
		// Año actual
		"anioActual": func() int {
			return time.Now().Year()
		},
		// Fecha de hoy para inputs
		"hoy": func() string {
			return time.Now().Format("2006-01-02")
		},
		// Mayor de dos enteros
		"max": func(a, b int) int {
			if a > b {
				return a
			}
			return b
		},
		// Convertir int a string
		"itoa": func(i int) string {
			return fmt.Sprintf("%d", i)
		},
		// Sumar 1 a un entero (para índices en templates)
		"add1": func(i int) int {
			return i + 1
		},
		// Resta de dos enteros (para el balance neto de la matriz de dominancia)
		"resta": func(a, b int) int {
			return a - b
		},
		// Calcular un porcentaje a partir de dos enteros (parte/total*100)
		"mulPct": func(parte, total int) float64 {
			if total == 0 {
				return 0
			}
			return float64(parte) / float64(total) * 100
		},
		// Catálogo de iconos Mana Font para selector de avatar
		"iconosJugador": models.IconosJugador,
		// Renderizar avatar: icono de Mana Font (clase validada) o emoji legacy escapado
		"renderAvatar": models.RenderAvatar,
	}

	// Parsear templates — cada template es autosuficiente con su propio HTML completo.
	// base.html solo define el partial "nav" que los demás incluyen con {{template "nav" .}}
	// El orden importa: base.html primero para registrar el partial nav.
	tmpl := template.Must(template.New("").Funcs(funciones).ParseGlob("templates/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("templates/public/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("templates/admin/*.html"))

	return tmpl
}
