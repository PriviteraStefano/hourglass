package activity

import (
	"context"
	"testing"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// originFixture exposes every DI slot so origin tests can seed memberships,
// tickets, and the routing repos directly.
type originFixture struct {
	svc          *Service
	activityRepo *testdata.MockActivityRepo
	orgRepo      *testdata.MockOrgRepo
	ticketRepo   *testdata.MockTicketRepo
	unitRepo     *testdata.MockUnitRepo
	wgRepo       *testdata.MockWorkingGroupRepo
	routingSvc    *routing.Service
}

func setupOrigin(t *testing.T) *originFixture {
	t.Helper()
	f := &originFixture{
		activityRepo: &testdata.MockActivityRepo{},
		orgRepo:      &testdata.MockOrgRepo{},
		ticketRepo:   &testdata.MockTicketRepo{},
		unitRepo:     &testdata.MockUnitRepo{},
		wgRepo:       &testdata.MockWorkingGroupRepo{},
	}
	f.routingSvc = routing.NewService(f.wgRepo, f.activityRepo, f.unitRepo)
	f.svc = NewService(f.activityRepo, &testdata.MockContractRepo{}, f.unitRepo, f.orgRepo, f.ticketRepo, f.routingSvc)
	return f
}

// seedMembership adds an active org membership for the user (D-02).
func (f *originFixture) seedMembership(orgID, userID uuid.UUID) {
	if f.orgRepo.Memberships == nil {
		f.orgRepo.Memberships = make(map[string]*auth.OrganizationMembership)
	}
	f.orgRepo.Memberships[userID.String()+":"+orgID.String()] = &auth.OrganizationMembership{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		Role:           string(models.RoleEmployee),
		IsActive:       true,
	}
}

// seedTicket adds a ticket to the ticket mock.
func (f *originFixture) seedTicket(orgID uuid.UUID, status string) *ticketdomain.Ticket {
	t := &ticketdomain.Ticket{
		ID:          uuid.New(),
		OrgID:       orgID,
		Title:       "Test ticket",
		Kind:        ticketdomain.KindBug,
		Status:      status,
		RequesterID: uuid.New(),
	}
	if f.ticketRepo.Tickets == nil {
		f.ticketRepo.Tickets = make(map[uuid.UUID]*ticketdomain.Ticket)
	}
	f.ticketRepo.Tickets[t.ID] = t
	return t
}

// ---------------------------------------------------------------------------
// TestActivityOrigin_ManagerAssignment
// ---------------------------------------------------------------------------

func TestActivityOrigin_ManagerAssignment(t *testing.T) {
	orgID := uuid.New()
	manager, assignee := uuid.New(), uuid.New()

	t.Run("non-manager role forbidden", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		ot := activitydomain.OriginTypeManagerAssignment

		req := validCreateReq()
		req.OriginType = &ot
		req.AssignedBy = &manager
		req.AssignedTo = &assignee
		created, err := f.svc.Create(context.Background(), string(models.RoleEmployee), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, created)
	})

	t.Run("missing refs rejected", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		ot := activitydomain.OriginTypeManagerAssignment

		req := validCreateReq()
		req.OriginType = &ot
		req.AssignedBy = &manager
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("ref not an org member rejected (D-02)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		// only the assignee has a membership; assigned_by does not
		f.seedMembership(orgID, assignee)
		ot := activitydomain.OriginTypeManagerAssignment

		req := validCreateReq()
		req.OriginType = &ot
		req.AssignedBy = &manager
		req.AssignedTo = &assignee
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("manager role with both members passes", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		f.seedMembership(orgID, manager)
		f.seedMembership(orgID, assignee)
		ot := activitydomain.OriginTypeManagerAssignment

		req := validCreateReq()
		req.OriginType = &ot
		req.AssignedBy = &manager
		req.AssignedTo = &assignee
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.NotNil(t, created.OriginType)
		assert.Equal(t, activitydomain.OriginTypeManagerAssignment, *created.OriginType)
		require.NotNil(t, created.AssignedBy)
		assert.Equal(t, manager, *created.AssignedBy)
		require.NotNil(t, created.AssignedTo)
		assert.Equal(t, assignee, *created.AssignedTo)
	})
}

// ---------------------------------------------------------------------------
// TestActivityOrigin_EmployeeProposal
// ---------------------------------------------------------------------------

func TestActivityOrigin_EmployeeProposal(t *testing.T) {
	orgID := uuid.New()
	employee := uuid.New()

	t.Run("proposed_by required", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		ot := activitydomain.OriginTypeEmployeeProposal

		req := validCreateReq()
		req.OriginType = &ot
		created, err := f.svc.Create(context.Background(), string(models.RoleEmployee), orgID, employee, req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("proposed_by != actor forbidden (spoofing guard, D-04)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		ot := activitydomain.OriginTypeEmployeeProposal
		otherUser := uuid.New()

		req := validCreateReq()
		req.OriginType = &ot
		req.ProposedBy = &otherUser
		created, err := f.svc.Create(context.Background(), string(models.RoleEmployee), orgID, employee, req)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, created)
	})

	t.Run("proposal forces is_active=false (D-12)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		ot := activitydomain.OriginTypeEmployeeProposal

		req := validCreateReq()
		req.OriginType = &ot
		req.ProposedBy = &employee
		created, err := f.svc.Create(context.Background(), string(models.RoleEmployee), orgID, employee, req)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.False(t, created.IsActive)
		require.NotNil(t, created.ProposedBy)
		assert.Equal(t, employee, *created.ProposedBy)
	})

	t.Run("explicit false is_active without proposal origin rejected (D-12)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}

		req := validCreateReq()
		falseVal := false
		req.IsActive = &falseVal
		created, err := f.svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})
}

