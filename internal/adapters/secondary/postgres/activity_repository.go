package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// ActivityRepository implements ports.ActivityRepository using a pgxpool.
// It is the single repository for the recursive activities table
// (ADR-P-007 D-1/D-2/D-3/D-7, ADR-BE-014 R-6).
type ActivityRepository struct {
	pool *pgxpool.Pool
}

// Compile-time assertion: ActivityRepository implements the full port.
var _ ports.ActivityRepository = (*ActivityRepository)(nil)

func NewActivityRepository(pool *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{pool: pool}
}

// baseActivityQuery returns the SELECT list and FROM clause used by List/Get.
// $1 is reserved for orgID (is_adopted EXISTS subquery).
func baseActivityQuery() string {
	return `SELECT a.id, a.org_id, a.parent_id, a.name, a.description, a.kind, a.contract_id,
		a.governance_model, a.created_by_org_id, a.is_shared, a.billable, a.budget_amount,
		a.is_active, a.created_at, a.updated_at,
		COALESCE(p.name, '') AS parent_name,
		COALESCE(c.name, '') AS contract_name,
		COALESCE(o.name, '') AS created_by_org_name,
		(SELECT COUNT(*) FROM activity_adoptions aa WHERE aa.activity_id = a.id) AS adoption_count,
		EXISTS(SELECT 1 FROM activity_adoptions aa2 WHERE aa2.activity_id = a.id AND aa2.organization_id = $1) AS is_adopted
	FROM activities a
	LEFT JOIN activities p ON p.id = a.parent_id
	LEFT JOIN contracts c ON c.id = a.contract_id
	LEFT JOIN organizations o ON o.id = a.created_by_org_id`
}

// List returns activities for an org filtered by scope and optional filters.
func (r *ActivityRepository) List(ctx context.Context, orgID uuid.UUID, filter *activitydomain.ActivityFilter) ([]activitydomain.ActivityResponse, error) {
	query := baseActivityQuery()
	var conditions []string
	var args []interface{}

	args = append(args, orgID) // $1 = orgID for is_adopted EXISTS subquery

	if filter == nil {
		filter = &activitydomain.ActivityFilter{}
	}

	switch filter.Scope {
	case "adopted":
		conditions = append(conditions, fmt.Sprintf("a.id IN (SELECT aa.activity_id FROM activity_adoptions aa WHERE aa.organization_id = $%d)", len(args)+1))
		args = append(args, orgID)
	case "all":
		conditions = append(conditions, "a.is_shared = true")
	default:
		conditions = append(conditions, fmt.Sprintf("a.created_by_org_id = $%d", len(args)+1))
		args = append(args, orgID)
	}

	if filter.ContractID != "" {
		cid, err := uuid.Parse(filter.ContractID)
		if err != nil {
			return nil, fmt.Errorf("parse contract_id: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("a.contract_id = $%d", len(args)+1))
		args = append(args, cid)
	}
	if filter.ParentID != "" {
		pid, err := uuid.Parse(filter.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parse parent_id: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("a.parent_id = $%d", len(args)+1))
		args = append(args, pid)
	}
	if filter.Kind != "" {
		conditions = append(conditions, fmt.Sprintf("a.kind = $%d", len(args)+1))
		args = append(args, filter.Kind)
	}
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("a.is_active = $%d", len(args)+1))
		args = append(args, *filter.IsActive)
	}

	query += " WHERE " + strings.Join(conditions, " AND ")
	query += " ORDER BY a.name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}
	defer rows.Close()

	return scanActivityResponses(rows)
}

// Get returns a single activity by ID.
func (r *ActivityRepository) Get(ctx context.Context, orgID, activityID uuid.UUID) (*activitydomain.ActivityResponse, error) {
	query := baseActivityQuery() + ` WHERE a.id = $2`
	res, err := scanActivityResponse(r.pool.QueryRow(ctx, query, orgID, activityID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, activitydomain.ErrActivityNotFound
		}
		return nil, fmt.Errorf("get activity: %w", err)
	}
	return res, nil
}

// Create inserts a new activity and returns the full response.
func (r *ActivityRepository) Create(ctx context.Context, orgID uuid.UUID, req *activitydomain.CreateActivityRequest) (*activitydomain.ActivityResponse, error) {
	id := uuid.New()

	_, err := r.pool.Exec(ctx, `INSERT INTO activities (id, org_id, parent_id, name, description, kind,
		contract_id, governance_model, created_by_org_id, is_shared, billable, budget_amount,
		is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, true, NOW(), NOW())`,
		id, orgID, req.ParentID, req.Name, req.Description, req.Kind,
		req.ContractID, req.GovernanceModel, orgID, req.IsShared, req.Billable, req.BudgetAmount)
	if err != nil {
		return nil, wrapPGError(err, "create activity")
	}

	return r.Get(ctx, orgID, id)
}

