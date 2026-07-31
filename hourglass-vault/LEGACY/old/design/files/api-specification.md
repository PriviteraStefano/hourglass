# SurrealDB API Specification: Time Entry & Expense Workflows

**Document:** REST API endpoints for approval workflows  
**Date:** 2026-04-12  
**Base URL:** `/api/v1`

---

## Time Entry Endpoints

### 1. Create Time Entry (User)
```
POST /time-entries
Content-Type: application/json
Authorization: Bearer $token

{
  "project_id": "p:123",
  "subproject_id": "sp:456",
  "hours": 8,
  "description": "Implemented backend API endpoints",
  "entry_date": "2026-04-11",
  "unit_id": "u:789"  // Optional if enforce_unit_tuple
}

Response (201):
{
  "data": {
    "id": "te:1001",
    "status": "draft",
    "user_id": "u:456",
    "project_id": "p:123",
    "hours": 8,
    "created_at": "2026-04-12T09:00:00Z"
  }
}
```

**Implementation Notes:**
- If `unit_id` not provided and WG has `enforce_unit_tuple = true`, auto-assign from WG config
- If `enforce_unit_tuple = false`, user must choose from their assigned units
- User can only create for themselves (unless admin/manager)
- Status starts as 'draft'

---

### 2. List Time Entries (User Dashboard)
```
GET /time-entries?status=draft&project_id=p:123
Authorization: Bearer $token

Response (200):
{
  "data": [
    {
      "id": "te:1001",
      "status": "draft",
      "hours": 8,
      "project_name": "ERP System Implementation",
      "entry_date": "2026-04-11",
      "wg_name": "Backend Engineering"
    }
  ],
  "meta": {
    "total": 5,
    "page": 1
  }
}
```

**Query Parameters:**
- `status`: 'draft', 'submitted', 'approved'
- `project_id`: filter by project
- `start_date`, `end_date`: date range
- `wg_id`: filter by working group

---

### 3. Submit Time Entry (User → WG Manager)
```
POST /time-entries/{id}/submit
Authorization: Bearer $token

{
  "notes": "Completed backend work as described"
}

Response (200):
{
  "data": {
    "id": "te:1001",
    "status": "submitted",
    "submitted_at": "2026-04-12T10:00:00Z"
  }
}

// Audit log created:
{
  entry_id: "te:1001",
  entry_type: "time_entry",
  action: "submitted",
  actor_role: "user",
  actor_id: "u:456",
  timestamp: now()
}
```

**Implementation:**
- Change status from 'draft' → 'submitted'
- User can only submit own entries
- Create audit log entry

---

### 4. List Pending Approvals (WG Manager)
```
GET /approvals/pending?role=wg_manager
Authorization: Bearer $token

Response (200):
{
  "data": [
    {
      "id": "te:1001",
      "user_name": "Sarah",
      "project_name": "ERP Implementation",
      "hours": 8,
      "description": "Implemented backend API",
      "submitted_at": "2026-04-12T10:00:00Z",
      "age_hours": 2
    }
  ]
}
```

**Query:**
```sql
SELECT 
  te.id, 
  te.user_id, 
  (SELECT name FROM users WHERE id == te.user_id),
  te.project_id,
  te.hours,
  te.description,
  te.created_at
FROM time_entries te
WHERE te.wg_id IN (
  SELECT id FROM working_groups WHERE manager_id == $user_id
)
AND te.status == 'submitted'
AND te.is_deleted == false
ORDER BY te.created_at ASC;
```

---

### 5. Approve Time Entry (WG Manager)
```
POST /time-entries/{id}/approve
Authorization: Bearer $token
X-Role: wg_manager

{}

Response (200):
{
  "data": {
    "id": "te:1001",
    "status": "approved",
    "approved_by": "John (Backend WG Manager)",
    "approved_at": "2026-04-12T11:00:00Z"
  }
}

// Audit log:
{
  entry_id: "te:1001",
  action: "approved",
  actor_role: "wg_manager",
  actor_id: $manager_id
}
```

---

### 6. Reject Time Entry (WG Manager)
```
POST /time-entries/{id}/reject
Authorization: Bearer $token
X-Role: wg_manager

{
  "reason": "Please clarify which feature this work was for"
}

Response (200):
{
  "data": {
    "id": "te:1001",
    "status": "draft",
    "rejected_reason": "Please clarify which feature this work was for",
    "returned_at": "2026-04-12T11:00:00Z"
  }
}

// Audit log:
{
  entry_id: "te:1001",
  action: "rejected",
  actor_role: "wg_manager",
  actor_id: $manager_id,
  reason: "Please clarify which feature this work was for"
}
```

