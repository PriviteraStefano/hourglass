package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
)

func TestRefreshTokenRepository_Add(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	userRepo := NewUserRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)

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

	err := userRepo.Add(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}

	tokenHash := "test-hash-" + uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	err = refreshRepo.Add(context.Background(), user.ID, tokenHash, expiresAt)
	if err != nil {
		t.Fatalf("failed to add refresh token: %v", err)
	}
}

func TestRefreshTokenRepository_FindByHash(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	userRepo := NewUserRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)

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

	err := userRepo.Add(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}

	tokenHash := "find-test-hash-" + uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	err = refreshRepo.Add(context.Background(), user.ID, tokenHash, expiresAt)
	if err != nil {
		t.Fatalf("failed to add refresh token: %v", err)
	}

	found, err := refreshRepo.FindByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatalf("failed to find refresh token: %v", err)
	}
	if found == nil {
		t.Fatal("expected refresh token, got nil")
	}
	if found.UserID != user.ID {
		t.Errorf("expected user id %s, got %s", user.ID, found.UserID)
	}
}

func TestRefreshTokenRepository_RevokeByHash(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	db := GetDB()
	userRepo := NewUserRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)

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

	err := userRepo.Add(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}

	tokenHash := "revoke-test-hash-" + uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	err = refreshRepo.Add(context.Background(), user.ID, tokenHash, expiresAt)
	if err != nil {
		t.Fatalf("failed to add refresh token: %v", err)
	}

	err = refreshRepo.RevokeByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatalf("failed to revoke refresh token: %v", err)
	}

	found, err := refreshRepo.FindByHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatalf("failed to find refresh token after revoke: %v", err)
	}
	if found != nil {
		t.Error("expected nil after revoke, got token")
	}
}
