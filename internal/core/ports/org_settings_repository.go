package ports

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
)

// OrgSettingsRepository is the org-level planning policy persistence
// surface (D-13-18..22, ADR-BE-018 §4/§7): generic key/value JSONB storage
// over org_settings(org_id, key, value, updated_at, PK(org_id, key)).
//
// Vocabulary validation lives in the domain (orgsettings.ValidateKey per
// known key — CHECK on JSONB isn't feasible, D-13-18); this port is a
// faithful store. Upsert is value-replacement semantics (ON CONFLICT) with
// its settings-updated audit row written IN THE SAME TRANSACTION (D-13-22,
// BE-016 Pitfall 2): the caller passes the audit row to write — entity
// type orgsettings.AuditEntityOrgSettings, entity_id = the ORG id
// (audit_logs.entity_id is UUID NOT NULL), payload {key, before, after}.
type OrgSettingsRepository interface {
	// Get returns the raw JSON value for a key, org-scoped. Returns
	// (nil, nil) when the key is absent — absence is not an error; the
	// service applies code-level defaults (e.g. DefaultDailyHours).
	Get(ctx context.Context, orgID uuid.UUID, key string) (json.RawMessage, error)

	// List returns every stored key/value pair for the org (empty map
	// when none are stored — absent keys are covered by code-level
	// defaults service-side).
	List(ctx context.Context, orgID uuid.UUID) (map[string]json.RawMessage, error)

	// Upsert stores (or replaces) one key's value with the settings-
	// updated audit row written in the same transaction (D-13-22).
	Upsert(ctx context.Context, orgID uuid.UUID, key string, value json.RawMessage, audit *audit.AuditLog) error
}
