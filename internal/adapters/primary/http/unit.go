package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	unitsvc "github.com/stefanoprivitera/hourglass/internal/core/services/unit"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type UnitHandler struct {
	service *unitsvc.Service
}

func NewUnitHandler(service *unitsvc.Service) *UnitHandler {
	return &UnitHandler{service: service}
}

type CreateUnitRequest struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	ParentUnitID *string `json:"parent_unit_id"`
	Code         string  `json:"code"`
}

type UpdateUnitRequest struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Code         string  `json:"code"`
	ParentUnitID *string `json:"parent_unit_id"`
}

func (h *UnitHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	units, err := h.service.ListByOrg(ctx, orgID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch units")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, units)
}

func (h *UnitHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	if unitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id")
		return
	}

	u, err := h.service.Get(ctx, unitID)
	if err != nil {
		if err == unit.ErrUnitNotFound {
			api.RespondWithError(w, http.StatusNotFound, "unit not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get unit")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, u)
}

func (h *UnitHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateUnitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		api.RespondWithError(w, http.StatusBadRequest, "name is required")
		return
	}

	orgID := middleware.GetOrganizationID(ctx)

	parentID := ""
	if req.ParentUnitID != nil {
		parentID = *req.ParentUnitID
	}

	svcReq := &unit.CreateUnitRequest{
		OrgID:        orgID,
		Name:         req.Name,
		Description:  req.Description,
		ParentUnitID: parentID,
		Code:         req.Code,
	}

	u, err := h.service.Create(ctx, svcReq)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create unit")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, u)
}

func (h *UnitHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	if unitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id")
		return
	}

	var req UpdateUnitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.service.Update(ctx, unitID, &unit.UpdateUnitRequest{
		Name:         req.Name,
		Description:  req.Description,
		Code:         req.Code,
		ParentUnitID: req.ParentUnitID,
	})
	if err != nil {
		if err == unit.ErrUnitNotFound {
			api.RespondWithError(w, http.StatusNotFound, "unit not found")
			return
		}
		if err == unit.ErrCircularParent {
			api.RespondWithError(w, http.StatusBadRequest, "cannot make unit a descendant of itself")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update unit")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, u)
}

func (h *UnitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	if unitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id")
		return
	}

	err := h.service.Delete(ctx, unitID)
	if err != nil {
		if err == unit.ErrCannotDeleteWithMembers {
			api.RespondWithError(w, http.StatusBadRequest, "cannot delete unit with members")
			return
		}
		if err == unit.ErrUnitNotFound {
			api.RespondWithError(w, http.StatusNotFound, "unit not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete unit")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UnitHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	tree, err := h.service.GetTree(ctx, orgID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get unit tree")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, tree)
}

func (h *UnitHandler) GetDescendants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	if unitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id")
		return
	}

	descendants, err := h.service.GetDescendants(ctx, unitID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get descendants")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, descendants)
}

type AddUnitMemberRequest struct {
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	IsPrimary bool   `json:"is_primary"`
}

func (h *UnitHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	if unitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id")
		return
	}

	if _, err := uuid.Parse(unitID); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id format")
		return
	}

	members, err := h.service.ListMembers(ctx, unitID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch members")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, members)
}

func (h *UnitHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	if unitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit id")
		return
	}

	orgID := middleware.GetOrganizationID(ctx)

	var req AddUnitMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	m, err := h.service.AddMember(ctx, unitID, orgID, &unit.AddUnitMemberRequest{
		UserID:    userID,
		Role:      req.Role,
		IsPrimary: req.IsPrimary,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, m)
}

func (h *UnitHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	membershipID := r.PathValue("membership_id")

	if membershipID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "invalid membership id")
		return
	}

	err := h.service.RemoveMember(ctx, membershipID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
