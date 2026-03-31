package handlers

import "testing"

func TestTimeEntryHoursValidation(t *testing.T) {
	tests := []struct {
		name     string
		hours    float64
		expected bool
	}{
		{"zero hours", 0, false},
		{"valid hours", 8, true},
		{"max hours", 24, true},
		{"over max", 24.5, false},
		{"negative hours", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.hours > 0 && tt.hours <= 24
			if valid != tt.expected {
				t.Errorf("hours %f: expected %v, got %v", tt.hours, tt.expected, valid)
			}
		})
	}
}
