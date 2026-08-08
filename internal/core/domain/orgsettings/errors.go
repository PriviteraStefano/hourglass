package orgsettings

import "errors"

// Sentinel errors for the org_settings plane (D-13-18/23, ADR-BE-018 §7).
// Sentinels carry plain error messages; HTTP handlers map them to status
// codes via errors.Is, and JSONNames serves stable serialization/UI keys
// (coverage/ticket house shape).
var (
	// ErrUnknownKey — the key has no code-enforced validator (D-13-18,
	// 400).
	ErrUnknownKey = errors.New("unknown settings key")
	// ErrInvalidValue — the value fails the known key's validator
	// (D-13-18, 400).
	ErrInvalidValue = errors.New("invalid settings value")
	// ErrForbidden — the actor is not manager+ (D-13-23 PUT gate, 403).
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound — the org or the key is absent.
	ErrNotFound = errors.New("not found")
)

// JSONNames maps sentinel errors to stable JSON-safe names (house style:
// sentinels carry plain error messages; the map serves serialization/UI
// keys).
var JSONNames = map[error]string{
	ErrUnknownKey:   "unknown_key",
	ErrInvalidValue: "invalid_value",
	ErrForbidden:    "forbidden",
	ErrNotFound:     "not_found",
}
