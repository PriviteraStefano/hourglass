---
phase: 10-information-architecture-implementation
plan: 03
subsystem: ui
tags: [react, tanstack-router, page-shell, header, body, wrap-migration, typescript]

# Dependency graph
requires:
  - phase: 10-information-architecture-implementation (10-01)
    provides: activities pages (index/$id) post-rename, carried-over surface state, route-error conventions
  - phase: 10-information-architecture-implementation (10-02)
    provides: sidebar regroup with /activities nav item (shell surfaces reachable)
  - phase: 08-pre-deployment-hardening-p0-audit-fixes
    provides: leaf-level errorComponent convention (P0-4) preserved on wrapped routes
provides:
  - Locked Header+Body page shell on all nine carried-over authenticated page roots (time-entries, expenses, exports, contracts index+detail, customers index+detail, activities index+detail)
  - @/components/layout barrel (Header/Body exports) as the canonical shell import path
  - UI-SPEC shell composition: single h1 per page inside the 48px Header band, page content in Body with inner scroll container
  - 4 pre-existing contracts typecheck errors resolved (deferred-items 10-03 ownership) — baseline rot reduced 10 → 6
affects: [10-04 Today (landing replaces _authenticated/index redirect), phase-level UAT visual walk against UI-SPEC focal-point table]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Page shell composition: <Header>{title + page-global affordances}</Header><Body>{h-full overflow-y-auto inner scroll container + padded content}</Body> as direct children of SidebarInset"
    - "Header overflow rule: affordances that don't fit the 48px band stay in Body pinned atop the content (contracts status filter, contract-detail adopted-from line)"
    - "In-page pending/error states render inside Body within the shell frame; route-level errorComponent stays leaf-level per Phase 8 convention"
    - "Layout barrel (index.ts) as the import home for shell primitives"

key-files:
  created:
    - web/src/components/layout/index.ts
  modified:
    - web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx
    - web/src/routes/_authenticated/expenses/-components/expenses-page.tsx
    - web/src/routes/_authenticated/exports/-components/exports-page.tsx
    - web/src/routes/_authenticated/contracts/-components/contract-list.tsx
    - web/src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx
    - web/src/routes/_authenticated/contracts/$id/-components/contract-detail.tsx
    - web/src/routes/_authenticated/customers/-components/customers-page.tsx
    - web/src/routes/_authenticated/customers/-components/customer-detail.tsx
    - web/src/routes/_authenticated/activities/-components/activity-list.tsx
    - web/src/routes/_authenticated/activities/-components/activity-detail.tsx

key-decisions:
  - "Layout barrel index.ts created so @/components/layout resolves (task 1 acceptance criterion); exports Header/Body only — minimal, no behavior change"
  - "Tab triggers (List/Calendar/Export) stay inside Body, not moved to Header: the binding acceptance criterion 'existing control tree unmodified apart from being wrapped' + threat T-10-03-1 override the action text's 'tab triggers that fit the band'; moving TabsList out of the Tabs root would break base-ui Tabs context wiring and constitute re-layout"
  - "Shell page titles use UI-SPEC Heading style (text-xl font-semibold) in the Header on every wrapped page"
  - "Pages with no prior padding (exports/contracts/customers/activities) get the UI-SPEC lg 24px inner container (p-6) inside Body; time-entries/expenses keep their existing p-2 (plan: keep existing padding container as-is)"
  - "All wrapped pages get the h-full overflow-y-auto inner scroll container inside Body (UI-SPEC: long content scrolls inside Body, window never scrolls)"
  - "Leaf-level errorComponent preserved verbatim per plan instruction; in-page pending/error UI (activity-detail loading/not-found) now renders inside Body within the shell frame (threat T-10-03-2)"
  - "Contracts typecheck rot fixed inline (deferred-items assigns those 4 errors to 10-03): currency v??undefined, Customer.company_name, navigate search:{from:'owned'}, Select onChange wrapper"

requirements-completed: [UI-SPEC-SHELL, P-011-D2]

# Metrics
duration: 9min
completed: 2026-07-31
---

# Phase 10 Plan 03: Page shell migration — Header+Body wrap on all carried-over pages (wrap-only)

**Header+Body page shell applied to all nine carried-over authenticated page roots (time-entries, expenses, exports, contracts index+detail, customers index+detail, activities index+detail) — single h1 per page in the shell's 48px Header band, content wrapped in Body with inner scroll container, and the 4 contracts typecheck errors assigned to this plan resolved (baseline rot 10 → 6)**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-31T21:21:48Z
- **Completed:** 2026-07-31T21:30:38Z
- **Tasks:** 3
- **Files modified:** 10 (1 created, 9 modified)

## Accomplishments

