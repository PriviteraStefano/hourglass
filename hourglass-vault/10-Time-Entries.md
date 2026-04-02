# Time Entries

Complete reference for time entry tracking and approval workflows.

---

## Overview

**Time Entries** let employees log hours worked on projects.

**Features:**
- Per-project hour tracking
- Draft/submission workflow
- Multi-level approvals (manager → finance)
- Immutable approval history
- CSV export support

---

## Data Model

### time_entries (Header)

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Entry identifier |
| organization_id | UUID FK | Owner org |
| user_id | UUID FK | Employee who logged entry |
| status | VARCHAR CHECK | draft, submitted, pending_manager, pending_finance, approved, rejected |
| current_approver_role | VARCHAR | Next reviewer (manager or finance) |
| work_date | DATE | Date of work (flattened in Phase 2) |
| submitted_at | TIMESTAMP | When submitted (nullable) |
| notes | TEXT | Entry-level notes (nullable) |
| created_at | TIMESTAMP | Creation time |
| updated_at | TIMESTAMP | Last modification |

**Indexes:** `organization_id`, `user_id`, `status`

### time_entry_items (Line Items)

Each time entry contains multiple items (one per project):

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Item identifier |
| time_entry_id | UUID FK | References time_entries(id) |
| project_id | UUID FK | Project worked on |
| hours | DECIMAL | Hours logged |
| notes | TEXT | Item-level notes (nullable) |

**Indexes:** `time_entry_id`

### time_entry_approvals (History)

Immutable approval records:

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Approval record ID |
| time_entry_id | UUID FK | References time_entries(id) |
| approver_id | UUID FK | User who approved |
| approver_role | VARCHAR | Role of approver (manager, finance) |
| action | VARCHAR | submit, approve, reject, edit_approve, edit_return, partial_approve, delegate |
| reason | TEXT | Optional explanation (nullable) |
| created_at | TIMESTAMP | Action timestamp |

---

## Approval Workflow

### Status Flow

```
Employee creates entry
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
    │   Employee can edit and resubmit
    │
    └─ Edit & Approve → Finance edits + approves
        Status: approved (final)
```

### Status Details

| Status | Editable | Visible To | Notes |
|--------|----------|-----------|-------|
| draft | Yes (employee) | Employee only | Not submitted |
| submitted | No | Employee + all roles | Waiting for manager |
| pending_manager | No | Employee + manager + finance | Manager reviewing |
| pending_finance | No | Employee + finance | Finance reviewing |
| approved | No | All | Complete |
| rejected | Yes (employee) | Employee + rejector | Can edit and resubmit |

---

## API Endpoints

### POST /time-entries (Create)

**Request:**
```json
{
  "work_date": "2024-04-01",
  "notes": "Weekly sprint work",
  "items": [
    {
      "project_id": "proj-uuid",
      "hours": 8,
      "notes": "Frontend work"
    }
  ]
}
```

**Flow:**
1. Validate projects exist and user can access
2. Create time_entries record with status=draft
3. Create time_entry_items for each project
4. Return entry with items

**Response (201):**
```json
{
  "data": {
    "id": "entry-uuid",
    "status": "draft",
    "work_date": "2024-04-01",
    "items": [
      {
        "id": "item-uuid",
        "project_id": "proj-uuid",
        "hours": 8,
        "notes": "Frontend work"
      }
    ],
    "created_at": "2024-04-01T10:00:00Z"
  }
}
```

---

### GET /time-entries (List)

**Query Parameters:**
```
?status=draft           # Filter by status
?start_date=2024-04-01 # Filter by date range
?end_date=2024-04-30
?limit=50              # Pagination
?offset=0
```

**Flow:**
1. Validate user organization access
2. If user is employee: return only own entries
3. If user is manager: return all entries in org
4. If user is finance: return all entries in org
5. Apply filters (status, date range)
6. Return entries with items

**Response (200):**
```json
{
  "data": [
    {
      "id": "entry-uuid",
      "user_id": "user-uuid",
      "status": "pending_manager",
      "work_date": "2024-04-01",
      "current_approver_role": "manager",
      "items": [
        {
          "id": "item-uuid",
          "project_id": "proj-uuid",
          "hours": 8,
          "notes": "Frontend work"
        }
      ],
      "created_at": "2024-04-01T10:00:00Z"
    }
  ]
}
```

---

### PUT /time-entries/{id} (Update)

**Only allowed if:**
- User is owner and status is `draft`
- User is owner and status is `rejected` (resubmit)

**Request:**
```json
{
  "work_date": "2024-04-01",
  "notes": "Updated notes",
  "items": [
    {
      "project_id": "proj-uuid",
      "hours": 7,
      "notes": "Less hours"
    }
  ]
}
```

**Flow:**
1. Check authorization (owner or admin)
2. Check status allows editing (draft or rejected)
3. Delete old items, create new ones
4. Update entry metadata
5. Return updated entry

**Response (200):** Same as create

---

### POST /time-entries/{id}/submit (Submit)

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
6. Return updated entry

**Response (200):**
```json
{
  "data": {
    "id": "entry-uuid",
    "status": "submitted",
    "current_approver_role": "manager",
    "submitted_at": "2024-04-01T14:30:00Z"
  }
}
```

