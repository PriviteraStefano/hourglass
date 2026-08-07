package ports

import (
	"context"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
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

	// KindExists reports whether the kind label is in the org's activity_kinds
	// catalog (ADR-P-007 D-2 — kind is a catalog label, not an enum). The
	// service-layer Create validation depends on it so unknown kinds surface as
	// clean sentinels instead of FK violations.
	KindExists(ctx context.Context, orgID uuid.UUID, kind string) (bool, error)
	// ListKinds returns the org's activity_kinds catalog (ADR-P-007 D-2),
	// ordered by name. Backs the GET /api/activity-kinds endpoint.
	ListKinds(ctx context.Context, orgID uuid.UUID) ([]activitydomain.ActivityKind, error)

	ListManagers(ctx context.Context, activityID uuid.UUID) ([]activitydomain.ActivityManager, error)
	AddManager(ctx context.Context, activityID, userID uuid.UUID) (*activitydomain.ActivityManager, error)
	RemoveManager(ctx context.Context, activityID, userID uuid.UUID) error

	// HasChildren reports whether the activity has at least one child.
	HasChildren(ctx context.Context, activityID uuid.UUID) (bool, error)
	HasActiveTimeEntries(ctx context.Context, activityID uuid.UUID) (bool, bool, error)
	HasActiveExpenses(ctx context.Context, activityID uuid.UUID) (bool, error)

	// IsLinkedTicketDismissed reports whether the activity — or any of its
	// ancestors — is a customer_ticket-origin activity whose linked ticket
	// is in the dismissed state (WR-06). Backs the entry Submit gate: a
	// dismissed ticket is terminal, so drafts on its activities must never
	// be submitted afterwards (hours logged on a dismissed ticket after the
	// fact). Consistent with the ticket repo's subtree "linked" definition.
	IsLinkedTicketDismissed(ctx context.Context, activityID uuid.UUID) (bool, error)

	// ApproveProposal flips is_active=true and writes the
	// proposal_approved audit row IN THE SAME TRANSACTION (Pitfall 2,
	// ADR-BE-016, T-11-08): the state write is not durable without its
	// event — a failure rolls back both, never a partial commit. Mirrors
	// TicketRepository.UpdateState.
	ApproveProposal(ctx context.Context, orgID, activityID uuid.UUID, auditLog *audit.AuditLog) (*activitydomain.ActivityResponse, error)
}
