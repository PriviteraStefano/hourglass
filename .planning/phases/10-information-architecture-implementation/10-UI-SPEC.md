---
phase: 10
slug: information-architecture-implementation
status: approved
reviewed_at: 2026-07-31
shadcn_initialized: true
preset: base-mira (olive base color, cssVariables, @base-ui/react primitives)
created: 2026-07-31
---

# Phase 10 — UI Design Contract

> Visual and interaction contract for the Information Architecture Implementation phase (ADR-P-011: sidebar regrouping, `/projects` → `/activities` rename, Today landing, Approvals queue, Working Groups surface, role-scoped visibility).
> Sources: ADR-P-011 (locked IA decisions), ADR-P-004 (Today view rules), existing shadcn/token system in `web/src/index.css` + `web/components.json`.

---

## Design System

| Property | Value |
|----------|-------|
| Tool | shadcn (already initialized — do NOT re-init) |
| Preset | `base-mira`, baseColor `olive`, CSS variables on, RSC off |
| Component library | `@base-ui/react` primitives via `@/components/ui` (full shadcn set already installed: sidebar, tabs, table, dialog, alert-dialog, badge, card, empty, skeleton, sonner, combobox, select, tooltip, dropdown-menu, avatar, separator, scroll-area) |
| Icon library | `lucide-react` (existing import convention: named imports, `type LucideIcon` for nav item typing) |
| Font | Inter Variable (`@fontsource-variable/inter`, `font-sans` on `html`); Martian Mono 200 for `.font-display`, Geist Mono 300 for `.font-text` — display/mono fonts NOT used for nav or body copy |

No new dependencies may be added in this phase. Every surface composes the installed `ui/` primitives.

---

## Spacing Scale

Declared values (multiples of 4, Tailwind default scale — already the project convention):

| Token | Value | Usage |
|-------|-------|-------|
| xs | 4px (`p-1`, `gap-1`) | Icon↔label gaps inside nav buttons and badges |
| sm | 8px (`p-2`, `gap-2`) | Sidebar menu item padding, badge padding, queue row inner gaps |
| md | 16px (`p-4`, `gap-4`) | Default card/section padding; Today section internal spacing; queue row padding |
| lg | 24px (`p-6`, `gap-6`) | Page content padding; gaps between Today composition sections |
| xl | 32px (`p-8`, `gap-8`) | Empty-state vertical rhythm (icon → heading → body → CTA) |
| 2xl | 48px | Top-level page header block separation (title → content) on Today, Approvals, Working Groups |
| 3xl | 64px | Not used in this phase (no marketing-scale surfaces) |

Exceptions:
* Sidebar collapsed (icon-only) touch targets keep the shadcn `sidebar` primitive's built-in sizing — do not hand-override; the primitive already guarantees ≥40px hit area.
* Approvals queue rows: 12px vertical padding (`py-3`) is permitted as the single exception to the pure-4px ladder, matching the existing `entries-table` row density for visual continuity.

---

## Typography

Inter Variable is the sole UI typeface for this phase. Declare exactly 4 sizes, 2 weights:

| Role | Size | Weight | Line Height | Usage |
|------|------|--------|-------------|-------|
| Label | 12px (`text-xs`) | 500 (`font-medium`) | 1.33 | Sidebar group labels (Track / Work / People / Economics / Review / Reports / Admin), table column headers, badge text |
| Body | 14px (`text-sm`) | 400 (`font-normal`) | 1.5 | Nav item labels, queue row content, empty-state body, form copy |
| Heading | 20px (`text-xl`) | 600 (`font-semibold`) | 1.2 | Page titles (Today, Approvals, Working Groups, Activities), section headings inside Today |
| Display | 28px (`text-3xl`) | 600 (`font-semibold`) | 1.2 | Today page greeting/title only — the single largest text on any Phase 10 surface |

