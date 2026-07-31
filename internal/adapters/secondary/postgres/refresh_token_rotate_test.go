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

// TestRefreshTokenRepository_Rotate_MidFailure_RollsBack proves rotation is
// atomic: when the successor insert fails, the whole transaction rolls back
// and the old token stays valid — there is no window where the old token is
// consumed (rotated) without a successor existing.
func TestRefreshTokenRepository_Rotate_MidFailure_RollsBack(t *testing.T) {
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

	// Attempt to rotate A onto B's already-taken hash: the successor insert
	// violates the token_hash UNIQUE constraint and the tx must roll back.
	_, err := repo.Rotate(ctx, hashA, hashB, now.Add(7*24*time.Hour))
	require.Error(t, err, "inserting a duplicate successor hash must fail")

	// A is untouched: still findable, not rotated, not revoked.
	got, err := repo.FindByHash(ctx, hashA)
	require.NoError(t, err)
	require.NotNil(t, got, "old token must survive a failed rotation")
	assert.Nil(t, got.RotatedAt, "old token must not be marked rotated on rollback")
	assert.Nil(t, got.RevokedAt, "old token must not be revoked on rollback")

	// B is untouched too.
	gotB, err := repo.FindByHash(ctx, hashB)
	require.NoError(t, err)
	require.NotNil(t, gotB)
}

// TestRefreshTokenRepository_Rotate_ConcurrentRace runs two simultaneous
// rotations of the SAME token and asserts the deterministic outcome set:
// exactly one succeeds and the loser receives ports.ErrTokenReuse.
//
// Chosen semantics (locked by 08-01, audit P0-5): with SELECT ... FOR UPDATE
// the two transactions serialize — whichever commits first rotates the token;
// the loser then observes the committed rotation and is indistinguishable from
// an attacker replay, so it revokes the whole family. Distinguishing a
// legitimate multi-tab race loser from an attacker replay (e.g. via client
// fingerprinting) is audit item T9 and is explicitly out of scope. The
// outcome set below holds for ANY goroutine scheduling — no wall-clock timing
// is involved, only the FOR UPDATE row lock.
func TestRefreshTokenRepository_Rotate_ConcurrentRace(t *testing.T) {
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

	start := make(chan struct{})
	type result struct {
		newHash string
		consumed *ports.RefreshToken
		err      error
	}
	results := make(chan result, 2)
	newHashes := []string{uuid.New().String(), uuid.New().String()}
	for _, newHash := range newHashes {
		go func(h string) {
			<-start
			consumed, err := repo.Rotate(ctx, oldHash, h, now.Add(7*24*time.Hour))
			results <- result{newHash: h, consumed: consumed, err: err}
		}(newHash)
	}
	close(start)

	successes := 0
	var winnerHash string
	var loserErr error
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil {
			successes++
			require.NotNil(t, r.consumed, "winner must return the consumed token")
			winnerHash = r.newHash
		} else {
			loserErr = r.err
		}
	}
	assert.Equal(t, 1, successes, "exactly one concurrent rotation must succeed")
	require.ErrorIs(t, loserErr, ports.ErrTokenReuse, "the race loser follows the replay path")

	// Post-race consistency: the old token is consumed and the winner's
	// successor is revoked by the loser's family revocation — the family is
	// consistently dead, never partially rotated.
	old, err := repo.FindByHash(ctx, oldHash)
	require.NoError(t, err)
	assert.Nil(t, old, "old token must be consumed after the race")

	winner, err := repo.FindByHash(ctx, winnerHash)
	require.NoError(t, err)
	assert.Nil(t, winner, "winner's successor must be revoked by the race loser (strict reuse semantics)")
}

// compile-time guard: the real repo satisfies the extended interface.
var _ ports.RefreshTokenRepository = (*RefreshTokenRepository)(nil)
