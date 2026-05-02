# Expenses

Complete reference for expense tracking and approval workflows.

---

## Overview

**Expenses** let employees report business costs (mileage, meals, hotels, etc.).

**Features:**
- Multiple expense categories
- Mileage tracking with automatic cost calculation
- Draft/submission approval workflow
- Multi-level approvals (manager → finance)
- Immutable approval history
- Receipt attachment support (future)

---

## Data Model

### expenses (Header)

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Expense identifier |
| organization_id | UUID FK | Owner org |
| user_id | UUID FK | Employee who logged expense |
| status | VARCHAR CHECK | draft, submitted, pending_manager, pending_finance, approved, rejected |
| current_approver_role | VARCHAR | Next reviewer (manager or finance) |
| category | VARCHAR CHECK | mileage, meal, accommodation, other |
| amount | DECIMAL | Total expense amount |
| currency | VARCHAR | e.g., "USD", "EUR" |
| description | TEXT | What the expense was for |
| expense_date | DATE | When expense occurred |
| submitted_at | TIMESTAMP | When submitted (nullable) |
| created_at | TIMESTAMP | Creation time |
| updated_at | TIMESTAMP | Last modification |

**Indexes:** `organization_id`, `user_id`, `status`, `category`

### expense_mileage_details

Additional data for mileage expenses:

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Record identifier |
| expense_id | UUID FK | References expenses(id) |
| contract_id | UUID FK | Contract for km rate |
| distance_km | DECIMAL | Distance traveled |
| rate_per_km | DECIMAL | Rate applied (from contract or org default) |

**Formula:** `amount = distance_km * rate_per_km`

### expense_approvals

Immutable approval history:

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Approval record ID |
| expense_id | UUID FK | References expenses(id) |
| approver_id | UUID FK | User who approved |
| approver_role | VARCHAR | Role of approver (manager, finance) |
| action | VARCHAR | submit, approve, reject, edit_approve, edit_return, partial_approve, delegate |
| reason | TEXT | Optional explanation (nullable) |
| created_at | TIMESTAMP | Action timestamp |

---

## Approval Workflow

Same as Time Entries but for expenses.

### Status Flow

```
Employee creates expense
    ↓
Status: draft (only employee can edit)
    ↓
Employee submits
    ↓
Status: submitted → pending_manager
current_approver_role: manager
    ↓
Manager reviews
    ├─ Approves → Status: pending_finance
    │   current_approver_role: finance
    │
    ├─ Rejects → Status: rejected
    │   Employee can edit and resubmit
    │
    └─ Edit & Approve → Manager edits + approves
        Status: pending_finance
    ↓
Finance reviews
    ├─ Approves → Status: approved (final)
    │
    ├─ Rejects → Status: rejected
    │
    └─ Edit & Approve → Finance edits + approves
        Status: approved (final)
```

---

## Expense Categories

### mileage

**For:** Vehicle travel (car, motorcycle, bicycle)

**Input:**
- Distance in kilometers
- Contract (for km rate)
- Org default km rate (if no contract)

**Calculation:**
```
amount = distance_km * rate_per_km
```

**Example:**
- Contract: 0.50 EUR per km
- Distance: 100 km
- Amount: 50.00 EUR

### meal

**For:** Food and beverages during business

**Input:**
- Amount (user enters)
- Description (lunch, team dinner, etc.)

**Limits:** Some orgs set daily/meal limits (not enforced in current version)

### accommodation

**For:** Hotel, lodging during travel

**Input:**
- Amount
- Dates (implicit in expense_date)

### other

**For:** Everything else (parking, tolls, supplies, etc.)

**Input:**
- Amount
- Description (required to clarify)

---

## API Endpoints

### POST /expenses (Create)

**Request (Mileage):**
```json
{
  "category": "mileage",
  "contract_id": "contract-uuid",
  "distance_km": 120.5,
  "currency": "EUR",
  "description": "Client visit in Berlin",
  "expense_date": "2024-04-01"
}
```

**Request (Meal/Other):**
```json
{
  "category": "meal",
  "amount": 45.50,
  "currency": "EUR",
  "description": "Team lunch meeting",
  "expense_date": "2024-04-01"
}
```

**Flow:**
1. Validate category and inputs
2. For mileage: fetch contract km_rate, calculate amount
3. Create expenses record with status=draft
4. If mileage: create expense_mileage_details record
5. Return expense

**Response (201):**
```json
{
  "data": {
    "id": "expense-uuid",
    "category": "mileage",
    "amount": 60.25,
    "currency": "EUR",
    "status": "draft",
    "description": "Client visit in Berlin",
    "expense_date": "2024-04-01",
    "mileage_details": {
      "distance_km": 120.5,
      "rate_per_km": 0.50
    },
    "created_at": "2024-04-01T10:00:00Z"
  }
}
```

---

### GET /expenses (List)

**Query Parameters:**
```
?status=draft           # Filter by status
?category=mileage      # Filter by category
?start_date=2024-04-01 # Date range
?end_date=2024-04-30
?limit=50              # Pagination
?offset=0
```

**Flow:**
1. Apply same visibility rules as time entries
2. Filter by status, category, date range
3. Return expenses with mileage details if applicable

