package activity

import (
	"context"
	"testing"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	unitdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestService_BeneficiaryUnit — COV-05 / T-12-06: the same-org fetch-and-
// compare gate on both Create and Update. A unit from another org must be
// rejected; a same-org unit passes and round-trips through the mock.
// ---------------------------------------------------------------------------

// seedUnitInOrg registers a unit belonging to orgID in the mock unit repo
// (MockUnitRepo keys by string id, matching the adapter's GetByID(string)).
func seedUnitInOrg(unitRepo *testdata.MockUnitRepo, orgID uuid.UUID, overrides ...func(*unitdomain.Unit)) uuid.UUID {
	u := testdata.NewUnit(overrides...)
	u.OrgID = orgID
	if unitRepo.Units == nil {
		unitRepo.Units = make(map[string]*unitdomain.Unit)
	}
	unitRepo.Units[u.ID] = &u
	id, _ := uuid.Parse(u.ID)
	return id
}

func TestService_Create_BeneficiaryUnit(t *testing.T) {
	orgID := uuid.New()

	t.Run("same-org unit accepted and round-trips", func(t *testing.T) {
		svc, repo, _, unitRepo := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		unitID := seedUnitInOrg(unitRepo, orgID)

		req := validCreateReq()
		req.BeneficiaryUnitID = &unitID
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.NotNil(t, created.BeneficiaryUnitID)
		assert.Equal(t, unitID, *created.BeneficiaryUnitID)
	})

	t.Run("cross-org unit rejected (T-12-06)", func(t *testing.T) {
		svc, repo, _, unitRepo := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		otherOrgID := uuid.New()
		crossOrgUnitID := seedUnitInOrg(unitRepo, otherOrgID)

		req := validCreateReq()
		req.BeneficiaryUnitID = &crossOrgUnitID
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("missing unit rejected as invalid request (WR-05)", func(t *testing.T) {
		svc, repo, _, _ := setupService(t)
		repo.Kinds = map[string]bool{orgID.String() + ":engagement": true}

		req := validCreateReq()
		req.BeneficiaryUnitID = ptr(uuid.New())
		created, err := svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})
}

func TestService_Update_BeneficiaryUnit(t *testing.T) {
	orgID := uuid.New()

	t.Run("same-org unit accepted on update", func(t *testing.T) {
		svc, repo, _, unitRepo := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })
		unitID := seedUnitInOrg(unitRepo, orgID)

		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, a.ID, &activitydomain.UpdateActivityRequest{
			BeneficiaryUnitID: &unitID,
		})
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.NotNil(t, updated.BeneficiaryUnitID)
		assert.Equal(t, unitID, *updated.BeneficiaryUnitID)
	})

	t.Run("cross-org unit rejected on update (T-12-06)", func(t *testing.T) {
		svc, repo, _, unitRepo := setupService(t)
		a := seedActivity(repo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID; a.CreatedByOrgID = orgID })
		otherOrgID := uuid.New()
		crossOrgUnitID := seedUnitInOrg(unitRepo, otherOrgID)

		updated, err := svc.Update(context.Background(), string(models.RoleFinance), orgID, a.ID, &activitydomain.UpdateActivityRequest{
			BeneficiaryUnitID: &crossOrgUnitID,
		})
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, updated)
	})
}
