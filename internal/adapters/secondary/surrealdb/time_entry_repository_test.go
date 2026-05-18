package surrealdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdb "github.com/surrealdb/surrealdb.go"
)

func setupTimeEntryRepo(t *testing.T) (*TimeEntryRepository, func()) {
	t.Helper()

	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	ns := "test_timeentry_" + uuid.New().String()
	db, err := GetTestDBWithNamespace(ns, ns)
	if err != nil {
		t.Skipf("SurrealDB not available: %v", err)
	}

	repo := NewTimeEntryRepository(db)
	return repo, func() { db.Close(context.Background()) }
}

func seedTimeEntryUserAndOrg(t *testing.T, db *sdb.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()

	orgRepo := NewOrganizationRepository(db)
	orgID := uuid.New()
	org := &auth.Organization{
		ID:        orgID,
		Name:      "Test Org " + uuid.New().String()[:8],
		Slug:      "test-org-" + uuid.New().String()[:8],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := orgRepo.Add(context.Background(), org)
	require.NoError(t, err, "failed to seed org")

	userRepo := NewUserRepository(db)
	userID := uuid.New()
	user := &auth.User{
		ID:           userID,
		Email:        uuid.New().String() + "@test.com",
		Username:     "user_" + uuid.New().String()[:8],
		FirstName:    "Test",
		LastName:     "User",
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = userRepo.Add(context.Background(), user)
	require.NoError(t, err, "failed to seed user")

	return orgID, userID
}

func TestTimeEntryRepo_Create(t *testing.T) {
	repo, cleanup := setupTimeEntryRepo(t)
	defer cleanup()
	orgID, userID := seedTimeEntryUserAndOrg(t, repo.db)

	t.Run("valid entry", func(t *testing.T) {
		entry := &time_entry.TimeEntry{
			ID:          uuid.New(),
			OrgID:       orgID,
			UserID:      userID,
			Hours:       8.0,
			Description: "Worked on project X",
			EntryDate:   time.Now().Truncate(24 * time.Hour),
			Status:      time_entry.StatusDraft,
			IsDeleted:   false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		created, err := repo.Create(context.Background(), entry)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, created.ID, "expected ID to be set")
		assert.Equal(t, time_entry.StatusDraft, created.Status)
		assert.Equal(t, entry.Hours, created.Hours)
		assert.Equal(t, entry.Description, created.Description)
	})
}

func TestTimeEntryRepo_GetByID(t *testing.T) {
	repo, cleanup := setupTimeEntryRepo(t)
	defer cleanup()
	orgID, userID := seedTimeEntryUserAndOrg(t, repo.db)

	t.Run("existing", func(t *testing.T) {
		entry := &time_entry.TimeEntry{
			ID:          uuid.New(),
			OrgID:       orgID,
			UserID:      userID,
			Hours:       5.0,
			Description: "Team meeting",
			EntryDate:   time.Now().Truncate(24 * time.Hour),
			Status:      time_entry.StatusDraft,
			IsDeleted:   false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		created, err := repo.Create(context.Background(), entry)
		require.NoError(t, err)

		found, err := repo.GetByID(context.Background(), created.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, entry.Description, found.Description)
		assert.Equal(t, entry.Hours, found.Hours)
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.GetByID(context.Background(), uuid.New())
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestTimeEntryRepo_List(t *testing.T) {
	repo, cleanup := setupTimeEntryRepo(t)
	defer cleanup()
	orgID, userID := seedTimeEntryUserAndOrg(t, repo.db)

	t.Run("by org", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			entry := &time_entry.TimeEntry{
				ID:          uuid.New(),
				OrgID:       orgID,
				UserID:      userID,
				Hours:       2.0,
				Description: "Entry " + uuid.New().String()[:8],
				EntryDate:   time.Now().Truncate(24 * time.Hour),
				Status:      time_entry.StatusDraft,
				IsDeleted:   false,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			_, err := repo.Create(context.Background(), entry)
			require.NoError(t, err)
		}

		filters := ports.ListFilters{IsDeleted: false}
		entries, err := repo.List(context.Background(), orgID, filters)
		require.NoError(t, err)
		assert.Len(t, entries, 3)
	})

	t.Run("filters by user", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &auth.User{
			ID:           otherUserID,
			Email:        uuid.New().String() + "@test.com",
			Username:     "user_" + uuid.New().String()[:8],
			PasswordHash: "hash",
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		userRepo := NewUserRepository(repo.db)
		err := userRepo.Add(context.Background(), otherUser)
		require.NoError(t, err)

		entry := &time_entry.TimeEntry{
			ID:          uuid.New(),
			OrgID:       orgID,
			UserID:      otherUserID,
			Hours:       3.0,
			Description: "Other user entry",
			EntryDate:   time.Now().Truncate(24 * time.Hour),
			Status:      time_entry.StatusDraft,
			IsDeleted:   false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		_, err = repo.Create(context.Background(), entry)
		require.NoError(t, err)

		filters := ports.ListFilters{
			IsDeleted: false,
			UserID:    otherUserID.String(),
		}
		entries, err := repo.List(context.Background(), orgID, filters)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1)
	})
}

func TestTimeEntryRepo_Update(t *testing.T) {
	repo, cleanup := setupTimeEntryRepo(t)
	defer cleanup()
	orgID, userID := seedTimeEntryUserAndOrg(t, repo.db)

	t.Run("update hours and description", func(t *testing.T) {
		entry := &time_entry.TimeEntry{
			ID:          uuid.New(),
			OrgID:       orgID,
			UserID:      userID,
			Hours:       4.0,
			Description: "Original",
			EntryDate:   time.Now().Truncate(24 * time.Hour),
			Status:      time_entry.StatusDraft,
			IsDeleted:   false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		created, err := repo.Create(context.Background(), entry)
		require.NoError(t, err)

		created.Hours = 7.5
		created.Description = "Updated description"
		created.UpdatedAt = time.Now()

		updated, err := repo.Update(context.Background(), created)
		require.NoError(t, err)
		assert.Equal(t, 7.5, updated.Hours)
		assert.Equal(t, "Updated description", updated.Description)
	})
}

func TestTimeEntryRepo_Delete(t *testing.T) {
	repo, cleanup := setupTimeEntryRepo(t)
	defer cleanup()
	orgID, userID := seedTimeEntryUserAndOrg(t, repo.db)

	t.Run("soft delete", func(t *testing.T) {
		entry := &time_entry.TimeEntry{
			ID:          uuid.New(),
			OrgID:       orgID,
			UserID:      userID,
			Hours:       3.0,
			Description: "To delete",
			EntryDate:   time.Now().Truncate(24 * time.Hour),
			Status:      time_entry.StatusDraft,
			IsDeleted:   false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		created, err := repo.Create(context.Background(), entry)
		require.NoError(t, err)

		err = repo.Delete(context.Background(), created.ID)
		require.NoError(t, err)

		filters := ports.ListFilters{IsDeleted: true}
		entries, err := repo.List(context.Background(), orgID, filters)
		require.NoError(t, err)

		var found bool
		for _, e := range entries {
			if e.ID == created.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "deleted entry should be findable with IsDeleted=true")
	})
}

func TestTimeEntryRepo_IsPeriodLocked(t *testing.T) {
	repo, cleanup := setupTimeEntryRepo(t)
	defer cleanup()
	orgID, _ := seedTimeEntryUserAndOrg(t, repo.db)

	t.Run("not locked", func(t *testing.T) {
		locked, err := repo.IsPeriodLocked(context.Background(), orgID, uuid.Nil, "2026-01-15")
		require.NoError(t, err)
		assert.False(t, locked, "expected period not locked")
	})
}
