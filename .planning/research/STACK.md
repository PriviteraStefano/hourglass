# Stack Research — Hourglass v0.2 (UX polish, ticket ontology, availability)

**Domain:** Time & expense tracking + service-desk tickets + availability/capacity planning (full-stack Go 1.26 / React 19 / PostgreSQL)
**Researched:** 2026-08-01
**Confidence:** HIGH for versions & licensing (verified against npm registry dist-tags/peerDeps and first-party pricing pages); MEDIUM for pattern-level recommendations

## Verdict

v0.2 needs **almost no new infrastructure**: two new frontend packages at most, **zero new Go dependencies**, zero new backend services. The three feature areas map onto the existing stack as follows:

1. **Sketch UX workflow** — pure HTML/CSS/JS per gsd-sketch conventions (throwaway mockups under `.planning/sketches/`, opened directly in a browser). No tooling to add. Optionally a one-command static server for live-reload iteration, nothing more.
2. **Ticket lists** — adopt **@tanstack/react-table v8 (8.21.3, stable)** for the ticket queue and capacity grids. It is headless (logic/state only), renders with the project's existing shadcn/Base UI markup, and supports the sort/filter/pagination/row-selection combination a service-desk queue needs. TanStack Table **v9 is still beta** — do not touch it.
3. **Calendar/date-range** — **keep react-day-picker ^9.14.0** (range mode is stable and the shadcn `base-mira` Calendar wrapper is built on the v9 API; v10 renames the package to `@daypicker/react` — a controlled future task, not a mid-milestone change). Capacity/resource views must be **custom-built grids on date-fns v4 + Tailwind**: every calendar library that ships resource/timeline views (FullCalendar, schedule-x) puts them behind a **commercial license** ($480+ FullCalendar Premium; schedule-x `@sx-premium`).
4. **Go side for tickets** — plain PostgreSQL tables following in-repo precedent (the immutable `*_approvals` tables already model an append-only event stream). Comments/activity = append-only `ticket_events` table; tags = `ticket_tags` join table (PostgreSQL docs: "arrays are not sets"); request counting per customer = a COUNT query. **No message queue, no SSE, no websockets** for v0.2 — in-app notifications, if any, are TanStack Query `refetchInterval` polling (30–60s), with SSE as the natural later upgrade (Go stdlib streams it; `EventSource` + `withCredentials` works with the existing HttpOnly-cookie auth).

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `@tanstack/react-table` | **8.21.3** (add) | Headless tables for ticket queues, ticket history, capacity grids | Stable v8 (v9.0.0-beta.69 is a rewrite still in beta). Headless = logic + state only, renders into the project's existing shadcn/Base UI markup — matches the "no magic" stack. Sorting, filtering, manual server-side pagination, row selection, column visibility all via controlled state slices (`state.sorting` / `onSortingChange`, etc.). Peer deps `react >=16.8` → React 19 fine. 7.6k code snippets, high reputation. |
| `react-day-picker` | **^9.14.0 (keep — do NOT bump)** | Absence date-range picking, calendar tabs | Already installed and wrapped by shadcn `base-mira` Calendar (`getDefaultClassNames`, `DayButton` = v9 API). v9 range mode covers everything absences need: `mode="range"`, `min`/`max` days, `disabled` matchers, `excludeDisabled` (9.0.2+), `resetOnSelect` (9.14+). Latest is v10.0.1 which renames the package to `@daypicker/react` and removes deprecated v9 APIs — a milestone-sized change, not a v0.2 one. |
| `date-fns` | **^4.4.0 (keep)** | Day arithmetic for capacity grids, absence overlap checks, week/month ranges | Already installed; v4.4.0 is current. All date math for a custom capacity grid (`eachDayOfInterval`, `differenceInCalendarDays`, `format`) — no new dependency needed. |
| PostgreSQL (tables, no new engine) | 18 | `ticket_events` (append-only activity/comments), `ticket_tags` (join table), ticket request counts | Tags: official PG docs advise a separate table over arrays ("Arrays are not sets… use a separate table with a row for each item") — also matches the schema's existing relational style (memberships, approvals join tables). Activity stream: append-only pattern already proven in-repo by immutable `*_approvals` tables; no event bus, no outbox. |
| Go stdlib + `pgx/v5` | 1.26.1 / v5.10.0 (keep) | All ticket + availability endpoints | Hand-written SQL repos already exist for every feature; tickets/availability are the same hexagonal shape (service + pgx repo + thin HTTP adapter). **Zero new Go imports** — verified: nothing in the ticket/comment/tag/availability space needs a library. |

