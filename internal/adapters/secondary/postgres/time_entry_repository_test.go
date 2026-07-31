package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stretchr/testify/require"
)

// seedContract creates a test contract linked to an org.
func seedContract(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO contracts (id, name, km_rate, currency, governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, true, $8, $8)`,
		id, "Test Contract", 0.42, "EUR", "creator_controlled", orgID, false, now)
	require.NoError(t, err)
	return id
}

// seedCustomer creates a test customer linked to an org.
func seedCustomer(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO customers (id, org_id, name, contact_name, email, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $6)`,
		id, orgID, "Test Customer", "Contact", "customer@test.com", now)
	require.NoError(t, err)
	return id
}

// seedWorkingGroup creates a test working group anchored to an activity.
func seedWorkingGroup(t *testing.T, pool *pgxpool.Pool, orgID, activityID, managerID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO working_groups (id, org_id, activity_id, name, description, unit_ids, manager_id, delegate_ids, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9, $9)`,
		id, orgID, activityID, "Test WG", "Test working group", []interface{}{}, managerID, []interface{}{}, now)
	require.NoError(t, err)
	return id
}

// seedFinancialCutoffPeriod creates a test financial cutoff period for an activity.
func seedFinancialCutoffPeriod(t *testing.T, pool *pgxpool.Pool, orgID, activityID uuid.UUID, start, end time.Time, locked bool) {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO financial_cutoff_periods (id, org_id, activity_id, period_start, period_end, cutoff_date, is_locked, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		id, orgID, activityID, start, end, end, locked)
	require.NoError(t, err)
}

// seedEntryActivity creates an engagement + task child activity and links the
// engagement to a contract, mirroring the migrated MVP topology.
func seedEntryActivity(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, now time.Time) (activityID uuid.UUID) {
	t.Helper()
	activityID = seedActivity(t, pool, orgID, "engagement", nil, now)
	contractID := seedContract(t, pool, orgID, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE activities SET contract_id = $1 WHERE id = $2`,
		contractID, activityID)
	require.NoError(t, err)
	return activityID
}

func TestTimeEntryRepository_Create_GetByID(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTimeEntryRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	activityID := seedEntryActivity(t, pool, orgID, now)
	wgID := seedWorkingGroup(t, pool, orgID, activityID, userID, now)

	// Create a time entry with created_from_entry_id = nil
	entry := &time_entry.TimeEntry{
		OrgID:       orgID,
		UserID:      userID,
		ActivityID:  activityID,
		UnitID:      unitID,
		Hours:       7.5,
		Description: "Test time entry",
		EntryDate:   now,
		Status:      time_entry.StatusDraft,
		IsDeleted:   false,
	}

	created, err := repo.Create(context.Background(), entry)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, orgID, created.OrgID)
	require.Equal(t, userID, created.UserID)
	require.Equal(t, activityID, created.ActivityID)
	require.Equal(t, unitID, created.UnitID)
	require.Equal(t, 7.5, created.Hours)
	require.Equal(t, "Test time entry", created.Description)
	require.Equal(t, time_entry.StatusDraft, created.Status)
	require.False(t, created.IsDeleted)
	require.Nil(t, created.CreatedFromEntryID)
	require.False(t, created.CreatedAt.IsZero())
	require.False(t, created.UpdatedAt.IsZero())

	// Get by ID
	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Hours, got.Hours)
	require.Equal(t, created.ActivityID, got.ActivityID)

	// Create with created_from_entry_id
	entry2 := &time_entry.TimeEntry{
		OrgID:              orgID,
		UserID:             userID,
		ActivityID:         activityID,
		UnitID:             unitID,
		Hours:              5.0,
		Description:        "Copied entry",
		EntryDate:          now,
		Status:             time_entry.StatusDraft,
		IsDeleted:          false,
		CreatedFromEntryID: &created.ID,
	}
	created2, err := repo.Create(context.Background(), entry2)
	require.NoError(t, err)
	require.NotNil(t, created2)
	require.NotNil(t, created2.CreatedFromEntryID)
	require.Equal(t, created.ID, *created2.CreatedFromEntryID)

	// WG seed stays unused by the repo itself — but its FK validity proves
	// the activity anchor works end-to-end
	_ = wgID
}

func TestTimeEntryRepository_GetByID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTimeEntryRepository(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, time_entry.ErrTimeEntryNotFound)
}

