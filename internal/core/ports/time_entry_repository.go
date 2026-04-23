package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
)

type TimeEntryRepository interface {
	List(ctx context.Context, orgID uuid.UUID, filters ListFilters) ([]time_entry.TimeEntry, error)
	GetByID(ctx context.Context, id uuid.UUID) (*time_entry.TimeEntry, error)
	Create(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error)
	Update(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IsPeriodLocked(ctx context.Context, orgID, projectID uuid.UUID, entryDate string) (bool, error)
	ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]time_entry.TimeEntry, error)
}

type ListFilters struct {
	OrgID         interface{}
	Date          string
	Month         string
	Year          string
	UserID        string
	Status        string
	WGID          string
	ProjectID     string
	Role          string
	IsDeleted     bool
	RequestUserID string
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *time_entry.AuditLog) error
}
