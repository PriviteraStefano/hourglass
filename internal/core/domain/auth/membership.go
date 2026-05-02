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