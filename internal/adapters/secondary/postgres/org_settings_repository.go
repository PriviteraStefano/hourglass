package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// OrgSettingsRepository implements ports.OrgSettingsRepository using a
// pgxpool (schema 022): generic org policy key/value storage over
// org_settings(org_id, key, value JSONB, updated_at, PK(org_id, key))
// (D-13-18). Vocabulary validation lives in the orgsettings domain
// (ValidateKey per known key — CHECK on JSONB isn't feasible); this repo is
// a faithful store.
//
// Upsert is value-replacement (ON CONFLICT) with its 'settings-updated'
// audit row written IN THE SAME TRANSACTION (D-13-22, BE-016 Pitfall 2):
// a failed upsert rolls back the audit — no silent unlogged change
// (T-13-12).
type OrgSettingsRepository struct {
	pool *pgxpool.Pool
}

// Compile-time assertion: OrgSettingsRepository implements the port.
var _ ports.OrgSettingsRepository = (*OrgSettingsRepository)(nil)

func NewOrgSettingsRepository(pool *pgxpool.Pool) *OrgSettingsRepository {
	return &OrgSettingsRepository{pool: pool}
}

// Get returns the raw JSON value for a key, org-scoped. (nil, nil) when the
// key is absent — absence is not an error; the service applies code-level
// defaults (e.g. orgsettings.DefaultDailyHours).
func (r *OrgSettingsRepository) Get(ctx context.Context, orgID uuid.UUID, key string) (json.RawMessage, error) {
	var value []byte
	err := r.pool.QueryRow(ctx,
		`SELECT value FROM org_settings WHERE org_id = $1 AND key = $2`,
		orgID, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, wrapPGError(err, "get org setting")
	}
	return json.RawMessage(value), nil
}

// List returns every stored key/value pair for the org (empty map when none
// are stored — absent keys are covered by code-level defaults service-side).
func (r *OrgSettingsRepository) List(ctx context.Context, orgID uuid.UUID) (map[string]json.RawMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT key, value FROM org_settings WHERE org_id = $1`, orgID)
	if err != nil {
		return nil, wrapPGError(err, "list org settings")
	}
	defer rows.Close()

	out := make(map[string]json.RawMessage)
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, wrapPGError(err, "scan org setting")
		}
		out[key] = json.RawMessage(value)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapPGError(err, "list org settings")
	}
	return out, nil
}

// Upsert stores (or replaces) one key's value with the 'settings-updated'
// audit row written in the same transaction (D-13-22). The value is written
// as JSONB verbatim; validation is the service's job (D-13-18).
func (r *OrgSettingsRepository) Upsert(ctx context.Context, orgID uuid.UUID, key string, value json.RawMessage, auditLog *audit.AuditLog) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin upsert org setting: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
		`INSERT INTO org_settings (org_id, key, value, updated_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (org_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
		orgID, key, value, time.Now().UTC())
	if err != nil {
		return wrapPGError(err, "upsert org setting")
	}

	if auditLog != nil {
		if err := insertOrgSettingsAudit(ctx, tx, auditLog); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert org setting: %w", err)
	}
	return nil
}

// insertOrgSettingsAudit writes one audit_logs row inside the given
// transaction (BE-016 — mirrors insertCoverageAudit, coverage_repository.go):
// entity_type is written from log.EntityType so the caller controls the
// vocabulary (the org_settings plane pins orgsettings.AuditEntityOrgSettings
// with entity_id = the ORG id — audit_logs.entity_id is UUID NOT NULL);
// payload JSONB marshaled from the audit's Payload map ({key, before, after},
// D-13-22); nil actor/empty comment written as SQL NULL. Never
// fire-and-forget: the caller's tx rollback discards this row with the
// failed upsert (T-13-12).
func insertOrgSettingsAudit(ctx context.Context, tx pgx.Tx, log *audit.AuditLog) error {
	id := uuid.New()

	var payload any
	if len(log.Payload) > 0 {
		payloadJSON, err := json.Marshal(log.Payload)
		if err != nil {
			return fmt.Errorf("marshal org settings audit payload: %w", err)
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
		return wrapPGError(err, "insert org settings audit log")
	}
	return nil
}
