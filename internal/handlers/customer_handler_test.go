package handlers

import (
	"testing"
)

func TestParseIntParam_ValidInput(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"10", 10},
		{"50", 50},
		{"0", 0},
		{"100", 100},
	}

	for _, tt := range tests {
		result, err := parseIntParam(tt.input)
		if err != nil {
			t.Errorf("unexpected error for input %s: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("expected %d, got %d for input %s", tt.expected, result, tt.input)
		}
	}
}

func TestParseIntParam_InvalidInput(t *testing.T) {
	_, err := parseIntParam("abc")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestNullString_Empty(t *testing.T) {
	result := nullString("")
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", result)
	}
}

func TestNullString_NonEmpty(t *testing.T) {
	result := nullString("test")
	if result != "test" {
		t.Errorf("expected 'test', got %v", result)
	}
}
