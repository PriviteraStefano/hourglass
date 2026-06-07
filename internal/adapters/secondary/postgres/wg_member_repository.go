package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
)

// WGMemberRepository handles wg_members table operations.
type WGMemberRepository struct {
	pool *pgxpool.Pool
}

func NewWGMemberRepository(pool *pgxpool.Pool) *WGMemberRepository {
	return &WGMemberRepository{pool: pool}
}

// ListByWG returns all members of a working group.
func (r *WGMemberRepository) ListByWG(ctx context.Context, wgID uuid.UUID) ([]working_group.WorkingGroupMember, error) {
	query := `SELECT id, wg_id, user_id, unit_id, role, is_default_subproject, start_date, end_date, created_at
		FROM wg_members WHERE wg_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, wgID)
	if err != nil {
		return nil, fmt.Errorf("list wg members: %w", err)
	}
	defer rows.Close()

	var members []working_group.WorkingGroupMember
	for rows.Next() {
		var m working_group.WorkingGroupMember
		err := rows.Scan(
			&m.ID, &m.WGID, &m.UserID, &m.UnitID, &m.Role,
			&m.IsDefaultSubproject, &m.StartDate, &m.EndDate, &m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan wg member: %w", err)
		}
		members = append(members, m)
	}
	if members == nil {
		members = []working_group.WorkingGroupMember{}
	}
	return members, rows.Err()
}

// Add inserts a new working group member.
func (r *WGMemberRepository) Add(ctx context.Context, m *working_group.WorkingGroupMember) (*working_group.WorkingGroupMember, error) {
	m.ID = uuid.New()
	query := `INSERT INTO wg_members (id, wg_id, user_id, unit_id, role, is_default_subproject, start_date, end_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, wg_id, user_id, unit_id, role, is_default_subproject, start_date, end_date, created_at`

	var member working_group.WorkingGroupMember
	err := r.pool.QueryRow(ctx, query,
		m.ID, m.WGID, m.UserID, m.UnitID, m.Role,
		m.IsDefaultSubproject, m.StartDate, m.EndDate,
	).Scan(
		&member.ID, &member.WGID, &member.UserID, &member.UnitID, &member.Role,
		&member.IsDefaultSubproject, &member.StartDate, &member.EndDate, &member.CreatedAt,
	)
	if err != nil {
		return nil, wrapPGError(err, "add wg member")
	}
	return &member, nil
}

// Remove deletes a working group member by ID.
func (r *WGMemberRepository) Remove(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM wg_members WHERE id = $1`, id)
	return wrapPGError(err, "remove wg member")
}
