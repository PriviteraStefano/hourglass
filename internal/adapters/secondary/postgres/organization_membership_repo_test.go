package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOrganizationMembershipRepository_ListByUser(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationMembershipRepository(pool)
	now := time.Now().UTC()

	userID := uuid.New()
	orgID1 := uuid.New()
	orgID2 := uuid.New()

	// Seed users and orgs
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, uniqueEmail(), uniqueUsername(), "List", "User", "hash", true, now, now)
	require.NoError(t, err)

	for _, oid := range []uuid.UUID{orgID1, orgID2} {
		_, err = pool.Exec(context.Background(),
			`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			oid, "Org-"+uuid.New().String()[:8], "slug-"+uuid.New().String()[:8], now, now)
		require.NoError(t, err)
	}

	// Create two memberships
	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), userID, orgID1, "employee", true, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), userID, orgID2, "manager", true, now)
	require.NoError(t, err)

	memberships, err := repo.ListByUser(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, memberships, 2)

	orgIDs := map[uuid.UUID]bool{}
	for _, m := range memberships {
		orgIDs[m.OrganizationID] = true
		require.Equal(t, userID, m.UserID)
	}
	require.True(t, orgIDs[orgID1])
	require.True(t, orgIDs[orgID2])

	// No memberships returns empty slice
	empty, err := repo.ListByUser(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestOrganizationMembershipRepository_ListByOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrganizationMembershipRepository(pool)
	now := time.Now().UTC()

	orgID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	// Seed org and users
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Org List", "org-list-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	for _, uid := range []uuid.UUID{userID1, userID2} {
		_, err = pool.Exec(context.Background(),
			`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			uid, uniqueEmail(), uniqueUsername(), "User", "Test", "hash", true, now, now)
		require.NoError(t, err)
	}

	// Create two memberships
	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), userID1, orgID, "employee", true, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), userID2, orgID, "finance", true, now)
	require.NoError(t, err)

	memberships, err := repo.ListByOrg(context.Background(), orgID)
	require.NoError(t, err)
	require.Len(t, memberships, 2)

	roles := map[string]bool{}
	for _, m := range memberships {
		roles[m.Role] = true
		require.Equal(t, orgID, m.OrganizationID)
	}
	require.True(t, roles["employee"])
	require.True(t, roles["finance"])

	// No memberships returns empty slice
	empty, err := repo.ListByOrg(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, empty)
}
