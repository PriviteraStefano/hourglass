package surrealdb

import (
	"time"

	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type SurrealUser struct {
	ID           models.RecordID `json:"id,omitempty"`
	Email        string          `json:"email"`
	Username     string          `json:"username"`
	Firstname    string          `json:"firstname"`
	Lastname     string          `json:"lastname"`
	Name         string          `json:"name"`
	PasswordHash string          `json:"password_hash"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type SurrealUserCount struct {
	Count int `json:"count"`
}

func (u *SurrealUser) ToDomain() *auth.User {
	if u == nil {
		return nil
	}
	user := &auth.User{}
	user.ID = recordIDToUUID(u.ID)
	user.Email = u.Email
	user.Username = u.Username
	user.FirstName = u.Firstname
	user.LastName = u.Lastname
	user.Name = u.Name
	user.PasswordHash = u.PasswordHash
	user.IsActive = u.IsActive
	user.CreatedAt = u.CreatedAt
	user.UpdatedAt = u.UpdatedAt
	return user
}

func SurrealUserFromDomain(u *auth.User) *SurrealUser {
	if u == nil {
		return nil
	}
	return &SurrealUser{
		ID:           uuidToRecordID("users", u.ID),
		Email:        u.Email,
		Username:     u.Username,
		Firstname:    u.FirstName,
		Lastname:     u.LastName,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

type SurrealOrganization struct {
	ID                    models.RecordID        `json:"id,omitempty"`
	Name                  string                 `json:"name"`
	Slug                  string                 `json:"slug"`
	Description           string                 `json:"description"`
	FinancialCutoffDays   int                    `json:"financial_cutoff_days"`
	FinancialCutoffConfig map[string]interface{} `json:"financial_cutoff_config"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

func (o *SurrealOrganization) ToDomain() *auth.Organization {
	if o == nil {
		return nil
	}
	org := &auth.Organization{}
	org.ID = recordIDToUUID(o.ID)
	org.Name = o.Name
	org.Slug = o.Slug
	org.Description = o.Description
	org.FinancialCutoffDays = o.FinancialCutoffDays
	org.FinancialCutoffConfig = o.FinancialCutoffConfig
	org.CreatedAt = o.CreatedAt
	org.UpdatedAt = o.UpdatedAt
	return org
}

func SurrealOrganizationFromDomain(o *auth.Organization) *SurrealOrganization {
	if o == nil {
		return nil
	}
	return &SurrealOrganization{
		ID:                    uuidToRecordID("organizations", o.ID),
		Name:                  o.Name,
		Slug:                  o.Slug,
		Description:           o.Description,
		FinancialCutoffDays:   o.FinancialCutoffDays,
		FinancialCutoffConfig: o.FinancialCutoffConfig,
		CreatedAt:             o.CreatedAt,
		UpdatedAt:             o.UpdatedAt,
	}
}

type SurrealRefreshToken struct {
	ID        models.RecordID `json:"id,omitempty"`
	UserID    models.RecordID `json:"user_id"`
	TokenHash string          `json:"token_hash"`
	ExpiresAt time.Time       `json:"expires_at"`
	RevokedAt *time.Time      `json:"revoked_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type QueryResponse[T any] struct {
	Result []T `json:"result"`
}

type QueryResultWrapper struct {
	Result []map[string]any `json:"result"`
}

type SurrealInvitation struct {
	ID             models.RecordID `json:"id,omitempty"`
	OrganizationID models.RecordID `json:"organization_id"`
	Code           string          `json:"code"`
	InviteToken    string          `json:"invite_token"`
	Email          string          `json:"email,omitempty"`
	Status         string          `json:"status"`
	ExpiresAt      time.Time       `json:"expires_at"`
	CreatedBy      string          `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
}

type SurrealPasswordReset struct {
	ID        models.RecordID `json:"id,omitempty"`
	UserID    models.RecordID `json:"user_id"`
	CodeHash  string          `json:"code_hash"`
	ExpiresAt time.Time       `json:"expires_at"`
	UsedAt    *time.Time      `json:"used_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

func (pr *SurrealPasswordReset) ToDomain() *password_reset.PasswordReset {
	if pr == nil {
		return nil
	}
	return &password_reset.PasswordReset{
		ID:        recordIDToUUID(pr.ID),
		UserID:    recordIDToUUID(pr.UserID),
		CodeHash:  pr.CodeHash,
		ExpiresAt: pr.ExpiresAt,
		UsedAt:    pr.UsedAt,
		CreatedAt: pr.CreatedAt,
	}
}

type SurrealUnit struct {
	ID             models.RecordID `json:"id,omitempty"`
	OrgID          models.RecordID `json:"org_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	ParentUnitID   models.RecordID `json:"parent_unit_id,omitempty"`
	HierarchyLevel int             `json:"hierarchy_level"`
	Code           string          `json:"code,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (su *SurrealUnit) ToDomain() *unit.Unit {
	if su == nil {
		return nil
	}
	return &unit.Unit{
		ID:             recordIDToUUID(su.ID),
		OrgID:          recordIDToUUID(su.OrgID),
		Name:           su.Name,
		Description:    su.Description,
		ParentUnitID:   recordIDToUUIDPtr(su.ParentUnitID),
		HierarchyLevel: su.HierarchyLevel,
		Code:           su.Code,
		CreatedAt:      su.CreatedAt,
		UpdatedAt:      su.UpdatedAt,
	}
}

func SurrealUnitFromDomain(u *unit.Unit) *SurrealUnit {
	if u == nil {
		return nil
	}
	return &SurrealUnit{
		ID:             uuidToRecordID("units", u.ID),
		OrgID:          uuidToRecordID("organizations", u.OrgID),
		Name:           u.Name,
		Description:    u.Description,
		ParentUnitID:   uuidToRecordIDPtr("units", u.ParentUnitID),
		HierarchyLevel: u.HierarchyLevel,
		Code:           u.Code,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

type SurrealWorkingGroup struct {
	ID               models.RecordID `json:"id,omitempty"`
	OrgID            models.RecordID `json:"org_id"`
	SubprojectID     models.RecordID `json:"subproject_id"`
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	UnitIDs          []string        `json:"unit_ids"`
	EnforceUnitTuple bool            `json:"enforce_unit_tuple"`
	ManagerID        models.RecordID `json:"manager_id"`
	DelegateIDs      []string        `json:"delegate_ids,omitempty"`
	IsActive         bool            `json:"is_active"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type SurrealWorkingGroupMember struct {
	ID                  models.RecordID `json:"id,omitempty"`
	WGID                models.RecordID `json:"wg_id"`
	UserID              models.RecordID `json:"user_id"`
	UnitID              models.RecordID `json:"unit_id"`
	Role                string          `json:"role"`
	IsDefaultSubproject bool            `json:"is_default_subproject"`
	StartDate           time.Time       `json:"start_date"`
	EndDate             *time.Time      `json:"end_date,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
}

type SurrealTimeEntry struct {
	ID                 models.RecordID `json:"id,omitempty"`
	OrgID              models.RecordID `json:"org_id"`
	UserID             models.RecordID `json:"user_id"`
	ProjectID          models.RecordID `json:"project_id"`
	SubprojectID       models.RecordID `json:"subproject_id"`
	WGID               models.RecordID `json:"wg_id"`
	UnitID             models.RecordID `json:"unit_id"`
	Hours              float64         `json:"hours"`
	Description        string          `json:"description"`
	EntryDate          time.Time       `json:"entry_date"`
	Status             string          `json:"status"`
	IsDeleted          bool            `json:"is_deleted"`
	CreatedFromEntryID models.RecordID `json:"created_from_entry_id,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type SurrealAuditLog struct {
	ID        models.RecordID `json:"id,omitempty"`
	OrgID     models.RecordID `json:"org_id"`
	EntryID   string          `json:"entry_id"`
	EntryType string          `json:"entry_type"`
	Action    string          `json:"action"`
	ActorRole string          `json:"actor_role"`
	ActorID   models.RecordID `json:"actor_id"`
	Reason    string          `json:"reason,omitempty"`
	Changes   map[string]any  `json:"changes,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	IPAddress string          `json:"ip_address,omitempty"`
}
