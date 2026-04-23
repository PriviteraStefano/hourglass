package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/surrealdb"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"

	sdb "github.com/surrealdb/surrealdb.go"
	sdkmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

type SurrealTimeEntryHandler struct {
	db *sdb.DB
}

func NewSurrealTimeEntryHandler(db *sdb.DB) *SurrealTimeEntryHandler {
	return &SurrealTimeEntryHandler{db: db}
}

func (h *SurrealTimeEntryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)

	query := `SELECT * FROM time_entries WHERE org_id = $org_id AND is_deleted = false`
	vars := map[string]interface{}{"org_id": orgID}

	date := r.URL.Query().Get("date")
	if date != "" {
		query += " AND entry_date = $date"
		vars["date"] = date
	}

	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	if month != "" && year != "" {
		query += " AND datetime::month(entry_date) = $month AND datetime::year(entry_date) = $year"
		vars["month"] = month
		vars["year"] = year
	}

	filterUserID := r.URL.Query().Get("user_id")
	if filterUserID != "" {
		if role == "employee" && filterUserID != userID.String() {
			api.RespondWithError(w, http.StatusForbidden, "can only view own entries")
			return
		}
		query += " AND user_id = $filter_user_id"
		vars["filter_user_id"] = filterUserID
	} else if role == "employee" {
		query += " AND user_id = $user_id"
		vars["user_id"] = userID.String()
	}

	status := r.URL.Query().Get("status")
	if status != "" {
		query += " AND status = $status"
		vars["status"] = status
	}

	wgID := r.URL.Query().Get("wg_id")
	if wgID != "" {
		query += " AND wg_id = $wg_id"
		vars["wg_id"] = wgID
	}

	projectID := r.URL.Query().Get("project_id")
	if projectID != "" {
		query += " AND project_id = $project_id"
		vars["project_id"] = projectID
	}

	query += " ORDER BY entry_date DESC, created_at DESC"

	results, err := sdb.Query[[]surrealdb.SurrealTimeEntry](ctx, h.db, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entries")
		return
	}

	if results == nil || len(*results) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.SurrTimeEntry{})
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.SurrTimeEntry{})
		return
	}

	entries := make([]models.SurrTimeEntry, len(resultItems))
	for i, e := range resultItems {
		entries[i] = surrealTimeEntryToEntry(e)
	}

	api.RespondWithJSON(w, http.StatusOK, entries)
}

