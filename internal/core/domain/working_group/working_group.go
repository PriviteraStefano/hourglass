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
	ID               uuid.UUID `json:"id"`
	OrgID            uuid.UUID `json:"org_id"`
	SubprojectID     uuid.UUID `json:"subproject_id"` // anchors to activities (D-5); field name is legacy, deferred to phase 10
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	UnitIDs          []string  `json:"unit_ids"`
	EnforceUnitTuple bool      `json:"enforce_unit_tuple"` // column dropped in 011; kept as legacy field for service compile
	ManagerID        uuid.UUID `json:"manager_id"`
	DelegateIDs      []string  `json:"delegate_ids"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateWorkingGroupRequest struct {
	OrgID            uuid.UUID `json:"org_id"`
	SubprojectID     uuid.UUID `json:"subproject_id"` // anchors to activities (D-5); field name is legacy, deferred to phase 10
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	UnitIDs          []string  `json:"unit_ids"`
	EnforceUnitTuple bool      `json:"enforce_unit_tuple"` // column dropped in 011; kept as legacy field for service compile
	ManagerID        uuid.UUID `json:"manager_id"`
	DelegateIDs      []string  `json:"delegate_ids"`
}

type UpdateWorkingGroupRequest struct {
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	UnitIDs          []string  `json:"unit_ids"`
	EnforceUnitTuple *bool     `json:"enforce_unit_tuple"` // column dropped in 011; kept as legacy field for service compile
	ManagerID        uuid.UUID `json:"manager_id"`
	DelegateIDs      []string  `json:"delegate_ids"`
}

type WorkingGroupMember struct {
	ID                  uuid.UUID  `json:"id"`
	WGID                uuid.UUID  `json:"wg_id"`
	UserID              uuid.UUID  `json:"user_id"`
	UnitID              uuid.UUID  `json:"unit_id"`
	Role                string     `json:"role"`
	IsDefaultSubproject bool       `json:"is_default_subproject"`
	StartDate           time.Time  `json:"start_date"`
	EndDate             *time.Time `json:"end_date"`
	CreatedAt           time.Time  `json:"created_at"`
}

type AddMemberRequest struct {
	WGID                uuid.UUID `json:"wg_id"`
	UserID              uuid.UUID `json:"user_id"`
	UnitID              uuid.UUID `json:"unit_id"`
	Role                string    `json:"role"`
	IsDefaultSubproject bool      `json:"is_default_subproject"`
}
