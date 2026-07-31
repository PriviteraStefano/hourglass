# SurrealDB Schema Implementation Guide

**Document:** Detailed schema walkthrough with query examples  
**Date:** 2026-04-12  
**Status:** Ready for implementation

---

## Table Overview & Relationships

### Organization Layer (5 tables)
- **organizations**: Tenant/company level
- **units**: Hierarchical org structure (parent_unit_id for graph)
- **users**: Cross-org user identities
- **unit_memberships**: Users → Units mapping (many-to-many)
- **customers**: External customers for project billing

### Project Layer (4 tables)
- **projects**: Planning level (billable/internal)
- **subprojects**: Logical breakdown (modules/phases)
- **working_groups**: Execution teams
- **wg_members**: User → WG assignment (enforced unit binding)

### Time Entry Layer (1 table)
- **time_entries**: Core time tracking records

### Expense Layer (1 table)
- **expenses**: Separate approval workflow (Finance-led)

### Audit & Config (4 tables)
- **audit_logs**: Immutable shared log (time entries + expenses)
- **financial_cutoff_periods**: Period-level cutoff config
- **budget_caps**: Expense limits per user/project/category
- **customers**: Billing customers

---

## Key Design Patterns

### 1. Unit Hierarchy (Recursive Graph)
```sql
-- Find all descendants of a unit (recursive)
SELECT * FROM units WHERE id IN (
  SELECT ->(unit_id) FROM (
    SELECT * FROM units WHERE parent_unit_id == $unit_id
  )
)

-- Find all ancestors of a unit
SELECT parent_unit_id FROM units 
WHERE id == $unit_id 
THEN SELECT * FROM units WHERE id == parent_unit_id THEN ...
```

**Implementation:**
- Use `parent_unit_id` for implicit graph (null = root)
- Query recursively or store denormalized path for fast hierarchy checks
- Unlimited depth: no constraints on nesting

### 2. User-to-Unit Mapping
```sql
-- Find primary unit for user
SELECT * FROM unit_memberships 
WHERE user_id == $user_id 
AND is_primary == true 
AND (end_date == null OR end_date > now());

-- Find all active units for user
SELECT * FROM unit_memberships 
WHERE user_id == $user_id 
AND (end_date == null OR end_date > now())
ORDER BY is_primary DESC;
```

**Implementation:**
- `is_primary = true` designates home unit
- `start_date` and `end_date` track membership history
- Support multiple concurrent unit memberships

### 3. Working Group Unit Binding (enforce_unit_tuple)
```sql
-- Get WG configuration and unit binding
SELECT id, name, enforce_unit_tuple, unit_ids FROM working_groups 
WHERE id == $wg_id;

-- If enforce_unit_tuple == true: 
-- Auto-assign time entry to configured unit
-- Ignore user's unit selection

-- If enforce_unit_tuple == false:
-- Allow user to choose from their assigned units
-- API validates user is in at least one of wg.unit_ids
```

**Implementation:**
- WG stores `unit_ids` array
- Flag `enforce_unit_tuple` controls flexibility
- API auto-corrects entry.unit_id to match WG config if enforced

### 4. Time Entry Status & Cutoff
```sql
-- Check if entry is within cutoff window
SELECT * FROM financial_cutoff_periods 
WHERE org_id == $org_id 
AND period_start <= $entry_date 
AND period_end >= $entry_date;

-- If entry_date is before cutoff_date: editable by WG manager
-- If entry_date is after cutoff_date: locked (only Finance can edit)

-- Check if entry is locked for this date
LET $cutoff = SELECT cutoff_date FROM financial_cutoff_periods 
  WHERE org_id == $org_id 
  AND period_start <= $entry_date 
  AND period_end >= $entry_date;

IF now() > $cutoff {
  -- Entry is locked; only Finance can edit (with reason)
}
```

**Implementation:**
- Time entries have immutable `entry_date`
- `financial_cutoff_periods` define locked ranges
- Enforcement logic in API layer (SurrealDB permissions are not sufficient)
- Finance overrides tracked as audit events with mandatory `reason`

