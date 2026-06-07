package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenRepository_Add_FindByHash(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewRefreshTokenRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	tokenHash := uuid.New().String()
	expiresAt := now.Add(24 * time.Hour)

	err := repo.Add(context.Background(), userID, orgID, tokenHash, expiresAt)
	require.NoError(t, err)

	got, err := repo.FindByHash(context.Background(), tokenHash)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, userID, got.UserID)
	require.Equal(t, orgID, got.OrganizationID)
	require.Equal(t, tokenHash, got.Hash)
}

func TestRefreshTokenRepository_FindByHash_Expired(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewRefreshTokenRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	tokenHash := uuid.New().String()
	expiresAt := now.Add(-1 * time.Hour) // already expired

	err := repo.Add(context.Background(), userID, orgID, tokenHash, expiresAt)
	require.NoError(t, err)

	got, err := repo.FindByHash(context.Background(), tokenHash)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestRefreshTokenRepository_RevokeByHash(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewRefreshTokenRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	tokenHash := uuid.New().String()
	expiresAt := now.Add(24 * time.Hour)

	err := repo.Add(context.Background(), userID, orgID, tokenHash, expiresAt)
	require.NoError(t, err)

	// Revoke
	err = repo.RevokeByHash(context.Background(), tokenHash)
	require.NoError(t, err)

	// Find should return nil
	got, err := repo.FindByHash(context.Background(), tokenHash)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestRefreshTokenRepository_FindByHash_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewRefreshTokenRepository(pool)

	got, err := repo.FindByHash(context.Background(), "nonexistent-hash")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestRefreshTokenRepository_RevokeAllByUser(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewRefreshTokenRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	// Create two tokens for the same user
	hash1 := uuid.New().String()
	hash2 := uuid.New().String()
	expiresAt := now.Add(24 * time.Hour)

	err := repo.Add(context.Background(), userID, orgID, hash1, expiresAt)
	require.NoError(t, err)
	err = repo.Add(context.Background(), userID, orgID, hash2, expiresAt)
	require.NoError(t, err)

	// Revoke all
	err = repo.RevokeAllByUser(context.Background(), userID)
	require.NoError(t, err)

	// Both should return nil
	got1, err := repo.FindByHash(context.Background(), hash1)
	require.NoError(t, err)
	require.Nil(t, got1)

	got2, err := repo.FindByHash(context.Background(), hash2)
	require.NoError(t, err)
	require.Nil(t, got2)
}
