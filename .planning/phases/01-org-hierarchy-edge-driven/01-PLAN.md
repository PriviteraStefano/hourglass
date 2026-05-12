# Phase 1: org-hierarchy-edge-driven - Plan

**Created:** 2026-05-12
**Status:** Ready for execution

## Goal Summary
Implement edge-driven reparenting via ReactFlow's onConnect API, add non-member disclaimer, fix broken add-members dialog, and handle delete cascade UX.

---

## Implementation Tasks

### Wave 0: Backend Bug Fixes (PREREQUISITE)

#### T-01: Fix /units/{id}/members API 500 error
- **File:** `internal/adapters/secondary/surrealdb/unit_repository.go`
- **Action:** Debug and fix the ListMembers SurrealDB query that returns 500
- **Verification:** `curl http://localhost:8080/units/{unit-id}/members` returns 200 with member list

#### T-02: Fix /organizations/members API 500 error  
- **File:** `internal/adapters/secondary/surrealdb/organization_management_repository.go`
- **Action:** Debug and fix the ListMembers SurrealDB query that returns 500
- **Verification:** `curl http://localhost:8080/organizations/members` returns 200 with member list

---

### Wave 1: Edge-Driven Reparenting Core

#### T-03: Replace node-drag with onConnect in org-hierarchy-page.tsx
- **File:** `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx`
- **Action:** 
  - Remove `onNodeDrag` and `onNodeDragStop` handlers used for reparenting
  - Add `onConnect` prop that receives `{source, target, ...}` when edge is created
  - Source = child unit, target = parent unit
- **Verification:** Dragging from source handle to target handle triggers onConnect callback

#### T-04: Add cycle prevention with isValidConnection
- **File:** `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx`
- **Action:** Implement `isValidConnection` function that checks if connection would create cycle (A→B→A)
- **Verification:** Attempting to create cycle connection is visually rejected

#### T-05: Update ReparentConfirmDialog for onConnect
- **File:** `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/reparent-confirm-dialog.tsx`
- **Action:** 
  - Adapt to receive source/target from edge params (not mouse position)
  - Show: "Move [child name] under [parent name]?"
- **Verification:** Dialog appears with correct unit names after onConnect fires

#### T-06: Wire onConnect to mutation and tree refetch
- **File:** `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx`
- **Action:**
  - On confirm: call `updateUnit(source, {parent_unit_id: target})`
  - React Query invalidates `units/tree` → tree refetch → ReactFlow re-renders
- **Verification:** After confirming reparent, tree refetches and shows new structure

---

### Wave 2: UX Enhancements

#### T-07: Add non-member disclaimer to toolbar
- **File:** `web/src/routes/_authenticated/org-hierarchy/-components/org-chart-toolbar.tsx`
- **Action:** 
  - Check if current user has any unit memberships via AuthApis
  - Show "Some users are not yet members of any unit" in toolbar when users exist without unit membership
- **Verification:** Toolbar shows notice when users lack unit assignments

#### T-08: Update DeleteConfirmDialog for children choice
- **File:** `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/delete-confirm-dialog.tsx`
- **Action:**
  - If unit has children: show choice "Delete with children" or "Cancel"
  - If no children: show standard delete confirmation
- **Verification:** Deleting unit with children shows choice dialog

#### T-09: Fix AddMemberPopover in unit-detail-panel.tsx
- **File:** `web/src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx`
- **Action:** 
  - Fix the member API calls now that T-01/T-02 are fixed
  - Ensure add-member flow works end-to-end
- **Verification:** Can add members to a unit via the detail panel

---

### Wave 3: Cleanup (Discretion)

#### T-10: Clean up obsolete state (optional)
- **Files:** Zustand store in `org-hierarchy-context.tsx`
- **Action:** 
  - Remove `dragHoverId` state if no longer needed (per D-20)
  - Optional: add cleanup for stale collapsedIds after tree refetch (per D-21)
- **Verification:** No console errors, state is minimal

---

## Verification Checklist

| Task | Verification Method | Expected Result |
|------|---------------------|-----------------|
| T-01 | `curl http://localhost:8080/units/{id}/members` | 200 + JSON array |
| T-02 | `curl http://localhost:8080/organizations/members` | 200 + JSON array |
| T-03 | Browser: drag from handle to handle | onConnect callback fires |
| T-04 | Browser: try create cycle | Connection visually rejected |
| T-05 | Browser: confirm reparent | Dialog shows correct names |
| T-06 | Browser: complete reparent | Tree refetches, structure correct |
| T-07 | Browser: view toolbar | Notice shows/hides appropriately |
| T-08 | Browser: delete unit with children | Choice dialog appears |
| T-09 | Browser: add member to unit | Member appears in unit |

---

## Dependencies

- **Blockers:** None identified
- **Prerequisites:** T-01, T-02 must complete before T-09 (add members dialog relies on API)

---

## Notes

- Edge deletion → child becomes root (per D-19: agent discretion, no dialog)
- Reconnect edge → update parent_unit_id (per D-18: agent discretion)
- Collapsed state preserved per D-07/D-08/D-11