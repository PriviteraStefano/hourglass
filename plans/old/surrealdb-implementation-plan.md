# SurrealDB Migration: Multi-Phase Implementation Plan

**Date:** 2026-04-12  
**Status:** Ready for Implementation  
**Approach:** Tracer-bullet vertical slices (each phase is independently complete & deployable)

---

## Executive Summary

This plan breaks the SurrealDB migration into **5 independent, valuable phases**. Each phase delivers a complete end-to-end slice of functionality that can be tested, deployed, and used in production.

**PRD Source:** SurrealDB Schema Design Package (grill-me stress-tested, 16 tables, 10+ API endpoints)

**Success Criteria:** 
- All time entries tracked with audit trail
- WG manager can approve/reject/split entries
- Org manager can reallocate entries
- Finance can override locked entries
- Reports show full approval history

---

## Phase Overview

| Phase | Theme | Duration | Endpoints | Tables | Key Deliverable |
|-------|-------|----------|-----------|--------|-----------------|
| 1 | **Foundation** | 1 week | 3 | 8 | Basic time entry CRUD + schema |
| 2 | **Approval Workflow** | 1.5 weeks | 5 | 3 + audit | WG manager approvals |
| 3 | **Financial Controls** | 1 week | 3 | 2 + audit | Cutoff enforcement |
| 4 | **Org Hierarchy & Reallocation** | 1 week | 2 | 2 + audit | Org manager actions |
| 5 | **Reporting & Compliance** | 1 week | 2 | 1 | Audit queries & reports |

**Total:** ~5-6 weeks

---

## Phase 1: Foundation (Schema + Basic CRUD)

**Goal:** Deploy SurrealDB schema and implement basic time entry CRUD operations.

**What Gets Delivered:**
- ✅ SurrealDB cluster running (dev/staging)
- ✅ All 16 tables created with indexes & permissions
- ✅ Time entry CRUD API (Create, Read, List)
- ✅ Basic user authentication
- ✅ Working group membership setup
- ✅ Test data fixtures

**Endpoints Implemented:**
1. `POST /time-entries` — Create time entry (draft)
2. `GET /time-entries` — List user's time entries
3. `GET /time-entries/{id}` — Get single entry

**Database Components:**
- organizations, units, users, unit_memberships (org layer)
- projects, subprojects, working_groups, wg_members (project layer)
- time_entries (empty, no approvals yet)
- Financial cutoff configs (setup only)

**Implementation Checklist:**
- [ ] Deploy SurrealDB cluster
- [ ] Load schema from surrealdb-schema.sql
- [ ] Implement SurrealDB client library
- [ ] Implement JWT auth middleware
- [ ] Implement POST /time-entries (create draft)
- [ ] Implement GET /time-entries (list)
- [ ] Implement GET /time-entries/{id} (read)
- [ ] Create test fixtures (orgs, units, users, WGs)
- [ ] Write unit tests for CRUD
- [ ] Deploy to staging
- [ ] Manual acceptance test

**Success Criteria:**
- [ ] Can create time entry in draft status
- [ ] Can list user's time entries
- [ ] Audit log created on entry creation
- [ ] Schema integrity verified
- [ ] All indexes created

**Testing:**
- Unit tests: CRUD operations
- Integration tests: End-to-end flow
- Permission tests: Only users can create their own entries

**Acceptance:** Stakeholder demo of time entry creation & listing

---

## Phase 2: Approval Workflow (WG Manager Powers)

**Goal:** Implement WG manager approval, rejection, splitting, and moving of time entries.

**What Gets Delivered:**
- ✅ WG manager approval interface
- ✅ Entry rejection with feedback
- ✅ Entry splitting (accept partial, reject partial)
- ✅ Entry moving to different projects
- ✅ Audit trail for all actions
- ✅ Approval dashboard for WG managers

