# Contracts & Projects

Complete guide to contract and project management, including shared resources.

---

## Overview

**Contracts** are billable agreements with customers.
**Projects** are work categories within contracts (internal or billable).
**Shared Resources** allow organizations to adopt contracts/projects created by others.

---

## Data Model

### contracts

| Column | Type | Purpose |\n|--------|------|---------|
| id | UUID PK | Contract identifier |
| name | VARCHAR | Contract name |
| km_rate | DECIMAL | Mileage reimbursement rate |
| currency | VARCHAR | e.g., \"USD\", \"EUR\" |
| customer_id | UUID FK | References customers(id) (nullable) |
| governance_model | VARCHAR CHECK | creator_controlled, unanimous, majority |
| created_by_org_id | UUID FK | Organization that created it |
| is_shared | BOOLEAN | Available to other orgs |
| is_active | BOOLEAN | Soft delete flag |
| created_at | TIMESTAMP | Creation time |

**Indexes:** `created_by_org_id`, `customer_id`

### projects

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Project identifier |
| name | VARCHAR | Project name |
| type | VARCHAR CHECK | billable, internal |
| contract_id | UUID FK | References contracts(id) |
| governance_model | VARCHAR | creator_controlled, unanimous, majority |
| created_by_org_id | UUID FK | Organization that created it |
| is_shared | BOOLEAN | Available to other orgs |
| is_active | BOOLEAN | Soft delete flag |
| created_at | TIMESTAMP | Creation time |

**Indexes:** `contract_id`, `created_by_org_id`

### contract_adoptions

Tracks which orgs have adopted a contract:

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Adoption record |
| contract_id | UUID FK | References contracts(id) |
| organization_id | UUID FK | Org that adopted it |
| adopted_at | TIMESTAMP | When adopted |

**Constraint:** UNIQUE(contract_id, organization_id)

### project_adoptions

Same pattern for projects:

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Adoption record |
| project_id | UUID FK | References projects(id) |
| organization_id | UUID FK | Org that adopted it |
| adopted_at | TIMESTAMP | When adopted |

**Constraint:** UNIQUE(project_id, organization_id)

### project_managers

Assigns specific managers to projects for approval routing:

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Assignment record |
| project_id | UUID FK | References projects(id) |
| user_id | UUID FK | References users(id) |
| assigned_at | TIMESTAMP | When assigned |

**Constraint:** UNIQUE(project_id, user_id)

---

## Governance Models

Used for approval workflows on contracts/projects:

### creator_controlled

Only the creator's manager approves expenses/time on this contract.

**Example:** Org A creates Contract A
- Only Org A's manager approves time/expenses on Contract A
- If Org B adopts Contract A, Org A's manager still approves (or Org B's manager?)
- Requires clarification in implementation

### unanimous

All assigned project managers must approve.

**Example:** 3 managers assigned to Project X
- All 3 must approve before entry is approved
- If any reject, entry is rejected
- Useful for multi-team projects

### majority

Majority of assigned project managers must approve.

**Example:** 5 managers assigned to Project X
- Need 3 of 5 approvals to proceed
- Quorum-based approval

---

## API Endpoints

### Contracts

#### POST /contracts (Create)

**Request:**
```json
{
  "name": \"Enterprise Support Contract\",
  \"km_rate\": 0.50,
  \"currency\": \"EUR\",
  \"customer_id\": \"customer-uuid\",
  \"governance_model\": \"creator_controlled\",
  \"is_shared\": true
}
```

**Flow:**
1. Validate org membership
2. Create contract in organization context
3. Return contract with ID

**Response (201):**
```json
{
  \"data\": {
    \"id\": \"contract-uuid\",
    \"name\": \"Enterprise Support Contract\",
    \"km_rate\": 0.50,
    \"currency\": \"EUR\",
    \"customer_id\": \"customer-uuid\",
    \"governance_model\": \"creator_controlled\",
    \"created_by_org_id\": \"org-uuid\",
    \"is_shared\": true,
    \"is_active\": true,
    \"created_at\": \"2024-04-01T10:00:00Z\"
  }
}
```

#### GET /contracts (List)

**Query:**
```
?created_by_org_id=org-uuid  # Filter
?is_shared=true              # Show only shared
```

**Returns:** Contracts created by org + adopted contracts

#### GET /contracts/{id} (Get)

#### PUT /contracts/{id} (Update)

**Only owner org can update**

#### POST /contracts/{id}/adopt (Adopt)

**Request:**
```json
{
  // No request body needed
}
```

**Flow:**
1. Check contract is_shared = true
2. Verify user org hasn't already adopted
3. Create contract_adoptions record
4. Return adoption record

**Response (201):**
```json
{
  \"data\": {
    \"id\": \"adoption-uuid\",
    \"contract_id\": \"contract-uuid\",
    \"organization_id\": \"adopting-org-uuid\",
    \"adopted_at\": \"2024-04-01T10:00:00Z\"
  }
}
```

#### POST /contracts/{id}/recalculate-mileage

When km_rate changes, recalculate all approved mileage expenses:

**Request:**
```json
{
  // No body
}
```

