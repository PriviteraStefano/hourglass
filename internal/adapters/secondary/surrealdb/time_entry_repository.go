package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type TimeEntryRepository struct {
	db *sdb.DB
}

func NewTimeEntryRepository(db *sdb.DB) *TimeEntryRepository {
	return &TimeEntryRepository{db: db}
}

func (r *TimeEntryRepository) buildQuery(filters ports.ListFilters) (string, map[string]interface{}) {
	query := `SELECT * FROM time_entries WHERE org_id = $org_id AND is_deleted = $is_deleted`
	vars := map[string]interface{}{
		"org_id":     filters.OrgID,
		"is_deleted": filters.IsDeleted,
	}

	if filters.Date != "" {
		query += " AND entry_date = $date"
		vars["date"] = filters.Date
	}
	if filters.Month != "" && filters.Year != "" {
		query += " AND datetime::month(entry_date) = $month AND datetime::year(entry_date) = $year"
		vars["month"] = filters.Month
		vars["year"] = filters.Year
	}
	if filters.UserID != "" {
		if filters.Role == "employee" && filters.UserID != filters.RequestUserID {
			query += " AND user_id = $request_user_id"
			vars["request_user_id"] = filters.RequestUserID
		} else {
			query += " AND user_id = $filter_user_id"
			vars["filter_user_id"] = filters.UserID
		}
	}
	if filters.Status != "" {
		query += " AND status = $status"
		vars["status"] = filters.Status
	}
	if filters.WGID != "" {
		query += " AND wg_id = $wg_id"
		vars["wg_id"] = filters.WGID
	}
	if filters.ProjectID != "" {
		query += " AND project_id = $project_id"
		vars["project_id"] = filters.ProjectID
	}
	query += " ORDER BY entry_date DESC, created_at DESC"
	return query, vars
}

func (r *TimeEntryRepository) List(ctx context.Context, orgID uuid.UUID, filters ports.ListFilters) ([]time_entry.TimeEntry, error) {
	orgRecordID := uuidToRecordID("organizations", orgID)
	filters.OrgID = orgRecordID

	query, vars := r.buildQuery(filters)

	results, err := sdb.Query[[]SurrealTimeEntry](ctx, r.db, query, vars)
	if err != nil {
		return nil, wrapErr(err, "list time entries")
	}
	if results == nil || len(*results) == 0 {
		return []time_entry.TimeEntry{}, nil
	}
	resultItems := (*results)[0].Result
	entries := make([]time_entry.TimeEntry, len(resultItems))
	for i, ste := range resultItems {
		entries[i] = *ste.ToDomain()
	}
	return entries, nil
}

func (r *TimeEntryRepository) GetByID(ctx context.Context, id uuid.UUID) (*time_entry.TimeEntry, error) {
	recordID := uuidToRecordID("time_entries", id)
	result, err := sdb.Select[SurrealTimeEntry](ctx, r.db, recordID)
	if err != nil {
		return nil, wrapErr(err, "get time entry by id")
	}
	return result.ToDomain(), nil
}

func (r *TimeEntryRepository) Create(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error) {
	ste := SurrealTimeEntryFromDomain(e)
	created, err := sdb.Create[SurrealTimeEntry](ctx, r.db, models.Table("time_entries"), ste)
	if err != nil {
		return nil, wrapErr(err, "create time entry")
	}
	return created.ToDomain(), nil
}

func (r *TimeEntryRepository) Update(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error) {
	recordID := uuidToRecordID("time_entries", e.ID)
	data := map[string]interface{}{
		"updated_at": e.UpdatedAt,
		"status":     e.Status,
	}
	if e.ProjectID != uuid.Nil {
		data["project_id"] = uuidToRecordID("projects", e.ProjectID)
	}
	if e.SubprojectID != uuid.Nil {
		data["subproject_id"] = uuidToRecordID("subprojects", e.SubprojectID)
	}
	if e.WGID != uuid.Nil {
		data["wg_id"] = uuidToRecordID("working_groups", e.WGID)
	}
	if e.UnitID != uuid.Nil {
		data["unit_id"] = uuidToRecordID("units", e.UnitID)
	}
	if e.Hours > 0 {
		data["hours"] = e.Hours
	}
	if e.Description != "" {
		data["description"] = e.Description
	}
	data["entry_date"] = e.EntryDate

	result, err := sdb.Merge[SurrealTimeEntry](ctx, r.db, recordID, data)
	if err != nil {
		return nil, wrapErr(err, "update time entry")
	}
	return result.ToDomain(), nil
}

