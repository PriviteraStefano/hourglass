# Hourglass Roadmap

## Milestones

- ✅ **v0.1 MVP Consolidation** — Phases 0-10 (shipped 2026-08-01)
- ✅ **v0.2 Ontology Extension — Origins, Tickets & Coverage + Direction** — Phases 11-16 (shipped 2026-08-25; phases 17-26 cancelled unbuilt)
- 🚧 **v0.2.1 Contract-first presentation — job clusters** — Phases 17-20 (in progress; job-cluster implementation inserted after Phase 20)

## Phases

<details>
<summary>✅ v0.1 MVP Consolidation (Phases 0-10) — SHIPPED 2026-08-01</summary>

- [x] Phase 0: Testing foundation (6/6 plans) — testcontainers-go, auth bug fixes, PG-backed test rewrite, E2E verification
- [x] Phase 1: Authorization (3/3 plans) — backend auth fixes, frontend auth integration, E2E verified
- [x] Phase 2: Org Hierarchy (3/3 plans) — ReactFlow tree, unit CRUD, member management, edge-driven reparenting, delete protection
- [x] Phase 3: Customers (3/3 plans) — full CRUD, internal customers, search, delete protection
- [x] Phase 4: Contracts (2/2 plans) — customer combobox, HasProjects delete guard
- [x] Phase 5: Projects (4/4 plans) — Update/Delete/ListSubprojects, EditProjectDialog — superseded by activities (Phase 9)
- [x] Phase 6: Time Entries + Expenses (5/5 plans) — flat model, calendar UI, two-stage approval workflow
- [x] Phase 7: Exports (3/3 plans) — CSV/XLSX, count endpoints, export tabs
- [x] Phase 8: Pre-Deployment Hardening (4/4 plans) — P0 audit gate closed, refresh-token reuse detection, list views, error boundaries
- [x] Phase 9: Activity Ontology (8/8 plans) — projects/subprojects → recursive activities, routing rewrite, staffing schema
- [x] Phase 10: Information Architecture (6/6 plans) — activity surface, pillar sidebar, page shell, Today landing, Approvals queue, Working Groups

</details>

<details>
<summary>✅ v0.2 Ontology Extension (Phases 11-16) — SHIPPED 2026-08-25</summary>

- [x] Phase 11: Foundations (8/8 plans) — origins + sold_hours + tickets backend; ADR-P-003 rev + P-013 (completed 2026-08-07)
- [x] Phase 12: Coverage Backend (7/7 plans) — allocation ledger, funding sources, to-cover queue, snapshots; leftover smokes in 16-01-SMOKE.md (completed 2026-08-08)
- [x] Phase 13: Direction Backend (10/10 plans) — plan plane, claim model, org policy, coverage read-model (completed 2026-08-08)
- [x] Phase 14: Availability Backend (11/11 plans) — absences + capacity (completed 2026-08-12)
- [x] Phase 15: UX Foundation (4/4 plans) — design tokens + shared components + sketch-loop contract (completed 2026-08-24)
- [x] Phase 16: Integrity Repair (1/1 plans) — own-coverage read, expense unit_id + receipt auth, WR-05, rate-limiter defects (completed 2026-08-24)

Phases 17–26 (route-oriented surfaces/polish) were **cancelled unbuilt**. Do not recreate them. Presentation continues in v0.2.1 as contract-first job clusters.

</details>

### 🚧 v0.2.1 Contract-first presentation — job clusters (In Progress)

**Milestone Goal:** Define how Hourglass is presented — visual language, chrome, five role contracts, and one composition map — then sketch and implement by job cluster. Not by current routes.

**Build order (locked):** Design-language contract → chrome contract → five role contracts → one composition map → amend sketch-loop contract only if still ambiguous → sketch → implement by job cluster. Do not implement before the contract/map sequence is settled. Do not recreate cancelled v0.2 Phases 17–26.

- [ ] **Phase 17: Design-language contract** - Type, color, density, motion, status vocabulary. Phase 15 tokens/components are inputs.
- [ ] **Phase 18: Chrome contract** - App shell, navigation, role-scoped chrome, page anatomy. No Admin/Settings.
- [ ] **Phase 19: Role contracts** - Employee, Manager, Finance, HR, Customer. Customer may conclude "no app surface".
- [ ] **Phase 20: Composition map + sketch-loop reconcile** - One cross-role map; amend SKETCH-LOOP-CONTRACT.md only if ambiguity remains.

Job-cluster implementation phases are **inserted after Phase 20**. They are not listed here yet.

## Phase Details

### Phase 17: Design-language contract

**Goal**: A design-language contract exists and is the source of truth for type, color, density, motion, and status vocabulary. Phase 15 tokens and frozen components are inputs, not a substitute.
**Depends on**: Nothing (first phase of v0.2.1; consumes Phase 15 artifacts)
**Requirements**: DL-01
**Success Criteria** (what must be TRUE):

  1. A design-language contract document exists and is referenced as source of truth for subsequent presentation work (DL-01)
  2. The contract covers type, color, density, motion, and status vocabulary — not only the Phase 15 token dump
  3. Phase 15 frozen components (PageHeader, FilterBar, DataTable, StatusBadge, EmptyState, ConfirmDialog) are listed as inputs, with gaps called out rather than silently reused
  4. No UI implementation, no sketches, no route work

