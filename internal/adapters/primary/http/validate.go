package http

import (
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/stefanoprivitera/hourglass/pkg/api"
)

// Per-field maximum string lengths enforced at the HTTP adapter boundary
// (audit S3). Domain value objects validate format, not max length; caps live
// here per hexagonal layering — no changes to domain types.
const (
	MaxEmailLength       = 320
	MaxNameLength        = 200
	MaxDescriptionLength = 4000
	MaxAddressLength     = 500
	MaxVATLength         = 50
	MaxPhoneLength       = 50
	MaxPasswordLength    = 128
	MaxShortStringLength = 500
)

type lengthCappedField struct {
	name  string
	value string
	max   int
}

// lengthField builds a length-cap check for a single decoded string field.
func lengthField(name, value string, max int) lengthCappedField {
	return lengthCappedField{name: name, value: value, max: max}
}

// validateStringLengths rejects any field whose rune length exceeds its cap.
// On the first violation it writes a 400 response with a field-level message
// ("<field> exceeds maximum length of N") and returns false. Callers must
// return immediately when it reports false.
func validateStringLengths(w http.ResponseWriter, fields ...lengthCappedField) bool {
	for _, f := range fields {
		if utf8.RuneCountInString(f.value) > f.max {
			api.RespondWithError(w, http.StatusBadRequest,
				fmt.Sprintf("%s exceeds maximum length of %d", f.name, f.max))
			return false
		}
	}
	return true
}
