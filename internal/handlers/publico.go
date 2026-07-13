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

	temporada, temporadas, ok := a.resolverTemporada(r)
	if !ok {
		// No hay ninguna temporada, mostrar página vacía
		a.renderizar(w, "inicio.html", map[string]interface{}{"SinDatos": true})
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

	// KPIs de líderes, derivados del ranking ya calculado (sin consultas extra).
	// El ranking viene ordenado por victorias desc, así que el primero con victorias
	// es el líder por victorias.
	var liderVict, liderWR *models.FilaRanking
	for i := range ranking {
		f := &ranking[i]
		if f.Victorias > 0 && (liderVict == nil || f.Victorias > liderVict.Victorias) {
			liderVict = f
		}
		if f.Partidas > 0 && (liderWR == nil ||
			f.WinRate > liderWR.WinRate ||
			(f.WinRate == liderWR.WinRate && f.Partidas > liderWR.Partidas)) {
			liderWR = f
		}
	}

	a.renderizar(w, "inicio.html", map[string]interface{}{
		"Temporada":      temporada,
		"Temporadas":     temporadas,
		"Ranking":        ranking,
		"LiderVict":      liderVict,
		"LiderWR":        liderWR,
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
	temporada, temporadas, ok := a.resolverTemporada(r)
	if !ok {
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
		"Temporada":  temporada,
		"Temporadas": temporadas,
		"Sesiones":   sesiones,
		"Ganadores":  ganadores,
		"SinDatos":   len(sesiones) == 0,
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
	temporada, temporadas, ok := a.resolverTemporada(r)
	if !ok {
		a.renderizar(w, "estadisticas.html", map[string]interface{}{"SinDatos": true})
		return
	}

	jugadores, err := db.ObtenerJugadores(a.DB)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	// Los jugadores INACTIVOS se ocultan de las estadísticas (siguen en el ranking).
	activos := map[int]bool{}
	for _, j := range jugadores {
		if j.Activo {
			activos[j.ID] = true
		}
	}

	// Cargar estadísticas completas de cada jugador ACTIVO con al menos una sesión
	var jugadoresStats []models.EstadisticasJugador
	for _, j := range jugadores {
		if !j.Activo {
			continue
		}
		stats, err := db.ObtenerEstadisticasCompletasJugador(a.DB, j, temporada.ID)
		if err != nil {
			continue
		}
		if stats.SesionesJugadas > 0 {
			jugadoresStats = append(jugadoresStats, stats)
		}
	}

	// Distribución de colores del grupo
	distColores, _ := db.ObtenerDistribucionColoresGrupo(a.DB, temporada.ID)

	// Gráficas de evolución (victorias acumuladas y win rate), solo jugadores activos
	evolucion, _ := db.ObtenerEvolucionVictorias(a.DB, temporada.ID)
	var evolucionActivos []db.EvolucionFila
	for _, f := range evolucion {
		if activos[f.JugadorID] {
			evolucionActivos = append(evolucionActivos, f)
		}
	}
	graficaEvolucion := construirGraficaEvolucion(evolucionActivos)
	graficaWinRate := construirGraficaWinRate(evolucionActivos)

	// Meta de colores del grupo (veces jugado + win rate por color)
	coloresMeta, _ := db.ObtenerColoresTemporada(a.DB, temporada.ID)

	// Premios de temporada, derivados del ranking (solo jugadores activos)
	ranking, _ := db.ObtenerRanking(a.DB, temporada.ID)
	var rankingActivos []models.FilaRanking
	for _, f := range ranking {
		if activos[f.Jugador.ID] {
			rankingActivos = append(rankingActivos, f)
		}
	}
	premios := construirPremios(rankingActivos)

	// Matriz head-to-head (todos contra todos) sobre los jugadores que han jugado
	matrizWins, _ := db.ObtenerMatrizH2H(a.DB, temporada.ID)
	var matrizJugadores []models.Jugador
	for _, js := range jugadoresStats {
		matrizJugadores = append(matrizJugadores, js.Jugador)
	}

	// Lista de activos para los selectores de head-to-head (los inactivos no se listan)
	var jugadoresActivos []models.Jugador
	for _, j := range jugadores {
		if j.Activo {
			jugadoresActivos = append(jugadoresActivos, j)
		}
	}

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
		"Temporada":        temporada,
		"Temporadas":       temporadas,
		"Jugadores":        jugadoresActivos,
		"JugadoresStats":   jugadoresStats,
		"DistColores":      distColores,
		"ColoresMeta":      coloresMeta,
		"Premios":          premios,
		"GraficaEvolucion": graficaEvolucion,
		"GraficaWinRate":   graficaWinRate,
		"MatrizJugadores":  matrizJugadores,
		"MatrizWins":       matrizWins,
		"H2H":              h2h,
		"J1Sel":            j1Str,
		"J2Sel":            j2Str,
		"SinDatos":         len(jugadores) == 0,
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

// PerfilJugador muestra la ficha completa de un jugador en la temporada seleccionada
func (a *AppHandlers) PerfilJugador(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	jugador, err := db.ObtenerJugador(a.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	temporada, temporadas, ok := a.resolverTemporada(r)
	if !ok {
		a.renderizar(w, "perfil_jugador.html", map[string]interface{}{
			"Jugador":  jugador,
			"SinDatos": true,
		})
		return
	}

	stats, _ := db.ObtenerEstadisticasCompletasJugador(a.DB, jugador, temporada.ID)
	historial, _ := db.ObtenerHistorialJugador(a.DB, id, temporada.ID)

	// Posición y fila en el ranking de la temporada
	ranking, _ := db.ObtenerRanking(a.DB, temporada.ID)
	var fila *models.FilaRanking
	posicion := 0
	for i := range ranking {
		if ranking[i].Jugador.ID == id {
			fila = &ranking[i]
			posicion = i + 1
			break
		}
	}

	a.renderizar(w, "perfil_jugador.html", map[string]interface{}{
		"Jugador":    jugador,
		"Temporada":  temporada,
		"Temporadas": temporadas,
		"Stats":      stats,
		"Historial":  historial,
		"Fila":       fila,
		"Posicion":   posicion,
		"SinDatos":   fila == nil && len(historial) == 0,
	})
}

// ModoTV muestra el ranking a pantalla completa para proyectarlo (auto-refresco)
func (a *AppHandlers) ModoTV(w http.ResponseWriter, r *http.Request) {
	temporada, _, ok := a.resolverTemporada(r)
	if !ok {
		a.renderizar(w, "tv.html", map[string]interface{}{"SinDatos": true})
		return
	}
	ranking, err := db.ObtenerRanking(a.DB, temporada.ID)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}
	a.renderizar(w, "tv.html", map[string]interface{}{
		"Temporada": temporada,
		"Ranking":   ranking,
		"Sesiones":  db.ContarSesionesPorTemporada(a.DB, temporada.ID),
		"SinDatos":  len(ranking) == 0,
	})
}

// ========== HELPERS ==========

// resolverTemporada elige la temporada a mostrar: la del parámetro ?temporada=ID si
// es válido; si no, la activa; y si no hay activa, la más reciente. Devuelve también
// la lista completa (para el selector) y si existe alguna temporada.
func (a *AppHandlers) resolverTemporada(r *http.Request) (models.Temporada, []models.Temporada, bool) {
	temporadas, err := db.ObtenerTemporadas(a.DB)
	if err != nil {
		log.Println("Error al obtener temporadas:", err)
	}
	if idStr := r.URL.Query().Get("temporada"); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil {
			for _, t := range temporadas {
				if t.ID == id {
					return t, temporadas, true
				}
			}
		}
	}
	if t, err := db.ObtenerTemporadaActiva(a.DB); err == nil {
		return t, temporadas, true
	}
	if len(temporadas) > 0 {
		return temporadas[0], temporadas, true
	}
	return models.Temporada{}, temporadas, false
}

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
