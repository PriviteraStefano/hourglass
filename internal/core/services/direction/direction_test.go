package directionsvc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
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

	t.Run("WG row whose activity is the anchor succeeds via the approver set (A10)", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		wgID := uuid.New()
		// WG anchored to the row's activity itself; the routing call on the
		// anchored activity resolves this WG's manager + delegates.
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

	t.Run("supersede-on-create writes created + superseded audit rows (CR-02)", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		f.seedMembership(actorID, orgID, &selfPlanned, nil, nil)
		targetID := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: targetID, OrgID: orgID, DirectedBy: actorID, DirectedTo: &actorID,
			ActivityID: activityID, Status: directiondomain.StatusDraft,
		})

		req := baseReq()
		req.SupersedesID = &targetID
		created, _, err := f.svc.Create(ctx, orgID, actorID, "employee", req)
		require.NoError(t, err)
		require.NotNil(t, created)

		require.Len(t, f.dirRepo.Audits, 2, "supersede-on-create must write created + superseded (CR-02)")
		a0 := f.dirRepo.Audits[0]
		assert.Equal(t, directiondomain.AuditActionCreated, a0.Action)
		assert.Equal(t, created.ID, a0.EntityID)
		require.NotNil(t, a0.ActorID)
		assert.Equal(t, actorID, *a0.ActorID)
		assert.Nil(t, a0.Payload)

		a1 := f.dirRepo.Audits[1]
		assert.Equal(t, directiondomain.AuditActionSuperseded, a1.Action)
		assert.Equal(t, targetID, a1.EntityID, "the superseded audit addresses the flipped target")
		assert.Equal(t, directiondomain.AuditEntityDirection, a1.EntityType)
		require.NotNil(t, a1.ActorID)
		assert.Equal(t, actorID, *a1.ActorID)
		assert.Nil(t, a1.Payload)
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

// ---------------------------------------------------------------------------
// TestService_Activate — Task 2: matrix fast-fail + creator/manager gate
// ---------------------------------------------------------------------------

func TestService_Activate(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	employeeID := uuid.New()
	otherID := uuid.New()
	activityID := uuid.New()
	ctx := context.Background()

	seedDraft := func(f *fixture, directedBy uuid.UUID) uuid.UUID {
		id := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: id, OrgID: orgID, DirectedBy: directedBy, DirectedTo: &employeeID,
			ActivityID: activityID, Status: directiondomain.StatusDraft,
		})
		return id
	}

	t.Run("creator activates their own draft row", func(t *testing.T) {
		f := setup(t)
		id := seedDraft(f, actorID)

		activated, err := f.svc.Activate(ctx, orgID, actorID, "employee", id)
		require.NoError(t, err)
		require.NotNil(t, activated)
		assert.Equal(t, directiondomain.StatusActive, activated.Status)

		require.Len(t, f.dirRepo.Audits, 1)
		a := f.dirRepo.Audits[0]
		assert.Equal(t, directiondomain.AuditActionActivated, a.Action)
		assert.Equal(t, id, a.EntityID)
		require.NotNil(t, a.ActorID)
		assert.Equal(t, actorID, *a.ActorID)
	})

	t.Run("cancelled row matrix fast-fails before the repo call", func(t *testing.T) {
		f := setup(t)
		id := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: id, OrgID: orgID, DirectedBy: actorID, DirectedTo: &employeeID,
			ActivityID: activityID, Status: directiondomain.StatusCancelled, Reason: ptrStr("done"),
		})
		trap := false
		f.dirRepo.ActivateFn = func(ctx context.Context, orgID, id uuid.UUID, a *audit.AuditLog) (*directiondomain.Direction, error) {
			trap = true
			return nil, nil
		}

		_, err := f.svc.Activate(ctx, orgID, actorID, "employee", id)
		require.ErrorIs(t, err, directiondomain.ErrInvalidTransition)
		assert.False(t, trap, "the repo must not be reached when the pool-level matrix fails")
	})

	t.Run("non-creator manager activates via the approver set", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		id := seedDraft(f, otherID)
		f.seedWG(uuid.New(), orgID, activityID, actorID)

		activated, err := f.svc.Activate(ctx, orgID, actorID, "employee", id)
		require.NoError(t, err)
		require.NotNil(t, activated)
		assert.Equal(t, directiondomain.StatusActive, activated.Status)
	})

	t.Run("non-creator without manager reach is forbidden", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		id := seedDraft(f, otherID)

		_, err := f.svc.Activate(ctx, orgID, actorID, "employee", id)
		require.ErrorIs(t, err, directiondomain.ErrForbidden)
	})

	t.Run("missing row surfaces ErrDirectionNotFound", func(t *testing.T) {
		f := setup(t)
		_, err := f.svc.Activate(ctx, orgID, actorID, "employee", uuid.New())
		require.ErrorIs(t, err, directiondomain.ErrDirectionNotFound)
	})
}

