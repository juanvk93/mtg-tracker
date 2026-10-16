// elo.go — Rating Elo por temporada, calculado en Go a partir del histórico de victorias
package db

import (
	"database/sql"
	"math"
)

const (
	// EloInicial es el rating con el que empieza todo jugador en cada temporada
	EloInicial = 1500.0
	// eloK es cuánto puede moverse el rating en una sola partida
	eloK = 32.0
)

// partidaElo es una partida individual: Ganador venció a Perdedor en la sesión SesionID
type partidaElo struct {
	SesionID, Ganador, Perdedor int
}

// ResultadoElo es el rating final de un jugador y lo que varió en su último draft
type ResultadoElo struct {
	Rating      float64
	DeltaUltimo float64 // variación en la última sesión en la que jugó alguna partida
}

// calcularElo aplica el Elo sesión a sesión. Las partidas deben venir agrupadas por
// sesión y en orden cronológico. Como no se sabe en qué orden se jugaron las partidas
// dentro de un draft, todas las de una sesión se calculan con los ratings del INICIO de
// esa sesión y los cambios se aplican juntos al final: así el resultado no depende del
// orden en que se guardaron las victorias.
func calcularElo(partidas []partidaElo) map[int]ResultadoElo {
	rating := map[int]float64{}
	ultimo := map[int]float64{}
	ratingDe := func(id int) float64 {
		if r, ok := rating[id]; ok {
			return r
		}
		return EloInicial
	}

	for i := 0; i < len(partidas); {
		sesion := partidas[i].SesionID
		delta := map[int]float64{}
		for ; i < len(partidas) && partidas[i].SesionID == sesion; i++ {
			p := partidas[i]
			// Probabilidad esperada de que gane el ganador según los ratings
			esperado := 1 / (1 + math.Pow(10, (ratingDe(p.Perdedor)-ratingDe(p.Ganador))/400))
			cambio := eloK * (1 - esperado)
			delta[p.Ganador] += cambio
			delta[p.Perdedor] -= cambio
		}
		for id, d := range delta {
			rating[id] = ratingDe(id) + d
			ultimo[id] = d
		}
	}

	res := make(map[int]ResultadoElo, len(rating))
	for id, r := range rating {
		res[id] = ResultadoElo{Rating: r, DeltaUltimo: ultimo[id]}
	}
	return res
}

// ObtenerElo calcula el Elo de todos los jugadores con partidas en la temporada.
// Quien no tiene partidas no aparece en el mapa (su rating es EloInicial).
func ObtenerElo(db *sql.DB, temporadaID int) (map[int]ResultadoElo, error) {
	rows, err := db.Query(`
		SELECT s.id, r.jugador_id, v.rival_id
		FROM victorias v
		JOIN resultados r ON r.id = v.resultado_id
		JOIN sesiones s ON s.id = r.sesion_id
		WHERE s.temporada_id = $1
		ORDER BY s.fecha ASC, s.id ASC, r.jugador_id ASC, v.rival_id ASC`, temporadaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var partidas []partidaElo
	for rows.Next() {
		var p partidaElo
		if err := rows.Scan(&p.SesionID, &p.Ganador, &p.Perdedor); err != nil {
			return nil, err
		}
		partidas = append(partidas, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return calcularElo(partidas), nil
}
