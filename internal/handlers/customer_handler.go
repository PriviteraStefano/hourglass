package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/internal/models"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type CustomerHandler struct {
	db *sql.DB
}

func NewCustomerHandler(db *sql.DB) *CustomerHandler {
	return &CustomerHandler{db: db}
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
	orgID := middleware.GetOrganizationID(r.Context())

	query := r.URL.Query()
	limit := 50
	offset := 0
	if l := query.Get("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := query.Get("offset"); o != "" {
		if parsed, err := parseIntParam(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	rows, err := h.db.Query(`
		SELECT id, organization_id, company_name, contact_name, email, phone, vat_number, address, is_active, created_at
		FROM customers
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch customers")
		return
	}
	defer rows.Close()

	customers := []models.Customer{}
	for rows.Next() {
		var c models.Customer
		var contactName, email, phone, vatNumber, address sql.NullString
		err := rows.Scan(&c.ID, &c.OrganizationID, &c.CompanyName, &contactName, &email, &phone, &vatNumber, &address, &c.IsActive, &c.CreatedAt)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to scan customer")
			return
		}
		if contactName.Valid {
			c.ContactName = contactName.String
		}
		if email.Valid {
			c.Email = email.String
		}
		if phone.Valid {
			c.Phone = phone.String
		}
		if vatNumber.Valid {
			c.VATNumber = vatNumber.String
		}
		if address.Valid {
			c.Address = address.String
		}
		customers = append(customers, c)
	}

	api.RespondWithJSON(w, http.StatusOK, customers)
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	userRole := middleware.GetRole(r.Context())
	if userRole != string(models.RoleFinance) {
		api.RespondWithError(w, http.StatusForbidden, "only finance users can create customers")
		return
	}

	orgID := middleware.GetOrganizationID(r.Context())

	var req CustomerCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CompanyName == "" {
		api.RespondWithError(w, http.StatusBadRequest, "company_name is required")
		return
	}

	id := uuid.New()
	_, err := h.db.Exec(`
		INSERT INTO customers (id, organization_id, company_name, contact_name, email, phone, vat_number, address, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, NOW())
	`, id, orgID, req.CompanyName, nullString(req.ContactName), nullString(req.Email), nullString(req.Phone), nullString(req.VATNumber), nullString(req.Address))
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create customer")
		return
	}

	var customer models.Customer
	err = h.db.QueryRow(`
		SELECT id, organization_id, company_name, contact_name, email, phone, vat_number, address, is_active, created_at
		FROM customers WHERE id = $1
	`, id).Scan(&customer.ID, &customer.OrganizationID, &customer.CompanyName, &customer.ContactName, &customer.Email, &customer.Phone, &customer.VATNumber, &customer.Address, &customer.IsActive, &customer.CreatedAt)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch created customer")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, customer)
}

func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	var customer models.Customer
	err = h.db.QueryRow(`
		SELECT id, organization_id, company_name, contact_name, email, phone, vat_number, address, is_active, created_at
		FROM customers WHERE id = $1
	`, id).Scan(&customer.ID, &customer.OrganizationID, &customer.CompanyName, &customer.ContactName, &customer.Email, &customer.Phone, &customer.VATNumber, &customer.Address, &customer.IsActive, &customer.CreatedAt)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "customer not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "database error")
		return
	}

	linkedContracts := []models.Contract{}
	rows, err := h.db.Query(`
		SELECT id, name, km_rate, currency, customer_id, governance_model, created_by_org_id, is_shared, is_active, created_at
		FROM contracts WHERE customer_id = $1
	`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c models.Contract
			var kmRate sql.NullFloat64
			var customerID sql.NullString
			err := rows.Scan(&c.ID, &c.Name, &kmRate, &c.Currency, &customerID, &c.GovernanceModel, &c.CreatedByOrgID, &c.IsShared, &c.IsActive, &c.CreatedAt)
			if err == nil {
				if kmRate.Valid {
					c.KmRate = kmRate.Float64
				}
				if customerID.Valid {
					c.CustomerID = parseUUID(customerID.String)
				}
				linkedContracts = append(linkedContracts, c)
			}
		}
	}

	response := map[string]interface{}{
		"customer":         customer,
		"linked_contracts": linkedContracts,
	}

	api.RespondWithJSON(w, http.StatusOK, response)
}

func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	userRole := middleware.GetRole(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

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

	var ownerOrgID uuid.UUID
	err = h.db.QueryRow("SELECT organization_id FROM customers WHERE id = $1", id).Scan(&ownerOrgID)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "customer not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "database error")
		return
	}

	if ownerOrgID != orgID {
		api.RespondWithError(w, http.StatusForbidden, "only the creating organization can edit this customer")
		return
	}

	if userRole != string(models.RoleFinance) {
		api.RespondWithError(w, http.StatusForbidden, "only finance users can update customers")
		return
	}

	var req CustomerUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var customer models.Customer
	err = h.db.QueryRow(`
		UPDATE customers
		SET company_name = COALESCE(NULLIF($1, ''), company_name),
		    contact_name = COALESCE($2, contact_name),
		    email = COALESCE($3, email),
		    phone = COALESCE($4, phone),
		    vat_number = COALESCE($5, vat_number),
		    address = COALESCE($6, address),
		    is_active = COALESCE($7, is_active)
		WHERE id = $8
		RETURNING id, organization_id, company_name, contact_name, email, phone, vat_number, address, is_active, created_at
	`, req.CompanyName, nullString(req.ContactName), nullString(req.Email), nullString(req.Phone), nullString(req.VATNumber), nullString(req.Address), req.IsActive, id).Scan(
		&customer.ID, &customer.OrganizationID, &customer.CompanyName, &customer.ContactName, &customer.Email, &customer.Phone, &customer.VATNumber, &customer.Address, &customer.IsActive, &customer.CreatedAt,
	)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update customer")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userRole := middleware.GetRole(r.Context())
	orgID := middleware.GetOrganizationID(r.Context())

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

	var ownerOrgID uuid.UUID
	err = h.db.QueryRow("SELECT organization_id FROM customers WHERE id = $1", id).Scan(&ownerOrgID)
	if err == sql.ErrNoRows {
		api.RespondWithError(w, http.StatusNotFound, "customer not found")
		return
	}
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "database error")
		return
	}

	if ownerOrgID != orgID {
		api.RespondWithError(w, http.StatusForbidden, "only the creating organization can delete this customer")
		return
	}

	if userRole != string(models.RoleFinance) {
		api.RespondWithError(w, http.StatusForbidden, "only finance users can delete customers")
		return
	}

	var linkedCount int
	err = h.db.QueryRow("SELECT COUNT(*) FROM contracts WHERE customer_id = $1", id).Scan(&linkedCount)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to check contracts")
		return
	}

	if linkedCount > 0 {
		api.RespondWithError(w, http.StatusConflict, "cannot delete customer linked to contracts")
		return
	}

	_, err = h.db.Exec("UPDATE customers SET is_active = false WHERE id = $1", id)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to deactivate customer")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func parseUUID(s string) *uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

func parseIntParam(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
