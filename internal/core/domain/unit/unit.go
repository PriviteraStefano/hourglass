package unit

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUnitNotFound            = errors.New("unit not found")
	ErrInvalidParentUnit       = errors.New("invalid parent unit")
	ErrCannotDeleteWithMembers = errors.New("cannot delete unit with members")
)

type Unit struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	Name           string
	Description    string
	ParentUnitID   *uuid.UUID
	HierarchyLevel int
	Code           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateUnitRequest struct {
	OrgID        uuid.UUID
	Name         string
	Description  string
	ParentUnitID *uuid.UUID
	Code         string
}

type UpdateUnitRequest struct {
	Name        string
	Description string
	Code        string
}

func (u *Unit) IsDescendantOf(ancestorID uuid.UUID) bool {
	if u.ParentUnitID == nil {
		return false
	}
	if *u.ParentUnitID == ancestorID {
		return true
	}
	return false
}

type UnitTreeNode struct {
	Unit     Unit
	Children []UnitTreeNode
}

func BuildTree(units []Unit, parentID *uuid.UUID) []UnitTreeNode {
	var tree []UnitTreeNode
	for _, u := range units {
		var unitParentID *uuid.UUID
		if u.ParentUnitID != nil && *u.ParentUnitID != uuid.Nil {
			unitParentID = u.ParentUnitID
		}
		matches := (parentID == nil && unitParentID == nil) ||
			(parentID != nil && unitParentID != nil && *parentID == *unitParentID)
		if matches {
			node := UnitTreeNode{
				Unit:     u,
				Children: BuildTree(units, &u.ID),
			}
			tree = append(tree, node)
		}
	}
	return tree
}
