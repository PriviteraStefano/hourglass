package surrealdb

import (
	"context"

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

func (r *UnitRepository) GetByID(ctx context.Context, id uuid.UUID) (*unit.Unit, error) {
	recordID := uuidToRecordID("units", id)
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
	recordID := uuidToRecordID("units", u.ID)
	data := map[string]interface{}{
		"name":       u.Name,
		"updated_at": u.UpdatedAt,
	}
	if u.Description != "" {
		data["description"] = u.Description
	}
	if u.Code != "" {
		data["code"] = u.Code
	}
	result, err := sdb.Merge[SurrealUnit](ctx, r.db, recordID, data)
	if err != nil {
		return nil, wrapErr(err, "update unit")
	}
	return result.ToDomain(), nil
}

func (r *UnitRepository) Delete(ctx context.Context, id uuid.UUID) error {
	recordID := uuidToRecordID("units", id)
	_, err := sdb.Delete[SurrealUnit](ctx, r.db, recordID)
	return wrapErr(err, "delete unit")
}

func (r *UnitRepository) GetDescendants(ctx context.Context, id uuid.UUID) ([]unit.Unit, error) {
	unitRecordID := uuidToRecordID("units", id)
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

func (r *UnitRepository) HasMembers(ctx context.Context, id uuid.UUID) (bool, error) {
	unitRecordID := uuidToRecordID("units", id)
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
