package coveragesvc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/coverage"
	time_entrydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	unitdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// coverageFixture wires the coverage service against the hermetic mocks: the
// routing.Service is constructed from the same mock repos the coverage
// service receives, so the D-08 gate resolution is deterministic per test
// (seed a WG → ApproverIDs path; leave repos empty → RoleGated path).
type coverageFixture struct {
	svc          *Service
	repo         *testdata.MockCoverageRepo
	activityRepo *testdata.MockActivityRepo
	contractRepo *testdata.MockContractRepo
	unitRepo     *testdata.MockUnitRepo
	entryRepo    *testdata.MockTimeEntryRepo
	wgRepo       *testdata.MockWorkingGroupRepo
}

func setupCoverage(t *testing.T) *coverageFixture {
	t.Helper()
	f := &coverageFixture{
		repo:         &testdata.MockCoverageRepo{},
		activityRepo: &testdata.MockActivityRepo{},
		contractRepo: &testdata.MockContractRepo{Contracts: make(map[uuid.UUID]*contractdomain.ContractResponse)},
		unitRepo:     &testdata.MockUnitRepo{},
		entryRepo:    &testdata.MockTimeEntryRepo{Entries: make(map[uuid.UUID]*time_entrydomain.TimeEntry)},
		wgRepo:       &testdata.MockWorkingGroupRepo{},
	}
	f.svc = NewService(f.repo, f.activityRepo, f.contractRepo, f.unitRepo, f.entryRepo,
		routing.NewService(f.wgRepo, f.activityRepo, f.unitRepo))
	return f
}

// seedEntry adds an approved time entry to the mock repo.
func (f *coverageFixture) seedEntry(orgID uuid.UUID, overrides ...func(*time_entrydomain.TimeEntry)) *time_entrydomain.TimeEntry {
	t := testdata.NewTimeEntry(overrides...)
	t.OrgID = orgID
	t.Status = time_entrydomain.StatusApproved
	f.entryRepo.Entries[t.ID] = &t
	return &t
}

// seedContract adds a contract response to the contract mock.
func (f *coverageFixture) seedContract(orgID uuid.UUID, overrides ...func(*contractdomain.ContractResponse)) *contractdomain.ContractResponse {
	c := &contractdomain.ContractResponse{Contract: testdata.NewContract()}
	c.CreatedByOrgID = orgID
	for _, o := range overrides {
		o(c)
	}
	f.contractRepo.Contracts[c.ID] = c
	return c
}

// seedUnit adds a unit to the unit mock (cross-org check uses OrgID).
func (f *coverageFixture) seedUnit(orgID uuid.UUID, unitID uuid.UUID) *unitdomain.Unit {
	u := testdata.NewUnit()
	u.ID = unitID.String()
	u.OrgID = orgID
	if f.unitRepo.Units == nil {
		f.unitRepo.Units = make(map[string]*unitdomain.Unit)
	}
	f.unitRepo.Units[u.ID] = &u
	return &u
}

// seedWG anchors a working group to the entry's activity with the given
// manager — drives the routing ApproverIDs path.
func (f *coverageFixture) seedWG(orgID, activityID, managerID uuid.UUID) {
	if f.wgRepo.Groups == nil {
		f.wgRepo.Groups = make(map[uuid.UUID]*working_group.WorkingGroup)
	}
	f.wgRepo.Groups[uuid.New()] = &working_group.WorkingGroup{
		ID:           uuid.New(),
		OrgID:        orgID,
		SubprojectID: activityID,
		Name:         "Test WG",
		ManagerID:    managerID,
		IsActive:     true,
	}
}

func f64(v float64) *float64 { return &v }

func strPtr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// TestDefaultSource — the D-04 decision matrix (6 cases)
// ---------------------------------------------------------------------------

