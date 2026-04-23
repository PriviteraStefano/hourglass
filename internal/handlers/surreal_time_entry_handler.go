package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type SurrealTimeEntryHandler struct {
	sdb *db.SurrealDB
}

func NewSurrealTimeEntryHandler(sdb *db.SurrealDB) *SurrealTimeEntryHandler {
	return &SurrealTimeEntryHandler{sdb: sdb}
}

func (h *SurrealTimeEntryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)

	query := `
		SELECT * FROM time_entries 
		WHERE org_id = $org_id 
		AND is_deleted = false
	`
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

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entries")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithJSON(w, http.StatusOK, []models.SurrTimeEntry{})
		return
	}

	var entries []models.SurrTimeEntry
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &entries); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse time entries")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, entries)
}

func (h *SurrealTimeEntryHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)
	entryID := r.PathValue("id")

	query := `SELECT * FROM $entry_id`
	vars := map[string]interface{}{"entry_id": entryID}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	var entries []models.SurrTimeEntry
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &entries); err != nil || len(entries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse time entry")
		return
	}

	entry := entries[0]

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

	query := `
		CREATE time_entries SET
			org_id = $org_id,
			user_id = $user_id,
			project_id = $project_id,
			subproject_id = $subproject_id,
			wg_id = $wg_id,
			unit_id = $unit_id,
			hours = $hours,
			description = $description,
			entry_date = $entry_date,
			status = 'draft',
			is_deleted = false,
			created_at = $now,
			updated_at = $now
	`
	vars := map[string]interface{}{
		"org_id":        orgID,
		"user_id":       userID.String(),
		"project_id":    req.ProjectID,
		"subproject_id": req.SubprojectID,
		"wg_id":         req.WGID,
		"unit_id":       req.UnitID,
		"hours":         req.Hours,
		"description":   req.Description,
		"entry_date":    entryDate,
		"now":           now,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create time entry")
		return
	}

	var entries []models.SurrTimeEntry
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &entries); err != nil || len(entries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse created time entry")
		return
	}

	entry := entries[0]

	auditQuery := `
		CREATE audit_logs SET
			org_id = $org_id,
			entry_id = $entry_id,
			entry_type = 'time_entry',
			action = 'created',
			actor_role = $actor_role,
			actor_id = $actor_id,
			timestamp = $now
	`
	auditVars := map[string]interface{}{
		"org_id":     orgID,
		"entry_id":   entry.ID,
		"actor_role": "user",
		"actor_id":   userID.String(),
		"now":        now,
	}
	h.sdb.Query(ctx, auditQuery, auditVars)

	api.RespondWithJSON(w, http.StatusCreated, entry)
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

	var status string
	var entryUserID string
	checkQuery := `SELECT status, user_id FROM $entry_id`
	checkVars := map[string]interface{}{"entry_id": entryID}
	checkResult, err := h.sdb.Query(ctx, checkQuery, checkVars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if len(*checkResult) == 0 || (*checkResult)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	var entries []map[string]interface{}
	checkBytes, _ := json.Marshal((*checkResult)[0].Result)
	if err := json.Unmarshal(checkBytes, &entries); err != nil || len(entries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse time entry")
		return
	}

	status, _ = entries[0]["status"].(string)
	entryUserID, _ = entries[0]["user_id"].(string)

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

	mergeFields := make(map[string]interface{})
	if req.ProjectID != "" {
		mergeFields["project_id"] = req.ProjectID
	}
	if req.SubprojectID != "" {
		mergeFields["subproject_id"] = req.SubprojectID
	}
	if req.WGID != "" {
		mergeFields["wg_id"] = req.WGID
	}
	if req.UnitID != "" {
		mergeFields["unit_id"] = req.UnitID
	}
	if req.Hours > 0 {
		if req.Hours > 24 {
			api.RespondWithError(w, http.StatusBadRequest, "hours cannot exceed 24")
			return
		}
		mergeFields["hours"] = req.Hours
	}
	if req.Description != "" {
		mergeFields["description"] = req.Description
	}
	if req.Date != "" {
		entryDate, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid date format")
			return
		}
		mergeFields["entry_date"] = entryDate
	}
	mergeFields["updated_at"] = now

	query := `UPDATE $entry_id MERGE $fields`
	vars := map[string]interface{}{
		"entry_id": entryID,
		"fields":   mergeFields,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update time entry")
		return
	}

	var updatedEntries []models.SurrTimeEntry
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &updatedEntries); err != nil || len(updatedEntries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse updated time entry")
		return
	}

	auditQuery := `
		CREATE audit_logs SET
			org_id = $org_id,
			entry_id = $entry_id,
			entry_type = 'time_entry',
			action = 'edited',
			actor_role = $actor_role,
			actor_id = $actor_id,
			timestamp = $now
	`
	auditVars := map[string]interface{}{
		"org_id":     orgID,
		"entry_id":   entryID,
		"actor_role": "user",
		"actor_id":   userID.String(),
		"now":        now,
	}
	h.sdb.Query(ctx, auditQuery, auditVars)

	api.RespondWithJSON(w, http.StatusOK, updatedEntries[0])
}

