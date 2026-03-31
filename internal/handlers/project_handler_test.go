package handlers

import "testing"

func TestProjectManagerResponse_Fields(t *testing.T) {
	m := ProjectManagerResponse{
		ID:        "test-id",
		ProjectID: "project-id",
		UserID:    "user-id",
		UserName:  "Test User",
		Email:     "test@example.com",
		CreatedAt: "2024-01-01T00:00:00Z",
	}

	if m.ID != "test-id" {
		t.Errorf("expected ID test-id, got %s", m.ID)
	}
	if m.UserName != "Test User" {
		t.Errorf("expected UserName Test User, got %s", m.UserName)
	}
}
