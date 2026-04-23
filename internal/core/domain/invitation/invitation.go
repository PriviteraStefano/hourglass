package invitation

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation has expired")
	ErrInvitationUsed     = errors.New("invitation already used")
)

type Invitation struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	InviteToken    string
	Email          string
	Status         InvitationStatus
	ExpiresAt      time.Time
	CreatedBy      string
	CreatedAt      time.Time
}

type InvitationStatus string

const (
	InvitationStatusPending InvitationStatus = "pending"
	InvitationStatusExpired InvitationStatus = "expired"
	InvitationStatusUsed    InvitationStatus = "used"
)

func (i *Invitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt) || i.Status == InvitationStatusExpired
}

func (i *Invitation) IsUsable() bool {
	return i.Status == InvitationStatusPending && !i.IsExpired()
}

type CreateInvitationRequest struct {
	OrganizationID uuid.UUID
	Email          string
	ExpiresInDays  int
}

type AcceptInvitationRequest struct {
	Token    string
	Email    string
	Username string
	Password string
}
