package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPasswordResetRepo(t *testing.T) (*PasswordResetRepository, func()) {
	t.Helper()

	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	ns := "test_pwreset_" + uuid.New().String()
	db, err := GetTestDBWithNamespace(ns, ns)
	if err != nil {
		t.Skipf("SurrealDB not available: %v", err)
	}

	repo := NewPasswordResetRepository(db)
	return repo, func() { db.Close(context.Background()) }
}

func seedPasswordResetUser(t *testing.T, repo *PasswordResetRepository) uuid.UUID {
	t.Helper()

	userRepo := NewUserRepository(repo.db)
	userID := uuid.New()
	user := &auth.User{
		ID:           userID,
		Email:        uuid.New().String() + "@test.com",
		Username:     "pwreset_user_" + uuid.New().String()[:8],
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err := userRepo.Add(context.Background(), user)
	require.NoError(t, err, "failed to seed user")

	return userID
}

func TestPasswordResetRepo_Create(t *testing.T) {
	repo, cleanup := setupPasswordResetRepo(t)
	defer cleanup()
	userID := seedPasswordResetUser(t, repo)

	pr := &password_reset.PasswordReset{
		ID:        uuid.New(),
		UserID:    userID,
		CodeHash:  "hashed-code-" + uuid.New().String(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	created, err := repo.Create(context.Background(), pr)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotEqual(t, uuid.Nil, created.ID)
	assert.Equal(t, pr.CodeHash, created.CodeHash)
	assert.WithinDuration(t, pr.ExpiresAt, created.ExpiresAt, time.Second)
}

func TestPasswordResetRepo_FindActiveByUserID(t *testing.T) {
	repo, cleanup := setupPasswordResetRepo(t)
	defer cleanup()
	userID := seedPasswordResetUser(t, repo)

	t.Run("existing", func(t *testing.T) {
		pr := &password_reset.PasswordReset{
			ID:        uuid.New(),
			UserID:    userID,
			CodeHash:  "find-hash-" + uuid.New().String(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
			CreatedAt: time.Now(),
		}

		_, err := repo.Create(context.Background(), pr)
		require.NoError(t, err)

		found, err := repo.FindActiveByUserID(context.Background(), userID.String())
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, pr.CodeHash, found.CodeHash)
		assert.False(t, found.IsValid() == false, "active reset should be valid")
	})

	t.Run("not found for random user", func(t *testing.T) {
		found, err := repo.FindActiveByUserID(context.Background(), uuid.New().String())
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestPasswordResetRepo_MarkUsed(t *testing.T) {
	repo, cleanup := setupPasswordResetRepo(t)
	defer cleanup()
	userID := seedPasswordResetUser(t, repo)

	pr := &password_reset.PasswordReset{
		ID:        uuid.New(),
		UserID:    userID,
		CodeHash:  "mark-used-hash-" + uuid.New().String(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	created, err := repo.Create(context.Background(), pr)
	require.NoError(t, err)

	err = repo.MarkUsed(context.Background(), created.ID.String())
	require.NoError(t, err)

	found, err := repo.FindActiveByUserID(context.Background(), userID.String())
	assert.Error(t, err, "should not find marked-used reset")
	assert.Nil(t, found)
}

func TestPasswordResetRepo_UpdateUserPassword(t *testing.T) {
	repo, cleanup := setupPasswordResetRepo(t)
	defer cleanup()
	userID := seedPasswordResetUser(t, repo)

	err := repo.UpdateUserPassword(context.Background(), userID, "new-hashed-password")
	require.NoError(t, err)

	fetchedUser, err := NewUserRepository(repo.db).GetByID(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, fetchedUser)
	assert.Equal(t, "new-hashed-password", fetchedUser.PasswordHash)
}
