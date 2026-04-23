package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	wgsvc "github.com/stefanoprivitera/hourglass/internal/core/services/working_group"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type WorkingGroupHandler struct {
	service *wgsvc.Service
}

func NewWorkingGroupHandler(service *wgsvc.Service) *WorkingGroupHandler {
	return &WorkingGroupHandler{service: service}
}

type CreateWorkingGroupRequest struct {
	OrgID            string   `json:"org_id"`
	SubprojectID     string   `json:"subproject_id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	UnitIDs          []string `json:"unit_ids"`
	EnforceUnitTuple bool     `json:"enforce_unit_tuple"`
	ManagerID        string   `json:"manager_id"`
	DelegateIDs      []string `json:"delegate_ids"`
}

type UpdateWorkingGroupRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	UnitIDs          []string `json:"unit_ids"`
	EnforceUnitTuple *bool    `json:"enforce_unit_tuple"`
	ManagerID        string   `json:"manager_id"`
	DelegateIDs      []string `json:"delegate_ids"`
}

type AddMemberRequest struct {
	UserID              string `json:"user_id"`
	UnitID              string `json:"unit_id"`
	Role                string `json:"role"`
	IsDefaultSubproject bool   `json:"is_default_subproject"`
}

func (h *WorkingGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	subprojectIDStr := r.URL.Query().Get("subproject_id")
	var subprojectID *uuid.UUID
	if subprojectIDStr != "" {
		pid, err := uuid.Parse(subprojectIDStr)
		if err == nil {
			subprojectID = &pid
		}
	}

	wgs, err := h.service.ListByOrg(ctx, orgID, subprojectID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch working groups")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, wgs)
}

func (h *WorkingGroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgIDStr := r.PathValue("id")

	wgID, err := uuid.Parse(wgIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid working group id")
		return
	}

	wg, err := h.service.Get(ctx, wgID)
	if err != nil {
		if err == working_group.ErrWorkingGroupNotFound {
			api.RespondWithError(w, http.StatusNotFound, "working group not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get working group")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, wg)
}

func (h *WorkingGroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	var req CreateWorkingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		api.RespondWithError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.SubprojectID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "subproject_id is required")
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid org_id")
		return
	}
	subprojectID, err := uuid.Parse(req.SubprojectID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid subproject_id")
		return
	}
	managerID, err := uuid.Parse(req.ManagerID)
	if err != nil {
		managerID, _ = uuid.Parse(userID.String())
	}

	svcReq := &working_group.CreateWorkingGroupRequest{
		OrgID:            orgID,
		SubprojectID:     subprojectID,
		Name:             req.Name,
		Description:      req.Description,
		UnitIDs:          req.UnitIDs,
		EnforceUnitTuple: req.EnforceUnitTuple,
		ManagerID:        managerID,
		DelegateIDs:      req.DelegateIDs,
	}

	wg, err := h.service.Create(ctx, svcReq)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create working group")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, wg)
}

func (h *WorkingGroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgIDStr := r.PathValue("id")

	wgID, err := uuid.Parse(wgIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid working group id")
		return
	}

	var req UpdateWorkingGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var managerID uuid.UUID
	if req.ManagerID != "" {
		managerID, err = uuid.Parse(req.ManagerID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid manager_id")
			return
		}
	}

	wg, err := h.service.Update(ctx, wgID, &working_group.UpdateWorkingGroupRequest{
		Name:             req.Name,
		Description:      req.Description,
		UnitIDs:          req.UnitIDs,
		EnforceUnitTuple: req.EnforceUnitTuple,
		ManagerID:        managerID,
		DelegateIDs:      req.DelegateIDs,
	})
	if err != nil {
		if err == working_group.ErrWorkingGroupNotFound {
			api.RespondWithError(w, http.StatusNotFound, "working group not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update working group")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, wg)
}

func (h *WorkingGroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgIDStr := r.PathValue("id")

	wgID, err := uuid.Parse(wgIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid working group id")
		return
	}

	err = h.service.Delete(ctx, wgID)
	if err != nil {
		if err == working_group.ErrCannotDeleteWithMembers {
			api.RespondWithError(w, http.StatusBadRequest, "cannot delete working group with members")
			return
		}
		if err == working_group.ErrWorkingGroupNotFound {
			api.RespondWithError(w, http.StatusNotFound, "working group not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete working group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkingGroupHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgIDStr := r.PathValue("id")

	wgID, err := uuid.Parse(wgIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid working group id")
		return
	}

	members, err := h.service.ListMembers(ctx, wgID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch members")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, members)
}

func (h *WorkingGroupHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgIDStr := r.PathValue("id")

	wgID, err := uuid.Parse(wgIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid working group id")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" || req.UnitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "user_id and unit_id are required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	unitID, err := uuid.Parse(req.UnitID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit_id")
		return
	}

	m, err := h.service.AddMember(ctx, &working_group.AddMemberRequest{
		WGID:                wgID,
		UserID:              userID,
		UnitID:              unitID,
		Role:                req.Role,
		IsDefaultSubproject: req.IsDefaultSubproject,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, m)
}

func (h *WorkingGroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	memberIDStr := r.PathValue("member_id")

	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid member id")
		return
	}

	err = h.service.RemoveMember(ctx, memberID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
