package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	orgdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/organization"
	orgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/organization"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type OrganizationHandler struct {
	service *orgsvc.Service
}

func NewOrganizationHandler(service *orgsvc.Service) *OrganizationHandler {
	return &OrganizationHandler{service: service}
}

type CreateOrganizationRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	org, err := h.service.Create(r.Context(), middleware.GetUserID(r.Context()), &orgdomain.CreateOrganizationRequest{Name: req.Name, Slug: req.Slug})
	if err != nil {
		if err == orgdomain.ErrInvalidRequest {
			api.RespondWithError(w, http.StatusBadRequest, "name is required")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create organization")
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, org)
}

func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgUUID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	org, err := h.service.Get(r.Context(), orgUUID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "organization not found")
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
	membershipID, invitedAt, err := h.service.Invite(r.Context(), orgID, userID, &orgdomain.InviteRequest{Email: req.Email, Role: req.Role})
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "email and valid role are required")
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         membershipID,
		"email":      req.Email,
		"role":       req.Role,
		"invited_at": invitedAt,
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
	api.RespondWithJSON(w, http.StatusCreated, map[string]string{"message": "customer invitation created", "email": req.Email})
}

func (h *OrganizationHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	orgUUID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	settings, err := h.service.GetSettings(r.Context(), orgUUID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "organization settings not found")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, settings)
}

type UpdateSettingsRequest struct {
	DefaultKmRate       *float64 `json:"default_km_rate,omitempty"`
	Currency            string   `json:"currency,omitempty"`
	WeekStartDay        *int     `json:"week_start_day,omitempty"`
	Timezone            string   `json:"timezone,omitempty"`
	ShowApprovalHistory *bool    `json:"show_approval_history,omitempty"`
}

func (h *OrganizationHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	orgUUID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid organization id")
		return
	}
	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	settings, err := h.service.UpdateSettings(r.Context(), middleware.GetRole(r.Context()), orgUUID, &orgdomain.UpdateSettingsRequest{
		DefaultKmRate:       req.DefaultKmRate,
		Currency:            req.Currency,
		WeekStartDay:        req.WeekStartDay,
		Timezone:            req.Timezone,
		ShowApprovalHistory: req.ShowApprovalHistory,
	})
	if err != nil {
		if err == orgdomain.ErrForbidden {
			api.RespondWithError(w, http.StatusForbidden, "only finance users can update organization settings")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, settings)
}

func (h *OrganizationHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	members, err := h.service.ListMembers(r.Context(), middleware.GetOrganizationID(r.Context()))
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch members")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, members)
}

type UpdateRolesRequest struct {
	Roles []string `json:"roles"`
}

func (h *OrganizationHandler) UpdateMemberRoles(w http.ResponseWriter, r *http.Request) {
	memberID, err := uuid.Parse(r.PathValue("member_id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid member id")
		return
	}
	var req UpdateRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	err = h.service.UpdateMemberRoles(r.Context(), middleware.GetRole(r.Context()), middleware.GetOrganizationID(r.Context()), memberID, req.Roles)
	if err != nil {
		switch err {
		case orgdomain.ErrForbidden:
			api.RespondWithError(w, http.StatusForbidden, "only finance users can update member roles")
		case orgdomain.ErrInvalidRequest:
			api.RespondWithError(w, http.StatusBadRequest, "at least one valid role is required")
		case orgdomain.ErrMemberNotFound:
			api.RespondWithError(w, http.StatusNotFound, "member not found")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to update role")
		}
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"id": memberID.String(), "roles": req.Roles})
}

func (h *OrganizationHandler) DeactivateMember(w http.ResponseWriter, r *http.Request) {
	memberID, err := uuid.Parse(r.PathValue("member_id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid member id")
		return
	}
	err = h.service.DeactivateMember(r.Context(), middleware.GetRole(r.Context()), middleware.GetOrganizationID(r.Context()), memberID)
	if err != nil {
		switch err {
		case orgdomain.ErrForbidden:
			api.RespondWithError(w, http.StatusForbidden, "only finance users can deactivate members")
		case orgdomain.ErrLastFinance:
			api.RespondWithError(w, http.StatusBadRequest, "cannot deactivate the last finance user in the organization")
		case orgdomain.ErrMemberNotFound:
			api.RespondWithError(w, http.StatusNotFound, "member not found")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to deactivate member")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
