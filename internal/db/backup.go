// backup.go — Exportación e importación de datos en formato JSON
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"mtg-tracker/internal/models"
)

// VersionBackup es la versión del formato. 1.1 añade jugadores.activo y resultados.notas;
// los backups 1.0 se siguen importando (jugadores activos y notas vacías).
const VersionBackup = "1.1"

// ========== ESTRUCTURAS DEL BACKUP ==========

// Backup representa el volcado completo de la base de datos
type Backup struct {
	Version     string            `json:"version"`
	FechaExport time.Time         `json:"fecha_export"`
	Jugadores   []BackupJugador   `json:"jugadores"`
	Temporadas  []BackupTemporada `json:"temporadas"`
	Sesiones    []BackupSesion    `json:"sesiones"`
	Resultados  []BackupResultado `json:"resultados"`
}

// BackupJugador representa un jugador en el backup
type BackupJugador struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	Color  string `json:"color"`
	Avatar string `json:"avatar"`
	// Activo es puntero para distinguir "no viene en el backup" (1.0 → activo) de false
	Activo *bool `json:"activo,omitempty"`
}

// BackupTemporada representa una temporada en el backup
type BackupTemporada struct {
	ID     int    `json:"id"`
	Anio   int    `json:"anio"`
	Estado string `json:"estado"`
}

// BackupSesion representa una sesión en el backup
type BackupSesion struct {
	ID                  int    `json:"id"`
	TemporadaID         int    `json:"temporada_id"`
	Fecha               string `json:"fecha"` // "2006-01-02"
	Descripcion         string `json:"descripcion"`
	EnfrentamientoUnico bool   `json:"enfrentamiento_unico,omitempty"`
}

// BackupResultado representa el resultado completo de un jugador en una sesión
type BackupResultado struct {
	SesionID  int      `json:"sesion_id"`
	JugadorID int      `json:"jugador_id"`
	Victorias []int    `json:"victorias"` // IDs de rivales vencidos
	Colores   []string `json:"colores"`   // códigos: W, U, B, R, G
	Notas     string   `json:"notas,omitempty"`
}

// ========== EXPORTAR ==========

