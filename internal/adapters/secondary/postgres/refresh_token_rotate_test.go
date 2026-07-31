package postgres

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// TestMigration010_AppliesAndRollsBackCleanly verifies migration 010
// (refresh token reuse detection) applies against a fresh schema, rolls back
// cleanly, and re-applies without losing data semantics.
func TestMigration010_AppliesAndRollsBackCleanly(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	ctx := context.Background()
	root := findProjectRoot(t)
	readMig := func(name string) string {
		content, err := os.ReadFile(filepath.Join(root, "migrations", name))
		require.NoError(t, err)
		return string(content)
	}

	// Applied via SetupTestSchema — columns must exist.
	var colCount int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
		WHERE table_name = 'refresh_tokens' AND column_name IN ('family_id', 'rotated_at')`).Scan(&colCount)
	require.NoError(t, err)
	assert.Equal(t, 2, colCount, "migration 010 should add family_id and rotated_at")

	// Roll back.
	_, err = pool.Exec(ctx, readMig("010_refresh_token_reuse_detection.down.sql"))
	require.NoError(t, err, "010 down should apply cleanly")
	err = pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
		WHERE table_name = 'refresh_tokens' AND column_name IN ('family_id', 'rotated_at')`).Scan(&colCount)
	require.NoError(t, err)
	assert.Equal(t, 0, colCount, "rollback should drop family_id and rotated_at")

	// Re-apply (restores state for any later tests in this package).
	_, err = pool.Exec(ctx, readMig("010_refresh_token_reuse_detection.up.sql"))
	require.NoError(t, err, "010 up should re-apply cleanly")
	err = pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
		WHERE table_name = 'refresh_tokens' AND column_name IN ('family_id', 'rotated_at')`).Scan(&colCount)
	require.NoError(t, err)
	assert.Equal(t, 2, colCount, "re-apply should restore family_id and rotated_at")
}

func TestRefreshTokenRepository_Rotate(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewRefreshTokenRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	oldHash := uuid.New().String()
	require.NoError(t, repo.Add(ctx, userID, orgID, oldHash, now.Add(7*24*time.Hour)))

	// Family id is assigned on insert.
	seeded, err := repo.FindByHash(ctx, oldHash)
	require.NoError(t, err)
	require.NotNil(t, seeded)
	require.NotEqual(t, uuid.Nil, seeded.FamilyID, "Add should assign a family id")

	newHash := uuid.New().String()
	consumed, err := repo.Rotate(ctx, oldHash, newHash, now.Add(7*24*time.Hour))
	require.NoError(t, err)
	require.NotNil(t, consumed)
	assert.Equal(t, userID, consumed.UserID)

	// Old hash is no longer findable (rotated), successor is valid with same family.
	old, err := repo.FindByHash(ctx, oldHash)
	require.NoError(t, err)
	assert.Nil(t, old, "rotated token should not be findable")

	succ, err := repo.FindByHash(ctx, newHash)
	require.NoError(t, err)
	require.NotNil(t, succ)
	assert.Equal(t, consumed.FamilyID, succ.FamilyID, "successor must inherit the family id")
}

func TestRefreshTokenRepository_Rotate_ReplayRevokesFamily(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewRefreshTokenRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	oldHash := uuid.New().String()
	require.NoError(t, repo.Add(ctx, userID, orgID, oldHash, now.Add(7*24*time.Hour)))
	seeded, err := repo.FindByHash(ctx, oldHash)
	require.NoError(t, err)
	require.NotNil(t, seeded)

	// Legitimate rotation oldHash -> newHash.
	newHash := uuid.New().String()
	_, err = repo.Rotate(ctx, oldHash, newHash, now.Add(7*24*time.Hour))
	require.NoError(t, err)

	// Replaying the rotated token revokes the family and signals reuse.
	replayed, err := repo.Rotate(ctx, oldHash, uuid.New().String(), now.Add(7*24*time.Hour))
	require.ErrorIs(t, err, ports.ErrTokenReuse)
	require.NotNil(t, replayed)

	// The successor died with its family.
	succ, err := repo.FindByHash(ctx, newHash)
	require.NoError(t, err)
	assert.Nil(t, succ, "successor token should be revoked with its family")

	// Unknown hash -> (nil, nil), not ErrTokenReuse.
	unknown, err := repo.Rotate(ctx, uuid.New().String(), uuid.New().String(), now.Add(7*24*time.Hour))
	require.NoError(t, err)
	assert.Nil(t, unknown)
}

func TestRefreshTokenRepository_RevokeFamily(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewRefreshTokenRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	userID := seedUser(t, pool, now)

	hashA := uuid.New().String()
	hashB := uuid.New().String()
	require.NoError(t, repo.Add(ctx, userID, orgID, hashA, now.Add(7*24*time.Hour)))
	require.NoError(t, repo.Add(ctx, userID, orgID, hashB, now.Add(7*24*time.Hour)))

	famA, err := repo.FindByHash(ctx, hashA)
	require.NoError(t, err)
	require.NotNil(t, famA)
	famB, err := repo.FindByHash(ctx, hashB)
	require.NoError(t, err)
	require.NotNil(t, famB)
	require.NotEqual(t, famA.FamilyID, famB.FamilyID, "each Add should create its own family")

	require.NoError(t, repo.RevokeFamily(ctx, famA.FamilyID))

	gotA, err := repo.FindByHash(ctx, hashA)
	require.NoError(t, err)
	assert.Nil(t, gotA, "token in revoked family should be tombstoned")

	gotB, err := repo.FindByHash(ctx, hashB)
	require.NoError(t, err)
	assert.NotNil(t, gotB, "token in a different family should be untouched")
}

// compile-time guard: the real repo satisfies the extended interface.
var _ ports.RefreshTokenRepository = (*RefreshTokenRepository)(nil)
