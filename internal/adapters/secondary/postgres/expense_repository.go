package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// ExpenseRepository implements ports.ExpenseRepository using a pgxpool.
type ExpenseRepository struct {
	pool *pgxpool.Pool
}

// NewExpenseRepository creates a new ExpenseRepository.
func NewExpenseRepository(pool *pgxpool.Pool) *ExpenseRepository {
	return &ExpenseRepository{pool: pool}
}

// expenseSelectColumns returns the columns for expense queries.
// Maps columns to models.Expense fields (where available).
const expenseSelectColumns = `id, org_id, user_id, project_id, customer_id,
	unit_id, category, amount, km_distance, description, expense_date,
	status, created_at, updated_at`

// expenseRowScanner is satisfied by pgx.Row and pgx.Rows.
type expenseRowScanner interface {
	Scan(dest ...any) error
}

// scanExpense scans a single expense row into models.Expense.
func scanExpense(s expenseRowScanner) (*models.Expense, error) {
	var e models.Expense
	var category string
	err := s.Scan(
		&e.ID,
		&e.OrganizationID,
		&e.UserID,
		&e.ProjectID,
		&e.CustomerID,
		&e.UnitID,
		&category,
		&e.Amount,
		&e.KmDistance,
		&e.Description,
		&e.Date,
		&e.Status,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Map category string to ExpenseCategory
	ec := models.ExpenseCategory(category)
	e.Type = &ec
	return &e, nil
}

// scanExpenses scans multiple expense rows.
func scanExpenses(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]models.Expense, error) {
	var expenses []models.Expense
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, fmt.Errorf("scan expense: %w", err)
		}
		expenses = append(expenses, *e)
	}
	if expenses == nil {
		expenses = []models.Expense{}
	}
	return expenses, rows.Err()
}

// Create inserts a new expense and returns the created record.
func (r *ExpenseRepository) Create(ctx context.Context, e *models.Expense) (*models.Expense, error) {
	e.ID = uuid.New()

	category := ""
	if e.Type != nil {
		category = string(*e.Type)
	}

	query := `INSERT INTO expenses (id, org_id, user_id, project_id, customer_id,
		unit_id, category, amount, km_distance, description, expense_date,
		status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		RETURNING ` + expenseSelectColumns

	created, err := scanExpense(r.pool.QueryRow(ctx, query,
		e.ID, e.OrganizationID, e.UserID, e.ProjectID, e.CustomerID,
		e.UnitID, category, e.Amount, e.KmDistance, e.Description, e.Date,
		e.Status,
	))
	if err != nil {
		return nil, wrapPGError(err, "create expense")
	}
	return created, nil
}

// GetByID returns a single expense by ID, or ports.ErrNotFound.
func (r *ExpenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Expense, error) {
	query := `SELECT ` + expenseSelectColumns + ` FROM expenses WHERE id = $1`
	e, err := scanExpense(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get expense: %w", ports.ErrNotFound)
		}
		return nil, fmt.Errorf("get expense by id: %w", err)
	}
	return e, nil
}

// ListByOrg returns expenses for an org with pagination, excluding soft-deleted rows.
func (r *ExpenseRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.Expense, error) {
	query := `SELECT ` + expenseSelectColumns + ` FROM expenses
		WHERE org_id = $1 AND is_deleted = false
		ORDER BY expense_date DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list expenses by org: %w", err)
	}
	defer rows.Close()
	return scanExpenses(rows)
}

// Update performs a full-field update on an expense and returns the updated record.
func (r *ExpenseRepository) Update(ctx context.Context, e *models.Expense) (*models.Expense, error) {
	category := ""
	if e.Type != nil {
		category = string(*e.Type)
	}

	query := `UPDATE expenses SET
		project_id = $2, customer_id = $3, unit_id = $4,
		category = $5, amount = $6, km_distance = $7,
		description = $8, expense_date = $9, status = $10,
		updated_at = NOW()
		WHERE id = $1
		RETURNING ` + expenseSelectColumns

	updated, err := scanExpense(r.pool.QueryRow(ctx, query,
		e.ID, e.ProjectID, e.CustomerID, e.UnitID,
		category, e.Amount, e.KmDistance,
		e.Description, e.Date, e.Status,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("update expense: %w", ports.ErrNotFound)
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
		return fmt.Errorf("delete expense: %w", ports.ErrNotFound)
	}
	return nil
}

// Ensure ExpenseRepository implements ports.ExpenseRepository.
var _ ports.ExpenseRepository = (*ExpenseRepository)(nil)

// JSONB types for receipt_ocr_data scan (when needed).
// receiptOCRSanner handles nullable json.RawMessage.
type _ json.RawMessage
