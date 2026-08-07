package testdata

import (
	"context"
	"sync"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// MockTicketRepo implements ports.TicketRepository for service unit tests.
// Get returns the configured ticket for the exact (orgID, ticketID) pair;
// a missing pair — or a cross-org id — surfaces ticket.ErrTicketNotFound
// (same-org semantics, D-02). Override with GetFn when a test needs a
// non-derived answer.
//
// Mutators record the calls (and the audit rows they were handed) so tests
// can assert both the state change and the in-tx audit payload without a DB.
type MockTicketRepo struct {
	mu      sync.Mutex
	Tickets map[uuid.UUID]*ticketdomain.Ticket
	GetFn   func(ctx context.Context, orgID, ticketID uuid.UUID) (*ticketdomain.Ticket, error)

	// Audit capture: every audit row passed to a mutator lands here.
	Audits []*audit.AuditLog

	// Behavior knobs for the read-only guards.
	LoggedHoursResult        float64
	HasNonTerminalResult     bool
	TriageActivities         []*activitydomain.ActivityResponse
	LoggedHoursFn            func(ctx context.Context, ticketID uuid.UUID) (float64, error)
	HasNonTerminalActivitiesFn func(ctx context.Context, ticketID uuid.UUID) (bool, error)
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

func (m *MockTicketRepo) Create(ctx context.Context, orgID uuid.UUID, t *ticketdomain.Ticket, a *audit.AuditLog) (*ticketdomain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Tickets == nil {
		m.Tickets = make(map[uuid.UUID]*ticketdomain.Ticket)
	}
	m.Tickets[t.ID] = t
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return t, nil
}

func (m *MockTicketRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, status, kind string) ([]ticketdomain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []ticketdomain.Ticket
	for _, t := range m.Tickets {
		if t.OrgID != orgID {
			continue
		}
		if status != "" && t.Status != status {
			continue
		}
		if kind != "" && t.Kind != kind {
			continue
		}
		out = append(out, *t)
	}
	return out, nil
}

func (m *MockTicketRepo) UpdateDetails(ctx context.Context, orgID, ticketID uuid.UUID, title, description *string, assigneeID *uuid.UUID, a *audit.AuditLog) (*ticketdomain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.Tickets[ticketID]
	if !ok || t.OrgID != orgID {
		return nil, ticketdomain.ErrTicketNotFound
	}
	if title != nil {
		t.Title = *title
	}
	if description != nil {
		t.Description = *description
	}
	if assigneeID != nil {
		t.AssigneeID = assigneeID
	}
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return t, nil
}

func (m *MockTicketRepo) UpdateState(ctx context.Context, orgID, ticketID uuid.UUID, to string, note *string, a *audit.AuditLog) (*ticketdomain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.Tickets[ticketID]
	if !ok || t.OrgID != orgID {
		return nil, ticketdomain.ErrTicketNotFound
	}
	t.Status = to
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return t, nil
}

func (m *MockTicketRepo) Dismiss(ctx context.Context, orgID, ticketID uuid.UUID, hours float64, a *audit.AuditLog) (*ticketdomain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.Tickets[ticketID]
	if !ok || t.OrgID != orgID {
		return nil, ticketdomain.ErrTicketNotFound
	}
	t.Status = ticketdomain.StatusDismissed
	t.DismissedHours = &hours
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return t, nil
}

func (m *MockTicketRepo) Triage(ctx context.Context, orgID, ticketID uuid.UUID, kind *string, plans []*activitydomain.CreateActivityRequest, audits []*audit.AuditLog) (*ticketdomain.Ticket, []*activitydomain.ActivityResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.Tickets[ticketID]
	if !ok || t.OrgID != orgID {
		return nil, nil, ticketdomain.ErrTicketNotFound
	}
	t.Status = ticketdomain.StatusPlanned
	if kind != nil {
		t.Kind = *kind
	}
	for _, a := range audits {
		if a != nil {
			m.Audits = append(m.Audits, a)
		}
	}
	return t, m.TriageActivities, nil
}

func (m *MockTicketRepo) AddComment(ctx context.Context, orgID, ticketID uuid.UUID, c *ticketdomain.TicketComment, a *audit.AuditLog) (*ticketdomain.TicketComment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.Tickets[ticketID]
	if !ok || t.OrgID != orgID {
		return nil, ticketdomain.ErrTicketNotFound
	}
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return c, nil
}

func (m *MockTicketRepo) ListComments(ctx context.Context, orgID, ticketID uuid.UUID) ([]ticketdomain.TicketComment, error) {
	return nil, nil
}

func (m *MockTicketRepo) ListHistory(ctx context.Context, orgID, ticketID uuid.UUID) ([]audit.AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []audit.AuditLog
	for _, a := range m.Audits {
		if a.EntityID == ticketID && a.OrgID == orgID {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (m *MockTicketRepo) LoggedHours(ctx context.Context, ticketID uuid.UUID) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.LoggedHoursFn != nil {
		return m.LoggedHoursFn(ctx, ticketID)
	}
	return m.LoggedHoursResult, nil
}

func (m *MockTicketRepo) HasNonTerminalActivities(ctx context.Context, ticketID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.HasNonTerminalActivitiesFn != nil {
		return m.HasNonTerminalActivitiesFn(ctx, ticketID)
	}
	return m.HasNonTerminalResult, nil
}
