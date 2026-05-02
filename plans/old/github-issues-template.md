# GitHub Issues: SurrealDB Migration (Tracer-Bullet Phases)

This file contains templates for breaking the implementation plan into independent GitHub issues.

**Approach:** Each issue represents one vertical slice (end-to-end, independently completable, deployable).

---

## Issue Template: Phase 1 - Foundation & Schema Deployment

**Title:** Phase 1: Deploy SurrealDB & Implement Basic Time Entry CRUD

**Description:**
```
## Overview
Deploy SurrealDB cluster and implement basic time entry creation, reading, and listing operations. This is the foundation phase that sets up the database and core API endpoints.

## What Gets Delivered
- ✅ SurrealDB cluster running (dev/staging)
- ✅ All 16 tables created with indexes & permissions
- ✅ Time entry CRUD API (Create, Read, List)
- ✅ Basic user authentication
- ✅ Working group membership setup
- ✅ Test data fixtures

## Endpoints Implemented
1. POST /time-entries — Create time entry (draft)
2. GET /time-entries — List user's time entries
3. GET /time-entries/{id} — Get single entry

## Acceptance Criteria
- [ ] SurrealDB cluster deployed
- [ ] Schema loaded (surrealdb-schema.sql)
- [ ] All 16 tables created
- [ ] All 24 indexes created
- [ ] JWT auth working
- [ ] POST /time-entries creates entries in draft status
- [ ] GET /time-entries lists user's entries
- [ ] GET /time-entries/{id} retrieves single entry
- [ ] Audit log records entry creation
- [ ] Unit tests: CRUD operations (> 90% coverage)
- [ ] Integration tests: End-to-end flow
- [ ] Permission tests: Only users can create their own entries
- [ ] Deployed to staging environment
- [ ] Stakeholder acceptance demo passed

## Tasks
- [ ] Subtask 1: Deploy SurrealDB cluster
- [ ] Subtask 2: Load schema & verify all tables
- [ ] Subtask 3: Implement SurrealDB client library wrapper
- [ ] Subtask 4: Implement JWT auth middleware
- [ ] Subtask 5: Implement POST /time-entries
- [ ] Subtask 6: Implement GET /time-entries & GET /time-entries/{id}
- [ ] Subtask 7: Create test fixtures (seed data)
- [ ] Subtask 8: Write unit tests
- [ ] Subtask 9: Write integration tests
- [ ] Subtask 10: Deploy to staging & acceptance test

## Testing Checklist
- Unit tests: CRUD operations
- Integration tests: End-to-end flow
- Permission tests: RBAC enforcement
- Audit trail verification: Entry creation logged

## Estimated Duration
1 week (5 days)

## Team
- Backend (2 devs)
- DevOps (1 person for infrastructure)
- QA (1 person for testing)

## Dependencies
None (first phase)

## Related Documentation
- surrealdb-schema.sql — Complete schema definition
- schema-implementation-guide.md — Design patterns
- api-specification.md — API endpoint details
```

**Labels:** `enhancement`, `surrealdb-migration`, `phase-1`  
**Assignee:** Backend Lead  
**Estimate:** 1 week

---

## Issue Template: Phase 2 - Approval Workflow

**Title:** Phase 2: Implement WG Manager Approval Workflow (Approve, Reject, Split, Move)