### 5. Time Entry Split (Lineage Tracking)
```sql
-- When splitting entry (e.g., accept 4h, reject 4h):
-- 1. Mark original as soft-deleted
UPDATE time_entries SET is_deleted = true WHERE id == $original_entry_id;

-- 2. Create first child entry (accepted 4h)
INSERT INTO time_entries {
  user_id: $user_id,
  project_id: $project_id,
  subproject_id: $subproject_id,
  wg_id: $wg_id,
  unit_id: $unit_id,
  hours: 4,
  status: 'submitted',
  created_from_entry_id: $original_entry_id
};

-- 3. Create second child entry (rejected 4h, moved to "down" project)
INSERT INTO time_entries {
  user_id: $user_id,
  project_id: $down_project_id,
  hours: 4,
  status: 'submitted',
  created_from_entry_id: $original_entry_id
};

-- 4. Log audit events for split
INSERT INTO audit_logs {
  entry_id: $original_entry_id,
  entry_type: 'time_entry',
  action: 'split',
  actor_role: 'wg_manager',
  actor_id: $manager_id,
  reason: 'Split due to partial work validity',
  changes: {
    split_into: [$child1_id, $child2_id]
  }
};
```

**Implementation:**
- Original entries never deleted, marked `is_deleted = true`
- `created_from_entry_id` links children to parent
- Audit trail shows action: 'split' with child entry IDs
- New entries start as 'submitted' for manager review

### 6. Audit Trail (Immutable Log)
```sql
-- Query all approvals for entry
SELECT * FROM audit_logs 
WHERE entry_id == $entry_id 
AND entry_type == 'time_entry'
ORDER BY timestamp;

-- Query all actions by manager in date range
SELECT * FROM audit_logs 
WHERE actor_id == $manager_id 
AND timestamp >= $start_date 
AND timestamp <= $end_date
ORDER BY timestamp DESC;

-- Query Finance overrides (post-cutoff edits)
SELECT * FROM audit_logs 
WHERE actor_role == 'finance'
AND action == 'finance_override'
AND entry_type == 'time_entry'
ORDER BY timestamp DESC;

-- Verify audit trail integrity (immutable)
-- All records are appended, never modified or deleted
-- PERMISSIONS ensure create-only (no update/delete)
```

**Implementation:**
- Append-only log (SurrealDB PERMISSIONS: update/delete disabled)
- All approval actions create new audit records
- `reason` field mandatory for Finance overrides
- `changes` field captures before/after state (optional for simple approvals)

### 7. Expense Workflow (Finance-Led)
```sql
-- Expense approval (Finance only, not WG manager)
UPDATE expenses SET status = 'approved' WHERE id == $expense_id;

-- Log approval
INSERT INTO audit_logs {
  entry_id: $expense_id,
  entry_type: 'expense',
  action: 'approved',
  actor_role: 'finance',
  actor_id: $finance_user_id,
  timestamp: now()
};

-- Check budget cap for expense
SELECT * FROM budget_caps 
WHERE org_id == $org_id 
AND (user_id == $user_id OR user_id == null)
AND (project_id == $project_id OR project_id == null)
AND (category == $category OR category == null)
AND period == 'monthly';

-- Validate: SUM(expenses.amount WHERE category == $category) < budget_cap.limit_amount
```

**Implementation:**
- Separate from time entries
- Finance approval only (no WG manager involved)
- Budget cap validation in API layer
- OCR data stored in `receipt_ocr_data` (optional object)

### 8. Org Manager Reallocation (Post-Approval)
```sql
-- Org manager reallocates entry to different unit
UPDATE time_entries 
SET unit_id = $new_unit_id 
WHERE id == $entry_id;

-- Log reallocation (audit only, no status change)
INSERT INTO audit_logs {
  entry_id: $entry_id,
  entry_type: 'time_entry',
  action: 'reallocated',
  actor_role: 'org_manager',
  actor_id: $org_manager_id,
  reason: 'Incorrect unit attribution, corrected to appropriate department',
  changes: {
    unit_id: {
      before: $old_unit_id,
      after: $new_unit_id
    }
  }
};

-- Verify org manager permission: is manager of old_unit or parent?
SELECT COUNT(*) FROM unit_memberships 
WHERE user_id == $org_manager_id 
AND (unit_id == $old_unit_id OR unit_id IN (
  SELECT id FROM units WHERE parent_unit_id == $old_unit_id RECURSIVELY
))
```

**Implementation:**
- Org manager can only reallocate entries for units they manage
- Reallocation is metadata update (unit_id field)
- Tracked as audit event (action: 'reallocated')
- No status change (entry stays 'approved')

---

## Critical Query Patterns

