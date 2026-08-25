package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func New(dataSourceName string) (*DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// NewPool creates a pgxpool.Pool from DATABASE_URL environment variable.
// Used by cmd/server for production and integration test connections.
func NewPool() (*pgxpool.Pool, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}
	// WR-06: pin the session timezone so ::date casts on TIMESTAMPTZ are
	// UTC-deterministic (the coverage period-close predicates included).
	// Without this the server's zone applied — a non-UTC VPS silently
	// shifted period closes by one day. The handler parses period bounds as
	// UTC midnights, so UTC is the only correct comparison zone.
	connConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"

	// Bound the pool size. pgxpool defaults to 4 when unspecified, which
	// serializes all authenticated traffic under load (CONCERNS.md #15).
	// Tunable via DB_MAX_CONNS; default 20.
	const defaultMaxConns = 20
	connConfig.MaxConns = defaultMaxConns
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
			connConfig.MaxConns = int32(n)
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// ClosePool closes a pgxpool.Pool, if non-nil.
func ClosePool(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
