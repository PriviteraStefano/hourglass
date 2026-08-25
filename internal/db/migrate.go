package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const schemaMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version text PRIMARY KEY,
	applied_at timestamptz NOT NULL DEFAULT now()
)`

func (db *DB) ensureSchemaMigrations() error {
	if _, err := db.Exec(schemaMigrationsTable); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}
	return nil
}

func (db *DB) migrationApplied(version string) (bool, error) {
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check migration %s: %w", version, err)
	}
	return exists, nil
}

func (db *DB) MigrateUp(migrationsDir string) error {
	if err := db.ensureSchemaMigrations(); err != nil {
		return err
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var upFiles []string
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sql" && len(f.Name()) > 3 && f.Name()[len(f.Name())-6:] == ".up.sql" {
			upFiles = append(upFiles, f.Name())
		}
	}

	sort.Strings(upFiles)

	for _, f := range upFiles {
		version := strings.TrimSuffix(f, ".up.sql")

		applied, err := db.migrationApplied(version)
		if err != nil {
			return err
		}
		if applied {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", f, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", version, err)
		}
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", f, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}
	}

	return nil
}

func (db *DB) MigrateDown(migrationsDir string) error {
	if err := db.ensureSchemaMigrations(); err != nil {
		return err
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var downFiles []string
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sql" && len(f.Name()) > 3 && f.Name()[len(f.Name())-8:] == ".down.sql" {
			downFiles = append(downFiles, f.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(downFiles)))

	for _, f := range downFiles {
		version := strings.TrimSuffix(f, ".down.sql")

		applied, err := db.migrationApplied(version)
		if err != nil {
			return err
		}
		if !applied {
			log.Printf("Migration %s not applied, skipping rollback", version)
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", f, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", version, err)
		}
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", f, err)
		}
		if _, err := tx.Exec(`DELETE FROM schema_migrations WHERE version = $1`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to unrecord migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit rollback %s: %w", version, err)
		}
	}

	return nil
}
