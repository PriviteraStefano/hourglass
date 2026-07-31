package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stretchr/testify/require"
)

// seedContractLinkedActivity creates an engagement activity with a contract
// (and its customer) so the commercial chain (D-3) resolves.
func seedContractLinkedActivity(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) (contractID, activityID uuid.UUID) {
	t.Helper()
	customerID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO customers (id, org_id, name, contact_name, email, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $6)`,
		customerID, orgID, "Activity Customer", "Contact", "activity@test.com", now)
	require.NoError(t, err)

	contractID = uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO contracts (id, name, km_rate, currency, customer_id, governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, false, true, $8, $8)`,
		contractID, "Activity Contract", 0.42, "EUR", customerID, "creator_controlled", orgID, now)
	require.NoError(t, err)

	activityID = seedActivity(t, pool, orgID, "engagement", nil, now)
	_, err = pool.Exec(context.Background(),
		`UPDATE activities SET contract_id = $1 WHERE id = $2`,
		contractID, activityID)
	require.NoError(t, err)
	return contractID, activityID
}

func TestActivityRepository_Create_Get(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	contractID, _ := seedContractLinkedActivity(t, pool, orgID, now)
	seedActivityKind(t, pool, orgID, "phase")

	req := &activitydomain.CreateActivityRequest{
		Name:            "Mobile App",
		Description:     "Customer-facing mobile app",
		Kind:            "phase",
		ContractID:      &contractID,
		GovernanceModel: "creator_controlled",
		IsShared:        false,
	}

	created, err := repo.Create(context.Background(), orgID, req)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, orgID, created.OrgID)
	require.Equal(t, "Mobile App", created.Name)
	require.Equal(t, activitydomain.ActivityKind("phase"), created.Kind)
	require.Equal(t, contractID, *created.ContractID)
	require.Equal(t, "Activity Contract", created.ContractName)
	require.True(t, created.IsActive)
	require.Nil(t, created.ParentID)
	require.Nil(t, created.Billable)

	// Get
	got, err := repo.Get(context.Background(), orgID, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "Activity Contract", got.ContractName)
}

func TestActivityRepository_Get_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	_, err := repo.Get(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, activitydomain.ErrActivityNotFound)
}

func TestActivityRepository_List_Scopes(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	otherOrgID := seedOrg(t, pool, now)

	own := seedActivity(t, pool, orgID, "engagement", nil, now)
	_ = own
	_ = otherOrgID

	// own scope
	ownList, err := repo.List(context.Background(), orgID, &activitydomain.ActivityFilter{})
	require.NoError(t, err)
	require.Len(t, ownList, 1)

	// other org sees nothing
	empty, err := repo.List(context.Background(), otherOrgID, &activitydomain.ActivityFilter{})
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestActivityRepository_ListChildren_ListByContract(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	contractID, rootID := seedContractLinkedActivity(t, pool, orgID, now)

	child1 := seedActivity(t, pool, orgID, "phase", &rootID, now)
	child2 := seedActivity(t, pool, orgID, "task", &rootID, now)

	children, err := repo.ListChildren(context.Background(), rootID)
	require.NoError(t, err)
	require.Len(t, children, 2)
	gotIDs := map[uuid.UUID]bool{}
	for _, c := range children {
		gotIDs[c.ID] = true
	}
	require.True(t, gotIDs[child1])
	require.True(t, gotIDs[child2])

	byContract, err := repo.ListByContract(context.Background(), contractID)
	require.NoError(t, err)
	require.Len(t, byContract, 1)
	require.Equal(t, rootID, byContract[0].ID)
}

func TestActivityRepository_Adopt(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	adopterID := seedOrg(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	adoption, err := repo.Adopt(context.Background(), adopterID, activityID)
	require.NoError(t, err)
	require.Equal(t, adopterID, adoption.OrganizationID)
	require.Equal(t, activityID, adoption.ActivityID)

	// duplicate adoption rejected
	_, err = repo.Adopt(context.Background(), adopterID, activityID)
	require.ErrorIs(t, err, activitydomain.ErrAlreadyAdopted)

	// adopter sees it in "adopted" scope
	adopted, err := repo.List(context.Background(), adopterID, &activitydomain.ActivityFilter{Scope: "adopted"})
	require.NoError(t, err)
	require.Len(t, adopted, 1)
	require.True(t, adopted[0].IsAdopted)
}

func TestActivityRepository_Update_Delete(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)

	name := "Renamed"
	isShared := true
	updated, err := repo.Update(context.Background(), orgID, activityID, &activitydomain.UpdateActivityRequest{
		Name:     name,
		IsShared: &isShared,
	})
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.Name)
	require.True(t, updated.IsShared)

	// wrong org cannot update
	_, err = repo.Update(context.Background(), uuid.New(), activityID, &activitydomain.UpdateActivityRequest{Name: "Nope"})
	require.ErrorIs(t, err, activitydomain.ErrActivityNotFound)

	// delete
	err = repo.Delete(context.Background(), orgID, activityID)
	require.NoError(t, err)
	_, err = repo.Get(context.Background(), orgID, activityID)
	require.ErrorIs(t, err, activitydomain.ErrActivityNotFound)
}

