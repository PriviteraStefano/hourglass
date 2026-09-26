# Phase 18: Chrome contract - Research

**Researched:** 2026-08-29
**Domain:** Docs-only design contract — app shell / chrome (frame, navigation, role-scoped chrome, page anatomy)
**Confidence:** HIGH (every factual claim below was read from source this session; see `[VERIFIED: path:line]` tags)

> All discrete values, signatures, and nav model below are quoted verbatim from source read in this session. Phase 17 patterns and the ADR-P-011 input are likewise sourced this session.

---

## User Constraints

Copied verbatim from `18-CONTEXT.md` (`/Users/stefanoprivitera/Projects/hourglass/.planning/phases/18-chrome-contract/18-CONTEXT.md`). These are LOCKED — the planner must honor them, not explore alternatives.

### Navigation (revise ADR-P-011)
- **D-18-01:** Revise, do not adopt. CHROME.md re-litigates the navigation model; ADR-P-011 is treated as an input to *confirm or revise* per SC3, NOT silently kept as the chrome. — **Reversibility:** costly — the nav taxonomy is consumed by every page and by the Phase 20 composition map; changing the group set later means re-reconciling both.
- **D-18-02:** Lifecycle semantic subsets. The top-level nav is **four lifecycle groups**: `Record` (capture facts) · `Organize` (structure) · `Approve` (governance) · `Report` (outputs). This replaces ADR-P-011's job-language/role-grouped nav (Today/Track/Work/People/Economics/Review/Reports/Admin). Rationale (user, verbatim): *"we should try and reorganize all features (new and old) in semantic subsets, this will help us in future ABAC implementation."* — **Reversibility:** one-way for the ABAC intent — these groups are the seed of the future ABAC resource taxonomy; relabeling later re-lays that foundation.
- **D-18-03:** `Today` is the landing route `/`, outside the four lifecycle groups (carries ADR-P-011 D-2: read-only compose view, never blank). CHROME.md notes `Today` as the landing, not a nav group.
- **D-18-04:** Enumeration depth = groups + principle + clear-end features only. CHROME.md enumerates the obvious mappings (Time entries / Expenses / Tickets → `Record`; Approvals → `Approve`; Exports → `Report`) and states the lifecycle principle. The full middle mapping (Activities, Working Groups, Org/Units, Availability, Contracts, Customers, Direction, Coverage → `Organize`) is recorded as a **delta for Phase 20 composition** to finalize. — **Reversibility:** reversible (mapping is data, deferred).
- **D-18-05:** Role-visibility granularity = **mechanism only**. CHROME.md defines the role-scoping mechanism (predicate-driven, built on `web/src/lib/role-visibility.ts`) and records the detailed per-role / per-feature visibility matrix as a Phase 19/20 reconciliation input. It does NOT resolve the matrix now. — **Reversibility:** reversible (matrix is deferred data).

### Frame layout
- **D-18-06:** Frame = collapsible left sidebar + a top bar that *is* the PageHeader + a content region. This confirms the **current** app state (user, verbatim): *"Confirm the current state, which is a collapsible sidebar and the top bar that wraps around the feature content."* The sidebar holds identity (org-switcher, profile menu, theme toggle) **and** the four lifecycle nav groups. There is **no separate global identity top bar** — identity chrome lives in the sidebar. — **Reversibility:** reversible (documents existing code).
- **D-18-07:** The top bar is the PageHeader (per-page). Row 1: feature title + primary action button. Row 2: tabs + filter controls (via the Phase 15 frozen `FilterBar` component). — **Reversibility:** reversible.

### Role-scoped chrome
- **D-18-08:** A single predicate-driven role-visibility mechanism (`web/src/lib/role-visibility.ts`) governs **both** nav show/hide **and** page-action enable/disable. Detailed per-role manifests (which actions are disabled, read-only banners, hover states) are Phase 19 role-contract territory, not chrome. — **Reversibility:** costly — the mechanism is shared by nav and actions; changing it ripples across both.
- **D-18-09:** Customer is unspecced in ADR-P-011 and CUST-01 (Phase 19) may conclude "no app surface." Chrome treats the customer as deferred: no chrome element assumes a customer surface exists. The customer entry is resolved in Phase 19, not here.

### Page anatomy
- **D-18-10:** Content region = **shell rules only**. CHROME.md pins: max content width, vertical scroll behavior, `EmptyState` when no data, and `ConfirmDialog` for destructive actions. Detailed list/detail/table layouts (master-detail, form sections) are Phase 19 workflow-contract territory. — **Reversibility:** reversible.
- **D-18-11:** Frozen components are contract inputs, **absent on disk**. PageHeader / FilterBar / DataTable / EmptyState / ConfirmDialog are referenced as the intended implementers; CHROME.md records they are NOT yet present in `web/src/components/shared/` (D-17-45/122). Job-cluster implementation (post-Phase 20) builds them. The chrome contract MUST NOT claim they exist.

