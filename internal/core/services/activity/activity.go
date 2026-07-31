package activity

import (
	"context"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// Service implements the activity business rules (ADR-P-007 D-2/D-3/D-5/D-8)
// on top of ports.ActivityRepository. It replaces the collapsed project +
// subproject services (ADR-BE-014 R-6).
//
// UnitRepository is wired per the R-4 visibility axis: entry-level visibility
// (unit-subtree gating) lives in the entry repositories, and the activity List
// is org-scoped via the filter — the dependency is reserved for handler-level
// visibility scoping that 09-05 may compose.
type Service struct {
	activityRepo ports.ActivityRepository
	contractRepo ports.ContractRepository
	unitRepo     ports.UnitRepository
}

func NewService(activityRepo ports.ActivityRepository, contractRepo ports.ContractRepository, unitRepo ports.UnitRepository) *Service {
	return &Service{
		activityRepo: activityRepo,
		contractRepo: contractRepo,
		unitRepo:     unitRepo,
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
func (s *Service) Create(ctx context.Context, orgID uuid.UUID, req *activitydomain.CreateActivityRequest) (*activitydomain.ActivityResponse, error) {
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

	return s.activityRepo.Create(ctx, orgID, req)
}

// Update is finance-role gated with an owner check (same pattern as the old
// project service): only the org that created the activity may update it.
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
	if req.ParentID != nil {
		if err := s.validateParent(ctx, orgID, activityID, req.ParentID); err != nil {
			return nil, err
		}
	}
	return s.activityRepo.Update(ctx, orgID, activityID, req)
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
