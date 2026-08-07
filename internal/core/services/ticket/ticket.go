package ticket

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// Service implements the ticket lifecycle business rules (ADR-P-003 rev,
// D-A/D-14/D-15, ADR-BE-016): the state machine with reopen and guarded
// dismissal, atomic triage, comments, and the immutable history stream.
//
// Permission gates (D-15, TICK-06): any employee creates; owner/assignee/
// manager+ update + comment + transition; manager|finance triage + dismiss;
// the customer role is rejected everywhere (internal-only, D-E). Every state
// change writes its audit row IN THE SAME TRANSACTION as the state write
// (Pitfall 2, ADR-BE-016) — the repo takes the audit row(s) to persist.
//
// The dismissal guard and the resolved-check read hours from time_entries
// server-side (LoggedHours / HasNonTerminalActivities) — never from the
// client (T-11-07).
type Service struct {
	repo         ports.TicketRepository
	activityRepo ports.ActivityRepository
	contractRepo ports.ContractRepository
	orgRepo      ports.OrganizationRepository
}

func NewService(ticketRepo ports.TicketRepository, activityRepo ports.ActivityRepository, contractRepo ports.ContractRepository, orgRepo ports.OrganizationRepository) *Service {
	return &Service{
		repo:         ticketRepo,
		activityRepo: activityRepo,
		contractRepo: contractRepo,
		orgRepo:      orgRepo,
	}
}

// CreateTicketRequest is the DTO for creating a ticket (TICK-01).
type CreateTicketRequest struct {
	Title       string
	Description string
	Kind        string
	AssigneeID  *uuid.UUID
}

// closedKindSet is the four-kind vocabulary (D-A, TICK-01) — a closed set,
// unlike the org-extensible activity_kinds catalog.
var closedKindSet = map[string]bool{
	ticketdomain.KindQuestion:  true,
	ticketdomain.KindBug:       true,
	ticketdomain.KindChange:    true,
	ticketdomain.KindEvolution: true,
}

// closedStatusSet is the seven-status vocabulary (D-A lifecycle).
var closedStatusSet = map[string]bool{
	ticketdomain.StatusOpen:       true,
	ticketdomain.StatusTriage:     true,
	ticketdomain.StatusPlanned:    true,
	ticketdomain.StatusInProgress: true,
	ticketdomain.StatusResolved:   true,
	ticketdomain.StatusClosed:     true,
	ticketdomain.StatusDismissed:  true,
}

func isValidKind(k string) bool  { return closedKindSet[k] }
func isValidStatus(s string) bool { return closedStatusSet[s] }

// Create gates per D-15 (any employee; customer rejected — T-11-04), then
// validates the closed-set kind (TICK-01) and the same-org assignee (D-02),
// and persists ticket + 'created' audit row atomically.
func (s *Service) Create(ctx context.Context, orgID, actorID uuid.UUID, role string, req *CreateTicketRequest) (*ticketdomain.Ticket, error) {
	if role == string(models.RoleCustomer) {
		return nil, ticketdomain.ErrForbidden
	}
	if req.Title == "" || !isValidKind(req.Kind) {
		return nil, ticketdomain.ErrInvalidRequest
	}
	if req.AssigneeID != nil && !s.isOrgMember(ctx, orgID, *req.AssigneeID) {
		return nil, ticketdomain.ErrInvalidRequest
	}

	now := time.Now().UTC()
	t := &ticketdomain.Ticket{
		ID:          uuid.New(),
		OrgID:       orgID,
		Title:       req.Title,
		Description: req.Description,
		Kind:        req.Kind,
		Status:      ticketdomain.StatusOpen,
		RequesterID: actorID,
		AssigneeID:  req.AssigneeID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	actor := actorID
	return s.repo.Create(ctx, orgID, t, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: "ticket",
		EntityID:   t.ID,
		Action:     "created",
		ActorID:    &actor,
		Payload:    map[string]any{"kind": req.Kind},
		CreatedAt:  now,
	})
}

// List returns the org's tickets with optional status/kind filters. Gate per
// D-15: all internal members; customer rejected. Filter values are validated
// against the closed vocabularies (TICK-01/TICK-02) else ErrInvalidRequest.
func (s *Service) List(ctx context.Context, orgID uuid.UUID, role, status, kind string) ([]ticketdomain.Ticket, error) {
	if role == string(models.RoleCustomer) {
		return nil, ticketdomain.ErrForbidden
	}
	if status != "" && !isValidStatus(status) {
		return nil, ticketdomain.ErrInvalidRequest
	}
	if kind != "" && !isValidKind(kind) {
		return nil, ticketdomain.ErrInvalidRequest
	}
	return s.repo.ListByOrg(ctx, orgID, status, kind)
}

