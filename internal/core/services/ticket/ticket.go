package ticket

import (
	"context"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
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
	// Title length mirrors the VARCHAR(255) column (migration 014): an
	// oversized title would otherwise surface as a 500 from the DB (WR-04).
	if req.Title == "" || len(req.Title) > 255 || !isValidKind(req.Kind) {
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
		// Empty titles (IN-01) and titles beyond the VARCHAR(255) column
		// (WR-04) are rejected here — the payload map and repo call must not
		// execute for invalid titles.
		if *title == "" || len(*title) > 255 {
			return nil, ticketdomain.ErrInvalidRequest
		}
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
//
// The pool-level CanTransition / HasNonTerminalActivities checks here are
// FAST-FAIL UX only (Pitfall 7, CR-01): the repo re-validates the matrix and
// the resolved-block authoritatively inside the mutator tx under the
// FOR UPDATE ticket row lock — never check-then-act across the tx boundary.
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
	// The matrix pins open→dismissed / triage→dismissed, but those edges are
	// consumed ONLY by Dismiss — the guarded path (D-11 gate + D-13 hours
	// guard + dismissed_hours snapshot, T-11-07). Allowing them here would
	// let an owner/assignee bypass the guard, so Transition rejects them.
	if to == ticketdomain.StatusDismissed {
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

// ---------------------------------------------------------------------------
// Dismiss — the guarded dismissal path (TICK-04, D-13, T-11-07)
// ---------------------------------------------------------------------------

// Dismiss permanently closes the ticket via the guarded path: role ∈
// {manager, finance} (D-11); the pinned matrix must allow the current →
// 'dismissed' edge (open|triage); and the guard blocks dismissal while any
// linked activity carries logged hours (D-13 raw Σ from time_entries —
// computed server-side by the repo, NEVER client-supplied, T-11-07). The
// hours snapshot is persisted in dismissed_hours and rendered as the note
// "dismissed with N h logged" on read (TICK-04).
//
// The pool-level CanTransition / LoggedHours checks here are FAST-FAIL UX
// only (Pitfall 7, CR-01): the repo re-locks the ticket row + linked
// activities FOR UPDATE and re-computes the Σ inside the dismiss tx
// (loggedHoursTx) — the authoritative T-11-07 gate, check-and-act in one tx.
//
// Dismissal is intentionally NOT reachable through Transition: that path has
// no hours guard and would bypass T-11-07, so Transition rejects
// to == "dismissed" (ErrInvalidTransition) — Dismiss is the only door.
func (s *Service) Dismiss(ctx context.Context, orgID, actorID uuid.UUID, role string, ticketID uuid.UUID) (*ticketdomain.Ticket, error) {
	if role != string(models.RoleManager) && role != string(models.RoleFinance) {
		return nil, ticketdomain.ErrForbidden
	}
	t, err := s.repo.Get(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if !ticketdomain.CanTransition(t.Status, ticketdomain.StatusDismissed) {
		return nil, ticketdomain.ErrInvalidTransition
	}
	hours, err := s.repo.LoggedHours(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if hours > 0 {
		return nil, ticketdomain.ErrDismissalBlocked
	}
	actor := actorID
	return s.repo.Dismiss(ctx, orgID, ticketID, hours, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: "ticket",
		EntityID:   ticketID,
		Action:     "dismissed",
		ActorID:    &actor,
		Payload:    map[string]any{"hours": hours},
		CreatedAt:  time.Now().UTC(),
	})
}

// ---------------------------------------------------------------------------
// Triage — atomic kind reclassification + 1..N customer_ticket activities
// ---------------------------------------------------------------------------

// TriageActivityPlan is the service-level plan for one activity created
// during triage (D-10, TICK-03). Mirrors activitydomain.CreateActivityRequest
// minus the origin axis — the origin is forced to customer_ticket by the
// repo inside the triage tx.
type TriageActivityPlan struct {
	Name            string
	Description     string
	Kind            string
	ParentID        *uuid.UUID
	ContractID      *uuid.UUID
	GovernanceModel models.GovernanceModel
	IsShared        bool
	Billable        *bool
	BudgetAmount    *float64
}

// Triage atomically converts the ticket into 1..N customer_ticket-origin
// activities and flips it to 'planned' (D-10, TICK-03). Gates: role ∈
// {manager, finance} (D-11); the matrix must allow current → 'planned'
// (triage → planned); an optional kind override must stay in the closed
// four-kind set; at least one plan is required.
//
// Structural plan checks run here (pure string/pointer validation — no DB
// reads): name non-empty, kind non-empty, governance model valid. The
// DB-read validations (kind in the org catalog, parent same-org, contract
// same-org) are AUTHORITATIVE inside the repo's tx (Pitfall 7, T-11-06) —
// the service additionally fast-fails on the same rules via pool-level
// reads (kindExists/parent/contract) as optional UX only; the in-tx checks
// and the DB FK/CHECK constraints are the correctness guarantee. Likewise
// the pool-level CanTransition(current → 'planned') check here is FAST-FAIL
// UX only: the repo re-validates the matrix against the status it reads
// under the FOR UPDATE lock inside the triage tx (CR-01 — a dismissed
// ticket can never be resurrected).
//
// Both audit rows ('triaged' + 'activities_created') are written in the same
// tx as the state write and the activity inserts (TICK-03, ADR-BE-016).
func (s *Service) Triage(ctx context.Context, orgID, actorID uuid.UUID, role string, ticketID uuid.UUID, kind *string, plans []*TriageActivityPlan) (*ticketdomain.Ticket, []*activitydomain.ActivityResponse, error) {
	if role != string(models.RoleManager) && role != string(models.RoleFinance) {
		return nil, nil, ticketdomain.ErrForbidden
	}
	t, err := s.repo.Get(ctx, orgID, ticketID)
	if err != nil {
		return nil, nil, err
	}
	if !ticketdomain.CanTransition(t.Status, ticketdomain.StatusPlanned) {
		return nil, nil, ticketdomain.ErrInvalidTransition
	}
	if kind != nil && !isValidKind(*kind) {
		return nil, nil, ticketdomain.ErrInvalidRequest
	}
	if len(plans) == 0 {
		return nil, nil, ticketdomain.ErrInvalidRequest
	}

	converted := make([]*activitydomain.CreateActivityRequest, 0, len(plans))
	for _, p := range plans {
		if p.Name == "" || p.Kind == "" || !p.GovernanceModel.IsValid() {
			return nil, nil, ticketdomain.ErrInvalidRequest
		}
		// Optional fast-fail (UX only — the repo's in-tx checks are the
		// authoritative gate, Pitfall 7).
		if !s.activityKindExists(ctx, orgID, p.Kind) {
			return nil, nil, ticketdomain.ErrInvalidRequest
		}
		if p.ParentID != nil && !s.activityExists(ctx, orgID, *p.ParentID) {
			return nil, nil, ticketdomain.ErrInvalidRequest
		}
		if p.ContractID != nil && !s.contractExists(ctx, orgID, *p.ContractID) {
			return nil, nil, ticketdomain.ErrInvalidRequest
		}
		converted = append(converted, &activitydomain.CreateActivityRequest{
			ParentID:        p.ParentID,
			Name:            p.Name,
			Description:     p.Description,
			Kind:            activitydomain.ActivityKind(p.Kind),
			ContractID:      p.ContractID,
			GovernanceModel: p.GovernanceModel,
			IsShared:        p.IsShared,
			Billable:        p.Billable,
			BudgetAmount:    p.BudgetAmount,
		})
	}

	actor := actorID
	now := time.Now().UTC()
	audits := []*audit.AuditLog{
		{OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "triaged", ActorID: &actor, CreatedAt: now},
		{OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "activities_created", ActorID: &actor, CreatedAt: now},
	}
	return s.repo.Triage(ctx, orgID, ticketID, kind, converted, audits)
}

// ---------------------------------------------------------------------------
// AddComment / ListHistory — the append-only conversation + event stream
// ---------------------------------------------------------------------------

// AddComment appends a first-class comment (D-06) with its 'comment_added'
// audit row written in the same tx (TICK-05). Gate: owner/assignee/manager+
// (D-15). Body must be non-empty.
func (s *Service) AddComment(ctx context.Context, orgID, actorID uuid.UUID, role string, ticketID uuid.UUID, body string) (*ticketdomain.TicketComment, error) {
	t, err := s.repo.Get(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if !s.canUpdate(role, actorID, t) {
		return nil, ticketdomain.ErrForbidden
	}
	if body == "" {
		return nil, ticketdomain.ErrInvalidRequest
	}
	now := time.Now().UTC()
	c := &ticketdomain.TicketComment{
		ID:        uuid.New(),
		TicketID:  ticketID,
		AuthorID:  actorID,
		Body:      body,
		CreatedAt: now,
	}
	actor := actorID
	return s.repo.AddComment(ctx, orgID, ticketID, c, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: "ticket",
		EntityID:   ticketID,
		Action:     "comment_added",
		ActorID:    &actor,
		CreatedAt:  now,
	})
}

// ListHistory returns the ticket's append-only audit stream (TICK-05) for
// any internal member; the customer role is rejected (internal-only, D-E).
func (s *Service) ListHistory(ctx context.Context, orgID uuid.UUID, role string, ticketID uuid.UUID) ([]audit.AuditLog, error) {
	if role == string(models.RoleCustomer) {
		return nil, ticketdomain.ErrForbidden
	}
	return s.repo.ListHistory(ctx, orgID, ticketID)
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

// activityKindExists is the optional fast-fail for triage plans: kind label
// present in the org's activity_kinds catalog.
func (s *Service) activityKindExists(ctx context.Context, orgID uuid.UUID, kind string) bool {
	exists, err := s.activityRepo.KindExists(ctx, orgID, kind)
	return err == nil && exists
}

// activityExists is the optional fast-fail for triage plans: parent activity
// exists and belongs to the org.
func (s *Service) activityExists(ctx context.Context, orgID, activityID uuid.UUID) bool {
	a, err := s.activityRepo.Get(ctx, orgID, activityID)
	return err == nil && a != nil && a.OrgID == orgID
}

// contractExists is the optional fast-fail for triage plans: contract exists
// and belongs to the org.
func (s *Service) contractExists(ctx context.Context, orgID, contractID uuid.UUID) bool {
	c, err := s.contractRepo.Get(ctx, orgID, contractID)
	return err == nil && c != nil && c.CreatedByOrgID == orgID
}
