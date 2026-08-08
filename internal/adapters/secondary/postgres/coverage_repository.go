package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/coverage"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// CoverageRepository implements ports.CoverageRepository using a pgxpool
// (schema 019 + 020). It is the correctness layer of the coverage plane
// (CR-01 lesson): every invariant the service fast-fails at the pool level is
// re-validated HERE inside the mutator transaction under FOR UPDATE locks —
// pool-level checks are fast-fail UX only, the in-tx re-check is the
// guarantee. State writes and their audit_logs rows commit in the SAME
// transaction (BE-016, insertTicketAudit precedent).
type CoverageRepository struct {
	pool *pgxpool.Pool
}

// Compile-time assertion: CoverageRepository implements the port.
var _ ports.CoverageRepository = (*CoverageRepository)(nil)

func NewCoverageRepository(pool *pgxpool.Pool) *CoverageRepository {
	return &CoverageRepository{pool: pool}
}

// insertCoverageAudit writes one audit_logs row inside the given transaction
// (BE-016 — mirrors insertTicketAudit, ticket_repository.go): entity_type is
// written from log.EntityType so the caller controls the vocabulary (the
// coverage plane pins coverage.AuditEntityCoverageAllocation); payload JSONB
// marshaled from the audit's Payload map; nil actor/empty comment written as
// SQL NULL. The row id is generated here — the AuditLog's ID field is not
// persisted. Never fire-and-forget: mutators write the audit row in-tx and
// roll back the whole operation if it fails.
func insertCoverageAudit(ctx context.Context, tx pgx.Tx, log *audit.AuditLog) error {
	id := uuid.New()

	var payload any
	if len(log.Payload) > 0 {
		payloadJSON, err := json.Marshal(log.Payload)
		if err != nil {
			return fmt.Errorf("marshal coverage audit payload: %w", err)
		}
		payload = payloadJSON
	}

	var comment any
	if log.Comment != "" {
		comment = log.Comment
	}

	_, err := tx.Exec(ctx,
		`INSERT INTO audit_logs (id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, log.OrgID, log.EntityType, log.EntityID, log.Action, log.ActorID, comment, payload, log.CreatedAt)
	if err != nil {
		return wrapPGError(err, "insert coverage audit log")
	}
	return nil
}

// allocationColumns is the canonical SELECT column list for
// coverage_allocations rows.
const allocationColumns = `id, org_id, entry_type, entry_id, source_type,
	contract_id, unit_id, hours, reason, justification, created_at, updated_at`

// scanCoverageAllocationRow scans a pgx.Row into a CoverageAllocation,
// normalizing nullable columns. source_type is nullable in the schema (3VL
// legacy rows pass the 019 CHECKs) — a NULL scans as "" so every row is
// representable in the domain shape.
func scanCoverageAllocationRow(row pgx.Row) (*coverage.CoverageAllocation, error) {
	var a coverage.CoverageAllocation
	var sourceType *string
	if err := row.Scan(&a.ID, &a.OrgID, &a.EntryType, &a.EntryID, &sourceType,
		&a.ContractID, &a.UnitID, &a.Hours, &a.Reason, &a.Justification,
		&a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	if sourceType != nil {
		a.SourceType = *sourceType
	}
	return &a, nil
}

// ReplaceAllocations atomically replaces the full allocation set for an entry
// (D-07): one transaction holding the entry row FOR UPDATE while it re-checks
// status/is_deleted and the Σ invariant, then DELETE-all + INSERT-set + audit
// row, then Commit — partial states are impossible by construction (COV-01,
// CR-01 closure).
//
// Validation is AUTHORITATIVE inside this tx (Pitfall 1, T-12-14): the
// service's pool-level checks are fast-fail UX only; the lock + in-tx
// re-checks are the correctness guarantee. A violating state never commits —
// proven by the TestCoverageReplace_Concurrent battery.
func (r *CoverageRepository) ReplaceAllocations(ctx context.Context, orgID, entryID uuid.UUID, allocs []*coverage.CoverageAllocation, auditLog *audit.AuditLog) ([]*coverage.CoverageAllocation, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin replace allocations: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Lock the entry row and re-validate (CR-01: in-tx check, never
	// pool-only). The FOR UPDATE lock serializes concurrent replace-sets for
	// the same entry; the status/is_deleted predicate re-reads the committed
	// row, so a draft/pending/deleted entry can never be covered.
	var entryHours float64
	err = tx.QueryRow(ctx,
		`SELECT hours FROM time_entries
		 WHERE id = $1 AND org_id = $2 AND status = 'approved' AND is_deleted = false
		 FOR UPDATE`, entryID, orgID).Scan(&entryHours)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, coverage.ErrEntryNotCoverable
	}
	if err != nil {
		return nil, fmt.Errorf("lock entry for replace allocations: %w", err)
	}

	// 2. Authoritative in-tx Σ re-validation (Pitfall 1): the set must sum to
	// the entry's hours exactly. Cents arithmetic avoids float64 artifacts.
	var sumCents int64
	for _, a := range allocs {
		sumCents += int64(math.Round(a.Hours * 100))
	}
	if sumCents != int64(math.Round(entryHours*100)) {
		return nil, coverage.ErrAllocationSumMismatch
	}

	// 3. DELETE the existing set for the entry (the ledger is replace-only —
	// no incremental CRUD, D-07).
	if _, err := tx.Exec(ctx,
		`DELETE FROM coverage_allocations WHERE entry_id = $1 AND entry_type = 'time'`,
		entryID); err != nil {
		return nil, wrapPGError(err, "delete allocations for replace")
	}

	// 4. INSERT the full set (1..N rows). The boundary DTO never carries
	// allocation ids (D-07), so every row's id is generated here — the PK is
	// table-wide, and a uuid.Nil id would collide with the first row inserted
	// anywhere in the ledger (CR-01). The DB CHECKs (019) backstop the
	// refs-to-type and mandatory-field vocabularies as a third line.
	now := time.Now().UTC()
	for _, a := range allocs {
		if a.ID == uuid.Nil {
			a.ID = uuid.New()
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO coverage_allocations (id, org_id, entry_type, entry_id, source_type,
				contract_id, unit_id, hours, reason, justification, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)`,
			a.ID, orgID, coverage.EntryTypeTime, entryID, a.SourceType,
			a.ContractID, a.UnitID, a.Hours, a.Reason, a.Justification, now); err != nil {
			return nil, wrapPGError(err, "insert allocation row")
		}
	}

	// 5. Audit row in the SAME tx (BE-016, T-12-16): the set change is not
	// durable without its event; a failed audit insert rolls back the replace.
	if auditLog != nil {
		if err := insertCoverageAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit replace allocations: %w", err)
	}
	return r.ListByEntry(ctx, orgID, entryID)
}