**Flow:**
1. Validate contract ownership
2. Find all expense_mileage_details linked to this contract
3. For each approved expense:
   - Calculate new amount = distance_km * new_km_rate
   - Create audit record (old amount → new amount)
   - Update expenses.amount
4. Return summary

**Response (200):**
```json
{
  \"data\": {
    \"contract_id\": \"contract-uuid\",
    \"recalculated_count\": 15,
    \"total_change\": -25.50,
    \"updated_expenses\": [
      {
        \"expense_id\": \"uuid\",
        \"old_amount\": 60.25,
        \"new_amount\": 55.00
      }
    ]
  }
}
```

---

### Projects

#### POST /projects (Create)

**Request:**
```json
{
  \"name\": \"Frontend Development\",
  \"type\": \"billable\",
  \"contract_id\": \"contract-uuid\",
  \"governance_model\": \"creator_controlled\",
  \"is_shared\": true
}
```

**Flow:**
1. Validate contract exists and user org has access
2. Create project
3. Return project

#### GET /projects (List)

Returns projects in user's org + adopted projects

#### GET /projects/{id} (Get)

#### PUT /projects/{id} (Update)

Only owner org

#### POST /projects/{id}/adopt (Adopt)

Same pattern as contracts

---

### Project Managers

#### POST /projects/{id}/managers (Assign Manager)

**Request:**
```json
{
  \"user_id\": \"user-uuid\"
}
```

**Flow:**
1. Validate user is in same org
2. Create project_managers record
3. Return assignment

**Response (201):**
```json
{
  \"data\": {
    \"id\": \"assignment-uuid\",
    \"project_id\": \"project-uuid\",
    \"user_id\": \"user-uuid\",
    \"assigned_at\": \"2024-04-01T10:00:00Z\"
  }
}
```

#### DELETE /projects/{id}/managers/{user_id} (Remove Manager)

Removes manager from project approval routing.

---

## Business Rules

### Contract Ownership

- Only creating org can modify contract details
- Other orgs can only adopt (read-only)
- Km_rate change affects all expenses linked to contract

### Project Ownership

- Only creating org can modify project details
- Other orgs can adopt and use for time/expenses

### Shared vs Private

- `is_shared = false`: Only creator org can use
- `is_shared = true`: Any org can adopt and use

### Adoption

- Organization can adopt shared contracts/projects
- Creates contract_adoptions or project_adoptions record
- Both creator and adopter can now use it
- Adoption is permanent (no un-adopt)

### Active Status

- `is_active = false`: Soft delete, hidden in lists
- Existing time entries/expenses can still reference inactive projects
- Prevents new entries on inactive projects

---

## Frontend Components

### Contract Form

- Contract name input
- Customer selector (dropdown, linked to org)
- Km rate input (with currency)
- Governance model selector
- Share checkbox
- Save button

### Contract List

- Table with name, customer, km_rate, governance
- Edit/delete actions (owner only)
- Adopt button (for shared contracts from other orgs)
- View projects link

### Project Form

- Project name input
- Contract selector (dropdown)
- Type selector (billable/internal)
- Governance model selector
- Share checkbox
- Manager list (add/remove)

### Project List

- Table with name, contract, type, manager count
- Filter by contract
- Edit/delete (owner only)
- Adopt button (for shared)

---

## Workflows

### Create Contract & Projects

```
1. Org creates contract (e.g., \"Client XYZ\")
   - Set km_rate, customer, governance
   - Mark is_shared=true if want to reuse

2. Add projects to contract
   - \"Frontend Development\" (billable)
   - \"Support\" (billable)
   - \"Internal\" (internal)

3. Assign project managers
   - Manager A → Frontend
   - Manager B → Frontend & Support
   - Finance reviews both

4. Share with other orgs
   - POST /contracts/{id}/adopt (by other org)
   - Now both orgs can use contract/projects

5. Log time/expenses against projects
   - Employee picks project
   - Manager/Finance approve
```

### Update Mileage Rate

```
1. Org updates contract km_rate
2. POST /contracts/{id}/recalculate-mileage
3. All approved mileage expenses recalculated
4. Finance team reviews changes
5. Audit trail shows old → new amounts
```

### Governance Scenarios

**creator_controlled:**
```
Time entry against contract
  ↓
Only creator org's manager approves
  ↓
Finance approves
```

**unanimous:**
```
Time entry against project with 3 assigned managers
  ↓
All 3 managers must approve (any can reject)
  ↓
Finance approves
```

**majority:**
```
Time entry against project with 5 assigned managers
  ↓
Need 3+ managers to approve
  ↓
Finance approves
```

---

## Type System

```typescript
// types/api.ts
export interface Contract {
  id: string
  name: string
  km_rate: number
  currency: string
  customer_id?: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  created_by_org_id: string
  is_shared: boolean
  is_active: boolean
  created_at: string
}

export interface Project {
  id: string
  name: string
  type: 'billable' | 'internal'
  contract_id: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  created_by_org_id: string
  is_shared: boolean
  is_active: boolean
  created_at: string
}
```

---

**Next**: [[13-Organization-Users]] for org and user management.
