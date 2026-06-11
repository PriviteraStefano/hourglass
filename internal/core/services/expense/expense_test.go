package expense

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
)

func setupService(t *testing.T) (*Service, *testdata.MockExpenseRepo) {
	t.Helper()
	repo := &testdata.MockExpenseRepo{Expenses: make(map[uuid.UUID]*expense.Expense)}
	svc := NewService(repo)
	return svc, repo
}

func seedExpense(repo *testdata.MockExpenseRepo, overrides ...func(*expense.Expense)) *expense.Expense {
	e := testdata.NewExpenseDomain(overrides...)
	repo.Expenses[e.ID] = &e
	return &e
}

// ---------------------------------------------------------------------------
// TestExpenseService_List
// ---------------------------------------------------------------------------

func TestExpenseService_List(t *testing.T) {
	orgID := uuid.New()

	t.Run("returns expenses for org", func(t *testing.T) {
		svc, repo := setupService(t)

		seedExpense(repo, func(e *expense.Expense) {
			e.OrgID = orgID
		})
		seedExpense(repo, func(e *expense.Expense) {
			e.OrgID = orgID
		})
		seedExpense(repo, func(e *expense.Expense) {
			e.OrgID = uuid.New()
		})

		entries, err := svc.List(context.Background(), orgID, ports.ExpenseListFilters{})
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("empty org returns empty", func(t *testing.T) {
		svc, _ := setupService(t)

		entries, err := svc.List(context.Background(), orgID, ports.ExpenseListFilters{})
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Get
// ---------------------------------------------------------------------------

func TestExpenseService_Get(t *testing.T) {
	t.Run("existing expense returns", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo)

		got, err := svc.Get(context.Background(), e.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, e.ID, got.ID)
	})

	t.Run("nonexistent returns error", func(t *testing.T) {
		svc, _ := setupService(t)

		got, err := svc.Get(context.Background(), uuid.New())
		assert.ErrorIs(t, err, expense.ErrExpenseNotFound)
		assert.Nil(t, got)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Create
// ---------------------------------------------------------------------------

func TestExpenseService_Create(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("valid expense creates draft", func(t *testing.T) {
		svc, repo := setupService(t)

		req := &expense.CreateExpenseRequest{
			OrgID:     orgID,
			UserID:    userID,
			ProjectID: projectID,
			Category:  expense.CategoryMileage,
			Amount:    100.0,
			Date:      "2026-05-18",
		}

		e, err := svc.Create(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, e)
		assert.Equal(t, expense.StatusDraft, e.Status)
		assert.NotEqual(t, uuid.Nil, e.ID)
		assert.Equal(t, 100.0, e.Amount)
		assert.Equal(t, expense.CategoryMileage, e.Category)

		// Verify stored
		stored, ok := repo.Expenses[e.ID]
		require.True(t, ok)
		assert.Equal(t, expense.StatusDraft, stored.Status)
	})

	t.Run("period locked returns error", func(t *testing.T) {
		svc, repo := setupService(t)
		repo.PeriodLocked = true

		req := &expense.CreateExpenseRequest{
			OrgID:     orgID,
			UserID:    userID,
			ProjectID: projectID,
			Category:  expense.CategoryMeal,
			Amount:    50.0,
			Date:      "2026-05-18",
		}

		e, err := svc.Create(context.Background(), req)
		assert.ErrorIs(t, err, expense.ErrPeriodLocked)
		assert.Nil(t, e)
	})

	t.Run("invalid category returns error", func(t *testing.T) {
		svc, _ := setupService(t)

		req := &expense.CreateExpenseRequest{
			OrgID:     orgID,
			UserID:    userID,
			ProjectID: projectID,
			Category:  "invalid_category",
			Amount:    50.0,
			Date:      "2026-05-18",
		}

		e, err := svc.Create(context.Background(), req)
		assert.ErrorIs(t, err, expense.ErrInvalidCategory)
		assert.Nil(t, e)
	})

	t.Run("invalid date format", func(t *testing.T) {
		svc, _ := setupService(t)

		req := &expense.CreateExpenseRequest{
			OrgID:     orgID,
			UserID:    userID,
			ProjectID: projectID,
			Category:  expense.CategoryOther,
			Amount:    25.0,
			Date:      "not-a-date",
		}

		e, err := svc.Create(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, e)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Update
// ---------------------------------------------------------------------------

func TestExpenseService_Update(t *testing.T) {
	userID := uuid.New()
	newAmount := 200.0

	t.Run("owner updates draft expense", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
			ex.Amount = 100.0
		})

		req := &expense.UpdateExpenseRequest{
			Amount: &newAmount,
		}

		updated, err := svc.Update(context.Background(), e.ID, userID, req)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, 200.0, updated.Amount)
	})

	t.Run("owner updates submitted expense", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusSubmitted
		})

		req := &expense.UpdateExpenseRequest{
			Amount: &newAmount,
		}

		updated, err := svc.Update(context.Background(), e.ID, userID, req)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, 200.0, updated.Amount)
	})

	t.Run("cannot update approved expense", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusApproved
		})

		req := &expense.UpdateExpenseRequest{
			Amount: &newAmount,
		}

		updated, err := svc.Update(context.Background(), e.ID, userID, req)
		assert.ErrorIs(t, err, expense.ErrEntryNotDraft)
		assert.Nil(t, updated)
	})

	t.Run("non-owner cannot update", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
		})

		req := &expense.UpdateExpenseRequest{
			Amount: &newAmount,
		}

		otherUserID := uuid.New()
		updated, err := svc.Update(context.Background(), e.ID, otherUserID, req)
		assert.ErrorIs(t, err, expense.ErrNotOwner)
		assert.Nil(t, updated)
	})

	t.Run("invalid category on update returns error", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
		})

		badCat := "invalid_category"
		req := &expense.UpdateExpenseRequest{
			Category: &badCat,
		}

		updated, err := svc.Update(context.Background(), e.ID, userID, req)
		assert.ErrorIs(t, err, expense.ErrInvalidCategory)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Delete