### The agent's Discretion
- CHROME.md formatting follows `LANGUAGE.md`'s established pattern: dual-audience, RFC 2119 (`MUST`/`MUST NOT`/`SHOULD`), changelog, authority stack, and a "Not in this file" fence (D-17-02/06/09/13/20).
- Exact ordering of the four lifecycle groups, and the exact filter-control composition within PageHeader row 2, are planner/researcher discretion consistent with the principle.
- Visual styling of chrome elements stays in `LANGUAGE.md` / CSS; CHROME.md adds layout / copy / composition only and MUST NOT override `LANGUAGE.md` (D-17-09).

### Deferred Ideas (out of scope — ignore completely)
- **Future ABAC implementation** — the lifecycle subsets are the resource taxonomy it would consume. ABAC itself is NOT in v0.2.1 scope (current model is RBAC). Recorded as intent only; not Phase 18 work.
- **Admin/Settings chrome** — explicitly out of scope (SC2); ADR-P-011's Admin group is excluded from the chrome.
- **Full feature→subset mapping + per-role visibility matrix** — Phase 19 (role contracts) / Phase 20 (composition map) reconcile.
- **Detailed page bodies** (list/detail/master-detail/form layouts) — Phase 19 workflow contracts.
- **Job-cluster implementation** (build PageHeader/FilterBar/DataTable/EmptyState/ConfirmDialog) — inserted after Phase 20.
- **Customer surface** — CUST-01 (Phase 19) may conclude "no app surface"; chrome defers to that decision (D-18-09).
- **Sketch sessions** — SKETCH-01 is Phase 20; chrome contract is docs-only (SC4), no sketch.

---

## Summary

Phase 18 is a docs-only contract (`docs/design/CHROME.md`) that becomes the source of truth for the app shell: the frame (collapsible sidebar + per-page PageHeader + content region), the navigation model (four lifecycle groups `Record`/`Organize`/`Approve`/`Report` + `Today` landing), role-scoped chrome (predicate-driven, built on the existing `role-visibility.ts`), and page anatomy (shell rules only). It revises — not adopts — ADR-P-011's pillar-mapped nav (Today/Track/Work/People/Economics/Review/Reports/Admin) and explicitly excludes Admin/Settings.

Investigation of the live frontend confirms the **current** app state matches D-18-06 exactly: `AppShell` renders only `AppSidebar` + a `SidebarInset` content region (no global identity bar); identity chrome (OrgSwitcher, ProfileMenu, ThemeToggle) lives in the sidebar; each page composes its own `Header`+`Body`. The existing `sidebar.tsx` still groups by the ADR-P-011 job-language set and `role-visibility.ts` currently gates **only** nav groups (Economics / Review / Admin + approval-stage derivation) — there is **no** page-action predicate yet, so CHROME.md must define the page-action extension as a target. The five frozen components named in D-18-11 are confirmed absent from `web/src/components/shared/`; only `status-badge.tsx` (raw-Tailwind debt) and feature-specific `entries-*` files exist.

**Primary recommendation:** Author `docs/design/CHROME.md` as a dual-audience, RFC 2119 contract that (a) documents the *current* frame accurately, (b) specifies the **target** four-lifecycle nav + `Today` landing and the target two-row PageHeader/page anatomy using the absent frozen components *as intended implementers, never as existing*, (c) defines the single predicate-driven role-visibility mechanism (nav + page-action) built on `role-visibility.ts`, (d) pins the shell rules for page anatomy, and (e) carries a "Not in this file" fence + authority stack deferring type/color/density/motion/status to `LANGUAGE.md`. Treat ADR-P-011 as a revised input and defer the middle `Organize` mapping + full visibility matrix to Phase 19/20. INDEX.md already reserves `CHROME.md` correctly — likely no change.

---

## Existing Code State (read-only confirmation)

### Frame layout — sidebar + content region, identity chrome in sidebar
[VERIFIED: web/src/components/layout/app-shell.tsx:1-11]
- `AppShell` = `<SidebarProvider><AppSidebar /><SidebarInset className="overflow-clip">{children}</SidebarInset></SidebarProvider>`. **No global top/identity bar is rendered by the shell.** This confirms D-18-06 ("no separate global identity top bar").

[VERIFIED: web/src/components/layout/sidebar.tsx:163-223]
- `AppSidebar` renders `Sidebar variant="inset" collapsible="icon"` with:
  - `SidebarHeader` → `<Suspense><OrgSwitcher /></Suspense>` (line 166-168) — **identity chrome (org) lives in the sidebar header**.
  - `SidebarContent` → the nav groups (regrouped target; today uses ADR-P-011 groups — see below).
  - `SidebarFooter` → `<ProfileMenu />` + `<ThemeToggle />` (line 219-222) — **identity chrome (profile + theme) lives in the sidebar footer**.
