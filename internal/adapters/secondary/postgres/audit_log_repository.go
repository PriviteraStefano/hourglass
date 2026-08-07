package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	auditdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// GeneralAuditLogRepository appends rows to the general audit_logs table
// (schema 017, D-05). Create is synchronous — the caller awaits the insert
// and the error propagates (T-11-08, Pitfall 2). The table is append-only:
// no update/delete methods exist on this type.
//
// The type is named General* because postgres.AuditLogRepository already
// exists for the entry-scoped audit written to time_entry_approvals
// (BE-012 legacy behavior, untouched).
type GeneralAuditLogRepository struct {
	pool *pgxpool.Pool
}

// Compile-time assertion: GeneralAuditLogRepository implements the port.
var _ ports.GeneralAuditLogRepository = (*GeneralAuditLogRepository)(nil)

func NewGeneralAuditLogRepository(pool *pgxpool.Pool) *GeneralAuditLogRepository {
	return &GeneralAuditLogRepository{pool: pool}
}

// Create appends one audit_logs row. actor_id/comment/payload are nullable:
// nil actor and empty comment/payload are written as SQL NULL. All values
// are parameterized (ADR-BE-003, T-11-09).
func (r *GeneralAuditLogRepository) Create(ctx context.Context, log *auditdomain.AuditLog) error {
	id := uuid.New()

	var payload any
	if len(log.Payload) > 0 {
		payloadJSON, err := json.Marshal(log.Payload)
		if err != nil {
			return fmt.Errorf("marshal audit payload: %w", err)
		}
		payload = payloadJSON
	}

	var comment any
	if log.Comment != "" {
		comment = log.Comment
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_logs (id, org_id, entity_type, entity_id, action, actor_id, comment, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, log.OrgID, log.EntityType, log.EntityID, log.Action, log.ActorID, comment, payload, log.CreatedAt)
	return wrapPGError(err, "create audit log")
}