// ---------------------------------------------------------------------------
// TestService_Cancel — Task 2: reason requirement + matrix + gate
// ---------------------------------------------------------------------------

func TestService_Cancel(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	employeeID := uuid.New()
	otherID := uuid.New()
	activityID := uuid.New()
	ctx := context.Background()

	t.Run("cancel without reason is rejected before the pool read", func(t *testing.T) {
		f := setup(t)
		// No row seeded: ErrCancelReasonRequired must win without a Get.
		_, err := f.svc.Cancel(ctx, orgID, actorID, "employee", uuid.New(), "")
		require.ErrorIs(t, err, directiondomain.ErrCancelReasonRequired)
	})

	t.Run("creator cancels own draft row with a reason", func(t *testing.T) {
		f := setup(t)
		id := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: id, OrgID: orgID, DirectedBy: actorID, DirectedTo: &employeeID,
			ActivityID: activityID, Status: directiondomain.StatusDraft,
		})

		cancelled, err := f.svc.Cancel(ctx, orgID, actorID, "employee", id, "scope changed")
		require.NoError(t, err)
		require.NotNil(t, cancelled)
		assert.Equal(t, directiondomain.StatusCancelled, cancelled.Status)
		require.NotNil(t, cancelled.Reason)
		assert.Equal(t, "scope changed", *cancelled.Reason)

		require.Len(t, f.dirRepo.Audits, 1)
		a := f.dirRepo.Audits[0]
		assert.Equal(t, directiondomain.AuditActionCancelled, a.Action)
		assert.Equal(t, id, a.EntityID)
		require.NotNil(t, a.Payload)
		assert.Equal(t, "scope changed", a.Payload["reason"])
	})

	t.Run("cancelled row matrix fast-fails before the repo call", func(t *testing.T) {
		f := setup(t)
		id := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: id, OrgID: orgID, DirectedBy: actorID, DirectedTo: &employeeID,
			ActivityID: activityID, Status: directiondomain.StatusCancelled, Reason: ptrStr("done"),
		})
		trap := false
		f.dirRepo.CancelFn = func(ctx context.Context, orgID, id uuid.UUID, reason string, a *audit.AuditLog) (*directiondomain.Direction, error) {
			trap = true
			return nil, nil
		}

		_, err := f.svc.Cancel(ctx, orgID, actorID, "employee", id, "again")
		require.ErrorIs(t, err, directiondomain.ErrInvalidTransition)
		assert.False(t, trap, "the repo must not be reached when the pool-level matrix fails")
	})

	t.Run("non-creator without manager reach is forbidden", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		id := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: id, OrgID: orgID, DirectedBy: otherID, DirectedTo: &employeeID,
			ActivityID: activityID, Status: directiondomain.StatusDraft,
		})

		_, err := f.svc.Cancel(ctx, orgID, actorID, "employee", id, "scope changed")
		require.ErrorIs(t, err, directiondomain.ErrForbidden)
	})
}

// ---------------------------------------------------------------------------
// TestService_Claim — Task 2: active + membership + hours fast-fails
// ---------------------------------------------------------------------------

