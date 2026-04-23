package time_entry

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type Service struct {
	repo      ports.TimeEntryRepository
	auditRepo ports.AuditLogRepository
}

func NewService(repo ports.TimeEntryRepository, auditRepo ports.AuditLogRepository) *Service {
	return &Service{repo: repo, auditRepo: auditRepo}
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, filters ports.ListFilters) ([]time_entry.TimeEntry, error) {
	return s.repo.List(ctx, orgID, filters)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*time_entry.TimeEntry, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req *time_entry.CreateTimeEntryRequest) (*time_entry.TimeEntry, error) {
	locked, err := s.repo.IsPeriodLocked(ctx, req.OrgID, req.ProjectID, req.Date)
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
		ID:           uuid.New(),
		OrgID:        req.OrgID,
		UserID:       req.UserID,
		ProjectID:    req.ProjectID,
		SubprojectID: req.SubprojectID,
		WGID:         req.WGID,
		UnitID:       req.UnitID,
		Hours:        req.Hours,
		Description:  req.Description,
		EntryDate:    entryDate,
		Status:       time_entry.StatusDraft,
		IsDeleted:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
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

	if req.ProjectID != nil {
		e.ProjectID = *req.ProjectID
	}
	if req.SubprojectID != nil {
		e.SubprojectID = *req.SubprojectID
	}
	if req.WGID != nil {
		e.WGID = *req.WGID
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

	if !e.CanEdit() {
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

	e.Status = time_entry.StatusSubmitted
	e.UpdatedAt = time.Now()

	return s.repo.Update(ctx, e)
}

func (s *Service) Approve(ctx context.Context, id, userID uuid.UUID, role string) (*time_entry.TimeEntry, error) {
	if role != "wg_manager" && role != "admin" {
		return nil, time_entry.ErrForbidden
	}

	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if e.Status != time_entry.StatusSubmitted {
		return nil, time_entry.ErrEntryNotSubmitted
	}

	e.Status = time_entry.StatusApproved
	e.UpdatedAt = time.Now()

	return s.repo.Update(ctx, e)
}

func (s *Service) Reject(ctx context.Context, id, userID uuid.UUID, role, reason string) (*time_entry.TimeEntry, error) {
	if role != "wg_manager" && role != "admin" {
		return nil, time_entry.ErrForbidden
	}

	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if e.Status != time_entry.StatusSubmitted {
		return nil, time_entry.ErrEntryNotSubmitted
	}

	e.Status = time_entry.StatusDraft
	e.UpdatedAt = time.Now()

	return s.repo.Update(ctx, e)
}

func (s *Service) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]time_entry.TimeEntry, error) {
	return s.repo.ListPending(ctx, orgID, role, userID)
}

func (s *Service) CreateAuditLog(ctx context.Context, orgID uuid.UUID, entryID, entryType, action, actorRole, actorID, reason string, changes map[string]interface{}) {
	auditLog := &time_entry.AuditLog{
		ID:        uuid.New(),
		OrgID:     orgID,
		EntryID:   entryID,
		EntryType: entryType,
		Action:    action,
		ActorRole: actorRole,
		ActorID:   uuid.MustParse(actorID),
		Reason:    reason,
		Changes:   changes,
		Timestamp: time.Now(),
	}
	go s.auditRepo.Create(ctx, auditLog)
}
