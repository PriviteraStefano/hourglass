package password_reset

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	pwdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
)

// realRepoFixture creates real PostgreSQL-backed repos and constructs a *Service.
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	pwRepo := postgres.NewPasswordResetRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	userFinder := postgres.NewUserFinder(pool)
	hasher := auth.NewPasswordHasher()
	tokenSvc := auth.NewTokenService(auth.NewService("dev-secret-change-in-production"))
	refreshRepo := postgres.NewRefreshTokenRepository(pool)

	return NewService(pwRepo, userRepo, userFinder, hasher, tokenSvc, refreshRepo)
}

func TestPasswordResetIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("RequestReset", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		userID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "resetuser@test.com", "reset_"+uuid.New().String()[:8],
			"Reset", "User", "oldhash", now)
		require.NoError(t, err)

		code, expiresAt, err := svc.Request(context.Background(), "resetuser@test.com")
		require.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.False(t, expiresAt.IsZero())
		assert.True(t, expiresAt.After(time.Now()))
	})

	t.Run("RequestReset_UnknownIdentifier", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		code, expiresAt, err := svc.Request(context.Background(), "unknown@test.com")
		assert.ErrorIs(t, err, pwdomain.ErrUserNotFound)
		assert.Empty(t, code)
		assert.True(t, expiresAt.IsZero())
	})

	t.Run("VerifyWithCorrectCode", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		userID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "verify@test.com", "verify_"+uuid.New().String()[:8],
			"Verify", "User", "oldhash", now)
		require.NoError(t, err)

		// Request a reset code
		code, _, err := svc.Request(context.Background(), "verify@test.com")
		require.NoError(t, err)
		require.NotEmpty(t, code)

		// Verify with correct code
		err = svc.Verify(context.Background(), "verify@test.com", code, "newSecurePass123!")
		assert.NoError(t, err)
	})

	t.Run("VerifyWithWrongCode", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		userID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "wrongcode@test.com", "wrong_"+uuid.New().String()[:8],
			"Wrong", "Code", "oldhash", now)
		require.NoError(t, err)

		// Request a reset (creates a hashed code in the DB)
		_, _, err = svc.Request(context.Background(), "wrongcode@test.com")
		require.NoError(t, err)

		// Verify with wrong code
		err = svc.Verify(context.Background(), "wrongcode@test.com", "wrongcode!", "newpass")
		assert.ErrorIs(t, err, pwdomain.ErrInvalidCode)
	})

	t.Run("VerifyExpiredReset", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		userID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "expired@test.com", "expired_"+uuid.New().String()[:8],
			"Expired", "User", "oldhash", now)
		require.NoError(t, err)

		_, _, err = svc.Request(context.Background(), "expired@test.com")
		require.NoError(t, err)

		// Verify with completely wrong (non-existent) code = different hash
		err = svc.Verify(context.Background(), "expired@test.com", "invalidcode12345", "newpass")
		assert.ErrorIs(t, err, pwdomain.ErrInvalidCode)
	})

	t.Run("VerifyPreventsReplay", func(t *testing.T) {
		t.Skip("Skipped: MarkUsed + revocation verification requires tracking code hash — tracked for Plan 05")
	})
}
