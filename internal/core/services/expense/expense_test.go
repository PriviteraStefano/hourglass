package expense

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	domainexpense "github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stretchr/testify/require"
)

// TestService_Create_MapsUnitID proves POST /expenses no longer drops unit_id:
// the requested unit_id is mapped through the service into the expense the repo
// persists (expense_repository.go already inserts the column; the prior break
// was the handler/service never supplying it — Phase 16 known bug).
func TestService_Create_MapsUnitID(t *testing.T) {
	repo := &testdata.MockExpenseRepo{}
	svc := NewService(repo, &testdata.MockWorkingGroupRepo{}, &testdata.MockActivityRepo{}, &testdata.MockUnitRepo{})

	orgID := uuid.New()
	userID := uuid.New()
	activityID := uuid.New()
	unitID := uuid.New()

	created, err := svc.Create(context.Background(), &domainexpense.CreateExpenseRequest{
		OrgID:      orgID,
		UserID:     userID,
		ActivityID: activityID,
		UnitID:     &unitID,
		Category:   "mileage",
		Amount:     100,
		Date:       "2026-08-01",
	})
	require.NoError(t, err)
	require.Equal(t, unitID, created.UnitID, "service must return the mapped unit_id")

	// The repo received the unit_id — it was not silently dropped.
	stored, ok := repo.Expenses[created.ID]
	require.True(t, ok)
	require.Equal(t, unitID, stored.UnitID)
}

// TestService_Create_OmitsUnitIDWhenAbsent ensures a create without unit_id does
// not inherit a garbage value.
func TestService_Create_OmitsUnitIDWhenAbsent(t *testing.T) {
	repo := &testdata.MockExpenseRepo{}
	svc := NewService(repo, &testdata.MockWorkingGroupRepo{}, &testdata.MockActivityRepo{}, &testdata.MockUnitRepo{})

	created, err := svc.Create(context.Background(), &domainexpense.CreateExpenseRequest{
		OrgID:      uuid.New(),
		UserID:     uuid.New(),
		ActivityID: uuid.New(),
		Category:   "meal",
		Amount:     12.50,
		Date:       time.Now().Format("2006-01-02"),
	})
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, created.UnitID, "no unit_id requested → stored as zero UUID")
}
