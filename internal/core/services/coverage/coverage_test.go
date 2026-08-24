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

// ---------------------------------------------------------------------------
// TestService_GetOwnCoverage — the employee self-read path (Phase 16, Task 1)
// ---------------------------------------------------------------------------

func TestService_GetOwnCoverage(t *testing.T) {
	orgID := uuid.New()
	owner := uuid.New()
	otherUser := uuid.New()

	t.Run("employee reads their own entry's coverage", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(orgID, func(e *time_entrydomain.TimeEntry) {
			e.UserID = owner
			e.ActivityID = uuid.New()
			e.Hours = 8
		})
		contractID := uuid.New()
		project := "project"
		f.activityRepo.ResolveFundingContextFn = func(ctx context.Context, activityID uuid.UUID) (*activitydomain.FundingContext, error) {
			return &activitydomain.FundingContext{ContractID: &contractID, ContractType: &project, SoldHours: f64(40)}, nil
		}
		f.repo.Allocations = map[uuid.UUID][]*coverage.CoverageAllocation{
			e.ID: {{ID: uuid.New(), OrgID: orgID, EntryType: coverage.EntryTypeTime, EntryID: e.ID, SourceType: coverage.SourceTypeContract, ContractID: &contractID, Hours: 8}},
		}

		proposal, allocs, err := f.svc.GetOwnCoverage(context.Background(), orgID, e.ID, owner.String())
		require.NoError(t, err)
		require.NotNil(t, proposal)
		assert.Equal(t, coverage.SourceTypeContract, proposal.SourceType)
		assert.Equal(t, 8.0, proposal.Hours)
		assert.Len(t, allocs, 1)
	})

	t.Run("employee is forbidden from reading another user's entry", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(orgID, func(e *time_entrydomain.TimeEntry) {
			e.UserID = owner
			e.ActivityID = uuid.New()
		})

		_, _, err := f.svc.GetOwnCoverage(context.Background(), orgID, e.ID, otherUser.String())
		require.ErrorIs(t, err, coverage.ErrForbidden)
	})

	t.Run("cross-org entry is not coverable", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(uuid.New(), func(e *time_entrydomain.TimeEntry) {
			e.UserID = owner
		})

		_, _, err := f.svc.GetOwnCoverage(context.Background(), orgID, e.ID, owner.String())
		require.ErrorIs(t, err, coverage.ErrEntryNotCoverable)
	})

	t.Run("manager|finance read behavior is unchanged (Propose still gated)", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(orgID, func(e *time_entrydomain.TimeEntry) {
			e.UserID = owner
			e.ActivityID = uuid.New()
		})

		// Propose still requires manager|finance and is unaffected.
		_, _, err := f.svc.Propose(context.Background(), orgID, e.ID, "employee", uuid.New().String())
		require.ErrorIs(t, err, coverage.ErrForbidden)
		_, _, err = f.svc.Propose(context.Background(), orgID, e.ID, "manager", owner.String())
		require.NoError(t, err)
	})
}

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

// ---------------------------------------------------------------------------
// TestService_ReplaceAllocations — Σ + D-K + gate + refs + audit
// ---------------------------------------------------------------------------

// contractAllocation is a valid single-row request for an 8h entry.
func contractAllocation(entryID, contractID uuid.UUID) *coverage.CoverageAllocation {
	return &coverage.CoverageAllocation{
		EntryType:  coverage.EntryTypeTime,
		EntryID:    entryID,
		SourceType: coverage.SourceTypeContract,
		ContractID: &contractID,
		Hours:      8,
	}
}

