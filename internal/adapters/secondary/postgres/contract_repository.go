package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
)

// ContractRepository implements ports.ContractRepository using a pgxpool.
type ContractRepository struct {
	pool *pgxpool.Pool
}

func NewContractRepository(pool *pgxpool.Pool) *ContractRepository {
	return &ContractRepository{pool: pool}
}

// baseContractQuery returns the SELECT list and FROM clause used by List and Get.
// $1 is reserved for orgID (used in the is_adopted EXISTS subquery).
func baseContractQuery() string {
	return `SELECT c.id, c.name, c.km_rate, c.currency, c.customer_id, c.governance_model,
		c.created_by_org_id, c.is_shared, c.is_active, c.created_at,
		COALESCE(o.name, '') AS created_by_org_name,
		(SELECT COUNT(*) FROM contract_adoptions ca WHERE ca.contract_id = c.id) AS adoption_count,
		EXISTS(SELECT 1 FROM contract_adoptions ca2 WHERE ca2.contract_id = c.id AND ca2.organization_id = $1) AS is_adopted,
		COALESCE((SELECT cu.name FROM customers cu WHERE cu.id = c.customer_id), '') AS customer_name,
		(SELECT COUNT(*) FROM time_entries te WHERE te.project_id IN (SELECT p.id FROM projects p WHERE p.contract_id = c.id)) AS time_entries_count
	FROM contracts c
	LEFT JOIN organizations o ON o.id = c.created_by_org_id`
}

// List returns contracts filtered by scope and optional isActive.
func (r *ContractRepository) List(ctx context.Context, orgID uuid.UUID, scope string, isActive *bool) ([]contractdomain.ContractResponse, error) {
	query := baseContractQuery()
	var conditions []string
	var args []interface{}

	args = append(args, orgID) // $1 = orgID for is_adopted EXISTS subquery

	switch scope {
	case "adopted":
		conditions = append(conditions, fmt.Sprintf("c.id IN (SELECT ca.contract_id FROM contract_adoptions ca WHERE ca.organization_id = $%d)", len(args)+1))
		args = append(args, orgID)
	case "all":
		conditions = append(conditions, "c.is_shared = true")
	default:
		conditions = append(conditions, fmt.Sprintf("c.created_by_org_id = $%d", len(args)+1))
		args = append(args, orgID)
	}

	if isActive != nil {
		conditions = append(conditions, fmt.Sprintf("c.is_active = $%d", len(args)+1))
		args = append(args, *isActive)
	}

	query += " WHERE " + strings.Join(conditions, " AND ")
	query += " ORDER BY c.name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	return scanContractResponses(rows)
}

// Create inserts a new contract and returns the full response.
func (r *ContractRepository) Create(ctx context.Context, orgID uuid.UUID, req *contractdomain.CreateContractRequest) (*contractdomain.ContractResponse, error) {
	id := uuid.New()

	_, err := r.pool.Exec(ctx, `INSERT INTO contracts (id, name, km_rate, currency, governance_model,
		created_by_org_id, is_shared, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW())`,
		id, req.Name, req.KmRate, req.Currency, req.GovernanceModel, orgID, req.IsShared)
	if err != nil {
		return nil, wrapPGError(err, "create contract")
	}

	return r.Get(ctx, orgID, id)
}

// Get returns a single contract with aggregates.
func (r *ContractRepository) Get(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractResponse, error) {
	query := baseContractQuery() + ` WHERE c.id = $2`
	res, err := scanContractResponse(r.pool.QueryRow(ctx, query, orgID, contractID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, contractdomain.ErrContractNotFound
		}
		return nil, fmt.Errorf("get contract: %w", err)
	}
	return res, nil
}

