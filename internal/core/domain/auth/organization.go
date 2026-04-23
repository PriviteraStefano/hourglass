package auth

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID                    uuid.UUID
	Name                  string
	Slug                  string
	Description           string
	FinancialCutoffDays   int
	FinancialCutoffConfig map[string]interface{}
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func NewOrganization(name, slug, description string) *Organization {
	now := time.Now()
	return &Organization{
		ID:                  uuid.New(),
		Name:                name,
		Slug:                slug,
		Description:         description,
		FinancialCutoffDays: 7,
		FinancialCutoffConfig: map[string]interface{}{
			"cutoff_day_of_month": 28,
			"grace_days":          7,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
