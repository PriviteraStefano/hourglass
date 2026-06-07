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
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Add inserts a new user row.
func (r *UserRepository) Add(ctx context.Context, user *auth.User) error {
	query := `INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Username, user.FirstName, user.LastName,
		user.PasswordHash, user.IsActive, user.CreatedAt, user.UpdatedAt)
	return wrapPGError(err, "add user")
}

// GetByEmail returns the first user matching the given email, or ErrUserNotFound.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	query := `SELECT id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at
		FROM users WHERE email = $1`
	return scanUser(r.pool.QueryRow(ctx, query, email))
}

// GetByUsername returns the first user matching the given username, or ErrUserNotFound.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*auth.User, error) {
	query := `SELECT id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at
		FROM users WHERE username = $1`
	return scanUser(r.pool.QueryRow(ctx, query, username))
}

// GetByID returns the user with the given id, or ErrUserNotFound.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	query := `SELECT id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at
		FROM users WHERE id = $1`
	return scanUser(r.pool.QueryRow(ctx, query, id))
}

// EmailExists returns true if a user with the given email exists.
func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("email exists: %w", err)
	}
	return exists, nil
}

// UsernameExists returns true if a user with the given username exists.
func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("username exists: %w", err)
	}
	return exists, nil
}

// AnyExists returns true if at least one user row exists.
func (r *UserRepository) AnyExists(ctx context.Context) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users)`
	var exists bool
	err := r.pool.QueryRow(ctx, query).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("any user exists: %w", err)
	}
	return exists, nil
}

// UpdatePassword sets the password_hash and bumps updated_at for the given user.
func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, passwordHash, userID)
	return wrapPGError(err, "update password")
}

// GetMemberships returns all organization memberships for the given user.
func (r *UserRepository) GetMemberships(ctx context.Context, userID uuid.UUID) ([]auth.OrganizationMembership, error) {
	query := `SELECT id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at
		FROM organization_memberships WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get memberships: %w", err)
	}
	defer rows.Close()

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

// AddWithMembership creates a user and a membership in a single transaction.
func (r *UserRepository) AddWithMembership(ctx context.Context, user *auth.User, membership *auth.OrganizationMembership) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	userQuery := `INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = tx.Exec(ctx, userQuery,
		user.ID, user.Email, user.Username, user.FirstName, user.LastName,
		user.PasswordHash, user.IsActive, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return wrapPGError(err, "add user with membership")
	}

	membershipQuery := `INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = tx.Exec(ctx, membershipQuery,
		membership.ID, membership.UserID, membership.OrganizationID, membership.Role, membership.IsActive,
		membership.InvitedBy, membership.InvitedAt, membership.ActivatedAt, membership.CreatedAt, membership.UpdatedAt)
	if err != nil {
		return wrapPGError(err, "add membership")
	}

	return tx.Commit(ctx)
}

// AddWithOrgAndMembership creates an organization, a user, and a membership in a single transaction.
func (r *UserRepository) AddWithOrgAndMembership(ctx context.Context, user *auth.User, org *auth.Organization, membership *auth.OrganizationMembership) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	orgConfigJSON, err := json.Marshal(org.FinancialCutoffConfig)
	if err != nil {
		return fmt.Errorf("marshal org config: %w", err)
	}
	orgQuery := `INSERT INTO organizations (id, name, slug, description, financial_cutoff_days, financial_cutoff_config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = tx.Exec(ctx, orgQuery,
		org.ID, org.Name, org.Slug, org.Description, org.FinancialCutoffDays, orgConfigJSON, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return wrapPGError(err, "add org with user")
	}

	userQuery := `INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = tx.Exec(ctx, userQuery,
		user.ID, user.Email, user.Username, user.FirstName, user.LastName,
		user.PasswordHash, user.IsActive, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return wrapPGError(err, "add user with org")
	}

	membershipQuery := `INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = tx.Exec(ctx, membershipQuery,
		membership.ID, membership.UserID, membership.OrganizationID, membership.Role, membership.IsActive,
		membership.InvitedBy, membership.InvitedAt, membership.ActivatedAt, membership.CreatedAt, membership.UpdatedAt)
	if err != nil {
		return wrapPGError(err, "add org membership")
	}

	return tx.Commit(ctx)
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanUser scans a single user row. It maps pgx.ErrNoRows to ports.ErrUserNotFound.
func scanUser(s scanner) (*auth.User, error) {
	var user auth.User
	err := s.Scan(
		&user.ID, &user.Email, &user.Username,
		&user.FirstName, &user.LastName,
		&user.PasswordHash, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &user, nil
}
