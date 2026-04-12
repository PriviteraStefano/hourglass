package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type WorkingGroupHandler struct {
	sdb *db.SurrealDB
}

func NewWorkingGroupHandler(sdb *db.SurrealDB) *WorkingGroupHandler {
	return &WorkingGroupHandler{sdb: sdb}
}

func (h *WorkingGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	subprojectID := r.URL.Query().Get("subproject_id")

	query := `
		SELECT * FROM working_groups 
		WHERE org_id = $org_id 
		AND is_active = true
	`
	vars := map[string]interface{}{
		"org_id": orgID,
	}

	if subprojectID != "" {
		query += " AND subproject_id = $subproject_id"
		vars["subproject_id"] = subprojectID
	}

	query += " ORDER BY name"

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch working groups")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithJSON(w, http.StatusOK, []models.WorkingGroup{})
		return
	}

	var wgs []models.WorkingGroup
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &wgs); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse working groups")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, wgs)
}

func (h *WorkingGroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgID := r.PathValue("id")

	query := `SELECT * FROM $wg_id`
	vars := map[string]interface{}{
		"wg_id": wgID,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch working group")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "working group not found")
		return
	}

	var wgs []models.WorkingGroup
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &wgs); err != nil || len(wgs) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse working group")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, wgs[0])
}

func (h *WorkingGroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	var req models.CreateWorkingGroupRequest
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

	if req.ManagerID == "" {
		req.ManagerID = userID.String()
	}

	now := time.Now()

	query := `
		CREATE working_groups SET
			org_id = $org_id,
			subproject_id = $subproject_id,
			name = $name,
			description = $description,
			unit_ids = $unit_ids,
			enforce_unit_tuple = $enforce_unit_tuple,
			manager_id = $manager_id,
			delegate_ids = $delegate_ids,
			is_active = true,
			created_at = $now,
			updated_at = $now
	`
	vars := map[string]interface{}{
		"org_id":             req.OrgID,
		"subproject_id":      req.SubprojectID,
		"name":               req.Name,
		"description":        req.Description,
		"unit_ids":           req.UnitIDs,
		"enforce_unit_tuple": req.EnforceUnitTuple,
		"manager_id":         req.ManagerID,
		"delegate_ids":       req.DelegateIDs,
		"now":                now,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create working group")
		return
	}

	var wgs []models.WorkingGroup
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &wgs); err != nil || len(wgs) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse created working group")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, wgs[0])
}

func (h *WorkingGroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgID := r.PathValue("id")

	var req struct {
		Name             string   `json:"name,omitempty"`
		Description      string   `json:"description,omitempty"`
		UnitIDs          []string `json:"unit_ids,omitempty"`
		EnforceUnitTuple *bool    `json:"enforce_unit_tuple,omitempty"`
		ManagerID        string   `json:"manager_id,omitempty"`
		DelegateIDs      []string `json:"delegate_ids,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()

	query := `
		UPDATE $wg_id MERGE {
			name: $name,
			description: $description,
			unit_ids: $unit_ids,
			enforce_unit_tuple: $enforce_unit_tuple,
			manager_id: $manager_id,
			delegate_ids: $delegate_ids,
			updated_at: $now
		}
	`
	vars := map[string]interface{}{
		"wg_id":              wgID,
		"name":               req.Name,
		"description":        req.Description,
		"unit_ids":           req.UnitIDs,
		"enforce_unit_tuple": req.EnforceUnitTuple,
		"manager_id":         req.ManagerID,
		"delegate_ids":       req.DelegateIDs,
		"now":                now,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update working group")
		return
	}

	var wgs []models.WorkingGroup
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &wgs); err != nil || len(wgs) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse updated working group")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, wgs[0])
}

func (h *WorkingGroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgID := r.PathValue("id")

	checkQuery := `
		SELECT count() FROM wg_members WHERE wg_id = $wg_id GROUP ALL
	`
	checkVars := map[string]interface{}{"wg_id": wgID}
	checkResult, err := h.sdb.Query(ctx, checkQuery, checkVars)
	if err == nil && len(*checkResult) > 0 && (*checkResult)[0].Result != nil {
		var counts []map[string]interface{}
		checkBytes, _ := json.Marshal((*checkResult)[0].Result)
		if json.Unmarshal(checkBytes, &counts) == nil && len(counts) > 0 {
			if count, ok := counts[0]["count"].(float64); ok && count > 0 {
				api.RespondWithError(w, http.StatusBadRequest, "cannot delete working group with members")
				return
			}
		}
	}

	query := `DELETE $wg_id`
	vars := map[string]interface{}{"wg_id": wgID}

	_, err = h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete working group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkingGroupHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgID := r.PathValue("id")

	query := `
		SELECT * FROM wg_members 
		WHERE wg_id = $wg_id
		ORDER BY created_at
	`
	vars := map[string]interface{}{
		"wg_id": wgID,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch working group members")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithJSON(w, http.StatusOK, []models.WorkingGroupMember{})
		return
	}

	var members []models.WorkingGroupMember
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &members); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse members")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, members)
}

func (h *WorkingGroupHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.AddWGMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" || req.UnitID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "user_id and unit_id are required")
		return
	}

	now := time.Now()

	query := `
		CREATE wg_members SET
			wg_id = $wg_id,
			user_id = $user_id,
			unit_id = $unit_id,
			role = $role,
			is_default_subproject = $is_default_subproject,
			start_date = $now,
			created_at = $now
	`
	vars := map[string]interface{}{
		"wg_id":                 req.WGID,
		"user_id":               req.UserID,
		"unit_id":               req.UnitID,
		"role":                  req.Role,
		"is_default_subproject": req.IsDefaultSubproject,
		"now":                   now,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	var members []models.WorkingGroupMember
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &members); err != nil || len(members) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse created member")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, members[0])
}

func (h *WorkingGroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	memberID := r.PathValue("member_id")

	query := `DELETE $member_id`
	vars := map[string]interface{}{"member_id": memberID}

	_, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