// Adopt adds an organization to the activity's adoptions.
func (r *ActivityRepository) Adopt(ctx context.Context, orgID, activityID uuid.UUID) (*activitydomain.ActivityAdoption, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM activity_adoptions WHERE activity_id = $1 AND organization_id = $2)`,
		activityID, orgID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check adoption exists: %w", err)
	}
	if exists {
		return nil, activitydomain.ErrAlreadyAdopted
	}

	adoptionID := uuid.New()
	var adoption activitydomain.ActivityAdoption
	err = r.pool.QueryRow(ctx,
		`INSERT INTO activity_adoptions (id, activity_id, organization_id, created_at)
		 VALUES ($1, $2, $3, NOW()) RETURNING id, activity_id, organization_id, created_at`,
		adoptionID, activityID, orgID).Scan(
		&adoption.ID, &adoption.ActivityID, &adoption.OrganizationID, &adoption.AdoptedAt)
	if err != nil {
		return nil, wrapPGError(err, "adopt activity")
	}
	return &adoption, nil
}

// ListChildren returns the direct children of an activity.
func (r *ActivityRepository) ListChildren(ctx context.Context, parentID uuid.UUID) ([]activitydomain.ActivityResponse, error) {
	query := baseActivityQuery() + ` WHERE a.parent_id = $2 ORDER BY a.name`
	rows, err := r.pool.Query(ctx, query, uuid.Nil, parentID)
	if err != nil {
		return nil, fmt.Errorf("list activity children: %w", err)
	}
	defer rows.Close()
	return scanActivityResponses(rows)
}

// ListByContract returns activities directly linked to a contract.
func (r *ActivityRepository) ListByContract(ctx context.Context, contractID uuid.UUID) ([]activitydomain.ActivityResponse, error) {
	query := baseActivityQuery() + ` WHERE a.contract_id = $2 ORDER BY a.name`
	rows, err := r.pool.Query(ctx, query, uuid.Nil, contractID)
	if err != nil {
		return nil, fmt.Errorf("list activities by contract: %w", err)
	}
	defer rows.Close()
	return scanActivityResponses(rows)
}

// GetAncestry walks parent_id upward from the leaf to the root (recursive CTE).
// Used for commercial-chain resolution (D-3) and billability inheritance (D-7).
func (r *ActivityRepository) GetAncestry(ctx context.Context, id uuid.UUID) ([]activitydomain.Activity, error) {
	query := `WITH RECURSIVE ancestry AS (
		SELECT id, org_id, parent_id, name, description, kind, contract_id, governance_model,
			created_by_org_id, is_shared, billable, budget_amount, is_active, created_at, updated_at
		FROM activities WHERE id = $1
		UNION ALL
		SELECT a.id, a.org_id, a.parent_id, a.name, a.description, a.kind, a.contract_id,
			a.governance_model, a.created_by_org_id, a.is_shared, a.billable, a.budget_amount,
			a.is_active, a.created_at, a.updated_at
		FROM activities a
		INNER JOIN ancestry anc ON a.id = anc.parent_id
	)
	SELECT id, org_id, parent_id, name, description, kind, contract_id, governance_model,
		created_by_org_id, is_shared, billable, budget_amount, is_active, created_at, updated_at
	FROM ancestry`

	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("get ancestry: %w", err)
	}
	defer rows.Close()

	var chain []activitydomain.Activity
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ancestry activity: %w", err)
		}
		chain = append(chain, *a)
	}
	if chain == nil {
		chain = []activitydomain.Activity{}
	}
	return chain, rows.Err()
}

// ResolveCommercialContext walks the ancestry to the nearest ancestor with
// contract_id set and returns (contract_id, customer_id), or nil for a purely
// internal tree (D-3 — derived, never stored).
func (r *ActivityRepository) ResolveCommercialContext(ctx context.Context, activityID uuid.UUID) (*activitydomain.CommercialContext, error) {
	query := `WITH RECURSIVE chain AS (
		SELECT id, parent_id, contract_id FROM activities WHERE id = $1
		UNION ALL
		SELECT a.id, a.parent_id, a.contract_id
		FROM activities a
		INNER JOIN chain c ON a.id = c.parent_id
		WHERE c.contract_id IS NULL
	)
	SELECT c.contract_id, ct.customer_id
	FROM chain c
	LEFT JOIN contracts ct ON ct.id = c.contract_id
	WHERE c.contract_id IS NOT NULL
	LIMIT 1`

	var contractID uuid.UUID
	var customerID *uuid.UUID
	err := r.pool.QueryRow(ctx, query, activityID).Scan(&contractID, &customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolve commercial context: %w", err)
	}
	return &activitydomain.CommercialContext{
		ContractID: &contractID,
		CustomerID: customerID,
	}, nil
}

// ResolveBillability walks the ancestry and returns the resolved billability:
// the nearest non-NULL billable wins (an explicit FALSE overrides); if no
// explicit billable exists anywhere, the contract link decides
// (contract-linked = billable, internal = not, per D-7). Returns a non-nil
// *bool always — the fully resolved value.
func (r *ActivityRepository) ResolveBillability(ctx context.Context, activityID uuid.UUID) (*bool, error) {
	query := `WITH RECURSIVE chain AS (
		SELECT id, parent_id, contract_id, billable, 0 AS depth FROM activities WHERE id = $1
		UNION ALL
		SELECT a.id, a.parent_id, a.contract_id, a.billable, c.depth + 1
		FROM activities a
		INNER JOIN chain c ON a.id = c.parent_id
	)
	SELECT contract_id, billable
	FROM chain
	WHERE billable IS NOT NULL OR contract_id IS NOT NULL
	ORDER BY depth`

	rows, err := r.pool.Query(ctx, query, activityID)
	if err != nil {
		return nil, fmt.Errorf("resolve billability: %w", err)
	}
	defer rows.Close()

	// Rows come leaf-first (depth ascending). The first explicit billable is
	// the nearest one and wins — scan the raw nullable value so an explicit
	// FALSE is distinguishable from a NULL+contract row.
	contractLinked := false
	for rows.Next() {
		var contractID *uuid.UUID
		var billable *bool
		if err := rows.Scan(&contractID, &billable); err != nil {
			return nil, fmt.Errorf("scan billability: %w", err)
		}
		if billable != nil {
			return billable, nil
		}
		if contractID != nil {
			contractLinked = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resolve billability rows: %w", err)
	}

	if contractLinked {
		trueVal := true
		return &trueVal, nil
	}
	// Internal work with no explicit billable: non-billable (D-7).
	falseVal := false
	return &falseVal, nil
}

// KindExists reports whether the kind label exists in the org's activity_kinds
// catalog (ADR-P-007 D-2).
func (r *ActivityRepository) KindExists(ctx context.Context, orgID uuid.UUID, kind string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM activity_kinds WHERE org_id = $1 AND name = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, orgID, kind).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check activity kind exists: %w", err)
	}
	return exists, nil
}

// ListKinds returns the org's activity_kinds catalog ordered by name
// (ADR-P-007 D-2 — the catalog is org-scoped and extensible).
func (r *ActivityRepository) ListKinds(ctx context.Context, orgID uuid.UUID) ([]activitydomain.ActivityKind, error) {
	rows, err := r.pool.Query(ctx, `SELECT name FROM activity_kinds WHERE org_id = $1 ORDER BY name`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list activity kinds: %w", err)
	}
	defer rows.Close()

	var kinds []activitydomain.ActivityKind
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan activity kind: %w", err)
		}
		kinds = append(kinds, activitydomain.ActivityKind(name))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list activity kinds: %w", err)
	}
	if kinds == nil {
		kinds = []activitydomain.ActivityKind{}
	}
	return kinds, nil
}

// ListManagers returns the governance managers for an activity.
func (r *ActivityRepository) ListManagers(ctx context.Context, activityID uuid.UUID) ([]activitydomain.ActivityManager, error) {
	query := `SELECT am.id, am.activity_id, am.user_id, u.firstname || ' ' || u.lastname AS user_name, u.email, am.created_at
		FROM activity_managers am
		LEFT JOIN users u ON u.id = am.user_id
		WHERE am.activity_id = $1
		ORDER BY am.created_at DESC`

	rows, err := r.pool.Query(ctx, query, activityID)
	if err != nil {
		return nil, fmt.Errorf("list activity managers: %w", err)
	}
	defer rows.Close()

	var managers []activitydomain.ActivityManager
	for rows.Next() {
		var m activitydomain.ActivityManager
		err := rows.Scan(&m.ID, &m.ActivityID, &m.UserID, &m.UserName, &m.Email, &m.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan activity manager: %w", err)
		}
		managers = append(managers, m)
	}
	if managers == nil {
		managers = []activitydomain.ActivityManager{}
	}
	return managers, rows.Err()
}

// AddManager adds a user as governance manager to an activity.
func (r *ActivityRepository) AddManager(ctx context.Context, activityID, userID uuid.UUID) (*activitydomain.ActivityManager, error) {
	mid := uuid.New()
	var m activitydomain.ActivityManager
	err := r.pool.QueryRow(ctx,
		`INSERT INTO activity_managers (id, activity_id, user_id, created_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING id, activity_id, user_id, created_at`,
		mid, activityID, userID).Scan(
		&m.ID, &m.ActivityID, &m.UserID, &m.CreatedAt)
	if err != nil {
		return nil, wrapPGError(err, "add activity manager")
	}

	err = r.pool.QueryRow(ctx,
		`SELECT firstname || ' ' || lastname, email FROM users WHERE id = $1`, userID).Scan(
		&m.UserName, &m.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, activitydomain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get manager user info: %w", err)
	}
	return &m, nil
}

// RemoveManager removes a user from activity managers.
func (r *ActivityRepository) RemoveManager(ctx context.Context, activityID, userID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx,
		`DELETE FROM activity_managers WHERE activity_id = $1 AND user_id = $2`,
		activityID, userID)
	if err != nil {
		return wrapPGError(err, "remove activity manager")
	}
	if cmd.RowsAffected() == 0 {
		return activitydomain.ErrActivityNotFound
	}
	return nil
}

// Update dynamically builds a SET clause from non-zero fields.
func (r *ActivityRepository) Update(ctx context.Context, orgID, activityID uuid.UUID, req *activitydomain.UpdateActivityRequest) (*activitydomain.ActivityResponse, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != "" {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, req.Name)
		argIdx++
	}
	if req.Description != "" {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, req.Description)
		argIdx++
	}
	if req.Kind != "" {
		sets = append(sets, fmt.Sprintf("kind = $%d", argIdx))
		args = append(args, req.Kind)
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
	if req.Billable != nil {
		sets = append(sets, fmt.Sprintf("billable = $%d", argIdx))
		args = append(args, *req.Billable)
		argIdx++
	}
	if req.BudgetAmount != nil {
		sets = append(sets, fmt.Sprintf("budget_amount = $%d", argIdx))
		args = append(args, *req.BudgetAmount)
		argIdx++
	}
	if req.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.ParentID != nil {
		sets = append(sets, fmt.Sprintf("parent_id = $%d", argIdx))
		args = append(args, *req.ParentID)
		argIdx++
	}
	if req.ContractID != nil {
		sets = append(sets, fmt.Sprintf("contract_id = $%d", argIdx))
		args = append(args, *req.ContractID)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	allSets := append(sets, "updated_at = NOW()")
	whereIdx := argIdx
	args = append(args, activityID, orgID)

	query := fmt.Sprintf(`UPDATE activities SET %s WHERE id = $%d AND created_by_org_id = $%d`,
		strings.Join(allSets, ", "), whereIdx, whereIdx+1)

	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, wrapPGError(err, "update activity")
	}
	if cmd.RowsAffected() == 0 {
		return nil, activitydomain.ErrActivityNotFound
	}
	return r.Get(ctx, orgID, activityID)
}

// Delete removes an activity and its adoptions in a single transaction.
// Children and referenced entries are protected by ON DELETE RESTRICT /
// FK constraints at the schema level.
func (r *ActivityRepository) Delete(ctx context.Context, orgID, activityID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM activity_adoptions WHERE activity_id = $1`, activityID)
	if err != nil {
		return wrapPGError(err, "delete activity adoptions")
	}

	cmd, err := tx.Exec(ctx,
		`DELETE FROM activities WHERE id = $1 AND created_by_org_id = $2`,
		activityID, orgID)
	if err != nil {
		return wrapPGError(err, "delete activity")
	}
	if cmd.RowsAffected() == 0 {
		return activitydomain.ErrActivityNotFound
	}

	return tx.Commit(ctx)
}

