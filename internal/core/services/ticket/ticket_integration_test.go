package ticket

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/postgres"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	ticketdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/ticket"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realRepoFixture wires the ticket service with real postgres repos backed
// by the package test container (schema lifecycle handled here).
func realRepoFixture(t *testing.T, pool *pgxpool.Pool) (*Service, *postgres.TicketRepository, uuid.UUID, uuid.UUID) {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	ticketRepo := postgres.NewTicketRepository(pool)
	svc := NewService(
		ticketRepo,
		postgres.NewActivityRepository(pool),
		postgres.NewContractRepository(pool),
		postgres.NewOrganizationRepository(pool),
	)

	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	seedUnit(t, pool, orgID, now)
	return svc, ticketRepo, orgID, userID
}

// realRepoFixtureWithUnit is realRepoFixture plus the seeded unit id (for
// time-entry seeding).
func realRepoFixtureWithUnit(t *testing.T, pool *pgxpool.Pool) (*Service, *postgres.TicketRepository, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	postgres.SetupTestSchema(t, pool)
	t.Cleanup(func() { postgres.TeardownTestSchema(t, pool) })

	ticketRepo := postgres.NewTicketRepository(pool)
	svc := NewService(
		ticketRepo,
		postgres.NewActivityRepository(pool),
		postgres.NewContractRepository(pool),
		postgres.NewOrganizationRepository(pool),
	)

	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	return svc, ticketRepo, orgID, userID, unitID
}

func seedOrg(t *testing.T, pool *pgxpool.Pool, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		id, "Ticket Org", "tk-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)
	return id
}

func seedUser(t *testing.T, pool *pgxpool.Pool, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7)`,
		id, uuid.New().String()+"@test.com", "tkuser"+uuid.New().String()[:8], "Tk", "User", "hash", now)
	require.NoError(t, err)
	return id
}

func seedUnit(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO units (id, org_id, name, code, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, orgID, "Tk Unit", "TKU", now, now)
	require.NoError(t, err)
	return id
}

// seedTicketKind ensures the org's activity_kinds catalog has the kind label
// (triage fast-fail + in-tx validation both need it).
func seedTicketKind(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, kind string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO activity_kinds (org_id, name, is_seed) VALUES ($1, $2, true)
		 ON CONFLICT (org_id, name) DO NOTHING`,
		orgID, kind)
	require.NoError(t, err)
}

// seedLinkedActivity inserts an activity with the customer_ticket origin
// linked to the ticket (D-10 shape, mirroring what repo.Triage writes).
func seedLinkedActivity(t *testing.T, pool *pgxpool.Pool, orgID, ticketID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	seedTicketKind(t, pool, orgID, "engagement")
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO activities (id, org_id, name, description, kind,
			governance_model, created_by_org_id, is_shared, is_active,
			origin_type, ticket_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 'creator_controlled', $2, false, true,
			'customer_ticket', $6, $7, $7)`,
		id, orgID, "Linked activity", "from ticket", "engagement", ticketID, now)
	require.NoError(t, err)
	return id
}

// seedTimeEntry inserts a time entry against an activity with the given
// status (raw Σ per D-13: submitted+approved count, draft does not).
func seedTimeEntry(t *testing.T, pool *pgxpool.Pool, orgID, userID, activityID, unitID uuid.UUID, hours float64, status string, now time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours,
			description, entry_date, status, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, $8, $8)`,
		uuid.New(), orgID, userID, activityID, unitID, hours, "entry", now, status)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// TestTicketTriage — TICK-03 atomicity + origin + audit rows
// ---------------------------------------------------------------------------

