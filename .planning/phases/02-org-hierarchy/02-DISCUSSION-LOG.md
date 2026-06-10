# Phase 2: Org Hierarchy - Discussion Log

**Date:** 2026-06-10

## Areas Discussed

### 1. Primary Unit Designation
- **Question:** Do you need primary unit designation in v0.1?
- **Answer:** Yes, include now
- **Question:** Where should 'Set as Primary' action appear?
- **Answer:** In side panel only
- **Notes:** Backend needs new PUT members endpoint. Member rows in side panel get a "Make Primary" button.

### 2. Dedicated Pages vs Side Panel
- **Question:** Dedicated pages (units/$id, members/$id) or Sheet-only?
- **Answer:** Sheet is fine for now, add to backlog for v0.0.2
- **Notes:** No route pages needed for v0.1.

### 3. Delete Protection & Error Handling
- **Question:** Full backend enforcement or frontend-only?
- **Answer:** User asked about tree traversal capabilities. Confirmed recursive CTE exists (GetDescendants).
- **Question:** Full enforcement (root + children + members) or frontend-only?
- **Answer:** Full backend enforcement. Add root unit check + children check.
- **Notes:** Backend currently only checks HasMembers. FK constraint catches children with generic error. Need proper checks + clean error messages.

### 4. Reparent Mutation API
- **Question:** Use dedicated reparent mutation or keep updateUnit?
- **Answer:** Use dedicated reparent mutation (reparentUnitMutationOpts)
- **Notes:** Switch ReparentConfirmDialog to use reparentUnitMutationOpts.

### 5. Member End-Date Support
- **Question:** Purpose of end_date?
- **Answer:** Explained as soft-deactivation pattern. User understood.
- **Question:** Keep schema-only, drop frontend type, or wire later?
- **Answer:** Schema-only, no UI.

### 6. Tree Default Expand State
- **Question:** Fully expanded, collapsed to root, or remember state?
- **Answer:** Fully expanded (current behavior).

### 7. Auto-Generated Unit Code
- **Question:** Keep auto-generate or manual only?
- **Answer:** Keep auto-generate.

### 8. Member View Performance
- **Question:** Defer perf issue or add batch endpoint?
- **Answer:** Add a batch endpoint (GET /units/members?unit_ids=...)

### 9. Reparent Flow Bug
- **Question:** pendingEdgeConnect is dead state. Fix options?
- **Answer:** Remove pendingEdgeConnect from store.

### 10. Subtree Members in Side Panel
- **Question:** How to show descendant members in detail panel?
- **Answer:** In graph view: node members only. In side panel: expandable groups per child unit showing their members recursively.

## Deferred Ideas
- Dedicated unit/member pages → v0.0.2
- Member end-date UI → schema-only for v0.1
- Error boundary for tree fetch → future
- Manual refresh button → not needed for v0.1
- Persist collapsed state to localStorage → not needed for v0.1

## Decisions Summary

| # | Decision | Value |
|---|----------|-------|
| D-01 | Primary unit in v0.1, side panel only | Include |
| D-02 | Planner adds PUT /units/{id}/members/{membershipId} | New endpoint |
| D-03 | One primary per user, unset others on change | Enforce |
| D-04 | No dedicated routes, Sheet-only | Keep |
| D-05 | Full backend delete protection (root + children + members) | Enforce |
| D-06 | Proper 400 errors for delete constraints | Return |
| D-07 | Frontend cascade delete kept for backward compat | Keep |
| D-08 | Switch to reparentUnitMutationOpts | Switch |
| D-09 | Remove pendingEdgeConnect from store | Remove |
| D-10 | end_date stays schema-only | Keep |
| D-11 | Fully expanded tree | Keep |
| D-12 | Auto-generate code from name | Keep |
| D-13 | Batch members endpoint | Add |
| D-14 | Per-node fallback for single unit | Keep |
| D-15 | Subtree members in side panel | Add |
| D-16 | Expandable groups per child unit | Add |
| D-17 | Fetch strategy TBD by planner | TBD |

---

*Discussion: 2026-06-10*
