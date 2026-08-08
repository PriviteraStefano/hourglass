package direction

import "errors"

// Sentinel errors for the direction plane (house style — coverage/ticket
// domain analogs, ADR-BE-001). Sentinels carry plain error messages; HTTP
// handlers map them to status codes via errors.Is, and JSONNames serves
// stable serialization/UI keys.
var (
	// ErrDirectionNotFound — the direction row is missing or out of org
	// scope (same-org semantics; Get fast-fail for Activate/Cancel/Claim
	// and the supersede fast-fail).
	ErrDirectionNotFound = errors.New("direction not found")
	// ErrInvalidRequest — malformed or semantically invalid request input
	// (cross-org refs, WG row with planned_date, WG-scope violation).
	ErrInvalidRequest = errors.New("invalid request")
	// ErrForbidden — the actor is not allowed to perform the operation
	// (mode gate, manager reach, activation permission).
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidTransition — the pinned matrix forbids the status change
	// (D-13-07); supersede is create-only, never a transition.
	ErrInvalidTransition = errors.New("invalid direction status transition")
	// ErrClaimOverBudget — the claim would over-subscribe the WG row's
	// est_hours budget (D-13-13, 409 — the only 409 channel for claim
	// over-subscription, T-13-09).
	ErrClaimOverBudget = errors.New("claim exceeds the working group row budget")
	// ErrNotWgMember — the claimant is not a member of the WG owning the
	// row (D-13-12, 403).
	ErrNotWgMember = errors.New("user is not a member of the working group")
	// ErrWgRowNotActive — the WG row is not in a claimable state
	// (draft|active); superseded/cancelled rows are closed (D-13-16).
	ErrWgRowNotActive = errors.New("working group direction row is not active")
	// ErrCancelReasonRequired — cancellation (and unclaim) require a
	// reason (D-13-10/16, mirroring the reject-with-reason pattern).
	ErrCancelReasonRequired = errors.New("cancellation requires a reason")
	// ErrInvalidHours — est_hours is ≤ 0, has sub-cent precision, or a
	// scheduled row lacks est_hours (D-13-02/03, A2).
	ErrInvalidHours = errors.New("invalid hours")
	// ErrInvalidTarget — the XOR target is violated (both/neither of
	// directed_to/wg_id set, D-13-05) or a superseding row would change
	// the row shape (WG-shaped superseding row, ADR-BE-018 §5).
	ErrInvalidTarget = errors.New("invalid direction target")
)

// JSONNames maps sentinel errors to stable JSON-safe names (house style:
// sentinels carry plain error messages; the map serves serialization/UI keys
// — coverage domain analog).
var JSONNames = map[error]string{
	ErrDirectionNotFound:    "direction_not_found",
	ErrInvalidRequest:       "invalid_request",
	ErrForbidden:            "forbidden",
	ErrInvalidTransition:    "invalid_transition",
	ErrClaimOverBudget:      "claim_over_budget",
	ErrNotWgMember:          "not_wg_member",
	ErrWgRowNotActive:       "wg_row_not_active",
	ErrCancelReasonRequired: "cancel_reason_required",
	ErrInvalidHours:         "invalid_hours",
	ErrInvalidTarget:        "invalid_target",
}
