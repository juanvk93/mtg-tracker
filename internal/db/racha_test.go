package db

import (
	"math"
	"testing"
)

func TestCalcularRacha(t *testing.T) {
	casos := []struct {
		nombre    string
		historial []sesionVD // orden: de la sesión más reciente a la más antigua
		wantN     int
		wantTipo  string
	}{
		{"sin historial", nil, 0, ""},
		{"una victoria", []sesionVD{{Victorias: 2, Derrotas: 0}}, 1, "W"},
		{"una derrota", []sesionVD{{Victorias: 0, Derrotas: 2}}, 1, "L"},
		{"empate en la última", []sesionVD{{Victorias: 1, Derrotas: 1}}, 0, ""},
		{
			"racha de 3 victorias",
			[]sesionVD{{Victorias: 3, Derrotas: 1}, {Victorias: 2, Derrotas: 0}, {Victorias: 1, Derrotas: 0}, {Victorias: 0, Derrotas: 2}},
			3, "W",
		},
		{
			"racha de 2 derrotas",
			[]sesionVD{{Victorias: 0, Derrotas: 2}, {Victorias: 1, Derrotas: 3}, {Victorias: 2, Derrotas: 1}},
			2, "L",
		},
		{
			"un empate corta la racha",
			[]sesionVD{{Victorias: 2, Derrotas: 1}, {Victorias: 1, Derrotas: 1}, {Victorias: 3, Derrotas: 0}},
			1, "W",
		},
	}
	for _, c := range casos {
		gotN, gotTipo := calcularRacha(c.historial)
		if gotN != c.wantN || gotTipo != c.wantTipo {
			t.Errorf("%s: calcularRacha = (%d, %q); quiero (%d, %q)", c.nombre, gotN, gotTipo, c.wantN, c.wantTipo)
		}
	}
}

func TestCalcularDesviacionWinRate(t *testing.T) {
	casos := []struct {
		nombre string
		wrs    []float64
		want   float64
	}{
		{"vacío", nil, 0},
		{"un solo valor", []float64{50}, 0},
		{"todos iguales", []float64{40, 40, 40}, 0},
		{"máxima dispersión", []float64{0, 100}, 50}, // media 50, desviación poblacional = 50
	}
	for _, c := range casos {
		if got := CalcularDesviacionWinRate(c.wrs); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: CalcularDesviacionWinRate = %v; quiero %v", c.nombre, got, c.want)
		}
	}
}
