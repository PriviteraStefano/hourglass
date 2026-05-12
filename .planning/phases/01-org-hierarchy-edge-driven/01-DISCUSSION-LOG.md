# Phase 1: org-hierarchy-edge-driven - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-12
**Phase:** 1-org-hierarchy-edge-driven
**Areas discussed:** Edge → hierarchy data flow, Re-parent confirmation UX, Collapsed state preservation, Delete cascade behavior

---

## Edge → hierarchy data flow

| Option | Description | Selected |
|--------|-------------|----------|
| Edges derive from tree (current) | User drags node → edge changes → tree refetched → DOM stays stable | |
| Edges trigger tree mutations | User drags node → edges update → tree mutation sent → react query invalidates → tree refetched | |
| Optimistic: edges local, tree server | Edges update locally → mutation sent → UI waits (block interaction during pending) | |
| Use onConnect | User drags the edge starting from the node — not the node itself. Use ReactFlow's onConnect function. | ✓ |

**User's choice:** the user shouldn't drag the node itself, but the edge starting from the node, I think you should use the onConnect function in ReactFlow. Here is an overview https://reactflow.dev/examples/overview

**Notes:** Fetched ReactFlow overview page to understand onConnect API. onConnect receives connection params (source, target, etc.) and calls addEdge.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Source=child → Target=parent | Edge source = child, target = parent. onConnect → update target.parent_unit_id | ✓ |
| Source=parent → Target=child | Edge source = parent, target = child. onConnect → update source.parent_unit_id | |

**User's choice:** Source=child → Target=parent

**Notes:** onConnect receives {source, target} where source=child unit, target=new parent unit

---

| Option | Description | Selected |
|--------|-------------|----------|
| Prevent + visual warning | Visual feedback (red edge), action prevented — no mutation sent | ✓ |
| Allow + server reject | User connects A→B then B→A → server returns error → edge removed, error shown | |
| Auto-remove old edge on reconnect | When reconnecting an edge, auto-remove the old edge | |

**User's choice:** Prevent + visual warning

**Notes:** Cycles (A→B→A) should be prevented visually. No mutation sent if it would create a cycle.

---

## Re-parent confirmation UX

| Option | Description | Selected |
|--------|-------------|----------|
| Confirm dialog first | onConnect shows dialog first → user confirms → mutation → refetch (Recommended) | ✓ |
| Optimistic immediate | onConnect sends mutation immediately → optimistic update → refetch on success | |

**User's choice:** Confirm dialog first

**Notes:** Dialog: "Move [child] under [parent]?" Confirm → send updateUnit mutation → invalidate tree → refetch

---

## Collapsed state preservation

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-purge stale collapsed | Collapsed IDs not in new tree → silently remove from collapsedIds set | |
| Preserve all collapsed state | Keep collapsed IDs in state → show empty spaces where deleted nodes were | ✓ |
| Smart purge | If node or ancestor was deleted, remove from collapsedIds; otherwise keep | |

**User's choice:** Preserve all collapsed state

**Notes:** No auto-purge. Stale collapsedIds referencing deleted nodes are intentionally kept.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Preserve all (keeps stale) | CollapsedIds referencing deleted nodes → silently removed | |
| Auto-purge stale IDs | CollapsedIds referencing deleted nodes → silently removed | ✓ |

**User's choice:** Preserve all (keeps stale)

**Notes:** Confirmed: keep stale collapsedIds. Won't cause harm since deleted nodes don't render.

---

## Delete cascade behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Purge stale IDs | Removed subtree (children of deleted node) become root-level | |
| Promote orphans to root | Removed subtree (children of deleted node) become root-level | |
| Cascade delete children | Removed subtree is also deleted (cascading delete) | |
| Ask user for orphans | Prompt: delete entire subtree or promote orphans to root? | ✓ |

**User's choice:** Ask user for orphans

**Notes:** When deleting a unit with children, show dialog with choice. Members are always removed.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Backend cascade (current) | Backend deletes children; tree refetched; collapsedIds updated | |
| Prevent if has children | Backend rejects delete if unit has children; show error | |
| Prevent if has members | Backend rejects if unit has members; show error (ignore children) | |

**User's choice:** when deleting show a dialog to the user where he can choose whether to keep children units or not, members are always removed

**Notes:** Frontend handles dialog with two options: "Delete with children" or "Cancel". Backend may or may not cascade — frontend dialog handles both cases.

---

## the agent's Discretion

- Reconnect edge: when user reconnects an existing edge, update source node's parent_unit_id to new target
- Edge deletion: agent decides whether to show confirm dialog or just make child a root
- onNodeDrag/onNodeDragStop: agent decides which handlers to remove/update once onConnect is implemented
- Collapsed state cleanup: agent has discretion to add cleanup if stale IDs cause visual issues

## Deferred Ideas

- Edge deletion → make child a root — agent discretion
- Reconnect edge behavior — agent discretion
- Collapsed state cleanup for stale IDs — agent discretion