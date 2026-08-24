package expense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type Service struct {
	repo         ports.ExpenseRepository
	wgRepo       ports.WorkingGroupRepository
	activityRepo ports.ActivityRepository
	unitRepo     ports.UnitRepository
}

func NewService(repo ports.ExpenseRepository, wgRepo ports.WorkingGroupRepository, activityRepo ports.ActivityRepository, unitRepo ports.UnitRepository) *Service {
	return &Service{
		repo:         repo,
		wgRepo:       wgRepo,
		activityRepo: activityRepo,
		unitRepo:     unitRepo,
	}
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, filters ports.ExpenseListFilters) ([]expense.Expense, error) {
	return s.repo.List(ctx, orgID, filters)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*expense.Expense, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req *expense.CreateExpenseRequest) (*expense.Expense, error) {
	if !expense.IsValidCategory(req.Category) {
		return nil, expense.ErrInvalidCategory
	}

	locked, err := s.repo.IsPeriodLocked(ctx, req.OrgID, req.ActivityID, req.Date)
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, expense.ErrPeriodLocked
	}

	entryDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	now := time.Now()
	e := &expense.Expense{
		ID:          uuid.New(),
		OrgID:       req.OrgID,
		UserID:      req.UserID,
		ActivityID:  req.ActivityID,
		Category:    req.Category,
		Amount:      req.Amount,
		KmDistance:  req.KmDistance,
		Description: req.Description,
		EntryDate:   entryDate,
		Status:      expense.StatusDraft,
		IsDeleted:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	// Map the requested unit_id through to the persisted record. The postgres
	// repo already inserts unit_id (expense_repository.go); previously the
	// handler/service never supplied it, so it was silently dropped (Phase 16
	// known bug).
	if req.UnitID != nil {
		e.UnitID = *req.UnitID
	}

	return s.repo.Create(ctx, e)
}

func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, req *expense.UpdateExpenseRequest) (*expense.Expense, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !e.IsOwner(userID) {
		return nil, expense.ErrNotOwner
	}
	if !e.CanEdit() {
		return nil, expense.ErrEntryNotDraft
	}

	if req.ActivityID != nil {
		e.ActivityID = *req.ActivityID
	}
	if req.Category != nil {
		if !expense.IsValidCategory(*req.Category) {
			return nil, expense.ErrInvalidCategory
		}
		e.Category = *req.Category
	}
	if req.Amount != nil {
		e.Amount = *req.Amount
	}
	if req.KmDistance != nil {
		e.KmDistance = req.KmDistance
	}
	if req.Description != nil {
		e.Description = *req.Description
	}
	if req.Date != nil {
		entryDate, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %w", err)
		}
		e.EntryDate = entryDate
	}
	e.UpdatedAt = time.Now()

	return s.repo.Update(ctx, e)
}

func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if e.Status != expense.StatusDraft {
		return expense.ErrEntryNotDraft
	}
	if !e.IsOwner(userID) {
		return expense.ErrNotOwner
	}

	return s.repo.Delete(ctx, id)
}

// managerResolution is the resolved manager stage for an entry (ADR-BE-014
// R-1/R-2). Identical to the time-entry service: the approver set is the
// anchored WG's manager + delegates (R-1) or the submitter's unit manager
// (R-2 fallback for personal activities). roleGated marks the terminal
// unit-tree case, which routes to the org-role manager.
type managerResolution struct {
	approverIDs   []uuid.UUID
	roleGated     bool
	skipToFinance bool // D-11: the entry owner IS in the approver set
}

