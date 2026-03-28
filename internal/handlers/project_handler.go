package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type ProjectHandler struct {
	db *sql.DB
}

func NewProjectHandler(db *sql.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

type CreateProjectRequest struct {
	Name            string                 `json:"name"`
	Type            models.ProjectType     `json:"type"`
	ContractID      string                 `json:"contract_id"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        bool                   `json:"is_shared"`
}

type ProjectResponse struct {
	models.Project
	ContractName     string `json:"contract_name,omitempty"`
	CreatedByOrgName string `json:"created_by_org_name,omitempty"`
	AdoptionCount    int    `json:"adoption_count,omitempty"`
	IsAdopted        bool   `json:"is_adopted,omitempty"`
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())

	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "owned"
	}

	contractID := r.URL.Query().Get("contract_id")

	var rows *sql.Rows
	var err error

	selectClause := `
		SELECT p.id, p.name, p.type, p.contract_id, p.governance_model,
			   p.created_by_org_id, p.is_shared, p.is_active, p.created_at,
			   c.name as contract_name,
			   o.name as created_by_org_name`
	fromClause := `
		FROM projects p
		LEFT JOIN contracts c ON p.contract_id = c.id
		LEFT JOIN organizations o ON p.created_by_org_id = o.id
		WHERE p.is_active = true
	`
	baseQuery := selectClause + fromClause
	args := []interface{}{}
	argIndex := 1

	switch scope {
	case "adopted":
		baseQuery += ` AND p.id IN (SELECT project_id FROM project_adoptions WHERE organization_id = $` + strconv.Itoa(argIndex) + `)`
		args = append(args, orgID)
		argIndex++
	case "all":
		selectClause += ", EXISTS(SELECT 1 FROM project_adoptions WHERE project_id = p.id AND organization_id = $" + strconv.Itoa(argIndex) + ") as is_adopted"
		baseQuery = selectClause + fromClause
		baseQuery += ` AND p.is_shared = true`
		args = append(args, orgID)
		argIndex++
	default:
		baseQuery += ` AND p.created_by_org_id = $` + strconv.Itoa(argIndex)
		args = append(args, orgID)
		argIndex++
	}

	if contractID != "" {
		baseQuery += ` AND p.contract_id = $` + strconv.Itoa(argIndex)
		args = append(args, contractID)
		argIndex++
	}

	baseQuery += ` ORDER BY p.created_at DESC`

	rows, err = h.db.Query(baseQuery, args...)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch projects")
		return
	}
	defer rows.Close()

	var projects []ProjectResponse
	for rows.Next() {
		var p models.Project
		var contractName sql.NullString
		var orgName sql.NullString
		var isAdopted sql.NullBool

		var scanErr error
		if scope == "all" {
			scanErr = rows.Scan(
				&p.ID, &p.Name, &p.Type, &p.ContractID, &p.GovernanceModel,
				&p.CreatedByOrgID, &p.IsShared, &p.IsActive, &p.CreatedAt,
				&contractName, &orgName, &isAdopted,
			)
		} else {
			scanErr = rows.Scan(
				&p.ID, &p.Name, &p.Type, &p.ContractID, &p.GovernanceModel,
				&p.CreatedByOrgID, &p.IsShared, &p.IsActive, &p.CreatedAt,
				&contractName, &orgName,
			)
		}
		if scanErr != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan project")
			return
		}

		resp := ProjectResponse{Project: p}
		if contractName.Valid {
			resp.ContractName = contractName.String
		}
		if orgName.Valid {
			resp.CreatedByOrgName = orgName.String
		}
		if scope == "all" {
			resp.IsAdopted = isAdopted.Valid && isAdopted.Bool
		}

		projects = append(projects, resp)
	}

	if projects == nil {
		projects = []ProjectResponse{}
	}

	api.RespondWithJSON(w, http.StatusOK, projects)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		api.RespondWithError(w, http.StatusBadRequest, "name is required")
		return
	}

	if !req.Type.IsValid() {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project type")
		return
	}

	if !req.GovernanceModel.IsValid() {
		api.RespondWithError(w, http.StatusBadRequest, "invalid governance model")
		return
	}

	contractID, err := uuid.Parse(req.ContractID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	var contractExists bool
	err = h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM contracts
			WHERE id = $1 AND is_active = true
			AND (created_by_org_id = $2 OR is_shared = true OR id IN (
				SELECT contract_id FROM contract_adoptions WHERE organization_id = $2
			))
		)
	`, contractID, orgID).Scan(&contractExists)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to verify contract")
		return
	}

	if !contractExists {
		api.RespondWithError(w, http.StatusBadRequest, "contract not found or not accessible")
		return
	}

	var project models.Project
	var contractName sql.NullString
	var orgName sql.NullString
	err = h.db.QueryRow(`
		INSERT INTO projects (name, type, contract_id, governance_model, created_by_org_id, is_shared, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id, name, type, contract_id, governance_model, created_by_org_id, is_shared, is_active, created_at,
		(SELECT name FROM contracts WHERE id = $3),
		(SELECT name FROM organizations WHERE id = $5)
	`, req.Name, req.Type, contractID, req.GovernanceModel, orgID, req.IsShared).Scan(
		&project.ID, &project.Name, &project.Type, &project.ContractID,
		&project.GovernanceModel, &project.CreatedByOrgID, &project.IsShared,
		&project.IsActive, &project.CreatedAt, &contractName, &orgName,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	resp := ProjectResponse{Project: project}
	if contractName.Valid {
		resp.ContractName = contractName.String
	}
	if orgName.Valid {
		resp.CreatedByOrgName = orgName.String
	}

	api.RespondWithJSON(w, http.StatusCreated, resp)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	projectIDStr := r.PathValue("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var project models.Project
	var contractName sql.NullString
	var orgName sql.NullString
	err = h.db.QueryRow(`
		SELECT p.id, p.name, p.type, p.contract_id, p.governance_model,
			   p.created_by_org_id, p.is_shared, p.is_active, p.created_at,
			   c.name as contract_name,
			   o.name as created_by_org_name
		FROM projects p
		LEFT JOIN contracts c ON p.contract_id = c.id
		LEFT JOIN organizations o ON p.created_by_org_id = o.id
		WHERE p.id = $1 AND p.is_active = true
	`, projectID).Scan(
		&project.ID, &project.Name, &project.Type, &project.ContractID,
		&project.GovernanceModel, &project.CreatedByOrgID, &project.IsShared,
		&project.IsActive, &project.CreatedAt, &contractName, &orgName,
	)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch project")
		return
	}

	var adoptionCount int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM project_adoptions WHERE project_id = $1
	`, projectID).Scan(&adoptionCount)

	var isAdopted bool
	h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM project_adoptions WHERE project_id = $1 AND organization_id = $2)
	`, projectID, orgID).Scan(&isAdopted)

	resp := ProjectResponse{
		Project:       project,
		AdoptionCount: adoptionCount,
		IsAdopted:     isAdopted,
	}
	if contractName.Valid {
		resp.ContractName = contractName.String
	}
	if orgName.Valid {
		resp.CreatedByOrgName = orgName.String
	}

	api.RespondWithJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) Adopt(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())

	projectIDStr := r.PathValue("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var project models.Project
	err = h.db.QueryRow(`
		SELECT id, is_shared, is_active FROM projects WHERE id = $1
	`, projectID).Scan(&project.ID, &project.IsShared, &project.IsActive)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch project")
		return
	}

	if !project.IsShared {
		api.RespondWithError(w, http.StatusBadRequest, "project is not shared")
		return
	}

	if !project.IsActive {
		api.RespondWithError(w, http.StatusBadRequest, "project is not active")
		return
	}

	var existingCount int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM project_adoptions WHERE project_id = $1 AND organization_id = $2
	`, projectID, orgID).Scan(&existingCount)
	if existingCount > 0 {
		api.RespondWithError(w, http.StatusConflict, "project already adopted")
		return
	}

	var adoption models.ProjectAdoption
	err = h.db.QueryRow(`
		INSERT INTO project_adoptions (project_id, organization_id)
		VALUES ($1, $2)
		RETURNING id, project_id, organization_id, adopted_at
	`, projectID, orgID).Scan(&adoption.ID, &adoption.ProjectID, &adoption.OrganizationID, &adoption.AdoptedAt)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to adopt project")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, adoption)
}
