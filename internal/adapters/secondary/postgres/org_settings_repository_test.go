package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/orgsettings"
	"github.com/stretchr/testify/require"
)

// orgSettingsAudit builds the 'settings-updated' audit row the service
// passes to Upsert (entity_type pinned to orgsettings.AuditEntityOrgSettings,
// entity_id = the ORG id, payload {key, before, after} — D-13-22,
// ADR-BE-018 §3).
func orgSettingsAudit(orgID, actorID uuid.UUID, key string, before, after any, now time.Time) *audit.AuditLog {
	return &audit.AuditLog{
		OrgID:      orgID,
		EntityType: orgsettings.AuditEntityOrgSettings,
		EntityID:   orgID,
		Action:     orgsettings.AuditActionSettingsUpdated,
		ActorID:    &actorID,
		Payload:    map[string]any{"key": key, "before": before, "after": after},
		CreatedAt:  now,
	}
}

// countOrgSettingsAudits is unused — kept out: the suite queries the pool
// directly.
// ---------------------------------------------------------------------------
// Upsert — value-replacement semantics + the in-tx audit row (D-13-22)
// ---------------------------------------------------------------------------

func TestOrgSettingsRepository_Upsert_CreatesAndUpdates(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrgSettingsRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	actorID := seedUser(t, pool, now)

	err := repo.Upsert(ctx, orgID, orgsettings.KeyPlanningDailyHours, json.RawMessage(`8.0`),
		orgSettingsAudit(orgID, actorID, orgsettings.KeyPlanningDailyHours, nil, 8.0, now))
	require.NoError(t, err)

	// First write creates the row.
	got, err := repo.Get(ctx, orgID, orgsettings.KeyPlanningDailyHours)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.JSONEq(t, `8.0`, string(got))

	// Second write replaces the value (ON CONFLICT DO UPDATE).
	err = repo.Upsert(ctx, orgID, orgsettings.KeyPlanningDailyHours, json.RawMessage(`7.5`),
		orgSettingsAudit(orgID, actorID, orgsettings.KeyPlanningDailyHours, 8.0, 7.5, now))
	require.NoError(t, err)

	got, err = repo.Get(ctx, orgID, orgsettings.KeyPlanningDailyHours)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.JSONEq(t, `7.5`, string(got))

	var rowCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM org_settings WHERE org_id = $1`, orgID).Scan(&rowCount)
	require.NoError(t, err)
	require.Equal(t, 1, rowCount, "upsert must not duplicate rows (PK org_id, key)")
}

func TestOrgSettingsRepository_Upsert_WritesAuditInSameTx(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrgSettingsRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	actorID := seedUser(t, pool, now)

	err := repo.Upsert(ctx, orgID, orgsettings.KeyPlanningDailyHours, json.RawMessage(`7.5`),
		orgSettingsAudit(orgID, actorID, orgsettings.KeyPlanningDailyHours, nil, 7.5, now))
	require.NoError(t, err)

	// The audit row was written in the same tx: entity_type='org_settings',
	// entity_id = the ORG id, action='settings-updated', payload {key, before,
	// after} (D-13-22, ADR-BE-018 §3).
	var payload []byte
	var actorScan uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT payload, actor_id FROM audit_logs
		 WHERE org_id = $1 AND entity_type = 'org_settings' AND entity_id = $1 AND action = 'settings-updated'`,
		orgID).Scan(&payload, &actorScan)
	require.NoError(t, err, "exactly one settings-updated audit row must exist")
	require.Equal(t, actorID, actorScan)

	var auditPayload map[string]any
	require.NoError(t, json.Unmarshal(payload, &auditPayload))
	require.Equal(t, orgsettings.KeyPlanningDailyHours, auditPayload["key"])
	require.Nil(t, auditPayload["before"], "first write has no before value")
	require.Equal(t, 7.5, auditPayload["after"])
}

func TestOrgSettingsRepository_Upsert_FailedTxLeavesNoRowAndNoAudit(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrgSettingsRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	actorID := seedUser(t, pool, now)

	// A foreign-key violation (org does not exist) fails the insert AFTER the
	// audit was attempted in the same tx — the rollback must discard both
	// (T-13-12: no silent unlogged change, and no orphan audit for a write
	// that never landed).
	ghostOrg := uuid.New()
	err := repo.Upsert(ctx, ghostOrg, orgsettings.KeyPlanningDailyHours, json.RawMessage(`8.0`),
		orgSettingsAudit(ghostOrg, actorID, orgsettings.KeyPlanningDailyHours, nil, 8.0, now))
	require.Error(t, err, "upsert against a nonexistent org must fail")

	var valueCount, auditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM org_settings WHERE org_id = $1`, ghostOrg).Scan(&valueCount)
	require.NoError(t, err)
	require.Zero(t, valueCount, "failed upsert must not leave a value row")
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE org_id = $1`, ghostOrg).Scan(&auditCount)
	require.NoError(t, err)
	require.Zero(t, auditCount, "failed upsert must roll back its audit row (T-13-12)")
}

// ---------------------------------------------------------------------------
// List / Get — faithful store shape
// ---------------------------------------------------------------------------

func TestOrgSettingsRepository_List_ReturnsOnlyStoredKeys(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrgSettingsRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)
	otherOrgID := seedOrg(t, pool, now)

	require.NoError(t, repo.Upsert(ctx, orgID, orgsettings.KeyPlanningDailyHours, json.RawMessage(`8.0`), nil))
	require.NoError(t, repo.Upsert(ctx, orgID, orgsettings.KeyPlanningHorizon, json.RawMessage(`"month"`), nil))
	require.NoError(t, repo.Upsert(ctx, otherOrgID, orgsettings.KeyPlanningMode, json.RawMessage(`"self_planned"`), nil))

	got, err := repo.List(ctx, orgID)
	require.NoError(t, err)
	require.Len(t, got, 2, "List must return only this org's stored keys")
	require.JSONEq(t, `8.0`, string(got[orgsettings.KeyPlanningDailyHours]))
	require.JSONEq(t, `"month"`, string(got[orgsettings.KeyPlanningHorizon]))
	_, hasOther := got[orgsettings.KeyPlanningMode]
	require.False(t, hasOther, "other orgs' keys must not leak")

	// Empty org → empty map (absent keys are covered by code-level defaults).
	empty, err := repo.List(ctx, seedOrg(t, pool, now))
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestOrgSettingsRepository_Get_AbsentKeyReturnsNil(t *testing.T) {
	pool := TestPool(t)
	SetupTestSchema(t, pool)
	t.Cleanup(func() { TeardownTestSchema(t, pool) })

	repo := NewOrgSettingsRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC()
	orgID := seedOrg(t, pool, now)

	got, err := repo.Get(ctx, orgID, orgsettings.KeyPlanningDailyHours)
	require.NoError(t, err)
	require.Nil(t, got, "absent key is (nil, nil) — absence is not an error")

	require.NoError(t, repo.Upsert(ctx, orgID, orgsettings.KeyPlanningDailyHours, json.RawMessage(`8.0`), nil))
	got, err = repo.Get(ctx, orgID, orgsettings.KeyPlanningDailyHours)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.JSONEq(t, `8.0`, string(got))
}
