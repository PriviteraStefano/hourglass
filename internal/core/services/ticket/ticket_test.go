package ticket

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ticketFixture wires the service with testdata mocks so every DI slot is
// visible to the tests.
type ticketFixture struct {
	svc        *Service
	ticketRepo *testdata.MockTicketRepo
	orgRepo    *testdata.MockOrgRepo
}

func setupTicket(t *testing.T) *ticketFixture {
	t.Helper()
	f := &ticketFixture{
		ticketRepo: &testdata.MockTicketRepo{},
		orgRepo:    &testdata.MockOrgRepo{},
	}
	f.svc = NewService(f.ticketRepo, &testdata.MockActivityRepo{}, &testdata.MockContractRepo{}, f.orgRepo)
	return f
}

// seedMembership adds an active org membership for the user (D-02).
func (f *ticketFixture) seedMembership(orgID, userID uuid.UUID) {
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
func (f *ticketFixture) seedTicket(orgID uuid.UUID, status string) *ticketdomain.Ticket {
	t := &ticketdomain.Ticket{
		ID:          uuid.New(),
		OrgID:       orgID,
		Title:       "Seeded ticket",
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
// TestTicketCreate — TICK-01, D-15 create gate
// ---------------------------------------------------------------------------

func TestTicketCreate(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()

	t.Run("customer role forbidden (T-11-04)", func(t *testing.T) {
		f := setupTicket(t)
		created, err := f.svc.Create(context.Background(), orgID, actorID, string(models.RoleCustomer),
			&CreateTicketRequest{Title: "Hi", Kind: ticketdomain.KindBug})
		assert.ErrorIs(t, err, ticketdomain.ErrForbidden)
		assert.Nil(t, created)
	})

	t.Run("empty title rejected (TICK-01)", func(t *testing.T) {
		f := setupTicket(t)
		created, err := f.svc.Create(context.Background(), orgID, actorID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "", Kind: ticketdomain.KindBug})
		assert.ErrorIs(t, err, ticketdomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("kind outside the closed set rejected (TICK-01)", func(t *testing.T) {
		f := setupTicket(t)
		created, err := f.svc.Create(context.Background(), orgID, actorID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Hi", Kind: "feature"})
		assert.ErrorIs(t, err, ticketdomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("assignee must be an org member (D-02)", func(t *testing.T) {
		f := setupTicket(t)
		outsider := uuid.New()
		created, err := f.svc.Create(context.Background(), orgID, actorID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Hi", Kind: ticketdomain.KindBug, AssigneeID: &outsider})
		assert.ErrorIs(t, err, ticketdomain.ErrInvalidRequest)
		assert.Nil(t, created)
	})

	t.Run("employee creates open ticket with 'created' audit (D-15, TICK-05)", func(t *testing.T) {
		f := setupTicket(t)
		assignee := uuid.New()
		f.seedMembership(orgID, assignee)

		created, err := f.svc.Create(context.Background(), orgID, actorID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Billing bug", Description: "Off by a cent", Kind: ticketdomain.KindBug, AssigneeID: &assignee})
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, ticketdomain.StatusOpen, created.Status)
		assert.Equal(t, actorID, created.RequesterID)
		require.NotNil(t, created.AssigneeID)
		assert.Equal(t, assignee, *created.AssigneeID)

		require.Len(t, f.ticketRepo.Audits, 1)
		assert.Equal(t, "created", f.ticketRepo.Audits[0].Action)
		assert.Equal(t, "ticket", f.ticketRepo.Audits[0].EntityType)
		assert.Equal(t, created.ID, f.ticketRepo.Audits[0].EntityID)
		require.NotNil(t, f.ticketRepo.Audits[0].ActorID)
		assert.Equal(t, actorID, *f.ticketRepo.Audits[0].ActorID)
		assert.Equal(t, ticketdomain.KindBug, f.ticketRepo.Audits[0].Payload["kind"])
	})

	t.Run("all four kinds accepted (TICK-01)", func(t *testing.T) {
		for _, kind := range []string{
			ticketdomain.KindQuestion,
			ticketdomain.KindBug,
			ticketdomain.KindChange,
			ticketdomain.KindEvolution,
		} {
			f := setupTicket(t)
			created, err := f.svc.Create(context.Background(), orgID, actorID, string(models.RoleEmployee),
				&CreateTicketRequest{Title: "Kind " + kind, Kind: kind})
			require.NoError(t, err)
			assert.Equal(t, kind, created.Kind)
		}
	})

	t.Run("manager and finance can create too (D-15)", func(t *testing.T) {
		for _, role := range []string{string(models.RoleManager), string(models.RoleFinance)} {
			f := setupTicket(t)
			created, err := f.svc.Create(context.Background(), orgID, actorID, role,
				&CreateTicketRequest{Title: "Hi", Kind: ticketdomain.KindChange})
			require.NoError(t, err)
			assert.NotNil(t, created)
		}
	})
}

// ---------------------------------------------------------------------------
// TestTicketLifecycle — TICK-02 state machine, reopen, resolved block,
// permission gates
// ---------------------------------------------------------------------------

func TestTicketLifecycle(t *testing.T) {
	orgID := uuid.New()
	owner := uuid.New()

	t.Run("full happy path open→triage→planned→in_progress→resolved→closed", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)
		tkt.RequesterID = owner

		steps := []struct {
			to   string
			note *string
		}{
			{ticketdomain.StatusTriage, nil},
			{ticketdomain.StatusPlanned, nil},
			{ticketdomain.StatusInProgress, nil},
			{ticketdomain.StatusResolved, strPtr("done")},
			{ticketdomain.StatusClosed, nil},
		}
		for _, step := range steps {
			got, err := f.svc.Transition(context.Background(), orgID, owner, string(models.RoleEmployee), tkt.ID, step.to, step.note)
			require.NoError(t, err)
			assert.Equal(t, step.to, got.Status)
		}

		// Each transition wrote a 'status_changed' audit row with from/to.
		require.Len(t, f.ticketRepo.Audits, len(steps))
		for i, a := range f.ticketRepo.Audits {
			assert.Equal(t, "status_changed", a.Action)
			assert.Equal(t, steps[i].to, a.Payload["to"])
		}
	})

	t.Run("reopen resolved→in_progress allowed (D-A)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusResolved)
		tkt.RequesterID = owner

		got, err := f.svc.Transition(context.Background(), orgID, owner, string(models.RoleEmployee), tkt.ID, ticketdomain.StatusInProgress, nil)
		require.NoError(t, err)
		assert.Equal(t, ticketdomain.StatusInProgress, got.Status)
	})

	t.Run("invalid edge rejected with ErrInvalidTransition (D-14)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)
		tkt.RequesterID = owner

		// open → planned is not in the pinned matrix (must pass through triage).
		_, err := f.svc.Transition(context.Background(), orgID, owner, string(models.RoleEmployee), tkt.ID, ticketdomain.StatusPlanned, nil)
		assert.ErrorIs(t, err, ticketdomain.ErrInvalidTransition)

		// open → closed is not in the matrix either.
		_, err = f.svc.Transition(context.Background(), orgID, owner, string(models.RoleEmployee), tkt.ID, ticketdomain.StatusClosed, nil)
		assert.ErrorIs(t, err, ticketdomain.ErrInvalidTransition)

		// terminal states never transition.
		f2 := setupTicket(t)
		closed := f2.seedTicket(orgID, ticketdomain.StatusClosed)
		closed.RequesterID = owner
		_, err = f2.svc.Transition(context.Background(), orgID, owner, string(models.RoleEmployee), closed.ID, ticketdomain.StatusOpen, nil)
		assert.ErrorIs(t, err, ticketdomain.ErrInvalidTransition)
	})

	t.Run("resolved blocked while activities non-terminal (OQ2)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusInProgress)
		tkt.RequesterID = owner
		f.ticketRepo.HasNonTerminalResult = true

		_, err := f.svc.Transition(context.Background(), orgID, owner, string(models.RoleEmployee), tkt.ID, ticketdomain.StatusResolved, nil)
		assert.ErrorIs(t, err, ticketdomain.ErrActivityNotTerminal)
	})

	t.Run("resolved allowed when subtree terminal", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusInProgress)
		tkt.RequesterID = owner
		f.ticketRepo.HasNonTerminalResult = false

		got, err := f.svc.Transition(context.Background(), orgID, owner, string(models.RoleEmployee), tkt.ID, ticketdomain.StatusResolved, nil)
		require.NoError(t, err)
		assert.Equal(t, ticketdomain.StatusResolved, got.Status)
	})

	t.Run("employee non-owner non-assignee transition forbidden (T-11-05)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)
		tkt.RequesterID = uuid.New() // someone else

		_, err := f.svc.Transition(context.Background(), orgID, uuid.New(), string(models.RoleEmployee), tkt.ID, ticketdomain.StatusTriage, nil)
		assert.ErrorIs(t, err, ticketdomain.ErrForbidden)
	})

	t.Run("assignee can transition (D-15)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)
		tkt.RequesterID = uuid.New()
		assignee := uuid.New()
		tkt.AssigneeID = &assignee

		got, err := f.svc.Transition(context.Background(), orgID, assignee, string(models.RoleEmployee), tkt.ID, ticketdomain.StatusTriage, nil)
		require.NoError(t, err)
		assert.Equal(t, ticketdomain.StatusTriage, got.Status)
	})

	t.Run("manager can transition any ticket (D-15)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)

		got, err := f.svc.Transition(context.Background(), orgID, uuid.New(), string(models.RoleManager), tkt.ID, ticketdomain.StatusTriage, nil)
		require.NoError(t, err)
		assert.Equal(t, ticketdomain.StatusTriage, got.Status)
	})

	t.Run("customer cannot transition (internal-only, D-E)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)

		_, err := f.svc.Transition(context.Background(), orgID, uuid.New(), string(models.RoleCustomer), tkt.ID, ticketdomain.StatusTriage, nil)
		assert.ErrorIs(t, err, ticketdomain.ErrForbidden)
	})

	t.Run("missing ticket surfaces ErrTicketNotFound", func(t *testing.T) {
		f := setupTicket(t)
		_, err := f.svc.Transition(context.Background(), orgID, owner, string(models.RoleManager), uuid.New(), ticketdomain.StatusTriage, nil)
		assert.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)
	})
}

