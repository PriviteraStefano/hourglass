package handlers

import "github.com/google/uuid"

// import (
// 	"database/sql"
// 	"encoding/json"
// 	"log"
// 	"net/http"
// 	"os"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/stefanoprivitera/hourglass/internal/auth"
// 	"github.com/stefanoprivitera/hourglass/internal/cookies"
// 	"github.com/stefanoprivitera/hourglass/internal/middleware"
// 	"github.com/stefanoprivitera/hourglass/internal/models"
// 	"github.com/stefanoprivitera/hourglass/pkg/api"
// )

// type UserHandler struct {
// 	db          *sql.DB
// 	authService *auth.Service
// }

// func NewUserHandler(db *sql.DB, authService *auth.Service) *UserHandler {
// 	return &UserHandler{
// 		db:          db,
// 		authService: authService,
// 	}
// }

// type LoginRequest struct {
// 	Email    string `json:"email"`
// 	Password string `json:"password"`
// }

// type ActivateRequest struct {
// 	Token    string `json:"token"`
// 	Password string `json:"password"`
// }

// func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
// 	var req RegisterRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
// 		return
// 	}

// 	if req.Email == "" || req.Password == "" || req.Name == "" {
// 		api.RespondWithError(w, http.StatusBadRequest, "email, password, and name are required")
// 		return
// 	}

// 	if len(req.Password) < 8 {
// 		api.RespondWithError(w, http.StatusBadRequest, "password must be at least 8 characters")
// 		return
// 	}

// 	passwordHash, err := h.authService.HashPassword(req.Password)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
// 		return
// 	}

// 	userID := uuid.New()
// 	orgID := uuid.New()
// 	membershipID := uuid.New()
// 	now := time.Now()

// 	tx, err := h.db.Begin()
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
// 		return
// 	}
// 	defer tx.Rollback()

// 	if req.OrganizationName != "" {
// 		slug := generateSlug(req.OrganizationName)
// 		_, err = tx.Exec(`
// 			INSERT INTO organizations (id, name, slug, created_at)
// 			VALUES ($1, $2, $3, $4)
// 		`, orgID, req.OrganizationName, slug, now)
// 		if err != nil {
// 			api.RespondWithError(w, http.StatusInternalServerError, "failed to create organization")
// 			return
// 		}
// 	}

// 	_, err = tx.Exec(`
// 		INSERT INTO users (id, email, password_hash, name, is_active, created_at)
// 		VALUES ($1, $2, $3, $4, $5, $6)
// 	`, userID, req.Email, passwordHash, req.Name, false, now)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to create user")
// 		return
// 	}

// 	verificationToken := generateVerificationToken()
// 	_, err = tx.Exec(`
// 		INSERT INTO verification_tokens (user_id, token, type, expires_at)
// 		VALUES ($1, $2, 'email_verification', $3)
// 	`, userID, verificationToken, now.Add(24*time.Hour))
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to create verification token")
// 		return
// 	}

// 	_, err = tx.Exec(`
// 		INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, activated_at)
// 		VALUES ($1, $2, $3, $4, $5, $6)
// 	`, membershipID, userID, orgID, models.RoleFinance, false, nil)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to create membership")
// 		return
// 	}

// 	if err = tx.Commit(); err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
// 		return
// 	}

// 	env := os.Getenv("ENV")
// 	if env == "" || env == "development" {
// 		log.Printf("Verification token for %s: %s", req.Email, verificationToken)
// 	}

// 	response := map[string]interface{}{
// 		"message":      "registration successful, please check your email to verify your account",
// 		"email":        req.Email,
// 		"verify_token": verificationToken,
// 	}

// 	api.RespondWithJSON(w, http.StatusCreated, response)
// }

// func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
// 	var req LoginRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
// 		return
// 	}

// 	if req.Email == "" || req.Password == "" {
// 		api.RespondWithError(w, http.StatusBadRequest, "email and password are required")
// 		return
// 	}

// 	var userID uuid.UUID
// 	var passwordHash string
// 	var name string
// 	var isActive bool

// 	err := h.db.QueryRow(`
// 		SELECT id, password_hash, name, is_active
// 		FROM users
// 		WHERE email = $1
// 	`, req.Email).Scan(&userID, &passwordHash, &name, &isActive)
// 	if err == sql.ErrNoRows {
// 		api.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
// 		return
// 	}
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "database error")
// 		return
// 	}

// 	if !isActive {
// 		api.RespondWithError(w, http.StatusUnauthorized, "account not activated")
// 		return
// 	}

// 	if !h.authService.CheckPassword(req.Password, passwordHash) {
// 		api.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
// 		return
// 	}

