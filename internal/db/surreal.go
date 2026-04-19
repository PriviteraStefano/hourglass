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

		db, err := surrealdb.FromEndpointURLString(ctx,url)
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

func (s *SurrealDB) Query(ctx context.Context, query string, vars map[string]interface{}) (*[]surrealdb.QueryResult[map[string]any], error) {
	return surrealdb.Query[map[string]any](ctx, s.db, query, vars)
}

func (s *SurrealDB) Create(ctx context.Context, table string, data interface{}) (*map[string]any, error) {
	return surrealdb.Create[map[string]any](ctx, s.db, table, data)
}

func (s *SurrealDB) Select(ctx context.Context, what string) (*map[string]any, error) {
	return surrealdb.Select[map[string]any](ctx, s.db, what)
}

func (s *SurrealDB) SelectMany(ctx context.Context, what string) (*[]map[string]any, error) {
	return surrealdb.Select[[]map[string]any](ctx, s.db, what)
}

func (s *SurrealDB) Update(ctx context.Context, what string, data interface{}) (*map[string]any, error) {
	return surrealdb.Update[map[string]any](ctx, s.db, what, data)
}

func (s *SurrealDB) Delete(ctx context.Context, what string) (*map[string]any, error) {
	return surrealdb.Delete[map[string]any](ctx, s.db, what)
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
