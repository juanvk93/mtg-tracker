package handlers

import (
	"net/url"
	"testing"
)

func TestVictoriaCruzada(t *testing.T) {
	jugadores := map[int]bool{1: true, 2: true, 3: true}

	sinCruce := url.Values{"victoria_1": {"2"}, "victoria_2": {"3"}, "victoria_3": {"1"}}
	if _, _, ok := victoriaCruzada(sinCruce, jugadores); ok {
		t.Error("un ciclo A→B→C→A no es una victoria cruzada")
	}

	cruce := url.Values{"victoria_1": {"2", "3"}, "victoria_3": {"1"}}
	a, b, ok := victoriaCruzada(cruce, jugadores)
	if !ok || a != 1 || b != 3 {
		t.Errorf("esperaba cruce 1–3, obtuve %d–%d (ok=%v)", a, b, ok)
	}

	// Un jugador que no está en la sesión no cuenta
	fuera := url.Values{"victoria_1": {"4"}, "victoria_4": {"1"}}
	if _, _, ok := victoriaCruzada(fuera, jugadores); ok {
		t.Error("los jugadores fuera de la sesión no deben contar")
	}
}