**Description:**
```
## Overview
Implement WG manager approval, rejection, splitting, and moving of time entries. This phase adds the core approval workflow.

## What Gets Delivered
- ✅ WG manager approval interface
- ✅ Entry rejection with feedback
- ✅ Entry splitting (accept partial, reject partial)
- ✅ Entry moving to different projects
- ✅ Audit trail for all actions
- ✅ Approval dashboard for WG managers

## Endpoints Implemented
1. POST /time-entries/{id}/submit — User submits entry
2. GET /approvals/pending — List pending for WG manager
3. POST /time-entries/{id}/approve — WG manager approves
4. POST /time-entries/{id}/reject — WG manager rejects (returns to draft)
5. POST /time-entries/{id}/split — WG manager splits entry

## Acceptance Criteria
- [ ] POST /time-entries/{id}/submit changes status to submitted
- [ ] GET /approvals/pending returns only entries for this WG manager
- [ ] Permission check: Only WG managers can approve their entries
- [ ] POST /time-entries/{id}/approve changes status to approved
- [ ] POST /time-entries/{id}/reject returns entry to draft with reason
- [ ] POST /time-entries/{id}/split creates multiple child entries
  - [ ] Splits honor accept/reject specification
  - [ ] Original entry marked as deleted
  - [ ] Child entries have created_from_entry_id lineage
  - [ ] Audit trail shows split action
- [ ] POST /time-entries/{id}/move reassigns to different project
- [ ] Audit logs created for all actions
- [ ] Unit tests: Split algorithm edge cases (> 95% coverage)
  - [ ] 4h accept + 4h reject
  - [ ] 0h accept (all rejected)
  - [ ] 8h accept (none rejected)
- [ ] Integration tests: Full approval workflow
- [ ] Permission tests: Only WG managers can approve
- [ ] Load test: 100+ concurrent approvals
- [ ] Approval dashboard displays pending entries
- [ ] Deployed to staging & acceptance demo passed

## Tasks
- [ ] Subtask 1: Implement POST /time-entries/{id}/submit
- [ ] Subtask 2: Implement GET /approvals/pending with WG routing
- [ ] Subtask 3: Implement POST /time-entries/{id}/approve
- [ ] Subtask 4: Implement POST /time-entries/{id}/reject
- [ ] Subtask 5: Implement POST /time-entries/{id}/split logic
  - [ ] Create child entries with lineage
  - [ ] Soft-delete original
  - [ ] Route to appropriate project
- [ ] Subtask 6: Implement POST /time-entries/{id}/move
- [ ] Subtask 7: Add WG manager permission checks
- [ ] Subtask 8: Create audit logs for all actions
- [ ] Subtask 9: Build WG approval dashboard UI
- [ ] Subtask 10: Write comprehensive tests
- [ ] Subtask 11: Load test (100 concurrent approvals)
- [ ] Subtask 12: Deploy to staging & acceptance test

## Testing Checklist
- Unit tests: Split algorithm (various combinations)
- Integration tests: Full approval workflow (submit → approve → done)
- Permission tests: Only WG managers can approve their entries
- Load test: 100+ concurrent approvals
- Audit trail verification: All actions logged

## Estimated Duration
1.5 weeks (7-8 days)

## Team
- Backend (2 devs)
- Frontend (1 dev for dashboard)
- QA (1 person)

## Dependencies
- Phase 1: Foundation (schema deployed, CRUD working)

## Related Documentation
- schema-implementation-guide.md — Split design pattern
- api-specification.md — Endpoint specifications
```

**Labels:** `enhancement`, `surrealdb-migration`, `phase-2`, `approval-workflow`  
**Assignee:** Backend Lead  
**Estimate:** 1.5 weeks

---

## Issue Template: Phase 3 - Financial Controls

**Title:** Phase 3: Implement Financial Cutoff Enforcement & Finance Override

**Description:**
```
## Overview
Implement financial period cutoff enforcement and Finance role override capability for locked entries.

## What Gets Delivered
- ✅ Financial cutoff period enforcement
- ✅ Entry locking after cutoff date
- ✅ Finance role override capability
- ✅ Mandatory reason tracking for overrides
- ✅ Cutoff configuration per org/project
- ✅ Finance override audit trail
- ✅ Expense CRUD (separate workflow)
- ✅ Budget cap validation

## Endpoints Implemented
1. GET /financial-periods — List cutoff periods
2. POST /time-entries/{id}/finance-override — Finance edits locked entries
3. POST /expenses — Create expense (draft)
4. POST /expenses/{id}/submit — User submits expense
5. POST /expenses/{id}/approve — Finance approves expense

## Acceptance Criteria
- [ ] financial_cutoff_periods table configured
- [ ] Entry lock check implemented: now() > cutoff_date
- [ ] WG manager cannot edit entries after cutoff
- [ ] POST /time-entries/{id}/finance-override succeeds when locked
- [ ] Finance override requires mandatory reason (non-empty string)
- [ ] Override reason immutable in audit trail
- [ ] Expense CRUD endpoints working
- [ ] Budget cap validation prevents overspend
- [ ] Unit tests: Cutoff date logic (edge cases)
  - [ ] Before cutoff: editable
  - [ ] On cutoff date: editable
  - [ ] After cutoff: locked
- [ ] Integration tests: WG manager blocked post-cutoff
- [ ] Integration tests: Finance override succeeds with reason
- [ ] Rejection tests: Override fails without reason
- [ ] Permission tests: Only Finance can override
- [ ] Finance override dashboard built
- [ ] Deployed to staging & acceptance demo passed

## Tasks
- [ ] Subtask 1: Create financial_cutoff_periods entries
- [ ] Subtask 2: Implement entry lock check (is_locked logic)
- [ ] Subtask 3: Implement POST /time-entries/{id}/finance-override
- [ ] Subtask 4: Add mandatory reason validation
- [ ] Subtask 5: Create expense CRUD endpoints
- [ ] Subtask 6: Implement budget cap validation
- [ ] Subtask 7: Add Finance role permission checks
- [ ] Subtask 8: Create audit logs for overrides
- [ ] Subtask 9: Build Finance override dashboard
- [ ] Subtask 10: Write cutoff edge case tests
- [ ] Subtask 11: Load test Finance queries
- [ ] Subtask 12: Deploy to staging & acceptance test

## Testing Checklist
- Unit tests: Cutoff date logic (before/on/after cutoff)
- Integration tests: WG manager blocked post-cutoff
- Integration tests: Finance override with reason succeeds
- Rejection tests: Override without reason fails
- Permission tests: Non-Finance users cannot override
- Audit trail verification: Override reason recorded

## Estimated Duration
1 week (5 days)

## Team
- Backend (2 devs)
- QA (1 person)

## Dependencies
- Phase 2: Approval Workflow (entry approval flow working)

## Related Documentation
- schema-implementation-guide.md — Cutoff design pattern
- api-specification.md — Finance override endpoint
```

