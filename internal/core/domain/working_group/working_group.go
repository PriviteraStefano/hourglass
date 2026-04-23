package working_group

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrWorkingGroupNotFound    = errors.New("working group not found")
	ErrCannotDeleteWithMembers = errors.New("cannot delete working group with members")
)

type WorkingGroup struct {
	ID               uuid.UUID
	OrgID            uuid.UUID
	SubprojectID     uuid.UUID
	Name             string
	Description      string
	UnitIDs          []string
	EnforceUnitTuple bool
	ManagerID        uuid.UUID
	DelegateIDs      []string
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateWorkingGroupRequest struct {
	OrgID            uuid.UUID
	SubprojectID     uuid.UUID
	Name             string
	Description      string
	UnitIDs          []string
	EnforceUnitTuple bool
	ManagerID        uuid.UUID
	DelegateIDs      []string
}

type UpdateWorkingGroupRequest struct {
	Name             string
	Description      string
	UnitIDs          []string
	EnforceUnitTuple *bool
	ManagerID        uuid.UUID
	DelegateIDs      []string
}

type WorkingGroupMember struct {
	ID                  uuid.UUID
	WGID                uuid.UUID
	UserID              uuid.UUID
	UnitID              uuid.UUID
	Role                string
	IsDefaultSubproject bool
	StartDate           time.Time
	EndDate             *time.Time
	CreatedAt           time.Time
}

type AddMemberRequest struct {
	WGID                uuid.UUID
	UserID              uuid.UUID
	UnitID              uuid.UUID
	Role                string
	IsDefaultSubproject bool
}
