---
phase: pg-2-adapters
plan: 04
type: execute
wave: 2
depends_on: [pg-2-01]
files_modified:
  - internal/adapters/secondary/postgres/project_repository.go
  - internal/adapters/secondary/postgres/project_repository_test.go
  - internal/adapters/secondary/postgres/subproject_repository.go
  - internal/adapters/secondary/postgres/subproject_repository_test.go
  - internal/adapters/secondary/postgres/contract_repository.go
  - internal/adapters/secondary/postgres/contract_repository_test.go
  - internal/adapters/secondary/postgres/customer_repository.go
  - internal/adapters/secondary/postgres/customer_repository_test.go
autonomous: true
requirements: []
must_haves:
  truths:
    - ProjectRepository supports List, Create, Get, Adopt, ListManagers, AddManager, RemoveManager with scope-based filtering (adopted/all/own) and aggregate fields
    - ContractRepository supports List, Create, Get, Adopt, Update (dynamic SET), RecalculateMileage, Delete, HasTimeEntries with LEFT JOINs replacing nested subqueries
    - CustomerRepository supports ListByOrg, Create, GetByID, Update, Deactivate, ListContractsByCustomer, CountContractsByCustomer
    - SubprojectRepository supports ListByProject, GetByID, Create, Update, Delete
    - All JOIN queries use SQL aliases and scan structs (not domain types directly)
    - Aggregate fields (adoption_count, is_adopted, time_entries_count) use SQL subqueries
  artifacts:
    - path: internal/adapters/secondary/postgres/project_repository.go
      provides: implements ports.ProjectRepository (7 methods with JOINs + scope filtering)
    - path: internal/adapters/secondary/postgres/subproject_repository.go
      provides: implements ports.SubprojectRepository (5 methods, simple CRUD)
    - path: internal/adapters/secondary/postgres/contract_repository.go
      provides: implements ports.ContractRepository (8 methods with aggregates + dynamic SET)
    - path: internal/adapters/secondary/postgres/customer_repository.go
      provides: implements ports.CustomerRepository (7 methods with org_id→OrganizationID mapping)
  key_links:
    - from: project_repository.go
      to: contracts, organizations tables
      via: LEFT JOIN for contract_name, created_by_org_name
    - from: contract_repository.go
      to: organizations, customers, contract_adoptions tables
      via: LEFT JOIN + aggregate subqueries
    - from: contract_repository.go Update
      to: dynamic SET building
      via: fmt.Sprintf("name = $%d", n) for non-zero fields
    - from: customer_repository.go
      to: customers table (name→CompanyName, org_id→OrganizationID)
      via: positional Scan
---

<objective>
Implement ProjectRepository (with SubprojectRepository), ContractRepository, and CustomerRepository for PostgreSQL.

Purpose: Port the billings/customer domain repos from SurrealDB. These repos have the most complex SQL — scope-based filtering with JOINs for project listing, aggregate fields (adoption_count, is_adopted, time_entries_count) for contracts, and dynamic UPDATE SET building for contract updates.

Output:
- project_repository.go + subproject_repository.go + tests
- contract_repository.go + tests
- customer_repository.go + tests
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-2-adapters/pg-2-PATTERNS.md
@.planning/phases/pg-2-adapters/pg-2-RESEARCH.md

# Port interfaces
@internal/core/ports/project_repository.go
@internal/core/ports/contract_repository.go
@internal/core/ports/customer_repository.go

# SurrealDB analogs (patterns to port + JOIN translation)
@internal/adapters/secondary/surrealdb/project_repository.go
@internal/adapters/secondary/surrealdb/contract_repository.go
@internal/adapters/secondary/surrealdb/customer_repository.go

# Domain models
@internal/core/domain/project/project.go
@internal/core/domain/contract/contract.go
@internal/core/domain/customer/customer.go

# Foundation helpers
@internal/adapters/secondary/postgres/postgres.go
@internal/adapters/secondary/postgres/exported_test_helpers.go

# Schema
@migrations/002_full_schema.up.sql (projects, subprojects, project_adoptions, project_managers, contracts, contract_adoptions, customers tables)

<interfaces>
From internal/core/ports/project_repository.go:
```go
type ProjectRepository interface {
    List(ctx context.Context, orgID uuid.UUID, scope, contractID string) ([]projectdomain.ProjectResponse, error)
    Create(ctx context.Context, orgID uuid.UUID, req *projectdomain.CreateProjectRequest) (*projectdomain.ProjectResponse, error)
    Get(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectResponse, error)
    Adopt(ctx context.Context, orgID, projectID uuid.UUID) (*projectdomain.ProjectAdoption, error)
    ListManagers(ctx context.Context, projectID uuid.UUID) ([]projectdomain.ProjectManager, error)
    AddManager(ctx context.Context, projectID, userID uuid.UUID) (*projectdomain.ProjectManager, error)
    RemoveManager(ctx context.Context, projectID, userID uuid.UUID) error
}
```

