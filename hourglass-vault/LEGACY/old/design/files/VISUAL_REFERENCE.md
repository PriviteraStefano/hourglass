# SurrealDB Schema Visual Reference

**Quick Reference for Table Relationships & Query Patterns**

---

## Entity Relationship Diagram (Simplified)

```
┌─────────────────────────────────────────────────────────────────┐
│                        ORGANIZATIONS                            │
│  (org_id: root tenant identifier)                               │
└────────────┬────────────────────────────────────────────────────┘
             │
             ├─────────────────┬──────────────────┬──────────────────┐
             ▼                 ▼                  ▼                  ▼
         UNITS            CUSTOMERS         PROJECTS         BUDGET_CAPS
    (hierarchy)           (billing)          (planning)          (limits)
         │                                      │
         │                                      ├──── SUBPROJECTS
         │                                      │     (structure)
         │                                      │          │
         │                                      │          └── WORKING_GROUPS
         │                                      │             (execution)
         │                                      │                 │
         ├─────────────────────────────────────┼─────────────────┤
         │                                      │                 │
         ▼                                      ▼                 ▼
    USERS ─┐           UNIT_MEMBERSHIPS   WG_MEMBERS       TIME_ENTRIES
    (auth) │           (many-to-many)     (many-to-many)   (core tracking)
           │                                    │
           │                                    ├─ User
           │                                    ├─ Project
           │                                    ├─ Subproject
           │                                    ├─ WG
           │                                    └─ Unit
           │
           └──────────────────────────────────────────────────────┐
                                                                   │
        EXPENSES ◄─────────────────────────────────────────────────┤
        (Finance approval)                                         │
             │                                                     │
             └─────────────────┬──────────────────────────────────┘
                               │
                               ▼
                        AUDIT_LOGS (immutable)
                      (shared for time & expense)
                      ├─ Created
                      ├─ Submitted
                      ├─ Approved
                      ├─ Rejected
                      ├─ Split (lineage: created_from_entry_id)
                      ├─ Moved
                      ├─ Reallocated
                      └─ Finance Override
```

---

## Table-by-Table Quick Reference

### ORGANIZATIONS
```
id (PK)
name
slug (UNIQUE)
financial_cutoff_days (default: 7)
financial_cutoff_config (JSON: cutoff_day_of_month, grace_days)
```
**Key Queries:**
- `SELECT * FROM organizations WHERE slug == 'acme'`

---

### UNITS (Hierarchical)
```
id (PK)
org_id (FK) ──┐
name          │
parent_unit_id (FK, RECURSIVE) ◄── Enables unlimited nesting
hierarchy_level (0 = Company, 1+ for subunits)
code
```
**Key Queries:**
- Find all descendants: `WHERE parent_unit_id IN (...) RECURSIVELY`
- Find all ancestors: Traverse up via parent_unit_id
- Find level 2 units: `WHERE hierarchy_level == 2`

---

### USERS
```
id (PK)
email (UNIQUE)
name
password_hash
```
**Key Queries:**
- `SELECT * FROM users WHERE email == $email`

---

### UNIT_MEMBERSHIPS (Many-to-Many)
```
id (PK)
org_id (FK)
user_id (FK)
unit_id (FK)
is_primary (bool, identifies home unit)
role (string: employee, manager, lead)
start_date, end_date (track history)
```
**Key Queries:**
- Find primary unit: `WHERE user_id == $user_id AND is_primary == true`
- Find all units: `WHERE user_id == $user_id AND end_date > now()`

---

### PROJECTS
```
id (PK)
org_id (FK)
name
project_type (ENUM: billable, internal)
customer_id (FK, optional)
budget_amount
financial_cutoff_config (optional override)
```
**Key Queries:**
- Find all projects for org: `WHERE org_id == $org_id`
- Find billable projects: `WHERE project_type == 'billable'`

---

