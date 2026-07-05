// Handlers de administración del tracker
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mtg-tracker/internal/db"
	"mtg-tracker/internal/models"
)

// ========== AUTENTICACIÓN ADMIN ==========

// MiddlewareAdmin verifica la contraseña de admin en la sesión de cookie
func MiddlewareAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("admin_auth")
		if err != nil || cookie.Value != passwordAdmin() {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// passwordAdmin devuelve la contraseña de admin desde la variable de entorno
func passwordAdmin() string {
	p := os.Getenv("ADMIN_PASSWORD")
	if p == "" {
		return "admin" // fallback solo para desarrollo
	}
	return p
}

// AdminLogin muestra el formulario de login
func (a *AppHandlers) AdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		password := r.FormValue("password")
		if password == passwordAdmin() {
			http.SetCookie(w, &http.Cookie{
				Name:     "admin_auth",
				Value:    password,
				Path:     "/",
				HttpOnly: true,
				MaxAge:   86400 * 30, // 30 días
			})
			http.Redirect(w, r, "/admin", http.StatusFound)
			return
		}
		a.renderizar(w, "login.html", map[string]interface{}{"Error": "Contraseña incorrecta"})
		return
	}
	a.renderizar(w, "login.html", nil)
}

// AdminLogout cierra la sesión de admin
func (a *AppHandlers) AdminLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "admin_auth",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// AdminPanel muestra el panel principal de administración
func (a *AppHandlers) AdminPanel(w http.ResponseWriter, r *http.Request) {
	temporadas, err := db.ObtenerTemporadas(a.DB)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	jugadores, err := db.ObtenerJugadores(a.DB)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	temporada, _ := db.ObtenerTemporadaActiva(a.DB)

	var sesiones []models.Sesion
	if temporada.ID != 0 {
		sesiones, _ = db.ObtenerSesiones(a.DB, temporada.ID)
	}

	a.renderizar(w, "panel.html", map[string]interface{}{
		"Temporadas":      temporadas,
		"TemporadaActiva": temporada,
		"Jugadores":       jugadores,
		"Sesiones":        sesiones,
		"AnioActual":      time.Now().Year(),
	})
}

// ========== TEMPORADAS ==========

