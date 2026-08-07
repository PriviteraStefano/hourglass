package audit

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog is the general append-only audit row (D-05) addressed by
// (entity_type, entity_id) and scoped by org. It generalizes the entry-side
// time_entry.AuditLog: any entity (tickets, activities, ...) writes events
// here. The table is append-only — no UPDATE/DELETE paths exist (T-11-05..
// T-11-08). Writes are synchronous: the caller awaits the insert and the
// error propagates (Pitfall 2 — never fire-and-forget).
type AuditLog struct {
	ID         uuid.UUID      `json:"id"`
	OrgID      uuid.UUID      `json:"org_id"`
	EntityType string         `json:"entity_type"`
	EntityID   uuid.UUID      `json:"entity_id"`
	Action     string         `json:"action"`
	ActorID    *uuid.UUID     `json:"actor_id,omitempty"` // nil = system-initiated event
	Comment    string         `json:"comment,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"` // event-specific data (JSONB)
	CreatedAt  time.Time      `json:"created_at"`
}
