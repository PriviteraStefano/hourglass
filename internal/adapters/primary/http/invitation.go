package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
	invitationsvc "github.com/stefanoprivitera/hourglass/internal/core/services/invitation"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type InvitationHandler struct {
	service *invitationsvc.Service
}

func NewInvitationHandler(service *invitationsvc.Service) *InvitationHandler {
	return &InvitationHandler{service: service}
}

type CreateInvitationRequest struct {
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email,omitempty"`
	ExpiresInDays  int    `json:"expires_in_days,omitempty"`
}

func (h *InvitationHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OrganizationID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "organization_id is required")
		return
	}

	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid organization_id")
		return
	}

	svcReq := &invitation.CreateInvitationRequest{
		OrganizationID: orgID,
		Email:          req.Email,
		ExpiresInDays:  req.ExpiresInDays,
	}

	inv, err := h.service.Create(ctx, svcReq)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create invitation")
		return
	}

	link := "https://app.example.com/invite/" + inv.InviteToken

	api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":              inv.ID.String(),
		"code":            inv.Code,
		"token":           inv.InviteToken,
		"link":            link,
		"email":           inv.Email,
		"status":          inv.Status,
		"expires_at":      inv.ExpiresAt,
		"organization_id": inv.OrganizationID.String(),
	})
}

func (h *InvitationHandler) ValidateCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := r.PathValue("code")

	inv, err := h.service.ValidateCode(ctx, code)
	if err != nil {
		if err == invitation.ErrInvitationNotFound {
			api.RespondWithError(w, http.StatusNotFound, "invitation not found")
			return
		}
		if err == invitation.ErrInvitationExpired {
			api.RespondWithError(w, http.StatusGone, "invitation has expired")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to validate invitation")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":              inv.ID.String(),
		"code":            inv.Code,
		"email":           inv.Email,
		"status":          inv.Status,
		"expires_at":      inv.ExpiresAt,
		"organization_id": inv.OrganizationID.String(),
	})
}

func (h *InvitationHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := r.PathValue("token")

	inv, err := h.service.ValidateToken(ctx, token)
	if err != nil {
		if err == invitation.ErrInvitationNotFound {
			api.RespondWithError(w, http.StatusNotFound, "invitation not found")
			return
		}
		if err == invitation.ErrInvitationExpired {
			api.RespondWithError(w, http.StatusGone, "invitation has expired")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to validate invitation")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":              inv.ID.String(),
		"code":            inv.Code,
		"email":           inv.Email,
		"status":          inv.Status,
		"expires_at":      inv.ExpiresAt,
		"organization_id": inv.OrganizationID.String(),
	})
}

type AcceptInvitationRequest struct {
	Token    string `json:"token"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *InvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req AcceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" || req.Email == "" || req.Password == "" {
		api.RespondWithError(w, http.StatusBadRequest, "token, email, and password are required")
		return
	}

	if len(req.Password) < 8 {
		api.RespondWithError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	inv, err := h.service.Accept(ctx, req.Token, req.Email, req.Username, req.Password)
	if err != nil {
		if err == invitation.ErrInvitationNotFound {
			api.RespondWithError(w, http.StatusNotFound, "invitation not found")
			return
		}
		if err == invitation.ErrInvitationUsed {
			api.RespondWithError(w, http.StatusGone, "invitation already used")
			return
		}
		if err == invitation.ErrInvitationExpired {
			api.RespondWithError(w, http.StatusGone, "invitation has expired")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to accept invitation")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "invitation valid, user creation would proceed here",
		"organization_id": inv.OrganizationID.String(),
	})
}
