package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	auditdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/models"
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

// TestTicketRepository_Create covers the Create round-trip: the ticket row
// lands with requester_id/status, and the 'created' audit row lands with
// entity_type 'ticket' in the SAME tx (TICK-05, ADR-BE-016).
func TestTicketRepository_Create(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)

	tkt := &ticketdomain.Ticket{
		ID:          uuid.New(),
		OrgID:       orgID,
		Title:       "Created via repo",
		Description: "round-trip",
		Kind:        ticketdomain.KindBug,
		Status:      ticketdomain.StatusOpen,
		RequesterID: requesterID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	actor := requesterID
	created, err := repo.Create(context.Background(), orgID, tkt, &auditdomain.AuditLog{
		OrgID:      orgID,
		EntityType: "ticket",
		EntityID:   tkt.ID,
		Action:     "created",
		ActorID:    &actor,
		Payload:    map[string]any{"kind": ticketdomain.KindBug},
		CreatedAt:  now,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, tkt.ID, created.ID)
	require.Equal(t, ticketdomain.StatusOpen, created.Status)
	require.Equal(t, requesterID, created.RequesterID)

	// Audit row exists with entity_type 'ticket' and the same actor.
	var action string
	var actorBack *uuid.UUID
	var payloadJSON []byte
	err = pool.QueryRow(context.Background(),
		`SELECT action, actor_id, payload FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1`,
		tkt.ID).Scan(&action, &actorBack, &payloadJSON)
	require.NoError(t, err)
	require.Equal(t, "created", action)
	require.NotNil(t, actorBack)
	require.Equal(t, requesterID, *actorBack)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(payloadJSON, &payload))
	require.Equal(t, ticketdomain.KindBug, payload["kind"])
}

// TestTicketRepository_UpdateState covers the state write + audit row landing
// in the same tx (TICK-05): the status persists and exactly one
// 'status_changed' row exists afterwards.
func TestTicketRepository_UpdateState(t *testing.T) {
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
		ticketID, orgID, "State test", "", "bug", "open", requesterID, now)
	require.NoError(t, err)

	actor := requesterID
	note := "moving on"
	updated, err := repo.UpdateState(context.Background(), orgID, ticketID, "triage", &note, &auditdomain.AuditLog{
		OrgID:      orgID,
		EntityType: "ticket",
		EntityID:   ticketID,
		Action:     "status_changed",
		ActorID:    &actor,
		Payload:    map[string]any{"from": "open", "to": "triage", "note": note},
		CreatedAt:  now,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "triage", updated.Status)

	// Exactly one audit row for the change, with from/to payload.
	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action = 'status_changed'`,
		ticketID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var payloadJSON []byte
	err = pool.QueryRow(context.Background(),
		`SELECT payload FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action = 'status_changed'`,
		ticketID).Scan(&payloadJSON)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(payloadJSON, &payload))
	require.Equal(t, "open", payload["from"])
	require.Equal(t, "triage", payload["to"])
	require.Equal(t, note, payload["note"])
}

// TestTicketRepository_TriageAuditRows covers the triage tx: status flips to
// 'planned', the activity lands with origin customer_ticket + ticket_id, and
// BOTH audit rows ('triaged' + 'activities_created') exist afterwards.
func TestTicketRepository_TriageAuditRows(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)
	seedActivityKind(t, pool, orgID, "engagement")

	ticketID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO tickets (id, org_id, title, description, kind, status, requester_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 'triage', $6, $7, $7)`,
		ticketID, orgID, "Triage me", "", "bug", requesterID, now)
	require.NoError(t, err)

	kind := ticketdomain.KindBug
	plans := []*activitydomain.CreateActivityRequest{
		{
			Name:            "Investigate",
			Kind:            activitydomain.ActivityKind("engagement"),
			GovernanceModel: models.GovernanceCreatorControlled,
		},
	}
	actor := requesterID
	audits := []*auditdomain.AuditLog{
		{OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "triaged", ActorID: &actor, CreatedAt: now},
		{OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "activities_created", ActorID: &actor, CreatedAt: now},
	}

	updated, activities, err := repo.Triage(context.Background(), orgID, ticketID, &kind, plans, audits)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, ticketdomain.StatusPlanned, updated.Status)
	require.Equal(t, ticketdomain.KindBug, updated.Kind)
	require.Len(t, activities, 1)

	created := activities[0]
	require.NotNil(t, created.OriginType)
	require.Equal(t, activitydomain.OriginTypeCustomerTicket, *created.OriginType)
	require.NotNil(t, created.TicketID)
	require.Equal(t, ticketID, *created.TicketID)

	var triaged, activitiesCreated int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action = 'triaged'`,
		ticketID).Scan(&triaged)
	require.NoError(t, err)
	require.Equal(t, 1, triaged)
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action = 'activities_created'`,
		ticketID).Scan(&activitiesCreated)
	require.NoError(t, err)
	require.Equal(t, 1, activitiesCreated)
}

// TestTicketRepository_NoAuditMutation is a grep-level guard: the repo must
// never UPDATE or DELETE rows in audit_logs / ticket_comments (TICK-05,
// append-only stream). It scans the source file for forbidden statements.
func TestTicketRepository_NoAuditMutation(t *testing.T) {
	src, err := os.ReadFile("ticket_repository.go")
	require.NoError(t, err)
	for _, forbidden := range []string{
		"UPDATE audit_logs",
		"DELETE FROM audit_logs",
		"DELETE FROM ticket_comments",
	} {
		require.NotContains(t, string(src), forbidden, "repo must not mutate the append-only stream")
	}
}
