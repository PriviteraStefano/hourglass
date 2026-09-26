# Hourglass Chrome Contract

## Purpose

- **Authority split (D-17-09):** LANGUAGE.md wins on type, color, density, motion, and status vocabulary. This file adds surface, layout, copy, and composition only and MUST NOT override LANGUAGE.md. If a chrome surface needs a language change, amend `LANGUAGE.md` first.
- **RFC 2119 selectively:** `MUST` / `MUST NOT` for hard conformance, `SHOULD` for defaults, rationale in prose (D-17-06).
- **No invented values:** Do not invent CSS tokens or copy oklch/px/rem tables. Point at the live value stores by repo-root path.

## Changelog

- 2026-08-29 · Phase 18 · Added chrome contract

## Frame

The frame is the app shell that wraps every authenticated page (D-18-06/07).

- Frame `MUST` be a collapsible left `Sidebar` + a per-page `PageHeader` top bar + a content `Body` region.
- Identity chrome — `OrgSwitcher`, `ProfileMenu`, `ThemeToggle` — lives in the sidebar (header + footer). This file `MUST NOT` render a separate global identity top bar. This confirms the current `AppShell`, which renders only `AppSidebar` + `SidebarInset` (no global bar).
- The top bar `MUST` be the per-page `PageHeader` (D-18-07). Row 1 = feature title + primary action button. Row 2 = tabs + filter controls (via the frozen `FilterBar` component). `PageHeader` and `FilterBar` are referenced as the **intended implementers / absent inputs** (D-18-11), not existing components. The current `web/src/components/layout/header.tsx` `Header` is a 48px title-only placeholder and `Body` is the content wrapper, used by 9 authenticated pages (tabs currently inside `Body`). The two-row `PageHeader` is the **target spec**, not the current state, and this file `MUST NOT` claim the frozen `PageHeader` or `FilterBar` exist.

## Navigation

The navigation model **revises** ADR-P-011 — it does not adopt it (D-18-01, SC3). ADR-P-011 D-1 is replaced by this contract; ADR-P-011 is treated as an input to confirm or revise, not silently kept. This contract keeps ADR-P-011 D-2 (Today read-only / never blank), D-3 (Review role-gated, HR excluded), D-5 (visibility is UX scoping; backend authoritative), D-6 (route names stay), and drops the Admin group (SC2).

- Top-level nav `MUST` be four lifecycle groups (D-18-02): `Record` (capture facts) · `Organize` (structure) · `Approve` (governance) · `Report` (outputs). Features are organized into semantic lifecycle subsets to seed the future ABAC resource taxonomy (D-18-02).
- `Today` `MUST` be the landing route `/`, outside the four lifecycle groups (D-18-03); it carries ADR-P-011 D-2 (read-only compose view, never blank). `Today` is the landing, not a nav group.
- Enumeration depth `MUST` be groups + principle + clear-end features only (D-18-04): `Time entries / Expenses / Tickets → Record`; `Approvals → Approve`; `Exports → Report`. The full middle mapping (Activities, Working Groups, Org/Units, Availability, Contracts, Customers, Direction, Coverage → `Organize`) is recorded as a **delta for Phase 20 composition** to finalize; this file `MUST NOT` resolve it here.
- This file `MUST NOT` include an `Admin` / `Settings` nav group (SC2, success criterion 2). The live `navStructure` (`web/src/components/layout/sidebar.tsx`) still encodes the ADR-P-011 groups and `sidebar-groups.test.tsx` asserts them — this contract describes the **target**; the migration (regroup into four lifecycle groups + `Today`) is a later job-cluster task (post-Phase 20), not Phase 18.

## Role-scoped chrome

Role-scoped chrome is governed by a single predicate-driven mechanism (D-18-05/08/09).

