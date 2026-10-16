// Handlers de administración del tracker
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"mtg-tracker/internal/db"
	"mtg-tracker/internal/models"
)

// ========== AUTENTICACIÓN ADMIN ==========

// MiddlewareAdmin verifica la contraseña de admin en la sesión de cookie y, en las
// peticiones que modifican datos, que vengan de la propia web (protección CSRF).
func MiddlewareAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("admin_auth")
		if err != nil || cookie.Value != passwordAdmin() {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && !mismoOrigen(r) {
			http.Error(w, "Petición rechazada: no procede de esta web", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// mismoOrigen comprueba que un POST lo envía un formulario de esta misma web. Los
// navegadores ponen Origin (o al menos Referer) en los POST; si una web ajena intenta
// enviar un formulario con la cookie del admin, su origen no coincide con nuestro Host.
// Sin ninguna de las dos cabeceras no es un navegador (p. ej. curl) y no hay CSRF posible.
func mismoOrigen(r *http.Request) bool {
	origen := r.Header.Get("Origin")
	if origen == "" {
		origen = r.Header.Get("Referer")
	}
	if origen == "" {
		return true
	}
	u, err := url.Parse(origen)
	if err != nil || u.Host == "" {
		return false // incluye Origin: null
	}
	// Hosts válidos: el de la petición y, por si el proxy de Render lo reescribe, el
	// reenviado y el de la URL pública que define Render
	validos := []string{r.Host, r.Header.Get("X-Forwarded-Host")}
	if pub, err := url.Parse(os.Getenv("RENDER_EXTERNAL_URL")); err == nil {
		validos = append(validos, pub.Host)
	}
	for _, h := range validos {
		if h != "" && strings.EqualFold(u.Host, h) {
			return true
		}
	}
	return false
}

// cookieAdmin construye la cookie de sesión admin. Secure solo cuando la petición llega
// por HTTPS (Render pone X-Forwarded-Proto), para no romper el login en local por http.
// SameSite=Lax hace que el navegador no la envíe en POST lanzados desde otras webs.
func cookieAdmin(r *http.Request, valor string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "admin_auth",
		Value:    valor,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
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
			http.SetCookie(w, cookieAdmin(r, password, 86400*30)) // 30 días
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
	http.SetCookie(w, cookieAdmin(r, "", -1))
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
	enfrentamientoUnico := r.FormValue("enfrentamiento_unico") == "on"

	sesionID, err := db.CrearSesion(a.DB, temporadaID, fecha, descripcion, enfrentamientoUnico)
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

	// Cargar resultados existentes para pre-rellenar el formulario
	detalle, _ := db.ObtenerDetalleSesion(a.DB, id)

	// La lista del formulario = jugadores activos + los que ya participaron en ESTA sesión
	// (aunque estén inactivos), para poder seguir editando sesiones antiguas.
	activos, err := db.ObtenerJugadoresActivos(a.DB)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}
	enLista := map[int]bool{}
	var jugadores []models.Jugador
	for _, j := range activos {
		enLista[j.ID] = true
		jugadores = append(jugadores, j)
	}
	for _, rd := range detalle.Resultados {
		if !enLista[rd.Jugador.ID] {
			enLista[rd.Jugador.ID] = true
			jugadores = append(jugadores, rd.Jugador)
		}
	}
	sort.Slice(jugadores, func(i, j int) bool { return jugadores[i].Nombre < jugadores[j].Nombre })

	// Mapas de pre-relleno para el modo matriz (y consistencia con la vista detallada)
	participa := map[int]bool{}
	victMap := map[int]map[int]bool{}
	colorMap := map[int]map[string]bool{}
	notasMap := map[int]string{}
	for _, j := range jugadores {
		victMap[j.ID] = map[int]bool{}
		colorMap[j.ID] = map[string]bool{}
	}
	for _, rd := range detalle.Resultados {
		participa[rd.Jugador.ID] = true
		notasMap[rd.Jugador.ID] = rd.Notas
		if m, ok := victMap[rd.Jugador.ID]; ok {
			for _, v := range rd.Victorias {
				m[v.ID] = true
			}
		}
		if m, ok := colorMap[rd.Jugador.ID]; ok {
			for _, c := range rd.Colores {
				m[c] = true
			}
		}
	}

	a.renderizar(w, "resultados.html", map[string]interface{}{
		"Sesion":        sesion,
		"Jugadores":     jugadores,
		"Detalle":       detalle,
		"Colores":       []string{"W", "U", "B", "R", "G"},
		"Participa":     participa,
		"VictMap":       victMap,
		"ColorMap":      colorMap,
		"NotasMap":      notasMap,
		"SinResultados": len(detalle.Resultados) == 0,
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
	if _, err := db.ObtenerSesion(a.DB, sesionID); err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al parsear formulario", http.StatusBadRequest)
		return
	}

	descripcion := r.FormValue("descripcion")
	enfrentamientoUnico := r.FormValue("enfrentamiento_unico") == "on"

	// Obtener los jugadores seleccionados para esta sesión
	jugadoresStr := r.FormValue("jugadores_sesion")
	if jugadoresStr == "" && r.FormValue("vaciar") == "1" {
		// Sin jugadores y confirmado en el formulario: se borran todos los resultados
		if err := db.GuardarResultadosSesion(a.DB, sesionID, descripcion, enfrentamientoUnico, nil); err != nil {
			log.Printf("Error al vaciar la sesión %d: %v", sesionID, err)
			http.Error(w, "No se pudo vaciar la sesión (no se ha modificado nada).", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/admin/sesion/%d/resultados", sesionID), http.StatusFound)
		return
	}
	if jugadoresStr == "" {
		// Sin jugadores y sin confirmar el vaciado: solo se guarda la descripción
		if err := db.ActualizarDescripcionSesion(a.DB, sesionID, descripcion, enfrentamientoUnico); err != nil {
			log.Println("Error al actualizar la sesión:", err)
			http.Error(w, "Error al guardar la sesión", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/admin/sesion/%d/resultados", sesionID), http.StatusFound)
		return
	}

	// Construir el conjunto de IDs de jugadores que van a estar en la sesión
	jugadoresNuevos := make(map[int]bool)
	for _, jugIDStr := range strings.Split(jugadoresStr, ",") {
		if jugID, err := strconv.Atoi(strings.TrimSpace(jugIDStr)); err == nil && jugID > 0 {
			jugadoresNuevos[jugID] = true
		}
	}

	// En modo "cada pareja se enfrenta una vez" no puede haber victorias cruzadas
	// (A venció a B y B venció a A). El JS ya lo impide; esto es la red de seguridad.
	// Se valida ANTES de escribir nada, para que un envío rechazado no deje cambios a medias.
	if enfrentamientoUnico {
		if a1, a2, ok := victoriaCruzada(r.Form, jugadoresNuevos); ok {
			j1, _ := db.ObtenerJugador(a.DB, a1)
			j2, _ := db.ObtenerJugador(a.DB, a2)
			http.Error(w, fmt.Sprintf("Cada pareja se enfrenta una sola vez: %s y %s no pueden haberse ganado mutuamente. Vuelve atrás y corrige el formulario.", j1.Nombre, j2.Nombre), http.StatusBadRequest)
			return
		}
	}

	resultados := make([]db.ResultadoJugador, 0, len(jugadoresNuevos))
	for jugID := range jugadoresNuevos {
		resultados = append(resultados, db.ResultadoJugador{
			JugadorID: jugID,
			Victorias: victoriasValidas(r.Form[fmt.Sprintf("victoria_%d", jugID)], jugID, jugadoresNuevos),
			Colores:   coloresValidos(r.Form[fmt.Sprintf("color_%d", jugID)]),
			Notas:     strings.TrimSpace(r.FormValue(fmt.Sprintf("notas_%d", jugID))),
		})
	}

	// Toda la sesión en una transacción: si algo falla no se guarda nada
	if err := db.GuardarResultadosSesion(a.DB, sesionID, descripcion, enfrentamientoUnico, resultados); err != nil {
		log.Printf("Error al guardar los resultados de la sesión %d: %v", sesionID, err)
		http.Error(w, "No se pudieron guardar los resultados (no se ha modificado nada). Vuelve atrás e inténtalo de nuevo.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/sesion/%d", sesionID), http.StatusFound)
}

// victoriasValidas filtra los rivales enviados: solo jugadores que participan en la
// sesión, nunca uno mismo y sin repetidos. Una victoria sobre alguien que no juega
// sumaría una V sin la D correspondiente y descuadraría el ranking.
func victoriasValidas(valores []string, jugID int, enSesion map[int]bool) []int {
	vistos := map[int]bool{}
	var victorias []int
	for _, v := range valores {
		rival, err := strconv.Atoi(v)
		if err != nil || rival == jugID || !enSesion[rival] || vistos[rival] {
			continue
		}
		vistos[rival] = true
		victorias = append(victorias, rival)
	}
	return victorias
}

// coloresValidos deja solo códigos WUBRG, sin repetidos
func coloresValidos(valores []string) []string {
	vistos := map[string]bool{}
	var colores []string
	for _, c := range valores {
		if !strings.Contains("WUBRG", c) || len(c) != 1 || vistos[c] {
			continue
		}
		vistos[c] = true
		colores = append(colores, c)
	}
	return colores
}

// victoriaCruzada busca una pareja de jugadores de la sesión que se haya ganado
// mutuamente en el formulario (victoria_A contiene B y victoria_B contiene A).
func victoriaCruzada(form url.Values, jugadores map[int]bool) (int, int, bool) {
	gano := map[[2]int]bool{}
	for jugID := range jugadores {
		for _, vs := range form[fmt.Sprintf("victoria_%d", jugID)] {
			if rival, err := strconv.Atoi(vs); err == nil && jugadores[rival] {
				gano[[2]int{jugID, rival}] = true
			}
		}
	}
	for par := range gano {
		if par[0] < par[1] && gano[[2]int{par[1], par[0]}] {
			return par[0], par[1], true
		}
	}
	return 0, 0, false
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
	avatar = models.NormalizarAvatar(avatar)

	if err := db.CrearJugador(a.DB, nombre, color, avatar); err != nil {
		log.Println("Error al crear jugador:", err)
		http.Error(w, "Error al crear jugador: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

// AdminToggleJugador activa o desactiva un jugador (mantiene su histórico y lo saca
// de los formularios de resultados). Recibe el nuevo estado en el campo `activo`.
func (a *AppHandlers) AdminToggleJugador(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	activo := r.FormValue("activo") == "true"
	if err := db.CambiarEstadoJugador(a.DB, id, activo); err != nil {
		log.Println("Error al cambiar estado del jugador:", err)
		http.Error(w, "Error al cambiar estado del jugador", http.StatusInternalServerError)
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

	avatar = models.NormalizarAvatar(avatar)

	if err := db.ActualizarJugador(a.DB, id, nombre, color, avatar); err != nil {
		http.Error(w, "Error al actualizar jugador", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}
