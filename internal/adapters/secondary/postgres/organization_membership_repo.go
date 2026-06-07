package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
)

// OrganizationMembershipRepository provides list queries for memberships.
type OrganizationMembershipRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationMembershipRepository(pool *pgxpool.Pool) *OrganizationMembershipRepository {
	return &OrganizationMembershipRepository{pool: pool}
}

// ListByUser returns all memberships for the given user.
func (r *OrganizationMembershipRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]auth.OrganizationMembership, error) {
	query := `SELECT id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at
		FROM organization_memberships WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships by user: %w", err)
	}
	defer rows.Close()

	return scanMemberships(rows)
}

// ListByOrg returns all memberships for the given organization.
func (r *OrganizationMembershipRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]auth.OrganizationMembership, error) {
	query := `SELECT id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at
		FROM organization_memberships WHERE organization_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list memberships by org: %w", err)
	}
	defer rows.Close()

	return scanMemberships(rows)
}

func scanMemberships(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]auth.OrganizationMembership, error) {
	var memberships []auth.OrganizationMembership
	for rows.Next() {
		var m auth.OrganizationMembership
		err := rows.Scan(
			&m.ID, &m.UserID, &m.OrganizationID, &m.Role, &m.IsActive,
			&m.InvitedBy, &m.InvitedAt, &m.ActivatedAt, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		memberships = append(memberships, m)
	}
	if memberships == nil {
		memberships = []auth.OrganizationMembership{}
	}
	return memberships, rows.Err()
}
