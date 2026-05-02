# SurrealDB Migration: Granular GitHub Issues

## Master Issue Template

**Title:** `[EPIC] SurrealDB Migration: Full Implementation (6 Weeks)`

**Description:**

Complete migration of Hourglass from PostgreSQL to SurrealDB with full approval workflow, financial controls, and reporting.

**Scope:**
- Deploy SurrealDB infrastructure
- Migrate schema & implement API
- Build approval workflows (WG manager, Org manager, Finance)
- Implement financial controls & cutoff enforcement
- Build reporting & compliance suite
- Execute production migration

**Timeline:** ~6 weeks (5 phases)

**Success Criteria:**
- [ ] Phase 1 Complete: Time entry CRUD working
- [ ] Phase 2 Complete: Approval workflow working
- [ ] Phase 3 Complete: Financial controls & ready for migration
- [ ] Phase 4 Complete: Org hierarchy & reallocation working
- [ ] Phase 5 Complete: Reporting working & production ready
- [ ] Zero data loss in production migration

**Related Documentation:**
- [Implementation Plan](./surrealdb-implementation-plan.md)
- [API Specification](../files/api-specification.md)
- [Schema Design](../files/surrealdb-schema.sql)

---

## Phase 1: Foundation & Schema (1 Week)

### Sub-Issue 1.1: Infrastructure & Database Setup

**Title:** `[Phase 1.1] Deploy SurrealDB Cluster & Load Schema`

**Description:**

Set up SurrealDB infrastructure, deploy cluster, and load the complete schema with permissions.

**Tasks:**
- [ ] Provision SurrealDB cluster (dev/staging)
- [ ] Configure connection pooling & backups
- [ ] Load surrealdb-schema.sql into database
- [ ] Verify all 16 tables created
- [ ] Verify all 24 indexes created
- [ ] Test SurrealDB PERMISSIONS (role-based access)
- [ ] Create test fixtures: orgs, units, users, working groups
- [ ] Document connection string & credentials

**Acceptance Criteria:**
- [ ] SurrealDB cluster operational
- [ ] Schema fully loaded (16 tables visible)
- [ ] All indexes created and operational
- [ ] PERMISSIONS enforced (admin role can read, user role limited)
- [ ] Test data fixtures loaded

**Testing:**
- [ ] `SELECT * FROM organizations` returns fixture data
- [ ] Permission checks working (user can't access other user's data)
- [ ] Index queries fast (< 50ms)

**Dependencies:** None

**Estimated Duration:** 2-3 days

**Team:** DevOps (1 person)

---

### Sub-Issue 1.2: Authentication & JWT Setup

**Title:** `[Phase 1.2] Implement JWT Authentication Endpoints`

**Description:**

Implement user registration, login, and JWT token generation.

**Tasks:**
- [ ] Implement POST /auth/register (create user, hash password)
- [ ] Implement POST /auth/login (verify password, return JWT)
- [ ] Implement GET /auth/me (profile endpoint, return current user)
- [ ] Implement JWT middleware (verify token, inject user context)
- [ ] Setup JWT secret in environment variables
- [ ] Add password hashing (bcrypt) with salt
- [ ] Write auth unit tests

**Acceptance Criteria:**
- [ ] User can register new account
- [ ] User can login with correct credentials
- [ ] User cannot login with wrong password
- [ ] JWT token expires after configured time
- [ ] GET /auth/me returns current user profile
- [ ] Invalid tokens rejected with 401

**Testing:**
- [ ] Unit tests: password hashing, token generation
- [ ] Integration tests: full register/login/profile flow
- [ ] Security tests: invalid passwords, expired tokens

**Dependencies:** Sub-Issue 1.1

**Estimated Duration:** 2-3 days

**Team:** Backend (1 person)

---

### Sub-Issue 1.3: Time Entry CRUD Endpoints

**Title:** `[Phase 1.3] Implement Time Entry CRUD Operations`

**Description:**

Implement create, read, list, and basic update operations for time entries.