func TestDefaultSource(t *testing.T) {
	contractID := uuid.New()
	unitID := uuid.New()
	support := "support"
	project := "project"

	cases := []struct {
		name          string
		chain         *activitydomain.FundingContext
		wantType      string
		wantContract  *uuid.UUID
		wantUnit      *uuid.UUID
		wantFlagged   bool
		wantFlagReason string
	}{
		{
			name:         "project contract with sold hours draws the contract budget",
			chain:        &activitydomain.FundingContext{ContractID: &contractID, ContractType: &project, SoldHours: f64(40)},
			wantType:     coverage.SourceTypeContract,
			wantContract: &contractID,
		},
		{
			name:         "support contract draws the bucket",
			chain:        &activitydomain.FundingContext{ContractID: &contractID, ContractType: &support, SoldHours: f64(8)},
			wantType:     coverage.SourceTypeContract,
			wantContract: &contractID,
		},
		{
			name:         "project contract with sold 0 is a service-request draw (A3)",
			chain:        &activitydomain.FundingContext{ContractID: &contractID, ContractType: &project, SoldHours: f64(0)},
			wantType:     coverage.SourceTypeContract,
			wantContract: &contractID,
		},
		{
			name:         "project contract with NULL sold hours is a service-request draw (A3)",
			chain:        &activitydomain.FundingContext{ContractID: &contractID, ContractType: &project, SoldHours: nil},
			wantType:     coverage.SourceTypeContract,
			wantContract: &contractID,
		},
		{
			name:         "no contract with beneficiary unit draws absorption with the unit",
			chain:        &activitydomain.FundingContext{BeneficiaryUnitID: &unitID},
			wantType:     coverage.SourceTypeAbsorption,
			wantUnit:     &unitID,
		},
		{
			name:           "no contract and no unit flags no-source (D-06)",
			chain:          &activitydomain.FundingContext{},
			wantFlagged:    true,
			wantFlagReason: "no eligible source — needs a unit or contract",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sourceType, cid, uid, flagged, flagReason := DefaultSource(tc.chain)
			assert.Equal(t, tc.wantType, sourceType)
			if tc.wantContract != nil {
				require.NotNil(t, cid, "contract ref must be pinned")
				assert.Equal(t, *tc.wantContract, *cid)
			} else {
				assert.Nil(t, cid)
			}
			if tc.wantUnit != nil {
				require.NotNil(t, uid, "unit ref must be pinned")
				assert.Equal(t, *tc.wantUnit, *uid)
			} else {
				assert.Nil(t, uid)
			}
			assert.Equal(t, tc.wantFlagged, flagged)
			assert.Equal(t, tc.wantFlagReason, flagReason)
		})
	}
}

// ---------------------------------------------------------------------------
// TestService_Propose
// ---------------------------------------------------------------------------

func TestService_Propose(t *testing.T) {
	orgID := uuid.New()

	t.Run("returns the computed proposal and the current allocations", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(orgID, func(e *time_entrydomain.TimeEntry) { e.ActivityID = uuid.New() })
		contractID := uuid.New()
		project := "project"
		f.activityRepo.ResolveFundingContextFn = func(ctx context.Context, activityID uuid.UUID) (*activitydomain.FundingContext, error) {
			return &activitydomain.FundingContext{ContractID: &contractID, ContractType: &project, SoldHours: f64(40)}, nil
		}
		f.repo.Allocations = map[uuid.UUID][]*coverage.CoverageAllocation{
			e.ID: {{ID: uuid.New(), OrgID: orgID, EntryType: coverage.EntryTypeTime, EntryID: e.ID, SourceType: coverage.SourceTypeContract, ContractID: &contractID, Hours: 8}},
		}

		proposal, allocs, err := f.svc.Propose(context.Background(), orgID, e.ID, "manager", uuid.New().String())
		require.NoError(t, err)
		require.NotNil(t, proposal)
		assert.False(t, proposal.Flagged)
		assert.Equal(t, coverage.SourceTypeContract, proposal.SourceType)
		require.NotNil(t, proposal.ContractID)
		assert.Equal(t, contractID, *proposal.ContractID)
		assert.Equal(t, 8.0, proposal.Hours)
		assert.Len(t, allocs, 1)
	})

	t.Run("no contract with beneficiary unit proposes absorption", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(orgID)
		unitID := uuid.New()
		f.activityRepo.ResolveFundingContextFn = func(ctx context.Context, activityID uuid.UUID) (*activitydomain.FundingContext, error) {
			return &activitydomain.FundingContext{}, nil
		}
		f.activityRepo.ResolveBeneficiaryUnitFn = func(ctx context.Context, activityID uuid.UUID) (*uuid.UUID, error) {
			return &unitID, nil
		}

		proposal, _, err := f.svc.Propose(context.Background(), orgID, e.ID, "finance", uuid.New().String())
		require.NoError(t, err)
		assert.Equal(t, coverage.SourceTypeAbsorption, proposal.SourceType)
		require.NotNil(t, proposal.UnitID)
		assert.Equal(t, unitID, *proposal.UnitID)
	})

	t.Run("no contract and no unit flags the proposal", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(orgID)
		f.activityRepo.ResolveFundingContextFn = func(ctx context.Context, activityID uuid.UUID) (*activitydomain.FundingContext, error) {
			return nil, nil
		}
		f.activityRepo.ResolveBeneficiaryUnitFn = func(ctx context.Context, activityID uuid.UUID) (*uuid.UUID, error) {
			return nil, nil
		}

		proposal, _, err := f.svc.Propose(context.Background(), orgID, e.ID, "manager", uuid.New().String())
		require.NoError(t, err)
		assert.True(t, proposal.Flagged)
		assert.Equal(t, "no eligible source — needs a unit or contract", proposal.FlagReason)
	})

	t.Run("cross-org entry is not coverable", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(uuid.New())

		_, _, err := f.svc.Propose(context.Background(), orgID, e.ID, "manager", uuid.New().String())
		require.ErrorIs(t, err, coverage.ErrEntryNotCoverable)
	})

	t.Run("employee is forbidden on the read path", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(orgID)

		_, _, err := f.svc.Propose(context.Background(), orgID, e.ID, "employee", uuid.New().String())
		require.ErrorIs(t, err, coverage.ErrForbidden)
	})
}

