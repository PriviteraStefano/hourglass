package working_group

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type Service struct {
	repo ports.WorkingGroupRepository
}

func NewService(repo ports.WorkingGroupRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByOrg(ctx context.Context, orgID uuid.UUID, subprojectID *uuid.UUID) ([]working_group.WorkingGroup, error) {
	return s.repo.ListByOrg(ctx, orgID, subprojectID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*working_group.WorkingGroup, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req *working_group.CreateWorkingGroupRequest) (*working_group.WorkingGroup, error) {
	now := time.Now()
	wg := &working_group.WorkingGroup{
		ID:               uuid.New(),
		OrgID:            req.OrgID,
		SubprojectID:     req.SubprojectID,
		Name:             req.Name,
		Description:      req.Description,
		UnitIDs:          req.UnitIDs,
		EnforceUnitTuple: req.EnforceUnitTuple,
		ManagerID:        req.ManagerID,
		DelegateIDs:      req.DelegateIDs,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return s.repo.Create(ctx, wg)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req *working_group.UpdateWorkingGroupRequest) (*working_group.WorkingGroup, error) {
	wg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		wg.Name = req.Name
	}
	if req.Description != "" {
		wg.Description = req.Description
	}
	if req.UnitIDs != nil {
		wg.UnitIDs = req.UnitIDs
	}
	if req.EnforceUnitTuple != nil {
		wg.EnforceUnitTuple = *req.EnforceUnitTuple
	}
	if req.ManagerID != uuid.Nil {
		wg.ManagerID = req.ManagerID
	}
	if req.DelegateIDs != nil {
		wg.DelegateIDs = req.DelegateIDs
	}
	wg.UpdatedAt = time.Now()

	return s.repo.Update(ctx, wg)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	hasMembers, err := s.repo.HasMembers(ctx, id)
	if err != nil {
		return err
	}
	if hasMembers {
		return working_group.ErrCannotDeleteWithMembers
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) ListMembers(ctx context.Context, wgID uuid.UUID) ([]working_group.WorkingGroupMember, error) {
	return s.repo.ListMembers(ctx, wgID)
}

func (s *Service) AddMember(ctx context.Context, req *working_group.AddMemberRequest) (*working_group.WorkingGroupMember, error) {
	m := &working_group.WorkingGroupMember{
		ID:                  uuid.New(),
		WGID:                req.WGID,
		UserID:              req.UserID,
		UnitID:              req.UnitID,
		Role:                req.Role,
		IsDefaultSubproject: req.IsDefaultSubproject,
		StartDate:           time.Now(),
		CreatedAt:           time.Now(),
	}
	return s.repo.AddMember(ctx, m)
}

func (s *Service) RemoveMember(ctx context.Context, id uuid.UUID) error {
	return s.repo.RemoveMember(ctx, id)
}