func TestService_Claim(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	activityID := uuid.New()
	ctx := context.Background()

	seedWgRow := func(f *fixture, status string) uuid.UUID {
		wgID := uuid.New()
		wgRowID := uuid.New()
		f.seedWG(wgID, orgID, activityID, uuid.New())
		f.seedDirectionRow(&directiondomain.Direction{
			ID: wgRowID, OrgID: orgID, DirectedBy: uuid.New(), WgID: &wgID,
			ActivityID: activityID, Status: status,
		})
		return wgRowID
	}

	t.Run("member claims an active WG row", func(t *testing.T) {
		f := setup(t)
		wgRowID := seedWgRow(f, directiondomain.StatusActive)
		f.seedWgMember(*f.dirRepo.Directions[wgRowID].WgID, actorID)

		claim, err := f.svc.Claim(ctx, orgID, actorID, "employee", wgRowID, 4.0)
		require.NoError(t, err)
		require.NotNil(t, claim)
		assert.Equal(t, wgRowID, *claim.OriginDirectionID)
		require.NotNil(t, claim.DirectedTo)
		assert.Equal(t, actorID, *claim.DirectedTo)

		require.Len(t, f.dirRepo.Audits, 1)
		a := f.dirRepo.Audits[0]
		assert.Equal(t, directiondomain.AuditActionClaimed, a.Action)
		assert.Equal(t, uuid.Nil, a.EntityID, "the repo pins entity_id to the claim row it creates (13-05)")
		require.NotNil(t, a.Payload)
		assert.Equal(t, wgRowID.String(), a.Payload["wg_row_id"])
		assert.Equal(t, 4.0, a.Payload["est_hours"])
	})

	t.Run("claim on a draft WG row is rejected with ErrWgRowNotActive", func(t *testing.T) {
		f := setup(t)
		wgRowID := seedWgRow(f, directiondomain.StatusDraft)
		f.seedWgMember(*f.dirRepo.Directions[wgRowID].WgID, actorID)

		_, err := f.svc.Claim(ctx, orgID, actorID, "employee", wgRowID, 4.0)
		require.ErrorIs(t, err, directiondomain.ErrWgRowNotActive)
	})

	t.Run("claim by a non-member is rejected with ErrNotWgMember", func(t *testing.T) {
		f := setup(t)
		wgRowID := seedWgRow(f, directiondomain.StatusActive)

		_, err := f.svc.Claim(ctx, orgID, actorID, "employee", wgRowID, 4.0)
		require.ErrorIs(t, err, directiondomain.ErrNotWgMember)
	})

	t.Run("claim with non-positive or sub-cent hours is rejected with ErrInvalidHours", func(t *testing.T) {
		f := setup(t)
		for _, hours := range []float64{0, -2, 1.005} {
			wgRowID := seedWgRow(f, directiondomain.StatusActive)
			f.seedWgMember(*f.dirRepo.Directions[wgRowID].WgID, actorID)
			_, err := f.svc.Claim(ctx, orgID, actorID, "employee", wgRowID, hours)
			require.ErrorIs(t, err, directiondomain.ErrInvalidHours, "est_hours %v must be rejected", hours)
		}
	})

	t.Run("claim on a user-targeted row returns ErrDirectionNotFound without touching ListMembers (CR-01)", func(t *testing.T) {
		f := setup(t)
		rowID := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: rowID, OrgID: orgID, DirectedBy: uuid.New(), DirectedTo: &actorID,
			ActivityID: activityID, Status: directiondomain.StatusActive,
		})
		trap := false
		f.wgRepo.ListMembersFn = func(ctx context.Context, wgID uuid.UUID) ([]wgdomain.WorkingGroupMember, error) {
			trap = true
			return nil, nil
		}

		_, err := f.svc.Claim(ctx, orgID, actorID, "employee", rowID, 4.0)
		require.ErrorIs(t, err, directiondomain.ErrDirectionNotFound,
			"user-targeted rows are not claimable — the repo wg_id IS NOT NULL predicate mirrored at the service (CR-01)")
		assert.False(t, trap, "ListMembers must never be reached for a user-targeted row (no deref path remains)")
	})

	t.Run("claim on a missing row surfaces ErrDirectionNotFound", func(t *testing.T) {
		f := setup(t)
		_, err := f.svc.Claim(ctx, orgID, actorID, "employee", uuid.New(), 4.0)
		require.ErrorIs(t, err, directiondomain.ErrDirectionNotFound)
	})
}

