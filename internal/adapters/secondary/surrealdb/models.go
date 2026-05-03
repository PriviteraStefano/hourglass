package surrealdb

import (
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
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

type SurrealOrganizationMembership struct {
	ID             models.RecordID `json:"id,omitempty"`
	UserID         models.RecordID `json:"user_id"`
	OrganizationID models.RecordID `json:"organization_id"`
	Role           string          `json:"role"`
	IsActive       bool            `json:"is_active"`
	InvitedBy      *string         `json:"invited_by,omitempty"`
	InvitedAt      *time.Time      `json:"invited_at,omitempty"`
	ActivatedAt    *time.Time      `json:"activated_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (m *SurrealOrganizationMembership) ToDomain() *auth.OrganizationMembership {
	if m == nil {
		return nil
	}
	var invitedBy *uuid.UUID
	if m.InvitedBy != nil {
		id, _ := uuid.Parse(*m.InvitedBy)
		invitedBy = &id
	}
	return &auth.OrganizationMembership{
		ID:             recordIDToUUID(m.ID),
		UserID:         recordIDToUUID(m.UserID),
		OrganizationID: recordIDToUUID(m.OrganizationID),
		Role:           m.Role,
		IsActive:       m.IsActive,
		InvitedBy:      invitedBy,
		InvitedAt:      m.InvitedAt,
		ActivatedAt:    m.ActivatedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func SurrealOrganizationMembershipFromDomain(m *auth.OrganizationMembership) *SurrealOrganizationMembership {
	if m == nil {
		return nil
	}
	var invitedBy *string
	if m.InvitedBy != nil {
		s := m.InvitedBy.String()
		invitedBy = &s
	}
	return &SurrealOrganizationMembership{
		ID:             uuidToRecordID("organization_memberships", m.ID),
		UserID:         uuidToRecordID("users", m.UserID),
		OrganizationID: uuidToRecordID("organizations", m.OrganizationID),
		Role:           m.Role,
		IsActive:       m.IsActive,
		InvitedBy:      invitedBy,
		InvitedAt:      m.InvitedAt,
		ActivatedAt:    m.ActivatedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

type SurrealRefreshToken struct {
	ID             models.RecordID `json:"id,omitempty"`
	UserID         models.RecordID `json:"user_id"`
	OrganizationID models.RecordID `json:"organization_id"`
	TokenHash      string          `json:"token_hash"`
	ExpiresAt      time.Time       `json:"expires_at"`
	RevokedAt      *time.Time      `json:"revoked_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
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

func (swg *SurrealWorkingGroup) ToDomain() *working_group.WorkingGroup {
	if swg == nil {
		return nil
	}
	return &working_group.WorkingGroup{
		ID:               recordIDToUUID(swg.ID),
		OrgID:            recordIDToUUID(swg.OrgID),
		SubprojectID:     recordIDToUUID(swg.SubprojectID),
		Name:             swg.Name,
		Description:      swg.Description,
		UnitIDs:          swg.UnitIDs,
		EnforceUnitTuple: swg.EnforceUnitTuple,
		ManagerID:        recordIDToUUID(swg.ManagerID),
		DelegateIDs:      swg.DelegateIDs,
		IsActive:         swg.IsActive,
		CreatedAt:        swg.CreatedAt,
		UpdatedAt:        swg.UpdatedAt,
	}
}

func SurrealWorkingGroupFromDomain(wg *working_group.WorkingGroup) *SurrealWorkingGroup {
	if wg == nil {
		return nil
	}
	return &SurrealWorkingGroup{
		ID:               uuidToRecordID("working_groups", wg.ID),
		OrgID:            uuidToRecordID("organizations", wg.OrgID),
		SubprojectID:     uuidToRecordID("subprojects", wg.SubprojectID),
		Name:             wg.Name,
		Description:      wg.Description,
		UnitIDs:          wg.UnitIDs,
		EnforceUnitTuple: wg.EnforceUnitTuple,
		ManagerID:        uuidToRecordID("users", wg.ManagerID),
		DelegateIDs:      wg.DelegateIDs,
		IsActive:         wg.IsActive,
		CreatedAt:        wg.CreatedAt,
		UpdatedAt:        wg.UpdatedAt,
	}
}

func (swgm *SurrealWorkingGroupMember) ToDomain() *working_group.WorkingGroupMember {
	if swgm == nil {
		return nil
	}
	return &working_group.WorkingGroupMember{
		ID:                  recordIDToUUID(swgm.ID),
		WGID:                recordIDToUUID(swgm.WGID),
		UserID:              recordIDToUUID(swgm.UserID),
		UnitID:              recordIDToUUID(swgm.UnitID),
		Role:                swgm.Role,
		IsDefaultSubproject: swgm.IsDefaultSubproject,
		StartDate:           swgm.StartDate,
		EndDate:             swgm.EndDate,
		CreatedAt:           swgm.CreatedAt,
	}
}

func SurrealWorkingGroupMemberFromDomain(m *working_group.WorkingGroupMember) *SurrealWorkingGroupMember {
	if m == nil {
		return nil
	}
	return &SurrealWorkingGroupMember{
		ID:                  uuidToRecordID("wg_members", m.ID),
		WGID:                uuidToRecordID("working_groups", m.WGID),
		UserID:              uuidToRecordID("users", m.UserID),
		UnitID:              uuidToRecordID("units", m.UnitID),
		Role:                m.Role,
		IsDefaultSubproject: m.IsDefaultSubproject,
		StartDate:           m.StartDate,
		EndDate:             m.EndDate,
		CreatedAt:           m.CreatedAt,
	}
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

func (ste *SurrealTimeEntry) ToDomain() *time_entry.TimeEntry {
	if ste == nil {
		return nil
	}
	return &time_entry.TimeEntry{
		ID:                 recordIDToUUID(ste.ID),
		OrgID:              recordIDToUUID(ste.OrgID),
		UserID:             recordIDToUUID(ste.UserID),
		ProjectID:          recordIDToUUID(ste.ProjectID),
		SubprojectID:       recordIDToUUID(ste.SubprojectID),
		WGID:               recordIDToUUID(ste.WGID),
		UnitID:             recordIDToUUID(ste.UnitID),
		Hours:              ste.Hours,
		Description:        ste.Description,
		EntryDate:          ste.EntryDate,
		Status:             ste.Status,
		IsDeleted:          ste.IsDeleted,
		CreatedFromEntryID: recordIDToUUIDPtr(ste.CreatedFromEntryID),
		CreatedAt:          ste.CreatedAt,
		UpdatedAt:          ste.UpdatedAt,
	}
}

func SurrealTimeEntryFromDomain(e *time_entry.TimeEntry) *SurrealTimeEntry {
	if e == nil {
		return nil
	}
	return &SurrealTimeEntry{
		ID:                 uuidToRecordID("time_entries", e.ID),
		OrgID:              uuidToRecordID("organizations", e.OrgID),
		UserID:             uuidToRecordID("users", e.UserID),
		ProjectID:          uuidToRecordID("projects", e.ProjectID),
		SubprojectID:       uuidToRecordID("subprojects", e.SubprojectID),
		WGID:               uuidToRecordID("working_groups", e.WGID),
		UnitID:             uuidToRecordID("units", e.UnitID),
		Hours:              e.Hours,
		Description:        e.Description,
		EntryDate:          e.EntryDate,
		Status:             e.Status,
		IsDeleted:          e.IsDeleted,
		CreatedFromEntryID: uuidToRecordIDPtr("time_entries", e.CreatedFromEntryID),
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
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

func (sal *SurrealAuditLog) ToDomain() *time_entry.AuditLog {
	if sal == nil {
		return nil
	}
	return &time_entry.AuditLog{
		ID:        recordIDToUUID(sal.ID),
		OrgID:     recordIDToUUID(sal.OrgID),
		EntryID:   sal.EntryID,
		EntryType: sal.EntryType,
		Action:    sal.Action,
		ActorRole: sal.ActorRole,
		ActorID:   recordIDToUUID(sal.ActorID),
		Reason:    sal.Reason,
		Changes:   sal.Changes,
		Timestamp: sal.Timestamp,
	}
}
