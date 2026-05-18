package time_entry

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
)

func setupService(t *testing.T) (*Service, *testdata.MockTimeEntryRepo, *testdata.MockAuditLogRepo) {
	t.Helper()
	repo := &testdata.MockTimeEntryRepo{Entries: make(map[uuid.UUID]*time_entry.TimeEntry)}
	auditRepo := &testdata.MockAuditLogRepo{}
	svc := NewService(repo, auditRepo)
	return svc, repo, auditRepo
}

// seedEntry adds a time entry to the mock repo and returns its pointer.
func seedEntry(repo *testdata.MockTimeEntryRepo, overrides ...func(*time_entry.TimeEntry)) *time_entry.TimeEntry {
	e := testdata.NewTimeEntry(overrides...)
	repo.Entries[e.ID] = &e
	return &e
}

// ---------------------------------------------------------------------------
// TestService_List
// ---------------------------------------------------------------------------

func TestService_List(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()

	t.Run("returns entries for org", func(t *testing.T) {
		svc, repo, _ := setupService(t)

		seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.Status = time_entry.StatusDraft
		})
		seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.Status = time_entry.StatusSubmitted
		})
		seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.OrgID = otherOrgID
		})

		entries, err := svc.List(context.Background(), orgID, ports.ListFilters{})
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("empty org returns empty", func(t *testing.T) {
		svc, _, _ := setupService(t)

		entries, err := svc.List(context.Background(), orgID, ports.ListFilters{})
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// ---------------------------------------------------------------------------
// TestService_Get
// ---------------------------------------------------------------------------

func TestService_Get(t *testing.T) {
	t.Run("existing entry returns", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo)

		got, err := svc.Get(context.Background(), entry.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, entry.ID, got.ID)
	})

	t.Run("nonexistent returns error", func(t *testing.T) {
		svc, _, _ := setupService(t)

		got, err := svc.Get(context.Background(), uuid.New())
		assert.ErrorIs(t, err, time_entry.ErrTimeEntryNotFound)
		assert.Nil(t, got)
	})
}

// ---------------------------------------------------------------------------
// TestService_Create
// ---------------------------------------------------------------------------

func TestService_Create(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("valid entry creates draft", func(t *testing.T) {
		svc, repo, _ := setupService(t)

		req := &time_entry.CreateTimeEntryRequest{
			OrgID:     orgID,
			UserID:    userID,
			ProjectID: projectID,
			Hours:     8.0,
			Date:      "2026-05-18",
		}

		entry, err := svc.Create(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, time_entry.StatusDraft, entry.Status)
		assert.NotEqual(t, uuid.Nil, entry.ID)
		assert.Equal(t, 8.0, entry.Hours)

		// Verify stored
		stored, ok := repo.Entries[entry.ID]
		require.True(t, ok)
		assert.Equal(t, time_entry.StatusDraft, stored.Status)
	})

	t.Run("period locked returns error", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		repo.PeriodLocked = true

		req := &time_entry.CreateTimeEntryRequest{
			OrgID:     orgID,
			UserID:    userID,
			ProjectID: projectID,
			Hours:     8.0,
			Date:      "2026-05-18",
		}

		entry, err := svc.Create(context.Background(), req)
		assert.ErrorIs(t, err, time_entry.ErrPeriodLocked)
		assert.Nil(t, entry)
	})

	t.Run("invalid date format", func(t *testing.T) {
		svc, _, _ := setupService(t)

		req := &time_entry.CreateTimeEntryRequest{
			OrgID:     orgID,
			UserID:    userID,
			ProjectID: projectID,
			Hours:     8.0,
			Date:      "not-a-date",
		}

		entry, err := svc.Create(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, entry)
	})
}

// ---------------------------------------------------------------------------
// TestService_Submit
// ---------------------------------------------------------------------------

func TestService_Submit(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	t.Run("owner submits draft entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
		})

		updated, err := svc.Submit(context.Background(), entry.ID, userID)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, time_entry.StatusSubmitted, updated.Status)
	})

	t.Run("non-owner cannot submit", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
		})

		updated, err := svc.Submit(context.Background(), entry.ID, otherUserID)
		assert.ErrorIs(t, err, time_entry.ErrNotOwner)
		assert.Nil(t, updated)
	})

	t.Run("cannot submit already submitted", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Submit(context.Background(), entry.ID, userID)
		assert.ErrorIs(t, err, time_entry.ErrEntryNotDraft)
		assert.Nil(t, updated)
	})

	t.Run("cannot submit approved entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusApproved
		})

		updated, err := svc.Submit(context.Background(), entry.ID, userID)
		assert.ErrorIs(t, err, time_entry.ErrEntryNotDraft)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_Approve — role x action x state matrix
// ---------------------------------------------------------------------------

