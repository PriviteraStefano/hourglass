package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
)

func uniqueEmail() string {
	return uuid.New().String() + "@test.com"
}

func uniqueUsername() string {
	return "user_" + uuid.New().String()[:8]
}

func TestUserRepository_Add(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	repo := NewUserRepository(db)

	user := &auth.User{
		ID:           uuid.New(),
		Email:        uniqueEmail(),
		Username:     uniqueUsername(),
		FirstName:    "Test",
		LastName:     "User",
		Name:         "Test User",
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := repo.Add(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	repo := NewUserRepository(db)

	email := uniqueEmail()
	user := &auth.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     uniqueUsername(),
		FirstName:    "Test",
		LastName:     "User",
		Name:         "Test User",
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := repo.Add(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}

	found, err := repo.GetByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}
	if found == nil {
		t.Fatal("expected user, got nil")
	}
	if found.Email != email {
		t.Errorf("expected email %s, got %s", email, found.Email)
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	repo := NewUserRepository(db)

	user := &auth.User{
		ID:           uuid.New(),
		Email:        uniqueEmail(),
		Username:     uniqueUsername(),
		FirstName:    "Test",
		LastName:     "User",
		Name:         "Test User",
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := repo.Add(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}

	found, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("failed to get user by id: %v", err)
	}
	if found == nil {
		t.Fatal("expected user, got nil")
	}
	if found.ID != user.ID {
		t.Errorf("expected id %s, got %s", user.ID, found.ID)
	}
}

func TestUserRepository_EmailExists(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	repo := NewUserRepository(db)

	email := uniqueEmail()
	user := &auth.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     uniqueUsername(),
		FirstName:    "Test",
		LastName:     "User",
		Name:         "Test User",
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	exists, err := repo.EmailExists(context.Background(), email)
	if err != nil {
		t.Fatalf("failed to check email exists: %v", err)
	}
	if exists {
		t.Error("expected email to not exist before adding user")
	}

	err = repo.Add(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}

	exists, err = repo.EmailExists(context.Background(), email)
	if err != nil {
		t.Fatalf("failed to check email exists: %v", err)
	}
	if !exists {
		t.Error("expected email to exist after adding user")
	}
}

func TestUserRepository_UsernameExists(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	repo := NewUserRepository(db)

	username := uniqueUsername()
	user := &auth.User{
		ID:           uuid.New(),
		Email:        uniqueEmail(),
		Username:     username,
		FirstName:    "Test",
		LastName:     "User",
		Name:         "Test User",
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	exists, err := repo.UsernameExists(context.Background(), username)
	if err != nil {
		t.Fatalf("failed to check username exists: %v", err)
	}
	if exists {
		t.Error("expected username to not exist before adding user")
	}

	err = repo.Add(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}

	exists, err = repo.UsernameExists(context.Background(), username)
	if err != nil {
		t.Fatalf("failed to check username exists: %v", err)
	}
	if !exists {
		t.Error("expected username to exist after adding user")
	}
}