// ---------------------------------------------------------------------------

func TestExpenseService_Delete(t *testing.T) {
	userID := uuid.New()

	t.Run("owner deletes draft expense", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
		})

		err := svc.Delete(context.Background(), e.ID, userID)
		require.NoError(t, err)

		_, err = repo.GetByID(context.Background(), e.ID)
		assert.ErrorIs(t, err, expense.ErrExpenseNotFound)
	})

	t.Run("cannot delete approved expense", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusApproved
		})

		err := svc.Delete(context.Background(), e.ID, userID)
		assert.ErrorIs(t, err, expense.ErrEntryNotDraft)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Submit
// ---------------------------------------------------------------------------

func TestExpenseService_Submit(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	t.Run("owner submits draft expense", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
		})

		updated, err := svc.Submit(context.Background(), e.ID, userID)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusSubmitted, updated.Status)
		require.NotNil(t, updated.CurrentApproverRole)
		assert.Equal(t, "manager", *updated.CurrentApproverRole)
		require.NotNil(t, updated.SubmittedAt)
	})

	t.Run("owner submits rejected expense", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusRejected
		})

		updated, err := svc.Submit(context.Background(), e.ID, userID)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusSubmitted, updated.Status)
	})

	t.Run("non-owner cannot submit", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
		})

		updated, err := svc.Submit(context.Background(), e.ID, otherUserID)
		assert.ErrorIs(t, err, expense.ErrNotOwner)
		assert.Nil(t, updated)
	})

	t.Run("cannot submit already submitted", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := svc.Submit(context.Background(), e.ID, userID)
		assert.ErrorIs(t, err, expense.ErrEntryNotDraft)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Approve — two-stage approval
// ---------------------------------------------------------------------------

func TestExpenseService_Approve(t *testing.T) {
	managerID := uuid.New()
	financeID := uuid.New()
	creatorID := uuid.New()

	t.Run("manager approves submitted → pending_finance", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusSubmitted
			ex.CurrentApproverRole = strPtr("manager")
		})

		updated, err := svc.Approve(context.Background(), e.ID, managerID, "manager")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusPendingFinance, updated.Status)
		require.NotNil(t, updated.CurrentApproverRole)
		assert.Equal(t, "finance", *updated.CurrentApproverRole)

		// Verify approval record created
		require.Len(t, repo.Approvals, 1)
		assert.Equal(t, "approve", repo.Approvals[0].Action)
		assert.Equal(t, managerID, repo.Approvals[0].ActorUserID)
	})

	t.Run("finance approves pending_finance → approved", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusPendingFinance
			ex.CurrentApproverRole = strPtr("finance")
		})

		updated, err := svc.Approve(context.Background(), e.ID, financeID, "finance")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusApproved, updated.Status)
		assert.Nil(t, updated.CurrentApproverRole)

		// Verify approval record created
		require.Len(t, repo.Approvals, 1)
		assert.Equal(t, "approve", repo.Approvals[0].Action)
		assert.Equal(t, financeID, repo.Approvals[0].ActorUserID)
	})

	t.Run("finance cannot approve submitted directly", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := svc.Approve(context.Background(), e.ID, financeID, "finance")
		assert.ErrorIs(t, err, expense.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})

	t.Run("manager cannot approve pending_finance", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusPendingFinance
		})

		updated, err := svc.Approve(context.Background(), e.ID, managerID, "manager")
		assert.ErrorIs(t, err, expense.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})

	t.Run("self-approve by manager is forbidden", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = managerID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := svc.Approve(context.Background(), e.ID, managerID, "manager")
		assert.ErrorIs(t, err, expense.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("self-approve by finance is forbidden", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = financeID
			ex.Status = expense.StatusPendingFinance
		})

		updated, err := svc.Approve(context.Background(), e.ID, financeID, "finance")
		assert.ErrorIs(t, err, expense.ErrForbidden)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Reject
// ---------------------------------------------------------------------------

func TestExpenseService_Reject(t *testing.T) {
	managerID := uuid.New()
	financeID := uuid.New()
	employeeID := uuid.New()
	creatorID := uuid.New()

	t.Run("manager rejects submitted expense", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := svc.Reject(context.Background(), e.ID, managerID, "manager", "Incorrect amount")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusRejected, updated.Status)

		// Verify approval record with reason
		require.Len(t, repo.Approvals, 1)
		assert.Equal(t, "reject", repo.Approvals[0].Action)
		assert.Equal(t, managerID, repo.Approvals[0].ActorUserID)
		assert.Equal(t, "Incorrect amount", repo.Approvals[0].Comment)
	})

	t.Run("finance rejects pending_finance expense", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusPendingFinance
		})

		updated, err := svc.Reject(context.Background(), e.ID, financeID, "finance", "Budget exceeded")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusRejected, updated.Status)

		// Verify approval record
		require.Len(t, repo.Approvals, 1)
		assert.Equal(t, "reject", repo.Approvals[0].Action)
		assert.Equal(t, financeID, repo.Approvals[0].ActorUserID)
		assert.Equal(t, "Budget exceeded", repo.Approvals[0].Comment)
	})

	t.Run("employee cannot reject", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := svc.Reject(context.Background(), e.ID, employeeID, "employee", "")
		assert.ErrorIs(t, err, expense.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("reject without reason still works", func(t *testing.T) {
		svc, repo := setupService(t)
		e := seedExpense(repo, func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := svc.Reject(context.Background(), e.ID, managerID, "manager", "")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusRejected, updated.Status)

		require.Len(t, repo.Approvals, 1)
		assert.Empty(t, repo.Approvals[0].Comment)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_ListPending
// ---------------------------------------------------------------------------

func TestExpenseService_ListPending(t *testing.T) {
	orgID := uuid.New()

	t.Run("returns pending expenses for role", func(t *testing.T) {
		svc, repo := setupService(t)

		seedExpense(repo, func(e *expense.Expense) {
			e.OrgID = orgID
			e.Status = expense.StatusSubmitted
		})
		seedExpense(repo, func(e *expense.Expense) {
			e.OrgID = orgID
			e.Status = expense.StatusDraft
		})

		entries, err := svc.ListPending(context.Background(), orgID, "manager", uuid.New().String())
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("empty org returns empty", func(t *testing.T) {
		svc, _ := setupService(t)

		entries, err := svc.ListPending(context.Background(), uuid.New(), "manager", uuid.New().String())
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}