// ---------------------------------------------------------------------------
// TestService_Unclaim — Task 2: claim-row guard + reason + claimant/creator/
// manager gate (D-13-16)
// ---------------------------------------------------------------------------

func TestService_Unclaim(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	employeeID := uuid.New()
	otherID := uuid.New()
	activityID := uuid.New()
	ctx := context.Background()

	seedClaimRow := func(f *fixture, directedTo, directedBy *uuid.UUID) uuid.UUID {
		id := uuid.New()
		origin := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: id, OrgID: orgID, DirectedBy: *directedBy, DirectedTo: directedTo,
			ActivityID: activityID, Status: directiondomain.StatusDraft, OriginDirectionID: &origin,
		})
		return id
	}

	t.Run("unclaim of a non-claim row is rejected with ErrInvalidRequest", func(t *testing.T) {
		f := setup(t)
		id := uuid.New()
		f.seedDirectionRow(&directiondomain.Direction{
			ID: id, OrgID: orgID, DirectedBy: actorID, DirectedTo: &employeeID,
			ActivityID: activityID, Status: directiondomain.StatusDraft,
		})

		_, err := f.svc.Unclaim(ctx, orgID, actorID, "employee", id, "no longer wanted")
		require.ErrorIs(t, err, directiondomain.ErrInvalidRequest)
	})

	t.Run("unclaim without a reason is rejected with ErrCancelReasonRequired", func(t *testing.T) {
		f := setup(t)
		id := seedClaimRow(f, &actorID, &actorID)

		_, err := f.svc.Unclaim(ctx, orgID, actorID, "employee", id, "")
		require.ErrorIs(t, err, directiondomain.ErrCancelReasonRequired)
	})

	t.Run("claimant unclaims their own claim row with a reason", func(t *testing.T) {
		f := setup(t)
		id := seedClaimRow(f, &actorID, &otherID)

		unclaimed, err := f.svc.Unclaim(ctx, orgID, actorID, "employee", id, "no longer wanted")
		require.NoError(t, err)
		require.NotNil(t, unclaimed)
		assert.Equal(t, directiondomain.StatusCancelled, unclaimed.Status)
		require.NotNil(t, unclaimed.Reason)
		assert.Equal(t, "no longer wanted", *unclaimed.Reason)

		require.Len(t, f.dirRepo.Audits, 1)
		a := f.dirRepo.Audits[0]
		assert.Equal(t, directiondomain.AuditActionCancelled, a.Action)
		assert.Equal(t, id, a.EntityID)
		require.NotNil(t, a.Payload)
		assert.Equal(t, "no longer wanted", a.Payload["reason"])
	})

	t.Run("creator can unclaim the claim row", func(t *testing.T) {
		f := setup(t)
		id := seedClaimRow(f, &employeeID, &actorID)

		unclaimed, err := f.svc.Unclaim(ctx, orgID, actorID, "employee", id, "manager recall")
		require.NoError(t, err)
		require.NotNil(t, unclaimed)
		assert.Equal(t, directiondomain.StatusCancelled, unclaimed.Status)
	})

	t.Run("neither claimant nor creator nor manager is forbidden", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		id := seedClaimRow(f, &employeeID, &otherID)

		_, err := f.svc.Unclaim(ctx, orgID, actorID, "employee", id, "nope")
		require.ErrorIs(t, err, directiondomain.ErrForbidden)
	})

	t.Run("manager unclaims via the approver set", func(t *testing.T) {
		f := setup(t)
		f.seedActivity(orgID, activityID)
		id := seedClaimRow(f, &employeeID, &otherID)
		f.seedWG(uuid.New(), orgID, activityID, actorID)

		unclaimed, err := f.svc.Unclaim(ctx, orgID, actorID, "employee", id, "manager recall")
		require.NoError(t, err)
		require.NotNil(t, unclaimed)
		assert.Equal(t, directiondomain.StatusCancelled, unclaimed.Status)
	})
}