- `ThemeToggle` is imported at sidebar.tsx:41; `OrgSwitcher`/`ProfileMenu` at 42-43. All three identity elements are sidebar-resident. No separate top bar.

[VERIFIED: web/src/components/app/org-switcher.tsx:25-103]
- `OrgSwitcher` reads `AuthApis.profileQueryOpts` + `AuthApis.membershipsQueryOpts` (suspense), switches org via `switchOrganizationMutationOpts`, clears + invalidates `auth/me`. Confirms identity chrome is a sidebar component.

[VERIFIED: web/src/components/app/profile-menu.tsx:17-76]
- `ProfileMenu` renders avatar + dropdown (Profile / Log out). Pure identity chrome, sidebar-resident.

[VERIFIED: web/src/routes/_authenticated.tsx:7-18]
- The `_authenticated` layout's `beforeLoad` hydrates auth via `client.fetchQuery(AuthApis.profileQueryOpts)` (`GET /auth/me`) and redirects to `/login` on failure; role is therefore available client-side at render time (ADR-P-011 consequence; backs D-18-05/D-18-08).

### Frame layout — per-page header is the PageHeader stand-in (frozen PageHeader absent)
[VERIFIED: web/src/components/layout/header.tsx:1-21]
- `Header` = `<header className="flex h-12 shrink-0 items-center bg-sidebar gap-2 px-4">{children}</header>` — a bare 48px band. This is the current per-page header placeholder (the "top bar that wraps the feature content" per D-18-06).
- Barrel re-export: `web/src/components/layout/index.ts` line 1: `export { Header } from "./header.tsx"; export { Body } from "./body.tsx";`.

[VERIFIED: web/src/components/layout/body.tsx:1-21]
- `Body` = `<div className="flex-1 rounded-lg bg-background overflow-clip shadow-lg ring-1 ring-sidebar-border m-0.5">{children}</div>` — the content region wrapper.

