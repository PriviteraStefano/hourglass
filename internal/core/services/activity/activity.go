package activity

import (
	"context"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// Service implements the activity business rules (ADR-P-007 D-2/D-3/D-5/D-8,
// ADR-P-013 origins) on top of ports.ActivityRepository. It replaces the
// collapsed project + subproject services (ADR-BE-014 R-6).
//
// UnitRepository is wired per the R-4 visibility axis: entry-level visibility
// (unit-subtree gating) lives in the entry repositories, and the activity List
// is org-scoped via the filter — the dependency is reserved for handler-level
// visibility scoping that 09-05 may compose.
//
// Origin dependencies (ADR-P-013): orgRepo validates origin refs are org
// members (D-02), ticketRepo validates customer_ticket refs exist in-org and
// checks the ticket-state precondition (OQ5/ADR-BE-016), and routing
// resolves the proposal approver set (D-G parity with entry approval). The
// proposal_approved audit row is written by the activity repo IN THE SAME
// TRANSACTION as the is_active flip (Pitfall 2, ADR-BE-016, T-11-08).
type Service struct {
	activityRepo ports.ActivityRepository
	contractRepo ports.ContractRepository
	unitRepo     ports.UnitRepository
	orgRepo      ports.OrganizationRepository
	ticketRepo   ports.TicketRepository
	routing      *routing.Service
}

func NewService(activityRepo ports.ActivityRepository, contractRepo ports.ContractRepository, unitRepo ports.UnitRepository, orgRepo ports.OrganizationRepository, ticketRepo ports.TicketRepository, routing *routing.Service) *Service {
	return &Service{
		activityRepo: activityRepo,
		contractRepo: contractRepo,
		unitRepo:     unitRepo,
		orgRepo:      orgRepo,
		ticketRepo:   ticketRepo,
		routing:      routing,
	}
}

// List returns the org's activities filtered by scope, contract, parent, kind
// and active state (delegates to the repository — visibility gating stays
// repo-side, R-4).
func (s *Service) List(ctx context.Context, orgID uuid.UUID, filter *activitydomain.ActivityFilter) ([]activitydomain.ActivityResponse, error) {
	return s.activityRepo.List(ctx, orgID, filter)
}

// ListChildren returns the direct children of an activity.
func (s *Service) ListChildren(ctx context.Context, parentID uuid.UUID) ([]activitydomain.ActivityResponse, error) {
	return s.activityRepo.ListChildren(ctx, parentID)
}

// ListKinds returns the org's activity_kinds catalog (ADR-P-007 D-2) — the
// labels Create validates against. Backs GET /api/activity-kinds.
func (s *Service) ListKinds(ctx context.Context, orgID uuid.UUID) ([]activitydomain.ActivityKind, error) {
	return s.activityRepo.ListKinds(ctx, orgID)
}

// GetByID returns a single activity scoped to the org.
func (s *Service) GetByID(ctx context.Context, orgID, activityID uuid.UUID) (*activitydomain.ActivityResponse, error) {
	return s.activityRepo.Get(ctx, orgID, activityID)
}

// Create validates the activity against the org's context before delegating:
//   - name + governance model are required (mirrors the old project service)
//   - kind must exist in the org's activity_kinds catalog (D-2) — unknown kinds
//     surface as ErrInvalidRequest instead of a raw FK violation
//   - a parent, when provided, must exist and belong to the same org (D-2)
//   - a contract, when provided, must exist (D-3) — contracts are optional;
//     personal/internal activities with no contract are first-class
//
// Origin payloads (ADR-P-013, D-D) are validated per type when
// req.OriginType is non-nil — see validateOrigin. The role + actor identity
// gate each type (D-04): manager_assignment and customer_ticket require
// manager|finance; employee_proposal requires proposed_by == actor.
func (s *Service) Create(ctx context.Context, role string, orgID, userID uuid.UUID, req *activitydomain.CreateActivityRequest) (*activitydomain.ActivityResponse, error) {
	if req.Name == "" || !req.GovernanceModel.IsValid() {
		return nil, activitydomain.ErrInvalidRequest
	}

	exists, err := s.activityRepo.KindExists(ctx, orgID, string(req.Kind))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, activitydomain.ErrInvalidRequest
	}

	if req.ParentID != nil {
		if err := s.validateParent(ctx, orgID, uuid.Nil, req.ParentID); err != nil {
			return nil, err
		}
	}

	if req.ContractID != nil {
		if _, err := s.contractRepo.Get(ctx, orgID, *req.ContractID); err != nil {
			return nil, err
		}
	}

	// COV-05/T-12-06: a beneficiary unit is accepted only when it belongs to
	// the same org — a cross-org unit id is rejected (fetch-and-compare via
	// unitRepo.GetByID, the same pattern the expense service applies).
	if req.BeneficiaryUnitID != nil {
		u, err := s.unitRepo.GetByID(ctx, req.BeneficiaryUnitID.String())
		if err != nil {
			return nil, err
		}
		if u.OrgID != orgID {
			return nil, activitydomain.ErrInvalidRequest
		}
	}

	if err := s.validateOrigin(ctx, role, orgID, userID, req); err != nil {
		return nil, err
	}

	return s.activityRepo.Create(ctx, orgID, req)
}

