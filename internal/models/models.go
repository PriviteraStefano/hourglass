package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleFinance  Role = "finance"
	RoleCustomer Role = "customer"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleEmployee, RoleManager, RoleFinance, RoleCustomer:
		return true
	default:
		return false
	}
}

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type OrganizationMembership struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Role           Role       `json:"role"`
	IsActive       bool       `json:"is_active"`
	InvitedBy      *uuid.UUID `json:"invited_by,omitempty"`
	InvitedAt      *time.Time `json:"invited_at,omitempty"`
	ActivatedAt    *time.Time `json:"activated_at,omitempty"`
}

type UserWithMembership struct {
	User         User                   `json:"user"`
	Membership   OrganizationMembership `json:"membership"`
	Organization Organization           `json:"organization"`
}

type GovernanceModel string

const (
	GovernanceCreatorControlled GovernanceModel = "creator_controlled"
	GovernanceUnanimous         GovernanceModel = "unanimous"
	GovernanceMajority          GovernanceModel = "majority"
)

func (g GovernanceModel) IsValid() bool {
	switch g {
	case GovernanceCreatorControlled, GovernanceUnanimous, GovernanceMajority:
		return true
	default:
		return false
	}
}

type ProjectType string

const (
	ProjectTypeBillable ProjectType = "billable"
	ProjectTypeInternal ProjectType = "internal"
)

func (p ProjectType) IsValid() bool {
	switch p {
	case ProjectTypeBillable, ProjectTypeInternal:
		return true
	default:
		return false
	}
}

type Contract struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	KmRate          float64         `json:"km_rate"`
	Currency        string          `json:"currency"`
	GovernanceModel GovernanceModel `json:"governance_model"`
	CreatedByOrgID  uuid.UUID       `json:"created_by_org_id"`
	IsShared        bool            `json:"is_shared"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
}

type Project struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Type            ProjectType     `json:"type"`
	ContractID      uuid.UUID       `json:"contract_id"`
	GovernanceModel GovernanceModel `json:"governance_model"`
	CreatedByOrgID  uuid.UUID       `json:"created_by_org_id"`
	IsShared        bool            `json:"is_shared"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
}

type ContractAdoption struct {
	ID             uuid.UUID `json:"id"`
	ContractID     uuid.UUID `json:"contract_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	AdoptedAt      time.Time `json:"adopted_at"`
}

type ProjectAdoption struct {
	ID             uuid.UUID `json:"id"`
	ProjectID      uuid.UUID `json:"project_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	AdoptedAt      time.Time `json:"adopted_at"`
}

type EntryStatus string

const (
	StatusDraft          EntryStatus = "draft"
	StatusSubmitted      EntryStatus = "submitted"
	StatusPendingManager EntryStatus = "pending_manager"
	StatusPendingFinance EntryStatus = "pending_finance"
	StatusApproved       EntryStatus = "approved"
	StatusRejected       EntryStatus = "rejected"
)

func (s EntryStatus) IsValid() bool {
	switch s {
	case StatusDraft, StatusSubmitted, StatusPendingManager, StatusPendingFinance, StatusApproved, StatusRejected:
		return true
	default:
		return false
	}
}