// ---------------------------------------------------------------------------
// TestService_ToCoverQueue
// ---------------------------------------------------------------------------

func TestService_ToCoverQueue(t *testing.T) {
	orgID := uuid.New()

	t.Run("attaches proposals and flags no-source rows", func(t *testing.T) {
		f := setupCoverage(t)
		activityID := uuid.New()
		contractID := uuid.New()
		project := "project"
		f.activityRepo.ResolveFundingContextFn = func(ctx context.Context, aid uuid.UUID) (*activitydomain.FundingContext, error) {
			if aid == activityID {
				return &activitydomain.FundingContext{ContractID: &contractID, ContractType: &project, SoldHours: f64(40)}, nil
			}
			return &activitydomain.FundingContext{}, nil
		}
		f.activityRepo.ResolveBeneficiaryUnitFn = func(ctx context.Context, aid uuid.UUID) (*uuid.UUID, error) {
			return nil, nil
		}
		f.repo.ToCoverQueueRows = []coverage.ToCoverQueueRow{
			{EntryID: uuid.New(), ActivityID: activityID, Hours: 8, UncoveredHours: 3},
			{EntryID: uuid.New(), ActivityID: uuid.New(), Hours: 6, UncoveredHours: 6},
		}

		rows, err := f.svc.ToCoverQueue(context.Background(), orgID, "manager", uuid.New().String())
		require.NoError(t, err)
		require.Len(t, rows, 2)
		require.NotNil(t, rows[0].Proposal)
		assert.Equal(t, coverage.SourceTypeContract, rows[0].Proposal.SourceType)
		assert.Equal(t, 3.0, rows[0].Proposal.Hours) // uncovered hours
		require.NotNil(t, rows[1].Proposal)
		assert.True(t, rows[1].Proposal.Flagged)
		assert.Equal(t, "no eligible source — needs a unit or contract", rows[1].Proposal.FlagReason)
		assert.Equal(t, 6.0, rows[1].Proposal.Hours)
	})

	t.Run("employee is forbidden", func(t *testing.T) {
		f := setupCoverage(t)
		_, err := f.svc.ToCoverQueue(context.Background(), orgID, "employee", uuid.New().String())
		require.ErrorIs(t, err, coverage.ErrForbidden)
	})
}

// ---------------------------------------------------------------------------
// TestService_BucketBalance / GetSnapshot / ListHistory — passthrough + gate
// ---------------------------------------------------------------------------

func TestService_BucketBalance(t *testing.T) {
	orgID := uuid.New()
	contractID := uuid.New()

	t.Run("returns negative balances unchanged (D-03)", func(t *testing.T) {
		f := setupCoverage(t)
		f.repo.BucketBalanceResult = -3.5

		balance, err := f.svc.BucketBalance(context.Background(), orgID, contractID, "manager", uuid.New().String())
		require.NoError(t, err)
		assert.Equal(t, -3.5, balance)
	})

	t.Run("employee is forbidden", func(t *testing.T) {
		f := setupCoverage(t)
		_, err := f.svc.BucketBalance(context.Background(), orgID, contractID, "employee", uuid.New().String())
		require.ErrorIs(t, err, coverage.ErrForbidden)
	})
}

func TestService_GetSnapshot(t *testing.T) {
	orgID := uuid.New()
	f := setupCoverage(t)
	closeID := uuid.New()
	pc := &coverage.PeriodClose{ID: closeID, OrgID: orgID, PeriodStart: time.Now(), PeriodEnd: time.Now()}
	f.repo.Snapshots = map[uuid.UUID]*coverage.PeriodClose{closeID: pc}

	got, err := f.svc.GetSnapshot(context.Background(), orgID, closeID, "finance", uuid.New().String())
	require.NoError(t, err)
	assert.Equal(t, closeID, got.ID)

	_, err = f.svc.GetSnapshot(context.Background(), orgID, closeID, "customer", uuid.New().String())
	require.ErrorIs(t, err, coverage.ErrForbidden)
}

func TestService_ListHistory(t *testing.T) {
	orgID := uuid.New()
	f := setupCoverage(t)
	entryID := uuid.New()
	f.repo.Audits = []*audit.AuditLog{
		{OrgID: orgID, EntityID: entryID, Action: coverage.AuditActionAllocationsSet},
	}

	got, err := f.svc.ListHistory(context.Background(), orgID, entryID, "manager", uuid.New().String())
	require.NoError(t, err)
	require.Len(t, got, 1)

	_, err = f.svc.ListHistory(context.Background(), orgID, entryID, "employee", uuid.New().String())
	require.ErrorIs(t, err, coverage.ErrForbidden)
}
