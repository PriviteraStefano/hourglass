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
// Manager scope resolves through the activity → working-group chain
// (ADR-BE-014 R-1): the user is the WG manager or delegate.
// Returns the SQL fragment and the total number of query parameters.
func roleFilter(field string, role string) (string, int) {
	switch role {
	case "employee":
		return fmt.Sprintf(" AND %s = $4", field), 4
	case "manager":
		return fmt.Sprintf(" AND (%s = $4 OR activity_id IN (SELECT activity_id FROM working_groups WHERE manager_id = $4 OR $4 = ANY(delegate_ids)))", field), 4
	default:
		return "", 3
	}
}

// commercialChainCTE walks each entry's activity upward until a contract is
// found (ADR-P-007 D-3 — commercial context derived, never stored). The CTE
// yields one row per (entry, ancestor) until the contract-bearing ancestor
// (or root); the final select picks the deepest row per entry, which carries
// the entry's activity name and the resolved contract.
const commercialChainCTE = `commercial AS (
	SELECT te.id AS entry_id, a.id AS activity_id, a.parent_id, a.contract_id, a.name AS activity_name, 0 AS depth
	FROM time_entries te
	JOIN activities a ON a.id = te.activity_id
	UNION ALL
	SELECT c.entry_id, c.activity_id, a.parent_id, a.contract_id, c.activity_name, c.depth + 1
	FROM commercial c
	JOIN activities a ON a.id = c.parent_id
	WHERE c.contract_id IS NULL
),
commercial_resolved AS (
	SELECT DISTINCT ON (entry_id) entry_id, activity_id, activity_name, contract_id
	FROM commercial
	ORDER BY entry_id, depth DESC
)`

// commercialChainCTEExpenses is the expense variant of commercialChainCTE.
const commercialChainCTEExpenses = `commercial AS (
	SELECT e.id AS entry_id, a.id AS activity_id, a.parent_id, a.contract_id, a.name AS activity_name, 0 AS depth
	FROM expenses e
	JOIN activities a ON a.id = e.activity_id
	UNION ALL
	SELECT c.entry_id, c.activity_id, a.parent_id, a.contract_id, c.activity_name, c.depth + 1
	FROM commercial c
	JOIN activities a ON a.id = c.parent_id
	WHERE c.contract_id IS NULL
),
commercial_resolved AS (
	SELECT DISTINCT ON (entry_id) entry_id, activity_id, activity_name, contract_id
	FROM commercial
	ORDER BY entry_id, depth DESC
)`

// Timesheets returns time entry export rows for the given org and date range.
func (r *ExportRepository) Timesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	roleSQL, paramCount := roleFilter("te.user_id", role)

	query := `WITH RECURSIVE ` + commercialChainCTE + `
	SELECT 'time_entry' AS entry_type, te.entry_date AS date,
		CONCAT(COALESCE(u.firstname, ''), ' ', COALESCE(u.lastname, '')) AS employee,
		COALESCE(cr.activity_name, '') AS project,
		COALESCE(c.name, '') AS contract,
		COALESCE(cu.name, '') AS customer,
		te.hours, NULL::decimal AS amount, NULL::decimal AS km_distance,
		''::varchar AS type, te.description, te.status
	FROM time_entries te
	LEFT JOIN users u ON u.id = te.user_id
	LEFT JOIN commercial_resolved cr ON cr.entry_id = te.id
	LEFT JOIN contracts c ON c.id = cr.contract_id
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

	query := `WITH RECURSIVE ` + commercialChainCTEExpenses + `
	SELECT 'expense' AS entry_type, e.expense_date AS date,
		CONCAT(COALESCE(u.firstname, ''), ' ', COALESCE(u.lastname, '')) AS employee,
		COALESCE(cr.activity_name, '') AS project,
		COALESCE(c.name, '') AS contract,
		COALESCE(cu.name, '') AS customer,
		NULL::decimal AS hours, e.amount, e.km_distance,
		COALESCE(e.category, '') AS type, e.description, e.status
	FROM expenses e
	LEFT JOIN users u ON u.id = e.user_id
	LEFT JOIN commercial_resolved cr ON cr.entry_id = e.id
	LEFT JOIN contracts c ON c.id = cr.contract_id
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

// CountTimesheets returns the count of time entries for the given org and date range.
func (r *ExportRepository) CountTimesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) (int, error) {
	roleSQL, paramCount := roleFilter("te.user_id", role)

	query := `SELECT COUNT(*)
	FROM time_entries te
	WHERE te.org_id = $1 AND te.entry_date >= $2 AND te.entry_date <= $3 AND te.is_deleted = false` + roleSQL

	var count int
	var err error
	if paramCount == 4 {
		err = r.pool.QueryRow(ctx, query, orgID, from, to, userID).Scan(&count)
	} else {
		err = r.pool.QueryRow(ctx, query, orgID, from, to).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("count timesheets: %w", err)
	}
	return count, nil
}

// CountExpenses returns the count of expenses for the given org and date range.
func (r *ExportRepository) CountExpenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) (int, error) {
	roleSQL, paramCount := roleFilter("e.user_id", role)

	query := `SELECT COUNT(*)
	FROM expenses e
	WHERE e.org_id = $1 AND e.expense_date >= $2 AND e.expense_date <= $3 AND e.is_deleted = false` + roleSQL

	var count int
	var err error
	if paramCount == 4 {
		err = r.pool.QueryRow(ctx, query, orgID, from, to, userID).Scan(&count)
	} else {
		err = r.pool.QueryRow(ctx, query, orgID, from, to).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("count expenses: %w", err)
	}
	return count, nil
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
