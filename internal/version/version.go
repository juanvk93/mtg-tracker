// Paquete version contiene la información de versión y changelog de la aplicación
package version

// Version actual de la aplicación
const Version = "1.8.2"

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
			Version: "1.8.2",
			Fecha:   "2026-07-13",
			Cambios: []string{
				"La matriz de dominancia muestra el balance neto (victorias − derrotas): +N si domina, −N si le dominan (detalle al pasar el ratón)",
			},
		},
		{
			Version: "1.8.1",
			Fecha:   "2026-07-13",
			Cambios: []string{
				"Corrección: en móvil la barra de navegación vuelve a mostrarse abajo (el desenfoque del nav la enviaba arriba)",
				"En la tabla de clasificación ahora se muestra solo el color más jugado de cada jugador, no todos",
			},
		},
		{
			Version: "1.8.0",
			Fecha:   "2026-07-13",
			Cambios: []string{
				"Nueva paleta fría (azul-noche) con acento turquesa y difuminados suaves",
				"Tarjeta-héroe del MVP en el ranking, con brillo turquesa y estadísticas destacadas",
				"En la tabla de clasificación, puntos con los colores de maná que juega cada jugador",
				"Fila del líder resaltada en turquesa y podio en oro/plata/bronce",
			},
		},
		{
			Version: "1.7.0",
			Fecha:   "2026-07-13",
			Cambios: []string{
				"Rediseño visual: acento dorado, tipografía Cinzel en titulares y tarjetas con relieve",
				"Podio en el ranking: líder destacado y posiciones 1/2/3 en oro, plata y bronce",
				"KPIs rediseñados; el líder y el mejor win rate se fusionan en una tarjeta MVP cuando es el mismo jugador",
				"Win rate mostrado como píldora de color y rachas de victorias con icono de fuego",
				"Barra de navegación translúcida, con estado activo y botón de tema con icono sol/luna",
				"Premios de temporada con la tarjeta de MVP destacada",
			},
		},
		{
			Version: "1.6.0",
			Fecha:   "2026-07-05",
			Cambios: []string{
				"Premios de temporada en Estadísticas: MVP, Villano, Mejor racha y Cabeza de turco",
				"Gráfica de evolución del win rate acumulado por sesión",
				"Meta de colores del grupo: win rate por color de la temporada",
				"Jugadores inactivos: se pueden desactivar sin borrarlos (mantienen su histórico y salen de los formularios)",
				"Arquetipo/notas por jugador en cada draft, visibles en la sesión y en el perfil",
				"Modo TV: ranking a pantalla completa con auto-refresco para proyectarlo en las quedadas",
				"Tema claro/oscuro con botón manual (además del automático del sistema)",
				"La aplicación es instalable como PWA (icono en el móvil y carga rápida sin conexión)",
			},
		},
		{
			Version: "1.5.0",
			Fecha:   "2026-07-05",
			Cambios: []string{
				"Perfil de cada jugador: ficha con récord, colores jugados, destacados e historial de todos sus drafts",
				"Selector de temporada en Ranking, Historial y Estadísticas para consultar temporadas cerradas",
				"Ranking: KPIs de líder por victorias y por win rate, y ordenación por cualquier columna",
				"El win rate ahora también se muestra en la tabla de ranking en móvil",
				"Estadísticas: gráfica de evolución del ranking (victorias acumuladas por sesión)",
				"Estadísticas: matriz head-to-head de todos contra todos (quién domina a quién)",
				"Enlace desde el ranking a la ficha de cada jugador",
				"Modo matriz para registrar resultados de una sesión de forma rápida",
				"Rendimiento: ranking, detalle de sesión y estadísticas cargan con muchas menos consultas",
			},
		},
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