[VERIFIED: grep `from "@/components/layout"` across web/src/routes/_authenticated/**]
- Nine authenticated pages import `{ Header, Body }`: `activities/-components/activity-detail.tsx`, `activity-list.tsx`, `contracts/$id/-components/contract-detail.tsx`, `contracts/-components/contract-list.tsx`, `customers/-components/customer-detail.tsx`, `customers-page.tsx`, `expenses/-components/expenses-page.tsx`, `exports/-components/exports-page.tsx`, `time-entries/-components/time-entries-page.tsx`, `working-groups/-components/working-groups-page.tsx`. So the per-page header pattern is established and used everywhere — but it is the lightweight `Header` placeholder, NOT the frozen two-row `PageHeader`.

[VERIFIED: web/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx:34-64]
- Current page-header usage does **not** match the D-18-07 target: `<Header><h1 className="text-xl font-semibold">Time</h1></Header>` (title only, no primary action), and the `Tabs` (list/calendar/export) are placed *inside* `<Body>`, not in a Header row 2. No `FilterBar` usage. So the live code is a single-row title band + tabs-in-body — the two-row PageHeader (title+action / tabs+FilterBar) is a **target**, not current state.

### Navigation — sidebar still uses ADR-P-011 job-language groups (target is four lifecycle groups)
[VERIFIED: web/src/components/layout/sidebar.tsx:65-130]
- Current `navStructure` (verbatim group/label/href; `disabled` + tooltip quoted where present):
  ```ts
  { group: null,      items: [{ label: "Today", href: "/" }] },
  { group: "Track",   items: [
      { label: "Time", href: "/time-entries" },
      { label: "Expenses", href: "/expenses" },
      { label: "Tickets", href: "/tickets", disabled: true,
        tooltip: "Tickets arrive in v0.2" } ] },
  { group: "Work",    items: [
      { label: "Activities", href: "/activities" },
      { label: "Working Groups", href: "/working-groups" } ] },
  { group: "People",  items: [
      { label: "Org", href: "/org-hierarchy" },
      { label: "Availability", href: "/availability", disabled: true,
        tooltip: "Availability lands with the staffing schema" } ] },
  { group: "Economics", items: [
      { label: "Contracts", href: "/contracts" },
      { label: "Customers", href: "/customers" } ] },
  { group: "Review",  items: [{ label: "Approvals", href: "/approvals" }] },
  { group: "Reports", items: [{ label: "Exports", href: "/exports" }] },
  { group: "Admin",   items: [
      { label: "Settings", href: "/settings", disabled: true } ] },
  ```
- This is **exactly** the ADR-P-011 D-1 pillar set (Today/Track/Work/People/Economics/Review/Reports/Admin). It does **not** use the four lifecycle groups yet. Confirms CONTEXT's "currently groups by ADR-P-011 … must be regrouped" — CHROME.md therefore describes a **target** not yet in code (consistent with docs-only Phase 18; regrouping lands in a post-Phase-20 job cluster).
- `Today` is the `group: null` ungrouped top item at `/` — confirms D-18-03 (landing, outside the groups).

[VERIFIED: web/src/routes/_authenticated/-components/today-page.tsx exists; `_authenticated/index.tsx` renders it]
- `Today` is the landing compose view at `/` (ADR-P-011 D-2: read-only, never blank). No contradiction with D-18-03.

### Role-visibility wiring today (inline switch, nav-only)
[VERIFIED: web/src/components/layout/sidebar.tsx:144-161]
- The sidebar derives `stages = deriveApprovalStages(profile, workingGroups)` and `role = profile.membership.role`, then filters `navStructure` by group via an inline `switch`:
  ```ts
  case "Economics": return isEconomicsVisible(role);
  case "Review":    return isReviewVisible(role, stages);
  case "Admin":     return isAdminVisible(role); // false for every v0.1 role
  default:          return true;
  ```
- Composition is currently a **hardcoded switch in the sidebar**, not a generic predicate registry. CHROME.md can propose a unified predicate registry (surface/action → predicate) that both nav rendering and page-action enable/disable consume — see Open Questions.

---

## Role-Visibility Mechanism API (build target for D-18-05/08)

[VERIFIED: web/src/lib/role-visibility.ts:1-67 — full file, quoted verbatim signatures]

```ts
// line 10
export type ApprovalStage = "manager" | "finance";

// lines 25-48
export function deriveApprovalStages(
  profile: UserWithMembership,
  workingGroups: WorkingGroup[] | undefined,
): ApprovalStage[]

// lines 51-53
export function isReviewVisible(role: Role, stages: ApprovalStage[]): boolean

// lines 56-58
export function isEconomicsVisible(role: Role): boolean

// lines 64-67
export function isAdminVisible(role: Role): boolean
```

**Behavior (verbatim from source):**
- `deriveApprovalStages` (lines 25-48): `hr` → `[]` (HR never holds a stage, ADR-P-008 D-4); `finance` → `["finance"]`; `manager` → `["manager"]`; any non-hr role (incl. employee, customer) gains `"manager"` when the user id matches a WG `manager_id` or appears in `delegate_ids`; `undefined` WG list → org-role stages only.
- `isReviewVisible` (51-53): `stages.length > 0 && role !== "hr"` — i.e. Review renders only for approvers, never HR.
- `isEconomicsVisible` (56-58): `role !== "employee" && role !== "customer"` — Economics hidden from employee and customer.
- `isAdminVisible` (64-67): `return false` unconditionally — Admin hidden from every v0.1 role (no org-admin role yet).

**Inputs/types:** `Role`, `UserWithMembership`, `WorkingGroup` from `@/types` (role-visibility.ts:1). The role set referenced in code is **5 roles**: `employee | manager | finance | hr | customer` (HR is handled explicitly at lines 33/52).

**Surfaces currently gated (nav groups only):**
- `Economics` group visibility → `isEconomicsVisible(role)`
- `Review` group visibility → `isReviewVisible(role, stages)`
- `Admin` group visibility → `isAdminVisible(role)` (always false today)
- Approval-stage derivation (drives Review group + Approvals page stage tabs) → `deriveApprovalStages`

**Critical finding for the planner (GAP A):** The mechanism today gates **only nav-group show/hide**. There is **no** page-action enable/disable predicate (no `isActionEnabled(...)`, no per-feature/per-action predicate). D-18-08 requires CHROME.md to define the mechanism as governing **both** nav and page-actions. Therefore CHROME.md must specify the page-action extension as a **target contract**; the live `role-visibility.ts` is the seed (pure, testable predicates) but must be extended (new predicate(s) for actions, shared registry) in a later job-cluster phase. This is consistent with D-18-05 ("mechanism only", matrix deferred) and D-18-08 ("detailed per-role manifests … are Phase 19 territory").

**Authority note (verbatim from source, role-visibility.ts:3-9):** *"UX scoping only — every role-restricted surface stays backend-gated. Hiding a group here must never be mistaken for authorization."* CHROME.md should carry the same warning (mirrors ADR-P-011 D-5).

---

## Frozen-Component Confirmation (D-18-11)

[VERIFIED: glob web/src/components/shared/ — directory listing]
- Present in `web/src/components/shared/`: `entries-filters.tsx`, `status-badge.tsx`, `entries-table.tsx` (plus `__tests__/`).
- **Absent** from `web/src/components/shared/`: `page-header.tsx`, `filter-bar.tsx`, `data-table.tsx`, `empty-state.tsx`, `confirm-dialog.tsx`.

[VERIFIED: docs/design/LANGUAGE.md:119-130 — the authoritative frozen-component input table]
- The Phase 17 contract lists the frozen set with live status (2026-08-26), which matches the live tree today:
  | Input | Live tree |
  |-------|-----------|
  | StatusBadge | Present at `web/src/components/shared/status-badge.tsx` — raw Tailwind, not role tokens |
  | PageHeader | Absent |
  | FilterBar | Absent |
  | DataTable | Absent |
  | EmptyState | Absent |
  | ConfirmDialog | Absent |
- LANGUAGE.md:130 also notes `entries-table.tsx` and `entries-filters.tsx` are present but are **NOT** the frozen set (feature-specific time-entries components).

[VERIFIED: web/src/components/shared/status-badge.tsx exists; LANGUAGE.md:76, D-17-44]
- `status-badge.tsx` is present but uses raw Tailwind palettes (`yellow`/`blue`/`green`/`purple`/`emerald`/`red` with `dark:` variants) — known historical debt. **CHROME.md MUST NOT modify it**, and MUST NOT claim StatusBadge/PageHeader/FilterBar/DataTable/EmptyState/ConfirmDialog exist.

**Conclusion for D-18-11:** Confirmed. The five frozen components named in D-18-11 (PageHeader, FilterBar, DataTable, EmptyState, ConfirmDialog) are **absent on disk**. `StatusBadge` is the one frozen component that *did* land (as raw-Tailwind debt). The feature-specific `entries-table.tsx` / `entries-filters.tsx` are NOT the generic frozen set. CHROME.md must reference all six as **intended implementers / inputs**, never as existing.

[VERIFIED: glob web/src/components/layout/ — directory listing]
- `layout/` contains: `sidebar.tsx`, `index.ts`, `route-error.tsx`, `body.tsx`, `header.tsx`, `app-shell.tsx` (+ `__tests__/sidebar-groups.test.tsx`, `route-error.test.tsx`). No `page-header.tsx` or `filter-bar.tsx` in layout either.
- `sidebar-groups.test.tsx` exists — the existing nav-group structure is unit-tested; a later regroup (D-18-02) will need this test updated (flag for planner/job-cluster phase).

---

## Design-Authority Constraints (D-17-09; CHROME.md must not override LANGUAGE.md)

[VERIFIED: docs/design/LANGUAGE.md:1-142 (full) and docs/design/INDEX.md:1-8 (full)]

### Authority stack (verbatim, LANGUAGE.md:7-9)
- *"CSS wins on values (the literal oklch/px/rem numbers). `LANGUAGE.md` wins on meaning and usage … LANGUAGE.md wins on type, color, density, motion, and status vocabulary over every later presentation doc."*
- *"Authority stack (D-17-09): This file wins on type, color, density, motion, and status vocabulary. Later design docs (`CHROME.md`, workflow contracts under `docs/design/workflows/`, `COMPOSITION.md`) and later GSD `UI-SPEC.md` files may only add surface, layout, copy, or composition. They `MUST NOT` override this file."*
- **Implication for CHROME.md:** it may add surface / layout / copy / composition only. It MUST NOT redefine type roles, color roles (incl. `accent` as interaction, `destructive` as action-not-status), density rhythm, motion roles, or status vocabulary (`neutral/info/success/warning/danger`). Visual styling of chrome stays in `LANGUAGE.md`/CSS.

### INDEX.md already reserves CHROME.md (likely no Phase 18 edit needed)
[VERIFIED: docs/design/INDEX.md:5-7]
```
- `docs/design/CHROME.md` (reserved — Phase 18) — app-shell / chrome contract (frame, navigation, role-scoped chrome, page anatomy).
```
- The reserved line is correct and complete. Per 18-CONTEXT.md write-set note #2, INDEX.md likely needs **no change**. If CHROME.md's scope wording needs adjusting, edit only that one reserved line.

### Established CHROME.md doc pattern to mirror (from Phase 17, D-17-02/06/09/13/16/20/21)
[VERIFIED: .planning/phases/17-design-language-contract/17-CONTEXT.md:31-54 and 17-01-PLAN.md:67-101]
- **Dual-audience**, one file, no agent-only section (D-17-02).
- **RFC 2119 selectively**: `MUST`/`MUST NOT` for hard conformance, `SHOULD` for defaults, rationale in prose (D-17-06).
- **Authority stack** statement included (D-17-09).
- **Changelog**: compact dated entries keyed by GSD phase, latest first, no semver, no Keep-a-Changelog headings; form `2026-08-29 · Phase 18 · Added chrome contract` (D-17-05/21).
- **"Not in this file" fence** (D-17-16): explicit list of what is out of scope — must include: no type/color/density/motion/status vocabulary (those are LANGUAGE.md), no component API catalog, no workflow/copy contracts (Phase 19), no composition map (Phase 20), no Admin/Settings chrome (SC2), no token-value tables, no screenshots.
- **Section order to mirror (D-17-20):** Title → Purpose → changelog → (foundations/contract body) → do/don't → pointers → not-in-this-file → (input notes) → Gaps. For CHROME.md the "foundations" become Frame / Navigation / Role-scoped chrome / Page anatomy; the "Gaps" become the deferred middle `Organize` mapping + the page-action predicate extension + the per-role visibility matrix (all deferred to Phase 19/20).
- **Do/don't** is Don't-only and must mirror already-stated `MUST NOT`s (D-17-22).
- **Pointers** are repo-root citations only (`web/src/index.css`, `web/components.json`, `web/src/lib/role-visibility.ts`, `web/src/components/layout/sidebar.tsx`) — no markdown links to CSS/config (D-17-18).
- **15-UI-SPEC note equivalent:** CHROME.md should note `15-UI-SPEC.md` is historical input / not authority, and list the frozen components as inputs with live status (D-17-11/45).

### INDEX.md update pattern (from 17-02-PLAN.md — for reference if a CHROME edit is needed)
[VERIFIED: 17-02-PLAN.md:56-85] — INDEX.md is a simple path map: title + one-sentence intro + path list with one-line purposes. Existing files are markdown links; reserved paths are backtick + `(reserved — Phase N)`. No stack, no MUST NOTs, no Gaps. Editing the CHROME reserved line (if at all) must keep this style.

---

## ADR-P-011 Revision Analysis (D-18-01/02/03/04)

[VERIFIED: hourglass-vault/decisions/project/ADR-P-011 — Information Architecture & Role-Scoped Surfaces.md:1-99 (full)]

### Current nav model (input to revise, not adopt)
ADR-P-011 **D-1** (lines 28-41) — pillar-mapped, job-language groups (verbatim group→items→pillar):
| Group | Items | Pillar |
|-------|-------|--------|
| *(landing)* | **Today** `/` | Insight |
| **Track** | Time · Expenses · Tickets | Capture |
| **Work** | Activities · Working Groups | Structure |
| **People** | Org · Availability | Structure |
| **Economics** | Contracts · Customers | Structure (commercial) |
| **Review** | Approvals | Control |
| **Reports** | Exports | Insight |
| **Admin** | Invitations · Activity kinds · Roles/Settings | Control |

ADR-P-011 **D-2** (43-47): Landing is Today from v0.1; ticketless composition admitted; read-only, never blank.
ADR-P-011 **D-5** (57-72): role-scoped visibility matrix (✓ full / read / — hidden); key rows (verbatim intent):
- Today: ✓ all roles except Customer (—).
- Track: ✓ all except Customer (—).
- Work: read (employee) / ✓ form-edit (manager) / read (finance, hr) / read-own (customer, unspecced).
- People: declare-own-windows (employee) / subtree+holiday-confirm (manager) / read (finance) / **curator all** (hr) / — (customer).
- Economics: — (employee) / read (manager) / ✓ (finance) / read payroll-link (hr) / read-own (customer, unspecced).
- Review: — (employee) / manager-stage (manager) / finance-stage (finance) / **✗ never** (hr) / — (customer).
- Reports: own (employee) / subtree (manager) / org-wide (finance) / payroll-view (hr) / — (customer).
- Admin: — all except "org admin only" (no such role in v0.1 → effectively hidden; matches `isAdminVisible` returning false).

ADR-P-011 **D-3** (49-51): Review is its own group, role-gated; HR never sees Review.
ADR-P-011 **D-6** (74-78): route naming follows ontology (`/projects`→`/activities`, `/working-groups`, `/availability`, `/approvals`, `/` reserved for Today).

### How the four lifecycle groups map onto the existing feature set (research proposal for the planner)
Derived from D-18-02/04 + the live `navStructure` (sidebar.tsx:65-130) + ADR-P-011 D-1 items. This is the **enumeration-depth** mapping CHROME.md should state (groups + principle + clear-end features); the middle is a Phase 20 delta.

| Lifecycle group | Clear-end features (state now) | Deferred middle (Phase 20 delta) |
|-----------------|--------------------------------|-----------------------------------|
| **Today** (landing, not a group) | `/` read-only compose view | — |
| **Record** (capture facts) | Time `/time-entries`, Expenses `/expenses`, Tickets `/tickets` (v0.2, currently disabled placeholder) | — |
| **Organize** (structure) | Activities `/activities`, Working Groups `/working-groups`, Org `/org-hierarchy`, Availability `/availability` (P-008, currently disabled placeholder), Contracts `/contracts`, Customers `/customers` | Direction (backend P-13, no UI yet), Coverage (backend P-12, no UI yet) |
| **Approve** (governance) | Approvals `/approvals` | — |
| **Report** (outputs) | Exports `/exports` | — |
| *(excluded)* **Admin/Settings** | Settings `/settings` (disabled placeholder) | **Out of scope (SC2)** — ADR-P-011 Admin group excluded from chrome |

- This matches D-18-04's stated clear-end mappings exactly (Time/Expenses/Tickets → Record; Approvals → Approve; Exports → Report) and its deferred middle list (Activities, Working Groups, Org/Units, Availability, Contracts, Customers, Direction, Coverage → Organize).
- **ABAC seed intent (D-18-02):** the four lifecycle groups are the seed of the future ABAC resource taxonomy. CHROME.md should state this intent (reversibility note) but NOT design ABAC (deferred; current model is RBAC).

### Revision posture for CHROME.md
- CHROME.md **revises** ADR-P-011 D-1 (replaces the 7+1 job-language groups with 4 lifecycle groups + Today landing) and **keeps** ADR-P-011 D-2 (Today read-only/never-blank), D-3 (Review role-gated, HR excluded), D-5 (visibility is UX scoping, backend stays authoritative), D-6 (route names stay). Admin/Settings (D-1 last row) is dropped from chrome (SC2).
- The per-role visibility **matrix** from D-5 is carried as an *input*; CHROME.md records it and defers the authoritative per-role/per-feature reconciliation to Phase 19 (role contracts) + Phase 20 (composition map), per D-18-05.
- Flag: the live `navStructure` still encodes the ADR-P-011 groups; `sidebar-groups.test.tsx` asserts them. CHROME.md describes the target; the migration is a later job-cluster task (post-Phase 20), not Phase 18.

---

## Contradictions / Gaps Between Locked CONTEXT and Live Code

All flagged items are **consistent with** the locked decisions (the decisions explicitly describe target/deferred states), but the planner must know the live code has **not** reached the target yet:

1. **GAP A — page-action predicate absent (D-18-08 vs code).** `role-visibility.ts` gates nav groups only; no page-action enable/disable predicate exists. CHROME.md defines the mechanism's intended scope (nav + actions) as a target. Not a decision conflict — D-18-08 is a contract requirement, and D-18-05 defers the matrix. Live mechanism = nav-only seed.
2. **GAP B — sidebar still uses ADR-P-011 groups (D-18-02/04 vs code).** Live `navStructure` = Today/Track/Work/People/Economics/Review/Reports/Admin. Four lifecycle groups are not in code. CONTEXT's code_context explicitly says "must be regrouped" — so this is expected, but CHROME.md must present the lifecycle groups as the **target**, and the planner must not assume regrouping is done. `sidebar-groups.test.tsx` will need updating when it lands.
3. **GAP C — per-page header is a bare `<h1>` band, not the two-row PageHeader (D-18-07 vs code).** Live pages put a single `<h1>` in `Header` and Tabs inside `Body`; frozen `PageHeader`/`FilterBar` are absent. CHROME.md must describe the target two-row anatomy and reference the absent frozen components as intended implementers (D-18-11). No contradiction — D-18-11 confirms absence.
4. **GAP D — AGENTS.md role list is stale (minor doc inaccuracy, not a locked-decision conflict).** `AGENTS.md:180` lists roles as `employee, manager, finance, customer` (4) and omits `hr`. The codebase (`role-visibility.ts` handles `hr` at lines 33/52), ADR-P-011 D-5 (HR column), and Phase 19 requirements (HR-01) all use **5 roles**. CHROME.md and the visibility mechanism should use the 5-role set (`employee | manager | finance | hr | customer`). Recommend the planner note this so CHROME.md doesn't under-scope HR.
5. **No contradiction found** on D-18-06 (frame = sidebar + per-page PageHeader + content; identity chrome in sidebar; no global identity bar — confirmed by `app-shell.tsx`), D-18-03 (Today landing at `/`, outside groups — confirmed), D-18-09 (customer deferred — confirmed, customer column mostly hidden/unspecced in ADR-P-011), D-18-10 (page anatomy shell rules — confirmed as target; frozen EmptyState/ConfirmDialog absent), D-18-11 (frozen components absent — confirmed).

---

## Open Questions for the Planner

1. **Predicate registry shape (D-18-08).** CHROME.md should specify a single predicate-driven mechanism for *both* nav and page-actions. Today it's an inline `switch` in `sidebar.tsx:150-161` over `is{Economics,Review,Admin}Visible`. Should CHROME.md mandate a unified registry (e.g. `surfaceVisibility: Record<SurfaceId, Predicate>` + `actionVisibility: Record<ActionId, Predicate>` sharing `deriveApprovalStages`/role inputs), or describe the mechanism in prose and leave the registry shape to the job-cluster phase? Recommend: specify the *contract* (one predicate type, shared role/stage inputs, nav + action consumers) and defer the concrete registry refactor to implementation.
2. **Lifecycle group ordering (discretion per CONTEXT).** D-18-02 names the four groups but leaves exact render order to planner discretion. Suggested order: `Record → Organize → Approve → Report` (capture → structure → govern → output), with `Today` pinned at top as the ungrouped landing. Planner to confirm.
3. **`Today` placement.** Confirmed landing at `/`, outside groups (D-18-03). Should CHROME.md also pin Today's compose-rule invariants (read-only, never blank, approval-stage-gated "Waiting on you" + "Your week")? These are ADR-P-004/P-011 D-2 — recommend a one-line pointer, full body in Phase 19.
4. **Deferred `Organize` middle + Direction/Coverage.** Direction (P-13) and Coverage (P-12) have **no UI route** today (absent from `navStructure`). CHROME.md records them as Phase 20 delta only; must not invent nav entries for them now. Confirm CHROME.md states they are backend-only today.
5. **Customer surface (D-18-09).** No chrome element may assume a customer surface. CHROME.md should state customer visibility is carried from ADR-P-011 D-5 (mostly hidden/unspecced) and resolved in Phase 19 (CUST-01 may conclude "no app surface"). Recommend a one-line "customer deferred" fence.
6. **INDEX.md edit.** Reserved `CHROME.md` line already correct (INDEX.md:6). Planner should confirm **no INDEX.md change** is needed (matches 18-CONTEXT.md write-set note #2). If scope wording is adjusted, edit only that one reserved line.
7. **Role-set cardinality (GAP D).** Confirm CHROME.md uses the 5-role set (`employee | manager | finance | hr | customer`), not AGENTS.md's stale 4-role list.
8. **Existing test impact.** `sidebar-groups.test.tsx` asserts the ADR-P-011 groups; a later regroup (post-Phase 20) updates it. CHROME.md is docs-only and must not touch tests, but the planner should note the test as a downstream touchpoint.

---

## Project Constraints (from AGENTS.md)

[VERIFIED: /Users/stefanoprivitera/Projects/hourglass/AGENTS.md:1-2, 130-149, 178-201]

- **Design gate (AGENTS.md:2):** *"Before any `web/src` change to UI, tokens, components, copy, or layout, open `docs/design/INDEX.md` first. Backend-only work skips this. No other design text belongs in `AGENTS.md`."* Phase 18 is docs-only (no `web/src` change) — it *authors* the referenced contract, so it complies by definition.
- **Protected routes (AGENTS.md:130-142):** `_authenticated.tsx` `beforeLoad` hydrates `GET /auth/me`; role available client-side. Backs D-18-05/08.
- **Roles (AGENTS.md:180):** lists `employee, manager, finance, customer` — **stale** (omits `hr`); see GAP D. Use the 5-role set.
- **Stack (AGENTS.md:23-27, 57-70):** React 19, TanStack Router v1, TanStack React Query v5, Vite, TypeScript, Tailwind CSS (v4 CSS-first), shadcn/ui (`base-mira`/olive per `web/components.json`), lucide-react icons. Frontend component files kebab-case; routes kebab-case; TanStack Router file-based with `index.tsx` folder routes.
- **No recreation of cancelled v0.2 Phases 17–26; contract-first sequence DL → chrome → roles → map → (sketch-loop amend iff needed) → sketch → implement (ROADMAP:46).** Phase 18 must stay docs-only.

---

## Standard Stack / Architecture Patterns (for plan authoring)

Docs-only phase — no new dependencies. CHROME.md will *reference* (not import) these live constructs:
- **TanStack Router** `_authenticated` layout as the auth-hydration shell (`web/src/routes/_authenticated.tsx`) — role available via route context after `beforeLoad`.
- **shadcn/ui `sidebar`** primitive (`web/src/components/ui/sidebar.tsx`) — `Sidebar`/`SidebarContent`/`SidebarHeader`/`SidebarFooter`/`SidebarGroup`/`SidebarMenu*`; `collapsible="icon"`, `variant="inset"`. CHROME.md documents the frame built on this.
- **`role-visibility.ts`** predicate module — the single mechanism seed (nav-only today; extend to actions per D-18-08).
- **`LANGUAGE.md`** authority for all visual vocabulary — CHROME.md adds surface/layout/copy/composition only.

**Don't hand-roll:** do not invent CSS tokens or restyle the shadcn kit in CHROME.md (LANGUAGE.md D-17-19); do not create `docs/design/workflows/` or `COMPOSITION.md` (Phase 19/20); do not modify `web/src`, `index.css`, `components.json`, or `15-UI-SPEC.md` (mirrors D-17-46); do not implement, sketch, or do route work (SC4).

---

*Research complete — all claims sourced from files read this session (see `[VERIFIED: path:line]` tags). No external/web lookup was required; this is an in-repo read-only documentation contract.*
