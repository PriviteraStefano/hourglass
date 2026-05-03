package surrealdb

import (
	"context"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type OrganizationRepository struct {
	db *sdb.DB
}

func NewOrganizationRepository(db *sdb.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

func (r *OrganizationRepository) Add(ctx context.Context, org *auth.Organization) error {
	so := SurrealOrganizationFromDomain(org)
	_, err := sdb.Create[[]SurrealOrganization](ctx, r.db, models.Table("organizations"), so)
	if err != nil {
		return wrapErr(err, "create organization")
	}
	return nil
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*auth.Organization, error) {
	recordID := uuidToRecordID("organizations", id)
	result, err := sdb.Select[SurrealOrganization](ctx, r.db, recordID)
	if err != nil {
		return nil, wrapErr(err, "get organization by id")
	}
	return result.ToDomain(), nil
}

func (r *OrganizationRepository) GetMembership(ctx context.Context, userID, orgID uuid.UUID) (*auth.OrganizationMembership, error) {
	results, err := sdb.Query[[]SurrealOrganizationMembership](ctx, r.db,
		"SELECT * FROM organization_memberships WHERE user_id = $user_id AND organization_id = $org_id LIMIT 1",
		map[string]any{
			"user_id": "users:" + userID.String(),
			"org_id":  "organizations:" + orgID.String(),
		})
	if err != nil {
		return nil, wrapErr(err, "get membership")
	}
	if results == nil || len(*results) == 0 {
		return nil, nil
	}
	resultData := *results
	if len(resultData) == 0 {
		return nil, nil
	}
	items := resultData[0].Result
	if len(items) == 0 {
		return nil, nil
	}
	return items[0].ToDomain(), nil
}

func (r *OrganizationRepository) AddMembership(ctx context.Context, membership *auth.OrganizationMembership) error {
	sm := SurrealOrganizationMembershipFromDomain(membership)
	_, err := sdb.Create[[]SurrealOrganizationMembership](ctx, r.db, models.Table("organization_memberships"), sm)
	if err != nil {
		return wrapErr(err, "create membership")
	}
	return nil
}
