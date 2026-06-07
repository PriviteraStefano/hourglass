package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	orgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/require"
)

func TestOrganizationManagementRepository_CreateOrganization(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	now := time.Now()
	userID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, uniqueEmail(), uniqueUsername(), "Owner", "User", "hash", true, now, now)
	require.NoError(t, err)

	repo := NewOrganizationManagementRepository(pool)
	org := &orgdomain.Organization{
		ID:        uuid.New(),
		Name:      "Created Org",
		Slug:      "created-org-" + uuid.New().String()[:8],
		CreatedAt: now,
	}

	err = repo.CreateOrganization(context.Background(), org, userID, models.RoleManager)
	require.NoError(t, err)

	// Verify org exists
	got, err := repo.GetOrganization(context.Background(), org.ID)
	require.NoError(t, err)
	require.Equal(t, org.Name, got.Name)
	require.Equal(t, org.Slug, got.Slug)

	// Verify membership exists (owner membership)
	members, err := repo.ListMembers(context.Background(), org.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.Equal(t, userID, *members[0].UserID)
	require.Equal(t, models.RoleManager, members[0].Role)

	// Settings should be auto-created by the trigger
	settings, err := repo.GetSettings(context.Background(), org.ID)
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.Equal(t, "EUR", settings.Currency)
}

func TestOrganizationManagementRepository_GetOrganization_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	_, err := repo.GetOrganization(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, orgdomain.ErrOrganizationNotFound)
}

func TestOrganizationManagementRepository_GetSettings(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	now := time.Now().UTC()

	// Create org (the trigger auto-creates settings)
	orgID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Settings Org", "settings-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	settings, err := repo.GetSettings(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.Equal(t, orgID, settings.OrganizationID)
	require.Equal(t, "EUR", settings.Currency)
	require.Equal(t, 1, settings.WeekStartDay)
	require.Equal(t, "UTC", settings.Timezone)
	require.True(t, settings.ShowApprovalHistory)
	require.Nil(t, settings.DefaultKmRate)
}

func TestOrganizationManagementRepository_GetSettings_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	_, err := repo.GetSettings(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, orgdomain.ErrOrganizationNotFound)
}

func TestOrganizationManagementRepository_UpdateSettings(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	now := time.Now().UTC()

	orgID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Update Org", "update-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	kmRate := 0.42
	wsd := 0
	showHistory := false
	updated, err := repo.UpdateSettings(context.Background(), orgID, &orgdomain.UpdateSettingsRequest{
		DefaultKmRate:       &kmRate,
		Currency:            "USD",
		WeekStartDay:        &wsd,
		Timezone:            "America/New_York",
		ShowApprovalHistory: &showHistory,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)

	require.Equal(t, orgID, updated.OrganizationID)
	require.NotNil(t, updated.DefaultKmRate)
	require.Equal(t, 0.42, *updated.DefaultKmRate)
	require.Equal(t, "USD", updated.Currency)
	require.Equal(t, 0, updated.WeekStartDay)
	require.Equal(t, "America/New_York", updated.Timezone)
	require.False(t, updated.ShowApprovalHistory)
}

func TestOrganizationManagementRepository_ListMembers(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	now := time.Now().UTC()

	orgID := uuid.New()
	userID := uuid.New()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "List Members Org", "list-mem-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, uniqueEmail(), uniqueUsername(), "Listed", "Member", "hash", true, now, now)
	require.NoError(t, err)

	// Add membership via raw SQL
	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), userID, orgID, "employee", true, now)
	require.NoError(t, err)

	members, err := repo.ListMembers(context.Background(), orgID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.Equal(t, userID, *members[0].UserID)
	require.Equal(t, "Listed Member", members[0].UserName)
	require.NotEmpty(t, members[0].UserEmail)

	// No members
	empty, err := repo.ListMembers(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestOrganizationManagementRepository_UpdateMemberRole(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	now := time.Now().UTC()

	orgID := uuid.New()
	userID := uuid.New()
	membershipID := uuid.New()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Role Org", "role-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, uniqueEmail(), uniqueUsername(), "Role", "Test", "hash", true, now, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		membershipID, userID, orgID, "employee", true, now)
	require.NoError(t, err)

	// Change to manager
	err = repo.UpdateMemberRole(context.Background(), orgID, membershipID, models.RoleManager)
	require.NoError(t, err)

	// Verify
	role, err := repo.GetMemberRole(context.Background(), membershipID)
	require.NoError(t, err)
	require.Equal(t, models.RoleManager, role)
}

func TestOrganizationManagementRepository_DeactivateMember(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	now := time.Now().UTC()

	orgID := uuid.New()
	userID := uuid.New()
	membershipID := uuid.New()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Deact Org", "deact-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, uniqueEmail(), uniqueUsername(), "Deact", "Test", "hash", true, now, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		membershipID, userID, orgID, "employee", true, now)
	require.NoError(t, err)

	err = repo.DeactivateMember(context.Background(), orgID, membershipID)
	require.NoError(t, err)

	members, err := repo.ListMembers(context.Background(), orgID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.False(t, members[0].IsActive)
}

func TestOrganizationManagementRepository_CountActiveFinance(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	now := time.Now().UTC()

	orgID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Finance Org", "fin-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	for _, uid := range []uuid.UUID{userID1, userID2} {
		_, err = pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			uid, uniqueEmail(), uniqueUsername(), "Finance", "User", "hash", true, now, now)
		require.NoError(t, err)
	}

	// One active finance
	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), userID1, orgID, "finance", true, now)
	require.NoError(t, err)

	// One inactive finance
	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), userID2, orgID, "finance", false, now)
	require.NoError(t, err)

	count, err := repo.CountActiveFinance(context.Background(), orgID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Org with no finance members
	emptyCount, err := repo.CountActiveFinance(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, 0, emptyCount)
}

func TestOrganizationManagementRepository_InviteMember(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	now := time.Now().UTC()

	orgID := uuid.New()
	invitedBy := uuid.New()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Invite Org", "invite-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		invitedBy, uniqueEmail(), uniqueUsername(), "Inviter", "User", "hash", true, now, now)
	require.NoError(t, err)

	memberID, invitedAt, err := repo.InviteMember(context.Background(), orgID, &orgdomain.InviteRequest{
		Email: "invited@test.com",
		Role:  models.RoleEmployee,
	}, invitedBy)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, memberID)
	require.False(t, invitedAt.IsZero())

	// Verify the invited member has no user_id
	members, err := repo.ListMembers(context.Background(), orgID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.Nil(t, members[0].UserID)
	require.Equal(t, models.RoleEmployee, members[0].Role)
	require.NotNil(t, members[0].InvitedBy)
	require.Equal(t, invitedBy, *members[0].InvitedBy)
}

func TestOrganizationManagementRepository_GetMemberRole_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationManagementRepository(pool)
	_, err := repo.GetMemberRole(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, orgdomain.ErrMemberNotFound)
}