**Labels:** `enhancement`, `surrealdb-migration`, `phase-3`, `financial-controls`  
**Assignee:** Backend Lead  
**Estimate:** 1 week

---

## Issue Template: Phase 4 - Org Hierarchy & Reallocation

**Title:** Phase 4: Implement Org Manager Reallocation & Hierarchy Navigation

**Description:**
```
## Overview
Implement org manager capabilities for unit reallocation and recursive hierarchy queries.

## What Gets Delivered
- ✅ Org manager reallocation of entries
- ✅ Recursive hierarchy queries
- ✅ Unit permission checks (manager can only reallocate own units)
- ✅ Org manager dashboard (units, members, allocations)
- ✅ Unit membership history tracking
- ✅ Reallocation audit trail

## Endpoints Implemented
1. POST /time-entries/{id}/reallocate — Org manager changes unit
2. GET /units/{id}/descendants — Org manager views org structure
3. GET /org-manager/team-allocation — Allocation report

## Acceptance Criteria
- [ ] Org manager role & permissions implemented
- [ ] POST /time-entries/{id}/reallocate succeeds for own units
- [ ] Permission check prevents cross-unit reallocation
- [ ] Unit permission check: can only reallocate units under org manager
- [ ] Recursive unit hierarchy queries work
- [ ] Query: Find all descendants of a unit (working on 5+ levels)
- [ ] Query: Find all units managed by org manager
- [ ] Reallocation audit trail shows before/after unit
- [ ] Org manager dashboard displays unit structure
- [ ] Unit membership history tracked (start/end dates)
- [ ] Unit tests: Permission checks
  - [ ] Cannot reallocate to unmanaged units
  - [ ] Can reallocate within own hierarchy
- [ ] Integration tests: Recursive hierarchy traversal
- [ ] Permission tests: Cannot reallocate entries for other units
- [ ] Load test: 10K units, find all descendants (< 200ms)
- [ ] Org manager dashboard deployed
- [ ] Deployed to staging & acceptance demo passed

## Tasks
- [ ] Subtask 1: Implement org manager role
- [ ] Subtask 2: Implement POST /time-entries/{id}/reallocate
- [ ] Subtask 3: Add unit permission checks
- [ ] Subtask 4: Implement recursive unit hierarchy queries
- [ ] Subtask 5: Implement GET /units/{id}/descendants
- [ ] Subtask 6: Implement GET /org-manager/team-allocation report
- [ ] Subtask 7: Build org structure visualization
- [ ] Subtask 8: Create reallocation audit trail logging
- [ ] Subtask 9: Build org manager dashboard
- [ ] Subtask 10: Write permission tests
- [ ] Subtask 11: Load test hierarchy queries (10K units)
- [ ] Subtask 12: Deploy to staging & acceptance test

## Testing Checklist
- Unit tests: Permission checks
- Integration tests: Recursive hierarchy queries
- Permission tests: Cannot reallocate to unauthorized units
- Load test: 10K units, all descendants queries (< 200ms)
- Audit trail verification: Reallocation unit change recorded

## Estimated Duration
1 week (5 days)

## Team
- Backend (2 devs)
- Frontend (1 dev for org structure visualization)
- QA (1 person)

## Dependencies
- Phase 3: Financial Controls (org hierarchy config stable)

## Related Documentation
- schema-implementation-guide.md — Hierarchy design pattern
- api-specification.md — Reallocation endpoint
```

**Labels:** `enhancement`, `surrealdb-migration`, `phase-4`, `org-hierarchy`  
**Assignee:** Backend Lead  
**Estimate:** 1 week

---

## Issue Template: Phase 5 - Reporting & Compliance

