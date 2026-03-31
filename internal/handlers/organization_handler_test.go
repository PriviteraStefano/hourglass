package handlers

import (
	"testing"
)

func TestIsValidCurrency_ValidCodes(t *testing.T) {
	validCodes := []string{"EUR", "USD", "GBP", "JPY", "CHF", "AUD", "CAD", "CNY"}
	for _, code := range validCodes {
		if !isValidCurrency(code) {
			t.Errorf("expected %s to be valid", code)
		}
	}
}

func TestIsValidCurrency_InvalidCodes(t *testing.T) {
	invalidCodes := []string{"XXX", "ABC", "123", "eur", "usd", ""}
	for _, code := range invalidCodes {
		if isValidCurrency(code) {
			t.Errorf("expected %s to be invalid", code)
		}
	}
}