// ---------------------------------------------------------------------------
// TestTicketUpdateDetails — D-15 update gate
// ---------------------------------------------------------------------------

func TestTicketUpdateDetails(t *testing.T) {
	orgID := uuid.New()

	t.Run("owner updates title + description with 'updated' audit", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)
		tkt.RequesterID = owner()

		got, err := f.svc.UpdateDetails(context.Background(), orgID, tkt.RequesterID, string(models.RoleEmployee),
			tkt.ID, strPtr("New title"), strPtr("New description"), nil)
		require.NoError(t, err)
		assert.Equal(t, "New title", got.Title)
		assert.Equal(t, "New description", got.Description)

		require.Len(t, f.ticketRepo.Audits, 1)
		a := f.ticketRepo.Audits[0]
		assert.Equal(t, "updated", a.Action)
		assert.Equal(t, "New title", a.Payload["title"])
		assert.Equal(t, "New description", a.Payload["description"])
	})

	t.Run("non-owner employee forbidden (T-11-05)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)
		tkt.RequesterID = uuid.New()

		_, err := f.svc.UpdateDetails(context.Background(), orgID, uuid.New(), string(models.RoleEmployee),
			tkt.ID, strPtr("Hijack"), nil, nil)
		assert.ErrorIs(t, err, ticketdomain.ErrForbidden)
	})

	t.Run("finance can update any ticket (D-15)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)

		got, err := f.svc.UpdateDetails(context.Background(), orgID, uuid.New(), string(models.RoleFinance),
			tkt.ID, strPtr("By finance"), nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "By finance", got.Title)
	})

	t.Run("assignee change requires same-org member (D-02)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)
		tkt.RequesterID = owner()

		outsider := uuid.New()
		_, err := f.svc.UpdateDetails(context.Background(), orgID, tkt.RequesterID, string(models.RoleEmployee),
			tkt.ID, nil, nil, &outsider)
		assert.ErrorIs(t, err, ticketdomain.ErrInvalidRequest)

		member := uuid.New()
		f.seedMembership(orgID, member)
		got, err := f.svc.UpdateDetails(context.Background(), orgID, tkt.RequesterID, string(models.RoleEmployee),
			tkt.ID, nil, nil, &member)
		require.NoError(t, err)
		require.NotNil(t, got.AssigneeID)
		assert.Equal(t, member, *got.AssigneeID)
	})
}

