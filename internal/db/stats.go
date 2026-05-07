// Estadísticas avanzadas: combinaciones de colores, mejor sesión, rival más frecuente,
// racha histórica récord y distribución de colores del grupo.
package db

import (
	"database/sql"
	"math"
	"sort"
	"strings"
	"time"

	"mtg-tracker/internal/models"
)

// ObtenerCombinacionTopJugador devuelve la combinación de colores más frecuente
// de un jugador en una temporada. Devuelve nil si no hay datos.
func ObtenerCombinacionTopJugador(db *sql.DB, jugadorID, temporadaID int) (*models.CombinacionColor, error) {
	// Obtener todas las combinaciones de colores que ha jugado en cada resultado
	rows, err := db.Query(`
		SELECT r.id
		FROM resultados r
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2`, jugadorID, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultadoIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		resultadoIDs = append(resultadoIDs, id)
	}

	if len(resultadoIDs) == 0 {
		return nil, nil
	}

	// Para cada resultado, obtener su combinación de colores y contarla
	combinaciones := make(map[string]int) // clave = "B,U" ordenado
	for _, rid := range resultadoIDs {
		cRows, err := db.Query(`SELECT color FROM colores_jugados WHERE resultado_id=$1 ORDER BY color`, rid)
		if err != nil {
			return nil, err
		}
		var colores []string
		for cRows.Next() {
			var c string
			cRows.Scan(&c)
			colores = append(colores, c)
		}
		cRows.Close()

		if len(colores) == 0 {
			continue
		}

		clave := strings.Join(colores, ",")
		combinaciones[clave]++
	}

	if len(combinaciones) == 0 {
		return nil, nil
	}

	// Encontrar la combinación con más usos
	var topClave string
	var topCount int
	for k, n := range combinaciones {
		if n > topCount {
			topCount = n
			topClave = k
		}
	}

	return &models.CombinacionColor{
		Colores: strings.Split(topClave, ","),
		Veces:   topCount,
	}, nil
}

