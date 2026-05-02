package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/models"
	sdb "github.com/surrealdb/surrealdb.go"
	sdbmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

type ContractRepository struct {
	db *sdb.DB
}

func NewContractRepository(db *sdb.DB) *ContractRepository {
	return &ContractRepository{db: db}
}

type surrealContractCompat struct {
	ID              sdbmodels.RecordID `json:"id,omitempty"`
	Name            string             `json:"name"`
	KmRate          float64            `json:"km_rate"`
	Currency        string             `json:"currency"`
	CustomerID      sdbmodels.RecordID `json:"customer_id,omitempty"`
	GovernanceModel string             `json:"governance_model"`
	CreatedByOrgID  sdbmodels.RecordID `json:"created_by_org_id"`
	IsShared        bool               `json:"is_shared"`
	IsActive        bool               `json:"is_active"`
	CreatedAt       time.Time          `json:"created_at"`
}

type contractJoined struct {
	surrealContractCompat
	OrgName     string `json:"created_by_org_name,omitempty"`
	AdoptionCnt int    `json:"adoption_count,omitempty"`
	IsAdopted   bool   `json:"is_adopted,omitempty"`
}

func (r *ContractRepository) List(ctx context.Context, orgID uuid.UUID, scope string) ([]contractdomain.ContractResponse, error) {
	where := "WHERE c.is_active = true"
	vars := map[string]interface{}{"org_id": uuidToRecordID("organizations", orgID)}
	switch scope {
	case "adopted":
		where += " AND c.id IN (SELECT VALUE contract_id FROM contract_adoptions WHERE organization_id = $org_id)"
	case "all":
		where += " AND c.is_shared = true"
	default:
		where += " AND c.created_by_org_id = $org_id"
	}
	results, err := sdb.Query[[]contractJoined](ctx, r.db, `
		SELECT c.*,
			(SELECT VALUE name FROM organizations WHERE id = c.created_by_org_id LIMIT 1)[0] AS created_by_org_name,
			count((SELECT VALUE id FROM contract_adoptions WHERE contract_id = c.id)) AS adoption_count,
			(SELECT VALUE count() > 0 FROM contract_adoptions WHERE contract_id = c.id AND organization_id = $org_id GROUP ALL)[0] AS is_adopted
		FROM contracts c `+where+` ORDER BY c.created_at DESC`, vars)
	if err != nil || results == nil || len(*results) == 0 {
		return []contractdomain.ContractResponse{}, nil
	}
	out := make([]contractdomain.ContractResponse, 0, len((*results)[0].Result))
	for _, c := range (*results)[0].Result {
		resp := contractdomain.ContractResponse{
			Contract: contractdomain.Contract{
				ID:              recordIDToUUID(c.ID),
				Name:            c.Name,
				KmRate:          c.KmRate,
				Currency:        c.Currency,
				GovernanceModel: models.GovernanceModel(c.GovernanceModel),
				CreatedByOrgID:  recordIDToUUID(c.CreatedByOrgID),
				IsShared:        c.IsShared,
				IsActive:        c.IsActive,
				CreatedAt:       c.CreatedAt,
			},
			CreatedByOrgName: c.OrgName,
			AdoptionCount:    c.AdoptionCnt,
			IsAdopted:        c.IsAdopted,
		}
		if cid := recordIDToUUIDPtr(c.CustomerID); cid != nil {
			resp.CustomerID = cid
		}
		out = append(out, resp)
	}
	return out, nil
}

func (r *ContractRepository) Create(ctx context.Context, orgID uuid.UUID, req *contractdomain.CreateContractRequest) (*contractdomain.ContractResponse, error) {
	id := uuid.New()
	now := time.Now()
	data := map[string]interface{}{
		"id":                uuidToRecordID("contracts", id),
		"name":              req.Name,
		"km_rate":           req.KmRate,
		"currency":          req.Currency,
		"governance_model":  string(req.GovernanceModel),
		"created_by_org_id": uuidToRecordID("organizations", orgID),
		"is_shared":         req.IsShared,
		"is_active":         true,
		"created_at":        now,
		"updated_at":        now,
	}
	if _, err := sdb.Create[surrealContractCompat](ctx, r.db, sdbmodels.Table("contracts"), data); err != nil {
		return nil, wrapErr(err, "create contract")
	}
	return r.Get(ctx, orgID, id)
}