// 	var membershipID uuid.UUID
// 	var orgID uuid.UUID
// 	var role string
// 	var membershipActivatedAt sql.NullTime

// 	err = h.db.QueryRow(`
// 		SELECT id, organization_id, role, activated_at
// 		FROM organization_memberships
// 		WHERE user_id = $1 AND is_active = true
// 		LIMIT 1
// 	`, userID).Scan(&membershipID, &orgID, &role, &membershipActivatedAt)
// 	if err == sql.ErrNoRows {
// 		api.RespondWithError(w, http.StatusInternalServerError, "no active organization membership")
// 		return
// 	}
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "database error")
// 		return
// 	}

// 	accessToken, err := h.authService.GenerateToken(userID, orgID, role, req.Email)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate access token")
// 		return
// 	}

// 	refreshToken, err := h.authService.GenerateRefreshToken()
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate refresh token")
// 		return
// 	}

// 	refreshTokenHash := auth.HashRefreshToken(refreshToken)
// 	now := time.Now()
// 	_, err = h.db.Exec(`
// 		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
// 		VALUES ($1, $2, $3)
// 	`, userID, refreshTokenHash, now.Add(auth.RefreshTokenExpiry))
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to store refresh token")
// 		return
// 	}

// 	secure := cookies.IsSecureRequest(r)
// 	cookies.SetAccessTokenCookie(w, accessToken, secure)
// 	cookies.SetRefreshTokenCookie(w, refreshToken, secure)

// 	var orgName string
// 	err = h.db.QueryRow(`SELECT name FROM organizations WHERE id = $1`, orgID).Scan(&orgName)
// 	if err != nil {
// 		orgName = ""
// 	}

// 	response := models.UserWithMembership{
// 		User: models.User{
// 			ID:        userID,
// 			Email:     req.Email,
// 			Name:      name,
// 			IsActive:  true,
// 			CreatedAt: time.Now(),
// 		},
// 		Membership: models.OrganizationMembership{
// 			ID:             membershipID,
// 			UserID:         userID,
// 			OrganizationID: orgID,
// 			Role:           models.Role(role),
// 			IsActive:       true,
// 		},
// 		Organization: models.Organization{
// 			ID:   orgID,
// 			Name: orgName,
// 		},
// 	}

// 	api.RespondWithJSON(w, http.StatusOK, response)
// }

// func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
// 	refreshToken, err := cookies.GetRefreshTokenFromCookie(r)
// 	if err == nil {
// 		tokenHash := auth.HashRefreshToken(refreshToken)
// 		_, _ = h.db.Exec(`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1`, tokenHash)
// 	}

// 	cookies.ClearAuthCookies(w)
// 	api.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
// }

// type VerifyRequest struct {
// 	Token string `json:"token"`
// }

// func (h *UserHandler) Verify(w http.ResponseWriter, r *http.Request) {
// 	var req VerifyRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
// 		return
// 	}

// 	if req.Token == "" {
// 		api.RespondWithError(w, http.StatusBadRequest, "token is required")
// 		return
// 	}

// 	tx, err := h.db.Begin()
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
// 		return
// 	}
// 	defer tx.Rollback()

// 	var userID uuid.UUID
// 	err = tx.QueryRow(`
// 		SELECT user_id FROM verification_tokens
// 		WHERE token = $1 AND type = 'email_verification' AND expires_at > NOW()
// 	`, req.Token).Scan(&userID)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid or expired verification token")
// 		return
// 	}

// 	_, err = tx.Exec("UPDATE users SET is_active = true WHERE id = $1", userID)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to activate user")
// 		return
// 	}

// 	_, err = tx.Exec("DELETE FROM verification_tokens WHERE token = $1", req.Token)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete verification token")
// 		return
// 	}

// 	if err = tx.Commit(); err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
// 		return
// 	}

// 	api.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "email verified successfully"})
// }

// type ForgotPasswordRequest struct {
// 	Email string `json:"email"`
// }

// func (h *UserHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
// 	var req ForgotPasswordRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
// 		return
// 	}

// 	if req.Email == "" {
// 		api.RespondWithError(w, http.StatusBadRequest, "email is required")
// 		return
// 	}

// 	var userID uuid.UUID
// 	err := h.db.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)
// 	if err != nil {
// 		api.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a reset link has been sent"})
// 		return
// 	}

// 	resetToken := generateVerificationToken()
// 	_, err = h.db.Exec(`
// 		INSERT INTO verification_tokens (user_id, token, type, expires_at)
// 		VALUES ($1, $2, 'password_reset', NOW() + INTERVAL '1 hour')
// 	`, userID, resetToken)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to create reset token")
// 		return
// 	}

