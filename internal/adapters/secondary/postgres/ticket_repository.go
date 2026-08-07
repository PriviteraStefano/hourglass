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
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// TicketRepository implements ports.TicketRepository using a pgxpool
// (schema 014 + 017). State writes and their audit_logs rows commit in the
// SAME transaction (Pitfall 2, ADR-BE-016): a ticket operation fails as a
// whole if the audit row cannot be written — the event stream is the
// guarantee (OQ4).
type TicketRepository struct {
	pool *pgxpool.Pool
}

// Compile-time assertion: TicketRepository implements the port.
var _ ports.TicketRepository = (*TicketRepository)(nil)

func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{pool: pool}
}

// ticketColumns is the canonical SELECT column list for tickets rows.
const ticketColumns = `id, org_id, title, description, kind, status, requester_id,
	assignee_id, dismissed_hours, created_at, updated_at`

// scanTicketRow scans a pgx.Row into a Ticket, normalizing nullable columns.
func scanTicketRow(row pgx.Row) (*ticketdomain.Ticket, error) {
	var t ticketdomain.Ticket
	var assigneeID *uuid.UUID
	var dismissedHours *float64
	err := row.Scan(&t.ID, &t.OrgID, &t.Title, &t.Description, &t.Kind, &t.Status,
		&t.RequesterID, &assigneeID, &dismissedHours, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.AssigneeID = assigneeID
	t.DismissedHours = dismissedHours
	return &t, nil
}

// Get returns a single ticket scoped to the org. A missing ticket — or a
// ticket that exists in another org — surfaces as ticket.ErrTicketNotFound
// (cross-org id fails the same-org precondition, D-02).
func (r *TicketRepository) Get(ctx context.Context, orgID, ticketID uuid.UUID) (*ticketdomain.Ticket, error) {
	t, err := scanTicketRow(r.pool.QueryRow(ctx,
		`SELECT `+ticketColumns+` FROM tickets WHERE id = $1 AND org_id = $2`,
		ticketID, orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ticketdomain.ErrTicketNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get ticket: %w", err)
	}
	return t, nil
}

// insertTicketAudit writes one audit_logs row inside the given transaction.
// entity_type is always 'ticket' (ADR-BE-016); payload JSONB marshaled from
// the audit's Payload map; nil actor/empty comment written as SQL NULL.
// The row id is generated here — the AuditLog's ID field is not persisted.
func insertTicketAudit(ctx context.Context, tx pgx.Tx, log *audit.AuditLog) error {
	id := uuid.New()

	var payload any
	if len(log.Payload) > 0 {
		payloadJSON, err := json.Marshal(log.Payload)
		if err != nil {
			return fmt.Errorf("marshal ticket audit payload: %w", err)
		}
		payload = payloadJSON
	}

	var comment any
	if log.Comment != "" {
		comment = log.Comment
	}

	_, err := tx.Exec(ctx,
		`INSERT INTO audit_logs (id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)
		 VALUES ($1, $2, 'ticket', $3, $4, $5, $6, $7, $8)`,
		id, log.OrgID, log.EntityID, log.Action, log.ActorID, comment, payload, log.CreatedAt)
	if err != nil {
		return wrapPGError(err, "insert ticket audit log")
	}
	return nil
}

// Create inserts the ticket and its 'created' audit row atomically. The
// ticket's status is honored as given (the service pins 'open').
func (r *TicketRepository) Create(ctx context.Context, orgID uuid.UUID, t *ticketdomain.Ticket, auditLog *audit.AuditLog) (*ticketdomain.Ticket, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin ticket create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()
	_, err = tx.Exec(ctx,
		`INSERT INTO tickets (id, org_id, title, description, kind, status, requester_id, assignee_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`,
		t.ID, orgID, t.Title, t.Description, t.Kind, t.Status, t.RequesterID, t.AssigneeID, now)
	if err != nil {
		return nil, wrapPGError(err, "create ticket")
	}

	if auditLog != nil {
		if err := insertTicketAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ticket create: %w", err)
	}
	return r.Get(ctx, orgID, t.ID)
}

// ListByOrg returns the org's tickets, optionally filtered by status and/or
// kind, newest first. Filters are validated by the service against the closed
// vocabularies (TICK-01/TICK-02).
func (r *TicketRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, status, kind string) ([]ticketdomain.Ticket, error) {
	query := `SELECT ` + ticketColumns + ` FROM tickets WHERE org_id = $1`
	args := []any{orgID}
	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if kind != "" {
		args = append(args, kind)
		query += fmt.Sprintf(" AND kind = $%d", len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, wrapPGError(err, "list tickets")
	}
	defer rows.Close()

	var tickets []ticketdomain.Ticket
	for rows.Next() {
		t, err := scanTicketRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ticket row: %w", err)
		}
		tickets = append(tickets, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ticket rows: %w", err)
	}
	return tickets, nil
}

// UpdateDetails applies the non-nil field branches (title/description/
// assignee_id), bumps updated_at, and writes the 'updated' audit row in the
// same tx. A missing/cross-org ticket surfaces ErrTicketNotFound.
func (r *TicketRepository) UpdateDetails(ctx context.Context, orgID, ticketID uuid.UUID, title, description *string, assigneeID *uuid.UUID, auditLog *audit.AuditLog) (*ticketdomain.Ticket, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin ticket update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sets := []string{"updated_at = $1"}
	args := []any{time.Now().UTC()}
	if title != nil {
		args = append(args, *title)
		sets = append(sets, fmt.Sprintf("title = $%d", len(args)))
	}
	if description != nil {
		args = append(args, *description)
		sets = append(sets, fmt.Sprintf("description = $%d", len(args)))
	}
	if assigneeID != nil {
		args = append(args, *assigneeID)
		sets = append(sets, fmt.Sprintf("assignee_id = $%d", len(args)))
	}
	args = append(args, orgID, ticketID)
	query := fmt.Sprintf("UPDATE tickets SET %s WHERE id = $%d AND org_id = $%d",
		strings.Join(sets, ", "), len(args)-1, len(args))

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, wrapPGError(err, "update ticket details")
	}
	if ct.RowsAffected() == 0 {
		return nil, ticketdomain.ErrTicketNotFound
	}

	if auditLog != nil {
		if err := insertTicketAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ticket update: %w", err)
	}
	return r.Get(ctx, orgID, ticketID)
}

// UpdateState sets the ticket status, bumps updated_at, and writes the
// 'status_changed' audit row in the same tx (TICK-05 — the transition is not
// durable without its event).
func (r *TicketRepository) UpdateState(ctx context.Context, orgID, ticketID uuid.UUID, to string, note *string, auditLog *audit.AuditLog) (*ticketdomain.Ticket, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin ticket state update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx,
		`UPDATE tickets SET status = $1, updated_at = $2 WHERE id = $3 AND org_id = $4`,
		to, time.Now().UTC(), ticketID, orgID)
	if err != nil {
		return nil, wrapPGError(err, "update ticket status")
	}
	if ct.RowsAffected() == 0 {
		return nil, ticketdomain.ErrTicketNotFound
	}

	if auditLog != nil {
		if err := insertTicketAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ticket state update: %w", err)
	}
	return r.Get(ctx, orgID, ticketID)
}

// Dismiss sets status='dismissed' + dismissed_hours (TICK-04, D-13 raw Σ)
// and writes the 'dismissed' audit row in the same tx. The hours value is
// computed server-side by the service via LoggedHours — never client-supplied
// (T-11-07).
func (r *TicketRepository) Dismiss(ctx context.Context, orgID, ticketID uuid.UUID, hours float64, auditLog *audit.AuditLog) (*ticketdomain.Ticket, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin ticket dismissal: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx,
		`UPDATE tickets SET status = 'dismissed', dismissed_hours = $1, updated_at = $2
		 WHERE id = $3 AND org_id = $4`,
		hours, time.Now().UTC(), ticketID, orgID)
	if err != nil {
		return nil, wrapPGError(err, "dismiss ticket")
	}
	if ct.RowsAffected() == 0 {
		return nil, ticketdomain.ErrTicketNotFound
	}

	if auditLog != nil {
		if err := insertTicketAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ticket dismissal: %w", err)
	}
	return r.Get(ctx, orgID, ticketID)
}

