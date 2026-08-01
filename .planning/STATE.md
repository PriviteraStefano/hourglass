---
gsd_state_version: 1.0
milestone: v0.1
milestone_name: MVP Consolidation
status: verifying
last_updated: "2026-08-01T06:44:01.622Z"
last_activity: 2026-08-01
progress:
  total_phases: 14
  completed_phases: 13
  total_plans: 61
  completed_plans: 60
  percent: 93
---

# Phase State

## Session

- **Last activity:** 2026-08-01
- **Completed:** Plan 10-06 (Working Groups surface — /working-groups list/search/create/edit/members against the live WG API; WorkingGroupsApis completed with 5 mutations; api<T> 204 helper fix; 5/5 e2e green; phase 10 complete, ready for verification)
- **Source:** `.planning/phases/10-information-architecture-implementation/10-06-PLAN.md`
- **Previous:** Plan 10-05 (Approvals queue — /approvals with stage-filtered Manager/Finance tabs; merged pending TE+expense queue with approve/reject; ListPending handler gate relaxed to admit WG manager/delegate via Service.IsWGManager; 4/4 approvals e2e green)
- **Intel:** `.planning/intel/`

## Phase 0: testing-foundation

- **Status:** Phase complete — ready for verification
- **Plans:**
  - 00-02-PLAN.md — Testcontainers infrastructure (Wave 1) [completed]
  - 00-01-PLAN.md — Auth bug fixes + cleanup (Wave 2, depends on 02) [completed]
  - 00-03-PLAN.md — Service test rewrite (Wave 3, depends on 01+02) [completed]
  - 00-04-PLAN.md — Handler test rewrite (Wave 4, depends on 03) [completed]
  - 00-05-PLAN.md — Bug buffer with human review (Wave 5, depends on 04) [completed]
  - 00-06-PLAN.md — E2E verification (Wave 6, depends on 05) [completed]
- **Last Activity:** 2026-06-09

## Phase 1: authorization

- **Status:** In progress
- **Goal:** Fix broken auth endpoints
- **Depends on:** Phase 0
- **Day:** Tue June 9
- **Plans:**
  - 01-01-PLAN.md — Backend auth fixes (Register cookies + password reset entropy) [completed]
  - 01-02-PLAN.md — Frontend auth integration (OrgSwitcher redirect API type) [completed]

## Phase 2: org-hierarchy

- **Status:** Complete
- **Goal:** Org tree visualization with ReactFlow
- **Depends on:** Phase 1
- **Day:** Wed-Thu June 10-11
- **Plans:**
  - 02-01-PLAN.md — Backend: delete protection + PUT member endpoint + batch members [completed]
  - 02-02-PLAN.md — Frontend: reparent mutation switch + pendingEdgeConnect cleanup [completed]
  - 02-03-PLAN.md — Frontend: "Make Primary" UI + subtree member groups [completed]

## Phase 3: customers

- **Status:** In progress
- **Goal:** Full customer CRUD
- **Depends on:** Phase 1
- **Day:** Wed-Thu June 10-11
- **Plans:**
  - 03-01-PLAN.md — Backend foundation: migration, search, internal customer support [completed]
  - 03-02-PLAN.md — Frontend core: API layer, sidebar, customer detail page [completed]
  - 03-03-PLAN.md — Frontend polish: internal badge, clickable cards, form lock, toast fix, tests [completed]

## Phase 4: contracts

- **Status:** In progress
- **Goal:** Contract CRUD with customer dropdown
- **Depends on:** Phase 3
- **Day:** Fri June 12
- **Plans:**
  - 04-01-PLAN.md — Backend: customer_id on create + HasProjects delete guard [completed]
  - 04-02-PLAN.md — Frontend: customer combobox + internal customer indicator [completed]

## Phase 5: projects

