# Organization Hierarchy Management Page Design

## Overview

Implement a frontend page for managing organization hierarchy (business units) using React Flow for visualization. The page allows viewing, creating, editing, deleting, and reordering business units via a visual org chart with drag-and-drop support.

**Approach**: Full-Screen Canvas + Floating Controls (React Flow fills the page, floating toolbar with search and actions, side panel for details).

## Architecture & Components

**Route**: `web/src/routes/_authenticated/org-hierarchy.tsx` (protected route under `_authenticated`)

**Components**:
- `OrgHierarchyPage` — main page container, fetches unit tree via `queryClient.ensureQueryData`, manages React Flow state
- `BUOrgChart` — React Flow canvas wrapper with custom node types, handles node drag events
- `BUNode` — custom React Flow node component (shadcn Card styling, displays unit name, code, hierarchy level)
- `OrgChartToolbar` — floating toolbar (shadcn Card) with search input, zoom controls, and "Add Unit" button
- `UnitDetailPanel` — slide-in side panel (shadcn Sheet) showing selected unit details with edit/delete actions
- `UnitFormDialog` — shadcn Dialog for creating/editing units (name, description, code, parent unit selector)

**Libraries to add**: `reactflow` (plus `@types/reactflow` if needed)

## Data Flow & API Integration

### Data Types (Zod schemas)

```typescript
const UnitSchema = z.object({
  id: z.string().uuid(),
  org_id: z.string().uuid(),
  name: z.string(),
  description: z.string(),
  parent_unit_id: z.string().uuid().nullable(),
  hierarchy_level: z.number(),
  code: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
})

type Unit = z.infer<typeof UnitSchema>

const UnitTreeNodeSchema: z.ZodType<UnitTreeNode> = z.lazy(() =>
  z.object({
    unit: UnitSchema,
    children: z.array(UnitTreeNodeSchema),
  })
)

type UnitTreeNode = { unit: Unit; children: UnitTreeNode[] }
```

### API Endpoints

- `GET /units/tree` — returns `UnitTreeNode[]` for the organization (uses `GetTree` handler)
- `POST /units` — create unit (body: `CreateUnitRequest`)
- `PUT /units/:id` — update unit (body: `UpdateUnitRequest`)
- `DELETE /units/:id` — delete unit
- `GET /units/:id` — get single unit (for detail panel)

### Flow

1. **Route preloading**: `beforeLoad` calls `queryClient.ensureQueryData(unitTreeQueryOpts)` to fetch tree data
2. **Transform**: Convert `UnitTreeNode[]` to React Flow format:
   - Nodes: `{ id: unit.id, type: 'bu', position: { x: level * 300, y: index * 150 }, data: { unit } }`
   - Edges: `{ source: unit.parent_unit_id, target: unit.id }` (skip if parent_unit_id is null)
3. **Drag-and-drop**: On node drag stop, call `PUT /units/:id` with updated `parent_unit_id` (if changed) → invalidate query
4. **Unit selection**: Click node → open `UnitDetailPanel` with unit data
5. **CRUD operations**:
   - Create: "Add Unit" button → `UnitFormDialog` → `POST /units` → invalidate query
   - Edit: In detail panel, click edit → `UnitFormDialog` prefilled → `PUT /units/:id` → invalidate query
   - Delete: In detail panel, click delete → confirm → `DELETE /units/:id` → invalidate query
6. **Search**: Filter nodes by `name` or `code`, hide non-matching nodes in React Flow (set `hidden` property)

## Error Handling

- API errors: Display via `sonner` toasts (existing pattern in app)
- Invalid parent assignment: Backend returns error, show toast
- Delete with members: Backend returns 400, show specific message
- Network errors: React Query error handling with retry: false (existing client config)

## Testing

- Unit tests for tree-to-nodes/edges transformation logic
- Component tests for `BUNode`, `OrgChartToolbar`, `UnitFormDialog` (using existing test setup)
- Integration test: mock API, render page, simulate create/edit/delete flows
- E2E test (Playwright): full flow from navigation to CRUD operations

## Dependencies

- `reactflow` — visual org chart with drag-and-drop
- Existing: `zod`, `@tanstack/react-query`, `shadcn` components, `lucide-react` icons, `sonner`
