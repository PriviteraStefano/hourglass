package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	contractsvc "github.com/stefanoprivitera/hourglass/internal/core/services/contract"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type ContractHandler struct {
	service *contractsvc.Service
}

func NewContractHandler(service *contractsvc.Service) *ContractHandler {
	return &ContractHandler{service: service}
}

type CreateContractRequest struct {
	Name            string                 `json:"name"`
	KmRate          float64                `json:"km_rate"`
	Currency        string                 `json:"currency"`
	GovernanceModel models.GovernanceModel `json:"governance_model"`
	IsShared        bool                   `json:"is_shared"`
	CustomerID      *string                `json:"customer_id,omitempty"`
	ContractType    *string                `json:"contract_type,omitempty"`
	SoldHours       *float64               `json:"sold_hours,omitempty"`
	SoldPeriod      *string                `json:"sold_period,omitempty"`
}

func (h *ContractHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "owned"
	}
	isActiveStr := r.URL.Query().Get("is_active")
	var isActive *bool
	if isActiveStr == "true" {
		v := true
		isActive = &v
	} else if isActiveStr == "false" {
		v := false
		isActive = &v
	}
	contracts, err := h.service.List(r.Context(), orgID, scope, isActive)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch contracts")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, contracts)
}

func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	var req CreateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !validateStringLengths(w,
		lengthField("name", req.Name, MaxNameLength),
		lengthField("currency", req.Currency, MaxShortStringLength),
	) {
		return
	}

	var parsedCustomerID *uuid.UUID
	if req.CustomerID != nil && *req.CustomerID != "" {
		cid, err := uuid.Parse(*req.CustomerID)
		if err != nil {
			api.RespondWithError(w, http.StatusBadRequest, "invalid customer_id")
			return
		}
		parsedCustomerID = &cid
	}

	contract, err := h.service.Create(r.Context(), orgID, &contractdomain.CreateContractRequest{
		Name:            req.Name,
		KmRate:          req.KmRate,
		Currency:        req.Currency,
		GovernanceModel: req.GovernanceModel,
		IsShared:        req.IsShared,
		CustomerID:      parsedCustomerID,
		ContractType:    req.ContractType,
		SoldHours:       req.SoldHours,
		SoldPeriod:      req.SoldPeriod,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract payload")
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, contract)
}

func (h *ContractHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	contractID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
		return
	}
	contract, err := h.service.Get(r.Context(), orgID, contractID)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "contract not found")
		return
	}
	api.RespondWithJSON(w, http.StatusOK, contract)
}

func (h *ContractHandler) Adopt(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	contractID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
		return
	}
	adoption, err := h.service.Adopt(r.Context(), orgID, contractID)
	if err != nil {
		if err == contractdomain.ErrAlreadyAdopted {
			api.RespondWithError(w, http.StatusConflict, "contract already adopted")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to adopt contract")
		return
	}
	api.RespondWithJSON(w, http.StatusCreated, adoption)
}

type UpdateContractRequest struct {
	Name            string                 `json:"name,omitempty"`
	KmRate          *float64               `json:"km_rate,omitempty"`
	Currency        string                 `json:"currency,omitempty"`
	GovernanceModel models.GovernanceModel `json:"governance_model,omitempty"`
	IsShared        *bool                  `json:"is_shared,omitempty"`
	IsActive        *bool                  `json:"is_active,omitempty"`
	CustomerID      *string                `json:"customer_id,omitempty"`
	ContractType    *string                `json:"contract_type,omitempty"`
	SoldHours       *float64               `json:"sold_hours,omitempty"`
	SoldPeriod      *string                `json:"sold_period,omitempty"`
}

func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	contractID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
		return
	}
	var req UpdateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !validateStringLengths(w,
		lengthField("name", req.Name, MaxNameLength),
		lengthField("currency", req.Currency, MaxShortStringLength),
	) {
		return
	}

	updated, affectedMileageCount, err := h.service.Update(r.Context(), middleware.GetRole(r.Context()), orgID, contractID, &contractdomain.UpdateContractRequest{
		Name:            req.Name,
		KmRate:          req.KmRate,
		Currency:        req.Currency,
		GovernanceModel: req.GovernanceModel,
		IsShared:        req.IsShared,
		IsActive:        req.IsActive,
		CustomerID:      req.CustomerID,
		ContractType:    req.ContractType,
		SoldHours:       req.SoldHours,
		SoldPeriod:      req.SoldPeriod,
	})
	if err != nil {
		switch err {
		case contractdomain.ErrForbidden:
			api.RespondWithError(w, http.StatusForbidden, "only finance users can update contracts")
		case contractdomain.ErrContractNotFound:
			api.RespondWithError(w, http.StatusNotFound, "contract not found")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to update contract")
		}
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"contract":               updated,
		"affected_mileage_count": affectedMileageCount,
	})
}

type RecalculateMileageRequest struct {
	FromDate string `json:"from_date"`
}

func (h *ContractHandler) RecalculateMileage(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	contractID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
		return
	}
	var req RecalculateMileageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	count, err := h.service.RecalculateMileage(r.Context(), middleware.GetRole(r.Context()), orgID, contractID, req.FromDate, middleware.GetUserID(r.Context()))
	if err != nil {
		switch err {
		case contractdomain.ErrForbidden:
			api.RespondWithError(w, http.StatusForbidden, "only finance users can recalculate mileage")
		case contractdomain.ErrInvalidRequest:
			api.RespondWithError(w, http.StatusBadRequest, "from_date is required")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to recalculate mileage")
		}
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"recalculated_count": count})
}

func (h *ContractHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrganizationID(r.Context())
	contractID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid contract id")
		return
	}
	err = h.service.Delete(r.Context(), middleware.GetRole(r.Context()), orgID, contractID)
	if err != nil {
		switch err {
		case contractdomain.ErrForbidden:
			api.RespondWithError(w, http.StatusForbidden, "only finance users can delete contracts")
		case contractdomain.ErrContractNotFound:
			api.RespondWithError(w, http.StatusNotFound, "contract not found")
		case contractdomain.ErrHasTimeEntries:
			api.RespondWithError(w, http.StatusConflict, "contract has time entries and cannot be deleted")
		case contractdomain.ErrHasActiveProjects:
			api.RespondWithError(w, http.StatusConflict, "contract has projects and cannot be deleted")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to delete contract")
		}
		return
	}
	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "contract deleted"})
}