func TestTicketTriage(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("single failing plan rolls back everything (tx atomicity)", func(t *testing.T) {
		svc, _, orgID, userID := realRepoFixture(t, pool)
		seedTicketKind(t, pool, orgID, "engagement")

		tkt, err := svc.Create(context.Background(), orgID, userID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Atomic", Kind: ticketdomain.KindBug})
		require.NoError(t, err)
		// open → triage so the triage edge is legal.
		tkt, err = svc.Transition(context.Background(), orgID, userID, string(models.RoleEmployee), tkt.ID, ticketdomain.StatusTriage, nil)
		require.NoError(t, err)

		valid := &TriageActivityPlan{Name: "Good plan", Kind: "engagement", GovernanceModel: models.GovernanceCreatorControlled}
		invalid := &TriageActivityPlan{Name: "Bad plan", Kind: "not_a_kind", GovernanceModel: models.GovernanceCreatorControlled}

		_, _, err = svc.Triage(context.Background(), orgID, userID, string(models.RoleManager), tkt.ID, nil,
			[]*TriageActivityPlan{valid, invalid})
		require.ErrorIs(t, err, ticketdomain.ErrInvalidRequest)

		// Assert NO partial writes: ticket still triage, no activities
		// linked, no 'triaged'/'activities_created' audit rows.
		got, err := svc.repo.Get(context.Background(), orgID, tkt.ID)
		require.NoError(t, err)
		assert.Equal(t, ticketdomain.StatusTriage, got.Status)

		var activityCount int
		err = pool.QueryRow(context.Background(),
			`SELECT COUNT(*) FROM activities WHERE ticket_id = $1`, tkt.ID).Scan(&activityCount)
		require.NoError(t, err)
		assert.Zero(t, activityCount)

		var auditCount int
		err = pool.QueryRow(context.Background(),
			`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action IN ('triaged','activities_created')`,
			tkt.ID).Scan(&auditCount)
		require.NoError(t, err)
		assert.Zero(t, auditCount)
	})

	t.Run("valid triage creates activities with origin + two audit rows", func(t *testing.T) {
		svc, _, orgID, userID := realRepoFixture(t, pool)
		seedTicketKind(t, pool, orgID, "engagement")

		tkt, err := svc.Create(context.Background(), orgID, userID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Triage me", Kind: ticketdomain.KindChange})
		require.NoError(t, err)
		tkt, err = svc.Transition(context.Background(), orgID, userID, string(models.RoleEmployee), tkt.ID, ticketdomain.StatusTriage, nil)
		require.NoError(t, err)

		kind := ticketdomain.KindBug // kind override stays in the closed set
		updated, activities, err := svc.Triage(context.Background(), orgID, userID, string(models.RoleManager), tkt.ID, &kind,
			[]*TriageActivityPlan{
				{Name: "Investigate", Kind: "engagement", GovernanceModel: models.GovernanceCreatorControlled},
				{Name: "Follow-up", Kind: "engagement", GovernanceModel: models.GovernanceCreatorControlled},
			})
		require.NoError(t, err)
		assert.Equal(t, ticketdomain.StatusPlanned, updated.Status)
		assert.Equal(t, ticketdomain.KindBug, updated.Kind)
		require.Len(t, activities, 2)

		// Activities carry the customer_ticket origin + ticket_id (D-10).
		for _, a := range activities {
			require.NotNil(t, a.OriginType)
			assert.Equal(t, activitydomain.OriginTypeCustomerTicket, *a.OriginType)
			require.NotNil(t, a.TicketID)
			assert.Equal(t, tkt.ID, *a.TicketID)
			assert.True(t, a.IsActive)
		}

		// Both audit rows landed in the same tx.
		var auditCount int
		err = pool.QueryRow(context.Background(),
			`SELECT COUNT(*) FROM audit_logs WHERE entity_type = 'ticket' AND entity_id = $1 AND action IN ('triaged','activities_created')`,
			tkt.ID).Scan(&auditCount)
		require.NoError(t, err)
		assert.Equal(t, 2, auditCount)
	})
}

// ---------------------------------------------------------------------------
// TestDismissalGuard — TICK-04 raw Σ (D-13) blocks dismissal on logged hours
// ---------------------------------------------------------------------------

func TestDismissalGuard(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("submitted entry on linked activity blocks dismissal", func(t *testing.T) {
		svc, _, orgID, userID, unitID := realRepoFixtureWithUnit(t, pool)
		now := time.Now().UTC()

		tkt, err := svc.Create(context.Background(), orgID, userID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Guarded", Kind: ticketdomain.KindBug})
		require.NoError(t, err)
		linkedID := seedLinkedActivity(t, pool, orgID, tkt.ID, now)
		seedTimeEntry(t, pool, orgID, userID, linkedID, unitID, 4.0, "submitted", now)

		_, err = svc.Dismiss(context.Background(), orgID, userID, string(models.RoleManager), tkt.ID)
		require.ErrorIs(t, err, ticketdomain.ErrDismissalBlocked)

		// The ticket is untouched — no partial dismissal.
		got, err := svc.repo.Get(context.Background(), orgID, tkt.ID)
		require.NoError(t, err)
		assert.Equal(t, ticketdomain.StatusOpen, got.Status)
		assert.Nil(t, got.DismissedHours)
	})

	t.Run("only draft entries allow dismissal (raw Σ excludes draft, snapshot counts them)", func(t *testing.T) {
		svc, _, orgID, userID, unitID := realRepoFixtureWithUnit(t, pool)
		now := time.Now().UTC()

		tkt, err := svc.Create(context.Background(), orgID, userID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Draft only", Kind: ticketdomain.KindBug})
		require.NoError(t, err)
		linkedID := seedLinkedActivity(t, pool, orgID, tkt.ID, now)
		seedTimeEntry(t, pool, orgID, userID, linkedID, unitID, 8.0, "draft", now)

		dismissed, err := svc.Dismiss(context.Background(), orgID, userID, string(models.RoleManager), tkt.ID)
		require.NoError(t, err)
		assert.Equal(t, ticketdomain.StatusDismissed, dismissed.Status)
		// WR-06 (TICK-04): the guard blocks only submitted/approved, so the
		// dismissal succeeds — but the snapshot is the dismissal-time total
		// across ALL non-deleted entries (drafts included), so the note is
		// meaningful instead of always 0.
		require.NotNil(t, dismissed.DismissedHours)
		assert.Equal(t, 8.0, *dismissed.DismissedHours)
		require.NotNil(t, dismissed.DismissedNote)
		assert.Equal(t, "dismissed with 8 h logged", *dismissed.DismissedNote)
	})

	t.Run("deleted submitted entry does not block (is_deleted=false only)", func(t *testing.T) {
		svc, _, orgID, userID, unitID := realRepoFixtureWithUnit(t, pool)
		now := time.Now().UTC()

		tkt, err := svc.Create(context.Background(), orgID, userID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Deleted entry", Kind: ticketdomain.KindBug})
		require.NoError(t, err)
		linkedID := seedLinkedActivity(t, pool, orgID, tkt.ID, now)

		_, err = pool.Exec(context.Background(),
			`INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours,
				description, entry_date, status, is_deleted, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'submitted', true, $8, $8)`,
			uuid.New(), orgID, userID, linkedID, unitID, 4.0, "deleted", now)
		require.NoError(t, err)

		dismissed, err := svc.Dismiss(context.Background(), orgID, userID, string(models.RoleManager), tkt.ID)
		require.NoError(t, err)
		assert.Equal(t, ticketdomain.StatusDismissed, dismissed.Status)
	})
}

// ---------------------------------------------------------------------------
// TestTicketAudit — TICK-05 append-only stream, ordered history
// ---------------------------------------------------------------------------

func TestTicketAudit(t *testing.T) {
	pool := postgres.SetupPackageContainer(t)

	t.Run("comments + transitions land ordered in history", func(t *testing.T) {
		svc, _, orgID, userID := realRepoFixture(t, pool)

		tkt, err := svc.Create(context.Background(), orgID, userID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Stream", Kind: ticketdomain.KindBug})
		require.NoError(t, err)

		_, err = svc.Transition(context.Background(), orgID, userID, string(models.RoleEmployee), tkt.ID, ticketdomain.StatusTriage, nil)
		require.NoError(t, err)
		_, err = svc.AddComment(context.Background(), orgID, userID, string(models.RoleEmployee), tkt.ID, "first comment")
		require.NoError(t, err)

		history, err := svc.ListHistory(context.Background(), orgID, string(models.RoleEmployee), tkt.ID)
		require.NoError(t, err)
		require.Len(t, history, 3) // created + status_changed + comment_added
		assert.Equal(t, "created", history[0].Action)
		assert.Equal(t, "status_changed", history[1].Action)
		assert.Equal(t, "comment_added", history[2].Action)

		// Every row is org-scoped and addressed to this ticket.
		for _, h := range history {
			assert.Equal(t, orgID, h.OrgID)
			assert.Equal(t, tkt.ID, h.EntityID)
			assert.Equal(t, "ticket", h.EntityType)
		}
	})

	t.Run("actor captured on every audit row", func(t *testing.T) {
		svc, _, orgID, userID := realRepoFixture(t, pool)

		tkt, err := svc.Create(context.Background(), orgID, userID, string(models.RoleEmployee),
			&CreateTicketRequest{Title: "Actor", Kind: ticketdomain.KindBug})
		require.NoError(t, err)
		_, err = svc.AddComment(context.Background(), orgID, userID, string(models.RoleEmployee), tkt.ID, "who wrote this")
		require.NoError(t, err)

		history, err := svc.ListHistory(context.Background(), orgID, string(models.RoleEmployee), tkt.ID)
		require.NoError(t, err)
		require.Len(t, history, 2)
		for _, h := range history {
			require.NotNil(t, h.ActorID)
			assert.Equal(t, userID, *h.ActorID)
		}
	})
}
