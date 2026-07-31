# SurrealDB Migration: Complete Schema Design Package

**Date:** 2026-04-12  
**Status:** ✅ Schema Design Complete  
**Next Phase:** API Implementation

---

## Deliverables Summary

This package contains all artifacts for SurrealDB migration:

### 1. **surrealdb-schema.sql** (20.8 KB)
Complete SurrealDB schema definition with:
- 16 tables (organizations, units, users, projects, subprojects, working_groups, time_entries, expenses, audit_logs, financial_cutoff_periods, budget_caps, customers, + supporting tables)
- Full field definitions with types and constraints
- Indexes on all common query paths
- SurrealDB PERMISSIONS for role-based access control
- Ready to execute: `surrealdb load --file surrealdb-schema.sql`

### 2. **schema-implementation-guide.md** (12 KB)
Detailed walkthrough with:
- Table relationships and entity-relationship diagrams
- 8 key design patterns (hierarchy, unit binding, cutoff enforcement, etc.)
- SQL query examples for each pattern
- Critical query patterns for common operations
- Permission model explained
- Migration path from PostgreSQL
- Performance considerations
- Complete testing checklist

### 3. **api-specification.md** (12 KB)
REST API endpoints with:
- 10 time entry endpoints (create, submit, approve, reject, split, move, reallocate, override)
- 5 expense endpoints (create, submit, approve)
- Audit log endpoints (trail, compliance reports)
- Complete request/response examples
- Implementation notes for each endpoint
- Error response formats
- Full implementation checklist

### 4. **plan.md** (in session workspace)
Project plan with:
- Executive summary of grill-me decisions
- 12 refined architecture decisions explained
- 6 implementation phases (foundation → testing)
- Key design constraints
- Success criteria (all ✅ complete)

---

## Key Architecture Highlights

### Approval Authority (No Ambiguity)
```
Submitted Entry
    ↓
    ├─ WG Manager approves (time entries) → Approved
    │  └─ Can also: split, move, reject
    │
    └─ Finance approves (expenses) → Approved

Post-Approval:
    ├─ Org Manager can reallocate (audit trail only)
    └─ Finance can override if in cutoff period (with mandatory reason)
```

### Financial Period Cutoff (Compliance-Ready)
```
Timeline:
  Period Start ─────── Period End ──────────── Cutoff Date ────────┐
  (Entry date OK)   (Entry date OK)      (WG manager LOCKED)    (Finance locked)
  
Before cutoff: WG manager can edit/revert entries
After cutoff:  Only Finance can edit (with mandatory reason in audit trail)
```

### Time Entry Status Flow (Simplified)
```
Draft ──(user submits)──> Submitted ──(WG approves)──> Approved
  ↑                           ↑
  └──(WG rejects)─────────────┘

Post-approval actions (no status change, audit trail only):
  • Reallocate (org manager)
  • Split (WG manager creates new entries)
  • Finance override (edit locked entries)
```

### Unit Hierarchy (Recursive Graph)
```
Company (root, parent_unit_id = null)
  ├─ Tech Division
  │   ├─ R&D Department
  │   │   ├─ ML Team
  │   │   └─ Data Team
  │   └─ Infra Department
  └─ Finance Division
      └─ Accounting

Org manager of R&D can see: R&D + ML + Data (all descendants)
Unlimited depth: no constraints
```

### Working Group Unit Binding (Flexible Enforcement)
```
If enforce_unit_tuple = true (default):
  WG configuration: unit_ids = [u:123, u:456]
  User submits entry with unit_id = u:999
  API auto-corrects: unit_id = u:123 (first configured unit)

If enforce_unit_tuple = false (advanced):
  User can choose from their assigned units
  API validates: unit_id must be in user's assigned units
  Org manager oversight required
```

### Audit Trail (Immutable, Shared)
```
Single audit_logs table:
  - entry_id: which entry was modified
  - entry_type: "time_entry" | "expense"
  - action: 'submitted', 'approved', 'split', 'reallocated', 'finance_override', ...
  - actor_role: 'user', 'wg_manager', 'org_manager', 'finance', 'admin'
  - reason: mandatory for finance_override
  - changes: before/after state (optional)

Immutable by design: PERMISSIONS disable update/delete
```

---

## Quick Start

### 1. Load Schema
```bash
cd /path/to/hourglass-vault/design/files
surrealdb load --file surrealdb-schema.sql
```

### 2. Test Basic Queries
```sql
-- Create organization
INSERT INTO organizations {
  id: 'org:1',
  name: 'Acme Corp',
  slug: 'acme'
};

-- Create root unit
INSERT INTO units {
  id: 'u:1',
  org_id: 'org:1',
  name: 'Company',
  hierarchy_level: 0
};

-- Create user
INSERT INTO users {
  id: 'u:100',
  email: 'sarah@acme.com',
  name: 'Sarah'
};

-- Add user to unit
INSERT INTO unit_memberships {
  id: 'um:1',
  org_id: 'org:1',
  user_id: 'u:100',
  unit_id: 'u:1',
  is_primary: true
};
```

### 3. Test Recursive Hierarchy
```sql
-- Find all descendants of a unit
SELECT * FROM units 
WHERE parent_unit_id IN (
  SELECT id FROM units WHERE parent_unit_id == 'u:1'
)
// Repeat recursively or use SurrealDB's graph traversal
```