func TestTimeEntryRepository_List_NoFilters(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTimeEntryRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	activityID := seedEntryActivity(t, pool, orgID, now)
	seedWorkingGroup(t, pool, orgID, activityID, userID, now)

	// Create two entries
	for _, hours := range []float64{4.0, 6.0} {
		entry := &time_entry.TimeEntry{
			OrgID: orgID, UserID: userID, ActivityID: activityID, UnitID: unitID,
			Hours: hours, Description: "List test", EntryDate: now,
			Status: time_entry.StatusDraft, IsDeleted: false,
		}
		_, err := repo.Create(context.Background(), entry)
		require.NoError(t, err)
	}

	entries, err := repo.List(context.Background(), orgID, ports.ListFilters{IsDeleted: false})
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestTimeEntryRepository_List_WithFilters(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTimeEntryRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	userID2 := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	activityID := seedEntryActivity(t, pool, orgID, now)
	seedWorkingGroup(t, pool, orgID, activityID, userID, now)

	// Entry for user 1 (submitted)
	entry1 := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID, ActivityID: activityID, UnitID: unitID,
		Hours: 4.0, Description: "User1 entry", EntryDate: now,
		Status: time_entry.StatusSubmitted, IsDeleted: false,
	}
	created1, err := repo.Create(context.Background(), entry1)
	require.NoError(t, err)

	// Entry for user 2 (draft)
	entry2 := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID2, ActivityID: activityID, UnitID: unitID,
		Hours: 5.0, Description: "User2 entry", EntryDate: now,
		Status: time_entry.StatusDraft, IsDeleted: false,
	}
	_, err = repo.Create(context.Background(), entry2)
	require.NoError(t, err)

	// Filter by user
	filtered, err := repo.List(context.Background(), orgID, ports.ListFilters{
		UserID:    userID.String(),
		IsDeleted: false,
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.Equal(t, created1.ID, filtered[0].ID)

	// Filter by status
	statusFiltered, err := repo.List(context.Background(), orgID, ports.ListFilters{
		Status:    time_entry.StatusDraft,
		IsDeleted: false,
	})
	require.NoError(t, err)
	require.Len(t, statusFiltered, 1)
	require.Equal(t, time_entry.StatusDraft, statusFiltered[0].Status)

	// Filter by activity
	activityFiltered, err := repo.List(context.Background(), orgID, ports.ListFilters{
		ActivityID: activityID.String(),
		IsDeleted:  false,
	})
	require.NoError(t, err)
	require.Len(t, activityFiltered, 2)
}

func TestTimeEntryRepository_Update(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTimeEntryRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	activityID := seedEntryActivity(t, pool, orgID, now)
	seedWorkingGroup(t, pool, orgID, activityID, userID, now)

	entry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID, ActivityID: activityID, UnitID: unitID,
		Hours: 4.0, Description: "Original", EntryDate: now,
		Status: time_entry.StatusDraft, IsDeleted: false,
	}
	created, err := repo.Create(context.Background(), entry)
	require.NoError(t, err)

	// Update
	updatedHours := 8.0
	created.Hours = updatedHours
	created.Description = "Updated description"
	created.Status = time_entry.StatusSubmitted

	updated, err := repo.Update(context.Background(), created)
	require.NoError(t, err)
	require.Equal(t, updatedHours, updated.Hours)
	require.Equal(t, "Updated description", updated.Description)
	require.Equal(t, time_entry.StatusSubmitted, updated.Status)
	require.Equal(t, activityID, updated.ActivityID)
}

