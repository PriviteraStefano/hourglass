package activity

import (
	"context"
	"testing"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupService(t *testing.T) (*Service, *testdata.MockActivityRepo, *testdata.MockContractRepo, *testdata.MockUnitRepo) {
	t.Helper()
	activityRepo := &testdata.MockActivityRepo{}
	contractRepo := &testdata.MockContractRepo{}
	unitRepo := &testdata.MockUnitRepo{}
	orgRepo := &testdata.MockOrgRepo{}
	ticketRepo := &testdata.MockTicketRepo{}
	auditRepo := &testdata.MockAuditLogRepo{}
	routingSvc := routing.NewService(&testdata.MockWorkingGroupRepo{}, activityRepo, unitRepo)
	svc := NewService(activityRepo, contractRepo, unitRepo, orgRepo, ticketRepo, auditRepo, routingSvc)
	return svc, activityRepo, contractRepo, unitRepo
}

func seedActivity(repo *testdata.MockActivityRepo, overrides ...func(*activitydomain.ActivityResponse)) *activitydomain.ActivityResponse {
	a := testdata.NewActivity(overrides...)
	if repo.Activities == nil {
		repo.Activities = make(map[uuid.UUID]*activitydomain.ActivityResponse)
	}
	repo.Activities[a.ID] = &a
	return &a
}

func validCreateReq() *activitydomain.CreateActivityRequest {
	return &activitydomain.CreateActivityRequest{
		Name:            "Engagement Alpha",
		Kind:            "engagement",
		GovernanceModel: models.GovernanceCreatorControlled,
	}
}

// ---------------------------------------------------------------------------
// TestService_Create
// ---------------------------------------------------------------------------

func TestService_Create(t *testing.T) {
	orgID := uuid.New()

	t.Run("valid engagement creates", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}

		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), validCreateReq())
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, "Engagement Alpha", created.Name)
		assert.Equal(t, orgID, created.OrgID)
		assert.True(t, created.IsActive)
	})

	t.Run("missing name rejected", func(t *testing.T) {
		svc, _, _, _ := setupService(t)
		req := validCreateReq()
		req.Name = ""
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("invalid governance model rejected", func(t *testing.T) {
		svc, _, _, _ := setupService(t)
		req := validCreateReq()
		req.GovernanceModel = "anarchy"
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("kind not in org catalog rejected (D-2)", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		// catalog only has "engagement"; req asks for "phase"
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		req := validCreateReq()
		req.Kind = "phase"

		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("parent from another org rejected (D-2)", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		otherOrgID := uuid.New()
		parent := seedActivity(repo, func(a *activitydomain.ActivityResponse) {
			a.OrgID = otherOrgID
		})

		req := validCreateReq()
		req.ParentID = &parent.ID
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("missing parent rejected", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}

		req := validCreateReq()
		req.ParentID = ptr(uuid.New())
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrActivityNotFound)
		assert.Nil(t, created)
	})

	t.Run("create with valid same-org parent succeeds (path check on insert)", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		parent := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID })

		req := validCreateReq()
		req.ParentID = &parent.ID
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, parent.ID, *created.ParentID)
	})

	t.Run("missing contract rejected (D-3)", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}

		req := validCreateReq()
		req.ContractID = ptr(uuid.New())
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, contractdomain.ErrContractNotFound)
		assert.Nil(t, created)
	})

	t.Run("existing contract accepted", func(t *testing.T) {
		svc, repo, contractRepo, _ := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		cid := uuid.New()
		contractRepo.Contracts = map[uuid.UUID]*contractdomain.ContractResponse{
			cid: {Contract: contractdomain.Contract{ID: cid, CreatedByOrgID: orgID}},
		}

		req := validCreateReq()
		req.ContractID = &cid
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, cid, *created.ContractID)
	})

	t.Run("personal activity without contract is first-class (D-8)", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":internal": true}

		req := &activitydomain.CreateActivityRequest{
			Name:            "Learn Rust",
			Kind:            "internal",
			GovernanceModel: models.GovernanceCreatorControlled,
		}
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Nil(t, created.ContractID)
	})
}

// ---------------------------------------------------------------------------
// TestService_GetByID
// ---------------------------------------------------------------------------