func TestService_Approve(t *testing.T) {
	userID := uuid.New()

	t.Run("wg_manager approves submitted entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Approve(context.Background(), entry.ID, userID, "wg_manager")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, time_entry.StatusApproved, updated.Status)
	})

	t.Run("admin approves submitted entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Approve(context.Background(), entry.ID, userID, "admin")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, time_entry.StatusApproved, updated.Status)
	})

	t.Run("employee cannot approve", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Approve(context.Background(), entry.ID, userID, "employee")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("manager role cannot approve", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Approve(context.Background(), entry.ID, userID, "manager")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("finance role cannot approve", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Approve(context.Background(), entry.ID, userID, "finance")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("customer cannot approve", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Approve(context.Background(), entry.ID, userID, "customer")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("cannot approve draft", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
		})

		updated, err := svc.Approve(context.Background(), entry.ID, userID, "wg_manager")
		assert.ErrorIs(t, err, time_entry.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})

	t.Run("cannot approve already approved", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusApproved
		})

		updated, err := svc.Approve(context.Background(), entry.ID, userID, "wg_manager")
		assert.ErrorIs(t, err, time_entry.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_Reject
// ---------------------------------------------------------------------------

func TestService_Reject(t *testing.T) {
	userID := uuid.New()

	t.Run("wg_manager rejects submitted entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Reject(context.Background(), entry.ID, userID, "wg_manager", "Incorrect hours")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, time_entry.StatusDraft, updated.Status, "reject should return entry to draft")
	})

	t.Run("admin rejects submitted entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Reject(context.Background(), entry.ID, userID, "admin", "Not approved")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, time_entry.StatusDraft, updated.Status)
	})

	t.Run("employee cannot reject", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		updated, err := svc.Reject(context.Background(), entry.ID, userID, "employee", "")
		assert.ErrorIs(t, err, time_entry.ErrForbidden)
		assert.Nil(t, updated)
	})

	t.Run("cannot reject draft", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
		})

		updated, err := svc.Reject(context.Background(), entry.ID, userID, "wg_manager", "")
		assert.ErrorIs(t, err, time_entry.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})

	t.Run("cannot reject approved entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusApproved
		})

		updated, err := svc.Reject(context.Background(), entry.ID, userID, "wg_manager", "")
		assert.ErrorIs(t, err, time_entry.ErrEntryNotSubmitted)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_Update
// ---------------------------------------------------------------------------

func TestService_Update(t *testing.T) {
	userID := uuid.New()
	newHours := 6.5
	newDesc := "Updated description"

	t.Run("owner updates draft entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
			e.Hours = 8.0
			e.Description = "Original"
		})

		req := &time_entry.UpdateTimeEntryRequest{
			Hours:       &newHours,
			Description: &newDesc,
		}

		updated, err := svc.Update(context.Background(), entry.ID, userID, req)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, 6.5, updated.Hours)
		assert.Equal(t, "Updated description", updated.Description)
	})

	t.Run("cannot update after submitted", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		req := &time_entry.UpdateTimeEntryRequest{
			Description: &newDesc,
		}

		updated, err := svc.Update(context.Background(), entry.ID, userID, req)
		assert.ErrorIs(t, err, time_entry.ErrEntryNotDraft)
		assert.Nil(t, updated)
	})

	t.Run("non-owner cannot update", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
		})

		req := &time_entry.UpdateTimeEntryRequest{
			Description: &newDesc,
		}

		otherUserID := uuid.New()
		updated, err := svc.Update(context.Background(), entry.ID, otherUserID, req)
		assert.ErrorIs(t, err, time_entry.ErrNotOwner)
		assert.Nil(t, updated)
	})
}

// ---------------------------------------------------------------------------
// TestService_ListPending
// ---------------------------------------------------------------------------

func TestService_ListPending(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	t.Run("returns entries for org and role", func(t *testing.T) {
		svc, repo, _ := setupService(t)

		seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.Status = time_entry.StatusSubmitted
		})
		seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.OrgID = orgID
			e.Status = time_entry.StatusDraft
		})

		entries, err := svc.ListPending(context.Background(), orgID, "wg_manager", userID.String())
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("empty org returns empty", func(t *testing.T) {
		svc, _, _ := setupService(t)

		entries, err := svc.ListPending(context.Background(), uuid.New(), "wg_manager", userID.String())
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// ---------------------------------------------------------------------------
// TestService_Delete
// ---------------------------------------------------------------------------

func TestService_Delete(t *testing.T) {
	userID := uuid.New()

	t.Run("owner deletes draft entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusDraft
		})

		err := svc.Delete(context.Background(), entry.ID, userID)
		require.NoError(t, err)

		// Verify deleted from repo
		_, err = repo.GetByID(context.Background(), entry.ID)
		assert.ErrorIs(t, err, time_entry.ErrTimeEntryNotFound)
	})

	t.Run("cannot delete submitted entry", func(t *testing.T) {
		svc, repo, _ := setupService(t)
		entry := seedEntry(repo, func(e *time_entry.TimeEntry) {
			e.UserID = userID
			e.Status = time_entry.StatusSubmitted
		})

		err := svc.Delete(context.Background(), entry.ID, userID)
		assert.ErrorIs(t, err, time_entry.ErrEntryNotDraft)
	})
}
