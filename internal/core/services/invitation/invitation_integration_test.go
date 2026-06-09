package invitation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	invitationdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
)

// realRepoFixture creates a real InvitationRepository-backed *Service.
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	repo := postgres.NewInvitationRepository(pool)
	return NewService(repo)
}

func TestInvitationIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateInvitation", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()
		orgID := uuid.New()
		userID := uuid.New()

		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $4)`,
			orgID, "Test Org", "test-org-"+uuid.New().String()[:8], now)
		require.NoError(t, err)

		_, err = pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "creator@test.com", "creator_"+uuid.New().String()[:8], "Creator", "User", "hash", now)
		require.NoError(t, err)

		inv, err := svc.Create(context.Background(), &invitationdomain.CreateInvitationRequest{
			OrganizationID: orgID,
			Email:          "invited@test.com",
			ExpiresInDays:  7,
			CreatedBy:      userID,
		})
		require.NoError(t, err)
		assert.NotNil(t, inv)
		assert.NotEmpty(t, inv.ID)
		assert.Equal(t, invitationdomain.InvitationStatusPending, inv.Status)
		assert.Equal(t, userID.String(), inv.CreatedBy)
	})

	t.Run("FindByCode", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()
		orgID := uuid.New()
		userID := uuid.New()

		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $4)`,
			orgID, "Find Org", "find-org-"+uuid.New().String()[:8], now)
		require.NoError(t, err)
		_, err = pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "findbycode@test.com", "findcode_"+uuid.New().String()[:8], "Find", "User", "hash", now)
		require.NoError(t, err)

		created, err := svc.Create(context.Background(), &invitationdomain.CreateInvitationRequest{
			OrganizationID: orgID,
			Email:          "findbycode@test.com",
			ExpiresInDays:  7,
			CreatedBy:      userID,
		})
		require.NoError(t, err)

		found, err := svc.ValidateCode(context.Background(), created.Code)
		require.NoError(t, err)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, created.Email, found.Email)
	})

	t.Run("FindByToken", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()
		orgID := uuid.New()
		userID := uuid.New()

		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $4)`,
			orgID, "Token Org", "token-org-"+uuid.New().String()[:8], now)
		require.NoError(t, err)
		_, err = pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "findbytoken@test.com", "findtoken_"+uuid.New().String()[:8], "Find", "User", "hash", now)
		require.NoError(t, err)

		created, err := svc.Create(context.Background(), &invitationdomain.CreateInvitationRequest{
			OrganizationID: orgID,
			Email:          "findbytoken@test.com",
			ExpiresInDays:  7,
			CreatedBy:      userID,
		})
		require.NoError(t, err)

		found, err := svc.ValidateToken(context.Background(), created.InviteToken)
		require.NoError(t, err)
		assert.Equal(t, created.ID, found.ID)
	})

	t.Run("AcceptInvitation", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()
		orgID := uuid.New()
		userID := uuid.New()

		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $4)`,
			orgID, "Accept Org", "accept-org-"+uuid.New().String()[:8], now)
		require.NoError(t, err)
		_, err = pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "accept@test.com", "accept_"+uuid.New().String()[:8], "Accept", "User", "hash", now)
		require.NoError(t, err)

		created, err := svc.Create(context.Background(), &invitationdomain.CreateInvitationRequest{
			OrganizationID: orgID,
			Email:          "accept@test.com",
			ExpiresInDays:  7,
			CreatedBy:      userID,
		})
		require.NoError(t, err)

		accepted, err := svc.Accept(context.Background(), created.InviteToken, "accept@test.com", "acceptuser", "password123")
		require.NoError(t, err)
		assert.Equal(t, invitationdomain.InvitationStatusUsed, accepted.Status)
	})

	t.Run("AcceptAlreadyUsed", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()
		orgID := uuid.New()
		userID := uuid.New()

		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $4)`,
			orgID, "Used Org", "used-org-"+uuid.New().String()[:8], now)
		require.NoError(t, err)
		_, err = pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "used@test.com", "used_"+uuid.New().String()[:8], "Used", "User", "hash", now)
		require.NoError(t, err)

		created, err := svc.Create(context.Background(), &invitationdomain.CreateInvitationRequest{
			OrganizationID: orgID,
			Email:          "used@test.com",
			ExpiresInDays:  7,
			CreatedBy:      userID,
		})
		require.NoError(t, err)

		// Accept once
		_, err = svc.Accept(context.Background(), created.InviteToken, "used@test.com", "useduser", "password123")
		require.NoError(t, err)

		// Accept again should fail
		_, err = svc.Accept(context.Background(), created.InviteToken, "used@test.com", "useduser", "password123")
		assert.ErrorIs(t, err, invitationdomain.ErrInvitationUsed)
	})

	t.Run("FindByCodeNotFound", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		_, err := svc.ValidateCode(context.Background(), "NONEXIST")
		assert.ErrorIs(t, err, invitationdomain.ErrInvitationNotFound)
	})
}