func TestService_ReplaceAllocations(t *testing.T) {
	orgID := uuid.New()
	owner := uuid.New()
	approver := uuid.New()
	manager := uuid.New()
	finance := uuid.New()
	contractID := uuid.New()

	// The happy-path fixture: an approved 8h entry whose WG manager is
	// `approver`, with a same-org contract the single allocation references.
	setupHappy := func(t *testing.T) (*coverageFixture, *time_entrydomain.TimeEntry) {
		f := setupCoverage(t)
		e := f.seedEntry(orgID, func(e *time_entrydomain.TimeEntry) {
			e.UserID = owner
			e.ActivityID = uuid.New()
			e.Hours = 8
		})
		f.seedWG(orgID, e.ActivityID, approver)
		f.seedContract(orgID, func(c *contractdomain.ContractResponse) { c.ID = contractID })
		return f, e
	}

	t.Run("approver in ApproverIDs replaces the set and audits allocations-set", func(t *testing.T) {
		f, e := setupHappy(t)
		req := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID)}

		stored, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, approver.String(), "employee")
		require.NoError(t, err)
		require.Len(t, stored, 1)
		require.Len(t, f.repo.Audits, 1)
		audit := f.repo.Audits[0]
		assert.Equal(t, coverage.AuditActionAllocationsSet, audit.Action)
		assert.Equal(t, coverage.AuditEntityCoverageAllocation, audit.EntityType)
		assert.Equal(t, e.ID, audit.EntityID)
		require.NotNil(t, audit.Payload)
		allocs, ok := audit.Payload["allocations"].([]map[string]any)
		require.True(t, ok, "payload must carry the full allocation set")
		require.Len(t, allocs, 1)
		assert.Equal(t, coverage.SourceTypeContract, allocs[0]["source_type"])
		assert.Equal(t, 8.0, allocs[0]["hours"])
	})

	t.Run("owner is structurally forbidden even as approver", func(t *testing.T) {
		f, e := setupHappy(t)
		// WG manager = owner would make routing return the owner in
		// ApproverIDs — the self-barrier fires before resolution.
		req := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID)}

		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, owner.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrForbidden)
	})

	t.Run("non-approver manager is rejected", func(t *testing.T) {
		f, e := setupHappy(t)
		req := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID)}

		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, manager.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrForbidden)
	})

	t.Run("finance outside the approver set is rejected", func(t *testing.T) {
		f, e := setupHappy(t)
		req := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID)}

		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, finance.String(), "finance")
		require.ErrorIs(t, err, coverage.ErrForbidden)
	})

	t.Run("RoleGated terminal state: employee claim rejected, manager accepted", func(t *testing.T) {
		// No WG and no unit manager → routing returns RoleGated=true.
		f := setupCoverage(t)
		e := f.seedEntry(orgID, func(e *time_entrydomain.TimeEntry) {
			e.UserID = owner
			e.Hours = 8
		})
		f.seedContract(orgID, func(c *contractdomain.ContractResponse) { c.ID = contractID })
		f.activityRepo.ResolveCommercialContextFn = func(ctx context.Context, activityID uuid.UUID) (*activitydomain.CommercialContext, error) {
			return &activitydomain.CommercialContext{}, nil // personal tree — no ErrActivityNotLoggable
		}
		req := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID)}

		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, manager.String(), "employee")
		require.ErrorIs(t, err, coverage.ErrForbidden)

		stored, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, manager.String(), "manager")
		require.NoError(t, err)
		require.Len(t, stored, 1)
	})

	t.Run("commercial activity without anchored WG maps to forbidden", func(t *testing.T) {
		f := setupCoverage(t)
		e := f.seedEntry(orgID, func(e *time_entrydomain.TimeEntry) {
			e.UserID = owner
			e.Hours = 8
		})
		f.seedContract(orgID, func(c *contractdomain.ContractResponse) { c.ID = contractID })
		f.activityRepo.ResolveCommercialContextFn = func(ctx context.Context, activityID uuid.UUID) (*activitydomain.CommercialContext, error) {
			return &activitydomain.CommercialContext{ContractID: &contractID}, nil
		}
		req := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID)}

		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, manager.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrForbidden)
	})

	t.Run("fractional-cent hours rejected (WR-01)", func(t *testing.T) {
		// 7.999 rounds to 800 cents in the Σ fast-fail (matches the 8h
		// entry), so only the step-5 whole-cent check can catch it — the
		// DB CHECK (23514) would otherwise surface as a 500.
		f, e := setupHappy(t)
		req := []*coverage.CoverageAllocation{{
			EntryType:  coverage.EntryTypeTime,
			EntryID:    e.ID,
			SourceType: coverage.SourceTypeContract,
			ContractID: &contractID,
			Hours:      7.999,
		}}

		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, approver.String(), "employee")
		require.ErrorIs(t, err, coverage.ErrInvalidRequest)
	})

	t.Run("Σ mismatch and empty set return ErrAllocationSumMismatch", func(t *testing.T) {
		f, e := setupHappy(t)
		bad := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID), {
			EntryType:  coverage.EntryTypeTime,
			EntryID:    e.ID,
			SourceType: coverage.SourceTypeContract,
			ContractID: &contractID,
			Hours:      1,
		}}
		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, bad, approver.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrAllocationSumMismatch)

		_, err = f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, []*coverage.CoverageAllocation{}, approver.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrAllocationSumMismatch)
	})

	t.Run("non-time entry_type is rejected by the D-K branch", func(t *testing.T) {
		f, e := setupHappy(t)
		req := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID)}
		req[0].EntryType = "expense"

		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, approver.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrInvalidRequest)
	})

	t.Run("draft or deleted entry is not coverable", func(t *testing.T) {
		f, e := setupHappy(t)
		req := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID)}
		e.Status = time_entrydomain.StatusSubmitted
		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, approver.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrEntryNotCoverable)
		e.Status = time_entrydomain.StatusApproved
		e.IsDeleted = true
		_, err = f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, approver.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrEntryNotCoverable)
	})

	t.Run("malformed rows are rejected before any repo call", func(t *testing.T) {
		cases := []struct {
			name   string
			mutate func(a *coverage.CoverageAllocation)
			// extraRows adds a compensating 8h row so a mutated row's hours
			// change still leaves Σ == entry hours — the per-row check (step
			// 5) must fire, not the Σ fast-fail (step 3).
			extraRows func(e *time_entrydomain.TimeEntry) []*coverage.CoverageAllocation
		}{
			{"hours zero", func(a *coverage.CoverageAllocation) { a.Hours = 0 }, func(e *time_entrydomain.TimeEntry) []*coverage.CoverageAllocation {
				return []*coverage.CoverageAllocation{{
					EntryType:  coverage.EntryTypeTime,
					EntryID:    e.ID,
					SourceType: coverage.SourceTypeContract,
					ContractID: &contractID,
					Hours:      8,
				}}
			}},
			{"source type outside vocabulary", func(a *coverage.CoverageAllocation) { a.SourceType = "bogus" }, nil},
			{"contract row with both refs pinned", func(a *coverage.CoverageAllocation) { a.UnitID = &uuid.UUID{} }, nil},
			{"contract row with no ref pinned", func(a *coverage.CoverageAllocation) { a.ContractID = nil }, nil},
			{"absorption with contract ref instead of unit", func(a *coverage.CoverageAllocation) {
				a.SourceType = coverage.SourceTypeAbsorption
				a.ContractID = &contractID
				a.UnitID = nil
				a.Reason = strPtr(coverage.AbsorptionReasonGoodwill)
			}, nil},
			{"absorption without reason", func(a *coverage.CoverageAllocation) {
				a.SourceType = coverage.SourceTypeAbsorption
				a.UnitID = &uuid.UUID{}
				a.ContractID = nil
				a.Reason = nil
			}, nil},
			{"absorption with reason outside vocabulary", func(a *coverage.CoverageAllocation) {
				a.SourceType = coverage.SourceTypeAbsorption
				a.UnitID = &uuid.UUID{}
				a.ContractID = nil
				a.Reason = strPtr("PlainInternal")
			}, nil},
			{"transfer without justification", func(a *coverage.CoverageAllocation) {
				a.SourceType = coverage.SourceTypeTransfer
				a.ContractID = &contractID
				a.UnitID = nil
				a.Justification = nil
			}, nil},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				f, e := setupHappy(t)
				req := []*coverage.CoverageAllocation{contractAllocation(e.ID, contractID)}
				tc.mutate(req[0])
				if tc.extraRows != nil {
					req = append(req, tc.extraRows(e)...)
				}

				_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, approver.String(), "manager")
				require.ErrorIs(t, err, coverage.ErrInvalidRequest)
				assert.Empty(t, f.repo.Audits, "no audit must be written for a rejected set")
			})
		}
	})

	t.Run("contract ref visibility matrix", func(t *testing.T) {
		otherOrg := uuid.New()
		cases := []struct {
			name      string
			contract  func() *contractdomain.ContractResponse
			wantError bool
		}{
			{"same-org contract accepted", func() *contractdomain.ContractResponse {
				c := &contractdomain.ContractResponse{Contract: testdata.NewContract()}
				c.CreatedByOrgID = orgID
				return c
			}, false},
			{"shared and adopted contract accepted", func() *contractdomain.ContractResponse {
				c := &contractdomain.ContractResponse{Contract: testdata.NewContract()}
				c.CreatedByOrgID = otherOrg
				c.IsShared = true
				c.IsAdopted = true
				return c
			}, false},
			{"cross-org non-shared contract rejected", func() *contractdomain.ContractResponse {
				c := &contractdomain.ContractResponse{Contract: testdata.NewContract()}
				c.CreatedByOrgID = otherOrg
				return c
			}, true},
			{"shared but not adopted rejected", func() *contractdomain.ContractResponse {
				c := &contractdomain.ContractResponse{Contract: testdata.NewContract()}
				c.CreatedByOrgID = otherOrg
				c.IsShared = true
				c.IsAdopted = false
				return c
			}, true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				f, e := setupHappy(t)
				c := tc.contract()
				c.ID = uuid.New()
				f.contractRepo.Contracts[c.ID] = c
				req := []*coverage.CoverageAllocation{contractAllocation(e.ID, c.ID)}

				_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, approver.String(), "manager")
				if tc.wantError {
					require.ErrorIs(t, err, coverage.ErrInvalidRequest)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("missing contract ref is rejected", func(t *testing.T) {
		f, e := setupHappy(t)
		req := []*coverage.CoverageAllocation{contractAllocation(e.ID, uuid.New())}

		_, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, approver.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrInvalidRequest)
	})

	t.Run("absorption unit must exist and belong to the org", func(t *testing.T) {
		f, e := setupHappy(t)
		unitID := uuid.New()
		f.seedUnit(orgID, unitID)
		okReq := []*coverage.CoverageAllocation{{
			EntryType:  coverage.EntryTypeTime,
			EntryID:    e.ID,
			SourceType: coverage.SourceTypeAbsorption,
			UnitID:     &unitID,
			Hours:      8,
			Reason:     strPtr(coverage.AbsorptionReasonWarrantyBug),
		}}
		stored, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, okReq, approver.String(), "manager")
		require.NoError(t, err)
		require.Len(t, stored, 1)

		crossOrgUnitID := uuid.New()
		f.seedUnit(uuid.New(), crossOrgUnitID)
		badReq := []*coverage.CoverageAllocation{{
			EntryType:  coverage.EntryTypeTime,
			EntryID:    e.ID,
			SourceType: coverage.SourceTypeAbsorption,
			UnitID:     &crossOrgUnitID,
			Hours:      8,
			Reason:     strPtr(coverage.AbsorptionReasonWarrantyBug),
		}}
		_, err = f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, badReq, approver.String(), "manager")
		require.ErrorIs(t, err, coverage.ErrInvalidRequest)
	})

	t.Run("transfer requires justification and an org-visible target contract", func(t *testing.T) {
		f, e := setupHappy(t)
		targetID := uuid.New()
		f.seedContract(orgID, func(c *contractdomain.ContractResponse) { c.ID = targetID })
		req := []*coverage.CoverageAllocation{{
			EntryType:     coverage.EntryTypeTime,
			EntryID:       e.ID,
			SourceType:    coverage.SourceTypeTransfer,
			ContractID:    &targetID,
			Hours:         8,
			Justification: strPtr("reallocated scope"),
		}}
		stored, err := f.svc.ReplaceAllocations(context.Background(), orgID, e.ID, req, approver.String(), "manager")
		require.NoError(t, err)
		require.Len(t, stored, 1)
	})
}