// ---------------------------------------------------------------------------
// TestService_Warnings — Task 3: the pure warning overlay (D-13-28/30/31)
// ---------------------------------------------------------------------------

func TestService_Warnings(t *testing.T) {
	orgID := uuid.New()
	empID := uuid.New()
	ctx := context.Background()
	period := func() (time.Time, time.Time) { return day(2026, 8, 10), day(2026, 8, 21) }

	t.Run("invalid employee (valid_until in the past) gets one invalid warning", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, ptrT(day(2026, 7, 31)))
		start, end := period()

		warnings, err := f.svc.computeWarnings(ctx, orgID, []uuid.UUID{empID}, start, end)
		require.NoError(t, err)
		require.Len(t, warnings, 1, "the day-less invalid message dedupes to one warning per employee")
		assert.Equal(t, directiondomain.WarningInvalid, warnings[0].Type)
		assert.Equal(t, "Outside validity period", warnings[0].Message)
	})

	t.Run("full absence window emits the contiguous away range (en dash)", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		f.dirRepo.SetAbsenceWindows([]directiondomain.AbsenceWindow{{
			EmployeeID: empID, Kind: "holiday",
			StartsOn: day(2026, 8, 10), EndsOn: day(2026, 8, 21),
		}})
		start, end := period()

		warnings, err := f.svc.computeWarnings(ctx, orgID, []uuid.UUID{empID}, start, end)
		require.NoError(t, err)
		require.Len(t, warnings, 1)
		assert.Equal(t, directiondomain.WarningAway, warnings[0].Type)
		assert.Equal(t, "Away 10 Aug\u201321 Aug", warnings[0].Message)
	})

	t.Run("single-day full absence emits the away day message", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		f.dirRepo.SetAbsenceWindows([]directiondomain.AbsenceWindow{{
			EmployeeID: empID, Kind: "unavailable",
			StartsOn: day(2026, 8, 14), EndsOn: day(2026, 8, 14),
		}})
		start, end := day(2026, 8, 14), day(2026, 8, 14)

		warnings, err := f.svc.computeWarnings(ctx, orgID, []uuid.UUID{empID}, start, end)
		require.NoError(t, err)
		require.Len(t, warnings, 1)
		assert.Equal(t, directiondomain.WarningAway, warnings[0].Type)
		assert.Equal(t, "Away 14 Aug", warnings[0].Message)
	})

	t.Run("partial-day permit emits the partial message", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		hours := 4.0
		f.dirRepo.SetAbsenceWindows([]directiondomain.AbsenceWindow{{
			EmployeeID: empID, Kind: "permit",
			StartsOn: day(2026, 8, 14), EndsOn: day(2026, 8, 14), Hours: &hours,
		}})
		start, end := day(2026, 8, 14), day(2026, 8, 14)

		warnings, err := f.svc.computeWarnings(ctx, orgID, []uuid.UUID{empID}, start, end)
		require.NoError(t, err)
		require.Len(t, warnings, 1)
		assert.Equal(t, directiondomain.WarningPartial, warnings[0].Type)
		assert.Equal(t, "Partial 14 Aug", warnings[0].Message)
	})

	t.Run("planned above capacity emits the over-capacity message", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		f.dirRepo.SetCoverageRows([]directiondomain.CoverageRow{{
			EmployeeID: empID, Date: day(2026, 8, 16), Capacity: 8, Planned: 10, Gap: -2,
		}})
		start, end := day(2026, 8, 16), day(2026, 8, 16)

		warnings, err := f.svc.computeWarnings(ctx, orgID, []uuid.UUID{empID}, start, end)
		require.NoError(t, err)
		require.Len(t, warnings, 1)
		assert.Equal(t, directiondomain.WarningOverCapacity, warnings[0].Type)
		assert.Equal(t, "Over capacity 16 Aug", warnings[0].Message)
	})

	t.Run("validity boundary days are valid (inclusive)", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, ptrT(day(2026, 8, 14)), nil)
		start, end := day(2026, 8, 14), day(2026, 8, 14)

		warnings, err := f.svc.computeWarnings(ctx, orgID, []uuid.UUID{empID}, start, end)
		require.NoError(t, err)
		assert.Empty(t, warnings)
	})

	t.Run("no absence/coverage data yields no warnings", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		start, end := period()

		warnings, err := f.svc.computeWarnings(ctx, orgID, []uuid.UUID{empID}, start, end)
		require.NoError(t, err)
		assert.Empty(t, warnings)
	})
}

