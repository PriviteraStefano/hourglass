package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
)

func uniqueOrgName() string {
	return "Org " + uuid.New().String()[:8]
}

func uniqueSlug() string {
	return "org-" + uuid.New().String()[:8]
}

func TestOrganizationRepository_Add(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	repo := NewOrganizationRepository(db)

	org := &auth.Organization{
		ID:                  uuid.New(),
		Name:                uniqueOrgName(),
		Slug:                uniqueSlug(),
		Description:         "Test organization",
		FinancialCutoffDays: 7,
		FinancialCutoffConfig: map[string]interface{}{
			"cutoff_day_of_month": 28,
			"grace_days":          7,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Add(context.Background(), org)
	if err != nil {
		t.Fatalf("failed to add organization: %v", err)
	}
}

func TestOrganizationRepository_GetByID(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	repo := NewOrganizationRepository(db)

	org := &auth.Organization{
		ID:                  uuid.New(),
		Name:                uniqueOrgName(),
		Slug:                uniqueSlug(),
		Description:         "Test organization",
		FinancialCutoffDays: 7,
		FinancialCutoffConfig: map[string]interface{}{
			"cutoff_day_of_month": 28,
			"grace_days":          7,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Add(context.Background(), org)
	if err != nil {
		t.Fatalf("failed to add organization: %v", err)
	}

	found, err := repo.GetByID(context.Background(), org.ID)
	if err != nil {
		t.Fatalf("failed to get organization by id: %v", err)
	}
	if found == nil {
		t.Fatal("expected organization, got nil")
	}
	if found.ID != org.ID {
		t.Errorf("expected id %s, got %s", org.ID, found.ID)
	}
	if found.Name != org.Name {
		t.Errorf("expected name %s, got %s", org.Name, found.Name)
	}
}
