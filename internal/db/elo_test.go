package db

import (
	"math"
	"testing"
)

func cerca(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestCalcularEloPrimeraPartida(t *testing.T) {
	// Entre iguales (1500 vs 1500) el ganador suma K/2 = 16 y el perdedor resta 16
	elo := calcularElo([]partidaElo{{SesionID: 1, Ganador: 1, Perdedor: 2}})
	if !cerca(elo[1].Rating, 1516) || !cerca(elo[2].Rating, 1484) {
		t.Errorf("got %v / %v, quiero 1516 / 1484", elo[1].Rating, elo[2].Rating)
	}
	if !cerca(elo[1].DeltaUltimo, 16) || !cerca(elo[2].DeltaUltimo, -16) {
		t.Errorf("deltas %v / %v", elo[1].DeltaUltimo, elo[2].DeltaUltimo)
	}
}

func TestCalcularEloGanarAlFuerteValeMas(t *testing.T) {
	// Sesión 1: A gana a B y a C (A sube, B y C bajan). Sesión 2: D gana a A, E gana a B.
	// D (que gana al líder) debe sumar más que E (que gana al que va abajo).
	elo := calcularElo([]partidaElo{
		{1, 1, 2}, {1, 1, 3},
		{2, 4, 1}, {2, 5, 2},
	})
	if !(elo[4].DeltaUltimo > elo[5].DeltaUltimo) {
		t.Errorf("ganar al líder (%.2f) debería valer más que ganar al último (%.2f)", elo[4].DeltaUltimo, elo[5].DeltaUltimo)
	}
}

func TestCalcularEloNoDependeDelOrdenDentroDeLaSesion(t *testing.T) {
	a := calcularElo([]partidaElo{{1, 1, 2}, {1, 2, 3}, {1, 3, 1}, {1, 1, 3}})
	b := calcularElo([]partidaElo{{1, 1, 3}, {1, 3, 1}, {1, 2, 3}, {1, 1, 2}})
	for id := 1; id <= 3; id++ {
		if !cerca(a[id].Rating, b[id].Rating) {
			t.Errorf("jugador %d: %.2f vs %.2f según el orden", id, a[id].Rating, b[id].Rating)
		}
	}
}

func TestCalcularEloSumaCero(t *testing.T) {
	// Lo que gana uno lo pierde otro: la media siempre sigue en 1500
	elo := calcularElo([]partidaElo{{1, 1, 2}, {1, 3, 2}, {2, 2, 1}, {2, 2, 3}, {3, 3, 1}})
	total := 0.0
	for _, r := range elo {
		total += r.Rating
	}
	if !cerca(total/float64(len(elo)), EloInicial) {
		t.Errorf("media %.2f, quiero %v", total/float64(len(elo)), EloInicial)
	}
}

func TestCalcularEloDeltaUltimoSoloDondeJugo(t *testing.T) {
	// El jugador 3 solo juega la sesión 1: su "último draft" es esa, no la 2
	elo := calcularElo([]partidaElo{{1, 3, 1}, {2, 1, 2}})
	if !cerca(elo[3].DeltaUltimo, 16) {
		t.Errorf("delta del jugador 3 = %.2f, quiero 16", elo[3].DeltaUltimo)
	}
}
