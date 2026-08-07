package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	auditdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stretchr/testify/require"
)

// TestTicketRepository_Get covers the org-scoped Get with nullable field
// handling (assignee_id NULL) and the cross-org not-found case
// (VALIDATION.md task 11-05-01, TestTicketAudit foundation).
func TestTicketRepository_Get(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)

	ticketID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO tickets (id, org_id, title, description, kind, status, requester_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		ticketID, orgID, "Billing bug", "Amounts are off by one cent", "bug", "open", requesterID, now)
	require.NoError(t, err)

	got, err := repo.Get(context.Background(), orgID, ticketID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, ticketID, got.ID)
	require.Equal(t, orgID, got.OrgID)
	require.Equal(t, "Billing bug", got.Title)
	require.Equal(t, "bug", got.Kind)
	require.Equal(t, "open", got.Status)
	require.Equal(t, requesterID, got.RequesterID)
	require.Nil(t, got.AssigneeID)      // nullable column handled as nil
	require.Nil(t, got.DismissedHours)  // nullable column handled as nil

	// same ticket id from another org → not found (same-org, D-02)
	_, err = repo.Get(context.Background(), uuid.New(), ticketID)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)

	// unknown id → not found
	_, err = repo.Get(context.Background(), orgID, uuid.New())
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)
}

// TestTicketRepository_Get_WithAssignee covers the nullable assignee_id
// populated path.
func TestTicketRepository_Get_WithAssignee(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)
	assigneeID := seedUser(t, pool, now)

	ticketID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO tickets (id, org_id, title, description, kind, status, requester_id, assignee_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`,
		ticketID, orgID, "Change request", "New report column", "change", "triage", requesterID, assigneeID, now)
	require.NoError(t, err)

	got, err := repo.Get(context.Background(), orgID, ticketID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.AssigneeID)
	require.Equal(t, assigneeID, *got.AssigneeID)
}

// TestTicketAudit_AuditLogRoundTrip covers the general audit_logs Create:
// a synchronous insert whose JSONB payload round-trips intact, plus the
// nullable actor path (system-initiated event).
func TestTicketAudit_AuditLogRoundTrip(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewGeneralAuditLogRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	actorID := seedUser(t, pool, now)

	entityID := uuid.New()
	err := repo.Create(context.Background(), &auditdomain.AuditLog{
		OrgID:      orgID,
		EntityType: "activity",
		EntityID:   entityID,
		Action:     "proposal_approved",
		ActorID:    &actorID,
		Comment:    "approved by finance",
		Payload:    map[string]any{"approver": actorID.String()},
		CreatedAt:  now,
	})
	require.NoError(t, err)

	// Read the row back and assert the payload JSONB round-trips as a map.
	var action string
	var comment *string
	var actorIDBack *uuid.UUID
	var payloadJSON []byte
	err = pool.QueryRow(context.Background(),
		`SELECT action, actor_id, comment, payload FROM audit_logs WHERE entity_type = $1 AND entity_id = $2`,
		"activity", entityID).Scan(&action, &actorIDBack, &comment, &payloadJSON)
	require.NoError(t, err)
	require.Equal(t, "proposal_approved", action)
	require.NotNil(t, actorIDBack)
	require.Equal(t, actorID, *actorIDBack)
	require.NotNil(t, comment)
	require.Equal(t, "approved by finance", *comment)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(payloadJSON, &payload))
	require.Equal(t, actorID.String(), payload["approver"])

	// System-initiated event: actor_id NULL, empty payload written as NULL.
	err = repo.Create(context.Background(), &auditdomain.AuditLog{
		OrgID:      orgID,
		EntityType: "ticket",
		EntityID:   entityID,
		Action:     "system_event",
		CreatedAt:  now,
	})
	require.NoError(t, err)

	var actorIDNil *uuid.UUID
	var payloadNil *[]byte
	err = pool.QueryRow(context.Background(),
		`SELECT actor_id, payload FROM audit_logs WHERE entity_type = $1 AND action = $2`,
		"ticket", "system_event").Scan(&actorIDNil, &payloadNil)
	require.NoError(t, err)
	require.Nil(t, actorIDNil)
	require.Nil(t, payloadNil)
}
