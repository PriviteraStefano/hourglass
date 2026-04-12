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

type UnitHandler struct {
	sdb *db.SurrealDB
}

func NewUnitHandler(sdb *db.SurrealDB) *UnitHandler {
	return &UnitHandler{sdb: sdb}
}

func (h *UnitHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	query := `
		SELECT * FROM units 
		WHERE org_id = $org_id 
		ORDER BY hierarchy_level, name
	`
	vars := map[string]interface{}{
		"org_id": orgID,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch units")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithJSON(w, http.StatusOK, []models.Unit{})
		return
	}

	var units []models.Unit
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &units); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse units")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, units)
}

func (h *UnitHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	query := `SELECT * FROM $unit_id`
	vars := map[string]interface{}{
		"unit_id": unitID,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch unit")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "unit not found")
		return
	}

	var units []models.Unit
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &units); err != nil || len(units) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse unit")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, units[0])
}

func (h *UnitHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.CreateUnitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		api.RespondWithError(w, http.StatusBadRequest, "name is required")
		return
	}

	now := time.Now()

	hierarchyLevel := 0
	if req.ParentUnitID != nil {
		parentQuery := `SELECT hierarchy_level FROM $parent_id`
		parentVars := map[string]interface{}{"parent_id": *req.ParentUnitID}
		parentResult, err := h.sdb.Query(ctx, parentQuery, parentVars)
		if err == nil && len(*parentResult) > 0 && (*parentResult)[0].Result != nil {
			var parents []map[string]interface{}
			parentBytes, _ := json.Marshal((*parentResult)[0].Result)
			if json.Unmarshal(parentBytes, &parents) == nil && len(parents) > 0 {
				if level, ok := parents[0]["hierarchy_level"].(float64); ok {
					hierarchyLevel = int(level) + 1
				}
			}
		}
	}

	query := `
		CREATE units SET
			org_id = $org_id,
			name = $name,
			description = $description,
			parent_unit_id = $parent_unit_id,
			hierarchy_level = $hierarchy_level,
			code = $code,
			created_at = $now,
			updated_at = $now
	`
	vars := map[string]interface{}{
		"org_id":          req.OrgID,
		"name":            req.Name,
		"description":     req.Description,
		"parent_unit_id":  req.ParentUnitID,
		"hierarchy_level": hierarchyLevel,
		"code":            req.Code,
		"now":             now,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create unit")
		return
	}

	var units []models.Unit
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &units); err != nil || len(units) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse created unit")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, units[0])
}

func (h *UnitHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	var req models.UpdateUnitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()

	query := `
		UPDATE $unit_id SET
			name = $name,
			description = $description,
			parent_unit_id = $parent_unit_id,
			code = $code,
			updated_at = $now
	`
	vars := map[string]interface{}{
		"unit_id":        unitID,
		"name":           req.Name,
		"description":    req.Description,
		"parent_unit_id": req.ParentUnitID,
		"code":           req.Code,
		"now":            now,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update unit")
		return
	}

	var units []models.Unit
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &units); err != nil || len(units) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse updated unit")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, units[0])
}

func (h *UnitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	checkQuery := `
		SELECT count() FROM unit_memberships WHERE unit_id = $unit_id GROUP ALL
	`
	checkVars := map[string]interface{}{"unit_id": unitID}
	checkResult, err := h.sdb.Query(ctx, checkQuery, checkVars)
	if err == nil && len(*checkResult) > 0 && (*checkResult)[0].Result != nil {
		var counts []map[string]interface{}
		checkBytes, _ := json.Marshal((*checkResult)[0].Result)
		if json.Unmarshal(checkBytes, &counts) == nil && len(counts) > 0 {
			if count, ok := counts[0]["count"].(float64); ok && count > 0 {
				api.RespondWithError(w, http.StatusBadRequest, "cannot delete unit with members")
				return
			}
		}
	}

	query := `DELETE $unit_id`
	vars := map[string]interface{}{"unit_id": unitID}

	_, err = h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete unit")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UnitHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	query := `
		SELECT * FROM units 
		WHERE org_id = $org_id 
		ORDER BY hierarchy_level, name
	`
	vars := map[string]interface{}{
		"org_id": orgID,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch units")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithJSON(w, http.StatusOK, []models.UnitTreeNode{})
		return
	}

	var units []models.Unit
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &units); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse units")
		return
	}

	tree := buildUnitTree(units, nil)

	api.RespondWithJSON(w, http.StatusOK, tree)
}

func buildUnitTree(units []models.Unit, parentID *string) []models.UnitTreeNode {
	var tree []models.UnitTreeNode

	for _, unit := range units {
		var unitParentID *string
		if unit.ParentUnitID != nil && *unit.ParentUnitID != "" {
			unitParentID = unit.ParentUnitID
		}

		matches := (parentID == nil && unitParentID == nil) ||
			(parentID != nil && unitParentID != nil && *parentID == *unitParentID)

		if matches {
			node := models.UnitTreeNode{
				Unit:     unit,
				Children: buildUnitTree(units, &unit.ID),
			}
			tree = append(tree, node)
		}
	}

	return tree
}

func (h *UnitHandler) GetDescendants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	query := `
		SELECT * FROM units 
		WHERE org_id = (SELECT VALUE org_id FROM $unit_id)[0]
		AND hierarchy_level > (SELECT VALUE hierarchy_level FROM $unit_id)[0]
	`
	vars := map[string]interface{}{
		"unit_id": unitID,
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch descendants")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithJSON(w, http.StatusOK, []models.Unit{})
		return
	}

	var units []models.Unit
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &units); err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse units")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, units)
}
