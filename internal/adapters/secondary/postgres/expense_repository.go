package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/expense"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// ExpenseRepository implements ports.ExpenseRepository using a pgxpool.
type ExpenseRepository struct {
	pool *pgxpool.Pool
}

// NewExpenseRepository creates a new ExpenseRepository.
func NewExpenseRepository(pool *pgxpool.Pool) *ExpenseRepository {
	return &ExpenseRepository{pool: pool}
}

var _ ports.ExpenseRepository = (*ExpenseRepository)(nil)

const expenseSelectColumns = `id, org_id, user_id, project_id, category, amount, km_distance,
	description, expense_date, status, COALESCE(current_approver_role, ''),
	submitted_at, COALESCE(receipt_url, ''), is_deleted, created_at, updated_at`

// expenseRowScanner is satisfied by pgx.Row and pgx.Rows.
type expenseRowScanner interface {
	Scan(dest ...any) error
}

// scanExpense scans a single expense row.
func scanExpense(s expenseRowScanner) (*expense.Expense, error) {
	var e expense.Expense
	var currentApproverRole string
	var submittedAt *time.Time
	var receiptURL string
	err := s.Scan(
		&e.ID, &e.OrgID, &e.UserID, &e.ProjectID,
		&e.Category, &e.Amount, &e.KmDistance,
		&e.Description, &e.EntryDate, &e.Status,
		&currentApproverRole, &submittedAt, &receiptURL,
		&e.IsDeleted, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if currentApproverRole != "" {
		e.CurrentApproverRole = &currentApproverRole
	}
	e.SubmittedAt = submittedAt
	if receiptURL != "" {
		e.ReceiptURL = &receiptURL
	}
	return &e, nil
}

// scanExpenses scans multiple expense rows.
func scanExpenses(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]expense.Expense, error) {
	var entries []expense.Expense
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, fmt.Errorf("scan expense: %w", err)
		}
		entries = append(entries, *e)
	}
	if entries == nil {
		entries = []expense.Expense{}
	}
	return entries, rows.Err()
}

// buildExpenseListQuery builds a dynamic SELECT query with numbered placeholders.
func buildExpenseListQuery(orgID uuid.UUID, filters ports.ExpenseListFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	args = append(args, orgID)
	conditions = append(conditions, "org_id = $1")

	args = append(args, filters.IsDeleted)
	conditions = append(conditions, fmt.Sprintf("is_deleted = $%d", len(args)))

	if filters.Date != "" {
		args = append(args, filters.Date)
		conditions = append(conditions, fmt.Sprintf("expense_date::date = $%d", len(args)))
	}
	if filters.Month != "" {
		args = append(args, filters.Month)
		conditions = append(conditions, fmt.Sprintf("EXTRACT(MONTH FROM expense_date) = $%d", len(args)))
	}
	if filters.Year != "" {
		args = append(args, filters.Year)
		conditions = append(conditions, fmt.Sprintf("EXTRACT(YEAR FROM expense_date) = $%d", len(args)))
	}
	if filters.Status != "" {
		args = append(args, filters.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if filters.ProjectID != "" {
		pID, err := uuid.Parse(filters.ProjectID)
		if err == nil {
			args = append(args, pID)
			conditions = append(conditions, fmt.Sprintf("project_id = $%d", len(args)))
		}
	}
	if filters.UserID != "" {
		uID, err := uuid.Parse(filters.UserID)
		if err == nil {
			args = append(args, uID)
			conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
		}
	}

	query := `SELECT ` + expenseSelectColumns + ` FROM expenses WHERE ` + strings.Join(conditions, " AND ")
	query += ` ORDER BY expense_date DESC`
	return query, args
}

// List returns expenses for an org filtered by the given filters.
func (r *ExpenseRepository) List(ctx context.Context, orgID uuid.UUID, filters ports.ExpenseListFilters) ([]expense.Expense, error) {
	query, args := buildExpenseListQuery(orgID, filters)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()
	return scanExpenses(rows)
}

// GetByID returns a single expense by ID, or expense.ErrExpenseNotFound.
func (r *ExpenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*expense.Expense, error) {
	query := `SELECT ` + expenseSelectColumns + ` FROM expenses WHERE id = $1`
	e, err := scanExpense(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, expense.ErrExpenseNotFound
		}
		return nil, fmt.Errorf("get expense by id: %w", err)
	}
	return e, nil
}

// Create inserts a new expense and returns the created record.
func (r *ExpenseRepository) Create(ctx context.Context, e *expense.Expense) (*expense.Expense, error) {
	e.ID = uuid.New()
	now := time.Now().UTC()

	query := `INSERT INTO expenses (id, org_id, user_id, project_id,
		category, amount, km_distance, description, expense_date,
		status, current_approver_role, submitted_at, receipt_url, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING ` + expenseSelectColumns

	created, err := scanExpense(r.pool.QueryRow(ctx, query,
		e.ID, e.OrgID, e.UserID, e.ProjectID,
		e.Category, e.Amount, e.KmDistance, e.Description, e.EntryDate,
		e.Status, e.CurrentApproverRole, e.SubmittedAt, e.ReceiptURL,
		e.IsDeleted, now, now,
	))
	if err != nil {
		return nil, wrapPGError(err, "create expense")
	}
	return created, nil
}

// Update performs a full-field update on an expense and returns the updated record.
func (r *ExpenseRepository) Update(ctx context.Context, e *expense.Expense) (*expense.Expense, error) {
	query := `UPDATE expenses SET
		project_id = $2, category = $3, amount = $4, km_distance = $5,
		description = $6, expense_date = $7, status = $8,
		current_approver_role = $9, submitted_at = $10, receipt_url = $11,
		updated_at = NOW()
		WHERE id = $1
		RETURNING ` + expenseSelectColumns

	updated, err := scanExpense(r.pool.QueryRow(ctx, query,
		e.ID, e.ProjectID, e.Category, e.Amount, e.KmDistance,
		e.Description, e.EntryDate, e.Status,
		e.CurrentApproverRole, e.SubmittedAt, e.ReceiptURL,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, expense.ErrExpenseNotFound
		}
		return nil, wrapPGError(err, "update expense")
	}
	return updated, nil
}

// Delete performs a soft delete on an expense.
func (r *ExpenseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE expenses SET is_deleted = true, updated_at = NOW() WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return wrapPGError(err, "delete expense")
	}
	if cmd.RowsAffected() == 0 {
		return expense.ErrExpenseNotFound
	}
	return nil
}

