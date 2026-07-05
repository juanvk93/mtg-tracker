// Consultas a la base de datos para el tracker de Magic — PostgreSQL
package db

import (
	"database/sql"
	"fmt"
	"mtg-tracker/internal/models"
	"strings"
	"time"
)

// ========== JUGADORES ==========

// ObtenerJugadores devuelve todos los jugadores ordenados por nombre
func ObtenerJugadores(db *sql.DB) ([]models.Jugador, error) {
	rows, err := db.Query(`SELECT id, nombre, color, avatar FROM jugadores ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jugadores []models.Jugador
	for rows.Next() {
		var j models.Jugador
		if err := rows.Scan(&j.ID, &j.Nombre, &j.Color, &j.Avatar); err != nil {
			return nil, err
		}
		jugadores = append(jugadores, j)
	}
	return jugadores, nil
}

// ObtenerJugador devuelve un jugador por ID
func ObtenerJugador(db *sql.DB, id int) (models.Jugador, error) {
	var j models.Jugador
	err := db.QueryRow(`SELECT id, nombre, color, avatar FROM jugadores WHERE id = $1`, id).
		Scan(&j.ID, &j.Nombre, &j.Color, &j.Avatar)
	return j, err
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
func GuardarResultadoSesion(db *sql.DB, sesionID, jugadorID int, victorias, derrotas []int, colores []string) error {
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
		`INSERT INTO resultados (sesion_id, jugador_id) VALUES ($1, $2) RETURNING id`,
		sesionID, jugadorID,
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

// ObtenerDetalleSesion carga el detalle completo de una sesión con todos los resultados
func ObtenerDetalleSesion(db *sql.DB, sesionID int) (models.SesionDetalle, error) {
	var detalle models.SesionDetalle

	// Cargar sesión
	sesion, err := ObtenerSesion(db, sesionID)
	if err != nil {
		return detalle, err
	}
	detalle.Sesion = sesion

	// Cargar temporada
	err = db.QueryRow(`SELECT id, anio, estado FROM temporadas WHERE id=$1`, sesion.TemporadaID).
		Scan(&detalle.Temporada.ID, &detalle.Temporada.Anio, &detalle.Temporada.Estado)
	if err != nil {
		return detalle, err
	}

	// Cargar resultados de la sesión
	rows, err := db.Query(`
		SELECT r.id, r.jugador_id, j.nombre, j.color, j.avatar
		FROM resultados r
		JOIN jugadores j ON j.id = r.jugador_id
		WHERE r.sesion_id = $1
		ORDER BY j.nombre`, sesionID)
	if err != nil {
		return detalle, err
	}
	defer rows.Close()

	// Mapa de todos los jugadores para resolver IDs
	todosJugadores, err := ObtenerJugadores(db)
	if err != nil {
		return detalle, err
	}
	mapaJugadores := make(map[int]models.Jugador)
	for _, j := range todosJugadores {
		mapaJugadores[j.ID] = j
	}

	for rows.Next() {
		var rd models.ResultadoDetalle
		var resultadoID int
		if err := rows.Scan(&resultadoID, &rd.Jugador.ID, &rd.Jugador.Nombre, &rd.Jugador.Color, &rd.Jugador.Avatar); err != nil {
			return detalle, err
		}

		// Cargar victorias de este resultado
		vRows, err := db.Query(`SELECT rival_id FROM victorias WHERE resultado_id=$1`, resultadoID)
		if err != nil {
			return detalle, err
		}
		for vRows.Next() {
			var rivalID int
			vRows.Scan(&rivalID)
			if j, ok := mapaJugadores[rivalID]; ok {
				rd.Victorias = append(rd.Victorias, j)
			}
		}
		vRows.Close()
		rd.TotalVictorias = len(rd.Victorias)

		// Derrotas = jugadores en la sesión que le ganaron a este
		dRows, err := db.Query(`
			SELECT r.jugador_id 
			FROM victorias v
			JOIN resultados r ON r.id = v.resultado_id
			WHERE r.sesion_id = $1 AND v.rival_id = $2`, sesionID, rd.Jugador.ID)
		if err != nil {
			return detalle, err
		}
		for dRows.Next() {
			var ganadorID int
			dRows.Scan(&ganadorID)
			if j, ok := mapaJugadores[ganadorID]; ok {
				rd.Derrotas = append(rd.Derrotas, j)
			}
		}
		dRows.Close()
		rd.TotalDerrotas = len(rd.Derrotas)

		// Colores jugados
		cRows, err := db.Query(`SELECT color FROM colores_jugados WHERE resultado_id=$1`, resultadoID)
		if err != nil {
			return detalle, err
		}
		for cRows.Next() {
			var color string
			cRows.Scan(&color)
			rd.Colores = append(rd.Colores, color)
		}
		cRows.Close()

		detalle.Resultados = append(detalle.Resultados, rd)
	}

	return detalle, nil
}

// ========== RANKING ==========

// ObtenerRanking calcula el ranking de una temporada
func ObtenerRanking(db *sql.DB, temporadaID int) ([]models.FilaRanking, error) {
	// Obtener todos los jugadores con resultados en la temporada
	rows, err := db.Query(`
		SELECT DISTINCT j.id, j.nombre, j.color, j.avatar
		FROM jugadores j
		JOIN resultados r ON r.jugador_id = j.id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE s.temporada_id = $1
		ORDER BY j.nombre`, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jugadores []models.Jugador
	for rows.Next() {
		var j models.Jugador
		rows.Scan(&j.ID, &j.Nombre, &j.Color, &j.Avatar)
		jugadores = append(jugadores, j)
	}

	var ranking []models.FilaRanking
	for _, j := range jugadores {
		fila, err := calcularFilaRanking(db, j, temporadaID)
		if err != nil {
			return nil, err
		}
		ranking = append(ranking, fila)
	}

	// Ordenar por victorias desc, luego win rate desc
	for i := 0; i < len(ranking); i++ {
		for k := i + 1; k < len(ranking); k++ {
			if ranking[k].Victorias > ranking[i].Victorias ||
				(ranking[k].Victorias == ranking[i].Victorias && ranking[k].WinRate > ranking[i].WinRate) {
				ranking[i], ranking[k] = ranking[k], ranking[i]
			}
		}
	}

	return ranking, nil
}

// calcularFilaRanking calcula las estadísticas de un jugador en una temporada
func calcularFilaRanking(db *sql.DB, j models.Jugador, temporadaID int) (models.FilaRanking, error) {
	fila := models.FilaRanking{Jugador: j}

	// Victorias totales: partidas individuales que este jugador ganó en la temporada.
	// Cuenta registros en 'victorias' donde el resultado pertenece a este jugador
	// y la sesión del resultado está en la temporada.
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2`, j.ID, temporadaID).Scan(&fila.Victorias)
	if err != nil {
		return fila, err
	}

	// Derrotas totales: partidas individuales que este jugador perdió en la temporada.
	// Busca registros en 'victorias' donde este jugador es el rival_id (fue vencido),
	// y filtra por sesiones de la temporada a través de la sesión donde ESTE jugador participó.
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM victorias v
		JOIN resultados r_ganador ON r_ganador.id = v.resultado_id
		JOIN resultados r_perdedor ON r_perdedor.sesion_id = r_ganador.sesion_id AND r_perdedor.jugador_id = $1
		JOIN sesiones s ON s.id = r_perdedor.sesion_id
		WHERE v.rival_id = $1 AND s.temporada_id = $2`, j.ID, temporadaID).Scan(&fila.Derrotas)
	if err != nil {
		return fila, err
	}

	fila.Partidas = fila.Victorias + fila.Derrotas
	if fila.Partidas > 0 {
		fila.WinRate = float64(fila.Victorias) / float64(fila.Partidas) * 100
	}

	// Calcular racha actual: obtener últimas partidas ordenadas por fecha
	sesionesRows, err := db.Query(`
		SELECT s.fecha, 
			(SELECT COUNT(*) FROM victorias v2 JOIN resultados r2 ON r2.id=v2.resultado_id WHERE r2.jugador_id=$1 AND r2.sesion_id=s.id) as vict,
			(SELECT COUNT(*) FROM victorias v3 JOIN resultados r3 ON r3.id=v3.resultado_id WHERE v3.rival_id=$2 AND r3.sesion_id=s.id) as derr
		FROM sesiones s
		JOIN resultados r ON r.sesion_id = s.id AND r.jugador_id = $3
		WHERE s.temporada_id = $4
		ORDER BY s.fecha DESC`, j.ID, j.ID, j.ID, temporadaID)
	if err != nil {
		return fila, err
	}
	defer sesionesRows.Close()

	type sesionStats struct {
		victorias int
		derrotas  int
	}
	var historial []sesionStats
	for sesionesRows.Next() {
		var fecha time.Time
		var vs sesionStats
		sesionesRows.Scan(&fecha, &vs.victorias, &vs.derrotas)
		historial = append(historial, vs)
	}

	// Calcular racha desde la sesión más reciente
	if len(historial) > 0 {
		ultima := historial[0]
		if ultima.victorias > ultima.derrotas {
			fila.RachaTipo = "W"
			fila.RachaActual = 1
			for i := 1; i < len(historial); i++ {
				if historial[i].victorias > historial[i].derrotas {
					fila.RachaActual++
				} else {
					break
				}
			}
		} else if ultima.derrotas > ultima.victorias {
			fila.RachaTipo = "L"
			fila.RachaActual = 1
			for i := 1; i < len(historial); i++ {
				if historial[i].derrotas > historial[i].victorias {
					fila.RachaActual++
				} else {
					break
				}
			}
		}
	}

	return fila, nil
}

// ========== ESTADÍSTICAS ==========

// ObtenerEstadisticasColores devuelve el win rate por color de un jugador en la temporada.
// Para cada color: itera los resultados donde usó ese color y suma V y D directamente.
// Win rate = victorias / (victorias + derrotas) × 100.
func ObtenerEstadisticasColores(db *sql.DB, jugadorID, temporadaID int) ([]models.EstadisticaColor, error) {
	colores := []string{"W", "U", "B", "R", "G"}
	var stats []models.EstadisticaColor

	for _, color := range colores {
		// Obtener los resultados donde este jugador usó este color
		rows, err := db.Query(`
			SELECT r.id, r.sesion_id
			FROM resultados r
			JOIN colores_jugados cj ON cj.resultado_id = r.id
			JOIN sesiones s ON s.id = r.sesion_id
			WHERE r.jugador_id = $1 AND cj.color = $2 AND s.temporada_id = $3`,
			jugadorID, color, temporadaID)
		if err != nil {
			return nil, err
		}

		type par struct{ resultadoID, sesionID int }
		var pares []par
		for rows.Next() {
			var p par
			rows.Scan(&p.resultadoID, &p.sesionID)
			pares = append(pares, p)
		}
		rows.Close()

		if len(pares) == 0 {
			continue
		}

		var ec models.EstadisticaColor
		ec.Color = color
		ec.Nombre = models.NombreColor(color)

		for _, p := range pares {
			var v int
			db.QueryRow(`SELECT COUNT(*) FROM victorias WHERE resultado_id=$1`, p.resultadoID).Scan(&v)
			ec.Victorias += v

			var d int
			db.QueryRow(`
				SELECT COUNT(*)
				FROM victorias v2
				JOIN resultados r2 ON r2.id = v2.resultado_id
				WHERE r2.sesion_id = $1 AND v2.rival_id = $2`,
				p.sesionID, jugadorID).Scan(&d)
			ec.Partidas += v + d
		}

		if ec.Partidas == 0 {
			continue
		}
		ec.WinRate = float64(ec.Victorias) / float64(ec.Partidas) * 100
		stats = append(stats, ec)
	}

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