### Existing Stack (unchanged, verified current)

| Technology | Version | Purpose | Notes |
|------------|---------|---------|-------|
| Go | 1.26.1 | Backend | `go.mod` verified |
| React | 19.2.8 | Frontend | — |
| TanStack Router / Query / Form | 1.170.x / 5.101.4 / 1.33.x | Routing, data, forms | Query 5.101.4 is current |
| `@base-ui/react` | 1.6.0 | Headless primitives (shadcn base-mira) | **No drag-and-drop component** — verified against the 1.6.0 exports map; `@base-ui/drag-and-drop` does not exist on npm (E404) |
| Tailwind CSS | 4.3.3 | Styling | `@tailwindcss/vite` |
| shadcn | 4.16.0 | UI kit (`base-mira` style) | Aliases point at `@base-ui/react` |
| `@xyflow/react` + `dagre` | 12.11.2 / 0.8.5 | Activity tree | Unchanged; not reused for capacity (grid, not graph) |
| `recharts` | 3.8.0 | Charts | Available for capacity/utilization charts if sketches want them |
| `zod` + `react-hook-form` | 4.4.3 / 7.83.0 | Ticket/absence form validation | Existing form pattern |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@dnd-kit/core` + `@dnd-kit/sortable` + `@dnd-kit/utilities` | 6.3.1 / 10.0.0 / 3.2.2 | Drag-and-drop kanban ticket board | **Only if a sketch variant lands on a board view.** Base UI has no DnD, so this is the sole candidate. Battle-tested classic API; peers `react >=16.8` → React 19 OK. A newer framework-agnostic generation (`@dnd-kit/react` 0.5.0) shipped June 2026 — too young for v0.2; revisit later. |
| `@tanstack/react-virtual` | 3.14.9 | Windowed list rendering | **Not for v0.2.** Needed only at 1000+ rows; org-scale ticket queues are hundreds at most. Add when a queue genuinely exceeds it. |

### Development Tools (sketch-driven UX workflow)

| Tool | Purpose | Notes |
|------|---------|-------|
| None (convention) | gsd-sketch mockups | Sketches are standalone HTML under `.planning/sketches/NNN-name/`, inline CSS, opened directly: `open .planning/sketches/NNN-name/index.html`. Shared `../themes/*.css` via relative `<link>` works from `file://` (same-origin link/script loads are unrestricted). Sketch toolbar (theme switcher / viewport / annotation) is a tiny inline snippet per `sketch-tooling.md`. |
| `npx serve` (optional) | Live-reload iteration server | Optional ergonomic only — `npx serve .planning/sketches`. Do not introduce Vite dev server, Tailwind CDN, or any framework into sketches: throwaway HTML must stay throwaway (a few hundred lines, no build step), or the options-exploration loop dies. |

## Installation

```bash
# Core (ticket queues + capacity grids)
cd web && bun add @tanstack/react-table

# Only if a sketch lands on a kanban board view
cd web && bun add @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities

# Backend: nothing to install — zero new Go dependencies
```

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Ticket-list tables | TanStack Table v8 (8.21.3) | Status quo custom lists | Custom lists are fine for simple lists but TanStack Table pays for itself when sort + filter + paginate + select combine — exactly the ticket queue; headless keeps shadcn markup |
| Ticket-list tables | TanStack Table v8 | TanStack Table v9 (`9.0.0-beta.69`) | Major rewrite (`features`/`useTable` API) still in beta — production app stays on stable v8 |
| Capacity/resource calendar | Custom grid (date-fns + CSS grid) | FullCalendar (`7.0.2` standard) | Timeline View + Vertical Resource View are **Premium, $480+** (verified on fullcalendar.io/pricing); standard views alone don't do resource planning |
| Capacity/resource calendar | Custom grid | schedule-x (`@schedule-x/calendar` 4.6.1 + `@schedule-x/react` 4.1.0) | Resource/time-grid views are **paid `@sx-premium`** packages (verified via docs); also drags in `temporal-polyfill` + a default theme CSS — heavy for a grid |
| Ticket-board drag-drop | `@dnd-kit/core` 6.3.1 | Base UI drag-and-drop | **Does not exist** — no DnD component in the 1.6.0 exports map, no npm package |
| Ticket-board drag-drop | `@dnd-kit/core` 6.3.1 | `@dnd-kit/react` 0.5.0 | New-generation architecture, ~2 months old (June 2026); classic core has years of docs/battle-testing |
| Ticket-board drag-drop | `@dnd-kit/core` 6.3.1 | `@hello-pangea/dnd` | dnd-kit is the de-facto standard; hello-pangea is a fork of the abandoned react-beautiful-dnd |
| Ticket tags | `ticket_tags` join table | `text[]` + GIN index | Official PG docs: "Arrays are not sets… use a separate table" — join table scales for filter + autocomplete + counts and matches schema style |
| Notifications | None for v0.2; TanStack Query `refetchInterval` polling if in-app | SSE | SSE is baseline-widely-available and Go stdlib streams it trivially — but it's one-way plumbing for a feature not yet specced; polling (30–60s on the queue query) reuses existing patterns with zero infra. 6-connections-per-browser HTTP/1.1 caveat noted for later |
| Notifications | Polling now, SSE later | Websockets | No bidirectional need; websocket libs (gorilla/nhooyr) add protocol/state complexity for nothing |
| Rich text on tickets | Plain textarea (existing shadcn Textarea) | TipTap / Lexical rich-text editors | Comment bodies are notes, consistent with time-entry/expense notes; an editor is a large surface (toolbars, serialization, a11y) with no requirement behind it |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| FullCalendar | Resource/timeline views (the actual capacity feature) are commercial ($480+, per-developer seats) | Custom date-fns grid (~200 lines, full design-system control) |
| schedule-x | Same commercial model (`@sx-premium` resource views) + temporal-polyfill dependency | Custom date-fns grid |
| react-day-picker **v10** (now) | Package renamed to `@daypicker/react`, deprecated v9 APIs removed; shadcn `base-mira` Calendar wrapper is v9-only. A mid-milestone bump would break every calendar on every page | Stay `^9.14.x`; plan the v10 migration as a standalone task when UX pass touches calendar styling anyway |
| TanStack Table v9 | Beta rewrite — API churn mid-milestone | `@tanstack/react-table@8.21.3` |
| Message queues (RabbitMQ/NATS/Redis streams) | Request counting, comments, and tags are plain SQL; nothing needs async delivery in v0.2 | Append-only `ticket_events` + COUNT queries |
| Websocket libraries | No bidirectional real-time requirement | Polling; SSE (Go `http.Flusher` + `EventSource`) if/when real-time is specced |
| `@tanstack/react-virtual` | Premature for sub-1000-row lists | Add only when a queue exceeds ~1000 rows |
| Rich-text editors (TipTap/Lexical) | Scope creep on comment bodies | shadcn Textarea (matches existing notes fields) |
| `react-big-calendar` | Stale maintenance; same resource-view gaps | Custom grid |
| Anything new on the Go side | Tickets/availability = the same hexagonal service + pgx repo + HTTP adapter shape as v0.1; no library fills a gap | Existing pgx/v5 hand-written SQL |

## Stack Patterns by Variant

**If sketches land on a kanban/board view for tickets:**
- Add `@dnd-kit/core` + `@dnd-kit/sortable` + `@dnd-kit/utilities` (6.3.1 / 10.0.0 / 3.2.2)
- Because Base UI ships no DnD; dnd-kit classic is the only battle-tested option. Keep it optional: the default ticket surface should be a TanStack Table list (sorting/filtering is the primary need; boards are a visual preference).

**If the capacity view is a month/week resource grid (likely):**
- Build with date-fns (`eachDayOfInterval`, `differenceInCalendarDays`) + Tailwind CSS grid; rows = people or activities/WGs, columns = days, cells colored by absence/ticket load
- Because resource views are paywalled in every calendar library, and this grid is 100–200 lines with full design-system control. Use `react-day-picker` only for the range *picker* in the absence form (mirror the existing `export-form.tsx` DayPicker range pattern).

**If capacity needs charts (utilization bars):**
- Use the already-installed `recharts` 3.8.0 — no new dependency.

**If the UX pass wants a "board + list" toggle:**
- Both render from the same TanStack Query ticket list query; the board is a layout choice over the same data — do not create a second data-fetching path.

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `@tanstack/react-table@8.21.3` | React 19 (peer `>=16.8`) | v9.0.0-beta.69 is a separate breaking line — never mix |
| `@dnd-kit/core@6.3.1` | React 19 (peer `>=16.8`); pairs with `@dnd-kit/sortable@10.0.0`, `@dnd-kit/utilities@3.2.2` | Do not mix with `@dnd-kit/react@0.5.x` (v2 architecture, separate API) |
| `react-day-picker@9.14.x` | React 19; shadcn `base-mira` Calendar wrapper | `10.0.1` exists but renames to `@daypicker/react` — treat as a separate migration |
| `@base-ui/react@1.6.0` | React `^17 \|\| ^18 \|\| ^19` | No DnD component; combobox/select/menu cover ticket assignee pickers |
| `@tanstack/react-query@5.101.4` | React 19 | `refetchInterval` = the polling mechanism for future in-app notifications |
| `date-fns@4.4.0` | React 19 (framework-agnostic) | Current latest |
| Go `pgx/v5 v5.10.0` | Go 1.26.1 | Handles `DATE` range queries for `availability_windows`; no new driver needed |

## Integration Points with Existing Stack (summary)

- **Ticket list** reuses: TanStack Query options pattern (`web/src/api/*.ts`), shared `api<T>()` client, shadcn table/card markup, `DateRangeFilter` from `components/shared/entries-filters.tsx`.
- **Absence form** reuses: the `DayPicker mode="range"` pattern from `export-form.tsx`, `react-hook-form` + `zod`, sonner toasts.
- **Capacity grid** consumes: existing `availability_windows` DATE columns + activities tree CTEs; renders with date-fns + Tailwind, no calendar lib.
- **Ticket backend** mirrors: hexagonal service → pgx repo → thin HTTP adapter; `ticket_events` append-only table mirrors `*_approvals`; `ticket_tags` mirrors membership join tables; `created_by`/`org_id` conventions from `availability_windows` (012_staffing_schema).
- **Sketches** live in `.planning/sketches/` per gsd-sketch; per-page UX phases re-render the final HTML variants as shadcn/Base UI components.

## Sources

- Context7 `/tanstack/table` — v8 stable vs v9 beta, headless state/`useTable` patterns (MEDIUM)
- Context7 `/mui/base-ui` — v1.6.0 exports map (40+ components, **no DnD**), React 19 peers (MEDIUM)
- Context7 `/gpbl/react-day-picker` — v9 range mode (PropsRange), custom modifiers, v8→v10 upgrade guide (MEDIUM)
- Context7 `/schedule-x/schedule-x` — views + `@sx-premium/time-grid-resource-view` (MEDIUM)
- npm registry (dist-tags + peerDependencies, 2026-08-01) — authoritative versions: `@tanstack/react-table` 8.21.3 / 9.0.0-beta.69, `react-day-picker` 9.14.0 / 10.0.1, `@dnd-kit/*`, `date-fns` 4.4.0 (HIGH)
- fullcalendar.io/pricing — Timeline/Resource views Premium from $480, standard MIT (HIGH)
- PostgreSQL 18 docs, §8.15 Arrays — "Arrays are not sets… use a separate table" + GIN note (HIGH)
- MDN "Using server-sent events" — EventSource, `withCredentials`, 6-connection HTTP/1.1 limit (HIGH)
- github.com/clauderic/dnd-kit/releases — v2 architecture `@dnd-kit/react` 0.5.0, June 2026 (MEDIUM)
- Local (HIGH): `.config/opencode/gsd-core/workflows/sketch.md` + `references/sketch-tooling.md` (sketch conventions); `web/components.json` (base-mira, `@base-ui/react` aliases); `web/package.json`; `go.mod`; `migrations/012_staffing_schema.up.sql` (`availability_windows` DATE columns)

---
*Stack research for: Hourglass v0.2 — UX polish, ticket ontology, availability*
*Researched: 2026-08-01*
