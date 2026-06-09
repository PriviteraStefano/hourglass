package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Add_GetByID(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)
	user := auth.NewUser(uniqueEmail(), uniqueUsername(), "Test", "User", "hash123")

	err := repo.Add(context.Background(), user)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, got.ID)
	require.Equal(t, user.Email, got.Email)
	require.Equal(t, user.Username, got.Username)
	require.Equal(t, user.FirstName, got.FirstName)
	require.Equal(t, user.LastName, got.LastName)
	require.Equal(t, user.PasswordHash, got.PasswordHash)
	require.True(t, got.IsActive)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, ports.ErrUserNotFound)
}

func TestUserRepository_GetByEmail(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)
	user := auth.NewUser(uniqueEmail(), uniqueUsername(), "Email", "Test", "hash456")
	err := repo.Add(context.Background(), user)
	require.NoError(t, err)

	got, err := repo.GetByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	require.Equal(t, user.ID, got.ID)
	require.Equal(t, user.Email, got.Email)

	// Not found
	_, err = repo.GetByEmail(context.Background(), "nonexistent@test.com")
	require.Error(t, err)
	require.ErrorIs(t, err, ports.ErrUserNotFound)
}

func TestUserRepository_GetByUsername(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)
	uname := uniqueUsername()
	user := auth.NewUser(uniqueEmail(), uname, "Username", "Test", "hash789")
	err := repo.Add(context.Background(), user)
	require.NoError(t, err)

	got, err := repo.GetByUsername(context.Background(), uname)
	require.NoError(t, err)
	require.Equal(t, user.ID, got.ID)
	require.Equal(t, user.Username, got.Username)

	// Not found
	_, err = repo.GetByUsername(context.Background(), "nonexistent_user")
	require.Error(t, err)
	require.ErrorIs(t, err, ports.ErrUserNotFound)
}

func TestUserRepository_EmailExists(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)
	email := uniqueEmail()

	// Should not exist yet
	exists, err := repo.EmailExists(context.Background(), email)
	require.NoError(t, err)
	require.False(t, exists)

	// Add user and check again
	user := auth.NewUser(email, uniqueUsername(), "Exists", "Test", "hash")
	err = repo.Add(context.Background(), user)
	require.NoError(t, err)

	exists, err = repo.EmailExists(context.Background(), email)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestUserRepository_UsernameExists(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)
	uname := uniqueUsername()

	exists, err := repo.UsernameExists(context.Background(), uname)
	require.NoError(t, err)
	require.False(t, exists)

	user := auth.NewUser(uniqueEmail(), uname, "Uname", "Test", "hash")
	err = repo.Add(context.Background(), user)
	require.NoError(t, err)

	exists, err = repo.UsernameExists(context.Background(), uname)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestUserRepository_AnyExists(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)

	// Initially false
	any, err := repo.AnyExists(context.Background())
	require.NoError(t, err)
	require.False(t, any)

	// After adding a user
	user := auth.NewUser(uniqueEmail(), uniqueUsername(), "Any", "Exists", "hash")
	err = repo.Add(context.Background(), user)
	require.NoError(t, err)

	any, err = repo.AnyExists(context.Background())
	require.NoError(t, err)
	require.True(t, any)
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)
	user := auth.NewUser(uniqueEmail(), uniqueUsername(), "Update", "Pw", "oldhash")
	err := repo.Add(context.Background(), user)
	require.NoError(t, err)

	err = repo.UpdatePassword(context.Background(), user.ID, "newhash")
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, "newhash", got.PasswordHash)
}

func TestUserRepository_GetMemberships(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)

	// Create user, org, membership via raw SQL
	userID := uuid.New()
	orgID := uuid.New()
	now := time.Now().UTC()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Test Org", "test-org-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, uniqueEmail(), uniqueUsername(), "Member", "Ship", "hash", true, now, now)
	require.NoError(t, err)

	membershipID := uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		membershipID, userID, orgID, "employee", true, now)
	require.NoError(t, err)

	memberships, err := repo.GetMemberships(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, memberships, 1)
	require.Equal(t, orgID, memberships[0].OrganizationID)
	require.Equal(t, userID, memberships[0].UserID)
	require.Equal(t, "employee", memberships[0].Role)

	// No memberships returns empty slice, not nil
	empty, err := repo.GetMemberships(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestUserRepository_AddWithMembership(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)

	// Pre-create an organization
	orgID := uuid.New()
	now := time.Now().UTC()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		orgID, "Org For Membership", "org-mem-"+uuid.New().String()[:8], now, now)
	require.NoError(t, err)

	user := auth.NewUser(uniqueEmail(), uniqueUsername(), "With", "Membership", "hash")
	membership := auth.NewOrganizationMembership(user.ID, orgID, "manager")

	err = repo.AddWithMembership(context.Background(), user, membership)
	require.NoError(t, err)

	// Verify user
	gotUser, err := repo.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.Email, gotUser.Email)

	// Verify membership
	memberships, err := repo.GetMemberships(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, memberships, 1)
	require.Equal(t, orgID, memberships[0].OrganizationID)
	require.Equal(t, "manager", memberships[0].Role)
}

func TestUserRepository_AddWithOrgAndMembership(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)

	user := auth.NewUser(uniqueEmail(), uniqueUsername(), "Full", "Flow", "hash")
	org := auth.NewOrganization("New Org", "new-org-"+uuid.New().String()[:8], "Description")
	membership := auth.NewOrganizationMembership(user.ID, org.ID, "manager")

	err := repo.AddWithOrgAndMembership(context.Background(), user, org, membership)
	require.NoError(t, err)

	// Verify org exists
	var orgCount int
	err = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM organizations WHERE id = $1`, org.ID).Scan(&orgCount)
	require.NoError(t, err)
	require.Equal(t, 1, orgCount)

	// Verify user exists
	gotUser, err := repo.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.Email, gotUser.Email)

	// Verify membership
	memberships, err := repo.GetMemberships(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, memberships, 1)
	require.Equal(t, org.ID, memberships[0].OrganizationID)
	require.Equal(t, "manager", memberships[0].Role)
}

func TestUserRepository_Add_DuplicateEmail(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewUserRepository(pool)
	email := uniqueEmail()

	user1 := auth.NewUser(email, uniqueUsername(), "First", "User", "hash1")
	err := repo.Add(context.Background(), user1)
	require.NoError(t, err)

	user2 := auth.NewUser(email, uniqueUsername(), "Second", "User", "hash2")
	err = repo.Add(context.Background(), user2)
	require.Error(t, err)
	require.ErrorIs(t, err, ports.ErrConflict)
}