func TestActivityRepository_Delete_WithChild(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	rootID := seedActivity(t, pool, orgID, "engagement", nil, now)
	seedActivity(t, pool, orgID, "task", &rootID, now)

	err := repo.Delete(context.Background(), orgID, rootID)
	require.Error(t, err) // ON DELETE RESTRICT on parent_id
}

func TestActivityRepository_Managers(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	activityID := seedActivity(t, pool, orgID, "engagement", nil, now)
	userID := seedUser(t, pool, now)

	m, err := repo.AddManager(context.Background(), activityID, userID)
	require.NoError(t, err)
	require.Equal(t, activityID, m.ActivityID)
	require.Equal(t, userID, m.UserID)
	require.NotEmpty(t, m.UserName)

	managers, err := repo.ListManagers(context.Background(), activityID)
	require.NoError(t, err)
	require.Len(t, managers, 1)

	err = repo.RemoveManager(context.Background(), activityID, userID)
	require.NoError(t, err)
	managers, err = repo.ListManagers(context.Background(), activityID)
	require.NoError(t, err)
	require.Empty(t, managers)
}

func TestActivityRepository_GetAncestry_ThreeLevel(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	engagementID := seedActivity(t, pool, orgID, "engagement", nil, now)
	phaseID := seedActivity(t, pool, orgID, "phase", &engagementID, now)
	taskID := seedActivity(t, pool, orgID, "task", &phaseID, now)

	chain, err := repo.GetAncestry(context.Background(), taskID)
	require.NoError(t, err)
	require.Len(t, chain, 3)
	require.Equal(t, taskID, chain[0].ID)
	require.Equal(t, phaseID, chain[1].ID)
	require.Equal(t, engagementID, chain[2].ID)

	// Ancestry of the root is just the root
	rootChain, err := repo.GetAncestry(context.Background(), engagementID)
	require.NoError(t, err)
	require.Len(t, rootChain, 1)
	require.Equal(t, engagementID, rootChain[0].ID)
}

func TestActivityRepository_ResolveCommercialContext_Grandparent(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	contractID, engagementID := seedContractLinkedActivity(t, pool, orgID, now)
	phaseID := seedActivity(t, pool, orgID, "phase", &engagementID, now)
	taskID := seedActivity(t, pool, orgID, "task", &phaseID, now)

	// Nested activity whose GRANDPARENT has the contract (D-3)
	ctx, err := repo.ResolveCommercialContext(context.Background(), taskID)
	require.NoError(t, err)
	require.NotNil(t, ctx)
	require.Equal(t, contractID, *ctx.ContractID)
	require.NotNil(t, ctx.CustomerID)

	// The contract-linked activity itself resolves directly
	ctx2, err := repo.ResolveCommercialContext(context.Background(), engagementID)
	require.NoError(t, err)
	require.NotNil(t, ctx2)
	require.Equal(t, contractID, *ctx2.ContractID)
}

func TestActivityRepository_ResolveCommercialContext_InternalNil(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	internalRoot := seedActivity(t, pool, orgID, "internal", nil, now)
	child := seedActivity(t, pool, orgID, "task", &internalRoot, now)

	ctx, err := repo.ResolveCommercialContext(context.Background(), child)
	require.NoError(t, err)
	require.Nil(t, ctx) // purely internal tree → nil
}

