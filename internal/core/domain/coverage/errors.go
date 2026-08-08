package coverage

import "errors"

// Sentinel errors for the coverage plane (house style — ticket domain
// analog, ADR-BE-001). Sentinels carry plain error messages; HTTP handlers
// map them to status codes via errors.Is, and JSONNames serves stable
// serialization/UI keys.
var (
	// ErrEntryNotCoverable — the referenced entry is not coverable: missing,
	// cross-org, not approved, deleted, or a non-'time' entry type (D-K).
	ErrEntryNotCoverable = errors.New("entry not coverable")
	// ErrAllocationSumMismatch — the allocation set's Σ hours does not equal
	// the entry's hours (COV-01 invariant violation).
	ErrAllocationSumMismatch = errors.New("allocation sum does not match entry hours")
	// ErrPeriodAlreadyClosed — a close was attempted for a period that
	// overlaps an already-closed period (A6, 409).
	ErrPeriodAlreadyClosed = errors.New("period already closed")
	// ErrForbidden — the actor is not allowed to perform the operation
	// (D-08 manager gate, structural self-barrier).
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidRequest — malformed or semantically invalid request input.
	ErrInvalidRequest = errors.New("invalid request")
	// ErrNotFound — the requested resource does not exist or is out of scope.
	ErrNotFound = errors.New("not found")
)

// JSONNames maps sentinel errors to stable JSON-safe names (house style:
// sentinels carry plain error messages; the map serves serialization/UI keys
// — ticket domain analog).
var JSONNames = map[error]string{
	ErrEntryNotCoverable:     "entry_not_coverable",
	ErrAllocationSumMismatch: "allocation_sum_mismatch",
	ErrPeriodAlreadyClosed:   "period_already_closed",
	ErrForbidden:             "forbidden",
	ErrInvalidRequest:        "invalid_request",
	ErrNotFound:              "not_found",
}