Rules:
* Only two weights exist in Phase 10 UI: 400 and 600. (Sidebar group labels at 500 are the third value — **locked exception**: they inherit the shadcn `SidebarGroupLabel` style and must not be re-weighted.)
* Muted copy (`text-muted-foreground`) is used for empty-state body, secondary metadata (dates, counts), and sidebar group labels.
* `.font-display` / `.font-text` (Martian/Geist Mono) are reserved for existing surfaces; Phase 10 pages do not introduce new mono usages except numeric hours/counts inside queue rows, which keep `.font-text`.

---

## Color

Existing oklch token system (`--base-*` olive neutrals + `--primary-*` green accent). No new color tokens.

| Role | Value | Usage |
|------|-------|-------|
| Dominant (60%) | `--background` / `--sidebar` (`base-100` light, `base-900` dark) | App shell, sidebar surface, page background |
| Secondary (30%) | `--card` (white / `base-950`), `--muted` (`base-50` / `base-900`) | Today composition cards, Approvals queue rows, Working Groups cards, table surfaces |
| Accent (10%) | `--primary` (`primary-400` green) | **Reserved for exactly these elements:** active sidebar item indicator, primary CTA buttons ("Log time", "Review now", "New working group", "New activity"), current-day marker, link hover states on Today next-action suggestions |
| Destructive | `--destructive` (oklch red, existing token) | Reject buttons on the Approvals queue, destructive confirmations only |

Status colors (existing `status-badge.tsx` convention — all six workflow states already distinct, approved recolored emerald in Phase 8): **reuse `StatusBadge` unchanged** on Today entry cards and Approvals queue rows. Do not introduce a seventh status color.

Accent reserved for: active nav indicator, primary CTAs listed above, current-day marker, Today next-action links. Never for: group labels, section headings, badges, borders, decorative fills.

---

## Visual Hierarchy & Focal Points

| Surface | Primary focal point | Secondary | Hierarchy rule |
|---------|--------------------|-----------|----------------|
| Today | "Waiting on you" section (top, accent CTA "Review now") | "Your week" entry cards | Approver block always renders first; page eye-path is top-down, no side-by-side competing cards |
| Approvals | First pending queue row + its Approve/Reject action pair | Stage tabs (Manager/Finance) | Tabs never compete with rows — actions are the only accent-colored elements below the tab bar |
| Working Groups | "New working group" CTA (page header, right-aligned) | WG cards grid | Header CTA is the single accent element; cards render in muted/card surfaces only |
| Activities (renamed) | Unchanged from existing Projects surface — rename only, no re-layout | — | Existing hierarchy preserved; only vocabulary changes |
| Sidebar | Active route indicator (accent) | Group labels (muted, 12px) | One accent element max in the sidebar at any time; group labels recede via muted color, not size reduction |

Accessibility rule: icon-only rendering exists only in the collapsed sidebar, where every item carries a `tooltip` (existing `SidebarMenuButton` pattern) — no unlabeled icon-only actions on any new Phase 10 surface.

---

## Copywriting Contract

