// graficas.go — Precalcula la geometría de la gráfica de evolución del ranking
// (victorias acumuladas por sesión) para que la plantilla solo tenga que pintar el SVG.
package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"mtg-tracker/internal/db"
	"mtg-tracker/internal/models"
)

// PuntoSVG es un punto en coordenadas del lienzo SVG.
type PuntoSVG struct {
	X float64
	Y float64
}

// SerieEvolucion es la línea de un jugador (victorias acumuladas por sesión).
type SerieEvolucion struct {
	Jugador   models.Jugador
	Puntos    []PuntoSVG
	PuntosStr string // atributo "points" de la polilínea
	Total     int
}

// EtiquetaEje es una etiqueta posicionada de un eje.
type EtiquetaEje struct {
	X     float64
	Y     float64
	Texto string
}

// GraficaEvolucion contiene todo lo necesario para pintar el SVG.
type GraficaEvolucion struct {
	Ancho  float64
	Alto   float64
	Series []SerieEvolucion
	EjeX   []EtiquetaEje
	EjeY   []EtiquetaEje
	PlotX0 float64
	PlotX1 float64
	Vacia  bool
}

// construirGraficaEvolucion transforma las filas cronológicas en geometría SVG.
func construirGraficaEvolucion(filas []db.EvolucionFila) GraficaEvolucion {
	g := GraficaEvolucion{Ancho: 680, Alto: 260}
	padLeft, padRight, padTop, padBottom := 34.0, 26.0, 16.0, 30.0
	if len(filas) == 0 {
		g.Vacia = true
		return g
	}

	// Sesiones únicas en orden cronológico (una fila representativa por sesión).
	var sesiones []db.EvolucionFila
	posSesion := map[int]int{}
	for _, f := range filas {
		if _, ok := posSesion[f.SesionID]; !ok {
			posSesion[f.SesionID] = len(sesiones)
			sesiones = append(sesiones, f)
		}
	}
	n := len(sesiones)

	// Acumulado de victorias por jugador y sesión, en orden de aparición.
	type acc struct {
		jug models.Jugador
		cum []int
	}
	orden := []int{}
	jugMap := map[int]*acc{}
	for _, f := range filas {
		a, ok := jugMap[f.JugadorID]
		if !ok {
			a = &acc{
				jug: models.Jugador{ID: f.JugadorID, Nombre: f.Nombre, Color: f.Color, Avatar: f.Avatar},
				cum: make([]int, n),
			}
			jugMap[f.JugadorID] = a
			orden = append(orden, f.JugadorID)
		}
		a.cum[posSesion[f.SesionID]] += f.VictoriasSesion
	}

	// Convertir a suma acumulada y hallar el máximo para escalar el eje Y.
	maxY := 1
	for _, id := range orden {
		a := jugMap[id]
		suma := 0
		for i := 0; i < n; i++ {
			suma += a.cum[i]
			a.cum[i] = suma
			if suma > maxY {
				maxY = suma
			}
		}
	}

	plotW := g.Ancho - padLeft - padRight
	plotH := g.Alto - padTop - padBottom
	xDe := func(i int) float64 {
		if n == 1 {
			return padLeft + plotW/2
		}
		return padLeft + float64(i)*plotW/float64(n-1)
	}
	yDe := func(v int) float64 {
		return padTop + plotH - float64(v)/float64(maxY)*plotH
	}

	for _, id := range orden {
		a := jugMap[id]
		s := SerieEvolucion{Jugador: a.jug, Total: a.cum[n-1]}
		var b strings.Builder
		for i := 0; i < n; i++ {
			x, y := redondea(xDe(i)), redondea(yDe(a.cum[i]))
			s.Puntos = append(s.Puntos, PuntoSVG{X: x, Y: y})
			if i > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "%g,%g", x, y)
		}
		s.PuntosStr = b.String()
		g.Series = append(g.Series, s)
	}

	for i := 0; i < n; i++ {
		g.EjeX = append(g.EjeX, EtiquetaEje{X: redondea(xDe(i)), Y: g.Alto - padBottom + 14, Texto: sesiones[i].Fecha.Format("02/01")})
	}
	for _, v := range gridValores(maxY) {
		g.EjeY = append(g.EjeY, EtiquetaEje{X: padLeft, Y: redondea(yDe(v)), Texto: strconv.Itoa(v)})
	}
	g.PlotX0 = padLeft
	g.PlotX1 = g.Ancho - padRight
	return g
}

