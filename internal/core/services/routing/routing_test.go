package routing

import (
	"context"
	"testing"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	unitdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	wgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type routingFixture struct {
	svc          *Service
	wgRepo       *testdata.MockWorkingGroupRepo
	activityRepo *testdata.MockActivityRepo
	unitRepo     *testdata.MockUnitRepo
}

func setupRouting(t *testing.T) *routingFixture {
	t.Helper()
	f := &routingFixture{
		wgRepo:       &testdata.MockWorkingGroupRepo{},
		activityRepo: &testdata.MockActivityRepo{},
		unitRepo:     &testdata.MockUnitRepo{},
	}
	f.svc = NewService(f.wgRepo, f.activityRepo, f.unitRepo)
	return f
}

// seedWG adds a working group anchored to an activity to the wg mock.
func (f *routingFixture) seedWG(orgID, activityID uuid.UUID, overrides ...func(*wgdomain.WorkingGroup)) *wgdomain.WorkingGroup {
	wg := &wgdomain.WorkingGroup{
		ID:           uuid.New(),
		OrgID:        orgID,
		SubprojectID: activityID, // maps to activities.activity_id (D-5)
		Name:         "Test WG",
		ManagerID:    uuid.New(),
		IsActive:     true,
	}
	for _, o := range overrides {
		o(wg)
	}
	if f.wgRepo.Groups == nil {
		f.wgRepo.Groups = make(map[uuid.UUID]*wgdomain.WorkingGroup)
	}
	f.wgRepo.Groups[wg.ID] = wg
	return wg
}

// seedActivity adds an activity to the activity mock.
func (f *routingFixture) seedActivity(orgID uuid.UUID, overrides ...func(*activitydomain.ActivityResponse)) *activitydomain.ActivityResponse {
	a := testdata.NewActivity(overrides...)
	a.OrgID = orgID
	if f.activityRepo.Activities == nil {
		f.activityRepo.Activities = make(map[uuid.UUID]*activitydomain.ActivityResponse)
	}
	f.activityRepo.Activities[a.ID] = &a
	return &a
}

// seedUnitWithManager adds a unit (optionally with a parent) whose members
// include a manager, to the unit mock.
func (f *routingFixture) seedUnitWithManager(unitID uuid.UUID, managerID uuid.UUID, parentUnitID string) *unitdomain.Unit {
	u := &unitdomain.Unit{
		ID:           unitID.String(),
		Name:         "Test Unit",
		ParentUnitID: parentUnitID,
	}
	if f.unitRepo.Units == nil {
		f.unitRepo.Units = make(map[string]*unitdomain.Unit)
	}
	f.unitRepo.Units[u.ID] = u
	if f.unitRepo.UnitMembers == nil {
		f.unitRepo.UnitMembers = make(map[string][]unitdomain.UnitMember)
	}
	f.unitRepo.UnitMembers[u.ID] = []unitdomain.UnitMember{
		{UserID: managerID, UnitID: u.ID, Role: "manager"},
	}
	return u
}

// ---------------------------------------------------------------------------
// TestRoutingAnchoredWGApproverSet — R-1 chain
// ---------------------------------------------------------------------------

func TestRoutingAnchoredWGApproverSet(t *testing.T) {
	orgID := uuid.New()
	activityID := uuid.New()
	wgManagerID := uuid.New()
	delegateID := uuid.New()

	t.Run("owner not in the set: approverIDs = manager + delegates, no skip", func(t *testing.T) {
		f := setupRouting(t)
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = wgManagerID
			wg.DelegateIDs = []string{delegateID.String()}
		})

		res, err := f.svc.ResolveManagerStage(context.Background(), orgID, activityID, uuid.New(), uuid.New())
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.ElementsMatch(t, []uuid.UUID{wgManagerID, delegateID}, res.ApproverIDs)
		assert.False(t, res.SkipToFinance)
		assert.False(t, res.RoleGated)
	})

	t.Run("owner IS the WG manager: skipToFinance", func(t *testing.T) {
		f := setupRouting(t)
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = wgManagerID
			wg.DelegateIDs = []string{delegateID.String()}
		})

		res, err := f.svc.ResolveManagerStage(context.Background(), orgID, activityID, uuid.New(), wgManagerID)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.ElementsMatch(t, []uuid.UUID{wgManagerID, delegateID}, res.ApproverIDs)
		assert.True(t, res.SkipToFinance)
	})

	t.Run("owner IS a delegate: skipToFinance", func(t *testing.T) {
		f := setupRouting(t)
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = wgManagerID
			wg.DelegateIDs = []string{delegateID.String()}
		})

		res, err := f.svc.ResolveManagerStage(context.Background(), orgID, activityID, uuid.New(), delegateID)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.SkipToFinance)
	})
}

