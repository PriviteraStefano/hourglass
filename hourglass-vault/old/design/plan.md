# SurrealDB Migration Architecture - Grill-Me Stress Test Results

**Session Date:** 2026-04-11  
**Document:** 2026-04-11-surrealdb-migration-architecture.md  
**Status:** ✅ Stress-tested & Refined

---

## Executive Summary

The SurrealDB migration architecture has been stress-tested via grill-me methodology. Major decisions resolved:

1. **Org hierarchy is reporting-only, not approval-authority** (WG managers approve time entries, org managers reallocate post-approval)
2. **Users map to ONE primary unit** (with optional secondary units); WG config determines if unit selection is flexible or enforced
3. **Simplified time entry statuses:** Draft → Submitted → Approved (no intermediate approval levels)
4. **Financial period cutoff** with Finance override capability (for post-cutoff edits)
5. **Separate expense workflow** (Finance approves, not WG managers)
6. **Normalized schema** (references only, no denormalization)
7. **Single AuditLog table** (immutable, shared for both time entries & expenses)

---

## Refined Architecture Decisions

### 1. Organization Hierarchy
- **Unlimited nesting:** No max depth constraint
- **Implementation:** SurrealDB relations with implicit graph (parent_unit_id pointers)
- **Authority:** Org managers view/report on hierarchies; approve/reject time entries post-WG approval
- **Reporting queries:** Use recursive traversal on parent_unit_id for "all descendants"

### 2. User-to-Unit Mapping
- **Primary unit:** Each user has exactly one `primary_unit_id` (home unit for default attribution)
- **Secondary units:** Optional array `secondary_units` (with start/end dates for historical tracking)
- **Time entry unit assignment:** Determined by WG config (see below)

### 3. Working Group (WG) Configuration
Each WG has a **`enforce_unit_tuple`** flag:
- **If true (default):** User + Unit binding is locked. Time entries auto-attributed to configured unit. If user tries to override via UI, API auto-corrects.
- **If false (advanced):** User can select unit from their assigned units when submitting. Org manager oversight required.

### 4. Time Entry Status Flow (Simplified)
```
Draft → Submitted → Approved
                    ↓
              [Post-approval actions tracked in AuditLog only]
                    ↓
            Relocated / Split / Finance Override
```

**Status meanings:**
- **Draft:** User creating entry locally
- **Submitted:** User submitted to WG manager; awaiting approval/action
- **Approved:** WG manager approved; entry is complete and final for cost allocation

**Post-approval actions (audit-trail only, no status change):**
- Org manager reallocates (changes project/unit)
- WG manager splits entry (creates new entries with lineage)
- Finance edits locked entries (after cutoff) with mandatory reason

### 5. WG Manager Powers (Before Approval)
1. **Accept** → Approve entry as-is (Submitted → Approved)
2. **Reject** → Return to user with reason (Submitted → Draft + motivation disclaimer)
3. **Split** → Accept partial hours, reject partial hours
   - Creates 2+ new TimeEntry records (Submitted status)
   - Original entry marked soft-deleted
   - Audit trail shows lineage (`created_from_entry_id`)
4. **Move/Reassign** → Move entry to different project/WG
   - Creates new TimeEntry linked to new project
   - Original soft-deleted
   - Audit trail shows lineage

### 6. Org Manager Powers (Post-Approval)
- **Reallocate:** Change entry's project/unit after approval
- **Tracked in AuditLog** with action type `reallocated` (no status change)
- **Permission:** Only units under org manager's hierarchy
- **Cannot revoke:** Once approved by WG, org manager cannot reject; only reallocate

### 7. Financial Period Cutoff
- **Organization default cutoff:** e.g., "End of month + 7 days"
- **Inherited by projects/WGs** (override allowed per project)
- **After cutoff:** All time entries for that period are locked
  - WG manager cannot edit/revert
  - Finance role CAN edit with mandatory reason (audit trail)
  - Reason is required and immutable
- **Effect:** Protects financial reconciliation; allows Finance to fix errors post-closure

### 8. Expense Workflow (Separate from Time Entries)
- **Approval authority:** Finance team (not WG managers)
- **Status flow:** Draft → Submitted → Approved (Finance approves)
- **User data:** Project (for customer tracking), category, amount, receipt (with OCR)
- **Budget caps:** Configurable per user/project/category
- **No WG involvement:** Expenses bypassed working group approval
- **Audit trail:** Finance edits tracked separately