**Title:** Phase 5: Implement Audit Trail Queries & Compliance Reports

**Description:**
```
## Overview
Implement audit trail viewer and compliance/financial reports for Finance and Org Manager teams.

## What Gets Delivered
- ✅ Audit trail viewer (full action history)
- ✅ Compliance report (Finance overrides, reversions)
- ✅ User allocation report (% time per project)
- ✅ Project cost breakdown (hours by unit)
- ✅ Immutability verification
- ✅ Export capabilities (CSV/PDF)

## Endpoints Implemented
1. GET /time-entries/{id}/audit — Full audit trail for entry
2. GET /audit?action=finance_override&date_range=... — Compliance report
3. GET /reports/user-allocation — User time allocation
4. GET /reports/project-cost — Project cost breakdown

## Acceptance Criteria
- [ ] GET /time-entries/{id}/audit returns all actions chronologically
- [ ] Audit response includes actor names & timestamps
- [ ] GET /audit supports filtering by action type
- [ ] GET /audit supports filtering by date range
- [ ] Compliance report shows all finance_override actions
- [ ] GET /reports/user-allocation shows % time per project
- [ ] User allocation report sums to 100% (or < 100% if part-time)
- [ ] GET /reports/project-cost aggregates hours by unit
- [ ] Project cost breakdown matches sum of parts
- [ ] CSV export for all reports working
- [ ] PDF export for compliance reports working
- [ ] Unit tests: Aggregation math (allocation %, cost totals)
- [ ] Integration tests: Full audit trail accuracy
- [ ] Integration tests: Report calculations correct
- [ ] Permission tests: Only Finance can view compliance
- [ ] Immutability tests: Verify audit records locked (no update/delete)
- [ ] Load test: Audit queries on 1M+ entries (< 500ms)
- [ ] Compliance dashboard built
- [ ] Allocation dashboard built
- [ ] Deployed to staging & acceptance demo passed

## Tasks
- [ ] Subtask 1: Implement GET /time-entries/{id}/audit
- [ ] Subtask 2: Implement GET /audit with filtering
- [ ] Subtask 3: Implement aggregation queries
- [ ] Subtask 4: Implement GET /reports/user-allocation
- [ ] Subtask 5: Implement GET /reports/project-cost
- [ ] Subtask 6: Implement CSV export
- [ ] Subtask 7: Implement PDF export
- [ ] Subtask 8: Build compliance dashboard
- [ ] Subtask 9: Build allocation dashboard
- [ ] Subtask 10: Write aggregation tests
- [ ] Subtask 11: Write immutability tests
- [ ] Subtask 12: Load test (1M+ entries)
- [ ] Subtask 13: Deploy to staging & acceptance test

## Testing Checklist
- Unit tests: Aggregation math (allocation %, cost)
- Integration tests: Full audit trail accuracy
- Integration tests: Report calculations correct
- Permission tests: Only Finance can view compliance
- Immutability tests: Audit records cannot be updated/deleted
- Load test: Audit query on 1M+ entries (< 500ms)

## Estimated Duration
1 week (5 days)

## Team
- Backend (1 dev for queries, 1 dev for dashboards)
- Frontend (1 dev for compliance/allocation dashboards)
- QA (1 person)

## Dependencies
- Phase 4: Org Hierarchy & Reallocation (all data stable)

## Related Documentation
- schema-implementation-guide.md — Query patterns
- api-specification.md — Report endpoints
```

**Labels:** `enhancement`, `surrealdb-migration`, `phase-5`, `reporting`  
**Assignee:** Backend Lead  
**Estimate:** 1 week

---

## Summary

| Phase | Title | Duration | Status |
|-------|-------|----------|--------|
| 1 | Deploy SurrealDB & Implement Basic Time Entry CRUD | 1 week | Not Started |
| 2 | Implement WG Manager Approval Workflow | 1.5 weeks | Blocked on Phase 1 |
| 3 | Implement Financial Cutoff Enforcement | 1 week | Blocked on Phase 2 |
| 4 | Implement Org Manager Reallocation & Hierarchy | 1 week | Blocked on Phase 3 |
| 5 | Implement Audit Trail Queries & Reports | 1 week | Blocked on Phase 4 |

**Total Duration:** ~6 weeks

---

## How to Use These Templates

1. Create GitHub issue for Phase 1
2. Use the description, acceptance criteria, and tasks as-is
3. Add to project board
4. When Phase 1 completes, create Phase 2 issue
5. Repeat for Phases 3-5

Each issue is independently completable and deployable. When one phase is done, the next can begin.

---

**Status:** Ready for GitHub issue creation
