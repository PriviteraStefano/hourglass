# Phase 18: Chrome contract - Context

**Gathered:** 2026-08-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Docs-only design contract for v0.2.1. This phase delivers CHR-01: `docs/design/CHROME.md` is the source of truth for the app shell — frame, navigation, role-scoped chrome, and page anatomy. Phase 17 (`LANGUAGE.md`) is the foundation this builds on; Phase 15 frozen components (PageHeader/FilterBar/DataTable/EmptyState/ConfirmDialog) are **inputs**, not a substitute.

**Write set (only these files):**
1. `docs/design/CHROME.md` (new — reserved in `docs/design/INDEX.md` by D-17-15)
2. Update `docs/design/INDEX.md` only if its reserved `CHROME.md` line needs adjustment (it already reserves it correctly — likely no change)

**Success criteria that must be TRUE:**
1. A chrome contract document exists covering frame, navigation, role-scoped chrome, and page anatomy (CHR-01)
2. Admin/Settings chrome is explicitly excluded
3. ADR-P-011 pillar IA is treated as an input to be confirmed or revised by later composition — not silently kept as the chrome
4. No UI implementation, no sketches, no route work

**Out of this phase:** UI implementation, component rewrites, route work, sketch sessions (SKETCH-01 is Phase 20), job-cluster implementation, Admin/Settings surfaces, the detailed per-role visibility matrix, and the full feature→subset mapping (both deferred to Phase 19/20).

</domain>

<decisions>
## Implementation Decisions

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

### the agent's Discretion
- CHROME.md formatting follows `LANGUAGE.md`'s established pattern: dual-audience, RFC 2119 (`MUST`/`MUST NOT`/`SHOULD`), changelog, authority stack, and a "Not in this file" fence (D-17-02/06/09/13/20).
- Exact ordering of the four lifecycle groups, and the exact filter-control composition within PageHeader row 2, are planner/researcher discretion consistent with the principle.
- Visual styling of chrome elements stays in `LANGUAGE.md` / CSS; CHROME.md adds layout / copy / composition only and MUST NOT override `LANGUAGE.md` (D-17-09).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone / phase
- `.planning/ROADMAP.md` — Phase 18 goal, CHR-01, four success criteria (UI hint: no)
- `.planning/REQUIREMENTS.md` — CHR-01; adjacent EMP/MGR/FIN/HR/CUST-01 (Phase 19) and COMP-01/SKETCH-01 (Phase 20)
- `.planning/PROJECT.md` — v0.2.1 is contract-first job clusters; do not recreate cancelled v0.2 Phases 17–26
- `.planning/STATE.md` — current focus Phase 18; do not sketch; do not implement

### Design-language authority (CHROME.md must not override)
- `docs/design/INDEX.md` — design doc map; reserves `CHROME.md` for Phase 18
- `docs/design/LANGUAGE.md` — foundation contract; authority stack (D-17-09): wins on type/color/density/motion/status vocabulary; CHROME.md may only add surface/layout/copy/composition
- `.planning/phases/17-design-language-contract/17-CONTEXT.md` — authority stack, doc style, frozen-component inputs (D-17-45/122), deferred list

### IA input to confirm/revise (SC3)
- `hourglass-vault/decisions/project/ADR-P-011 — Information Architecture & Role-Scoped Surfaces.md` — pillar-mapped nav + role-visibility matrix; treated as input, revised by D-18-01/02

### Frozen component inputs (absent on disk)
- `.planning/phases/15-ux-foundation-design-tokens-shared-components/15-CONTEXT.md` — frozen set intent (D-15-05/06/07/08: DataTable, StatusBadge, ConfirmDialog, PageHeader, EmptyState, FilterBar)
- `.planning/phases/15-ux-foundation-design-tokens-shared-components/15-UI-SPEC.md` — historical input, NOT authority; `LANGUAGE.md` wins on conflict

### Live chrome to document (read-only context)
- `web/src/lib/role-visibility.ts` — predicate-driven role-visibility system; mechanism to build on (D-18-05/08)
- `web/src/components/layout/sidebar.tsx` — existing collapsible sidebar (regroup nav into lifecycle sets)
- `web/src/components/app/org-switcher.tsx` — identity chrome (sidebar)
- `web/src/components/app/profile-menu.tsx` — identity chrome (sidebar)
- `web/src/routes/_authenticated.tsx` — protected shell / auth hydration context

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/lib/role-visibility.ts` — predicate system already implements per-role surface visibility; CHROME.md extends it to page-action gating (D-18-08).
- `web/src/components/layout/sidebar.tsx` — collapsible sidebar; currently groups by ADR-P-011 job-language groups; must be regrouped into the four lifecycle subsets (D-18-02).
- `web/src/components/app/org-switcher.tsx`, `profile-menu.tsx` — identity chrome already lives in the sidebar (D-18-06).
- `web/src/components/shared/status-badge.tsx` — present but raw-Tailwind (known debt, out of scope).

### Established Patterns
- TanStack Router `_authenticated` layout wraps all protected pages; role info available client-side via `GET /auth/me` (ADR-P-011 consequence).
- shadcn/ui + Tailwind v4 CSS-first; `LANGUAGE.md` cites `web/src/index.css` for values.
- Phase 15 frozen component set is **not** on disk — CHROME.md must reference them as intended implementers, never as existing (D-18-11).

### Integration Points
- New doc: `docs/design/CHROME.md` (reserved in INDEX.md). No code changes in this phase.
- `sidebar.tsx` nav structure + `role-visibility.ts` predicates are the concrete touchpoints a later job-cluster phase will modify; CHROME.md specifies the target shape only.

</code_context>

<specifics>
## Specific Ideas

- User wants semantic subsets across **all** features (existing v0.1/v0.2 + planned v0.2.1) regrouped, explicitly to seed a future **ABAC** (attribute-based access control) resource taxonomy. Lifecycle groups (Record/Organize/Approve/Report) are that taxonomy seed.
- Frame correction: identity chrome (org/profile/theme) is in the **sidebar**, and the "top bar" is the per-page **PageHeader** — there is no separate global identity bar (corrects an earlier agent assumption).
- `Today` remains the landing compose view; it is not one of the four lifecycle nav groups.

</specifics>

<deferred>
## Deferred Ideas

- **Future ABAC implementation** — the lifecycle subsets are the resource taxonomy it would consume. ABAC itself is NOT in v0.2.1 scope (current model is RBAC). Recorded as intent only; not Phase 18 work.
- **Admin/Settings chrome** — explicitly out of scope (SC2); ADR-P-011's Admin group is excluded from the chrome.
- **Full feature→subset mapping + per-role visibility matrix** — Phase 19 (role contracts) / Phase 20 (composition map) reconcile.
- **Detailed page bodies** (list/detail/master-detail/form layouts) — Phase 19 workflow contracts.
- **Job-cluster implementation** (build PageHeader/FilterBar/DataTable/EmptyState/ConfirmDialog) — inserted after Phase 20.
- **Customer surface** — CUST-01 (Phase 19) may conclude "no app surface"; chrome defers to that decision (D-18-09).
- **Sketch sessions** — SKETCH-01 is Phase 20; chrome contract is docs-only (SC4), no sketch.

</deferred>

---

*Phase: 18-Chrome contract*
*Context gathered: 2026-08-29*
