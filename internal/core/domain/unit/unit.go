package unit

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUnitNotFound            = errors.New("unit not found")
	ErrInvalidParentUnit       = errors.New("invalid parent unit")
	ErrCircularParent          = errors.New("cannot make unit a descendant of itself")
	ErrCannotDeleteWithMembers = errors.New("cannot delete unit with members")
	ErrMemberNotFound          = errors.New("unit member not found")
	ErrCannotDeleteRootUnit    = errors.New("cannot delete root unit")
	ErrCannotDeleteWithChildren = errors.New("cannot delete unit with child units")
)

type Unit struct {
	ID             string    `json:"id"`
	OrgID          uuid.UUID `json:"org_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	ParentUnitID   string    `json:"parent_unit_id,omitempty"`
	HierarchyLevel int       `json:"hierarchy_level"`
	Code           string    `json:"code"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateUnitRequest struct {
	OrgID        uuid.UUID
	Name         string
	Description  string
	ParentUnitID string
	Code         string
}

type UpdateUnitRequest struct {
	Name         string
	Description  string
	Code         string
	ParentUnitID *string
}

func (u *Unit) IsDescendantOf(ancestorID string) bool {
	if u.ParentUnitID == "" {
		return false
	}
	if u.ParentUnitID == ancestorID {
		return true
	}
	return false
}

type UnitTreeNode struct {
	Unit              Unit           `json:"unit"`
	MemberCount       int            `json:"member_count"`
	TotalMemberCount  int            `json:"total_member_count"`
	Children          []UnitTreeNode `json:"children,omitempty"`
}

type UnitMember struct {
	ID         string    `json:"id"`
	OrgID      uuid.UUID `json:"org_id"`
	UserID     uuid.UUID `json:"user_id"`
	UserName   string    `json:"user_name"`
	UserEmail  string    `json:"user_email"`
	UnitID     string    `json:"unit_id"`
	IsPrimary  bool      `json:"is_primary"`
	Role       string    `json:"role"`
	StartDate  time.Time `json:"start_date"`
	EndDate    *time.Time `json:"end_date,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type AddUnitMemberRequest struct {
	UserID    uuid.UUID
	Role      string
	IsPrimary bool
}

func BuildTree(units []Unit, parentID string) []UnitTreeNode {
	var tree []UnitTreeNode
	for _, u := range units {
		var unitParentID string
		if u.ParentUnitID != "" {
			unitParentID = u.ParentUnitID
		}
		matches := (parentID == "" && unitParentID == "") ||
			(parentID != "" && unitParentID != "" && parentID == unitParentID)
		if matches {
			node := UnitTreeNode{
				Unit:     u,
				Children: BuildTree(units, u.ID),
			}
			tree = append(tree, node)
		}
	}
	return tree
}