From internal/core/ports/contract_repository.go:
```go
type ContractRepository interface {
    List(ctx context.Context, orgID uuid.UUID, scope string, isActive *bool) ([]contractdomain.ContractResponse, error)
    Create(ctx context.Context, orgID uuid.UUID, req *contractdomain.CreateContractRequest) (*contractdomain.ContractResponse, error)
    Get(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractResponse, error)
    Adopt(ctx context.Context, orgID, contractID uuid.UUID) (*contractdomain.ContractAdoption, error)
    Update(ctx context.Context, orgID, contractID uuid.UUID, req *contractdomain.UpdateContractRequest) (*contractdomain.ContractResponse, int, error)
    RecalculateMileage(ctx context.Context, orgID, contractID uuid.UUID, fromDate string, actorUserID uuid.UUID) (int, error)
    Delete(ctx context.Context, orgID, contractID uuid.UUID) error
    HasTimeEntries(ctx context.Context, contractID uuid.UUID) (int, error)
}
```

From internal/core/ports/customer_repository.go:
```go
type CustomerRepository interface {
    ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]customer.Customer, error)
    Create(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
    GetByID(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
    Update(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
    Deactivate(ctx context.Context, id uuid.UUID) error
    ListContractsByCustomer(ctx context.Context, customerID uuid.UUID) ([]customer.ContractSummary, error)
    CountContractsByCustomer(ctx context.Context, customerID uuid.UUID) (int, error)
}
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: ProjectRepository + SubprojectRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/project_repository.go
    internal/adapters/secondary/postgres/subproject_repository.go
    internal/adapters/secondary/postgres/project_repository_test.go
    internal/adapters/secondary/postgres/subproject_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/project_repository.go (interface — 7 methods)
    @internal/core/domain/project/project.go (Project, ProjectResponse, ProjectAdoption, ProjectManager, CreateProjectRequest)
    @internal/adapters/secondary/surrealdb/project_repository.go (analog — scope filtering + JOIN pattern)
    @migrations/002_full_schema.up.sql (projects, project_adoptions, project_managers tables)
    @internal/adapters/secondary/postgres/exported_test_helpers.go
  </read_first>

  <action>
    **A) project_repository.go** — implements ports.ProjectRepository (7 methods)
    Struct `ProjectRepository` with `pool *pgxpool.Pool`, constructor `NewProjectRepository`.

    **SQL JOIN translation pattern** (per D-06):
    SurrealDB nested subqueries → SQL LEFT JOINs:
    ```
    -- SurrealDB:
    (SELECT VALUE name FROM contracts WHERE id = p.contract_id LIMIT 1)[0] AS contract_name
    -- PG:
    LEFT JOIN contracts c ON c.id = p.contract_id
    -- SELECT c.name AS contract_name
    ```

    **Scan struct needed** — projectResponseRow (since ProjectResponse embeds Project which can't be scanned directly):
    ```go
    type projectRow struct {
        ID              uuid.UUID `db:"id"`
        Name            string    `db:"name"`
        Type            string    `db:"type"`
        ContractID      uuid.UUID `db:"contract_id"`
        GovernanceModel string    `db:"governance_model"`
        CreatedByOrgID  uuid.UUID `db:"created_by_org_id"`
        IsShared        bool      `db:"is_shared"`
        IsActive        bool      `db:"is_active"`
        CreatedAt       time.Time `db:"created_at"`
        ContractName    string    `db:"contract_name"`
        OrgName         string    `db:"created_by_org_name"`
        AdoptionCount   int       `db:"adoption_count"`
        IsAdopted       bool      `db:"is_adopted"`
    }
    ```
    Use positional Scan() instead of CollectRows+RowToStructByName (avoids db: tag requirement).

    **List**: Build WHERE clause from scope ("adopted"/"all"/default=own org). If contractID != "", add filter.
    ```sql
    SELECT p.id, p.name, p.type, p.contract_id, p.governance_model, p.created_by_org_id,
           p.is_shared, p.is_active, p.created_at,
           c.name AS contract_name, o.name AS created_by_org_name,
           (SELECT COUNT(*) FROM project_adoptions pa WHERE pa.project_id = p.id) AS adoption_count,
           EXISTS(SELECT 1 FROM project_adoptions pa2 WHERE pa2.project_id = p.id AND pa2.organization_id = $1) AS is_adopted
    FROM projects p
    LEFT JOIN contracts c ON c.id = p.contract_id
    LEFT JOIN organizations o ON o.id = p.created_by_org_id
    WHERE p.is_active = true AND {scope_condition}
    ORDER BY p.created_at DESC
    ```
    Scope conditions:
    - "adopted": `p.id IN (SELECT pa.project_id FROM project_adoptions pa WHERE pa.organization_id = $N)`
    - "all": `p.is_shared = true`
    - default: `p.created_by_org_id = $N`
    Use numbered placeholders ($1, $2, etc.) with args slice.

    **Get**: Same SELECT with `WHERE p.id = $N AND p.is_active = true LIMIT 1`

    **Create**: INSERT into projects table with RETURNING *, then Get to return ProjectResponse.

    **Adopt**: First check existing with SELECT EXISTS; if already adopted return projectdomain.ErrAlreadyAdopted. Then INSERT into project_adoptions.

    **ListManagers**: 
    ```sql
    SELECT pm.id, pm.project_id, pm.user_id,
           u.firstname || ' ' || u.lastname AS user_name, u.email AS email, pm.created_at
    FROM project_managers pm
    LEFT JOIN users u ON u.id = pm.user_id
    WHERE pm.project_id = $1
    ORDER BY pm.created_at ASC
    ```

    **AddManager**: SELECT EXISTS on users to verify user exists and is active. INSERT into project_managers. ListManagers to find created entry. Return ProjectManager.

    **RemoveManager**: SELECT id FROM project_managers WHERE project_id=$1 AND user_id=$2, then DELETE by id.

    **B) subproject_repository.go** — implements ports.SubprojectRepository (5 methods)
    Struct `SubprojectRepository` with `pool *pgxpool.Pool`, constructor `NewSubprojectRepository`.

    Simple CRUD against `subprojects` table:
    - **ListByProject**: `SELECT id, project_id, name, description, sequence_order, is_active, created_at, updated_at FROM subprojects WHERE project_id = $1 AND is_active = true ORDER BY sequence_order`
    - **GetByID**: Same SELECT with `WHERE id = $1`. Parse id string to uuid.UUID.
      ErrNoRows → return nil, nil (no domain-specific error yet — Subproject is from models)
    - **Create**: INSERT with RETURNING *
    - **Update**: UPDATE SET with RETURNING *
    - **Delete**: DELETE FROM subprojects WHERE id = $1

    **C) Test files:**
    - `project_repository_test.go`: Test Create→Get (verify aggregate fields), Test List with scope="adopted"/"all"/default, Test Adopt (verify ErrAlreadyAdopted on duplicate), Test ListManagers (verify user_name is concatenated), Test AddManager→ListManagers→RemoveManager
    - `subproject_repository_test.go`: Test Create→GetByID, Test ListByProject, Test Update, Test Delete
    - Seed data: create org, user, contract, then projects/subprojects
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. ProjectRepository with 7 methods compiles, JOIN queries use LEFT JOIN (not nested subqueries)
    2. SubprojectRepository with 5 methods compiles
    3. Test files compile
    4. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

<task type="auto">
  <name>Task 2: ContractRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/contract_repository.go
    internal/adapters/secondary/postgres/contract_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/contract_repository.go (interface — 8 methods)
    @internal/core/domain/contract/contract.go (Contract, ContractResponse, ContractAdoption, CreateContractRequest, UpdateContractRequest)
    @internal/adapters/secondary/surrealdb/contract_repository.go (analog — aggregates, dynamic SET, mileage recalc)
    @internal/adapters/secondary/surrealdb/time_entry_repository.go (for RecalculateMileage pattern)
    @migrations/002_full_schema.up.sql (contracts, contract_adoptions tables)
  </read_first>

  <action>
    **A) contract_repository.go** — implements ports.ContractRepository (8 methods)
    Struct `ContractRepository` with `pool *pgxpool.Pool`, constructor `NewContractRepository`.

    **SQL for Get (the most complex query — all aggregate fields):**
    ```sql
    SELECT c.id, c.name, c.km_rate, c.currency, c.customer_id, c.governance_model,
           c.created_by_org_id, c.is_shared, c.is_active, c.created_at,
           o.name AS created_by_org_name,
           (SELECT COUNT(*) FROM contract_adoptions ca WHERE ca.contract_id = c.id) AS adoption_count,
           EXISTS(SELECT 1 FROM contract_adoptions ca2 WHERE ca2.contract_id = c.id AND ca2.organization_id = $1) AS is_adopted,
           COALESCE((SELECT cu.name FROM customers cu WHERE cu.id = c.customer_id), '') AS customer_name,
           (SELECT COUNT(*) FROM time_entries te WHERE te.project_id IN (SELECT p.id FROM projects p WHERE p.contract_id = c.id)) AS time_entries_count
    FROM contracts c
    LEFT JOIN organizations o ON o.id = c.created_by_org_id
    WHERE c.id = $2
    ```

    **Scan struct:**
    Use positional Scan with individual variables matching SELECT order:
    id, name, km_rate, currency, customer_id(*uuid.UUID), governance_model, created_by_org_id, is_shared, is_active, created_at, created_by_org_name, adoption_count, is_adopted, customer_name, time_entries_count

    **Methods:**

    1. **List**: Same SELECT as Get but with scope WHERE clause and ORDER BY c.created_at DESC, returns []ContractResponse. Scope: "adopted"/"all"/default.

    2. **Create**: INSERT with all fields from CreateContractRequest. Use uuid.New(). RETURNING not needed — call Get to return full ContractResponse.

    3. **Get**: The complex query above. ErrNoRows → contractdomain.ErrContractNotFound.

    4. **Adopt**: Check existing (SELECT EXISTS). If exists → contractdomain.ErrAlreadyAdopted. INSERT into contract_adoptions.

    5. **Update**: Dynamic SET building (per RESEARCH Pattern 11):
       ```go
       args := []any{}
       argN := 1
       setClauses := []string{}
       if req.Name != "" { setClauses = append(setClauses, fmt.Sprintf("name = $%d", argN)); args = append(args, req.Name); argN++ }
       if req.KmRate != nil { setClauses = append(setClauses, fmt.Sprintf("km_rate = $%d", argN)); args = append(args, *req.KmRate); argN++ }
       if req.Currency != "" { setClauses = append(setClauses, fmt.Sprintf("currency = $%d", argN)); args = append(args, req.Currency); argN++ }
       if req.GovernanceModel != "" { ... }
       if req.IsShared != nil { ... }
       if req.IsActive != nil { ... }
       if req.CustomerID != nil && *req.CustomerID != "" {
           cid, _ := uuid.Parse(*req.CustomerID)
           setClauses = append(setClauses, fmt.Sprintf("customer_id = $%d", argN))
           args = append(args, cid)
           argN++
       }
       setClauses = append(setClauses, "updated_at = NOW()")
       args = append(args, contractID, orgID)
       query := fmt.Sprintf("UPDATE contracts SET %s WHERE id = $%d AND created_by_org_id = $%d",
           strings.Join(setClauses, ", "), argN, argN+1)
       ```
       Then return r.Get(ctx, orgID, contractID). Return 0 for second return value (mileage count — not used here).

    6. **RecalculateMileage**: SELECT expenses with km_distance NOT NULL for projects under this contract. For each, compute amount = km * contract.km_rate, UPDATE expense. Return count of updated expenses.
       - Variation: Actually check the surrealdb version — it updates each expense amount individually. Use a simpler approach: UPDATE expenses SET amount = km_distance * $1 WHERE project_id IN (SELECT id FROM projects WHERE contract_id = $2) AND km_distance IS NOT NULL AND is_deleted = false. Return COUNT via RETURNING or a separate count.

    7. **Delete**: `DELETE FROM contracts WHERE id = $1 AND created_by_org_id = $2`

    8. **HasTimeEntries**: `SELECT COUNT(*) FROM time_entries WHERE project_id IN (SELECT id FROM projects WHERE contract_id = $1) AND is_deleted = false`

    **B) contract_repository_test.go:**
    - Test Create→Get (verify aggregate fields: adoption_count=0, is_adopted=false, customer_name="")
    - Test List with scope filtering
    - Test Update (change name, km_rate, verify via Get)
    - Test Adopt (verify is_adopted=true, adoption_count=1 after adoption)
    - Test HasTimeEntries (returns 0 for new contract with no entries)
    - Test Delete
    - Seed: org, customer, user (for FK), then contract
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. ContractRepository with 8 methods compiles
    2. Aggregate fields use SQL subqueries (not SurrealDB nested subquery pattern)
    3. Dynamic UPDATE SET building works for all UpdateContractRequest fields
    4. Test file compiles
    5. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

<task type="auto">
  <name>Task 3: CustomerRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/customer_repository.go
    internal/adapters/secondary/postgres/customer_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/customer_repository.go (interface — 7 methods)
    @internal/core/domain/customer/customer.go (Customer struct: OrganizationID uuid.UUID, CompanyName string, etc.)
    @internal/adapters/secondary/surrealdb/customer_repository.go (analog)
    @migrations/002_full_schema.up.sql (customers table: org_id UUID, name VARCHAR — column names differ from domain)
  </read_first>

  <action>
    **A) customer_repository.go** — implements ports.CustomerRepository (7 methods)
    Struct `CustomerRepository` with `pool *pgxpool.Pool`, constructor `NewCustomerRepository`.

    **Column mapping (SQL name ≠ domain field):**
    - SQL `org_id` → domain `OrganizationID`
    - SQL `name` → domain `CompanyName`
    - SQL `vat_number` → domain `VATNumber`
    Since Scan() is positional, column order matters, not name.

    **Methods:**

    1. **ListByOrg**: `SELECT id, org_id, name, contact_name, email, phone, vat_number, address, is_active, created_at, updated_at FROM customers WHERE org_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
       - Scan into variables, map to customer.Customer fields
       - Return []customer.Customer{} not nil

    2. **Create**: `INSERT INTO customers (id, org_id, name, contact_name, email, phone, vat_number, address, is_active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id, org_id, name, contact_name, email, phone, vat_number, address, is_active, created_at, updated_at`
       - Scan RETURNING into customer.Customer (positional)

    3. **GetByID**: `SELECT id, org_id, name, contact_name, email, phone, vat_number, address, is_active, created_at FROM customers WHERE id = $1`
       - Note: customers table has `created_at` but no `updated_at` column in schema! (Check schema: customers has created_at, updated_at). Include both in SELECT.
       - ErrNoRows → customer.ErrCustomerNotFound

    4. **Update**: `UPDATE customers SET name=$1, contact_name=$2, email=$3, phone=$4, vat_number=$5, address=$6, is_active=$7, updated_at=NOW() WHERE id=$8 RETURNING *`
       - Full-field update (all fields from Customer struct)

    5. **Deactivate**: `UPDATE customers SET is_active = false, updated_at = NOW() WHERE id = $1`

    6. **ListContractsByCustomer**: `SELECT id, name, km_rate, currency, customer_id, governance_model, created_by_org_id, is_shared, is_active, created_at FROM contracts WHERE customer_id = $1 ORDER BY created_at DESC`
       - Map to customer.ContractSummary (positional scan)

    7. **CountContractsByCustomer**: `SELECT COUNT(*) FROM contracts WHERE customer_id = $1`

    **B) customer_repository_test.go:**
    - Test ListByOrg (create 2 customers, list, verify count)
    - Test Create→GetByID (create, retrieve, assert CompanyName matches)
    - Test Update (change company name, verify)
    - Test Deactivate (deactivate, verify is_active=false)
    - Test ListContractsByCustomer (create contract with customer_id, list, verify count)
    - Test CountContractsByCustomer
    - Seed: org (for FK), then customers
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. CustomerRepository with 7 methods compiles
    2. Positional Scan correctly handles org_id→OrganizationID, name→CompanyName mapping
    3. Test file compiles
    4. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Application → PostgreSQL project/contract/customer queries | Billing domain data flows through parameterized queries |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg2-11 | Tampering | contract Update dynamic SET | mitigate | Dynamic SET only injects placeholder positions ($N), never raw values from user input. fmt.Sprintf with $d is safe. |
| T-pg2-12 | Spoofing | project List scope filtering | mitigate | Scope parameter is a fixed string ("adopted"/"all"/"other") — switch case, not direct string interpolation |
| T-pg2-13 | Information Disclosure | project_manager queries user_name | mitigate | Only returns concatenated firstname+lastname + email, never password_hash |
</threat_model>

<verification>
```bash
go vet ./internal/adapters/secondary/postgres/
go build ./internal/adapters/secondary/postgres/
# Integration tests:
# DATABASE_URL=... go test ./internal/adapters/secondary/postgres/ -run "TestProject|TestContract|TestCustomer|TestSubproject" -count=1 -v
```
</verification>

<success_criteria>
1. ProjectRepository implements all 7 methods with JOIN-based aggregate fields
2. ContractRepository implements all 8 methods with dynamic UPDATE SET
3. CustomerRepository implements all 7 methods with column→domain field mapping
4. SubprojectRepository implements all 5 methods
5. All tests compile
6. Files committed to git
</success_criteria>

<output>
After completion, create `.planning/phases/pg-2-adapters/pg-2-04-SUMMARY.md`
</output>
