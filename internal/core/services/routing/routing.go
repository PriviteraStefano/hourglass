package routing

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// Service resolves the manager stage for approval routing (ADR-BE-014
// R-1/R-2). It is shared by entry approval and proposal approval so the
// routing rules cannot drift (D-G parity).
type Service struct {
	wgRepo       ports.WorkingGroupRepository
	activityRepo ports.ActivityRepository
	unitRepo     ports.UnitRepository
}

func NewService(wgRepo ports.WorkingGroupRepository, activityRepo ports.ActivityRepository, unitRepo ports.UnitRepository) *Service {
	return &Service{
		wgRepo:       wgRepo,
		activityRepo: activityRepo,
		unitRepo:     unitRepo,
	}
}

// managerResolution is the resolved manager stage for an entry (ADR-BE-014
// R-1/R-2). The approver set is either the anchored WG's manager + delegates
// (R-1) or the submitter's unit manager (R-2 fallback for personal activities).
// roleGated marks the terminal unit-tree case (org root without a manager),
// which routes to the org-role manager — the service cannot pin a user there,
// so the actor's role gate governs (ADR-BE-014 consequences).
//
// The type is unexported (an implementation detail of the package); the
// fields are exported so callers — the time_entry service and, later, the
// proposal-approval path — can consume the resolution through the returned
// pointer exactly as they consumed the private struct.
type managerResolution struct {
	ApproverIDs   []uuid.UUID
	RoleGated     bool
	SkipToFinance bool // D-11: the entry owner IS in the approver set
}

// ResolveManagerStage resolves who approves the manager stage for an entry:
//
//   - R-1 chain: the activity's anchored WG → WG manager + delegates. The
//     D-11 skip (R-3) is role-based: if the owner is the WG manager OR any
//     delegate, submitted transitions directly to pending_finance. Being a WG
//     member is not enough — only membership in the approver set counts.
//   - R-2 enforcement: a commercial activity (contract via the derived chain)
//     with no anchored WG rejects the submission with ErrActivityNotLoggable.
//   - R-2 fallback: a personal activity (no contract, no WG — D-8) routes to
//     the submitter's unit manager, walking the unit tree upward.
func (s *Service) ResolveManagerStage(ctx context.Context, orgID, activityID, unitID, ownerID uuid.UUID) (*managerResolution, error) {
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
		return &managerResolution{ApproverIDs: approverIDs, SkipToFinance: ownerIsApprover}, nil
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
	managerID, found, err := s.ResolveUnitManager(ctx, unitID)
	if err != nil {
		return nil, err
	}
	if found {
		return &managerResolution{ApproverIDs: []uuid.UUID{managerID}, SkipToFinance: managerID == ownerID}, nil
	}

	// Terminal state: org root without a manager → role-gated manager stage.
	return &managerResolution{RoleGated: true}, nil
}

// ResolveUnitManager walks the unit tree upward from unitID and returns the
// nearest unit membership with role = 'manager' (ADR-P-001 Q2 / ADR-BE-014
// R-2). found=false means no manager exists anywhere up to the org root.
func (s *Service) ResolveUnitManager(ctx context.Context, unitID uuid.UUID) (uuid.UUID, bool, error) {
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