**Tasks:**
- [ ] Implement POST /time-entries (create entry in draft status)
- [ ] Implement GET /time-entries (list user's entries with pagination)
- [ ] Implement GET /time-entries/{id} (get single entry)
- [ ] Implement PATCH /time-entries/{id} (update draft entry)
- [ ] Create audit log entry on each operation
- [ ] Add validation: hours > 0, project_id exists, user in project
- [ ] Write comprehensive tests

**Acceptance Criteria:**
- [ ] Can create time entry in draft status
- [ ] Can list all user's entries
- [ ] Can retrieve specific entry by ID
- [ ] Can update draft entry (before submission)
- [ ] Cannot update submitted/approved entries
- [ ] Audit log created for each operation
- [ ] Pagination works (limit=10 by default)

**Testing:**
- [ ] Unit tests: validation logic
- [ ] Integration tests: full CRUD flow
- [ ] Permission tests: user can only access own entries
- [ ] Edge cases: 0 hours, invalid project, archived project

**Dependencies:** Sub-Issue 1.1, 1.2

**Estimated Duration:** 3-4 days

**Team:** Backend (1 person)

---

### Sub-Issue 1.4: Phase 1 Testing & Acceptance

**Title:** `[Phase 1.4] Phase 1 Testing, Integration & Acceptance`

**Description:**

Comprehensive testing, integration, and stakeholder acceptance for Phase 1 deliverables.

**Tasks:**
- [ ] Write Phase 1 integration tests (end-to-end flows)
- [ ] Run performance tests (latency, throughput)
- [ ] Run load tests (concurrent users)
- [ ] Deploy to staging environment
- [ ] Create test data for acceptance demo
- [ ] Run stakeholder acceptance test
- [ ] Document setup process & runbook

**Acceptance Criteria:**
- [ ] All Phase 1 endpoints working
- [ ] No permission leaks (user A can't access user B's data)
- [ ] Schema integrity verified
- [ ] Latency acceptable (CRUD < 100ms)
- [ ] Throughput adequate (100+ concurrent time entries/sec)
- [ ] Stakeholder acceptance obtained
- [ ] Production readiness checklist passed

**Testing:**
- [ ] Integration tests: 50+ test cases
- [ ] Performance: p99 latency < 200ms
- [ ] Load: 100 concurrent users
- [ ] Security: JWT validation, permission checks

**Dependencies:** Sub-Issues 1.1, 1.2, 1.3

**Estimated Duration:** 2-3 days

**Team:** QA (1 person)

---

## Phase 2: Approval Workflow (1.5 Weeks)

### Sub-Issue 2.1: Entry Submission & Status Tracking

**Title:** `[Phase 2.1] Implement Entry Submission & Status Tracking`

**Description:**

Implement time entry submission workflow and status transitions (draft → submitted → pending_manager).

**Tasks:**
- [ ] Implement POST /time-entries/{id}/submit (change status to submitted)
- [ ] Implement GET /approvals/pending (list pending approvals for WG manager)
- [ ] Add entry status validation (draft only can submit)
- [ ] Create audit log entry for submission
- [ ] Add notifications (WG manager notified of pending approval)
- [ ] Write tests

**Acceptance Criteria:**
- [ ] Can submit draft entry
- [ ] Cannot submit already-submitted entry
- [ ] Status changes to "submitted"
- [ ] Audit log shows submission
- [ ] WG manager can see pending approvals

**Testing:**
- [ ] Unit tests: status transitions
- [ ] Integration tests: submit flow
- [ ] Edge cases: double-submit, invalid states

**Dependencies:** Sub-Issues 1.1, 1.3

**Estimated Duration:** 2 days

**Team:** Backend (1 person)

---

### Sub-Issue 2.2: WG Manager Approval & Rejection

**Title:** `[Phase 2.2] Implement WG Manager Approval & Rejection`

**Description:**

Implement WG manager ability to approve or reject submitted time entries with feedback.

**Tasks:**
- [ ] Implement POST /time-entries/{id}/approve (approve entry)
- [ ] Implement POST /time-entries/{id}/reject (reject with reason)
- [ ] Add authorization check: only WG manager of user's WG can approve
- [ ] Create audit log with decision
- [ ] Add rejection reason tracking
- [ ] Create approval dashboard for WG managers
- [ ] Write tests

**Acceptance Criteria:**
- [ ] WG manager can approve entry (status → approved)
- [ ] WG manager can reject with reason (status → rejected)
- [ ] Non-WG-managers cannot approve
- [ ] Approval tracked in audit log with timestamp
- [ ] Rejection reason stored & visible

**Testing:**
- [ ] Unit tests: approval logic
- [ ] Permission tests: only WG manager can approve
- [ ] Integration tests: full approval flow
- [ ] Edge cases: user self-approval blocked

**Dependencies:** Sub-Issue 2.1

**Estimated Duration:** 2-3 days

**Team:** Backend (1 person)

---

### Sub-Issue 2.3: Time Entry Splitting

**Title:** `[Phase 2.3] Implement Time Entry Splitting`

**Description:**

Implement WG manager ability to split time entries (accept partial hours, reject partial).

**Tasks:**
- [ ] Implement POST /time-entries/{id}/split (split entry into 2 entries)
- [ ] Add validation: split hours sum to original entry
- [ ] Create new entries with `created_from_entry_id` lineage
- [ ] Mark original as soft-deleted (is_deleted = true)
- [ ] Accepted portion → submitted status (automatic approval not needed)
- [ ] Rejected portion → rejected status
- [ ] Create audit trail for split action
- [ ] Write tests

**Acceptance Criteria:**
- [ ] Can split 8-hour entry into 5h accepted + 3h rejected
- [ ] New entries created with correct status
- [ ] Original entry marked deleted
- [ ] Audit log shows split action
- [ ] Child entries linked to parent via lineage

**Testing:**
- [ ] Unit tests: split algorithm (4h+4h, 1h+7h, etc.)
- [ ] Integration tests: full split + approval flow
- [ ] Edge cases: unequal splits, rounding errors

**Dependencies:** Sub-Issue 2.2

**Estimated Duration:** 3 days

**Team:** Backend (1 person)

---

### Sub-Issue 2.4: Time Entry Moving

**Title:** `[Phase 2.4] Implement Time Entry Moving Between Projects`

**Description:**

Implement WG manager ability to move approved time entries to different projects.

**Tasks:**
- [ ] Implement POST /time-entries/{id}/move (move to different project)
- [ ] Add validation: destination project in same WG or WG manager's units
- [ ] Update entry.project_id
- [ ] Create audit log for move action
- [ ] Verify entry is already approved (cannot move draft/pending)
- [ ] Write tests

**Acceptance Criteria:**
- [ ] Can move approved entry to another project
- [ ] Cannot move draft/submitted entries
- [ ] Audit log shows move with from/to projects
- [ ] Entry status remains "approved"

**Testing:**
- [ ] Unit tests: project validation
- [ ] Integration tests: full move flow
- [ ] Permission tests: WG manager can only move their units' entries

**Dependencies:** Sub-Issue 2.2

**Estimated Duration:** 2 days

**Team:** Backend (1 person)

---

### Sub-Issue 2.5: Approval Workflow Testing & Acceptance

**Title:** `[Phase 2.5] Phase 2 Testing, Performance & Acceptance`

**Description:**

Comprehensive testing of approval workflow, performance validation, and stakeholder acceptance.

**Tasks:**
- [ ] Write Phase 2 integration tests (approval workflows)
- [ ] Run performance tests (approval queries, splits, moves)
- [ ] Run load tests (100+ concurrent approvals)
- [ ] Create approval dashboard UI mockups (design only)
- [ ] Run stakeholder acceptance test
- [ ] Validate audit trail completeness

**Acceptance Criteria:**
- [ ] All approval endpoints working
- [ ] Split algorithm tested thoroughly (20+ test cases)
- [ ] No approval authority leaks
- [ ] Audit trail complete & immutable
- [ ] Performance adequate for high volume
- [ ] Stakeholder acceptance obtained

**Testing:**
- [ ] Integration tests: 100+ test cases
- [ ] Performance: approval queries < 200ms
- [ ] Load: 100 concurrent approvals/sec
- [ ] Audit: all actions logged & queryable

**Dependencies:** Sub-Issues 2.1-2.4

**Estimated Duration:** 3 days

**Team:** QA (1 person)

---

## Phase 3: Financial Controls (1 Week)

### Sub-Issue 3.1: Financial Cutoff Enforcement

**Title:** `[Phase 3.1] Implement Financial Cutoff Enforcement`

**Description:**

Implement financial period cutoff logic that locks entries after cutoff date.

**Tasks:**
- [ ] Load financial_cutoff_periods table with config
- [ ] Implement cutoff validation in time entry endpoints
- [ ] Block WG manager edits after cutoff date
- [ ] Allow Finance override after cutoff
- [ ] Return clear error message when entry locked
- [ ] Add cutoff date to time entry response
- [ ] Write tests

**Acceptance Criteria:**
- [ ] Entries lockable by cutoff date
- [ ] Before cutoff: WG manager can edit
- [ ] After cutoff: WG manager blocked
- [ ] After cutoff: Finance can override
- [ ] Error message clear (cutoff date shown)

**Testing:**
- [ ] Unit tests: cutoff logic
- [ ] Integration tests: before/after cutoff scenarios
- [ ] Edge cases: on cutoff date itself

**Dependencies:** Sub-Issues 1.1, 1.3

**Estimated Duration:** 2 days

**Team:** Backend (1 person)

---

### Sub-Issue 3.2: Finance Override Capability

**Title:** `[Phase 3.2] Implement Finance Override of Locked Entries`

**Description:**

Implement Finance user ability to override cutoff locks and edit/approve entries with mandatory reason.

**Tasks:**
- [ ] Implement POST /time-entries/{id}/finance-override (edit locked entry)
- [ ] Add mandatory reason field (audit trail)
- [ ] Add Finance role authorization check
- [ ] Create audit log with reason
- [ ] Add Finance dashboard (pending overrides)
- [ ] Write tests

**Acceptance Criteria:**
- [ ] Finance can edit locked entries
- [ ] Reason required (cannot override without reason)
- [ ] Audit log shows reason & Finance user
- [ ] Non-Finance users blocked

**Testing:**
- [ ] Unit tests: authorization checks
- [ ] Integration tests: override flow
- [ ] Permission tests: only Finance can override

**Dependencies:** Sub-Issue 3.1

**Estimated Duration:** 2 days

**Team:** Backend (1 person)

---

### Sub-Issue 3.3: Expense Management & Budget Caps

**Title:** `[Phase 3.3] Implement Expense CRUD & Budget Cap Validation`

**Description:**

Implement expense submission, Finance approval, and budget cap enforcement.

**Tasks:**
- [ ] Implement POST /expenses (create expense)
- [ ] Implement GET /expenses (list expenses)
- [ ] Implement POST /expenses/{id}/approve (Finance approval)
- [ ] Implement POST /expenses/{id}/reject (Finance rejection)
- [ ] Load budget_caps table with org/unit budget limits
- [ ] Validate expense against budget cap
- [ ] Create audit log for all expense actions
- [ ] Write tests

**Acceptance Criteria:**
- [ ] Can submit expense in draft
- [ ] Finance can approve/reject
- [ ] Cannot exceed budget cap
- [ ] Clear error when budget exceeded
- [ ] Audit trail complete

**Testing:**
- [ ] Unit tests: budget cap logic
- [ ] Integration tests: full expense flow
- [ ] Edge cases: rounding, multiple expenses

**Dependencies:** Sub-Issues 1.1, 1.2

**Estimated Duration:** 3 days

**Team:** Backend (1 person)

---

### Sub-Issue 3.4: Phase 3 Testing & Migration Readiness

**Title:** `[Phase 3.4] Phase 3 Testing & PostgreSQL→SurrealDB Migration Preparation`

**Description:**

Comprehensive testing of financial controls and preparation for production migration.

**Tasks:**
- [ ] Write Phase 3 integration tests (cutoff, override, expenses)
- [ ] Run load tests (Finance override throughput)
- [ ] Create PostgreSQL→SurrealDB migration runbook
- [ ] Design data validation checks (pre/post migration)
- [ ] Create rollback plan
- [ ] Run stakeholder acceptance test
- [ ] Document financial compliance validation

**Acceptance Criteria:**
- [ ] All financial endpoints working
- [ ] Cutoff enforcement tested thoroughly
- [ ] Budget caps validated
- [ ] Migration runbook complete & reviewed
- [ ] Data validation checks ready
- [ ] Rollback plan documented
- [ ] Stakeholder acceptance obtained

**Testing:**
- [ ] Integration tests: 50+ financial scenarios
- [ ] Performance: override operations < 300ms
- [ ] Load: 50 concurrent Finance overrides/sec
- [ ] Migration: data integrity tests ready

**Dependencies:** Sub-Issues 3.1-3.3

**Estimated Duration:** 3 days

**Team:** QA (1 person) + DevOps (0.5 person)

---

## Phase 4: Org Hierarchy & Reallocation (1 Week)

### Sub-Issue 4.1: Org Manager Reallocation

**Title:** `[Phase 4.1] Implement Org Manager Reallocation of Approved Entries`

**Description:**

Implement org manager ability to reallocate approved entries to different units (post-approval action).

**Tasks:**
- [ ] Implement POST /time-entries/{id}/reallocate (move to different unit)
- [ ] Add authorization: only org manager of source unit can reallocate
- [ ] Update entry.unit_id
- [ ] Create audit log (action: 'reallocated')
- [ ] Entry status remains "approved" (not re-submitted)
- [ ] Build reallocation dashboard for org managers
- [ ] Write tests

**Acceptance Criteria:**
- [ ] Org manager can reallocate approved entry
- [ ] Cannot reallocate draft/submitted entries
- [ ] Audit log shows reallocation with source/target units
- [ ] Status doesn't change (approved → approved)

**Testing:**
- [ ] Unit tests: authorization checks
- [ ] Integration tests: full reallocation flow
- [ ] Permission tests: only org manager can reallocate own units

**Dependencies:** Sub-Issues 2.2

**Estimated Duration:** 2 days

**Team:** Backend (1 person)

---

### Sub-Issue 4.2: Recursive Unit Hierarchy Queries

**Title:** `[Phase 4.2] Implement Recursive Unit Hierarchy Queries`

**Description:**

Implement efficient queries for org hierarchy navigation (find descendants, find managers, etc.).

**Tasks:**
- [ ] Implement GET /units/{id}/descendants (all child units recursively)
- [ ] Implement GET /units/{id}/ancestors (all parent units recursively)
- [ ] Implement GET /units/{id}/managers (WG managers for unit)
- [ ] Add query caching for hierarchy (time-limited)
- [ ] Optimize indexes for hierarchy traversal
- [ ] Write performance tests
- [ ] Document query patterns

**Acceptance Criteria:**
- [ ] Hierarchy queries complete & correct
- [ ] Large hierarchies (10K+ units) query in < 200ms
- [ ] Caching improves performance
- [ ] All query patterns documented

**Testing:**
- [ ] Unit tests: hierarchy traversal logic
- [ ] Performance tests: < 200ms for large hierarchies
- [ ] Edge cases: circular references (if possible), deep nesting

**Dependencies:** Sub-Issues 1.1

**Estimated Duration:** 3 days

**Team:** Backend (1 person)

---

### Sub-Issue 4.3: Org Structure Dashboard

**Title:** `[Phase 4.3] Build Org Structure Navigation Dashboard (UI)`

**Description:**

Build frontend dashboard for org managers to navigate hierarchy and view reallocation history.

**Tasks:**
- [ ] Build org structure tree view (React component)
- [ ] Implement unit selection & drill-down
- [ ] Display reallocation history per unit
- [ ] Show WG members per unit
- [ ] Implement reallocation UI (drag-and-drop or modal)
- [ ] Add search/filter for units
- [ ] Write frontend tests

**Acceptance Criteria:**
- [ ] Org structure tree renders correctly
- [ ] Can navigate hierarchy smoothly
- [ ] Reallocation history visible
- [ ] UI responsive (mobile-friendly)

**Testing:**
- [ ] Unit tests: React components
- [ ] Integration tests: full UI flow
- [ ] UI tests: responsiveness, accessibility

**Dependencies:** Sub-Issues 1.1, 4.1, 4.2

**Estimated Duration:** 3-4 days

**Team:** Frontend (1 person)

---

### Sub-Issue 4.4: Phase 4 Testing & Acceptance

**Title:** `[Phase 4.4] Phase 4 Testing, Performance & Acceptance`

**Description:**

Comprehensive testing of org hierarchy and reallocation functionality.

**Tasks:**
- [ ] Write Phase 4 integration tests (hierarchy, reallocation)
- [ ] Run performance tests (hierarchy queries, reallocation)
- [ ] Run load tests (large hierarchies, bulk reallocations)
- [ ] Run UI tests (dashboard navigation)
- [ ] Run stakeholder acceptance test
- [ ] Validate org security (no cross-org leaks)

**Acceptance Criteria:**
- [ ] All org hierarchy endpoints working
- [ ] Reallocation audit trail complete
- [ ] Dashboard fully functional
- [ ] No org data leaks
- [ ] Performance adequate
- [ ] Stakeholder acceptance obtained

**Testing:**
- [ ] Integration tests: 50+ hierarchy scenarios
- [ ] Performance: hierarchy queries < 200ms
- [ ] Load: 50 concurrent reallocations/sec
- [ ] Security: org isolation verified

**Dependencies:** Sub-Issues 4.1-4.3

**Estimated Duration:** 2-3 days

**Team:** QA (1 person)

---

## Phase 5: Reporting & Compliance (1 Week)

### Sub-Issue 5.1: Audit Trail Query Endpoints

**Title:** `[Phase 5.1] Implement Audit Trail Query Endpoints`

**Description:**

Implement comprehensive audit trail query endpoints for compliance reporting.

**Tasks:**
- [ ] Implement GET /audit-logs (list all audit events, paginated)
- [ ] Implement GET /audit-logs/entries/{entry_id} (all events for entry)
- [ ] Implement GET /audit-logs/by-user/{user_id} (all events by user)
- [ ] Implement GET /audit-logs/by-action (filter by action type)
- [ ] Add filtering: date range, user, action, entry
- [ ] Add sorting: date, user, action
- [ ] Write tests

**Acceptance Criteria:**
- [ ] Can query audit log by entry
- [ ] Can query audit log by user
- [ ] Can filter by date range & action
- [ ] Results paginated & sorted
- [ ] Immutability verified (cannot modify audit logs)

**Testing:**
- [ ] Unit tests: query logic
- [ ] Integration tests: full audit queries
- [ ] Security tests: immutability enforced

**Dependencies:** Sub-Issues 1.1, 2.2, 3.2, 4.1

**Estimated Duration:** 2-3 days

**Team:** Backend (1 person)

---

### Sub-Issue 5.2: Compliance & Allocation Reports

**Title:** `[Phase 5.2] Implement Compliance & Allocation Reports`

**Description:**

Implement reports for compliance verification and allocation accuracy.

**Tasks:**
- [ ] Implement GET /reports/approval-status (entries by status & approver)
- [ ] Implement GET /reports/allocation-by-unit (time allocation per unit)
- [ ] Implement GET /reports/allocation-by-project (time allocation per project)
- [ ] Implement GET /reports/overrides (Finance overrides with reasons)
- [ ] Implement GET /reports/budget-status (budget cap utilization)
- [ ] Add export to CSV/JSON
- [ ] Write tests

**Acceptance Criteria:**
- [ ] Reports accurate (100% data integrity)
- [ ] Reports performant (< 2 seconds)
- [ ] Reports exportable to CSV
- [ ] Date range filtering works
- [ ] Drill-down from summary to details

**Testing:**
- [ ] Unit tests: aggregation logic
- [ ] Integration tests: full report generation
- [ ] Accuracy tests: spot-check reports against source data

**Dependencies:** Sub-Issues 1.1, 5.1

**Estimated Duration:** 3-4 days

**Team:** Backend (1 person)

---

### Sub-Issue 5.3: Reporting Dashboards (UI)

**Title:** `[Phase 5.3] Build Reporting & Compliance Dashboards (UI)`

**Description:**

Build frontend dashboards for reporting and compliance visibility.

**Tasks:**
- [ ] Build approval status dashboard
- [ ] Build allocation heatmap (units vs. projects)
- [ ] Build Finance override log
- [ ] Build budget utilization tracker
- [ ] Implement date range picker
- [ ] Add export buttons (CSV, PDF)
- [ ] Write frontend tests

**Acceptance Criteria:**
- [ ] All dashboards render correctly
- [ ] Data refreshes correctly
- [ ] Date filtering works
- [ ] Export generates correct files
- [ ] UI responsive

**Testing:**
- [ ] Unit tests: React components
- [ ] Integration tests: full dashboard flow
- [ ] UI tests: responsiveness

**Dependencies:** Sub-Issues 1.1, 5.1, 5.2

**Estimated Duration:** 4-5 days

**Team:** Frontend (1 person)

---

### Sub-Issue 5.4: Performance Tuning & Production Readiness

**Title:** `[Phase 5.4] Performance Tuning, Load Testing & Production Readiness`

**Description:**

Final performance optimization, load testing, and production deployment readiness.

**Tasks:**
- [ ] Run comprehensive load tests (1000+ concurrent users)
- [ ] Profile & optimize slow queries
- [ ] Tune SurrealDB indexes & configurations
- [ ] Run stress tests (peak load scenarios)
- [ ] Prepare production deployment runbook
- [ ] Create monitoring & alerting setup
- [ ] Run final security audit
- [ ] Create disaster recovery plan

**Acceptance Criteria:**
- [ ] All endpoints < 200ms p95 latency
- [ ] Throughput adequate (100+ ops/sec)
- [ ] Load test passed (1000 concurrent users)
- [ ] Zero data loss scenarios tested
- [ ] Production deployment plan ready
- [ ] Monitoring & alerting configured

**Testing:**
- [ ] Load tests: 1000 concurrent users
- [ ] Stress tests: 3x peak load
- [ ] Failover tests: database recovery
- [ ] Security audit: OWASP top 10

**Dependencies:** Sub-Issues 5.1-5.3

**Estimated Duration:** 3-4 days

**Team:** QA (1 person) + DevOps (0.5 person) + Backend (0.5 person)

---

### Sub-Issue 5.5: Final Acceptance & Production Deployment

**Title:** `[Phase 5.5] Final Acceptance Testing & Production Deployment`

**Description:**

Final stakeholder acceptance and production deployment.

**Tasks:**
- [ ] Run full end-to-end acceptance test
- [ ] Stakeholder sign-off on all features
- [ ] Execute PostgreSQL→SurrealDB migration
- [ ] Run post-migration validation
- [ ] Monitor production for 48 hours
- [ ] Create knowledge base documentation
- [ ] Conduct team retrospective

**Acceptance Criteria:**
- [ ] All features accepted by stakeholders
- [ ] Zero data loss in migration
- [ ] Production performing as expected
- [ ] All alerts functioning
- [ ] Documentation complete

**Testing:**
- [ ] End-to-end UAT
- [ ] Production smoke tests
- [ ] Post-migration validation

**Dependencies:** All sub-issues from Phases 1-5

**Estimated Duration:** 5 days

**Team:** Full team + stakeholders

---

## Summary

**Total Sub-Issues:** 24

| Phase | Sub-Issues | Duration | Team | Notes |
|-------|-----------|----------|------|-------|
| 1 | 4 | 1 week | 1 Backend, 1 DevOps, 1 QA | Foundation |
| 2 | 5 | 1.5 weeks | 2 Backend, 1 QA | Core workflow |
| 3 | 4 | 1 week | 1 Backend, 1 DevOps, 1 QA | Financial controls |
| 4 | 4 | 1 week | 1 Backend, 1 Frontend, 1 QA | Org features |
| 5 | 5 | 1 week | 1 Backend, 1 Frontend, 1 DevOps, 1 QA | Reporting |

**Timeline:** 6 weeks total (all phases sequential)

**Team Allocation:**
- Backend: 2 people (full-time, all phases)
- Frontend: 1 person (Phases 4-5)
- DevOps: 1 person (Phases 1, 3, 5)
- QA: 1 person (all phases)

**Dependencies:**
- Phase 1 → Phase 2 (foundational)
- Phase 2 → Phase 3 (approval → financial)
- Phase 3 → Phase 4 (can run parallel but serial in plan)
- Phase 4 → Phase 5 (all features → reporting)
