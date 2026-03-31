package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase2MigrationFilesExistAndDeclareNewSchema(t *testing.T) {
	migrationsDir := filepath.Join("..", "..", "migrations")
	upPath := filepath.Join(migrationsDir, "007_phase2_schema.up.sql")
	downPath := filepath.Join(migrationsDir, "007_phase2_schema.down.sql")

	up, err := os.ReadFile(upPath)
	if err != nil {
		t.Fatalf("expected phase 2 up migration at %s: %v", upPath, err)
	}
	down, err := os.ReadFile(downPath)
	if err != nil {
		t.Fatalf("expected phase 2 down migration at %s: %v", downPath, err)
	}

	upContents := string(up)
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS customers",
		"CREATE TABLE IF NOT EXISTS organization_settings",
		"CREATE TABLE IF NOT EXISTS project_managers",
		"ALTER TABLE contracts ADD COLUMN IF NOT EXISTS customer_id",
		"organization_memberships_user_org_role_key",
		"ALTER TABLE time_entries ADD COLUMN IF NOT EXISTS project_id",
		"ALTER TABLE expenses ADD COLUMN IF NOT EXISTS project_id",
		"ALTER TABLE expense_receipts ADD COLUMN IF NOT EXISTS receipt_data",
	} {
		if !strings.Contains(upContents, want) {
			t.Fatalf("expected phase 2 up migration to contain %q", want)
		}
	}

	downContents := string(down)
	for _, want := range []string{
		"DROP TABLE IF EXISTS project_managers",
		"DROP TABLE IF EXISTS organization_settings",
		"DROP TABLE IF EXISTS customers",
		"ALTER TABLE contracts DROP COLUMN IF EXISTS customer_id",
		"DROP INDEX IF EXISTS idx_project_managers_project",
	} {
		if !strings.Contains(downContents, want) {
			t.Fatalf("expected phase 2 down migration to contain %q", want)
		}
	}
}