// HasChildren returns true if the activity has at least one child.
func (r *ActivityRepository) HasChildren(ctx context.Context, activityID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM activities WHERE parent_id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, activityID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has children: %w", err)
	}
	return exists, nil
}

// HasActiveTimeEntries checks the activity's whole subtree for active
// (non-approved, non-rejected) time entries. Returns (hasEntries,
// hasDescendantEntries) — the first bool covers entries on the activity
// itself, the second covers entries on any descendant.
func (r *ActivityRepository) HasActiveTimeEntries(ctx context.Context, activityID uuid.UUID) (bool, bool, error) {
	query := `WITH RECURSIVE subtree AS (
		SELECT id FROM activities WHERE id = $1
		UNION ALL
		SELECT a.id FROM activities a
		INNER JOIN subtree s ON a.parent_id = s.id
	)
	SELECT
		EXISTS(SELECT 1 FROM time_entries te
			WHERE te.activity_id = $1
			AND te.status NOT IN ('approved', 'rejected')
			AND te.is_deleted = false) AS has_entries,
		EXISTS(SELECT 1 FROM time_entries te
			WHERE te.activity_id IN (SELECT id FROM subtree)
			AND te.activity_id != $1
			AND te.status NOT IN ('approved', 'rejected')
			AND te.is_deleted = false) AS has_descendant_entries`
	var hasEntries, hasDescendantEntries bool
	err := r.pool.QueryRow(ctx, query, activityID).Scan(&hasEntries, &hasDescendantEntries)
	if err != nil {
		return false, false, fmt.Errorf("has active time entries: %w", err)
	}
	return hasEntries, hasDescendantEntries, nil
}

