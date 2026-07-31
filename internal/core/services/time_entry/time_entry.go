package time_entry

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type Service struct {
	repo         ports.TimeEntryRepository
	approvalRepo ports.TimeEntryApprovalRepository
	wgRepo       ports.WorkingGroupRepository
	activityRepo ports.ActivityRepository
	unitRepo     ports.UnitRepository
}

func NewService(repo ports.TimeEntryRepository, approvalRepo ports.TimeEntryApprovalRepository, wgRepo ports.WorkingGroupRepository, activityRepo ports.ActivityRepository, unitRepo ports.UnitRepository) *Service {
	return &Service{
		repo:         repo,
		approvalRepo: approvalRepo,
		wgRepo:       wgRepo,
		activityRepo: activityRepo,
		unitRepo:     unitRepo,
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

// managerResolution is the resolved manager stage for an entry (ADR-BE-014
// R-1/R-2). The approver set is either the anchored WG's manager + delegates
// (R-1) or the submitter's unit manager (R-2 fallback for personal activities).
// roleGated marks the terminal unit-tree case (org root without a manager),
// which routes to the org-role manager — the service cannot pin a user there,
// so the actor's role gate governs (ADR-BE-014 consequences).
type managerResolution struct {
	approverIDs   []uuid.UUID
	roleGated     bool
	skipToFinance bool // D-11: the entry owner IS in the approver set
}

// resolveManagerStage resolves who approves the manager stage for an entry:
//
//   - R-1 chain: the activity's anchored WG → WG manager + delegates. The
//     D-11 skip (R-3) is role-based: if the owner is the WG manager OR any
//     delegate, submitted transitions directly to pending_finance. Being a WG
//     member is not enough — only membership in the approver set counts.
//   - R-2 enforcement: a commercial activity (contract via the derived chain)
//     with no anchored WG rejects the submission with ErrActivityNotLoggable.
//   - R-2 fallback: a personal activity (no contract, no WG — D-8) routes to
//     the submitter's unit manager, walking the unit tree upward.
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

	// No anchored WG: commercial activities must anchor one before accepting
	// entries (ADR-BE-014 R-2). Only personal activities use the fallback.
	commercial, err := s.activityRepo.ResolveCommercialContext(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if commercial != nil && commercial.ContractID != nil {
		return nil, activity.ErrActivityNotLoggable
	}

	// R-2 fallback: submitter's unit manager (upward walk). Self-approval is
	// impossible here by construction — the unit manager is a different
	// membership row than the submitter's — so the D-11 skip is a no-op, but
	// the role-based check is applied uniformly.
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
		res, err := s.resolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
		if err != nil {
			if errors.Is(err, activity.ErrActivityNotLoggable) {
				// No legitimate manager-stage approver exists for this entry.
				return nil, time_entry.ErrForbidden
			}
			return nil, err
		}
		if !res.roleGated && !contains(res.approverIDs, userID) {
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
func (s *Service) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]time_entry.TimeEntry, error) {
	return s.repo.ListPending(ctx, orgID, role, userID)
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
