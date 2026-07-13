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
	// Colores de cada resultado del jugador en UNA sola consulta (antes N+1),
	// agrupados por resultado en memoria.
	rows, err := db.Query(`
		SELECT r.id, cj.color
		FROM resultados r
		JOIN sesiones s ON s.id = r.sesion_id
		LEFT JOIN colores_jugados cj ON cj.resultado_id = r.id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2
		ORDER BY r.id, cj.color`, jugadorID, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coloresPorResultado := map[int][]string{}
	var orden []int
	for rows.Next() {
		var rid int
		var color sql.NullString
		if err := rows.Scan(&rid, &color); err != nil {
			return nil, err
		}
		if _, ok := coloresPorResultado[rid]; !ok {
			orden = append(orden, rid)
			coloresPorResultado[rid] = nil
		}
		if color.Valid {
			coloresPorResultado[rid] = append(coloresPorResultado[rid], color.String)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Contar la frecuencia de cada combinación (clave = "B,U" ordenado)
	combinaciones := make(map[string]int)
	for _, rid := range orden {
		cs := coloresPorResultado[rid]
		if len(cs) == 0 {
			continue
		}
		combinaciones[strings.Join(cs, ",")]++
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

	// Si ninguna combinación se repite (todas ×1), la "favorita" sería arbitraria.
	// En ese caso mostramos el color más jugado, que es más informativo.
	if topCount <= 1 {
		orden := map[string]int{"W": 0, "U": 1, "B": 2, "R": 3, "G": 4}
		conteo := map[string]int{}
		for _, cs := range coloresPorResultado {
			for _, c := range cs {
				conteo[c]++
			}
		}
		if len(conteo) == 0 {
			return nil, nil
		}
		topColor, topN := "", 0
		for c, n := range conteo {
			if n > topN || (n == topN && orden[c] < orden[topColor]) {
				topColor, topN = c, n
			}
		}
		return &models.CombinacionColor{Colores: []string{topColor}, Veces: topN}, nil
	}

	return &models.CombinacionColor{
		Colores: strings.Split(topClave, ","),
		Veces:   topCount,
	}, nil
}

// ObtenerDesempenoJugador calcula, para un jugador en una temporada:
//   - ganados: nº de sesiones en las que fue quien más victorias logró (empates incl.)
//   - posMedia: puesto medio en las sesiones jugadas (1 = mejor)
//   - tendencia: "up"/"down"/"flat" según el win rate reciente vs el anterior
func ObtenerDesempenoJugador(db *sql.DB, jugadorID, temporadaID int) (ganados int, posMedia float64, tendencia string, err error) {
	tendencia = "flat"

	// Victorias de TODOS los participantes en las sesiones donde jugó este jugador.
	rows, err := db.Query(`
		SELECT r.sesion_id, r.jugador_id,
			(SELECT COUNT(*) FROM victorias v WHERE v.resultado_id = r.id) AS vict
		FROM resultados r
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE s.temporada_id = $1
		  AND r.sesion_id IN (SELECT sesion_id FROM resultados WHERE jugador_id = $2)`,
		temporadaID, jugadorID)
	if err != nil {
		return 0, 0, "flat", err
	}
	victPorSesion := map[int]map[int]int{} // sesión -> jugador -> victorias
	for rows.Next() {
		var sid, jid, v int
		if err := rows.Scan(&sid, &jid, &v); err != nil {
			rows.Close()
			return 0, 0, "flat", err
		}
		if victPorSesion[sid] == nil {
			victPorSesion[sid] = map[int]int{}
		}
		victPorSesion[sid][jid] = v
	}
	rows.Close()

	nSes, sumaPos := 0, 0
	for _, jugs := range victPorSesion {
		mine, ok := jugs[jugadorID]
		if !ok {
			continue
		}
		nSes++
		maxV, pos := 0, 1
		for _, v := range jugs {
			if v > maxV {
				maxV = v
			}
			if v > mine {
				pos++
			}
		}
		sumaPos += pos
		if mine == maxV && maxV > 0 {
			ganados++
		}
	}
	if nSes > 0 {
		posMedia = float64(sumaPos) / float64(nSes)
	}

	// Win rate por sesión en orden cronológico → tendencia.
	wrRows, err := db.Query(`
		SELECT
			(SELECT COUNT(*) FROM victorias v JOIN resultados r2 ON r2.id = v.resultado_id
				WHERE r2.jugador_id = $1 AND r2.sesion_id = s.id) AS vict,
			(SELECT COUNT(*) FROM victorias v JOIN resultados r2 ON r2.id = v.resultado_id
				WHERE v.rival_id = $1 AND r2.sesion_id = s.id) AS derr
		FROM sesiones s
		JOIN resultados r ON r.sesion_id = s.id AND r.jugador_id = $1
		WHERE s.temporada_id = $2
		ORDER BY s.fecha ASC`, jugadorID, temporadaID)
	if err != nil {
		return ganados, posMedia, "flat", nil
	}
	var wr []float64
	for wrRows.Next() {
		var v, d int
		if err := wrRows.Scan(&v, &d); err != nil {
			break
		}
		if v+d > 0 {
			wr = append(wr, float64(v)/float64(v+d)*100)
		}
	}
	wrRows.Close()
	return ganados, posMedia, calcularTendencia(wr), nil
}

// calcularTendencia compara la media del win rate de la mitad reciente con la mitad
// antigua de las sesiones. Devuelve "up"/"down"/"flat" (umbral de 5 puntos).
func calcularTendencia(wr []float64) string {
	n := len(wr)
	if n < 2 {
		return "flat"
	}
	media := func(s []float64) float64 {
		if len(s) == 0 {
			return 0
		}
		t := 0.0
		for _, x := range s {
			t += x
		}
		return t / float64(len(s))
	}
	mitad := n / 2
	delta := media(wr[mitad:]) - media(wr[:mitad])
	switch {
	case delta > 5:
		return "up"
	case delta < -5:
		return "down"
	default:
		return "flat"
	}
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
		gano   bool // true si ganó la sesión, false si perdió, sesiones empatadas se ignoran
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
//
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

	// Cargar todos los jugadores UNA vez para resolver nombres (antes era 1 consulta por rival)
	todos, err := ObtenerJugadores(db)
	if err != nil {
		return nil, nil, err
	}
	mapaJugadores := make(map[int]models.Jugador, len(todos))
	for _, j := range todos {
		mapaJugadores[j.ID] = j
	}

	// Calcular win rate y quedarnos con verdugo (peor WR) y víctima (mejor WR)
	const minPartidas = 2 // umbral mínimo para considerar a un rival
	var verdugo, victima *models.RivalConWinRate
	var peorWR, mejorWR float64 = 101, -1

	for rid, rf := range rivales {
		rf.Partidas = rf.Victorias + rf.Derrotas
		if rf.Partidas < minPartidas {
			continue
		}
		rf.WinRate = float64(rf.Victorias) / float64(rf.Partidas) * 100

		rival, ok := mapaJugadores[rid]
		if !ok || !rival.Activo {
			continue // los jugadores inactivos no aparecen como verdugo/víctima
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
// con 1, 2, 3, 4 o 5 colores en una temporada.
func ObtenerDistribucionDraftsJugador(db *sql.DB, jugadorID, temporadaID int) (models.DistribucionDrafts, error) {
	var dist models.DistribucionDrafts

	// Nº de colores por resultado del jugador, en UNA sola consulta (antes N+1).
	rows, err := db.Query(`
		SELECT COUNT(cj.id) AS n
		FROM resultados r
		JOIN sesiones s ON s.id = r.sesion_id
		LEFT JOIN colores_jugados cj ON cj.resultado_id = r.id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2
		GROUP BY r.id`, jugadorID, temporadaID)
	if err != nil {
		return dist, err
	}
	defer rows.Close()

	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return dist, err
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
		case 4:
			dist.Cuatricolor++
		default:
			dist.Pentacolor++ // 5 colores (o más, por robustez)
		}
		if n > 0 {
			dist.Total++
		}
	}

	return dist, rows.Err()
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

// ObtenerMatrizH2H devuelve, para una temporada, las victorias de cada jugador sobre
// cada rival: wins[ganadorID][rivalID] = nº de veces que el ganador venció al rival.
// Una sola consulta agregada.
func ObtenerMatrizH2H(db *sql.DB, temporadaID int) (map[int]map[int]int, error) {
	rows, err := db.Query(`
		SELECT r.jugador_id AS ganador, v.rival_id, COUNT(*)
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE s.temporada_id = $1
		GROUP BY r.jugador_id, v.rival_id`, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wins := map[int]map[int]int{}
	for rows.Next() {
		var ganador, rival, n int
		if err := rows.Scan(&ganador, &rival, &n); err != nil {
			return nil, err
		}
		if wins[ganador] == nil {
			wins[ganador] = map[int]int{}
		}
		wins[ganador][rival] = n
	}
	return wins, rows.Err()
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

// ObtenerHistorialJugador devuelve, por sesión (más reciente primero), el resultado
// resumido de un jugador en una temporada: victorias, derrotas, colores y si fue quien
// más victorias logró esa sesión. Usa 2 consultas (no N+1).
func ObtenerHistorialJugador(db *sql.DB, jugadorID, temporadaID int) ([]models.DraftJugador, error) {
	rows, err := db.Query(`
		SELECT s.id, s.fecha, s.descripcion,
			(SELECT COUNT(*) FROM victorias v
				JOIN resultados r2 ON r2.id = v.resultado_id
				WHERE r2.jugador_id = $1 AND r2.sesion_id = s.id) AS vict,
			(SELECT COUNT(*) FROM victorias v
				JOIN resultados r2 ON r2.id = v.resultado_id
				WHERE v.rival_id = $1 AND r2.sesion_id = s.id) AS derr,
			(SELECT COALESCE(MAX(cnt), 0) FROM (
				SELECT COUNT(*) AS cnt FROM victorias v
				JOIN resultados r3 ON r3.id = v.resultado_id
				WHERE r3.sesion_id = s.id
				GROUP BY r3.jugador_id
			) m) AS maxvict,
			r.notas
		FROM sesiones s
		JOIN resultados r ON r.sesion_id = s.id AND r.jugador_id = $1
		WHERE s.temporada_id = $2
		ORDER BY s.fecha DESC`, jugadorID, temporadaID)
	if err != nil {
		return nil, err
	}

	var drafts []models.DraftJugador
	idxPorSesion := map[int]int{}
	for rows.Next() {
		var d models.DraftJugador
		var maxVict int
		if err := rows.Scan(&d.SesionID, &d.Fecha, &d.Descripcion, &d.Victorias, &d.Derrotas, &maxVict, &d.Notas); err != nil {
			rows.Close()
			return nil, err
		}
		d.Top = d.Victorias > 0 && d.Victorias == maxVict
		idxPorSesion[d.SesionID] = len(drafts)
		drafts = append(drafts, d)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	// Colores por sesión (una sola consulta)
	cRows, err := db.Query(`
		SELECT r.sesion_id, cj.color
		FROM colores_jugados cj
		JOIN resultados r ON r.id = cj.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE r.jugador_id = $1 AND s.temporada_id = $2
		ORDER BY cj.color`, jugadorID, temporadaID)
	if err != nil {
		return nil, err
	}
	defer cRows.Close()
	for cRows.Next() {
		var sesionID int
		var color string
		if err := cRows.Scan(&sesionID, &color); err != nil {
			return nil, err
		}
		if i, ok := idxPorSesion[sesionID]; ok {
			drafts[i].Colores = append(drafts[i].Colores, color)
		}
	}
	return drafts, cRows.Err()
}

// EvolucionFila es una fila de la evolución por sesión (victorias y derrotas).
type EvolucionFila struct {
	JugadorID       int
	Nombre          string
	Color           string
	Avatar          string
	SesionID        int
	Fecha           time.Time
	VictoriasSesion int
	DerrotasSesion  int
}

// ObtenerEvolucionVictorias devuelve, en orden cronológico, las victorias y derrotas de
// cada jugador en cada sesión de la temporada (para las gráficas de evolución).
func ObtenerEvolucionVictorias(db *sql.DB, temporadaID int) ([]EvolucionFila, error) {
	rows, err := db.Query(`
		SELECT
			j.id, j.nombre, j.color, j.avatar,
			s.id, s.fecha,
			(SELECT COUNT(*) FROM victorias v WHERE v.resultado_id = r.id) AS vict,
			(SELECT COUNT(*) FROM victorias v2
				JOIN resultados r2 ON r2.id = v2.resultado_id
				WHERE r2.sesion_id = r.sesion_id AND v2.rival_id = r.jugador_id) AS derr
		FROM resultados r
		JOIN sesiones s ON s.id = r.sesion_id
		JOIN jugadores j ON j.id = r.jugador_id
		WHERE s.temporada_id = $1
		ORDER BY s.fecha ASC, s.id ASC, j.nombre ASC`, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filas []EvolucionFila
	for rows.Next() {
		var f EvolucionFila
		if err := rows.Scan(&f.JugadorID, &f.Nombre, &f.Color, &f.Avatar, &f.SesionID, &f.Fecha, &f.VictoriasSesion, &f.DerrotasSesion); err != nil {
			return nil, err
		}
		filas = append(filas, f)
	}
	return filas, rows.Err()
}

// ObtenerColoresTemporada devuelve, por color, las estadísticas de TODO el grupo en la
// temporada (veces jugado, victorias, derrotas y win rate normalizado). Para la "meta".
func ObtenerColoresTemporada(db *sql.DB, temporadaID int) ([]models.EstadisticaColor, error) {
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
		WHERE s.temporada_id = $1
		GROUP BY cj.color`, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orden := map[string]int{"W": 0, "U": 1, "B": 2, "R": 3, "G": 4}
	var stats []models.EstadisticaColor
	for rows.Next() {
		var ec models.EstadisticaColor
		var derrotas int
		if err := rows.Scan(&ec.Color, &ec.Veces, &ec.Victorias, &derrotas); err != nil {
			return nil, err
		}
		ec.Nombre = models.NombreColor(ec.Color)
		ec.Partidas = ec.Victorias + derrotas
		if ec.Partidas > 0 {
			ec.WinRate = float64(ec.Victorias) / float64(ec.Partidas) * 100
		}
		stats = append(stats, ec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(stats, func(i, j int) bool { return orden[stats[i].Color] < orden[stats[j].Color] })
	return stats, nil
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

	// Drafts ganados, posición media y tendencia
	if g, pm, t, err := ObtenerDesempenoJugador(db, jugador.ID, temporadaID); err == nil {
		stats.DraftsGanados = g
		stats.PosicionMedia = pm
		stats.Tendencia = t
	}

	return stats, nil
}
