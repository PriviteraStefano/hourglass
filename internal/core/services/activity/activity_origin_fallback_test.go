package activity

import (
	"context"
	"testing"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	directiondomain "github.com/stefanoprivitera/hourglass/internal/core/domain/direction"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fallbackFixture wires the activity service with the direction-refs mock —
// the origin-fallback seam (D-13-32..34, FND-04): activities with empty
// origin refs derive manager-assignment refs from the first direction row,
// read-only, never written back.
type fallbackFixture struct {
	svc           *Service
	activityRepo  *testdata.MockActivityRepo
	directionRepo *testdata.MockDirectionRepo
}

func setupFallback() *fallbackFixture {
	activityRepo := &testdata.MockActivityRepo{}
	directionRepo := &testdata.MockDirectionRepo{}
	routingSvc := routing.NewService(&testdata.MockWorkingGroupRepo{}, activityRepo, &testdata.MockUnitRepo{})
	svc := NewService(activityRepo, &testdata.MockContractRepo{}, &testdata.MockUnitRepo{}, &testdata.MockOrgRepo{}, &testdata.MockTicketRepo{}, directionRepo, routingSvc)
	return &fallbackFixture{svc: svc, activityRepo: activityRepo, directionRepo: directionRepo}
}

// TestActivityOriginFallback_GetByID proves the D-13-32..34 contract on the
// single-read path: the A4 predicate (OriginType == nil) triggers derivation
// from FirstDirectionRefs; empty refs stay empty; stored origin refs block
// the fallback entirely (never overridden, never written back).
func TestActivityOriginFallback_GetByID(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("empty origin derives refs from the first direction row", func(t *testing.T) {
		f := setupFallback()
		a := seedActivity(f.activityRepo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID })
		assignedBy, assignedTo := uuid.New(), uuid.New()
		f.directionRepo.SetDirectionRefs(&directiondomain.DirectionRefs{AssignedBy: &assignedBy, AssignedTo: &assignedTo})

		got, err := f.svc.GetByID(ctx, orgID, a.ID)
		require.NoError(t, err)
		require.NotNil(t, got.AssignedBy, "derived assigned_by must be set")
		require.NotNil(t, got.AssignedTo, "derived assigned_to must be set")
		assert.Equal(t, assignedBy, *got.AssignedBy)
		assert.Equal(t, assignedTo, *got.AssignedTo)
	})

	t.Run("no direction rows leaves refs empty", func(t *testing.T) {
		f := setupFallback()
		a := seedActivity(f.activityRepo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID })

		got, err := f.svc.GetByID(ctx, orgID, a.ID)
		require.NoError(t, err)
		assert.Nil(t, got.AssignedBy, "refs stay empty when no direction row exists")
		assert.Nil(t, got.AssignedTo)
	})

	t.Run("stored origin refs stay authoritative — fallback not invoked", func(t *testing.T) {
		f := setupFallback()
		ot := activitydomain.OriginTypeCustomerTicket
		storedBy, storedTo := uuid.New(), uuid.New()
		a := seedActivity(f.activityRepo, func(a *activitydomain.ActivityResponse) {
			a.OrgID = orgID
			a.OriginType = &ot
			a.AssignedBy = &storedBy
			a.AssignedTo = &storedTo
		})
		calls := 0
		f.directionRepo.FirstDirectionRefsFn = func(ctx context.Context, oid, aid uuid.UUID) (*directiondomain.DirectionRefs, error) {
			calls++
			return &directiondomain.DirectionRefs{}, nil
		}

		got, err := f.svc.GetByID(ctx, orgID, a.ID)
		require.NoError(t, err)
		assert.Zero(t, calls, "the fallback must not run when origin_type is set (D-13-34)")
		assert.Equal(t, storedBy, *got.AssignedBy, "stored refs are authoritative")
		assert.Equal(t, storedTo, *got.AssignedTo)
	})
}

// TestActivityOriginFallback_List proves the batch enrichment on the org
// list path: every empty-origin row derives from FirstDirectionRefs, rows
// with a stored origin never derive (Pitfall 5).
func TestActivityOriginFallback_List(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("List enriches every empty-origin row", func(t *testing.T) {
		f := setupFallback()
		assignedBy, assignedTo := uuid.New(), uuid.New()
		f.directionRepo.SetDirectionRefs(&directiondomain.DirectionRefs{AssignedBy: &assignedBy, AssignedTo: &assignedTo})
		a1 := seedActivity(f.activityRepo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID })
		a2 := seedActivity(f.activityRepo, func(a *activitydomain.ActivityResponse) { a.OrgID = orgID })
		ot := activitydomain.OriginTypeEmployeeProposal
		propBy := uuid.New()
		a3 := seedActivity(f.activityRepo, func(a *activitydomain.ActivityResponse) {
			a.OrgID = orgID
			a.OriginType = &ot
			a.ProposedBy = &propBy
		})

		got, err := f.svc.List(ctx, orgID, nil)
		require.NoError(t, err)
		require.Len(t, got, 3)

		byID := make(map[uuid.UUID]activitydomain.ActivityResponse, len(got))
		for _, a := range got {
			byID[a.ID] = a
		}
		assert.Equal(t, assignedBy, *byID[a1.ID].AssignedBy, "empty-origin row 1 derives")
		assert.Equal(t, assignedTo, *byID[a2.ID].AssignedTo, "empty-origin row 2 derives")
		assert.Nil(t, byID[a3.ID].AssignedBy, "stored-origin rows never derive (Pitfall 5)")
		assert.Nil(t, byID[a3.ID].AssignedTo)
	})
}
