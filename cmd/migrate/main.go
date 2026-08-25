package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable"
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	args := os.Args[1:]
	cmd := getCommand(args)
	dir := getMigrationsDir(args)

	switch cmd {
	case "up":
		if err := migrateUp(db, dir); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Println("Migrations applied successfully")
	case "down":
		if err := migrateDown(db, dir); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		log.Println("Migrations rolled back successfully")
	default:
		log.Fatal("Usage: migrate -up|-down [-dir <migrations_dir>]")
	}
}

func getCommand(args []string) string {
	for _, arg := range args {
		if arg == "-up" {
			return "up"
		}
		if arg == "-down" {
			return "down"
		}
	}
	return ""
}

func getMigrationsDir(args []string) string {
	for i, arg := range args {
		if arg == "-dir" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return "migrations"
}

const schemaMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version text PRIMARY KEY,
	applied_at timestamptz NOT NULL DEFAULT now()
)`

func ensureSchemaMigrations(db *sql.DB) error {
	if _, err := db.Exec(schemaMigrationsTable); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}
	return nil
}

func migrationApplied(db *sql.DB, version string) (bool, error) {
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check migration %s: %w", version, err)
	}
	return exists, nil
}

func migrateUp(db *sql.DB, dir string) error {
	if err := ensureSchemaMigrations(db); err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	sort.Strings(files)

	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".up.sql")

		applied, err := migrationApplied(db, version)
		if err != nil {
			return err
		}
		if applied {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		// Apply each migration in its own transaction so a multi-statement
		// file that fails partway leaves no partial schema and no ledger row.
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", version, err)
		}
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %s: %w", version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}
		log.Printf("Applied migration: %s", version)
	}

	return nil
}

func migrateDown(db *sql.DB, dir string) error {
	if err := ensureSchemaMigrations(db); err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.down.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".down.sql")

		applied, err := migrationApplied(db, version)
		if err != nil {
			return err
		}
		if !applied {
			log.Printf("Migration %s not applied, skipping rollback", version)
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", version, err)
		}
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to rollback migration %s: %w", version, err)
		}
		if _, err := tx.Exec(`DELETE FROM schema_migrations WHERE version = $1`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to unrecord migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit rollback %s: %w", version, err)
		}
		log.Printf("Rolled back migration: %s", version)
	}

	return nil
}