// IsPeriodLocked checks if a period is locked for the given org, project, and entry date.
func (r *ExpenseRepository) IsPeriodLocked(ctx context.Context, orgID, projectID uuid.UUID, entryDate string) (bool, error) {
	query := `SELECT EXISTS(
		SELECT 1 FROM financial_cutoff_periods
		WHERE org_id = $1 AND project_id = $2 AND $3::timestamptz BETWEEN period_start AND period_end AND is_locked = true
	)`
	var locked bool
	err := r.pool.QueryRow(ctx, query, orgID, projectID, entryDate).Scan(&locked)
	if err != nil {
		return false, fmt.Errorf("check expense period locked: %w", err)
	}
	return locked, nil
}

// ListPending returns pending expenses for an org, role-differentiated.
func (r *ExpenseRepository) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]expense.Expense, error) {
	var query string
	var args []interface{}

	switch role {
	case "manager":
		uID, err := uuid.Parse(userID)
		if err != nil {
			return nil, fmt.Errorf("parse user_id: %w", err)
		}
		query = `SELECT ` + expenseSelectColumns + ` FROM expenses
			WHERE org_id = $1 AND status IN ('submitted', 'pending_manager') AND is_deleted = false
			AND project_id IN (
				SELECT p.id FROM projects p
				JOIN project_managers pm ON pm.project_id = p.id
				WHERE pm.user_id = $2
			)`
		args = []interface{}{orgID, uID}
	case "finance":
		query = `SELECT ` + expenseSelectColumns + ` FROM expenses
			WHERE org_id = $1 AND status = 'pending_finance' AND is_deleted = false`
		args = []interface{}{orgID}
	default:
		query = `SELECT ` + expenseSelectColumns + ` FROM expenses
			WHERE org_id = $1 AND status = 'submitted' AND is_deleted = false`
		args = []interface{}{orgID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list pending expenses: %w", err)
	}
	defer rows.Close()
	return scanExpenses(rows)
}

// CreateApproval inserts a new approval record into expense_approvals.
func (r *ExpenseRepository) CreateApproval(ctx context.Context, a *expense.Approval) error {
	query := `INSERT INTO expense_approvals (id, expense_id, user_id, action, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(ctx, query,
		a.ID, a.EntryID, a.ActorUserID, a.Action, a.Comment, a.CreatedAt)
	return wrapPGError(err, "create expense approval")
}