### Find Ready-for-Approval Time Entries (WG Manager)
```sql
SELECT * FROM time_entries 
WHERE status == 'submitted' 
AND wg_id == $wg_id
AND is_deleted == false
ORDER BY created_at ASC;
```

### Find Entries Approaching Cutoff (Finance Alert)
```sql
SELECT * FROM time_entries 
WHERE org_id == $org_id
AND status == 'approved'
AND entry_date >= (SELECT cutoff_date - 5 DAYS FROM financial_cutoff_periods WHERE org_id == $org_id)
ORDER BY entry_date DESC;
```

### Calculate User Allocation (Org Manager Report)
```sql
SELECT 
  user_id,
  project_id,
  SUM(hours) as total_hours,
  (SUM(hours) / (SELECT SUM(hours) FROM time_entries WHERE user_id == $user_id AND status == 'approved') * 100) as allocation_percent
FROM time_entries
WHERE user_id == $user_id 
AND status == 'approved'
AND entry_date >= NOW() - 1 MONTH
GROUP BY project_id;
```

### Find Audit Trail for Compliance (Finance Review)
```sql
SELECT * FROM audit_logs 
WHERE org_id == $org_id 
AND action IN ['finance_override', 'split', 'moved']
AND timestamp >= $quarter_start 
AND timestamp <= $quarter_end
ORDER BY timestamp DESC;
```

---

## Permission Model (SurrealDB)

### Time Entries
- **Select:** User (own entries), WG manager, Org manager, Finance, Admin
- **Create:** User (own), WG manager, Admin
- **Update:** User (own, draft only), WG manager (submitted), Org manager (approved, reallocation), Finance (locked entries)
- **Delete:** Never (soft-delete via `is_deleted` flag)

### Audit Logs
- **Select:** Admin, Finance, Org manager, Org members
- **Create:** API layer (system writes)
- **Update:** Never
- **Delete:** Never

### Expenses
- **Select:** User (own), Finance, Admin
- **Create:** User (own), Finance, Admin
- **Update:** User (draft), Finance (any status, with reason)
- **Delete:** Never

---

## Migration Path from PostgreSQL

1. **Export data** from PostgreSQL tables to JSON
2. **Transform** JSON to match SurrealDB field types (UUID → string)
3. **Load into SurrealDB** using `INSERT INTO table {content}` or bulk import
4. **Verify data integrity** (row counts, foreign keys, audit trail)
5. **Test queries** on sample data
6. **Run in parallel** (PostgreSQL + SurrealDB) during transition
7. **Cutover** when confident

### Schema Mapping (PostgreSQL → SurrealDB)
| PostgreSQL | SurrealDB |
|------------|-----------|
| UUID | string (with `id:` prefix) |
| JSONB | object |
| ARRAY | array |
| TIMESTAMP | datetime |
| BOOLEAN | bool |
| ENUM | string with ENUM constraint |

---

## Performance Considerations

1. **Indexes on common queries:**
   - `(org_id)` for org-scoped queries
   - `(user_id, status)` for user dashboards
   - `(entry_date)` for cutoff window checks
   - `(wg_id, status)` for WG manager approval lists

2. **Recursive hierarchy queries:**
   - Cache unit ancestry on query (compute once, reuse)
   - Or store denormalized `unit_path` (e.g., `/Company/Tech/R&D/ML`)

3. **Audit log queries:**
   - Partition by date range if table grows very large
   - Archive old logs to separate read-only table

4. **Aggregate queries (reports):**
   - Use SurrealDB's native `GROUP BY` and aggregation
   - Cache results for monthly/quarterly reports

---

## Testing Checklist

- [ ] Create org with 5-level unit hierarchy
- [ ] Add users to multiple units (test is_primary logic)
- [ ] Create WG with enforce_unit_tuple = true/false
- [ ] Submit time entry, test unit auto-assignment
- [ ] Test WG manager split operation (verify lineage)
- [ ] Test org manager reallocation (verify audit trail)
- [ ] Test financial cutoff (verify lock behavior)
- [ ] Test Finance override (verify reason captured)
- [ ] Test expense budget cap (verify validation)
- [ ] Test audit trail immutability (verify no updates)
- [ ] Benchmark recursive hierarchy query (1K units)
- [ ] Benchmark full audit log scan (100K entries)

---

**Status:** Schema design complete. Ready for API implementation phase.