### 4. Test Audit Trail
```sql
-- Create time entry
INSERT INTO time_entries {
  id: 'te:1',
  org_id: 'org:1',
  user_id: 'u:100',
  project_id: 'p:1',
  subproject_id: 'sp:1',
  wg_id: 'wg:1',
  unit_id: 'u:1',
  hours: 8,
  description: 'Backend work',
  entry_date: d'2026-04-11',
  status: 'draft'
};

-- Submit entry
UPDATE time_entries SET status = 'submitted' WHERE id == 'te:1';

-- Log audit event
INSERT INTO audit_logs {
  org_id: 'org:1',
  entry_id: 'te:1',
  entry_type: 'time_entry',
  action: 'submitted',
  actor_role: 'user',
  actor_id: 'u:100'
};

-- View audit trail
SELECT * FROM audit_logs 
WHERE entry_id == 'te:1' 
ORDER BY timestamp;
```

---

## Migration Strategy (PostgreSQL → SurrealDB)

### Step 1: Parallel Setup (Day 1)
- Deploy SurrealDB cluster alongside PostgreSQL
- Load schema (surrealdb-schema.sql)
- Validate schema integrity

### Step 2: Data Export (Day 2)
- Export PostgreSQL tables to JSON
- Transform UUIDs → strings (with `id:` prefix)
- Map ENUM values exactly
- Handle null/default values

### Step 3: Data Import (Day 2)
- Bulk load JSON into SurrealDB
- Verify row counts match
- Validate referential integrity
- Run sample queries

### Step 4: Testing (Day 3)
- Test all API endpoints on SurrealDB
- Verify cutoff enforcement
- Test audit trail immutability
- Load test (100K entries)

### Step 5: Cutover (Day 4)
- Run time entry export at T-0
- Final PostgreSQL → SurrealDB sync
- Switch API to SurrealDB
- Monitor for errors

### Step 6: Rollback Plan (Ready at T-0)
- Keep PostgreSQL online for 24 hours
- If issues detected, revert API to PostgreSQL
- Fix SurrealDB, re-test, try again

---

## File Structure (Session Workspace)

```
~/.copilot/session-state/db5e771a-c70a-4ed2-9441-94efc449e2ff/
├── plan.md                           # Project plan (this session)
└── files/
    ├── surrealdb-schema.sql          # Full schema definition
    ├── schema-implementation-guide.md # Query patterns & design
    ├── api-specification.md          # API endpoints & examples
    └── SCHEMA_DESIGN_PACKAGE.md      # This summary
```

---

## Success Criteria (All ✅)

- [x] Stress-tested approval workflow (grill-me: 21 questions)
- [x] Clear separation: WG approval vs. org reporting vs. finance
- [x] Immutable audit trail (compliance-ready)
- [x] Normalized schema (single source of truth)
- [x] Financial period protection (cutoff + Finance override)
- [x] Flexible yet enforced unit binding (WG config controls)
- [x] Simplified status flow (Draft → Submitted → Approved)
- [x] Complete schema with 16 tables, indexes, permissions
- [x] Query patterns for all common operations
- [x] API specification with 10+ endpoints
- [x] Migration strategy documented

---

## Known Limitations / Post-MVP

1. **Recursive hierarchy queries** may be slow at 10K+ units
   - Solution: Cache unit ancestry, or store denormalized `unit_path`
   - Test at scale before production

2. **Delegation auto-approval** deferred
   - Basic delegation: delegates approve on behalf of manager
   - Smart auto-approval (AI-based anomaly detection): Phase 2

3. **Expense OCR integration** is placeholder
   - Design includes `receipt_ocr_data` field
   - Actual OCR vendor integration (Google Vision, Tesseract): Phase 2

4. **Multi-WG per project** is designed but untested
   - Assumption: most projects have 1 WG
   - Test with 3+ WGs per project before scaling

5. **Budget caps** validation is in API layer
   - SurrealDB schema stores caps
   - API must validate on expense create/update

---

## Next Steps

### For Backend Team
1. Implement API endpoints (see api-specification.md)
2. Implement SurrealDB client (SDK or REST wrapper)
3. Implement cutoff enforcement logic
4. Implement permission checks (role-based access)

### For DevOps Team
1. Deploy SurrealDB cluster (HA, backups)
2. Set up replication/failover
3. Configure monitoring (audit log growth, query latency)
4. Prepare migration runbook

### For QA Team
1. Test schema with sample data (see schema-implementation-guide.md)
2. Test API endpoints (all 10+ scenarios)
3. Test financial cutoff enforcement
4. Test audit trail immutability
5. Load test (1M entries, 100K users)

---

## Questions?

Refer to:
- **Design decisions?** → plan.md (grill-me outcomes)
- **Schema details?** → surrealdb-schema.sql + schema-implementation-guide.md
- **API usage?** → api-specification.md
- **Query patterns?** → schema-implementation-guide.md (Critical Query Patterns section)

---

**Status:** ✅ Complete. Ready for API implementation.  
**Artifacts:** 4 documents + 1 schema SQL file (total ~56 KB)