**Plans**: 2 plans
**Plan list**:
- [ ] 17-01-PLAN.md — Author docs/design/LANGUAGE.md design-language contract (type, color, density, motion, status vocabulary + overlay + do/don't + Gaps)
- [ ] 17-02-PLAN.md — Author docs/design/INDEX.md map and insert the AGENTS.md design gate

**UI hint**: no

### Phase 18: Chrome contract

**Goal**: A chrome contract exists and is the source of truth for the app shell: frame, navigation, role-scoped chrome, and page anatomy. Admin/Settings chrome is out of scope.
**Depends on**: Phase 17 (design language)
**Requirements**: CHR-01
**Success Criteria** (what must be TRUE):

  1. A chrome contract document exists covering frame, navigation, role-scoped chrome, and page anatomy (CHR-01)
  2. Admin/Settings chrome is explicitly excluded
  3. ADR-P-011 pillar IA is treated as an input to be confirmed or revised by later composition — not silently kept as the chrome
  4. No UI implementation, no sketches, no route work

**Plans**: TBD
**UI hint**: no

### Phase 19: Role contracts

**Goal**: Five role contracts exist. Each names the jobs that role performs and the surfaces those jobs need. Jobs are not current routes. Customer may conclude "no app surface".
**Depends on**: Phase 18 (chrome)
**Requirements**: EMP-01, MGR-01, FIN-01, HR-01, CUST-01
**Success Criteria** (what must be TRUE):

  1. Employee role contract exists and names jobs, not routes (EMP-01)
  2. Manager role contract exists; org-tree work is a manager job, not Admin (MGR-01)
  3. Finance role contract exists covering cutoffs, coverage money-labeling, reporting (FIN-01)
  4. HR role contract exists; org-tree / people composition is shared with manager, not Admin (HR-01)
  5. Customer role contract exists and may conclude "no app surface"; customer portal remains out of scope (CUST-01, D-E)
  6. Archived v0.2 leftovers (TICK-06, AVAIL-03..05, SURF-*, POLS-*) are used as job-shaped hints, not copied as page requirements
  7. No UI implementation, no sketches, no route work

**Plans**: TBD
**UI hint**: no

### Phase 20: Composition map + sketch-loop reconcile

**Goal**: One cross-role composition map exists. After it, the sketch-loop contract is reconciled — amended only if ambiguity remains. Sketching follows this phase; it does not precede it.
**Depends on**: Phase 19 (all five role contracts)
**Requirements**: COMP-01, SKETCH-01
**Success Criteria** (what must be TRUE):

  1. One composition map shows how the five role contracts share chrome and surfaces (COMP-01)
  2. Org tree is placed as manager/HR composition, not Admin/Settings (COMP-01)
  3. SKETCH-LOOP-CONTRACT.md is reviewed against the map. Amend it only if ambiguity remains (stale applies-to phases 16-26 / UI-only ban). Do not run a sketch session to close leftover UXFD-02 (SKETCH-01)
  4. No job-cluster implementation starts in this phase
  5. After this phase, job-cluster phases may be inserted; they must not recreate cancelled v0.2 route phases

**Plans**: TBD
**UI hint**: no

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 0. Testing | v0.1 | 6/6 | Complete | 2026-06-09 |
| 1. Authorization | v0.1 | 3/3 | Complete | 2026-06-09 |
| 2. Org Hierarchy | v0.1 | 3/3 | Complete | 2026-06-11 |
| 3. Customers | v0.1 | 3/3 | Complete | 2026-06-11 |
| 4. Contracts | v0.1 | 2/2 | Complete | 2026-06-12 |
| 5. Projects | v0.1 | 4/4 | Complete | 2026-06-12 |
| 6. Time+Expenses | v0.1 | 5/5 | Complete | 2026-06-14 |
| 7. Exports | v0.1 | 3/3 | Complete | 2026-06-14 |
| 8. Hardening | v0.1 | 4/4 | Complete | 2026-07-31 |
| 9. Activity Ont. | v0.1 | 8/8 | Complete | 2026-07-31 |
| 10. IA Impl. | v0.1 | 6/6 | Complete | 2026-08-01 |
| 11. Foundations | v0.2 | 8/8 | Complete | 2026-08-07 |
| 12. Coverage Backend | v0.2 | 7/7 | Complete | 2026-08-08 |
| 13. Direction Backend | v0.2 | 10/10 | Complete | 2026-08-08 |
| 14. Availability Backend | v0.2 | 11/11 | Complete | 2026-08-12 |
| 15. UX Foundation | v0.2 | 4/4 | Complete | 2026-08-24 |
| 16. Integrity Repair | v0.2 | 1/1 | Complete | 2026-08-24 |
| 17. Design-language contract | v0.2.1 | 0/TBD | Not started | - |
| 18. Chrome contract | v0.2.1 | 0/TBD | Not started | - |
| 19. Role contracts | v0.2.1 | 0/TBD | Not started | - |
| 20. Composition map | v0.2.1 | 0/TBD | Not started | - |

*Full v0.1 phase details: [milestones/v0.1-ROADMAP.md](milestones/v0.1-ROADMAP.md)*
*Full v0.2 phase details: [milestones/v0.2-ROADMAP.md](milestones/v0.2-ROADMAP.md)*
*v0.2 requirements archive: [milestones/v0.2-REQUIREMENTS.md](milestones/v0.2-REQUIREMENTS.md)*
*v0.2.1 requirements: [REQUIREMENTS.md](REQUIREMENTS.md)*
