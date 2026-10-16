package db

import (
	"testing"

	"mtg-tracker/internal/models"
)

func TestClaveWUBRG(t *testing.T) {
	// Menos colores primero y, a igualdad, orden WUBRG (no alfabético)
	if !(claveWUBRG("W") < claveWUBRG("B,U")) {
		t.Error("una combinación de 1 color debe ir antes que una de 2")
	}
	if !(claveWUBRG("B,U") < claveWUBRG("G,R")) {
		t.Error("UB debe ir antes que RG en orden WUBRG")
	}
	if claveWUBRG("U,B") != claveWUBRG("B,U") {
		t.Error("el orden de entrada no debe importar")
	}
}

func TestElegirVerdugoYVictima(t *testing.T) {
	rival := func(id int, nombre string, v, d int) models.RivalConWinRate {
		return models.RivalConWinRate{Rival: models.Jugador{ID: id, Nombre: nombre}, Victorias: v, Derrotas: d,
			Partidas: v + d, WinRate: float64(v) / float64(v+d) * 100}
	}

	// Empate de win rate: siempre el mismo, sin importar el orden de entrada
	a, b := rival(1, "Ana", 1, 3), rival(2, "Beto", 1, 3)
	for _, orden := range [][]models.RivalConWinRate{{a, b}, {b, a}} {
		verdugo, _ := elegirVerdugoYVictima(orden)
		if verdugo == nil || verdugo.Rival.Nombre != "Ana" {
			t.Errorf("verdugo en empate = %v, quiero Ana (desempate por nombre)", verdugo)
		}
	}
	// A igual win rate, gana quien tiene más partidas
	c := rival(3, "Carla", 2, 6)
	if verdugo, _ := elegirVerdugoYVictima([]models.RivalConWinRate{a, c}); verdugo.Rival.Nombre != "Carla" {
		t.Errorf("verdugo = %s, quiero Carla (más partidas)", verdugo.Rival.Nombre)
	}
	// Único rival al 50%: ni verdugo ni víctima
	if v, w := elegirVerdugoYVictima([]models.RivalConWinRate{rival(4, "Dani", 1, 1)}); v != nil || w != nil {
		t.Error("un único rival al 50% no debe ser ni verdugo ni víctima")
	}
	// Único rival al 75%: solo víctima
	if v, w := elegirVerdugoYVictima([]models.RivalConWinRate{rival(5, "Eva", 3, 1)}); v != nil || w == nil {
		t.Error("un único rival al que se gana debe salir solo como víctima")
	}
}
