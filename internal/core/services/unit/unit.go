package unit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type Service struct {
	repo ports.UnitRepository
}

func NewService(repo ports.UnitRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]unit.Unit, error) {
	return s.repo.ListByOrg(ctx, orgID)
}

func (s *Service) Get(ctx context.Context, id string) (*unit.Unit, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req *unit.CreateUnitRequest) (*unit.Unit, error) {
	hierarchyLevel := 0
	if req.ParentUnitID != "" {
		parent, err := s.repo.GetByID(ctx, req.ParentUnitID)
		if err == nil && parent != nil {
			hierarchyLevel = parent.HierarchyLevel + 1
		}
	}

	now := time.Now()
	u := &unit.Unit{
		ID:             uuid.New().String(),
		OrgID:          req.OrgID,
		Name:           req.Name,
		Description:    req.Description,
		ParentUnitID:   req.ParentUnitID,
		HierarchyLevel: hierarchyLevel,
		Code:           req.Code,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return s.repo.Create(ctx, u)
}

func (s *Service) Update(ctx context.Context, id string, req *unit.UpdateUnitRequest) (*unit.Unit, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		u.Name = req.Name
	}
	if req.Description != "" {
		u.Description = req.Description
	}
	if req.Code != "" {
		u.Code = req.Code
	}
	if req.ParentUnitID != nil {
		newParentID := *req.ParentUnitID
		if newParentID == u.ParentUnitID {
			u.UpdatedAt = time.Now()
			return s.repo.Update(ctx, u)
		}
		if newParentID != "" {
			if newParentID == id {
				return nil, unit.ErrCircularParent
			}
			descendants, err := s.repo.GetDescendants(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, d := range descendants {
				if d.ID == newParentID {
					return nil, unit.ErrCircularParent
				}
			}
			parent, err := s.repo.GetByID(ctx, newParentID)
			if err != nil {
				return nil, unit.ErrInvalidParentUnit
			}
			u.ParentUnitID = newParentID
			u.HierarchyLevel = parent.HierarchyLevel + 1
		} else {
			u.ParentUnitID = ""
			u.HierarchyLevel = 0
		}
		if err := s.cascadeHierarchyLevel(ctx, u); err != nil {
			return nil, err
		}
	}
	u.UpdatedAt = time.Now()

	return s.repo.Update(ctx, u)
}

func (s *Service) cascadeHierarchyLevel(ctx context.Context, parent *unit.Unit) error {
	descendants, err := s.repo.GetDescendants(ctx, parent.ID)
	if err != nil {
		return err
	}
	for _, d := range descendants {
		d.HierarchyLevel = parent.HierarchyLevel + 1
		d.UpdatedAt = time.Now()
		if _, err := s.repo.Update(ctx, &d); err != nil {
			return err
		}
		if err := s.cascadeHierarchyLevel(ctx, &d); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u.HierarchyLevel == 0 {
		return unit.ErrCannotDeleteRootUnit
	}
	hasChildren, err := s.repo.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return unit.ErrCannotDeleteWithChildren
	}
	hasMembers, err := s.repo.HasMembers(ctx, id)
	if err != nil {
		return err
	}
	if hasMembers {
		return unit.ErrCannotDeleteWithMembers
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) GetTree(ctx context.Context, orgID uuid.UUID) ([]unit.UnitTreeNode, error) {
	units, err := s.repo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	memberCounts, err := s.repo.GetMemberCountsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	tree := unit.BuildTree(units, "")
	annotateMemberCounts(tree, memberCounts)
	return tree, nil
}

func annotateMemberCounts(nodes []unit.UnitTreeNode, counts map[string]int) {
	for i := range nodes {
		nodes[i].MemberCount = counts[nodes[i].Unit.ID]
		total := nodes[i].MemberCount
		annotateMemberCounts(nodes[i].Children, counts)
		for _, child := range nodes[i].Children {
			total += child.TotalMemberCount
		}
		nodes[i].TotalMemberCount = total
	}
}

func (s *Service) GetDescendants(ctx context.Context, id string) ([]unit.Unit, error) {
	return s.repo.GetDescendants(ctx, id)
}

func (s *Service) ListMembers(ctx context.Context, unitID string) ([]unit.UnitMember, error) {
	return s.repo.ListMembers(ctx, unitID)
}

func (s *Service) AddMember(ctx context.Context, unitID string, orgID uuid.UUID, req *unit.AddUnitMemberRequest) (*unit.UnitMember, error) {
	m := &unit.UnitMember{
		ID:        uuid.New().String(),
		OrgID:     orgID,
		UserID:    req.UserID,
		UnitID:    unitID,
		Role:      req.Role,
		IsPrimary: req.IsPrimary,
		StartDate: time.Now(),
		CreatedAt: time.Now(),
	}
	return s.repo.AddMember(ctx, m)
}

func (s *Service) RemoveMember(ctx context.Context, id string) error {
	return s.repo.RemoveMember(ctx, id)
}

func (s *Service) UpdateMember(ctx context.Context, unitID, membershipID string, isPrimary bool, endDate *time.Time) (*unit.UnitMember, error) {
	members, err := s.repo.ListMembers(ctx, unitID)
	if err != nil {
		return nil, err
	}

	var targetMember *unit.UnitMember
	for _, m := range members {
		if m.ID == membershipID {
			targetMember = &m
			break
		}
	}
	if targetMember == nil {
		return nil, unit.ErrMemberNotFound
	}

	if isPrimary {
		allMemberships, err := s.repo.ListMembershipsForUser(ctx, targetMember.UserID)
		if err != nil {
			return nil, err
		}
		for _, m := range allMemberships {
			if m.IsPrimary && m.ID != membershipID {
				m.IsPrimary = false
				if _, err := s.repo.UpdateMember(ctx, &m); err != nil {
					return nil, err
				}
			}
		}
	}

	targetMember.IsPrimary = isPrimary
	targetMember.EndDate = endDate
	return s.repo.UpdateMember(ctx, targetMember)
}

func (s *Service) ListMembersByUnitIDs(ctx context.Context, orgID uuid.UUID, unitIDs []string) ([]unit.UnitMember, error) {
	return s.repo.ListMembersByUnitIDs(ctx, orgID, unitIDs)
}
