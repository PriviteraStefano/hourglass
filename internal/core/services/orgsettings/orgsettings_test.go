package orgsettingssvc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/orgsettings"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture wires the orgsettings service against the hermetic testdata mocks
// (13-03): MockOrgSettingsRepo for the store, MockOrgRepo for membership
// reads (the planning_mode override seam, D-13-19).
type fixture struct {
	svc     *Service
	repo    *testdata.MockOrgSettingsRepo
	orgRepo *testdata.MockOrgRepo
}

func setupOrgSettings(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		repo:    &testdata.MockOrgSettingsRepo{},
		orgRepo: &testdata.MockOrgRepo{},
	}
	f.svc = NewService(f.repo, f.orgRepo)
	return f
}

func raw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func strPtr(s string) *string { return &s }

// seedMembership anchors the employee in the mock org repo with the given
// planning_mode override (nil = no override).
func (f *fixture) seedMembership(userID, orgID uuid.UUID, planningMode *string) {
	if f.orgRepo.Memberships == nil {
		f.orgRepo.Memberships = make(map[string]*auth.OrganizationMembership)
	}
	f.orgRepo.Memberships[userID.String()+":"+orgID.String()] = &auth.OrganizationMembership{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "employee",
		IsActive:       true,
		PlanningMode:   planningMode,
	}
}

// ---------------------------------------------------------------------------
// TestOrgSettingsService_Put — D-13-18/23 gate matrix + D-13-22 audit shape
// ---------------------------------------------------------------------------

func TestOrgSettingsService_Put(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	ctx := context.Background()

	t.Run("unknown key is rejected with ErrUnknownKey", func(t *testing.T) {
		f := setupOrgSettings(t)
		_, err := f.svc.Put(ctx, orgID, actorID, "manager", map[string]json.RawMessage{"not_a_known_key": raw(1)})
		require.ErrorIs(t, err, orgsettings.ErrUnknownKey)
		assert.Empty(t, f.repo.Audits, "no audit must be written for a rejected key")
	})

	t.Run("invalid planning_horizon value is rejected with ErrInvalidValue", func(t *testing.T) {
		f := setupOrgSettings(t)
		_, err := f.svc.Put(ctx, orgID, actorID, "manager", map[string]json.RawMessage{orgsettings.KeyPlanningHorizon: raw("year")})
		require.ErrorIs(t, err, orgsettings.ErrInvalidValue)
		assert.Empty(t, f.repo.Audits, "no audit must be written for an invalid value")
	})

	t.Run("non-manager role is rejected with ErrForbidden before any write", func(t *testing.T) {
		f := setupOrgSettings(t)
		for _, role := range []string{"employee", "finance", "customer"} {
			_, err := f.svc.Put(ctx, orgID, actorID, role, map[string]json.RawMessage{orgsettings.KeyPlanningDailyHours: raw(8.0)})
			require.ErrorIs(t, err, orgsettings.ErrForbidden, "role %s must be forbidden", role)
		}
		assert.Empty(t, f.repo.Values, "no value may be written for a forbidden role")
		assert.Empty(t, f.repo.Audits)
	})

	t.Run("manager PUT writes one audit row per key with {key, before, after}", func(t *testing.T) {
		f := setupOrgSettings(t)
		values := map[string]json.RawMessage{
			orgsettings.KeyPlanningDailyHours: raw(7.5),
			orgsettings.KeyPlanningHorizon:    raw("month"),
		}
		got, err := f.svc.Put(ctx, orgID, actorID, "manager", values)
		require.NoError(t, err)
		require.Equal(t, 7.5, got[orgsettings.KeyPlanningDailyHours])
		require.Equal(t, "month", got[orgsettings.KeyPlanningHorizon])

		require.Len(t, f.repo.Audits, 2, "one audit row per written key (D-13-22)")
		seen := make(map[string]bool)
		for _, a := range f.repo.Audits {
			assert.Equal(t, orgID, a.OrgID)
			assert.Equal(t, orgsettings.AuditEntityOrgSettings, a.EntityType)
			assert.Equal(t, orgID, a.EntityID, "audit entity_id is the ORG id")
			assert.Equal(t, orgsettings.AuditActionSettingsUpdated, a.Action)
			require.NotNil(t, a.ActorID)
			assert.Equal(t, actorID, *a.ActorID)
			require.NotNil(t, a.Payload)
			key, ok := a.Payload["key"].(string)
			require.True(t, ok, "payload must carry the key")
			assert.Nil(t, a.Payload["before"], "first write has no before value")
			seen[key] = true
		}
		assert.True(t, seen[orgsettings.KeyPlanningDailyHours], "audit for planning_daily_hours")
		assert.True(t, seen[orgsettings.KeyPlanningHorizon], "audit for planning_horizon")
	})

	t.Run("second write records the previous value as before", func(t *testing.T) {
		f := setupOrgSettings(t)
		f.repo.Values = map[string]json.RawMessage{orgsettings.KeyPlanningDailyHours: raw(8.0)}

		_, err := f.svc.Put(ctx, orgID, actorID, "manager", map[string]json.RawMessage{orgsettings.KeyPlanningDailyHours: raw(7.5)})
		require.NoError(t, err)
		require.Len(t, f.repo.Audits, 1)
		require.NotNil(t, f.repo.Audits[0].Payload["before"])
		assert.Equal(t, 8.0, f.repo.Audits[0].Payload["before"])
		assert.Equal(t, 7.5, f.repo.Audits[0].Payload["after"])
	})
}

