package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stretchr/testify/require"
)

func TestWGMemberRepository_ListByWG(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewWorkingGroupRepository(pool)
	now := time.Now().UTC()
	orgID, activityID, managerID, unitID := seedWGSetup(t, pool, now)

	wg, err := repo.Create(context.Background(), &working_group.WorkingGroup{
		OrgID: orgID, SubprojectID: activityID, Name: "WG Members Test",
		ManagerID: managerID, IsActive: true,
	})
	require.NoError(t, err)

	userID := seedUser(t, pool, now)
	m, err := repo.AddMember(context.Background(), &working_group.WorkingGroupMember{
		WGID: wg.ID, UserID: userID, UnitID: unitID,
		Role: "member", StartDate: now,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, m.ID)

	members, err := repo.ListMembers(context.Background(), wg.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.Equal(t, userID, members[0].UserID)
}

func TestWGMemberRepository_Add_ListByWG(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewWorkingGroupRepository(pool)
	now := time.Now().UTC()
	orgID, activityID, managerID, unitID := seedWGSetup(t, pool, now)

	wg, err := repo.Create(context.Background(), &working_group.WorkingGroup{
		OrgID: orgID, SubprojectID: activityID, Name: "WG Add Test",
		ManagerID: managerID, IsActive: true,
	})
	require.NoError(t, err)

	userID := seedUser(t, pool, now)
	m, err := repo.AddMember(context.Background(), &working_group.WorkingGroupMember{
		WGID: wg.ID, UserID: userID, UnitID: unitID,
		Role: "lead", IsDefaultSubproject: true, StartDate: now,
	})
	require.NoError(t, err)
	require.Equal(t, "lead", m.Role)
	require.True(t, m.IsDefaultSubproject)
}

func TestWGMemberRepository_Remove(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewWorkingGroupRepository(pool)
	now := time.Now().UTC()
	orgID, activityID, managerID, unitID := seedWGSetup(t, pool, now)

	wg, err := repo.Create(context.Background(), &working_group.WorkingGroup{
		OrgID: orgID, SubprojectID: activityID, Name: "WG Remove Test",
		ManagerID: managerID, IsActive: true,
	})
	require.NoError(t, err)

	userID := seedUser(t, pool, now)
	m, err := repo.AddMember(context.Background(), &working_group.WorkingGroupMember{
		WGID: wg.ID, UserID: userID, UnitID: unitID,
		Role: "member", StartDate: now,
	})
	require.NoError(t, err)

	err = repo.RemoveMember(context.Background(), m.ID)
	require.NoError(t, err)

	members, err := repo.ListMembers(context.Background(), wg.ID)
	require.NoError(t, err)
	require.Empty(t, members)
}
