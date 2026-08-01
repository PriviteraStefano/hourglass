# Milestones

## v0.1 MVP Consolidation (Shipped: 2026-08-01)

**Phases completed:** 16 phases, 62 plans, 138 tasks

**Known deferred items at close:** 9 (see STATE.md Deferred Items — 4 UAT gaps, 3 verification gaps, 2 quick tasks)

**Delivered:** Full PostgreSQL time/expense tracking with two-stage approval workflows, activity ontology, working groups, approvals queue, exports, and P0 hardening.

**Key accomplishments:**

- Full auth cleanup: 4 known bugs fixed, refresh token rotation, cookie name unification, password reset hardening, rate limiting, mock repairs, and 6 testcontainer-backed integration tests
- testcontainers-go infrastructure — fully self-contained PostgreSQL container per package, replacing DATABASE_URL-dependent TestPool
- Service-layer integration tests running against real testcontainers-backed PostgreSQL — 42 passing subtests across 9 core service packages
- Handler integration tests rewritten to use testcontainers-backed PostgreSQL with shared handlerFixture, 54 passing integration subtests across all 8 handler families
- Fixed 12+ documented bugs across test files, schema, and production code — full test suite green with zero skipped tests
- E2E verification of full app stack (Go backend + PostgreSQL + React frontend) via Playwright — 16/19 tests passing, all 16 Go package suites green
- Create Playwright E2E test files covering CRUD flows for time entries, projects, contracts, customers, and org hierarchy.
- Fixed 7 categories of pre-existing test failures and code bugs — full green test suite across 30 Go packages and 51 Vitest tests, completing Phase 0 testing foundation
- Register endpoint now sets auth_token/refresh_token cookies matching Login/Bootstrap, password reset codes use unbiased crypto/rand.Int distribution
- Real org memberships in OrgSwitcher with switch mutation, `/` redirect to `/time-entries`, password reset API type cleanup
- 01-authorization
- Backend foundation for primary unit designation (PUT /units/{id}/members/{membershipId}) and delete protection enforcement (root/children/members) with all supporting port, repository, service, and handler layers plus integration tests
- Dedicated reparent mutation with clean contract, dead state removal across all consumers
- Frontend type, API mutation, and UI for "Make Primary" unit member designation and expandable sub-unit member browsing in the side panel
- DB migration + ILIKE search + CreateInternalCustomer + company_name lock + auto-creation on org creation — full backend foundation for Phase 3 internal customer support.
- Customer API module updated with `is_internal` and backend search, sidebar link added, client-side filter removed, customer detail page at `/customers/:id` with linked contracts.
- 2026-06-11
- Add customer combobox to CreateContractDialog with search, "No customer" option, and "(Internal)" suffix for internal customers, plus updated test coverage.
- Wire HTTP Update/Delete/ListSubprojects handler methods + subproject repository injection + route registration in main.go
- EditProjectDialog forked from CreateProjectDialog with pre-populated fields, delete confirmation with inline 409 error handling, and expandable subproject section with lazy fetch.
- ✅ Complete
- 1. [Rule 2 - Missing testability] MockExpenseRepo.CreateApproval didn't track approvals
- Extended TimeEntryRepository with CurrentApproverRole/SubmittedAt scanning and role-differentiated ListPending. Rewrote ExpenseRepository from models.Expense to domainexpense.Expense with dynamic List query builder, ListPending, IsPeriodLocked, and CreateApproval. Created ExpenseService (full CRUD + two-stage approval) and Expense HTTP handler (10 endpoints). Updated TimeEntryHandler for manager/finance roles. Registered all expense routes in main.go.
- Flattened TimeEntry types, client-side calendar statuses, flat entry list components, and shared approval UI components.
- Full expense frontend UI with calendar-based navigation, CRUD forms, receipt upload, and approval workflow using shared components — includes type definitions, API module, TanStack Router route, calendar+detail page layout, and inline create/edit form.
- Backend export extensions: count endpoints, XLSX generation via excelize, format param switching, filter param helpers, and auth-gated count route wiring
- Shared download hook, export API module, reusable ExportForm component, combined export page rewrite, and sidebar navigation — all 3 tasks committed atomically
- Export tabs added to time entries and expenses pages — [List] [Calendar] [Export] using shadcn Tabs with @base-ui/react primitives
- 2026-06-11
- Refresh-token reuse detection with family revocation (atomic single-transaction rotation, replay → 401 + both cookies cleared) and handler-boundary string length caps (audit S3) across all write handlers
- /customers index route, filterable list views for time entries and expenses, and route-level error boundaries — closing P0-2, P0-3, and P0-4 with shared table/filter/badge components and 24 new tests (Vitest + Playwright)
- Reuse-detection regression suites (unit + real-PG repo/service/handler, incl. a deterministic concurrent-race test) proving the P0-5 layer stays dead, an S3 length-cap boundary table over every limit class, and an E2E auth spec proving the silent-refresh cookie rotation is observable in the browser with replay → 401
- Browser-proven list views, customers route, and error recovery — closing the audit's P0 gate: all six P0 rows now read Fixed, with a leaf-level errorComponent fix for an intermittent recovery stuck-state and 24 new E2E assertions across four suites
- Big-bang schema rewrite replacing projects/subprojects with the recursive `activities` table (ADR-P-007 D-1…D-8, ADR-BE-014 R-5): all six FK rewrites land in one atomic migration, MVP seed data migrates zero-orphan, and a bidirectional 011 up/down pair with a testcontainers cycle test proves up → down → up is clean.
- Additive staffing schema per ADR-P-008 (D-1, D-1a, D-2, D-4): the `availability_windows` table with typed absence kinds, three nullable membership-validity DATE columns, and the `hr` org role added to the role CHECK — one bidirectional 012 migration pair, zero FK coupling to the 011 activity ontology, proven by a testcontainers up/down/up cycle test and live-PostgreSQL behavioral checks.
- Projects and subprojects collapse into one recursive `Activity` entity end-to-end: `Activity` domain type with `ActivityKind` string catalog (D-2), one `ActivityRepository` port + PG adapter (R-6) whose `GetAncestry`/`ResolveCommercialContext`/`ResolveBillability` walk `parent_id` upward as recursive CTEs (D-3/D-7 derived-never-stored), and both entry repositories rewritten onto `activity_id` + `unit_id` with `ListPending` manager stage resolving through the activity → WG → manager/delegate chain (R-1) — proven by testcontainers integration tests.
- Approval routing moves from pinned FKs to the activity chain end-to-end: a new ActivityService (CRUD + D-2/D-3 validation + sentinel-guarded Delete) replaces the collapsed project service, and both time-entry and expense services now resolve the manager stage through activity → anchored WG → manager/delegate with the unit-manager fallback for personal activities (R-2), ErrActivityNotLoggable enforcement for commercial-no-WG (409), and the D-11 skip extended to delegates (R-3) — with Approve verifying the actor against the re-resolved approver set so routing is enforceable at approval time, not just submission.
- The backend API surface is fully activity-shaped: a new ActivityHandler exposes List/Create/detail-Get (ancestry + commercial context + resolved billable)/Update/Delete/ListChildren/activity-kinds under `/activities` + `/activity-kinds` replacing the deleted project+subproject handlers, both entry handlers and repositories are rewritten onto the required, subtree-resolving `activity_id` FK with joined `activity_name`/`activity_kind` display fields and `ErrActivityNotLoggable`→409 mapping, and the router (`cmd/server/main.go` + both test fixtures) is rewired so `go build ./...`, `go vet ./...`, and the full test suite are green — the frontend rename (Phase 10) is now a pure rename against stable endpoints.
- Forward migration 013 relabels subproject-derived activities from `kind='task'` to `kind='phase'` per SPEC acceptance #6, and the 011 migration cycle test now asserts the corrected distribution (engagement 6 / phase 6 / task 0 / internal 1), closing VERIFICATION gap 1 without violating the ADR-BE-004 append-only migration rule.
- Re-seeded `seedWGData` in the working_group integration test onto the activity ontology schema (activity_kinds + activities), closing VERIFICATION GAP 2 — the package no longer touches the migration-011-dropped projects/subprojects tables and all 4 subtests pass against the real activity-backed schema
- ErrActivityCycle sentinel + shared service-layer validateParent (Get + same-org + GetAncestry path check) enforced on both Create and Update, closing VERIFICATION GAP 3 and the latent Update same-org gap, with 7 new unit tests and a 400 handler mapping
- Frontend rebuilt onto the Phase 9 activity ontology: Activity types + ActivitiesApis API layer replace the dead projects surface, /projects routes renamed to /activities with activity vocabulary, entry pickers rewired to activity_id, and the projects e2e suite replaced by activities e2e with recursive-activity seeding
- ADR-P-011 D-1 pillar groups (Today/Track/Work/People/Economics/Review/Reports/Admin) replace the mechanic-named Tracking/Management sidebar, with D-5 role-scoped visibility computed by pure, unit-tested helpers deriving approval stages from the route-context profile + WG manager_id/delegate_ids payloads
- Header+Body page shell applied to all nine carried-over authenticated page roots (time-entries, expenses, exports, contracts index+detail, customers index+detail, activities index+detail) — single h1 per page in the shell's 48px Header band, content wrapped in Body with inner scroll container, and the 4 contracts typecheck errors assigned to this plan resolved (baseline rot 10 → 6)
- The `/` landing is now the Today page — a read-only composition of "Waiting on you" (approval-stage gated) + "Your week" (current-ISO-week draft/submitted/rejected entries) with locked empty states for new users and all-caught-up users, replacing the `<Navigate to="/time-entries" />` redirect
- `/approvals` is live: a single page rendering stage-filtered Manager/Finance queues of merged pending time entries + expenses, with reason-required reject and approve round-trips against the live backend — and the backend ListPending gate now admits WG manager/delegate approvers whose org role is employee (T-10-05-3), closing the employee-with-WG-stage 403 gap
- Working Groups top-level surface under Work — /working-groups list with client-side search, create/edit dialog (activity + manager + delegate chips), member management dialog distinct from the approver set, and destructive delete, all against the live Phase 9 WG API with a 5/5-green e2e suite
- PostgreSQL adapter foundation — schema correction, shared sentinel errors, new port interfaces, wrapPGError helper, test infrastructure (TestPool/SetupTestSchema), and UserFinder implementation via pgx
- PostgreSQL implementations of UserRepository (11 methods), OrganizationRepository (4 methods), OrganizationMembershipRepository (2 methods), and OrganizationManagementRepository (10 methods) with full test coverage.
- ProjectRepository
- TimeEntryRepository
- Postgres repo wiring in cmd/server/main.go, adapter files for TokenService/PasswordHasher, CORS middleware extraction, and auth test rewrite — zero SurrealDB imports remain in the server entrypoint.
- Deleted all SurrealDB source files (4 paths, ~7300 lines), cleaned docker-compose.yml, updated AGENTS.md/Makefile/go.mod, and removed the surrealdb.go dependency with `go mod tidy` — verified by `go build ./...` and `go vet ./...`
- Automated smoke test for Postgres-wired server with health check, auth flow, and authenticated data access verification

---