**Endpoints Implemented:**
1. `POST /time-entries/{id}/submit` — User submits entry
2. `GET /approvals/pending` — List pending for WG manager
3. `POST /time-entries/{id}/approve` — WG manager approves
4. `POST /time-entries/{id}/reject` — WG manager rejects (returns to draft)
5. `POST /time-entries/{id}/split` — WG manager splits entry

**Database Components:**
- time_entries (new status field: submitted, approved)
- audit_logs (new actions: submitted, approved, rejected, split, moved)
- Working group manager routing

**Implementation Checklist:**
- [ ] Implement POST /time-entries/{id}/submit
- [ ] Implement GET /approvals/pending (WG manager only)
- [ ] Implement POST /time-entries/{id}/approve
- [ ] Implement POST /time-entries/{id}/reject
- [ ] Implement POST /time-entries/{id}/split logic
  - [ ] Create child entries with lineage
  - [ ] Soft-delete original
  - [ ] Route to appropriate project
- [ ] Implement POST /time-entries/{id}/move
- [ ] Create audit log for each action
- [ ] Implement WG manager permission checks
- [ ] Build WG approval dashboard UI
- [ ] Write tests: split operation edge cases
- [ ] Load test: 100 concurrent approvals

**Success Criteria:**
- [ ] WG manager can approve entries
- [ ] WG manager can reject (entry returns to draft)
- [ ] WG manager can split entries (4h accept + 4h reject)
- [ ] WG manager can move entries to different projects
- [ ] Audit trail shows all actions with timestamps
- [ ] Original entries marked as deleted after split

**Testing:**
- Unit tests: Split algorithm (various hour combinations)
- Integration tests: Full approval workflow
- Permission tests: Only WG managers can approve their entries
- Load test: 100+ concurrent approvals

**Acceptance:** WG manager demo (approve, reject, split workflows)

---

## Phase 3: Financial Controls (Cutoff Enforcement)

**Goal:** Implement financial period cutoffs and Finance role overrides.

**What Gets Delivered:**
- ✅ Financial cutoff period enforcement
- ✅ Entry locking after cutoff date
- ✅ Finance role override capability
- ✅ Mandatory reason tracking for overrides
- ✅ Cutoff configuration per org/project
- ✅ Finance override audit trail

**Endpoints Implemented:**
1. `GET /financial-periods` — List cutoff periods
2. `POST /time-entries/{id}/finance-override` — Finance edits locked entries
3. `GET /approvals/expenses?status=submitted` — List pending expenses

**Database Components:**
- financial_cutoff_periods (period configuration)
- Expense table (separate workflow)
- audit_logs (new action: finance_override with mandatory reason)
- budget_caps (configuration only)

**Implementation Checklist:**
- [ ] Implement financial cutoff period logic
- [ ] Create financial_cutoff_periods table entries
- [ ] Implement entry lock check (is_locked = now() > cutoff_date)
- [ ] Implement Finance role override endpoint
- [ ] Add mandatory reason field to override
- [ ] Create expense CRUD endpoints (draft/submitted)
- [ ] Implement expense budget cap validation
- [ ] Create audit logs for Finance overrides
- [ ] Build Finance override dashboard
- [ ] Write tests: cutoff enforcement (before/after dates)
- [ ] Write tests: reason validation (must not be empty)

**Success Criteria:**
- [ ] Entries locked after cutoff date
- [ ] WG manager cannot edit locked entries
- [ ] Finance can override (with reason)
- [ ] Override reason is immutable in audit trail
- [ ] Expenses can be created and submitted
- [ ] Budget cap validation prevents overspend

**Testing:**
- Unit tests: Cutoff date logic (edge cases)
- Integration tests: WG manager blocked post-cutoff
- Integration tests: Finance override succeeds with reason
- Rejection tests: Finance override fails without reason

**Acceptance:** Finance demo (override locked entry, budget cap enforcement)

---

## Phase 4: Org Hierarchy & Reallocation

**Goal:** Implement org manager capabilities for unit reallocation and hierarchy navigation.

