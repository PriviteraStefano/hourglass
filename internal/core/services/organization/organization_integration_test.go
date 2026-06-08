package organization

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// realRepoFixture creates a real OrgManagementRepository-backed *Service.
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	repo := postgres.NewOrganizationManagementRepository(pool)
	return NewService(repo)
}

// seedUser creates a user row and returns the ID.
func seedUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
		id, uuid.New().String()+"@test.com", "user_"+uuid.New().String()[:8],
		"Owner", "User", "hash", time.Now())
	require.NoError(t, err)
	return id
}

func TestOrgIntegration(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("CreateOrganization", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		ownerID := seedUser(t, pool)

		org, err := svc.Create(context.Background(), ownerID,
			&orgdomain.CreateOrganizationRequest{Name: "My New Org"})
		require.NoError(t, err)
		require.NotNil(t, org)
		assert.NotEmpty(t, org.ID)
		assert.Equal(t, "My New Org", org.Name)
		assert.Equal(t, "my-new-org", org.Slug)

		got, err := svc.Get(context.Background(), org.ID)
		require.NoError(t, err)
		assert.Equal(t, org.ID, got.ID)
		assert.Equal(t, "My New Org", got.Name)
	})

	t.Run("CreateOrganization_WithCustomSlug", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		ownerID := seedUser(t, pool)

		org, err := svc.Create(context.Background(), ownerID,
			&orgdomain.CreateOrganizationRequest{Name: "Custom Slug Org", Slug: "my-custom-slug"})
		require.NoError(t, err)
		assert.Equal(t, "my-custom-slug", org.Slug)

		got, err := svc.Get(context.Background(), org.ID)
		require.NoError(t, err)
		assert.Equal(t, "my-custom-slug", got.Slug)
	})

	t.Run("CreateOrganization_EmptyName", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		_, err := svc.Create(context.Background(), uuid.New(),
			&orgdomain.CreateOrganizationRequest{Name: ""})
		assert.ErrorIs(t, err, orgdomain.ErrInvalidRequest)
	})

	t.Run("ListMembers", func(t *testing.T) {
		svc := realRepoFixture(t, pool)
		now := time.Now()

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Team Org", "team-org-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)

		userID := uuid.New()
		_, err = pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
			userID, "member@test.com", "member_"+uuid.New().String()[:8], "Team", "Member", "hash", now)
		require.NoError(t, err)

		membershipID := uuid.New()
		_, err = pool.Exec(context.Background(),
			`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
			 VALUES ($1, $2, $3, $4, true, $5)`,
			membershipID, userID, orgID, models.RoleEmployee, now)
		require.NoError(t, err)

		members, err := svc.ListMembers(context.Background(), orgID)
		require.NoError(t, err)
		require.Len(t, members, 1)
		assert.Equal(t, userID, *members[0].UserID)
		assert.Equal(t, models.RoleEmployee, members[0].Role)
		assert.True(t, members[0].IsActive)
	})

	t.Run("ListMembers_EmptyOrg", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Empty Org", "empty-org-"+uuid.New().String()[:8], time.Now(), time.Now())
		require.NoError(t, err)

		members, err := svc.ListMembers(context.Background(), orgID)
		require.NoError(t, err)
		assert.Empty(t, members)
	})

	t.Run("GetOrganization_NotFound", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		org, err := svc.Get(context.Background(), uuid.New())
		assert.ErrorIs(t, err, orgdomain.ErrOrganizationNotFound)
		assert.Nil(t, org)
	})

	t.Run("UpdateSettings", func(t *testing.T) {
		t.Skip("Skipped: organization_settings table missing from schema — tracked for Plan 05")
	})

	t.Run("UpdateSettings_ForbiddenForNonFinance", func(t *testing.T) {
		svc := realRepoFixture(t, pool)

		orgID := uuid.New()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			orgID, "Forbidden Org", "forbidden-org-"+uuid.New().String()[:8], time.Now(), time.Now())
		require.NoError(t, err)

		_, err = svc.UpdateSettings(context.Background(), string(models.RoleEmployee), orgID,
			&orgdomain.UpdateSettingsRequest{Currency: "EUR"})
		assert.ErrorIs(t, err, orgdomain.ErrForbidden)
	})
}

// ---------------------------------------------------------------------------
// Compile-time checks
// ---------------------------------------------------------------------------

var _ ports.OrganizationManagementRepository = (*postgres.OrganizationManagementRepository)(nil)
