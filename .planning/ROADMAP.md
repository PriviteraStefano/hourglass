# Hourglass Roadmap

## Milestones

- ✅ **v0.1 MVP Consolidation** — Phases 0-10 (shipped 2026-08-01)
- ✅ **v0.2 Ontology Extension — Origins, Tickets & Coverage + Direction** — Phases 11-16 (shipped 2026-08-25; phases 17-26 cancelled unbuilt)

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

*Full v0.1 phase details: [milestones/v0.1-ROADMAP.md](milestones/v0.1-ROADMAP.md)*
*Full v0.2 phase details: [milestones/v0.2-ROADMAP.md](milestones/v0.2-ROADMAP.md)*
*v0.2 requirements archive: [milestones/v0.2-REQUIREMENTS.md](milestones/v0.2-REQUIREMENTS.md)*