// ---------------------------------------------------------------------------
// TestOrgSettingsService_Get — D-13-24 code-level defaults overlay
// ---------------------------------------------------------------------------

func TestOrgSettingsService_Get(t *testing.T) {
	orgID := uuid.New()
	ctx := context.Background()

	t.Run("absent planning_daily_hours defaults to 8.0", func(t *testing.T) {
		f := setupOrgSettings(t)
		got, err := f.svc.Get(ctx, orgID)
		require.NoError(t, err)
		assert.Equal(t, 8.0, got[orgsettings.KeyPlanningDailyHours])
	})

	t.Run("stored values win over defaults and absent unknown keys are omitted", func(t *testing.T) {
		f := setupOrgSettings(t)
		f.repo.Values = map[string]json.RawMessage{
			orgsettings.KeyPlanningDailyHours: raw(7.5),
			orgsettings.KeyPlanningHorizon:    raw("week"),
		}
		got, err := f.svc.Get(ctx, orgID)
		require.NoError(t, err)
		assert.Equal(t, 7.5, got[orgsettings.KeyPlanningDailyHours])
		assert.Equal(t, "week", got[orgsettings.KeyPlanningHorizon])
		_, hasDeadline := got[orgsettings.KeyPlanningDeadline]
		_, hasMode := got[orgsettings.KeyPlanningMode]
		assert.False(t, hasDeadline, "absent non-defaulted keys are omitted (no seed rows)")
		assert.False(t, hasMode, "absent non-defaulted keys are omitted (no seed rows)")
	})
}

// ---------------------------------------------------------------------------
// TestOrgSettingsService_ResolvePlanningMode — D-13-19 precedence:
// membership override → org default key → manager_planned fallback
// ---------------------------------------------------------------------------

func TestOrgSettingsService_ResolvePlanningMode(t *testing.T) {
	orgID := uuid.New()
	employeeID := uuid.New()
	ctx := context.Background()

	t.Run("membership override wins over the org default", func(t *testing.T) {
		f := setupOrgSettings(t)
		f.seedMembership(employeeID, orgID, strPtr(orgsettings.ModeSelfPlanned))
		f.repo.Values = map[string]json.RawMessage{orgsettings.KeyPlanningMode: raw(orgsettings.ModeManagerPlanned)}

		mode, err := f.svc.ResolvePlanningMode(ctx, orgID, employeeID)
		require.NoError(t, err)
		assert.Equal(t, orgsettings.ModeSelfPlanned, mode)
	})

	t.Run("org default wins when no override", func(t *testing.T) {
		f := setupOrgSettings(t)
		f.seedMembership(employeeID, orgID, nil)
		f.repo.Values = map[string]json.RawMessage{orgsettings.KeyPlanningMode: raw(orgsettings.ModeSelfPlanned)}

		mode, err := f.svc.ResolvePlanningMode(ctx, orgID, employeeID)
		require.NoError(t, err)
		assert.Equal(t, orgsettings.ModeSelfPlanned, mode)
	})

	t.Run("manager_planned fallback when neither override nor key is set", func(t *testing.T) {
		f := setupOrgSettings(t)
		f.seedMembership(employeeID, orgID, nil)

		mode, err := f.svc.ResolvePlanningMode(ctx, orgID, employeeID)
		require.NoError(t, err)
		assert.Equal(t, orgsettings.ModeManagerPlanned, mode)
	})

	t.Run("no membership row falls back to the org default", func(t *testing.T) {
		f := setupOrgSettings(t)
		f.repo.Values = map[string]json.RawMessage{orgsettings.KeyPlanningMode: raw(orgsettings.ModeManagerPlanned)}

		mode, err := f.svc.ResolvePlanningMode(ctx, orgID, employeeID)
		require.NoError(t, err)
		assert.Equal(t, orgsettings.ModeManagerPlanned, mode)
	})

	t.Run("invalid override value is rejected with ErrInvalidValue", func(t *testing.T) {
		f := setupOrgSettings(t)
		f.seedMembership(employeeID, orgID, strPtr("bogus_mode"))

		_, err := f.svc.ResolvePlanningMode(ctx, orgID, employeeID)
		require.ErrorIs(t, err, orgsettings.ErrInvalidValue)
	})

	t.Run("invalid stored org default value is rejected with ErrInvalidValue", func(t *testing.T) {
		f := setupOrgSettings(t)
		f.seedMembership(employeeID, orgID, nil)
		f.repo.Values = map[string]json.RawMessage{orgsettings.KeyPlanningMode: raw("bogus_mode")}

		_, err := f.svc.ResolvePlanningMode(ctx, orgID, employeeID)
		require.ErrorIs(t, err, orgsettings.ErrInvalidValue)
	})
}
