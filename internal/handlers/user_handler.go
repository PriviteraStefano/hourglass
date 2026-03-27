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

type UserHandler struct {
	db          *sql.DB
	authService *auth.Service
}

func NewUserHandler(db *sql.DB, authService *auth.Service) *UserHandler {
	return &UserHandler{
		db:          db,
		authService: authService,
	}
}

type RegisterRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	Name             string `json:"name"`
	OrganizationName string `json:"organization_name"`
	InviteToken      string `json:"invite_token,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ActivateRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User  models.UserWithMembership `json:"user"`
	Token string                    `json:"token"`
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		api.RespondWithError(w, http.StatusBadRequest, "email, password, and name are required")
		return
	}

	if len(req.Password) < 8 {
		api.RespondWithError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	userID := uuid.New()
	orgID := uuid.New()
	membershipID := uuid.New()
	now := time.Now()

	tx, err := h.db.Begin()
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	if req.OrganizationName != "" {
		slug := generateSlug(req.OrganizationName)
		_, err = tx.Exec(`
			INSERT INTO organizations (id, name, slug, created_at)
			VALUES ($1, $2, $3, $4)
		`, orgID, req.OrganizationName, slug, now)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create organization")
			return
		}
	}

	_, err = tx.Exec(`
		INSERT INTO users (id, email, password_hash, name, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, req.Email, passwordHash, req.Name, true, now)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create user")
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

	token, err := h.authService.GenerateToken(userID, orgID, string(models.RoleFinance), req.Email)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	response := AuthResponse{
		User: models.UserWithMembership{
			User: models.User{
				ID:        userID,
				Email:     req.Email,
				Name:      req.Name,
				IsActive:  true,
				CreatedAt: now,
			},
			Membership: models.OrganizationMembership{
				ID:             membershipID,
				UserID:         userID,
				OrganizationID: orgID,
				Role:           models.RoleFinance,
				IsActive:       true,
			},
			Organization: models.Organization{
				ID:        orgID,
				Name:      req.OrganizationName,
				CreatedAt: now,
			},
		},
		Token: token,
	}

	api.RespondWithJSON(w, http.StatusCreated, response)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		api.RespondWithError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	var userID uuid.UUID
	var passwordHash string
	var name string
	var isActive bool

	err := h.db.QueryRow(`
		SELECT id, password_hash, name, is_active
		FROM users
		WHERE email = $1
	`, req.Email).Scan(&userID, &passwordHash, &name, &isActive)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "database error")
		return
	}

	if !isActive {
		api.RespondWithError(w, http.StatusUnauthorized, "account not activated")
		return
	}

	if !h.authService.CheckPassword(req.Password, passwordHash) {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	var membershipID uuid.UUID
	var orgID uuid.UUID
	var role string
	var membershipActivatedAt sql.NullTime

	err = h.db.QueryRow(`
		SELECT id, organization_id, role, activated_at
		FROM organization_memberships
		WHERE user_id = $1 AND is_active = true
		LIMIT 1
	`, userID).Scan(&membershipID, &orgID, &role, &membershipActivatedAt)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusInternalServerError, "no active organization membership")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "database error")
		return
	}

	token, err := h.authService.GenerateToken(userID, orgID, role, req.Email)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	var orgName string
	err = h.db.QueryRow(`SELECT name FROM organizations WHERE id = $1`, orgID).Scan(&orgName)
	if err != nil {
		orgName = ""
	}

	response := AuthResponse{
		User: models.UserWithMembership{
			User: models.User{
				ID:        userID,
				Email:     req.Email,
				Name:      name,
				IsActive:  true,
				CreatedAt: time.Now(),
			},
			Membership: models.OrganizationMembership{
				ID:             membershipID,
				UserID:         userID,
				OrganizationID: orgID,
				Role:           models.Role(role),
				IsActive:       true,
			},
			Organization: models.Organization{
				ID:   orgID,
				Name: orgName,
			},
		},
		Token: token,
	}

	api.RespondWithJSON(w, http.StatusOK, response)
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	api.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

func (h *UserHandler) Activate(w http.ResponseWriter, r *http.Request) {
	var req ActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" || req.Password == "" {
		api.RespondWithError(w, http.StatusBadRequest, "token and password are required")
		return
	}

	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	result, err := h.db.Exec(`
		UPDATE users
		SET password_hash = $1, is_active = true
		WHERE id = (
			SELECT user_id FROM organization_memberships
			WHERE invited_at IS NOT NULL AND activated_at IS NULL
			AND invited_at > NOW() - INTERVAL '7 days'
		)
	`, passwordHash)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to activate user")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		api.RespondWithError(w, http.StatusBadRequest, "invalid or expired activation token")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "account activated successfully"})
}

func generateSlug(name string) string {
	slug := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			slug += string(c)
		} else if c == ' ' || c == '-' || c == '_' {
			slug += "-"
		}
	}
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

	var user models.User
	err := h.db.QueryRow(`
		SELECT id, email, name, is_active, created_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.Name, &user.IsActive, &user.CreatedAt)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	var org models.Organization
	err = h.db.QueryRow(`
		SELECT id, name, slug, created_at
		FROM organizations WHERE id = $1
	`, orgID).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch organization")
		return
	}

	response := AuthResponse{
		User: models.UserWithMembership{
			User:         user,
			Organization: org,
		},
	}

	api.RespondWithJSON(w, http.StatusOK, response)
}
