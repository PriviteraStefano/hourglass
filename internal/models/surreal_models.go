package models

import "time"

// SurrealDB Record ID type (e.g., "users:abc123")
type RecordID string

// Organization (SurrealDB schema)
type SurrOrganization struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Slug                string                 `json:"slug"`
	Description         string                 `json:"description,omitempty"`
	FinancialCutoffDays int                    `json:"financial_cutoff_days"`
	FinancialCutoffCfg  map[string]interface{} `json:"financial_cutoff_config"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// Unit (org hierarchy with unlimited nesting)
type Unit struct {
	ID             string    `json:"id"`
	OrgID          string    `json:"org_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	ParentUnitID   *string   `json:"parent_unit_id,omitempty"`
	HierarchyLevel int       `json:"hierarchy_level"`
	Code           string    `json:"code,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UnitMembership (users → units, many-to-many)
type UnitMembership struct {
	ID        string     `json:"id"`
	OrgID     string     `json:"org_id"`
	UserID    string     `json:"user_id"`
	UnitID    string     `json:"unit_id"`
	IsPrimary bool       `json:"is_primary"`
	Role      string     `json:"role"` // employee, manager, finance
	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Subproject (structure level under projects)
type Subproject struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	SequenceOrder int       `json:"sequence_order"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// WorkingGroup (execution teams)
type WorkingGroup struct {
	ID               string    `json:"id"`
	OrgID            string    `json:"org_id"`
	SubprojectID     string    `json:"subproject_id"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	UnitIDs          []string  `json:"unit_ids"`
	EnforceUnitTuple bool      `json:"enforce_unit_tuple"`
	ManagerID        string    `json:"manager_id"`
	DelegateIDs      []string  `json:"delegate_ids,omitempty"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// WorkingGroupMember (users assigned to WG from units)
type WorkingGroupMember struct {
	ID                  string     `json:"id"`
	WGID                string     `json:"wg_id"`
	UserID              string     `json:"user_id"`
	UnitID              string     `json:"unit_id"`
	Role                string     `json:"role"`
	IsDefaultSubproject bool       `json:"is_default_subproject"`
	StartDate           time.Time  `json:"start_date"`
	EndDate             *time.Time `json:"end_date,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// SurrTimeEntry (SurrealDB schema - simplified status)
type SurrTimeEntry struct {
	ID                 string    `json:"id"`
	OrgID              string    `json:"org_id"`
	UserID             string    `json:"user_id"`
	ProjectID          string    `json:"project_id"`
	SubprojectID       string    `json:"subproject_id"`
	WGID               string    `json:"wg_id"`
	UnitID             string    `json:"unit_id"`
	Hours              float64   `json:"hours"`
	Description        string    `json:"description"`
	EntryDate          time.Time `json:"entry_date"`
	Status             string    `json:"status"` // draft, submitted, approved
	IsDeleted          bool      `json:"is_deleted"`
	CreatedFromEntryID *string   `json:"created_from_entry_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// AuditLog (unified for time entries and expenses)
type AuditLog struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	EntryID   string    `json:"entry_id"`
	EntryType string    `json:"entry_type"` // time_entry, expense
	Action    string    `json:"action"`     // created, submitted, approved, rejected, split, moved, reallocated, edited, finance_override, reverted
	ActorRole string    `json:"actor_role"` // user, wg_manager, org_manager, finance, admin
	ActorID   string    `json:"actor_id"`
	Reason    *string   `json:"reason,omitempty"`
	Changes   *string   `json:"changes,omitempty"` // JSON of before/after
	Timestamp time.Time `json:"timestamp"`
	IPAddress *string   `json:"ip_address,omitempty"`
}

// SurrExpense (SurrealDB schema)
type SurrExpense struct {
	ID             string    `json:"id"`
	OrgID          string    `json:"org_id"`
	UserID         string    `json:"user_id"`
	ProjectID      *string   `json:"project_id,omitempty"`
	UnitID         string    `json:"unit_id"`
	Category       string    `json:"category"` // mileage, meal, accommodation, other
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Description    string    `json:"description,omitempty"`
	ExpenseDate    time.Time `json:"expense_date"`
	ReceiptURL     *string   `json:"receipt_url,omitempty"`
	ReceiptOcrData *string   `json:"receipt_ocr_data,omitempty"`
	Status         string    `json:"status"` // draft, submitted, approved, rejected
	IsDeleted      bool      `json:"is_deleted"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// FinancialCutoffPeriod
type FinancialCutoffPeriod struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	ProjectID   *string   `json:"project_id,omitempty"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	CutoffDate  time.Time `json:"cutoff_date"`
	IsLocked    bool      `json:"is_locked"`
	CreatedAt   time.Time `json:"created_at"`
}

// BudgetCap
type BudgetCap struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	UserID      *string   `json:"user_id,omitempty"`
	ProjectID   *string   `json:"project_id,omitempty"`
	Category    *string   `json:"category,omitempty"`
	LimitAmount float64   `json:"limit_amount"`
	Period      string    `json:"period"` // daily, weekly, monthly, yearly
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
}