// validateOrigin applies the per-type origin gates (ADR-P-013 D-01/D-02/D-04,
// ADR-BE-016 OQ5 fast path):
//   - manager_assignment: role ∈ {manager, finance}; assigned_by + assigned_to
//     required; both must be org members (D-02)
//   - employee_proposal: proposed_by required and must equal the actor
//     (spoofing guard, D-04); is_active forced false (D-12)
//   - customer_ticket: role ∈ {manager, finance}; ticket_id required and must
//     resolve in-org (D-02); the ticket must be open|triage (OQ5 fast path)
//   - reviewed_by non-nil on ANY origin → rejected (Phase 11 resolution: the
//     approver lives in the audit row, ADR-P-013)
//   - unknown origin_type → rejected (closed set)
func (s *Service) validateOrigin(ctx context.Context, role string, orgID, userID uuid.UUID, req *activitydomain.CreateActivityRequest) error {
	if req.OriginType == nil {
		// No origin discriminator: reject an explicit false is_active — the
		// only legitimate false-at-create path is the employee proposal,
		// which is forced below (D-12).
		if req.IsActive != nil && !*req.IsActive {
			return activitydomain.ErrInvalidRequest
		}
		return nil
	}

	if req.ReviewedBy != nil {
		return activitydomain.ErrInvalidRequest
	}

	switch *req.OriginType {
	case activitydomain.OriginTypeManagerAssignment:
		if role != string(models.RoleManager) && role != string(models.RoleFinance) {
			return activitydomain.ErrForbidden
		}
		if req.AssignedBy == nil || req.AssignedTo == nil {
			return activitydomain.ErrInvalidRequest
		}
		if !s.isOrgMember(ctx, orgID, *req.AssignedBy) {
			return activitydomain.ErrInvalidRequest
		}
		if !s.isOrgMember(ctx, orgID, *req.AssignedTo) {
			return activitydomain.ErrInvalidRequest
		}
		return nil

	case activitydomain.OriginTypeEmployeeProposal:
		if req.ProposedBy == nil {
			return activitydomain.ErrInvalidRequest
		}
		if *req.ProposedBy != userID {
			return activitydomain.ErrForbidden
		}
		falseVal := false
		req.IsActive = &falseVal
		return nil

	case activitydomain.OriginTypeCustomerTicket:
		if role != string(models.RoleManager) && role != string(models.RoleFinance) {
			return activitydomain.ErrForbidden
		}
		if req.TicketID == nil {
			return activitydomain.ErrInvalidRequest
		}
		t, err := s.ticketRepo.Get(ctx, orgID, *req.TicketID)
		if err != nil {
			// Cross-org or unknown ticket: same-org precondition (D-02).
			return activitydomain.ErrInvalidRequest
		}
		if t.Status != ticket.StatusOpen && t.Status != ticket.StatusTriage {
			// OQ5 fast path (ADR-BE-016): activities may only link to tickets
			// still open for work.
			return activitydomain.ErrInvalidRequest
		}
		return nil

	default:
		return activitydomain.ErrInvalidRequest
	}
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

// Update is finance-role gated with an owner check (same pattern as the old
// project service): only the org that created the activity may update it.
// Origin fields in an update request are rejected with ErrOriginImmutable
// (D-03, T-11-10) — the discriminator and its reference set are set-once.
func (s *Service) Update(ctx context.Context, role string, orgID, activityID uuid.UUID, req *activitydomain.UpdateActivityRequest) (*activitydomain.ActivityResponse, error) {
	if role != string(models.RoleFinance) {
		return nil, activitydomain.ErrForbidden
	}
	existing, err := s.activityRepo.Get(ctx, orgID, activityID)
	if err != nil {
		return nil, err
	}
	if existing.CreatedByOrgID != orgID {
		return nil, activitydomain.ErrForbidden
	}
	// WR-04: mirror Create — the kind must exist in the org's catalog (D-2)
	// and the governance model must be valid. Without this, a bogus
	// kind/governance_model was written straight through to the UPDATE and
	// surfaced as a raw FK (23503) / CHECK (23514) 500 at the handler.
	if req.Kind != "" {
		exists, err := s.activityRepo.KindExists(ctx, orgID, string(req.Kind))
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, activitydomain.ErrInvalidRequest
		}
	}
	if req.GovernanceModel != "" && !req.GovernanceModel.IsValid() {
		return nil, activitydomain.ErrInvalidRequest
	}
	if hasOriginFields(req) {
		return nil, activitydomain.ErrOriginImmutable
	}
	if req.ParentID != nil {
		if err := s.validateParent(ctx, orgID, activityID, req.ParentID); err != nil {
			return nil, err
		}
	}
	// COV-05/T-12-06: same-org fetch-and-compare on the update path too —
	// the beneficiary unit is editable (unlike origin refs) so every write
	// re-validates it. hasOriginFields stays untouched: this is not an
	// origin ref.
	if req.BeneficiaryUnitID != nil {
		u, err := s.unitRepo.GetByID(ctx, req.BeneficiaryUnitID.String())
		if err != nil {
			return nil, err
		}
		if u.OrgID != orgID {
			return nil, activitydomain.ErrInvalidRequest
		}
	}
	return s.activityRepo.Update(ctx, orgID, activityID, req)
}

