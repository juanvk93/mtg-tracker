// Paquete version contiene la información de versión y changelog de la aplicación
package version

// Version actual de la aplicación
const Version = "1.15.0"

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
			Version: "1.15.0",
			Fecha:   "2026-10-16",
			Cambios: []string{
				"Rating Elo por temporada: nueva columna en el ranking, el modo TV y el perfil, con lo que subió o bajó en el último draft (ganar a los de arriba vale más)",
				"Sesiones: ahora se puede corregir la fecha y borrar una sesión entera (pide la contraseña de admin)",
				"Contador de vidas: vida inicial de 20, 25 o 30 y contador de veneno (a los 10 la mitad se pone en rojo), en el nuevo panel ⚙",
				"Contador de vidas: relojes de turno por jugador (⌛), tipo ajedrez, para ver quién tarda más; siguen contando aunque se bloquee el móvil",
			},
		},
		{
			Version: "1.14.0",
			Fecha:   "2026-10-16",
			Cambios: []string{
				"Gráfica de win rate: la línea de cada jugador empieza en su primera partida (antes salía al 0% en las sesiones previas)",
				"Combinación favorita, verdugo y víctima ya no cambian entre recargas cuando hay empates; un rival al 50% no sale a la vez como verdugo y víctima",
				"Rachas y listados estables aunque haya dos sesiones el mismo día",
				"Mensajes claros al repetir un año o un nombre, al editar algo que ya no existe o al crear una sesión en una temporada cerrada; reabrir una temporada borrada ya no deja sin temporada activa",
				"Colores de jugador validados, errores de base de datos registrados en vez de ignorados y 404 solo cuando algo no existe de verdad",
				"Quitar a un jugador de la sesión por error ya no borra las victorias marcadas contra él",
				"Dado: cambiar de dado durante la tirada ya no mezcla el resultado. Vidas: la cuenta de 5 en 5 se corta si el navegador pierde el dedo",
				"Servidor con timeouts de conexión y Mana Font con versión fija",
			},
		},
		{
			Version: "1.13.0",
			Fecha:   "2026-10-16",
			Cambios: []string{
				"Head to head de Estadísticas: ahora muestra solo la temporada seleccionada (antes mezclaba todas) y ya no rompe la página si falla una consulta",
				"Racha récord: un draft empatado corta la racha, igual que en el ranking y los premios",
				"Se puede vaciar una sesión: guardar sin ningún jugador marcado borra sus resultados, tras pedir confirmación",
				"Vista rápida (matriz) en una sesión nueva: empieza sin nadie marcado como 'Jugó', igual que la vista detallada",
				"El contador de vidas y el dado funcionan de verdad sin conexión (el service worker ahora controla toda la web)",
			},
		},
		{
			Version: "1.12.0",
			Fecha:   "2026-10-16",
			Cambios: []string{
				"Backup completo: ahora incluye las notas/arquetipos de cada resultado y qué jugadores están inactivos",
				"Importar ya no deja dos temporadas activas ni fusiona dos drafts del mismo día en uno",
				"Guardar resultados es todo o nada: si algo falla no se guarda nada y se avisa (antes podía quedar la sesión a medias)",
				"El servidor descarta victorias imposibles (contra uno mismo, contra quien no jugó o repetidas) y colores no válidos",
				"La vista detallada y la matriz de resultados se sincronizan al cambiar de una a otra: ya no se pierde lo editado",
				"Seguridad: avatares validados y escapados, cookie de admin más segura y rechazo de peticiones de admin enviadas desde otras webs",
			},
		},
		{
			Version: "1.11.0",
			Fecha:   "2026-10-16",
			Cambios: []string{
				"Nuevo contador de vidas para 2 jugadores cara a cara (Extras → Vidas): empieza en 20, un toque suma o resta 1 y mantener pulsado va de 5 en 5",
				"Muestra el cambio acumulado de cada jugador, se tiñe de rojo al llegar a 0 y se reinicia con doble toque en ⟲",
				"Mantiene la pantalla encendida mientras está abierto (con indicador de estado), guarda la partida en el móvil y funciona sin conexión",
			},
		},
		{
			Version: "1.10.0",
			Fecha:   "2026-10-16",
			Cambios: []string{
				"Nuevo menú 'Extras' en la barra superior con un dado (d6, d12 o d20; por defecto d20): se tira tocando la pantalla o con el botón TIRAR",
				"Nueva opción por sesión 'Cada pareja se enfrenta una sola vez': si A gana a B, la casilla 'B ganó a A' queda bloqueada en rojo (también en la vista de matriz) y el servidor rechaza victorias cruzadas",
				"Keep-alive: cada 2 minutos la app lanza una consulta ligera a la BD y se llama a sí misma por su URL pública, para que ni Render duerma el servicio ni Supabase pause el proyecto",
			},
		},
		{
			Version: "1.9.3",
			Fecha:   "2026-07-13",
			Cambios: []string{
				"Quitado el icono de televisor del botón 'Modo TV'",
				"Win rate por color: el récord ahora se muestra con V y D (ej: '58.8% · 10V–7D')",
			},
		},
		{
			Version: "1.9.2",
			Fecha:   "2026-07-13",
			Cambios: []string{
				"El 'Récord' de racha ahora se lee en claro: 'N drafts seguidos' (victorias) o 'N drafts perdidos seguidos' (derrotas), en vez de '2W consecutivas'",
			},
		},
		{
			Version: "1.9.1",
			Fecha:   "2026-07-13",
			Cambios: []string{
				"Tipos de draft: los de 4 y 5 colores se separan en 'Cuatri' y 'Penta' (antes se agrupaban como '4+')",
				"Cada tipo de draft tiene su propio color (el mismo para 'Bi' en todos los jugadores) y la barra es más vistosa",
				"La leyenda separa el tipo del número con un guion (ej: 'Mono - 3')",
			},
		},
		{
			Version: "1.9.0",
			Fecha:   "2026-07-13",
			Cambios: []string{
				"Estadísticas por jugador: nuevos datos de 'Drafts ganados', 'Posición media' y 'Tendencia' (↗/→/↘ según el win rate reciente)",
				"Los jugadores desactivados dejan de aparecer en Estadísticas (gráficas, fichas, matriz, premios y comparativas), pero se mantienen en el ranking y el histórico",
				"'Favorita': si no hay una combinación repetida, muestra el color más jugado en solitario",
				"Solo se muestra el icono del color (sin la palabra Blanco/Azul/…); el nombre queda como descripción al pasar el ratón",
			},
		},
		{
			Version: "1.8.3",
			Fecha:   "2026-07-13",
			Cambios: []string{
				"Win rate por color: ahora muestra el récord V–D y solo el icono del color",
				"Quitado 'Rival top' de las estadísticas por jugador (poco útil con un grupo fijo)",
			},
		},
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
