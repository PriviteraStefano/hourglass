package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stretchr/testify/require"
)

// seedWGSetup creates all FK dependencies needed for a working group and returns
// (orgID, activityID, managerID, unitID).
func seedWGSetup(t *testing.T, pool *pgxpool.Pool, now time.Time) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	orgID := seedOrg(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	managerID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	return orgID, activityID, managerID, unitID
}

func TestWorkingGroupRepository_ListByOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewWorkingGroupRepository(pool)
	now := time.Now().UTC()
	orgID, activityID, managerID, _ := seedWGSetup(t, pool, now)

	wg1 := &working_group.WorkingGroup{
		OrgID: orgID, SubprojectID: activityID, Name: "Alpha Team",
		ManagerID: managerID, IsActive: true,
	}
	wg2 := &working_group.WorkingGroup{
		OrgID: orgID, SubprojectID: activityID, Name: "Beta Team",
		ManagerID: managerID, IsActive: true,
	}

	_, err := repo.Create(context.Background(), wg1)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), wg2)
	require.NoError(t, err)

	// List all by org
	wgs, err := repo.ListByOrg(context.Background(), orgID, nil)
	require.NoError(t, err)
	require.Len(t, wgs, 2)
	require.Equal(t, "Alpha Team", wgs[0].Name)

	// Filter by activity
	filtered, err := repo.ListByOrg(context.Background(), orgID, &activityID)
	require.NoError(t, err)
	require.Len(t, filtered, 2)

	// Different activity — should be empty
	otherActivity := seedActivity(t, pool, orgID, "internal", nil, now)
	empty, err := repo.ListByOrg(context.Background(), orgID, &otherActivity)
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestWorkingGroupRepository_Create_GetByID(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewWorkingGroupRepository(pool)
	now := time.Now().UTC()
	orgID, activityID, managerID, unitID := seedWGSetup(t, pool, now)

	wg := &working_group.WorkingGroup{
		OrgID:        orgID,
		SubprojectID: activityID,
		Name:        "Core Team",
		Description: "Core development team",
		UnitIDs:     []string{unitID.String()},
		ManagerID:   managerID,
		DelegateIDs: []string{unitID.String()},
		IsActive:    true,
	}

	created, err := repo.Create(context.Background(), wg)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, "Core Team", created.Name)
	require.Len(t, created.UnitIDs, 1)
	require.Len(t, created.DelegateIDs, 1)
	require.Equal(t, unitID.String(), created.UnitIDs[0])

	// Round-trip via GetByID
	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Name, got.Name)
	require.Equal(t, created.Description, got.Description)
	require.Equal(t, len(created.UnitIDs), len(got.UnitIDs))
	require.Equal(t, created.UnitIDs[0], got.UnitIDs[0])
	require.Equal(t, created.DelegateIDs[0], got.DelegateIDs[0])
	require.True(t, got.IsActive)
}

func TestWorkingGroupRepository_GetByID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewWorkingGroupRepository(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, working_group.ErrWorkingGroupNotFound)
}

func TestWorkingGroupRepository_Update(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewWorkingGroupRepository(pool)
	now := time.Now().UTC()
	orgID, activityID, managerID, unitID := seedWGSetup(t, pool, now)

	created, err := repo.Create(context.Background(), &working_group.WorkingGroup{
		OrgID: orgID, SubprojectID: activityID, Name: "Old Name",
		ManagerID: managerID, IsActive: true,
	})
	require.NoError(t, err)

	created.Name = "Updated Name"
	created.Description = "Updated description"
	created.UnitIDs = []string{unitID.String()}

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.Equal(t, "Updated Name", updated.Name)
	require.Equal(t, "Updated description", updated.Description)
	require.Len(t, updated.UnitIDs, 1)

	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "Updated Name", got.Name)
}

func TestWorkingGroupRepository_Delete(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewWorkingGroupRepository(pool)
	now := time.Now().UTC()
	orgID, activityID, managerID, _ := seedWGSetup(t, pool, now)

	created, err := repo.Create(context.Background(), &working_group.WorkingGroup{
		OrgID: orgID, SubprojectID: activityID, Name: "To Delete",
		ManagerID: managerID, IsActive: true,
	})
	require.NoError(t, err)

	err = repo.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(context.Background(), created.ID)
	require.ErrorIs(t, err, working_group.ErrWorkingGroupNotFound)
}

func TestWorkingGroupRepository_HasMembers(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewWorkingGroupRepository(pool)
	now := time.Now().UTC()
	orgID, activityID, managerID, unitID := seedWGSetup(t, pool, now)

	wg, err := repo.Create(context.Background(), &working_group.WorkingGroup{
		OrgID: orgID, SubprojectID: activityID, Name: "Member Check",
		ManagerID: managerID, IsActive: true,
	})
	require.NoError(t, err)

	hasMembers, err := repo.HasMembers(context.Background(), wg.ID)
	require.NoError(t, err)
	require.False(t, hasMembers)

	// Add a member
	userID := seedUser(t, pool, now)
	_, err = repo.AddMember(context.Background(), &working_group.WorkingGroupMember{
		WGID: wg.ID, UserID: userID, UnitID: unitID,
		Role: "member", StartDate: now,
	})
	require.NoError(t, err)

	hasMembers, err = repo.HasMembers(context.Background(), wg.ID)
	require.NoError(t, err)
	require.True(t, hasMembers)
}