// construirGraficaWinRate dibuja la evolución del win rate acumulado (0-100%) por sesión.
func construirGraficaWinRate(filas []db.EvolucionFila) GraficaEvolucion {
	g := GraficaEvolucion{Ancho: 680, Alto: 260}
	padLeft, padRight, padTop, padBottom := 40.0, 26.0, 16.0, 30.0
	if len(filas) == 0 {
		g.Vacia = true
		return g
	}

	var sesiones []db.EvolucionFila
	posSesion := map[int]int{}
	for _, f := range filas {
		if _, ok := posSesion[f.SesionID]; !ok {
			posSesion[f.SesionID] = len(sesiones)
			sesiones = append(sesiones, f)
		}
	}
	n := len(sesiones)

	type acc struct {
		jug        models.Jugador
		cumV, cumD []int
	}
	orden := []int{}
	jugMap := map[int]*acc{}
	for _, f := range filas {
		a, ok := jugMap[f.JugadorID]
		if !ok {
			a = &acc{
				jug:  models.Jugador{ID: f.JugadorID, Nombre: f.Nombre, Color: f.Color, Avatar: f.Avatar},
				cumV: make([]int, n),
				cumD: make([]int, n),
			}
			jugMap[f.JugadorID] = a
			orden = append(orden, f.JugadorID)
		}
		p := posSesion[f.SesionID]
		a.cumV[p] += f.VictoriasSesion
		a.cumD[p] += f.DerrotasSesion
	}

	plotW := g.Ancho - padLeft - padRight
	plotH := g.Alto - padTop - padBottom
	xDe := func(i int) float64 {
		if n == 1 {
			return padLeft + plotW/2
		}
		return padLeft + float64(i)*plotW/float64(n-1)
	}
	yDe := func(pct float64) float64 { return padTop + plotH - pct/100*plotH }

	for _, id := range orden {
		a := jugMap[id]
		s := SerieEvolucion{Jugador: a.jug}
		var b strings.Builder
		sumV, sumD := 0, 0
		var ultimoPct float64
		for i := 0; i < n; i++ {
			sumV += a.cumV[i]
			sumD += a.cumD[i]
			pct := 0.0
			if sumV+sumD > 0 {
				pct = float64(sumV) / float64(sumV+sumD) * 100
			}
			ultimoPct = pct
			x, y := redondea(xDe(i)), redondea(yDe(pct))
			s.Puntos = append(s.Puntos, PuntoSVG{X: x, Y: y})
			if i > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "%g,%g", x, y)
		}
		s.PuntosStr = b.String()
		s.Total = int(ultimoPct + 0.5)
		g.Series = append(g.Series, s)
	}

	for i := 0; i < n; i++ {
		g.EjeX = append(g.EjeX, EtiquetaEje{X: redondea(xDe(i)), Y: g.Alto - padBottom + 14, Texto: sesiones[i].Fecha.Format("02/01")})
	}
	for _, v := range []int{0, 50, 100} {
		g.EjeY = append(g.EjeY, EtiquetaEje{X: padLeft, Y: redondea(yDe(float64(v))), Texto: strconv.Itoa(v)})
	}
	g.PlotX0 = padLeft
	g.PlotX1 = g.Ancho - padRight
	return g
}

// gridValores devuelve los valores de las líneas guía del eje Y.
func gridValores(maxY int) []int {
	if maxY <= 2 {
		vals := make([]int, 0, maxY+1)
		for v := 0; v <= maxY; v++ {
			vals = append(vals, v)
		}
		return vals
	}
	return []int{0, maxY / 2, maxY}
}

// redondea deja un decimal para no ensuciar el SVG con floats largos.
func redondea(f float64) float64 {
	return float64(int(f*10)) / 10
}

// ---------- Premios de temporada ----------

// Premio es una insignia de temporada derivada del ranking.
type Premio struct {
	Emoji   string
	Titulo  string
	Jugador models.Jugador
	Valor   string
}

// construirPremios deriva los premios del ranking ya calculado (sin consultas extra).
func construirPremios(ranking []models.FilaRanking) []Premio {
	var mvp, villano, racha, saco *models.FilaRanking
	for i := range ranking {
		f := &ranking[i]
		if f.Victorias > 0 && (villano == nil || f.Victorias > villano.Victorias) {
			villano = f
		}
		if f.Partidas >= 3 && (mvp == nil || f.WinRate > mvp.WinRate) {
			mvp = f
		}
		if f.MejorRacha > 0 && (racha == nil || f.MejorRacha > racha.MejorRacha) {
			racha = f
		}
		if f.Derrotas > 0 && (saco == nil || f.Derrotas > saco.Derrotas) {
			saco = f
		}
	}
	// Si nadie llega al mínimo de 3 partidas, el MVP es el mejor win rate con juego.
	if mvp == nil {
		for i := range ranking {
			f := &ranking[i]
			if f.Partidas > 0 && (mvp == nil || f.WinRate > mvp.WinRate) {
				mvp = f
			}
		}
	}

	var premios []Premio
	if mvp != nil {
		premios = append(premios, Premio{Emoji: "🏆", Titulo: "MVP · mejor win rate", Jugador: mvp.Jugador, Valor: fmt.Sprintf("%.1f%%", mvp.WinRate)})
	}
	if villano != nil {
		premios = append(premios, Premio{Emoji: "😈", Titulo: "Villano · más derrotas infligidas", Jugador: villano.Jugador, Valor: fmt.Sprintf("%d victorias", villano.Victorias)})
	}
	if racha != nil {
		premios = append(premios, Premio{Emoji: "🔥", Titulo: "Mejor racha de victorias", Jugador: racha.Jugador, Valor: fmt.Sprintf("%d seguidas", racha.MejorRacha)})
	}
	if saco != nil {
		premios = append(premios, Premio{Emoji: "💀", Titulo: "Cabeza de turco · más derrotas", Jugador: saco.Jugador, Valor: fmt.Sprintf("%d derrotas", saco.Derrotas)})
	}
	return premios
}