| Element | Copy |
|---------|------|
| Primary CTA (Today, approver) | "Review now" |
| Primary CTA (Today, contributor) | "Log time" |
| Primary CTA (Working Groups) | "New working group" |
| Primary CTA (Activities, renamed) | "New activity" |
| Today heading | "Today" |
| Today section headings | "Waiting on you" · "Your week" |
| Approvals page title | "Approvals" |
| Approvals queue tabs | "Manager" · "Finance" (render only the tab(s) matching the user's approval stage) |
| Working Groups page title | "Working Groups" |
| Empty state — Today (nothing pending, no drafts) | Heading: "You're all caught up" · Body: "Nothing is waiting on you. When drafts, rejections, or approvals land, they'll show up here." |
| Empty state — Today (no data at all, new user) | Heading: "Welcome to Hourglass" · Body: "Start by logging time against an activity. Your week and anything waiting on you will appear here." · CTA: "Log time" |
| Empty state — Approvals queue | Heading: "Queue is clear" · Body: "There are no {stage} approvals waiting. Submitted entries will appear here for review." |
| Empty state — Working Groups | Heading: "No working groups yet" · Body: "Working groups assign people to activities. Create one to start staffing work." · CTA: "New working group" |
| Error state (any Phase 10 surface) | "We couldn't load {surface}. {reason}. Try again." — recovery via `errorComponent` + `router.invalidate()` per ADR-FE-014/Phase-8 convention |
| Destructive confirmation — Reject approval | "Reject entry: This sends the entry back to {name} with your reason. They can edit and resubmit." (requires reason input — existing `approval-buttons.tsx` pattern) |
| Sidebar group labels | Track · Work · People · Economics · Review · Reports · Admin (exact casing, per ADR-P-011 D-1; pillar names never user-facing) |
| Renamed nav item | "Activities" (replaces "Projects"; href `/activities`) |
| Tickets nav item | "Tickets" — `disabled: true`, tooltip "Tickets arrive in v0.2" (ADR-P-003 not yet shipped) |
| Availability nav item | "Availability" — `disabled: true`, tooltip "Availability lands with the staffing schema" (P-008 surfaces follow schema) |
| Settings nav item | stays `disabled: true` under Admin group (existing state) |

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| shadcn official (installed set) | sidebar, tabs, table, badge, card, empty, skeleton, alert-dialog, dialog, button, tooltip, avatar, dropdown-menu, separator, scroll-area, sonner | not required — all already vendored in `web/src/components/ui/` |
| third-party | none | not applicable — `components.json` `"registries": {}` and Phase 10 adds zero new blocks |

Gate result: **no third-party registry usage in this phase** — vetting gate not triggered.

---

## Surface Composition Locks

Per ADR-P-011/ADR-P-004, these are binding for the checker:

1. **Sidebar groups** (ADR-P-011 D-1), in render order: Today (landing, ungrouped top item) → **Track** (Time, Expenses, Tickets-disabled) → **Work** (Activities, Working Groups) → **People** (Org, Availability-disabled) → **Economics** (Contracts, Customers) → **Review** (Approvals) → **Reports** (Exports) → **Admin** (Settings-disabled). "Org Hierarchy" nav label becomes **"Org"**; URL `/org-hierarchy` unchanged (D-6).
2. **Today** (`/`): read-only composition only — no new state, no charts/KPIs, never blank (ADR-P-004). Sections: "Waiting on you" (approver's pending approvals, only if user holds an approval stage) + "Your week" (own draft/submitted/rejected entries) + locked empty states above.
3. **Approvals** (`/approvals`): one page, stage-filtered tabs (Manager/Finance). Rendered **only** for users holding an approval stage (WG manager/delegate, org-role manager/finance). HR never sees the Review group (P-008 D-4). Hidden-from-sidebar is UX scoping, not authorization — backend stays authoritative.
4. **Role-scoped visibility matrix** (ADR-P-011 D-5) drives sidebar rendering: Employee (no Review/Economics/Admin), Manager (Review manager stage), Finance (Review finance stage, Economics), HR (no Review — curator pattern, People full), Admin group hidden from all but org admin.
5. **Route rename**: `/projects` → `/activities` rides the Phase 9 big-bang; the nav label, page title, and CTA all switch to "activity" vocabulary in the same phase.
6. Role data source: current membership role from `GET /auth/me` (already available client-side — no new endpoint).

---

## Checker Sign-Off

- [x] Dimension 1 Copywriting: PASS
- [x] Dimension 2 Visuals: PASS
- [x] Dimension 3 Color: PASS
- [x] Dimension 4 Typography: PASS
- [x] Dimension 5 Spacing: PASS
- [x] Dimension 6 Registry Safety: PASS

**Approval:** approved 2026-07-31
