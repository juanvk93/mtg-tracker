// Consultas a la base de datos para el tracker de Magic — PostgreSQL
package db

import (
	"database/sql"
	"fmt"
	"mtg-tracker/internal/models"
	"sort"
	"strings"
	"time"
)

// ========== JUGADORES ==========

// ObtenerJugadores devuelve todos los jugadores ordenados por nombre
func ObtenerJugadores(db *sql.DB) ([]models.Jugador, error) {
	rows, err := db.Query(`SELECT id, nombre, color, avatar, activo FROM jugadores ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jugadores []models.Jugador
	for rows.Next() {
		var j models.Jugador
		if err := rows.Scan(&j.ID, &j.Nombre, &j.Color, &j.Avatar, &j.Activo); err != nil {
			return nil, err
		}
		jugadores = append(jugadores, j)
	}
	return jugadores, nil
}

// ObtenerJugadoresActivos devuelve solo los jugadores activos (para los formularios).
func ObtenerJugadoresActivos(db *sql.DB) ([]models.Jugador, error) {
	rows, err := db.Query(`SELECT id, nombre, color, avatar, activo FROM jugadores WHERE activo = true ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jugadores []models.Jugador
	for rows.Next() {
		var j models.Jugador
		if err := rows.Scan(&j.ID, &j.Nombre, &j.Color, &j.Avatar, &j.Activo); err != nil {
			return nil, err
		}
		jugadores = append(jugadores, j)
	}
	return jugadores, nil
}

// ObtenerJugador devuelve un jugador por ID
func ObtenerJugador(db *sql.DB, id int) (models.Jugador, error) {
	var j models.Jugador
	err := db.QueryRow(`SELECT id, nombre, color, avatar, activo FROM jugadores WHERE id = $1`, id).
		Scan(&j.ID, &j.Nombre, &j.Color, &j.Avatar, &j.Activo)
	return j, err
}

// CambiarEstadoJugador activa o desactiva un jugador
func CambiarEstadoJugador(db *sql.DB, id int, activo bool) error {
	_, err := db.Exec(`UPDATE jugadores SET activo=$1 WHERE id=$2`, activo, id)
	return err
}

// CrearJugador inserta un nuevo jugador
func CrearJugador(db *sql.DB, nombre, color, avatar string) error {
	_, err := db.Exec(`INSERT INTO jugadores (nombre, color, avatar) VALUES ($1, $2, $3)`,
		nombre, color, avatar)
	return err
}

// ActualizarJugador actualiza los datos de un jugador
func ActualizarJugador(db *sql.DB, id int, nombre, color, avatar string) error {
	_, err := db.Exec(`UPDATE jugadores SET nombre=$1, color=$2, avatar=$3 WHERE id=$4`,
		nombre, color, avatar, id)
	return err
}

// ========== TEMPORADAS ==========

// ObtenerTemporadas devuelve todas las temporadas ordenadas por año desc
func ObtenerTemporadas(db *sql.DB) ([]models.Temporada, error) {
	rows, err := db.Query(`SELECT id, anio, estado FROM temporadas ORDER BY anio DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var temporadas []models.Temporada
	for rows.Next() {
		var t models.Temporada
		if err := rows.Scan(&t.ID, &t.Anio, &t.Estado); err != nil {
			return nil, err
		}
		temporadas = append(temporadas, t)
	}
	return temporadas, nil
}

// ObtenerTemporadaActiva devuelve la temporada activa o error si no hay ninguna
func ObtenerTemporadaActiva(db *sql.DB) (models.Temporada, error) {
	var t models.Temporada
	err := db.QueryRow(`SELECT id, anio, estado FROM temporadas WHERE estado='activa' ORDER BY anio DESC LIMIT 1`).
		Scan(&t.ID, &t.Anio, &t.Estado)
	return t, err
}

// CrearTemporada crea una nueva temporada activa.
// Si ya hay otra temporada activa, la cierra automáticamente para garantizar
// que solo haya una activa a la vez.
func CrearTemporada(db *sql.DB, anio int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Cerrar cualquier temporada activa existente
	if _, err := tx.Exec(`UPDATE temporadas SET estado='cerrada' WHERE estado='activa'`); err != nil {
		return err
	}

	// Crear la nueva temporada como activa
	if _, err := tx.Exec(`INSERT INTO temporadas (anio, estado) VALUES ($1, 'activa')`, anio); err != nil {
		return err
	}

	return tx.Commit()
}

// CerrarTemporada cambia el estado de una temporada a 'cerrada'
func CerrarTemporada(db *sql.DB, id int) error {
	_, err := db.Exec(`UPDATE temporadas SET estado='cerrada' WHERE id=$1`, id)
	return err
}

// ReabrirTemporada cambia el estado de una temporada a 'activa'.
// Solo puede haber una temporada activa a la vez: si ya hay otra activa, esta función la cierra primero.
func ReabrirTemporada(db *sql.DB, id int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Cerrar cualquier otra temporada activa que no sea esta
	_, err = tx.Exec(`UPDATE temporadas SET estado='cerrada' WHERE estado='activa' AND id <> $1`, id)
	if err != nil {
		return err
	}

	// Reabrir la temporada solicitada
	_, err = tx.Exec(`UPDATE temporadas SET estado='activa' WHERE id=$1`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// BorrarTemporada elimina una temporada y todos sus datos asociados (sesiones, resultados, victorias y colores).
// Esta operación es irreversible.
func BorrarTemporada(db *sql.DB, id int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Obtener IDs de sesiones de esta temporada
	rows, err := tx.Query(`SELECT id FROM sesiones WHERE temporada_id=$1`, id)
	if err != nil {
		return err
	}
	var sesionIDs []int
	for rows.Next() {
		var sid int
		if err := rows.Scan(&sid); err != nil {
			rows.Close()
			return err
		}
		sesionIDs = append(sesionIDs, sid)
	}
	rows.Close()

	// Borrar resultados de cada sesión (las victorias y colores se borran en cascada por FK)
	for _, sid := range sesionIDs {
		if _, err := tx.Exec(`DELETE FROM resultados WHERE sesion_id=$1`, sid); err != nil {
			return err
		}
	}

	// Borrar las sesiones
	if _, err := tx.Exec(`DELETE FROM sesiones WHERE temporada_id=$1`, id); err != nil {
		return err
	}

	// Borrar la temporada
	if _, err := tx.Exec(`DELETE FROM temporadas WHERE id=$1`, id); err != nil {
		return err
	}

	return tx.Commit()
}

// ResetearTodo BORRA ABSOLUTAMENTE TODOS los datos: jugadores, temporadas, sesiones,
// resultados, victorias y colores. Las tablas quedan vacías y los IDs se reinician.
// Esta operación es irreversible y solo debe ejecutarse con confirmación de admin.
func ResetearTodo(db *sql.DB) error {
	// TRUNCATE con CASCADE borra todas las tablas dependientes y reinicia los SERIAL.
	// PostgreSQL admite múltiples tablas en un solo TRUNCATE.
	_, err := db.Exec(`TRUNCATE TABLE 
		colores_jugados, victorias, resultados, sesiones, temporadas, jugadores 
		RESTART IDENTITY CASCADE`)
	return err
}

// ========== SESIONES ==========

// ObtenerSesiones devuelve todas las sesiones de una temporada ordenadas por fecha desc
func ObtenerSesiones(db *sql.DB, temporadaID int) ([]models.Sesion, error) {
	rows, err := db.Query(`
		SELECT id, temporada_id, fecha, descripcion 
		FROM sesiones 
		WHERE temporada_id = $1 
		ORDER BY fecha DESC`, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sesiones []models.Sesion
	for rows.Next() {
		var s models.Sesion
		// PostgreSQL DATE se escanea directamente como time.Time
		if err := rows.Scan(&s.ID, &s.TemporadaID, &s.Fecha, &s.Descripcion); err != nil {
			return nil, err
		}
		sesiones = append(sesiones, s)
	}
	return sesiones, nil
}

// ObtenerTodasLasSesiones devuelve todas las sesiones ordenadas por fecha desc
func ObtenerTodasLasSesiones(db *sql.DB) ([]models.Sesion, error) {
	rows, err := db.Query(`
		SELECT id, temporada_id, fecha, descripcion 
		FROM sesiones 
		ORDER BY fecha DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sesiones []models.Sesion
	for rows.Next() {
		var s models.Sesion
		if err := rows.Scan(&s.ID, &s.TemporadaID, &s.Fecha, &s.Descripcion); err != nil {
			return nil, err
		}
		sesiones = append(sesiones, s)
	}
	return sesiones, nil
}

// ObtenerSesion devuelve una sesión por ID
func ObtenerSesion(db *sql.DB, id int) (models.Sesion, error) {
	var s models.Sesion
	err := db.QueryRow(`SELECT id, temporada_id, fecha, descripcion FROM sesiones WHERE id=$1`, id).
		Scan(&s.ID, &s.TemporadaID, &s.Fecha, &s.Descripcion)
	return s, err
}

// ActualizarDescripcionSesion cambia la descripción de una sesión existente
func ActualizarDescripcionSesion(db *sql.DB, id int, descripcion string) error {
	_, err := db.Exec(`UPDATE sesiones SET descripcion=$1 WHERE id=$2`, descripcion, id)
	return err
}

// GanadorSesion contiene info del ganador (o ganadores en caso de empate) de una sesión
type GanadorSesion struct {
	Ganadores []models.Jugador // uno o más si hay empate
	Victorias int              // victorias del ganador(es)
	Empate    bool             // true si hay empate en primer lugar
}

// ObtenerGanadorSesion calcula quién ganó una sesión (más victorias).
// Si hay empate en victorias, devuelve todos los empatados.
// Devuelve nil si la sesión no tiene resultados.
func ObtenerGanadorSesion(database *sql.DB, sesionID int) *GanadorSesion {
	rows, err := database.Query(`
		SELECT r.jugador_id, j.nombre, j.color, j.avatar, COUNT(v.id) as vict
		FROM resultados r
		JOIN jugadores j ON j.id = r.jugador_id
		LEFT JOIN victorias v ON v.resultado_id = r.id
		WHERE r.sesion_id = $1
		GROUP BY r.jugador_id, j.nombre, j.color, j.avatar
		ORDER BY vict DESC`, sesionID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	type jugadorVict struct {
		jugador   models.Jugador
		victorias int
	}
	var lista []jugadorVict
	for rows.Next() {
		var jv jugadorVict
		rows.Scan(&jv.jugador.ID, &jv.jugador.Nombre, &jv.jugador.Color, &jv.jugador.Avatar, &jv.victorias)
		lista = append(lista, jv)
	}

	if len(lista) == 0 {
		return nil
	}

	maxVict := lista[0].victorias
	if maxVict == 0 {
		return nil // nadie ganó ninguna partida
	}

	resultado := &GanadorSesion{Victorias: maxVict}
	for _, jv := range lista {
		if jv.victorias == maxVict {
			resultado.Ganadores = append(resultado.Ganadores, jv.jugador)
		}
	}
	resultado.Empate = len(resultado.Ganadores) > 1

	return resultado
}

// CrearSesion inserta una nueva sesión de draft, devuelve el ID generado
func CrearSesion(db *sql.DB, temporadaID int, fecha time.Time, descripcion string) (int64, error) {
	var id int64
	err := db.QueryRow(
		`INSERT INTO sesiones (temporada_id, fecha, descripcion) VALUES ($1, $2, $3) RETURNING id`,
		temporadaID, fecha.Format("2006-01-02"), descripcion,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ========== RESULTADOS ==========

// GuardarResultadoSesion guarda el resultado completo de un jugador en una sesión.
// Usa una transacción para asegurar consistencia.
func GuardarResultadoSesion(db *sql.DB, sesionID, jugadorID int, victorias, derrotas []int, colores []string, notas string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Eliminar resultado previo si existe (para re-edición)
	// Las tablas hijas se borran en cascada gracias a ON DELETE CASCADE
	var resultadoID int
	err = tx.QueryRow(`SELECT id FROM resultados WHERE sesion_id=$1 AND jugador_id=$2`, sesionID, jugadorID).Scan(&resultadoID)
	if err == nil {
		_, err = tx.Exec(`DELETE FROM resultados WHERE id=$1`, resultadoID)
		if err != nil {
			return err
		}
	}

	// Insertar resultado principal con RETURNING id
	var nuevoID int64
	err = tx.QueryRow(
		`INSERT INTO resultados (sesion_id, jugador_id, notas) VALUES ($1, $2, $3) RETURNING id`,
		sesionID, jugadorID, notas,
	).Scan(&nuevoID)
	if err != nil {
		return err
	}

	// Insertar victorias
	for _, rivalID := range victorias {
		if rivalID == 0 {
			continue
		}
		_, err = tx.Exec(`INSERT INTO victorias (resultado_id, rival_id) VALUES ($1, $2)`, nuevoID, rivalID)
		if err != nil {
			return err
		}
	}

	// Insertar colores jugados
	for _, color := range colores {
		if color == "" {
			continue
		}
		_, err = tx.Exec(`INSERT INTO colores_jugados (resultado_id, color) VALUES ($1, $2)`, nuevoID, color)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ObtenerDetalleSesion carga el detalle completo de una sesión con todos los
// resultados usando un número FIJO de consultas (no depende del nº de jugadores):
// sesión+temporada, participantes, todas las victorias y todos los colores. Las
// derrotas se derivan en memoria a partir de las victorias de la sesión.
func ObtenerDetalleSesion(db *sql.DB, sesionID int) (models.SesionDetalle, error) {
	var detalle models.SesionDetalle

	// 1) Sesión + temporada en una sola consulta
	err := db.QueryRow(`
		SELECT s.id, s.temporada_id, s.fecha, s.descripcion,
		       t.id, t.anio, t.estado
		FROM sesiones s
		JOIN temporadas t ON t.id = s.temporada_id
		WHERE s.id = $1`, sesionID).Scan(
		&detalle.Sesion.ID, &detalle.Sesion.TemporadaID, &detalle.Sesion.Fecha, &detalle.Sesion.Descripcion,
		&detalle.Temporada.ID, &detalle.Temporada.Anio, &detalle.Temporada.Estado)
	if err != nil {
		return detalle, err
	}

	// 2) Participantes (resultados + datos del jugador)
	rows, err := db.Query(`
		SELECT j.id, j.nombre, j.color, j.avatar, r.notas
		FROM resultados r
		JOIN jugadores j ON j.id = r.jugador_id
		WHERE r.sesion_id = $1
		ORDER BY j.nombre`, sesionID)
	if err != nil {
		return detalle, err
	}
	resultados := []models.ResultadoDetalle{}
	idxPorJugador := map[int]int{}        // jugadorID -> índice en resultados
	jugadores := map[int]models.Jugador{} // jugadorID -> jugador (para resolver rivales)
	for rows.Next() {
		var rd models.ResultadoDetalle
		if err := rows.Scan(&rd.Jugador.ID, &rd.Jugador.Nombre, &rd.Jugador.Color, &rd.Jugador.Avatar, &rd.Notas); err != nil {
			rows.Close()
			return detalle, err
		}
		idxPorJugador[rd.Jugador.ID] = len(resultados)
		jugadores[rd.Jugador.ID] = rd.Jugador
		resultados = append(resultados, rd)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return detalle, err
	}
	rows.Close()

	// 3) Todas las victorias de la sesión: ganador -> rival. De aquí salen a la vez
	//    las victorias de cada jugador y (reflejadas) las derrotas del rival.
	vRows, err := db.Query(`
		SELECT r.jugador_id AS ganador_id, v.rival_id
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		WHERE r.sesion_id = $1`, sesionID)
	if err != nil {
		return detalle, err
	}
	for vRows.Next() {
		var ganadorID, rivalID int
		if err := vRows.Scan(&ganadorID, &rivalID); err != nil {
			vRows.Close()
			return detalle, err
		}
		if gi, ok := idxPorJugador[ganadorID]; ok {
			if rival, ok := jugadores[rivalID]; ok {
				resultados[gi].Victorias = append(resultados[gi].Victorias, rival)
			}
		}
		if ri, ok := idxPorJugador[rivalID]; ok {
			if ganador, ok := jugadores[ganadorID]; ok {
				resultados[ri].Derrotas = append(resultados[ri].Derrotas, ganador)
			}
		}
	}
	if err := vRows.Err(); err != nil {
		vRows.Close()
		return detalle, err
	}
	vRows.Close()

	// 4) Colores jugados por cada participante
	cRows, err := db.Query(`
		SELECT r.jugador_id, cj.color
		FROM colores_jugados cj
		JOIN resultados r ON r.id = cj.resultado_id
		WHERE r.sesion_id = $1`, sesionID)
	if err != nil {
		return detalle, err
	}
	for cRows.Next() {
		var jugadorID int
		var color string
		if err := cRows.Scan(&jugadorID, &color); err != nil {
			cRows.Close()
			return detalle, err
		}
		if i, ok := idxPorJugador[jugadorID]; ok {
			resultados[i].Colores = append(resultados[i].Colores, color)
		}
	}
	if err := cRows.Err(); err != nil {
		cRows.Close()
		return detalle, err
	}
	cRows.Close()

	for i := range resultados {
		resultados[i].TotalVictorias = len(resultados[i].Victorias)
		resultados[i].TotalDerrotas = len(resultados[i].Derrotas)
	}
	detalle.Resultados = resultados
	return detalle, nil
}

// ========== RANKING ==========

// ObtenerRanking calcula el ranking completo de una temporada en UNA sola consulta.
// Antes se hacían ~3 consultas por jugador (patrón N+1), lento contra el pooler de
// Supabase; ahora se trae todo de golpe (victorias/derrotas por sesión) y se agrega
// en memoria. El resultado es idéntico al cálculo anterior.
func ObtenerRanking(db *sql.DB, temporadaID int) ([]models.FilaRanking, error) {
	rows, err := db.Query(`
		SELECT
			j.id, j.nombre, j.color, j.avatar,
			s.fecha,
			(SELECT COUNT(*) FROM victorias v
				WHERE v.resultado_id = r.id) AS vict,
			(SELECT COUNT(*) FROM victorias v2
				JOIN resultados r2 ON r2.id = v2.resultado_id
				WHERE r2.sesion_id = r.sesion_id AND v2.rival_id = r.jugador_id) AS derr
		FROM resultados r
		JOIN sesiones s ON s.id = r.sesion_id
		JOIN jugadores j ON j.id = r.jugador_id
		WHERE s.temporada_id = $1
		ORDER BY j.nombre, s.fecha DESC`, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Acumular por jugador preservando el orden de aparición. El historial de cada
	// jugador queda en orden de fecha descendente, justo lo que pide calcularRacha.
	type acumulado struct {
		fila      models.FilaRanking
		historial []sesionVD
	}
	orden := []int{}
	porJugador := map[int]*acumulado{}

	for rows.Next() {
		var (
			id                    int
			nombre, color, avatar string
			fecha                 time.Time
			vict, derr            int
		)
		if err := rows.Scan(&id, &nombre, &color, &avatar, &fecha, &vict, &derr); err != nil {
			return nil, err
		}
		a, ok := porJugador[id]
		if !ok {
			a = &acumulado{fila: models.FilaRanking{
				Jugador: models.Jugador{ID: id, Nombre: nombre, Color: color, Avatar: avatar},
			}}
			porJugador[id] = a
			orden = append(orden, id)
		}
		a.fila.Victorias += vict
		a.fila.Derrotas += derr
		a.historial = append(a.historial, sesionVD{Victorias: vict, Derrotas: derr})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ranking := make([]models.FilaRanking, 0, len(orden))
	for _, id := range orden {
		a := porJugador[id]
		a.fila.Partidas = a.fila.Victorias + a.fila.Derrotas
		if a.fila.Partidas > 0 {
			a.fila.WinRate = float64(a.fila.Victorias) / float64(a.fila.Partidas) * 100
		}
		a.fila.RachaActual, a.fila.RachaTipo = calcularRacha(a.historial)
		a.fila.MejorRacha = calcularMejorRacha(a.historial)
		ranking = append(ranking, a.fila)
	}

	// Ordenar por victorias desc, luego win rate desc (estable → desempates deterministas por nombre)
	sort.SliceStable(ranking, func(i, k int) bool {
		if ranking[i].Victorias != ranking[k].Victorias {
			return ranking[i].Victorias > ranking[k].Victorias
		}
		return ranking[i].WinRate > ranking[k].WinRate
	})

	// Colores distintos jugados por cada jugador (para los puntos de maná de la tabla)
	if colRows, err := db.Query(`
		SELECT r.jugador_id, cj.color
		FROM colores_jugados cj
		JOIN resultados r ON r.id = cj.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE s.temporada_id = $1
		GROUP BY r.jugador_id, cj.color`, temporadaID); err == nil {
		coloresJug := map[int][]string{}
		for colRows.Next() {
			var jid int
			var c string
			if err := colRows.Scan(&jid, &c); err == nil {
				coloresJug[jid] = append(coloresJug[jid], c)
			}
		}
		colRows.Close()
		ordenC := map[string]int{"W": 0, "U": 1, "B": 2, "R": 3, "G": 4}
		for i := range ranking {
			cs := coloresJug[ranking[i].Jugador.ID]
			sort.Slice(cs, func(a, b int) bool { return ordenC[cs[a]] < ordenC[cs[b]] })
			ranking[i].Colores = cs
		}
	}

	return ranking, nil
}

// sesionVD resume las victorias y derrotas de un jugador en una sola sesión.
type sesionVD struct {
	Victorias int
	Derrotas  int
}

// calcularRacha calcula la racha actual a partir del historial de sesiones, que debe
// venir ordenado de la más reciente a la más antigua. Devuelve la longitud de la racha
// y su tipo ("W" victorias, "L" derrotas, "" si la última sesión fue empate).
func calcularRacha(historial []sesionVD) (int, string) {
	if len(historial) == 0 {
		return 0, ""
	}
	ultima := historial[0]

	switch {
	case ultima.Victorias > ultima.Derrotas:
		n := 1
		for i := 1; i < len(historial); i++ {
			if historial[i].Victorias > historial[i].Derrotas {
				n++
			} else {
				break
			}
		}
		return n, "W"
	case ultima.Derrotas > ultima.Victorias:
		n := 1
		for i := 1; i < len(historial); i++ {
			if historial[i].Derrotas > historial[i].Victorias {
				n++
			} else {
				break
			}
		}
		return n, "L"
	default:
		return 0, ""
	}
}

// calcularMejorRacha devuelve la racha de victorias más larga del historial (número
// máximo de sesiones consecutivas ganadas, en cualquier orden del historial).
func calcularMejorRacha(historial []sesionVD) int {
	mejor, actual := 0, 0
	for _, s := range historial {
		if s.Victorias > s.Derrotas {
			actual++
			if actual > mejor {
				mejor = actual
			}
		} else {
			actual = 0
		}
	}
	return mejor
}

// ========== ESTADÍSTICAS ==========

// ObtenerEstadisticasColores devuelve, por color, las estadísticas de un jugador en la
// temporada, en UNA sola consulta (antes ~10 por jugador). Veces = nº de drafts con ese
// color; Partidas = partidas individuales (V+D); Win rate = V/(V+D)×100.
func ObtenerEstadisticasColores(db *sql.DB, jugadorID, temporadaID int) ([]models.EstadisticaColor, error) {
	rows, err := db.Query(`
		SELECT
			cj.color,
			COUNT(DISTINCT r.id)      AS veces,
			COALESCE(SUM(pj.vict), 0) AS victorias,
			COALESCE(SUM(pj.derr), 0) AS derrotas
		FROM colores_jugados cj
		JOIN resultados r ON r.id = cj.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		JOIN (
			SELECT r2.id,
				(SELECT COUNT(*) FROM victorias v
					WHERE v.resultado_id = r2.id) AS vict,
				(SELECT COUNT(*) FROM victorias v2
					JOIN resultados r3 ON r3.id = v2.resultado_id
					WHERE r3.sesion_id = r2.sesion_id AND v2.rival_id = r2.jugador_id) AS derr
			FROM resultados r2
		) pj ON pj.id = r.id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2
		GROUP BY cj.color`, jugadorID, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Ordenar en el orden canónico WUBRG
	orden := map[string]int{"W": 0, "U": 1, "B": 2, "R": 3, "G": 4}
	var stats []models.EstadisticaColor
	for rows.Next() {
		var ec models.EstadisticaColor
		var derrotas int
		if err := rows.Scan(&ec.Color, &ec.Veces, &ec.Victorias, &derrotas); err != nil {
			return nil, err
		}
		ec.Partidas = ec.Victorias + derrotas
		if ec.Partidas == 0 {
			continue // color usado pero sin partidas registradas (comportamiento previo)
		}
		ec.Nombre = models.NombreColor(ec.Color)
		ec.WinRate = float64(ec.Victorias) / float64(ec.Partidas) * 100
		stats = append(stats, ec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(stats, func(i, j int) bool { return orden[stats[i].Color] < orden[stats[j].Color] })
	return stats, nil
}

// ObtenerHeadToHead devuelve el historial entre dos jugadores en toda la historia
func ObtenerHeadToHead(db *sql.DB, jugador1ID, jugador2ID int) (models.HeadToHead, error) {
	var h2h models.HeadToHead
	var err error

	h2h.Jugador1, err = ObtenerJugador(db, jugador1ID)
	if err != nil {
		return h2h, err
	}
	h2h.Jugador2, err = ObtenerJugador(db, jugador2ID)
	if err != nil {
		return h2h, err
	}

	// Victorias de J1 sobre J2
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		WHERE r.jugador_id = $1 AND v.rival_id = $2`, jugador1ID, jugador2ID).Scan(&h2h.VictoriasJ1)
	if err != nil {
		return h2h, err
	}

	// Victorias de J2 sobre J1
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		WHERE r.jugador_id = $1 AND v.rival_id = $2`, jugador2ID, jugador1ID).Scan(&h2h.VictoriasJ2)
	if err != nil {
		return h2h, err
	}

	h2h.Total = h2h.VictoriasJ1 + h2h.VictoriasJ2
	return h2h, nil
}

// ObtenerColorMasWinRate devuelve el color con mejor win rate del grupo en la temporada activa.
// Para cada color: suma las victorias y derrotas de todos los jugadores que lo usaron.
// Win rate = victorias totales / (victorias + derrotas totales) × 100.
func ObtenerColorMasWinRate(db *sql.DB, temporadaID int) (string, float64) {
	colores := []string{"W", "U", "B", "R", "G"}
	mejorColor := ""
	mejorWR := -1.0

	for _, color := range colores {
		// Obtener todos los resultados donde se jugó este color en la temporada
		rows, err := db.Query(`
			SELECT r.id, r.jugador_id, r.sesion_id
			FROM resultados r
			JOIN colores_jugados cj ON cj.resultado_id = r.id
			JOIN sesiones s ON s.id = r.sesion_id
			WHERE cj.color = $1 AND s.temporada_id = $2`, color, temporadaID)
		if err != nil {
			continue
		}

		type par struct{ resultadoID, jugadorID, sesionID int }
		var pares []par
		for rows.Next() {
			var p par
			rows.Scan(&p.resultadoID, &p.jugadorID, &p.sesionID)
			pares = append(pares, p)
		}
		rows.Close()

		if len(pares) < 2 { // mínimo 2 jugadores usando el color
			continue
		}

		var victorias, derrotas int
		for _, p := range pares {
			// Victorias de este jugador en esta sesión
			var v int
			db.QueryRow(`SELECT COUNT(*) FROM victorias WHERE resultado_id=$1`, p.resultadoID).Scan(&v)
			victorias += v

			// Derrotas de este jugador en esta sesión
			var d int
			db.QueryRow(`
				SELECT COUNT(*)
				FROM victorias v2
				JOIN resultados r2 ON r2.id = v2.resultado_id
				WHERE r2.sesion_id = $1 AND v2.rival_id = $2`,
				p.sesionID, p.jugadorID).Scan(&d)
			derrotas += d
		}

		total := victorias + derrotas
		if total < 4 {
			continue
		}

		wr := float64(victorias) / float64(total) * 100
		if wr > mejorWR {
			mejorWR = wr
			mejorColor = color
		}
	}

	return mejorColor, mejorWR
}

// ObtenerColorMasJugado devuelve el color más frecuente del grupo en la temporada
func ObtenerColorMasJugado(db *sql.DB, temporadaID int) string {
	colores := []string{"W", "U", "B", "R", "G"}
	mejorColor := ""
	maxUsos := 0

	for _, color := range colores {
		var usos int
		db.QueryRow(`
			SELECT COUNT(cj.id)
			FROM colores_jugados cj
			JOIN resultados r ON r.id = cj.resultado_id
			JOIN sesiones s ON s.id = r.sesion_id
			WHERE cj.color = $1 AND s.temporada_id = $2`, color, temporadaID).Scan(&usos)

		if usos > maxUsos {
			maxUsos = usos
			mejorColor = color
		}
	}

	return mejorColor
}

// ContarSesionesPorTemporada devuelve el número de sesiones de una temporada
func ContarSesionesPorTemporada(db *sql.DB, temporadaID int) int {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sesiones WHERE temporada_id=$1`, temporadaID).Scan(&count)
	return count
}

// ObtenerJugadoresEnSesion devuelve los IDs de jugadores con resultado en una sesión
func ObtenerJugadoresEnSesion(db *sql.DB, sesionID int) ([]int, error) {
	rows, err := db.Query(`SELECT jugador_id FROM resultados WHERE sesion_id=$1`, sesionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

// EliminarResultadosSesion borra todos los resultados de una sesión (cascada borra victorias y colores)
func EliminarResultadosSesion(db *sql.DB, sesionID int) error {
	_, err := db.Exec(`DELETE FROM resultados WHERE sesion_id=$1`, sesionID)
	return err
}

// FormatearColores convierte slice de colores a string separado por coma
func FormatearColores(colores []string) string {
	return strings.Join(colores, ",")
}

// ColorHex devuelve el color hex de un símbolo de Magic
func ColorHex(codigo string) string {
	colores := map[string]string{
		"W": "#f8f4e8",
		"U": "#3a8fc9",
		"B": "#2d2d2d",
		"R": "#e63946",
		"G": "#2a9d5c",
	}
	if c, ok := colores[codigo]; ok {
		return c
	}
	return "#888"
}

// ObtenerTemporada devuelve una temporada por ID
func ObtenerTemporada(db *sql.DB, id int) (models.Temporada, error) {
	var t models.Temporada
	err := db.QueryRow(`SELECT id, anio, estado FROM temporadas WHERE id=$1`, id).
		Scan(&t.ID, &t.Anio, &t.Estado)
	if err != nil {
		return t, fmt.Errorf("temporada %d: %w", id, err)
	}
	return t, nil
}
