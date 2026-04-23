package time_entry

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTimeEntryNotFound = errors.New("time entry not found")
	ErrEntryNotDraft     = errors.New("entry is not in draft status")
	ErrEntryNotSubmitted = errors.New("entry is not in submitted status")
	ErrPeriodLocked      = errors.New("cannot modify entry for locked period")
	ErrNotOwner          = errors.New("can only modify own entries")
	ErrForbidden         = errors.New("forbidden")
)

const (
	StatusDraft     = "draft"
	StatusSubmitted = "submitted"
	StatusApproved  = "approved"
)

type TimeEntry struct {
	ID                 uuid.UUID
	OrgID              uuid.UUID
	UserID             uuid.UUID
	ProjectID          uuid.UUID
	SubprojectID       uuid.UUID
	WGID               uuid.UUID
	UnitID             uuid.UUID
	Hours              float64
	Description        string
	EntryDate          time.Time
	Status             string
	IsDeleted          bool
	CreatedFromEntryID *uuid.UUID
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateTimeEntryRequest struct {
	OrgID        uuid.UUID
	UserID       uuid.UUID
	ProjectID    uuid.UUID
	SubprojectID uuid.UUID
	WGID         uuid.UUID
	UnitID       uuid.UUID
	Hours        float64
	Description  string
	Date         string
}

type UpdateTimeEntryRequest struct {
	ProjectID    *uuid.UUID
	SubprojectID *uuid.UUID
	WGID         *uuid.UUID
	UnitID       *uuid.UUID
	Hours        *float64
	Description  *string
	Date         *string
}

type AuditLog struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	EntryID   string
	EntryType string
	Action    string
	ActorRole string
	ActorID   uuid.UUID
	Reason    string
	Changes   map[string]interface{}
	Timestamp time.Time
}

func (e *TimeEntry) IsOwner(userID uuid.UUID) bool {
	return e.UserID == userID
}

func (e *TimeEntry) CanEdit() bool {
	return e.Status == StatusDraft
}

func (e *TimeEntry) CanSubmit() bool {
	return e.Status == StatusDraft
}
