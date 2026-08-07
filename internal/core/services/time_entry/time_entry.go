package time_entry

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
)

type Service struct {
	repo         ports.TimeEntryRepository
	approvalRepo ports.TimeEntryApprovalRepository
	wgRepo       ports.WorkingGroupRepository
	activityRepo ports.ActivityRepository
	unitRepo     ports.UnitRepository
	routing      *routing.Service
}

func NewService(repo ports.TimeEntryRepository, approvalRepo ports.TimeEntryApprovalRepository, wgRepo ports.WorkingGroupRepository, activityRepo ports.ActivityRepository, unitRepo ports.UnitRepository, routingSvc *routing.Service) *Service {
	return &Service{
		repo:         repo,
		approvalRepo: approvalRepo,
		wgRepo:       wgRepo,
		activityRepo: activityRepo,
		unitRepo:     unitRepo,
		routing:      routingSvc,
	}
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, filters ports.ListFilters) ([]time_entry.TimeEntry, error) {
	return s.repo.List(ctx, orgID, filters)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*time_entry.TimeEntry, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req *time_entry.CreateTimeEntryRequest) (*time_entry.TimeEntry, error) {
	locked, err := s.repo.IsPeriodLocked(ctx, req.OrgID, req.ActivityID, req.Date)
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, time_entry.ErrPeriodLocked
	}

	entryDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	e := &time_entry.TimeEntry{
		ID:          uuid.New(),
		OrgID:       req.OrgID,
		UserID:      req.UserID,
		ActivityID:  req.ActivityID,
		UnitID:      req.UnitID,
		Hours:       req.Hours,
		Description: req.Description,
		EntryDate:   entryDate,
		Status:      time_entry.StatusDraft,
		IsDeleted:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, e)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *time_entry.UpdateTimeEntryRequest) (*time_entry.TimeEntry, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !e.CanEdit() {
		return nil, time_entry.ErrEntryNotDraft
	}
	if !e.IsOwner(userID) {
		return nil, time_entry.ErrNotOwner
	}

	if req.ActivityID != nil {
		e.ActivityID = *req.ActivityID
	}
	if req.UnitID != nil {
		e.UnitID = *req.UnitID
	}
	if req.Hours != nil {
		e.Hours = *req.Hours
	}
	if req.Description != nil {
		e.Description = *req.Description
	}
	if req.Date != nil {
		entryDate, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return nil, err
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

	if e.Status != time_entry.StatusDraft {
		return time_entry.ErrEntryNotDraft
	}
	if !e.IsOwner(userID) {
		return time_entry.ErrNotOwner
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) Submit(ctx context.Context, id, userID uuid.UUID) (*time_entry.TimeEntry, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !e.CanSubmit() {
		return nil, time_entry.ErrEntryNotDraft
	}
	if !e.IsOwner(userID) {
		return nil, time_entry.ErrNotOwner
	}

	// WR-06: a draft entry on an activity linked to a DISMISSED customer
	// ticket can never be submitted — the ticket is terminal and must not
	// acquire new logged hours after the fact. The check walks the
	// activity's ancestry (repo-side) so entries on descendant activities of
	// a dismissed ticket are blocked too, matching the ticket repo's subtree
	// "linked" semantics.
	dismissed, err := s.activityRepo.IsLinkedTicketDismissed(ctx, e.ActivityID)
	if err != nil {
		return nil, err
	}
	if dismissed {
		return nil, time_entry.ErrTicketDismissed
	}

	// R-1/R-2: resolve the manager stage from the activity chain (or reject
	// commercial activities with no anchored WG — ErrActivityNotLoggable).
	res, err := s.routing.ResolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	e.SubmittedAt = &now
	e.UpdatedAt = now
	if res.SkipToFinance {
		// D-11 (R-3): the approver role (WG manager/delegate) coincides with
		// the owner — skip the manager stage, never self-approval.
		e.Status = time_entry.StatusPendingFinance
		e.CurrentApproverRole = strPtr("finance")
	} else {
		e.Status = time_entry.StatusSubmitted
		e.CurrentApproverRole = strPtr("manager")
	}

	return s.repo.Update(ctx, e)
}

func (s *Service) Approve(ctx context.Context, id, userID uuid.UUID, role string) (*time_entry.TimeEntry, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Structural self-approval barrier: the owner can never approve their own
	// entry in any role (ADR-P-001 Q4 — the D-11 skip handles the case where
	// the owner IS the legitimate approver by skipping the stage instead).
	if e.UserID == userID {
		return nil, time_entry.ErrForbidden
	}

	switch {
	case role == "manager" && e.Status == time_entry.StatusSubmitted:
		// R-1/R-2 routing: the actor must be in the resolved approver set
		// (WG manager/delegate or unit manager). The terminal role-gated case
		// delegates to the handler's role resolution.
		res, err := s.routing.ResolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
		if err != nil {
			if errors.Is(err, activity.ErrActivityNotLoggable) {
				// No legitimate manager-stage approver exists for this entry.
				return nil, time_entry.ErrForbidden
			}
			return nil, err
		}
		if !res.RoleGated && !contains(res.ApproverIDs, userID) {
			return nil, time_entry.ErrForbidden
		}
		e.Status = time_entry.StatusPendingFinance
		e.CurrentApproverRole = strPtr("finance")
	case role == "finance" && e.Status == time_entry.StatusPendingFinance:
		e.Status = time_entry.StatusApproved
		e.CurrentApproverRole = nil
	default:
		return nil, time_entry.ErrEntryNotSubmitted
	}

	e.UpdatedAt = time.Now()
	result, err := s.repo.Update(ctx, e)
	if err != nil {
		return nil, err
	}

	if err := s.approvalRepo.CreateApproval(ctx, &time_entry.Approval{
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

func (s *Service) Reject(ctx context.Context, id, userID uuid.UUID, role, reason string) (*time_entry.TimeEntry, error) {
	if role != "manager" && role != "finance" {
		return nil, time_entry.ErrForbidden
	}

	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if e.Status != time_entry.StatusSubmitted && e.Status != time_entry.StatusPendingFinance {
		return nil, time_entry.ErrEntryNotSubmitted
	}

	e.Status = time_entry.StatusRejected
	e.CurrentApproverRole = nil
	e.UpdatedAt = time.Now()

	result, err := s.repo.Update(ctx, e)
	if err != nil {
		return nil, err
	}

	if err := s.approvalRepo.CreateApproval(ctx, &time_entry.Approval{
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
// (R-4) lives repo-side and is resolved from role + userID there. The service
// deliberately does not add a second filter.
//
// Gate semantics (T-10-05-3): the handler admits org-role manager/finance OR
// any working-group manager/delegate in the org, resolving WG membership via
// IsWGManager and passing role "wg_manager" so the repository WG-scopes the
// queue. This method must never be called with an unresolved org role — the
// caller decides admission.
func (s *Service) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]time_entry.TimeEntry, error) {
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