- **Status:** In progress
- **Goal:** Project CRUD with subprojects
- **Depends on:** Phase 4
- **Day:** Fri June 12
- **Plans:**
  - 05-01-PLAN.md — Backend: domain types, port interface, mocks, PG repo Update/Delete/HasActiveTimeEntries, service Update/Delete [completed]
  - 05-02-PLAN.md — Backend: HTTP handlers (Update/Delete/ListSubprojects) + route wiring [completed]
  - 05-03-PLAN.md — Frontend: types, API mutations/queries, EditProjectDialog, detail page wiring [completed]
  - 05-04-PLAN.md — Tests: service tests + handler tests + build verification [completed]

## Phase 6: time-entries-and-expenses

- **Status:** In progress
- **Goal:** Full CRUD + approval workflow
- **Depends on:** Phase 5
- **Day:** Sat-Sun June 13-14
- **Plans:**
  - 06-01-PLAN.md — Backend foundation: domain models, port interfaces, mocks, factories, migrations [completed]
  - 06-02-PLAN.md — Backend service layer: TimeEntryService two-stage approval + ExpenseService CRUD/approval [completed]
  - 06-03-PLAN.md — PG repositories extend + HTTP handlers + route wiring [completed]
  - 06-04-PLAN.md — Frontend time entry rewrite: flat model, client-side calendar, approval components [completed]

## Phase 7: exports

- **Status:** Complete
- **Goal:** Downloadable CSV/Excel exports
- **Depends on:** Phase 6
- **Day:** Sun June 14
- **Plans:**
  - Frontend: Exports page with date range + type selector [completed]
  - 07-01-PLAN.md — Backend export extensions (count endpoints, XLSX, format params, filter helpers) [completed]
  - 07-02-PLAN.md — Frontend core: useDownload hook, ExportApis, ExportForm, exports page rewrite, sidebar [completed]
  - 07-03-PLAN.md — Frontend: Export tabs on time entries and expenses pages [completed]

## Superseded Phases

The following phases from the previous milestone structure are superseded:

- Phase 1 (org-hierarchy-edge-driven) — Superseded by Phase 2
- Phase 2 (customers-management-page) — Superseded by Phase 3
- Phase 3 (contracts-add-projects-display) — Superseded by Phase 4
- Phase 4 (integrate-customers-into-contracts) — Superseded by Phase 4
- Phase 5 (mvp-consolidation-seed) — Delivered
- Phase 6 (api-audit) — Superseded by Phase 1 auth verification
- Phase Pg-1 (foundation) — Complete, archived
- Phase Pg-2 (postgres-adapters) — Complete, archived
- Phase Pg-3 (wiring) — Complete, archived

## Decisions

- **2026-06-08:** testcontainers-go v0.42.0 selected as integration test infrastructure, replacing DATABASE_URL-dependent TestPool with SetupPackageContainer using sync.Once container lifecycle. Migration paths resolve relative to Go module root.
- **2026-06-09:** Token generation guarded behind orgID != uuid.Nil to avoid FK violation on refresh_tokens when registering without an organization
- **2026-06-09:** Added crypto/rand.Int with math/big for unbiased password reset code distribution (replacing modulo-biased rand.Read)
- **2026-06-09:** Register returns 200 instead of 201 to match Login/Bootstrap convention (response now includes tokens)
- **2026-06-09:** OrgSwitcher self-fetches memberships via useSuspenseQuery instead of receiving organizations prop — follows ProfileMenu self-contained component pattern
- **2026-06-09:** Full cache clear (queryClient.clear() + invalidateQueries) on org switch ensures no stale data from previous org context
- **2026-06-09:** Landing page uses Navigate redirect to /time-entries — minimal approach per D-07, no Dashboard page in v0.1

- **2026-06-11:** CustomerID on CreateContractRequest uses *uuid.UUID (nullable pointer) for domain, *string for HTTP handler (JSON-native), parsed at handler boundary
- **2026-06-11:** HasProjects counts ALL projects (not just active) — consistent with ON DELETE RESTRICT FK constraint
- **2026-06-11:** HasProjects check runs after HasTimeEntries check in Delete service method

