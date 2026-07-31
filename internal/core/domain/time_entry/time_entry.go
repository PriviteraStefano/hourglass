package time_entry

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTimeEntryNotFound    = errors.New("time entry not found")
	ErrEntryNotDraft        = errors.New("entry is not in draft status")
	ErrEntryNotSubmitted    = errors.New("entry is not in submitted status")
	ErrEntryAlreadyApproved = errors.New("entry is already approved")
	ErrPeriodLocked         = errors.New("cannot modify entry for locked period")
	ErrNotOwner             = errors.New("can only modify own entries")
	ErrForbidden            = errors.New("forbidden")
)

const (
	StatusDraft          = "draft"
	StatusSubmitted      = "submitted"
	StatusPendingManager = "pending_manager"
	StatusPendingFinance = "pending_finance"
	StatusApproved       = "approved"
	StatusRejected       = "rejected"
)

type TimeEntry struct {
	ID                 uuid.UUID       `json:"id"`
	OrgID              uuid.UUID       `json:"org_id"`
	UserID             uuid.UUID       `json:"user_id"`
	ActivityID         uuid.UUID       `json:"activity_id"`
	UnitID             uuid.UUID       `json:"unit_id"`
	Hours              float64         `json:"hours"`
	Description        string          `json:"description"`
	EntryDate          time.Time       `json:"entry_date"`
	Status             string          `json:"status"`
	CurrentApproverRole *string         `json:"current_approver_role,omitempty"`
	SubmittedAt         *time.Time      `json:"submitted_at,omitempty"`
	IsDeleted          bool            `json:"is_deleted"`
	CreatedFromEntryID *uuid.UUID      `json:"created_from_entry_id,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type CreateTimeEntryRequest struct {
	OrgID      uuid.UUID `json:"org_id"`
	UserID     uuid.UUID `json:"user_id"`
	ActivityID uuid.UUID `json:"activity_id"`
	UnitID     uuid.UUID `json:"unit_id"`
	Hours      float64   `json:"hours"`
	Description string   `json:"description"`
	Date       string    `json:"date"`
}

type UpdateTimeEntryRequest struct {
	ActivityID  *uuid.UUID `json:"activity_id,omitempty"`
	UnitID      *uuid.UUID `json:"unit_id,omitempty"`
	Hours       *float64   `json:"hours,omitempty"`
	Description *string    `json:"description,omitempty"`
	Date        *string    `json:"date,omitempty"`
}

type Approval struct {
	ID          uuid.UUID `json:"id"`
	EntryID     uuid.UUID `json:"entry_id"`
	Action      string    `json:"action"`
	ActorUserID uuid.UUID `json:"actor_user_id"`
	ActorRole   string    `json:"actor_role"`
	Comment     string    `json:"comment,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuditLog struct {
	ID        uuid.UUID          `json:"id"`
	OrgID     uuid.UUID          `json:"org_id"`
	EntryID   string             `json:"entry_id"`
	EntryType string             `json:"entry_type"`
	Action    string             `json:"action"`
	ActorRole string             `json:"actor_role"`
	ActorID   uuid.UUID          `json:"actor_id"`
	Reason    string             `json:"reason"`
	Changes   map[string]any     `json:"changes"`
	Timestamp time.Time          `json:"timestamp"`
}

func (e *TimeEntry) IsOwner(userID uuid.UUID) bool {
	return e.UserID == userID
}

func (e *TimeEntry) CanEdit() bool {
	return e.Status == StatusDraft || e.Status == StatusSubmitted || e.Status == StatusRejected
}

func (e *TimeEntry) CanSubmit() bool {
	return e.Status == StatusDraft || e.Status == StatusRejected
}
