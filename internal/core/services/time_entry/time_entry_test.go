package time_entry

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	unitdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	wgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type serviceFixture struct {
	svc          *Service
	repo         *testdata.MockTimeEntryRepo
	approvalRepo *testdata.MockTimeEntryApprovalRepo
	wgRepo       *testdata.MockWorkingGroupRepo
	activityRepo *testdata.MockActivityRepo
	unitRepo     *testdata.MockUnitRepo
}

func setupService(t *testing.T) *serviceFixture {
	t.Helper()
	f := &serviceFixture{
		repo:         &testdata.MockTimeEntryRepo{Entries: make(map[uuid.UUID]*time_entry.TimeEntry)},
		approvalRepo: &testdata.MockTimeEntryApprovalRepo{},
		wgRepo:       &testdata.MockWorkingGroupRepo{},
		activityRepo: &testdata.MockActivityRepo{},
		unitRepo:     &testdata.MockUnitRepo{},
	}
	f.svc = NewService(f.repo, f.approvalRepo, f.wgRepo, f.activityRepo, f.unitRepo, routing.NewService(f.wgRepo, f.activityRepo, f.unitRepo))
	return f
}

// seedEntry adds a time entry to the mock repo and returns its pointer.
func (f *serviceFixture) seedEntry(overrides ...func(*time_entry.TimeEntry)) *time_entry.TimeEntry {
	e := testdata.NewTimeEntry(overrides...)
	f.repo.Entries[e.ID] = &e
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

// mustParse is a convenience for parsing a time string in test assertions.
func mustParse(value string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", value)
	if err != nil {
		panic(err)
	}
	return t
}

// ---------------------------------------------------------------------------
// TestService_List
// ---------------------------------------------------------------------------

func TestService_List(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()

	t.Run("returns entries for org", func(t *testing.T) {
		f := setupService(t)

		f.seedEntry(func(e *time_entry.TimeEntry) { e.OrgID = orgID; e.Status = time_entry.StatusDraft })
		f.seedEntry(func(e *time_entry.TimeEntry) { e.OrgID = orgID; e.Status = time_entry.StatusSubmitted })
		f.seedEntry(func(e *time_entry.TimeEntry) { e.OrgID = otherOrgID })

		entries, err := f.svc.List(context.Background(), orgID, ports.ListFilters{})
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("empty org returns empty", func(t *testing.T) {
		f := setupService(t)

		entries, err := f.svc.List(context.Background(), orgID, ports.ListFilters{})
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// ---------------------------------------------------------------------------
// TestService_Get
// ---------------------------------------------------------------------------

func TestService_Get(t *testing.T) {
	t.Run("existing entry returns", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry()

		got, err := f.svc.Get(context.Background(), entry.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, entry.ID, got.ID)
	})

	t.Run("nonexistent returns error", func(t *testing.T) {
		f := setupService(t)

		got, err := f.svc.Get(context.Background(), uuid.New())
		assert.ErrorIs(t, err, time_entry.ErrTimeEntryNotFound)
		assert.Nil(t, got)
	})
}

// ---------------------------------------------------------------------------
// TestService_Create
// ---------------------------------------------------------------------------

func TestService_Create(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	activityID := uuid.New()

	t.Run("valid entry creates draft", func(t *testing.T) {
		f := setupService(t)

		req := &time_entry.CreateTimeEntryRequest{
			OrgID:      orgID,
			UserID:     userID,
			ActivityID: activityID,
			Hours:      8.0,
			Date:       "2026-05-18",
		}

		entry, err := f.svc.Create(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, time_entry.StatusDraft, entry.Status)
		assert.NotEqual(t, uuid.Nil, entry.ID)
		assert.Equal(t, 8.0, entry.Hours)
		assert.Equal(t, activityID, entry.ActivityID)

		stored, ok := f.repo.Entries[entry.ID]
		require.True(t, ok)
		assert.Equal(t, time_entry.StatusDraft, stored.Status)
	})

	t.Run("period locked returns error", func(t *testing.T) {
		f := setupService(t)
		f.repo.PeriodLocked = true

		req := &time_entry.CreateTimeEntryRequest{
			OrgID:      orgID,
			UserID:     userID,
			ActivityID: activityID,
			Hours:      8.0,
			Date:       "2026-05-18",
		}

		entry, err := f.svc.Create(context.Background(), req)
		assert.ErrorIs(t, err, time_entry.ErrPeriodLocked)
		assert.Nil(t, entry)
	})

	t.Run("invalid date format", func(t *testing.T) {
		f := setupService(t)

		req := &time_entry.CreateTimeEntryRequest{
			OrgID:      orgID,
			UserID:     userID,
			ActivityID: activityID,
			Hours:      8.0,
			Date:       "not-a-date",
		}

		entry, err := f.svc.Create(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, entry)
	})
}

// ---------------------------------------------------------------------------
// TestService_Submit — R-1/R-2/R-3 routing
// ---------------------------------------------------------------------------

func TestService_Submit_Routing(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	wgManagerID := uuid.New()
	delegateID := uuid.New()

	// activityID anchored to a WG whose manager+delegate are NOT the owner
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
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = ownerID
			e.ActivityID = activityID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), entry.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, time_entry.StatusSubmitted, updated.Status)
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
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = ownerID
			e.ActivityID = activityID
			e.UnitID = unitID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), entry.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, time_entry.StatusSubmitted, updated.Status)
		assert.Equal(t, "manager", *updated.CurrentApproverRole)

		// The manager stage lands on the unit manager: a random user cannot
		// act at the manager stage; the unit manager can.
		_, err = f.svc.Approve(context.Background(), entry.ID, uuid.New(), "manager")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)

		approved, err := f.svc.Approve(context.Background(), entry.ID, unitManagerID, "manager")
		require.NoError(t, err)
		assert.Equal(t, time_entry.StatusPendingFinance, approved.Status)
	})

	t.Run("unit manager fallback walks the unit tree upward (R-2)", func(t *testing.T) {
		f := setupService(t)
		activityID := uuid.New()
		childUnitID := uuid.New()
		parentUnitID := uuid.New()
		unitManagerID := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) { a.ID = activityID })
		// child unit has no manager; the manager sits on the parent unit
		child := &unitdomain.Unit{ID: childUnitID.String(), Name: "Child", ParentUnitID: parentUnitID.String()}
		f.unitRepo.Units = map[string]*unitdomain.Unit{child.ID: child}
		f.seedUnitWithManager(parentUnitID, unitManagerID, "")
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = ownerID
			e.ActivityID = activityID
			e.UnitID = childUnitID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), entry.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, time_entry.StatusSubmitted, updated.Status)

		approved, err := f.svc.Approve(context.Background(), entry.ID, unitManagerID, "manager")
		require.NoError(t, err)
		assert.Equal(t, time_entry.StatusPendingFinance, approved.Status)
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
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = ownerID
			e.ActivityID = activityID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), entry.ID, ownerID)
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
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = ownerID
			e.ActivityID = activityID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), entry.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, time_entry.StatusPendingFinance, updated.Status)
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
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = ownerID
			e.ActivityID = activityID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), entry.ID, ownerID)
		require.NoError(t, err)
		assert.Equal(t, time_entry.StatusPendingFinance, updated.Status)
		assert.Equal(t, "finance", *updated.CurrentApproverRole)
	})

	t.Run("D-11 does NOT fire when owner is merely a WG member (R-3)", func(t *testing.T) {
		f := setupService(t)
		activityID := uuid.New()
		cid := uuid.New()
		memberID := ownerID // the owner is only a member, not manager/delegate
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = wgManagerID
			wg.DelegateIDs = []string{delegateID.String()}
		})
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = memberID
			e.ActivityID = activityID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), entry.ID, memberID)
		require.NoError(t, err)
		assert.Equal(t, time_entry.StatusSubmitted, updated.Status)
		assert.Equal(t, "manager", *updated.CurrentApproverRole)
	})

	t.Run("non-owner cannot submit", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = ownerID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Submit(context.Background(), entry.ID, uuid.New())
		assert.ErrorIs(t, err, time_entry.ErrNotOwner)
		assert.Nil(t, updated)
	})

	t.Run("cannot submit already submitted", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = ownerID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := f.svc.Submit(context.Background(), entry.ID, ownerID)
		assert.ErrorIs(t, err, time_entry.ErrEntryNotDraft)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_Approve — two-stage approval with approver-set verification
// ---------------------------------------------------------------------------

func TestService_Approve(t *testing.T) {
	orgID := uuid.New()
	financeID := uuid.New()
	creatorID := uuid.New()
	wgManagerID := uuid.New()

	// seedSubmittedWG creates an entry on an activity with an anchored WG whose
	// manager is wgManagerID, already in submitted state.
	seedSubmittedWG := func(f *serviceFixture) *time_entry.TimeEntry {
		activityID := uuid.New()
		cid := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = wgManagerID
		})
		return f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = creatorID
			e.ActivityID = activityID
			e.Status = time_entry.StatusSubmitted
			e.CurrentApproverRole = strPtr("manager")
		})
	}

	t.Run("WG manager approves submitted → pending_finance", func(t *testing.T) {
		f := setupService(t)
		entry := seedSubmittedWG(f)

		updated, err := f.svc.Approve(context.Background(), entry.ID, wgManagerID, "manager")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, time_entry.StatusPendingFinance, updated.Status)
		require.NotNil(t, updated.CurrentApproverRole)
		assert.Equal(t, "finance", *updated.CurrentApproverRole)

		require.Len(t, f.approvalRepo.Approvals, 1)
		assert.Equal(t, "approve", f.approvalRepo.Approvals[0].Action)
		assert.Equal(t, wgManagerID, f.approvalRepo.Approvals[0].ActorUserID)
	})

	t.Run("non-approver cannot act at the manager stage", func(t *testing.T) {
		f := setupService(t)
		entry := seedSubmittedWG(f)

		updated, err := f.svc.Approve(context.Background(), entry.ID, uuid.New(), "manager")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("finance approves pending_finance → approved", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = creatorID
			e.Status = time_entry.StatusPendingFinance
			e.CurrentApproverRole = strPtr("finance")
		})

		updated, err := f.svc.Approve(context.Background(), entry.ID, financeID, "finance")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, time_entry.StatusApproved, updated.Status)
		assert.Nil(t, updated.CurrentApproverRole)

		require.Len(t, f.approvalRepo.Approvals, 1)
		assert.Equal(t, financeID, f.approvalRepo.Approvals[0].ActorUserID)
	})

	t.Run("finance cannot approve submitted directly", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = creatorID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := f.svc.Approve(context.Background(), entry.ID, financeID, "finance")
		assert.ErrorIs(t, err, time_entry.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})

	t.Run("manager cannot approve pending_finance", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = creatorID
			e.Status = time_entry.StatusPendingFinance
		})

		updated, err := f.svc.Approve(context.Background(), entry.ID, wgManagerID, "manager")
		assert.ErrorIs(t, err, time_entry.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})

	t.Run("self-approve by owner is structurally impossible", func(t *testing.T) {
		f := setupService(t)
		// Owner IS the WG manager on a seeded-submitted entry (simulating an
		// entry that reached submitted state without the D-11 skip, e.g. a
		// pre-rewrite entry). The owner still cannot approve their own entry.
		activityID := uuid.New()
		cid := uuid.New()
		f.seedActivity(orgID, func(a *activitydomain.ActivityResponse) {
			a.ID = activityID
			a.ContractID = &cid
		})
		f.seedWG(orgID, activityID, func(wg *wgdomain.WorkingGroup) {
			wg.ManagerID = creatorID
		})
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.UserID = creatorID
			e.ActivityID = activityID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := f.svc.Approve(context.Background(), entry.ID, creatorID, "manager")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("self-approve by finance is forbidden", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = financeID
			e.Status = time_entry.StatusPendingFinance
		})

		updated, err := f.svc.Approve(context.Background(), entry.ID, financeID, "finance")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("cannot approve draft", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = creatorID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Approve(context.Background(), entry.ID, wgManagerID, "manager")
		assert.ErrorIs(t, err, time_entry.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})

	t.Run("cannot approve already approved", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = creatorID
			e.Status = time_entry.StatusApproved
		})

		updated, err := f.svc.Approve(context.Background(), entry.ID, wgManagerID, "manager")
		assert.ErrorIs(t, err, time_entry.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_Reject
// ---------------------------------------------------------------------------

func TestService_Reject(t *testing.T) {
	managerID := uuid.New()
	financeID := uuid.New()
	creatorID := uuid.New()

	t.Run("manager rejects submitted entry", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = creatorID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := f.svc.Reject(context.Background(), entry.ID, managerID, "manager", "Incorrect hours")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, time_entry.StatusRejected, updated.Status)

		require.Len(t, f.approvalRepo.Approvals, 1)
		assert.Equal(t, "reject", f.approvalRepo.Approvals[0].Action)
		assert.Equal(t, managerID, f.approvalRepo.Approvals[0].ActorUserID)
		assert.Equal(t, "Incorrect hours", f.approvalRepo.Approvals[0].Comment)
	})

	t.Run("finance rejects pending_finance entry", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = creatorID
			e.Status = time_entry.StatusPendingFinance
		})

		updated, err := f.svc.Reject(context.Background(), entry.ID, financeID, "finance", "Budget exceeded")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, time_entry.StatusRejected, updated.Status)

		require.Len(t, f.approvalRepo.Approvals, 1)
		assert.Equal(t, "reject", f.approvalRepo.Approvals[0].Action)
		assert.Equal(t, financeID, f.approvalRepo.Approvals[0].ActorUserID)
	})

	t.Run("employee cannot reject", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = creatorID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := f.svc.Reject(context.Background(), entry.ID, uuid.New(), "employee", "")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("cannot reject draft", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = creatorID
			e.Status = time_entry.StatusDraft
		})

		updated, err := f.svc.Reject(context.Background(), entry.ID, managerID, "manager", "")
		assert.ErrorIs(t, err, time_entry.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_Update
// ---------------------------------------------------------------------------

func TestService_Update(t *testing.T) {
	userID := uuid.New()
	newHours := 6.5
	newDesc := "Updated description"

	t.Run("owner updates draft entry", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
			e.Hours = 8.0
			e.Description = "Original"
		})

		req := &time_entry.UpdateTimeEntryRequest{
			Hours:       &newHours,
			Description: &newDesc,
		}

		updated, err := f.svc.Update(context.Background(), entry.ID, userID, req)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, 6.5, updated.Hours)
		assert.Equal(t, "Updated description", updated.Description)
	})

	t.Run("owner updates activity_id", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
		})
		newActivityID := uuid.New()

		req := &time_entry.UpdateTimeEntryRequest{ActivityID: &newActivityID}

		updated, err := f.svc.Update(context.Background(), entry.ID, userID, req)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, newActivityID, updated.ActivityID)
	})

	t.Run("cannot update approved entry", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusApproved
		})

		req := &time_entry.UpdateTimeEntryRequest{Description: &newDesc}

		updated, err := f.svc.Update(context.Background(), entry.ID, userID, req)
		assert.ErrorIs(t, err, time_entry.ErrEntryNotDraft)
		assert.Nil(t, updated)
	})

	t.Run("non-owner cannot update", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
		})

		req := &time_entry.UpdateTimeEntryRequest{Description: &newDesc}

		updated, err := f.svc.Update(context.Background(), entry.ID, uuid.New(), req)
		assert.ErrorIs(t, err, time_entry.ErrNotOwner)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_ListPending — R-4 visibility pass-through
