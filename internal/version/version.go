// Paquete version contiene la información de versión y changelog de la aplicación
package version

// Version actual de la aplicación
const Version = "1.4.0"

// EntradaChangelog representa una entrada del registro de cambios
type EntradaChangelog struct {
	Version string
	Fecha   string
	Cambios []string
}

// Changelog devuelve la lista completa de cambios ordenada de más reciente a más antiguo
func Changelog() []EntradaChangelog {
	return []EntradaChangelog{
		{
			Version: "1.4.0",
			Fecha:   "2026-05-11",
			Cambios: []string{
				"Símbolos oficiales de Magic (Mana Font) para los colores de maná y avatares de jugador",
				"Selector de 30 iconos de Mana Font para avatares: tipos de carta, habilidades y símbolos especiales (incluido Phyrexian)",
				"Corrección: win rate por color ahora es real (victorias / partidas jugadas, nunca supera 100%)",
				"Corrección: al editar una sesión ya no se pierden victorias por el orden de carga del formulario",
				"Corrección: al guardar resultados solo se borran los jugadores eliminados de la sesión, no todos",
				"Barra de navegación adaptada a móvil: los enlaces pasan a una barra fija en la parte inferior",
				"Historial muestra el ganador de cada draft (o empate si hay igualdad de victorias)",
				"Edición de descripción de sesión directamente desde el formulario de resultados",
				"Icono de ojo SVG monocromático en la columna de sesiones del panel de admin",
				"Logo de la barra de navegación reemplazado por el símbolo Planeswalker de Mana Font",
				"Botón Farewell usa el icono de Deathtouch de Mana Font",
			},
		},
		{
			Version: "1.3.0",
			Fecha:   "2026-05-07",
			Cambios: []string{
				"Resumen ejecutivo de cada temporada con podio (campeón, subcampeón, tercero), totales y clasificación final",
				"Acceso al resumen desde el panel de admin pulsando el icono de ojo en cada temporada",
				"Botón Farewell en zona peligrosa: borra todos los datos con doble confirmación (contraseña + texto 'FAREWELL')",
			},
		},
		{
			Version: "1.2.0",
			Fecha:   "2026-05-07",
			Cambios: []string{
				"Verdugo y Víctima: rivales contra los que peor y mejor win rate tiene cada jugador",
				"Promedio de victorias por sesión",
				"Constancia: medida de regularidad del win rate (alta/media/baja según desviación típica)",
				"Distribución de drafts: porcentaje de drafts mono-color, bicolor, tricolor o más",
				"Head-to-head extendido: tabla con cada encuentro mostrando los colores que jugaba cada uno",
				"En la sección 'Por jugador' aparecen todos los jugadores que han participado en al menos una sesión",
			},
		},
		{
			Version: "1.1.0",
			Fecha:   "2026-05-07",
			Cambios: []string{
				"Estadísticas avanzadas por jugador: combinación favorita, mejor sesión, rival más frecuente y racha récord",
				"Distribución global de colores del grupo en la temporada activa",
				"Solo puede haber una temporada activa a la vez",
				"Reabrir y borrar temporadas con confirmación de contraseña de admin",
				"En el formulario de resultados los rivales no participantes se ocultan automáticamente",
			},
		},
		{
			Version: "1.0.0",
			Fecha:   "2026-05-06",
			Cambios: []string{
				"Versión inicial pública",
				"Ranking de la temporada activa con win rates y rachas",
				"Historial de sesiones con detalle por jugador",
				"Estadísticas de colores por jugador y head-to-head",
				"Panel de administración protegido por contraseña",
				"Crear sesiones y registrar resultados (victorias y colores jugados)",
				"Importar y exportar backup en JSON",
			},
		},
	}
}