func TestService_GetByID(t *testing.T) {
	t.Run("existing activity returns", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		a := seedActivity(repo)

		got, err := svc.GetByID(context.Background(), a.OrgID, a.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, a.ID, got.ID)
	})

	t.Run("nonexistent returns error", func(t *testing.T) {
		svc, _, _, _ := setupService(t)
		got, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
		assert.ErrorIs(t, err, activitydomain.ErrActivityNotFound)
		assert.Nil(t, got)
	})
}

// ---------------------------------------------------------------------------
// TestService_Update
// ---------------------------------------------------------------------------

func TestService_Update(t *testing.T) {
	orgID := uuid.New()

	t.Run("non-finance role forbidden", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })

		updated, err := svc.Update(context.Background(), string(models.RoleEmployee), orgID, a.ID, &activitydomain.UpdateActivityRequest{Name: "X"})
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("finance can update own activity", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })

		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, a.ID, &activitydomain.UpdateActivityRequest{Name: "Renamed"})
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "Renamed", updated.Name)
	})

	t.Run("finance cannot update another org's activity", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		otherOrgID := uuid.New()
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = otherOrgID; a.CreatedByOrgID = otherOrgID })

		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, a.ID, &activitydomain.UpdateActivityRequest{Name: "X"})
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_UpdateParentValidation
// ---------------------------------------------------------------------------

// TestService_UpdateParentValidation covers the SPEC in-scope cycle-prevention
// path check on update (parent must exist, belong to the same org, and not
// sit inside the activity's own subtree) via the shared validateParent walk.
func TestService_UpdateParentValidation(t *testing.T) {
	orgID := uuid.New()
	svc, repo, _, _ := setupService(t)
	repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}

	// 3-level tree: root → child → grandchild
	root := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })
	child := seedActivity(repo, func(a *activitydomain.ActivityResponse) {
		a.OrgID = orgID
		a.CreatedByOrgID = orgID
		a.ParentID = &root.ID
	})
	grandchild := seedActivity(repo, func(a *activitydomain.ActivityResponse) {
		a.OrgID = orgID
		a.CreatedByOrgID = orgID
		a.ParentID = &child.ID
	})

	t.Run("reparent root under its grandchild rejected", func(t *testing.T) {
		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, root.ID, &activitydomain.UpdateActivityRequest{ParentID: &grandchild.ID})
		assert.ErrorIs(t, err, activitydomain.ErrActivityCycle)
		assert.Nil(t, updated)
	})

	t.Run("reparent child under its own descendant rejected", func(t *testing.T) {
		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, child.ID, &activitydomain.UpdateActivityRequest{ParentID: &grandchild.ID})
		assert.ErrorIs(t, err, activitydomain.ErrActivityCycle)
		assert.Nil(t, updated)
	})

	t.Run("self-parent rejected", func(t *testing.T) {
		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, root.ID, &activitydomain.UpdateActivityRequest{ParentID: &root.ID})
		assert.ErrorIs(t, err, activitydomain.ErrActivityCycle)
		assert.Nil(t, updated)
	})

	t.Run("valid reparent within same org succeeds", func(t *testing.T) {
		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, grandchild.ID, &activitydomain.UpdateActivityRequest{ParentID: &root.ID})
		require.NoError(t, err)
		require.NotNil(t, updated)
	})

	t.Run("parent from another org rejected", func(t *testing.T) {
		otherOrgID := uuid.New()
		otherOrgActivity := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = otherOrgID; a.CreatedByOrgID = otherOrgID })

		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, root.ID, &activitydomain.UpdateActivityRequest{ParentID: &otherOrgActivity.ID})
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, updated)
	})

	t.Run("missing parent rejected", func(t *testing.T) {
		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, root.ID, &activitydomain.UpdateActivityRequest{ParentID: ptr(uuid.New())})
		assert.ErrorIs(t, err, activitydomain.ErrActivityNotFound)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_Delete
// ---------------------------------------------------------------------------

func TestService_Delete(t *testing.T) {
	orgID := uuid.New()

	t.Run("non-finance role forbidden", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })

		err := svc.Delete(context.Background(), string(models.RoleEmployee), orgID, a.ID)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
	})

	t.Run("not owner forbidden", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		otherOrgID := uuid.New()
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = otherOrgID; a.CreatedByOrgID = otherOrgID })

		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, a.ID)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
	})

	t.Run("has children blocked", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })
		repo.HasChildrenFn = func(ctx context.Context, id uuid.UUID) (bool, error) { return true, nil }

		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, a.ID)
		assert.ErrorIs(t, err, activitydomain.ErrHasChildren)
	})

	t.Run("active time entries blocked", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })
		repo.HasActiveTimeEntriesFn = func(ctx context.Context, id uuid.UUID) (bool, bool, error) { return true, false, nil }

		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, a.ID)
		assert.ErrorIs(t, err, activitydomain.ErrHasActiveTimeEntries)
	})

	t.Run("descendant time entries blocked", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })
		repo.HasActiveTimeEntriesFn = func(ctx context.Context, id uuid.UUID) (bool, bool, error) { return false, true, nil }

		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, a.ID)
		assert.ErrorIs(t, err, activitydomain.ErrHasActiveTimeEntries)
	})

	t.Run("active expenses blocked", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })
		repo.HasActiveExpensesFn = func(ctx context.Context, id uuid.UUID) (bool, error) { return true, nil }

		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, a.ID)
		assert.ErrorIs(t, err, activitydomain.ErrHasActiveExpenses)
	})

	t.Run("clean activity deletes", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })

		err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, a.ID)
		require.NoError(t, err)
		_, err = repo.Get(context.Background(), orgID, a.ID)
		assert.ErrorIs(t, err, activitydomain.ErrActivityNotFound)
	})
}

