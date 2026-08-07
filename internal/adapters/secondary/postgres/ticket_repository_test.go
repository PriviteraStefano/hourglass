package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
	require.Nil(t, got.AssigneeID)     // nullable column handled as nil
	require.Nil(t, got.DismissedHours) // nullable column handled as nil

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

// TestTicketRepository_DismissedNote covers the IN-02 derived note: every
// read of a dismissed ticket carries DismissedNote == "dismissed with {N} h
// logged" rendered from dismissed_hours (TICK-04, D-13 raw Σ), while a
// dismissed ticket with NULL hours and a non-dismissed ticket both read back
// with a nil note. The note is derived at scan time — never persisted.
func TestTicketRepository_DismissedNote(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)

	// insertTicket seeds a ticket row with the given status and
	// dismissed_hours (nil means the column stays NULL).
	insertTicket := func(title, status string, hours *float64) uuid.UUID {
		t.Helper()
		id := uuid.New()
		var err error
		if hours != nil {
			_, err = pool.Exec(context.Background(),
				`INSERT INTO tickets (id, org_id, title, description, kind, status, requester_id, dismissed_hours, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`,
				id, orgID, title, "", "bug", status, requesterID, *hours, now)
		} else {
			_, err = pool.Exec(context.Background(),
				`INSERT INTO tickets (id, org_id, title, description, kind, status, requester_id, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
				id, orgID, title, "", "bug", status, requesterID, now)
		}
		require.NoError(t, err)
		return id
	}

	// 1. Dismissed with 5h logged → note "dismissed with 5 h logged"
	//    (FormatFloat precision -1 trims the trailing zeros of DECIMAL 5.00).
	five := 5.0
	dismissedID := insertTicket("Dismissed ticket", ticketdomain.StatusDismissed, &five)
	got, err := repo.Get(context.Background(), orgID, dismissedID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.DismissedNote)
	require.Equal(t, "dismissed with 5 h logged", *got.DismissedNote)

	// 2. Dismissed with NULL dismissed_hours → no note.
	nullID := insertTicket("Dismissed without hours", ticketdomain.StatusDismissed, nil)
	got, err = repo.Get(context.Background(), orgID, nullID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Nil(t, got.DismissedNote)

	// 3. Non-dismissed ('planned') ticket → no note.
	plannedID := insertTicket("Planned ticket", ticketdomain.StatusPlanned, nil)
	got, err = repo.Get(context.Background(), orgID, plannedID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Nil(t, got.DismissedNote)

	// The note also rides the list read (same scanTicketRow funnel).
	all, err := repo.ListByOrg(context.Background(), orgID, "", "")
	require.NoError(t, err)
	var listed *ticketdomain.Ticket
	for i := range all {
		if all[i].ID == dismissedID {
			listed = &all[i]
		}
	}
	require.NotNil(t, listed, "dismissed ticket must appear in the list")
	require.NotNil(t, listed.DismissedNote)
	require.Equal(t, "dismissed with 5 h logged", *listed.DismissedNote)
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

// ---------------------------------------------------------------------------
// CR-01 gap-closure battery (11-07) — concurrency invariants for the ticket
// state machine + dismissal guard (VERIFICATION.md gaps 1+2).
//
// These tests are RED on the pre-fix code (pool-level checks, no FOR UPDATE
// on Dismiss/UpdateState, no status precondition) and GREEN after the in-tx
// re-validation lands. The goroutine races replicate the house pattern from
// TestRefreshTokenRepository_Rotate_ConcurrentRace: a start channel + a
// buffered results channel, asserting the deterministic outcome set — no
// wall-clock timing decides anything, only the FOR UPDATE row lock.
// ---------------------------------------------------------------------------

// insertTicketWithStatus is a raw seeding helper for the battery (mirrors
// TestTicketRepository_TriageAuditRows' seeding shape).
func insertTicketWithStatus(t *testing.T, pool *pgxpool.Pool, orgID, requesterID uuid.UUID, status string, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO tickets (id, org_id, title, description, kind, status, requester_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		id, orgID, "Race ticket", "", "bug", status, requesterID, now)
	require.NoError(t, err)
	return id
}

// insertCustomerTicketActivity seeds a linked activity with the
// customer_ticket origin — the shape Triage produces (D-10, TICK-03).
func insertCustomerTicketActivity(t *testing.T, pool *pgxpool.Pool, orgID, ticketID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO activities (id, org_id, parent_id, name, description, kind,
			governance_model, created_by_org_id, is_shared, is_active,
			origin_type, ticket_id, created_at, updated_at)
		 VALUES ($1, $2, NULL, $3, '', 'engagement', 'creator_controlled', $2, false, true,
			'customer_ticket', $4, $5, $5)`,
		id, orgID, "Linked activity", ticketID, now)
	require.NoError(t, err)
	return id
}

// dismissAudit builds the 'dismissed' audit row the service would pass.
func dismissAudit(orgID, ticketID, actorID uuid.UUID, now time.Time) *auditdomain.AuditLog {
	return &auditdomain.AuditLog{
		OrgID:      orgID,
		EntityType: "ticket",
		EntityID:   ticketID,
		Action:     "dismissed",
		ActorID:    &actorID,
		Payload:    map[string]any{"hours": float64(0)},
		CreatedAt:  now,
	}
}

// TestDismissalGuard_RaceWithPendingSubmit is the deterministic 2-tx pin for
// CR-01 race 3 (TICK-04, T-11-07): the dismissal Σ must be re-computed inside
// the dismiss tx, serialized against the entry-submit path by the linked-
// activity FOR UPDATE lock. An uncommitted submitted time-entry INSERT on a
// linked activity must BLOCK Dismiss; after that entry tx commits, Dismiss
// must observe the Σ and refuse with ErrDismissalBlocked — never commit a
// dismissal with dismissed_hours=0 while committed logged hours exist.
//
// RED on the pre-fix code: the lock-free Dismiss returns success immediately
// while the submit is still in flight (check-then-act bypass).
func TestDismissalGuard_RaceWithPendingSubmit(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	seedActivityKind(t, pool, orgID, "engagement")

	ticketID := insertTicketWithStatus(t, pool, orgID, requesterID, "triage", now)
	activityID := insertCustomerTicketActivity(t, pool, orgID, ticketID, now)

	// Uncommitted submitted entry on the linked activity — the pending submit
	// the guard must serialize against (FK KEY SHARE on the activity row).
	entryTx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = entryTx.Rollback(ctx) }()
	_, err = entryTx.Exec(ctx,
		`INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours, description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 5.0, 'pending submit', $6, 'submitted', false, $6, $6)`,
		uuid.New(), orgID, requesterID, activityID, unitID, now)
	require.NoError(t, err)

	// Dismiss must BLOCK until the entry tx commits — the in-tx Σ serializes
	// behind the linked-activity FOR UPDATE. If it returns while the submit
	// is still pending, the guard has been bypassed (the pre-fix bug).
	dismissDone := make(chan error, 1)
	go func() {
		_, err := repo.Dismiss(ctx, orgID, ticketID, 0, dismissAudit(orgID, ticketID, requesterID, now))
		dismissDone <- err
	}()

	select {
	case err := <-dismissDone:
		t.Fatalf("dismiss returned while a submit was pending (err=%v): the guard must serialize", err)
	case <-time.After(500 * time.Millisecond):
		// blocked as expected — the submit holds the activity lock
	}

	// Commit the submit; Dismiss must now see the Σ and refuse.
	require.NoError(t, entryTx.Commit(ctx))

	select {
	case err := <-dismissDone:
		require.ErrorIs(t, err, ticketdomain.ErrDismissalBlocked)
	case <-time.After(15 * time.Second):
		t.Fatal("dismiss did not complete after the pending submit committed")
	}

	// Ticket unchanged: still 'triage', dismissed_hours NULL, no dismissed audit.
	var status string
	var dismissedHours *float64
	err = pool.QueryRow(ctx,
		`SELECT status, dismissed_hours FROM tickets WHERE id = $1`, ticketID).Scan(&status, &dismissedHours)
	require.NoError(t, err)
	require.Equal(t, "triage", status)
	require.Nil(t, dismissedHours, "dismissal must not commit dismissed_hours when blocked")

	var dismissedAudits int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action = 'dismissed'`,
		ticketID).Scan(&dismissedAudits)
	require.NoError(t, err)
	require.Zero(t, dismissedAudits)
}

// TestDismissalRace_VsTriage fires repo.Triage and repo.Dismiss concurrently
// on a 'triage' ticket and asserts the deterministic outcome set (CR-01 race
// 1 + 2): EXACTLY ONE succeeds and the loser gets ErrInvalidTransition, with
// the final state consistent with the winner — a dismissed (terminal) ticket
// can never be resurrected to 'planned', and 'planned → dismissed' never
// lands.
//
// RED on the pre-fix code: both goroutines succeed today (Triage never
// validates currentStatus; Dismiss writes lock-free).
func TestDismissalRace_VsTriage(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)
	seedActivityKind(t, pool, orgID, "engagement")

	ticketID := insertTicketWithStatus(t, pool, orgID, requesterID, "triage", now)
	actor := requesterID

	kind := ticketdomain.KindBug
	plans := []*activitydomain.CreateActivityRequest{
		{
			Name:            "Race activity",
			Kind:            activitydomain.ActivityKind("engagement"),
			GovernanceModel: models.GovernanceCreatorControlled,
		},
	}
	audits := []*auditdomain.AuditLog{
		{OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "triaged", ActorID: &actor, CreatedAt: now},
		{OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "activities_created", ActorID: &actor, CreatedAt: now},
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, _, err := repo.Triage(ctx, orgID, ticketID, &kind, plans, audits)
		results <- err
	}()
	go func() {
		<-start
		_, err := repo.Dismiss(ctx, orgID, ticketID, 0, dismissAudit(orgID, ticketID, requesterID, now))
		results <- err
	}()
	close(start)

	successes := 0
	var loserErr error
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			successes++
		} else {
			loserErr = err
		}
	}
	require.Equal(t, 1, successes, "exactly one of triage/dismiss must win the race")
	require.ErrorIs(t, loserErr, ticketdomain.ErrInvalidTransition, "the race loser must fail with the matrix error")

	// Winner-consistent final state.
	var status string
	err := pool.QueryRow(ctx, `SELECT status FROM tickets WHERE id = $1`, ticketID).Scan(&status)
	require.NoError(t, err)

	var activityCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket'`,
		ticketID).Scan(&activityCount)
	require.NoError(t, err)

	if status == ticketdomain.StatusDismissed {
		require.Zero(t, activityCount, "dismiss winner leaves zero customer_ticket activities")
		var triagedAudits int
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action = 'triaged'`,
			ticketID).Scan(&triagedAudits)
		require.NoError(t, err)
		require.Zero(t, triagedAudits, "dismiss winner leaves zero 'triaged' audit rows")
	} else {
		require.Equal(t, ticketdomain.StatusPlanned, status, "triage winner flips the ticket to 'planned'")
		require.Equal(t, 1, activityCount, "triage winner creates exactly one customer_ticket activity")
		var triagedAudits, activitiesCreatedAudits int
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action = 'triaged'`,
			ticketID).Scan(&triagedAudits)
		require.NoError(t, err)
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action = 'activities_created'`,
			ticketID).Scan(&activitiesCreatedAudits)
		require.NoError(t, err)
		require.Equal(t, 1, triagedAudits)
		require.Equal(t, 1, activitiesCreatedAudits)
	}
}

// TestDismiss_RejectsPlanned pins race 2's illegal edge sequentially: a
// 'planned' ticket can never be dismissed (the matrix has no
// planned → dismissed edge) — the in-tx re-check must reject it even though
// the pool-level service check was bypassed (direct repo call).
//
// RED on the pre-fix code: Dismiss has no matrix re-check at all.
func TestDismiss_RejectsPlanned(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)

	ticketID := insertTicketWithStatus(t, pool, orgID, requesterID, "planned", now)

	_, err := repo.Dismiss(ctx, orgID, ticketID, 0, dismissAudit(orgID, ticketID, requesterID, now))
	require.ErrorIs(t, err, ticketdomain.ErrInvalidTransition, "planned → dismissed is illegal per the locked matrix")

	// The ticket stays 'planned' with no dismissal side effects.
	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM tickets WHERE id = $1`, ticketID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "planned", status)

	var dismissedAudits int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action = 'dismissed'`,
		ticketID).Scan(&dismissedAudits)
	require.NoError(t, err)
	require.Zero(t, dismissedAudits)
}

// TestTransitionRace_VsDismiss fires repo.UpdateState(to='planned') and
// repo.Dismiss concurrently on a 'triage' ticket and asserts EXACTLY ONE
// winner (CR-01 race 2): the loser returns ErrInvalidTransition and the final
// status is the winner's — 'dismissed' XOR 'planned', never both effects.
//
// RED on the pre-fix code: UpdateState's lock-free UPDATE lands even after
// Dismiss committed, resurrecting/flipping the terminal state.
func TestTransitionRace_VsDismiss(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)

	ticketID := insertTicketWithStatus(t, pool, orgID, requesterID, "triage", now)
	actor := requesterID

	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, err := repo.UpdateState(ctx, orgID, ticketID, ticketdomain.StatusPlanned, nil, &auditdomain.AuditLog{
			OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "status_changed",
			ActorID: &actor, Payload: map[string]any{"from": "triage", "to": "planned"}, CreatedAt: now,
		})
		results <- err
	}()
	go func() {
		<-start
		_, err := repo.Dismiss(ctx, orgID, ticketID, 0, dismissAudit(orgID, ticketID, requesterID, now))
		results <- err
	}()
	close(start)

	successes := 0
	var loserErr error
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			successes++
		} else {
			loserErr = err
		}
	}
	require.Equal(t, 1, successes, "exactly one of transition/dismiss must win the race")
	require.ErrorIs(t, loserErr, ticketdomain.ErrInvalidTransition, "the race loser must fail with the matrix error")

	// The final status is exactly the winner's — no mixed effects.
	var status string
	err := pool.QueryRow(ctx, `SELECT status FROM tickets WHERE id = $1`, ticketID).Scan(&status)
	require.NoError(t, err)
	require.True(t, status == ticketdomain.StatusPlanned || status == ticketdomain.StatusDismissed,
		"final status must be the winner's (planned XOR dismissed), got %q", status)
}

// TestTriage_RejectsDismissed pins race 1 sequentially (CR-01): a dismissed
// (terminal) ticket can never be resurrected to 'planned' — Triage must
// re-validate currentStatus against the matrix under its FOR UPDATE lock.
//
// RED on the pre-fix code: Triage never validates the scanned currentStatus.
func TestTriage_RejectsDismissed(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)
	seedActivityKind(t, pool, orgID, "engagement")

	ticketID := insertTicketWithStatus(t, pool, orgID, requesterID, "triage", now)

	// Dismiss first — the ticket is now terminal.
	_, err := repo.Dismiss(ctx, orgID, ticketID, 0, dismissAudit(orgID, ticketID, requesterID, now))
	require.NoError(t, err)

	// Triage on the dismissed ticket must be rejected.
	kind := ticketdomain.KindBug
	plans := []*activitydomain.CreateActivityRequest{
		{
			Name:            "Resurrect me",
			Kind:            activitydomain.ActivityKind("engagement"),
			GovernanceModel: models.GovernanceCreatorControlled,
		},
	}
	audits := []*auditdomain.AuditLog{
		{OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "triaged", ActorID: &requesterID, CreatedAt: now},
		{OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "activities_created", ActorID: &requesterID, CreatedAt: now},
	}
	_, _, err = repo.Triage(ctx, orgID, ticketID, &kind, plans, audits)
	require.ErrorIs(t, err, ticketdomain.ErrInvalidTransition, "a dismissed ticket can never be triaged back to 'planned'")

	// The ticket stays 'dismissed' with zero activities created.
	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM tickets WHERE id = $1`, ticketID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, ticketdomain.StatusDismissed, status)

	var activityCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM activities WHERE ticket_id = $1 AND origin_type = 'customer_ticket'`,
		ticketID).Scan(&activityCount)
	require.NoError(t, err)
	require.Zero(t, activityCount)
}

// TestUpdateState_ResolvedBlocked pins the resolved-block re-checked inside
// the mutator tx (CR-01, OQ2): UpdateState(to='resolved') must reject while a
// non-terminal (draft) entry exists on the linked-activity subtree — even
// when the pool-level fast-fail was bypassed (direct repo call) — and must
// succeed once the entry is deleted.
//
// RED on the pre-fix code: UpdateState has no in-tx
// HasNonTerminalActivities re-check at all.
func TestUpdateState_ResolvedBlocked(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTicketRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	requesterID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	seedActivityKind(t, pool, orgID, "engagement")

	ticketID := insertTicketWithStatus(t, pool, orgID, requesterID, "in_progress", now)
	activityID := insertCustomerTicketActivity(t, pool, orgID, ticketID, now)

	// Committed draft entry on the linked activity — non-terminal (OQ2).
	_, err := pool.Exec(ctx,
		`INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours, description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 4.0, 'still drafting', $6, 'draft', false, $6, $6)`,
		uuid.New(), orgID, requesterID, activityID, unitID, now)
	require.NoError(t, err)

	actor := requesterID
	audit := &auditdomain.AuditLog{
		OrgID: orgID, EntityType: "ticket", EntityID: ticketID, Action: "status_changed",
		ActorID: &actor, Payload: map[string]any{"from": "in_progress", "to": "resolved"}, CreatedAt: now,
	}

	// Blocked while the draft entry exists.
	_, err = repo.UpdateState(ctx, orgID, ticketID, ticketdomain.StatusResolved, nil, audit)
	require.ErrorIs(t, err, ticketdomain.ErrActivityNotTerminal)

	// Mark the entry deleted — the subtree is terminal, resolve succeeds.
	_, err = pool.Exec(ctx, `UPDATE time_entries SET is_deleted = true WHERE activity_id = $1`, activityID)
	require.NoError(t, err)

	updated, err := repo.UpdateState(ctx, orgID, ticketID, ticketdomain.StatusResolved, nil, audit)
	require.NoError(t, err)
	require.Equal(t, ticketdomain.StatusResolved, updated.Status)
}