// ---------------------------------------------------------------------------
// TestService_ListPlan — Task 3: the self-or-manager read gate + warnings
// ---------------------------------------------------------------------------

func TestService_ListPlan(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	otherID := uuid.New()
	ctx := context.Background()
	start, end := day(2026, 8, 10), day(2026, 8, 21)

	t.Run("non-manager without an employee filter is forbidden (org-wide view is manager-only)", func(t *testing.T) {
		f := setup(t)
		trap := false
		f.dirRepo.ListPlanFn = func(ctx context.Context, orgID uuid.UUID, employeeID *uuid.UUID, ps, pe time.Time) ([]directiondomain.PlanRow, error) {
			trap = true
			return nil, nil
		}

		_, err := f.svc.ListPlan(ctx, orgID, actorID, "employee", nil, start, end)
		require.ErrorIs(t, err, directiondomain.ErrForbidden)
		assert.False(t, trap, "the repo must not be reached for a forbidden view")
	})

	t.Run("non-manager requesting another employee's plan is forbidden", func(t *testing.T) {
		f := setup(t)
		_, err := f.svc.ListPlan(ctx, orgID, actorID, "employee", &otherID, start, end)
		require.ErrorIs(t, err, directiondomain.ErrForbidden)
	})

	t.Run("non-manager reading their own plan succeeds with rows + warnings", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(actorID, orgID, nil, nil, nil)
		f.dirRepo.SetPlanRows([]directiondomain.PlanRow{{Direction: directiondomain.Direction{
			ID: uuid.New(), OrgID: orgID, DirectedBy: uuid.New(), DirectedTo: &actorID,
			ActivityID: uuid.New(), Status: directiondomain.StatusDraft,
		}}})
		f.dirRepo.SetAbsenceWindows([]directiondomain.AbsenceWindow{{
			EmployeeID: actorID, Kind: "unavailable",
			StartsOn: day(2026, 8, 14), EndsOn: day(2026, 8, 14),
		}})

		resp, err := f.svc.ListPlan(ctx, orgID, actorID, "employee", &actorID, start, end)
		require.NoError(t, err)
		require.Len(t, resp.Rows, 1)
		require.Len(t, resp.Warnings, 1)
		assert.Equal(t, directiondomain.WarningAway, resp.Warnings[0].Type)
		assert.Equal(t, "Away 14 Aug", resp.Warnings[0].Message)
	})

	t.Run("manager reads the org-wide plan", func(t *testing.T) {
		f := setup(t)
		f.dirRepo.SetPlanRows([]directiondomain.PlanRow{{Direction: directiondomain.Direction{
			ID: uuid.New(), OrgID: orgID, DirectedBy: uuid.New(), DirectedTo: &otherID,
			ActivityID: uuid.New(), Status: directiondomain.StatusActive,
		}}})

		resp, err := f.svc.ListPlan(ctx, orgID, actorID, "manager", nil, start, end)
		require.NoError(t, err)
		require.Len(t, resp.Rows, 1)
	})
}

// ---------------------------------------------------------------------------
// TestService_Coverage — Task 3: scope resolution + gates + exclusions +
// totals (D-13-25/26/31)
// ---------------------------------------------------------------------------

