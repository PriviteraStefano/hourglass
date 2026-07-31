package expense

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrExpenseNotFound      = errors.New("expense not found")
	ErrEntryNotDraft        = errors.New("expense is not in draft status")
	ErrEntryNotSubmitted    = errors.New("expense is not in submitted status")
	ErrEntryAlreadyApproved = errors.New("expense is already approved")
	ErrPeriodLocked         = errors.New("cannot modify expense for locked period")
	ErrNotOwner             = errors.New("can only modify own expenses")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidCategory      = errors.New("invalid expense category")
)

const (
	StatusDraft          = "draft"
	StatusSubmitted      = "submitted"
	StatusPendingManager = "pending_manager"
	StatusPendingFinance = "pending_finance"
	StatusApproved       = "approved"
	StatusRejected       = "rejected"
)

const (
	CategoryMileage       = "mileage"
	CategoryMeal          = "meal"
	CategoryAccommodation = "accommodation"
	CategoryParking       = "parking"
	CategoryTravelTickets = "travel_tickets"
	CategoryTolls         = "tolls"
	CategoryTaxi          = "taxi"
	CategoryEquipment     = "equipment"
	CategoryOther         = "other"
)

func IsValidCategory(cat string) bool {
	switch cat {
	case CategoryMileage, CategoryMeal, CategoryAccommodation, CategoryParking, CategoryTravelTickets, CategoryTolls, CategoryTaxi, CategoryEquipment, CategoryOther:
		return true
	default:
		return false
	}
}

type Expense struct {
	ID                  uuid.UUID  `json:"id"`
	OrgID               uuid.UUID  `json:"org_id"`
	UserID              uuid.UUID  `json:"user_id"`
	ActivityID          uuid.UUID  `json:"activity_id"`
	ActivityName        string     `json:"activity_name,omitempty"` // joined display field (09-05)
	ActivityKind        string     `json:"activity_kind,omitempty"` // joined display field (09-05)
	UnitID              uuid.UUID  `json:"unit_id"`
	Category            string     `json:"category"`
	Amount              float64    `json:"amount"`
	KmDistance          *float64   `json:"km_distance,omitempty"`
	Description         string     `json:"description"`
	EntryDate           time.Time  `json:"entry_date"`
	Status              string     `json:"status"`
	CurrentApproverRole *string    `json:"current_approver_role,omitempty"`
	SubmittedAt         *time.Time `json:"submitted_at,omitempty"`
	ReceiptURL          *string    `json:"receipt_url,omitempty"`
	IsDeleted           bool       `json:"is_deleted"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (e *Expense) IsOwner(userID uuid.UUID) bool {
	return e.UserID == userID
}

func (e *Expense) CanEdit() bool {
	return e.Status == StatusDraft || e.Status == StatusSubmitted || e.Status == StatusRejected
}

func (e *Expense) CanSubmit() bool {
	return e.Status == StatusDraft || e.Status == StatusRejected
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

type CreateExpenseRequest struct {
	OrgID       uuid.UUID `json:"org_id"`
	UserID      uuid.UUID `json:"user_id"`
	ActivityID  uuid.UUID `json:"activity_id"`
	Category    string    `json:"category"`
	Amount      float64   `json:"amount"`
	KmDistance  *float64  `json:"km_distance,omitempty"`
	Description string    `json:"description"`
	Date        string    `json:"date"`
}

type UpdateExpenseRequest struct {
	ActivityID  *uuid.UUID `json:"activity_id,omitempty"`
	Category    *string    `json:"category,omitempty"`
	Amount      *float64   `json:"amount,omitempty"`
	KmDistance  *float64   `json:"km_distance,omitempty"`
	Description *string    `json:"description,omitempty"`
	Date        *string    `json:"date,omitempty"`
}
