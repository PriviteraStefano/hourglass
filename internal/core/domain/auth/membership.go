package auth

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationMembership struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           string
	IsActive       bool
	InvitedBy      *uuid.UUID
	InvitedAt      *time.Time
	ActivatedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	// PlanningMode is the nullable per-member planning-mode override
	// (D-13-19, migration 022): NULL falls back to the org default
	// planning_mode key, which itself falls back to manager_planned
	// (ADR-BE-018 §8). Consumed by orgsettings.ResolvePlanningMode (13-04)
	// and the direction service mode gate (13-07).
	PlanningMode *string `json:"planning_mode,omitempty"`
	// ValidFrom/ValidUntil are the employment validity window (D-2,
	// migration 012): NULL valid_until = open-ended. Consumed by the
	// direction service validity warnings (13-07).
	ValidFrom  *time.Time `json:"valid_from,omitempty"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
}

func NewOrganizationMembership(userID, organizationID uuid.UUID, role string) *OrganizationMembership {
	now := time.Now()
	activated := now
	return &OrganizationMembership{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: organizationID,
		Role:           role,
		IsActive:       true,
		ActivatedAt:    &activated,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}