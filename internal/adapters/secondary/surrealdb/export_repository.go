package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
	sdb "github.com/surrealdb/surrealdb.go"
)

type ExportRepository struct {
	db *sdb.DB
}

func NewExportRepository(db *sdb.DB) *ExportRepository {
	return &ExportRepository{db: db}
}

func (r *ExportRepository) Timesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	return r.queryRows(ctx, `
		SELECT
			'time_entry' as entry_type,
			entry_date as date,
			(SELECT VALUE name FROM users WHERE id = user_id LIMIT 1)[0] as employee,
			(SELECT VALUE name FROM projects WHERE id = project_id LIMIT 1)[0] as project,
			(SELECT VALUE name FROM contracts WHERE id = (SELECT VALUE contract_id FROM projects WHERE id = project_id LIMIT 1)[0] LIMIT 1)[0] as contract,
			(SELECT VALUE name FROM customers WHERE id = (SELECT VALUE customer_id FROM contracts WHERE id = (SELECT VALUE contract_id FROM projects WHERE id = project_id LIMIT 1)[0] LIMIT 1)[0] LIMIT 1)[0] as customer,
			hours,
			NULL as amount,
			NULL as km_distance,
			NULL as type,
			description,
			status
		FROM time_entries
		WHERE org_id = $org_id AND entry_date >= $from AND entry_date <= $to AND is_deleted = false`+r.roleFilter("user_id", role),
		map[string]interface{}{
			"org_id": uuidToRecordID("organizations", orgID),
			"from":   from,
			"to":     to,
			"user_id": uuidToRecordID("users", userID),
		})
}

func (r *ExportRepository) Expenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	return r.queryRows(ctx, `
		SELECT
			'expense' as entry_type,
			expense_date as date,
			(SELECT VALUE name FROM users WHERE id = user_id LIMIT 1)[0] as employee,
			(SELECT VALUE name FROM projects WHERE id = project_id LIMIT 1)[0] as project,
			(SELECT VALUE name FROM contracts WHERE id = (SELECT VALUE contract_id FROM projects WHERE id = project_id LIMIT 1)[0] LIMIT 1)[0] as contract,
			(SELECT VALUE name FROM customers WHERE id = (SELECT VALUE customer_id FROM contracts WHERE id = (SELECT VALUE contract_id FROM projects WHERE id = project_id LIMIT 1)[0] LIMIT 1)[0] LIMIT 1)[0] as customer,
			NULL as hours,
			amount,
			km_distance,
			category as type,
			description,
			status
		FROM expenses
		WHERE org_id = $org_id AND expense_date >= $from AND expense_date <= $to AND is_deleted = false`+r.roleFilter("user_id", role),
		map[string]interface{}{
			"org_id": uuidToRecordID("organizations", orgID),
			"from":   from,
			"to":     to,
			"user_id": uuidToRecordID("users", userID),
		})
}

func (r *ExportRepository) queryRows(ctx context.Context, query string, vars map[string]interface{}) ([]ports.ExportRow, error) {
	results, err := sdb.Query[[]exportRow](ctx, r.db, query, vars)
	if err != nil {
		return nil, wrapErr(err, "export query")
	}
	if results == nil || len(*results) == 0 {
		return []ports.ExportRow{}, nil
	}
	rows := make([]ports.ExportRow, 0, len((*results)[0].Result))
	for _, row := range (*results)[0].Result {
		rows = append(rows, row.toPort())
	}
	return rows, nil
}

func (r *ExportRepository) roleFilter(field, role string) string {
	switch role {
	case string(models.RoleEmployee):
		return " AND " + field + " = $user_id"
	case string(models.RoleManager):
		return " AND (" + field + " = $user_id OR project_id IN (SELECT VALUE project_id FROM project_managers WHERE user_id = $user_id))"
	default:
		return ""
	}
}

type exportRow struct {
	EntryType   string     `json:"entry_type"`
	Date        time.Time  `json:"date"`
	Employee    string     `json:"employee"`
	Project     string     `json:"project"`
	Contract    string     `json:"contract"`
	Customer    string     `json:"customer"`
	Hours       *float64   `json:"hours"`
	Amount      *float64   `json:"amount"`
	KmDistance  *float64   `json:"km_distance"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
}

func (e exportRow) toPort() ports.ExportRow {
	return ports.ExportRow{
		EntryType:   e.EntryType,
		Date:        e.Date,
		Employee:    e.Employee,
		Project:     e.Project,
		Contract:    e.Contract,
		Customer:    e.Customer,
		Hours:       e.Hours,
		Amount:      e.Amount,
		KmDistance:  e.KmDistance,
		Type:        e.Type,
		Description: e.Description,
		Status:      e.Status,
	}
}
