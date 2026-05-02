package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	orgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	"github.com/stefanoprivitera/hourglass/internal/models"
	sdb "github.com/surrealdb/surrealdb.go"
	sdbmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

type OrganizationManagementRepository struct {
	db *sdb.DB
}

func NewOrganizationManagementRepository(db *sdb.DB) *OrganizationManagementRepository {
	return &OrganizationManagementRepository{db: db}
}

type surrealOrgMembership struct {
	ID             sdbmodels.RecordID `json:"id,omitempty"`
	UserID         sdbmodels.RecordID `json:"user_id,omitempty"`
	OrganizationID sdbmodels.RecordID `json:"organization_id"`
	Role           string             `json:"role"`
	IsActive       bool               `json:"is_active"`
	InvitedBy      sdbmodels.RecordID `json:"invited_by,omitempty"`
	InvitedAt      *time.Time         `json:"invited_at,omitempty"`
	ActivatedAt    *time.Time         `json:"activated_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

type surrealOrgSettings struct {
	OrganizationID      sdbmodels.RecordID `json:"organization_id"`
	DefaultKmRate       *float64           `json:"default_km_rate,omitempty"`
	Currency            string             `json:"currency"`
	WeekStartDay        int                `json:"week_start_day"`
	Timezone            string             `json:"timezone"`
	ShowApprovalHistory bool               `json:"show_approval_history"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

func (r *OrganizationManagementRepository) CreateOrganization(ctx context.Context, org *orgdomain.Organization, ownerUserID uuid.UUID, ownerRole models.Role) error {
	orgData := map[string]interface{}{
		"id":         uuidToRecordID("organizations", org.ID),
		"name":       org.Name,
		"slug":       org.Slug,
		"created_at": org.CreatedAt,
		"updated_at": org.CreatedAt,
	}
	if _, err := sdb.Create[SurrealOrganization](ctx, r.db, sdbmodels.Table("organizations"), orgData); err != nil {
		return wrapErr(err, "create organization")
	}

	membership := map[string]interface{}{
		"id":              uuidToRecordID("organization_memberships", uuid.New()),
		"user_id":         uuidToRecordID("users", ownerUserID),
		"organization_id": uuidToRecordID("organizations", org.ID),
		"role":            string(ownerRole),
		"is_active":       true,
		"activated_at":    org.CreatedAt,
		"created_at":      org.CreatedAt,
	}
	if _, err := sdb.Create[surrealOrgMembership](ctx, r.db, sdbmodels.Table("organization_memberships"), membership); err != nil {
		return wrapErr(err, "create organization membership")
	}
	return nil
}

func (r *OrganizationManagementRepository) GetOrganization(ctx context.Context, id uuid.UUID) (*orgdomain.Organization, error) {
	recordID := uuidToRecordID("organizations", id)
	result, err := sdb.Select[SurrealOrganization](ctx, r.db, recordID)
	if err != nil {
		return nil, orgdomain.ErrOrganizationNotFound
	}
	return &orgdomain.Organization{
		ID:        recordIDToUUID(result.ID),
		Name:      result.Name,
		Slug:      result.Slug,
		CreatedAt: result.CreatedAt,
	}, nil
}

func (r *OrganizationManagementRepository) InviteMember(ctx context.Context, orgID uuid.UUID, req *orgdomain.InviteRequest, invitedBy uuid.UUID) (uuid.UUID, time.Time, error) {
	id := uuid.New()
	now := time.Now()
	data := map[string]interface{}{
		"id":              uuidToRecordID("organization_memberships", id),
		"user_id":         nil,
		"organization_id": uuidToRecordID("organizations", orgID),
		"role":            string(req.Role),
		"is_active":       true,
		"invited_by":      uuidToRecordID("users", invitedBy),
		"invited_at":      now,
		"created_at":      now,
	}
	if _, err := sdb.Create[surrealOrgMembership](ctx, r.db, sdbmodels.Table("organization_memberships"), data); err != nil {
		return uuid.Nil, time.Time{}, wrapErr(err, "invite member")
	}
	return id, now, nil
}

func (r *OrganizationManagementRepository) GetSettings(ctx context.Context, orgID uuid.UUID) (*orgdomain.Settings, error) {
	results, err := sdb.Query[[]surrealOrgSettings](ctx, r.db,
		`SELECT * FROM organization_settings WHERE organization_id = $org_id LIMIT 1`,
		map[string]interface{}{"org_id": uuidToRecordID("organizations", orgID)})
	if err != nil || results == nil || len(*results) == 0 || len((*results)[0].Result) == 0 {
		return nil, orgdomain.ErrOrganizationNotFound
	}
	s := (*results)[0].Result[0]
	return &orgdomain.Settings{
		OrganizationID:      recordIDToUUID(s.OrganizationID),
		DefaultKmRate:       s.DefaultKmRate,
		Currency:            s.Currency,
		WeekStartDay:        s.WeekStartDay,
		Timezone:            s.Timezone,
		ShowApprovalHistory: s.ShowApprovalHistory,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}, nil
}

func (r *OrganizationManagementRepository) UpdateSettings(ctx context.Context, orgID uuid.UUID, req *orgdomain.UpdateSettingsRequest) (*orgdomain.Settings, error) {
	recordID := uuidToRecordID("organizations", orgID)
	_, _ = sdb.Create[surrealOrgSettings](ctx, r.db, sdbmodels.Table("organization_settings"), map[string]interface{}{
		"organization_id":       recordID,
		"currency":              "EUR",
		"week_start_day":        1,
		"timezone":              "UTC",
		"show_approval_history": true,
		"created_at":            time.Now(),
		"updated_at":            time.Now(),
	})

	results, err := sdb.Query[[]surrealOrgSettings](ctx, r.db,
		`UPDATE organization_settings SET
			default_km_rate = COALESCE($default_km_rate, default_km_rate),
			currency = IF $currency = NONE OR $currency = "" THEN currency ELSE $currency END,
			week_start_day = COALESCE($week_start_day, week_start_day),
			timezone = IF $timezone = NONE OR $timezone = "" THEN timezone ELSE $timezone END,
			show_approval_history = COALESCE($show_approval_history, show_approval_history),
			updated_at = $updated_at
		WHERE organization_id = $org_id RETURN AFTER`,
		map[string]interface{}{
			"org_id":                recordID,
			"default_km_rate":       req.DefaultKmRate,
			"currency":              req.Currency,
			"week_start_day":        req.WeekStartDay,
			"timezone":              req.Timezone,
			"show_approval_history": req.ShowApprovalHistory,
			"updated_at":            time.Now(),
		})
	if err != nil || results == nil || len(*results) == 0 || len((*results)[0].Result) == 0 {
		return nil, wrapErr(err, "update org settings")
	}
	s := (*results)[0].Result[0]
	return &orgdomain.Settings{
		OrganizationID:      recordIDToUUID(s.OrganizationID),
		DefaultKmRate:       s.DefaultKmRate,
		Currency:            s.Currency,
		WeekStartDay:        s.WeekStartDay,
		Timezone:            s.Timezone,
		ShowApprovalHistory: s.ShowApprovalHistory,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}, nil
}

func (r *OrganizationManagementRepository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]orgdomain.Member, error) {
	type row struct {
		ID          sdbmodels.RecordID `json:"id"`
		UserID      sdbmodels.RecordID `json:"user_id,omitempty"`
		Role        string             `json:"role"`
		IsActive    bool               `json:"is_active"`
		InvitedBy   sdbmodels.RecordID `json:"invited_by,omitempty"`
		InvitedAt   *time.Time         `json:"invited_at,omitempty"`
		ActivatedAt *time.Time         `json:"activated_at,omitempty"`
		UserName    string             `json:"user_name,omitempty"`
		UserEmail   string             `json:"user_email,omitempty"`
	}
	results, err := sdb.Query[[]row](ctx, r.db, `
		SELECT id, user_id, role, is_active, invited_by, invited_at, activated_at,
			(SELECT VALUE name FROM users WHERE id = user_id LIMIT 1)[0] AS user_name,
			(SELECT VALUE email FROM users WHERE id = user_id LIMIT 1)[0] AS user_email
		FROM organization_memberships
		WHERE organization_id = $org_id
		ORDER BY created_at DESC
	`, map[string]interface{}{"org_id": uuidToRecordID("organizations", orgID)})
	if err != nil {
		return nil, wrapErr(err, "list members")
	}
	if results == nil || len(*results) == 0 {
		return []orgdomain.Member{}, nil
	}
	out := make([]orgdomain.Member, 0, len((*results)[0].Result))
	for _, m := range (*results)[0].Result {
		member := orgdomain.Member{
			ID:          recordIDToUUID(m.ID),
			Role:        models.Role(m.Role),
			IsActive:    m.IsActive,
			InvitedAt:   m.InvitedAt,
			ActivatedAt: m.ActivatedAt,
			UserName:    m.UserName,
			UserEmail:   m.UserEmail,
		}
		if uid := recordIDToUUIDPtr(m.UserID); uid != nil {
			member.UserID = uid
		}
		if ib := recordIDToUUIDPtr(m.InvitedBy); ib != nil {
			member.InvitedBy = ib
		}
		out = append(out, member)
	}
	return out, nil
}

