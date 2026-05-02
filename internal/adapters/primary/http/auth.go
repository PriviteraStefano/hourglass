package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/services/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type AuthHandler struct {
	authService       *auth.Service
	invitationService *invitation.Service
}

func NewAuthHandler(authService *auth.Service, invitationService *invitation.Service) *AuthHandler {
	return &AuthHandler{authService: authService, invitationService: invitationService}
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
		"data": resp,
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

	// Validate identifier format BEFORE calling service
	isEmail := strings.Contains(req.Identifier, "@")
	if !isEmail {
		// Validate username format
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

	api.RespondWithJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		_ = h.authService.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	api.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		api.RespondWithError(w, http.StatusUnauthorized, "refresh token required")
		return
	}
	resp, err := h.authService.Refresh(ctx, cookie.Value)
	if err != nil {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
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
	api.RespondWithJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	if userID == uuid.Nil {
		api.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.authService.GetProfile(ctx, userID, orgID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get profile")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	serviceReq := auth.BootstrapRequest{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
		OrgName:   req.OrganizationName,
	}
	resp, err := h.authService.Bootstrap(ctx, serviceReq)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "bootstrap failed: "+err.Error())
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

	api.RespondWithJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) BootstrapCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	anyExists, err := h.authService.AnyExists(ctx)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to check bootstrap status")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]bool{"needs_bootstrap": !anyExists})
}

type SwitchOrgRequest struct {
	OrganizationID string `json:"organization_id"`
}

func (h *AuthHandler) SwitchOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	if userID == uuid.Nil {
		api.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SwitchOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid organization id")
		return
	}

	resp, err := h.authService.SwitchOrganization(ctx, userID, orgID)
	if err != nil {
		switch err {
		case auth.ErrMembershipNotFound:
			api.RespondWithError(w, http.StatusForbidden, "not a member of this organization")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to switch organization")
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

	api.RespondWithJSON(w, http.StatusOK, resp)
}

type MembershipsResponse struct {
	Memberships []MembershipWithOrg `json:"memberships"`
}

type MembershipWithOrg struct {
	Membership   auth.Membership   `json:"membership"`
	Organization auth.Organization `json:"organization"`
}

func (h *AuthHandler) GetMemberships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	if userID == uuid.Nil {
		api.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	memberships, err := h.authService.GetMemberships(ctx, userID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get memberships")
		return
	}

	result := make([]MembershipWithOrg, 0, len(memberships))
	for _, m := range memberships {
		org, _ := h.authService.GetOrgByID(ctx, m.OrganizationID)
		activatedAt := m.ActivatedAt
		result = append(result, MembershipWithOrg{
			Membership: auth.Membership{
				ID:             m.ID.String(),
				UserID:         m.UserID.String(),
				OrganizationID: m.OrganizationID.String(),
				Role:           m.Role,
				IsActive:       m.IsActive,
				ActivatedAt:    activatedAt,
			},
			Organization: auth.Organization{
				ID:   org.ID.String(),
				Name: org.Name,
				Slug: org.Slug,
			},
		})
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"memberships": result})
}
