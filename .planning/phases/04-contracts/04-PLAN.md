---
phase: 04-contracts
plan: 01
type: execute
wave: 1
depends_on: [03-01, 03-02, 03-03]
files_modified:
  - internal/core/domain/contract/contract.go
  - internal/core/ports/contract_repository.go
  - internal/core/services/contract/contract.go
  - internal/core/services/contract/contract_test.go
  - internal/core/services/testdata/mocks.go
  - internal/adapters/primary/http/contract.go
  - internal/adapters/secondary/postgres/contract_repository.go
  - web/src/types/api.ts
  - web/src/types/models.ts
  - web/src/api/contracts.ts
  - web/src/api/__tests__/contracts.test.ts
  - web/src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx
autonomous: true
requirements: [CTRT-02, CTRT-04, CTRT-06, CTRT-07]

must_haves:
  truths:
    - "User can create a contract and optionally link a customer via combobox"
    - "User cannot delete a contract that has projects (specific 409 error: 'contract has projects')"
    - "User cannot delete a contract that has time entries (existing 409: 'contract has time entries')"
    - "Internal customers show '(Internal)' suffix in customer combobox dropdown"
    - "'No customer' option appears as first item in the customer combobox"
    - "Backend stores customer_id when creating contract with customer selected"
    - "Backend stores NULL customer_id when creating contract with 'No customer'"

  artifacts:
    - path: internal/core/domain/contract/contract.go
      provides: CustomerID field on CreateContractRequest + ErrHasActiveProjects error
      min_lines: 65
    - path: internal/core/ports/contract_repository.go
      provides: HasProjects method on ContractRepository interface
      exports: ContractRepository
    - path: internal/core/services/contract/contract.go
      provides: HasProjects check in Delete service method
      min_lines: 83
    - path: internal/core/services/contract/contract_test.go
      provides: Unit tests for HasProjects check in Delete + customer_id in Create
      min_lines: 220
    - path: internal/core/services/testdata/mocks.go
      provides: HasProjects mock implementation on MockContractRepo
      exports: MockContractRepo
    - path: internal/adapters/primary/http/contract.go
      provides: CustomerID parsing on Create handler
      min_lines: 75
    - path: internal/adapters/secondary/postgres/contract_repository.go
      provides: customer_id in INSERT + HasProjects SQL query
      min_lines: 320
    - path: web/src/types/api.ts
      provides: customer_id on CreateContractRequest
    - path: web/src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx
      provides: Customer combobox with search, "(Internal)" suffix, "No customer" option
      min_lines: 200
    - path: web/src/api/__tests__/contracts.test.ts
      provides: Updated test for create with customer_id
      min_lines: 100

  key_links:
    - from: web/src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx
      to: web/src/api/contracts.ts
      via: "createContractMutationOpts"
      pattern: "createContract\\.mutate"
    - from: web/src/api/contracts.ts
      to: "POST /contracts"
      via: "api<Contract>('/contracts', {method: 'POST', body: JSON.stringify(data)})"
      pattern: "api<Contract>\\('/contracts'"
    - from: internal/adapters/primary/http/contract.go
      to: internal/core/services/contract/contract.go
      via: "conversion from handler CreateContractRequest to domain CreateContractRequest"
      pattern: "h.service.Create"
    - from: internal/core/services/contract/contract.go
      to: internal/core/ports/contract_repository.go
      via: "s.repo.HasProjects(ctx, contractID)"
      pattern: "s\\.repo\\.HasProjects"
    - from: internal/core/services/contract/contract.go
      to: contractdomain.ErrHasActiveProjects
      via: "return contractdomain.ErrHasActiveProjects"
      pattern: "ErrHasActiveProjects"
    - from: internal/adapters/secondary/postgres/contract_repository.go
      to: projects table
      via: "SELECT COUNT(*)" FROM projects WHERE contract_id = $1"
      pattern: "SELECT COUNT\\(\\*\\) FROM projects"
    - from: internal/adapters/primary/http/contract.go
      to: contractdomain.ErrHasActiveProjects
      via: "Delete handler error switch"
      pattern: "ErrHasActiveProjects"
