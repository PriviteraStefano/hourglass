package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
)

// UnitMemberRepository handles unit_memberships table operations.
type UnitMemberRepository struct {
	pool *pgxpool.Pool
}

func NewUnitMemberRepository(pool *pgxpool.Pool) *UnitMemberRepository {
	return &UnitMemberRepository{pool: pool}
}

// ListByUnit returns all members of a unit with user name and email via LEFT JOIN.
func (r *UnitMemberRepository) ListByUnit(ctx context.Context, unitID string) ([]unit.UnitMember, error) {
	uid, err := uuid.Parse(unitID)
	if err != nil {
		return nil, fmt.Errorf("parse unit id: %w", err)
	}
	query := `SELECT um.id, um.org_id, um.user_id, um.unit_id, um.is_primary, um.role,
		um.start_date, um.end_date, um.created_at,
		COALESCE(u.firstname || ' ' || u.lastname, '') AS user_name,
		COALESCE(u.email, '') AS user_email
		FROM unit_memberships um
		LEFT JOIN users u ON um.user_id = u.id
		WHERE um.unit_id = $1
		ORDER BY um.created_at DESC`
	rows, err := r.pool.Query(ctx, query, uid)
	if err != nil {
		return nil, fmt.Errorf("list members by unit: %w", err)
	}
	defer rows.Close()

	var members []unit.UnitMember
	for rows.Next() {
		m, err := scanUnitMember(rows)
		if err != nil {
			return nil, fmt.Errorf("scan unit member: %w", err)
		}
		members = append(members, *m)
	}
	if members == nil {
		members = []unit.UnitMember{}
	}
	return members, rows.Err()
}

// Add inserts a new unit membership and returns it.
func (r *UnitMemberRepository) Add(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error) {
	id := uuid.New()
	m.ID = id.String()

	unitID, err := uuid.Parse(m.UnitID)
	if err != nil {
		return nil, fmt.Errorf("parse unit id: %w", err)
	}

	query := `INSERT INTO unit_memberships (id, org_id, user_id, unit_id, is_primary, role, start_date, end_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, org_id, user_id, unit_id, is_primary, role, start_date, end_date, created_at`

	var sqlID uuid.UUID
	var sqlUnitID uuid.UUID
	var um unit.UnitMember
	err = r.pool.QueryRow(ctx, query,
		id, m.OrgID, m.UserID, unitID, m.IsPrimary, m.Role, m.StartDate, m.EndDate,
	).Scan(
		&sqlID, &um.OrgID, &um.UserID, &sqlUnitID, &um.IsPrimary, &um.Role,
		&um.StartDate, &um.EndDate, &um.CreatedAt,
	)
	if err != nil {
		return nil, wrapPGError(err, "add unit member")
	}
	um.ID = sqlID.String()
	um.UnitID = sqlUnitID.String()
	return &um, nil
}

// Remove deletes a unit membership by ID.
func (r *UnitMemberRepository) Remove(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse member id: %w", err)
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM unit_memberships WHERE id = $1`, uid)
	return wrapPGError(err, "remove unit member")
}

// Update updates a unit membership (is_primary, end_date) and returns it.
func (r *UnitMemberRepository) Update(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error) {
	id, err := uuid.Parse(m.ID)
	if err != nil {
		return nil, fmt.Errorf("parse member id: %w", err)
	}

	query := `UPDATE unit_memberships
		SET is_primary = $2, end_date = $3
		WHERE id = $1
		RETURNING id, org_id, user_id, unit_id, is_primary, role, start_date, end_date, created_at`

	var sqlID uuid.UUID
	var sqlUnitID uuid.UUID
	var um unit.UnitMember
	err = r.pool.QueryRow(ctx, query, id, m.IsPrimary, m.EndDate).Scan(
		&sqlID, &um.OrgID, &um.UserID, &sqlUnitID, &um.IsPrimary, &um.Role,
		&um.StartDate, &um.EndDate, &um.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, unit.ErrMemberNotFound
		}
		return nil, wrapPGError(err, "update unit member")
	}
	um.ID = sqlID.String()
	um.UnitID = sqlUnitID.String()
	return &um, nil
}

// ListByUnitIDs returns members for multiple unit IDs at once using ANY($1).
func (r *UnitMemberRepository) ListByUnitIDs(ctx context.Context, orgID uuid.UUID, unitIDs []string) ([]unit.UnitMember, error) {
	ids := make([]uuid.UUID, 0, len(unitIDs))
	for _, id := range unitIDs {
		uid, err := uuid.Parse(strings.TrimSpace(id))
		if err != nil {
			continue
		}
		ids = append(ids, uid)
	}
	if len(ids) == 0 {
		return []unit.UnitMember{}, nil
	}

	query := `SELECT um.id, um.org_id, um.user_id, um.unit_id, um.is_primary, um.role,
		um.start_date, um.end_date, um.created_at,
		COALESCE(u.firstname || ' ' || u.lastname, '') AS user_name,
		COALESCE(u.email, '') AS user_email
		FROM unit_memberships um
		LEFT JOIN users u ON um.user_id = u.id
		WHERE um.unit_id = ANY($1) AND um.org_id = $2
		ORDER BY um.unit_id, um.created_at DESC`

	rows, err := r.pool.Query(ctx, query, ids, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members by unit ids: %w", err)
	}
	defer rows.Close()

	var members []unit.UnitMember
	for rows.Next() {
		m, err := scanUnitMember(rows)
		if err != nil {
			return nil, fmt.Errorf("scan unit member: %w", err)
		}
		members = append(members, *m)
	}
	if members == nil {
		members = []unit.UnitMember{}
	}
	return members, rows.Err()
}

// ListMembershipsForUser returns all unit memberships for a user across all units.
func (r *UnitMemberRepository) ListMembershipsForUser(ctx context.Context, userID uuid.UUID) ([]unit.UnitMember, error) {
	query := `SELECT um.id, um.org_id, um.user_id, um.unit_id, um.is_primary, um.role,
		um.start_date, um.end_date, um.created_at,
		COALESCE(u.firstname || ' ' || u.lastname, '') AS user_name,
		COALESCE(u.email, '') AS user_email
		FROM unit_memberships um
		LEFT JOIN users u ON um.user_id = u.id
		WHERE um.user_id = $1
		ORDER BY um.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships for user: %w", err)
	}
	defer rows.Close()

	var members []unit.UnitMember
	for rows.Next() {
		m, err := scanUnitMember(rows)
		if err != nil {
			return nil, fmt.Errorf("scan unit member: %w", err)
		}
		members = append(members, *m)
	}
	if members == nil {
		members = []unit.UnitMember{}
	}
	return members, rows.Err()
}

// memberScanner is satisfied by both pgx.Row and pgx.Rows.
type memberScanner interface {
	Scan(dest ...any) error
}

// scanUnitMember scans a single unit member row (with user_name and user_email from JOIN).
func scanUnitMember(s memberScanner) (*unit.UnitMember, error) {
	var m unit.UnitMember
	var sqlID uuid.UUID
	var sqlUnitID uuid.UUID

	err := s.Scan(
		&sqlID, &m.OrgID, &m.UserID, &sqlUnitID, &m.IsPrimary, &m.Role,
		&m.StartDate, &m.EndDate, &m.CreatedAt,
		&m.UserName, &m.UserEmail,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, unit.ErrMemberNotFound
		}
		return nil, fmt.Errorf("scan unit member: %w", err)
	}
	m.ID = sqlID.String()
	m.UnitID = sqlUnitID.String()
	return &m, nil
}
