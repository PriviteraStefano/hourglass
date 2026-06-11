package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// TimeEntryRepository implements ports.TimeEntryRepository using a pgxpool.
type TimeEntryRepository struct {
	pool *pgxpool.Pool
}

// NewTimeEntryRepository creates a new TimeEntryRepository.
func NewTimeEntryRepository(pool *pgxpool.Pool) *TimeEntryRepository {
	return &TimeEntryRepository{pool: pool}
}

const timeEntrySelectColumns = `id, org_id, user_id, project_id, subproject_id, wg_id, unit_id, hours, description, entry_date, status, COALESCE(current_approver_role, ''), submitted_at, is_deleted, created_from_entry_id, created_at, updated_at`

// timeEntryRowScanner is satisfied by pgx.Row and pgx.Rows.
type timeEntryRowScanner interface {
	Scan(dest ...any) error
}

// scanTimeEntry scans a single time entry row.
func scanTimeEntry(s timeEntryRowScanner) (*time_entry.TimeEntry, error) {
	var e time_entry.TimeEntry
	var currentApproverRole string
	var submittedAt *time.Time
	err := s.Scan(
		&e.ID, &e.OrgID, &e.UserID, &e.ProjectID, &e.SubprojectID,
		&e.WGID, &e.UnitID, &e.Hours, &e.Description, &e.EntryDate,
		&e.Status, &currentApproverRole, &submittedAt,
		&e.IsDeleted, &e.CreatedFromEntryID, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if currentApproverRole != "" {
		e.CurrentApproverRole = &currentApproverRole
	}
	e.SubmittedAt = submittedAt
	return &e, nil
}

// scanTimeEntries scans multiple time entry rows.
func scanTimeEntries(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]time_entry.TimeEntry, error) {
	var entries []time_entry.TimeEntry
	for rows.Next() {
		e, err := scanTimeEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan time entry: %w", err)
		}
		entries = append(entries, *e)
	}
	if entries == nil {
		entries = []time_entry.TimeEntry{}
	}
	return entries, rows.Err()
}