- Locked UI-SPEC page shell composition (`<Header>{page affordances}</Header>` → `<Body>{page content}</Body>`, org-hierarchy reference pattern) applied to every carried-over authenticated page: TimeEntriesPage, ExpensesPage, ExportsPage, ContractList, ContractDetail, CustomersPage, CustomerDetail, ActivityList, ActivityDetail — no re-layout, no control redesign on any content tree
- New `web/src/components/layout/index.ts` barrel so `@/components/layout` resolves to `Header`/`Body` (task 1 acceptance criterion imports it literally)
- Per-page shell headers: titles in UI-SPEC Heading style (`text-xl font-semibold`, the page's only `h1`); right-aligned existing CTAs (contracts "Create", customers "Add Customer", activities "New activity" — completing UI-SPEC's Activities header row); back-nav affordances left of title on detail pages (contracts/customers/activities); activities ancestry breadcrumb left of title
- Header overflow rule applied where the 48px band can't hold everything: contracts status filter stays pinned atop the table in Body; contract-detail "Adopted from" line moved into Body
- All wrapped pages scroll inside Body (`h-full overflow-y-auto` inner scroll container); the shell Header stays fixed
- In-page pending/error states (activity-detail loading / not-found) now render inside Body within the shell frame; leaf-level route `errorComponent` preserved verbatim (Phase 8 P0-4 convention, plan instruction)
- 4 pre-existing contracts typecheck errors fixed (deferred-items assigns contracts/* to 10-03): base-ui Select `string | null` → `string | undefined` (currency), `Customer.name` → `Customer.company_name`, post-create `navigate` missing required `search`, `setCurrency` passed as `onValueChange` handler

## Task Commits

Each task was committed atomically:

1. **Task 1: wrap time-entries + expenses pages** - `31596b7` (feat)
2. **Task 2: wrap exports + contracts + customers pages (+ owned contracts type fixes)** - `d4a7de6` (feat)
3. **Task 3: wrap activities pages + compliance sweep** - `4a697f7` (feat)

**Plan metadata:** pending (docs commit)

## Files Created/Modified

- `web/src/components/layout/index.ts` - barrel re-exporting `Header`/`Body` so `@/components/layout` resolves (acceptance criterion import path)
- `web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx` - `<Header><h1>Time</h1></Header>` + `<Body><div className="h-full overflow-y-auto"><Tabs className="p-2">…` (existing tree unchanged)
- `web/src/routes/_authenticated/expenses/-components/expenses-page.tsx` - same wrap, title "Expenses"
- `web/src/routes/_authenticated/exports/-components/exports-page.tsx` - h1 in Header; icon+subtitle+ExportForm in Body (`p-6 space-y-6`)
- `web/src/routes/_authenticated/contracts/-components/contract-list.tsx` - Header: h1 + search + Create CTA; Body: tabs + status filter + rows + dialogs (`p-6`)
- `web/src/routes/_authenticated/contracts/$id/-components/contract-detail.tsx` - Header: back nav + h1 + badges + Edit; Body: adopted-from + Details/Actions cards (`p-6 space-y-4`); + currency/customer type fixes
- `web/src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx` - navigate `search: { from: "owned" }` + Select `onValueChange` wrapper (type fixes only)
- `web/src/routes/_authenticated/customers/-components/customers-page.tsx` - Header: h1 + search + Add Customer; Body: card grid + dialogs (`p-6`)
- `web/src/routes/_authenticated/customers/-components/customer-detail.tsx` - Header: back nav + h1 + badges + Edit/Delete; Body: info/contracts cards (`p-6 space-y-4`)
- `web/src/routes/_authenticated/activities/-components/activity-list.tsx` - Header: h1 + search + "New activity" CTA; Body: tabs + rows + dialog (`p-6`)
- `web/src/routes/_authenticated/activities/-components/activity-detail.tsx` - Header: back nav + ancestry breadcrumb + h1 + badges + Edit/Delete; loading/not-found states shell-wrapped; Body: details card + children accordion + dialogs (`p-6 space-y-4`)

## Decisions Made

- **Layout barrel:** `@/components/layout` was not resolvable (no index); the barrel fixes the acceptance criterion import literally. Exports only `Header`/`Body` — minimal surface, zero behavior change.
- **Tabs stay in Body:** the action text's "tab triggers that fit the 48px band" conflicts with the binding acceptance criterion "existing control tree unmodified apart from being wrapped" and threat T-10-03-1 ("no re-layout"). Moving `TabsList` out of the `Tabs` root also breaks base-ui Tabs context wiring. Resolved toward the acceptance criteria: tab triggers remain in Body, Header carries only the title on time-entries/expenses.
- **Title sizing:** every shell title uses UI-SPEC Heading (`text-xl font-semibold`) regardless of the pre-wrap heading class (`text-lg` on contracts/customers lists, `text-2xl` on detail pages) — the locked shell contract unifies page-title typography.
- **Inner padding:** pages without a prior padding container adopt the UI-SPEC lg 24px (`p-6`) inner container inside Body (padding ownership rule); time-entries/expenses keep their existing `p-2` per plan instruction.
- **Scroll ownership:** all wrapped pages wrap content in `h-full overflow-y-auto` inside Body — long content scrolls inside Body, the window never scrolls (UI-SPEC Scrolling row).
- **Error-state shell:** leaf `errorComponent` (P0-4 Phase 8 convention) preserved verbatim per plan instruction; the shell frame (sidebar) survives loader errors. In-page query loading/not-found states now render inside Body with a Header band — the maximal shell-survival achievable without restructuring the route error boundary (which the plan forbids).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Pre-existing contracts typecheck errors fixed (deferred-items 10-03 ownership)**
- **Found during:** Task 2 (verification — `bun run typecheck` gate)
- **Issue:** 4 baseline errors lived in the contracts files this plan wraps: `contract-detail.tsx(220)` base-ui Select `string | null` vs `string | undefined` (currency), `contract-detail.tsx(280)` `Customer.name` (no such prop — `company_name`), `create-contract-dialog.tsx(101)` post-create navigate missing required `search`, `create-contract-dialog.tsx(151)` `Dispatch<SetStateAction<string>>` not assignable to Select `onValueChange`.
- **Fix:** `currency: v ?? undefined`; `cust.company_name`; `search: { from: "owned" }`; `onValueChange={(v) => setCurrency(v ?? "EUR")}`.
- **Files modified:** contract-detail.tsx, create-contract-dialog.tsx
- **Verification:** typecheck errors in contracts files → 0 (baseline 10 → 6; the remaining 6 are out-of-scope: api.test.ts, __root.tsx, _auth forms ×3, unit-detail-panel)
- **Committed in:** d4a7de6 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required for the task's own `typecheck` acceptance gate; deferred-items explicitly assigned these errors to 10-03. No scope creep — both files were already in the plan's wrap surface (contract-detail) or its direct dependency (create-contract-dialog, imported by contract-list).

## Issues Encountered

- **Baseline typecheck rot (10 errors):** unchanged except the 4 contracts errors this plan was assigned to fix — now 6 remaining, all in out-of-phase files (api.test.ts, __root.tsx, _auth bootstrap/invite/password-reset forms, org-hierarchy unit-detail-panel). The plan's literal `typecheck exits 0` criterion remains technically unattainable at baseline; this plan's files contribute zero errors (grep-verified).
- **Action/acceptance tension on tab triggers:** task 1 action suggests moving "tab triggers that fit the 48px band" into the Header, while the same task's acceptance criterion mandates the existing control tree remain unmodified apart from being wrapped. Resolved in favor of the acceptance criteria (see Decisions) — this is an interpretation, not a deviation.

## Known Stubs

- `activity-detail.tsx` — "Adoption Count" renders "—" (ActivityDetail payload carries no `adoption_count`). Pre-existing, documented in 10-01; unchanged by the wrap.
- No other stubs: all wrapped surfaces are wired to live data with existing interactions intact.

## Threat Flags

None — no new network endpoints, auth paths, file access, or schema changes. Plan threats addressed:

- T-10-03-1 (wrap subtly changes layout): wrap-only rule honored — content trees unmodified apart from wrapping; per-page e2e gates (time-entries 11, expenses 11, contracts+customers 12) all green; full suite 41/41.
- T-10-03-2 (error/pending bypass the shell): in-page pending/error UI (activity-detail loading/not-found) now renders inside Body; leaf-level errorComponent preserved per plan instruction with the AppShell sidebar frame surviving loader errors (error-boundary.spec.ts 4/4 green).
- T-10-03-3 (Header overflow): overflow rule applied — contracts status filter stays in Body pinned atop the table; contract-detail "Adopted from" line moved into Body.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 10-04 (Today):** the `_authenticated/index.tsx` landing (currently a `<Navigate to="/time-entries">` redirect) is the only remaining unwrapped authenticated page — it was intentionally excluded from this plan's sweep and 10-04 replaces it with the Today composition (which per UI-SPEC renders its Display-28px title in the shell Header).
- **Ready for phase-level UAT:** manual visual walk against the UI-SPEC focal-point table deferred to phase verification per plan — all pages now render through the locked shell so the walk can verify the header band + Body surface consistently.
- **Blockers:** the 6 remaining pre-existing typecheck errors (deferred-items.md) are outside this phase's surface; none block later Phase 10 plans.

---
*Phase: 10-information-architecture-implementation*
*Completed: 2026-07-31*

## Self-Check: PASSED

- All 10 created/modified files exist on disk (11/11 FOUND); all 3 task commits verified in git log (31596b7, d4a7de6, 4a697f7)
- `bun run test`: 108/108 unit tests pass (14 files)
- `bunx playwright test`: 41/41 e2e pass (full suite incl. error-boundary 4/4, activities, org-hierarchy, all carried-over surfaces)
- `bun run lint`: 0 errors (126 pre-existing warnings, none in plan files — same baseline as 10-02)
- `bun run typecheck`: zero errors in all plan files; 6 pre-existing out-of-scope errors remain (down from 10 — the 4 contracts errors assigned to this plan are resolved; deferred-items.md)
