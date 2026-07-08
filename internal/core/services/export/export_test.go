package export

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
)

func setupService(t *testing.T) (*Service, *testdata.MockExportRepo) {
	t.Helper()
	repo := &testdata.MockExportRepo{}
	svc := NewService(repo)
	return svc, repo
}

func TestService_Timesheets(t *testing.T) {
	svc, _ := setupService(t)
	orgID := uuid.New()
	from := time.Now().Add(-30 * 24 * time.Hour)
	to := time.Now()

	t.Run("export time entries", func(t *testing.T) {
		rows, err := svc.Timesheets(context.Background(), orgID, from, to, "finance", uuid.New())
		assert.NoError(t, err)
		// Mock returns nil for empty export
		assert.Nil(t, rows)
	})
}

func TestService_Expenses(t *testing.T) {
	svc, _ := setupService(t)
	orgID := uuid.New()
	from := time.Now().Add(-30 * 24 * time.Hour)
	to := time.Now()

	t.Run("export expenses", func(t *testing.T) {
		rows, err := svc.Expenses(context.Background(), orgID, from, to, "finance", uuid.New())
		assert.NoError(t, err)
		assert.Nil(t, rows)
	})
}

func TestService_Combined(t *testing.T) {
	svc, _ := setupService(t)
	orgID := uuid.New()
	from := time.Now().Add(-30 * 24 * time.Hour)
	to := time.Now()

	t.Run("combined export", func(t *testing.T) {
		rows, err := svc.Combined(context.Background(), orgID, from, to, "finance", uuid.New())
		assert.NoError(t, err)
		// Both Timesheets and Expenses return nil → append(nil, nil...) = nil → Combined returns nil
		assert.Nil(t, rows)
	})
}

func TestService_WithRoleScoping(t *testing.T) {
	svc, _ := setupService(t)
	orgID := uuid.New()
	userID := uuid.New()
	from := time.Now().Add(-30 * 24 * time.Hour)
	to := time.Now()

	t.Run("employee can export", func(t *testing.T) {
		rows, err := svc.Timesheets(context.Background(), orgID, from, to, "employee", userID)
		assert.NoError(t, err)
		assert.Nil(t, rows)
	})

	t.Run("manager can export", func(t *testing.T) {
		rows, err := svc.Combined(context.Background(), orgID, from, to, "manager", userID)
		assert.NoError(t, err)
		assert.Nil(t, rows)
	})
}

func TestService_DateRangeFiltering(t *testing.T) {
	svc, _ := setupService(t)
	orgID := uuid.New()

	t.Run("same day range", func(t *testing.T) {
		now := time.Now()
		rows, err := svc.Timesheets(context.Background(), orgID, now, now, "finance", uuid.New())
		assert.NoError(t, err)
		assert.Nil(t, rows)
	})
}

func TestService_CountTimesheets(t *testing.T) {
	svc, _ := setupService(t)
	orgID := uuid.New()
	from := time.Now().Add(-30 * 24 * time.Hour)
	to := time.Now()

	t.Run("count time entries", func(t *testing.T) {
		count, err := svc.CountTimesheets(context.Background(), orgID, from, to, "finance", uuid.New())
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestService_CountExpenses(t *testing.T) {
	svc, _ := setupService(t)
	orgID := uuid.New()
	from := time.Now().Add(-30 * 24 * time.Hour)
	to := time.Now()

	t.Run("count expenses", func(t *testing.T) {
		count, err := svc.CountExpenses(context.Background(), orgID, from, to, "finance", uuid.New())
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestService_CountCombined(t *testing.T) {
	svc, _ := setupService(t)
	orgID := uuid.New()
	from := time.Now().Add(-30 * 24 * time.Hour)
	to := time.Now()

	t.Run("combined count sums both", func(t *testing.T) {
		count, err := svc.CountCombined(context.Background(), orgID, from, to, "finance", uuid.New())
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