func (h *SurrealTimeEntryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	_ = middleware.GetOrganizationID(ctx)
	entryID := r.PathValue("id")

	checkQuery := `SELECT status, user_id FROM $entry_id`
	checkVars := map[string]interface{}{"entry_id": entryID}
	checkResult, err := h.sdb.Query(ctx, checkQuery, checkVars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if len(*checkResult) == 0 || (*checkResult)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	var entries []map[string]interface{}
	checkBytes, _ := json.Marshal((*checkResult)[0].Result)
	if err := json.Unmarshal(checkBytes, &entries); err != nil || len(entries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse time entry")
		return
	}

	status, _ := entries[0]["status"].(string)
	entryUserID, _ := entries[0]["user_id"].(string)

	if status != models.SurrStatusDraft {
		api.RespondWithError(w, http.StatusBadRequest, "can only delete draft entries")
		return
	}

	if entryUserID != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only delete own entries")
		return
	}

	now := time.Now()

	query := `UPDATE $entry_id SET is_deleted = true, updated_at = $now`
	vars := map[string]interface{}{
		"entry_id": entryID,
		"now":      now,
	}

	_, err = h.sdb.Query(ctx, query, vars)
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

	checkQuery := `SELECT status, user_id FROM $entry_id`
	checkVars := map[string]interface{}{"entry_id": entryID}
	checkResult, err := h.sdb.Query(ctx, checkQuery, checkVars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if len(*checkResult) == 0 || (*checkResult)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	var entries []map[string]interface{}
	checkBytes, _ := json.Marshal((*checkResult)[0].Result)
	if err := json.Unmarshal(checkBytes, &entries); err != nil || len(entries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse time entry")
		return
	}

	status, _ := entries[0]["status"].(string)
	entryUserID, _ := entries[0]["user_id"].(string)

	if status != models.SurrStatusDraft {
		api.RespondWithError(w, http.StatusBadRequest, "can only submit draft entries")
		return
	}

	if entryUserID != userID.String() {
		api.RespondWithError(w, http.StatusForbidden, "can only submit own entries")
		return
	}

	now := time.Now()

	query := `UPDATE $entry_id SET status = 'submitted', updated_at = $now`
	vars := map[string]interface{}{
		"entry_id": entryID,
		"now":      now,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to submit time entry")
		return
	}

	var updatedEntries []models.SurrTimeEntry
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &updatedEntries); err != nil || len(updatedEntries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse submitted time entry")
		return
	}

	auditQuery := `
		CREATE audit_logs SET
			org_id = $org_id,
			entry_id = $entry_id,
			entry_type = 'time_entry',
			action = 'submitted',
			actor_role = 'user',
			actor_id = $actor_id,
			timestamp = $now
	`
	auditVars := map[string]interface{}{
		"org_id":   orgID,
		"entry_id": entryID,
		"actor_id": userID.String(),
		"now":      now,
	}
	h.sdb.Query(ctx, auditQuery, auditVars)

	api.RespondWithJSON(w, http.StatusOK, updatedEntries[0])
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

	checkQuery := `SELECT status FROM $entry_id`
	checkVars := map[string]interface{}{"entry_id": entryID}
	checkResult, err := h.sdb.Query(ctx, checkQuery, checkVars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if len(*checkResult) == 0 || (*checkResult)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	var entries []map[string]interface{}
	checkBytes, _ := json.Marshal((*checkResult)[0].Result)
	if err := json.Unmarshal(checkBytes, &entries); err != nil || len(entries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse time entry")
		return
	}

	status, _ := entries[0]["status"].(string)

	if status != models.SurrStatusSubmitted {
		api.RespondWithError(w, http.StatusBadRequest, "can only approve submitted entries")
		return
	}

	now := time.Now()

	query := `UPDATE $entry_id SET status = 'approved', updated_at = $now`
	vars := map[string]interface{}{
		"entry_id": entryID,
		"now":      now,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to approve time entry")
		return
	}

	var updatedEntries []models.SurrTimeEntry
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &updatedEntries); err != nil || len(updatedEntries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse approved time entry")
		return
	}

	actorRole := models.ActorRoleWGManager
	if role == "admin" {
		actorRole = models.ActorRoleAdmin
	}

	auditQuery := `
		CREATE audit_logs SET
			org_id = $org_id,
			entry_id = $entry_id,
			entry_type = 'time_entry',
			action = 'approved',
			actor_role = $actor_role,
			actor_id = $actor_id,
			timestamp = $now
	`
	auditVars := map[string]interface{}{
		"org_id":     orgID,
		"entry_id":   entryID,
		"actor_role": actorRole,
		"actor_id":   userID.String(),
		"now":        now,
	}
	h.sdb.Query(ctx, auditQuery, auditVars)

	api.RespondWithJSON(w, http.StatusOK, updatedEntries[0])
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

	checkQuery := `SELECT status FROM $entry_id`
	checkVars := map[string]interface{}{"entry_id": entryID}
	checkResult, err := h.sdb.Query(ctx, checkQuery, checkVars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch time entry")
		return
	}

	if len(*checkResult) == 0 || (*checkResult)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "time entry not found")
		return
	}

	var entries []map[string]interface{}
	checkBytes, _ := json.Marshal((*checkResult)[0].Result)
	if err := json.Unmarshal(checkBytes, &entries); err != nil || len(entries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse time entry")
		return
	}

	status, _ := entries[0]["status"].(string)

	if status != models.SurrStatusSubmitted {
		api.RespondWithError(w, http.StatusBadRequest, "can only reject submitted entries")
		return
	}

	now := time.Now()

	query := `UPDATE $entry_id SET status = 'draft', updated_at = $now`
	vars := map[string]interface{}{
		"entry_id": entryID,
		"now":      now,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to reject time entry")
		return
	}

	var updatedEntries []models.SurrTimeEntry
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &updatedEntries); err != nil || len(updatedEntries) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse rejected time entry")
		return
	}

	actorRole := models.ActorRoleWGManager
	if role == "admin" {
		actorRole = models.ActorRoleAdmin
	}

	auditQuery := `
		CREATE audit_logs SET
			org_id = $org_id,
			entry_id = $entry_id,
			entry_type = 'time_entry',
			action = 'rejected',
			actor_role = $actor_role,
			actor_id = $actor_id,
			reason = $reason,
			timestamp = $now
	`
	auditVars := map[string]interface{}{
		"org_id":     orgID,
		"entry_id":   entryID,
		"actor_role": actorRole,
		"actor_id":   userID.String(),
		"reason":     req.Reason,
		"now":        now,
	}
	h.sdb.Query(ctx, auditQuery, auditVars)

	api.RespondWithJSON(w, http.StatusOK, updatedEntries[0])
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

	query := `
		SELECT * FROM time_entries 
		WHERE org_id = $org_id 
		AND status = 'submitted'
		AND is_deleted = false
	`
	vars := map[string]interface{}{"org_id": orgID}

	if role == "wg_manager" {
		wgQuery := `
			SELECT VALUE id FROM working_groups 
			WHERE manager_id = $user_id OR array::contains(delegate_ids, $user_id)
		`
		wgVars := map[string]interface{}{"user_id": userID.String()}
		wgResult, err := h.sdb.Query(ctx, wgQuery, wgVars)
		if err == nil && len(*wgResult) > 0 && (*wgResult)[0].Result != nil {
			var wgIDs []string
			wgBytes, _ := json.Marshal((*wgResult)[0].Result)
			if json.Unmarshal(wgBytes, &wgIDs) == nil && len(wgIDs) > 0 {
				query += " AND wg_id IN $wg_ids"
				vars["wg_ids"] = wgIDs
			}
		}
	}

	query += " ORDER BY created_at ASC"

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch pending entries")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithJSON(w, http.StatusOK, []models.SurrTimeEntry{})
		return
	}

	var entries []models.SurrTimeEntry
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &entries); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse pending entries")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, entries)
}

