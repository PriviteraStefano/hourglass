package expense

import (
	"context"
	"testing"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	unitdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	wgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type serviceFixture struct {
	svc          *Service
	repo         *testdata.MockExpenseRepo
	wgRepo       *testdata.MockWorkingGroupRepo
	activityRepo *testdata.MockActivityRepo
	unitRepo     *testdata.MockUnitRepo
}

func setupService(t *testing.T) *serviceFixture {
	t.Helper()
	f := &serviceFixture{
		repo:         &testdata.MockExpenseRepo{Expenses: make(map[uuid.UUID]*expense.Expense)},
		wgRepo:       &testdata.MockWorkingGroupRepo{},
		activityRepo: &testdata.MockActivityRepo{},
		unitRepo:     &testdata.MockUnitRepo{},
	}
	f.svc = NewService(f.repo, f.wgRepo, f.activityRepo, f.unitRepo)
	return f
}

func (f *serviceFixture) seedExpense(overrides ...func(*expense.Expense)) *expense.Expense {
	e := testdata.NewExpenseDomain(overrides...)
	f.repo.Expenses[e.ID] = &e
	return &e
}

// seedWG adds a working group anchored to an activity to the wg mock.
func (f *serviceFixture) seedWG(orgID, activityID uuid.UUID, overrides ...func(*wgdomain.WorkingGroup)) *wgdomain.WorkingGroup {
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
func (f *serviceFixture) seedActivity(orgID uuid.UUID, overrides ...func(*activitydomain.ActivityResponse)) *activitydomain.ActivityResponse {
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
func (f *serviceFixture) seedUnitWithManager(unitID uuid.UUID, managerID uuid.UUID, parentUnitID string) *unitdomain.Unit {
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
// TestExpenseService_List
// ---------------------------------------------------------------------------

func TestExpenseService_List(t *testing.T) {
	orgID := uuid.New()

	t.Run("returns expenses for org", func(t *testing.T) {
		f := setupService(t)

		f.seedExpense(func(e *expense.Expense) { e.OrgID = orgID })
		f.seedExpense(func(e *expense.Expense) { e.OrgID = orgID })
		f.seedExpense(func(e *expense.Expense) { e.OrgID = uuid.New() })

		entries, err := f.svc.List(context.Background(), orgID, ports.ExpenseListFilters{})
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("empty org returns empty", func(t *testing.T) {
		f := setupService(t)

		entries, err := f.svc.List(context.Background(), orgID, ports.ExpenseListFilters{})
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Get
// ---------------------------------------------------------------------------

func TestExpenseService_Get(t *testing.T) {
	t.Run("existing expense returns", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense()

		got, err := f.svc.Get(context.Background(), e.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, e.ID, got.ID)
	})

	t.Run("nonexistent returns error", func(t *testing.T) {
		f := setupService(t)

		got, err := f.svc.Get(context.Background(), uuid.New())
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
	activityID := uuid.New()

	t.Run("valid expense creates draft", func(t *testing.T) {
		f := setupService(t)

		req := &expense.CreateExpenseRequest{
			OrgID:      orgID,
			UserID:     userID,
			ActivityID: activityID,
			Category:   expense.CategoryMileage,
			Amount:     100.0,
			Date:       "2026-05-18",
		}

		e, err := f.svc.Create(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, e)
		assert.Equal(t, expense.StatusDraft, e.Status)
		assert.NotEqual(t, uuid.Nil, e.ID)
		assert.Equal(t, 100.0, e.Amount)
		assert.Equal(t, expense.CategoryMileage, e.Category)
		assert.Equal(t, activityID, e.ActivityID)

		stored, ok := f.repo.Expenses[e.ID]
		require.True(t, ok)
		assert.Equal(t, expense.StatusDraft, stored.Status)
	})

	t.Run("period locked returns error", func(t *testing.T) {
		f := setupService(t)
		f.repo.PeriodLocked = true

		req := &expense.CreateExpenseRequest{
			OrgID:      orgID,
			UserID:     userID,
			ActivityID: activityID,
			Category:   expense.CategoryMeal,
			Amount:     50.0,
			Date:       "2026-05-18",
		}

		e, err := f.svc.Create(context.Background(), req)
		assert.ErrorIs(t, err, expense.ErrPeriodLocked)
		assert.Nil(t, e)
	})

	t.Run("invalid category returns error", func(t *testing.T) {
		f := setupService(t)

		req := &expense.CreateExpenseRequest{
			OrgID:      orgID,
			UserID:     userID,
			ActivityID: activityID,
			Category:   "invalid_category",
			Amount:     50.0,
			Date:       "2026-05-18",
		}

		e, err := f.svc.Create(context.Background(), req)
		assert.ErrorIs(t, err, expense.ErrInvalidCategory)
		assert.Nil(t, e)
	})

	t.Run("invalid date format", func(t *testing.T) {
		f := setupService(t)

		req := &expense.CreateExpenseRequest{
			OrgID:      orgID,
			UserID:     userID,
			ActivityID: activityID,
			Category:   expense.CategoryOther,
			Amount:     25.0,
			Date:       "not-a-date",
		}

		e, err := f.svc.Create(context.Background(), req)
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
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
			ex.Amount = 100.0
		})

		req := &expense.UpdateExpenseRequest{Amount: &newAmount}

		updated, err := f.svc.Update(context.Background(), e.ID, userID, req)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, 200.0, updated.Amount)
	})

	t.Run("owner updates activity_id", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
		})
		newActivityID := uuid.New()

		req := &expense.UpdateExpenseRequest{ActivityID: &newActivityID}

		updated, err := f.svc.Update(context.Background(), e.ID, userID, req)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, newActivityID, updated.ActivityID)
	})

	t.Run("cannot update approved expense", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusApproved
		})

		req := &expense.UpdateExpenseRequest{Amount: &newAmount}

		updated, err := f.svc.Update(context.Background(), e.ID, userID, req)
		assert.ErrorIs(t, err, expense.ErrEntryNotDraft)
		assert.Nil(t, updated)
	})

	t.Run("non-owner cannot update", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
		})

		req := &expense.UpdateExpenseRequest{Amount: &newAmount}

		updated, err := f.svc.Update(context.Background(), e.ID, uuid.New(), req)
		assert.ErrorIs(t, err, expense.ErrNotOwner)
		assert.Nil(t, updated)
	})

	t.Run("invalid category on update returns error", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
		})

		badCat := "invalid_category"
		req := &expense.UpdateExpenseRequest{Category: &badCat}

		updated, err := f.svc.Update(context.Background(), e.ID, userID, req)
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
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusDraft
		})

		err := f.svc.Delete(context.Background(), e.ID, userID)
		require.NoError(t, err)

		_, err = f.repo.GetByID(context.Background(), e.ID)
		assert.ErrorIs(t, err, expense.ErrExpenseNotFound)
	})

	t.Run("cannot delete approved expense", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = userID
			ex.Status = expense.StatusApproved
		})

		err := f.svc.Delete(context.Background(), e.ID, userID)
		assert.ErrorIs(t, err, expense.ErrEntryNotDraft)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Submit — routing identical to time entries (ADR-P-001 Q1)