### 9. Data Model: Normalized References
- **TimeEntry:** Stores `user_id`, `project_id`, `subproject_id`, `wg_id`, `unit_id` (not nested objects)
- **Expense:** Stores `user_id`, `project_id`, `category` (not nested)
- **Benefit:** Single source of truth for user/project metadata; clean updates; no denormalization overhead

### 10. Audit Trail: Single Immutable Log
- **Table:** `AuditLog` (shared for all entry types)
- **Immutable by design:** No corrections to audit records; only append new records
- **Fields:** `entry_id`, `type` ("time_entry" | "expense"), `action`, `actor_role`, `actor_id`, `timestamp`, `reason`, `changes` (before/after)
- **Query examples:**
  - "All approvals by manager M:" `SELECT * FROM audit_log WHERE actor_id = "m:111" ORDER BY timestamp`
  - "All edits to entry E:" `SELECT * FROM audit_log WHERE entry_id = "te:123" ORDER BY timestamp`
  - "All Finance overrides in April:" `SELECT * FROM audit_log WHERE actor_role = "finance" AND type = "time_entry" AND timestamp >= 2026-04-01`

### 11. Delegation (MVP Scope)
- **Current:** One manager per WG, delegates can approve on behalf
- **Post-MVP TODO:** Smart auto-approval rules (e.g., "auto-approve if hours < 8"), AI integration for anomaly detection

### 12. Project → Subproject → WG Hierarchy
- **Project:** Planning/business level (e.g., "ERP System Implementation")
- **Subproject:** Logical breakdown (e.g., "Backend Module", "Frontend Module")
- **WG:** Execution team (e.g., "Backend Team", "Frontend Team")
- **Benefit:** Separates planning intent from team structure
- **Flexibility:** Multiple WGs per subproject allowed; each WG maps to different units/managers

---

## Implementation Todos

### Phase 1: Schema Design (Foundation)
- [ ] Define SurrealDB tables: Organization, Unit, User, UnitMembership
- [ ] Define Project, Subproject, WorkingGroup
- [ ] Define TimeEntry, Expense, AuditLog
- [ ] Define Financial cutoff configs
- [ ] Test recursive unit hierarchy queries

### Phase 2: Time Entry Workflow
- [ ] Implement TimeEntry CRUD with Draft → Submitted → Approved statuses
- [ ] Implement WG manager approval (Accept/Reject/Split/Move)
- [ ] Implement org manager reallocation (post-approval)
- [ ] Implement financial period cutoff enforcement
- [ ] Implement Finance override for locked entries

### Phase 3: Expense Workflow
- [ ] Implement Expense CRUD with Finance approval
- [ ] Implement receipt OCR integration
- [ ] Implement budget cap validation
- [ ] Implement audit trail for expense edits

### Phase 4: Audit Trail & Compliance
- [ ] Implement immutable AuditLog
- [ ] Query tools for compliance reports (all actions by role, date range, etc.)
- [ ] Verify audit trail integrity

### Phase 5: Org Hierarchy & Reporting
- [ ] Implement recursive hierarchy queries (all descendants, all ancestors)
- [ ] Implement org manager permission model (can manage units under them)
- [ ] Build resource allocation reports (% time per project per user)

### Phase 6: Testing & Migration
- [ ] Test cutoff enforcement edge cases
- [ ] Test Finance overrides with reason tracking
- [ ] Migrate data from PostgreSQL to SurrealDB
- [ ] Validate data integrity post-migration

---

## Key Design Constraints

1. **WG manager approval is final** (org manager can't override before approval, only after)
2. **Audit trail is immutable** (no corrections to audit records)
3. **Financial cutoff is enforced** (except by Finance role with reason)
4. **Unit binding can be enforced** (WG config determines flexibility)
5. **Expenses ≠ Time Entries** (separate workflows, separate approval authority)

---

## Open Questions / Post-MVP

1. **Delegation auto-approval rules:** Implement smart rules for auto-approving low-risk entries
2. **Multi-unit user handling:** Test edge cases (user in 3+ units, conflicting managers)
3. **SurrealDB performance:** Benchmark recursive hierarchy queries at scale (1M+ entries, 10K+ units)
4. **Financial reporting integration:** Design dashboards for cost breakdown by unit/project/manager

---

## Success Criteria

- [x] Stress-tested approval workflow (no ambiguous authority)
- [x] Clear separation of concerns (WG approval vs. org reporting vs. finance)
- [x] Immutable audit trail (compliance-ready)
- [x] Normalized schema (single source of truth)
- [x] Financial period protection (cutoff + override)
- [x] Flexible yet enforced unit binding (WG config controls)

---

**Status:** Ready for detailed SurrealDB schema design phase.
