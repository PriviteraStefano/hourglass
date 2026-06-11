package expense

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type Service struct {
	repo ports.ExpenseRepository
}

func NewService(repo ports.ExpenseRepository) *Service {
	return &Service{repo: repo}
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

	locked, err := s.repo.IsPeriodLocked(ctx, req.OrgID, req.ProjectID, req.Date)
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
		ProjectID:   req.ProjectID,
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

	if req.ProjectID != nil {
		e.ProjectID = *req.ProjectID
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

	now := time.Now()
	e.Status = expense.StatusSubmitted
	e.CurrentApproverRole = strPtr("manager")
	e.SubmittedAt = &now
	e.UpdatedAt = now

	return s.repo.Update(ctx, e)
}

func (s *Service) Approve(ctx context.Context, id, userID uuid.UUID, role string) (*expense.Expense, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if e.UserID == userID {
		return nil, expense.ErrForbidden
	}

	switch {
	case role == "manager" && e.Status == expense.StatusSubmitted:
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

func (s *Service) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]expense.Expense, error) {
	return s.repo.ListPending(ctx, orgID, role, userID)
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

func strPtr(s string) *string { return &s }

func timePtr(t time.Time) *time.Time { return &t }