// ObtenerMejorSesionJugador devuelve la sesión con más victorias de un jugador.
// Si hay empate, devuelve la más reciente. Devuelve nil si no hay datos.
func ObtenerMejorSesionJugador(db *sql.DB, jugadorID, temporadaID int) (*models.MejorSesion, error) {
	rows, err := db.Query(`
		SELECT s.id, s.fecha, s.descripcion,
			(SELECT COUNT(*) FROM victorias v 
				JOIN resultados r2 ON r2.id = v.resultado_id 
				WHERE r2.jugador_id = $1 AND r2.sesion_id = s.id) AS vict,
			(SELECT COUNT(*) FROM victorias v 
				JOIN resultados r2 ON r2.id = v.resultado_id 
				WHERE v.rival_id = $2 AND r2.sesion_id = s.id) AS derr
		FROM sesiones s
		JOIN resultados r ON r.sesion_id = s.id AND r.jugador_id = $3
		WHERE s.temporada_id = $4
		ORDER BY s.fecha DESC`, jugadorID, jugadorID, jugadorID, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mejor *models.MejorSesion
	for rows.Next() {
		var ms models.MejorSesion
		if err := rows.Scan(&ms.SesionID, &ms.Fecha, &ms.Descripcion, &ms.Victorias, &ms.Derrotas); err != nil {
			return nil, err
		}
		// Quedarnos con la primera sesión donde tenga más victorias
		// (las filas vienen por fecha DESC, así que el primer match con el máximo es la más reciente)
		if mejor == nil || ms.Victorias > mejor.Victorias {
			copia := ms
			mejor = &copia
		}
	}

	// Si todas las sesiones tienen 0 victorias, no hay "mejor"
	if mejor != nil && mejor.Victorias == 0 {
		return nil, nil
	}

	return mejor, nil
}

// ObtenerRivalMasFrecuente devuelve el rival contra quien más ha jugado un jugador
// en una temporada. Devuelve nil si no hay datos.
func ObtenerRivalMasFrecuente(db *sql.DB, jugadorID, temporadaID int) (*models.RivalFrecuente, error) {
	// Para cada otro jugador, contar partidas (victorias + derrotas) contra él
	tipoJugador := make(map[int]*models.RivalFrecuente)

	// Victorias del jugador sobre cada rival
	rows, err := db.Query(`
		SELECT v.rival_id, COUNT(*)
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2
		GROUP BY v.rival_id`, jugadorID, temporadaID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var rivalID, count int
		rows.Scan(&rivalID, &count)
		if tipoJugador[rivalID] == nil {
			tipoJugador[rivalID] = &models.RivalFrecuente{}
		}
		tipoJugador[rivalID].Victorias = count
	}
	rows.Close()

	// Derrotas del jugador contra cada rival
	rows, err = db.Query(`
		SELECT r.jugador_id, COUNT(*)
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE v.rival_id = $1 AND s.temporada_id = $2
		GROUP BY r.jugador_id`, jugadorID, temporadaID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var rivalID, count int
		rows.Scan(&rivalID, &count)
		if tipoJugador[rivalID] == nil {
			tipoJugador[rivalID] = &models.RivalFrecuente{}
		}
		tipoJugador[rivalID].Derrotas = count
	}
	rows.Close()

	if len(tipoJugador) == 0 {
		return nil, nil
	}

	// Encontrar el rival con más partidas totales
	var topRivalID int
	var topPartidas int
	for rid, rf := range tipoJugador {
		total := rf.Victorias + rf.Derrotas
		if total > topPartidas {
			topPartidas = total
			topRivalID = rid
		}
	}

	if topRivalID == 0 {
		return nil, nil
	}

	rival, err := ObtenerJugador(db, topRivalID)
	if err != nil {
		return nil, err
	}

	rf := tipoJugador[topRivalID]
	rf.Rival = rival
	rf.Partidas = rf.Victorias + rf.Derrotas
	return rf, nil
}

// ObtenerRachaRecordJugador devuelve la mejor racha histórica de un jugador en una temporada.
// Considera todas las sesiones (no solo las recientes) y busca la secuencia consecutiva más larga.
func ObtenerRachaRecordJugador(db *sql.DB, jugadorID, temporadaID int) (models.RachaRecord, error) {
	rows, err := db.Query(`
		SELECT s.fecha, 
			(SELECT COUNT(*) FROM victorias v 
				JOIN resultados r2 ON r2.id = v.resultado_id 
				WHERE r2.jugador_id = $1 AND r2.sesion_id = s.id) AS vict,
			(SELECT COUNT(*) FROM victorias v 
				JOIN resultados r2 ON r2.id = v.resultado_id 
				WHERE v.rival_id = $2 AND r2.sesion_id = s.id) AS derr
		FROM sesiones s
		JOIN resultados r ON r.sesion_id = s.id AND r.jugador_id = $3
		WHERE s.temporada_id = $4
		ORDER BY s.fecha ASC`, jugadorID, jugadorID, jugadorID, temporadaID)
	if err != nil {
		return models.RachaRecord{}, err
	}
	defer rows.Close()

	type sesionResult struct {
		gano bool // true si ganó la sesión, false si perdió, sesiones empatadas se ignoran
		jugada bool
	}

	var historial []sesionResult
	for rows.Next() {
		var fecha time.Time
		var vict, derr int
		rows.Scan(&fecha, &vict, &derr)
		if vict > derr {
			historial = append(historial, sesionResult{gano: true, jugada: true})
		} else if derr > vict {
			historial = append(historial, sesionResult{gano: false, jugada: true})
		}
		// Empates: se ignoran (rompen la racha pero no cuentan como W ni L)
	}

	if len(historial) == 0 {
		return models.RachaRecord{}, nil
	}

	// Buscar la racha más larga de victorias consecutivas
	mejorW := 0
	mejorL := 0
	actualW := 0
	actualL := 0
	for _, s := range historial {
		if s.gano {
			actualW++
			actualL = 0
			if actualW > mejorW {
				mejorW = actualW
			}
		} else {
			actualL++
			actualW = 0
			if actualL > mejorL {
				mejorL = actualL
			}
		}
	}

	// La racha "récord" es la más larga, sea de victorias o de derrotas.
	// Priorizamos victorias en caso de empate (más motivador).
	if mejorW >= mejorL {
		return models.RachaRecord{Longitud: mejorW, Tipo: "W"}, nil
	}
	return models.RachaRecord{Longitud: mejorL, Tipo: "L"}, nil
}

// ObtenerDistribucionColoresGrupo devuelve cuántas veces se ha jugado cada color
// en toda la temporada (sumando jugadores y sesiones).
func ObtenerDistribucionColoresGrupo(db *sql.DB, temporadaID int) ([]models.DistribucionColor, error) {
	codigos := []string{"W", "U", "B", "R", "G"}
	dist := make([]models.DistribucionColor, 0, 5)
	total := 0

	for _, c := range codigos {
		var veces int
		err := db.QueryRow(`
			SELECT COUNT(cj.id)
			FROM colores_jugados cj
			JOIN resultados r ON r.id = cj.resultado_id
			JOIN sesiones s ON s.id = r.sesion_id
			WHERE cj.color = $1 AND s.temporada_id = $2`, c, temporadaID).Scan(&veces)
		if err != nil {
			return nil, err
		}
		dist = append(dist, models.DistribucionColor{
			Color:  c,
			Nombre: models.NombreColor(c),
			Veces:  veces,
		})
		total += veces
	}

	// Calcular porcentajes
	if total > 0 {
		for i := range dist {
			dist[i].Porcentaje = float64(dist[i].Veces) / float64(total) * 100
		}
	}

	// Ordenar de más a menos jugado
	sort.Slice(dist, func(i, j int) bool {
		return dist[i].Veces > dist[j].Veces
	})

	return dist, nil
}

// ObtenerVerdugoYVictima analiza los rivales del jugador y devuelve:
//   - verdugo: rival contra el que peor win rate tiene (con mínimo 2 partidas)
//   - víctima: rival contra el que mejor win rate tiene (con mínimo 2 partidas)
// Devuelve (nil, nil) si no hay rivales con suficientes partidas.
func ObtenerVerdugoYVictima(db *sql.DB, jugadorID, temporadaID int) (*models.RivalConWinRate, *models.RivalConWinRate, error) {
	rivales := make(map[int]*models.RivalConWinRate)

	// Victorias del jugador sobre cada rival
	rows, err := db.Query(`
		SELECT v.rival_id, COUNT(*)
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2
		GROUP BY v.rival_id`, jugadorID, temporadaID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var rid, count int
		rows.Scan(&rid, &count)
		if rivales[rid] == nil {
			rivales[rid] = &models.RivalConWinRate{}
		}
		rivales[rid].Victorias = count
	}
	rows.Close()

	// Derrotas (rivales que ganaron al jugador)
	rows, err = db.Query(`
		SELECT r.jugador_id, COUNT(*)
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE v.rival_id = $1 AND s.temporada_id = $2
		GROUP BY r.jugador_id`, jugadorID, temporadaID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var rid, count int
		rows.Scan(&rid, &count)
		if rivales[rid] == nil {
			rivales[rid] = &models.RivalConWinRate{}
		}
		rivales[rid].Derrotas = count
	}
	rows.Close()

	if len(rivales) == 0 {
		return nil, nil, nil
	}

	// Cargar info de los rivales y calcular win rate
	const minPartidas = 2 // umbral mínimo para considerar a un rival
	var verdugo, victima *models.RivalConWinRate
	var peorWR, mejorWR float64 = 101, -1

	for rid, rf := range rivales {
		rf.Partidas = rf.Victorias + rf.Derrotas
		if rf.Partidas < minPartidas {
			continue
		}
		rf.WinRate = float64(rf.Victorias) / float64(rf.Partidas) * 100

		rival, err := ObtenerJugador(db, rid)
		if err != nil {
			continue
		}
		rf.Rival = rival

		// Verdugo: peor win rate (este rival "le da palos")
		if rf.WinRate < peorWR {
			peorWR = rf.WinRate
			copia := *rf
			verdugo = &copia
		}
		// Víctima: mejor win rate (a este rival "le gana siempre")
		if rf.WinRate > mejorWR {
			mejorWR = rf.WinRate
			copia := *rf
			victima = &copia
		}
	}

	// Si hay un único rival, no tiene sentido distinguir verdugo y víctima
	if verdugo != nil && victima != nil && verdugo.Rival.ID == victima.Rival.ID {
		// Si solo hay un rival, lo asignamos según su win rate
		if verdugo.WinRate < 50 {
			victima = nil
		} else if verdugo.WinRate > 50 {
			verdugo = nil
		}
	}

	return verdugo, victima, nil
}

// ObtenerDistribucionDraftsJugador cuenta cuántos drafts ha hecho el jugador
// con 1, 2, 3 o más colores en una temporada.
func ObtenerDistribucionDraftsJugador(db *sql.DB, jugadorID, temporadaID int) (models.DistribucionDrafts, error) {
	var dist models.DistribucionDrafts

	rows, err := db.Query(`
		SELECT r.id
		FROM resultados r
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2`, jugadorID, temporadaID)
	if err != nil {
		return dist, err
	}
	defer rows.Close()

	var resultadoIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		resultadoIDs = append(resultadoIDs, id)
	}

	for _, rid := range resultadoIDs {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM colores_jugados WHERE resultado_id=$1`, rid).Scan(&n)
		if err != nil {
			continue
		}
		switch n {
		case 0:
			// Resultado sin colores marcados — no contar
		case 1:
			dist.Mono++
		case 2:
			dist.Bicolor++
		case 3:
			dist.Tricolor++
		default:
			dist.MasColor++
		}
		if n > 0 {
			dist.Total++
		}
	}

	return dist, nil
}

// ObtenerWinRatesPorSesion devuelve los win rates del jugador en cada sesión jugada.
// Útil para calcular desviación típica (constancia).
func ObtenerWinRatesPorSesion(db *sql.DB, jugadorID, temporadaID int) ([]float64, error) {
	rows, err := db.Query(`
		SELECT 
			(SELECT COUNT(*) FROM victorias v 
				JOIN resultados r2 ON r2.id = v.resultado_id 
				WHERE r2.jugador_id = $1 AND r2.sesion_id = s.id) AS vict,
			(SELECT COUNT(*) FROM victorias v 
				JOIN resultados r2 ON r2.id = v.resultado_id 
				WHERE v.rival_id = $2 AND r2.sesion_id = s.id) AS derr
		FROM sesiones s
		JOIN resultados r ON r.sesion_id = s.id AND r.jugador_id = $3
		WHERE s.temporada_id = $4`, jugadorID, jugadorID, jugadorID, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var winRates []float64
	for rows.Next() {
		var v, d int
		rows.Scan(&v, &d)
		total := v + d
		if total == 0 {
			continue
		}
		winRates = append(winRates, float64(v)/float64(total)*100)
	}

	return winRates, nil
}

// CalcularDesviacionWinRate devuelve la desviación típica de un slice de win rates.
// 0 = muy constante, valores altos (>30) = muy irregular.
func CalcularDesviacionWinRate(winRates []float64) float64 {
	if len(winRates) < 2 {
		return 0 // No tiene sentido con menos de 2 sesiones
	}

	// Media
	var suma float64
	for _, wr := range winRates {
		suma += wr
	}
	media := suma / float64(len(winRates))

	// Suma de cuadrados de diferencias
	var sumaCuadrados float64
	for _, wr := range winRates {
		diff := wr - media
		sumaCuadrados += diff * diff
	}

	// Desviación típica poblacional
	return math.Sqrt(sumaCuadrados / float64(len(winRates)))
}

// ContarSesionesJugadasPorJugador devuelve cuántas sesiones ha jugado un jugador en una temporada
func ContarSesionesJugadasPorJugador(db *sql.DB, jugadorID, temporadaID int) (int, error) {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM resultados r
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2`, jugadorID, temporadaID).Scan(&n)
	return n, err
}