func (h *SurrealTimeEntryHandler) isPeriodLocked(ctx context.Context, orgID interface{}, projectID string, entryDate time.Time) (bool, error) {
	query := `
		SELECT * FROM financial_cutoff_periods 
		WHERE org_id = $org_id 
		AND period_start <= $entry_date 
		AND period_end >= $entry_date
		AND is_locked = true
	`
	vars := map[string]interface{}{
		"org_id":     orgID,
		"entry_date": entryDate,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		return false, err
	}

	if len(*results) > 0 && (*results)[0].Result != nil {
		var periods []map[string]interface{}
		resultBytes, _ := json.Marshal((*results)[0].Result)
		if json.Unmarshal(resultBytes, &periods) == nil && len(periods) > 0 {
			return true, nil
		}
	}

	if projectID != "" {
		projectQuery := `
			SELECT * FROM financial_cutoff_periods 
			WHERE project_id = $project_id 
			AND period_start <= $entry_date 
			AND period_end >= $entry_date
			AND is_locked = true
		`
		projectVars := map[string]interface{}{
			"project_id": projectID,
			"entry_date": entryDate,
		}

		projectResults, err := h.sdb.Query(ctx, projectQuery, projectVars)
		if err != nil {
			return false, err
		}

		if len(*projectResults) > 0 && (*projectResults)[0].Result != nil {
			var periods []map[string]interface{}
			resultBytes, _ := json.Marshal((*projectResults)[0].Result)
			if json.Unmarshal(resultBytes, &periods) == nil && len(periods) > 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

func (h *SurrealTimeEntryHandler) canFinanceOverride(role string) bool {
	return role == "finance" || role == "admin"
}
