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

type UnitHandler struct {
	db *sdb.DB
}

func NewUnitHandler(db *sdb.DB) *UnitHandler {
	return &UnitHandler{db: db}
}

func (h *UnitHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	results, err := sdb.Query[[]surrealdb.SurrealUnit](ctx, h.db,
		`SELECT * FROM units WHERE org_id = $org_id ORDER BY hierarchy_level, name`,
		map[string]interface{}{"org_id": orgID})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch units")
		return
	}

	if results == nil || len(*results) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.Unit{})
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.Unit{})
		return
	}

	units := make([]models.Unit, len(resultItems))
	for i, u := range resultItems {
		units[i] = surrealUnitToUnit(u)
	}

	api.RespondWithJSON(w, http.StatusOK, units)
}

func (h *UnitHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	recordID := sdkmodels.NewRecordID("units", unitID)
	result, err := sdb.Select[surrealdb.SurrealUnit](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "unit not found")
		return
	}

	unit := surrealUnitToUnit(*result)
	api.RespondWithJSON(w, http.StatusOK, unit)
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
		parentRecordID := sdkmodels.NewRecordID("units", *req.ParentUnitID)
		parentResult, err := sdb.Select[surrealdb.SurrealUnit](ctx, h.db, parentRecordID)
		if err == nil && parentResult != nil {
			hierarchyLevel = parentResult.HierarchyLevel + 1
		}
	}

	unit := &surrealdb.SurrealUnit{
		OrgID:          sdkmodels.NewRecordID("organizations", req.OrgID),
		Name:           req.Name,
		Description:    req.Description,
		HierarchyLevel: hierarchyLevel,
		Code:           req.Code,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if req.ParentUnitID != nil {
		unit.ParentUnitID = sdkmodels.NewRecordID("units", *req.ParentUnitID)
	}

	created, err := sdb.Create[surrealdb.SurrealUnit](ctx, h.db, sdkmodels.Table("units"), unit)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create unit")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, surrealUnitToUnit(*created))
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
	recordID := sdkmodels.NewRecordID("units", unitID)

	data := map[string]interface{}{
		"updated_at": now,
	}
	if req.Name != "" {
		data["name"] = req.Name
	}
	if req.Description != "" {
		data["description"] = req.Description
	}
	if req.Code != "" {
		data["code"] = req.Code
	}

	result, err := sdb.Merge[surrealdb.SurrealUnit](ctx, h.db, recordID, data)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update unit")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, surrealUnitToUnit(*result))
}

func (h *UnitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unitID := r.PathValue("id")

	checkResults, err := sdb.Query[[]map[string]interface{}](ctx, h.db,
		`SELECT count() FROM unit_memberships WHERE unit_id = $unit_id GROUP ALL`,
		map[string]interface{}{"unit_id": sdkmodels.NewRecordID("units", unitID)})
	if err == nil && checkResults != nil && len(*checkResults) > 0 {
		resultItems := (*checkResults)[0].Result
		if len(resultItems) > 0 {
			if count, ok := resultItems[0]["count"].(float64); ok && count > 0 {
				api.RespondWithError(w, http.StatusBadRequest, "cannot delete unit with members")
				return
			}
		}
	}

	recordID := sdkmodels.NewRecordID("units", unitID)
	_, err = sdb.Delete[surrealdb.SurrealUnit](ctx, h.db, recordID)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to delete unit")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UnitHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	results, err := sdb.Query[[]surrealdb.SurrealUnit](ctx, h.db,
		`SELECT * FROM units WHERE org_id = $org_id ORDER BY hierarchy_level, name`,
		map[string]interface{}{"org_id": orgID})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch units")
		return
	}

	if results == nil || len(*results) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.UnitTreeNode{})
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.UnitTreeNode{})
		return
	}

	units := make([]models.Unit, len(resultItems))
	for i, u := range resultItems {
		units[i] = surrealUnitToUnit(u)
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

	results, err := sdb.Query[[]surrealdb.SurrealUnit](ctx, h.db,
		`SELECT * FROM units WHERE org_id = (SELECT VALUE org_id FROM units:$unit_id)[0] AND hierarchy_level > (SELECT VALUE hierarchy_level FROM units:$unit_id)[0]`,
		map[string]interface{}{"unit_id": unitID})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch descendants")
		return
	}

	if results == nil || len(*results) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.Unit{})
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithJSON(w, http.StatusOK, []models.Unit{})
		return
	}

	units := make([]models.Unit, len(resultItems))
	for i, u := range resultItems {
		units[i] = surrealUnitToUnit(u)
	}

	api.RespondWithJSON(w, http.StatusOK, units)
}

func surrealUnitToUnit(u surrealdb.SurrealUnit) models.Unit {
	var parentUnitID *string
	if u.ParentUnitID.ID != nil {
		switch v := u.ParentUnitID.ID.(type) {
		case string:
			parentUnitID = &v
		case sdkmodels.UUID:
			id := uuid.UUID(v.UUID)
			s := id.String()
			parentUnitID = &s
		}
	}

	return models.Unit{
		ID:             recordIDToStr(u.ID),
		OrgID:          recordIDToStr(u.OrgID),
		Name:           u.Name,
		Description:    u.Description,
		ParentUnitID:   parentUnitID,
		HierarchyLevel: u.HierarchyLevel,
		Code:           u.Code,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func recordIDToStr(id sdkmodels.RecordID) string {
	switch v := id.ID.(type) {
	case string:
		return v
	case sdkmodels.UUID:
		return uuid.UUID(v.UUID).String()
	default:
		return ""
	}
}