// ---------------------------------------------------------------------------
// TestService_ClosePeriod — manager-only gate + audit + 409 propagation
// ---------------------------------------------------------------------------

func TestService_ClosePeriod(t *testing.T) {
	orgID := uuid.New()
	manager := uuid.New()
	periodStart := time.Now().AddDate(0, -1, 0)
	periodEnd := time.Now()

	t.Run("employee and finance are forbidden", func(t *testing.T) {
		f := setupCoverage(t)
		_, err := f.svc.ClosePeriod(context.Background(), orgID, periodStart, periodEnd, manager, "employee")
		require.ErrorIs(t, err, coverage.ErrForbidden)
		_, err = f.svc.ClosePeriod(context.Background(), orgID, periodStart, periodEnd, manager, "finance")
		require.ErrorIs(t, err, coverage.ErrForbidden)
	})

	t.Run("manager closes and passes the coverage-closed audit", func(t *testing.T) {
		f := setupCoverage(t)

		got, err := f.svc.ClosePeriod(context.Background(), orgID, periodStart, periodEnd, manager, "manager")
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, got.ID, "the service must generate the close id")
		require.Len(t, f.repo.Audits, 1)
		assert.Equal(t, coverage.AuditActionCoverageClosed, f.repo.Audits[0].Action)
		assert.Equal(t, coverage.AuditEntityCoverageAllocation, f.repo.Audits[0].EntityType)
		assert.Equal(t, got.ID, f.repo.Audits[0].EntityID)
		require.NotNil(t, f.repo.Audits[0].ActorID)
		assert.Equal(t, manager, *f.repo.Audits[0].ActorID)
		require.NotNil(t, f.repo.Audits[0].Payload)
		_, hasStart := f.repo.Audits[0].Payload["period_start"]
		_, hasEnd := f.repo.Audits[0].Payload["period_end"]
		assert.True(t, hasStart && hasEnd, "payload carries the closed period")
	})

	t.Run("overlapping close propagates ErrPeriodAlreadyClosed", func(t *testing.T) {
		f := setupCoverage(t)
		f.repo.ClosePeriodOverlapping = true

		_, err := f.svc.ClosePeriod(context.Background(), orgID, periodStart, periodEnd, manager, "manager")
		require.ErrorIs(t, err, coverage.ErrPeriodAlreadyClosed)
	})

	t.Run("inverted period rejected (WR-02)", func(t *testing.T) {
		f := setupCoverage(t)

		_, err := f.svc.ClosePeriod(context.Background(), orgID, periodEnd, periodStart, manager, "manager")
		require.ErrorIs(t, err, coverage.ErrInvalidRequest)
	})
}
