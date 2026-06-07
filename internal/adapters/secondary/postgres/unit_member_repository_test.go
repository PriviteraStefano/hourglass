package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stretchr/testify/require"
)

func TestUnitMemberRepository_ListByUnit(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	u, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Test Unit", Code: "TU", HierarchyLevel: 0,
	})
	require.NoError(t, err)

	m, err := repo.AddMember(context.Background(), &unit.UnitMember{
		OrgID: orgID, UserID: userID, UnitID: u.ID,
		Role: "employee", StartDate: now,
	})
	require.NoError(t, err)
	require.NotEmpty(t, m.ID)

	members, err := repo.ListMembers(context.Background(), u.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.Equal(t, "Test User", members[0].UserName)
	require.Contains(t, members[0].UserEmail, "@test.com")
}

func TestUnitMemberRepository_Add_ListByUnit(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	u, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Member Unit", Code: "MU", HierarchyLevel: 0,
	})
	require.NoError(t, err)

	m, err := repo.AddMember(context.Background(), &unit.UnitMember{
		OrgID: orgID, UserID: userID, UnitID: u.ID,
		Role: "manager", StartDate: now,
	})
	require.NoError(t, err)
	require.Equal(t, "manager", m.Role)
	require.Equal(t, u.ID, m.UnitID)
}

func TestUnitMemberRepository_Remove(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	u, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Remove Unit", Code: "RU", HierarchyLevel: 0,
	})
	require.NoError(t, err)

	m, err := repo.AddMember(context.Background(), &unit.UnitMember{
		OrgID: orgID, UserID: userID, UnitID: u.ID,
		Role: "employee", StartDate: now,
	})
	require.NoError(t, err)

	err = repo.RemoveMember(context.Background(), m.ID)
	require.NoError(t, err)

	members, err := repo.ListMembers(context.Background(), u.ID)
	require.NoError(t, err)
	require.Empty(t, members)
}
