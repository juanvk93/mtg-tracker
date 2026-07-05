// Paquete db gestiona la conexión y el esquema de la base de datos PostgreSQL (Supabase)
package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Inicializar abre la conexión a PostgreSQL y aplica las migraciones necesarias.
// Configura pgx para que NO use prepared statements: el pooler de Supabase en modo
// Transaction reutiliza conexiones de forma agresiva y el caché de statements de pgx
// produce el error "prepared statement already exists" (SQLSTATE 42P05).
func Inicializar(databaseURL string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsear DATABASE_URL: %w", err)
	}

	// Modo "simple protocol": pgx envía las queries sin PREPARE/BIND/EXECUTE.
	// Es la forma recomendada de conectar a través de PgBouncer / Supabase pooler
	// en modo Transaction. La conversión de tipos (time.Time, int, etc.) sigue
	// funcionando porque pgx serializa los parámetros antes de enviarlos.
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	connStr := stdlib.RegisterConnConfig(cfg)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := aplicarEsquema(db); err != nil {
		return nil, err
	}

	log.Println("Base de datos PostgreSQL inicializada correctamente")
	return db, nil
}

// aplicarEsquema crea las tablas si no existen, ejecutando cada sentencia individualmente
// (pgx no admite múltiples sentencias en un solo Exec)
func aplicarEsquema(db *sql.DB) error {
	sentencias := []string{
		`CREATE TABLE IF NOT EXISTS jugadores (
			id      SERIAL PRIMARY KEY,
			nombre  TEXT NOT NULL UNIQUE,
			color   TEXT NOT NULL DEFAULT '#6c757d',
			avatar  TEXT NOT NULL DEFAULT 'ms-planeswalker'
		)`,
		`CREATE TABLE IF NOT EXISTS temporadas (
			id     SERIAL PRIMARY KEY,
			anio   INTEGER NOT NULL UNIQUE,
			estado TEXT NOT NULL DEFAULT 'activa' CHECK(estado IN ('activa', 'cerrada'))
		)`,
		`CREATE TABLE IF NOT EXISTS sesiones (
			id           SERIAL PRIMARY KEY,
			temporada_id INTEGER NOT NULL REFERENCES temporadas(id),
			fecha        DATE NOT NULL,
			descripcion  TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS resultados (
			id         SERIAL PRIMARY KEY,
			sesion_id  INTEGER NOT NULL REFERENCES sesiones(id) ON DELETE CASCADE,
			jugador_id INTEGER NOT NULL REFERENCES jugadores(id),
			UNIQUE(sesion_id, jugador_id)
		)`,
		`CREATE TABLE IF NOT EXISTS victorias (
			id            SERIAL PRIMARY KEY,
			resultado_id  INTEGER NOT NULL REFERENCES resultados(id) ON DELETE CASCADE,
			rival_id      INTEGER NOT NULL REFERENCES jugadores(id)
		)`,
		`CREATE TABLE IF NOT EXISTS colores_jugados (
			id           SERIAL PRIMARY KEY,
			resultado_id INTEGER NOT NULL REFERENCES resultados(id) ON DELETE CASCADE,
			color        TEXT NOT NULL CHECK(color IN ('W', 'U', 'B', 'R', 'G'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sesiones_temporada ON sesiones(temporada_id)`,
		`CREATE INDEX IF NOT EXISTS idx_resultados_sesion ON resultados(sesion_id)`,
		`CREATE INDEX IF NOT EXISTS idx_resultados_jugador ON resultados(jugador_id)`,
		`CREATE INDEX IF NOT EXISTS idx_victorias_resultado ON victorias(resultado_id)`,
		`CREATE INDEX IF NOT EXISTS idx_victorias_rival ON victorias(rival_id)`,
	}

	for _, s := range sentencias {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