// Request/Response types for new handlers

type CreateUnitRequest struct {
	OrgID        string  `json:"org_id"`
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	ParentUnitID *string `json:"parent_unit_id,omitempty"`
	Code         string  `json:"code,omitempty"`
}

type UpdateUnitRequest struct {
	Name         string  `json:"name,omitempty"`
	Description  string  `json:"description,omitempty"`
	ParentUnitID *string `json:"parent_unit_id,omitempty"`
	Code         string  `json:"code,omitempty"`
}

type CreateUnitMembershipRequest struct {
	OrgID     string `json:"org_id"`
	UserID    string `json:"user_id"`
	UnitID    string `json:"unit_id"`
	IsPrimary bool   `json:"is_primary"`
	Role      string `json:"role"`
}

type CreateWorkingGroupRequest struct {
	OrgID            string   `json:"org_id"`
	SubprojectID     string   `json:"subproject_id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	UnitIDs          []string `json:"unit_ids"`
	EnforceUnitTuple bool     `json:"enforce_unit_tuple"`
	ManagerID        string   `json:"manager_id"`
	DelegateIDs      []string `json:"delegate_ids,omitempty"`
}

type AddWGMemberRequest struct {
	WGID                string `json:"wg_id"`
	UserID              string `json:"user_id"`
	UnitID              string `json:"unit_id"`
	Role                string `json:"role"`
	IsDefaultSubproject bool   `json:"is_default_subproject"`
}

// Unit tree node for hierarchical queries
type UnitTreeNode struct {
	Unit     Unit           `json:"unit"`
	Children []UnitTreeNode `json:"children,omitempty"`
}

// Status constants for simplified workflow (SurrealDB)
const (
	SurrStatusDraft     = "draft"
	SurrStatusSubmitted = "submitted"
	SurrStatusApproved  = "approved"
	SurrStatusRejected  = "rejected"
)

// Entry type constants
const (
	EntryTypeTimeEntry = "time_entry"
	EntryTypeExpense   = "expense"
)

// Audit action constants
const (
	ActionCreated         = "created"
	ActionSubmitted       = "submitted"
	ActionApproved        = "approved"
	ActionRejected        = "rejected"
	ActionSplit           = "split"
	ActionMoved           = "moved"
	ActionReallocated     = "reallocated"
	ActionEdited          = "edited"
	ActionFinanceOverride = "finance_override"
	ActionReverted        = "reverted"
)

// Actor role constants
const (
	ActorRoleUser       = "user"
	ActorRoleWGManager  = "wg_manager"
	ActorRoleOrgManager = "org_manager"
	ActorRoleFinance    = "finance"
	ActorRoleAdmin      = "admin"
)