// AdminCrearTemporada crea una nueva temporada
func (a *AppHandlers) AdminCrearTemporada(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	anioStr := r.FormValue("anio")
	anio, err := strconv.Atoi(anioStr)
	if err != nil || anio < 2000 || anio > 2100 {
		http.Error(w, "Año inválido", http.StatusBadRequest)
		return
	}

	if err := db.CrearTemporada(a.DB, anio); err != nil {
		log.Println("Error al crear temporada:", err)
		http.Error(w, "Error al crear temporada: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

// AdminCerrarTemporada cambia el estado de una temporada a cerrada
func (a *AppHandlers) AdminCerrarTemporada(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	if err := db.CerrarTemporada(a.DB, id); err != nil {
		http.Error(w, "Error al cerrar temporada", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

// AdminReabrirTemporada cambia el estado de una temporada a activa
func (a *AppHandlers) AdminReabrirTemporada(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	if err := db.ReabrirTemporada(a.DB, id); err != nil {
		log.Println("Error al reabrir temporada:", err)
		http.Error(w, "Error al reabrir temporada", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

// AdminBorrarTemporada elimina una temporada y todos sus datos.
// Requiere re-confirmación de la contraseña de admin como medida de seguridad.
func (a *AppHandlers) AdminBorrarTemporada(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	// Verificación obligatoria de contraseña antes de borrar
	password := r.FormValue("password")
	if password != passwordAdmin() {
		http.Error(w, "Contraseña incorrecta. La operación de borrado se ha cancelado.", http.StatusUnauthorized)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	if err := db.BorrarTemporada(a.DB, id); err != nil {
		log.Println("Error al borrar temporada:", err)
		http.Error(w, "Error al borrar la temporada: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

// AdminFarewell BORRA TODOS LOS DATOS de la aplicación.
// Requiere doble confirmación: la contraseña de admin y el texto exacto "FAREWELL".
// Esta operación es irreversible.
func (a *AppHandlers) AdminFarewell(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	// Confirmación 1: contraseña de admin
	password := r.FormValue("password")
	if password != passwordAdmin() {
		http.Error(w, "Contraseña incorrecta. La operación se ha cancelado.", http.StatusUnauthorized)
		return
	}

	// Confirmación 2: texto literal de seguridad
	confirmacion := strings.TrimSpace(r.FormValue("confirmacion"))
	if confirmacion != "FAREWELL" {
		http.Error(w, "Texto de confirmación incorrecto. Debe ser exactamente 'FAREWELL'.", http.StatusBadRequest)
		return
	}

	if err := db.ResetearTodo(a.DB); err != nil {
		log.Println("Error en farewell:", err)
		http.Error(w, "Error al resetear los datos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("⚠️  FAREWELL ejecutado: todos los datos han sido borrados")
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// ========== SESIONES ==========

// AdminCrearSesion crea una nueva sesión de draft
func (a *AppHandlers) AdminCrearSesion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	temporadaIDStr := r.FormValue("temporada_id")
	temporadaID, err := strconv.Atoi(temporadaIDStr)
	if err != nil {
		http.Error(w, "Temporada inválida", http.StatusBadRequest)
		return
	}

	fechaStr := r.FormValue("fecha")
	fecha, err := time.Parse("2006-01-02", fechaStr)
	if err != nil {
		http.Error(w, "Fecha inválida", http.StatusBadRequest)
		return
	}

	descripcion := r.FormValue("descripcion")

	sesionID, err := db.CrearSesion(a.DB, temporadaID, fecha, descripcion)
	if err != nil {
		log.Println("Error al crear sesión:", err)
		http.Error(w, "Error al crear sesión", http.StatusInternalServerError)
		return
	}

	// Redirigir directamente a registrar resultados
	http.Redirect(w, r, fmt.Sprintf("/admin/sesion/%d/resultados", sesionID), http.StatusFound)
}

// AdminFormResultados muestra el formulario para registrar resultados de una sesión
func (a *AppHandlers) AdminFormResultados(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	sesion, err := db.ObtenerSesion(a.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	jugadores, err := db.ObtenerJugadores(a.DB)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	// Cargar resultados existentes para pre-rellenar el formulario
	detalle, _ := db.ObtenerDetalleSesion(a.DB, id)

	a.renderizar(w, "resultados.html", map[string]interface{}{
		"Sesion":    sesion,
		"Jugadores": jugadores,
		"Detalle":   detalle,
		"Colores":   []string{"W", "U", "B", "R", "G"},
	})
}

// AdminGuardarResultados guarda los resultados de una sesión
func (a *AppHandlers) AdminGuardarResultados(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	idStr := r.PathValue("id")
	sesionID, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al parsear formulario", http.StatusBadRequest)
		return
	}

	// Actualizar descripción de la sesión si cambió
	descripcion := r.FormValue("descripcion")
	db.ActualizarDescripcionSesion(a.DB, sesionID, descripcion)

	// Obtener los jugadores seleccionados para esta sesión
	jugadoresStr := r.FormValue("jugadores_sesion")
	if jugadoresStr == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/sesion/%d/resultados", sesionID), http.StatusFound)
		return
	}

	jugadoresSesion := strings.Split(jugadoresStr, ",")

	// Construir el conjunto de IDs de jugadores que van a estar en la sesión
	jugadoresNuevos := make(map[int]bool)
	for _, jugIDStr := range jugadoresSesion {
		jugIDStr = strings.TrimSpace(jugIDStr)
		if jugID, err := strconv.Atoi(jugIDStr); err == nil && jugID > 0 {
			jugadoresNuevos[jugID] = true
		}
	}

	// Borrar solo los resultados de jugadores que YA NO están en la sesión
	// (los que sí están se actualizarán individualmente en GuardarResultadoSesion)
	jugadoresAnteriores, _ := db.ObtenerJugadoresEnSesion(a.DB, sesionID)
	for _, jugID := range jugadoresAnteriores {
		if !jugadoresNuevos[jugID] {
			// Este jugador ya no juega: borrar su resultado
			a.DB.Exec(`DELETE FROM resultados WHERE sesion_id=$1 AND jugador_id=$2`, sesionID, jugID)
		}
	}

	// Guardar/actualizar resultado de cada jugador activo
	for jugID := range jugadoresNuevos {
		victoriasKey := fmt.Sprintf("victoria_%d", jugID)
		victoriasStrs := r.Form[victoriasKey]
		var victorias []int
		for _, vs := range victoriasStrs {
			if id, err := strconv.Atoi(vs); err == nil {
				victorias = append(victorias, id)
			}
		}

		coloresKey := fmt.Sprintf("color_%d", jugID)
		colores := r.Form[coloresKey]

		if err := db.GuardarResultadoSesion(a.DB, sesionID, jugID, victorias, nil, colores); err != nil {
			log.Printf("Error al guardar resultado del jugador %d: %v", jugID, err)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/sesion/%d", sesionID), http.StatusFound)
}

// ========== JUGADORES ==========

// AdminCrearJugador crea un nuevo jugador
func (a *AppHandlers) AdminCrearJugador(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	nombre := strings.TrimSpace(r.FormValue("nombre"))
	color := r.FormValue("color")
	avatar := strings.TrimSpace(r.FormValue("avatar"))

	if nombre == "" {
		http.Error(w, "El nombre es obligatorio", http.StatusBadRequest)
		return
	}
	if color == "" {
		color = "#6c757d"
	}
	if avatar == "" {
		avatar = "ms-planeswalker"
	}

	if err := db.CrearJugador(a.DB, nombre, color, avatar); err != nil {
		log.Println("Error al crear jugador:", err)
		http.Error(w, "Error al crear jugador: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

// AdminEditarJugadorForm muestra el formulario de edición de un jugador
func (a *AppHandlers) AdminEditarJugadorForm(w http.ResponseWriter, r *http.Request) {
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

	a.renderizar(w, "editar_jugador.html", map[string]interface{}{
		"Jugador": jugador,
	})
}

// AdminActualizarJugador actualiza los datos de un jugador
func (a *AppHandlers) AdminActualizarJugador(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	nombre := strings.TrimSpace(r.FormValue("nombre"))
	color := r.FormValue("color")
	avatar := strings.TrimSpace(r.FormValue("avatar"))

	if nombre == "" {
		http.Error(w, "El nombre es obligatorio", http.StatusBadRequest)
		return
	}

	if err := db.ActualizarJugador(a.DB, id, nombre, color, avatar); err != nil {
		http.Error(w, "Error al actualizar jugador", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}
