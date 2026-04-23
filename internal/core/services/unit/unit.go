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

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*unit.Unit, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req *unit.CreateUnitRequest) (*unit.Unit, error) {
	hierarchyLevel := 0
	if req.ParentUnitID != nil {
		parent, err := s.repo.GetByID(ctx, *req.ParentUnitID)
		if err == nil && parent != nil {
			hierarchyLevel = parent.HierarchyLevel + 1
		}
	}

	now := time.Now()
	u := &unit.Unit{
		ID:             uuid.New(),
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

func (s *Service) Update(ctx context.Context, id uuid.UUID, req *unit.UpdateUnitRequest) (*unit.Unit, error) {
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
	u.UpdatedAt = time.Now()

	return s.repo.Update(ctx, u)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
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
	return unit.BuildTree(units, nil), nil
}

func (s *Service) GetDescendants(ctx context.Context, id uuid.UUID) ([]unit.Unit, error) {
	return s.repo.GetDescendants(ctx, id)
}
