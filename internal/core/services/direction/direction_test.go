package directionsvc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	directiondomain "github.com/stefanoprivitera/hourglass/internal/core/domain/direction"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/orgsettings"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	wgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	orgsettingssvc "github.com/stefanoprivitera/hourglass/internal/core/services/orgsettings"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture wires the direction service against the hermetic testdata mocks
// (13-03/13-06): MockDirectionRepo for the store + read-model stubs,
// MockActivityRepo/MockWorkingGroupRepo/MockUnitRepo for the gate chain
// inputs, MockOrgRepo for membership validity + the planning-mode override
// seam, and MockOrgSettingsRepo for the org default key. The routing and
// orgsettings services are built on the SAME mock instances (D-G parity —
// single shared resolution, exactly as cmd/server wires them).
type fixture struct {
	svc         *Service
	dirRepo     *testdata.MockDirectionRepo
	actRepo     *testdata.MockActivityRepo
	wgRepo      *testdata.MockWorkingGroupRepo
	unitRepo    *testdata.MockUnitRepo
	orgRepo     *testdata.MockOrgRepo
	settings    *testdata.MockOrgSettingsRepo
	settingsSvc *orgsettingssvc.Service
	routingSvc  *routing.Service
}

func setup(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		dirRepo:  &testdata.MockDirectionRepo{},
		actRepo:  &testdata.MockActivityRepo{},
		wgRepo:   &testdata.MockWorkingGroupRepo{},
		unitRepo: &testdata.MockUnitRepo{},
		orgRepo:  &testdata.MockOrgRepo{},
		settings: &testdata.MockOrgSettingsRepo{},
	}
	f.routingSvc = routing.NewService(f.wgRepo, f.actRepo, f.unitRepo)
	f.settingsSvc = orgsettingssvc.NewService(f.settings, f.orgRepo)
	f.svc = NewService(f.dirRepo, f.actRepo, f.wgRepo, f.unitRepo, f.orgRepo, f.settingsSvc, f.routingSvc)
	return f
}

// day returns a UTC-midnight day stamp (the read-model day semantics are
// timezone-free — the 13-06 repo normalizes DATE columns to UTC midnight).
func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func ptrF(v float64) *float64     { return &v }
func ptrT(v time.Time) *time.Time { return &v }
func ptrStr(v string) *string     { return &v }

// seedActivity anchors an org-scoped activity in the mock activity repo.
func (f *fixture) seedActivity(orgID, activityID uuid.UUID) {
	if f.actRepo.Activities == nil {
		f.actRepo.Activities = make(map[uuid.UUID]*activitydomain.ActivityResponse)
	}
	f.actRepo.Activities[activityID] = &activitydomain.ActivityResponse{
		Activity: activitydomain.Activity{ID: activityID, OrgID: orgID, IsActive: true},
	}
}

// seedMembership anchors a membership with an optional planning-mode override
// (nil = org default) and an optional validity window (nil bounds = open).
func (f *fixture) seedMembership(userID, orgID uuid.UUID, planningMode *string, validFrom, validUntil *time.Time) {
	if f.orgRepo.Memberships == nil {
		f.orgRepo.Memberships = make(map[string]*auth.OrganizationMembership)
	}
	f.orgRepo.Memberships[userID.String()+":"+orgID.String()] = &auth.OrganizationMembership{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "employee",
		IsActive:       true,
		PlanningMode:   planningMode,
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
	}
}

// seedWG anchors a working group in the mock WG repo.
func (f *fixture) seedWG(wgID, orgID, subprojectID, managerID uuid.UUID) {
	if f.wgRepo.Groups == nil {
		f.wgRepo.Groups = make(map[uuid.UUID]*wgdomain.WorkingGroup)
	}
	f.wgRepo.Groups[wgID] = &wgdomain.WorkingGroup{
		ID:           wgID,
		OrgID:        orgID,
		SubprojectID: subprojectID,
		ManagerID:    managerID,
		IsActive:     true,
	}
}

// seedWgMember anchors a WG membership (D-13-12 membership check input).
func (f *fixture) seedWgMember(wgID, userID uuid.UUID) {
	if f.wgRepo.WGMembers == nil {
		f.wgRepo.WGMembers = make(map[string][]wgdomain.WorkingGroupMember)
	}
	key := wgID.String()
	f.wgRepo.WGMembers[key] = append(f.wgRepo.WGMembers[key], wgdomain.WorkingGroupMember{
		ID: uuid.New(), WGID: wgID, UserID: userID, UnitID: uuid.New(), Role: "employee",
	})
}

// seedUnitMember anchors a unit membership (scope resolution + primary-unit
// resolution inputs).
func (f *fixture) seedUnitMember(unitID string, userID uuid.UUID, isPrimary bool) {
	if f.unitRepo.UnitMembers == nil {
		f.unitRepo.UnitMembers = make(map[string][]unit.UnitMember)
	}
	key := unitID
	f.unitRepo.UnitMembers[key] = append(f.unitRepo.UnitMembers[key], unit.UnitMember{
		ID: uuid.New().String(), UserID: userID, UnitID: unitID, IsPrimary: isPrimary, Role: "employee",
	})
}

