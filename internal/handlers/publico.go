// Paquete handlers gestiona las rutas HTTP del tracker
package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"mtg-tracker/internal/db"
	"mtg-tracker/internal/models"
	"mtg-tracker/internal/version"
)

// AppHandlers contiene la instancia de BD y templates para los handlers
type AppHandlers struct {
	DB        *sql.DB
	Templates *template.Template
}

// NuevoAppHandlers crea una nueva instancia de AppHandlers
func NuevoAppHandlers(database *sql.DB, tmpl *template.Template) *AppHandlers {
	return &AppHandlers{DB: database, Templates: tmpl}
}

// ========== VISTAS PÚBLICAS ==========

// Inicio muestra el ranking de la temporada activa
func (a *AppHandlers) Inicio(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	temporada, err := db.ObtenerTemporadaActiva(a.DB)
	if err != nil {
		// No hay temporada activa, mostrar página vacía
		a.renderizar(w, "inicio.html", map[string]interface{}{
			"Temporada": nil,
			"Ranking":   nil,
			"SinDatos":  true,
		})
		return
	}

	ranking, err := db.ObtenerRanking(a.DB, temporada.ID)
	if err != nil {
		log.Println("Error al obtener ranking:", err)
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	colorTop := db.ObtenerColorMasJugado(a.DB, temporada.ID)
	colorWR, wrPct := db.ObtenerColorMasWinRate(a.DB, temporada.ID)
	sesiones := db.ContarSesionesPorTemporada(a.DB, temporada.ID)

	a.renderizar(w, "inicio.html", map[string]interface{}{
		"Temporada":      temporada,
		"Ranking":        ranking,
		"ColorTop":       colorTop,
		"ColorTopNombre": models.NombreColor(colorTop),
		"ColorWR":        colorWR,
		"ColorWRNombre":  models.NombreColor(colorWR),
		"ColorWRPct":     wrPct,
		"Sesiones":       sesiones,
		"SinDatos":       len(ranking) == 0,
	})
}

// Historial muestra el historial de sesiones de la temporada activa
func (a *AppHandlers) Historial(w http.ResponseWriter, r *http.Request) {
	temporada, err := db.ObtenerTemporadaActiva(a.DB)
	if err != nil {
		a.renderizar(w, "historial.html", map[string]interface{}{"SinDatos": true})
		return
	}

	sesiones, err := db.ObtenerSesiones(a.DB, temporada.ID)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	// Calcular el ganador de cada sesión
	ganadores := make(map[int]*db.GanadorSesion)
	for _, s := range sesiones {
		ganadores[s.ID] = db.ObtenerGanadorSesion(a.DB, s.ID)
	}

	a.renderizar(w, "historial.html", map[string]interface{}{
		"Temporada": temporada,
		"Sesiones":  sesiones,
		"Ganadores": ganadores,
		"SinDatos":  len(sesiones) == 0,
	})
}

// DetalleSesion muestra el detalle de una sesión específica
func (a *AppHandlers) DetalleSesion(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	detalle, err := db.ObtenerDetalleSesion(a.DB, id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Println("Error al obtener detalle de sesión:", err)
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	a.renderizar(w, "sesion.html", map[string]interface{}{
		"Detalle": detalle,
	})
}

// Estadisticas muestra estadísticas generales y head-to-head
func (a *AppHandlers) Estadisticas(w http.ResponseWriter, r *http.Request) {
	temporada, err := db.ObtenerTemporadaActiva(a.DB)
	if err != nil {
		a.renderizar(w, "estadisticas.html", map[string]interface{}{"SinDatos": true})
		return
	}

	jugadores, err := db.ObtenerJugadores(a.DB)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	// Cargar estadísticas completas para cada jugador que haya jugado al menos una sesión
	var jugadoresStats []models.EstadisticasJugador
	for _, j := range jugadores {
		stats, err := db.ObtenerEstadisticasCompletasJugador(a.DB, j, temporada.ID)
		if err != nil {
			continue
		}
		// Incluir jugador si ha participado en al menos una sesión
		if stats.SesionesJugadas > 0 {
			jugadoresStats = append(jugadoresStats, stats)
		}
	}

	// Distribución de colores del grupo
	distColores, _ := db.ObtenerDistribucionColoresGrupo(a.DB, temporada.ID)

	// Head-to-head extendido si se especifican los jugadores
	j1Str := r.URL.Query().Get("j1")
	j2Str := r.URL.Query().Get("j2")
	var h2h *models.H2HExtendido

	if j1Str != "" && j2Str != "" {
		j1, err1 := strconv.Atoi(j1Str)
		j2, err2 := strconv.Atoi(j2Str)
		if err1 == nil && err2 == nil && j1 != j2 {
			resultado, err := db.ObtenerH2HExtendido(a.DB, j1, j2)
			if err == nil {
				h2h = resultado
			}
		}
	}

	a.renderizar(w, "estadisticas.html", map[string]interface{}{
		"Temporada":         temporada,
		"Jugadores":         jugadores,
		"JugadoresStats":    jugadoresStats,
		"DistColores":       distColores,
		"H2H":               h2h,
		"J1Sel":             j1Str,
		"J2Sel":             j2Str,
		"SinDatos":          len(jugadores) == 0,
	})
}

// Changelog muestra la lista de versiones y cambios
func (a *AppHandlers) Changelog(w http.ResponseWriter, r *http.Request) {
	a.renderizar(w, "changelog.html", map[string]interface{}{
		"Version":   version.Version,
		"Changelog": version.Changelog(),
	})
}

// ResumenTemporada muestra el resumen ejecutivo de una temporada (activa o cerrada)
func (a *AppHandlers) ResumenTemporada(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	resumen, err := db.ObtenerResumenTemporada(a.DB, id)
	if err != nil {
		log.Println("Error al obtener resumen de temporada:", err)
		http.NotFound(w, r)
		return
	}

	a.renderizar(w, "resumen_temporada.html", map[string]interface{}{
		"Resumen": resumen,
	})
}

// ========== HELPERS ==========

// renderizar ejecuta el template dado con los datos proporcionados
func (a *AppHandlers) renderizar(w http.ResponseWriter, tmpl string, datos interface{}) {
	// Renderizar a buffer primero para evitar respuestas parciales en caso de error
	var buf strings.Builder
	if err := a.Templates.ExecuteTemplate(&buf, tmpl, datos); err != nil {
		log.Printf("Error al renderizar template %s: %v", tmpl, err)
		http.Error(w, "Error al renderizar página", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(buf.String()))
}