**What Gets Delivered:**
- ✅ Org manager reallocation of entries
- ✅ Recursive hierarchy queries
- ✅ Unit permission checks (manager can only reallocate own units)
- ✅ Org manager dashboard (units, members, allocations)
- ✅ Unit membership history tracking
- ✅ Reallocation audit trail

**Endpoints Implemented:**
1. `POST /time-entries/{id}/reallocate` — Org manager changes unit
2. `GET /units/{id}/descendants` — Org manager views org structure
3. `GET /org-manager/team-allocation` — Allocation report

**Database Components:**
- units (recursive traversal logic)
- unit_memberships (history tracking)
- audit_logs (new action: reallocated with before/after unit)

**Implementation Checklist:**
- [ ] Implement org manager role & permissions
- [ ] Implement POST /time-entries/{id}/reallocate
- [ ] Add unit permission check (can only reallocate own units)
- [ ] Implement recursive unit hierarchy queries
- [ ] Query: Find all descendants of a unit
- [ ] Query: Find all units managed by org manager
- [ ] Build org structure visualization
- [ ] Implement reallocation audit trail
- [ ] Create org manager dashboard
- [ ] Write tests: Permission checks (cannot reallocate other units)
- [ ] Write tests: Hierarchy queries (5-level deep nesting)
- [ ] Load test: 10K unit hierarchy

**Success Criteria:**
- [ ] Org manager can reallocate entries
- [ ] Permission check prevents cross-unit reallocation
- [ ] Recursive queries work on 5+ level hierarchy
- [ ] Audit trail shows unit change (before/after)
- [ ] Dashboard shows unit structure & members

**Testing:**
- Unit tests: Permission checks
- Integration tests: Recursive hierarchy traversal
- Permission tests: Cannot reallocate entries for other units
- Load test: 10K units, find all descendants

**Acceptance:** Org manager demo (reallocate entry, view hierarchy, team allocation report)

---

## Phase 5: Reporting & Compliance

**Goal:** Implement audit trail queries and compliance/financial reports.

**What Gets Delivered:**
- ✅ Audit trail viewer (full action history)
- ✅ Compliance report (Finance overrides, reversions)
- ✅ User allocation report (% time per project)
- ✅ Project cost breakdown (hours by unit)
- ✅ Immutability verification
- ✅ Export capabilities (CSV/PDF)

**Endpoints Implemented:**
1. `GET /time-entries/{id}/audit` — Full audit trail for entry
2. `GET /audit?action=finance_override&date_range=...` — Compliance report
3. `GET /reports/user-allocation` — User time allocation
4. `GET /reports/project-cost` — Project cost breakdown

**Database Components:**
- audit_logs (query only, all data read)
- time_entries (read for reporting)
- units (for rollup reporting)

**Implementation Checklist:**
- [ ] Implement GET /time-entries/{id}/audit (with actor names)
- [ ] Implement GET /audit (compliance query with filters)
- [ ] Implement date range filtering
- [ ] Implement action filtering (finance_override, split, moved, etc.)
- [ ] Implement GET /reports/user-allocation
- [ ] Implement GET /reports/project-cost (with unit rollup)
- [ ] Implement CSV export for reports
- [ ] Build compliance dashboard (Finance team)
- [ ] Build allocation dashboard (Org manager)
- [ ] Write tests: Audit trail immutability (cannot update old records)
- [ ] Write tests: Report accuracy (sum of parts = total)
- [ ] Load test: Audit query on 1M+ entries

**Success Criteria:**
- [ ] Audit trail shows all actions chronologically
- [ ] Compliance report filters by action type & date
- [ ] User allocation report sums to 100% per period
- [ ] Project cost breakdown aggregates correctly
- [ ] Audit records cannot be updated or deleted
- [ ] Reports export to CSV/PDF

**Testing:**
- Unit tests: Aggregation math (allocation %, cost totals)
- Integration tests: Full audit trail accuracy
- Permission tests: Only Finance can view compliance
- Immutability tests: Verify audit records locked

**Acceptance:** Finance & Org Manager demo (audit trail, compliance report, allocation report)

