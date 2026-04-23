package ports

import (
	"context"

	"github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
)

type InvitationRepository interface {
	Create(ctx context.Context, inv *invitation.Invitation) (*invitation.Invitation, error)
	FindByCode(ctx context.Context, code string) (*invitation.Invitation, error)
	FindByToken(ctx context.Context, token string) (*invitation.Invitation, error)
	Update(ctx context.Context, inv *invitation.Invitation) (*invitation.Invitation, error)
}
