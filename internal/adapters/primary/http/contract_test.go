package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	contractsvc "github.com/stefanoprivitera/hourglass/internal/core/services/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/services/testdata"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

func TestContractHandler_Create_InvalidBody(t *testing.T) {
	h := NewContractHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/contracts", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestContractHandler_Get_InvalidID(t *testing.T) {
	h := NewContractHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/contracts/invalid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestContractHandler_Update_InvalidBody(t *testing.T) {
	h := NewContractHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/contracts/"+uuid.NewString(), strings.NewReader("{"))
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestContractHandler_Update_InvalidID(t *testing.T) {
	h := NewContractHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/contracts/invalid", strings.NewReader(`{"name":"test"}`))
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestContractHandler_Delete_InvalidID(t *testing.T) {
	h := NewContractHandler(nil)

	req := httptest.NewRequest(http.MethodDelete, "/contracts/invalid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.Delete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestContractHandler_RecalculateMileage_InvalidBody(t *testing.T) {
	h := NewContractHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/contracts/"+uuid.NewString()+"/recalculate-mileage", strings.NewReader("{"))
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.RecalculateMileage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestContractHandler_RecalculateMileage_InvalidID(t *testing.T) {
	h := NewContractHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/contracts/invalid/recalculate-mileage", strings.NewReader(`{"from_date":"2026-01-01"}`))
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.RecalculateMileage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestContractHandler_Adopt_InvalidID(t *testing.T) {
	h := NewContractHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/contracts/invalid/adopt", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	req = req.WithContext(ctx)

	h.Adopt(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestContractHandler_Update_InvalidSoldConfig(t *testing.T) {
	// WR-03: converting a support contract to project WITHOUT clearing
	// sold_period is a client-side sold-config error → 422, never a 500.
	repo := &testdata.MockContractRepo{}
	support := "support"
	month := "month"
	hours := 100.0
	contractID := uuid.New()
	repo.Contracts = map[uuid.UUID]*contractdomain.ContractResponse{
		contractID: {
			Contract: contractdomain.Contract{
				ID:              contractID,
				Name:            "Support contract",
				GovernanceModel: models.GovernanceCreatorControlled,
				CreatedByOrgID:  uuid.New(),
				IsActive:        true,
				ContractType:    &support,
				SoldHours:       &hours,
				SoldPeriod:      &month,
			},
		},
	}
	svc := contractsvc.NewService(repo)
	h := NewContractHandler(svc)

	req := httptest.NewRequest(http.MethodPut, "/contracts/"+contractID.String(), strings.NewReader(`{"contract_type":"project"}`))
	req.SetPathValue("id", contractID.String())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.Update(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected %d, got %d", http.StatusUnprocessableEntity, rec.Code)
	}
}

func TestContractHandler_Update_ValidConversion(t *testing.T) {
	// WR-03: support→project WITH sold_period:"" clear passes the merged
	// validation and reaches the repo (the conversion is no longer a
	// dead-end). The mock repo applies the update and returns the contract.
	repo := &testdata.MockContractRepo{}
	support := "support"
	month := "month"
	hours := 100.0
	contractID := uuid.New()
	repo.Contracts = map[uuid.UUID]*contractdomain.ContractResponse{
		contractID: {
			Contract: contractdomain.Contract{
				ID:              contractID,
				Name:            "Support contract",
				GovernanceModel: models.GovernanceCreatorControlled,
				CreatedByOrgID:  uuid.New(),
				IsActive:        true,
				ContractType:    &support,
				SoldHours:       &hours,
				SoldPeriod:      &month,
			},
		},
	}
	svc := contractsvc.NewService(repo)
	h := NewContractHandler(svc)

	req := httptest.NewRequest(http.MethodPut, "/contracts/"+contractID.String(), strings.NewReader(`{"contract_type":"project","sold_period":""}`))
	req.SetPathValue("id", contractID.String())
	rec := httptest.NewRecorder()

	ctx := middleware.SetUserID(req.Context(), uuid.New())
	ctx = middleware.SetOrganizationID(ctx, uuid.New())
	ctx = middleware.SetRole(ctx, "finance")
	req = req.WithContext(ctx)

	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