**Implementation:**
- Change status: 'submitted' → 'draft'
- Prepend rejection reason to description as disclaimer
- Create audit log with reason

---

### 7. Split Time Entry (WG Manager)
```
POST /time-entries/{id}/split
Authorization: Bearer $token
X-Role: wg_manager

{
  "splits": [
    {
      "hours": 6,
      "status": "approved",
      "description": "Actual backend work completed"
    },
    {
      "hours": 2,
      "status": "rejected",
      "description": "Partial work, needs clarification",
      "move_to_project": "p:999"  // Optional, default "down" project
    }
  ],
  "reason": "Split work: 6h valid, 2h needs review"
}

Response (201):
{
  "data": {
    "original_entry_id": "te:1001",
    "split_entries": [
      {
        "id": "te:2001",
        "hours": 6,
        "status": "approved"
      },
      {
        "id": "te:2002",
        "hours": 2,
        "status": "submitted"
      }
    ]
  }
}
```

**Implementation:**
1. Mark original entry soft-deleted: `UPDATE time_entries SET is_deleted = true WHERE id == "te:1001"`
2. For each split:
   - Create new TimeEntry with `created_from_entry_id = "te:1001"`
   - If status == 'approved', immediately approve
   - If status == 'rejected', start as 'submitted' for manager review
3. Create audit logs for each action
4. Link entries via `created_from_entry_id`

---

### 8. Move Time Entry (WG Manager)
```
POST /time-entries/{id}/move
Authorization: Bearer $token
X-Role: wg_manager

{
  "new_project_id": "p:999",  // "down" project or other project
  "reason": "Work doesn't fit project requirements, moving to internal"
}

Response (200):
{
  "data": {
    "id": "te:1001",
    "new_project_id": "p:999",
    "original_project_id": "p:123",
    "status": "approved"
  }
}

// Audit log:
{
  entry_id: "te:1001",
  action: "moved",
  actor_role: "wg_manager",
  actor_id: $manager_id,
  reason: "Work doesn't fit project requirements",
  changes: {
    project_id: { before: "p:123", after: "p:999" }
  }
}
```

---

### 9. Reallocate Time Entry (Org Manager, Post-Approval)
```
POST /time-entries/{id}/reallocate
Authorization: Bearer $token
X-Role: org_manager

{
  "new_unit_id": "u:888",
  "reason": "Correcting unit attribution per org structure review"
}

Response (200):
{
  "data": {
    "id": "te:1001",
    "old_unit_id": "u:789",
    "new_unit_id": "u:888",
    "status": "approved"
  }
}

// Audit log:
{
  entry_id: "te:1001",
  action: "reallocated",
  actor_role: "org_manager",
  actor_id: $org_manager_id,
  reason: "Correcting unit attribution per org structure review",
  changes: {
    unit_id: { before: "u:789", after: "u:888" }
  }
}
```

**Permission Check:**
```
-- Org manager can only reallocate entries for units they manage
SELECT COUNT(*) FROM unit_memberships 
WHERE user_id == $org_manager_id 
AND unit_id == (SELECT parent_unit_id FROM units WHERE id == $old_unit_id)
```

---

### 10. Finance Override for Locked Entry
```
POST /time-entries/{id}/finance-override
Authorization: Bearer $token
X-Role: finance

{
  "action": "approve",  // or "reject"
  "reason": "Correcting erroneous approval post-cutoff, valid work confirmed"
}

Response (200):
{
  "data": {
    "id": "te:1001",
    "status": "approved",
    "override_by": "Finance Team",
    "override_reason": "Correcting erroneous approval..."
  }
}

// Audit log:
{
  entry_id: "te:1001",
  action: "finance_override",
  actor_role: "finance",
  actor_id: $finance_user_id,
  reason: "Correcting erroneous approval post-cutoff, valid work confirmed"
}
```

**Preconditions:**
- Entry must be in cutoff period (is_locked == true)
- `reason` field mandatory (cannot be empty)
- Audit trail captures this as special action

---

## Expense Endpoints

### 1. Create Expense (User)
```
POST /expenses
Content-Type: application/json
Authorization: Bearer $token

{
  "category": "meal",
  "amount": 25.50,
  "currency": "EUR",
  "description": "Team lunch during sprint planning",
  "expense_date": "2026-04-11",
  "project_id": "p:123",  // Optional
  "receipt_url": "https://..."  // Optional, will OCR
}

Response (201):
{
  "data": {
    "id": "exp:1001",
    "status": "draft",
    "user_id": "u:456",
    "amount": 25.50
  }
}
```