- A single predicate-driven role-visibility mechanism (`web/src/lib/role-visibility.ts`) `MUST` govern **both** nav show/hide **and** page-action enable/disable (D-18-08). The current predicates are `deriveApprovalStages`, `isReviewVisible`, `isEconomicsVisible`, `isAdminVisible`; today they gate **nav groups only**. The page-action predicate is a target contract (GAP A) built in a later job-cluster phase.
- The ADR-P-011 D-5 per-role visibility matrix is carried as an **input**, not resolved here (D-18-05); the authoritative per-role / per-feature reconciliation is Phase 19 (role contracts) + Phase 20 (composition map). This file `MUST NOT` resolve the per-role visibility matrix here.
- Customer (`employee | manager | finance | hr | customer` — the 5 roles referenced in code) is deferred (D-18-09): no chrome element `MUST` assume a customer surface exists; CUST-01 (Phase 19) may conclude "no app surface". This file `MUST NOT` treat nav-group hiding as authorization.

> UX scoping only — every role-restricted surface stays backend-gated. Hiding a group here must never be mistaken for authorization.

## Page anatomy

The content region is governed by shell rules only (D-18-10/11).

- The content region `MUST` pin: max content width, vertical scroll behavior, `EmptyState` when no data, and `ConfirmDialog` for destructive actions. `EmptyState` and `ConfirmDialog` are referenced as **absent inputs** (D-18-11), not existing components, and this file `MUST NOT` claim they exist.
- Detailed list / detail / master-detail / form layouts are Phase 19 workflow-contract territory; this file `MUST NOT` specify them here.

## Do / don't

- Don't override `LANGUAGE.md` type/color/density/motion/status vocabulary.
- Don't claim PageHeader / FilterBar / DataTable / EmptyState / ConfirmDialog exist — they are frozen inputs, absent on disk (D-18-11).
- Don't invent a customer app surface (D-18-09 — deferred to Phase 19 CUST-01).
- Don't include Admin/Settings chrome (SC2).
- Don't resolve the per-role visibility matrix here (Phase 19/20, D-18-05/08).
- Don't treat nav-group hiding as authorization (`role-visibility.ts` is UX scoping only).

## Pointers

- `web/src/lib/role-visibility.ts` — predicate-driven role-visibility mechanism (seed for nav + page-action gating).
- `web/src/components/layout/sidebar.tsx` — existing collapsible sidebar (regroup target).
- `web/src/components/layout/header.tsx` + `body.tsx` — current per-page header/body placeholders.
- `web/src/index.css` — live tokens for type, color, density, light/dark maps.
- `web/components.json` — shadcn kit identity (`base-mira` / olive).

## Not in this file

This file is the chrome/shell contract only. Explicitly out of scope here:

- No type/color/density/motion/status vocabulary (that is `LANGUAGE.md`).
- No component API catalog or primitives inventory.
- No workflow/copy contracts (those are Phase 19).
- No composition map (that is Phase 20, `COMPOSITION.md`).
- No Admin/Settings chrome (SC2 — ADR-P-011 Admin group excluded).
- No token-value tables (oklch/hex/px) — values stay in CSS.
- No screenshots or worked examples.

## 15-UI-SPEC note

`docs/history/planning/phases/15-ux-foundation-design-tokens-shared-components/15-UI-SPEC.md` is untouched historical input and is **not** authority. If it conflicts with `LANGUAGE.md` on vocabulary, `LANGUAGE.md` wins.

The Phase 15 frozen components are listed here as *inputs* with live status (2026-08-29), not silently reused:

| Input | Live tree (2026-08-29) |
|-------|------------------------|
| StatusBadge | Present at `web/src/components/shared/status-badge.tsx` — raw Tailwind, not role tokens (known debt) |
| PageHeader | Absent |
| FilterBar | Absent |
| DataTable | Absent |
| EmptyState | Absent |
| ConfirmDialog | Absent |

`entries-table.tsx` and `entries-filters.tsx` are present in the live tree but are **not** the frozen set. The five absent components `MUST NOT` be claimed to exist in this file (D-18-11).

## Gaps

- `Organize` middle mapping — Activities, Working Groups, Org/Units, Availability, Contracts, Customers, Direction, Coverage → deferred to Phase 20 composition delta.
- `page-action` predicate — `role-visibility.ts` gates nav only today; the page-action enable/disable predicate is a target contract, built in a later job-cluster phase (GAP A).
- `per-role visibility matrix` — carried from ADR-P-011 D-5 as input; authoritative reconciliation is Phase 19 (role contracts) + Phase 20 (composition map).
- `customer surface` — CUST-01 (Phase 19) may conclude "no app surface"; chrome defers to that decision (D-18-09).
