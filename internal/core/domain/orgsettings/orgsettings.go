// Package orgsettings implements the org-level planning policy domain
// (D-13-18..24, ADR-BE-018 §4): the settings key vocabulary with
// code-enforced validation per known key — CHECK on JSONB isn't feasible,
// so the vocabulary lives here as exported constants (T-13-07: repos,
// services and handlers reference constants, never string literals).
package orgsettings

import (
	"encoding/json"
	"time"
)

// Closed settings key vocabulary (A7, ADR-BE-018 §4): org_settings is a
// generic key/value store (D-13-18); these four keys are code-validated per
// known key, anything else is ErrUnknownKey (→ 400).
const (
	KeyPlanningDailyHours = "planning_daily_hours"
	KeyPlanningDeadline   = "planning_deadline"
	KeyPlanningHorizon    = "planning_horizon"
	KeyPlanningMode       = "planning_mode"
)

// Closed planning-mode vocabulary (D-13-19, 13-UI-SPEC): the org default
// plus the nullable per-employee override on
// organization_memberships.planning_mode (NULL → org default).
const (
	ModeManagerPlanned = "manager_planned"
	ModeSelfPlanned    = "self_planned"
)

// Closed planning-horizon vocabulary (D-13-21): stored, never enforced
// (the dynamic period is UI cadence + policy, not a schema dimension).
const (
	HorizonDay   = "day"
	HorizonWeek  = "week"
	HorizonMonth = "month"
)

// DefaultDailyHours is the code-level default for KeyPlanningDailyHours
// when the key is absent (D-13-24): no seed rows exist (ADR-BE-018 §8.3 —
// the coverage read-model applies it when the key is absent).
const DefaultDailyHours = 8.0

// Audit vocabulary (ADR-BE-018 §3): settings changes write entity_type
// 'org_settings' with entity_id = the ORG id (audit_logs.entity_id is UUID
// NOT NULL — migration 017) and action 'settings-updated'; payload
// {key, before, after} (D-13-22).
const (
	AuditEntityOrgSettings     = "org_settings"
	AuditActionSettingsUpdated = "settings-updated"
)

// ValidatorFn validates one raw JSON value for a known settings key.
type ValidatorFn func(value json.RawMessage) error

// knownKeys is the code-enforced vocabulary (D-13-18): every accepted key
// has a per-key validator; anything else is ErrUnknownKey. The map is
// consulted through IsKnownKey/ValidateKey so callers never bypass the
// vocabulary.
var knownKeys = map[string]ValidatorFn{
	KeyPlanningDailyHours: validateDailyHours,
	KeyPlanningDeadline:   validateDeadline,
	KeyPlanningHorizon:    validateHorizon,
	KeyPlanningMode:       validateMode,
}

func validateDailyHours(value json.RawMessage) error {
	var hours float64
	if err := json.Unmarshal(value, &hours); err != nil {
		return ErrInvalidValue
	}
	if hours <= 0 {
		return ErrInvalidValue
	}
	return nil
}

func validateDeadline(value json.RawMessage) error {
	var s string
	if err := json.Unmarshal(value, &s); err != nil {
		return ErrInvalidValue
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return ErrInvalidValue
	}
	return nil
}

func validateHorizon(value json.RawMessage) error {
	var s string
	if err := json.Unmarshal(value, &s); err != nil {
		return ErrInvalidValue
	}
	switch s {
	case HorizonDay, HorizonWeek, HorizonMonth:
		return nil
	}
	return ErrInvalidValue
}

func validateMode(value json.RawMessage) error {
	var s string
	if err := json.Unmarshal(value, &s); err != nil {
		return ErrInvalidValue
	}
	switch s {
	case ModeManagerPlanned, ModeSelfPlanned:
		return nil
	}
	return ErrInvalidValue
}

// IsKnownKey reports whether key has a code-enforced validator (D-13-18).
func IsKnownKey(key string) bool {
	_, ok := knownKeys[key]
	return ok
}

// ValidateKey validates a raw JSON value against the known key's validator
// (D-13-18). Returns ErrUnknownKey when the key has no validator and
// ErrInvalidValue when the value fails validation — the service (13-04)
// maps both to 400.
func ValidateKey(key string, value json.RawMessage) error {
	fn, ok := knownKeys[key]
	if !ok {
		return ErrUnknownKey
	}
	return fn(value)
}
