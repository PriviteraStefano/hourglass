package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	surrealdb "github.com/surrealdb/surrealdb.go"
)

func main() {
	url := getEnvOrDefault("SURREALDB_URL", "ws://localhost:8000/rpc")
	user := getEnvOrDefault("SURREALDB_USER", "root")
	pass := getEnvOrDefault("SURREALDB_PASS", "root")
	ns := getEnvOrDefault("SURREALDB_NS", "hourglass")
	dbName := getEnvOrDefault("SURREALDB_DB", "main")
	schemaDir := getEnvOrDefault("SCHEMA_DIR", "schema")

	ctx := context.Background()

	db, err := surrealdb.FromEndpointURLString(ctx,url)
	if err != nil {
		log.Fatalf("Failed to connect to SurrealDB: %v", err)
	}

	if _, err := db.SignIn(ctx, &surrealdb.Auth{
		Username: user,
		Password: pass,
	}); err != nil {
		log.Fatalf("Failed to sign in: %v", err)
	}

	if err := db.Use(ctx, ns, dbName); err != nil {
		log.Fatalf("Failed to use namespace/database: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(schemaDir, "*.surql"))
	if err != nil {
		log.Fatalf("Failed to read schema directory: %v", err)
	}

	sort.Strings(files)

	for _, file := range files {
		filename := filepath.Base(file)
		log.Printf("Applying schema: %s", filename)

		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read file %s: %v", file, err)
		}

		queries := strings.Split(string(content), ";")
		for _, query := range queries {
			query = strings.TrimSpace(query)
			if query == "" {
				continue
			}

			if _, err := surrealdb.Query[map[string]any](ctx, db, query, nil); err != nil {
				log.Fatalf("Failed to execute query from %s: %v\nQuery: %s", filename, err, query)
			}
		}

		log.Printf("Applied: %s", filename)
	}

	log.Println("Schema migration completed successfully")
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
