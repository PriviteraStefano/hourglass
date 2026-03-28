package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type OrganizationHandler struct {
	db          *sql.DB
	authService *auth.Service
}

func NewOrganizationHandler(db *sql.DB, authService *auth.Service) *OrganizationHandler {
	return &OrganizationHandler{
		db:          db,
		authService: authService,
	}
}

type CreateOrganizationRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		api.RespondWithError(w, http.StatusBadRequest, "name is required")
		return
	}

	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}

	orgID := uuid.New()
	membershipID := uuid.New()
	now := time.Now()

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO organizations (id, name, slug, created_at)
		VALUES ($1, $2, $3, $4)
	`, orgID, req.Name, slug, now)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create organization")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, activated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, membershipID, userID, orgID, models.RoleFinance, true, now)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create membership")
		return
	}

	if err = tx.Commit(); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	org := models.Organization{
		ID:        orgID,
		Name:      req.Name,
		Slug:      slug,
		CreatedAt: now,
	}

	api.RespondWithJSON(w, http.StatusCreated, org)
}

func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "organization id is required")
		return
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	var org models.Organization
	err = h.db.QueryRow(`
		SELECT id, name, slug, created_at
		FROM organizations WHERE id = $1
	`, orgUUID).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "organization not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "database error")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, org)
}

func (h *OrganizationHandler) Invite(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req struct {
		Email string      `json:"email"`
		Role  models.Role `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || !req.Role.IsValid() {
		api.RespondWithError(w, http.StatusBadRequest, "email and valid role are required")
		return
	}

	membershipID := uuid.New()
	now := time.Now()

	_, err := h.db.Exec(`
		INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, invited_by, invited_at)
		VALUES ($1, NULL, $2, $3, $4, $5, $6)
	`, membershipID, orgID, req.Role, true, userID, now)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create invitation")
		return
	}

	// TODO: Send invitation email with activation token

	api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         membershipID,
		"email":      req.Email,
		"role":       req.Role,
		"invited_at": now,
	})
}

func (h *OrganizationHandler) InviteCustomer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string   `json:"email"`
		ContractIDs []string `json:"contract_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || len(req.ContractIDs) == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "email and contract_ids are required")
		return
	}

	// TODO: Implement customer invitation with contract assignments

	api.RespondWithJSON(w, http.StatusCreated, map[string]string{
		"message": "customer invitation created",
		"email":   req.Email,
	})
}
