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

// UnitRepository implements ports.UnitRepository using a pgxpool.
type UnitRepository struct {
	pool    *pgxpool.Pool
	members *UnitMemberRepository
}

func NewUnitRepository(pool *pgxpool.Pool) *UnitRepository {
	return &UnitRepository{
		pool:    pool,
		members: NewUnitMemberRepository(pool),
	}
}

// ListByOrg returns all units for an organization, ordered by name.
func (r *UnitRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]unit.Unit, error) {
	query := `SELECT id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at
		FROM units WHERE org_id = $1 ORDER BY name`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list units by org: %w", err)
	}
	defer rows.Close()

	var units []unit.Unit
	for rows.Next() {
		u, err := scanUnitRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan unit: %w", err)
		}
		units = append(units, *u)
	}
	if units == nil {
		units = []unit.Unit{}
	}
	return units, rows.Err()
}

// GetByID returns a unit by ID, or unit.ErrUnitNotFound.
func (r *UnitRepository) GetByID(ctx context.Context, id string) (*unit.Unit, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse unit id: %w", err)
	}
	query := `SELECT id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at
		FROM units WHERE id = $1`
	u, err := scanUnit(r.pool.QueryRow(ctx, query, uid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, unit.ErrUnitNotFound
		}
		return nil, fmt.Errorf("get unit by id: %w", err)
	}
	return u, nil
}

// Create inserts a new unit and returns it with generated fields.
func (r *UnitRepository) Create(ctx context.Context, u *unit.Unit) (*unit.Unit, error) {
	uid := uuid.New()
	u.ID = uid.String()

	var parentID *uuid.UUID
	if u.ParentUnitID != "" {
		pid, err := uuid.Parse(u.ParentUnitID)
		if err != nil {
			return nil, fmt.Errorf("parse parent unit id: %w", err)
		}
		parentID = &pid
	}

	query := `INSERT INTO units (id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at`
	return scanUnit(r.pool.QueryRow(ctx, query,
		uid, u.OrgID, u.Name, u.Description, parentID, u.HierarchyLevel, u.Code))
}

// Update dynamically builds a SET clause. name is always set; description, code,
// and parent_unit_id are set only when non-empty.
func (r *UnitRepository) Update(ctx context.Context, u *unit.Unit) (*unit.Unit, error) {
	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, fmt.Errorf("parse unit id: %w", err)
	}

	sets := []string{"name = $2"}
	args := []interface{}{uid, u.Name}
	argIdx := 3

	if u.Description != "" {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, u.Description)
		argIdx++
	}
	if u.Code != "" {
		sets = append(sets, fmt.Sprintf("code = $%d", argIdx))
		args = append(args, u.Code)
		argIdx++
	}
	if u.ParentUnitID != "" {
		pid, err := uuid.Parse(u.ParentUnitID)
		if err != nil {
			return nil, fmt.Errorf("parse parent unit id: %w", err)
		}
		sets = append(sets, fmt.Sprintf("parent_unit_id = $%d", argIdx))
		args = append(args, pid)
		argIdx++
	}

	query := fmt.Sprintf(`UPDATE units SET %s, updated_at = NOW() WHERE id = $1
		RETURNING id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at`,
		strings.Join(sets, ", "))
	return scanUnit(r.pool.QueryRow(ctx, query, args...))
}

// Delete removes a unit by ID.
func (r *UnitRepository) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse unit id: %w", err)
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM units WHERE id = $1`, uid)
	return wrapPGError(err, "delete unit")
}

// GetDescendants returns all descendants of a unit using a recursive CTE,
// excluding the unit itself.
func (r *UnitRepository) GetDescendants(ctx context.Context, id string) ([]unit.Unit, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse unit id: %w", err)
	}
	query := `WITH RECURSIVE descendants AS (
		SELECT id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at
		FROM units WHERE parent_unit_id = $1
		UNION ALL
		SELECT u.id, u.org_id, u.name, u.description, u.parent_unit_id, u.hierarchy_level, u.code, u.created_at, u.updated_at
		FROM units u
		INNER JOIN descendants d ON u.parent_unit_id = d.id
	)
	SELECT id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at
	FROM descendants WHERE id != $1
	ORDER BY name`

	rows, err := r.pool.Query(ctx, query, uid)
	if err != nil {
		return nil, fmt.Errorf("get descendants: %w", err)
	}
	defer rows.Close()

	var descendants []unit.Unit
	for rows.Next() {
		u, err := scanUnitRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan descendant: %w", err)
		}
		descendants = append(descendants, *u)
	}
	if descendants == nil {
		descendants = []unit.Unit{}
	}
	return descendants, rows.Err()
}

// HasMembers returns true if the unit has at least one member.
func (r *UnitRepository) HasMembers(ctx context.Context, id string) (bool, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return false, fmt.Errorf("parse unit id: %w", err)
	}
	query := `SELECT EXISTS(SELECT 1 FROM unit_memberships WHERE unit_id = $1)`
	var exists bool
	err = r.pool.QueryRow(ctx, query, uid).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has members: %w", err)
	}
	return exists, nil
}

// ListMembers delegates to UnitMemberRepository.
func (r *UnitRepository) ListMembers(ctx context.Context, unitID string) ([]unit.UnitMember, error) {
	return r.members.ListByUnit(ctx, unitID)
}

// AddMember delegates to UnitMemberRepository.
func (r *UnitRepository) AddMember(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error) {
	return r.members.Add(ctx, m)
}

// RemoveMember delegates to UnitMemberRepository.
func (r *UnitRepository) RemoveMember(ctx context.Context, id string) error {
	return r.members.Remove(ctx, id)
}

// GetMemberCountsByOrg returns member counts keyed by unit ID (string).
func (r *UnitRepository) GetMemberCountsByOrg(ctx context.Context, orgID uuid.UUID) (map[string]int, error) {
	query := `SELECT unit_id, COUNT(*) FROM unit_memberships WHERE org_id = $1 GROUP BY unit_id`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("get member counts by org: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var sqlUnitID uuid.UUID
		var count int
		if err := rows.Scan(&sqlUnitID, &count); err != nil {
			return nil, fmt.Errorf("scan member count: %w", err)
		}
		counts[sqlUnitID.String()] = count
	}
	return counts, rows.Err()
}

// unitScanner is satisfied by both pgx.Row and pgx.Rows.
type unitScanner interface {
	Scan(dest ...any) error
}

// scanUnit scans a single unit row and converts UUID IDs to strings.
// Returns raw error — caller wraps with domain sentinel as needed.
func scanUnit(s unitScanner) (*unit.Unit, error) {
	var u unit.Unit
	var sqlID uuid.UUID
	var parentID *uuid.UUID

	err := s.Scan(
		&sqlID, &u.OrgID, &u.Name, &u.Description,
		&parentID, &u.HierarchyLevel, &u.Code,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.ID = sqlID.String()
	if parentID != nil {
		u.ParentUnitID = parentID.String()
	}
	return &u, nil
}

// scanUnitRow is an alias for scanUnit used in row-iteration contexts.
func scanUnitRow(s unitScanner) (*unit.Unit, error) {
	return scanUnit(s)
}
