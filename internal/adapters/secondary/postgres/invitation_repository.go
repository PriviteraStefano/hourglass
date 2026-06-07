package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
)

// InvitationRepository implements ports.InvitationRepository using a pgxpool.
type InvitationRepository struct {
	pool *pgxpool.Pool
}

func NewInvitationRepository(pool *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{pool: pool}
}

// Create inserts a new invitation and returns it.
func (r *InvitationRepository) Create(ctx context.Context, inv *invitation.Invitation) (*invitation.Invitation, error) {
	inv.ID = uuid.New()

	// created_by is UUID in PG, string in domain — parse at boundary.
	createdBy, err := uuid.Parse(inv.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("parse created_by: %w", err)
	}

	query := `INSERT INTO invitations (id, organization_id, code, invite_token, email, status, created_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, organization_id, code, invite_token, email, status, created_by, expires_at, created_at`
	return scanInvitation(r.pool.QueryRow(ctx, query,
		inv.ID, inv.OrganizationID, inv.Code, inv.InviteToken, inv.Email,
		inv.Status, createdBy, inv.ExpiresAt))
}

// FindByCode looks up an invitation by its code.
// Returns invitation.ErrInvitationNotFound when no row exists.
func (r *InvitationRepository) FindByCode(ctx context.Context, code string) (*invitation.Invitation, error) {
	query := `SELECT id, organization_id, code, invite_token, email, status, created_by, expires_at, created_at
		FROM invitations WHERE code = $1`
	inv, err := scanInvitation(r.pool.QueryRow(ctx, query, code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, invitation.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("find invitation by code: %w", err)
	}
	return inv, nil
}

// FindByToken looks up an invitation by its invite token.
func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*invitation.Invitation, error) {
	query := `SELECT id, organization_id, code, invite_token, email, status, created_by, expires_at, created_at
		FROM invitations WHERE invite_token = $1`
	inv, err := scanInvitation(r.pool.QueryRow(ctx, query, token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, invitation.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("find invitation by token: %w", err)
	}
	return inv, nil
}

// Update modifies an existing invitation (typically status) and returns it.
func (r *InvitationRepository) Update(ctx context.Context, inv *invitation.Invitation) (*invitation.Invitation, error) {
	query := `UPDATE invitations SET status = $1 WHERE id = $2
		RETURNING id, organization_id, code, invite_token, email, status, created_by, expires_at, created_at`
	return scanInvitation(r.pool.QueryRow(ctx, query, inv.Status, inv.ID))
}

// invScanner is satisfied by both pgx.Row and pgx.Rows.
type invScanner interface {
	Scan(dest ...any) error
}

// scanInvitation scans a single invitation row, converting created_by UUID to string.
func scanInvitation(s invScanner) (*invitation.Invitation, error) {
	var inv invitation.Invitation
	var createdBy uuid.UUID

	err := s.Scan(
		&inv.ID, &inv.OrganizationID, &inv.Code, &inv.InviteToken,
		&inv.Email, &inv.Status, &createdBy, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	inv.CreatedBy = createdBy.String()
	return &inv, nil
}