**Response (200):**
```json
{
  "data": [
    {
      "id": "expense-uuid",
      "user_id": "user-uuid",
      "category": "mileage",
      "amount": 60.25,
      "status": "pending_manager",
      "current_approver_role": "manager",
      "description": "Client visit",
      "expense_date": "2024-04-01",
      "mileage_details": {
        "distance_km": 120.5,
        "rate_per_km": 0.50,
        "contract_id": "contract-uuid"
      },
      "created_at": "2024-04-01T10:00:00Z"
    }
  ]
}
```

---

### GET /expenses/{id} (Get Single)

Returns expense with full details including approval history.

---

### PUT /expenses/{id} (Update)

**Allowed if:**
- User is owner and status is `draft`
- User is owner and status is `rejected`

**Request:** Same fields as create

**Flow:**
1. Check ownership and status
2. If mileage: delete old details, create new
3. Update expense record
4. Return updated expense

---

### POST /expenses/{id}/submit (Submit)

**Request:**
```json
{
  "notes": "Ready for review"  // Optional
}
```

**Flow:**
1. Check status is draft
2. Set status to submitted
3. Set current_approver_role to manager
4. Create approval record (action: submit)
5. Set submitted_at timestamp

---

### POST /expenses/{id}/approve (Approve)

**Required role:** manager or finance

**Request:**
```json
{
  "reason": "Approved"  // Optional
}
```

**Flow:**
1. Check current_approver_role matches user role
2. Advance status (pending_manager → pending_finance, or final approve)
3. Create approval record (action: approve)

---

### POST /expenses/{id}/reject (Reject)

**Required role:** manager or finance

**Request:**
```json
{
  "reason": "Missing receipt"
}
```

**Flow:**
1. Check user can reject at current stage
2. Set status to rejected
3. Create approval record (action: reject)

---

### POST /expenses/{id}/edit-approve (Edit & Approve)

Allows approvers to adjust amounts while approving.

**Request:**
```json
{
  "amount": 55.00,      // Corrected amount
  "reason": "Corrected amount, approved"
}
```

**Flow:**
1. Check current_approver_role
2. Update amount (and mileage details if applicable)
3. Advance status
4. Create approval record (action: edit_approve)

---

### GET /expenses/{id}/approvals (History)

Get immutable approval history.

---

## Frontend Components

### Expense Form

See `web/src/components/expense-form.tsx`

**Features:**
- Category selector (radio or dropdown)
- Conditional fields based on category:
  - **Mileage:** Distance input + contract selector
  - **Meal/Other:** Amount input + description
- Date picker (defaults to today)
- Currency selector (defaults to org currency)
- Draft save button
- Submit button

### Expense List

See `web/src/components/expenses-list.tsx`

**Features:**
- Table with category icons
- Amount display with currency
- Status badges
- Quick approvals (for managers/finance)
- Inline view/edit actions

---

## Business Rules

### Validation

1. **Category:** Must be one of valid categories
2. **Amount:** Must be > 0
3. **Mileage Distance:** Must be > 0 km
4. **Contract:** Must exist if mileage category
5. **Expense Date:** Should be in past (allow today)
6. **Description:** Required for "other" category

### Rate Calculation

**For Mileage:**
1. Fetch contract by contract_id
2. Use contract.km_rate if available
3. Fallback to organization_settings.default_km_rate
4. If neither: error (must have a rate)

**Recalculation:**
If contract rate changes, API supports:
```
POST /contracts/{id}/recalculate-mileage
```
This updates all approved expenses linked to that contract (respecting permissions).

### Permissions

| Action | Who Can Do It |
|--------|--------------|
| Create | Any employee |
| Edit (draft) | Entry owner |
| Edit (rejected) | Entry owner |
| Submit | Entry owner |
| Approve (manager) | Users with manager role |
| Approve (finance) | Users with finance role |
| Reject | Same as approvers |
| View | Owner + approvers + finance |
| Edit-Approve | Managers/finance (corrects before approval) |

---

## Common Workflows

### Mileage Submission

```
1. Employee enters distance and contract
2. System calculates amount automatically
3. Employee submits
4. Manager approves
5. Finance reviews and approves
6. Amount is final and immutable
```

### Meal Submission

```
1. Employee enters amount and description
2. Employee submits
3. Manager reviews (checks if reasonable)
4. Manager approves
5. Finance approves
```

### Correction During Review

```
1. Manager sees expense is slightly off
2. Uses edit-approve to correct amount
3. Entry moves to finance for final approval
4. Finance sees manager's correction in history
5. Finance approves
```

### Batch Mileage Update

When contract rate changes:
```
POST /contracts/{id}/recalculate-mileage

Endpoint updates all approved expenses:
- Finds expenses with mileage_details for this contract
- Recalculates amount based on new rate
- Creates audit record showing old → new amount
- Updates related financial reports
```

---

## CSV Export

See [[14-CSV-Exports]] for generating expense reports.

---

## Testing

See [[17-Testing]] for test patterns.

Quick test:
```go
func TestMileageExpense(t *testing.T) {
    // 1. Create expense with mileage category
    // 2. Verify amount calculated correctly
    // 3. Submit
    // 4. Manager approves
    // 5. Finance approves
    // 6. Verify final status and amount immutable
}
```

---

**Next**: [[12-Contracts-Projects]] for contract management, or [[14-CSV-Exports]] for reporting.
