package activity

import (
	"context"
	"testing"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	auditdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	unitdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	wgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedProposal stores an inactive employee_proposal in the activity mock
// (the shape Create leaves behind per D-12).
func (f *originFixture) seedProposal(orgID, proposer uuid.UUID, overrides ...func(*activitydomain.ActivityResponse)) *activitydomain.ActivityResponse {
	ot := activitydomain.OriginTypeEmployeeProposal
	a := testdata.NewActivity(func(a *activitydomain.ActivityResponse) {
		a.OrgID = orgID
		a.CreatedByOrgID = orgID
		a.OriginType = &ot
		a.ProposedBy = &proposer
		a.IsActive = false
	})
	for _, o := range overrides {
		o(&a)
	}
	if f.activityRepo.Activities == nil {
		f.activityRepo.Activities = make(map[uuid.UUID]*activitydomain.ActivityResponse)
	}
	f.activityRepo.Activities[a.ID] = &a
	return &a
}

// seedWGForActivity anchors a working group to the activity with the given
// manager (BE-014 R-1 approver set = manager + delegates).
func (f *originFixture) seedWGForActivity(orgID, activityID, managerID uuid.UUID, delegateIDs ...string) {
	wg := &wgdomain.WorkingGroup{
		ID:           uuid.New(),
		OrgID:        orgID,
		SubprojectID: activityID,
		Name:         "Proposal WG",
		ManagerID:    managerID,
		DelegateIDs:  delegateIDs,
		IsActive:     true,
	}
	if f.wgRepo.Groups == nil {
		f.wgRepo.Groups = make(map[uuid.UUID]*wgdomain.WorkingGroup)
	}
	f.wgRepo.Groups[wg.ID] = wg
}

// seedPrimaryUnit gives the user a primary unit membership so the routing
// walk starts from a concrete unit.
func (f *originFixture) seedPrimaryUnit(userID uuid.UUID, unitID uuid.UUID) {
	if f.unitRepo.UnitMembers == nil {
		f.unitRepo.UnitMembers = make(map[string][]unitdomain.UnitMember)
	}
	f.unitRepo.UnitMembers[unitID.String()] = []unitdomain.UnitMember{
		{ID: uuid.New().String(), UserID: userID, UnitID: unitID.String(), IsPrimary: true, Role: "employee"},
	}
}

// ---------------------------------------------------------------------------
// TestProposal_Approve
// ---------------------------------------------------------------------------

func TestProposal_Approve(t *testing.T) {
	orgID := uuid.New()
	proposer := uuid.New()
	approver := uuid.New()

	t.Run("approver in set flips is_active + writes audit row", func(t *testing.T) {
		f := setupOrigin(t)
		prop := f.seedProposal(orgID, proposer)
		f.seedWGForActivity(orgID, prop.ID, approver)

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, approver, prop.ID)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.True(t, updated.IsActive)

		require.Len(t, f.auditRepo.Logs, 1)
		log := f.auditRepo.Logs[0]
		assert.Equal(t, "proposal_approved", log.Action)
		assert.Equal(t, orgID, log.OrgID)
		assert.Equal(t, prop.ID, log.EntityID)
		assert.Equal(t, "activity", log.EntityType)
		require.NotNil(t, log.ActorID)
		assert.Equal(t, approver, *log.ActorID)
		require.NotNil(t, log.Payload)
		assert.Equal(t, approver.String(), log.Payload["approver"])
	})

	t.Run("actor not in approver set forbidden", func(t *testing.T) {
		f := setupOrigin(t)
		prop := f.seedProposal(orgID, proposer)
		f.seedWGForActivity(orgID, prop.ID, approver)
		outsider := uuid.New()

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, outsider, prop.ID)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, updated)
		assert.Empty(t, f.auditRepo.Logs)
	})

	t.Run("actor == proposer forbidden (no self-approval)", func(t *testing.T) {
		f := setupOrigin(t)
		prop := f.seedProposal(orgID, proposer)
		f.seedWGForActivity(orgID, prop.ID, approver)

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, proposer, prop.ID)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, updated)
		assert.Empty(t, f.auditRepo.Logs)
	})

	t.Run("wrong origin not approvable", func(t *testing.T) {
		f := setupOrigin(t)
		ot := activitydomain.OriginTypeManagerAssignment
		prop := testdata.NewActivity(func(a *activitydomain.ActivityResponse) {
			a.OrgID = orgID
			a.CreatedByOrgID = orgID
			a.OriginType = &ot
			a.IsActive = true
		})
		if f.activityRepo.Activities == nil {
			f.activityRepo.Activities = make(map[uuid.UUID]*activitydomain.ActivityResponse)
		}
		f.activityRepo.Activities[prop.ID] = &prop

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, approver, prop.ID)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, updated)
	})

	t.Run("already approved not approvable", func(t *testing.T) {
		f := setupOrigin(t)
		prop := f.seedProposal(orgID, proposer, func(a *activitydomain.ActivityResponse) { a.IsActive = true })
		f.seedWGForActivity(orgID, prop.ID, approver)

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, approver, prop.ID)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, updated)
	})

	t.Run("missing activity not found", func(t *testing.T) {
		f := setupOrigin(t)
		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, approver, uuid.New())
		assert.ErrorIs(t, err, activitydomain.ErrActivityNotFound)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestProposal_RoutingModes
// ---------------------------------------------------------------------------

func TestProposal_RoutingModes(t *testing.T) {
	orgID := uuid.New()
	proposer := uuid.New()
	approver := uuid.New()
	delegateID := uuid.New()

	t.Run("skipToFinance (proposer is the only approver) rejected", func(t *testing.T) {
		f := setupOrigin(t)
		prop := f.seedProposal(orgID, proposer)
		// D-11: the WG manager IS the proposer → the resolution skips to finance
		f.seedWGForActivity(orgID, prop.ID, proposer)

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, approver, prop.ID)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, updated)
		assert.Empty(t, f.auditRepo.Logs)
	})

	t.Run("skipToFinance with delegate: delegate is a legitimate approver (WR-04)", func(t *testing.T) {
		f := setupOrigin(t)
		prop := f.seedProposal(orgID, proposer)
		// D-11 skip fires (WG manager IS the proposer), but the approver
		// set is {proposer, delegate} — the delegate must be able to approve
		// (pre-fix: the skip short-circuited to ErrForbidden before the
		// membership check, making the proposal unapprovable).
		f.seedWGForActivity(orgID, prop.ID, proposer, delegateID.String())

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, delegateID, prop.ID)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.True(t, updated.IsActive)
		require.Len(t, f.auditRepo.Logs, 1)
		assert.Equal(t, "proposal_approved", f.auditRepo.Logs[0].Action)
	})

	t.Run("roleGated resolution: manager role passes", func(t *testing.T) {
		f := setupOrigin(t)
		// No WG, no contract, no unit manager → terminal role-gated resolution
		prop := f.seedProposal(orgID, proposer)

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, approver, prop.ID)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.True(t, updated.IsActive)
		require.Len(t, f.auditRepo.Logs, 1)
		assert.Equal(t, "proposal_approved", f.auditRepo.Logs[0].Action)
	})

	t.Run("roleGated resolution: employee role fails", func(t *testing.T) {
		f := setupOrigin(t)
		prop := f.seedProposal(orgID, proposer)

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleEmployee), orgID, approver, prop.ID)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, updated)
		assert.Empty(t, f.auditRepo.Logs)
	})

	t.Run("unit-manager fallback path (R-2) resolves through primary unit", func(t *testing.T) {
		f := setupOrigin(t)
		prop := f.seedProposal(orgID, proposer)
		unitID := uuid.New()
		f.seedPrimaryUnit(proposer, unitID)
		// unit manager = approver (R-2 fallback for personal activities)
		f.unitRepo.UnitMembers[unitID.String()] = append(f.unitRepo.UnitMembers[unitID.String()],
			unitdomain.UnitMember{ID: uuid.New().String(), UserID: approver, UnitID: unitID.String(), IsPrimary: false, Role: "manager"})

		updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, approver, prop.ID)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.True(t, updated.IsActive)
		require.Len(t, f.auditRepo.Logs, 1)
	})
}

