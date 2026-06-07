package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	"github.com/stretchr/testify/require"
)

func TestPasswordResetRepository_Create_FindActiveByUserID(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewPasswordResetRepository(pool)
	now := time.Now().UTC()
	userID := seedUser(t, pool, now)

	pr := &password_reset.PasswordReset{
		ID:        uuid.New(),
		UserID:    userID,
		CodeHash:  "test-code-hash",
		ExpiresAt: now.Add(1 * time.Hour),
	}

	created, err := repo.Create(context.Background(), pr)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, pr.ID, created.ID)
	require.Equal(t, pr.UserID, created.UserID)
	require.Equal(t, pr.CodeHash, created.CodeHash)
	require.WithinDuration(t, pr.ExpiresAt, created.ExpiresAt, time.Second)
	require.Nil(t, created.UsedAt)

	// Find active
	got, err := repo.FindActiveByUserID(context.Background(), userID.String())
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.CodeHash, got.CodeHash)
}

func TestPasswordResetRepository_FindActiveByUserID_Expired(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewPasswordResetRepository(pool)
	now := time.Now().UTC()
	userID := seedUser(t, pool, now)

	pr := &password_reset.PasswordReset{
		ID:        uuid.New(),
		UserID:    userID,
		CodeHash:  "expired-code-hash",
		ExpiresAt: now.Add(-1 * time.Hour), // already expired
	}

	_, err := repo.Create(context.Background(), pr)
	require.NoError(t, err)

	got, err := repo.FindActiveByUserID(context.Background(), userID.String())
	require.ErrorIs(t, err, password_reset.ErrResetNotFound)
	require.Nil(t, got)
}

func TestPasswordResetRepository_FindActiveByUserID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewPasswordResetRepository(pool)

	got, err := repo.FindActiveByUserID(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, password_reset.ErrResetNotFound)
	require.Nil(t, got)
}

func TestPasswordResetRepository_MarkUsed(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewPasswordResetRepository(pool)
	now := time.Now().UTC()
	userID := seedUser(t, pool, now)

	pr := &password_reset.PasswordReset{
		ID:        uuid.New(),
		UserID:    userID,
		CodeHash:  "mark-used-hash",
		ExpiresAt: now.Add(1 * time.Hour),
	}

	created, err := repo.Create(context.Background(), pr)
	require.NoError(t, err)

	// Mark as used
	err = repo.MarkUsed(context.Background(), created.ID.String())
	require.NoError(t, err)

	// Find active should return not found
	got, err := repo.FindActiveByUserID(context.Background(), userID.String())
	require.ErrorIs(t, err, password_reset.ErrResetNotFound)
	require.Nil(t, got)
}

func TestPasswordResetRepository_UpdateUserPassword(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewPasswordResetRepository(pool)
	now := time.Now().UTC()
	userID := seedUser(t, pool, now)

	// Change password
	err := repo.UpdateUserPassword(context.Background(), userID, "new-password-hash")
	require.NoError(t, err)

	// Verify via direct query
	var passwordHash string
	err = pool.QueryRow(context.Background(), `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&passwordHash)
	require.NoError(t, err)
	require.Equal(t, "new-password-hash", passwordHash)
}