// ---------------------------------------------------------------------------
// TestService_List + ListChildren
// ---------------------------------------------------------------------------

func TestService_List(t *testing.T) {
	orgID := uuid.New()

	t.Run("returns org activities", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID })
		seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID })
		seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = uuid.New() })

		activities, err := svc.List(context.Background(), orgID, nil)
		require.NoError(t, err)
		assert.Len(t, activities, 2)
	})
}

func TestService_ListChildren(t *testing.T) {
	parentID := uuid.New()

	t.Run("returns direct children only", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.ParentID = &parentID })
		seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.ParentID = &parentID })
		seedActivity(repo) // no parent

		children, err := svc.ListChildren(context.Background(), parentID)
		require.NoError(t, err)
		assert.Len(t, children, 2)
	})
}

// ---------------------------------------------------------------------------
// TestService_Managers + Adopt (passthrough with role gating)
// ---------------------------------------------------------------------------

func TestService_AddRemoveManager(t *testing.T) {
	activityID := uuid.New()
	userID := uuid.New()

	t.Run("non-finance cannot add manager", func(t *testing.T) {
		svc, _, _, _ := setupService(t)
		m, err := svc.AddManager(context.Background(), string(models.RoleEmployee), activityID, userID)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, m)
	})

	t.Run("finance can add manager", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		m, err := svc.AddManager(context.Background(), string(models.RoleFinance), activityID, userID)
		require.NoError(t, err)
		require.NotNil(t, m)
		assert.Equal(t, activityID, m.ActivityID)
		_ = repo
	})

	t.Run("non-finance cannot remove manager", func(t *testing.T) {
		svc, _, _, _ := setupService(t)
		err := svc.RemoveManager(context.Background(), string(models.RoleEmployee), activityID, userID)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
	})

	t.Run("finance can remove manager", func(t *testing.T) {
		svc, _, _, _ := setupService(t)
		err := svc.RemoveManager(context.Background(), string(models.RoleFinance), activityID, userID)
		require.NoError(t, err)
	})
}

func TestService_Adopt(t *testing.T) {
	svc, _, _, _ := setupService(t)
	orgID, activityID := uuid.New(), uuid.New()

	adoption, err := svc.Adopt(context.Background(), orgID, activityID)
	require.NoError(t, err)
	require.NotNil(t, adoption)
	assert.Equal(t, activityID, adoption.ActivityID)
	assert.Equal(t, orgID, adoption.OrganizationID)
}

func ptr[T any](v T) *T { return &v }
