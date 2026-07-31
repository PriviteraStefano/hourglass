---
gsd_state_version: 1.0
milestone: v0.1
milestone_name: MVP Consolidation
status: executing
last_updated: "2026-07-31T10:51:28.804Z"
last_activity: 2026-07-31
progress:
  total_phases: 12
  completed_phases: 10
  total_plans: 47
  completed_plans: 43
  percent: 83
---

# Phase State

## Session

- **Last activity:** 2026-07-31
- **Completed:** Plan 08-01 (Backend security hardening: refresh-token reuse detection + input caps)
- **Source:** `.planning/phases/08-pre-deployment-hardening-p0-audit-fixes/08-01-PLAN.md`
- **Previous:** Plan 07-03 (Export tabs on time entries and expenses pages)
- **Intel:** `.planning/intel/`

## Phase 0: testing-foundation

- **Status:** Ready to execute
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

## ## Decisions

- **2026-06-08:** testcontainers-go v0.42.0 selected as integration test infrastructure, replacing DATABASE_URL-dependent TestPool with SetupPackageContainer using sync.Once container lifecycle. Migration paths resolve relative to Go module root.
- [Phase 00-testing-foundation]: ---
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
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Refresh-token replay of any rotated OR revoked token revokes the entire token family (strict reuse model, per audit P0-5) — Refresh-token replay of any rotated OR revoked token revokes the entire token family (strict reuse model, per audit P0-5)
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Concurrent-refresh race: second simultaneous refresh of the same token is treated as replay and revokes the family — documented in repo/test comments; client fingerprinting out of scope (audit T9) — Concurrent-refresh race: second simultaneous refresh of the same token is treated as replay and revokes the family — documented in repo/test comments; client fingerprinting out of scope (audit T9)
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Rotation is one repo-level transaction (pgx BeginTx + FOR UPDATE): mark rotated_at + insert successor atomically; no crash window — Rotation is one repo-level transaction (pgx BeginTx + FOR UPDATE): mark rotated_at + insert successor atomically; no crash window
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: FindByHash now returns only non-rotated AND non-revoked tokens (tombstone-model semantic) — FindByHash now returns only non-rotated AND non-revoked tokens (tombstone-model semantic)
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Input length caps live at the HTTP adapter boundary as rune counts; domain value objects keep format validation only — Input length caps live at the HTTP adapter boundary as rune counts; domain value objects keep format validation only
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Refresh-token replay of any rotated OR revoked token revokes the entire token family (strict reuse model, per audit P0-5) — Refresh-token replay of any rotated OR revoked token revokes the entire token family (strict reuse model, per audit P0-5)
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Concurrent-refresh race: second simultaneous refresh of the same token is treated as replay and revokes the family - documented in repo/test comments; client fingerprinting out of scope (audit T9) — Concurrent-refresh race: second simultaneous refresh of the same token is treated as replay and revokes the family - documented in repo/test comments; client fingerprinting out of scope (audit T9)
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Rotation is one repo-level transaction (pgx BeginTx + FOR UPDATE): mark rotated_at + insert successor atomically; no crash window — Rotation is one repo-level transaction (pgx BeginTx + FOR UPDATE): mark rotated_at + insert successor atomically; no crash window
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: FindByHash now returns only non-rotated AND non-revoked tokens (tombstone-model semantic) — FindByHash now returns only non-rotated AND non-revoked tokens (tombstone-model semantic)
- [Phase 08-pre-deployment-hardening-p0-audit-fixes]: Input length caps live at the HTTP adapter boundary as rune counts; domain value objects keep format validation only — Input length caps live at the HTTP adapter boundary as rune counts; domain value objects keep format validation only

## Current Position

Phase: 08 (pre-deployment-hardening-p0-audit-fixes) — EXECUTING
Plan: 2 of 4
Status: Ready to execute
Last activity: 2026-07-31 -- Phase 08 execution started

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