// Get returns a ticket and its comments (detail read). Gate per D-15: all
// internal members; customer rejected.
func (s *Service) Get(ctx context.Context, orgID uuid.UUID, role string, ticketID uuid.UUID) (*ticketdomain.Ticket, []ticketdomain.TicketComment, error) {
	if role == string(models.RoleCustomer) {
		return nil, nil, ticketdomain.ErrForbidden
	}
	t, err := s.repo.Get(ctx, orgID, ticketID)
	if err != nil {
		return nil, nil, err
	}
	comments, err := s.repo.ListComments(ctx, orgID, ticketID)
	if err != nil {
		return nil, nil, err
	}
	return t, comments, nil
}

// UpdateDetails gates per D-15 (owner/assignee/manager+ — T-11-05), validates
// the same-org assignee (D-02), and persists the changed fields + 'updated'
// audit row atomically.
func (s *Service) UpdateDetails(ctx context.Context, orgID, actorID uuid.UUID, role string, ticketID uuid.UUID, title, description *string, assigneeID *uuid.UUID) (*ticketdomain.Ticket, error) {
	t, err := s.repo.Get(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if !s.canUpdate(role, actorID, t) {
		return nil, ticketdomain.ErrForbidden
	}
	if assigneeID != nil && !s.isOrgMember(ctx, orgID, *assigneeID) {
		return nil, ticketdomain.ErrInvalidRequest
	}

	payload := map[string]any{}
	if title != nil {
		payload["title"] = *title
	}
	if description != nil {
		payload["description"] = *description
	}
	if assigneeID != nil {
		payload["assignee_id"] = assigneeID.String()
	}

	actor := actorID
	return s.repo.UpdateDetails(ctx, orgID, ticketID, title, description, assigneeID, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: "ticket",
		EntityID:   ticketID,
		Action:     "updated",
		ActorID:    &actor,
		Payload:    payload,
		CreatedAt:  time.Now().UTC(),
	})
}

// Transition moves the ticket along the pinned matrix (D-14, ADR-BE-016).
// Gates: owner/assignee/manager+ (T-11-05); then CanTransition(current, to)
// else ErrInvalidTransition (TICK-02); the 'resolved' edge additionally
// requires every linked activity terminal (OQ2) else ErrActivityNotTerminal.
// The 'status_changed' audit row lands in the same tx as the state write
// (TICK-05).
func (s *Service) Transition(ctx context.Context, orgID, actorID uuid.UUID, role string, ticketID uuid.UUID, to string, note *string) (*ticketdomain.Ticket, error) {
	t, err := s.repo.Get(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if !s.canUpdate(role, actorID, t) {
		return nil, ticketdomain.ErrForbidden
	}
	if !ticketdomain.CanTransition(t.Status, to) {
		return nil, ticketdomain.ErrInvalidTransition
	}
	if to == ticketdomain.StatusResolved {
		nonTerminal, err := s.repo.HasNonTerminalActivities(ctx, ticketID)
		if err != nil {
			return nil, err
		}
		if nonTerminal {
			return nil, ticketdomain.ErrActivityNotTerminal
		}
	}

	payload := map[string]any{"from": t.Status, "to": to}
	if note != nil {
		payload["note"] = *note
	}
	actor := actorID
	return s.repo.UpdateState(ctx, orgID, ticketID, to, note, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: "ticket",
		EntityID:   ticketID,
		Action:     "status_changed",
		ActorID:    &actor,
		Payload:    payload,
		CreatedAt:  time.Now().UTC(),
	})
}

// canUpdate is the D-15 update/comment/transition gate: manager|finance, or
// the ticket owner (requester), or the assignee.
func (s *Service) canUpdate(role string, actorID uuid.UUID, t *ticketdomain.Ticket) bool {
	if role == string(models.RoleManager) || role == string(models.RoleFinance) {
		return true
	}
	return t.IsOwner(actorID) || t.IsAssignee(actorID)
}

// isOrgMember reports whether the user has an active membership row in the
// org (D-02 — origin refs must be org members).
func (s *Service) isOrgMember(ctx context.Context, orgID, userID uuid.UUID) bool {
	m, err := s.orgRepo.GetMembership(ctx, userID, orgID)
	if err != nil || m == nil {
		return false
	}
	return m.IsActive
}
