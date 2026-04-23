package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	tesvc "github.com/stefanoprivitera/hourglass/internal/core/services/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type TimeEntryHandler struct {
	service *tesvc.Service
}

func NewTimeEntryHandler(service *tesvc.Service) *TimeEntryHandler {
	return &TimeEntryHandler{service: service}
}

type CreateTimeEntryRequest struct {
	ProjectID    string  `json:"project_id"`
	SubprojectID string  `json:"subproject_id"`
	WGID         string  `json:"wg_id"`
	UnitID       string  `json:"unit_id"`
	Hours        float64 `json:"hours"`
	Description  string  `json:"description"`
	Date         string  `json:"date"`
}

type UpdateTimeEntryRequest struct {
	ProjectID    *string  `json:"project_id,omitempty"`
	SubprojectID *string  `json:"subproject_id,omitempty"`
	WGID         *string  `json:"wg_id,omitempty"`
	UnitID       *string  `json:"unit_id,omitempty"`
	Hours        *float64 `json:"hours,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Date         *string  `json:"date,omitempty"`
}

type RejectRequest struct {
	Reason string `json:"reason"`
}

func (h *TimeEntryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)

	filters := ports.ListFilters{
		Role:          role,
		RequestUserID: userID.String(),
		IsDeleted:     false,
	}

	if date := r.URL.Query().Get("date"); date != "" {
		filters.Date = date
	}
	if month := r.URL.Query().Get("month"); month != "" {
		filters.Month = month
	}
	if year := r.URL.Query().Get("year"); year != "" {
		filters.Year = year
	}
	if filterUserID := r.URL.Query().Get("user_id"); filterUserID != "" {
		filters.UserID = filterUserID
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = status
	}
	if wgID := r.URL.Query().Get("wg_id"); wgID != "" {
		filters.WGID = wgID
	}
	if projectID := r.URL.Query().Get("project_id"); projectID != "" {
		filters.ProjectID = projectID
	}

	entries, err := h.service.List(ctx, orgID, filters)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entries")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, entries)
}

func (h *TimeEntryHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	entryIDStr := r.PathValue("id")

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	e, err := h.service.Get(ctx, entryID)
	if err != nil {
		if err == time_entry.ErrTimeEntryNotFound {
			api.RespondWithError(w, http.StatusNotFound, "time entry not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to get time entry")
		return
	}

	if e.OrgID != orgID {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	if role == "employee" && e.UserID != userID {
		api.RespondWithError(w, http.StatusForbidden, "can only view own entries")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *TimeEntryHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)

	var req CreateTimeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ProjectID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "project_id is required")
		return
	}
	if req.SubprojectID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "subproject_id is required")
		return
	}
	if req.WGID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "wg_id is required")
		return
	}
	if req.UnitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "unit_id is required")
		return
	}
	if req.Hours <= 0 || req.Hours > 24 {
		api.RespondWithError(w, http.StatusBadRequest, "hours must be greater than 0 and not exceed 24")
		return
	}
	if req.Date == "" {
		api.RespondWithError(w, http.StatusBadRequest, "date is required")
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid project_id")
		return
	}
	subprojectID, err := uuid.Parse(req.SubprojectID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid subproject_id")
		return
	}
	wgID, err := uuid.Parse(req.WGID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid wg_id")
		return
	}
	unitID, err := uuid.Parse(req.UnitID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid unit_id")
		return
	}

	svcReq := &time_entry.CreateTimeEntryRequest{
		OrgID:        orgID,
		UserID:       userID,
		ProjectID:    projectID,
		SubprojectID: subprojectID,
		WGID:         wgID,
		UnitID:       unitID,
		Hours:        req.Hours,
		Description:  req.Description,
		Date:         req.Date,
	}

	e, err := h.service.Create(ctx, svcReq)
	if err != nil {
		if err == time_entry.ErrPeriodLocked {
			api.RespondWithError(w, http.StatusBadRequest, "cannot create entry for locked period")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create time entry")
		return
	}

	h.service.CreateAuditLog(ctx, orgID, e.ID.String(), "time_entry", "created", "user", userID.String(), "", nil)

	api.RespondWithJSON(w, http.StatusCreated, e)
}

func (h *TimeEntryHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	entryIDStr := r.PathValue("id")

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var req UpdateTimeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	svcReq := &time_entry.UpdateTimeEntryRequest{}
	if req.ProjectID != nil {
		pid, err := uuid.Parse(*req.ProjectID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid project_id")
			return
		}
		svcReq.ProjectID = &pid
	}
	if req.SubprojectID != nil {
		sid, err := uuid.Parse(*req.SubprojectID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid subproject_id")
			return
		}
		svcReq.SubprojectID = &sid
	}
	if req.WGID != nil {
		wgid, err := uuid.Parse(*req.WGID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid wg_id")
			return
		}
		svcReq.WGID = &wgid
	}
	if req.UnitID != nil {
		uid, err := uuid.Parse(*req.UnitID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid unit_id")
			return
		}
		svcReq.UnitID = &uid
	}
	if req.Hours != nil {
		if *req.Hours > 24 {
			api.RespondWithError(w, http.StatusBadRequest, "hours cannot exceed 24")
			return
		}
		svcReq.Hours = req.Hours
	}
	if req.Description != nil {
		svcReq.Description = req.Description
	}
	if req.Date != nil {
		svcReq.Date = req.Date
	}

	e, err := h.service.Update(ctx, entryID, userID, svcReq)
	if err != nil {
		if err == time_entry.ErrTimeEntryNotFound {
			api.RespondWithError(w, http.StatusNotFound, "time entry not found")
			return
		}
		if err == time_entry.ErrEntryNotDraft {
			api.RespondWithError(w, http.StatusBadRequest, "can only update draft entries")
			return
		}
		if err == time_entry.ErrNotOwner {
			api.RespondWithError(w, http.StatusForbidden, "can only update own entries")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update time entry")
		return
	}

	h.service.CreateAuditLog(ctx, orgID, entryIDStr, "time_entry", "edited", "user", userID.String(), "", nil)

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *TimeEntryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	entryIDStr := r.PathValue("id")

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	err = h.service.Delete(ctx, entryID, userID)
	if err != nil {
		if err == time_entry.ErrTimeEntryNotFound {
			api.RespondWithError(w, http.StatusNotFound, "time entry not found")
			return
		}
		if err == time_entry.ErrEntryNotDraft {
			api.RespondWithError(w, http.StatusBadRequest, "can only delete draft entries")
			return
		}
		if err == time_entry.ErrNotOwner {
			api.RespondWithError(w, http.StatusForbidden, "can only delete own entries")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete time entry")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TimeEntryHandler) Submit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	entryIDStr := r.PathValue("id")

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	e, err := h.service.Submit(ctx, entryID, userID)
	if err != nil {
		if err == time_entry.ErrTimeEntryNotFound {
			api.RespondWithError(w, http.StatusNotFound, "time entry not found")
			return
		}
		if err == time_entry.ErrEntryNotDraft {
			api.RespondWithError(w, http.StatusBadRequest, "can only submit draft entries")
			return
		}
		if err == time_entry.ErrNotOwner {
			api.RespondWithError(w, http.StatusForbidden, "can only submit own entries")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to submit time entry")
		return
	}

	h.service.CreateAuditLog(ctx, orgID, entryIDStr, "time_entry", "submitted", "user", userID.String(), "", nil)

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *TimeEntryHandler) Approve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	entryIDStr := r.PathValue("id")

	if role != "wg_manager" && role != "admin" {
		api.RespondWithError(w, http.StatusForbidden, "only working group managers can approve entries")
		return
	}

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	e, err := h.service.Approve(ctx, entryID, userID, role)
	if err != nil {
		if err == time_entry.ErrTimeEntryNotFound {
			api.RespondWithError(w, http.StatusNotFound, "time entry not found")
			return
		}
		if err == time_entry.ErrEntryNotSubmitted {
			api.RespondWithError(w, http.StatusBadRequest, "can only approve submitted entries")
			return
		}
		if err == time_entry.ErrForbidden {
			api.RespondWithError(w, http.StatusForbidden, "only working group managers can approve entries")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to approve time entry")
		return
	}

	actorRole := "wg_manager"
	if role == "admin" {
		actorRole = "admin"
	}
	h.service.CreateAuditLog(ctx, orgID, entryIDStr, "time_entry", "approved", actorRole, userID.String(), "", nil)

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *TimeEntryHandler) Reject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	entryIDStr := r.PathValue("id")

	if role != "wg_manager" && role != "admin" {
		api.RespondWithError(w, http.StatusForbidden, "only working group managers can reject entries")
		return
	}

	var req RejectRequest
	json.NewDecoder(r.Body).Decode(&req)

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	e, err := h.service.Reject(ctx, entryID, userID, role, req.Reason)
	if err != nil {
		if err == time_entry.ErrTimeEntryNotFound {
			api.RespondWithError(w, http.StatusNotFound, "time entry not found")
			return
		}
		if err == time_entry.ErrEntryNotSubmitted {
			api.RespondWithError(w, http.StatusBadRequest, "can only reject submitted entries")
			return
		}
		if err == time_entry.ErrForbidden {
			api.RespondWithError(w, http.StatusForbidden, "only working group managers can reject entries")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to reject time entry")
		return
	}

	actorRole := "wg_manager"
	if role == "admin" {
		actorRole = "admin"
	}
	h.service.CreateAuditLog(ctx, orgID, entryIDStr, "time_entry", "rejected", actorRole, userID.String(), req.Reason, nil)

	api.RespondWithJSON(w, http.StatusOK, e)
}

func (h *TimeEntryHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	userID := middleware.GetUserID(ctx)

	if role != "wg_manager" && role != "admin" {
		api.RespondWithError(w, http.StatusForbidden, "only working group managers can view pending entries")
		return
	}

	entries, err := h.service.ListPending(ctx, orgID, role, userID.String())
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch pending entries")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, entries)
}