// HasActiveExpenses checks the activity's whole subtree for active expenses.
func (r *ActivityRepository) HasActiveExpenses(ctx context.Context, activityID uuid.UUID) (bool, error) {
	query := `WITH RECURSIVE subtree AS (
		SELECT id FROM activities WHERE id = $1
		UNION ALL
		SELECT a.id FROM activities a
		INNER JOIN subtree s ON a.parent_id = s.id
	)
	SELECT EXISTS(SELECT 1 FROM expenses e
		WHERE e.activity_id IN (SELECT id FROM subtree)
		AND e.status NOT IN ('approved', 'rejected')
		AND e.is_deleted = false)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, activityID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has active expenses: %w", err)
	}
	return exists, nil
}

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

// activityScanner is satisfied by pgx.Row and pgx.Rows.
type activityScanner interface {
	Scan(dest ...any) error
}

// scanActivity scans a single activities row into the domain type.
// parent_id, contract_id, billable, budget_amount are nullable.
func scanActivity(s activityScanner) (*activitydomain.Activity, error) {
	var a activitydomain.Activity
	var parentID *uuid.UUID
	var contractID *uuid.UUID
	var billable *bool
	var budgetAmount *float64

	err := s.Scan(
		&a.ID, &a.OrgID, &parentID, &a.Name, &a.Description, &a.Kind, &contractID,
		&a.GovernanceModel, &a.CreatedByOrgID, &a.IsShared, &billable, &budgetAmount,
		&a.IsActive, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	a.ParentID = parentID
	a.ContractID = contractID
	a.Billable = billable
	a.BudgetAmount = budgetAmount
	return &a, nil
}

func scanActivityResponse(s activityScanner) (*activitydomain.ActivityResponse, error) {
	var a activitydomain.ActivityResponse
	var parentID *uuid.UUID
	var contractID *uuid.UUID
	var billable *bool
	var budgetAmount *float64

	err := s.Scan(
		&a.ID, &a.OrgID, &parentID, &a.Name, &a.Description, &a.Kind, &contractID,
		&a.GovernanceModel, &a.CreatedByOrgID, &a.IsShared, &billable, &budgetAmount,
		&a.IsActive, &a.CreatedAt, &a.UpdatedAt,
		&a.ParentName, &a.ContractName, &a.CreatedByOrgName,
		&a.AdoptionCount, &a.IsAdopted,
	)
	if err != nil {
		return nil, err
	}
	a.ParentID = parentID
	a.ContractID = contractID
	a.Billable = billable
	a.BudgetAmount = budgetAmount
	return &a, nil
}

func scanActivityResponses(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]activitydomain.ActivityResponse, error) {
	var activities []activitydomain.ActivityResponse
	for rows.Next() {
		a, err := scanActivityResponse(rows)
		if err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		activities = append(activities, *a)
	}
	if activities == nil {
		activities = []activitydomain.ActivityResponse{}
	}
	return activities, rows.Err()
}
