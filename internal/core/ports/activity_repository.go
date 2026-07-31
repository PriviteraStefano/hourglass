package ports

import (
	"context"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
)

// ActivityRepository is the single repository port replacing the collapsed
// project + subproject repositories (ADR-BE-014 R-6). It operates against the
// recursive activities table (ADR-P-007 D-1/D-2/D-3/D-7).
type ActivityRepository interface {
	List(ctx context.Context, orgID uuid.UUID, filter *activitydomain.ActivityFilter) ([]activitydomain.ActivityResponse, error)
	Get(ctx context.Context, orgID, activityID uuid.UUID) (*activitydomain.ActivityResponse, error)
	Create(ctx context.Context, orgID uuid.UUID, req *activitydomain.CreateActivityRequest) (*activitydomain.ActivityResponse, error)
	Update(ctx context.Context, orgID, activityID uuid.UUID, req *activitydomain.UpdateActivityRequest) (*activitydomain.ActivityResponse, error)
	Delete(ctx context.Context, orgID, activityID uuid.UUID) error
	Adopt(ctx context.Context, orgID, activityID uuid.UUID) (*activitydomain.ActivityAdoption, error)

	ListChildren(ctx context.Context, parentID uuid.UUID) ([]activitydomain.ActivityResponse, error)
	ListByContract(ctx context.Context, contractID uuid.UUID) ([]activitydomain.ActivityResponse, error)
	GetAncestry(ctx context.Context, id uuid.UUID) ([]activitydomain.Activity, error)

	// ResolveCommercialContext walks parent_id upward to the nearest ancestor
	// with a contract and returns (contract_id, customer_id), or nil for a
	// purely internal tree (D-3 — derived, never stored).
	ResolveCommercialContext(ctx context.Context, activityID uuid.UUID) (*activitydomain.CommercialContext, error)
	// ResolveBillability walks ancestry: nearest non-NULL billable wins; if the
	// walk hits a contract-linked ancestor, defer to the contract default (D-7).
	ResolveBillability(ctx context.Context, activityID uuid.UUID) (*bool, error)

	ListManagers(ctx context.Context, activityID uuid.UUID) ([]activitydomain.ActivityManager, error)
	AddManager(ctx context.Context, activityID, userID uuid.UUID) (*activitydomain.ActivityManager, error)
	RemoveManager(ctx context.Context, activityID, userID uuid.UUID) error

	HasChildren(ctx context.Context, activityID uuid.UUID) (bool, error)
	HasActiveTimeEntries(ctx context.Context, activityID uuid.UUID) (bool, bool, error)
	HasActiveExpenses(ctx context.Context, activityID uuid.UUID) (bool, error)
}
