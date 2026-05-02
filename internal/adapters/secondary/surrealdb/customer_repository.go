package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	customerdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/customer"
	appmodels "github.com/stefanoprivitera/hourglass/internal/models"
	sdb "github.com/surrealdb/surrealdb.go"
	sdbmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

type CustomerRepository struct {
	db *sdb.DB
}

func NewCustomerRepository(db *sdb.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

type surrealCustomer struct {
	ID          sdbmodels.RecordID `json:"id,omitempty"`
	OrgID       sdbmodels.RecordID `json:"org_id"`
	Name        string          `json:"name"`
	ContactName string          `json:"contact_name,omitempty"`
	Email       string          `json:"email,omitempty"`
	Phone       string          `json:"phone,omitempty"`
	Address     string          `json:"address,omitempty"`
	VATNumber   string          `json:"vat_number,omitempty"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type surrealContract struct {
	ID              sdbmodels.RecordID `json:"id,omitempty"`
	Name            string          `json:"name"`
	KmRate          float64         `json:"km_rate"`
	Currency        string          `json:"currency"`
	CustomerID      sdbmodels.RecordID `json:"customer_id,omitempty"`
	GovernanceModel string          `json:"governance_model"`
	CreatedByOrgID  sdbmodels.RecordID `json:"created_by_org_id"`
	IsShared        bool            `json:"is_shared"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (r *CustomerRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]customerdomain.Customer, error) {
	orgRecordID := uuidToRecordID("organizations", orgID)
	results, err := sdb.Query[[]surrealCustomer](ctx, r.db, `
		SELECT * FROM customers
		WHERE org_id = $org_id
		ORDER BY created_at DESC
		LIMIT $limit START $offset
	`, map[string]interface{}{
		"org_id": orgRecordID,
		"limit":  limit,
		"offset": offset,
	})
	if err != nil {
		return nil, wrapErr(err, "list customers")
	}
	if results == nil || len(*results) == 0 {
		return []customerdomain.Customer{}, nil
	}

	items := (*results)[0].Result
	out := make([]customerdomain.Customer, 0, len(items))
	for _, c := range items {
		out = append(out, customerdomain.Customer{
			ID:             recordIDToUUID(c.ID),
			OrganizationID: recordIDToUUID(c.OrgID),
			CompanyName:    c.Name,
			ContactName:    c.ContactName,
			Email:          c.Email,
			Phone:          c.Phone,
			VATNumber:      c.VATNumber,
			Address:        c.Address,
			IsActive:       c.IsActive,
			CreatedAt:      c.CreatedAt,
		})
	}
	return out, nil
}

func (r *CustomerRepository) Create(ctx context.Context, c *customerdomain.Customer) (*customerdomain.Customer, error) {
	data := map[string]interface{}{
		"id":           uuidToRecordID("customers", c.ID),
		"org_id":       uuidToRecordID("organizations", c.OrganizationID),
		"name":         c.CompanyName,
		"contact_name": c.ContactName,
		"email":        c.Email,
		"phone":        c.Phone,
		"vat_number":   c.VATNumber,
		"address":      c.Address,
		"is_active":    c.IsActive,
		"created_at":   c.CreatedAt,
		"updated_at":   c.CreatedAt,
	}
	created, err := sdb.Create[surrealCustomer](ctx, r.db, sdbmodels.Table("customers"), data)
	if err != nil {
		return nil, wrapErr(err, "create customer")
	}

	return &customerdomain.Customer{
		ID:             recordIDToUUID(created.ID),
		OrganizationID: recordIDToUUID(created.OrgID),
		CompanyName:    created.Name,
		ContactName:    created.ContactName,
		Email:          created.Email,
		Phone:          created.Phone,
		VATNumber:      created.VATNumber,
		Address:        created.Address,
		IsActive:       created.IsActive,
		CreatedAt:      created.CreatedAt,
	}, nil
}

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*customerdomain.Customer, error) {
	recordID := uuidToRecordID("customers", id)
	result, err := sdb.Select[surrealCustomer](ctx, r.db, recordID)
	if err != nil {
		return nil, customerdomain.ErrCustomerNotFound
	}
	return &customerdomain.Customer{
		ID:             recordIDToUUID(result.ID),
		OrganizationID: recordIDToUUID(result.OrgID),
		CompanyName:    result.Name,
		ContactName:    result.ContactName,
		Email:          result.Email,
		Phone:          result.Phone,
		VATNumber:      result.VATNumber,
		Address:        result.Address,
		IsActive:       result.IsActive,
		CreatedAt:      result.CreatedAt,
	}, nil
}

