package handlers

import (
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

type WorkingGroupHandler struct {
	db *sdb.DB
}

func NewWorkingGroupHandler(db *sdb.DB) *WorkingGroupHandler {
	return &WorkingGroupHandler{db: db}
}

func (h *WorkingGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	subprojectID := r.URL.Query().Get("subproject_id")

	query := `SELECT * FROM working_groups WHERE org_id = $org_id AND is_active = true`
	vars := map[string]interface{}{"org_id": orgID}

	if subprojectID != "" {
		query += " AND subproject_id = $subproject_id"
		vars["subproject_id"] = subprojectID
	}
	query += " ORDER BY name"

	results, err := sdb.Query[[]surrealdb.SurrealWorkingGroup](ctx, h.db, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch working groups")
		return
	}

	if results == nil || len(*results) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.WorkingGroup{})
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.WorkingGroup{})
		return
	}

	wgs := make([]models.WorkingGroup, len(resultItems))
	for i, wg := range resultItems {
		wgs[i] = surrealWGToWG(wg)
	}

	api.RespondWithJSON(w, http.StatusOK, wgs)
}

func (h *WorkingGroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgID := r.PathValue("id")

	recordID := sdkmodels.NewRecordID("working_groups", wgID)
	result, err := sdb.Select[surrealdb.SurrealWorkingGroup](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "working group not found")
		return
	}

	wg := surrealWGToWG(*result)
	api.RespondWithJSON(w, http.StatusOK, wg)
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

	wg := &surrealdb.SurrealWorkingGroup{
		OrgID:            sdkmodels.NewRecordID("organizations", req.OrgID),
		SubprojectID:     sdkmodels.NewRecordID("subprojects", req.SubprojectID),
		Name:             req.Name,
		Description:      req.Description,
		UnitIDs:          req.UnitIDs,
		EnforceUnitTuple: req.EnforceUnitTuple,
		ManagerID:        sdkmodels.NewRecordID("users", req.ManagerID),
		DelegateIDs:      req.DelegateIDs,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	created, err := sdb.Create[surrealdb.SurrealWorkingGroup](ctx, h.db, sdkmodels.Table("working_groups"), wg)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create working group")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, surrealWGToWG(*created))
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
	recordID := sdkmodels.NewRecordID("working_groups", wgID)

	data := map[string]interface{}{
		"updated_at": now,
	}
	if req.Name != "" {
		data["name"] = req.Name
	}
	if req.Description != "" {
		data["description"] = req.Description
	}
	if req.UnitIDs != nil {
		data["unit_ids"] = req.UnitIDs
	}
	if req.EnforceUnitTuple != nil {
		data["enforce_unit_tuple"] = *req.EnforceUnitTuple
	}
	if req.ManagerID != "" {
		data["manager_id"] = sdkmodels.NewRecordID("users", req.ManagerID)
	}
	if req.DelegateIDs != nil {
		data["delegate_ids"] = req.DelegateIDs
	}

	result, err := sdb.Merge[surrealdb.SurrealWorkingGroup](ctx, h.db, recordID, data)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update working group")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, surrealWGToWG(*result))
}

func (h *WorkingGroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgID := r.PathValue("id")

	checkResults, err := sdb.Query[[]map[string]interface{}](ctx, h.db,
		`SELECT count() FROM wg_members WHERE wg_id = $wg_id GROUP ALL`,
		map[string]interface{}{"wg_id": sdkmodels.NewRecordID("working_groups", wgID)})
	if err == nil && checkResults != nil && len(*checkResults) > 0 {
		resultItems := (*checkResults)[0].Result
		if len(resultItems) > 0 {
			if count, ok := resultItems[0]["count"].(float64); ok && count > 0 {
				api.RespondWithError(w, http.StatusBadRequest, "cannot delete working group with members")
				return
			}
		}
	}

	recordID := sdkmodels.NewRecordID("working_groups", wgID)
	_, err = sdb.Delete[surrealdb.SurrealWorkingGroup](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete working group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkingGroupHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wgID := r.PathValue("id")

	results, err := sdb.Query[[]surrealdb.SurrealWorkingGroupMember](ctx, h.db,
		`SELECT * FROM wg_members WHERE wg_id = $wg_id ORDER BY created_at`,
		map[string]interface{}{"wg_id": sdkmodels.NewRecordID("working_groups", wgID)})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch working group members")
		return
	}

	if results == nil || len(*results) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.WorkingGroupMember{})
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.WorkingGroupMember{})
		return
	}

	members := make([]models.WorkingGroupMember, len(resultItems))
	for i, m := range resultItems {
		members[i] = surrealWGMemberToMember(m)
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

	member := &surrealdb.SurrealWorkingGroupMember{
		WGID:                sdkmodels.NewRecordID("working_groups", req.WGID),
		UserID:              sdkmodels.NewRecordID("users", req.UserID),
		UnitID:              sdkmodels.NewRecordID("units", req.UnitID),
		Role:                req.Role,
		IsDefaultSubproject: req.IsDefaultSubproject,
		StartDate:           now,
		CreatedAt:           now,
	}

	created, err := sdb.Create[surrealdb.SurrealWorkingGroupMember](ctx, h.db, sdkmodels.Table("wg_members"), member)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, surrealWGMemberToMember(*created))
}

func (h *WorkingGroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	memberID := r.PathValue("member_id")

	recordID := sdkmodels.NewRecordID("wg_members", memberID)
	_, err := sdb.Delete[surrealdb.SurrealWorkingGroupMember](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func wgRecordIDToStr(id sdkmodels.RecordID) string {
	switch v := id.ID.(type) {
	case string:
		return v
	case sdkmodels.UUID:
		return uuid.UUID(v.UUID).String()
	default:
		return ""
	}
}

func surrealWGToWG(wg surrealdb.SurrealWorkingGroup) models.WorkingGroup {
	var delegateIDs []string
	if len(wg.DelegateIDs) > 0 {
		delegateIDs = wg.DelegateIDs
	}

	return models.WorkingGroup{
		ID:               wgRecordIDToStr(wg.ID),
		OrgID:            wgRecordIDToStr(wg.OrgID),
		SubprojectID:     wgRecordIDToStr(wg.SubprojectID),
		Name:             wg.Name,
		Description:      wg.Description,
		UnitIDs:          wg.UnitIDs,
		EnforceUnitTuple: wg.EnforceUnitTuple,
		ManagerID:        wgRecordIDToStr(wg.ManagerID),
		DelegateIDs:      delegateIDs,
		IsActive:         wg.IsActive,
		CreatedAt:        wg.CreatedAt,
		UpdatedAt:        wg.UpdatedAt,
	}
}

func surrealWGMemberToMember(m surrealdb.SurrealWorkingGroupMember) models.WorkingGroupMember {
	return models.WorkingGroupMember{
		ID:                  wgRecordIDToStr(m.ID),
		WGID:                wgRecordIDToStr(m.WGID),
		UserID:              wgRecordIDToStr(m.UserID),
		UnitID:              wgRecordIDToStr(m.UnitID),
		Role:                m.Role,
		IsDefaultSubproject: m.IsDefaultSubproject,
		StartDate:           m.StartDate,
		EndDate:             m.EndDate,
		CreatedAt:           m.CreatedAt,
	}
}