func TestService_Coverage(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	empID := uuid.New()
	otherID := uuid.New()
	ctx := context.Background()
	start, end := day(2026, 8, 10), day(2026, 8, 21)

	t.Run("unknown scope is rejected with ErrInvalidRequest (D-13-25)", func(t *testing.T) {
		f := setup(t)
		_, err := f.svc.Coverage(ctx, orgID, actorID, "manager", "team", "x", start, end)
		require.ErrorIs(t, err, directiondomain.ErrInvalidRequest)
	})

	t.Run("unit scope is manager-only", func(t *testing.T) {
		f := setup(t)
		_, err := f.svc.Coverage(ctx, orgID, actorID, "employee", "unit", "u1", start, end)
		require.ErrorIs(t, err, directiondomain.ErrForbidden)

		resp, err := f.svc.Coverage(ctx, orgID, actorID, "manager", "unit", "u1", start, end)
		require.NoError(t, err, "unknown unit ids degrade to an empty employee set — never a 500")
		assert.Empty(t, resp.Rows)
	})

	t.Run("wg scope is manager-only", func(t *testing.T) {
		f := setup(t)
		_, err := f.svc.Coverage(ctx, orgID, actorID, "employee", "wg", "w1", start, end)
		require.ErrorIs(t, err, directiondomain.ErrForbidden)

		_, err = f.svc.Coverage(ctx, orgID, actorID, "manager", "wg", "w1", start, end)
		require.ErrorIs(t, err, directiondomain.ErrInvalidRequest, "unparseable scope ids must not 500")
	})

	t.Run("employee scope as non-manager requires scope_id == actorID (self-view)", func(t *testing.T) {
		f := setup(t)
		_, err := f.svc.Coverage(ctx, orgID, actorID, "employee", "employee", otherID.String(), start, end)
		require.ErrorIs(t, err, directiondomain.ErrForbidden)

		resp, err := f.svc.Coverage(ctx, orgID, actorID, "employee", "employee", actorID.String(), start, end)
		require.NoError(t, err)
		assert.Empty(t, resp.Rows)
		assert.Empty(t, resp.Totals)
	})

	t.Run("employee scope resolves to the one employee", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		var captured [][]uuid.UUID
		f.dirRepo.CoverageFn = func(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, ps, pe time.Time) ([]directiondomain.CoverageRow, error) {
			captured = append(captured, employeeIDs)
			return nil, nil
		}

		_, err := f.svc.Coverage(ctx, orgID, actorID, "manager", "employee", empID.String(), start, end)
		require.NoError(t, err)
		require.NotEmpty(t, captured)
		require.Equal(t, []uuid.UUID{empID}, captured[0])
	})

	t.Run("validity-outside employee gets the invalid warning and is dropped from the repo call (D-13-31)", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, ptrT(day(2026, 7, 31)))
		var captured [][]uuid.UUID
		f.dirRepo.CoverageFn = func(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, ps, pe time.Time) ([]directiondomain.CoverageRow, error) {
			captured = append(captured, employeeIDs)
			return nil, nil
		}

		resp, err := f.svc.Coverage(ctx, orgID, actorID, "manager", "employee", empID.String(), start, end)
		require.NoError(t, err)
		require.NotEmpty(t, captured)
		assert.NotContains(t, captured[0], empID, "the invalid employee must not reach the coverage repo call")
		require.Len(t, resp.Warnings, 1)
		assert.Equal(t, directiondomain.WarningInvalid, resp.Warnings[0].Type)
		assert.Equal(t, "Outside validity period", resp.Warnings[0].Message)
	})

	t.Run("unit scope aggregates the unit and descendant members (A6)", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		f.seedMembership(otherID, orgID, nil, nil, nil)
		unitID, descID := "unit-1", "unit-2"
		f.unitRepo.Descendants = map[string][]unit.Unit{unitID: {{ID: descID, OrgID: orgID, Name: "desc"}}}
		f.seedUnitMember(unitID, empID, true)
		f.seedUnitMember(descID, otherID, true)
		var captured [][]uuid.UUID
		f.dirRepo.CoverageFn = func(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, ps, pe time.Time) ([]directiondomain.CoverageRow, error) {
			captured = append(captured, employeeIDs)
			return nil, nil
		}

		_, err := f.svc.Coverage(ctx, orgID, actorID, "manager", "unit", unitID, start, end)
		require.NoError(t, err)
		require.NotEmpty(t, captured)
		require.ElementsMatch(t, []uuid.UUID{empID, otherID}, captured[0])
	})

	t.Run("wg scope aggregates the WG members", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		f.seedMembership(otherID, orgID, nil, nil, nil)
		wgID := uuid.New()
		f.seedWgMember(wgID, empID)
		f.seedWgMember(wgID, otherID)
		var captured [][]uuid.UUID
		f.dirRepo.CoverageFn = func(ctx context.Context, orgID uuid.UUID, employeeIDs []uuid.UUID, ps, pe time.Time) ([]directiondomain.CoverageRow, error) {
			captured = append(captured, employeeIDs)
			return nil, nil
		}

		_, err := f.svc.Coverage(ctx, orgID, actorID, "manager", "wg", wgID.String(), start, end)
		require.NoError(t, err)
		require.NotEmpty(t, captured)
		require.ElementsMatch(t, []uuid.UUID{empID, otherID}, captured[0])
	})

	t.Run("away day (capacity 0) is excluded from uncovered rows but the warning is present (D-13-26)", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		f.dirRepo.SetCoverageRows([]directiondomain.CoverageRow{
			{EmployeeID: empID, Date: day(2026, 8, 16), Capacity: 8, Planned: 6, Gap: 2},
			{EmployeeID: empID, Date: day(2026, 8, 17), Capacity: 0, Planned: 0, Gap: 0},
		})
		f.dirRepo.SetAbsenceWindows([]directiondomain.AbsenceWindow{{
			EmployeeID: empID, Kind: "unavailable",
			StartsOn: day(2026, 8, 17), EndsOn: day(2026, 8, 17),
		}})

		resp, err := f.svc.Coverage(ctx, orgID, actorID, "manager", "employee", empID.String(), start, end)
		require.NoError(t, err)
		require.Len(t, resp.Rows, 1, "the fully-absent day is excluded from uncovered surfacing")
		assert.Equal(t, day(2026, 8, 16), resp.Rows[0].Date)
		require.Len(t, resp.Warnings, 1)
		assert.Equal(t, directiondomain.WarningAway, resp.Warnings[0].Type)
		assert.Equal(t, "Away 17 Aug", resp.Warnings[0].Message)
	})

	t.Run("over-capacity day is surfaced with the warning and counted in totals", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		f.dirRepo.SetCoverageRows([]directiondomain.CoverageRow{
			{EmployeeID: empID, Date: day(2026, 8, 16), Capacity: 8, Planned: 6, Gap: 2},
			{EmployeeID: empID, Date: day(2026, 8, 17), Capacity: 8, Planned: 10, Gap: -2},
		})

		resp, err := f.svc.Coverage(ctx, orgID, actorID, "manager", "employee", empID.String(), start, end)
		require.NoError(t, err)
		require.Len(t, resp.Rows, 2)
		require.Len(t, resp.Warnings, 1)
		assert.Equal(t, directiondomain.WarningOverCapacity, resp.Warnings[0].Type)
		assert.Equal(t, "Over capacity 17 Aug", resp.Warnings[0].Message)
		require.Len(t, resp.Totals, 1)
		assert.Equal(t, empID, resp.Totals[0].EmployeeID)
		assert.Equal(t, 16.0, resp.Totals[0].Planned)
		assert.Equal(t, 16.0, resp.Totals[0].Capacity)
		assert.Equal(t, 0.0, resp.Totals[0].Gap)
	})

	t.Run("partial-day permit surfaces the partial warning", func(t *testing.T) {
		f := setup(t)
		f.seedMembership(empID, orgID, nil, nil, nil)
		hours := 4.0
		f.dirRepo.SetAbsenceWindows([]directiondomain.AbsenceWindow{{
			EmployeeID: empID, Kind: "permit",
			StartsOn: day(2026, 8, 14), EndsOn: day(2026, 8, 14), Hours: &hours,
		}})

		resp, err := f.svc.Coverage(ctx, orgID, actorID, "manager", "employee", empID.String(), start, end)
		require.NoError(t, err)
		require.Len(t, resp.Warnings, 1)
		assert.Equal(t, directiondomain.WarningPartial, resp.Warnings[0].Type)
		assert.Equal(t, "Partial 14 Aug", resp.Warnings[0].Message)
	})
}