// ---------------------------------------------------------------------------

func TestService_ListPending(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	t.Run("passes through to repository (repo gates on unit subtree)", func(t *testing.T) {
		f := setupService(t)

		f.seedEntry(func(e *time_entry.TimeEntry) { e.OrgID = orgID; e.Status = time_entry.StatusSubmitted })
		f.seedEntry(func(e *time_entry.TimeEntry) { e.OrgID = orgID; e.Status = time_entry.StatusDraft })

		entries, err := f.svc.ListPending(context.Background(), orgID, "wg_manager", userID.String())
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})
}

// ---------------------------------------------------------------------------
// TestService_Delete
// ---------------------------------------------------------------------------

func TestService_Delete(t *testing.T) {
	userID := uuid.New()

	t.Run("owner deletes draft entry", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
		})

		err := f.svc.Delete(context.Background(), entry.ID, userID)
		require.NoError(t, err)

		_, err = f.repo.GetByID(context.Background(), entry.ID)
		assert.ErrorIs(t, err, time_entry.ErrTimeEntryNotFound)
	})

	t.Run("cannot delete submitted entry", func(t *testing.T) {
		f := setupService(t)
		entry := f.seedEntry(func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		err := f.svc.Delete(context.Background(), entry.ID, userID)
		assert.ErrorIs(t, err, time_entry.ErrEntryNotDraft)
	})
}
