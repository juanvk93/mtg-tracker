package handlers

import (
	"testing"
	"time"

	"mtg-tracker/internal/db"
)

func TestConstruirGraficaEvolucionVacia(t *testing.T) {
	g := construirGraficaEvolucion(nil)
	if !g.Vacia {
		t.Fatal("una gráfica sin filas debería marcarse como Vacia")
	}
}

func TestConstruirGraficaEvolucion(t *testing.T) {
	f1 := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	f2 := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	filas := []db.EvolucionFila{
		{JugadorID: 1, Nombre: "Ana", Color: "#e63946", SesionID: 10, Fecha: f1, VictoriasSesion: 2},
		{JugadorID: 2, Nombre: "Bea", Color: "#2a9d5c", SesionID: 10, Fecha: f1, VictoriasSesion: 0},
		{JugadorID: 1, Nombre: "Ana", Color: "#e63946", SesionID: 11, Fecha: f2, VictoriasSesion: 1},
		{JugadorID: 2, Nombre: "Bea", Color: "#2a9d5c", SesionID: 11, Fecha: f2, VictoriasSesion: 3},
	}

	g := construirGraficaEvolucion(filas)
	if g.Vacia {
		t.Fatal("no debería estar vacía")
	}
	if len(g.Series) != 2 {
		t.Fatalf("quiero 2 series, tengo %d", len(g.Series))
	}
	if len(g.EjeX) != 2 {
		t.Fatalf("quiero 2 etiquetas de eje X (2 sesiones), tengo %d", len(g.EjeX))
	}

	// La serie está en orden de aparición: Ana primero. Acumulados: Ana 2→3, Bea 0→3.
	if g.Series[0].Jugador.Nombre != "Ana" || g.Series[0].Total != 3 {
		t.Errorf("serie 0 = (%q, total %d); quiero (Ana, 3)", g.Series[0].Jugador.Nombre, g.Series[0].Total)
	}
	if g.Series[1].Jugador.Nombre != "Bea" || g.Series[1].Total != 3 {
		t.Errorf("serie 1 = (%q, total %d); quiero (Bea, 3)", g.Series[1].Jugador.Nombre, g.Series[1].Total)
	}
	for _, s := range g.Series {
		if len(s.Puntos) != 2 {
			t.Errorf("serie %s: %d puntos, quiero 2", s.Jugador.Nombre, len(s.Puntos))
		}
		if s.PuntosStr == "" {
			t.Errorf("serie %s: PuntosStr no debería estar vacío", s.Jugador.Nombre)
		}
	}
}
