package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	customerdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
	customersvc "github.com/stefanoprivitera/hourglass/internal/core/services/customer"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type CustomerHandler struct {
	service *customersvc.Service
}

func NewCustomerHandler(service *customersvc.Service) *CustomerHandler {
	return &CustomerHandler{service: service}
}

type CustomerCreateRequest struct {
	CompanyName string `json:"company_name"`
	ContactName string `json:"contact_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	VATNumber   string `json:"vat_number,omitempty"`
	Address     string `json:"address,omitempty"`
}

type CustomerUpdateRequest struct {
	CompanyName string `json:"company_name,omitempty"`
	ContactName string `json:"contact_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	VATNumber   string `json:"vat_number,omitempty"`
	Address     string `json:"address,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	customers, err := h.service.List(ctx, orgID, limit, offset)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch customers")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, customers)
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.GetOrganizationID(ctx)
	role := middleware.GetRole(ctx)

	var req CustomerCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.service.Create(ctx, orgID, role, &customerdomain.CreateCustomerRequest{
		CompanyName: req.CompanyName,
		ContactName: req.ContactName,
		Email:       req.Email,
		Phone:       req.Phone,
		VATNumber:   req.VATNumber,
		Address:     req.Address,
	})
	if err != nil {
		switch err {
		case customerdomain.ErrForbidden:
			api.RespondWithError(w, http.StatusForbidden, "only finance users can create customers")
		case customerdomain.ErrInvalidCustomer:
			api.RespondWithError(w, http.StatusBadRequest, "company_name is required")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create customer")
		}
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, created)
}

func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	customerID := r.PathValue("id")
	if customerID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "customer id is required")
		return
	}

	id, err := uuid.Parse(customerID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid customer id")
		return
	}

	c, linkedContracts, err := h.service.Get(ctx, id)
	if err != nil {
		if err == customerdomain.ErrCustomerNotFound {
			api.RespondWithError(w, http.StatusNotFound, "customer not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "database error")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"customer":         c,
		"linked_contracts": linkedContracts,
	})
}

func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	role := middleware.GetRole(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	customerID := r.PathValue("id")
	if customerID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "customer id is required")
		return
	}

	id, err := uuid.Parse(customerID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid customer id")
		return
	}

	var req CustomerUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.service.Update(ctx, id, orgID, role, &customerdomain.UpdateCustomerRequest{
		CompanyName: req.CompanyName,
		ContactName: req.ContactName,
		Email:       req.Email,
		Phone:       req.Phone,
		VATNumber:   req.VATNumber,
		Address:     req.Address,
		IsActive:    req.IsActive,
	})
	if err != nil {
		switch err {
		case customerdomain.ErrCustomerNotFound:
			api.RespondWithError(w, http.StatusNotFound, "customer not found")
		case customerdomain.ErrForbidden:
			api.RespondWithError(w, http.StatusForbidden, "only the creating organization finance users can update this customer")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to update customer")
		}
		return
	}

	api.RespondWithJSON(w, http.StatusOK, updated)
}

func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	role := middleware.GetRole(ctx)
	orgID := middleware.GetOrganizationID(ctx)
	customerID := r.PathValue("id")
	if customerID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "customer id is required")
		return
	}

	id, err := uuid.Parse(customerID)
	if err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid customer id")
		return
	}

	err = h.service.Delete(ctx, id, orgID, role)
	if err != nil {
		switch err {
		case customerdomain.ErrCustomerNotFound:
			api.RespondWithError(w, http.StatusNotFound, "customer not found")
		case customerdomain.ErrForbidden:
			api.RespondWithError(w, http.StatusForbidden, "only the creating organization finance users can delete this customer")
		case customerdomain.ErrCustomerLinkedContract:
			api.RespondWithError(w, http.StatusConflict, "cannot delete customer linked to contracts")
		default:
			api.RespondWithError(w, http.StatusInternalServerError, "failed to deactivate customer")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