func TestActivityRepository_ResolveBillability_NearestWins(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	// Root explicitly non-billable; child explicitly billable → child wins (D-7)
	falseVal := false
	trueVal := true
	rootID := seedActivity(t, pool, orgID, "engagement", nil, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE activities SET billable = $1 WHERE id = $2`, falseVal, rootID)
	require.NoError(t, err)
	childID := seedActivity(t, pool, orgID, "task", &rootID, now)
	_, err = pool.Exec(context.Background(),
		`UPDATE activities SET billable = $1 WHERE id = $2`, trueVal, childID)
	require.NoError(t, err)

	res, err := repo.ResolveBillability(context.Background(), childID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, *res)

	// Nearest non-NULL: grandchild inherits child's explicit TRUE
	grandchildID := seedActivity(t, pool, orgID, "task", &childID, now)
	res2, err := repo.ResolveBillability(context.Background(), grandchildID)
	require.NoError(t, err)
	require.True(t, *res2)
}

func TestActivityRepository_ResolveBillability_ContractDefault(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	// Contract-linked root with NULL billable → contract default billable (D-7)
	_, engagementID := seedContractLinkedActivity(t, pool, orgID, now)
	phaseID := seedActivity(t, pool, orgID, "phase", &engagementID, now)

	res, err := repo.ResolveBillability(context.Background(), phaseID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, *res)

	// Internal root with NULL billable → non-billable (D-7)
	internalRoot := seedActivity(t, pool, orgID, "internal", nil, now)
	res2, err := repo.ResolveBillability(context.Background(), internalRoot)
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.False(t, *res2)

	// Explicit FALSE overrides the contract default
	falseVal := false
	_, err = pool.Exec(context.Background(),
		`UPDATE activities SET billable = $1 WHERE id = $2`, falseVal, phaseID)
	require.NoError(t, err)
	res3, err := repo.ResolveBillability(context.Background(), phaseID)
	require.NoError(t, err)
	require.False(t, *res3)
}

func TestActivityRepository_HasActiveTimeEntries(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)

	rootID := seedActivity(t, pool, orgID, "engagement", nil, now)
	childID := seedActivity(t, pool, orgID, "task", &rootID, now)

	// No entries yet
	has, hasDesc, err := repo.HasActiveTimeEntries(context.Background(), rootID)
	require.NoError(t, err)
	require.False(t, has)
	require.False(t, hasDesc)

	// Entry on the CHILD (subtree of root)
	_, err = pool.Exec(context.Background(),
		`INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours, description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, $10, $10)`,
		uuid.New(), orgID, userID, childID, unitID, 4.0, "subtree entry", now, "submitted", now)
	require.NoError(t, err)

	has, hasDesc, err = repo.HasActiveTimeEntries(context.Background(), rootID)
	require.NoError(t, err)
	require.False(t, has)
	require.True(t, hasDesc)

	// Entry directly on root
	_, err = pool.Exec(context.Background(),
		`INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours, description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, $10, $10)`,
		uuid.New(), orgID, userID, rootID, unitID, 3.0, "root entry", now, "draft", now)
	require.NoError(t, err)

	has, _, err = repo.HasActiveTimeEntries(context.Background(), rootID)
	require.NoError(t, err)
	require.True(t, has)
}

func TestActivityRepository_HasActiveExpenses(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewActivityRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)

	rootID := seedActivity(t, pool, orgID, "engagement", nil, now)

	has, err := repo.HasActiveExpenses(context.Background(), rootID)
	require.NoError(t, err)
	require.False(t, has)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO expenses (id, org_id, user_id, activity_id, unit_id, category, amount, description, expense_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, false, $11, $11)`,
		uuid.New(), orgID, userID, rootID, unitID, "meal", 20.0, "submitted expense", now, "submitted", now)
	require.NoError(t, err)

	has, err = repo.HasActiveExpenses(context.Background(), rootID)
	require.NoError(t, err)
	require.True(t, has)
}