func TestTimeEntryRepository_Delete(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTimeEntryRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	activityID := seedEntryActivity(t, pool, orgID, now)
	seedWorkingGroup(t, pool, orgID, activityID, userID, now)

	entry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID, ActivityID: activityID, UnitID: unitID,
		Hours: 4.0, Description: "To delete", EntryDate: now,
		Status: time_entry.StatusDraft, IsDeleted: false,
	}
	created, err := repo.Create(context.Background(), entry)
	require.NoError(t, err)

	// Soft delete
	err = repo.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	// Should not be found by List with is_deleted=false
	entries, err := repo.List(context.Background(), orgID, ports.ListFilters{IsDeleted: false})
	require.NoError(t, err)
	require.Empty(t, entries)

	// Delete on non-existent ID should return error
	err = repo.Delete(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestTimeEntryRepository_IsPeriodLocked(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTimeEntryRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	activityID := seedEntryActivity(t, pool, orgID, now)

	// Create a locked cutoff period for the activity (R-5: org + activity + date range)
	start := now.Add(-72 * time.Hour)
	end := now.Add(72 * time.Hour)
	seedFinancialCutoffPeriod(t, pool, orgID, activityID, start, end, true)

	// Check a date within the locked period
	locked, err := repo.IsPeriodLocked(context.Background(), orgID, activityID, now.Format(time.RFC3339))
	require.NoError(t, err)
	require.True(t, locked)

	// Check a date outside the locked period
	outsideDate := now.Add(168 * time.Hour) // 1 week later
	locked, err = repo.IsPeriodLocked(context.Background(), orgID, activityID, outsideDate.Format(time.RFC3339))
	require.NoError(t, err)
	require.False(t, locked)
}

func TestTimeEntryRepository_ListPending(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTimeEntryRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	activityID := seedEntryActivity(t, pool, orgID, now)
	wgID := seedWorkingGroup(t, pool, orgID, activityID, managerID, now)

	// Submitted entry on the manager's WG activity (R-1 chain)
	submittedEntry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: employeeID, ActivityID: activityID, UnitID: unitID,
		Hours: 4.0, Description: "Pending entry", EntryDate: now,
		Status: time_entry.StatusSubmitted, IsDeleted: false,
	}
	created, err := repo.Create(context.Background(), submittedEntry)
	require.NoError(t, err)

	// Draft entry (should not appear in pending)
	draftEntry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: employeeID, ActivityID: activityID, UnitID: unitID,
		Hours: 3.0, Description: "Draft entry", EntryDate: now,
		Status: time_entry.StatusDraft, IsDeleted: false,
	}
	_, err = repo.Create(context.Background(), draftEntry)
	require.NoError(t, err)

	// List all pending (no role filter)
	allPending, err := repo.ListPending(context.Background(), orgID, "", "")
	require.NoError(t, err)
	require.Len(t, allPending, 1)
	require.Equal(t, created.ID, allPending[0].ID)

	// List pending with wg_manager role filter — manager of the WG anchored
	// to the entry's activity
	wgPending, err := repo.ListPending(context.Background(), orgID, "wg_manager", managerID.String())
	require.NoError(t, err)
	require.Len(t, wgPending, 1)
	require.Equal(t, created.ID, wgPending[0].ID)

	_ = wgID
}

func TestTimeEntryRepository_ListPending_WGRoleFilter(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewTimeEntryRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	managerID := seedUser(t, pool, now)
	otherManagerID := seedUser(t, pool, now)
	employeeID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	activityID := seedEntryActivity(t, pool, orgID, now)
	activityID2 := seedEntryActivity(t, pool, orgID, now)

	seedWorkingGroup(t, pool, orgID, activityID, managerID, now)
	seedWorkingGroup(t, pool, orgID, activityID2, otherManagerID, now)

	// Entry in manager's WG activity
	entry1 := &time_entry.TimeEntry{
		OrgID: orgID, UserID: employeeID, ActivityID: activityID, UnitID: unitID,
		Hours: 4.0, Description: "In my WG", EntryDate: now,
		Status: time_entry.StatusSubmitted, IsDeleted: false,
	}
	_, err := repo.Create(context.Background(), entry1)
	require.NoError(t, err)

	// Entry in other manager's WG activity
	entry2 := &time_entry.TimeEntry{
		OrgID: orgID, UserID: employeeID, ActivityID: activityID2, UnitID: unitID,
		Hours: 5.0, Description: "Not my WG", EntryDate: now,
		Status: time_entry.StatusSubmitted, IsDeleted: false,
	}
	_, err = repo.Create(context.Background(), entry2)
	require.NoError(t, err)

	// Manager should only see entries on activities whose WG they manage
	wgPending, err := repo.ListPending(context.Background(), orgID, "wg_manager", managerID.String())
	require.NoError(t, err)
	require.Len(t, wgPending, 1)
	require.Equal(t, "In my WG", wgPending[0].Description)
}

func TestAuditLogRepository_Create(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	auditRepo := NewAuditLogRepository(pool)
	timeRepo := NewTimeEntryRepository(pool)
	now := time.Now().UTC()

	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)
	actorID := seedUser(t, pool, now)
	unitID := seedUnit(t, pool, orgID, now)
	activityID := seedEntryActivity(t, pool, orgID, now)
	seedWorkingGroup(t, pool, orgID, activityID, userID, now)

	// Create a time entry to reference
	entry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID, ActivityID: activityID, UnitID: unitID,
		Hours: 4.0, Description: "Audit test entry", EntryDate: now,
		Status: time_entry.StatusDraft, IsDeleted: false,
	}
	created, err := timeRepo.Create(context.Background(), entry)
	require.NoError(t, err)

	// Create audit log
	log := &time_entry.AuditLog{
		OrgID:     orgID,
		EntryID:   created.ID.String(),
		EntryType: "time_entry",
		Action:    "submit",
		ActorRole: "employee",
		ActorID:   actorID,
		Reason:    "Submitted for approval",
		Changes:   map[string]any{"status": "submitted"},
		Timestamp: now,
	}

	err = auditRepo.Create(context.Background(), log)
	require.NoError(t, err)

	// Verify via raw SQL
	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM time_entry_approvals WHERE time_entry_id = $1 AND action = 'submit'`,
		created.ID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