- **2026-07-08:** Export tabs follow PATTERNS.md exactly: Tabs defaultValue='list', List/Calendar/Export triggers, existing calendar content preserved under Calendar tab. List tab is empty placeholder — no list view component exists yet on either page.
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Refresh-token replay of any rotated OR revoked token revokes the entire token family (strict reuse model, per audit P0-5)

- [Phase 08-pre-deployment-hardening-p0-audit-fixes] (08-02): Expense amounts render currency resolved from the existing project→contract relationship; the expense payload carries no currency field (no new endpoint invented)
- [Phase 08-pre-deployment-hardening-p0-audit-fixes] (08-02): approved status badge recolored emerald so all six workflow states are visually distinct (was blue, colliding with submitted)
- [Phase 08-pre-deployment-hardening-p0-audit-fixes] (08-02): List filter state is URL-shareable via validateSearch (ADR-FE-017); arrays accept single, repeated, and JSON-serialized forms
- [Phase 08-pre-deployment-hardening-p0-audit-fixes] (08-02): Row click / New-entry affordance switch to the calendar tab and set the date search param, reusing the existing EntryDetail/ExpenseDetail surfaces
- [Phase 08-pre-deployment-hardening-p0-audit-fixes] (08-02): Error recovery uses errorComponent with router.invalidate() (not reset) so loaders re-run — TanStack Router v1 semantics
- [Phase 08-pre-deployment-hardening-p0-audit-fixes] (08-02): Customers e2e suite logs in once via API and injects cookies to stay under the backend 5/min anonymous login rate limit
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Race-loser semantics kept as locked in 08-01 (strict reuse model): the concurrent-refresh loser is indistinguishable from an attacker replay and revokes the family; tests assert exactly-one-success + ErrTokenReuse and document T9 as out of scope
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: ANONYMOUS_RATE_LIMIT env knob added for the outer route rate limiter (default 20/min unchanged) so full e2e suites can run; e2e runs raise RATE_LIMIT + ANONYMOUS_RATE_LIMIT
- [Phase 08]: Leaf-level errorComponent on the data routes (time-entries/expenses/customers index) with the layout boundary kept as fallback — layout matches persist across navigations and hold loader errors that navigation/invalidate intermittently fail to clear in TanStack Router v1.170 (stale panel / empty main after recovery)
- [Phase 08]: E2E seeding convention: shared helpers module, underscore-free seed email domains (native input[type=email] validation silently blocks submit otherwise), '' for optional string columns (pgx rejects NULL for *string scans)
- [Phase 09-activity-ontology]: Migration numbered 011 not 010 — 010_refresh_token_reuse_detection already exists; per ADR-BE-004 new files continue from the max — Migration numbered 011 not 010 — 010_refresh_token_reuse_detection already exists; per ADR-BE-004 new files continue from the max
- [Phase 09-activity-ontology]: budget_caps.project_id rewritten to activity_id — plan omitted it but its FK blocked DROP TABLE projects — budget_caps.project_id rewritten to activity_id — plan omitted it but its FK blocked DROP TABLE projects
- [Phase 09-activity-ontology]: Same-id migration strategy — activity rows keep old project/subproject UUIDs so down restores 1:1 — Same-id migration strategy — activity rows keep old project/subproject UUIDs so down restores 1:1
- [Phase 09-activity-ontology]: NULL-project expenses get per-org internal General & Admin fallback activity so activity_id is NOT NULL (D-4) — NULL-project expenses get per-org internal General & Admin fallback activity so activity_id is NOT NULL (D-4)
- [Phase 09-activity-ontology]: Migration numbered 012 not 011 (011 taken by activity ontology; ADR-BE-004 max+1) — Migration numbered 012 not 011 (011 taken by activity ontology; ADR-BE-004 max+1)
- [Phase 09-activity-ontology]: Down migration downgrades existing hr rows to employee before restoring role CHECK (rollback must not fail on violating rows) — Down migration downgrades existing hr rows to employee before restoring role CHECK (rollback must not fail on violating rows)
- [Phase ?]: Activity domain follows repo subdirectory convention: domain/activity/activity.go (plan flat path mapped); BudgetAmount as *float64 not *decimal (no decimal lib; codebase money is float64); working_group domain keeps legacy SubprojectID field name mapped to activities.activity_id (services not in 04/05 file lists); contract/export commercial-chain queries rewritten as activity-tree CTEs (Rule 3, old SQL hit dropped projects table); financial_cutoff_period queries live in entry repos IsPeriodLocked, updated in place to org+activity+date
- [Phase 09-activity-ontology]: ErrActivityNotLoggable sentinel lives in the activity domain (single source shared by both entry services) — Plan requires the same sentinel across both entry services; declared once in domain/activity per ADR-BE-001
- [Phase 09-activity-ontology]: Manager-stage approver set = WG row ManagerID + DelegateIDs; Approve re-resolves it and verifies membership — R-1 defines the manager stage as WG manager+delegates; Approve-side membership check makes routing enforceable (Rule 2)
- [Phase 09-activity-ontology]: Terminal unit-tree case (org root without manager) stays role-gated per ADR-BE-014 consequences — Service cannot pin an org-role manager user; handler role resolution governs that terminal state
- [Phase 09-activity-ontology]: KindExists port method added for D-2 kind-catalog validation on Create — Plan requires Create to validate kind against the org catalog; Plan-03 port had no way to express it (Rule 2)
- [Phase 09-activity-ontology]: Backend routes registered as /activities (no /api prefix) matching the Vite-proxy-strips-/api convention — the plan's /api/activities wording is the frontend-facing path
- [Phase 09-activity-ontology]: ListKinds added to ActivityRepository port/service/adapter/mock for GET /api/activity-kinds (D-2 catalog)
- [Phase 09-activity-ontology]: Detail composition (GetAncestry/ResolveCommercialContext/ResolveBillability) lives in the handler via the repo port per 09-04 heads-up
- [Phase 09-activity-ontology]: activity_name + activity_kind joined display fields on entry domain types: LEFT JOIN for reads, scalar-subquery RETURNING for Create/Update (data-modifying CTE rows invisible to base-table reads)
- [Phase 09-activity-ontology]: Adopt + manager-management endpoints intentionally not exposed on the activity HTTP surface (plan endpoint list omits them); service methods remain
- [Phase 10-information-architecture-implementation]: Billable checkbox in activity create/edit dialogs: checked = true, unchecked = undefined (inherit) - tri-state omitted per plan
- [Phase 10-information-architecture-implementation]: Activity detail shows derived commercial context + resolved billability from ActivityDetail payload; adoption count renders "-" (detail endpoint carries no adoption_count)
- [Phase 10-information-architecture-implementation]: Entry creation cascade simplified to activity -> children -> working-group -> unit; backend create needs only activity_id + unit_id
- [Phase 10]: Sidebar groups render from a single declarative navStructure filtered by pure role-matrix predicates; visibility logic never lives inline in JSX — ADR-P-011 D-1/D-5; keeps the matrix testable and the render loop dumb
- [Phase 10]: Approval stages derived client-side from route-context profile + GET /working-groups (manager_id/delegate_ids); hr stripped per ADR-P-008 D-4 — RESEARCH 2.1: no new backend endpoint needed; UX scoping only, backend stays authoritative
- [Phase 10]: workingGroupsQueryOpts named export + WorkingGroupsApis object; 60s staleTime keyed ['working-groups'] as the stable home for Plan 10-06 mutations — Satisfies the acceptance contract and the established API-layer object convention
- [Phase 10-information-architecture-implementation] (10-03): Layout barrel index.ts created so @/components/layout resolves (task 1 acceptance criterion); exports Header/Body only — minimal, no behavior change
- [Phase 10-information-architecture-implementation] (10-03): Tab triggers (List/Calendar/Export) stay inside Body, not moved to Header — the binding acceptance criterion 'existing control tree unmodified apart from being wrapped' + threat T-10-03-1 override the action text's 'tab triggers that fit the band'; moving TabsList out of the Tabs root would break base-ui Tabs context wiring and constitute re-layout
- [Phase 10-information-architecture-implementation] (10-03): Shell page titles use UI-SPEC Heading style (text-xl font-semibold) in the Header on every wrapped page
- [Phase 10-information-architecture-implementation] (10-03): Pages with no prior padding (exports/contracts/customers/activities) get the UI-SPEC lg 24px inner container (p-6) inside Body; time-entries/expenses keep their existing p-2 (plan: keep existing padding container as-is)
- [Phase 10-information-architecture-implementation] (10-03): All wrapped pages get the h-full overflow-y-auto inner scroll container inside Body (UI-SPEC: long content scrolls inside Body, window never scrolls)
- [Phase 10-information-architecture-implementation] (10-03): Leaf-level errorComponent preserved verbatim per plan instruction; in-page pending/error UI (activity-detail loading/not-found) now renders inside Body within the shell frame (threat T-10-03-2)
- [Phase 10-information-architecture-implementation] (10-03): Contracts typecheck rot fixed inline (deferred-items assigns those 4 errors to 10-03): currency v??undefined, Customer.company_name, navigate search:{from:'owned'}, Select onChange wrapper
- [Phase 10]: In-Body error state renders the locked copy 'We couldn't load Today. {reason}. Try again.' with router.invalidate() recovery; RouteError stays registered as the leaf boundary
- [Phase 10]: ISO-week filter compares entry_date.slice(0,10) against date-fns startOfWeek/endOfWeek bounds (string compare, timezone-immune, matches list-view convention)
- [Phase 10]: Links to /approvals typed as ToPathOption (route lands in 10-05); typed Link to would not compile
- [Phase 10]: Pending endpoints not called for employees/HR proven by msw request capture in unit tests (enabled gate), not just inspection
- [Phase 10]: Expense preview values render amount.toFixed(2) in .font-text (no hours on expenses); time entries render hours
- [Phase 10] (10-05): ListPending handler gate relaxed (T-10-05-3) — org-role manager/finance OR WG manager/delegate admitted via Service.IsWGManager + role wg_manager; the repo's existing wg_manager branch (WG-scoped on manager_id/delegate_ids) serves the queue; Approve/Reject service gates untouched (backend authoritative, T-10-05-1)
- [Phase 10] (10-05): Stage tabs render only from deriveApprovalStages output; single-stage users skip the tab bar and see their queue directly; stage derived from row status (pending_finance → finance; submitted/pending_manager → manager) mirroring BE-014
- [Phase 10] (10-05): 403 while tabs render → locked error state 'We couldn't load Approvals. {reason}. Try again.' with router.invalidate(); never 'Queue is clear' (T-10-05-2)
- [Phase 10] (10-05): URL-shareable stage via validateSearch (ADR-FE-017): /approvals?stage=finance
- [Phase 10-information-architecture-implementation]: Working Group frontend type keeps subproject_id (legacy activity anchor) — the plan's 'activity_id' name conflicts with the live API JSON key; the 10-02 type mirrors the Go payload exactly and is untouched — read_first verification against working_group.go; the API returns subproject_id for the activity anchor

