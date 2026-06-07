package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
)

// OrganizationRepository implements ports.OrganizationRepository using a pgxpool.
type OrganizationRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

// Add inserts a new organization row with financial_cutoff_config as JSONB.
func (r *OrganizationRepository) Add(ctx context.Context, org *auth.Organization) error {
	configJSON, err := json.Marshal(org.FinancialCutoffConfig)
	if err != nil {
		return fmt.Errorf("marshal financial cutoff config: %w", err)
	}
	query := `INSERT INTO organizations (id, name, slug, description, financial_cutoff_days, financial_cutoff_config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = r.pool.Exec(ctx, query,
		org.ID, org.Name, org.Slug, org.Description, org.FinancialCutoffDays, configJSON, org.CreatedAt, org.UpdatedAt)
	return wrapPGError(err, "add organization")
}

// GetByID returns the organization with the given id, or nil (not found).
func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*auth.Organization, error) {
	query := `SELECT id, name, slug, description, financial_cutoff_days, financial_cutoff_config, created_at, updated_at
		FROM organizations WHERE id = $1`
	var org auth.Organization
	var desc string
	var configRaw []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &desc, &org.FinancialCutoffDays,
		&configRaw, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get organization by id: %w", err)
	}
	org.Description = desc
	if configRaw != nil {
		if err := json.Unmarshal(configRaw, &org.FinancialCutoffConfig); err != nil {
			return nil, fmt.Errorf("unmarshal financial cutoff config: %w", err)
		}
	}
	return &org, nil
}

// GetMembership returns the membership for the given user and org, or nil (not found).
func (r *OrganizationRepository) GetMembership(ctx context.Context, userID, orgID uuid.UUID) (*auth.OrganizationMembership, error) {
	query := `SELECT id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at
		FROM organization_memberships WHERE user_id = $1 AND organization_id = $2`
	var m auth.OrganizationMembership
	err := r.pool.QueryRow(ctx, query, userID, orgID).Scan(
		&m.ID, &m.UserID, &m.OrganizationID, &m.Role, &m.IsActive,
		&m.InvitedBy, &m.InvitedAt, &m.ActivatedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get membership: %w", err)
	}
	return &m, nil
}

// AddMembership inserts a new organization membership row.
func (r *OrganizationRepository) AddMembership(ctx context.Context, membership *auth.OrganizationMembership) error {
	query := `INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.pool.Exec(ctx, query,
		membership.ID, membership.UserID, membership.OrganizationID, membership.Role, membership.IsActive,
		membership.InvitedBy, membership.InvitedAt, membership.ActivatedAt, membership.CreatedAt, membership.UpdatedAt)
	return wrapPGError(err, "add membership")
}
