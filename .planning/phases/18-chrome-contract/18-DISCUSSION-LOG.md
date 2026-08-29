# Phase 18: Chrome contract - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-29
**Phase:** 18-chrome-contract
**Areas discussed:** Navigation, Frame layout, Page anatomy, Role-scoped chrome

---

## Navigation

| Option | Description | Selected |
|--------|-------------|----------|
| Confirm 7 groups now | Adopt ADR-P-011's 7 non-Admin groups + matrix as locked nav | |
| Nav-agnostic (defer list) | Define nav container + behavior, defer item list to Phase 20 | |
| Revise nav now | Re-litigate group names/order/split; record deltas for Phase 20 | ✓ |

**User's choice:** Revise nav now.
**Notes:** SC3 requires ADR-P-011 to be confirmed or revised, not silently kept. Admin/Settings is out of scope (SC2), so the 8th ADR-P-011 group cannot be adopted.

| Option | Description | Selected |
|--------|-------------|----------|
| Surface pillar names | Show Capture/Structure/Control/Insight as user-facing labels | |
| Reconsider Review | Keep standalone / fold into Control / cross-cutting badge | |
| Fold Reports | Standalone / merge into Economics / Insight pillar | |
| Customer entry | Add surface / leave absent pending CUST-01 | |

**User's choice (free text):** *"we should try and reorganize all the features (new and old) in semantic subsets, this will help us in future ABAC implementation."*
**Notes:** Pivoted the whole nav revision toward semantic feature subsets as the seed for a future ABAC resource taxonomy.

| Option | Description | Selected |
|--------|-------------|----------|
| Domain-entity families | Effort/Demand/Plan/Commercial/People/Governance/Output | |
| Re-surfaced pillars | Capture/Structure/Control/Insight as visible groups | |
| Lifecycle stages | Record/Organize/Approve/Report | ✓ |

**User's choice:** Lifecycle stages.
**Notes:** 4 lifecycle subsets chosen as the nav's top-level groups.

| Option | Description | Selected |
|--------|-------------|----------|
| Groups + principle, defer detail | Enumerate clear-end features; middle mapping = Phase 20 delta | ✓ |
| Fully enumerate now | Resolve all feature placement in Phase 18 | |
| Groups only, no features | No feature list; Phase 19/20 fill placement | |

**User's choice:** Groups + principle, defer detail.

| Option | Description | Selected |
|--------|-------------|----------|
| Per-subset visibility | Each group shows/hides as a unit per role | |
| Per-feature visibility | Finer granularity; ABAC-aligned | |
| Mechanism only, defer matrix | Define predicate mechanism; defer detailed matrix | ✓ |

**User's choice:** Mechanism only, defer matrix.

---

## Frame layout

| Option | Description | Selected |
|--------|-------------|----------|
| Sidebar + top bar | Persistent left sidebar + fixed top bar (org/profile/theme) | |
| Collapsible sidebar | Left rail collapses to icons; top bar fixed | |
| Top-bar + palette | Nav in top bar + command palette; no persistent rail | |

**User's choice (free text):** *"Confirm the current state, which is a collapsible sidebar and the top bar that wraps around the feature content. So yeah, you missed the fact that in the current state we already have the collapsible bar, so nothing new there."*
**Notes:** Locked the existing frame (collapsible sidebar + content-wrapping top bar). Agent had incorrectly assumed a separate global identity top bar.

| Option | Description | Selected |
|--------|-------------|----------|
| Confirm current 3 | org-switcher + profile + theme toggle only | |
| Add command palette | Cmd-K entry point in top bar | |
| Add global search | Entity search in top bar | |

**User's choice (free text):** *"inside the top bar we have 'feature title', 'main action button' possibly filters, tabs. I think here we should try and visualize it and see our options."*
**Notes:** Clarified the top bar is the per-page header (title/action/tabs/filters), not global identity chrome. A formal sketch is Phase 20 (SKETCH-01); contract pins anatomy in text.

| Option | Description | Selected |
|--------|-------------|----------|
| Two-tier (separate) | Global top bar identity-only; per-page PageHeader band below | |
| Merged single bar | Title/action/tabs inside the one global bar | |
| Header + separate FilterBar | PageHeader; filters in content region | |

**User's choice (free text):** *"org, profile and theme are parts of the sidebar, the top bar is the PageHeader."*
**Notes:** Definitive correction: identity chrome lives in the sidebar; the top bar IS the PageHeader. Resolved the Frame + PageHeader composition.

| Option | Description | Selected |
|--------|-------------|----------|
| Filters in header band | Title/action row 1; tabs + filters row 2 (frozen FilterBar) | ✓ |
| Header + separate FilterBar | Filters in content region below header | |
| Filter drawer/sheet | Collapsible filter drawer from header button | |

**User's choice:** Filters in header band.

---

## Page anatomy

| Option | Description | Selected |
|--------|-------------|----------|
| Shell rules only | Max width, scroll, EmptyState, ConfirmDialog; bodies = Phase 19 | ✓ |
| Add master-detail | List + detail panes standard layout | |
| Mandate DataTable+form | Every list uses DataTable; detail uses form layout | |

**User's choice:** Shell rules only.

---

## Role-scoped chrome

| Option | Description | Selected |
|--------|-------------|----------|
| Mechanism + nav/action gating | Predicate system drives nav hide + action disable; manifests = Phase 19 | ✓ |
| Add view-only banner | Subtle banner when access is read-only | |
| Show role in sidebar | Visible role label under profile | |

**User's choice:** Mechanism + nav/action gating.

---

## the agent's Discretion

- CHROME.md formatting follows `LANGUAGE.md`'s pattern (dual-audience, RFC 2119, changelog, authority stack, "Not in this file" fence).
- Exact ordering of the four lifecycle groups and exact PageHeader row-2 filter composition — planner/researcher discretion.
- Visual styling stays in `LANGUAGE.md`/CSS; CHROME.md adds layout/copy/composition only.

## Deferred Ideas

- Future ABAC implementation (lifecycle subsets are the resource taxonomy seed; ABAC not in v0.2.1 scope).
- Admin/Settings chrome (out of scope per SC2).
- Full feature→subset mapping + per-role visibility matrix (Phase 19/20).
- Detailed page bodies / job-cluster implementation (post-Phase 20).
- Customer surface (CUST-01 Phase 19 may conclude no surface).
- Sketch sessions (SKETCH-01 Phase 20; chrome is docs-only).
