package testdata

import (
	"context"
	"sync"

	"github.com/google/uuid"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// MockTicketRepo implements ports.TicketRepository for service unit tests.
// Get returns the configured ticket for the exact (orgID, ticketID) pair;
// a missing pair — or a cross-org id — surfaces ticket.ErrTicketNotFound
// (same-org semantics, D-02). Override with GetFn when a test needs a
// non-derived answer.
type MockTicketRepo struct {
	mu      sync.Mutex
	Tickets map[uuid.UUID]*ticketdomain.Ticket
	GetFn   func(ctx context.Context, orgID, ticketID uuid.UUID) (*ticketdomain.Ticket, error)
}

var _ ports.TicketRepository = (*MockTicketRepo)(nil)

func (m *MockTicketRepo) Get(ctx context.Context, orgID, ticketID uuid.UUID) (*ticketdomain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetFn != nil {
		return m.GetFn(ctx, orgID, ticketID)
	}
	t, ok := m.Tickets[ticketID]
	if !ok || t.OrgID != orgID {
		return nil, ticketdomain.ErrTicketNotFound
	}
	return t, nil
}
