package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
)

// CustomerRepository implements ports.CustomerRepository using a pgxpool.
type CustomerRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: pool}
}

// ListByOrg returns customers for an organization with pagination.
func (r *CustomerRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]customer.Customer, error) {
	query := `SELECT id, org_id, name, contact_name, email, phone, address, vat_number, is_active, created_at
		FROM customers WHERE org_id = $1 ORDER BY name LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list customers by org: %w", err)
	}
	defer rows.Close()

	return scanCustomers(rows)
}

// Create inserts a new customer and returns it.
func (r *CustomerRepository) Create(ctx context.Context, c *customer.Customer) (*customer.Customer, error) {
	c.ID = uuid.New()

	query := `INSERT INTO customers (id, org_id, name, contact_name, email, phone, address, vat_number, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, org_id, name, contact_name, email, phone, address, vat_number, is_active, created_at`
	return scanCustomer(r.pool.QueryRow(ctx, query,
		c.ID, c.OrganizationID, c.CompanyName, c.ContactName, c.Email, c.Phone, c.Address, c.VATNumber, c.IsActive))
}

// GetByID returns a single customer, or customer.ErrCustomerNotFound.
func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	query := `SELECT id, org_id, name, contact_name, email, phone, address, vat_number, is_active, created_at
		FROM customers WHERE id = $1`
	c, err := scanCustomer(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, customer.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("get customer by id: %w", err)
	}
	return c, nil
}

// Update modifies an existing customer and returns it.
func (r *CustomerRepository) Update(ctx context.Context, c *customer.Customer) (*customer.Customer, error) {
	query := `UPDATE customers SET name = $1, contact_name = $2, email = $3, phone = $4,
		address = $5, vat_number = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING id, org_id, name, contact_name, email, phone, address, vat_number, is_active, created_at`
	return scanCustomer(r.pool.QueryRow(ctx, query,
		c.CompanyName, c.ContactName, c.Email, c.Phone, c.Address, c.VATNumber, c.IsActive, c.ID))
}

// Deactivate sets is_active to false for a customer.
func (r *CustomerRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx,
		`UPDATE customers SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deactivate customer: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return customer.ErrCustomerNotFound
	}
	return nil
}

// ListContractsByCustomer returns contract summaries for a customer.
func (r *CustomerRepository) ListContractsByCustomer(ctx context.Context, customerID uuid.UUID) ([]customer.ContractSummary, error) {
	query := `SELECT id, name, km_rate, currency, customer_id, governance_model,
		created_by_org_id, is_shared, is_active, created_at
		FROM contracts WHERE customer_id = $1 ORDER BY name`

	rows, err := r.pool.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("list contracts by customer: %w", err)
	}
	defer rows.Close()

	var summaries []customer.ContractSummary
	for rows.Next() {
		var s customer.ContractSummary
		err := rows.Scan(
			&s.ID, &s.Name, &s.KmRate, &s.Currency, &s.CustomerID, &s.GovernanceModel,
			&s.CreatedByOrgID, &s.IsShared, &s.IsActive, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan contract summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	if summaries == nil {
		summaries = []customer.ContractSummary{}
	}
	return summaries, rows.Err()
}

// CountContractsByCustomer returns the number of contracts for a customer.
func (r *CustomerRepository) CountContractsByCustomer(ctx context.Context, customerID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contracts WHERE customer_id = $1`, customerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count contracts by customer: %w", err)
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

type customerScanner interface {
	Scan(dest ...any) error
}

func scanCustomer(s customerScanner) (*customer.Customer, error) {
	var c customer.Customer
	err := s.Scan(
		&c.ID, &c.OrganizationID, &c.CompanyName, &c.ContactName,
		&c.Email, &c.Phone, &c.Address, &c.VATNumber,
		&c.IsActive, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanCustomers(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]customer.Customer, error) {
	var customers []customer.Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		customers = append(customers, *c)
	}
	if customers == nil {
		customers = []customer.Customer{}
	}
	return customers, rows.Err()
}