### SUBPROJECTS
```
id (PK)
project_id (FK)
name
description
sequence_order
```
**Key Queries:**
- Find all subprojects: `WHERE project_id == $project_id ORDER BY sequence_order`

---

### WORKING_GROUPS
```
id (PK)
org_id (FK)
subproject_id (FK)
name
unit_ids (ARRAY of record<units>) ◄── Determines unit binding
enforce_unit_tuple (bool, default: true) ◄── Controls flexibility
manager_id (FK)
delegate_ids (ARRAY of record<users>)
```
**Key Queries:**
- Find WGs for manager: `WHERE manager_id == $user_id OR manager_id IN delegate_ids`

---

### WG_MEMBERS
```
id (PK)
wg_id (FK)
user_id (FK)
unit_id (FK) ◄── Which unit this user is assigned from
role (e.g., backend_engineer, data_engineer)
is_default_subproject (bool)
start_date, end_date
```
**Key Queries:**
- Find members of WG: `WHERE wg_id == $wg_id AND end_date > now()`

---

### TIME_ENTRIES
```
id (PK)
org_id (FK)
user_id (FK)
project_id (FK)
subproject_id (FK)
wg_id (FK)
unit_id (FK) ◄── Determined by WG config or user selection
hours (number > 0)
description
entry_date (date, IMMUTABLE)
status (ENUM: draft, submitted, approved)
is_deleted (bool, soft-delete flag)
created_from_entry_id (optional, for split entries) ◄── Lineage
created_at, updated_at
```
**Key Queries:**
- Find pending approvals: `WHERE status == 'submitted' AND wg_id == $wg_id`
- Find user's entries: `WHERE user_id == $user_id AND is_deleted == false`
- Find entries in cutoff window: `WHERE entry_date >= $cutoff_start AND entry_date <= $cutoff_end`

---

### EXPENSES
```
id (PK)
org_id (FK)
user_id (FK)
project_id (FK, optional)
unit_id (FK) ◄── User's unit for attribution
category (ENUM: mileage, meal, accommodation, other)
amount (number > 0)
currency (string, default: EUR)
description
expense_date
receipt_url
receipt_ocr_data (JSON, optional)
status (ENUM: draft, submitted, approved, rejected)
is_deleted (bool)
```
**Key Queries:**
- Find submitted expenses: `WHERE status == 'submitted' AND is_deleted == false`
- Check budget cap: `SUM(amount WHERE category == $category AND expense_date >= $period_start)`

---

### AUDIT_LOGS (Immutable)
```
id (PK)
org_id (FK)
entry_id (string, not FK ◄── allows referencing deleted entries)
entry_type (ENUM: time_entry, expense)
action (ENUM: created, submitted, approved, rejected, split, moved, 
        reallocated, edited, finance_override, reverted)
actor_role (ENUM: user, wg_manager, org_manager, finance, admin)
actor_id (FK)
reason (optional, MANDATORY for finance_override)
changes (optional JSON: {field: {before: ..., after: ...}})
timestamp (datetime, DEFAULT now())
ip_address (optional)
```
**Key Queries:**
- Get entry audit trail: `WHERE entry_id == $entry_id ORDER BY timestamp`
- Finance compliance: `WHERE actor_role == 'finance' AND timestamp >= $quarter_start`
- Track splits: `WHERE action == 'split' AND changes.split_into CONTAINS $entry_id`

---

### FINANCIAL_CUTOFF_PERIODS
```
id (PK)
org_id (FK)
project_id (FK, optional override)
period_start (date)
period_end (date)
cutoff_date (date) ◄── After this, entries are locked
is_locked (bool)
```
**Key Queries:**
- Find cutoff for date: `WHERE period_start <= $entry_date AND period_end >= $entry_date`
- Check if locked: `WHERE cutoff_date < now()`

---

