package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestMismoOrigen(t *testing.T) {
	casos := []struct {
		nombre, origin, referer string
		ok                      bool
	}{
		{"mismo origen", "https://mtg.example.com", "", true},
		{"otra web", "https://malo.example.org", "", false},
		{"origin null", "null", "", false},
		{"solo referer propio", "", "https://mtg.example.com/admin", true},
		{"solo referer ajeno", "", "https://malo.example.org/x", false},
		{"sin cabeceras (no navegador)", "", "", true},
	}
	for _, c := range casos {
		r := httptest.NewRequest("POST", "https://mtg.example.com/admin/jugador/toggle", nil)
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		if c.referer != "" {
			r.Header.Set("Referer", c.referer)
		}
		if got := mismoOrigen(r); got != c.ok {
			t.Errorf("%s: mismoOrigen = %v, esperaba %v", c.nombre, got, c.ok)
		}
	}
}

func TestValidacionVictoriasYColores(t *testing.T) {
	enSesion := map[int]bool{1: true, 2: true, 3: true}
	got := victoriasValidas([]string{"2", "1", "7", "2", "x", "3"}, 1, enSesion)
	if len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("victoriasValidas = %v, esperaba [2 3]", got)
	}
	colores := coloresValidos([]string{"R", "R", "X", "", "WU", "G"})
	if len(colores) != 2 || colores[0] != "R" || colores[1] != "G" {
		t.Errorf("coloresValidos = %v, esperaba [R G]", colores)
	}
}