---

### POST /time-entries/{id}/approve (Approve)

**Required role:** manager or finance (depending on current_approver_role)

**Request:**
```json
{
  "reason": "Looks good"  // Optional
}
```

**Flow:**
1. Check current_approver_role matches user's role
2. If approver is manager:
   - Set status to pending_finance
   - Set current_approver_role to finance
3. If approver is finance:
   - Set status to approved
   - Clear current_approver_role
4. Create approval record (action: approve)
5. Return updated entry

**Response (200):**
```json
{
  "data": {
    "id": "entry-uuid",
    "status": "pending_finance",  // or "approved"
    "current_approver_role": "finance"  // or null
  }
}
```

---

### POST /time-entries/{id}/reject (Reject)

**Required role:** manager or finance

**Request:**
```json
{
  "reason": "Missing project details"  // Optional but recommended
}
```

**Flow:**
1. Check user role can reject at current stage
2. Set status to rejected
3. Clear current_approver_role
4. Create approval record (action: reject)
5. Return updated entry

**Response (200):**
```json
{
  "data": {
    "id": "entry-uuid",
    "status": "rejected",
    "current_approver_role": null
  }
}
```

---

### POST /time-entries/{id}/edit-approve (Edit & Approve)

**Required role:** manager or finance

Allows approvers to fix entries while approving (common workflow).

**Request:**
```json
{
  "items": [
    {
      "id": "item-uuid",  // Existing item to modify
      "hours": 7.5,       // Changed hours
      "notes": "Adjusted"
    }
  ],
  "reason": "Fixed hours, approving"
}
```

**Flow:**
1. Check current_approver_role matches user
2. Update items with new values
3. Advance status (like Approve endpoint)
4. Create approval record (action: edit_approve)
5. Return updated entry

**Response (200):** Same as approve

---

### GET /time-entries/{id}/approvals (History)

Get immutable approval history.

**Response (200):**
```json
{
  "data": [
    {
      "id": "approval-uuid",
      "time_entry_id": "entry-uuid",
      "approver_id": "user-uuid",
      "approver_role": "manager",
      "action": "submit",
      "reason": null,
      "created_at": "2024-04-01T14:30:00Z"
    },
    {
      "id": "approval-uuid-2",
      "approver_id": "manager-uuid",
      "approver_role": "manager",
      "action": "approve",
      "reason": "Looks good",
      "created_at": "2024-04-02T09:15:00Z"
    }
  ]
}
```

---

## Frontend Components

### Time Entry Form

See `web/src/components/time-entry-form.tsx`

**Features:**
- Date picker for work_date
- Dynamic item list (add/remove projects)
- Hour validation (0-24 per item)
- Total hours display
- Draft save button
- Submit button (only when valid)

### Time Entry List

See `web/src/components/time-entries-list.tsx`

**Features:**
- Table view with status badges
- Filter by status, date range
- Inline actions (view, edit, submit)
- Role-based action buttons
- Pagination

### Approval Actions

**Manager View:**
- Approve button
- Reject button
- Edit & Approve button
- View approval history

**Finance View:**
- Same as manager (different approval level)

---

## Business Rules

### Validation

1. **Work Date:** Must be in past or today
2. **Hours:** Each item 0 < hours ≤ 24
3. **Projects:** All must exist and be active
4. **Organization:** All projects must be in user's org

### Permissions

| Action | Who Can Do It |
|--------|--------------|
| Create | Any employee |
| Edit (draft) | Entry owner |
| Edit (rejected) | Entry owner |
| Edit (other) | No one (use edit-approve) |
| Submit | Entry owner |
| Approve (manager) | Users with manager role |
| Approve (finance) | Users with finance role |
| Reject | Same as approvers |
| View | Owner + their approvers + finance |

### Governance (Future)

Contracts have governance models that affect approval:
- `creator_controlled` — Creator's manager approves
- `unanimous` — All project managers must approve
- `majority` — Majority of project managers approve

---

## CSV Export

See [[14-CSV-Exports]] for report generation.

---

## Common Workflows

### Happy Path: Create → Submit → Approve

```
1. Employee creates entry (status: draft)
2. Employee submits (status: submitted → pending_manager)
3. Manager reviews (status: pending_finance)
4. Finance approves (status: approved)
```

### Rejection Loop

```
1. Employee creates and submits
2. Manager reviews, rejects (status: rejected)
3. Employee edits (status: draft)
4. Employee resubmits (status: submitted → pending_manager)
5. Manager re-reviews, approves
```

### Manager Fixes

```
1. Employee creates, submits
2. Manager sees issue, uses edit-approve
3. Entry approved with corrections
4. Employee sees corrected version in history
```

---

## Testing

See [[17-Testing]] for comprehensive test patterns.

Quick test:
```go
func TestCreateTimeEntry(t *testing.T) {
    // 1. Create entry
    // 2. Verify status = draft
    // 3. Verify items created
    // 4. Submit entry
    // 5. Verify status = submitted
    // 6. Manager approves
    // 7. Verify status = pending_finance
}
```

---

**Next**: [[11-Expenses]] for expense tracking, or [[14-CSV-Exports]] for reports.
