package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// ExportRepository implements ports.ExportRepository using a pgxpool.
type ExportRepository struct {
	pool *pgxpool.Pool
}

// NewExportRepository creates a new ExportRepository.
func NewExportRepository(pool *pgxpool.Pool) *ExportRepository {
	return &ExportRepository{pool: pool}
}

// roleFilter builds a SQL fragment for role-based user filtering.
// field is the user_id column expression (e.g., "te.user_id").
// Returns the SQL fragment and the total number of query parameters.
func roleFilter(field string, role string) (string, int) {
	switch role {
	case "employee":
		return fmt.Sprintf(" AND %s = $4", field), 4
	case "manager":
		return fmt.Sprintf(" AND (%s = $4 OR project_id IN (SELECT project_id FROM project_managers WHERE user_id = $4))", field), 4
	default:
		return "", 3
	}
}

// Timesheets returns time entry export rows for the given org and date range.
func (r *ExportRepository) Timesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	roleSQL, paramCount := roleFilter("te.user_id", role)

	query := `SELECT 'time_entry' AS entry_type, te.entry_date AS date,
		CONCAT(COALESCE(u.firstname, ''), ' ', COALESCE(u.lastname, '')) AS employee,
		COALESCE(p.name, '') AS project,
		COALESCE(c.name, '') AS contract,
		COALESCE(cu.name, '') AS customer,
		te.hours, NULL::decimal AS amount, NULL::decimal AS km_distance,
		''::varchar AS type, te.description, te.status
	FROM time_entries te
	LEFT JOIN users u ON u.id = te.user_id
	LEFT JOIN projects p ON p.id = te.project_id
	LEFT JOIN contracts c ON c.id = p.contract_id
	LEFT JOIN customers cu ON cu.id = c.customer_id
	WHERE te.org_id = $1 AND te.entry_date >= $2 AND te.entry_date <= $3 AND te.is_deleted = false` + roleSQL + `
	ORDER BY te.entry_date`

	var rows pgx.Rows
	var err error
	if paramCount == 4 {
		rows, err = r.pool.Query(ctx, query, orgID, from, to, userID)
	} else {
		rows, err = r.pool.Query(ctx, query, orgID, from, to)
	}
	if err != nil {
		return nil, fmt.Errorf("export timesheets: %w", err)
	}
	defer rows.Close()

	return scanExportRows(rows)
}

// Expenses returns expense export rows for the given org and date range.
func (r *ExportRepository) Expenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	roleSQL, paramCount := roleFilter("e.user_id", role)

	query := `SELECT 'expense' AS entry_type, e.expense_date AS date,
		CONCAT(COALESCE(u.firstname, ''), ' ', COALESCE(u.lastname, '')) AS employee,
		COALESCE(p.name, '') AS project,
		COALESCE(c.name, '') AS contract,
		COALESCE(cu.name, '') AS customer,
		NULL::decimal AS hours, e.amount, e.km_distance,
		COALESCE(e.category, '') AS type, e.description, e.status
	FROM expenses e
	LEFT JOIN users u ON u.id = e.user_id
	LEFT JOIN projects p ON p.id = e.project_id
	LEFT JOIN contracts c ON c.id = p.contract_id
	LEFT JOIN customers cu ON cu.id = c.customer_id
	WHERE e.org_id = $1 AND e.expense_date >= $2 AND e.expense_date <= $3 AND e.is_deleted = false` + roleSQL + `
	ORDER BY e.expense_date`

	var rows pgx.Rows
	var err error
	if paramCount == 4 {
		rows, err = r.pool.Query(ctx, query, orgID, from, to, userID)
	} else {
		rows, err = r.pool.Query(ctx, query, orgID, from, to)
	}
	if err != nil {
		return nil, fmt.Errorf("export expenses: %w", err)
	}
	defer rows.Close()

	return scanExportRows(rows)
}

// -- scan helpers ------------------------------------------------------------

// scanExportRows scans multiple export rows.
func scanExportRows(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]ports.ExportRow, error) {
	var results []ports.ExportRow
	for rows.Next() {
		r, err := scanExportRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan export row: %w", err)
		}
		results = append(results, *r)
	}
	if results == nil {
		results = []ports.ExportRow{}
	}
	return results, rows.Err()
}

// scanExportRow scans a single export row.
func scanExportRow(s interface {
	Scan(dest ...any) error
}) (*ports.ExportRow, error) {
	var r ports.ExportRow
	err := s.Scan(
		&r.EntryType,
		&r.Date,
		&r.Employee,
		&r.Project,
		&r.Contract,
		&r.Customer,
		&r.Hours,
		&r.Amount,
		&r.KmDistance,
		&r.Type,
		&r.Description,
		&r.Status,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