func (r *OrganizationManagementRepository) UpdateMemberRole(ctx context.Context, orgID, memberID uuid.UUID, role models.Role) error {
	recordID := uuidToRecordID("organization_memberships", memberID)
	_, err := sdb.Merge[surrealOrgMembership](ctx, r.db, recordID, map[string]interface{}{
		"organization_id": uuidToRecordID("organizations", orgID),
		"role":            string(role),
		"is_active":       true,
	})
	return wrapErr(err, "update member role")
}

func (r *OrganizationManagementRepository) DeactivateMember(ctx context.Context, orgID, memberID uuid.UUID) error {
	recordID := uuidToRecordID("organization_memberships", memberID)
	_, err := sdb.Merge[surrealOrgMembership](ctx, r.db, recordID, map[string]interface{}{
		"organization_id": uuidToRecordID("organizations", orgID),
		"is_active":       false,
	})
	return wrapErr(err, "deactivate member")
}

func (r *OrganizationManagementRepository) CountActiveFinance(ctx context.Context, orgID uuid.UUID) (int, error) {
	results, err := sdb.Query[[]map[string]interface{}](ctx, r.db, `
		SELECT count() FROM organization_memberships
		WHERE organization_id = $org_id AND role = $role AND is_active = true GROUP ALL
	`, map[string]interface{}{"org_id": uuidToRecordID("organizations", orgID), "role": string(models.RoleFinance)})
	if err != nil {
		return 0, wrapErr(err, "count finance members")
	}
	if results == nil || len(*results) == 0 || len((*results)[0].Result) == 0 {
		return 0, nil
	}
	if count, ok := (*results)[0].Result[0]["count"].(float64); ok {
		return int(count), nil
	}
	return 0, nil
}

func (r *OrganizationManagementRepository) GetMemberRole(ctx context.Context, memberID uuid.UUID) (models.Role, error) {
	recordID := uuidToRecordID("organization_memberships", memberID)
	result, err := sdb.Select[surrealOrgMembership](ctx, r.db, recordID)
	if err != nil {
		return "", orgdomain.ErrMemberNotFound
	}
	return models.Role(result.Role), nil
}
