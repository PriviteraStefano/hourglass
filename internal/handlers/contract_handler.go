package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type ContractHandler struct {
	db *sql.DB
}

func NewContractHandler(db *sql.DB) *ContractHandler {
	return &ContractHandler{db: db}
}

type CreateContractRequest struct {
	Name            string                 `json:"name"`
	KmRate          float64                `json:"km_rate"`
	Currency        string                 `json:"currency"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        bool                   `json:"is_shared"`
}

type ContractResponse struct {
	models.Contract
	CreatedByOrgName string `json:"created_by_org_name,omitempty"`
	AdoptionCount    int    `json:"adoption_count,omitempty"`
	IsAdopted        bool   `json:"is_adopted,omitempty"`
}

func (h *ContractHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())

	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "owned"
	}

	var rows *sql.Rows
	var err error
	switch scope {
	case "adopted":
		rows, err = h.db.Query(`
			SELECT c.id, c.name, c.km_rate, c.currency, c.governance_model, 
				   c.created_by_org_id, c.is_shared, c.is_active, c.created_at,
				   o.name
			FROM contracts c
			INNER JOIN contract_adoptions ca ON c.id = ca.contract_id
			LEFT JOIN organizations o ON c.created_by_org_id = o.id
			WHERE ca.organization_id = $1 AND c.is_active = true
			ORDER BY c.created_at DESC
		`, orgID)
	case "all":
		rows, err = h.db.Query(`
			SELECT c.id, c.name, c.km_rate, c.currency, c.governance_model,
				   c.created_by_org_id, c.is_shared, c.is_active, c.created_at,
				   o.name,
				   EXISTS(SELECT 1 FROM contract_adoptions WHERE contract_id = c.id AND organization_id = $1)
			FROM contracts c
			LEFT JOIN organizations o ON c.created_by_org_id = o.id
			WHERE c.is_shared = true AND c.is_active = true
			ORDER BY c.created_at DESC
		`, orgID)
	default:
		rows, err = h.db.Query(`
			SELECT c.id, c.name, c.km_rate, c.currency, c.governance_model,
				   c.created_by_org_id, c.is_shared, c.is_active, c.created_at,
				   o.name
			FROM contracts c
			LEFT JOIN organizations o ON c.created_by_org_id = o.id
			WHERE c.created_by_org_id = $1 AND c.is_active = true
			ORDER BY c.created_at DESC
		`, orgID)
	}

	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch contracts")
		return
	}
	defer rows.Close()

	var contracts []ContractResponse
	for rows.Next() {
		var c models.Contract
		var orgName sql.NullString
		var isAdopted sql.NullBool

		var scanErr error
		if scope == "all" {
			scanErr = rows.Scan(
				&c.ID, &c.Name, &c.KmRate, &c.Currency, &c.GovernanceModel,
				&c.CreatedByOrgID, &c.IsShared, &c.IsActive, &c.CreatedAt,
				&orgName, &isAdopted,
			)
		} else {
			scanErr = rows.Scan(
				&c.ID, &c.Name, &c.KmRate, &c.Currency, &c.GovernanceModel,
				&c.CreatedByOrgID, &c.IsShared, &c.IsActive, &c.CreatedAt,
				&orgName,
			)
		}
		if scanErr != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan contract")
			return
		}

		resp := ContractResponse{Contract: c}
		if orgName.Valid {
			resp.CreatedByOrgName = orgName.String
		}
		if scope == "all" {
			resp.IsAdopted = isAdopted.Valid && isAdopted.Bool
		}

		contracts = append(contracts, resp)
	}

	if contracts == nil {
		contracts = []ContractResponse{}
	}

	api.RespondWithJSON(w, http.StatusOK, contracts)
}

func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())

	var req CreateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		api.RespondWithError(w, http.StatusBadRequest, "name is required")
		return
	}

	if !req.GovernanceModel.IsValid() {
		api.RespondWithError(w, http.StatusBadRequest, "invalid governance model")
		return
	}

	if req.Currency == "" {
		req.Currency = "EUR"
	}

	var contract models.Contract
	var orgName sql.NullString
	err := h.db.QueryRow(`
		INSERT INTO contracts (name, km_rate, currency, governance_model, created_by_org_id, is_shared, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id, name, km_rate, currency, governance_model, created_by_org_id, is_shared, is_active, created_at,
		(SELECT name FROM organizations WHERE id = $5)
	`, req.Name, req.KmRate, req.Currency, req.GovernanceModel, orgID, req.IsShared).Scan(
		&contract.ID, &contract.Name, &contract.KmRate, &contract.Currency,
		&contract.GovernanceModel, &contract.CreatedByOrgID, &contract.IsShared,
		&contract.IsActive, &contract.CreatedAt, &orgName,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create contract")
		return
	}

	resp := ContractResponse{Contract: contract}
	if orgName.Valid {
		resp.CreatedByOrgName = orgName.String
	}

	api.RespondWithJSON(w, http.StatusCreated, resp)
}

func (h *ContractHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	contractIDStr := r.PathValue("id")
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	var contract models.Contract
	var orgName sql.NullString
	err = h.db.QueryRow(`
		SELECT c.id, c.name, c.km_rate, c.currency, c.governance_model, c.created_by_org_id, 
			   c.is_shared, c.is_active, c.created_at, o.name
		FROM contracts c
		LEFT JOIN organizations o ON c.created_by_org_id = o.id
		WHERE c.id = $1 AND c.is_active = true
	`, contractID).Scan(
		&contract.ID, &contract.Name, &contract.KmRate, &contract.Currency,
		&contract.GovernanceModel, &contract.CreatedByOrgID, &contract.IsShared,
		&contract.IsActive, &contract.CreatedAt, &orgName,
	)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "contract not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch contract")
		return
	}

	var adoptionCount int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM contract_adoptions WHERE contract_id = $1
	`, contractID).Scan(&adoptionCount)

	var isAdopted bool
	h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM contract_adoptions WHERE contract_id = $1 AND organization_id = $2)
	`, contractID, orgID).Scan(&isAdopted)

	resp := ContractResponse{
		Contract:      contract,
		AdoptionCount: adoptionCount,
		IsAdopted:     isAdopted,
	}
	if orgName.Valid {
		resp.CreatedByOrgName = orgName.String
	}

	api.RespondWithJSON(w, http.StatusOK, resp)
}

func (h *ContractHandler) Adopt(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())

	contractIDStr := r.PathValue("id")
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	var contract models.Contract
	err = h.db.QueryRow(`
		SELECT id, is_shared, is_active FROM contracts WHERE id = $1
	`, contractID).Scan(&contract.ID, &contract.IsShared, &contract.IsActive)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "contract not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch contract")
		return
	}

	if !contract.IsShared {
		api.RespondWithError(w, http.StatusBadRequest, "contract is not shared")
		return
	}

	if !contract.IsActive {
		api.RespondWithError(w, http.StatusBadRequest, "contract is not active")
		return
	}

	var existingCount int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM contract_adoptions WHERE contract_id = $1 AND organization_id = $2
	`, contractID, orgID).Scan(&existingCount)
	if existingCount > 0 {
		api.RespondWithError(w, http.StatusConflict, "contract already adopted")
		return
	}

	var adoption models.ContractAdoption
	err = h.db.QueryRow(`
		INSERT INTO contract_adoptions (contract_id, organization_id)
		VALUES ($1, $2)
		RETURNING id, contract_id, organization_id, adopted_at
	`, contractID, orgID).Scan(&adoption.ID, &adoption.ContractID, &adoption.OrganizationID, &adoption.AdoptedAt)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to adopt contract")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, adoption)
}