---

## Cross-Phase Considerations

### Security & Permissions
- Each phase implements role-based access control (RBAC)
- Permissions enforced at API and database level
- 5 roles: user, wg_manager, org_manager, finance, admin
- All actions logged to immutable audit trail

### Testing Strategy
- **Unit tests:** Business logic (split algorithm, cutoff logic, queries)
- **Integration tests:** End-to-end workflows (submit → approve → done)
- **Permission tests:** RBAC enforcement (cannot do what you shouldn't)
- **Load tests:** Scale (100K+ entries, 10K units)
- **Audit tests:** Immutability verification

### Deployment Strategy
- Each phase deployed independently
- Database schema fully deployed in Phase 1 (no migrations needed)
- API features gated behind feature flags
- Cutover from PostgreSQL happens after Phase 3 (when cutoffs working)
- Rollback plan: Keep PostgreSQL online for 24h

### Data Migration
- Phase 1: Schema deployed, PostgreSQL still active
- Phase 2: Approval logic tested on SurrealDB test data
- Phase 3: Cutoff logic validated
- Phase 4: Org structure validated
- Phase 5: Reports validated
- **Cutover Day:** Export PostgreSQL → Transform → Load into SurrealDB

### Team Allocation
- **Backend (2 devs):** Phases 1-2 (4 weeks), then Phases 3-5 (3 weeks)
- **Frontend (1 dev):** Builds dashboards in parallel (approval, org manager, finance dashboards)
- **DevOps (1 person):** SurrealDB infrastructure, migration runbook
- **QA (1 person):** Testing each phase, load testing

---

## Detailed Phase 1 Breakdown (Week 1)

**Phase 1: Foundation & Schema Deployment**

### Daily Breakdown

**Day 1: Infrastructure & Schema**
- Deploy SurrealDB cluster (dev environment)
- Load schema: `surrealdb load --file surrealdb-schema.sql`
- Verify schema (all 16 tables, 24 indexes created)
- Create database backup strategy

**Day 2: Auth & Client Library**
- Implement JWT auth middleware
- Create SurrealDB client library wrapper
- Test auth flow (login → token → auth header)
- Create test fixtures (seed data)

**Day 3: Basic CRUD**
- Implement `POST /time-entries` (create draft)
- Implement `GET /time-entries` (list all user's entries)
- Implement `GET /time-entries/{id}` (get single entry)
- Write unit tests for CRUD

**Day 4: Integration & Testing**
- Integration tests: Full CRUD flow
- Permission tests: Only can create own entries
- Audit log verification: Entry created logs recorded
- Load test: 1K entries

**Day 5: Staging & Acceptance**
- Deploy to staging environment
- Manual acceptance testing (stakeholder demo)
- Fix any blocking issues
- Prepare for Phase 2

---

## Phase 2 Breakdown (1.5 weeks)

**Phase 2: Approval Workflow**

### Day 1-2: Submit & List Endpoints
- Implement `POST /time-entries/{id}/submit`
- Implement `GET /approvals/pending` (WG manager view)
- Test permission checks (only WG manager can see)

### Day 3-4: Approve & Reject
- Implement `POST /time-entries/{id}/approve`
- Implement `POST /time-entries/{id}/reject`
- Test full workflow: draft → submitted → approved

### Day 5: Split & Move
- Implement `POST /time-entries/{id}/split` algorithm
- Test edge cases (0h accepted, 100% split, etc.)
- Implement `POST /time-entries/{id}/move`

### Day 6-7: Integration & Dashboard
- Build approval dashboard for WG managers
- Integrate audit trail logging
- Full integration tests
- Load test: 100 concurrent approvals

---

## Phase 3 Breakdown (1 week)

**Phase 3: Financial Controls**

### Day 1-2: Cutoff Configuration
- Create financial_cutoff_periods table entries
- Implement cutoff date logic (is_locked check)
- Test cutoff enforcement (before/after cutoff date)

### Day 3-4: Finance Override
- Implement `POST /time-entries/{id}/finance-override`
- Mandatory reason validation
- Audit trail recording

### Day 5: Expenses
- Implement expense CRUD (create, submit)
- Implement budget cap validation
- Test budget cap enforcement

---

## Phase 4 Breakdown (1 week)

**Phase 4: Org Hierarchy & Reallocation**

### Day 1-2: Org Manager Reallocation
- Implement `POST /time-entries/{id}/reallocate`
- Permission checks (only own units)
- Audit trail logging

### Day 3-4: Hierarchy Queries
- Implement recursive unit queries
- Test on 5-level hierarchy
- Org structure visualization

### Day 5: Dashboard & Testing
- Build org manager dashboard
- Load test: 10K units
- Full integration tests

---

## Phase 5 Breakdown (1 week)

**Phase 5: Reporting & Compliance**

### Day 1-2: Audit Trail Viewer
- Implement `GET /time-entries/{id}/audit`
- Show all actions with actors & timestamps
- Test immutability

### Day 3-4: Compliance Reports
- Implement `GET /audit?action=...&date=...` (compliance query)
- Finance override report
- Split/reallocation history

### Day 5: Financial Reports
- Implement user allocation report
- Implement project cost breakdown
- Export to CSV/PDF

---

## Success Metrics (End-to-End)

✅ **Phase 1 Success:**
- Time entries created in draft status
- CRUD operations working
- Schema validated
- Zero permission violations

✅ **Phase 2 Success:**
- WG manager can approve entries
- Entries split correctly
- Audit trail complete
- 100+ concurrent approvals handled

✅ **Phase 3 Success:**
- Entries locked after cutoff
- Finance can override (with reason)
- Budget caps enforced
- Zero unsanctioned overrides

✅ **Phase 4 Success:**
- Org manager can reallocate
- Permission checks working
- Hierarchy queries fast (< 100ms for 10K units)
- Org structure visible

✅ **Phase 5 Success:**
- Audit trail shows all actions
- Compliance reports accurate
- Allocation reports sum to 100%
- No audit record modifications

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| SurrealDB performance | Load test in Phase 1, cache hierarchy queries |
| Schema changes needed | All schema designed upfront, tested in stress-test |
| Permission bugs | Extensive permission tests in each phase |
| Data migration issues | Run PostgreSQL ↔ SurrealDB in parallel post-Phase 3 |
| Approval logic edge cases | Test split algorithm thoroughly in Phase 2 |
| Cutoff enforcement gaps | Test before/after cutoff in Phase 3 |

---

## Success Criteria (Final)

- ✅ All 10+ API endpoints implemented
- ✅ All 16 SurrealDB tables working
- ✅ All approval workflows functional
- ✅ Financial cutoff enforced
- ✅ Org hierarchy queryable
- ✅ Audit trail immutable
- ✅ Reports accurate
- ✅ Performance: < 200ms for 99th percentile queries
- ✅ Load test: 1M entries, 100K users
- ✅ Zero data loss during migration

---

## Timeline Summary

```
Week 1:     Phase 1 (Foundation) ✓
Week 2-3:   Phase 2 (Approval Workflow) ✓
Week 3:     Phase 3 (Financial Controls) ✓
Week 4:     Phase 4 (Org Hierarchy) ✓
Week 5:     Phase 5 (Reporting) ✓
Week 6:     Testing, Performance Tuning, Cutover Planning

Total: ~6 weeks to production
```

---

## Files Generated

This plan references:
- `surrealdb-schema.sql` — Complete schema
- `schema-implementation-guide.md` — Design patterns & queries
- `api-specification.md` — API endpoints (detailed)
- `VISUAL_REFERENCE.md` — Query templates

All available in session workspace: `~/.copilot/session-state/db5e771a-c70a-4ed2-9441-94efc449e2ff/files/`

---

**Status:** ✅ Ready for Phase 1 Kickoff

**Next Step:** Convert each phase into GitHub issues using tracer-bullet slices.