// ---------------------------------------------------------------------------
// TestProposal_EndToEnd — a proposal created through the Task 2 create path
// is approved through ApproveProposal (unit-level integration).
// ---------------------------------------------------------------------------

func TestProposal_EndToEnd(t *testing.T) {
	orgID := uuid.New()
	proposer := uuid.New()
	approver := uuid.New()

	f := setupOrigin(t)
	f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}

	ot := activitydomain.OriginTypeEmployeeProposal
	created, err := f.svc.Create(context.Background(), string(models.RoleEmployee), orgID, proposer, &activitydomain.CreateActivityRequest{
		Name:            "New engagement proposal",
		Kind:            "engagement",
		GovernanceModel: models.GovernanceCreatorControlled,
		OriginType:      &ot,
		ProposedBy:      &proposer,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.False(t, created.IsActive, "proposal must land is_active=false (D-12)")

	// route: anchor a WG with the approver as manager
	f.seedWGForActivity(orgID, created.ID, approver)

	updated, err := f.svc.ApproveProposal(context.Background(), string(models.RoleManager), orgID, approver, created.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.True(t, updated.IsActive, "approval flips is_active")

	require.Len(t, f.auditRepo.Logs, 1)
	require.IsType(t, &auditdomain.AuditLog{}, f.auditRepo.Logs[0])
	assert.Equal(t, "proposal_approved", f.auditRepo.Logs[0].Action)
	assert.Equal(t, created.ID, f.auditRepo.Logs[0].EntityID)
}