func (r *ContractRepository) Get(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractResponse, error) {
	results, err := sdb.Query[[]contractJoined](ctx, r.db, `
		SELECT c.*,
			(SELECT VALUE name FROM organizations WHERE id = c.created_by_org_id LIMIT 1)[0] AS created_by_org_name,
			count((SELECT VALUE id FROM contract_adoptions WHERE contract_id = c.id)) AS adoption_count,
			(SELECT VALUE count() > 0 FROM contract_adoptions WHERE contract_id = c.id AND organization_id = $org_id GROUP ALL)[0] AS is_adopted
		FROM contracts c WHERE c.id=$contract_id AND c.is_active=true LIMIT 1`, map[string]interface{}{
		"contract_id": uuidToRecordID("contracts", contractID),
		"org_id":      uuidToRecordID("organizations", orgID),
	})
	if err != nil || results == nil || len(*results) == 0 || len((*results)[0].Result) == 0 {
		return nil, contractdomain.ErrContractNotFound
	}
	c := (*results)[0].Result[0]
	resp := &contractdomain.ContractResponse{
		Contract: contractdomain.Contract{
			ID:              recordIDToUUID(c.ID),
			Name:            c.Name,
			KmRate:          c.KmRate,
			Currency:        c.Currency,
			GovernanceModel: models.GovernanceModel(c.GovernanceModel),
			CreatedByOrgID:  recordIDToUUID(c.CreatedByOrgID),
			IsShared:        c.IsShared,
			IsActive:        c.IsActive,
			CreatedAt:       c.CreatedAt,
		},
		CreatedByOrgName: c.OrgName,
		AdoptionCount:    c.AdoptionCnt,
		IsAdopted:        c.IsAdopted,
	}
	if cid := recordIDToUUIDPtr(c.CustomerID); cid != nil {
		resp.CustomerID = cid
	}
	return resp, nil
}

func (r *ContractRepository) Adopt(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractAdoption, error) {
	existing, _ := sdb.Query[[]map[string]interface{}](ctx, r.db, `SELECT count() FROM contract_adoptions WHERE contract_id=$contract_id AND organization_id=$org_id GROUP ALL`, map[string]interface{}{
		"contract_id": uuidToRecordID("contracts", contractID),
		"org_id":      uuidToRecordID("organizations", orgID),
	})
	if existing != nil && len(*existing) > 0 && len((*existing)[0].Result) > 0 {
		if cnt, ok := (*existing)[0].Result[0]["count"].(float64); ok && cnt > 0 {
			return nil, contractdomain.ErrAlreadyAdopted
		}
	}
	id := uuid.New()
	now := time.Now()
	if _, err := sdb.Create[map[string]interface{}](ctx, r.db, sdbmodels.Table("contract_adoptions"), map[string]interface{}{
		"id":              uuidToRecordID("contract_adoptions", id),
		"contract_id":     uuidToRecordID("contracts", contractID),
		"organization_id": uuidToRecordID("organizations", orgID),
		"adopted_at":      now,
	}); err != nil {
		return nil, wrapErr(err, "adopt contract")
	}
	return &contractdomain.ContractAdoption{ID: id, ContractID: contractID, OrganizationID: orgID, AdoptedAt: now}, nil
}

func (r *ContractRepository) Update(ctx context.Context, orgID, contractID uuid.UUID, req *contractdomain.UpdateContractRequest) (*contractdomain.ContractResponse, int, error) {
	current, err := r.Get(ctx, orgID, contractID)
	if err != nil {
		return nil, 0, err
	}
	if current.CreatedByOrgID != orgID {
		return nil, 0, contractdomain.ErrForbidden
	}
	data := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != "" {
		data["name"] = req.Name
	}
	if req.KmRate != nil {
		data["km_rate"] = *req.KmRate
	}
	if req.Currency != "" {
		data["currency"] = req.Currency
	}
	if req.GovernanceModel != "" {
		data["governance_model"] = string(req.GovernanceModel)
	}
	if req.IsShared != nil {
		data["is_shared"] = *req.IsShared
	}
	if req.IsActive != nil {
		data["is_active"] = *req.IsActive
	}
	if req.CustomerID != nil && *req.CustomerID != "" {
		if cid, err := uuid.Parse(*req.CustomerID); err == nil {
			data["customer_id"] = uuidToRecordID("customers", cid)
		}
	}
	if _, err := sdb.Merge[surrealContractCompat](ctx, r.db, uuidToRecordID("contracts", contractID), data); err != nil {
		return nil, 0, wrapErr(err, "update contract")
	}
	updated, err := r.Get(ctx, orgID, contractID)
	return updated, 0, err
}

func (r *ContractRepository) RecalculateMileage(ctx context.Context, orgID, contractID uuid.UUID, fromDate string, actorUserID uuid.UUID) (int, error) {
	results, err := sdb.Query[[]map[string]interface{}](ctx, r.db, `
		SELECT e.id, e.amount, e.km_distance
		FROM expenses e
		WHERE e.km_distance != NONE
		  AND e.deleted_at = NONE
		  AND e.project_id IN (SELECT VALUE id FROM projects WHERE contract_id = $contract_id)
		  AND e.expense_date >= <datetime>$from_date
	`, map[string]interface{}{
		"contract_id": uuidToRecordID("contracts", contractID),
		"from_date":   fromDate + "T00:00:00Z",
	})
	if err != nil || results == nil || len(*results) == 0 {
		return 0, nil
	}
	contract, err := r.Get(ctx, orgID, contractID)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range (*results)[0].Result {
		entryID, ok := row["id"].(sdbmodels.RecordID)
		if !ok {
			continue
		}
		km, ok := row["km_distance"].(float64)
		if !ok {
			continue
		}
		newAmount := km * contract.KmRate
		if _, err := sdb.Merge[map[string]interface{}](ctx, r.db, entryID, map[string]interface{}{"amount": newAmount, "updated_at": time.Now()}); err == nil {
			updated++
		}
	}
	return updated, nil
}
