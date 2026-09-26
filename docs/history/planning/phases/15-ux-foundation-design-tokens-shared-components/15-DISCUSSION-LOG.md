# Phase 15: UX Foundation — Design Tokens + Shared Components - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-12
**Phase:** 15-ux-foundation-design-tokens-shared-components
**Areas discussed:** Status palette design, Frozen component APIs, Sketch loop contract, Migration scope

---

## Status palette design

| Option | Description | Selected |
|--------|-------------|----------|
| Role-based roles | Small set: neutral/info/success/warning/danger; components consume roles, never raw colors | ✓ |
| Status-named tokens | One token per status; more expressive, more drift risk | |
| Hybrid: roles + mapping | Role tokens + a status→role map in code | |

**User's choice:** Role-based roles
**Notes:** Matches shadcn conventions; one contrast audit; shared with alerts/banners.

| Option | Description | Selected |
|--------|-------------|----------|
| Meaning-first | Same semantic meaning = same color everywhere; pending_manager AND pending_finance both amber; green = approved-only; purple dropped | ✓ |
| Keep current hues | pending_manager green, pending_finance purple; zero visual churn but semantically misleading | |
| Carve-out for finance chain | Role tokens except a dedicated purple/indigo pair for the two finance statuses | |

**User's choice:** Meaning-first
**Notes:** User accepts re-learning the palette; this is the moment to do it before new surfaces ship.

| Option | Description | Selected |
|--------|-------------|----------|
| Confirm the table | draft/open/planned/declared/superseded = neutral; submitted/in_progress/active/claimed = info; triage/pending_manager/pending_finance/lapsed = warning; approved/confirmed/resolved/closed/done = success; rejected/dismissed/cancelled/withdrawn = danger | ✓ |
| Change: closed = neutral | closed reads 'archived' not 'successful' | |
| Change: lapsed = danger | a lapsed plan row is a miss, not a warning | |

**User's choice:** Confirm the table
**Notes:** Governance badges = neutral variants.

| Option | Description | Selected |
|--------|-------------|----------|
| Seeded map | away = warning, partial = warning, over-capacity = danger, invalid = danger | ✓ |
| Softer: away = danger | only actual misses go red | |

**User's choice:** Seeded map

---

## Frozen component APIs

| Option | Description | Selected |
|--------|-------------|----------|
| Add TanStack Table | @tanstack/react-table headless + ui/table primitives; sorting/columns/pagination built in | ✓ |
| Hand-rolled, no new dep | Freeze the existing entries-table pattern as DataTable<T> | |

**User's choice:** Add TanStack Table
**Notes:** New dependency; entries-table migration to it happens in Phase 21.

| Option | Description | Selected |
|--------|-------------|----------|
| subtle/solid/outline/dot | Four variants; color from the status→role mapping | ✓ |
| subtle + dot only | Minimal surface | |

**User's choice:** subtle/solid/outline/dot

| Option | Description | Selected |
|--------|-------------|----------|
| Consolidate now | Shared ConfirmDialog absorbs the 3 per-page delete-confirm dialogs this phase | ✓ |
| Create only, absorb later | Dialogs get absorbed during their polish phases | |

**User's choice:** Consolidate now
**Notes:** Required-reason destructive pattern carried from D-13-10/D-13-16.

| Option | Description | Selected |
|--------|-------------|----------|
| Title+desc+actions | PageHeader = title + description + actions slot | |
| Add status summary strip | PageHeader also includes counts/key metric badges next to the title | ✓ |

**User's choice:** Add status summary strip
**Notes:** EmptyState = thin wrapper over ui/empty in both options.

---

## Sketch loop contract

| Option | Description | Selected |
|--------|-------------|----------|
| Standalone contract doc | `.planning/sketches/SKETCH-LOOP-CONTRACT.md`; roadmap phases reference it | ✓ |
| CONTEXT.md only | Contract lives only in this phase's CONTEXT.md | |
| Doc + roadmap pointers | Same doc plus checks in ROADMAP.md phase entries | |

**User's choice:** Standalone contract doc

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, UI-SPEC for 15 | Run /gsd-ui-phase 15; precedent 13-UI-SPEC | ✓ |
| No, first UI-SPEC at 16 | Skip the ceremony; CONTEXT decisions suffice | |

**User's choice:** Yes, UI-SPEC for 15

| Option | Description | Selected |
|--------|-------------|----------|
| Cap only | 3-round maximum is the only hard rule; phases decide their rounds | ✓ |
| Minimum one round | Every surface phase must sketch at least once | |

**User's choice:** Cap only

---

## Migration scope

| Option | Description | Selected |
|--------|-------------|----------|
| Shared only | status-badge + 3 dialogs + new status type unions; entries table/filters and page colors migrate in polish phases | ✓ |
| Also the entries table | Migrate entries-table to DataTable + entries-filters to FilterBar now | |
| Full sweep | Everything, including every hardcoded status color across all pages | |

**User's choice:** Shared only
**Notes:** User explicitly rejected full sweep — polish phases must have migration work left (21/23/24).

| Option | Description | Selected |
|--------|-------------|----------|
| Add unions now | TicketStatus/DirectionStatus/AbsenceStatus in web/src/types now; StatusBadge generic | ✓ |
| With their surfaces | Unions land with Phase 16/18/19 surfaces | |

**User's choice:** Add unions now

---

## the agent's Discretion

- Exact token variable names + danger-vs-destructive reuse (planner)
- FilterBar API details (search/filters/reset/count)
- PageHeader breadcrumb slot, EmptyState default icon set, DataTable pagination style
- Governance badges as StatusBadge neutral variants or separate component
- Component file layout under `web/src/components/shared/`
- Sidebar visual-verification task shape

## Deferred Ideas

- entries-table → DataTable + entries-filters → FilterBar: Phase 21
- Page-level hardcoded status colors sweep: polish phases 20–26
- P-011 IA revision: after prototypes land (research D-O)
- Sidebar collapsed-mode human visual verification (fix landed; check remains)