// ---------------------------------------------------------------------------
// TestTicketListGet — D-15 view gates + filter vocabulary
// ---------------------------------------------------------------------------

func TestTicketListGet(t *testing.T) {
	orgID := uuid.New()

	t.Run("customer cannot list or get (D-E)", func(t *testing.T) {
		f := setupTicket(t)
		_, err := f.svc.List(context.Background(), orgID, string(models.RoleCustomer), "", "")
		assert.ErrorIs(t, err, ticketdomain.ErrForbidden)

		_, _, err = f.svc.Get(context.Background(), orgID, string(models.RoleCustomer), uuid.New())
		assert.ErrorIs(t, err, ticketdomain.ErrForbidden)
	})

	t.Run("unknown status/kind filters rejected (TICK-01/TICK-02)", func(t *testing.T) {
		f := setupTicket(t)
		_, err := f.svc.List(context.Background(), orgID, string(models.RoleEmployee), "bogus", "")
		assert.ErrorIs(t, err, ticketdomain.ErrInvalidRequest)

		_, err = f.svc.List(context.Background(), orgID, string(models.RoleEmployee), "", "bogus")
		assert.ErrorIs(t, err, ticketdomain.ErrInvalidRequest)
	})

	t.Run("list returns org tickets with filters", func(t *testing.T) {
		f := setupTicket(t)
		openBug := f.seedTicket(orgID, ticketdomain.StatusOpen)
		openBug.Kind = ticketdomain.KindBug
		otherOrg := uuid.New()
		f.seedTicket(otherOrg, ticketdomain.StatusOpen)
		closedChange := f.seedTicket(orgID, ticketdomain.StatusClosed)
		closedChange.Kind = ticketdomain.KindChange

		all, err := f.svc.List(context.Background(), orgID, string(models.RoleEmployee), "", "")
		require.NoError(t, err)
		assert.Len(t, all, 2)

		onlyOpen, err := f.svc.List(context.Background(), orgID, string(models.RoleEmployee), ticketdomain.StatusOpen, "")
		require.NoError(t, err)
		assert.Len(t, onlyOpen, 1)
		assert.Equal(t, ticketdomain.StatusOpen, onlyOpen[0].Status)

		onlyBugs, err := f.svc.List(context.Background(), orgID, string(models.RoleEmployee), "", ticketdomain.KindBug)
		require.NoError(t, err)
		assert.Len(t, onlyBugs, 1)
	})

	t.Run("get returns ticket + comments (detail)", func(t *testing.T) {
		f := setupTicket(t)
		tkt := f.seedTicket(orgID, ticketdomain.StatusOpen)

		got, comments, err := f.svc.Get(context.Background(), orgID, string(models.RoleEmployee), tkt.ID)
		require.NoError(t, err)
		assert.Equal(t, tkt.ID, got.ID)
		assert.Empty(t, comments)
	})
}

// helper: owner returns a fresh uuid (avoids shadowing the loop var).
func owner() uuid.UUID { return uuid.New() }

// helper: strPtr returns a pointer to s for optional-field tests.
func strPtr(s string) *string { return &s }
