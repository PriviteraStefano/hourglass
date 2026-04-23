package surrealdb

import (
	"time"

	"github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type SurrealUser struct {
	ID           models.RecordID `json:"id,omitempty"`
	Email        string          `json:"email"`
	Username     string          `json:"username"`
	Firstname    string          `json:"firstname"`
	Lastname     string          `json:"lastname"`
	Name         string          `json:"name"`
	PasswordHash string          `json:"password_hash"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type SurrealUserCount struct {
	Count int `json:"count"`
}

func (u *SurrealUser) ToDomain() *auth.User {
	if u == nil {
		return nil
	}
	user := &auth.User{}
	user.ID = recordIDToUUID(u.ID)
	user.Email = u.Email
	user.Username = u.Username
	user.FirstName = u.Firstname
	user.LastName = u.Lastname
	user.Name = u.Name
	user.PasswordHash = u.PasswordHash
	user.IsActive = u.IsActive
	user.CreatedAt = u.CreatedAt
	user.UpdatedAt = u.UpdatedAt
	return user
}

func SurrealUserFromDomain(u *auth.User) *SurrealUser {
	if u == nil {
		return nil
	}
	return &SurrealUser{
		ID:           uuidToRecordID("users", u.ID),
		Email:        u.Email,
		Username:     u.Username,
		Firstname:    u.FirstName,
		Lastname:     u.LastName,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

type SurrealOrganization struct {
	ID                    models.RecordID        `json:"id,omitempty"`
	Name                  string                 `json:"name"`
	Slug                  string                 `json:"slug"`
	Description           string                 `json:"description"`
	FinancialCutoffDays   int                    `json:"financial_cutoff_days"`
	FinancialCutoffConfig map[string]interface{} `json:"financial_cutoff_config"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

func (o *SurrealOrganization) ToDomain() *auth.Organization {
	if o == nil {
		return nil
	}
	org := &auth.Organization{}
	org.ID = recordIDToUUID(o.ID)
	org.Name = o.Name
	org.Slug = o.Slug
	org.Description = o.Description
	org.FinancialCutoffDays = o.FinancialCutoffDays
	org.FinancialCutoffConfig = o.FinancialCutoffConfig
	org.CreatedAt = o.CreatedAt
	org.UpdatedAt = o.UpdatedAt
	return org
}

func SurrealOrganizationFromDomain(o *auth.Organization) *SurrealOrganization {
	if o == nil {
		return nil
	}
	return &SurrealOrganization{
		ID:                    uuidToRecordID("organizations", o.ID),
		Name:                  o.Name,
		Slug:                  o.Slug,
		Description:           o.Description,
		FinancialCutoffDays:   o.FinancialCutoffDays,
		FinancialCutoffConfig: o.FinancialCutoffConfig,
		CreatedAt:             o.CreatedAt,
		UpdatedAt:             o.UpdatedAt,
	}
}

type SurrealRefreshToken struct {
	ID        models.RecordID `json:"id,omitempty"`
	UserID    models.RecordID `json:"user_id"`
	TokenHash string          `json:"token_hash"`
	ExpiresAt time.Time       `json:"expires_at"`
	RevokedAt *time.Time      `json:"revoked_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type QueryResponse[T any] struct {
	Result []T `json:"result"`
}

type QueryResultWrapper struct {
	Result []map[string]any `json:"result"`
}
