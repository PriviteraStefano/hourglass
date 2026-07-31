package ports

import (
	"context"

	"github.com/google/uuid"
	domainexpense "github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
)

type ExpenseRepository interface {
	Create(ctx context.Context, e *domainexpense.Expense) (*domainexpense.Expense, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domainexpense.Expense, error)
	Update(ctx context.Context, e *domainexpense.Expense) (*domainexpense.Expense, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, orgID uuid.UUID, filters ExpenseListFilters) ([]domainexpense.Expense, error)
	ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]domainexpense.Expense, error)
	IsPeriodLocked(ctx context.Context, orgID, activityID uuid.UUID, entryDate string) (bool, error)
	CreateApproval(ctx context.Context, a *domainexpense.Approval) error
}

type ExpenseListFilters struct {
	OrgID         interface{}
	Date          string
	Month         string
	Year          string
	UserID        string
	Status        string
	ActivityID    string
	Role          string
	IsDeleted     bool
	RequestUserID string
}
