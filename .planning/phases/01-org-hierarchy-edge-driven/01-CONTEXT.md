# Phase 1: org-hierarchy-edge-driven - Context

**Gathered:** 2026-05-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Improve the org-hierarchy graph to use ReactFlow's onConnect edge API to update unit parentage, instead of the current node-drag→DOM-detection approach. Also verify existing functionality works as designed. Includes: non-member disclaimer, add-members dialog fix.
</domain>

<decisions>
## Implementation Decisions

### Edge → hierarchy data flow
- **D-01:** Use ReactFlow's `onConnect` to handle re-parenting. User drags from a node's handle to create a new edge — NOT node drag.
- **D-02:** onConnect direction: source = child unit, target = parent unit. The edge source fires `updateUnit(childId, {parent_unit_id: targetId})`.
- **D-03:** Cycle prevention: visually reject connections that would create a cycle (A→B→A). No mutation sent. Use ReactFlow's `isValidConnection` or similar.
- **D-04:** Existing node-drag→DOM-detection (`onNodeDrag`, `onNodeDragStop`) is replaced by edge-based `onConnect`.

### Re-parent confirmation UX
- **D-05:** onConnect shows a confirmation dialog before sending the mutation. Dialog: "Move [child] under [parent]?"
- **D-06:** On confirm: send `updateUnit` mutation → React Query invalidates `units/tree` → tree refetched → ReactFlow re-renders with new tree.

### Collapsed state preservation
- **D-07:** `collapsedIds` (Zustand Set) is preserved across all tree changes — no auto-purge.
- **D-08:** Stale collapsedIds (references to deleted nodes) are intentionally kept. When the tree refetches, `buildNodes`/`buildEdges` will simply not render those IDs (node doesn't exist), so no harm.
- **D-09:** Future: if a deleted node's ancestors are still visible, the stale `collapsedId` may cause visual artifacts. Agent has discretion to add a cleanup pass that runs after tree refetch if needed.

### Delete cascade behavior
- **D-10:** Backend is unchanged — it may cascade delete children or not. The frontend dialog handles the UX:
  - If unit has children: show dialog "Delete unit? Children will be [deleted/kept]. Members will be removed."
  - User chooses: "Delete with children" or "Cancel"
  - If unit has no children: show standard delete confirmation
  - Members are always removed (current behavior)
- **D-11:** `collapsedIds` is not purged on delete (per D-07/D-08).

### Non-member disclaimer
- **D-16:** Show a toolbar notice for users who are not assigned to any unit. Text: "Some users are not yet members of any unit." — displayed in the toolbar area of the org-hierarchy header.

### Add members dialog bug fix
- **D-17:** Both `/units/{id}/members` and `/organizations/members` APIs return 500 with "failed to fetch members". Root cause TBD — likely a SurrealDB query error or schema issue. Fix to return the member list correctly.

### the agent's Discretion
- **D-18:** Reconnect edge: when user reconnects an existing edge (A→parent becomes A→newParent), agent decides how to handle. Suggested: update the source node's `parent_unit_id` to the new target. No special multi-step needed.
- **D-19:** Edge deletion UX: when an edge is deleted (break connection), the child unit becomes a root unit. Agent decides whether to show a confirm dialog for edge deletion, or just do it.
- **D-20:** The `onNodeDrag`/`onNodeDragStop` handlers are currently used for reparenting. Agent decides which handlers to remove/update once onConnect is implemented. Also decides what to do with the `dragHoverId` state in Zustand if no longer needed.
- **D-21:** Collapsed state cleanup: if stale collapsedIds cause visual issues after tree refetch, agent has discretion to add a cleanup mechanism.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

- `https://reactflow.dev/examples/overview` — ReactFlow overview with onConnect, addEdge, reconnectEdge patterns
- `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx` — Current org-hierarchy implementation (replace node-drag with onConnect)
- `web/src/routes/_authenticated/org-hierarchy/-context/org-hierarchy-context.tsx` — Zustand store (collapsing, selectedUnit, form state)
- `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/reparent-confirm-dialog.tsx` — Existing reparent confirmation (will be reused/adapted)
- `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/delete-confirm-dialog.tsx` — Existing delete confirmation (needs update for children choice)
- `web/src/api/units.ts` — `updateUnitMutationOpts`, `reparentUnitMutationOpts` (invalidate `units/tree`)
- `web/src/types/unit.ts` — Unit, UnitTreeNode types
- `web/src/routes/_authenticated/org-hierarchy/-components/org-chart-toolbar.tsx` — Add non-member disclaimer here
- `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx` — AddMemberPopover (broken, needs fix)
- `web/src/api/auth.ts` — `AuthApis` to check current user unit membership for disclaimer logic
- Backend: `internal/adapters/primary/http/unit.go:ListMembers` — unit members endpoint (500 error)
- Backend: `internal/adapters/primary/http/organization.go:ListMembers` — org members endpoint (500 error)
- Backend: `internal/adapters/secondary/surrealdb/unit_repository.go:ListMembers` — SurrealDB query for unit members
- Backend: `internal/adapters/secondary/surrealdb/organization_management_repository.go:ListMembers` — SurrealDB query for org members

</canonical_refs>

<codebase_context>
## Existing Code Insights

### Reusable Assets
- `OrgHierarchyProvider` / `useOrgHierarchyStore` — Zustand store for all UI state
- `ReparentConfirmDialog` — Confirmation dialog for reparent (can be adapted for onConnect dialog)
- `DeleteConfirmDialog` — Delete confirmation (needs enhancement for orphan choice)
- `unitTreeQueryOpts` — React Query tree data (auto-invalidates on mutations)
- `updateUnitMutationOpts` — Sends PUT to `/units/:id`, invalidates tree

### Established Patterns
- ReactFlow used with Dagre layout (`getLayoutElements`)
- Edges built from tree (`buildEdges` walks tree to create `Edge[]`)
- Nodes/edges are recalculated on `tree`, `collapsedIds`, `viewMode`, `membersMap` changes
- `onNodeDrag`/`onNodeDragStop` currently handle reparent — will be replaced

### Integration Points
- `onConnect` → `updateUnit({parent_unit_id})` mutation → tree refetch → ReactFlow re-render
- Collapsed state in Zustand persists across tree mutations
- Edge delete → potential orphan → optional dialog

</codebase_context>

<specifics>
## Specific Ideas

- User wants to drag the edge (connecting line) from a node, not drag the node itself
- Reference: https://reactflow.dev/examples/overview for onConnect usage
- Source node = child unit, target node = new parent unit
- Non-member disclaimer: "Some users are not yet members of any unit" in toolbar
- Add members dialog broken: both `/units/{id}/members` (500) and `/organizations/members` (500) fail

</specifics>

<deferred>
## Deferred Ideas

- Edge deletion → make child a root (no dialog, just do it) — agent discretion
- Reconnect edge behavior — agent discretion
- Collapsed state cleanup for stale IDs — agent discretion

</deferred>

---

*Phase: 1-org-hierarchy-edge-driven*
*Context gathered: 2026-05-12*