// 	env := os.Getenv("ENV")
// 	if env == "" || env == "development" {
// 		log.Printf("Password reset token for %s: %s", req.Email, resetToken)
// 	}

// 	api.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a reset link has been sent"})
// }

// type ResetPasswordRequest struct {
// 	Token    string `json:"token"`
// 	Password string `json:"password"`
// }

// func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
// 	var req ResetPasswordRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
// 		return
// 	}

// 	if req.Token == "" || req.Password == "" {
// 		api.RespondWithError(w, http.StatusBadRequest, "token and password are required")
// 		return
// 	}

// 	if len(req.Password) < 8 {
// 		api.RespondWithError(w, http.StatusBadRequest, "password must be at least 8 characters")
// 		return
// 	}

// 	passwordHash, err := h.authService.HashPassword(req.Password)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
// 		return
// 	}

// 	tx, err := h.db.Begin()
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to begin transaction")
// 		return
// 	}
// 	defer tx.Rollback()

// 	var userID uuid.UUID
// 	err = tx.QueryRow(`
// 		SELECT user_id FROM verification_tokens
// 		WHERE token = $1 AND type = 'password_reset' AND expires_at > NOW()
// 	`, req.Token).Scan(&userID)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid or expired reset token")
// 		return
// 	}

// 	_, err = tx.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", passwordHash, userID)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to update password")
// 		return
// 	}

// 	_, err = tx.Exec("DELETE FROM verification_tokens WHERE token = $1", req.Token)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete reset token")
// 		return
// 	}

// 	if err = tx.Commit(); err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
// 		return
// 	}

// 	api.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "password reset successfully"})
// }

// func (h *UserHandler) Activate(w http.ResponseWriter, r *http.Request) {
// 	var req ActivateRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
// 		return
// 	}

// 	if req.Token == "" || req.Password == "" {
// 		api.RespondWithError(w, http.StatusBadRequest, "token and password are required")
// 		return
// 	}

// 	passwordHash, err := h.authService.HashPassword(req.Password)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
// 		return
// 	}

// 	result, err := h.db.Exec(`
// 		UPDATE users
// 		SET password_hash = $1, is_active = true
// 		WHERE id = (
// 			SELECT user_id FROM organization_memberships
// 			WHERE invited_at IS NOT NULL AND activated_at IS NULL
// 			AND invited_at > NOW() - INTERVAL '7 days'
// 		)
// 	`, passwordHash)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to activate user")
// 		return
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid or expired activation token")
// 		return
// 	}

// 	api.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "account activated successfully"})
// }

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

func generateVerificationToken() string {
	return uuid.New().String()
}

// func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
// 	userID := middleware.GetUserID(r.Context())
// 	orgID := middleware.GetOrganizationID(r.Context())

// 	var user models.User
// 	err := h.db.QueryRow(`
// 		SELECT id, email, name, is_active, created_at
// 		FROM users WHERE id = $1
// 	`, userID).Scan(&user.ID, &user.Email, &user.Name, &user.IsActive, &user.CreatedAt)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch user")
// 		return
// 	}

// 	var org models.Organization
// 	err = h.db.QueryRow(`
// 		SELECT id, name, slug, created_at
// 		FROM organizations WHERE id = $1
// 	`, orgID).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch organization")
// 		return
// 	}

// 	response := models.UserWithMembership{
// 		User:         user,
// 		Organization: org,
// 	}

// 	api.RespondWithJSON(w, http.StatusOK, response)
// }

// func (h *UserHandler) Refresh(w http.ResponseWriter, r *http.Request) {
// 	refreshToken, err := cookies.GetRefreshTokenFromCookie(r)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusUnauthorized, "missing refresh token")
// 		return
// 	}

// 	tokenHash := auth.HashRefreshToken(refreshToken)

// 	var userID uuid.UUID
// 	var expiresAt time.Time
// 	var revokedAt sql.NullTime

// 	err = h.db.QueryRow(`
// 		SELECT user_id, expires_at, revoked_at
// 		FROM refresh_tokens
// 		WHERE token_hash = $1
// 	`, tokenHash).Scan(&userID, &expiresAt, &revokedAt)
// 	if err == sql.ErrNoRows {
// 		api.RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
// 		return
// 	}
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "database error")
// 		return
// 	}

// 	if revokedAt.Valid {
// 		api.RespondWithError(w, http.StatusUnauthorized, "refresh token has been revoked")
// 		return
// 	}

// 	if time.Now().After(expiresAt) {
// 		api.RespondWithError(w, http.StatusUnauthorized, "refresh token has expired")
// 		return
// 	}

// 	var email string
// 	var name string
// 	err = h.db.QueryRow(`SELECT email, name FROM users WHERE id = $1`, userID).Scan(&email, &name)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch user")
// 		return
// 	}

