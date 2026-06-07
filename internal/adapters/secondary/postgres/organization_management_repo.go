package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	orgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// OrganizationManagementRepository implements ports.OrganizationManagementRepository.
type OrganizationManagementRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationManagementRepository(pool *pgxpool.Pool) *OrganizationManagementRepository {
	return &OrganizationManagementRepository{pool: pool}
}

// CreateOrganization inserts an org and an owner membership in a single transaction.
func (r *OrganizationManagementRepository) CreateOrganization(ctx context.Context, org *orgdomain.Organization, ownerUserID uuid.UUID, ownerRole models.Role) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	orgQuery := `INSERT INTO organizations (id, name, slug, created_at) VALUES ($1, $2, $3, $4)`
	_, err = tx.Exec(ctx, orgQuery, org.ID, org.Name, org.Slug, org.CreatedAt)
	if err != nil {
		return wrapPGError(err, "create organization")
	}

	now := time.Now()
	membershipQuery := `INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, activated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.Exec(ctx, membershipQuery,
		uuid.New(), ownerUserID, org.ID, string(ownerRole), true, now, now)
	if err != nil {
		return wrapPGError(err, "create owner membership")
	}

	return tx.Commit(ctx)
}

// GetOrganization returns the basic org info.
func (r *OrganizationManagementRepository) GetOrganization(ctx context.Context, id uuid.UUID) (*orgdomain.Organization, error) {
	query := `SELECT id, name, slug, created_at FROM organizations WHERE id = $1`
	var org orgdomain.Organization
	err := r.pool.QueryRow(ctx, query, id).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, orgdomain.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return &org, nil
}

// InviteMember inserts a pending membership with user_id=NULL.
func (r *OrganizationManagementRepository) InviteMember(ctx context.Context, orgID uuid.UUID, req *orgdomain.InviteRequest, invitedBy uuid.UUID) (uuid.UUID, time.Time, error) {
	id := uuid.New()
	now := time.Now()
	query := `INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, invited_by, invited_at, created_at)
		VALUES ($1, NULL, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query,
		id, orgID, string(req.Role), true, invitedBy, now, now)
	if err != nil {
		return uuid.Nil, time.Time{}, wrapPGError(err, "invite member")
	}
	return id, now, nil
}

// GetSettings reads the organization_settings row.
func (r *OrganizationManagementRepository) GetSettings(ctx context.Context, orgID uuid.UUID) (*orgdomain.Settings, error) {
	query := `SELECT organization_id, default_km_rate, currency, week_start_day, timezone, show_approval_history, created_at, updated_at
		FROM organization_settings WHERE organization_id = $1`
	var s orgdomain.Settings
	err := r.pool.QueryRow(ctx, query, orgID).Scan(
		&s.OrganizationID, &s.DefaultKmRate, &s.Currency, &s.WeekStartDay,
		&s.Timezone, &s.ShowApprovalHistory, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, orgdomain.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return &s, nil
}

// UpdateSettings applies partial updates via COALESCE logic and returns the new row.
func (r *OrganizationManagementRepository) UpdateSettings(ctx context.Context, orgID uuid.UUID, req *orgdomain.UpdateSettingsRequest) (*orgdomain.Settings, error) {
	query := `UPDATE organization_settings SET
		default_km_rate = CASE WHEN $1::numeric IS NULL THEN default_km_rate ELSE $1 END,
		currency = CASE WHEN $2 = '' OR $2 IS NULL THEN currency ELSE $2 END,
		week_start_day = CASE WHEN $3::smallint IS NULL THEN week_start_day ELSE $3 END,
		timezone = CASE WHEN $4 = '' OR $4 IS NULL THEN timezone ELSE $4 END,
		show_approval_history = CASE WHEN $5::boolean IS NULL THEN show_approval_history ELSE $5 END,
		updated_at = NOW()
	WHERE organization_id = $6
	RETURNING organization_id, default_km_rate, currency, week_start_day, timezone, show_approval_history, created_at, updated_at`

	var s orgdomain.Settings
	err := r.pool.QueryRow(ctx, query,
		req.DefaultKmRate, req.Currency, req.WeekStartDay, req.Timezone, req.ShowApprovalHistory, orgID,
	).Scan(
		&s.OrganizationID, &s.DefaultKmRate, &s.Currency, &s.WeekStartDay,
		&s.Timezone, &s.ShowApprovalHistory, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, orgdomain.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("update settings: %w", err)
	}
	return &s, nil
}

// ListMembers returns all members of an org with joined user info.
func (r *OrganizationManagementRepository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]orgdomain.Member, error) {
	query := `SELECT om.id, om.user_id, om.role, om.is_active, om.invited_by, om.invited_at, om.activated_at,
		COALESCE(u.firstname || ' ' || u.lastname, '') AS user_name,
		COALESCE(u.email, '') AS user_email
	FROM organization_memberships om
	LEFT JOIN users u ON om.user_id = u.id
	WHERE om.organization_id = $1
	ORDER BY om.created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []orgdomain.Member
	for rows.Next() {
		var m orgdomain.Member
		err := rows.Scan(
			&m.ID, &m.UserID, &m.Role, &m.IsActive,
			&m.InvitedBy, &m.InvitedAt, &m.ActivatedAt,
			&m.UserName, &m.UserEmail,
		)
		if err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	if members == nil {
		members = []orgdomain.Member{}
	}
	return members, rows.Err()
}

// UpdateMemberRole changes the role of a specific membership.
func (r *OrganizationManagementRepository) UpdateMemberRole(ctx context.Context, orgID, memberID uuid.UUID, role models.Role) error {
	query := `UPDATE organization_memberships SET role = $1 WHERE id = $2 AND organization_id = $3`
	_, err := r.pool.Exec(ctx, query, string(role), memberID, orgID)
	return wrapPGError(err, "update member role")
}

// DeactivateMember sets is_active to false.
func (r *OrganizationManagementRepository) DeactivateMember(ctx context.Context, orgID, memberID uuid.UUID) error {
	query := `UPDATE organization_memberships SET is_active = false WHERE id = $1 AND organization_id = $2`
	_, err := r.pool.Exec(ctx, query, memberID, orgID)
	return wrapPGError(err, "deactivate member")
}

// CountActiveFinance returns the count of active finance members in an org.
func (r *OrganizationManagementRepository) CountActiveFinance(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM organization_memberships
		WHERE organization_id = $1 AND role = 'finance' AND is_active = true`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active finance: %w", err)
	}
	return count, nil
}

// GetMemberRole returns the role for a specific membership record.
func (r *OrganizationManagementRepository) GetMemberRole(ctx context.Context, memberID uuid.UUID) (models.Role, error) {
	query := `SELECT role FROM organization_memberships WHERE id = $1`
	var role string
	err := r.pool.QueryRow(ctx, query, memberID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", orgdomain.ErrMemberNotFound
		}
		return "", fmt.Errorf("get member role: %w", err)
	}
	return models.Role(role), nil
}