// Adopt adds an organization to the contract's adoptions.
func (r *ContractRepository) Adopt(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractAdoption, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contract_adoptions WHERE contract_id = $1 AND organization_id = $2)`,
		contractID, orgID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check adoption exists: %w", err)
	}
	if exists {
		return nil, contractdomain.ErrAlreadyAdopted
	}

	adoptionID := uuid.New()
	var adoption contractdomain.ContractAdoption
	err = r.pool.QueryRow(ctx,
		`INSERT INTO contract_adoptions (id, contract_id, organization_id, created_at)
		 VALUES ($1, $2, $3, NOW()) RETURNING id, contract_id, organization_id, created_at`,
		adoptionID, contractID, orgID).Scan(
		&adoption.ID, &adoption.ContractID, &adoption.OrganizationID, &adoption.AdoptedAt)
	if err != nil {
		return nil, wrapPGError(err, "adopt contract")
	}
	return &adoption, nil
}

// Update dynamically builds a SET clause from non-zero fields and returns the updated contract.
func (r *ContractRepository) Update(ctx context.Context, orgID, contractID uuid.UUID, req *contractdomain.UpdateContractRequest) (*contractdomain.ContractResponse, int, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != "" {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, req.Name)
		argIdx++
	}
	if req.KmRate != nil {
		sets = append(sets, fmt.Sprintf("km_rate = $%d", argIdx))
		args = append(args, *req.KmRate)
		argIdx++
	}
	if req.Currency != "" {
		sets = append(sets, fmt.Sprintf("currency = $%d", argIdx))
		args = append(args, req.Currency)
		argIdx++
	}
	if req.GovernanceModel != "" {
		sets = append(sets, fmt.Sprintf("governance_model = $%d", argIdx))
		args = append(args, req.GovernanceModel)
		argIdx++
	}
	if req.IsShared != nil {
		sets = append(sets, fmt.Sprintf("is_shared = $%d", argIdx))
		args = append(args, *req.IsShared)
		argIdx++
	}
	if req.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.CustomerID != nil {
		if *req.CustomerID == "" {
			sets = append(sets, fmt.Sprintf("customer_id = $%d", argIdx))
			args = append(args, nil)
		} else {
			cid, err := uuid.Parse(*req.CustomerID)
			if err != nil {
				return nil, 0, fmt.Errorf("parse customer_id: %w", err)
			}
			sets = append(sets, fmt.Sprintf("customer_id = $%d", argIdx))
			args = append(args, cid)
		}
		argIdx++
	}

	if len(sets) == 0 {
		return nil, 0, fmt.Errorf("no fields to update")
	}

	// Append updated_at = NOW() to SET and the WHERE params at the end
	allSets := append(sets, "updated_at = NOW()")

	whereIdx := argIdx
	args = append(args, contractID, orgID)

	query := fmt.Sprintf(`UPDATE contracts SET %s WHERE id = $%d AND created_by_org_id = $%d`,
		strings.Join(allSets, ", "), whereIdx, whereIdx+1)

	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, 0, wrapPGError(err, "update contract")
	}

	if cmd.RowsAffected() == 0 {
		return nil, 0, contractdomain.ErrContractNotFound
	}

	// Fetch the full response
	resp, err := r.Get(ctx, orgID, contractID)
	if err != nil {
		return nil, 0, err
	}
	return resp, 0, nil
}

// RecalculateMileage updates expense amounts based on contract km_rate.
func (r *ContractRepository) RecalculateMileage(ctx context.Context, orgID, contractID uuid.UUID, fromDate string, actorUserID uuid.UUID) (int, error) {
	// Fetch the contract's km_rate first
	var kmRate float64
	err := r.pool.QueryRow(ctx, `SELECT km_rate FROM contracts WHERE id = $1`, contractID).Scan(&kmRate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, contractdomain.ErrContractNotFound
		}
		return 0, fmt.Errorf("get contract km_rate: %w", err)
	}

	query := `UPDATE expenses SET amount = km_distance * $1
		WHERE project_id IN (SELECT id FROM projects WHERE contract_id = $2 AND org_id = $3)
		AND km_distance IS NOT NULL`

	var args []interface{}
	args = append(args, kmRate, contractID, orgID)
	argIdx := 4

	if fromDate != "" {
		query += fmt.Sprintf(" AND expense_date >= $%d", argIdx)
		args = append(args, fromDate)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("recalculate mileage: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// Delete removes a contract by ID, only if it belongs to the org.
func (r *ContractRepository) Delete(ctx context.Context, orgID, contractID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx,
		`DELETE FROM contracts WHERE id = $1 AND created_by_org_id = $2`,
		contractID, orgID)
	if err != nil {
		return wrapPGError(err, "delete contract")
	}
	if cmd.RowsAffected() == 0 {
		return contractdomain.ErrContractNotFound
	}
	return nil
}

// HasTimeEntries returns the count of time entries for projects under this contract.
func (r *ContractRepository) HasTimeEntries(ctx context.Context, contractID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM time_entries
		WHERE project_id IN (SELECT id FROM projects WHERE contract_id = $1)`
	var count int
	err := r.pool.QueryRow(ctx, query, contractID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("has time entries: %w", err)
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

type contractResponseScanner interface {
	Scan(dest ...any) error
}

func scanContractResponse(s contractResponseScanner) (*contractdomain.ContractResponse, error) {
	var c contractdomain.ContractResponse
	err := s.Scan(
		&c.ID, &c.Name, &c.KmRate, &c.Currency, &c.CustomerID, &c.GovernanceModel,
		&c.CreatedByOrgID, &c.IsShared, &c.IsActive, &c.CreatedAt,
		&c.CreatedByOrgName, &c.AdoptionCount, &c.IsAdopted,
		&c.CustomerName, &c.TimeEntriesCount,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanContractResponses(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]contractdomain.ContractResponse, error) {
	var contracts []contractdomain.ContractResponse
	for rows.Next() {
		c, err := scanContractResponse(rows)
		if err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		contracts = append(contracts, *c)
	}
	if contracts == nil {
		contracts = []contractdomain.ContractResponse{}
	}
	return contracts, rows.Err()
}
