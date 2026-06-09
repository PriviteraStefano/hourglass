package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stretchr/testify/require"
)

func TestOrganizationRepository_Add_GetByID(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationRepository(pool)
	org := auth.NewOrganization("Test Org", "test-org-"+uuid.New().String()[:8], "A test org")
	org.FinancialCutoffDays = 14
	org.FinancialCutoffConfig = map[string]interface{}{
		"cutoff_day_of_month": 15,
		"grace_days":          5,
	}

	err := repo.Add(context.Background(), org)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), org.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, org.ID, got.ID)
	require.Equal(t, org.Name, got.Name)
	require.Equal(t, org.Slug, got.Slug)
	require.Equal(t, org.FinancialCutoffDays, got.FinancialCutoffDays)
	// Round-trip through JSON to match JSONB deserialization (ints become float64)
	cfgJSON, _ := json.Marshal(org.FinancialCutoffConfig)
	var expectedCfg map[string]interface{}
	json.Unmarshal(cfgJSON, &expectedCfg)
	require.Equal(t, expectedCfg, got.FinancialCutoffConfig)
}

func TestOrganizationRepository_GetByID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationRepository(pool)
	got, err := repo.GetByID(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestOrganizationRepository_AddMembership_GetMembership(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationRepository(pool)
	now := time.Now().UTC()

	// Seed org and user
	orgID := uuid.New()
	userID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Membership Org", "mem-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, uniqueEmail(), uniqueUsername(), "Member", "Test", "hash", true, now, now)
	require.NoError(t, err)

	// Add membership
	membership := auth.NewOrganizationMembership(userID, orgID, "manager")
	membership.InvitedBy = &userID
	membership.InvitedAt = &now
	err = repo.AddMembership(context.Background(), membership)
	require.NoError(t, err)

	// Get membership back
	got, err := repo.GetMembership(context.Background(), userID, orgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, orgID, got.OrganizationID)
	require.Equal(t, userID, got.UserID)
	require.Equal(t, "manager", got.Role)
	require.True(t, got.IsActive)
	require.NotNil(t, got.InvitedBy)
	require.Equal(t, userID, *got.InvitedBy)
}

func TestOrganizationRepository_GetMembership_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationRepository(pool)
	got, err := repo.GetMembership(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestOrganizationRepository_AddMembership_Duplicate(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationRepository(pool)
	now := time.Now().UTC()

	orgID := uuid.New()
	userID := uuid.New()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Dup Org", "dup-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, uniqueEmail(), uniqueUsername(), "Dup", "Test", "hash", true, now, now)
	require.NoError(t, err)

	m := auth.NewOrganizationMembership(userID, orgID, "employee")
	err = repo.AddMembership(context.Background(), m)
	require.NoError(t, err)

	// Duplicate should fail (unique constraint)
	m2 := auth.NewOrganizationMembership(userID, orgID, "manager")
	err = repo.AddMembership(context.Background(), m2)
	require.Error(t, err)
}
