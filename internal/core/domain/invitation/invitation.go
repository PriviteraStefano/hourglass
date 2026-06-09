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
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	Code           string          `json:"code"`
	InviteToken    string          `json:"invite_token"`
	Email          string          `json:"email"`
	Status         InvitationStatus `json:"status"`
	ExpiresAt      time.Time       `json:"expires_at"`
	CreatedBy      string          `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
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
	OrganizationID uuid.UUID `json:"organization_id"`
	Email          string    `json:"email"`
	ExpiresInDays  int       `json:"expires_in_days"`
	CreatedBy      uuid.UUID `json:"created_by"`
}

type AcceptInvitationRequest struct {
	Token    string `json:"token"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}
