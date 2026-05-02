package project

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrAlreadyAdopted  = errors.New("already adopted")
	ErrUserNotFound    = errors.New("user not found")
)

type Project struct {
	ID              uuid.UUID
	Name            string
	Type            models.ProjectType
	ContractID      uuid.UUID
	GovernanceModel models.GovernanceModel
	CreatedByOrgID  uuid.UUID
	IsShared        bool
	IsActive        bool
	CreatedAt       time.Time
}

type ProjectResponse struct {
	Project
	ContractName     string
	CreatedByOrgName string
	AdoptionCount    int
	IsAdopted        bool
}

type ProjectAdoption struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	OrganizationID uuid.UUID
	AdoptedAt      time.Time
}

type ProjectManager struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	UserID    uuid.UUID
	UserName  string
	Email     string
	CreatedAt time.Time
}

type CreateProjectRequest struct {
	Name            string
	Type            models.ProjectType
	ContractID      string
	GovernanceModel models.GovernanceModel
	IsShared        bool
}
