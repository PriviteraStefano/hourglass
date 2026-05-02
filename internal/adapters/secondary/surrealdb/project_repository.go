package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	projectdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/project"
	"github.com/stefanoprivitera/hourglass/internal/models"
	sdb "github.com/surrealdb/surrealdb.go"
	sdbmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

type ProjectRepository struct {
	db *sdb.DB
}

func NewProjectRepository(db *sdb.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

type surrealProjectCompat struct {
	ID             sdbmodels.RecordID `json:"id,omitempty"`
	Name           string             `json:"name"`
	Type           string             `json:"type"`
	ContractID     sdbmodels.RecordID `json:"contract_id,omitempty"`
	Governance     string             `json:"governance_model"`
	CreatedByOrgID sdbmodels.RecordID `json:"created_by_org_id"`
	IsShared       bool               `json:"is_shared"`
	IsActive       bool               `json:"is_active"`
	CreatedAt      time.Time          `json:"created_at"`
}

type projectJoined struct {
	surrealProjectCompat
	ContractName string `json:"contract_name,omitempty"`
	OrgName      string `json:"created_by_org_name,omitempty"`
	IsAdopted    bool   `json:"is_adopted,omitempty"`
	AdoptionCnt  int    `json:"adoption_count,omitempty"`
}

func (r *ProjectRepository) List(ctx context.Context, orgID uuid.UUID, scope, contractID string) ([]projectdomain.ProjectResponse, error) {
	where := "WHERE p.is_active = true"
	vars := map[string]interface{}{"org_id": uuidToRecordID("organizations", orgID)}
	switch scope {
	case "adopted":
		where += " AND p.id IN (SELECT VALUE project_id FROM project_adoptions WHERE organization_id = $org_id)"
	case "all":
		where += " AND p.is_shared = true"
	default:
		where += " AND p.created_by_org_id = $org_id"
	}
	if contractID != "" {
		if cid, err := uuid.Parse(contractID); err == nil {
			vars["contract_id"] = uuidToRecordID("contracts", cid)
			where += " AND p.contract_id = $contract_id"
		}
	}

	results, err := sdb.Query[[]projectJoined](ctx, r.db, `
		SELECT p.*,
			(SELECT VALUE name FROM contracts WHERE id = p.contract_id LIMIT 1)[0] AS contract_name,
			(SELECT VALUE name FROM organizations WHERE id = p.created_by_org_id LIMIT 1)[0] AS created_by_org_name,
			count((SELECT VALUE id FROM project_adoptions WHERE project_id = p.id)) AS adoption_count,
			(SELECT VALUE count() > 0 FROM project_adoptions WHERE project_id = p.id AND organization_id = $org_id GROUP ALL)[0] AS is_adopted
		FROM projects p `+where+` ORDER BY p.created_at DESC`,
		vars)
	if err != nil || results == nil || len(*results) == 0 {
		return []projectdomain.ProjectResponse{}, nil
	}
	out := make([]projectdomain.ProjectResponse, 0, len((*results)[0].Result))
	for _, p := range (*results)[0].Result {
		resp := projectdomain.ProjectResponse{
			Project: projectdomain.Project{
				ID:              recordIDToUUID(p.ID),
				Name:            p.Name,
				Type:            models.ProjectType(p.Type),
				ContractID:      recordIDToUUID(p.ContractID),
				GovernanceModel: models.GovernanceModel(p.Governance),
				CreatedByOrgID:  recordIDToUUID(p.CreatedByOrgID),
				IsShared:        p.IsShared,
				IsActive:        p.IsActive,
				CreatedAt:       p.CreatedAt,
			},
			ContractName:     p.ContractName,
			CreatedByOrgName: p.OrgName,
			AdoptionCount:    p.AdoptionCnt,
			IsAdopted:        p.IsAdopted,
		}
		out = append(out, resp)
	}
	return out, nil
}

func (r *ProjectRepository) Create(ctx context.Context, orgID uuid.UUID, req *projectdomain.CreateProjectRequest) (*projectdomain.ProjectResponse, error) {
	contractID, err := uuid.Parse(req.ContractID)
	if err != nil {
		return nil, projectdomain.ErrInvalidRequest
	}
	id := uuid.New()
	now := time.Now()
	data := map[string]interface{}{
		"id":                uuidToRecordID("projects", id),
		"name":              req.Name,
		"type":              string(req.Type),
		"project_type":      string(req.Type),
		"contract_id":       uuidToRecordID("contracts", contractID),
		"governance_model":  string(req.GovernanceModel),
		"created_by_org_id": uuidToRecordID("organizations", orgID),
		"org_id":            uuidToRecordID("organizations", orgID),
		"is_shared":         req.IsShared,
		"is_active":         true,
		"created_at":        now,
		"updated_at":        now,
	}
	if _, err := sdb.Create[surrealProjectCompat](ctx, r.db, sdbmodels.Table("projects"), data); err != nil {
		return nil, wrapErr(err, "create project")
	}
	return r.Get(ctx, orgID, id)
}

func (r *ProjectRepository) Get(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectResponse, error) {
	results, err := sdb.Query[[]projectJoined](ctx, r.db, `
		SELECT p.*,
			(SELECT VALUE name FROM contracts WHERE id = p.contract_id LIMIT 1)[0] AS contract_name,
			(SELECT VALUE name FROM organizations WHERE id = p.created_by_org_id LIMIT 1)[0] AS created_by_org_name,
			count((SELECT VALUE id FROM project_adoptions WHERE project_id = p.id)) AS adoption_count,
			(SELECT VALUE count() > 0 FROM project_adoptions WHERE project_id = p.id AND organization_id = $org_id GROUP ALL)[0] AS is_adopted
		FROM projects p WHERE p.id = $project_id AND p.is_active = true LIMIT 1`,
		map[string]interface{}{
			"org_id":     uuidToRecordID("organizations", orgID),
			"project_id": uuidToRecordID("projects", projectID),
		})
	if err != nil || results == nil || len(*results) == 0 || len((*results)[0].Result) == 0 {
		return nil, projectdomain.ErrProjectNotFound
	}
	p := (*results)[0].Result[0]
	return &projectdomain.ProjectResponse{
		Project: projectdomain.Project{
			ID:              recordIDToUUID(p.ID),
			Name:            p.Name,
			Type:            models.ProjectType(p.Type),
			ContractID:      recordIDToUUID(p.ContractID),
			GovernanceModel: models.GovernanceModel(p.Governance),
			CreatedByOrgID:  recordIDToUUID(p.CreatedByOrgID),
			IsShared:        p.IsShared,
			IsActive:        p.IsActive,
			CreatedAt:       p.CreatedAt,
		},
		ContractName:     p.ContractName,
		CreatedByOrgName: p.OrgName,
		AdoptionCount:    p.AdoptionCnt,
		IsAdopted:        p.IsAdopted,
	}, nil
}

func (r *ProjectRepository) Adopt(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectAdoption, error) {
	existing, _ := sdb.Query[[]map[string]interface{}](ctx, r.db, `SELECT count() FROM project_adoptions WHERE project_id=$project_id AND organization_id=$org_id GROUP ALL`, map[string]interface{}{
		"project_id": uuidToRecordID("projects", projectID),
		"org_id":     uuidToRecordID("organizations", orgID),
	})
	if existing != nil && len(*existing) > 0 && len((*existing)[0].Result) > 0 {
		if cnt, ok := (*existing)[0].Result[0]["count"].(float64); ok && cnt > 0 {
			return nil, projectdomain.ErrAlreadyAdopted
		}
	}
	id := uuid.New()
	now := time.Now()
	data := map[string]interface{}{
		"id":              uuidToRecordID("project_adoptions", id),
		"project_id":      uuidToRecordID("projects", projectID),
		"organization_id": uuidToRecordID("organizations", orgID),
		"adopted_at":      now,
	}
	if _, err := sdb.Create[map[string]interface{}](ctx, r.db, sdbmodels.Table("project_adoptions"), data); err != nil {
		return nil, wrapErr(err, "adopt project")
	}
	return &projectdomain.ProjectAdoption{ID: id, ProjectID: projectID, OrganizationID: orgID, AdoptedAt: now}, nil
}

func (r *ProjectRepository) ListManagers(ctx context.Context, projectID uuid.UUID) ([]projectdomain.ProjectManager, error) {
	type row struct {
		ID        sdbmodels.RecordID `json:"id"`
		ProjectID sdbmodels.RecordID `json:"project_id"`
		UserID    sdbmodels.RecordID `json:"user_id"`
		UserName  string             `json:"user_name"`
		Email     string             `json:"email"`
		CreatedAt time.Time          `json:"created_at"`
	}
	results, err := sdb.Query[[]row](ctx, r.db, `
		SELECT pm.*,
			(SELECT VALUE name FROM users WHERE id = pm.user_id LIMIT 1)[0] AS user_name,
			(SELECT VALUE email FROM users WHERE id = pm.user_id LIMIT 1)[0] AS email
		FROM project_managers pm WHERE pm.project_id = $project_id ORDER BY pm.created_at ASC`,
		map[string]interface{}{"project_id": uuidToRecordID("projects", projectID)})
	if err != nil || results == nil || len(*results) == 0 {
		return []projectdomain.ProjectManager{}, nil
	}
	out := make([]projectdomain.ProjectManager, 0, len((*results)[0].Result))
	for _, m := range (*results)[0].Result {
		out = append(out, projectdomain.ProjectManager{
			ID:        recordIDToUUID(m.ID),
			ProjectID: recordIDToUUID(m.ProjectID),
			UserID:    recordIDToUUID(m.UserID),
			UserName:  m.UserName,
			Email:     m.Email,
			CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

func (r *ProjectRepository) AddManager(ctx context.Context, projectID, userID uuid.UUID) (*projectdomain.ProjectManager, error) {
	userRec := uuidToRecordID("users", userID)
	exists, _ := sdb.Select[SurrealUser](ctx, r.db, userRec)
	if exists == nil || !exists.IsActive {
		return nil, projectdomain.ErrUserNotFound
	}
	id := uuid.New()
	now := time.Now()
	if _, err := sdb.Create[map[string]interface{}](ctx, r.db, sdbmodels.Table("project_managers"), map[string]interface{}{
		"id":         uuidToRecordID("project_managers", id),
		"project_id": uuidToRecordID("projects", projectID),
		"user_id":    userRec,
		"created_at": now,
	}); err != nil {
		return nil, wrapErr(err, "add manager")
	}
	list, _ := r.ListManagers(ctx, projectID)
	for _, m := range list {
		if m.ID == id {
			return &m, nil
		}
	}
	return &projectdomain.ProjectManager{ID: id, ProjectID: projectID, UserID: userID, CreatedAt: now}, nil
}

func (r *ProjectRepository) RemoveManager(ctx context.Context, projectID, userID uuid.UUID) error {
	type managerIDRow struct {
		ID sdbmodels.RecordID `json:"id"`
	}
	results, err := sdb.Query[[]managerIDRow](ctx, r.db, `SELECT id FROM project_managers WHERE project_id=$project_id AND user_id=$user_id LIMIT 1`, map[string]interface{}{
		"project_id": uuidToRecordID("projects", projectID),
		"user_id":    uuidToRecordID("users", userID),
	})
	if err != nil || results == nil || len(*results) == 0 || len((*results)[0].Result) == 0 {
		return projectdomain.ErrProjectNotFound
	}
	rawID := (*results)[0].Result[0].ID
	_, err = sdb.Delete[map[string]interface{}](ctx, r.db, rawID)
	return wrapErr(err, "remove manager")
}
