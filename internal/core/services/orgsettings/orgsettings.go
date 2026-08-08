// Package orgsettingssvc implements the org policy plane's business
// semantics (D-13-18..24, ADR-BE-018): the settings GET/PUT surface with
// code-enforced known-key validation, the manager+ write gate, and the
// ResolvePlanningMode precedence the direction service mode gate (13-07)
// consumes.
//
// The repository is a faithful store; every invariant a user can observe is
// decided here:
//
//   - D-13-18: PUT validates every key against the domain vocabulary BEFORE
//     any write — unknown key → ErrUnknownKey, failing validator →
//     ErrInvalidValue (both → 400). This is the ONLY gate on JSONB values
//     (CHECK on JSONB infeasible — T-13-11).
//   - D-13-23/T-13-10: the manager+ gate runs FIRST (role != manager →
//     ErrForbidden → 403) — no validation or write happens for non-managers.
//   - D-13-22/T-13-12: every key write builds its {key, before, after}
//     audit row service-side and hands it to the repo for the in-tx write.
//   - D-13-24: GET overlays code-level defaults for absent known keys
//     (planning_daily_hours → DefaultDailyHours); no seed rows exist
//     (ADR-BE-018 §8.3).
//   - D-13-19: ResolvePlanningMode resolves membership planning_mode
//     override → org default planning_mode key → ModeManagerPlanned
//     fallback (conservative reading of D-13-19/D-13-20, ADR-BE-018 §8).
//
// Never block on deadline/horizon values (D-13-20/21): this service stores
// and permission-gates only — no enforcement exists in the backend.
package orgsettingssvc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/orgsettings"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// Service implements the org_settings plane (D-13-18..23): known-key
// validated GET/PUT over the generic key/value store plus the
// ResolvePlanningMode seam (D-13-19) for the direction service mode gate
// (13-07).
type Service struct {
	repo    ports.OrgSettingsRepository
	orgRepo ports.OrganizationRepository // membership planning_mode override (D-13-19)
}

func NewService(repo ports.OrgSettingsRepository, orgRepo ports.OrganizationRepository) *Service {
	return &Service{repo: repo, orgRepo: orgRepo}
}

// ResolvePlanningMode resolves the effective planning mode for one employee
// in an org (D-13-19, ADR-BE-018 §8): the membership planning_mode override
// wins, then the org default planning_mode key, then the conservative
// ModeManagerPlanned fallback when neither is set. This is the seam the
// direction service mode gate (13-07) consumes. Stored values must be in the
// closed mode vocabulary — a mismatched value is ErrInvalidValue (the store
// is JSONB-unvalidated, T-13-11); the resolution surfaces it rather than
// silently defaulting.
func (s *Service) ResolvePlanningMode(ctx context.Context, orgID, employeeID uuid.UUID) (string, error) {
	m, err := s.orgRepo.GetMembership(ctx, employeeID, orgID)
	if err != nil {
		return "", err
	}
	if m != nil && m.PlanningMode != nil {
		mode := *m.PlanningMode
		switch mode {
		case orgsettings.ModeManagerPlanned, orgsettings.ModeSelfPlanned:
			return mode, nil
		}
		return "", orgsettings.ErrInvalidValue
	}

	raw, err := s.repo.Get(ctx, orgID, orgsettings.KeyPlanningMode)
	if err != nil {
		return "", err
	}
	if raw != nil {
		var mode string
		if err := json.Unmarshal(raw, &mode); err != nil {
			return "", orgsettings.ErrInvalidValue
		}
		switch mode {
		case orgsettings.ModeManagerPlanned, orgsettings.ModeSelfPlanned:
			return mode, nil
		}
		return "", orgsettings.ErrInvalidValue
	}

	// D-13-19/D-13-20 conservative reading: the org is manager-planned
	// when nothing is set (ADR-BE-018 §8).
	return orgsettings.ModeManagerPlanned, nil
}

// Get returns the org's settings with code-level defaults applied for absent
// known keys (D-13-24, ADR-BE-018 §8.3): planning_daily_hours defaults to
// DefaultDailyHours; other absent keys are omitted (no seed rows needed).
// Stored values are returned verbatim.
func (s *Service) Get(ctx context.Context, orgID uuid.UUID) (map[string]any, error) {
	raw, err := s.repo.List(ctx, orgID)
	if err != nil {
		return nil, err
	}

	out := make(map[string]any, len(raw)+1)
	for key, value := range raw {
		var v any
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, orgsettings.ErrInvalidValue
		}
		out[key] = v
	}
	if _, ok := raw[orgsettings.KeyPlanningDailyHours]; !ok {
		out[orgsettings.KeyPlanningDailyHours] = orgsettings.DefaultDailyHours
	}
	return out, nil
}

// Put validates and stores one-or-many {key: value} pairs (D-13-18/23):
// manager+ gate first (T-13-10), then every key validated against the
// domain vocabulary (T-13-11), then each key upserted with its {key, before,
// after} audit row in the same tx (D-13-22). Returns the post-state map
// (same shape as Get).
func (s *Service) Put(ctx context.Context, orgID, actorID uuid.UUID, role string, values map[string]json.RawMessage) (map[string]any, error) {
	// D-13-23/T-13-10: manager+ gate BEFORE any validation or write.
	if role != string(models.RoleManager) {
		return nil, orgsettings.ErrForbidden
	}

	// D-13-18: validate the full key set before any write — an invalid
	// batch never partially commits.
	for key, value := range values {
		if err := orgsettings.ValidateKey(key, value); err != nil {
			return nil, err
		}
	}

	for key, value := range values {
		before, err := s.repo.Get(ctx, orgID, key)
		if err != nil {
			return nil, err
		}

		var beforeVal any
		if before != nil {
			if err := json.Unmarshal(before, &beforeVal); err != nil {
				return nil, orgsettings.ErrInvalidValue
			}
		}
		var afterVal any
		if err := json.Unmarshal(value, &afterVal); err != nil {
			return nil, orgsettings.ErrInvalidValue
		}

		actor := actorID
		err = s.repo.Upsert(ctx, orgID, key, value, &audit.AuditLog{
			OrgID:      orgID,
			EntityType: orgsettings.AuditEntityOrgSettings,
			EntityID:   orgID, // audit_logs.entity_id is the ORG id (D-13-22)
			Action:     orgsettings.AuditActionSettingsUpdated,
			ActorID:    &actor,
			Payload:    map[string]any{"key": key, "before": beforeVal, "after": afterVal},
			CreatedAt:  time.Now().UTC(),
		})
		if err != nil {
			return nil, err
		}
	}

	return s.Get(ctx, orgID)
}