---

### 2. Submit Expense (User)
```
POST /expenses/{id}/submit
Authorization: Bearer $token

{}

Response (200):
{
  "data": {
    "id": "exp:1001",
    "status": "submitted"
  }
}
```

---

### 3. List Expenses for Approval (Finance)
```
GET /approvals/expenses?status=submitted
Authorization: Bearer $token
X-Role: finance

Response (200):
{
  "data": [
    {
      "id": "exp:1001",
      "user_name": "Sarah",
      "category": "meal",
      "amount": 25.50,
      "submitted_at": "2026-04-12T14:00:00Z"
    }
  ]
}
```

---

### 4. Approve Expense (Finance)
```
POST /expenses/{id}/approve
Authorization: Bearer $token
X-Role: finance

{}

Response (200):
{
  "data": {
    "id": "exp:1001",
    "status": "approved"
  }
}
```

---

### 5. Finance Override Expense
```
POST /expenses/{id}/finance-override
Authorization: Bearer $token
X-Role: finance

{
  "action": "reject",
  "reason": "Exceeded monthly meal budget cap of €100"
}

Response (200):
{
  "data": {
    "id": "exp:1001",
    "status": "rejected",
    "rejection_reason": "Exceeded monthly meal budget cap..."
  }
}
```

---

## Audit Log Endpoints

### 1. Get Audit Trail for Entry
```
GET /time-entries/{id}/audit
Authorization: Bearer $token

Response (200):
{
  "data": [
    {
      "id": "al:1001",
      "action": "submitted",
      "actor_name": "Sarah",
      "actor_role": "user",
      "timestamp": "2026-04-12T10:00:00Z"
    },
    {
      "id": "al:1002",
      "action": "approved",
      "actor_name": "John",
      "actor_role": "wg_manager",
      "timestamp": "2026-04-12T11:00:00Z"
    },
    {
      "id": "al:1003",
      "action": "reallocated",
      "actor_name": "Alice",
      "actor_role": "org_manager",
      "reason": "Correcting unit attribution",
      "timestamp": "2026-04-12T14:00:00Z"
    }
  ]
}
```

---

### 2. Compliance Report (Finance)
```
GET /audit?action=finance_override&start_date=2026-04-01&end_date=2026-04-30
Authorization: Bearer $token
X-Role: finance

Response (200):
{
  "data": [
    {
      "entry_id": "te:1001",
      "entry_type": "time_entry",
      "action": "finance_override",
      "actor_name": "Finance Team",
      "reason": "Post-cutoff correction",
      "timestamp": "2026-04-27T16:00:00Z"
    }
  ]
}
```

---

## Error Responses

### 400 Bad Request
```json
{
  "error": "Validation failed",
  "details": [
    "hours must be greater than 0",
    "unit_id must match WG enforce_unit_tuple config"
  ]
}
```

### 401 Unauthorized
```json
{
  "error": "Authentication required"
}
```

### 403 Forbidden
```json
{
  "error": "Permission denied",
  "reason": "Only WG managers can approve entries for this working group"
}
```

### 409 Conflict
```json
{
  "error": "Entry is locked",
  "reason": "Financial cutoff period has passed. Only Finance role can override."
}
```

---

## Implementation Checklist

### Phase 1: CRUD Operations
- [ ] Create time entry (auto-unit assignment per WG config)
- [ ] List time entries (user dashboard)
- [ ] Submit time entry (status: draft → submitted)
- [ ] Create expense

### Phase 2: Approval Workflow
- [ ] List pending approvals (WG manager)
- [ ] Approve entry (status: submitted → approved)
- [ ] Reject entry (status: submitted → draft + reason)
- [ ] Finance approve expense

### Phase 3: Complex Operations
- [ ] Split time entry (create new entries with lineage)
- [ ] Move time entry (reassign project)
- [ ] Reallocate entry (org manager, unit reassignment)
- [ ] Finance override (locked entries)

### Phase 4: Reporting & Audit
- [ ] Get audit trail for entry
- [ ] Compliance report (Finance)
- [ ] Cutoff enforcement (check is_locked)

### Phase 5: Validation & Edge Cases
- [ ] Validate budget caps (expenses)
- [ ] Validate WG unit binding (enforce_unit_tuple)
- [ ] Validate financial cutoff (is_locked)
- [ ] Validate permission checks (org_manager, finance)

---

**Status:** API specification complete. Ready for implementation.