// resolveManagerStage resolves who approves the manager stage for an expense —
// the exact chain as time entries (ADR-P-001 Q1, now implementable via the
// shared activity_id FK):
//
//   - R-1 chain: activity → anchored WG → WG manager + delegates; the D-11
//     skip (R-3) is role-based (owner in the approver set → pending_finance).
//   - R-2 enforcement: commercial activity (contract via the derived chain)
//     with no anchored WG → ErrActivityNotLoggable.
//   - R-2 fallback: personal activity (no contract, no WG — D-8) → the
//     submitter's unit manager, walking the unit tree upward.
func (s *Service) resolveManagerStage(ctx context.Context, orgID, activityID, unitID, ownerID uuid.UUID) (*managerResolution, error) {
	wgs, err := s.wgRepo.ListByOrg(ctx, orgID, &activityID)
	if err != nil {
		return nil, err
	}

	if len(wgs) > 0 {
		wg := wgs[0]
		set := map[uuid.UUID]struct{}{wg.ManagerID: {}}
		for _, d := range wg.DelegateIDs {
			uid, err := uuid.Parse(d)
			if err == nil {
				set[uid] = struct{}{}
			}
		}
		approverIDs := make([]uuid.UUID, 0, len(set))
		for uid := range set {
			approverIDs = append(approverIDs, uid)
		}
		_, ownerIsApprover := set[ownerID]
		return &managerResolution{approverIDs: approverIDs, skipToFinance: ownerIsApprover}, nil
	}

	commercial, err := s.activityRepo.ResolveCommercialContext(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if commercial != nil && commercial.ContractID != nil {
		return nil, activity.ErrActivityNotLoggable
	}

	// R-2 fallback: submitter's unit manager (upward walk). Self-approval is
	// impossible here by construction; the D-11 check is applied uniformly.
	managerID, found, err := s.resolveUnitManager(ctx, unitID)
	if err != nil {
		return nil, err
	}
	if found {
		return &managerResolution{approverIDs: []uuid.UUID{managerID}, skipToFinance: managerID == ownerID}, nil
	}

	// Terminal state: org root without a manager → role-gated manager stage.
	return &managerResolution{roleGated: true}, nil
}

// resolveUnitManager walks the unit tree upward from unitID and returns the
// nearest unit membership with role = 'manager' (ADR-P-001 Q2 / ADR-BE-014
// R-2). found=false means no manager exists anywhere up to the org root.
func (s *Service) resolveUnitManager(ctx context.Context, unitID uuid.UUID) (uuid.UUID, bool, error) {
	cur := unitID.String()
	for cur != "" {
		members, err := s.unitRepo.ListMembers(ctx, cur)
		if err != nil {
			if errors.Is(err, unit.ErrUnitNotFound) {
				return uuid.Nil, false, nil
			}
			return uuid.Nil, false, err
		}
		for _, m := range members {
			if m.Role == "manager" {
				return m.UserID, true, nil
			}
		}
		u, err := s.unitRepo.GetByID(ctx, cur)
		if err != nil {
			if errors.Is(err, unit.ErrUnitNotFound) {
				return uuid.Nil, false, nil
			}
			return uuid.Nil, false, err
		}
		cur = u.ParentUnitID
	}
	return uuid.Nil, false, nil
}

func (s *Service) Submit(ctx context.Context, id, userID uuid.UUID) (*expense.Expense, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !e.CanSubmit() {
		return nil, expense.ErrEntryNotDraft
	}
	if !e.IsOwner(userID) {
		return nil, expense.ErrNotOwner
	}

	// R-1/R-2: resolve the manager stage from the activity chain (or reject
	// commercial activities with no anchored WG — ErrActivityNotLoggable).
	res, err := s.resolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	e.SubmittedAt = &now
	e.UpdatedAt = now
	if res.skipToFinance {
		// D-11 (R-3): the approver role coincides with the owner — skip the
		// manager stage, never self-approval.
		e.Status = expense.StatusPendingFinance
		e.CurrentApproverRole = strPtr("finance")
	} else {
		e.Status = expense.StatusSubmitted
		e.CurrentApproverRole = strPtr("manager")
	}

	return s.repo.Update(ctx, e)
}

func (s *Service) Approve(ctx context.Context, id, userID uuid.UUID, role string) (*expense.Expense, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Structural self-approval barrier (same as time entries): the owner can
	// never approve their own expense in any role.
	if e.UserID == userID {
		return nil, expense.ErrForbidden
	}

	switch {
	case role == "manager" && e.Status == expense.StatusSubmitted:
		// R-1/R-2 routing: the actor must be in the resolved approver set.
		res, err := s.resolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
		if err != nil {
			if errors.Is(err, activity.ErrActivityNotLoggable) {
				return nil, expense.ErrForbidden
			}
			return nil, err
		}
		if !res.roleGated && !contains(res.approverIDs, userID) {
			return nil, expense.ErrForbidden
		}
		e.Status = expense.StatusPendingFinance
		e.CurrentApproverRole = strPtr("finance")
	case role == "finance" && e.Status == expense.StatusPendingFinance:
		e.Status = expense.StatusApproved
		e.CurrentApproverRole = nil
	default:
		return nil, expense.ErrEntryNotSubmitted
	}

	e.UpdatedAt = time.Now()
	result, err := s.repo.Update(ctx, e)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateApproval(ctx, &expense.Approval{
		ID:          uuid.New(),
		EntryID:     id,
		Action:      "approve",
		ActorUserID: userID,
		ActorRole:   role,
		CreatedAt:   time.Now(),
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) Reject(ctx context.Context, id, userID uuid.UUID, role, reason string) (*expense.Expense, error) {
	if role != "manager" && role != "finance" {
		return nil, expense.ErrForbidden
	}

	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if e.Status != expense.StatusSubmitted && e.Status != expense.StatusPendingFinance {
		return nil, expense.ErrEntryNotSubmitted
	}

	e.Status = expense.StatusRejected
	e.CurrentApproverRole = nil
	e.UpdatedAt = time.Now()

	result, err := s.repo.Update(ctx, e)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateApproval(ctx, &expense.Approval{
		ID:          uuid.New(),
		EntryID:     id,
		Action:      "reject",
		ActorUserID: userID,
		ActorRole:   role,
		Comment:     reason,
		CreatedAt:   time.Now(),
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// ListPending passes through to the repository — the unit-subtree gating
// (R-4) lives repo-side; the service deliberately does not add a second
// filter. Expenses route identically to time entries (ADR-P-001 Q1); there is
// no project-manager approval queue anywhere in this path (ADR-BE-014 R-2).
//
// Gate semantics (T-10-05-3): the handler admits org-role manager/finance OR
// any working-group manager/delegate in the org, resolving WG membership via
// IsWGManager and passing role "wg_manager" so the repository WG-scopes the
// queue. This method must never be called with an unresolved org role — the
// caller decides admission.
func (s *Service) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]expense.Expense, error) {
	return s.repo.ListPending(ctx, orgID, role, userID)
}

// IsWGManager reports whether the user is the manager or a delegate of any
// working group in the org — the manager-stage approver set, resolved
// server-side from the WG repo (mirrors resolveManagerStage's wgRepo path).
// Used by the ListPending handler gate to admit WG-stage approvers whose org
// role is employee (T-10-05-3). The handler passes role "wg_manager" when
// true, and the repository then WG-scopes the pending queue.
func (s *Service) IsWGManager(ctx context.Context, orgID uuid.UUID, userID string) (bool, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	wgs, err := s.wgRepo.ListByOrg(ctx, orgID, nil)
	if err != nil {
		return false, err
	}
	for _, wg := range wgs {
		if wg.ManagerID == uid {
			return true, nil
		}
		for _, d := range wg.DelegateIDs {
			if did, err := uuid.Parse(d); err == nil && did == uid {
				return true, nil
			}
		}
	}
	return false, nil
}

// SetReceiptURL sets or updates the receipt URL for an expense.
func (s *Service) SetReceiptURL(ctx context.Context, id uuid.UUID, receiptURL string) (*expense.Expense, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	e.ReceiptURL = &receiptURL
	e.UpdatedAt = time.Now()

	return s.repo.Update(ctx, e)
}

func contains(ids []uuid.UUID, id uuid.UUID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func strPtr(s string) *string { return &s }

func timePtr(t time.Time) *time.Time { return &t }
