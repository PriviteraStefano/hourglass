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

// seedWorkingGroup creates a test working group linked to a subproject.
func seedWorkingGroup(t *testing.T, pool *pgxpool.Pool, orgID, subprojectID, managerID uuid.UUID, now time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO working_groups (id, org_id, subproject_id, name, description, unit_ids, enforce_unit_tuple, manager_id, delegate_ids, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10, $10)`,
		id, orgID, subprojectID, "Test WG", "Test working group", []interface{}{}, true, managerID, []interface{}{}, now)
	require.NoError(t, err)
	return id
}

// seedFinancialCutoffPeriod creates a test financial cutoff period.
func seedFinancialCutoffPeriod(t *testing.T, pool *pgxpool.Pool, orgID, projectID uuid.UUID, start, end time.Time, locked bool) {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO financial_cutoff_periods (id, org_id, project_id, period_start, period_end, cutoff_date, is_locked, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		id, orgID, projectID, start, end, end, locked)
	require.NoError(t, err)
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
	customerID := seedCustomer(t, pool, orgID, now)
	contractID := seedContract(t, pool, orgID, now)
	projectID := seedProject(t, pool, orgID, now)
	// Override project FK contract
	_, err := pool.Exec(context.Background(),
		`UPDATE projects SET contract_id = $1, customer_id = $2 WHERE id = $3`,
		contractID, customerID, projectID)
	require.NoError(t, err)
	subprojectID := seedSubproject(t, pool, projectID, now)
	wgID := seedWorkingGroup(t, pool, orgID, subprojectID, userID, now)

	// Create a time entry with created_from_entry_id = nil
	entry := &time_entry.TimeEntry{
		OrgID:        orgID,
		UserID:       userID,
		ProjectID:    projectID,
		SubprojectID: subprojectID,
		WGID:         wgID,
		UnitID:       unitID,
		Hours:        7.5,
		Description:  "Test time entry",
		EntryDate:    now,
		Status:       time_entry.StatusDraft,
		IsDeleted:    false,
	}

	created, err := repo.Create(context.Background(), entry)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, orgID, created.OrgID)
	require.Equal(t, userID, created.UserID)
	require.Equal(t, projectID, created.ProjectID)
	require.Equal(t, subprojectID, created.SubprojectID)
	require.Equal(t, wgID, created.WGID)
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

	// Create with created_from_entry_id
	entry2 := &time_entry.TimeEntry{
		OrgID:              orgID,
		UserID:             userID,
		ProjectID:          projectID,
		SubprojectID:       subprojectID,
		WGID:               wgID,
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
	customerID := seedCustomer(t, pool, orgID, now)
	contractID := seedContract(t, pool, orgID, now)
	projectID := seedProject(t, pool, orgID, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE projects SET contract_id = $1, customer_id = $2 WHERE id = $3`,
		contractID, customerID, projectID)
	require.NoError(t, err)
	subprojectID := seedSubproject(t, pool, projectID, now)
	wgID := seedWorkingGroup(t, pool, orgID, subprojectID, userID, now)

	// Create two entries
	for _, hours := range []float64{4.0, 6.0} {
		entry := &time_entry.TimeEntry{
			OrgID: orgID, UserID: userID, ProjectID: projectID,
			SubprojectID: subprojectID, WGID: wgID, UnitID: unitID,
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
	customerID := seedCustomer(t, pool, orgID, now)
	contractID := seedContract(t, pool, orgID, now)
	projectID := seedProject(t, pool, orgID, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE projects SET contract_id = $1, customer_id = $2 WHERE id = $3`,
		contractID, customerID, projectID)
	require.NoError(t, err)
	subprojectID := seedSubproject(t, pool, projectID, now)
	wgID := seedWorkingGroup(t, pool, orgID, subprojectID, userID, now)

	// Entry for user 1 (submitted)
	entry1 := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID, ProjectID: projectID,
		SubprojectID: subprojectID, WGID: wgID, UnitID: unitID,
		Hours: 4.0, Description: "User1 entry", EntryDate: now,
		Status: time_entry.StatusSubmitted, IsDeleted: false,
	}
	created1, err := repo.Create(context.Background(), entry1)
	require.NoError(t, err)

	// Entry for user 2 (draft)
	entry2 := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID2, ProjectID: projectID,
		SubprojectID: subprojectID, WGID: wgID, UnitID: unitID,
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
	customerID := seedCustomer(t, pool, orgID, now)
	contractID := seedContract(t, pool, orgID, now)
	projectID := seedProject(t, pool, orgID, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE projects SET contract_id = $1, customer_id = $2 WHERE id = $3`,
		contractID, customerID, projectID)
	require.NoError(t, err)
	subprojectID := seedSubproject(t, pool, projectID, now)
	wgID := seedWorkingGroup(t, pool, orgID, subprojectID, userID, now)

	entry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID, ProjectID: projectID,
		SubprojectID: subprojectID, WGID: wgID, UnitID: unitID,
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
	customerID := seedCustomer(t, pool, orgID, now)
	contractID := seedContract(t, pool, orgID, now)
	projectID := seedProject(t, pool, orgID, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE projects SET contract_id = $1, customer_id = $2 WHERE id = $3`,
		contractID, customerID, projectID)
	require.NoError(t, err)
	subprojectID := seedSubproject(t, pool, projectID, now)
	wgID := seedWorkingGroup(t, pool, orgID, subprojectID, userID, now)

	entry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID, ProjectID: projectID,
		SubprojectID: subprojectID, WGID: wgID, UnitID: unitID,
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
	projectID := seedProject(t, pool, orgID, now)

	// Create a locked cutoff period
	start := now.Add(-72 * time.Hour)
	end := now.Add(72 * time.Hour)
	seedFinancialCutoffPeriod(t, pool, orgID, projectID, start, end, true)

	// Check a date within the locked period
	locked, err := repo.IsPeriodLocked(context.Background(), orgID, projectID, now.Format(time.RFC3339))
	require.NoError(t, err)
	require.True(t, locked)

	// Check a date outside the locked period
	outsideDate := now.Add(168 * time.Hour) // 1 week later
	locked, err = repo.IsPeriodLocked(context.Background(), orgID, projectID, outsideDate.Format(time.RFC3339))
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
	customerID := seedCustomer(t, pool, orgID, now)
	contractID := seedContract(t, pool, orgID, now)
	projectID := seedProject(t, pool, orgID, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE projects SET contract_id = $1, customer_id = $2 WHERE id = $3`,
		contractID, customerID, projectID)
	require.NoError(t, err)
	subprojectID := seedSubproject(t, pool, projectID, now)
	wgID := seedWorkingGroup(t, pool, orgID, subprojectID, managerID, now)

	// Submitted entry in manager's WG
	submittedEntry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: employeeID, ProjectID: projectID,
		SubprojectID: subprojectID, WGID: wgID, UnitID: unitID,
		Hours: 4.0, Description: "Pending entry", EntryDate: now,
		Status: time_entry.StatusSubmitted, IsDeleted: false,
	}
	created, err := repo.Create(context.Background(), submittedEntry)
	require.NoError(t, err)

	// Draft entry (should not appear in pending)
	draftEntry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: employeeID, ProjectID: projectID,
		SubprojectID: subprojectID, WGID: wgID, UnitID: unitID,
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

	// List pending with wg_manager role filter
	wgPending, err := repo.ListPending(context.Background(), orgID, "wg_manager", managerID.String())
	require.NoError(t, err)
	require.Len(t, wgPending, 1)
	require.Equal(t, created.ID, wgPending[0].ID)
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
	customerID := seedCustomer(t, pool, orgID, now)
	contractID := seedContract(t, pool, orgID, now)
	projectID := seedProject(t, pool, orgID, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE projects SET contract_id = $1, customer_id = $2 WHERE id = $3`,
		contractID, customerID, projectID)
	require.NoError(t, err)
	subprojectID := seedSubproject(t, pool, projectID, now)
	projectID2 := seedProject(t, pool, orgID, now)
	_, err = pool.Exec(context.Background(),
		`UPDATE projects SET contract_id = $1, customer_id = $2 WHERE id = $3`,
		contractID, customerID, projectID2)
	require.NoError(t, err)
	subprojectID2 := seedSubproject(t, pool, projectID2, now)

	wgID1 := seedWorkingGroup(t, pool, orgID, subprojectID, managerID, now)
	wgID2 := seedWorkingGroup(t, pool, orgID, subprojectID2, otherManagerID, now)

	// Entry in manager's WG
	entry1 := &time_entry.TimeEntry{
		OrgID: orgID, UserID: employeeID, ProjectID: projectID,
		SubprojectID: subprojectID, WGID: wgID1, UnitID: unitID,
		Hours: 4.0, Description: "In my WG", EntryDate: now,
		Status: time_entry.StatusSubmitted, IsDeleted: false,
	}
	_, err = repo.Create(context.Background(), entry1)
	require.NoError(t, err)

	// Entry in other manager's WG
	entry2 := &time_entry.TimeEntry{
		OrgID: orgID, UserID: employeeID, ProjectID: projectID2,
		SubprojectID: subprojectID2, WGID: wgID2, UnitID: unitID,
		Hours: 5.0, Description: "Not my WG", EntryDate: now,
		Status: time_entry.StatusSubmitted, IsDeleted: false,
	}
	_, err = repo.Create(context.Background(), entry2)
	require.NoError(t, err)

	// Manager should only see their own WG's entries
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
	customerID := seedCustomer(t, pool, orgID, now)
	contractID := seedContract(t, pool, orgID, now)
	projectID := seedProject(t, pool, orgID, now)
	_, err := pool.Exec(context.Background(),
		`UPDATE projects SET contract_id = $1, customer_id = $2 WHERE id = $3`,
		contractID, customerID, projectID)
	require.NoError(t, err)
	subprojectID := seedSubproject(t, pool, projectID, now)
	wgID := seedWorkingGroup(t, pool, orgID, subprojectID, userID, now)

	// Create a time entry to reference
	entry := &time_entry.TimeEntry{
		OrgID: orgID, UserID: userID, ProjectID: projectID,
		SubprojectID: subprojectID, WGID: wgID, UnitID: unitID,
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
