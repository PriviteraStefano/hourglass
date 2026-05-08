# Org Hierarchy UI Redesign Spec
## Motivation
The current organization hierarchy view suffers from three UI issues:
1. The global toolbar overlaps the canvas and node spawn area.
2. The side detail panel jumps out instantly instead of sliding smoothly away when unselected (due to an early null return unmounting the component before the exit animation plays).
3. The node actions (Edit, Delete, Add Sub-unit) require too much navigation, residing only at the bottom of the detail panel or globally on page.
## Proposed Changes
### 1. Page Layout & Toolbar
- Modify `org-hierarchy-page.tsx`: Shift from a full-page canvas overlaid with a toolbar to a standard flexbox column layout.
- The `OrgChartToolbar` will become a static header sitting above the `ReactFlow` canvas, taking it out of the floating `<Panel>` entirely.
- The canvas will occupy the remaining space (`flex-1`).
### 2. Smooth Detail Panel Exit Animation
- Modify `unit-detail-panel.tsx`: Remove the `if (!unit) return null;` early return.
- The `<Sheet>` component will render universally to process entry/exit animations based purely on the `open` prop.
- The inner content will safely render using conditional checks (e.g., `unit && (...)`) and optional chaining to prevent undefined access crashes while animating the exit.
### 3. Immediate Node-level Actions
- Modify `bu-node.tsx`: Add a dropdown menu (e.g., via a standard vertical ellipsis icon `⋮`) to the corner of the node card.
- Actions added to the node dropdown menu:
  - **Add Sub-unit**: Triggers the global creation modal with the parent pre-filled (requires passing down action handlers).
  - **Edit**: Triggers the edit modal direct from the node.
  - **Delete**: Triggers the delete confirmation.
- Modify `BUNodeData` in `dagre-layout.ts` (or `bu-node.tsx`) to supply the required interaction callbacks from the page root.
## Error Handling & Edge Cases
- State synchronization: Ensure the side panel doesn't crash on unmount while the parent nullifies `selectedUnit`. By leaving the internal content conditionally rendered, we avoid null pointer exceptions during the slide-out duration.
- Dropdown overlapping: Ensure node dropdowns aren't clipped by the node card by checking standard overflow and z-index settings on React Flow nodes.
