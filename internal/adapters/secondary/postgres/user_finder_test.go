package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stretchr/testify/require"
)

func TestUserFinder_FindByIdentifier_ByEmail(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)

	userID := uuid.New()
	email := uniqueEmail()
	now := time.Now().UTC()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, email, uniqueUsername(), "Test", "User", "hash", true, now, now)
	require.NoError(t, err)

	finder := NewUserFinder(pool)
	foundID, err := finder.FindByIdentifier(context.Background(), email)
	require.NoError(t, err)
	require.Equal(t, userID.String(), foundID)
}

func TestUserFinder_FindByIdentifier_ByUsername(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)

	userID := uuid.New()
	username := uniqueUsername()
	now := time.Now().UTC()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, uniqueEmail(), username, "Test", "User", "hash", true, now, now)
	require.NoError(t, err)

	finder := NewUserFinder(pool)
	foundID, err := finder.FindByIdentifier(context.Background(), username)
	require.NoError(t, err)
	require.Equal(t, userID.String(), foundID)
}

func TestUserFinder_FindByIdentifier_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)

	finder := NewUserFinder(pool)
	foundID, err := finder.FindByIdentifier(context.Background(), "nonexistent@test.com")
	require.Error(t, err)
	require.ErrorIs(t, err, ports.ErrUserNotFound)
	require.Empty(t, foundID)
}