// ObtenerH2HExtendido devuelve el head-to-head detallado con cada encuentro por sesión
// y los colores que jugaba cada uno en ese momento.
func ObtenerH2HExtendido(db *sql.DB, j1ID, j2ID int) (*models.H2HExtendido, error) {
	base, err := ObtenerHeadToHead(db, j1ID, j2ID)
	if err != nil {
		return nil, err
	}

	resultado := &models.H2HExtendido{HeadToHead: base}

	// Sesiones donde han coincidido (ambos tienen un resultado)
	rows, err := db.Query(`
		SELECT s.id, s.fecha, s.descripcion, r1.id, r2.id
		FROM sesiones s
		JOIN resultados r1 ON r1.sesion_id = s.id AND r1.jugador_id = $1
		JOIN resultados r2 ON r2.sesion_id = s.id AND r2.jugador_id = $2
		ORDER BY s.fecha DESC`, j1ID, j2ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type sesionPar struct {
		sesionID    int
		fecha       time.Time
		descripcion string
		r1ID, r2ID  int
	}
	var pares []sesionPar
	for rows.Next() {
		var sp sesionPar
		if err := rows.Scan(&sp.sesionID, &sp.fecha, &sp.descripcion, &sp.r1ID, &sp.r2ID); err != nil {
			return nil, err
		}
		pares = append(pares, sp)
	}

	// Para cada sesión, obtener colores de cada uno y conteo de victorias mutuas
	for _, p := range pares {
		enc := models.EncuentroH2H{
			SesionID:    p.sesionID,
			Fecha:       p.fecha,
			Descripcion: p.descripcion,
		}

		// Colores de J1
		cRows, _ := db.Query(`SELECT color FROM colores_jugados WHERE resultado_id=$1 ORDER BY color`, p.r1ID)
		for cRows.Next() {
			var c string
			cRows.Scan(&c)
			enc.ColoresJ1 = append(enc.ColoresJ1, c)
		}
		cRows.Close()

		// Colores de J2
		cRows, _ = db.Query(`SELECT color FROM colores_jugados WHERE resultado_id=$1 ORDER BY color`, p.r2ID)
		for cRows.Next() {
			var c string
			cRows.Scan(&c)
			enc.ColoresJ2 = append(enc.ColoresJ2, c)
		}
		cRows.Close()

		// Victorias mutuas en esa sesión
		db.QueryRow(`SELECT COUNT(*) FROM victorias WHERE resultado_id=$1 AND rival_id=$2`,
			p.r1ID, j2ID).Scan(&enc.GanadasPorJ1)
		db.QueryRow(`SELECT COUNT(*) FROM victorias WHERE resultado_id=$1 AND rival_id=$2`,
			p.r2ID, j1ID).Scan(&enc.GanadasPorJ2)

		// Solo añadir si realmente jugaron entre sí en esa sesión
		if enc.GanadasPorJ1 > 0 || enc.GanadasPorJ2 > 0 {
			resultado.Encuentros = append(resultado.Encuentros, enc)
		}
	}

	return resultado, nil
}

// ObtenerResumenTemporada genera el resumen ejecutivo de una temporada (sea activa o cerrada).
// Incluye campeón, subcampeón, tercero, totales y color dominante.
func ObtenerResumenTemporada(db *sql.DB, temporadaID int) (*models.ResumenTemporada, error) {
	temporada, err := ObtenerTemporada(db, temporadaID)
	if err != nil {
		return nil, err
	}

	resumen := &models.ResumenTemporada{Temporada: temporada}

	// Total sesiones
	resumen.TotalSesiones = ContarSesionesPorTemporada(db, temporadaID)

	// Ranking completo
	ranking, err := ObtenerRanking(db, temporadaID)
	if err != nil {
		return nil, err
	}
	resumen.Ranking = ranking
	resumen.TotalJugadores = len(ranking)

	// Top 3
	if len(ranking) > 0 {
		c := ranking[0]
		resumen.Campeon = &c
	}
	if len(ranking) > 1 {
		s := ranking[1]
		resumen.Subcampeon = &s
	}
	if len(ranking) > 2 {
		t := ranking[2]
		resumen.Tercero = &t
	}

	// Total partidas (suma de victorias del ranking — cada partida tiene exactamente un ganador)
	for _, fila := range ranking {
		resumen.TotalPartidas += fila.Victorias
	}

	// Color dominante de la temporada
	resumen.ColorDominante = ObtenerColorMasJugado(db, temporadaID)

	return resumen, nil
}

// ObtenerEstadisticasCompletasJugador agrupa todas las estadísticas avanzadas de un jugador
func ObtenerEstadisticasCompletasJugador(db *sql.DB, jugador models.Jugador, temporadaID int) (models.EstadisticasJugador, error) {
	stats := models.EstadisticasJugador{Jugador: jugador}

	if c, err := ObtenerEstadisticasColores(db, jugador.ID, temporadaID); err == nil {
		stats.Colores = c
	}
	if c, err := ObtenerCombinacionTopJugador(db, jugador.ID, temporadaID); err == nil {
		stats.CombinacionTop = c
	}
	if m, err := ObtenerMejorSesionJugador(db, jugador.ID, temporadaID); err == nil {
		stats.MejorSesion = m
	}
	if r, err := ObtenerRivalMasFrecuente(db, jugador.ID, temporadaID); err == nil {
		stats.RivalTop = r
	}
	if v, vc, err := ObtenerVerdugoYVictima(db, jugador.ID, temporadaID); err == nil {
		stats.Verdugo = v
		stats.Victima = vc
	}
	if r, err := ObtenerRachaRecordJugador(db, jugador.ID, temporadaID); err == nil {
		stats.RachaRecord = r
	}
	if d, err := ObtenerDistribucionDraftsJugador(db, jugador.ID, temporadaID); err == nil {
		stats.DistribucionDrafts = d
	}

	// Sesiones jugadas y promedio de victorias
	if n, err := ContarSesionesJugadasPorJugador(db, jugador.ID, temporadaID); err == nil {
		stats.SesionesJugadas = n
		if n > 0 {
			// Total victorias en la temporada
			var totalVict int
			db.QueryRow(`
				SELECT COUNT(*)
				FROM victorias v
				JOIN resultados r ON r.id = v.resultado_id
				JOIN sesiones s ON s.id = r.sesion_id
				WHERE r.jugador_id = $1 AND s.temporada_id = $2`,
				jugador.ID, temporadaID).Scan(&totalVict)
			stats.PromedioVictorias = float64(totalVict) / float64(n)
		}
	}

	// Desviación típica del win rate por sesión
	if wrs, err := ObtenerWinRatesPorSesion(db, jugador.ID, temporadaID); err == nil {
		stats.DesviacionWinRate = CalcularDesviacionWinRate(wrs)
	}

	return stats, nil
}