// hasOriginFields reports whether an update request carries any origin field
// (D-03 immutability guard).
func hasOriginFields(req *activitydomain.UpdateActivityRequest) bool {
	return req.OriginType != nil ||
		req.AssignedBy != nil ||
		req.AssignedTo != nil ||
		req.ProposedBy != nil ||
		req.ReviewedBy != nil ||
		req.TicketID != nil
}

// ApproveProposal approves an employee proposal (D-12, T-11-02). Proposals
// are is_active=false activities whose approval routes through the shared
// BE-014 machinery (D-G parity with entry approval) and lands in the general
// audit_logs — never in origin refs (ADR-P-013: reviewed_by stays NULL).
//
// Gate sequence:
//  1. the activity must exist in-org and be an employee_proposal
//  2. it must not already be approved (is_active)
//  3. no self-approval (actor != proposer)
//  4. the proposer's primary unit anchors the routing walk
//  5. routing.ResolveManagerStage resolves the approver set — propagated
//     errors include activity.ErrActivityNotLoggable (commercial proposal
//     without an anchored WG, R-2)
//  6. role-gated resolutions require manager|finance; D-11 skips (proposer
//     in the approver set) are rejected ONLY when the set is exactly
//     {proposer} — with delegates in the set they are legitimate approvers
//     (WR-04)
//  7. otherwise the actor must be in the resolved approver set
//
// Persistence flips is_active via the repo Update directly (bypassing the
// service Update's finance gate — the approver check above IS the gate) and
// writes a synchronous proposal_approved audit row (T-11-08).
func (s *Service) ApproveProposal(ctx context.Context, role string, orgID, actorID, activityID uuid.UUID) (*activitydomain.ActivityResponse, error) {
	existing, err := s.activityRepo.Get(ctx, orgID, activityID)
	if err != nil {
		return nil, err
	}

	if existing.OriginType == nil || *existing.OriginType != activitydomain.OriginTypeEmployeeProposal {
		return nil, activitydomain.ErrInvalidRequest
	}
	if existing.IsActive {
		return nil, activitydomain.ErrInvalidRequest
	}
	if existing.ProposedBy == nil {
		return nil, activitydomain.ErrInvalidRequest
	}
	if actorID == *existing.ProposedBy {
		return nil, activitydomain.ErrForbidden
	}

	// Resolve the proposer's primary unit — uuid.Nil when they have no
	// primary membership (routing walks from the unit tree; nil short-circuits
	// to the terminal role-gated resolution).
	unitID := uuid.Nil
	memberships, err := s.unitRepo.ListMembershipsForUser(ctx, *existing.ProposedBy)
	if err != nil {
		return nil, err
	}
	for _, m := range memberships {
		if m.IsPrimary {
			if id, err := uuid.Parse(m.UnitID); err == nil {
				unitID = id
			}
			break
		}
	}

	res, err := s.routing.ResolveManagerStage(ctx, orgID, activityID, unitID, *existing.ProposedBy)
	if err != nil {
		return nil, err
	}

	switch {
	case res.RoleGated:
		if role != string(models.RoleManager) && role != string(models.RoleFinance) {
			return nil, activitydomain.ErrForbidden
		}
	case res.SkipToFinance:
		// WR-04: the skip short-circuit must NOT reject before the
		// approver-set check. SkipToFinance means the proposer IS in the
		// approver set — only when the set is exactly {proposer} (no
		// delegates) is the proposal unapprovable (routing cannot
		// self-approve). With delegates, the delegate is a legitimate
		// approver and falls through to the membership check below.
		if len(res.ApproverIDs) == 1 { // set == {proposer}
			return nil, activitydomain.ErrForbidden
		}
		if !contains(res.ApproverIDs, actorID) {
			return nil, activitydomain.ErrForbidden
		}
	default:
		if !contains(res.ApproverIDs, actorID) {
			return nil, activitydomain.ErrForbidden
		}
	}

	// Persistence flips is_active AND writes the proposal_approved audit row
	// IN THE SAME TRANSACTION (Pitfall 2, ADR-BE-016, T-11-08, WR-05): the
	// state write is not durable without its event — a failure rolls back
	// both, never a partial commit. Mirrors the ticket repo's UpdateState.
	updated, err := s.activityRepo.ApproveProposal(ctx, orgID, activityID, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: "activity",
		EntityID:   activityID,
		Action:     "proposal_approved",
		ActorID:    &actorID,
		Payload:    map[string]any{"approver": actorID.String()},
		CreatedAt:  time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// contains reports whether id is in ids (mirrors the time_entry service
// helper for routing approver sets).
func contains(ids []uuid.UUID, id uuid.UUID) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}
	return false
}

