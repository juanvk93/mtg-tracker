// Paquete version contiene la información de versión y changelog de la aplicación
package version

// Version actual de la aplicación
const Version = "1.3.0"

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
				"En la sección 'Por jugador' aparecen ahora todos los jugadores que han participado en al menos una sesión",
				"Iconos SVG custom para los cinco colores de maná: sol, gota, calavera, llama, hoja",
			},
		},
		{
			Version: "1.1.0",
			Fecha:   "2026-05-07",
			Cambios: []string{
				"Estadísticas avanzadas por jugador: combinación de colores favorita, mejor sesión histórica, rival más frecuente y racha récord",
				"Distribución global de colores del grupo en la temporada activa",
				"Solo puede haber una temporada activa: al crear o reabrir una se cierra automáticamente cualquier otra",
				"Reabrir y borrar temporadas desde el panel de admin",
				"Borrar temporada exige re-confirmación de la contraseña de admin",
				"Selector visual de emojis para avatares de jugador con catálogo temático",
				"En el formulario de resultados, los rivales no participantes se ocultan automáticamente",
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