// ExportarBackup lee toda la base de datos y devuelve un Backup serializado como JSON
func ExportarBackup(db *sql.DB) ([]byte, error) {
	backup := Backup{
		Version:     VersionBackup,
		FechaExport: time.Now().UTC(),
	}

	// Jugadores
	if err := func() error {
		rows, err := db.Query(`SELECT id, nombre, color, avatar, activo FROM jugadores ORDER BY id`)
		if err != nil {
			return fmt.Errorf("exportar jugadores: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var j BackupJugador
			var activo bool
			if err := rows.Scan(&j.ID, &j.Nombre, &j.Color, &j.Avatar, &activo); err != nil {
				return err
			}
			j.Activo = &activo
			backup.Jugadores = append(backup.Jugadores, j)
		}
		return rows.Err()
	}(); err != nil {
		return nil, err
	}

	// Temporadas
	if err := func() error {
		rows, err := db.Query(`SELECT id, anio, estado FROM temporadas ORDER BY id`)
		if err != nil {
			return fmt.Errorf("exportar temporadas: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var t BackupTemporada
			if err := rows.Scan(&t.ID, &t.Anio, &t.Estado); err != nil {
				return err
			}
			backup.Temporadas = append(backup.Temporadas, t)
		}
		return rows.Err()
	}(); err != nil {
		return nil, err
	}

	// Sesiones
	if err := func() error {
		rows, err := db.Query(`SELECT id, temporada_id, fecha, descripcion, enfrentamiento_unico FROM sesiones ORDER BY id`)
		if err != nil {
			return fmt.Errorf("exportar sesiones: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var s BackupSesion
			var fecha time.Time
			if err := rows.Scan(&s.ID, &s.TemporadaID, &fecha, &s.Descripcion, &s.EnfrentamientoUnico); err != nil {
				return err
			}
			s.Fecha = fecha.Format("2006-01-02")
			backup.Sesiones = append(backup.Sesiones, s)
		}
		return rows.Err()
	}(); err != nil {
		return nil, err
	}

	// Resultados: primero obtener la lista de (id, sesion_id, jugador_id)
	type resultadoRaw struct {
		id        int
		sesionID  int
		jugadorID int
		notas     string
	}
	var rawResultados []resultadoRaw

	if err := func() error {
		rows, err := db.Query(`SELECT id, sesion_id, jugador_id, notas FROM resultados ORDER BY sesion_id, jugador_id`)
		if err != nil {
			return fmt.Errorf("exportar resultados: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var r resultadoRaw
			if err := rows.Scan(&r.id, &r.sesionID, &r.jugadorID, &r.notas); err != nil {
				return err
			}
			rawResultados = append(rawResultados, r)
		}
		return rows.Err()
	}(); err != nil {
		return nil, err
	}

	// Para cada resultado, obtener victorias y colores con queries independientes
	for _, raw := range rawResultados {
		br := BackupResultado{
			SesionID:  raw.sesionID,
			JugadorID: raw.jugadorID,
			Notas:     raw.notas,
			Victorias: []int{},    // inicializar como array vacío, nunca null en JSON
			Colores:   []string{}, // ídem
		}

		// Victorias
		if err := func() error {
			vRows, err := db.Query(`SELECT rival_id FROM victorias WHERE resultado_id=$1 ORDER BY rival_id`, raw.id)
			if err != nil {
				return fmt.Errorf("exportar victorias resultado %d: %w", raw.id, err)
			}
			defer vRows.Close()
			for vRows.Next() {
				var rivalID int
				if err := vRows.Scan(&rivalID); err != nil {
					return err
				}
				br.Victorias = append(br.Victorias, rivalID)
			}
			return vRows.Err()
		}(); err != nil {
			return nil, err
		}

		// Colores
		if err := func() error {
			cRows, err := db.Query(`SELECT color FROM colores_jugados WHERE resultado_id=$1 ORDER BY color`, raw.id)
			if err != nil {
				return fmt.Errorf("exportar colores resultado %d: %w", raw.id, err)
			}
			defer cRows.Close()
			for cRows.Next() {
				var color string
				if err := cRows.Scan(&color); err != nil {
					return err
				}
				br.Colores = append(br.Colores, color)
			}
			return cRows.Err()
		}(); err != nil {
			return nil, err
		}

		backup.Resultados = append(backup.Resultados, br)
	}

	// Garantizar arrays vacíos en lugar de null si no hay datos
	if backup.Jugadores == nil {
		backup.Jugadores = []BackupJugador{}
	}
	if backup.Temporadas == nil {
		backup.Temporadas = []BackupTemporada{}
	}
	if backup.Sesiones == nil {
		backup.Sesiones = []BackupSesion{}
	}
	if backup.Resultados == nil {
		backup.Resultados = []BackupResultado{}
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("serializar backup: %w", err)
	}
	return data, nil
}

// ========== IMPORTAR ==========

// ResultadoImport contiene el resultado de una importación
type ResultadoImport struct {
	JugadoresInsertados  int
	TemporadasInsertadas int
	SesionesInsertadas   int
	ResultadosInsertados int
	Advertencias         []string
}

// ImportarBackup restaura datos desde un JSON de backup.
// Usa una estrategia "upsert por contenido": inserta solo lo que no existe,
// remapeando IDs para evitar colisiones con datos existentes.
func ImportarBackup(db *sql.DB, data []byte) (*ResultadoImport, error) {
	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("JSON inválido: %w", err)
	}

	if backup.Version == "" {
		return nil, fmt.Errorf("el archivo no parece un backup válido de MTG Tracker (falta campo 'version')")
	}

	resultado := &ResultadoImport{}

	// Toda la importación en una sola transacción
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Mapas de ID-antiguo → ID-nuevo para resolver referencias
	mapaJugadores := make(map[int]int)  // ID backup → ID en BD
	mapaTemporadas := make(map[int]int) // ID backup → ID en BD
	mapaSesiones := make(map[int]int)   // ID backup → ID en BD

	// ── Jugadores ──
	// Estrategia: buscar por nombre. Si existe, usar su ID. Si no, insertar.
	for _, j := range backup.Jugadores {
		var idExistente int
		err := tx.QueryRow(`SELECT id FROM jugadores WHERE nombre=$1`, j.Nombre).Scan(&idExistente)
		if err == sql.ErrNoRows {
			// Insertar nuevo jugador
			activo := j.Activo == nil || *j.Activo
			err = tx.QueryRow(
				`INSERT INTO jugadores (nombre, color, avatar, activo) VALUES ($1, $2, $3, $4) RETURNING id`,
				j.Nombre, j.Color, models.NormalizarAvatar(j.Avatar), activo,
			).Scan(&idExistente)
			if err != nil {
				return nil, fmt.Errorf("insertar jugador '%s': %w", j.Nombre, err)
			}
			resultado.JugadoresInsertados++
		} else if err != nil {
			return nil, fmt.Errorf("buscar jugador '%s': %w", j.Nombre, err)
		} else {
			resultado.Advertencias = append(resultado.Advertencias,
				fmt.Sprintf("Jugador '%s' ya existe, se usa el existente", j.Nombre))
		}
		mapaJugadores[j.ID] = idExistente
	}

	// ── Temporadas ──
	// Estrategia: buscar por año. Si existe, usar su ID. Si no, insertar.
	// Solo puede haber UNA temporada activa: si ya hay una (en la BD o insertada antes
	// en esta importación), las demás que vengan como activas se importan cerradas.
	var hayActiva bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM temporadas WHERE estado='activa')`).Scan(&hayActiva); err != nil {
		return nil, fmt.Errorf("comprobar temporada activa: %w", err)
	}
	for _, t := range backup.Temporadas {
		var idExistente int
		err := tx.QueryRow(`SELECT id FROM temporadas WHERE anio=$1`, t.Anio).Scan(&idExistente)
		if err == sql.ErrNoRows {
			estado := "cerrada"
			if t.Estado == "activa" {
				if hayActiva {
					resultado.Advertencias = append(resultado.Advertencias,
						fmt.Sprintf("Temporada %d importada como cerrada: ya hay otra temporada activa", t.Anio))
				} else {
					estado = "activa"
					hayActiva = true
				}
			}
			err = tx.QueryRow(
				`INSERT INTO temporadas (anio, estado) VALUES ($1, $2) RETURNING id`,
				t.Anio, estado,
			).Scan(&idExistente)
			if err != nil {
				return nil, fmt.Errorf("insertar temporada %d: %w", t.Anio, err)
			}
			resultado.TemporadasInsertadas++
		} else if err != nil {
			return nil, fmt.Errorf("buscar temporada %d: %w", t.Anio, err)
		} else {
			resultado.Advertencias = append(resultado.Advertencias,
				fmt.Sprintf("Temporada %d ya existe, se usa la existente", t.Anio))
		}
		mapaTemporadas[t.ID] = idExistente
	}

	// ── Sesiones ──
	// Estrategia: buscar una sesión que YA existiera antes de importar con la misma
	// temporada y fecha (prefiriendo la misma descripción) y que no se haya asignado ya a
	// otra sesión del backup. Así dos drafts del mismo día no se fusionan en uno, y
	// reimportar el mismo backup sigue sin duplicar nada. Si no hay candidata, se inserta.
	var maxSesionPrevia int
	if err := tx.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM sesiones`).Scan(&maxSesionPrevia); err != nil {
		return nil, fmt.Errorf("leer sesiones existentes: %w", err)
	}
	sesionAsignada := make(map[int]bool) // ID en BD → ya usada por otra sesión del backup
	for _, s := range backup.Sesiones {
		nuevaTemporadaID, ok := mapaTemporadas[s.TemporadaID]
		if !ok {
			resultado.Advertencias = append(resultado.Advertencias,
				fmt.Sprintf("Sesión ID %d omitida: temporada ID %d no encontrada en el backup", s.ID, s.TemporadaID))
			continue
		}

		idExistente, err := buscarSesionExistente(tx, nuevaTemporadaID, s.Fecha, s.Descripcion, maxSesionPrevia, sesionAsignada)
		if err == sql.ErrNoRows {
			err = tx.QueryRow(
				`INSERT INTO sesiones (temporada_id, fecha, descripcion, enfrentamiento_unico) VALUES ($1, $2, $3, $4) RETURNING id`,
				nuevaTemporadaID, s.Fecha, s.Descripcion, s.EnfrentamientoUnico,
			).Scan(&idExistente)
			if err != nil {
				return nil, fmt.Errorf("insertar sesión %s: %w", s.Fecha, err)
			}
			resultado.SesionesInsertadas++
		} else if err != nil {
			return nil, fmt.Errorf("buscar sesión %s: %w", s.Fecha, err)
		} else {
			resultado.Advertencias = append(resultado.Advertencias,
				fmt.Sprintf("Sesión %s ya existe: se reutiliza y solo se añaden los resultados que le falten", s.Fecha))
		}
		sesionAsignada[idExistente] = true
		mapaSesiones[s.ID] = idExistente
	}

	// ── Resultados ──
	// Estrategia: buscar por (sesion_id, jugador_id). Si existe, omitir. Si no, insertar.
	// Participantes de cada sesión según el backup: una victoria solo es válida contra
	// otro jugador que también jugó esa sesión (si no, el ranking descuadra).
	participa := make(map[[2]int]bool) // (sesión, jugador) con IDs del backup
	for _, r := range backup.Resultados {
		participa[[2]int{r.SesionID, r.JugadorID}] = true
	}
	for _, r := range backup.Resultados {
		nuevaSesionID, ok := mapaSesiones[r.SesionID]
		if !ok {
			resultado.Advertencias = append(resultado.Advertencias,
				fmt.Sprintf("Resultado de jugador ID %d en sesión ID %d omitido: sesión no encontrada",
					r.JugadorID, r.SesionID))
			continue
		}
		nuevoJugadorID, ok := mapaJugadores[r.JugadorID]
		if !ok {
			resultado.Advertencias = append(resultado.Advertencias,
				fmt.Sprintf("Resultado omitido: jugador con ID %d (en el archivo de backup) no fue encontrado",
					r.JugadorID))
			continue
		}

		// Comprobar si ya existe este resultado
		var resultadoID int
		err := tx.QueryRow(
			`SELECT id FROM resultados WHERE sesion_id=$1 AND jugador_id=$2`,
			nuevaSesionID, nuevoJugadorID,
		).Scan(&resultadoID)

		if err == sql.ErrNoRows {
			// Insertar resultado
			err = tx.QueryRow(
				`INSERT INTO resultados (sesion_id, jugador_id, notas) VALUES ($1, $2, $3) RETURNING id`,
				nuevaSesionID, nuevoJugadorID, r.Notas,
			).Scan(&resultadoID)
			if err != nil {
				return nil, fmt.Errorf("insertar resultado jugador %d sesión %d: %w",
					nuevoJugadorID, nuevaSesionID, err)
			}
			resultado.ResultadosInsertados++

			// Insertar victorias
			rivalVisto := make(map[int]bool)
			for _, rivalIDAntiguo := range r.Victorias {
				if rivalIDAntiguo == r.JugadorID || rivalVisto[rivalIDAntiguo] || !participa[[2]int{r.SesionID, rivalIDAntiguo}] {
					resultado.Advertencias = append(resultado.Advertencias,
						fmt.Sprintf("Victoria omitida: el rival ID %d no es válido en la sesión ID %d", rivalIDAntiguo, r.SesionID))
					continue
				}
				rivalVisto[rivalIDAntiguo] = true
				nuevoRivalID, ok := mapaJugadores[rivalIDAntiguo]
				if !ok {
					resultado.Advertencias = append(resultado.Advertencias,
						fmt.Sprintf("Victoria omitida: rival ID %d no encontrado", rivalIDAntiguo))
					continue
				}
				_, err = tx.Exec(
					`INSERT INTO victorias (resultado_id, rival_id) VALUES ($1, $2)`,
					resultadoID, nuevoRivalID,
				)
				if err != nil {
					return nil, fmt.Errorf("insertar victoria: %w", err)
				}
			}

			// Insertar colores (sin repetidos: un color duplicado contaría doble en los KPIs)
			colorVisto := make(map[string]bool)
			for _, color := range r.Colores {
				if colorVisto[color] {
					continue
				}
				colorVisto[color] = true
				_, err = tx.Exec(
					`INSERT INTO colores_jugados (resultado_id, color) VALUES ($1, $2)`,
					resultadoID, color,
				)
				if err != nil {
					return nil, fmt.Errorf("insertar color '%s': %w", color, err)
				}
			}
		} else if err != nil {
			return nil, fmt.Errorf("buscar resultado: %w", err)
		} else {
			resultado.Advertencias = append(resultado.Advertencias,
				fmt.Sprintf("Resultado del jugador ID %d en sesión %d ya existe, omitido",
					nuevoJugadorID, nuevaSesionID))
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar transacción: %w", err)
	}

	return resultado, nil
}

// buscarSesionExistente devuelve la sesión previa a la importación (id <= maxPrevia) con esa
// temporada y fecha que aún no esté asignada, prefiriendo la de igual descripción.
// Devuelve sql.ErrNoRows si no hay ninguna.
func buscarSesionExistente(tx *sql.Tx, temporadaID int, fecha, descripcion string, maxPrevia int, asignadas map[int]bool) (int, error) {
	rows, err := tx.Query(`
		SELECT id, descripcion FROM sesiones
		WHERE temporada_id=$1 AND fecha=$2 AND id <= $3
		ORDER BY id`, temporadaID, fecha, maxPrevia)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	candidata := 0
	for rows.Next() {
		var id int
		var desc string
		if err := rows.Scan(&id, &desc); err != nil {
			return 0, err
		}
		if asignadas[id] {
			continue
		}
		if desc == descripcion {
			return id, nil
		}
		if candidata == 0 {
			candidata = id
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if candidata == 0 {
		return 0, sql.ErrNoRows
	}
	return candidata, nil
}