type TimeEntry struct {
	ID                  uuid.UUID       `json:"id"`
	UserID              uuid.UUID       `json:"user_id"`
	OrganizationID      uuid.UUID       `json:"organization_id"`
	Date                time.Time       `json:"date"`
	Status              EntryStatus     `json:"status"`
	CurrentApproverRole *string         `json:"current_approver_role,omitempty"`
	SubmittedAt         *time.Time      `json:"submitted_at,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	Items               []TimeEntryItem `json:"items,omitempty"`
}

type TimeEntryItem struct {
	ID          uuid.UUID `json:"id"`
	TimeEntryID uuid.UUID `json:"time_entry_id"`
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectName string    `json:"project_name,omitempty"`
	Hours       float64   `json:"hours"`
	Description string    `json:"description,omitempty"`
}

type TimeEntryCreateRequest struct {
	Date  string                       `json:"date"`
	Items []TimeEntryItemCreateRequest `json:"items"`
}

type TimeEntryItemCreateRequest struct {
	ProjectID   string  `json:"project_id"`
	Hours       float64 `json:"hours"`
	Description string  `json:"description,omitempty"`
}

type TimeEntryUpdateRequest struct {
	Items []TimeEntryItemCreateRequest `json:"items"`
}

type TimeEntryMonthlySummary struct {
	Days   []TimeEntryDaySummary `json:"days"`
	Totals map[string]float64    `json:"totals"`
	Matrix []TimeEntryMatrixRow  `json:"matrix"`
}

type TimeEntryDaySummary struct {
	Date       string                    `json:"date"`
	TotalHours float64                   `json:"total_hours"`
	Projects   []TimeEntryProjectSummary `json:"projects"`
}

type TimeEntryProjectSummary struct {
	ProjectID   string  `json:"project_id"`
	ProjectName string  `json:"project_name"`
	Hours       float64 `json:"hours"`
}

type TimeEntryMatrixRow struct {
	Project string             `json:"project"`
	Days    map[string]float64 `json:"days"`
	Total   float64            `json:"total"`
}

type ExpenseCategory string

const (
	CategoryMileage       ExpenseCategory = "mileage"
	CategoryMeal          ExpenseCategory = "meal"
	CategoryAccommodation ExpenseCategory = "accommodation"
	CategoryOther         ExpenseCategory = "other"
)

func (c ExpenseCategory) IsValid() bool {
	switch c {
	case CategoryMileage, CategoryMeal, CategoryAccommodation, CategoryOther:
		return true
	default:
		return false
	}
}

type Expense struct {
	ID                  uuid.UUID     `json:"id"`
	UserID              uuid.UUID     `json:"user_id"`
	OrganizationID      uuid.UUID     `json:"organization_id"`
	Date                time.Time     `json:"date"`
	Status              EntryStatus   `json:"status"`
	CurrentApproverRole *string       `json:"current_approver_role,omitempty"`
	SubmittedAt         *time.Time    `json:"submitted_at,omitempty"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
	Items               []ExpenseItem `json:"items,omitempty"`
}

type ExpenseItem struct {
	ID          uuid.UUID        `json:"id"`
	ExpenseID   uuid.UUID        `json:"expense_id"`
	ProjectID   uuid.UUID        `json:"project_id"`
	ProjectName string           `json:"project_name,omitempty"`
	Category    ExpenseCategory  `json:"category"`
	Amount      float64          `json:"amount"`
	KmDistance  *float64         `json:"km_distance,omitempty"`
	Description string           `json:"description,omitempty"`
	Receipts    []ExpenseReceipt `json:"receipts,omitempty"`
}