// ---------------------------------------------------------------------------
// TestRoutingCommercialWithoutWGNotLoggable — R-2 enforcement
// ---------------------------------------------------------------------------

func TestRoutingCommercialWithoutWGNotLoggable(t *testing.T) {
	f := setupRouting(t)
	orgID := uuid.New()
	activityID := uuid.New()
	cid := uuid.New()
	f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
		a.ID = activityID
		a.ContractID = &cid
	})

	res, err := f.svc.ResolveManagerStage(context.Background(), orgID, activityID, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, activitydomain.ErrActivityNotLoggable)
	assert.Nil(t, res)
}

// ---------------------------------------------------------------------------
// TestRoutingUnitManagerFallback — R-2 fallback for personal activities
// ---------------------------------------------------------------------------

func TestRoutingUnitManagerFallback(t *testing.T) {
	orgID := uuid.New()
	activityID := uuid.New()
	unitID := uuid.New()
	unitManagerID := uuid.New()

	t.Run("routes to unit manager when owner is not the manager", func(t *testing.T) {
		f := setupRouting(t)
		f.seedUnitWithManager(unitID, unitManagerID, "")

		res, err := f.svc.ResolveManagerStage(context.Background(), orgID, activityID, unitID, uuid.New())
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, []uuid.UUID{unitManagerID}, res.ApproverIDs)
		assert.False(t, res.SkipToFinance)
		assert.False(t, res.RoleGated)
	})

	t.Run("skipToFinance when the unit manager IS the owner", func(t *testing.T) {
		f := setupRouting(t)
		f.seedUnitWithManager(unitID, unitManagerID, "")

		res, err := f.svc.ResolveManagerStage(context.Background(), orgID, activityID, unitID, unitManagerID)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, []uuid.UUID{unitManagerID}, res.ApproverIDs)
		assert.True(t, res.SkipToFinance)
	})
}

// ---------------------------------------------------------------------------
// TestRoutingUnitManagerUpwardWalk — parent-chain resolution
// ---------------------------------------------------------------------------

func TestRoutingUnitManagerUpwardWalk(t *testing.T) {
	f := setupRouting(t)
	orgID := uuid.New()
	activityID := uuid.New()
	childUnitID := uuid.New()
	parentUnitID := uuid.New()
	unitManagerID := uuid.New()

	// child unit has no manager; the manager sits on the parent unit
	child := &unitdomain.Unit{ID: childUnitID.String(), Name: "Child", ParentUnitID: parentUnitID.String()}
	f.unitRepo.Units = map[string]*unitdomain.Unit{child.ID: child}
	f.seedUnitWithManager(parentUnitID, unitManagerID, "")

	res, err := f.svc.ResolveManagerStage(context.Background(), orgID, activityID, childUnitID, uuid.New())
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, []uuid.UUID{unitManagerID}, res.ApproverIDs)
	assert.False(t, res.SkipToFinance)
	assert.False(t, res.RoleGated)
}

// ---------------------------------------------------------------------------
// TestRoutingTerminalRoleGated — org root without any manager
// ---------------------------------------------------------------------------

func TestRoutingTerminalRoleGated(t *testing.T) {
	f := setupRouting(t)
	orgID := uuid.New()
	activityID := uuid.New()
	unitID := uuid.New()

	// org root: no members, no parent — the walk exhausts
	root := &unitdomain.Unit{ID: unitID.String(), Name: "Root"}
	f.unitRepo.Units = map[string]*unitdomain.Unit{root.ID: root}

	res, err := f.svc.ResolveManagerStage(context.Background(), orgID, activityID, unitID, uuid.New())
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.RoleGated)
	assert.False(t, res.SkipToFinance)
	assert.Nil(t, res.ApproverIDs)
}

// ---------------------------------------------------------------------------
// TestResolveUnitManagerNotFound — missing unit short-circuits the walk
// ---------------------------------------------------------------------------

func TestResolveUnitManagerNotFound(t *testing.T) {
	f := setupRouting(t)

	managerID, found, err := f.svc.ResolveUnitManager(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, managerID)
	assert.False(t, found)
}
