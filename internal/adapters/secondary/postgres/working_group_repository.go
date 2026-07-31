package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
)

// WorkingGroupRepository implements ports.WorkingGroupRepository using a pgxpool.
// Working groups anchor to activities at any depth (ADR-P-007 D-5); the
// legacy unit-tuple toggle column was dropped in migration 011 (ADR-P-001 Q3).
type WorkingGroupRepository struct {
	pool    *pgxpool.Pool
	members *WGMemberRepository
}

func NewWorkingGroupRepository(pool *pgxpool.Pool) *WorkingGroupRepository {
	return &WorkingGroupRepository{
		pool:    pool,
		members: NewWGMemberRepository(pool),
	}
}

// ListByOrg returns working groups for the given org, optionally filtered by activity.
func (r *WorkingGroupRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, activityID *uuid.UUID) ([]working_group.WorkingGroup, error) {
	var query string
	var args []interface{}
	if activityID != nil {
		query = `SELECT id, org_id, activity_id, name, description, unit_ids,
			manager_id, delegate_ids, is_active, created_at, updated_at
			FROM working_groups WHERE org_id = $1 AND activity_id = $2 ORDER BY name`
		args = []interface{}{orgID, *activityID}
	} else {
		query = `SELECT id, org_id, activity_id, name, description, unit_ids,
			manager_id, delegate_ids, is_active, created_at, updated_at
			FROM working_groups WHERE org_id = $1 ORDER BY name`
		args = []interface{}{orgID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list working groups by org: %w", err)
	}
	defer rows.Close()

	var wgs []working_group.WorkingGroup
	for rows.Next() {
		wg, err := scanWorkingGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan working group: %w", err)
		}
		wgs = append(wgs, *wg)
	}
	if wgs == nil {
		wgs = []working_group.WorkingGroup{}
	}
	return wgs, rows.Err()
}

// GetByID returns a working group by ID, or working_group.ErrWorkingGroupNotFound.
func (r *WorkingGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*working_group.WorkingGroup, error) {
	query := `SELECT id, org_id, activity_id, name, description, unit_ids,
		manager_id, delegate_ids, is_active, created_at, updated_at
		FROM working_groups WHERE id = $1`
	wg, err := scanWorkingGroup(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, working_group.ErrWorkingGroupNotFound
		}
		return nil, fmt.Errorf("get working group by id: %w", err)
	}
	return wg, nil
}

// Create inserts a new working group and returns it with generated fields.
func (r *WorkingGroupRepository) Create(ctx context.Context, wg *working_group.WorkingGroup) (*working_group.WorkingGroup, error) {
	wg.ID = uuid.New()

	unitIDs := toUUIDArray(wg.UnitIDs)
	delegateIDs := toUUIDArray(wg.DelegateIDs)

	query := `INSERT INTO working_groups (id, org_id, activity_id, name, description,
		unit_ids, manager_id, delegate_ids, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, org_id, activity_id, name, description, unit_ids,
			manager_id, delegate_ids, is_active, created_at, updated_at`
	return scanWorkingGroup(r.pool.QueryRow(ctx, query,
		wg.ID, wg.OrgID, wg.SubprojectID, wg.Name, wg.Description,
		unitIDs, wg.ManagerID, delegateIDs, wg.IsActive))
}

// Update dynamically builds a SET clause for a working group.
func (r *WorkingGroupRepository) Update(ctx context.Context, wg *working_group.WorkingGroup) (*working_group.WorkingGroup, error) {
	sets := []string{
		"name = $2",
		"description = $3",
		"unit_ids = $4",
		"manager_id = $5",
		"delegate_ids = $6",
	}
	args := []interface{}{
		wg.ID, wg.Name, wg.Description,
		toUUIDArray(wg.UnitIDs), wg.ManagerID,
		toUUIDArray(wg.DelegateIDs),
	}

	query := fmt.Sprintf(`UPDATE working_groups SET %s, updated_at = NOW() WHERE id = $1
		RETURNING id, org_id, activity_id, name, description, unit_ids,
			manager_id, delegate_ids, is_active, created_at, updated_at`,
		strings.Join(sets, ", "))
	return scanWorkingGroup(r.pool.QueryRow(ctx, query, args...))
}

// Delete removes a working group by ID.
func (r *WorkingGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM working_groups WHERE id = $1`, id)
	return wrapPGError(err, "delete working group")
}

// HasMembers returns true if the working group has at least one member.
func (r *WorkingGroupRepository) HasMembers(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM wg_members WHERE wg_id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has members: %w", err)
	}
	return exists, nil
}

// ListMembers delegates to WGMemberRepository.
func (r *WorkingGroupRepository) ListMembers(ctx context.Context, wgID uuid.UUID) ([]working_group.WorkingGroupMember, error) {
	return r.members.ListByWG(ctx, wgID)
}

// AddMember delegates to WGMemberRepository.
func (r *WorkingGroupRepository) AddMember(ctx context.Context, m *working_group.WorkingGroupMember) (*working_group.WorkingGroupMember, error) {
	return r.members.Add(ctx, m)
}

// RemoveMember delegates to WGMemberRepository.
func (r *WorkingGroupRepository) RemoveMember(ctx context.Context, id uuid.UUID) error {
	return r.members.Remove(ctx, id)
}

// ---------------------------------------------------------------------------
// UUID[] helper functions
// ---------------------------------------------------------------------------

// toUUIDArray converts a []string to pgtype.Array[pgtype.UUID] for PG.
func toUUIDArray(ids []string) pgtype.Array[pgtype.UUID] {
	var arr pgtype.Array[pgtype.UUID]
	arr.Elements = make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		uid, err := uuid.Parse(id)
		if err != nil {
			continue
		}
		arr.Elements = append(arr.Elements, pgtype.UUID{Bytes: uid, Valid: true})
	}
	arr.Valid = true
	arr.Dims = []pgtype.ArrayDimension{{Length: int32(len(arr.Elements)), LowerBound: 1}}
	return arr
}

// scanUUIDArray converts a pgtype.Array[pgtype.UUID] to []string.
func scanUUIDArray(arr pgtype.Array[pgtype.UUID]) []string {
	ids := make([]string, 0, len(arr.Elements))
	for _, u := range arr.Elements {
		if u.Valid {
			ids = append(ids, uuid.UUID(u.Bytes).String())
		}
	}
	return ids
}

// wgScanner is satisfied by both pgx.Row and pgx.Rows.
type wgScanner interface {
	Scan(dest ...any) error
}

// scanWorkingGroup scans a single working group row, converting UUID arrays
// to []string. Returns raw error — caller wraps with domain sentinel as needed.
func scanWorkingGroup(s wgScanner) (*working_group.WorkingGroup, error) {
	var wg working_group.WorkingGroup
	var unitIDs pgtype.Array[pgtype.UUID]
	var delegateIDs pgtype.Array[pgtype.UUID]

	err := s.Scan(
		&wg.ID, &wg.OrgID, &wg.SubprojectID, &wg.Name, &wg.Description,
		&unitIDs, &wg.ManagerID, &delegateIDs,
		&wg.IsActive, &wg.CreatedAt, &wg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	wg.UnitIDs = scanUUIDArray(unitIDs)
	wg.DelegateIDs = scanUUIDArray(delegateIDs)
	return &wg, nil
}
