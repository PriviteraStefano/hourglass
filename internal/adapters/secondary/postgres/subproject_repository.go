package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// SubprojectRepository implements ports.SubprojectRepository using a pgxpool.
type SubprojectRepository struct {
	pool *pgxpool.Pool
}

func NewSubprojectRepository(pool *pgxpool.Pool) *SubprojectRepository {
	return &SubprojectRepository{pool: pool}
}

// ListByProject returns all subprojects for a given project.
func (r *SubprojectRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Subproject, error) {
	query := `SELECT id, project_id, name, description, sequence_order, is_active, created_at, updated_at
		FROM subprojects WHERE project_id = $1 ORDER BY sequence_order, name`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("list subprojects by project: %w", err)
	}
	defer rows.Close()

	var subs []models.Subproject
	for rows.Next() {
		s, err := scanSubproject(rows)
		if err != nil {
			return nil, fmt.Errorf("scan subproject: %w", err)
		}
		subs = append(subs, *s)
	}
	if subs == nil {
		subs = []models.Subproject{}
	}
	return subs, rows.Err()
}

// GetByID returns a subproject by its string ID. Returns nil, nil when not found.
func (r *SubprojectRepository) GetByID(ctx context.Context, id string) (*models.Subproject, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse subproject id: %w", err)
	}

	query := `SELECT id, project_id, name, description, sequence_order, is_active, created_at, updated_at
		FROM subprojects WHERE id = $1`
	s, err := scanSubproject(r.pool.QueryRow(ctx, query, uid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subproject by id: %w", err)
	}
	return s, nil
}

// Create inserts a new subproject and returns it.
func (r *SubprojectRepository) Create(ctx context.Context, sp *models.Subproject) (*models.Subproject, error) {
	projectID, err := uuid.Parse(sp.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("parse project_id: %w", err)
	}

	sp.ID = uuid.New().String()

	query := `INSERT INTO subprojects (id, project_id, name, description, sequence_order, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, project_id, name, description, sequence_order, is_active, created_at, updated_at`
	return scanSubproject(r.pool.QueryRow(ctx, query,
		uuid.MustParse(sp.ID), projectID, sp.Name, sp.Description, sp.SequenceOrder, sp.IsActive))
}

// Update modifies an existing subproject and returns it.
func (r *SubprojectRepository) Update(ctx context.Context, sp *models.Subproject) (*models.Subproject, error) {
	uid, err := uuid.Parse(sp.ID)
	if err != nil {
		return nil, fmt.Errorf("parse subproject id: %w", err)
	}
	query := `UPDATE subprojects SET name = $1, description = $2, sequence_order = $3, is_active = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING id, project_id, name, description, sequence_order, is_active, created_at, updated_at`
	return scanSubproject(r.pool.QueryRow(ctx, query,
		sp.Name, sp.Description, sp.SequenceOrder, sp.IsActive, uid))
}

// Delete removes a subproject by its string ID.
func (r *SubprojectRepository) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse subproject id: %w", err)
	}

	_, err = r.pool.Exec(ctx, `DELETE FROM subprojects WHERE id = $1`, uid)
	return wrapPGError(err, "delete subproject")
}

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

type subprojectScanner interface {
	Scan(dest ...any) error
}

func scanSubproject(s subprojectScanner) (*models.Subproject, error) {
	var id uuid.UUID
	var projectID uuid.UUID
	var sp models.Subproject

	err := s.Scan(&id, &projectID, &sp.Name, &sp.Description,
		&sp.SequenceOrder, &sp.IsActive, &sp.CreatedAt, &sp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	sp.ID = id.String()
	sp.ProjectID = projectID.String()
	return &sp, nil
}