// seedDirectionRow seeds a direction row in the mock store (map key = id).
func (f *fixture) seedDirectionRow(d *directiondomain.Direction) {
	if f.dirRepo.Directions == nil {
		f.dirRepo.Directions = make(map[uuid.UUID]*directiondomain.Direction)
	}
	f.dirRepo.Directions[d.ID] = d
}

// ---------------------------------------------------------------------------
// TestService_Create — Task 1: the gate chain (XOR → hours → same-org →
// WG-scope → mode → routing), supersede fast-fail, warnings-in-response
// ---------------------------------------------------------------------------

func TestService_Create(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	colleagueID := uuid.New()
	activityID := uuid.New()
	ctx := context.Background()
	selfPlanned := orgsettings.ModeSelfPlanned
	managerPlanned := orgsettings.ModeManagerPlanned

	baseReq := func() *CreateDirectionRequest {
		return &CreateDirectionRequest{
			DirectedTo:  &actorID,
			ActivityID:  activityID,
			PlannedDate: ptrT(day(2026, 8, 14)),
			EstHours:    ptrF(8.0),
		}
	}

	t.Run("self-direction in self_planned mode succeeds without a routing call (D-S)", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &selfPlanned, nil, nil)

		// role=employee with NO WG/unit seeded: any routing call would resolve
		// to the role-gated terminal stage and reject — success proves the
		// self_planned self-direction path never calls routing (D-S).
		created, warnings, err := f.svc.Create(ctx, orgID, actorID, "employee", baseReq())
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, actorID, created.DirectedBy)
		assert.Equal(t, activityID, created.ActivityID)
		assert.Equal(t, directiondomain.StatusDraft, created.Status)
		assert.NotNil(t, created.DirectedTo)
		assert.Equal(t, actorID, *created.DirectedTo)
		assert.Empty(t, warnings, "no absence/validity/coverage data → no warnings")

		require.Len(t, f.dirRepo.Audits, 1, "the created audit lands in the same tx as the row")
		a := f.dirRepo.Audits[0]
		assert.Equal(t, directiondomain.AuditEntityDirection, a.EntityType)
		assert.Equal(t, directiondomain.AuditActionCreated, a.Action)
		assert.Equal(t, created.ID, a.EntityID)
		require.NotNil(t, a.ActorID)
		assert.Equal(t, actorID, *a.ActorID)
	})

	t.Run("self-direction in manager_planned mode passes when the actor is in the approver set", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &managerPlanned, nil, nil)
		// WG anchored to the activity with the actor as manager → the BE-014
		// approver set contains the actor (strict reading A9: even
		// self-direction must pass the routing gate in manager_planned mode).
		f.seedWG(uuid.New(), orgID, activityID, actorID)

		created, _, err := f.svc.Create(ctx, orgID, actorID, "employee", baseReq())
		require.NoError(t, err)
		require.NotNil(t, created)
	})

	t.Run("self-direction in manager_planned mode is role-gated when no manager stage resolves", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &managerPlanned, nil, nil)
		// No WG, no commercial context, no unit manager → terminal role-gated
		// stage: only the org role claim 'manager' passes.
		_, _, err := f.svc.Create(ctx, orgID, actorID, "employee", baseReq())
		require.ErrorIs(t, err, directiondomain.ErrForbidden)

		created, _, err := f.svc.Create(ctx, orgID, actorID, "manager", baseReq())
		require.NoError(t, err)
		require.NotNil(t, created)
	})

	t.Run("employee creating for a colleague in self_planned mode is forbidden", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(colleagueID, orgID, &selfPlanned, nil, nil)

		req := baseReq()
		req.DirectedTo = &colleagueID
		_, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.ErrorIs(t, err, directiondomain.ErrForbidden,
			"self_planned mode: only the employee creates their own rows (strict reading A9)")
	})

	t.Run("manager creates for an employee in manager_planned mode via the approver set", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(colleagueID, orgID, &managerPlanned, nil, nil)
		f.seedWG(uuid.New(), orgID, activityID, actorID)

		req := baseReq()
		req.DirectedTo = &colleagueID
		created, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, colleagueID, *created.DirectedTo)
	})

	t.Run("XOR violation: both targets rejected with ErrInvalidTarget", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &selfPlanned, nil, nil)

		req := baseReq()
		wgID := uuid.New()
		req.WgID = &wgID
		_, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.ErrorIs(t, err, directiondomain.ErrInvalidTarget)
	})

	t.Run("XOR violation: neither target rejected with ErrInvalidTarget", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)

		req := baseReq()
		req.DirectedTo = nil
		_, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.ErrorIs(t, err, directiondomain.ErrInvalidTarget)
	})

	t.Run("est_hours 0 / negative / sub-cent rejected with ErrInvalidHours", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &selfPlanned, nil, nil)

		for _, hours := range []float64{0, -1, 1.005} {
			req := baseReq()
			req.EstHours = &hours
			_, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
			require.ErrorIs(t, err, directiondomain.ErrInvalidHours, "est_hours %v must be rejected", hours)
		}
	})

	t.Run("scheduled row without est_hours rejected with ErrInvalidHours", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &selfPlanned, nil, nil)

		req := baseReq()
		req.EstHours = nil
		_, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.ErrorIs(t, err, directiondomain.ErrInvalidHours, "scheduled requires est_hours (D-13-02)")
	})

	t.Run("WG row with planned_date rejected with ErrInvalidRequest (queued-only D-13-17)", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		wgID := uuid.New()
		f.seedWG(wgID, orgID, activityID, actorID)

		req := baseReq()
		req.DirectedTo = nil
		req.WgID = &wgID
		_, _, err := f.svc.Create(ctx, orgID, actorID, "manager", req)
		require.ErrorIs(t, err, directiondomain.ErrInvalidRequest)
	})

	t.Run("WG row for an activity outside the WG scope rejected with ErrInvalidRequest", func(t *testing.T) {
		f := setup(t)
		otherActivityID := uuid.New()
		f.seedActivity(orgID, activityID)
		f.seedActivity(orgID, otherActivityID)
		wgID := uuid.New()
		// WG anchored to the OTHER activity; the requested activity shares no
		// ancestry with the anchor → scope predicate fails (A5, Pitfall 9).
		f.seedWG(wgID, orgID, otherActivityID, actorID)

		req := baseReq()
		req.DirectedTo = nil
		req.WgID = &wgID
		req.PlannedDate = nil
		req.EstHours = nil
		_, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.ErrorIs(t, err, directiondomain.ErrInvalidRequest)
	})

	t.Run("WG row in scope via ancestry succeeds and routes on the anchored activity (A10)", func(t *testing.T) {
		f := setup(t)
		anchorID := uuid.New()
		f.seedActivity(orgID, activityID)
		f.seedActivity(orgID, anchorID)
		// activity is a child of the WG's anchor → the anchor is in
		// GetAncestry(activity) → in scope.
		f.actRepo.Activities[activityID].ParentID = &anchorID
		wgID := uuid.New()
		f.seedWG(wgID, orgID, anchorID, actorID)

		req := baseReq()
		req.DirectedTo = nil
		req.WgID = &wgID
		req.PlannedDate = nil
		req.EstHours = nil
		created, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, wgID, *created.WgID)
	})

	t.Run("WG row routed by the role-gated manager when no WG anchors the anchored activity", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		wgID := uuid.New()
		// WG anchored to the activity itself; no OTHER WG resolves — the
		// routing call on the anchored activity finds this WG.
		f.seedWG(wgID, orgID, activityID, actorID)

		req := baseReq()
		req.DirectedTo = nil
		req.WgID = &wgID
		req.PlannedDate = nil
		req.EstHours = nil
		created, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.NoError(t, err)
		require.NotNil(t, created)
	})

	t.Run("supersede target already superseded rejected with ErrInvalidTransition", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &selfPlanned, nil, nil)
		targetID := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: targetID, OrgID: orgID, DirectedBy: actorID, DirectedTo: &actorID,
			ActivityID: activityID, Status: directiondomain.StatusSuperseded,
		})

		req := baseReq()
		req.SupersedesID = &targetID
		_, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.ErrorIs(t, err, directiondomain.ErrInvalidTransition)
	})

	t.Run("cross-org activity rejected with ErrInvalidRequest", func(t *testing.T) {
		f := setup(t)
		otherOrgID := uuid.New()
		f.seedActivity(otherOrgID, activityID)
		f.seedMembership(actorID, orgID, &selfPlanned, nil, nil)

		_, _, err := f.svc.Create(ctx, orgID, actorID, "employee", baseReq())
		require.ErrorIs(t, err, directiondomain.ErrInvalidRequest)
	})

	t.Run("queued self-row without est_hours is allowed (budget optional, D-AA)", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &selfPlanned, nil, nil)

		req := baseReq()
		req.PlannedDate = nil
		req.EstHours = nil
		created, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Nil(t, created.PlannedDate)
		assert.Nil(t, created.EstHours)
	})

	t.Run("create returns warnings without ever rejecting (D-13-28)", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &selfPlanned, nil, nil)
		// Full absence window covering the planned day → away warning rides
		// the create response; the write still succeeds.
		f.dirRepo.SetAbsenceWindows([]directiondomain.AbsenceWindow{{
			EmployeeID: actorID, Kind: "holiday", StartsOn: day(2026, 8, 14), EndsOn: day(2026, 8, 14),
		}})

		created, warnings, err := f.svc.Create(ctx, orgID, actorID, "employee", baseReq())
		require.NoError(t, err)
		require.NotNil(t, created)
		require.Len(t, warnings, 1)
		assert.Equal(t, directiondomain.WarningAway, warnings[0].Type)
		assert.Equal(t, "Away 14 Aug", warnings[0].Message)
	})
}
