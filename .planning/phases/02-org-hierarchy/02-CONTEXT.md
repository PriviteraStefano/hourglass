# Phase 2: Org Hierarchy - Context

**Gathered:** 2026-06-10
**Status:** Ready for planning

<domain>
## Phase Boundary

Org tree visualization using ReactFlow, unit CRUD with parent-unit hierarchy, member management (add/remove/primary), edge-driven reparenting, delete protection enforcement.

**Frontend ~80% built** — tree visualization, CRUD dialogs, reparenting, side panel, search, tree/members view toggle, non-member badge all exist. Missing: primary unit designation UI, subtree member expandable groups in detail panel, batch members endpoint, dedicated reparent mutation.

**Backend already exists** (UnitRepository, UnitMemberRepository, Unit handler). Backend gap: no `PUT /units/{id}/members/{membershipId}` endpoint for updating primary/end_date. Delete protection needs root unit + children checks added.

</domain>

<decisions>
## Implementation Decisions

### Primary Unit Designation
- **D-01:** Include primary unit toggle in v0.1. Place "Make Primary" action on member rows in the side panel only (not in tree node cards).
- **D-02:** Planner must add `PUT /units/{id}/members/{membershipId}` backend endpoint with `is_primary` field.
- **D-03:** When setting a member as primary, unset other primary flags for that user's memberships (one primary per user enforced at service layer).

### Dedicated Pages vs Side Panel
- **D-04:** [informational] Sheet-only for v0.1. No dedicated `units/$id.tsx` or `members/$id.tsx` routes. Deferred to v0.0.2 backlog.

### Delete Protection Enforcement
- **D-05:** Full backend enforcement. Planner adds to `Delete` service method:
  - Root unit check — cannot delete if `hierarchy_level === 0`
  - Children check — cannot delete if unit has child units (use `GetDescendants` or add `HasChildren`)
  - Members check — already exists (`HasMembers`)
- **D-06:** Return proper 400-level error messages for each constraint.
- **D-07:** [informational] Frontend cascading delete continues to handle deletion of children first (backward compat with current dialog).

### Reparent Mutation API
- **D-08:** Switch `ReparentConfirmDialog` to use dedicated `reparentUnitMutationOpts` instead of `updateUnitMutationOpts`. Cleaner contract — only sends `{parent_unit_id}`.
- **D-09:** Remove unused `pendingEdgeConnect` from Zustand store (`org-hierarchy-context.tsx`). Reparent flow: `onConnect` → `reparentUnit(dragUnit, targetUnit)` → dialog reads `draggingUnit`/`reparentTarget` → confirm calls `reparentUnitMutationOpts`.

### End-Date Support
- **D-10:** [informational] `end_date` stays schema-only for v0.1. No UI to set it. The `PUT` members endpoint (D-02) can accept `end_date` but frontend won't expose it.

### Tree Default Expand State
- **D-11:** [informational] Keep current behavior — fully expanded on load.

### Auto-Generated Unit Code
- **D-12:** [informational] Keep current UX — code auto-generates from name via slugify, user can override.

### Members View Performance
- **D-13:** Planner adds a batch endpoint `GET /units/members/batch?unit_ids=...` so the members view makes 1 request instead of N per visible unit.
- **D-14:** [informational] Keep per-node member fetching as fallback for single-unit detail.

### Subtree Members in Side Panel
- **D-15:** Side panel (UnitDetailPanel) shows both direct members and descendant unit members.
- **D-16:** Layout: "Direct Members" section (current list) followed by expandable groups per child unit. Each group header: "▶ [Sub-unit Name] (N members)". Expanded group shows its members and recursively its child units' members. All in the side panel sheet.
- **D-17:** Fetch strategy TBD by planner — either batch endpoint (D-13) extended to accept subtree scope, or recursive fetches for visible subtree. (RESOLVED: per-unit fetches via batch endpoint with `enabled` guards)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project context
- `.planning/PROJECT.md` — Project overview, key decisions, constraints
- `.planning/REQUIREMENTS.md` — Requirements (ORG-01 through ORG-05)
- `.planning/ROADMAP.md` — Phase definitions and dependency graph
- `.planning/STATE.md` — Current phase state and session info
- `docs/superpowers/specs/2026-06-08-mvp-consolidation-design.md` — Feature 2: Org Hierarchy design spec

### Backend: Unit
- `internal/adapters/primary/http/unit.go` — Unit HTTP handler (existing CRUD + member add/remove)
- `internal/core/services/unit/unit.go` — Unit service (CRUD, tree, descendants)
- `internal/core/ports/unit_repository.go` — Unit repository port interface
- `internal/adapters/secondary/postgres/unit_repository.go` — PG unit repo (ListByOrg, GetDescendants, HasMembers)
- `internal/adapters/secondary/postgres/unit_member_repository.go` — PG member repo (add/remove/list)

