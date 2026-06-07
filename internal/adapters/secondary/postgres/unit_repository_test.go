package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stretchr/testify/require"
)

func TestUnitRepository_ListByOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	u1 := &unit.Unit{OrgID: orgID, Name: "Engineering", Description: "Engineering dept", Code: "ENG", HierarchyLevel: 0}
	u2 := &unit.Unit{OrgID: orgID, Name: "Marketing", Description: "Marketing dept", Code: "MKT", HierarchyLevel: 0}

	_, err := repo.Create(context.Background(), u1)
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), u2)
	require.NoError(t, err)

	units, err := repo.ListByOrg(context.Background(), orgID)
	require.NoError(t, err)
	require.Len(t, units, 2)
	require.Equal(t, "Engineering", units[0].Name)
	require.Equal(t, "Marketing", units[1].Name)
}

func TestUnitRepository_Create_GetByID(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	created, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Engineering", Description: "Engineering dept",
		Code: "ENG", HierarchyLevel: 0,
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "Engineering", created.Name)

	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Name, got.Name)
	require.Equal(t, created.Code, got.Code)
}

func TestUnitRepository_GetByID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	_, err := repo.GetByID(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, unit.ErrUnitNotFound)
}

func TestUnitRepository_Update(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	created, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Old Name", Description: "Old desc", Code: "OLD", HierarchyLevel: 0,
	})
	require.NoError(t, err)

	created.Name = "New Name"
	created.Description = "New desc"
	created.Code = "NEW"

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.Equal(t, "New Name", updated.Name)
	require.Equal(t, "New desc", updated.Description)
	require.Equal(t, "NEW", updated.Code)

	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "New Name", got.Name)
	require.Equal(t, "New desc", got.Description)
	require.Equal(t, "NEW", got.Code)
}

func TestUnitRepository_Delete(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	created, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "To Delete", Code: "DEL", HierarchyLevel: 0,
	})
	require.NoError(t, err)

	err = repo.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(context.Background(), created.ID)
	require.ErrorIs(t, err, unit.ErrUnitNotFound)
}

func TestUnitRepository_GetDescendants(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	parent, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Parent", Code: "PAR", HierarchyLevel: 0,
	})
	require.NoError(t, err)

	child, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Child", Code: "CHD", HierarchyLevel: 1, ParentUnitID: parent.ID,
	})
	require.NoError(t, err)

	_, err = repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Grandchild", Code: "GCH", HierarchyLevel: 2, ParentUnitID: child.ID,
	})
	require.NoError(t, err)

	descendants, err := repo.GetDescendants(context.Background(), parent.ID)
	require.NoError(t, err)
	require.Len(t, descendants, 2) // child + grandchild, not parent
}

func TestUnitRepository_HasMembers(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	u, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Unit With Members", Code: "MEM", HierarchyLevel: 0,
	})
	require.NoError(t, err)

	hasMembers, err := repo.HasMembers(context.Background(), u.ID)
	require.NoError(t, err)
	require.False(t, hasMembers)
}

func TestUnitRepository_GetMemberCountsByOrg(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUnitRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	u1, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Unit A", Code: "UA", HierarchyLevel: 0,
	})
	require.NoError(t, err)
	u2, err := repo.Create(context.Background(), &unit.Unit{
		OrgID: orgID, Name: "Unit B", Code: "UB", HierarchyLevel: 0,
	})
	require.NoError(t, err)

	// Seed a user for the memberships
	userID := seedUser(t, pool, now)

	// Add 2 members to u1, 1 to u2
	for i := 0; i < 2; i++ {
		_, err = repo.AddMember(context.Background(), &unit.UnitMember{
			OrgID: orgID, UserID: userID, UnitID: u1.ID,
			Role: "employee", StartDate: now,
		})
		require.NoError(t, err)
	}
	_, err = repo.AddMember(context.Background(), &unit.UnitMember{
		OrgID: orgID, UserID: userID, UnitID: u2.ID,
		Role: "employee", StartDate: now,
	})
	require.NoError(t, err)

	counts, err := repo.GetMemberCountsByOrg(context.Background(), orgID)
	require.NoError(t, err)
	require.Equal(t, 2, counts[u1.ID])
	require.Equal(t, 1, counts[u2.ID])
}