### BUDGET_CAPS
```
id (PK)
org_id (FK)
user_id (FK, optional)
project_id (FK, optional)
category (optional)
limit_amount (number > 0)
period (ENUM: daily, weekly, monthly, yearly)
currency (string)
```
**Key Queries:**
- Find relevant caps: `WHERE org_id == $org_id AND (user_id == $user_id OR user_id == null)`
- Validate budget: Check SUM(expenses) < cap for applicable period

---

### CUSTOMERS
```
id (PK)
org_id (FK)
name
email
address
```
**Key Queries:**
- List customers: `WHERE org_id == $org_id`

---

## Critical Query Patterns (Copy-Paste Ready)

### 1. Find All Pending Approvals (WG Manager)
```sql
SELECT 
  te.id,
  te.user_id,
  (SELECT name FROM users WHERE id == te.user_id)[0].name as user_name,
  te.project_id,
  (SELECT name FROM projects WHERE id == te.project_id)[0].name as project_name,
  te.hours,
  te.description,
  te.created_at
FROM time_entries te
WHERE te.wg_id IN (
  SELECT id FROM working_groups 
  WHERE manager_id == $current_user_id
)
AND te.status == 'submitted'
AND te.is_deleted == false
ORDER BY te.created_at ASC;
```

### 2. Find User's Total Allocation (by project)
```sql
SELECT 
  project_id,
  (SELECT name FROM projects WHERE id == project_id)[0].name as project_name,
  SUM(hours) as total_hours,
  (SUM(hours) / (SELECT SUM(hours) FROM time_entries WHERE user_id == $user_id AND status == 'approved') * 100) as allocation_percent
FROM time_entries
WHERE user_id == $user_id 
AND status == 'approved'
AND entry_date >= (NOW() - 1 MONTH)
GROUP BY project_id
ORDER BY total_hours DESC;
```

### 3. Check if Entry is Locked (Financial Cutoff)
```sql
LET $cutoff = SELECT cutoff_date FROM financial_cutoff_periods 
  WHERE org_id == $org_id 
  AND period_start <= $entry_date 
  AND period_end >= $entry_date;

IF $cutoff[0].cutoff_date < NOW() {
  -- Entry is locked
  -- Only Finance can edit (with mandatory reason)
} ELSE {
  -- Entry is editable by WG manager
}
```

### 4. Get Entry's Audit Trail
```sql
SELECT * FROM audit_logs 
WHERE entry_id == $entry_id 
ORDER BY timestamp ASC;
```

### 5. Finance Compliance Report (All Overrides)
```sql
SELECT 
  al.entry_id,
  al.entry_type,
  (SELECT name FROM users WHERE id == al.actor_id)[0].name as actor_name,
  al.reason,
  al.timestamp
FROM audit_logs al
WHERE al.org_id == $org_id 
AND al.action == 'finance_override'
AND al.timestamp >= $quarter_start
AND al.timestamp <= $quarter_end
ORDER BY al.timestamp DESC;
```

---

## Index Strategy

```
✓ org_id on all tables (org-scoped queries)
✓ user_id + status on time_entries (user dashboard)
✓ wg_id + status on time_entries (WG approval list)
✓ entry_date on time_entries (cutoff window checks)
✓ actor_id on audit_logs (Finance compliance queries)
✓ is_deleted on time_entries (exclude soft-deleted)
✓ is_primary on unit_memberships (find home unit)
✓ parent_unit_id on units (hierarchy traversal)
```

---

## Permission Model (Summary)

| Table | Select | Create | Update | Delete |
|-------|--------|--------|--------|--------|
| time_entries | User/WG/Org/Finance | User/WG/Admin | User/WG/Org/Finance | Never |
| expenses | User/Finance/Admin | User/Finance/Admin | User (draft)/Finance | Never |
| audit_logs | Finance/Org | System | Never | Never |
| units | Org members | Org members | Org members | Org members |

---

**Ready for Schema Testing & API Implementation!**
