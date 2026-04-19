package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/db"
)

func TestInvitationHandler_Create(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewInvitationHandler(sdb)

	reqBody := CreateInvitationRequest{
		OrganizationID: "test-org-id",
		Email:          "invited@example.com",
		ExpiresInDays:  7,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	t.Logf("Create response: %d, body: %s", rec.Code, rec.Body.String())

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data in response, got: %+v", response)
	}
	if data["code"] == "" {
		t.Error("expected code in response")
	}
	if data["token"] == "" {
		t.Error("expected token in response")
	}
}

func TestInvitationHandler_ValidateCode(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewInvitationHandler(sdb)

	createReq := CreateInvitationRequest{
		OrganizationID: "test-org-id",
		Email:          "validate-test@example.com",
		ExpiresInDays:  7,
	}
	createBody, _ := json.Marshal(createReq)
	createHttpReq := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(createBody))
	createHttpReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.Create(createRec, createHttpReq)

	var createResponse map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	data, ok := createResponse["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data in response, got: %+v", createResponse)
	}
	code := data["code"].(string)

	req := httptest.NewRequest(http.MethodGet, "/invitations/validate/code/"+code, nil)
	rec := httptest.NewRecorder()

	handler.ValidateCode(rec, req)

	t.Logf("ValidateCode response: %d, body: %s", rec.Code, rec.Body.String())

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestInvitationHandler_ValidateCode_NotFound(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewInvitationHandler(sdb)

	req := httptest.NewRequest(http.MethodGet, "/invitations/validate/code/NONEXISTENT", nil)
	rec := httptest.NewRecorder()

	handler.ValidateCode(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestInvitationHandler_Accept(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewInvitationHandler(sdb)

	createReq := CreateInvitationRequest{
		OrganizationID: "test-org-id",
		Email:          "accept-test@example.com",
		ExpiresInDays:  7,
	}
	createBody, _ := json.Marshal(createReq)
	createHttpReq := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(createBody))
	createHttpReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.Create(createRec, createHttpReq)

	var createResponse map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	data, _ := createResponse["data"].(map[string]interface{})
	token := data["token"].(string)

	acceptReq := map[string]interface{}{
		"token":    token,
		"email":    "newuser@example.com",
		"username": "newuser" + t.Name(),
		"password": "password123",
	}
	acceptBody, _ := json.Marshal(acceptReq)
	acceptHttpReq := httptest.NewRequest(http.MethodPost, "/invitations/accept", bytes.NewReader(acceptBody))
	acceptHttpReq.Header.Set("Content-Type", "application/json")
	acceptRec := httptest.NewRecorder()

	handler.Accept(acceptRec, acceptHttpReq)

	t.Logf("Accept response: %d, body: %s", acceptRec.Code, acceptRec.Body.String())

	if acceptRec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", acceptRec.Code, acceptRec.Body.String())
	}
}

func TestInvitationHandler_Accept_Expired(t *testing.T) {
	if os.Getenv("SURREALDB_URL") == "" {
		t.Skip("SURREALDB_URL not set, skipping integration test")
	}

	sdb, err := db.NewSurrealDB()
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB: %v", err)
	}
	defer sdb.Close()

	handler := NewInvitationHandler(sdb)

	token := "expired-token-" + time.Now().Format("20060102150405")

	acceptReq := map[string]interface{}{
		"token":    token,
		"email":    "user@example.com",
		"password": "password123",
	}
	acceptBody, _ := json.Marshal(acceptReq)
	acceptHttpReq := httptest.NewRequest(http.MethodPost, "/invitations/accept", bytes.NewReader(acceptBody))
	acceptHttpReq.Header.Set("Content-Type", "application/json")
	acceptRec := httptest.NewRecorder()

	handler.Accept(acceptRec, acceptHttpReq)

	t.Logf("Accept expired response: %d, body: %s", acceptRec.Code, acceptRec.Body.String())

	if acceptRec.Code != http.StatusNotFound && acceptRec.Code != http.StatusGone {
		t.Errorf("expected status 404 or 410, got %d: %s", acceptRec.Code, acceptRec.Body.String())
	}
}
