package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func (db *DB) MigrateUp(migrationsDir string) error {
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
		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", f, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", f, err)
		}
	}

	return nil
}

func (db *DB) MigrateDown(migrationsDir string) error {
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
		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", f, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", f, err)
		}
	}

	return nil
}
