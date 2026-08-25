# Phase 15: UX Foundation — Design Tokens + Shared Components - Pattern Map

**Mapped:** 2026-08-12
**Files analyzed:** 19 new/modified files (+ 6 test files counted with their subjects)
**Analogs found:** 17 / 19 (2 no-analog: new npm dep config, sketch contract doc)

Source of truth: `15-UI-SPEC.md` (approved) pins every token value and component API contract. The patterns below show **which existing code to copy from** — the UI-SPEC supplies the *what*.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `web/src/index.css` | config (tokens) | theming | itself — `:root`/`.dark`/`@theme inline` (lines 33–195) | exact (extend in place) |
| `web/src/components/shared/status-badge.tsx` | component | render | itself (current file) + `ui/badge.tsx` (cva) | exact (self-rewrite) |
| `web/src/components/shared/data-table.tsx` | component | CRUD list render | `shared/entries-table.tsx` | role-match (hand-rolled → TanStack v8) |
| `web/src/components/shared/filter-bar.tsx` | component | CRUD filter state | `shared/entries-filters.tsx` | role-match |
| `web/src/components/shared/page-header.tsx` | component | render | `layout/header.tsx` + `customers-page.tsx:47-65` (Header block) | role-match |
| `web/src/components/shared/empty-state.tsx` | component | render | `ui/empty.tsx` (wrapper target) + `approvals-page.tsx:229-291` (Empty composition) | exact |
| `web/src/components/shared/confirm-dialog.tsx` | component | event-driven (destructive) | 3 delete dialogs (customers / org-hierarchy / working-groups) | exact (consolidation input) |
| `web/src/types/models.ts` | model | — | `EntryStatus` union (lines 3–9) | exact |
| `web/src/types/index.ts` | barrel | — | itself (line 1 `export * from "./models"`) | exact (verify only) |
| `web/src/components/ui/empty.tsx` | component | render | itself (EmptyTitle base class, line 63) | exact |
| 3 × per-page delete dialogs | component | event-driven | themselves (state wiring stays; chrome swaps to ConfirmDialog) | exact |
| `web/package.json` + `bun.lock` | config | — | no analog (new dep) — use `bun add @tanstack/react-table@^8.21.3` | no analog |
| `web/src/components/shared/__tests__/*.test.tsx` (6 files) | test | — | `shared/__tests__/entries-table.test.tsx` | exact |
| `.planning/sketches/SKETCH-LOOP-CONTRACT.md` | doc (process) | — | no analog (dir doesn't exist yet) — source: gsd-sketch SKILL.md + `~/.config/opencode/gsd-core/workflows/sketch.md` | no analog |

---

## Pattern Assignments

### `web/src/index.css` (config, theming — EXTEND IN PLACE)

**Analog:** itself — the existing `:root` / `.dark` / `@theme inline` token machinery.

**Token pair pattern — `:root` block** (`web/src/index.css:33-97`; insert after `--destructive` line 79):
```css
:root {
  /* existing (index.css:79) — danger reuses this (D-15-01) */
  --destructive: oklch(0.577 0.245 27.325);
  /* NEW: --status-{role} + --status-{role}-foreground pairs (UI-SPEC Color table) */
  --status-neutral: var(--base-600);
  --status-neutral-foreground: var(--color-white);
  --status-info: oklch(0.546 0.245 262.881);
  --status-info-foreground: var(--color-white);
  --status-warning: oklch(0.769 0.188 70.08);
  --status-warning-foreground: var(--color-black); /* black on amber */
  --status-danger: var(--destructive);
  --status-danger-foreground: var(--color-white);
}
```

**`.dark` block** (`web/src/index.css:99-131`; insert after `--destructive` line 114): same keys, dark values (`--status-neutral: var(--base-400)`, `--status-info: oklch(0.707 0.165 254.624)`, `--status-warning: oklch(0.828 0.189 84.429)`, danger stays `var(--destructive)` — dark swap red-600→red-400 rides along).

**`@theme inline` registration** (`web/src/index.css:133-195`; insert next to `--color-destructive` line 178):
```css
@theme inline {
  --color-destructive: var(--destructive);
  /* NEW: 10 keys — 5 roles × (base + foreground) */
  --color-status-neutral: var(--status-neutral);
  --color-status-neutral-foreground: var(--status-neutral-foreground);
  --color-status-info: var(--status-info);
  --color-status-info-foreground: var(--status-info-foreground);
  --color-status-success: var(--status-success);
  --color-status-success-foreground: var(--status-success-foreground);
  --color-status-warning: var(--status-warning);
  --color-status-warning-foreground: var(--status-warning-foreground);
  --color-status-danger: var(--status-danger);
  --color-status-danger-foreground: var(--status-danger-foreground);
}
```
**Must keep `@theme inline` (not plain `@theme`)** — utilities must reference the *runtime* var so `.dark` swaps resolve (research Pitfall 2). Alpha on custom-property colors uses the parenthesized form used by StatusBadge variants: `bg-(--status-info)/10` (research Pattern 1 / A2; fallback `bg-status-info/10` if the syntax fails at build).

---

### `web/src/components/shared/status-badge.tsx` (component, render — REWRITE)

**Analog:** the current file itself + `ui/badge.tsx` cva recipe + `ui/empty.tsx:28-41` cva shape.

**Import + props contract to keep stable** (current `status-badge.tsx:1-7`; the re-export file `time-entries/-components/status-badge.tsx:1-4` re-exports both `StatusBadge` and the `StatusBadgeProps` **type** — do not remove that export):
```tsx
import { type EntryStatus } from "@/types";
import { cn } from "@/lib/utils.ts";

export interface StatusBadgeProps {
  status: EntryStatus;
  className?: string;
}
```
Rewrite extends this: generic `StatusBadge<S extends string>({ status, variant = "subtle", label?, mapping?, className? })` with `export type StatusRole = "neutral" | "info" | "success" | "warning" | "danger"` and `export const STATUS_ROLE_MAP: Record<string, StatusRole>` (D-15-03 table). Keep `StatusBadgeProps` exported so the re-export file and 8 consumer sites compile with zero edits (research Pitfall 3).

**The hardcoded map being replaced** — current `status-badge.tsx:9-40` (`statusConfig` with `bg-yellow-100 text-yellow-800` per-status classes). Do NOT copy this pattern anywhere; anti-pattern (ROADMAP SC1).

**cva variant recipe to copy** — `ui/empty.tsx:28-41`:
```tsx
const emptyMediaVariants = cva(
  "mb-2 flex shrink-0 items-center justify-center [&_svg]:pointer-events-none [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-transparent",
        icon: "flex size-8 shrink-0 items-center justify-center rounded-md bg-muted ...",
      },
    },
    defaultVariants: { variant: "default" },
  }
);
```
Variant classes come from UI-SPEC (`subtle`/`solid`/`outline`/`dot` recipes) — static strings, role substituted at runtime; base: `inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-semibold whitespace-nowrap` (note **font-semibold**, remapped from the current `font-medium` at line 48 per the 2-weight typography contract).

**Label humanization:** snake_case → Title Case; unknown status → neutral + verbatim humanized label, never crash, never ad-hoc hex (UI-SPEC Color / D-15-03).

---

### `web/src/components/shared/data-table.tsx` (component, CRUD list render — NEW)

**Analog:** `shared/entries-table.tsx` (same role: generic presentational table; different engine: hand-rolled pagination → TanStack Table v8).

**Generic shell + props pattern to copy** (`entries-table.tsx:15-54`):
```tsx
export interface EntriesTableProps<T> {
  columns: EntriesColumn<T>[];
  rows: T[];
  getRowKey: (row: T) => string;
  onRowClick?: (row: T) => void;
  emptyState: ReactNode;
  pageSize?: number;
  ariaLabel?: string;
}
export function EntriesTable<T>({ columns, rows, getRowKey, onRowClick, emptyState, pageSize = 25, ariaLabel }: EntriesTableProps<T>) { ... }
```
For DataTable: same generic-presentational shape but `columns: ColumnDef<T>[]` (TanStack v8 — pinned `@tanstack/react-table@^8.21.3`, NOT v9; research Decision Note + Pitfall 1).

**ui/table primitives composition** (`entries-table.tsx:69-98` + `ui/table.tsx:7-20,68-92`): wrap in `<div className="rounded-lg border overflow-x-auto">`, `<Table aria-label>`, `<TableHeader>`/`<TableRow className="hover:bg-transparent">`/`<TableHead>`/`<TableBody>`/`<TableRow className={cn(onRowClick && "cursor-pointer")} onClick={...}>`/`<TableCell>`. `ui/table.tsx` already provides `data-slot` table primitives; cells default `whitespace-nowrap` — add `truncate` + `title` per UI-SPEC overflow contract.

**Pagination footer pattern** (`entries-table.tsx:100-126`) — copy the icon-button + aria-label idiom; upgrade the label from "Page X of Y · N entries" to UI-SPEC's "Showing {first}–{last} of {total}" + page-size select (10/20/50):
```tsx
<div className="flex items-center justify-between text-sm text-muted-foreground">
  <span>Page {currentPage + 1} of {pageCount} · {rows.length} entries</span>
  <div className="flex items-center gap-1">
    <Button variant="outline" size="sm" disabled={currentPage === 0}
      onClick={() => setPage((p) => Math.max(0, p - 1))} aria-label="Previous page">
      <ChevronLeftIcon className="h-4 w-4" />
    </Button>
    <Button variant="outline" size="sm" disabled={currentPage >= pageCount - 1}
      onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))} aria-label="Next page">
      <ChevronRightIcon className="h-4 w-4" />
    </Button>
  </div>
</div>
```
**TanStack v8 core (research Code Example 1 — the ONLY v8 API, keep in plan verbatim):** `useReactTable({ columns, data, state: { sorting, pagination }, onSortingChange, onPaginationChange, getCoreRowModel: getCoreRowModel(), getSortedRowModel: getSortedRowModel(), getPaginationRowModel: getPaginationRowModel() })`, `flexRender(header.column.columnDef.header, header.getContext())`, sort toggle `header.column.getToggleSortingHandler()` + `header.column.getIsSorted()`. **A11y contract (UI-SPEC):** sort buttons `aria-label="Sort by {label}"` → `"Sorted by {label} ({direction})"` when active + `aria-sort` on the th; pagination icon buttons labeled "Previous page"/"Next page"; missing cells render em-dash `—`; `isLoading` → 5 `Skeleton` rows (`ui/skeleton.tsx:3-11`, `animate-pulse rounded-md bg-muted`); empty → EmptyState default "No matching records" / "Try adjusting your search or filters.".

---

### `web/src/components/shared/filter-bar.tsx` (component, CRUD filter state — NEW)

**Analog:** `shared/entries-filters.tsx` — same role, same URL-agnostic/presentational philosophy ("URL-agnostic: the parent page owns the search-param schema and passes value/onChange in", lines 29-31, 106).

**Search-field pattern** — copy from `customers-page.tsx:50-59` (InputGroup + InputGroupInput + SearchIcon):
```tsx
<InputGroup>
  <InputGroupInput placeholder="Search customers..." value={searchQuery}
    onChange={(e) => setSearchQuery(e.target.value)} />
  <InputGroupAddon><SearchIcon className="h-4 w-4" /></InputGroupAddon>
</InputGroup>
```

**Select-filter pattern** — `entries-filters.tsx:33-96` (`StatusFilterSelect`): DropdownMenu + DropdownMenuTrigger `render={<Button variant="outline" size="sm" className="gap-1.5">}` + count chip `rounded bg-muted px-1.5 text-xs` + DropdownMenuCheckboxItem list. FilterBar's `kind: "select"` may simplify to single-select via `ui/select` (205 lines, `data-slot` primitives) — planner choice; the chip-count idiom transfers.

**Date-range pattern** — `entries-filters.tsx:107-171` (`DateRangeFilter`): Popover + PopoverTrigger render Button with `CalendarRangeIcon` + `format(new Date(...), "dd MMM")` label + `DayPicker mode="range" numberOfMonths={2}` + Clear button. Copy verbatim; FilterBar adds a stateless `onChange` contract.

**Layout contract (UI-SPEC):** `flex flex-wrap items-center gap-2` — search first (`min-w-40`, grows), then controls, `ml-auto`, active-count chip `bg-muted text-muted-foreground rounded-full px-2 py-0.5 text-xs` ("{N} active"), "Reset" ghost button visible only when ≥1 filter active. Stateless — values from route search params (CONVENTIONS: no state in frozen set).

---

### `web/src/components/shared/page-header.tsx` (component, render — NEW)

**Analog:** `layout/header.tsx:4-21` + the canonical in-page header block `customers-page.tsx:47-65`.

**Title + actions-slot pattern** (`customers-page.tsx:47-65` — this is what every page does today; PageHeader freezes it):
```tsx
<Header>
  <h1 className="text-xl font-semibold">Customers</h1>
  <div className="ml-auto flex items-center gap-4">
    {/* search, buttons */}
  </div>
</Header>
```
PageHeader per UI-SPEC: `{ title, description?, actions?, summary?, breadcrumb? }`; title = `text-xl font-semibold truncate` (Heading 20px/600); description = `text-sm text-muted-foreground`; actions slot right-aligned via `ml-auto`; `summary?: { label: string; count?: number; tone?: StatusRole }[]` chips rendered as StatusBadge `dot` variant + count, `flex flex-wrap`; breadcrumb slot above title. **Not** the `Header` component itself (that's the app chrome bar) — PageHeader is the content-area header; check where pages render their `<h1>` (all under `<Header>`/`<Body>` in `customers-page.tsx:46-67`, `approvals-page.tsx:222-224`) and replace with the frozen component at consuming sites.

---

### `web/src/components/shared/empty-state.tsx` (component, render — NEW)

**Analog:** `ui/empty.tsx` (the wrapper target) + its canonical composition in `approvals-page.tsx:229-291`.

**Wrapper composition to copy** (`approvals-page.tsx:229-243` — Empty/EmptyHeader/EmptyMedia/EmptyTitle/EmptyContent/EmptyDescription):
```tsx
<Empty>
  <EmptyHeader>
    <EmptyMedia variant="icon"><InboxIcon /></EmptyMedia>
    <EmptyTitle>Queue is clear</EmptyTitle>
  </EmptyHeader>
  <EmptyContent>
    <EmptyDescription>There are no {activeStage} approvals waiting. ...</EmptyDescription>
  </EmptyContent>
</Empty>
```
EmptyState per UI-SPEC: `{ icon?, title, description?, action? }` — default `Inbox` lucide icon in `EmptyMedia variant="icon"` (muted box `size-8`/`size-4` icon), `action?` ReactNode below. No invented copy (props only). Note the EmptyTitle base-class edit below is what makes the title contract (Label 14px/600) hold.

---

### `web/src/components/ui/empty.tsx` (component — MODIFY, 1-line remap)

**Analog:** itself. `EmptyTitle` base class line 62-64: `"font-heading text-sm font-medium tracking-tight"` → **`font-semibold`** (500 must not survive the frozen type system — research Pitfall 5; impacts today-page + approvals-page empty states, intended).

---

### `web/src/components/shared/confirm-dialog.tsx` (component, event-driven destructive — NEW)

**Analog:** the 3 per-page delete dialogs (customers `delete-confirm-dialog.tsx:1-65`, org-hierarchy `delete-confirm-dialog.tsx:1-61`, working-groups `delete-working-group-dialog.tsx:1-67`) — identical AlertDialog chrome to consolidate.

**AlertDialog composition to copy** (customers `delete-confirm-dialog.tsx:42-63` — the canonical chrome):
```tsx
<AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>Delete Customer</AlertDialogTitle>
      <AlertDialogDescription>Are you sure you want to delete <strong>{...}</strong>? This action cannot be undone.</AlertDialogDescription>
    </AlertDialogHeader>
    <AlertDialogFooter>
      <AlertDialogCancel>Cancel</AlertDialogCancel>
      <AlertDialogAction onClick={handleConfirm} disabled={deleteCustomer.isPending}
        className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
        {deleteCustomer.isPending ? "Deleting..." : "Delete"}
      </AlertDialogAction>
    </AlertDialogFooter>
  </AlertDialogContent>
</AlertDialog>
```
`ui/alert-dialog.tsx:144-168`: `AlertDialogAction` renders a `Button` with `variant`/`size` props — the destructive styling can be passed as `variant="destructive"` (cleaner than the raw class string the 3 dialogs use; either is acceptable).

**Required-reason gate precedent** — `approval-buttons.tsx:78-110` (the in-repo textarea + disabled-confirm idiom; server 400 invariant D-13-10/D-13-16):
```tsx
<Label htmlFor="reject-reason" className="text-xs">Reason for rejection (required)</Label>
<Textarea id="reject-reason" value={rejectReason} onChange={(e) => setRejectReason(e.target.value)}
  placeholder="Explain why this entry is being rejected" className="min-h-[60px] text-sm" />
...
<Button variant="destructive" size="sm" onClick={handleRejectConfirm}
  disabled={rejectReason.trim().length < 10 || isPending}>
```
ConfirmDialog freezes this as: `requiredReason` → confirm disabled until non-empty, validation copy "A reason is required.", `isSubmitting` → disabled + spinner, `error?` → inline muted-danger "Could not complete the action. Try again."

**State-wiring rule (research Pitfall 4 — critical):** ConfirmDialog stays controlled (`open`, `onOpenChange`, `onConfirm(reason?)`). The 3 sites keep their own state owners: customers zustand store (`useCustomersStore` selectors, customers dialog lines 17-20), org-hierarchy zustand (`useOrgHierarchyStore`, lines 17-20), working-groups props (`{ wg, onClose }`, lines 16-19, 26-29). Copy NO store/API imports into the shared component — presentational-only (CONVENTIONS). Delete the 3 dialog files' chrome; keep `reparent-confirm-dialog.tsx` untouched (D-15-12).

---

### `web/src/types/models.ts` (model — EXTEND IN PLACE)

**Analog:** itself. `EntryStatus` union at `models.ts:3-9` — the exact shape for the 3 new unions (D-15-13):
```tsx
export type EntryStatus =
  | "draft"
  | "submitted"
  | "pending_manager"
  | "pending_finance"
  | "approved"
  | "rejected";
```
Add after it (UI-SPEC Type unions): `TicketStatus = "open" | "triage" | "planned" | "in_progress" | "resolved" | "closed" | "dismissed"`, `DirectionStatus = "draft" | "active" | "superseded" | "cancelled"` (+ derived `"done" | "lapsed" | "claimed"`), `AbsenceStatus = "declared" | "confirmed" | "rejected" | "withdrawn"`. JSDoc mirror comments matching file style (e.g. lines 120-123 "Mirrors the backend ..." pattern).

### `web/src/types/index.ts` (barrel — verify only)
Line 1 is `export * from "./models";` — new unions flow through automatically; no edit needed unless explicit (D-15-13 says "add exports if explicit").

---

### Per-page delete dialogs (3 × component — MODIFY)

**Analog:** each dialog itself. Swap the AlertDialog chrome for `<ConfirmDialog ... />`, keeping each site's state/mutation wiring (customers: zustand + `CustomersApis.deleteCustomerMutationOpts` + 409 toast at lines 22-31; org-hierarchy: zustand + `deleteUnitMutationOpts` + `invalidateQueries({ queryKey: ["units"] })` at lines 22-37; working-groups: props + `WorkingGroupsApis.deleteWorkingGroupMutationOpts` at lines 30-35). None of these dialogs currently collect a reason — the absorbing sites pass `requiredReason` only where the server requires it (customers delete today doesn't; check the 400 invariant per endpoint when wiring, D-13-10).

---

### Tests — `web/src/components/shared/__tests__/*.test.tsx` (6 files, NEW)

**Analog:** `shared/__tests__/entries-table.test.tsx:1-143` — the in-repo convention: vitest globals, `@testing-library/react` (`render`, `screen`, `fireEvent`, `within`), `describe`/`it`/`expect`/`vi`, `getByRole` for a11y-label assertions, `it.each` for label tables. Copy the StatusBadge test table (lines 117-143) as the seed for `status-badge.test.tsx` (role-mapping table per D-15-03 + unknown-fallback case); copy the pagination/aria-label test idiom (lines 49-72) for `data-table.test.tsx` (sort `getByRole("button", { name: "Sort by Date" })`, "Previous page"/"Next page"). Run: `bun run test -- src/components/shared/__tests__/<file>.test.tsx` (vitest 4.1.10, jsdom, `web/vitest.config.ts` already configured).

---

### `.planning/sketches/SKETCH-LOOP-CONTRACT.md` (doc, process — NEW)

**No codebase analog** — `.planning/sketches/` does not exist yet (created this phase, D-15-09). Source of the pinned mechanics: `gsd-sketch/SKILL.md` + `~/.config/opencode/gsd-core/workflows/sketch.md`:
- 2–3 variants per sketch; saved to `.planning/sketches/NNN-descriptive-name/` with a `README.md` per sketch (winner marked in frontmatter: `winner: "B"`, sketch.md:252, 297)
- `.planning/sketches/MANIFEST.md` created/updated per sketch — "design direction, reference points, and sketch table with winners" (sketch.md:54, 195-212)
- `--wrap-up` → runs sketch-wrap-up workflow → produces `sketch-findings-*` skills under `.opencode/skills/` (sketch.md:56, 338)
- Commit convention `docs(sketch-NNN): [winner] ...` (sketch.md:303, 362)
- Contract pins per D-15-09/D-15-11: every surface/polish phase (16–26) runs gsd-sketch first; 2–3 variants; user agrees on one; UI-only plans; **≤3 sketch rounds (the only hard rule)** — wrap-up is NOT a round (research A6).

---

## Shared Patterns

### Presentational-only components
**Source:** `entries-table.tsx:28-39` + CONVENTIONS.md ("Components stay presentational — server state lives in routes/hooks") + research Pitfall 4.
**Apply to:** ALL six frozen components. Props-only: no `useQuery`, no zustand stores, no navigation, no `useState` beyond local UI state (DataTable sorting/pagination, ConfirmDialog reason). Each frozen component gets a JSDoc block citing this phase's decisions (e.g. `(D-15-06)`) per CONVENTIONS.md comment rules.

### Variant pattern (cva + cn + data-slot)
**Source:** `ui/empty.tsx:28-41` (cva), `ui/badge.tsx:7-28` (cva with `defaultVariants`), `web/src/lib/utils.ts:1-6` (`cn = twMerge(clsx(...))`).
**Apply to:** StatusBadge (4 variants), EmptyState wrapper, ConfirmDialog (`variant: "destructive" | "default"`). Named exports only, kebab-case filenames, `import type` for type-only imports, `@/` imports with explicit `.ts`/`.tsx` extensions (CONVENTIONS.md:24, 31), oxfmt (80-col, semicolons, double quotes).

### Token consumption rule
**Source:** `web/src/index.css:79` (`--destructive` precedent) + UI-SPEC Color.
**Apply to:** all frozen components — colors reference role tokens only (`bg-(--status-{role})/10`, `text-(--status-{role})`…), never hex/Tailwind palette classes; danger = `--status-danger` (which is `var(--destructive)`). Accent (`--primary`) never carries status semantics.

### Alert-dialog destructive chrome
**Source:** `customers/-components/dialogs/delete-confirm-dialog.tsx:42-63`.
**Apply to:** ConfirmDialog + the 3 absorbing sites. Title copy `"{Verb} {noun}?"`, description "This action cannot be undone.", destructive confirm styling, `isPending` → disabled + "Deleting...".

### A11y contract labels
**Source:** `entries-table.tsx:111,120` ("Previous page"/"Next page") + UI-SPEC.
**Apply to:** DataTable sort buttons ("Sort by {label}" / "Sorted by {label} ({direction})" + `aria-sort`), pagination icon buttons, FilterBar icon-only controls. Asserted in tests via `getByRole("button", { name: ... })`; enforced by oxlint jsx-a11y.

### Import ordering
**Source:** CONVENTIONS.md:37-45 — external packages first (`react`, `@tanstack/react-query`, `lucide-react`, `@tanstack/react-table`, `class-variance-authority`), then `@/` aliases, then relative. Followed by every analog file read this session.

---

## No Analog Found

| File | Role | Data Flow | Reason / Source to use |
|------|------|-----------|------------------------|
| `web/package.json` + `bun.lock` (`@tanstack/react-table@^8.21.3`) | config | — | No in-repo precedent for this dep. Use `bun add @tanstack/react-table@^8.21.3` behind `checkpoint:human-verify` (SUS verdict, research Package Legitimacy Audit). NOT v9 (3-day-old breaking major). |
| `.planning/sketches/SKETCH-LOOP-CONTRACT.md` | doc | — | No sketches dir exists yet. Compose from gsd-sketch SKILL.md + sketch.md workflow mechanics (pinned above in its assignment). |
| `15-UI-SPEC.md` | doc (contract) | — | Already exists + approved (commit `ff165ed`) — verify presence, do NOT regenerate (research A5). |

---

## Metadata

**Analog search scope:** `web/src/components/shared/`, `web/src/components/ui/`, `web/src/components/layout/`, `web/src/components/approval/`, `web/src/types/`, `web/src/routes/_authenticated/{customers,org-hierarchy,working-groups,approvals,time-entries}/-components/`, `web/src/lib/`, `~/.config/opencode/skills/gsd-sketch/`, `~/.config/opencode/gsd-core/workflows/sketch.md`
**Files scanned:** 19 source files + 2 test files + 1 CSS + 2 skill/workflow docs
**Pattern extraction date:** 2026-08-12