// ListByEntry returns the current allocation set for an entry (empty slice
// when none exist), org-scoped, oldest first. Read-only.
func (r *CoverageRepository) ListByEntry(ctx context.Context, orgID, entryID uuid.UUID) ([]*coverage.CoverageAllocation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+allocationColumns+`
		 FROM coverage_allocations
		 WHERE org_id = $1 AND entry_id = $2 AND entry_type = 'time'
		 ORDER BY created_at, id`, orgID, entryID)
	if err != nil {
		return nil, wrapPGError(err, "list allocations by entry")
	}
	defer rows.Close()

	var allocs []*coverage.CoverageAllocation
	for rows.Next() {
		a, err := scanCoverageAllocationRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan allocation row: %w", err)
		}
		allocs = append(allocs, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate allocation rows: %w", err)
	}
	return allocs, nil
}

// ToCoverQueue returns every approved, non-deleted, org-scoped 'time' entry
// with uncovered hours > 0 (COV-01/04, D-06): the LEFT JOIN + HAVING
// predicate keeps no-source entries (Σ = 0) present — uncovered work is a
// visible state, never an implicit gap. The employee name uses the
// firstname/lastname CONCAT house pattern (users has no `name` column); the
// queue is ordered by (entry_date, id) for a deterministic stable order.
// Proposal enrichment is service-side (D-06) — this supplies the raw
// covered/uncovered split only.
func (r *CoverageRepository) ToCoverQueue(ctx context.Context, orgID uuid.UUID) ([]coverage.ToCoverQueueRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT te.id, te.user_id,
		        CONCAT(COALESCE(u.firstname, ''), ' ', COALESCE(u.lastname, '')) AS employee,
		        te.entry_date, te.activity_id, COALESCE(a.name, ''),
		        te.hours, COALESCE(SUM(ca.hours), 0)
		 FROM time_entries te
		 LEFT JOIN coverage_allocations ca ON ca.entry_id = te.id AND ca.entry_type = 'time'
		 LEFT JOIN users u ON u.id = te.user_id
		 LEFT JOIN activities a ON a.id = te.activity_id
		 WHERE te.org_id = $1 AND te.status = 'approved' AND te.is_deleted = false
		 GROUP BY te.id, u.firstname, u.lastname, a.name
		 HAVING COALESCE(SUM(ca.hours), 0) < te.hours
		 ORDER BY te.entry_date, te.id`, orgID)
	if err != nil {
		return nil, wrapPGError(err, "list to-cover queue")
	}
	defer rows.Close()

	var queue []coverage.ToCoverQueueRow
	for rows.Next() {
		var row coverage.ToCoverQueueRow
		if err := rows.Scan(&row.EntryID, &row.EmployeeID, &row.EmployeeName,
			&row.EntryDate, &row.ActivityID, &row.ActivityName, &row.Hours,
			&row.CoveredHours); err != nil {
			return nil, fmt.Errorf("scan to-cover row: %w", err)
		}
		// Uncovered = hours − Σ drawn, rounded to cents (Pitfall 6: the
		// predicate must be exact — DECIMAL arithmetic in SQL is exact, the
		// float64 split only renders the read model).
		row.UncoveredHours = roundCents(row.Hours - row.CoveredHours)
		queue = append(queue, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate to-cover rows: %w", err)
	}
	return queue, nil
}

// BucketBalance returns the derived support-bucket balance for a contract
// (D-02, COV-02): sold_hours − Σ allocations drawn from it — computed on
// read, never stored. ANY source_type counts (transfers draw the target
// contract's balance, Pitfall 9 — the join scopes by contract_id only);
// negative balances are returned as-is, never gated (D-03 — the report is
// the control, not a gate).
//
// The pre-check admits only org-visible contracts via the adoption-aware
// predicate: a contract resolves for the org iff created_by_org_id = org OR
// (is_shared AND adopted) — the same visibility rule the 12-05 service
// applies to allocation contract refs; a non-adopted shared contract reads
// as coverage.ErrNotFound.
func (r *CoverageRepository) BucketBalance(ctx context.Context, orgID, contractID uuid.UUID) (float64, error) {
	var visible bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM contracts c
			WHERE c.id = $1 AND (c.created_by_org_id = $2 OR (c.is_shared = true AND EXISTS(
				SELECT 1 FROM contract_adoptions WHERE contract_id = c.id AND organization_id = $2
			)))
		 )`, contractID, orgID).Scan(&visible)
	if err != nil {
		return 0, wrapPGError(err, "check contract visibility for balance")
	}
	if !visible {
		return 0, coverage.ErrNotFound
	}

	var balance float64
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(c.sold_hours, 0) - COALESCE(SUM(ca.hours), 0)
		 FROM contracts c
		 LEFT JOIN coverage_allocations ca ON ca.contract_id = c.id
		 WHERE c.id = $1
		 GROUP BY c.sold_hours`, contractID).Scan(&balance)
	if err != nil {
		return 0, wrapPGError(err, "compute bucket balance")
	}
	return balance, nil
}

// roundCents rounds a float to two decimal places — the cents-safe compare
// unit for hours (DECIMAL(8,2) values).
func roundCents(v float64) float64 {
	return math.Round(v*100) / 100
}

// ClosePeriod writes the frozen period-close snapshot (D-10/D-11/D-12,
// COV-04, T-12-15) in ONE transaction: in-tx overlap rejection → lock the
// period's entries FOR UPDATE → copy each entry's current allocations into
// append-only snapshot rows → header + audit row in-tx → Commit. The FOR
// UPDATE lock serializes with concurrent ReplaceAllocations on the period's
// entries, so the frozen rows are the commit-adjacent allocation state — a
// reported period never changes retroactively, and later allocation edits
// never alter the snapshot (they touch live rows only).
//
// The overlap predicate is INCLUSIVE on both ends (A6): an existing close
// whose [period_start, period_end] shares even one day with the new period
// rejects it (period_start <= $3::date AND period_end >= $2::date) — catches
// contained, partial, and wider-than-existing closes, not just identical
// bounds. No UNIQUE constraint exists on 020; this in-tx check is
// authoritative (409 at the handler).
//
// WR-03: the check is made concurrency-safe by a per-org advisory xact lock
// taken BEFORE it — without it, two concurrent closes of the same period
// both observe "no overlap" at READ COMMITTED (no header exists yet), then
// serialize only on the entry FOR UPDATE (step 2, after the check) and both
// commit. The xact lock serializes closes for the org: the loser's check
// runs only after the winner's tx committed, so it sees the committed
// header and returns 409. (Closes are rare manager-only operations, so
// per-org serialization is free.)
func (r *CoverageRepository) ClosePeriod(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time, closeID, closedBy uuid.UUID, auditLog *audit.AuditLog) (*coverage.PeriodClose, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin close period: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 0. WR-03: serialize closes per org — released automatically at tx end.
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, orgID); err != nil {
		return nil, wrapPGError(err, "acquire close serialization lock")
	}

	// 1. In-tx overlap check (A6): inclusive-overlap predicate — any period
	// sharing even one day with an existing close for the org is rejected.
	var overlap bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM coverage_period_closes
			WHERE org_id = $1 AND period_start <= $3::date AND period_end >= $2::date
		 )`, orgID, periodStart, periodEnd).Scan(&overlap)
	if err != nil {
		return nil, wrapPGError(err, "check overlapping close period")
	}
	if overlap {
		return nil, coverage.ErrPeriodAlreadyClosed
	}

	// 2. Freeze source: lock the period's entries FOR UPDATE — serializes
	// with in-flight replaces (an uncommitted replace holds the entry row
	// lock; the allocation reads below wait for it to commit or roll back).
	type frozenEntry struct {
		id         uuid.UUID
		employeeID uuid.UUID
		entryDate  time.Time
		activityID uuid.UUID
	}
	var entries []frozenEntry
	entryRows, err := tx.Query(ctx,
		`SELECT id, user_id, entry_date, activity_id FROM time_entries
		 WHERE org_id = $1 AND status = 'approved' AND is_deleted = false
		   AND entry_date::date BETWEEN $2::date AND $3::date
		 FOR UPDATE`, orgID, periodStart, periodEnd)
	if err != nil {
		return nil, wrapPGError(err, "lock period entries for close")
	}
	for entryRows.Next() {
		var e frozenEntry
		if err := entryRows.Scan(&e.id, &e.employeeID, &e.entryDate, &e.activityID); err != nil {
			entryRows.Close()
			return nil, fmt.Errorf("scan period entry for close: %w", err)
		}
		entries = append(entries, e)
	}
	if err := entryRows.Err(); err != nil {
		entryRows.Close()
		return nil, fmt.Errorf("iterate period entries for close: %w", err)
	}
	entryRows.Close()

	// 3. Insert the header (id = caller-supplied closeID).
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx,
		`INSERT INTO coverage_period_closes (id, org_id, period_start, period_end, closed_by, closed_at)
		 VALUES ($1, $2, $3::date, $4::date, $5, $6)`,
		closeID, orgID, periodStart, periodEnd, closedBy, now); err != nil {
		return nil, wrapPGError(err, "insert close header")
	}

	// 4. One coverage_snapshot_rows row per current allocation of each frozen
	// entry — the resolved contract/unit refs ARE the chain snapshot (D-11),
	// copied as stored; employee_id + entry_date come from the entry row.
	// Entries with no allocations simply contribute no rows. Allocations are
	// read into memory first (a pgx rows cursor cannot be interleaved with
	// Exec on the same tx connection).
	rowCount := 0
	type frozenAlloc struct {
		sourceType    string
		contractID    *uuid.UUID
		unitID        *uuid.UUID
		hours         float64
		reason        *string
		justification *string
	}
	for _, e := range entries {
		var frozen []frozenAlloc
		allocRows, err := tx.Query(ctx,
			`SELECT source_type, contract_id, unit_id, hours, reason, justification
			 FROM coverage_allocations
			 WHERE entry_id = $1 AND entry_type = 'time'`, e.id)
		if err != nil {
			return nil, wrapPGError(err, "read allocations for close freeze")
		}
		for allocRows.Next() {
			var fa frozenAlloc
			if err := allocRows.Scan(&fa.sourceType, &fa.contractID, &fa.unitID,
				&fa.hours, &fa.reason, &fa.justification); err != nil {
				allocRows.Close()
				return nil, fmt.Errorf("scan allocation for close freeze: %w", err)
			}
			frozen = append(frozen, fa)
		}
		if err := allocRows.Err(); err != nil {
			allocRows.Close()
			return nil, fmt.Errorf("iterate allocations for close freeze: %w", err)
		}
		allocRows.Close()

		for _, fa := range frozen {
			if _, err := tx.Exec(ctx,
				`INSERT INTO coverage_snapshot_rows (id, close_id, entry_id, employee_id, entry_date, activity_id,
					source_type, contract_id, unit_id, hours, reason, justification)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				uuid.New(), closeID, e.id, e.employeeID, e.entryDate, e.activityID,
				fa.sourceType, fa.contractID, fa.unitID, fa.hours, fa.reason, fa.justification); err != nil {
				return nil, wrapPGError(err, "insert snapshot row")
			}
			rowCount++
		}
	}

	// 5. Audit row in the SAME tx (BE-016, T-12-16): the close is not durable
	// without its event; a failed audit insert rolls back the snapshot.
	if auditLog != nil {
		if err := insertCoverageAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit close period: %w", err)
	}
	return r.GetSnapshot(ctx, orgID, closeID)
}

// GetSnapshot returns a frozen period-close snapshot (header + rows) by close
// id, org-scoped. Read-only — snapshot rows have no UPDATE/DELETE surface
// (D-10, Pitfall 7); later allocation edits never alter the copy. A missing
// or cross-org close surfaces coverage.ErrNotFound.
func (r *CoverageRepository) GetSnapshot(ctx context.Context, orgID, closeID uuid.UUID) (*coverage.PeriodClose, error) {
	var pc coverage.PeriodClose
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, period_start, period_end, closed_by, closed_at
		 FROM coverage_period_closes
		 WHERE id = $1 AND org_id = $2`,
		closeID, orgID).Scan(&pc.ID, &pc.OrgID, &pc.PeriodStart, &pc.PeriodEnd, &pc.ClosedBy, &pc.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, coverage.ErrNotFound
	}
	if err != nil {
		return nil, wrapPGError(err, "get coverage snapshot header")
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, close_id, entry_id, employee_id, entry_date, activity_id,
		        source_type, contract_id, unit_id, hours, reason, justification
		 FROM coverage_snapshot_rows
		 WHERE close_id = $1
		 ORDER BY entry_date, entry_id, id`, closeID)
	if err != nil {
		return nil, wrapPGError(err, "get coverage snapshot rows")
	}
	defer rows.Close()

	for rows.Next() {
		var row coverage.SnapshotRow
		if err := rows.Scan(&row.ID, &row.CloseID, &row.EntryID, &row.EmployeeID,
			&row.EntryDate, &row.ActivityID, &row.SourceType, &row.ContractID,
			&row.UnitID, &row.Hours, &row.Reason, &row.Justification); err != nil {
			return nil, fmt.Errorf("scan snapshot row: %w", err)
		}
		pc.Rows = append(pc.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot rows: %w", err)
	}
	return &pc, nil
}

// ListHistory returns the entry's append-only audit stream (A7, T-12-16),
// filtered to entity_type='coverage_allocation', ordered by created_at — the
// trail behind every allocation change and the entry's period closes. The
// payload JSONB round-trips into a map (ListHistory analog).
func (r *CoverageRepository) ListHistory(ctx context.Context, orgID, entryID uuid.UUID) ([]audit.AuditLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at
		 FROM audit_logs
		 WHERE org_id = $1 AND entity_type = 'coverage_allocation' AND entity_id = $2
		 ORDER BY created_at, id`, orgID, entryID)
	if err != nil {
		return nil, wrapPGError(err, "list coverage history")
	}
	defer rows.Close()

	var logs []audit.AuditLog
	for rows.Next() {
		var l audit.AuditLog
		var actorID *uuid.UUID
		var comment *string
		var payloadJSON []byte
		if err := rows.Scan(&l.ID, &l.OrgID, &l.EntityType, &l.EntityID, &l.Action,
			&actorID, &comment, &payloadJSON, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan coverage audit log: %w", err)
		}
		l.ActorID = actorID
		if comment != nil {
			l.Comment = *comment
		}
		if len(payloadJSON) > 0 {
			var payload map[string]any
			if err := json.Unmarshal(payloadJSON, &payload); err != nil {
				return nil, fmt.Errorf("unmarshal coverage audit payload: %w", err)
			}
			l.Payload = payload
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate coverage audit rows: %w", err)
	}
	return logs, nil
}
