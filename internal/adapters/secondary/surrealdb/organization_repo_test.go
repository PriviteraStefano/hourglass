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

func TestOrganizationRepository_AddMembershipAndGetMembership(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	repo := NewOrganizationRepository(db)
	userRepo := NewUserRepository(db)

	org := &auth.Organization{
		ID:        uuid.New(),
		Name:      uniqueOrgName(),
		Slug:      uniqueSlug(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := repo.Add(context.Background(), org); err != nil {
		t.Fatalf("failed to add organization: %v", err)
	}

	user := auth.NewUser(uniqueOrgName()+"@example.com", "user_"+uuid.New().String()[:8], "Test", "User", "Test User", "hashed-password")
	if err := userRepo.Add(context.Background(), user); err != nil {
		t.Fatalf("failed to add user: %v", err)
	}

	membership := auth.NewOrganizationMembership(user.ID, org.ID, "employee")
	if err := repo.AddMembership(context.Background(), membership); err != nil {
		t.Fatalf("failed to add membership: %v", err)
	}

	found, err := repo.GetMembership(context.Background(), user.ID, org.ID)
	if err != nil {
		t.Fatalf("failed to get membership: %v", err)
	}
	if found == nil {
		t.Fatal("expected membership, got nil")
	}
	if found.UserID != user.ID {
		t.Errorf("expected user id %s, got %s", user.ID, found.UserID)
	}
	if found.OrganizationID != org.ID {
		t.Errorf("expected org id %s, got %s", org.ID, found.OrganizationID)
	}
	if found.Role != "employee" {
		t.Errorf("expected role employee, got %s", found.Role)
	}
}