// buildTimeEntryListQuery builds a dynamic SELECT query with numbered placeholders.
// orgID is always $1, is_deleted is always $2 (from filters.IsDeleted).
func buildTimeEntryListQuery(orgID uuid.UUID, filters ports.ListFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	// $1 = org_id
	args = append(args, orgID)
	conditions = append(conditions, "org_id = $1")

	// $2 = is_deleted
	args = append(args, filters.IsDeleted)
	conditions = append(conditions, fmt.Sprintf("is_deleted = $%d", len(args)))

	if filters.Date != "" {
		args = append(args, filters.Date)
		conditions = append(conditions, fmt.Sprintf("entry_date::date = $%d", len(args)))
	}
	if filters.Month != "" {
		args = append(args, filters.Month)
		conditions = append(conditions, fmt.Sprintf("EXTRACT(MONTH FROM entry_date) = $%d", len(args)))
	}
	if filters.Year != "" {
		args = append(args, filters.Year)
		conditions = append(conditions, fmt.Sprintf("EXTRACT(YEAR FROM entry_date) = $%d", len(args)))
	}
	if filters.Status != "" {
		args = append(args, filters.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if filters.WGID != "" {
		wgID, err := uuid.Parse(filters.WGID)
		if err == nil {
			args = append(args, wgID)
			conditions = append(conditions, fmt.Sprintf("wg_id = $%d", len(args)))
		}
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

	query := `SELECT ` + timeEntrySelectColumns + ` FROM time_entries WHERE ` + strings.Join(conditions, " AND ")
	query += ` ORDER BY entry_date DESC`
	return query, args
}

// List returns time entries for an org filtered by the given filters.
func (r *TimeEntryRepository) List(ctx context.Context, orgID uuid.UUID, filters ports.ListFilters) ([]time_entry.TimeEntry, error) {
	query, args := buildTimeEntryListQuery(orgID, filters)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list time entries: %w", err)
	}
	defer rows.Close()
	return scanTimeEntries(rows)
}

// GetByID returns a single time entry by ID, or ErrTimeEntryNotFound.
func (r *TimeEntryRepository) GetByID(ctx context.Context, id uuid.UUID) (*time_entry.TimeEntry, error) {
	query := `SELECT ` + timeEntrySelectColumns + ` FROM time_entries WHERE id = $1`
	e, err := scanTimeEntry(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, time_entry.ErrTimeEntryNotFound
		}
		return nil, fmt.Errorf("get time entry by id: %w", err)
	}
	return e, nil
}

// Create inserts a new time entry and returns the created record.
func (r *TimeEntryRepository) Create(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error) {
	e.ID = uuid.New()
	now := time.Now().UTC()

	query := `INSERT INTO time_entries (id, org_id, user_id, project_id, subproject_id, wg_id, unit_id,
		hours, description, entry_date, status, current_approver_role, submitted_at,
		is_deleted, created_from_entry_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING ` + timeEntrySelectColumns

	created, err := scanTimeEntry(r.pool.QueryRow(ctx, query,
		e.ID, e.OrgID, e.UserID, e.ProjectID, e.SubprojectID,
		e.WGID, e.UnitID, e.Hours, e.Description, e.EntryDate,
		e.Status, e.CurrentApproverRole, e.SubmittedAt,
		e.IsDeleted, e.CreatedFromEntryID, now, now,
	))
	if err != nil {
		return nil, wrapPGError(err, "create time entry")
	}
	return created, nil
}

// Update performs a full-field update on a time entry and returns the updated record.
func (r *TimeEntryRepository) Update(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error) {
	query := `UPDATE time_entries SET
		project_id = $2, subproject_id = $3, wg_id = $4, unit_id = $5,
		hours = $6, description = $7, entry_date = $8, status = $9,
		current_approver_role = $10, submitted_at = $11,
		created_from_entry_id = $12, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + timeEntrySelectColumns

	updated, err := scanTimeEntry(r.pool.QueryRow(ctx, query,
		e.ID, e.ProjectID, e.SubprojectID, e.WGID, e.UnitID,
		e.Hours, e.Description, e.EntryDate, e.Status,
		e.CurrentApproverRole, e.SubmittedAt, e.CreatedFromEntryID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, time_entry.ErrTimeEntryNotFound
		}
		return nil, wrapPGError(err, "update time entry")
	}
	return updated, nil
}

// Delete performs a soft delete on a time entry.
func (r *TimeEntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE time_entries SET is_deleted = true, updated_at = NOW() WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return wrapPGError(err, "delete time entry")
	}
	if cmd.RowsAffected() == 0 {
		return time_entry.ErrTimeEntryNotFound
	}
	return nil
}

// IsPeriodLocked checks if a period is locked for the given org, project, and entry date.
func (r *TimeEntryRepository) IsPeriodLocked(ctx context.Context, orgID, projectID uuid.UUID, entryDate string) (bool, error) {
	query := `SELECT EXISTS(
		SELECT 1 FROM financial_cutoff_periods
		WHERE org_id = $1 AND project_id = $2 AND $3::timestamptz BETWEEN period_start AND period_end AND is_locked = true
	)`
	var locked bool
	err := r.pool.QueryRow(ctx, query, orgID, projectID, entryDate).Scan(&locked)
	if err != nil {
		return false, fmt.Errorf("check period locked: %w", err)
	}
	return locked, nil
}

// ListPending returns pending time entries for an org, role-differentiated.
func (r *TimeEntryRepository) ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]time_entry.TimeEntry, error) {
	var query string
	var args []interface{}

	switch role {
	case "manager":
		uID, err := uuid.Parse(userID)
		if err != nil {
			return nil, fmt.Errorf("parse user_id: %w", err)
		}
		query = `SELECT ` + timeEntrySelectColumns + ` FROM time_entries
			WHERE org_id = $1 AND status IN ('submitted', 'pending_manager') AND is_deleted = false
			AND wg_id IN (SELECT id FROM working_groups WHERE manager_id = $2 OR $2 = ANY(delegate_ids))`
		args = []interface{}{orgID, uID}
	case "finance":
		query = `SELECT ` + timeEntrySelectColumns + ` FROM time_entries
			WHERE org_id = $1 AND status = 'pending_finance' AND is_deleted = false`
		args = []interface{}{orgID}
	default:
		query = `SELECT ` + timeEntrySelectColumns + ` FROM time_entries
			WHERE org_id = $1 AND status = 'submitted' AND is_deleted = false`
		args = []interface{}{orgID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list pending time entries: %w", err)
	}
	defer rows.Close()
	return scanTimeEntries(rows)
}

// AuditLogRepository implements ports.AuditLogRepository using a pgxpool.
type AuditLogRepository struct {
	pool *pgxpool.Pool
}

// NewAuditLogRepository creates a new AuditLogRepository.
func NewAuditLogRepository(pool *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{pool: pool}
}

var _ ports.TimeEntryApprovalRepository = (*TimeEntryRepository)(nil)

// CreateApproval inserts a new approval record into time_entry_approvals.
func (r *TimeEntryRepository) CreateApproval(ctx context.Context, a *time_entry.Approval) error {
	query := `INSERT INTO time_entry_approvals (id, time_entry_id, user_id, action, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(ctx, query,
		a.ID, a.EntryID, a.ActorUserID, a.Action, a.Comment, a.CreatedAt)
	return wrapPGError(err, "create time entry approval")
}

// Create inserts a new audit log entry into time_entry_approvals.
func (r *AuditLogRepository) Create(ctx context.Context, log *time_entry.AuditLog) error {
	id := uuid.New()

	entryID, err := uuid.Parse(log.EntryID)
	if err != nil {
		return fmt.Errorf("parse entry id from audit log: %w", err)
	}

	changesJSON, err := json.Marshal(log.Changes)
	if err != nil {
		return fmt.Errorf("marshal audit log changes: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO time_entry_approvals (id, time_entry_id, user_id, action, comment, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, entryID, log.ActorID, log.Action, string(changesJSON), log.Timestamp)
	return wrapPGError(err, "create audit log")
}
