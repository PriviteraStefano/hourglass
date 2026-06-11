package project

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

var (
	ErrProjectNotFound        = errors.New("project not found")
	ErrForbidden              = errors.New("forbidden")
	ErrInvalidRequest         = errors.New("invalid request")
	ErrAlreadyAdopted         = errors.New("already adopted")
	ErrUserNotFound           = errors.New("user not found")
	ErrHasActiveTimeEntries   = errors.New("project has active time entries")
	ErrHasActiveSubprojectEntries = errors.New("subproject has active time entries")
)

type Project struct {
	ID              uuid.UUID           `json:"id"`
	Name            string              `json:"name"`
	Type            models.ProjectType  `json:"type"`
	ContractID      uuid.UUID           `json:"contract_id"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	CreatedByOrgID  uuid.UUID           `json:"created_by_org_id"`
	IsShared        bool                `json:"is_shared"`
	IsActive        bool                `json:"is_active"`
	CreatedAt       time.Time           `json:"created_at"`
}

type ProjectResponse struct {
	Project
	ContractName     string `json:"contract_name"`
	CreatedByOrgName string `json:"created_by_org_name"`
	AdoptionCount    int    `json:"adoption_count"`
	IsAdopted        bool   `json:"is_adopted"`
}

type ProjectAdoption struct {
	ID             uuid.UUID `json:"id"`
	ProjectID      uuid.UUID `json:"project_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	AdoptedAt      time.Time `json:"adopted_at"`
}

type ProjectManager struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	UserName  string    `json:"user_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateProjectRequest struct {
	Name            string              `json:"name"`
	Type            models.ProjectType  `json:"type"`
	ContractID      string              `json:"contract_id"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        bool                `json:"is_shared"`
}

type UpdateProjectRequest struct {
	Name            string                 `json:"name"`
	Type            models.ProjectType     `json:"type"`
	ContractID      string                 `json:"contract_id"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        bool                   `json:"is_shared"`
}
