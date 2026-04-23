package surrealdb

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/working_group"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type WorkingGroupRepository struct {
	db *sdb.DB
}

func NewWorkingGroupRepository(db *sdb.DB) *WorkingGroupRepository {
	return &WorkingGroupRepository{db: db}
}

func (r *WorkingGroupRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, subprojectID *uuid.UUID) ([]working_group.WorkingGroup, error) {
	orgRecordID := uuidToRecordID("organizations", orgID)
	query := `SELECT * FROM working_groups WHERE org_id = $org_id AND is_active = true`
	vars := map[string]interface{}{"org_id": orgRecordID}

	if subprojectID != nil {
		query += " AND subproject_id = $subproject_id"
		vars["subproject_id"] = uuidToRecordID("subprojects", *subprojectID)
	}
	query += " ORDER BY name"

	results, err := sdb.Query[[]SurrealWorkingGroup](ctx, r.db, query, vars)
	if err != nil {
		return nil, wrapErr(err, "list working groups by org")
	}
	if results == nil || len(*results) == 0 {
		return []working_group.WorkingGroup{}, nil
	}
	resultItems := (*results)[0].Result
	wgs := make([]working_group.WorkingGroup, len(resultItems))
	for i, swg := range resultItems {
		wgs[i] = *swg.ToDomain()
	}
	return wgs, nil
}

func (r *WorkingGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*working_group.WorkingGroup, error) {
	recordID := uuidToRecordID("working_groups", id)
	result, err := sdb.Select[SurrealWorkingGroup](ctx, r.db, recordID)
	if err != nil {
		return nil, wrapErr(err, "get working group by id")
	}
	return result.ToDomain(), nil
}

func (r *WorkingGroupRepository) Create(ctx context.Context, wg *working_group.WorkingGroup) (*working_group.WorkingGroup, error) {
	swg := SurrealWorkingGroupFromDomain(wg)
	created, err := sdb.Create[SurrealWorkingGroup](ctx, r.db, models.Table("working_groups"), swg)
	if err != nil {
		return nil, wrapErr(err, "create working group")
	}
	return created.ToDomain(), nil
}

func (r *WorkingGroupRepository) Update(ctx context.Context, wg *working_group.WorkingGroup) (*working_group.WorkingGroup, error) {
	recordID := uuidToRecordID("working_groups", wg.ID)
	data := map[string]interface{}{
		"name":       wg.Name,
		"updated_at": wg.UpdatedAt,
	}
	if wg.Description != "" {
		data["description"] = wg.Description
	}
	if len(wg.UnitIDs) > 0 {
		data["unit_ids"] = wg.UnitIDs
	}
	data["enforce_unit_tuple"] = wg.EnforceUnitTuple
	data["manager_id"] = uuidToRecordID("users", wg.ManagerID)
	if len(wg.DelegateIDs) > 0 {
		data["delegate_ids"] = wg.DelegateIDs
	}
	result, err := sdb.Merge[SurrealWorkingGroup](ctx, r.db, recordID, data)
	if err != nil {
		return nil, wrapErr(err, "update working group")
	}
	return result.ToDomain(), nil
}

func (r *WorkingGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	recordID := uuidToRecordID("working_groups", id)
	_, err := sdb.Delete[SurrealWorkingGroup](ctx, r.db, recordID)
	return wrapErr(err, "delete working group")
}

func (r *WorkingGroupRepository) HasMembers(ctx context.Context, id uuid.UUID) (bool, error) {
	wgRecordID := uuidToRecordID("working_groups", id)
	results, err := sdb.Query[[]map[string]interface{}](ctx, r.db,
		`SELECT count() FROM wg_members WHERE wg_id = $wg_id GROUP ALL`,
		map[string]interface{}{"wg_id": wgRecordID})
	if err != nil {
		return false, wrapErr(err, "check wg members")
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

func (r *WorkingGroupRepository) ListMembers(ctx context.Context, wgID uuid.UUID) ([]working_group.WorkingGroupMember, error) {
	wgRecordID := uuidToRecordID("working_groups", wgID)
	results, err := sdb.Query[[]SurrealWorkingGroupMember](ctx, r.db,
		`SELECT * FROM wg_members WHERE wg_id = $wg_id ORDER BY created_at`,
		map[string]interface{}{"wg_id": wgRecordID})
	if err != nil {
		return nil, wrapErr(err, "list wg members")
	}
	if results == nil || len(*results) == 0 {
		return []working_group.WorkingGroupMember{}, nil
	}
	resultItems := (*results)[0].Result
	members := make([]working_group.WorkingGroupMember, len(resultItems))
	for i, sm := range resultItems {
		members[i] = *sm.ToDomain()
	}
	return members, nil
}

func (r *WorkingGroupRepository) AddMember(ctx context.Context, m *working_group.WorkingGroupMember) (*working_group.WorkingGroupMember, error) {
	sm := SurrealWorkingGroupMemberFromDomain(m)
	created, err := sdb.Create[SurrealWorkingGroupMember](ctx, r.db, models.Table("wg_members"), sm)
	if err != nil {
		return nil, wrapErr(err, "add wg member")
	}
	return created.ToDomain(), nil
}

func (r *WorkingGroupRepository) RemoveMember(ctx context.Context, id uuid.UUID) error {
	recordID := uuidToRecordID("wg_members", id)
	_, err := sdb.Delete[SurrealWorkingGroupMember](ctx, r.db, recordID)
	return wrapErr(err, "remove wg member")
}