// ---------------------------------------------------------------------------

func TestExpenseService_Submit_Routing(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	wgManagerID := uuid.New()
	delegateID := uuid.New()

	t.Run("commercial activity with anchored WG routes to WG manager stage (R-1)", func(t *testing.T) {
		f := setupService(t)
		activityID := uuid.New()
		cid := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = wgManagerID
			wg.DelegateIDs = []string{delegateID.String()}
		})
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.OrgID = orgID
			ex.UserID = ownerID
			ex.ActivityID = activityID
			ex.Status = expense.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), e.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, expense.StatusSubmitted, updated.Status)
		require.NotNil(t, updated.CurrentApproverRole)
		assert.Equal(t, "manager", *updated.CurrentApproverRole)
	})

	t.Run("personal activity without WG routes to unit manager stage (R-2)", func(t *testing.T) {
		f := setupService(t)
		activityID := uuid.New() // no contract, no WG → personal (D-8)
		unitID := uuid.New()
		unitManagerID := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) { a.ID = activityID })
		f.seedUnitWithManager(unitID, unitManagerID, "")
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.OrgID = orgID
			ex.UserID = ownerID
			ex.ActivityID = activityID
			ex.UnitID = unitID
			ex.Status = expense.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), e.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, expense.StatusSubmitted, updated.Status)
		assert.Equal(t, "manager", *updated.CurrentApproverRole)

		// A random user cannot act at the manager stage; the unit manager can.
		_, err = f.svc.Approve(context.Background(), e.ID, uuid.New(), "manager")
		assert.ErrorIs(t, err, expense.ErrForbidden)

		approved, err := f.svc.Approve(context.Background(), e.ID, unitManagerID, "manager")
		require.NoError(t, err)
		assert.Equal(t, expense.StatusPendingFinance, approved.Status)
	})

	t.Run("commercial activity without anchored WG rejected (R-2 enforcement)", func(t *testing.T) {
		f := setupService(t)
		activityID := uuid.New()
		cid := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		// no WG seeded for this activity
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.OrgID = orgID
			ex.UserID = ownerID
			ex.ActivityID = activityID
			ex.Status = expense.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), e.ID, ownerID)
		assert.ErrorIs(t, err, activitydomain.ErrActivityNotLoggable)
		assert.Nil(t, updated)
	})

	t.Run("D-11 skip fires when owner is the WG manager (R-3)", func(t *testing.T) {
		f := setupService(t)
		activityID := uuid.New()
		cid := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = ownerID // owner is the WG manager
			wg.DelegateIDs = []string{delegateID.String()}
		})
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.OrgID = orgID
			ex.UserID = ownerID
			ex.ActivityID = activityID
			ex.Status = expense.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), e.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, expense.StatusPendingFinance, updated.Status)
		require.NotNil(t, updated.CurrentApproverRole)
		assert.Equal(t, "finance", *updated.CurrentApproverRole)
	})

	t.Run("D-11 skip fires when owner is a WG delegate (R-3)", func(t *testing.T) {
		f := setupService(t)
		activityID := uuid.New()
		cid := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = wgManagerID
			wg.DelegateIDs = []string{delegateID.String(), ownerID.String()} // owner is a delegate
		})
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.OrgID = orgID
			ex.UserID = ownerID
			ex.ActivityID = activityID
			ex.Status = expense.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), e.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, expense.StatusPendingFinance, updated.Status)
		assert.Equal(t, "finance", *updated.CurrentApproverRole)
	})

	t.Run("D-11 does NOT fire when owner is merely a WG member (R-3)", func(t *testing.T) {
		f := setupService(t)
		activityID := uuid.New()
		cid := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = wgManagerID
			wg.DelegateIDs = []string{delegateID.String()}
		})
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.OrgID = orgID
			ex.UserID = ownerID
			ex.ActivityID = activityID
			ex.Status = expense.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), e.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, expense.StatusSubmitted, updated.Status)
		assert.Equal(t, "manager", *updated.CurrentApproverRole)
	})

	t.Run("non-owner cannot submit", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = ownerID
			ex.Status = expense.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), e.ID, uuid.New())
		assert.ErrorIs(t, err, expense.ErrNotOwner)
		assert.Nil(t, updated)
	})

	t.Run("cannot submit already submitted", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = ownerID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := f.svc.Submit(context.Background(), e.ID, ownerID)
		assert.ErrorIs(t, err, expense.ErrEntryNotDraft)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_Approve — two-stage approval with approver-set verification
// ---------------------------------------------------------------------------

func TestExpenseService_Approve(t *testing.T) {
	orgID := uuid.New()
	financeID := uuid.New()
	creatorID := uuid.New()
	wgManagerID := uuid.New()

	// seedSubmittedWG creates an expense on an activity with an anchored WG
	// whose manager is wgManagerID, already in submitted state.
	seedSubmittedWG := func(f *serviceFixture) *expense.Expense {
		activityID := uuid.New()
		cid := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = wgManagerID
		})
		return f.seedExpense(func(ex *expense.Expense) {
			ex.OrgID = orgID
			ex.UserID = creatorID
			ex.ActivityID = activityID
			ex.Status = expense.StatusSubmitted
			ex.CurrentApproverRole = strPtr("manager")
		})
	}

	t.Run("WG manager approves submitted → pending_finance", func(t *testing.T) {
		f := setupService(t)
		e := seedSubmittedWG(f)

		updated, err := f.svc.Approve(context.Background(), e.ID, wgManagerID, "manager")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusPendingFinance, updated.Status)
		require.NotNil(t, updated.CurrentApproverRole)
		assert.Equal(t, "finance", *updated.CurrentApproverRole)

		require.Len(t, f.repo.Approvals, 1)
		assert.Equal(t, "approve", f.repo.Approvals[0].Action)
		assert.Equal(t, wgManagerID, f.repo.Approvals[0].ActorUserID)
	})

	t.Run("non-approver cannot act at the manager stage", func(t *testing.T) {
		f := setupService(t)
		e := seedSubmittedWG(f)

		updated, err := f.svc.Approve(context.Background(), e.ID, uuid.New(), "manager")
		assert.ErrorIs(t, err, expense.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("finance approves pending_finance → approved", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.OrgID = orgID
			ex.UserID = creatorID
			ex.Status = expense.StatusPendingFinance
			ex.CurrentApproverRole = strPtr("finance")
		})

		updated, err := f.svc.Approve(context.Background(), e.ID, financeID, "finance")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusApproved, updated.Status)
		assert.Nil(t, updated.CurrentApproverRole)

		require.Len(t, f.repo.Approvals, 1)
		assert.Equal(t, financeID, f.repo.Approvals[0].ActorUserID)
	})

	t.Run("finance cannot approve submitted directly", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := f.svc.Approve(context.Background(), e.ID, financeID, "finance")
		assert.ErrorIs(t, err, expense.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})

	t.Run("manager cannot approve pending_finance", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusPendingFinance
		})

		updated, err := f.svc.Approve(context.Background(), e.ID, wgManagerID, "manager")
		assert.ErrorIs(t, err, expense.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})

	t.Run("self-approve by owner is structurally impossible", func(t *testing.T) {
		f := setupService(t)
		// Owner IS the WG manager on a seeded-submitted expense (no D-11 was
		// applied at submit). The owner still cannot approve their own entry.
		activityID := uuid.New()
		cid := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = creatorID
		})
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.OrgID = orgID
			ex.UserID = creatorID
			ex.ActivityID = activityID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := f.svc.Approve(context.Background(), e.ID, creatorID, "manager")
		assert.ErrorIs(t, err, expense.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("self-approve by finance is forbidden", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = financeID
			ex.Status = expense.StatusPendingFinance
		})

		updated, err := f.svc.Approve(context.Background(), e.ID, financeID, "finance")
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
	creatorID := uuid.New()

	t.Run("manager rejects submitted expense", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := f.svc.Reject(context.Background(), e.ID, managerID, "manager", "Incorrect amount")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusRejected, updated.Status)

		require.Len(t, f.repo.Approvals, 1)
		assert.Equal(t, "reject", f.repo.Approvals[0].Action)
		assert.Equal(t, managerID, f.repo.Approvals[0].ActorUserID)
		assert.Equal(t, "Incorrect amount", f.repo.Approvals[0].Comment)
	})

	t.Run("finance rejects pending_finance expense", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusPendingFinance
		})

		updated, err := f.svc.Reject(context.Background(), e.ID, financeID, "finance", "Budget exceeded")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusRejected, updated.Status)

		require.Len(t, f.repo.Approvals, 1)
		assert.Equal(t, "reject", f.repo.Approvals[0].Action)
		assert.Equal(t, financeID, f.repo.Approvals[0].ActorUserID)
		assert.Equal(t, "Budget exceeded", f.repo.Approvals[0].Comment)
	})

	t.Run("employee cannot reject", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := f.svc.Reject(context.Background(), e.ID, uuid.New(), "employee", "")
		assert.ErrorIs(t, err, expense.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("reject without reason still works", func(t *testing.T) {
		f := setupService(t)
		e := f.seedExpense(func(ex *expense.Expense) {
			ex.UserID = creatorID
			ex.Status = expense.StatusSubmitted
		})

		updated, err := f.svc.Reject(context.Background(), e.ID, managerID, "manager", "")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, expense.StatusRejected, updated.Status)

		require.Len(t, f.repo.Approvals, 1)
		assert.Empty(t, f.repo.Approvals[0].Comment)
	})
}

// ---------------------------------------------------------------------------
// TestExpenseService_ListPending — R-4 visibility pass-through
// ---------------------------------------------------------------------------

func TestExpenseService_ListPending(t *testing.T) {
	orgID := uuid.New()

	t.Run("passes through to repository (repo gates on unit subtree)", func(t *testing.T) {
		f := setupService(t)

		f.seedExpense(func(e *expense.Expense) { e.OrgID = orgID; e.Status = expense.StatusSubmitted })
		f.seedExpense(func(e *expense.Expense) { e.OrgID = orgID; e.Status = expense.StatusDraft })

		entries, err := f.svc.ListPending(context.Background(), orgID, "manager", uuid.New().String())
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("empty org returns empty", func(t *testing.T) {
		f := setupService(t)

		entries, err := f.svc.ListPending(context.Background(), uuid.New(), "manager", uuid.New().String())
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}