type ExpenseReceipt struct {
	ID               uuid.UUID `json:"id"`
	ExpenseItemID    uuid.UUID `json:"expense_item_id"`
	FilePath         string    `json:"file_path"`
	OriginalFilename string    `json:"original_filename"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

type ExpenseCreateRequest struct {
	Date  string                     `json:"date"`
	Items []ExpenseItemCreateRequest `json:"items"`
}

type ExpenseItemCreateRequest struct {
	ProjectID   string          `json:"project_id"`
	Category    ExpenseCategory `json:"category"`
	Amount      float64         `json:"amount"`
	KmDistance  *float64        `json:"km_distance,omitempty"`
	Description string          `json:"description,omitempty"`
}

type ExpenseUpdateRequest struct {
	Items []ExpenseItemCreateRequest `json:"items"`
}

type ExpenseMonthlySummary struct {
	Days       []ExpenseDaySummary `json:"days"`
	Totals     map[string]float64  `json:"totals"`
	Categories map[string]float64  `json:"categories"`
}

type ExpenseDaySummary struct {
	Date        string               `json:"date"`
	TotalAmount float64              `json:"total_amount"`
	Items       []ExpenseItemSummary `json:"items"`
}

type ExpenseItemSummary struct {
	ProjectID   string  `json:"project_id"`
	ProjectName string  `json:"project_name"`
	Category    string  `json:"category"`
	Amount      float64 `json:"amount"`
}

type ApprovalAction string

const (
	ActionSubmit         ApprovalAction = "submit"
	ActionApprove        ApprovalAction = "approve"
	ActionReject         ApprovalAction = "reject"
	ActionEditApprove    ApprovalAction = "edit_approve"
	ActionEditReturn     ApprovalAction = "edit_return"
	ActionPartialApprove ApprovalAction = "partial_approve"
	ActionDelegate       ApprovalAction = "delegate"
)

type TimeEntryApproval struct {
	ID          uuid.UUID      `json:"id"`
	TimeEntryID uuid.UUID      `json:"time_entry_id"`
	Action      ApprovalAction `json:"action"`
	ActorUserID uuid.UUID      `json:"actor_user_id"`
	ActorRole   string         `json:"actor_role"`
	Changes     string         `json:"changes,omitempty"`
	Comment     string         `json:"comment,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type ExpenseApproval struct {
	ID          uuid.UUID      `json:"id"`
	ExpenseID   uuid.UUID      `json:"expense_id"`
	Action      ApprovalAction `json:"action"`
	ActorUserID uuid.UUID      `json:"actor_user_id"`
	ActorRole   string         `json:"actor_role"`
	Changes     string         `json:"changes,omitempty"`
	Comment     string         `json:"comment,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type BackupApprover struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Role           Role      `json:"role"`
	UserID         uuid.UUID `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type PendingEntryGroup struct {
	UserID   uuid.UUID      `json:"user_id"`
	UserName string         `json:"user_name"`
	Month    int            `json:"month"`
	Year     int            `json:"year"`
	Entries  []PendingEntry `json:"entries"`
}

type PendingEntry struct {
	ID                  uuid.UUID   `json:"id"`
	Date                time.Time   `json:"date"`
	Status              EntryStatus `json:"status"`
	CurrentApproverRole string      `json:"current_approver_role,omitempty"`
	Items               interface{} `json:"items"`
}

type SubmitRequest struct {
	Comment string `json:"comment,omitempty"`
}

type SubmitMonthRequest struct {
	Month   int    `json:"month"`
	Year    int    `json:"year"`
	Comment string `json:"comment,omitempty"`
}

type RejectRequest struct {
	Comment string `json:"comment"`
}

type EditApproveRequest struct {
	Items        []TimeEntryItemCreateRequest `json:"items,omitempty"`
	ExpenseItems []ExpenseItemCreateRequest   `json:"expense_items,omitempty"`
	Comment      string                       `json:"comment,omitempty"`
}

type EditReturnRequest struct {
	Items        []TimeEntryItemCreateRequest `json:"items,omitempty"`
	ExpenseItems []ExpenseItemCreateRequest   `json:"expense_items,omitempty"`
	Comment      string                       `json:"comment"`
}

type PartialApproveRequest struct {
	ApprovedItemIDs []string `json:"approved_item_ids"`
	Comment         string   `json:"comment,omitempty"`
}

type DelegateRequest struct {
	DelegateToUserID uuid.UUID `json:"delegate_to_user_id"`
	Comment          string    `json:"comment,omitempty"`
}

type BatchApproveRequest struct {
	EntryIDs []uuid.UUID `json:"entry_ids"`
	Comment  string      `json:"comment,omitempty"`
}

type BatchRejectRequest struct {
	EntryIDs []uuid.UUID `json:"entry_ids"`
	Comment  string      `json:"comment"`
}

type BackupApproverCreateRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Role   Role      `json:"role"`
}
