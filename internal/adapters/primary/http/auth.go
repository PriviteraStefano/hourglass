package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type AuthHandler struct {
	authService *auth.Service
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Email            string `json:"email"`
	Username         string `json:"username"`
	FirstName        string `json:"firstname"`
	LastName         string `json:"lastname"`
	Name             string `json:"name"`
	Password         string `json:"password"`
	OrganizationName string `json:"organization_name"`
	InviteToken      string `json:"invite_token,omitempty"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	serviceReq := auth.RegisterRequest{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Name:      req.Name,
		Password:  req.Password,
		OrgName:   req.OrganizationName,
	}

	resp, err := h.authService.Register(ctx, serviceReq)
	if err != nil {
		switch err {
		case auth.ErrEmailExists:
			api.RespondWithError(w, http.StatusConflict, "email already registered")
		case auth.ErrUsernameExists:
			api.RespondWithError(w, http.StatusConflict, "username already taken")
		default:
			api.RespondWithError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      resp.UserID,
		"email":   resp.Email,
		"name":    resp.Name,
		"message": "registration successful",
	})
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Identifier == "" || req.Password == "" {
		api.RespondWithError(w, http.StatusBadRequest, "identifier and password are required")
		return
	}

	isEmail := strings.Contains(req.Identifier, "@")
	if !isEmail {
		usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
		if !usernameRegex.MatchString(req.Identifier) {
			api.RespondWithError(w, http.StatusBadRequest, "username can only contain letters, numbers, and underscores")
			return
		}
	}

	serviceReq := auth.LoginRequest{
		Identifier: req.Identifier,
		Password:   req.Password,
	}

	resp, err := h.authService.Login(ctx, serviceReq)
	if err != nil {
		switch err {
		case auth.ErrInvalidCreds:
			api.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
		case auth.ErrAccountDeactivated:
			api.RespondWithError(w, http.StatusForbidden, "account is deactivated")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "login failed")
		}
		return
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    resp.Token,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	refreshExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		Expires:  refreshExpiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":     resp.UserID,
		"email":  resp.Email,
		"name":   resp.Name,
		"role":   resp.Role,
		"org_id": resp.OrgID,
	})
}

func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	resp, err := h.authService.GetProfile(ctx, userID.String())
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":         resp.ID,
		"email":      resp.Email,
		"name":       resp.Name,
		"is_active":  resp.IsActive,
		"created_at": resp.CreatedAt,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		api.RespondWithError(w, http.StatusBadRequest, "refresh token is required")
		return
	}

	resp, err := h.authService.Refresh(ctx, cookie.Value)
	if err != nil {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    resp.Token,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"id":     resp.UserID,
			"email":  resp.Email,
			"name":   resp.Name,
			"role":   resp.Role,
			"org_id": resp.OrgID,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		h.authService.Logout(ctx, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "auth_token",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "refresh_token",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	w.WriteHeader(http.StatusNoContent)
}

type BootstrapRequest struct {
	OrgName   string `json:"org_name"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Password  string `json:"password"`
}

func (h *AuthHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req BootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	serviceReq := auth.BootstrapRequest{
		OrgName:   req.OrgName,
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
	}

	resp, err := h.authService.Bootstrap(ctx, serviceReq)
	if err != nil {
		switch err {
		case auth.ErrEmailExists:
			api.RespondWithError(w, http.StatusConflict, "email already registered")
		case auth.ErrUsernameExists:
			api.RespondWithError(w, http.StatusConflict, "username already taken")
		default:
			api.RespondWithError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"token": resp.Token,
		"user": map[string]interface{}{
			"id":       resp.UserID,
			"email":    resp.Email,
			"username": resp.Username,
			"name":     resp.Name,
		},
		"organization": map[string]interface{}{
			"id":   resp.OrgID,
			"name": resp.OrgName,
		},
	})
}