func (r *CustomerRepository) Update(ctx context.Context, c *customerdomain.Customer) (*customerdomain.Customer, error) {
	recordID := uuidToRecordID("customers", c.ID)
	result, err := sdb.Merge[surrealCustomer](ctx, r.db, recordID, map[string]interface{}{
		"name":         c.CompanyName,
		"contact_name": c.ContactName,
		"email":        c.Email,
		"phone":        c.Phone,
		"vat_number":   c.VATNumber,
		"address":      c.Address,
		"is_active":    c.IsActive,
		"updated_at":   time.Now(),
	})
	if err != nil {
		return nil, wrapErr(err, "update customer")
	}
	return &customerdomain.Customer{
		ID:             recordIDToUUID(result.ID),
		OrganizationID: recordIDToUUID(result.OrgID),
		CompanyName:    result.Name,
		ContactName:    result.ContactName,
		Email:          result.Email,
		Phone:          result.Phone,
		VATNumber:      result.VATNumber,
		Address:        result.Address,
		IsActive:       result.IsActive,
		CreatedAt:      result.CreatedAt,
	}, nil
}

func (r *CustomerRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	recordID := uuidToRecordID("customers", id)
	_, err := sdb.Merge[surrealCustomer](ctx, r.db, recordID, map[string]interface{}{
		"is_active":  false,
		"updated_at": time.Now(),
	})
	return wrapErr(err, "deactivate customer")
}

func (r *CustomerRepository) ListContractsByCustomer(ctx context.Context, customerID uuid.UUID) ([]customerdomain.ContractSummary, error) {
	customerRecordID := uuidToRecordID("customers", customerID)
	results, err := sdb.Query[[]surrealContract](ctx, r.db, `
		SELECT * FROM contracts WHERE customer_id = $customer_id
	`, map[string]interface{}{"customer_id": customerRecordID})
	if err != nil {
		return nil, wrapErr(err, "list linked contracts")
	}
	if results == nil || len(*results) == 0 {
		return []customerdomain.ContractSummary{}, nil
	}

	items := (*results)[0].Result
	out := make([]customerdomain.ContractSummary, 0, len(items))
	for _, c := range items {
		contract := customerdomain.ContractSummary{
			ID:              recordIDToUUID(c.ID),
			Name:            c.Name,
			KmRate:          c.KmRate,
			Currency:        c.Currency,
			GovernanceModel: appmodels.GovernanceModel(c.GovernanceModel),
			CreatedByOrgID:  recordIDToUUID(c.CreatedByOrgID),
			IsShared:        c.IsShared,
			IsActive:        c.IsActive,
			CreatedAt:       c.CreatedAt,
		}
		if cid := recordIDToUUIDPtr(c.CustomerID); cid != nil {
			contract.CustomerID = cid
		}
		out = append(out, contract)
	}
	return out, nil
}

func (r *CustomerRepository) CountContractsByCustomer(ctx context.Context, customerID uuid.UUID) (int, error) {
	customerRecordID := uuidToRecordID("customers", customerID)
	results, err := sdb.Query[[]map[string]interface{}](ctx, r.db, `
		SELECT count() FROM contracts WHERE customer_id = $customer_id GROUP ALL
	`, map[string]interface{}{"customer_id": customerRecordID})
	if err != nil {
		return 0, wrapErr(err, "count linked contracts")
	}
	if results == nil || len(*results) == 0 || len((*results)[0].Result) == 0 {
		return 0, nil
	}
	if count, ok := (*results)[0].Result[0]["count"].(float64); ok {
		return int(count), nil
	}
	return 0, nil
}
