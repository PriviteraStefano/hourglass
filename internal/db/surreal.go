package db

import (
	"context"
	"fmt"
	"os"
	"sync"

	surrealdb "github.com/surrealdb/surrealdb.go"
)

type SurrealDB struct {
	db *surrealdb.DB
}

var (
	instance *SurrealDB
	once     sync.Once
)

func NewSurrealDB() (*SurrealDB, error) {
	var initErr error
	once.Do(func() {
		ctx := context.Background()
		url := getEnvOrDefault("SURREALDB_URL", "ws://localhost:8000/rpc")
		user := getEnvOrDefault("SURREALDB_USER", "root")
		pass := getEnvOrDefault("SURREALDB_PASS", "root")
		ns := getEnvOrDefault("SURREALDB_NS", "hourglass")
		dbName := getEnvOrDefault("SURREALDB_DB", "main")

		db, err := surrealdb.FromEndpointURLString(ctx, url)
		if err != nil {
			initErr = fmt.Errorf("failed to connect to SurrealDB: %w", err)
			return
		}

		if _, err := db.SignIn(ctx, &surrealdb.Auth{
			Username: user,
			Password: pass,
		}); err != nil {
			initErr = fmt.Errorf("failed to sign in to SurrealDB: %w", err)
			return
		}

		if err := db.Use(ctx, ns, dbName); err != nil {
			initErr = fmt.Errorf("failed to use namespace/database: %w", err)
			return
		}

		instance = &SurrealDB{db: db}
	})

	if initErr != nil {
		return nil, initErr
	}
	return instance, nil
}

func (s *SurrealDB) DB() *surrealdb.DB {
	return s.db
}

func (s *SurrealDB) Close() error {
	return s.db.Close(context.Background())
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