// ---------------------------------------------------------------------------
// TestActivityOrigin_CustomerTicket
// ---------------------------------------------------------------------------

func TestActivityOrigin_CustomerTicket(t *testing.T) {
	orgID := uuid.New()

	t.Run("non-manager role forbidden", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		tk := f.seedTicket(orgID, ticketdomain.StatusOpen)
		ot := activitydomain.OriginTypeCustomerTicket

		req := validCreateReq()
		req.OriginType = &ot
		req.TicketID = &tk.ID
		created, err := f.svc.Create(context.Background(), string(models.RoleEmployee), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrForbidden)
		assert.Nil(t, created)
	})

	t.Run("missing ticket_id rejected", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		ot := activitydomain.OriginTypeCustomerTicket

		req := validCreateReq()
		req.OriginType = &ot
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("unknown ticket rejected (same-org, D-02)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		ot := activitydomain.OriginTypeCustomerTicket

		req := validCreateReq()
		req.OriginType = &ot
		req.TicketID = ptr(uuid.New())
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("cross-org ticket rejected (D-02)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		tk := f.seedTicket(uuid.New(), ticketdomain.StatusOpen) // different org
		ot := activitydomain.OriginTypeCustomerTicket

		req := validCreateReq()
		req.OriginType = &ot
		req.TicketID = &tk.ID
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("planned ticket rejected (state precondition, OQ5/ADR-BE-016)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		tk := f.seedTicket(orgID, ticketdomain.StatusPlanned)
		ot := activitydomain.OriginTypeCustomerTicket

		req := validCreateReq()
		req.OriginType = &ot
		req.TicketID = &tk.ID
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("dismissed ticket rejected (state precondition)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		tk := f.seedTicket(orgID, ticketdomain.StatusDismissed)
		ot := activitydomain.OriginTypeCustomerTicket

		req := validCreateReq()
		req.OriginType = &ot
		req.TicketID = &tk.ID
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("open ticket passes state check", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		tk := f.seedTicket(orgID, ticketdomain.StatusOpen)
		ot := activitydomain.OriginTypeCustomerTicket

		req := validCreateReq()
		req.OriginType = &ot
		req.TicketID = &tk.ID
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.NotNil(t, created.TicketID)
		assert.Equal(t, tk.ID, *created.TicketID)
	})

	t.Run("triage ticket passes state check", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		tk := f.seedTicket(orgID, ticketdomain.StatusTriage)
		ot := activitydomain.OriginTypeCustomerTicket

		req := validCreateReq()
		req.OriginType = &ot
		req.TicketID = &tk.ID
		created, err := f.svc.Create(context.Background(), string(models.RoleFinance), orgID, uuid.New(), req)
		require.NoError(t, err)
		require.NotNil(t, created)
	})
}

// ---------------------------------------------------------------------------
// TestActivityOrigin_ReviewedByAndUnknownType
// ---------------------------------------------------------------------------

func TestActivityOrigin_ReviewedByAndUnknownType(t *testing.T) {
	orgID := uuid.New()

	t.Run("reviewed_by on any create rejected (ADR-P-013)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		ot := activitydomain.OriginTypeEmployeeProposal
		employee := uuid.New()

		req := validCreateReq()
		req.OriginType = &ot
		req.ProposedBy = &employee
		req.ReviewedBy = ptr(uuid.New())
		created, err := f.svc.Create(context.Background(), string(models.RoleEmployee), orgID, employee, req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("unknown origin type rejected (closed set)", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}
		unknown := "alien_origin"

		req := validCreateReq()
		req.OriginType = &unknown
		created, err := f.svc.Create(context.Background(), string(models.RoleManager), orgID, uuid.New(), req)
		assert.ErrorIs(t, err, activitydomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("legacy create without origin still valid", func(t *testing.T) {
		f := setupOrigin(t)
		f.activityRepo.Kinds = map[string]bool{orgID.String() + ":engagement": true}

		created, err := f.svc.Create(context.Background(), string(models.RoleEmployee), orgID, uuid.New(), validCreateReq())
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Nil(t, created.OriginType)
	})
}

// ---------------------------------------------------------------------------
// TestActivityOrigin_UpdateImmutability
// ---------------------------------------------------------------------------

func TestActivityOrigin_UpdateImmutability(t *testing.T) {
	orgID := uuid.New()

	seedOwned := func(f *originFixture) uuid.UUID {
		a := testdata.NewActivity(func(a *activitydomain.ActivityResponse) {
			a.OrgID = orgID
			a.CreatedByOrgID = orgID
		})
		if f.activityRepo.Activities == nil {
			f.activityRepo.Activities = make(map[uuid.UUID]*activitydomain.ActivityResponse)
		}
		f.activityRepo.Activities[a.ID] = &a
		return a.ID
	}

	for _, tc := range []struct {
		name  string
		mut   func(*activitydomain.UpdateActivityRequest)
	}{
		{"origin_type", func(r *activitydomain.UpdateActivityRequest) { r.OriginType = ptr(activitydomain.OriginTypeManagerAssignment) }},
		{"assigned_by", func(r *activitydomain.UpdateActivityRequest) { r.AssignedBy = ptr(uuid.New()) }},
		{"assigned_to", func(r *activitydomain.UpdateActivityRequest) { r.AssignedTo = ptr(uuid.New()) }},
		{"proposed_by", func(r *activitydomain.UpdateActivityRequest) { r.ProposedBy = ptr(uuid.New()) }},
		{"reviewed_by", func(r *activitydomain.UpdateActivityRequest) { r.ReviewedBy = ptr(uuid.New()) }},
		{"ticket_id", func(r *activitydomain.UpdateActivityRequest) { r.TicketID = ptr(uuid.New()) }},
	} {
		t.Run("update with "+tc.name+" rejected (D-03)", func(t *testing.T) {
			f := setupOrigin(t)
			id := seedOwned(f)
			req := &activitydomain.UpdateActivityRequest{Name: "Renamed"}
			tc.mut(req)

			updated, err := f.svc.Update(context.Background(), string(models.RoleFinance), orgID, id, req)
			assert.ErrorIs(t, err, activitydomain.ErrOriginImmutable)
			assert.Nil(t, updated)
		})
	}

	t.Run("plain update still works", func(t *testing.T) {
		f := setupOrigin(t)
		id := seedOwned(f)

		updated, err := f.svc.Update(context.Background(), string(models.RoleFinance), orgID, id, &activitydomain.UpdateActivityRequest{Name: "Renamed"})
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "Renamed", updated.Name)
	})
}
