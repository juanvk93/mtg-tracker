// Paquete models define las estructuras de datos del tracker de Magic
package models

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// Temporada representa un año de juego (ej: 2026)
type Temporada struct {
	ID     int
	Anio   int
	Estado string // "activa" o "cerrada"
}

// Sesion representa una sesión de draft mensual
type Sesion struct {
	ID          int
	TemporadaID int
	Fecha       time.Time
	Descripcion string
}

// Jugador representa un participante del grupo
type Jugador struct {
	ID     int
	Nombre string
	Color  string // color identificativo en hex (ej: "#e63946")
	Avatar string // emoji o iniciales
	Activo bool   // false = jugador retirado (mantiene histórico, sale de los formularios)
}

// ResultadoSesion representa el resultado de un jugador en una sesión
type ResultadoSesion struct {
	ID        int
	SesionID  int
	JugadorID int
	Victorias []int    // IDs de jugadores a los que ganó
	Derrotas  []int    // IDs de jugadores contra los que perdió
	Colores   []string // colores jugados: W, U, B, R, G
}

// FilaRanking es una fila del ranking calculado
type FilaRanking struct {
	Jugador        Jugador
	Partidas       int
	Victorias      int
	Derrotas       int
	WinRate        float64
	RachaActual    int    // longitud de la racha actual (siempre >= 0)
	RachaTipo      string // "W" o "L"
	MejorRacha     int    // racha de victorias más larga de la temporada
	ColorPrincipal string // color de maná más jugado en la temporada ("" si ninguno)
}

// EstadisticaColor contiene estadísticas por color para un jugador
type EstadisticaColor struct {
	Color     string
	Nombre    string
	Veces     int // nº de drafts (resultados) en los que jugó este color
	Partidas  int // partidas individuales jugadas con este color (victorias + derrotas)
	Victorias int
	WinRate   float64
}

// CombinacionColor representa una combinación de colores (1-5 colores) y su frecuencia
type CombinacionColor struct {
	Colores []string // ej: ["U", "B"] para Azul/Negro
	Veces   int      // cuántas veces ha jugado esta combinación
}

// MejorSesion representa la mejor sesión de un jugador
type MejorSesion struct {
	SesionID    int
	Fecha       time.Time
	Descripcion string
	Victorias   int
	Derrotas    int
}

// RivalFrecuente representa el rival más jugado por un jugador
type RivalFrecuente struct {
	Rival     Jugador
	Partidas  int // total de partidas contra ese rival
	Victorias int // victorias del jugador sobre ese rival
	Derrotas  int // derrotas del jugador contra ese rival
}

// RivalConWinRate extiende RivalFrecuente con win rate y categoría
type RivalConWinRate struct {
	Rival     Jugador
	Partidas  int
	Victorias int
	Derrotas  int
	WinRate   float64 // % de victorias del jugador sobre este rival
}

// DistribucionDrafts contiene el conteo de drafts mono/bi/tri-color de un jugador
type DistribucionDrafts struct {
	Mono     int // drafts de 1 color
	Bicolor  int // drafts de 2 colores
	Tricolor int // drafts de 3 colores
	MasColor int // drafts de 4 o 5 colores
	Total    int
}

// H2HExtendido contiene el detalle de los enfrentamientos directos entre dos jugadores
type H2HExtendido struct {
	HeadToHead
	Encuentros []EncuentroH2H // un encuentro por sesión donde jugaron juntos
}

// EncuentroH2H representa un enfrentamiento concreto entre dos jugadores en una sesión
type EncuentroH2H struct {
	SesionID     int
	Fecha        time.Time
	Descripcion  string
	ColoresJ1    []string // colores que jugaba J1 en esa sesión
	ColoresJ2    []string // colores que jugaba J2 en esa sesión
	GanadasPorJ1 int      // partidas que J1 ganó a J2 en esa sesión
	GanadasPorJ2 int      // partidas que J2 ganó a J1 en esa sesión
}

// RachaRecord representa la mejor racha histórica de un jugador
type RachaRecord struct {
	Longitud int    // número de sesiones consecutivas ganadas
	Tipo     string // "W" (victorias) o "L" (derrotas)
}

// EstadisticasJugador agrupa todas las estadísticas individuales de un jugador
type EstadisticasJugador struct {
	Jugador            Jugador
	Colores            []EstadisticaColor
	CombinacionTop     *CombinacionColor // puede ser nil si no hay datos
	MejorSesion        *MejorSesion      // puede ser nil
	RivalTop           *RivalFrecuente   // rival más frecuente; puede ser nil
	Verdugo            *RivalConWinRate  // rival contra quien peor win rate tiene; puede ser nil
	Victima            *RivalConWinRate  // rival contra quien mejor win rate tiene; puede ser nil
	RachaRecord        RachaRecord
	PromedioVictorias  float64 // victorias / sesiones jugadas
	SesionesJugadas    int
	DesviacionWinRate  float64 // 0 = muy constante, valores altos = irregular
	DistribucionDrafts DistribucionDrafts
}

// DistribucionColor cuenta cuántas veces se ha jugado cada color en el grupo
type DistribucionColor struct {
	Color      string
	Nombre     string
	Veces      int
	Porcentaje float64
}

