package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
)

// ProjectRepository implements ports.ProjectRepository using a pgxpool.
type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

// baseProjectQuery returns the SELECT list and FROM clause used by List and Get.
func baseProjectQuery() string {
	return `SELECT p.id, p.name, p.type, p.contract_id, p.governance_model, p.created_by_org_id,
		p.is_shared, p.is_active, p.created_at,
		COALESCE(c.name, '') AS contract_name, COALESCE(o.name, '') AS created_by_org_name,
		(SELECT COUNT(*) FROM project_adoptions pa WHERE pa.project_id = p.id) AS adoption_count,
		EXISTS(SELECT 1 FROM project_adoptions pa2 WHERE pa2.project_id = p.id AND pa2.organization_id = $1) AS is_adopted
	FROM projects p
	LEFT JOIN contracts c ON c.id = p.contract_id
	LEFT JOIN organizations o ON o.id = p.created_by_org_id`
}

// List returns projects filtered by scope and optional contractID.
func (r *ProjectRepository) List(ctx context.Context, orgID uuid.UUID, scope, contractID string) ([]projectdomain.ProjectResponse, error) {
	query := baseProjectQuery()
	var conditions []string
	var args []interface{}

	args = append(args, orgID) // $1 = orgID for is_adopted EXISTS subquery

	conditions = append(conditions, "p.is_active = true")

	switch scope {
	case "adopted":
		conditions = append(conditions, fmt.Sprintf("p.id IN (SELECT pa.project_id FROM project_adoptions pa WHERE pa.organization_id = $%d)", len(args)+1))
		args = append(args, orgID)
	case "all":
		conditions = append(conditions, "p.is_shared = true")
	default:
		conditions = append(conditions, fmt.Sprintf("p.created_by_org_id = $%d", len(args)+1))
		args = append(args, orgID)
	}

	if contractID != "" {
		cid, err := uuid.Parse(contractID)
		if err != nil {
			return nil, fmt.Errorf("parse contract_id: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("p.contract_id = $%d", len(args)+1))
		args = append(args, cid)
	}

	query += " WHERE " + strings.Join(conditions, " AND ")
	query += " ORDER BY p.name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	return scanProjectResponses(rows)
}

// Get returns a single project by ID.
func (r *ProjectRepository) Get(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectResponse, error) {
	query := baseProjectQuery() + ` WHERE p.id = $2 AND p.is_active = true LIMIT 1`
	res, err := scanProjectResponse(r.pool.QueryRow(ctx, query, orgID, projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, projectdomain.ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	return res, nil
}

// Create inserts a new project and returns the full response.
func (r *ProjectRepository) Create(ctx context.Context, orgID uuid.UUID, req *projectdomain.CreateProjectRequest) (*projectdomain.ProjectResponse, error) {
	id := uuid.New()

	var contractID *uuid.UUID
	if req.ContractID != "" {
		cid, err := uuid.Parse(req.ContractID)
		if err != nil {
			return nil, fmt.Errorf("parse contract_id: %w", err)
		}
		contractID = &cid
	}

	_, err := r.pool.Exec(ctx, `INSERT INTO projects (id, org_id, name, type, project_type, contract_id,
		governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, NOW(), NOW())`,
		id, orgID, req.Name, req.Type, req.Type, contractID,
		req.GovernanceModel, orgID, req.IsShared)
	if err != nil {
		return nil, wrapPGError(err, "create project")
	}

	return r.Get(ctx, orgID, id)
}

// Adopt adds an organization to the project's adoptions.
func (r *ProjectRepository) Adopt(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectAdoption, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM project_adoptions WHERE project_id = $1 AND organization_id = $2)`,
		projectID, orgID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check adoption exists: %w", err)
	}
	if exists {
		return nil, projectdomain.ErrAlreadyAdopted
	}

	adoptionID := uuid.New()
	var adoption projectdomain.ProjectAdoption
	err = r.pool.QueryRow(ctx,
		`INSERT INTO project_adoptions (id, project_id, organization_id, created_at)
		 VALUES ($1, $2, $3, NOW()) RETURNING id, project_id, organization_id, created_at`,
		adoptionID, projectID, orgID).Scan(
		&adoption.ID, &adoption.ProjectID, &adoption.OrganizationID, &adoption.AdoptedAt)
	if err != nil {
		return nil, wrapPGError(err, "adopt project")
	}
	return &adoption, nil
}

// ListManagers returns managers for a project.
func (r *ProjectRepository) ListManagers(ctx context.Context, projectID uuid.UUID) ([]projectdomain.ProjectManager, error) {
	query := `SELECT pm.id, pm.project_id, pm.user_id, u.firstname || ' ' || u.lastname AS user_name, u.email, pm.created_at
		FROM project_managers pm
		LEFT JOIN users u ON u.id = pm.user_id
		WHERE pm.project_id = $1
		ORDER BY pm.created_at DESC`

	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project managers: %w", err)
	}
	defer rows.Close()

	var managers []projectdomain.ProjectManager
	for rows.Next() {
		var m projectdomain.ProjectManager
		err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &m.UserName, &m.Email, &m.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan project manager: %w", err)
		}
		managers = append(managers, m)
	}
	if managers == nil {
		managers = []projectdomain.ProjectManager{}
	}
	return managers, rows.Err()
}

// AddManager adds a user as manager to a project.
func (r *ProjectRepository) AddManager(ctx context.Context, projectID, userID uuid.UUID) (*projectdomain.ProjectManager, error) {
	mid := uuid.New()
	var m projectdomain.ProjectManager
	err := r.pool.QueryRow(ctx,
		`INSERT INTO project_managers (id, project_id, user_id, created_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING id, project_id, user_id, created_at`,
		mid, projectID, userID).Scan(
		&m.ID, &m.ProjectID, &m.UserID, &m.CreatedAt)
	if err != nil {
		return nil, wrapPGError(err, "add project manager")
	}

	// Fetch user name and email for the response
	err = r.pool.QueryRow(ctx,
		`SELECT firstname || ' ' || lastname, email FROM users WHERE id = $1`, userID).Scan(
		&m.UserName, &m.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, projectdomain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get manager user info: %w", err)
	}
	return &m, nil
}

// RemoveManager removes a user from project managers.
func (r *ProjectRepository) RemoveManager(ctx context.Context, projectID, userID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx,
		`DELETE FROM project_managers WHERE project_id = $1 AND user_id = $2`,
		projectID, userID)
	if err != nil {
		return wrapPGError(err, "remove project manager")
	}
	if cmd.RowsAffected() == 0 {
		return projectdomain.ErrProjectNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

// projectResponseScanner is satisfied by pgx.Row and pgx.Rows.
type projectResponseScanner interface {
	Scan(dest ...any) error
}

func scanProjectResponse(s projectResponseScanner) (*projectdomain.ProjectResponse, error) {
	var p projectdomain.ProjectResponse
	err := s.Scan(
		&p.ID, &p.Name, &p.Type, &p.ContractID, &p.GovernanceModel,
		&p.CreatedByOrgID, &p.IsShared, &p.IsActive, &p.CreatedAt,
		&p.ContractName, &p.CreatedByOrgName, &p.AdoptionCount, &p.IsAdopted,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanProjectResponses(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]projectdomain.ProjectResponse, error) {
	var projects []projectdomain.ProjectResponse
	for rows.Next() {
		p, err := scanProjectResponse(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, *p)
	}
	if projects == nil {
		projects = []projectdomain.ProjectResponse{}
	}
	return projects, rows.Err()
}
