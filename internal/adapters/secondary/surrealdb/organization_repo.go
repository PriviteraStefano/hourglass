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
	_, err := sdb.Create[SurrealOrganization](ctx, r.db, models.Table("organizations"), so)
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