func (h *SurrealTimeEntryHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	entryID := r.PathValue("id")

	recordID := sdkmodels.NewRecordID("time_entries", entryID)
	result, err := sdb.Select[surrealdb.SurrealTimeEntry](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	entry := surrealTimeEntryToEntry(*result)

	if entry.OrgID != orgID.String() {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	if role == "employee" && entry.UserID != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only view own entries")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, entry)
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

func (h *SurrealTimeEntryHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	entryDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid date format")
		return
	}

	locked, err := h.isPeriodLocked(ctx, orgID, req.ProjectID, entryDate)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to check period lock status")
		return
	}
	if locked {
		api.RespondWithError(w, http.StatusBadRequest, "cannot create entry for locked period")
		return
	}

	now := time.Now()

	entry := &surrealdb.SurrealTimeEntry{
		OrgID:        sdkmodels.NewRecordID("organizations", orgID.String()),
		UserID:       sdkmodels.NewRecordID("users", userID.String()),
		ProjectID:    sdkmodels.NewRecordID("projects", req.ProjectID),
		SubprojectID: sdkmodels.NewRecordID("subprojects", req.SubprojectID),
		WGID:         sdkmodels.NewRecordID("working_groups", req.WGID),
		UnitID:       sdkmodels.NewRecordID("units", req.UnitID),
		Hours:        req.Hours,
		Description:  req.Description,
		EntryDate:    entryDate,
		Status:       models.SurrStatusDraft,
		IsDeleted:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := sdb.Create[surrealdb.SurrealTimeEntry](ctx, h.db, sdkmodels.Table("time_entries"), entry)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create time entry")
		return
	}

	entryIDStr := teRecordIDToStr(created.ID)
	go h.createAuditLog(ctx, orgID.String(), entryIDStr, "time_entry", "created", "user", userID.String(), "", nil)

	api.RespondWithJSON(w, http.StatusCreated, surrealTimeEntryToEntry(*created))
}

type UpdateTimeEntryRequest struct {
	ProjectID    string  `json:"project_id,omitempty"`
	SubprojectID string  `json:"subproject_id,omitempty"`
	WGID         string  `json:"wg_id,omitempty"`
	UnitID       string  `json:"unit_id,omitempty"`
	Hours        float64 `json:"hours,omitempty"`
	Description  string  `json:"description,omitempty"`
	Date         string  `json:"date,omitempty"`
}

func (h *SurrealTimeEntryHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	entryID := r.PathValue("id")

	recordID := sdkmodels.NewRecordID("time_entries", entryID)

	checkResult, err := sdb.Select[surrealdb.SurrealTimeEntry](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	status := checkResult.Status
	entryUserID := teRecordIDToStr(checkResult.UserID)

	if status != models.SurrStatusDraft {
		api.RespondWithError(w, http.StatusBadRequest, "can only update draft entries")
		return
	}

	if entryUserID != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only update own entries")
		return
	}

	var req UpdateTimeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()

	data := map[string]interface{}{
		"updated_at": now,
	}
	if req.ProjectID != "" {
		data["project_id"] = sdkmodels.NewRecordID("projects", req.ProjectID)
	}
	if req.SubprojectID != "" {
		data["subproject_id"] = sdkmodels.NewRecordID("subprojects", req.SubprojectID)
	}
	if req.WGID != "" {
		data["wg_id"] = sdkmodels.NewRecordID("working_groups", req.WGID)
	}
	if req.UnitID != "" {
		data["unit_id"] = sdkmodels.NewRecordID("units", req.UnitID)
	}
	if req.Hours > 0 {
		if req.Hours > 24 {
			api.RespondWithError(w, http.StatusBadRequest, "hours cannot exceed 24")
			return
		}
		data["hours"] = req.Hours
	}
	if req.Description != "" {
		data["description"] = req.Description
	}
	if req.Date != "" {
		entryDate, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid date format")
			return
		}
		data["entry_date"] = entryDate
	}

	result, err := sdb.Merge[surrealdb.SurrealTimeEntry](ctx, h.db, recordID, data)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update time entry")
		return
	}

	go h.createAuditLog(ctx, orgID.String(), entryID, "time_entry", "edited", "user", userID.String(), "", nil)

	api.RespondWithJSON(w, http.StatusOK, surrealTimeEntryToEntry(*result))
}