---

<objective>
**Contract CRUD with customer dropdown, customer_id persistence on create, and project-based delete protection.**

**Purpose:** Fill the remaining gaps in the Contract feature — wire customer association into the create flow (matching the edit flow's existing capability) and harden delete protection (currently only guards time entries, not projects). This completes CTRT-02, CTRT-04, CTRT-06, and CTRT-07.

**Output:** Backend: domain model with `customer_id` on create + `ErrHasActiveProjects` error + `HasProjects` on repository + delete protection in service. Frontend: customer combobox in create dialog with "(Internal)" suffix and "No customer" option. Updated unit/tests.
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/04-contracts/04-CONTEXT.md
@.planning/phases/03-customers/03-02-PLAN.md
@.planning/phases/03-customers/03-03-PLAN.md

<interfaces>
<!-- Key types and contracts the executor needs. Extracted from codebase. -->

From internal/core/domain/contract/contract.go:
```go
type CreateContractRequest struct {
    Name            string              `json:"name"`
    KmRate          float64             `json:"km_rate"`
    Currency        string              `json:"currency"`
    GovernanceModel models.GovernanceModel `json:"governance_model"`
    IsShared        bool                `json:"is_shared"`
    // ADD: CustomerID *uuid.UUID       `json:"customer_id,omitempty"`
}

// ADD:
var ErrHasActiveProjects = errors.New("contract has active projects")
```

From internal/core/ports/contract_repository.go:
```go
type ContractRepository interface {
    // ... existing methods ...
    HasTimeEntries(ctx context.Context, contractID uuid.UUID) (int, error)
    // ADD: HasProjects(ctx context.Context, contractID uuid.UUID) (int, error)
}
```

From internal/adapters/primary/http/contract.go handler-level CreateContractRequest:
```go
type CreateContractRequest struct {
    Name            string                 `json:"name"`
    KmRate          float64                `json:"km_rate"`
    Currency        string                 `json:"currency"`
    GovernanceModel models.GovernanceModel `json:"governance_model"`
    IsShared        bool                   `json:"is_shared"`
    // ADD: CustomerID *string              `json:"customer_id,omitempty"`
}
```

From web/src/types/api.ts:
```typescript
export interface CreateContractRequest {
    name: string
    km_rate: number
    currency: string
    governance_model: 'creator_controlled' | 'unanimous' | 'majority'
    is_shared: boolean
    // ADD: customer_id?: string
}
```

From web/src/types/models.ts (Contract interface — already has customer_id? and customer_name?):
```typescript
export interface Contract {
    // ... existing fields ...
    customer_id?: string
    customer_name?: string
}
```

From web/src/api/customers.ts:
```typescript
export interface Customer {
    id: string
    // ...
    is_internal: boolean
}
export function customersQueryOpts(search?: string): QueryOptions<Customer[]>
```

From web/src/components/ui/combobox.tsx:
```typescript
// Exported components: Combobox, ComboboxInput, ComboboxContent, ComboboxList,
// ComboboxItem, ComboboxGroup, ComboboxLabel, ComboboxTrigger, ComboboxValue,
// ComboboxEmpty, ComboboxSeparator, useComboboxAnchor
```

From web/src/api/contracts.ts:
```typescript
const createContractMutationOpts = mutationOptions({
    mutationFn: (data: CreateContractRequest) =>
        api<Contract>('/contracts', {method: 'POST', body: JSON.stringify(data)}),
    // ...
})
```
</interfaces>
</context>

<tasks>

<!-- Wave 1: Backend domain + interface + implementation (parallel-safe, no file conflicts between sub-waves) -->

<task type="auto">
  <name>Task 1: Add customer_id to domain CreateContractRequest + ErrHasActiveProjects error</name>
  <files>
    internal/core/domain/contract/contract.go
  </files>
  <action>
    Per D-01 and D-07:
    1. Add `CustomerID *uuid.UUID \`json:"customer_id,omitempty"\`` to `CreateContractRequest` struct (append after `IsShared`)
    2. Add `var ErrHasActiveProjects = errors.New("contract has active projects")` alongside existing errors in the `var (...)` block (after `ErrHasTimeEntries`)

    The `UpdateContractRequest` struct already has `CustomerID *string` — do NOT modify it.
    The `Contract` struct already has `CustomerID *uuid.UUID` — do NOT modify it.
  </action>
  <verify>
    <automated>go vet ./internal/core/domain/contract/...</automated>
  </verify>
  <done>CreateContractRequest has CustomerID field, ErrHasActiveProjects is exported</done>
</task>

<task type="auto">
  <name>Task 2: Add HasProjects to ContractRepository interface + mock</name>
  <files>
    internal/core/ports/contract_repository.go
    internal/core/services/testdata/mocks.go
  </files>
  <action>
    Per D-06:
    1. In `internal/core/ports/contract_repository.go`, add `HasProjects(ctx context.Context, contractID uuid.UUID) (int, error)` method to the `ContractRepository` interface (append after `HasTimeEntries`)
    
    2. In `internal/core/services/testdata/mocks.go`, add a `HasProjects` method to `MockContractRepo`:
    ```go
    func (m *MockContractRepo) HasProjects(ctx context.Context, contractID uuid.UUID) (int, error) {
        return 0, nil
    }
    ```
    Match the existing mock pattern (no locking needed for this simple stub — consistent with mock's `Create` and `Adopt` patterns).
  </action>
  <verify>
    <automated>go vet ./internal/core/ports/... ./internal/core/services/testdata/...</automated>
  </verify>
  <done>ContractRepository interface includes HasProjects, mock compiles</done>
</task>

<task type="auto">
  <name>Task 3: Update PG repository — Create INSERT with customer_id + HasProjects implementation</name>
  <files>
    internal/adapters/secondary/postgres/contract_repository.go
  </files>
  <action>
    Per D-03 and D-09:
    
    1. **Update `Create` method INSERT:**
       - Change the INSERT column list from:
         `INSERT INTO contracts (id, name, km_rate, currency, governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)`
         To:
         `INSERT INTO contracts (id, name, km_rate, currency, customer_id, governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)`
       - Add the parameter placeholder `$5` for customer_id (shifting existing params by +1):
         `VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, NOW(), NOW())`
         becomes:
         `VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, NOW(), NOW())`
       - Actually re-count after insertion: the existing INSERT is:
         `VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW())`
         After adding customer_id at position 5:
         `VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, NOW(), NOW())`
       - Update args: add `req.CustomerID` at position 5 in the args list. If `req.CustomerID` is nil, the SQL parameter will be NULL (which is correct per D-01 nullable semantics).
       
       The args list was: `id, req.Name, req.KmRate, req.Currency, req.GovernanceModel, orgID, req.IsShared`
       It becomes: `id, req.Name, req.KmRate, req.Currency, req.CustomerID, req.GovernanceModel, orgID, req.IsShared`
       Verify that `*uuid.UUID` (pointer) can be passed as a pgx parameter (it can — pgx handles nil pointers as SQL NULL).

    2. **Add `HasProjects` method** (after existing `HasTimeEntries` method):
    ```go
    // HasProjects returns the count of projects linked to this contract.
    func (r *ContractRepository) HasProjects(ctx context.Context, contractID uuid.UUID) (int, error) {
        query := `SELECT COUNT(*) FROM projects WHERE contract_id = $1`
        var count int
        err := r.pool.QueryRow(ctx, query, contractID).Scan(&count)
        if err != nil {
            return 0, fmt.Errorf("has projects: %w", err)
        }
        return count, nil
    }
    ```
    Per D-09, this counts ALL projects (not just active ones), consistent with `ON DELETE RESTRICT` FK constraint.
  </action>
  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/...</automated>
  </verify>
  <done>Create INSERT includes customer_id column, HasProjects query exists and compiles</done>
</task>

<task type="auto">
  <name>Task 4: Update HTTP Create handler to parse customer_id + service Delete to check HasProjects</name>
  <files>
    internal/adapters/primary/http/contract.go
    internal/core/services/contract/contract.go
  </files>
  <action>
    Per D-02 and D-08:

    **HTTP handler** (`internal/adapters/primary/http/contract.go`):
    1. Add `CustomerID *string \`json:"customer_id,omitempty"\`` to the handler's `CreateContractRequest` struct (append to struct)
    2. In the `Create` handler method, when constructing the domain `CreateContractRequest`, parse `customer_id`:
    ```go
    var parsedCustomerID *uuid.UUID
    if req.CustomerID != nil && *req.CustomerID != "" {
        cid, err := uuid.Parse(*req.CustomerID)
        if err != nil {
            api.RespondWithError(w, http.StatusBadRequest, "invalid customer_id")
            return
        }
        parsedCustomerID = &cid
    }
    ```
    Then pass it:
    ```go
    contract, err := h.service.Create(r.Context(), orgID, &contractdomain.CreateContractRequest{
        Name:            req.Name,
        KmRate:          req.KmRate,
        Currency:        req.Currency,
        GovernanceModel: req.GovernanceModel,
        IsShared:        req.IsShared,
        CustomerID:      parsedCustomerID,
    })
    ```

    **Service layer** (`internal/core/services/contract/contract.go`):
    1. In the `Delete` method, after the existing `HasTimeEntries` check, add a `HasProjects` check:
    ```go
    projectCount, err := s.repo.HasProjects(ctx, contractID)
    if err != nil {
        return err
    }
    if projectCount > 0 {
        return contractdomain.ErrHasActiveProjects
    }
    ```
    
    2. Also update the HTTP `Delete` handler (`contract.go`) to return a specific 409 message for `ErrHasActiveProjects`:
    Add a case in the existing error switch:
    ```go
    case contractdomain.ErrHasActiveProjects:
        api.RespondWithError(w, http.StatusConflict, "contract has projects and cannot be deleted")
    ```
    Add this after the existing `case contractdomain.ErrHasTimeEntries:` block.
  </action>
  <verify>
    <automated>go vet ./internal/adapters/primary/http/... ./internal/core/services/contract/...</automated>
  </verify>
  <done>Create handler parses customer_id, service Delete checks HasProjects, HTTP returns specific 409</done>
</task>

<task type="auto">
  <name>Task 5: Update backend service unit tests for customer_id in Create + HasProjects in Delete</name>
  <files>
    internal/core/services/contract/contract_test.go
  </files>
  <action>
    Per D-18:
    
    1. **Update `TestService_Create`:**
       Add a new test case for creating a contract with a `customer_id` set:
       ```go
       {
           name: "valid contract with customer",
           req: &contractdomain.CreateContractRequest{
               Name:            "Customer Contract",
               GovernanceModel: models.GovernanceCreatorControlled,
               CustomerID:      &uuid.UUID{1},  // non-nil pointer
           },
           wantErr: nil,
       },
       ```
       Also add a case for creating with nil customer_id (already covered by existing "valid contract" case, so no extra test needed).

    2. **Update `TestService_Delete`:**
       The existing mock's `HasProjects` returns 0 (no projects), so existing tests still pass. Add a new test case for HasProjects blocking delete:
       ```go
       t.Run("blocked by projects", func(t *testing.T) {
           svc, repo := setupService(t)
           orgID := uuid.New()
           seeded := seedContract(repo, func(c *contractdomain.ContractResponse) { c.CreatedByOrgID = orgID })
           // Override mock to return 1 project
           repo.HasProjectsFn = func(ctx context.Context, contractID uuid.UUID) (int, error) {
               return 1, nil
           }
           err := svc.Delete(context.Background(), string(models.RoleFinance), orgID, seeded.ID)
           assert.ErrorIs(t, err, contractdomain.ErrHasActiveProjects)
       })
       ```
       
       For this to work, the `MockContractRepo` needs an overridable `HasProjectsFn` field. Update `MockContractRepo` struct in `mocks.go` to add:
       ```go
       type MockContractRepo struct {
           mu             sync.Mutex
           Contracts      map[uuid.UUID]*contractdomain.ContractResponse
           HasProjectsFn  func(ctx context.Context, contractID uuid.UUID) (int, error)
       }
       ```
       And update the `HasProjects` mock method to check `HasProjectsFn` first:
       ```go
       func (m *MockContractRepo) HasProjects(ctx context.Context, contractID uuid.UUID) (int, error) {
           if m.HasProjectsFn != nil {
               return m.HasProjectsFn(ctx, contractID)
           }
           return 0, nil
       }
       ```

    3. **Run tests:**
       ```bash
       go test -count=1 -run TestService_ ./internal/core/services/contract/...
       ```
  </action>
  <verify>
    <automated>go test -count=1 -timeout 60s -run TestService_ ./internal/core/services/contract/...</automated>
  </verify>
  <done>Create with customer_id passes, Delete blocked by HasProjects returns ErrHasActiveProjects</done>
</task>

<task type="auto">
  <name>Task 6: Update frontend CreateContractRequest type + API contracts</name>
  <files>
    web/src/types/api.ts
  </files>
  <action>
    Per D-04:
    Add `customer_id?: string` to the `CreateContractRequest` interface in `web/src/types/api.ts`:
    ```typescript
    export interface CreateContractRequest {
        name: string
        km_rate: number
        currency: string
        governance_model: 'creator_controlled' | 'unanimous' | 'majority'
        is_shared: boolean
        customer_id?: string
    }
    ```
    No changes needed to `web/src/api/contracts.ts` — the mutation already passes the full JSON body via `JSON.stringify(data)`, and `customer_id?: string` on the type definition will serialize correctly.
    No changes needed to `models.ts` — `Contract` already has `customer_id?: string` and `customer_name?`.
  </action>
  <verify>
    <automated>cd web && bun run build 2>&1 | head -20</automated>
  </verify>
  <done>TypeScript compiles with customer_id on CreateContractRequest</done>
</task>

<task type="auto">
  <name>Task 7: Add customer combobox to CreateContractDialog</name>
  <files>
    web/src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx
  </files>
  <action>
    Per D-05, D-10, D-11, D-12, D-13, D-14, D-15:

    **Imports to add:**
    ```typescript
    import {useSuspenseQuery} from '@tanstack/react-query'
    import {CustomersApis} from '@/api/customers'
    import {
        Combobox,
        ComboboxContent,
        ComboboxInput,
        ComboboxList,
        ComboboxItem,
        ComboboxValue,
        ComboboxTrigger,
        ComboboxEmpty,
        useComboboxAnchor,
    } from '@/components/ui/combobox'
    import type {Customer} from '@/api/customers'
    ```

    **State to add** alongside existing state declarations (after `const [isShared, setIsShared] = useState(false)`):
    ```typescript
    const [customerId, setCustomerId] = useState<string>('')
    const comboboxAnchor = useComboboxAnchor()
    ```

    **Data fetching** (inside the component, before `handleSubmit`):
    ```typescript
    const {data: customers} = useSuspenseQuery(CustomersApis.customersQueryOpts())
    ```

    **Update `handleSubmit`** to include customer_id in the mutation payload:
    ```typescript
    createContract.mutate(
        {
            name: name.trim(),
            km_rate: parseFloat(kmRate) || 0,
            currency,
            governance_model: governanceModel,
            is_shared: isShared,
            customer_id: customerId || undefined,
        },
        // ... existing onSuccess
    )
    ```

    **Update `resetForm`** to also reset customerId:
    ```typescript
    const resetForm = () => {
        setName('')
        setKmRate('0')
        setCurrency('EUR')
        setGovernanceModel('creator_controlled')
        setIsShared(false)
        setCustomerId('')
    }
    ```

    **Add the combobox field** in the form layout (after the currency/rate row, before governance model — per D-XX discretion but this follows the edit form order in `contract-detail.tsx`):
    ```tsx
    <div className="space-y-2" ref={comboboxAnchor}>
        <label className="text-sm font-medium">Customer</label>
        <Combobox
            value={customerId}
            onValueChange={(v) => setCustomerId(v ?? '')}
        >
            <ComboboxInput
                placeholder="Search customers..."
                showTrigger
                showClear={!!customerId}
            />
            <ComboboxContent>
                <ComboboxList>
                    <ComboboxItem value="">No customer</ComboboxItem>
                    {customers?.map((cust) => (
                        <ComboboxItem key={cust.id} value={cust.id}>
                            <div className="flex items-center gap-2">
                                <span>{cust.company_name}</span>
                                {cust.is_internal && (
                                    <span className="text-xs text-muted-foreground">(Internal)</span>
                                )}
                            </div>
                        </ComboboxItem>
                    ))}
                </ComboboxList>
            </ComboboxContent>
        </Combobox>
    </div>
    ```

    **Note on combobox usage**: The combobox from `@base-ui/react` requires `ComboboxValue` to display the selected value text, and the `onValueChange` callback receives the value string when an item is selected. The `useComboboxAnchor` ref needs to be attached to the container div for proper positioning. Use `customerId` state to track the selected value — empty string `""` means "No customer."

    **Styling per D-14/D-15/Discretion**: Internal customers show "(Internal)" in muted text (`text-muted-foreground text-xs`) next to the company name. The "No customer" option appears as the first list item. Use discretion for exact styling — muted text suffix is the recommended approach (lighter than a badge which would add visual noise in a list).
  </action>
  <verify>
    <automated>cd web && bun run build 2>&1 | head -20</automated>
  </verify>
  <done>Customer combobox renders in create dialog, "No customer" is first option, internal customers show "(Internal)" suffix</done>
</task>

<task type="auto">
  <name>Task 8: Update frontend tests for customer_id in create flow</name>
  <files>
    web/src/api/__tests__/contracts.test.ts
  </files>
  <action>
    Per D-17:

    1. **Update existing `createContractMutationOpts` test** to include `customer_id` in the contract data and captured body assertion:
    ```typescript
    it('createContractMutationOpts sends POST /api/contracts with customer_id', async () => {
        const contractData = {
            name: 'New Contract',
            km_rate: 0.42,
            currency: 'USD',
            governance_model: 'majority' as const,
            is_shared: true,
            customer_id: 'cust-1',
        }
        const mockContract = {
            id: 'c2', name: 'New Contract', km_rate: 0.42, currency: 'USD',
            governance_model: 'majority' as const,
            is_shared: true, is_active: true, created_by_org_id: 'o1',
            customer_id: 'cust-1',
            created_at: '2025-01-01T00:00:00Z',
        }
    
        let capturedBody: unknown = null
        server.use(
            http.post('/api/contracts', async ({ request }) => {
                capturedBody = await request.json()
                return HttpResponse.json({ data: mockContract })
            }),
        )
    
        const result = await ContractsApis.createContractMutationOpts.mutationFn(contractData)
        expect(capturedBody).toEqual(contractData)
        expect(result).toEqual(mockContract)
    })
    ```
    Replace the existing "createContractMutationOpts sends POST /api/contracts" test entirely with this version. The key changes are:
    - `customer_id: 'cust-1'` in contract data
    - `customer_id: 'cust-1'` in mock response
    - Test name updated to signal it tests with customer_id

    2. Add a test for creating a contract **without a customer_id** (undefined):
    ```typescript
    it('createContractMutationOpts without customer_id omits the field', async () => {
        const contractData = {
            name: 'No Customer Contract',
            km_rate: 0.50,
            currency: 'EUR',
            governance_model: 'creator_controlled' as const,
            is_shared: false,
        }
        const mockContract = {
            id: 'c3', name: 'No Customer Contract', km_rate: 0.50, currency: 'EUR',
            governance_model: 'creator_controlled' as const,
            is_shared: false, is_active: true, created_by_org_id: 'o1',
            created_at: '2025-01-01T00:00:00Z',
        }
    
        let capturedBody: unknown = null
        server.use(
            http.post('/api/contracts', async ({ request }) => {
                capturedBody = await request.json()
                return HttpResponse.json({ data: mockContract })
            }),
        )
    
        const result = await ContractsApis.createContractMutationOpts.mutationFn(contractData)
        expect(capturedBody).toEqual(contractData)
        // Verify customer_id is NOT in the request body
        expect((capturedBody as Record<string, unknown>).customer_id).toBeUndefined()
        expect(result).toEqual(mockContract)
    })
    ```

    Both tests should be added/updated in the `describe('ContractsApis', () => { ... })` block.
  </action>
  <verify>
    <automated>cd web && bunx vitest run src/api/__tests__/contracts.test.ts 2>&1 | tail -20</automated>
  </verify>
  <done>Frontend tests pass with customer_id in create requests and undefined-customer case</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| HTTP client → API POST /contracts | Untrusted customer_id input crosses here |
| HTTP client → API DELETE /contracts | Delete attempt crosses auth boundary |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-04-01 | Spoofing | POST /contracts customer_id | mitigate | Validate customer_id is valid UUID in HTTP handler before passing to service (returns 400 on parse failure, per existing pattern) |
| T-04-02 | Repudiation | DELETE /contracts | mitigate | Service logs ErrHasActiveProjects / ErrHasTimeEntries before returning — the error message distinguishes which guard blocked deletion |
| T-04-03 | Information Disclosure | Customer combobox | accept | All org customers visible in combobox — this is intentional UX for the creating user within their org; no cross-org leakage since customersQueryOpts is org-scoped |

</threat_model>

<verification>

### Backend Verification

```bash
# Compile check
go build ./...

# Vet check
go vet ./internal/...

# Service unit tests (excludes integration tests for fast feedback)
go test -count=1 -timeout 60s -run TestService_ ./internal/core/services/contract/...

# Full backend test suite
go test -count=1 -timeout 300s ./internal/...
```

### Frontend Verification

```bash
# Type-check + build
cd web && bun run build

# Run frontend unit tests
cd web && bunx vitest run src/api/__tests__/contracts.test.ts
```

</verification>

<success_criteria>

- [ ] `go build ./...` compiles without errors
- [ ] `go vet ./...` passes
- [ ] `go test -count=1 -run TestService_ ./internal/core/services/contract/...` — Delete tests include "blocked by projects" case, Create tests include "valid contract with customer" case
- [ ] `cd web && bun run build` — TypeScript compiles without errors
- [ ] `cd web && bunx vitest run src/api/__tests__/contracts.test.ts` — all tests pass including new customer_id tests
- [ ] Create contract with customer via API: POST body has `customer_id`, response includes `customer_id`, DB row has the correct value
- [ ] Create contract with "No customer": POST body omits `customer_id` (or sends `""`), DB row has NULL `customer_id`
- [ ] Delete contract with projects: returns 409 "contract has projects and cannot be deleted"
- [ ] Delete contract with time entries: returns 409 "contract has time entries and cannot be deleted" (unchanged)
- [ ] Create dialog shows "No customer" as first combobox item
- [ ] Create dialog shows internal customers with "(Internal)" suffix

</success_criteria>

<output>
After completion, create `.planning/phases/04-contracts/04-01-SUMMARY.md`
</output>
