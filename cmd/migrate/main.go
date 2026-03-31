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

func migrateUp(db *sql.DB, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	sort.Strings(files)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				log.Printf("Migration %s already applied, skipping", file)
				continue
			}
			return fmt.Errorf("failed to apply migration %s: %w", file, err)
		}
		log.Printf("Applied migration: %s", filepath.Base(file))
	}

	return nil
}

func migrateDown(db *sql.DB, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.down.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				log.Printf("Migration %s already rolled back, skipping", file)
				continue
			}
			return fmt.Errorf("failed to rollback migration %s: %w", file, err)
		}
		log.Printf("Rolled back migration: %s", filepath.Base(file))
	}

	return nil
}