### Frontend: Org Hierarchy
- `web/src/routes/_authenticated/org-hierarchy/index.tsx` — Route definition + tree prefetch
- `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx` — Main tree page (ReactFlow, nodes/edges, onConnect, members view)
- `web/src/routes/_authenticated/org-hierarchy/-context/org-hierarchy-context.tsx` — Zustand store (collapsedIds, selectedUnit, delete/reparent state)
- `web/src/routes/_authenticated/org-hierarchy/-components/org-chart-toolbar.tsx` — Toolbar (search, view mode toggle, add root, non-member badge)
- `web/src/routes/_authenticated/org-hierarchy/-components/flow/bu-node.tsx` — ReactFlow custom node (card with actions, member rows in members view)
- `web/src/routes/_authenticated/org-hierarchy/-components/flow/bu-node-data.ts` — Node data types
- `web/src/routes/_authenticated/org-hierarchy/-components/flow/dagre-layout.ts` — Dagre layout engine
- `web/src/routes/_authenticated/org-hierarchy/-components/utils/tree-utils.ts` — Tree utility functions
- `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-form-dialog.tsx` — Create/edit unit dialog
- `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/delete-confirm-dialog.tsx` — Delete dialog with children cascade
- `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/reparent-confirm-dialog.tsx` — Reparent confirmation dialog
- `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx` — Unit detail side panel (members, breadcrumbs, edit/delete)
- `web/src/api/units.ts` — API mutations/queries for units
- `web/src/types/unit.ts` — Unit, UnitTreeNode, UnitMember types

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `UnitDetailPanel` — Sheet component with breadcrumbs, member list, add/remove members. Needs subtree members section added (D-15/D-16).
- `ReparentConfirmDialog` — Already has reparent confirmation flow; switch mutation target per D-08.
- `MemberRow` (in `bu-node.tsx`) + `MemberRow` (in `unit-detail-panel.tsx`) — Slight duplication; refactor into shared component optional but not required for v0.1.
- `reparentUnitMutationOpts` — Already defined in `units.ts`, unused. Ready to wire.
- `updateUnitMutationOpts` — Currently used for reparent; will be replaced per D-08.

### Established Patterns
- Zustand store for UI state, React Query for server state
- Mutations with `onSuccess: (_, __, { client })` pattern for cache invalidation
- Co-located components in `-components/`, context in `-context/`
- `useSuspenseQuery` for data fetching + `useMutation` for writes

### Integration Points
- Sidebar nav link exists (`web/src/components/layout/sidebar.tsx:44` — "Org Hierarchy" → `/org-hierarchy`)
- Unit members handled via Sheet (no dedicated route — D-04)
- Tree data invalidated via `['units', 'tree']` query key on all mutations

### Missing Backend Endpoints
- `PUT /units/{id}/members/{membershipId}` — For updating `is_primary` and `end_date` (D-02)
- `GET /units/members/batch?unit_ids=...` — Batch members endpoint (D-13)
- Delete protection: root unit + children checks in service layer (D-05)
- Unit repository: `HasChildren` method needed

</code_context>

<specifics>
## Specific Ideas

- **Subtree members display**: Side panel shows direct members first, then expandable groups per sub-unit. Recursive nesting. Example: Engineering → [2 direct members] + ▶ Frontend (3 members) [expandable] + ▶ Backend (4 members) [expandable].
- **Edge-driven reparenting flow works**: `onConnect` validates via `isValidConnection` (cycle prevention). Confirmation dialog. Mutation invalidates tree.
- **Non-member badge**: Already shows count of org members not assigned to any unit. Works on page load.

</specifics>

<deferred>
## Deferred Ideas

- **Dedicated unit detail page** (`/org-hierarchy/units/$id`) — Deferred to v0.0.2. Sheet is sufficient for v0.1.
- **Dedicated member management page** (`/org-hierarchy/members/$id`) — Deferred to v0.0.2.
- **Member end-date UI** — Schema-only for v0.1. No UI to set end_date on memberships.
- **Error boundary for tree fetch** — Current `useSuspenseQuery` has no error fallback. If tree fails, app shows nothing. Worth adding in a later phase.
- **Manual refresh button** — Not needed for v0.1. Automatic invalidation on mutations covers the primary case.
- **Persist collapsed state to localStorage** — Not needed for v0.1. Collapsed state resets on page refresh, which is acceptable.

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-Org-Hierarchy*
*Context gathered: 2026-06-10*