// DraftJugador resume la participación de un jugador en una sesión (para su perfil).
type DraftJugador struct {
	SesionID    int
	Fecha       time.Time
	Descripcion string
	Victorias   int
	Derrotas    int
	Colores     []string
	Notas       string
	Top         bool // fue quien más victorias tuvo esa sesión (empatado o en solitario)
}

// ResumenTemporada contiene el resumen ejecutivo de una temporada
type ResumenTemporada struct {
	Temporada      Temporada
	TotalSesiones  int
	TotalJugadores int
	TotalPartidas  int
	Campeon        *FilaRanking // jugador con más victorias
	Subcampeon     *FilaRanking // segundo en el ranking
	Tercero        *FilaRanking // tercero en el ranking
	ColorDominante string       // color más jugado de la temporada
	Ranking        []FilaRanking
}

// HeadToHead contiene el historial entre dos jugadores
type HeadToHead struct {
	Jugador1    Jugador
	Jugador2    Jugador
	VictoriasJ1 int
	VictoriasJ2 int
	Total       int
}

// SesionDetalle contiene el detalle de una sesión con resultados
type SesionDetalle struct {
	Sesion     Sesion
	Temporada  Temporada
	Resultados []ResultadoDetalle
}

// ResultadoDetalle contiene el resultado de un jugador con nombres resueltos
type ResultadoDetalle struct {
	Jugador        Jugador
	Victorias      []Jugador
	Derrotas       []Jugador
	Colores        []string
	Notas          string // arquetipo/mazo o comentario libre de ese draft
	TotalVictorias int
	TotalDerrotas  int
}

// IconoJugador representa un icono de Mana Font disponible como avatar
type IconoJugador struct {
	Clase  string // clase CSS de Mana Font (ej: "ms-planeswalker")
	Nombre string // nombre legible en español
}

// IconosJugador devuelve un catálogo de 30 iconos de Mana Font para usar como avatar.
// No incluye los cinco colores de maná (W, U, B, R, G) para evitar confusión con los badges.
func IconosJugador() []IconoJugador {
	return []IconoJugador{
		// Tipos de carta
		{Clase: "ms-planeswalker", Nombre: "Planeswalker"},
		{Clase: "ms-creature", Nombre: "Criatura"},
		{Clase: "ms-instant", Nombre: "Instantáneo"},
		{Clase: "ms-sorcery", Nombre: "Conjuro"},
		{Clase: "ms-enchantment", Nombre: "Encantamiento"},
		{Clase: "ms-artifact", Nombre: "Artefacto"},
		{Clase: "ms-land", Nombre: "Tierra"},
		{Clase: "ms-token", Nombre: "Token"},
		{Clase: "ms-tribal", Nombre: "Tribal"},
		// Habilidades
		{Clase: "ms-ability-deathtouch", Nombre: "Toque mortal"},
		{Clase: "ms-ability-flying", Nombre: "Volar"},
		{Clase: "ms-ability-haste", Nombre: "Prisa"},
		{Clase: "ms-ability-hexproof", Nombre: "Antimaleficio"},
		{Clase: "ms-ability-lifelink", Nombre: "Vínculo vital"},
		{Clase: "ms-ability-trample", Nombre: "Arroyar"},
		{Clase: "ms-ability-vigilance", Nombre: "Vigilancia"},
		{Clase: "ms-ability-menace", Nombre: "Amenaza"},
		{Clase: "ms-ability-first-strike", Nombre: "Daño primero"},
		{Clase: "ms-ability-double-strike", Nombre: "Doble daño"},
		{Clase: "ms-ability-flash", Nombre: "Destello"},
		{Clase: "ms-ability-prowess", Nombre: "Proeza"},
		{Clase: "ms-ability-defender", Nombre: "Defensor"},
		// Símbolos especiales
		{Clase: "ms-p", Nombre: "Phyrexian"},
		{Clase: "ms-chaos", Nombre: "Caos"},
		{Clase: "ms-saga", Nombre: "Saga"},
		{Clase: "ms-dfc-spark", Nombre: "Chispa"},
		{Clase: "ms-dfc-night", Nombre: "Noche"},
		{Clase: "ms-dfc-moon", Nombre: "Luna"},
		{Clase: "ms-snow", Nombre: "Nieve"},
		{Clase: "ms-infinity", Nombre: "Infinito"},
	}
}

// NombreColor devuelve el nombre legible de un color de Magic
func NombreColor(codigo string) string {
	nombres := map[string]string{
		"W": "Blanco",
		"U": "Azul",
		"B": "Negro",
		"R": "Rojo",
		"G": "Verde",
	}
	if n, ok := nombres[codigo]; ok {
		return n
	}
	return codigo
}

// EmojiColor devuelve un icono de Mana Font (HTML) que representa el color de Magic.
// Usa la fuente tipográfica "Mana Font" de Andrew Gioia que contiene los símbolos
// oficiales de los colores de maná. Requiere cargar el CSS de Mana Font en el <head>.
// La clase ms-cost añade el círculo de fondo como en las cartas de Magic.
func EmojiColor(codigo string) template.HTML {
	codigos := map[string]bool{"W": true, "U": true, "B": true, "R": true, "G": true}
	if !codigos[codigo] {
		return template.HTML(codigo)
	}
	letra := strings.ToLower(codigo)
	return template.HTML(fmt.Sprintf(
		`<i class="ms ms-%s ms-cost" aria-hidden="true"></i>`,
		letra,
	))
}