// 	var membershipID uuid.UUID
// 	var orgID uuid.UUID
// 	var role string
// 	err = h.db.QueryRow(`
// 		SELECT id, organization_id, role
// 		FROM organization_memberships
// 		WHERE user_id = $1 AND is_active = true
// 		LIMIT 1
// 	`, userID).Scan(&membershipID, &orgID, &role)
// 	if err == sql.ErrNoRows {
// 		api.RespondWithError(w, http.StatusInternalServerError, "no active organization membership")
// 		return
// 	}
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "database error")
// 		return
// 	}

// 	accessToken, err := h.authService.GenerateToken(userID, orgID, role, email)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate access token")
// 		return
// 	}

// 	newRefreshToken, err := h.authService.GenerateRefreshToken()
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate refresh token")
// 		return
// 	}

// 	newTokenHash := auth.HashRefreshToken(newRefreshToken)
// 	_, err = h.db.Exec(`
// 		UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1
// 	`, tokenHash)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to revoke old token")
// 		return
// 	}

// 	_, err = h.db.Exec(`
// 		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
// 		VALUES ($1, $2, $3)
// 	`, userID, newTokenHash, time.Now().Add(auth.RefreshTokenExpiry))
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to store new refresh token")
// 		return
// 	}

// 	secure := cookies.IsSecureRequest(r)
// 	cookies.SetAccessTokenCookie(w, accessToken, secure)
// 	cookies.SetRefreshTokenCookie(w, newRefreshToken, secure)

// 	var orgName string
// 	err = h.db.QueryRow(`SELECT name FROM organizations WHERE id = $1`, orgID).Scan(&orgName)
// 	if err != nil {
// 		orgName = ""
// 	}

// 	response := models.UserWithMembership{
// 		User: models.User{
// 			ID:        userID,
// 			Email:     email,
// 			Name:      name,
// 			IsActive:  true,
// 			CreatedAt: time.Now(),
// 		},
// 		Membership: models.OrganizationMembership{
// 			ID:             membershipID,
// 			UserID:         userID,
// 			OrganizationID: orgID,
// 			Role:           models.Role(role),
// 			IsActive:       true,
// 		},
// 		Organization: models.Organization{
// 			ID:   orgID,
// 			Name: orgName,
// 		},
// 	}

// 	api.RespondWithJSON(w, http.StatusOK, response)
// }

// func (h *UserHandler) SwitchOrg(w http.ResponseWriter, r *http.Request) {
// 	userID := middleware.GetUserID(r.Context())

// 	var req struct {
// 		OrganizationID string `json:"organization_id"`
// 	}
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
// 		return
// 	}

// 	if req.OrganizationID == "" {
// 		api.RespondWithError(w, http.StatusBadRequest, "organization_id is required")
// 		return
// 	}

// 	orgID, err := uuid.Parse(req.OrganizationID)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusBadRequest, "invalid organization id")
// 		return
// 	}

// 	var membership models.OrganizationMembership
// 	var role string
// 	err = h.db.QueryRow(`
// 		SELECT id, user_id, organization_id, role, is_active
// 		FROM organization_memberships
// 		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
// 	`, userID, orgID).Scan(&membership.ID, &membership.UserID, &membership.OrganizationID, &role, &membership.IsActive)
// 	if err == sql.ErrNoRows {
// 		api.RespondWithError(w, http.StatusForbidden, "you do not have access to this organization")
// 		return
// 	}
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to verify membership")
// 		return
// 	}

// 	accessToken, err := h.authService.GenerateToken(userID, orgID, role, "")
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate token")
// 		return
// 	}

// 	refreshToken, err := h.authService.GenerateRefreshToken()
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate refresh token")
// 		return
// 	}

// 	refreshTokenHash := auth.HashRefreshToken(refreshToken)
// 	now := time.Now()
// 	_, err = h.db.Exec(`
// 		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
// 		VALUES ($1, $2, $3)
// 	`, userID, refreshTokenHash, now.Add(auth.RefreshTokenExpiry))
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to store refresh token")
// 		return
// 	}

// 	secure := cookies.IsSecureRequest(r)
// 	cookies.SetAccessTokenCookie(w, accessToken, secure)
// 	cookies.SetRefreshTokenCookie(w, refreshToken, secure)

// 	var org models.Organization
// 	err = h.db.QueryRow(`SELECT id, name, slug, created_at FROM organizations WHERE id = $1`, orgID).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt)
// 	if err != nil {
// 		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch organization")
// 		return
// 	}

// 	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
// 		"organization": org,
// 		"role":         role,
// 	})
// }