// validateParent is the single parent-validation point shared by Create and
// Update (ADR-P-007 D-2 + SPEC in-scope item "Cycle prevention on
// activities.parent_id (path check on insert/update)"). Contract: a nil
// parent is always valid; otherwise the parent must (1) exist (repo returns
// ErrActivityNotFound when missing), (2) belong to the same org
// (ErrInvalidRequest otherwise — the same-org rule Create already had, now
// also enforced on Update), and (3) not sit inside the activity's own
// subtree: walking GetAncestry of the proposed parent and rejecting with
// ErrActivityCycle when the chain contains ownActivityID. A fresh uuid.Nil
// (Create) can never appear in an existing ancestry, so the insert path
// check is structurally satisfied while running uniformly per the SPEC.
func (s *Service) validateParent(ctx context.Context, orgID, ownActivityID uuid.UUID, parentID *uuid.UUID) error {
	if parentID == nil {
		return nil
	}
	parent, err := s.activityRepo.Get(ctx, orgID, *parentID)
	if err != nil {
		return err
	}
	if parent.OrgID != orgID {
		return activitydomain.ErrInvalidRequest
	}
	ancestry, err := s.activityRepo.GetAncestry(ctx, *parentID)
	if err != nil {
		return err
	}
	for _, a := range ancestry {
		if a.ID == ownActivityID {
			return activitydomain.ErrActivityCycle
		}
	}
	return nil
}