## Current Position

Phase: 10 (information-architecture-implementation) — EXECUTING
Plan: 6 of 6
Status: Phase complete — ready for verification
Last activity: 2026-08-01 -- Completed quick task 260801-luy: Document demo deployment topology (Compose + Caddy + cloudflared) — ADR-BE-015, openwiki ops, deploy/demo artifacts
Next up: Phase 10 (information-architecture-implementation) — COMPLETE, 6 plans, 3 waves (ready for verification)

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260801-got | Fix collapsed-sidebar hover/click dead zone over icons | 2026-08-01 | 54f465a | [260801-got-investigate-sidebar-collapsed-mode-hover](./quick/260801-got-investigate-sidebar-collapsed-mode-hover/) |
| 260801-luy | Document demo deployment topology (Compose + Caddy + cloudflared) — ADR-BE-015 + deploy/demo artifacts | 2026-08-01 | f7637d0 | [260801-luy-document-demo-deployment-topology-compos](./quick/260801-luy-document-demo-deployment-topology-compos/) |

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 00-testing-foundation P02 | 4 min | 3 tasks | 5 files |
| Phase 00-testing-foundation P01 | 131min | 3 tasks | 11 files |
| Phase 00-testing-foundation P03 | 5 min | 3 tasks | 10 files |
| Phase 00-testing-foundation P04 | 42 min | 3 tasks | 6 files |
| Phase 00-testing-foundation P05 | 23 min | 3 tasks | 14 files |
| Phase 00-testing-foundation P06 | 20 min | 2 tasks | 7 files |
| Phase 01-authorization P01 | 3 min | 2 tasks | 3 files |
| Phase 01-authorization P02 | 2 min | 2 tasks | 4 files |
| Phase 01-authorization P02 | 2 min | 2 tasks | 4 files |
| Phase 03-customers P02 | 2 min | 2 tasks | 5 files |
| Phase 04-contracts P01 | 2 min | 5 tasks | 7 files |
| Phase 04-contracts P02 | 1 min | 3 tasks | 3 files |
| Phase 05-projects P01 | - | 3 tasks | 5 files |
| Phase 05-projects P03 | 5 min | 4 tasks | 5 files |
| Phase 05-projects P04 | 3 min | 3 tasks | 2 files |
| Phase 06-time-entries-expenses P01 | 10 min | 2 tasks | 10 files |
| Phase 06-time-entries-expenses P02 | 45 min | 3 tasks | 6 files |
| Phase 06-time-entries-expenses P03 | 25 min | 3 tasks | 12 files |
| Phase 06-time-entries-expenses P04 | 15 min | 3 tasks | 10 files |
| Phase 06-time-entries-expenses P05 | 12 min | 3 tasks | 10 files |
| Phase 07-exports P01 | 3 min | 3 tasks | 10 files |
| Phase 07-exports P02 | 3 min | 3 tasks | 5 files |
| Phase 07-exports P03 | 1 min | 2 tasks | 2 files |
| Phase 07-exports P03 | 1 min | 2 tasks | 2 files |
| Phase 08-pre-deployment-hardening-p0-audit-fixes P08-01 | 40min | 2 tasks | 21 files |
| Phase 08-pre-deployment-hardening-p0-audit-fixes P08-02 | 93 | 5 tasks | 24 files |
| Phase 08-pre-deployment-hardening-p0-audit-fixes P08-03 | 29min | 3 tasks | 8 files |
| Phase 08-pre-deployment-hardening-p0-audit-fixes P08-03 | 29min | 3 tasks | 8 files |
| Phase 08 P08-04 | 176min | 4 tasks | 11 files |
| Phase 09-activity-ontology P09-01 | 7min | 2 tasks | 4 files |
| Phase 09-activity-ontology P02 | 10min | 2 tasks | 4 files |
| Phase 09-activity-ontology PP09-03 | 20min | 3 tasks | 23 files |
| Phase 09-activity-ontology P09-04 | 7min | 3 tasks | 16 files |
| Phase 09-activity-ontology PP09-05 | 17min | 3 tasks | 18 files |
| Phase 10-information-architecture-implementation P01 | 37min | 3 tasks | 27 files |
| Phase 10 P02 | 9min | 3 tasks | 6 files |
| Phase 10-information-architecture-implementation P03 | 9min | 3 tasks | 10 files |
| Phase 10-information-architecture-implementation P04 | 17min | 3 tasks | 6 files |
| Phase 10-information-architecture-implementation P05 | 48min | 4 tasks | 10 files |
| Phase 10-information-architecture-implementation P06 | 453 | 4 tasks | 10 files |
