package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIVersion_DefaultsToV1(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := GetAPIVersion(r)
		if version != "v1" {
			t.Errorf("expected v1, got %s", version)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := APIVersion(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAPIVersion_ExtractsFromHeader(t *testing.T) {
	tests := []struct {
		name     string
		accept   string
		expected string
	}{
		{
			name:     "no header",
			accept:   "",
			expected: "v1",
		},
		{
			name:     "version=1",
			accept:   "application/vnd.hourglass+json; version=1",
			expected: "v1",
		},
		{
			name:     "version=2",
			accept:   "application/vnd.hourglass+json; version=2",
			expected: "2",
		},
		{
			name:     "vendor media type",
			accept:   "application/vnd.hourglass+json",
			expected: "v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				version := GetAPIVersion(r)
				if version != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, version)
				}
				w.WriteHeader(http.StatusOK)
			})

			middleware := APIVersion(handler)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			rec := httptest.NewRecorder()

			middleware.ServeHTTP(rec, req)
		})
	}
}
