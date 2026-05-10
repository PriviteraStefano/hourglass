package surrealdb

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/unit"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type UnitRepository struct {
	db *sdb.DB
}

func NewUnitRepository(db *sdb.DB) *UnitRepository {
	return &UnitRepository{db: db}
}

func (r *UnitRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]unit.Unit, error) {
	orgRecordID := uuidToRecordID("organizations", orgID)
	results, err := sdb.Query[[]SurrealUnit](ctx, r.db,
		`SELECT * FROM units WHERE org_id = $org_id ORDER BY hierarchy_level, name`,
		map[string]interface{}{"org_id": orgRecordID})
	if err != nil {
		return nil, wrapErr(err, "list units by org")
	}
	if results == nil || len(*results) == 0 {
		return []unit.Unit{}, nil
	}
	resultItems := (*results)[0].Result
	units := make([]unit.Unit, len(resultItems))
	for i, su := range resultItems {
		units[i] = *su.ToDomain()
	}
	return units, nil
}

func (r *UnitRepository) GetByID(ctx context.Context, id string) (*unit.Unit, error) {
	recordID := models.NewRecordID("units", id)
	result, err := sdb.Select[SurrealUnit](ctx, r.db, recordID)
	if err != nil {
		return nil, wrapErr(err, "get unit by id")
	}
	return result.ToDomain(), nil
}

func (r *UnitRepository) Create(ctx context.Context, u *unit.Unit) (*unit.Unit, error) {
	su := SurrealUnitFromDomain(u)
	created, err := sdb.Create[SurrealUnit](ctx, r.db, models.Table("units"), su)
	if err != nil {
		return nil, wrapErr(err, "create unit")
	}
	return created.ToDomain(), nil
}

func (r *UnitRepository) Update(ctx context.Context, u *unit.Unit) (*unit.Unit, error) {
	recordID := models.NewRecordID("units", u.ID)
	data := map[string]interface{}{
		"name":            u.Name,
		"hierarchy_level": u.HierarchyLevel,
		"updated_at":      u.UpdatedAt,
	}
	if u.Description != "" {
		data["description"] = u.Description
	}
	if u.Code != "" {
		data["code"] = u.Code
	}
	if u.ParentUnitID != "" {
		data["parent_unit_id"] = models.NewRecordID("units", u.ParentUnitID)
	} else {
		data["parent_unit_id"] = nil
	}
	result, err := sdb.Merge[SurrealUnit](ctx, r.db, recordID, data)
	if err != nil {
		return nil, wrapErr(err, "update unit")
	}
	return result.ToDomain(), nil
}

func (r *UnitRepository) Delete(ctx context.Context, id string) error {
	recordID := models.NewRecordID("units", id)
	_, err := sdb.Delete[SurrealUnit](ctx, r.db, recordID)
	return wrapErr(err, "delete unit")
}

func (r *UnitRepository) GetDescendants(ctx context.Context, id string) ([]unit.Unit, error) {
	unitRecordID := models.NewRecordID("units", id)
	results, err := sdb.Query[[]SurrealUnit](ctx, r.db,
		`SELECT * FROM units WHERE org_id = (SELECT VALUE org_id FROM units:$unit_id)[0] AND hierarchy_level > (SELECT VALUE hierarchy_level FROM units:$unit_id)[0]`,
		map[string]interface{}{"unit_id": unitRecordID})
	if err != nil {
		return nil, wrapErr(err, "get descendants")
	}
	if results == nil || len(*results) == 0 {
		return []unit.Unit{}, nil
	}
	resultItems := (*results)[0].Result
	units := make([]unit.Unit, len(resultItems))
	for i, su := range resultItems {
		units[i] = *su.ToDomain()
	}
	return units, nil
}

func (r *UnitRepository) HasMembers(ctx context.Context, id string) (bool, error) {
	unitRecordID := models.NewRecordID("units", id)
	results, err := sdb.Query[[]map[string]interface{}](ctx, r.db,
		`SELECT count() FROM unit_memberships WHERE unit_id = $unit_id GROUP ALL`,
		map[string]interface{}{"unit_id": unitRecordID})
	if err != nil {
		return false, wrapErr(err, "check unit members")
	}
	if results == nil || len(*results) == 0 {
		return false, nil
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		return false, nil
	}
	if count, ok := resultItems[0]["count"].(float64); ok && count > 0 {
		return true, nil
	}
	return false, nil
}

func (r *UnitRepository) ListMembers(ctx context.Context, unitID string) ([]unit.UnitMember, error) {
	unitRecordID := models.NewRecordID("units", unitID)
	results, err := sdb.Query[[]SurrealUnitMember](ctx, r.db,
		`SELECT um.*, u.name as user_name, u.email as user_email FROM unit_memberships um JOIN users u ON um.user_id = u.id WHERE um.unit_id = $unit_id ORDER BY um.created_at`,
		map[string]interface{}{"unit_id": unitRecordID})
	if err != nil {
		return nil, wrapErr(err, "list unit members")
	}
	if results == nil || len(*results) == 0 {
		return []unit.UnitMember{}, nil
	}
	resultItems := (*results)[0].Result
	members := make([]unit.UnitMember, len(resultItems))
	for i, sm := range resultItems {
		members[i] = *sm.ToDomain()
	}
	return members, nil
}

func (r *UnitRepository) AddMember(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error) {
	sm := SurrealUnitMemberFromDomain(m)
	created, err := sdb.Create[SurrealUnitMember](ctx, r.db, models.Table("unit_memberships"), sm)
	if err != nil {
		return nil, wrapErr(err, "add unit member")
	}
	return created.ToDomain(), nil
}

func (r *UnitRepository) RemoveMember(ctx context.Context, id string) error {
	recordID := models.NewRecordID("unit_memberships", id)
	_, err := sdb.Delete[SurrealUnitMember](ctx, r.db, recordID)
	return wrapErr(err, "remove unit member")
}

func (r *UnitRepository) GetMemberCountsByOrg(ctx context.Context, orgID uuid.UUID) (map[string]int, error) {
	orgRecordID := uuidToRecordID("organizations", orgID)
	results, err := sdb.Query[[]map[string]interface{}](ctx, r.db,
		`SELECT unit_id, count() as count FROM unit_memberships WHERE org_id = $org_id GROUP BY unit_id`,
		map[string]interface{}{"org_id": orgRecordID})
	if err != nil {
		return nil, wrapErr(err, "get member counts by org")
	}
	if results == nil || len(*results) == 0 {
		return map[string]int{}, nil
	}
	resultItems := (*results)[0].Result
	counts := make(map[string]int, len(resultItems))
	for _, item := range resultItems {
		unitIDStr := ""
		if unitID, ok := item["unit_id"]; ok {
			switch v := unitID.(type) {
			case string:
				unitIDStr = v
			case map[string]interface{}:
				if id, ok := v["id"]; ok {
					unitIDStr = fmt.Sprintf("%v", id)
				}
			default:
				unitIDStr = fmt.Sprintf("%v", unitID)
			}
		}
		count := 0
		if c, ok := item["count"].(float64); ok {
			count = int(c)
		}
		if unitIDStr != "" {
			counts[unitIDStr] = count
		}
	}
	return counts, nil
}