func (r *TimeEntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	recordID := uuidToRecordID("time_entries", id)
	now := time.Now()
	_, err := sdb.Merge[SurrealTimeEntry](ctx, r.db, recordID, map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	})
	return wrapErr(err, "delete time entry")
}

func (r *TimeEntryRepository) IsPeriodLocked(ctx context.Context, orgID, projectID uuid.UUID, entryDate string) (bool, error) {
	orgRecordID := uuidToRecordID("organizations", orgID)
	parsedDate, err := time.Parse("2006-01-02", entryDate)
	if err != nil {
		return false, err
	}

	results, err := sdb.Query[[]map[string]interface{}](ctx, r.db,
		`SELECT * FROM financial_cutoff_periods WHERE org_id = $org_id AND period_start <= $entry_date AND period_end >= $entry_date AND is_locked = true`,
		map[string]interface{}{
			"org_id":     orgRecordID,
			"entry_date": parsedDate,
		})
	if err != nil {
		return false, wrapErr(err, "check period lock")
	}
	if results != nil && len(*results) > 0 && len((*results)[0].Result) > 0 {
		return true, nil
	}

	if projectID != uuid.Nil {
		projectRecordID := uuidToRecordID("projects", projectID)
		projectResults, err := sdb.Query[[]map[string]interface{}](ctx, r.db,
			`SELECT * FROM financial_cutoff_periods WHERE project_id = $project_id AND period_start <= $entry_date AND period_end >= $entry_date AND is_locked = true`,
			map[string]interface{}{
				"project_id": projectRecordID,
				"entry_date": parsedDate,
			})
		if err != nil {
			return false, wrapErr(err, "check project period lock")
		}
		if projectResults != nil && len(*projectResults) > 0 && len((*projectResults)[0].Result) > 0 {
			return true, nil
		}
	}

	return false, nil
}

func (r *TimeEntryRepository) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]time_entry.TimeEntry, error) {
	orgRecordID := uuidToRecordID("organizations", orgID)
	query := `SELECT * FROM time_entries WHERE org_id = $org_id AND status = $status AND is_deleted = false`
	vars := map[string]interface{}{
		"org_id": orgRecordID,
		"status": time_entry.StatusSubmitted,
	}

	if role == "wg_manager" {
		wgQuery := `SELECT VALUE id FROM working_groups WHERE manager_id = $user_id OR array::contains(delegate_ids, $user_id)`
		wgResult, err := sdb.Query[[][]string](ctx, r.db, wgQuery, map[string]interface{}{"user_id": userID})
		if err == nil && wgResult != nil && len(*wgResult) > 0 && len((*wgResult)[0].Result) > 0 {
			query += " AND wg_id IN $wg_ids"
			vars["wg_ids"] = (*wgResult)[0].Result
		}
	}

	query += " ORDER BY created_at ASC"

	results, err := sdb.Query[[]SurrealTimeEntry](ctx, r.db, query, vars)
	if err != nil {
		return nil, wrapErr(err, "list pending entries")
	}
	if results == nil || len(*results) == 0 {
		return []time_entry.TimeEntry{}, nil
	}
	resultItems := (*results)[0].Result
	entries := make([]time_entry.TimeEntry, len(resultItems))
	for i, ste := range resultItems {
		entries[i] = *ste.ToDomain()
	}
	return entries, nil
}

type AuditLogRepository struct {
	db *sdb.DB
}

func NewAuditLogRepository(db *sdb.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *time_entry.AuditLog) error {
	sal := &SurrealAuditLog{
		ID:        uuidToRecordID("audit_logs", log.ID),
		OrgID:     uuidToRecordID("organizations", log.OrgID),
		EntryID:   log.EntryID,
		EntryType: log.EntryType,
		Action:    log.Action,
		ActorRole: log.ActorRole,
		ActorID:   uuidToRecordID("users", log.ActorID),
		Reason:    log.Reason,
		Changes:   log.Changes,
		Timestamp: log.Timestamp,
	}
	_, err := sdb.Create[SurrealAuditLog](ctx, r.db, models.Table("audit_logs"), sal)
	return wrapErr(err, "create audit log")
}