func (h *SurrealTimeEntryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	_ = middleware.GetOrganizationID(ctx)
	entryID := r.PathValue("id")

	recordID := sdkmodels.NewRecordID("time_entries", entryID)

	checkResult, err := sdb.Select[surrealdb.SurrealTimeEntry](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	status := checkResult.Status
	entryUserID := teRecordIDToStr(checkResult.UserID)

	if status != models.SurrStatusDraft {
		api.RespondWithError(w, http.StatusBadRequest, "can only delete draft entries")
		return
	}

	if entryUserID != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only delete own entries")
		return
	}

	now := time.Now()
	_, err = sdb.Merge[surrealdb.SurrealTimeEntry](ctx, h.db, recordID, map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete time entry")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SurrealTimeEntryHandler) Submit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	entryID := r.PathValue("id")

	recordID := sdkmodels.NewRecordID("time_entries", entryID)

	checkResult, err := sdb.Select[surrealdb.SurrealTimeEntry](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	status := checkResult.Status
	entryUserID := teRecordIDToStr(checkResult.UserID)

	if status != models.SurrStatusDraft {
		api.RespondWithError(w, http.StatusBadRequest, "can only submit draft entries")
		return
	}

	if entryUserID != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only submit own entries")
		return
	}

	now := time.Now()
	result, err := sdb.Merge[surrealdb.SurrealTimeEntry](ctx, h.db, recordID, map[string]interface{}{
		"status":     models.SurrStatusSubmitted,
		"updated_at": now,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to submit time entry")
		return
	}

	go h.createAuditLog(ctx, orgID.String(), entryID, "time_entry", "submitted", "user", userID.String(), "", nil)

	api.RespondWithJSON(w, http.StatusOK, surrealTimeEntryToEntry(*result))
}

func (h *SurrealTimeEntryHandler) Approve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	entryID := r.PathValue("id")

	if role != "wg_manager" && role != "admin" {
		api.RespondWithError(w, http.StatusForbidden, "only working group managers can approve entries")
		return
	}

	recordID := sdkmodels.NewRecordID("time_entries", entryID)

	checkResult, err := sdb.Select[surrealdb.SurrealTimeEntry](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	status := checkResult.Status

	if status != models.SurrStatusSubmitted {
		api.RespondWithError(w, http.StatusBadRequest, "can only approve submitted entries")
		return
	}

	now := time.Now()
	result, err := sdb.Merge[surrealdb.SurrealTimeEntry](ctx, h.db, recordID, map[string]interface{}{
		"status":     models.SurrStatusApproved,
		"updated_at": now,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to approve time entry")
		return
	}

	actorRole := models.ActorRoleWGManager
	if role == "admin" {
		actorRole = models.ActorRoleAdmin
	}

	go h.createAuditLog(ctx, orgID.String(), entryID, "time_entry", "approved", actorRole, userID.String(), "", nil)

	api.RespondWithJSON(w, http.StatusOK, surrealTimeEntryToEntry(*result))
}

type RejectRequest struct {
	Reason string `json:"reason"`
}

func (h *SurrealTimeEntryHandler) Reject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	entryID := r.PathValue("id")

	if role != "wg_manager" && role != "admin" {
		api.RespondWithError(w, http.StatusForbidden, "only working group managers can reject entries")
		return
	}

	var req RejectRequest
	json.NewDecoder(r.Body).Decode(&req)

	recordID := sdkmodels.NewRecordID("time_entries", entryID)

	checkResult, err := sdb.Select[surrealdb.SurrealTimeEntry](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	status := checkResult.Status

	if status != models.SurrStatusSubmitted {
		api.RespondWithError(w, http.StatusBadRequest, "can only reject submitted entries")
		return
	}

	now := time.Now()
	result, err := sdb.Merge[surrealdb.SurrealTimeEntry](ctx, h.db, recordID, map[string]interface{}{
		"status":     models.SurrStatusDraft,
		"updated_at": now,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to reject time entry")
		return
	}

	actorRole := models.ActorRoleWGManager
	if role == "admin" {
		actorRole = models.ActorRoleAdmin
	}

	go h.createAuditLog(ctx, orgID.String(), entryID, "time_entry", "rejected", actorRole, userID.String(), req.Reason, nil)

	api.RespondWithJSON(w, http.StatusOK, surrealTimeEntryToEntry(*result))
}

func (h *SurrealTimeEntryHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	userID := middleware.GetUserID(ctx)

	if role != "wg_manager" && role != "admin" {
		api.RespondWithError(w, http.StatusForbidden, "only working group managers can view pending entries")
		return
	}

	query := `SELECT * FROM time_entries WHERE org_id = $org_id AND status = 'submitted' AND is_deleted = false`
	vars := map[string]interface{}{"org_id": orgID}

	if role == "wg_manager" {
		wgQuery := `SELECT VALUE id FROM working_groups WHERE manager_id = $user_id OR array::contains(delegate_ids, $user_id)`
		wgResult, err := sdb.Query[[][]string](ctx, h.db, wgQuery, map[string]interface{}{"user_id": userID.String()})
		if err == nil && wgResult != nil && len(*wgResult) > 0 {
			resultItems := (*wgResult)[0].Result
			if len(resultItems) > 0 {
				query += " AND wg_id IN $wg_ids"
				vars["wg_ids"] = resultItems
			}
		}
	}

	query += " ORDER BY created_at ASC"

	results, err := sdb.Query[[]surrealdb.SurrealTimeEntry](ctx, h.db, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch pending entries")
		return
	}

	if results == nil || len(*results) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.SurrTimeEntry{})
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.SurrTimeEntry{})
		return
	}

	entries := make([]models.SurrTimeEntry, len(resultItems))
	for i, e := range resultItems {
		entries[i] = surrealTimeEntryToEntry(e)
	}

	api.RespondWithJSON(w, http.StatusOK, entries)
}

func (h *SurrealTimeEntryHandler) isPeriodLocked(ctx context.Context, orgID interface{}, projectID string, entryDate time.Time) (bool, error) {
	results, err := sdb.Query[[]map[string]interface{}](ctx, h.db,
		`SELECT * FROM financial_cutoff_periods WHERE org_id = $org_id AND period_start <= $entry_date AND period_end >= $entry_date AND is_locked = true`,
		map[string]interface{}{
			"org_id":     orgID,
			"entry_date": entryDate,
		})
	if err != nil {
		return false, err
	}

	if results != nil && len(*results) > 0 {
		resultItems := (*results)[0].Result
		if len(resultItems) > 0 {
			return true, nil
		}
	}

	if projectID != "" {
		projectResults, err := sdb.Query[[]map[string]interface{}](ctx, h.db,
			`SELECT * FROM financial_cutoff_periods WHERE project_id = $project_id AND period_start <= $entry_date AND period_end >= $entry_date AND is_locked = true`,
			map[string]interface{}{
				"project_id": projectID,
				"entry_date": entryDate,
			})
		if err != nil {
			return false, err
		}

		if projectResults != nil && len(*projectResults) > 0 {
			resultItems := (*projectResults)[0].Result
			if len(resultItems) > 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

func (h *SurrealTimeEntryHandler) createAuditLog(ctx context.Context, orgID, entryID, entryType, action, actorRole, actorID, reason string, changes map[string]interface{}) {
	now := time.Now()
	audit := &surrealdb.SurrealAuditLog{
		OrgID:     sdkmodels.NewRecordID("organizations", orgID),
		EntryID:   entryID,
		EntryType: entryType,
		Action:    action,
		ActorRole: actorRole,
		ActorID:   sdkmodels.NewRecordID("users", actorID),
		Reason:    reason,
		Changes:   changes,
		Timestamp: now,
	}
	sdb.Create[surrealdb.SurrealAuditLog](ctx, h.db, sdkmodels.Table("audit_logs"), audit)
}

func (h *SurrealTimeEntryHandler) canFinanceOverride(role string) bool {
	return role == "finance" || role == "admin"
}

func teRecordIDToStr(id sdkmodels.RecordID) string {
	switch v := id.ID.(type) {
	case string:
		return v
	case sdkmodels.UUID:
		return uuid.UUID(v.UUID).String()
	default:
		return ""
	}
}

func surrealTimeEntryToEntry(e surrealdb.SurrealTimeEntry) models.SurrTimeEntry {
	var createdFromEntryID *string
	if e.CreatedFromEntryID.ID != nil {
		s := teRecordIDToStr(e.CreatedFromEntryID)
		createdFromEntryID = &s
	}

	return models.SurrTimeEntry{
		ID:                 teRecordIDToStr(e.ID),
		OrgID:              teRecordIDToStr(e.OrgID),
		UserID:             teRecordIDToStr(e.UserID),
		ProjectID:          teRecordIDToStr(e.ProjectID),
		SubprojectID:       teRecordIDToStr(e.SubprojectID),
		WGID:               teRecordIDToStr(e.WGID),
		UnitID:             teRecordIDToStr(e.UnitID),
		Hours:              e.Hours,
		Description:        e.Description,
		EntryDate:          e.EntryDate,
		Status:             e.Status,
		IsDeleted:          e.IsDeleted,
		CreatedFromEntryID: createdFromEntryID,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}