// Delete is finance-role gated with an owner check, then guards the activity
// before removal: no children, no active time entries (own subtree), no active
// expenses. Each guard returns a clean sentinel per ADR-BE-001 instead of a
// raw PG constraint error; adoptions are cascade-cleaned by the repository in
// a transaction.
func (s *Service) Delete(ctx context.Context, role string, orgID, activityID uuid.UUID) error {
	if role != string(models.RoleFinance) {
		return activitydomain.ErrForbidden
	}
	existing, err := s.activityRepo.Get(ctx, orgID, activityID)
	if err != nil {
		return err
	}
	if existing.CreatedByOrgID != orgID {
		return activitydomain.ErrForbidden
	}

	hasChildren, err := s.activityRepo.HasChildren(ctx, activityID)
	if err != nil {
		return err
	}
	if hasChildren {
		return activitydomain.ErrHasChildren
	}

	hasEntries, hasDescendantEntries, err := s.activityRepo.HasActiveTimeEntries(ctx, activityID)
	if err != nil {
		return err
	}
	if hasEntries || hasDescendantEntries {
		return activitydomain.ErrHasActiveTimeEntries
	}

	hasExpenses, err := s.activityRepo.HasActiveExpenses(ctx, activityID)
	if err != nil {
		return err
	}
	if hasExpenses {
		return activitydomain.ErrHasActiveExpenses
	}

	return s.activityRepo.Delete(ctx, orgID, activityID)
}

// Adopt shares an activity with another org (sharing preserved from the
// project adoption model, ADR-P-007 D-6).
func (s *Service) Adopt(ctx context.Context, orgID, activityID uuid.UUID) (*activitydomain.ActivityAdoption, error) {
	return s.activityRepo.Adopt(ctx, orgID, activityID)
}

// ListManagers returns the governance managers (activity_managers) for an
// activity. Managers keep governance meaning but are NOT an approval queue
// (ADR-BE-014 R-2).
func (s *Service) ListManagers(ctx context.Context, activityID uuid.UUID) ([]activitydomain.ActivityManager, error) {
	return s.activityRepo.ListManagers(ctx, activityID)
}

// AddManager is finance-role gated.
func (s *Service) AddManager(ctx context.Context, actorRole string, activityID, userID uuid.UUID) (*activitydomain.ActivityManager, error) {
	if actorRole != string(models.RoleFinance) {
		return nil, activitydomain.ErrForbidden
	}
	return s.activityRepo.AddManager(ctx, activityID, userID)
}

// RemoveManager is finance-role gated.
func (s *Service) RemoveManager(ctx context.Context, actorRole string, activityID, userID uuid.UUID) error {
	if actorRole != string(models.RoleFinance) {
		return activitydomain.ErrForbidden
	}
	return s.activityRepo.RemoveManager(ctx, activityID, userID)
}