// Triage atomically reclassifies the ticket (kind update + status='planned')
// and creates 1..N activities with origin_type='customer_ticket' +
// ticket_id, all in ONE transaction (D-10, TICK-03, T-11-06).
//
// Validation is AUTHORITATIVE inside this tx (Pitfall 7, ADR-BE-016): every
// plan's kind must exist in the org's activity_kinds catalog, its parent must
// be an in-org activity, and its contract must be an in-org contract — each
// checked via SELECT EXISTS against the open tx, so there is no TOCTOU
// window. Any miss returns ticket.ErrInvalidRequest and rolls back the whole
// triage (no partial writes). The DB FK/CHECK constraints backstop as a
// third line.
//
// Both audit rows ('triaged' + 'activities_created') are written in the same
// tx as the state write and the activity inserts.
func (r *TicketRepository) Triage(ctx context.Context, orgID, ticketID uuid.UUID, kind *string, plans []*activitydomain.CreateActivityRequest, audits []*audit.AuditLog) (*ticketdomain.Ticket, []*activitydomain.ActivityResponse, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin ticket triage: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the ticket row in-org so concurrent state changes serialize.
	var currentStatus string
	err = tx.QueryRow(ctx,
		`SELECT status FROM tickets WHERE id = $1 AND org_id = $2 FOR UPDATE`,
		ticketID, orgID).Scan(&currentStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ticketdomain.ErrTicketNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("lock ticket for triage: %w", err)
	}

	// Authoritative in-tx plan validation (Pitfall 7) — no TOCTOU window.
	for i, p := range plans {
		var kindOK bool
		err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM activity_kinds WHERE org_id = $1 AND name = $2)`,
			orgID, string(p.Kind)).Scan(&kindOK)
		if err != nil {
			return nil, nil, fmt.Errorf("triage validate kind %d: %w", i, err)
		}
		if !kindOK {
			return nil, nil, ticketdomain.ErrInvalidRequest
		}

		if p.ParentID != nil {
			var parentOK bool
			err := tx.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM activities WHERE id = $1 AND org_id = $2)`,
				*p.ParentID, orgID).Scan(&parentOK)
			if err != nil {
				return nil, nil, fmt.Errorf("triage validate parent %d: %w", i, err)
			}
			if !parentOK {
				return nil, nil, ticketdomain.ErrInvalidRequest
			}
		}

		if p.ContractID != nil {
			var contractOK bool
			err := tx.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM contracts WHERE id = $1 AND org_id = $2)`,
				*p.ContractID, orgID).Scan(&contractOK)
			if err != nil {
				return nil, nil, fmt.Errorf("triage validate contract %d: %w", i, err)
			}
			if !contractOK {
				return nil, nil, ticketdomain.ErrInvalidRequest
			}
		}
	}

	// Ticket reclassification: optional kind override + status='planned'.
	sets := []string{"status = 'planned'", "updated_at = $1"}
	args := []any{time.Now().UTC()}
	if kind != nil {
		args = append(args, *kind)
		sets = append(sets, fmt.Sprintf("kind = $%d", len(args)))
	}
	args = append(args, ticketID, orgID)
	query := fmt.Sprintf("UPDATE tickets SET %s WHERE id = $%d AND org_id = $%d",
		strings.Join(sets, ", "), len(args)-1, len(args))
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return nil, nil, wrapPGError(err, "update ticket on triage")
	}

	// 1..N activity inserts with the customer_ticket origin (D-10).
	now := time.Now().UTC()
	created := make([]*activitydomain.ActivityResponse, 0, len(plans))
	for _, p := range plans {
		var id, orgIDOut, ticketIDOut uuid.UUID
		var parentIDOut, contractIDOut *uuid.UUID
		var nameOut, descOut, kindOut, originTypeOut, governanceOut string
		var budgetOut *float64
		var billableOut *bool
		var isSharedOut, isActiveOut bool
		var createdAtOut, updatedAtOut time.Time
		err := tx.QueryRow(ctx,
			`INSERT INTO activities (id, org_id, parent_id, name, description, kind,
				contract_id, governance_model, created_by_org_id, is_shared, billable, budget_amount,
				origin_type, assigned_by, assigned_to, proposed_by, reviewed_by, ticket_id,
				is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NULL, NULL, NULL, NULL, $14, true, $15, $15)
			 RETURNING id, org_id, parent_id, name, description, kind, contract_id, governance_model,
				created_by_org_id, is_shared, billable, budget_amount, origin_type, ticket_id,
				is_active, created_at, updated_at`,
			uuid.New(), orgID, p.ParentID, p.Name, p.Description, string(p.Kind),
			p.ContractID, p.GovernanceModel, orgID, p.IsShared, p.Billable, p.BudgetAmount,
			activitydomain.OriginTypeCustomerTicket, ticketID, now,
		).Scan(
			&id, &orgIDOut, &parentIDOut, &nameOut, &descOut, &kindOut,
			&contractIDOut, &governanceOut, &orgIDOut, &isSharedOut, &billableOut, &budgetOut,
			&originTypeOut, &ticketIDOut, &isActiveOut, &createdAtOut, &updatedAtOut,
		)
		if err != nil {
			return nil, nil, wrapPGError(err, "triage insert activity")
		}
		created = append(created, &activitydomain.ActivityResponse{
			Activity: activitydomain.Activity{
				ID:              id,
				OrgID:           orgIDOut,
				ParentID:        parentIDOut,
				Name:            nameOut,
				Description:     descOut,
				Kind:            activitydomain.ActivityKind(kindOut),
				ContractID:      contractIDOut,
				GovernanceModel: models.GovernanceModel(governanceOut),
				CreatedByOrgID:  orgIDOut,
				IsShared:        isSharedOut,
				Billable:        billableOut,
				BudgetAmount:    budgetOut,
				IsActive:        isActiveOut,
				CreatedAt:       createdAtOut,
				UpdatedAt:       updatedAtOut,
				OriginType:      &originTypeOut,
				TicketID:        &ticketIDOut,
			},
		})
	}

	// Both audit rows in the same tx (TICK-03).
	for _, a := range audits {
		if err := insertTicketAudit(ctx, tx, a); err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit ticket triage: %w", err)
	}

	t, err := r.Get(ctx, orgID, ticketID)
	if err != nil {
		return nil, nil, err
	}
	return t, created, nil
}

// AddComment inserts a first-class comment row (D-06) and its
// 'comment_added' audit row in the same tx (TICK-05). The ticket must exist
// in-org (the FK on ticket_id backstops).
func (r *TicketRepository) AddComment(ctx context.Context, orgID, ticketID uuid.UUID, c *ticketdomain.TicketComment, auditLog *audit.AuditLog) (*ticketdomain.TicketComment, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin ticket comment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM tickets WHERE id = $1 AND org_id = $2)`,
		ticketID, orgID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check ticket for comment: %w", err)
	}
	if !exists {
		return nil, ticketdomain.ErrTicketNotFound
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO ticket_comments (id, ticket_id, author_id, body, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		c.ID, ticketID, c.AuthorID, c.Body, c.CreatedAt)
	if err != nil {
		return nil, wrapPGError(err, "insert ticket comment")
	}

	if auditLog != nil {
		if err := insertTicketAudit(ctx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ticket comment: %w", err)
	}
	return c, nil
}

// ListComments returns the ticket's comments (org-scoped via the ticket
// join), oldest first. Read-only — no update/delete paths (TICK-05).
func (r *TicketRepository) ListComments(ctx context.Context, orgID, ticketID uuid.UUID) ([]ticketdomain.TicketComment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.ticket_id, c.author_id, c.body, c.created_at
		 FROM ticket_comments c
		 JOIN tickets t ON t.id = c.ticket_id
		 WHERE t.id = $1 AND t.org_id = $2
		 ORDER BY c.created_at`,
		ticketID, orgID)
	if err != nil {
		return nil, wrapPGError(err, "list ticket comments")
	}
	defer rows.Close()

	var comments []ticketdomain.TicketComment
	for rows.Next() {
		var c ticketdomain.TicketComment
		if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.Body, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan ticket comment: %w", err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ticket comments: %w", err)
	}
	return comments, nil
}

// ListHistory returns the ticket's append-only audit stream (TICK-05),
// ordered by created_at. The payload JSONB round-trips into a map.
func (r *TicketRepository) ListHistory(ctx context.Context, orgID, ticketID uuid.UUID) ([]audit.AuditLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at
		 FROM audit_logs
		 WHERE entity_type = 'ticket' AND entity_id = $1 AND org_id = $2
		 ORDER BY created_at`,
		ticketID, orgID)
	if err != nil {
		return nil, wrapPGError(err, "list ticket history")
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
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		l.ActorID = actorID
		if comment != nil {
			l.Comment = *comment
		}
		if len(payloadJSON) > 0 {
			var payload map[string]any
			if err := json.Unmarshal(payloadJSON, &payload); err != nil {
				return nil, fmt.Errorf("unmarshal audit payload: %w", err)
			}
			l.Payload = payload
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit rows: %w", err)
	}
	return logs, nil
}

// LoggedHours computes the raw Σ of logged hours across the ticket's linked
// activities (D-13): submitted + approved, is_deleted excluded, linked via
// ticket_id + origin_type='customer_ticket'. Stable signature — Phase 12
// swaps the computation to net-of-compensations without changing it (D-13).
func (r *TicketRepository) LoggedHours(ctx context.Context, ticketID uuid.UUID) (float64, error) {
	var hours float64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(hours),0) FROM time_entries
		 WHERE is_deleted = false AND status IN ('submitted','approved')
		   AND activity_id IN (SELECT id FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket')`,
		ticketID).Scan(&hours)
	if err != nil {
		return 0, wrapPGError(err, "compute ticket logged hours")
	}
	return hours, nil
}

// HasNonTerminalActivities reports whether the ticket's linked-activity
// subtree carries any non-terminal time entry (OQ2): draft/submitted/
// pending_manager/pending_finance, is_deleted=false, on the ticket's linked
// activities OR any of their descendants (recursive CTE). Blocks the
// 'resolved' transition until the subtree is terminal.
func (r *TicketRepository) HasNonTerminalActivities(ctx context.Context, ticketID uuid.UUID) (bool, error) {
	var has bool
	err := r.pool.QueryRow(ctx,
		`WITH RECURSIVE subtree AS (
			SELECT id FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket'
			UNION ALL
			SELECT a.id FROM activities a JOIN subtree s ON a.parent_id = s.id
		 )
		 SELECT EXISTS (
			SELECT 1 FROM time_entries te
			WHERE te.is_deleted = false
			  AND te.status IN ('draft','submitted','pending_manager','pending_finance')
			  AND te.activity_id IN (SELECT id FROM subtree)
		 )`,
		ticketID).Scan(&has)
	if err != nil {
		return false, wrapPGError(err, "check non-terminal activities")
	}
	return has, nil
}
