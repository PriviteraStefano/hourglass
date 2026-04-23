package surrealdb

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/invitation"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type InvitationRepository struct {
	db *sdb.DB
}

func NewInvitationRepository(db *sdb.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

func (r *InvitationRepository) Create(ctx context.Context, inv *invitation.Invitation) (*invitation.Invitation, error) {
	su := surrealInvitationFromDomain(inv)
	created, err := sdb.Create[SurrealInvitation](ctx, r.db, models.Table("invitations"), su)
	if err != nil {
		return nil, wrapErr(err, "create invitation")
	}
	return created.ToDomain(), nil
}

func (r *InvitationRepository) FindByCode(ctx context.Context, code string) (*invitation.Invitation, error) {
	results, err := sdb.Query[[]SurrealInvitation](ctx, r.db,
		"SELECT * FROM invitations WHERE code = $code LIMIT 1",
		map[string]interface{}{"code": code})
	if err != nil {
		return nil, wrapErr(err, "find invitation by code")
	}
	if results == nil || len(*results) == 0 {
		return nil, invitation.ErrInvitationNotFound
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		return nil, invitation.ErrInvitationNotFound
	}
	return resultItems[0].ToDomain(), nil
}

func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*invitation.Invitation, error) {
	results, err := sdb.Query[[]SurrealInvitation](ctx, r.db,
		"SELECT * FROM invitations WHERE invite_token = $invite_token LIMIT 1",
		map[string]interface{}{"invite_token": token})
	if err != nil {
		return nil, wrapErr(err, "find invitation by token")
	}
	if results == nil || len(*results) == 0 {
		return nil, invitation.ErrInvitationNotFound
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		return nil, invitation.ErrInvitationNotFound
	}
	return resultItems[0].ToDomain(), nil
}

func (r *InvitationRepository) Update(ctx context.Context, inv *invitation.Invitation) (*invitation.Invitation, error) {
	recordID := uuidToRecordID("invitations", inv.ID)
	data := map[string]interface{}{
		"status": inv.Status,
	}
	result, err := sdb.Merge[SurrealInvitation](ctx, r.db, recordID, data)
	if err != nil {
		return nil, wrapErr(err, "update invitation")
	}
	return result.ToDomain(), nil
}

func (r *InvitationRepository) FindByID(ctx context.Context, id uuid.UUID) (*invitation.Invitation, error) {
	recordID := uuidToRecordID("invitations", id)
	result, err := sdb.Select[SurrealInvitation](ctx, r.db, recordID)
	if err != nil {
		return nil, wrapErr(err, "find invitation by id")
	}
	return result.ToDomain(), nil
}

func (su *SurrealInvitation) ToDomain() *invitation.Invitation {
	if su == nil {
		return nil
	}
	return &invitation.Invitation{
		ID:             recordIDToUUID(su.ID),
		OrganizationID: recordIDToUUID(su.OrganizationID),
		Code:           su.Code,
		InviteToken:    su.InviteToken,
		Email:          su.Email,
		Status:         invitation.InvitationStatus(su.Status),
		ExpiresAt:      su.ExpiresAt,
		CreatedBy:      su.CreatedBy,
		CreatedAt:      su.CreatedAt,
	}
}

func surrealInvitationFromDomain(inv *invitation.Invitation) *SurrealInvitation {
	if inv == nil {
		return nil
	}
	return &SurrealInvitation{
		ID:             uuidToRecordID("invitations", inv.ID),
		OrganizationID: uuidToRecordID("organizations", inv.OrganizationID),
		Code:           inv.Code,
		InviteToken:    inv.InviteToken,
		Email:          inv.Email,
		Status:         string(inv.Status),
		ExpiresAt:      inv.ExpiresAt,
		CreatedBy:      inv.CreatedBy,
		CreatedAt:      inv.CreatedAt,
	}
}
