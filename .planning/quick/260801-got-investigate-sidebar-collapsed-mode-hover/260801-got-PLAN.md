---
slug: 260801-got-investigate-sidebar-collapsed-mode-hover
created: 2026-08-01
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified: [web/src/components/ui/sidebar.tsx]
autonomous: true
requirements: [GOT-260801-SIDEBAR-DEAD-ZONE]
must_haves:
  truths:
    - "In collapsed sidebar mode, hovering the icon of Today, Working Groups, Customers, and Approvals highlights the button"
    - "Clicking those icons in collapsed mode navigates to the target route"
    - "Expanded sidebar group labels still render and display as before"
  artifacts:
    - path: "web/src/components/ui/sidebar.tsx"
      provides: "SidebarGroupLabel with pointer-events-none"
      contains: "pointer-events-none"
  key_links:
    - from: "SidebarGroupLabel className"
      to: "nav buttons in web/src/components/layout/sidebar.tsx"
      via: "stop invisible collapsed label from intercepting pointer events"
      pattern: "group-data-\\[collapsible=icon\\]:-mt-8.*pointer-events-none"
---

# Fix collapsed-sidebar hover/click dead zone over icons

## Objective

**Problem:** In collapsed (icon) mode, the `SidebarGroupLabel` keeps its full box but is
hidden visually only (`opacity-0` + `-mt-8`), so the 2rem-tall invisible label slides up
over the previous group's last button (Today, Working Groups, Customers, Approvals) and
swallows its hover + click events. Root cause confirmed in `260801-got-RESEARCH.md`.

**Fix (LOCKED user decision — option 1):** add `pointer-events-none` to the
`SidebarGroupLabel` className so the invisible collapsed label stops intercepting pointer
events while the layout is untouched. Group labels are non-interactive `div` headers, so
this has no effect in expanded mode. **Do not revisit this decision; do not substitute
option 2 (`hidden`) or option 3 (z-index).**

**Purpose:** Restore hover highlighting and click navigation for the four affected nav
buttons in collapsed sidebar mode.

**Output:** Single class added to `web/src/components/ui/sidebar.tsx` line 415.

<context>
@.planning/quick/260801-got-investigate-sidebar-collapsed-mode-hover/260801-got-RESEARCH.md
@web/src/components/ui/sidebar.tsx
@.planning/STATE.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add pointer-events-none to SidebarGroupLabel</name>
  <files>web/src/components/ui/sidebar.tsx</files>
  <action>Per locked decision D-01 (fix option 1): edit the `cn(...)` call inside `SidebarGroupLabel` (currently at approx line 415) in `web/src/components/ui/sidebar.tsx`. Prepend the literal class `pointer-events-none` to the existing className string so it becomes the first utility in the list. The exact current string is:
  `"flex h-8 shrink-0 items-center rounded-md px-2 text-xs text-sidebar-foreground/70 ring-sidebar-ring outline-hidden transition-[margin,opacity] duration-200 ease-linear group-data-[collapsible=icon]:-mt-8 group-data-[collapsible=icon]:opacity-0 focus-visible:ring-2 [&>svg]:size-4 [&>svg]:shrink-0"`
  Resulting string starts with: `"pointer-events-none flex h-8 shrink-0 ...`.

  Scope: this is the ONLY change. Do NOT touch `SidebarGroup`, `SidebarGroupAction`, `SidebarMenuButton`, or the app-level layout file `web/src/components/layout/sidebar.tsx`. Do NOT change `-mt-8`, `opacity-0`, or any other utility — the collapsed layout must remain pixel-identical, only pointer events are disabled. Do NOT use a collapsed-mode variant like `group-data-[collapsible=icon]:pointer-events-none` — the locked decision is the plain class (group labels are never interactive in any mode). The `SidebarGroupAction` button is a separate element and is unaffected by this change.</action>
  <verify>
    <automated>
      grep -n "pointer-events-none" web/src/components/ui/sidebar.tsx | grep -c "pointer-events-none flex h-8"  # expect ≥1, and build passes
      cd web && bun run build   # type-check + build must succeed
    </automated>
    <human-check>Start `go run ./cmd/server` (or docker-compose) and `cd web && bun run dev`. Open the app, collapse the sidebar to icon mode. Hover the icon of each of Today, Working Groups, Customers, Approvals: the button must highlight. Click each: the route must navigate. Verify expanded mode is unchanged (labels still visible).</human-check>
  </verify>
  <done>Line 415 className contains `pointer-events-none`; `bun run build` passes; in collapsed mode the four previously-broken buttons (Today, Working Groups, Customers, Approvals) now highlight on icon hover and navigate on click; expanded-mode labels render identically to before.</done>
</task>

</tasks>

<verification>
- `grep -c "pointer-events-none flex h-8" web/src/components/ui/sidebar.tsx` returns ≥ 1
- `cd web && bun run build` exits 0 (type-check + production build)
- Human visual pass on collapsed sidebar: 4 affected buttons highlight + navigate; expanded mode unchanged
</verification>

<success_criteria>
- Single-utility diff (`pointer-events-none` added) in `SidebarGroupLabel` only
- Collapsed-mode hover/click dead zone over the last-button-of-group icons is eliminated
- No regression in expanded-mode label rendering or collapsed layout geometry
</success_criteria>

<output>
Create `.planning/quick/260801-got-investigate-sidebar-collapsed-mode-hover/SUMMARY.md` when done
</output>
