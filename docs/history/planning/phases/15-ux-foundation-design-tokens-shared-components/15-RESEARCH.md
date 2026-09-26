# Phase 15: UX Foundation — Design Tokens + Shared Components - Research

**Researched:** 2026-08-12
**Domain:** Design tokens (Tailwind v4 CSS-first theming) + shared React component system + sketch-loop process contract
**Confidence:** HIGH

## Summary

Phase 15 is the design-system freeze: it lands the semantic status/state role tokens (neutral/info/success/warning/danger) in `web/src/index.css` following the exact existing `:root` / `.dark` / `@theme inline` pattern, freezes the six shared components (StatusBadge rewrite, DataTable, FilterBar, PageHeader, EmptyState, ConfirmDialog) in `web/src/components/shared/`, adds three new status type unions, consolidates three per-page delete dialogs into ConfirmDialog, and writes the sketch-loop contract doc (UXFD-02). The UI-SPEC (`15-UI-SPEC.md`, approved 2026-08-12, committed as `ff165ed`) already pins every token value and every component API contract — the plan's job is implementing that contract, not designing it. This phase is UI-only; the design decisions (D-15-01…D-15-13) are locked in CONTEXT.md.

The single most important research finding is the **TanStack Table version decision**: `@tanstack/react-table` latest is now **9.1.2, published 2026-08-09 (3 days old)**, and v9 is a **breaking-change major** (`useReactTable` → `useTable`, row models → `tableFeatures()` registry, `flexRender` → `table.FlexRender`). Every ecosystem example (official TanStack pagination example, shadcn data-table docs) still shows the v8 API. **Recommendation: pin `@tanstack/react-table@8.21.3`** (battle-tested, 16 months stable, React 19 compatible) — the UI-SPEC DataTable contract (`ColumnDef<T>[]`, sorting, client pagination) is identical under both, so the wrapper's public API is version-independent and a later v9 migration is contained to one file. The package carries a SUS verdict (reason: "too-new" — the v9.1.2 publish timestamp), so the install must be gated behind `checkpoint:human-verify` per protocol. The v9 API is documented below so the planner can choose deliberately.

**Primary recommendation:** Treat `15-UI-SPEC.md` as the source of truth (every token value and component prop is already pinned); implement tokens first (they unblock badge variants), then StatusBadge → DataTable → FilterBar → PageHeader → EmptyState → ConfirmDialog, then the type unions; add `@tanstack/react-table@8.21.3` behind a human-verify checkpoint; write the SKETCH-LOOP-CONTRACT doc from the verified gsd-sketch mechanics in this research.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Status palette (UXFD-01)
- **D-15-01:** **Role-based semantic tokens, not status-named tokens** — `index.css` gains role tokens (neutral / info / success / warning / danger) as `:root` + `.dark` CSS variable pairs, mapped via `@theme inline` exactly like the existing `--primary`/`--destructive` pattern. Components (badges, alerts, banners) consume roles, never raw colors; the `--destructive` token is reused as the danger base where it fits. — **Reversibility:** costly — every component and surface built from Phase 16 on renders statuses through these tokens; renaming/restructuring later means touching all of them.
- **D-15-02:** **Meaning-first hues — same semantic meaning = same color everywhere.** The finance-chain distinction does NOT survive: `pending_manager` and `pending_finance` are BOTH warning/amber. Green means approved/done only; purple drops out of the status language entirely. Users re-learn that green = approved-only. — **Reversibility:** costly — hue assignments are visible in all surfaces and users learn them; restoring a distinct finance hue later means re-mapping across every badge.
- **D-15-03:** **Status→role mapping table (confirmed):**

  | Role | Statuses |
  |------|----------|
  | neutral (gray) | draft, open (ticket), planned (ticket), declared (absence), superseded (direction); governance badges (creator_controlled / unanimous / majority) as neutral variants |
  | info (blue) | submitted, in_progress (ticket), active (direction), claimed (direction, derived) |
  | warning (amber) | triage (ticket), pending_manager, pending_finance, lapsed (direction, derived) |
  | success (green) | approved, confirmed (absence), resolved (ticket), closed (ticket), done (direction, derived) |
  | danger (red) | rejected, dismissed (ticket), cancelled (direction), withdrawn (absence) |

- **D-15-04:** **Direction warning tokens seeded** (server-emitted vocabulary from 13-UI-SPEC, D-13-30): `away` = warning, `partial` = warning, `over-capacity` = danger, `invalid` = danger. Rendered as small inline warning labels in scheduler/queue surfaces (Phase 19), not full badges.

#### Frozen component set (UXFD-01 SC2)
- **D-15-05:** **DataTable built on `@tanstack/react-table`** — new dependency; headless table + the existing `ui/table.tsx` primitives. Sorting, column definitions, and pagination are built into the frozen DataTable. (The current hand-rolled `shared/entries-table.tsx` stays for Phase 21, which migrates it.)
- **D-15-06:** **StatusBadge variants frozen: subtle (default), solid, outline, dot.** Color comes from the D-15-03 status→role mapping; the hardcoded Tailwind classes in the current `status-badge.tsx` are fully replaced. Dot variant = color dot + label for compact lists.
- **D-15-07:** **ConfirmDialog is consolidated NOW** — the shared component absorbs the 3 existing per-page delete-confirm dialogs (customers, org-hierarchy, working-groups) in this phase. It supports the required-reason destructive pattern (precedent D-13-10/D-13-16: reason-less destructive writes are rejected server-side with 400, and the dialog must present the required-reason input). Every future destructive action (absence reject, claim unclaim) uses it.
- **D-15-08:** **PageHeader = title + optional description + right-aligned actions slot + status summary strip** (counts or key metric badges next to the title — chosen over the minimal title+desc+actions). **EmptyState = thin wrapper over the existing `ui/empty.tsx`** composing icon/title/description/action with a default look. **FilterBar** is the generic search/filter/reset component in the frozen set (details below are planner discretion).

#### Sketch loop contract (UXFD-02)
- **D-15-09:** **Standalone contract doc: `.planning/sketches/SKETCH-LOOP-CONTRACT.md`** — pins: every surface/polish phase (16–26) runs gsd-sketch first; 2–3 variants shown; user agrees on one; UI-only plans; ≤3 sketch rounds maximum; sketch MANIFEST updated; `--wrap-up` produces sketch-findings. Downstream phases inherit it from this file, not from CONTEXT.md.
- **D-15-10:** **This phase produces a UI-SPEC.md via `/gsd-ui-phase 15`** (precedent: `13-UI-SPEC.md`) — locks the design-system contract (tokens, spacing, typography, color roles) that planner and gsd-ui-checker verify against. *(Research note: already done — `15-UI-SPEC.md` exists, status `approved`, committed 2026-08-12 as `ff165ed`/`d9cf9ed`/`7fee81a`. Plan verifies presence, does not regenerate.)*
- **D-15-11:** **The 3-round cap is the ONLY hard rule** — no minimum round enforced; phase plans decide how many rounds they actually run (1–2 for polish, 2–3 for new surfaces like the scheduler).

#### Migration scope
- **D-15-12:** **Shared-only migration** — this phase refactors only the shared layer: `status-badge.tsx` rewrite (D-15-06), the 3 delete dialogs → ConfirmDialog (D-15-07), plus the type unions (D-15-13). The time-entries table/filters and every page-level hardcoded color stay untouched — their polish phases (21 Track, 23 Approvals+WG, 24 Customers+Contracts) do that migration using the frozen set. The "full sweep" option was explicitly rejected.
- **D-15-13:** **New status vocabularies typed in this phase** — `TicketStatus` / `DirectionStatus` / `AbsenceStatus` unions added to `web/src/types/models.ts` mirroring the backend vocabularies (tickets: open/triage/planned/in_progress/resolved/closed/dismissed; direction: draft/active/superseded/cancelled; absences: declared/confirmed/rejected/withdrawn), and StatusBadge is typed generically so Phase 16/18/19 surfaces consume it without type churn. No UI for these statuses this phase.

### the agent's Discretion
- Exact token variable names (`--status-{role}`, `--status-{role}-foreground`, dark-mode pairs) and whether danger reuses `--destructive` or gets a parallel `--status-danger` — must follow the existing `:root`/`.dark`/`@theme inline` pattern
- FilterBar's exact API (search input, select/dropdown filters, date range, reset, active-filter count) — it is in the frozen set, details open *(research note: UI-SPEC pinned the API contract; styling/layout details remain planner freedom)*
- PageHeader breadcrumb slot, EmptyState default icon set, DataTable pagination style (page-size selector etc.)
- Whether governance badges render as StatusBadge neutral variants or a separate small component
- Component file layout under `web/src/components/shared/` (kebab-case, named exports, `__tests__` colocation per CONVENTIONS.md)
- The sidebar human-visual-verification task shape (SC4 — fix already landed)

### Deferred Ideas (OUT OF SCOPE)
- **Time-entries table → DataTable + entries-filters → FilterBar migration** — Phase 21 (Track polish) via the frozen set (D-15-12)
- **Page-level hardcoded status colors sweep** (today, WG, activities, customers, contracts, exports pages) — folded into their polish phases (20–26); ROADMAP SC1 ("no surface/polish phase introduces ad-hoc hex values") applies from Phase 16 on
- **P-011 IA revision** — stays deferred until prototypes land (research D-O); do NOT touch IA in this phase
- **Sidebar collapsed-mode human visual verification** — the code fix is landed (commit `54f465a`); only a human check of collapsed-mode hover/navigation remains, surfaced as a task in this phase (SC4 triage complete)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| UXFD-01 | Design token foundation extended (status palette, state colors) and shared component set frozen before any surface/polish work | Token values + component API contracts pinned in approved 15-UI-SPEC.md; Tailwind v4 `@theme inline` pattern verified in-repo (index.css:33-195) and against official docs; TanStack Table version decision researched (recommend v8.21.3); cva/ui-primitive composition patterns verified from existing shared components |
| UXFD-02 | Sketch-driven loop established: each surface/polish phase shows 2–3 gsd-sketch options, user agrees, UI-only plans, ≤3 sketch rounds | gsd-sketch SKILL.md + sketch.md workflow read and distilled into the contract doc's required mechanics (`.planning/sketches/NNN-*/`, MANIFEST.md, winner frontmatter, `--wrap-up` → sketch-findings, ≤3-round cap D-15-11) |

## Project Constraints (from AGENTS.md)

- **Stack fixed**: React 19, TanStack Router v1 / React Query v5, Vite, TypeScript, Tailwind CSS, shadcn-based components — do not introduce alternative frameworks
- **File conventions**: kebab-case component filenames; named exports only (never default exports); `__tests__/` colocation with `*.test.tsx`; `.ts`/`.tsx` extensions on `@/` imports (`allowImportingTsExtensions`); oxfmt (80-col, semicolons, double quotes) — `bun run fmt`, `bun run lint` (oxlint, type-aware)
- **Components stay presentational**: server state lives in routes/hooks (React Query); the frozen set must not own data fetching, stores, or navigation
- **Type conventions**: `interface XxxProps` for props; `import type` for type-only imports; types re-exported via `web/src/types/index.ts` barrel
- **JSDoc comments cite plan/ADR references** (e.g., `(D-15-06)`) — new components cite this phase's decisions
- **Validation**: `bun run build` (tsc -b + vite build) must pass; tests via vitest; e2e via Playwright (not needed for pure component work)
- **graphify**: after modifying code files this session, run `python3 -c "from graphify.watch import _rebuild_code; from pathlib import Path; _rebuild_code(Path('.'))"` to keep the knowledge graph current
- **OpenWiki**: treat source code and tests as authoritative; prefer the narrowest quiet validation that proves changed behavior

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Semantic status tokens (CSS variables) | Browser / Client | — | CSS-first Tailwind v4 theming lives entirely in `web/src/index.css`; no server involvement |
| Shared component set (StatusBadge, DataTable, FilterBar, PageHeader, EmptyState, ConfirmDialog) | Browser / Client | — | Presentational React components consuming `ui/*` primitives; routes/hooks own data |
| Status vocabulary type unions | Browser / Client | API / Backend | Types land in `web/src/types/models.ts` as frontend mirrors; the vocabulary ORIGINATES from backend contracts (13-UI-SPEC, D-14-08) — the phase only mirrors it, never invents it |
| Sketch-loop process contract | Process (docs) | — | `.planning/sketches/SKETCH-LOOP-CONTRACT.md` is a planning artifact governing downstream phases; no runtime tier |
| Destructive-action confirmation | Browser / Client | API / Backend | ConfirmDialog presents the required-reason input client-side; the server 400 invariant (D-13-10) remains authoritative — client mirrors, server enforces |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Tailwind CSS | 4.3.3 (present) | Token system: `:root`/`.dark` pairs + `@theme inline` utilities | Already the project's theming engine; v4 CSS-first is the documented pattern for runtime-switchable dark mode |
| @tanstack/react-table | **8.21.3 (recommended)** | Headless DataTable engine: sorting, client pagination, column defs | D-15-05 locked; v8 is the stable, example-rich major (see Decision Note below) |
| class-variance-authority | 0.7.1 (present) | Variant system for StatusBadge/EmptyState | Existing project pattern (ui/empty.tsx, ui primitives) |
| lucide-react | ^1.27.0 (present) | Icons (ArrowUpDown, Inbox, ChevronLeft/Right, TriangleAlert…) | `components.json` iconLibrary; `Icon` suffix convention |
| @base-ui/react | ^1.6.0 (present) | Primitive engine under `ui/*` (alert-dialog, select, calendar…) | `components.json` `style: base-mira`; the frozen set composes these |
| React | 19.2.8 (present) | Component runtime | Fixed by project stack |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| react-day-picker + ui/calendar | ^9.14.0 (present) | Date-range picker in FilterBar (`kind: "date-range"`) | FilterBar date filters (planner discretion on exact widget) |
| date-fns | ^4.4.0 (present) | Date formatting/label building | FilterBar range labels, DataTable date cells |
| ui/alert-dialog + ui/textarea | present | ConfirmDialog composition | Required-reason destructive pattern (D-15-07) |
| ui/skeleton | present | DataTable `isLoading` rows | Loading state per UI-SPEC |
| ui/select, ui/input, ui/button, ui/table, ui/badge | present | FilterBar / DataTable building blocks | All already in `web/src/components/ui/` (verified 55+ files) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| @tanstack/react-table@8.21.3 | @tanstack/react-table@9.1.2 (latest) | v9 is a breaking major published 3 days ago: `useTable` + `tableFeatures()` registry replaces `useReactTable` + row models; every official example is v8-style. v8 = 16 months stable, 17.9M weekly downloads, React 19 OK. Public DataTable contract identical under both → migration later is one file. **Planner checkpoint: confirm with user before install** |
| @tanstack/react-table | Hand-rolled table (entries-table.tsx pattern) | Explicitly rejected (D-15-05): sorting/pagination/column-def edge cases are a solved problem; hand-rolling duplicates effort for 10+ consuming phases |
| StatusBadge on ui/badge | Standalone shared component | UI-SPEC pins custom base classes (`rounded-full border px-2 py-0.5 text-xs font-semibold` + role classes) — a dedicated shared component is the contract; ui/badge remains for non-status tags |

**Installation:**
```bash
cd web && bun add @tanstack/react-table@^8.21.3
```
(Behind `checkpoint:human-verify` — see Package Legitimacy Audit.)

**Version verification (run 2026-08-12, npm registry):**
- `@tanstack/react-table@latest` → **9.1.2** (published 2026-08-09); dist-tags: `latest: 9.1.2`, `beta: 9.0.0-beta.80`, `alpha: 9.0.0-alpha.54`
- `@tanstack/react-table@8` → **8.21.3** (published 2025-04-14; `8.20.6` 2024-12-13 — v8 line is in maintenance)
- peerDeps: v9 `react >=18`; v8 `react >=16.8` (both compatible with React 19.2.8)

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| @tanstack/react-table | npm | v8 line ~4 yrs; **latest publish 3 days** (9.1.2, 2026-08-09) | 17.9M/wk | github.com/TanStack/table | SUS (reason: "too-new" — recent v9 publish timestamp) | Flagged — planner adds `checkpoint:human-verify` before install |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** `@tanstack/react-table` — [WARNING: flagged as suspicious — verify before using.] The SUS signal is the 2026-08-09 v9.1.2 publish; the package itself is the TanStack org (same org as the project's existing `@tanstack/react-query` 5.101.4 and `@tanstack/react-router` 1.170.18), has a real GitHub repo, no postinstall script (`npm view scripts.postinstall` → null), not deprecated. The risk profile is *version freshness*, not provenance: **pin `@tanstack/react-table@^8.21.3`** to avoid the 3-day-old breaking major, and gate the install behind human verification per protocol. All other dependencies this phase are already installed and unmodified.

## Architecture Patterns

### System Architecture Diagram

```
┌──────────────────────────── Browser / Client (web/) ───────────────────────────┐
│                                                                                │
│  index.css (token layer)                                                       │
│  ┌──────────────────────────────────────────────────────────────────────────┐  │
│  │ :root  +  .dark   (semantic pairs: --status-neutral … --status-danger)   │  │
│  │        --status-{role}: <strong oklch / var(--base-*) / var(--destructive)>│ │
│  │        --status-{role}-foreground: <white | black>                        │  │
│  │        --status-danger: var(--destructive)  (reuse, both themes)          │  │
│  ├──────────────────────────────────────────────────────────────────────────┤  │
│  │ @theme inline { --color-status-{role}[-(foreground)]: var(--status-*) }   │  │
│  │   → utilities: bg-status-info, text-status-warning-foreground, …         │  │
│  │   (dark mode resolves because utilities reference the runtime var)        │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│                          │ consumed by                                        │
│                          ▼                                                    │
│  components/shared/ (frozen set — presentational, no data lifecycle)          │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐  │
│  │ StatusBadge  │ │   DataTable  │ │  FilterBar   │ │   PageHeader         │  │
│  │ <S> generic  │ │ <T> generic  │ │ stateless    │ │ title+desc+actions+  │  │
│  │ 4 variants   │ │ sorting+page │ │ values from  │ │ summary strip (dot   │  │
│  │ role mapping │ │ +empty+skeleton│ │ route search │ │ badges + counts)    │  │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘  │
│  ┌────────────────┐ ┌──────────────────────────────────────────────────────┐  │
│  │  EmptyState    │ │ ConfirmDialog (alert-dialog + textarea; required-    │  │
│  │ wrapper over   │ │ reason gating; absorbs 3 per-page delete dialogs)    │  │
│  │ ui/empty.tsx   │ └──────────────────────────────────────────────────────┘  │
│  └────────────────┘                         │ absorbed into                 │
│  components/ui/* (55+ shadcn primitives)    ▼                                │
│  types/models.ts (EntryStatus + NEW TicketStatus/DirectionStatus/AbsenceStatus)│
│                                                                                │
│  .planning/sketches/SKETCH-LOOP-CONTRACT.md  (process contract, UXFD-02)       │
└────────────────────────────────────────────────────────────────────────────────┘
```

Trace the primary use case: a surface phase (16+) renders a status → `StatusBadge` looks it up in `STATUS_ROLE_MAP` → variant class references `bg-(--status-{role})/10` → utility resolves the runtime var → `:root`/`.dark` pair supplies the theme-correct oklch color.

### Recommended Project Structure

```
web/src/
├── index.css                            # + --status-* pairs in :root/.dark + @theme inline block
├── types/
│   ├── models.ts                        # + TicketStatus / DirectionStatus / AbsenceStatus unions
│   └── index.ts                         # barrel — add exports if explicit (D-15-13)
└── components/shared/
    ├── status-badge.tsx                 # REWRITE (D-15-06): generic <S>, STATUS_ROLE_MAP, 4 variants
    ├── data-table.tsx                   # NEW (D-15-05): on @tanstack/react-table + ui/table
    ├── filter-bar.tsx                   # NEW (D-15-08): search + select/date-range + reset + count
    ├── page-header.tsx                  # NEW (D-15-08): title/desc/actions/breadcrumb/summary strip
    ├── empty-state.tsx                  # NEW (D-15-08): wrapper over ui/empty.tsx
    ├── confirm-dialog.tsx               # NEW (D-15-07): alert-dialog + textarea, requiredReason
    └── __tests__/                       # *.test.tsx colocated (CONVENTIONS.md)

.planning/sketches/
└── SKETCH-LOOP-CONTRACT.md              # NEW (D-15-09) — doc-only deliverable
```

### Pattern 1: Semantic role tokens (Tailwind v4 CSS-first)
**What:** Status colors are runtime CSS variables in `:root`/`.dark` pairs, registered as theme colors via `@theme inline` so utilities resolve the *runtime* variable (dark-mode swap works) rather than a baked static value.
**When to use:** Any color that must change per theme. Already the project's `--primary`/`--destructive` shape.
**Example (the exact in-repo shape, extended — index.css:33-131 pattern):**
```css
:root {
  /* existing pair (index.css:79) */
  --destructive: oklch(0.577 0.245 27.325);
  /* new status tokens (UI-SPEC values; neutral from --base-600) */
  --status-neutral: var(--base-600);
  --status-neutral-foreground: var(--color-white);
  --status-info: oklch(0.546 0.245 262.881);
  --status-info-foreground: var(--color-white);
  --status-warning: oklch(0.769 0.188 70.08);
  --status-warning-foreground: var(--color-black); /* black on amber */
  --status-danger: var(--destructive); /* reuse (D-15-01) */
  --status-danger-foreground: var(--color-white);
}
.dark {
  --destructive: oklch(0.704 0.191 22.216); /* index.css:114 */
  --status-neutral: var(--base-400);
  /* …dark pairs per UI-SPEC table… */
}
@theme inline {
  --color-status-neutral: var(--status-neutral);
  --color-status-neutral-foreground: var(--status-neutral-foreground);
  /* …all 10 (5 roles × 2)… */
}
```
Source: in-repo index.css (verified this session) + Tailwind v4 docs (`colors.mdx`/`theme.mdx`: "@theme inline" is the documented way to reference other CSS variables; `@theme` generates `--color-*` utilities).

### Pattern 2: StatusBadge generic role-mapped variants (D-15-06, D-15-13)
**What:** One badge component generic over any status union; a default `STATUS_ROLE_MAP: Record<string, StatusRole>` covers all five vocabularies; variants (subtle/solid/outline/dot) color exclusively from role tokens. Unknown status → neutral + humanized label (never crash, never ad-hoc hex).
**When to use:** Any status/state rendering from Phase 16 on.
**Example (variant classes from UI-SPEC, classes are static strings → unit-testable):**
```tsx
export type StatusRole = "neutral" | "info" | "success" | "warning" | "danger";
export const STATUS_ROLE_MAP: Record<string, StatusRole> = {
  draft: "neutral", submitted: "info", pending_manager: "warning",
  pending_finance: "warning", approved: "success", rejected: "danger",
  /* ticket/direction/absence vocabularies per D-15-03 */
};
const variantClasses: Record<StatusBadgeVariant, string> = {
  subtle:  "border-transparent bg-(--status-info)/10 text-(--status-info)",
  solid:   "border-transparent bg-(--status-info) text-(--status-info-foreground)",
  outline: "border-(--status-info)/40 bg-transparent text-(--status-info)",
  dot:     "border-transparent bg-transparent text-foreground",
};
// role substituted at runtime: cn(base, variantClasses[variant].replaceAll("info", role))
```
Source: 15-UI-SPEC (variant recipes) + Tailwind v4 docs (parenthesized custom-property syntax `bg-(--var)/10`).

### Pattern 3: DataTable on TanStack Table (v8 API)
**What:** Headless table wrapped by the frozen component: `useReactTable({ columns, data, getCoreRowModel, getSortedRowModel, getPaginationRowModel })`, headers render sort buttons (`header.column.getToggleSortingHandler()`), pagination footer drives `table.previousPage()/nextPage()/setPageSize()`; `ColumnDef<T>[]` is the frozen public contract.
**When to use:** All list surfaces from Phase 16+; migrates `entries-table.tsx` in Phase 21.
**Example:** See Code Examples — official pagination example core (Context7 / tanstack/table).

### Pattern 4: ConfirmDialog consolidation (D-15-07)
**What:** A controlled, presentational destructive-confirmation dialog on `ui/alert-dialog` + `ui/textarea`. `requiredReason` disables confirm until non-empty ("A reason is required." — mirrors the server 400 invariant D-13-10/D-13-16). Each consuming site keeps its own state wiring (zustand store for customers, props for working-groups/org-hierarchy).
**When to use:** Every destructive write. Absorbs the 3 existing dialogs without moving their state owners into the shared component (CONVENTIONS: components stay presentational).
**Example:** See Code Examples (required-reason gate).

### Anti-Patterns to Avoid
- **Baking status colors as static Tailwind palette classes** (`bg-yellow-100 text-yellow-800 …` — exactly what the current `status-badge.tsx:9-40` does): dies with the rewrite; no surface phase may reintroduce it (ROADMAP SC1).
- **Plain `@theme` instead of `@theme inline`**: utilities would reference the theme variable `--color-status-info` instead of the runtime `--status-info`; `.dark` redefines the latter → dark mode breaks silently.
- **Tokens only in `@theme`, missing `:root`/`.dark` pairs**: `bg-(--status-info)/10` reads the runtime var — with no `:root` definition the utility fails at runtime. Both layers are required.
- **Version-ambiguous TanStack code in plans**: v8 vs v9 APIs are incompatible (`useReactTable` vs `useTable`). Examples in plans MUST match the pinned version, or executors copy the wrong one.
- **Moving page state into frozen components**: ConfirmDialog must not absorb zustand stores; DataTable/FilterBar must not own React Query. Presentational-only keeps the frozen set stable across 10+ phases.
- **StatusBadge type churn**: exporting `StatusBadgeProps` unchanged (`status` + `className`) keeps the existing time-entries re-export file and all 7 consumer sites compiling with zero edits.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Table sorting / pagination / column defs | Custom table logic | @tanstack/react-table (v8.21.3) | Sorting state machine, page resets, column def typing, a11y plumbing are solved; D-15-05 locked |
| Status/state color system | Ad-hoc Tailwind classes per status | Semantic role tokens in index.css | Dark-mode pairs, alpha tints, and cross-surface consistency need one source of truth (D-15-01) |
| Confirm/cancel dialog chrome | Fresh dialog markup per page | ui/alert-dialog + frozen ConfirmDialog | Focus trap, escape, overlay behavior already in the primitive; 3 dialogs consolidate now (D-15-07) |
| Variant styling | Conditional class strings | cva (class-variance-authority) | Existing project pattern (ui/empty.tsx, ui primitives); type-safe variant props |
| Icons | Inline SVG | lucide-react | components.json iconLibrary; `Icon` suffix convention |
| Date-range picking | Native inputs | ui/calendar (react-day-picker) | Existing primitive; consistent look with the rest of the app |
| Status → role semantics | Heuristic guessing | Explicit STATUS_ROLE_MAP | The D-15-03 mapping is a locked domain decision — a data table, not logic |

**Key insight:** every "solved problem" in this list has an existing in-repo solution (ui primitives, cva, lucide, @tanstack family). The only new dependency is TanStack Table, and the entire risk surface of this phase is the v8-vs-v9 choice around it.

## Runtime State Inventory

> Partial-refactor phase (StatusBadge rewrite + dialog consolidation). No external runtime state exists; everything below is in-repo code, so every item is a **code edit**, not a data migration.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no database, datastore, or cache references phase 15 surface (UI-only; Postgres untouched) | None |
| Live service config | None — no external service config references the shared components | None |
| OS-registered state | None — no OS-level registrations involved | None |
| Secrets/env vars | None — no env vars touched (VITE_API_URL etc. unchanged) | None |
| Build artifacts | None — no binaries or installed packages carry the old component shape; `bun.lock` will gain @tanstack/react-table (new, not a rename) | `bun install` after `bun add` |
| In-repo consumer state (code) | 7 StatusBadge consumer sites (expense-detail, expense-row, expenses-list, today-page, time-entries-list, entry-row, approvals-page) + 1 re-export file (`time-entries/-components/status-badge.tsx` re-exports shared + its `StatusBadgeProps` type); 3 delete dialogs with 2 different state owners (customers zustand store; working-groups props; org-hierarchy props); `ui/empty.tsx` EmptyTitle consumed by today-page + approvals-page | Code edits only: keep `StatusBadgeProps` export shape; keep each dialog's state wiring in its page, swap chrome for ConfirmDialog |

## Common Pitfalls

### Pitfall 1: TanStack Table v8/v9 API confusion
**What goes wrong:** Executor installs latest (9.1.2) and follows a v8 example — `useReactTable` doesn't exist; or installs v8 but a v9 snippet (`table.FlexRender`) leaks in.
**Why it happens:** v9 (2026-08-09) is a breaking major; all official examples are v8-style; `bun add` defaults to latest.
**How to avoid:** Pin `@tanstack/react-table@^8.21.3` in the plan; include a v8-API example in the plan (see Code Examples); verify with a quick `bunx tsc -b` after install.
**Warning signs:** `Property 'getCoreRowModel' does not exist`, missing exports `useReactTable`/`flexRender`.

### Pitfall 2: Dark-mode breakage from wrong theme registration
**What goes wrong:** Badges look right in light mode, wrong/unreadable in dark; or utilities don't exist at all.
**Why it happens:** (a) plain `@theme` instead of `@theme inline` bakes the light value into utilities; (b) `:root`/`.dark` pairs missing so `bg-(--status-info)/10` resolves to nothing.
**How to avoid:** Follow the index.css:33-195 shape exactly — pairs in both `:root` and `.dark`, registration in `@theme inline` with `--color-status-*` keys; smoke-test one badge in both themes (`bun run dev`, toggle `.dark`).
**Warning signs:** `bg-status-info` utility not generated (unknown class), identical light/dark rendering.

### Pitfall 3: Breaking the 7 existing StatusBadge consumers
**What goes wrong:** The rewrite changes props/export shape; `time-entries/-components/status-badge.tsx` re-export and `StatusBadgeProps` import fail; or labels change silently.
**Why it happens:** Generic rewrite adds `mapping`/`variant` props; removing the `StatusBadgeProps` export breaks the re-export file.
**How to avoid:** Keep `status: EntryStatus` + `className?` props working unchanged; keep exporting `StatusBadgeProps` (extend, don't remove); humanized labels stay identical ("Pending Manager" → "Pending finance" — UI-SPEC says labels humanize to Title Case; confirm label table in the rewrite matches today's labels for the 6 EntryStatus values).
**Warning signs:** `bun run build` failures in routes/, or badge labels changing on the time-entries page (e.g. "Pending Manager" vs "Pending finance" — UI-SPEC: Title Case, keep current wording).

### Pitfall 4: Dialog absorption dragging page state into the shared layer
**What goes wrong:** ConfirmDialog grows store imports/query hooks; customers dialog's zustand wiring leaks in; the frozen component becomes page-coupled.
**Why it happens:** The 3 dialogs have different state owners (store vs props) — easiest path is "copy them all in".
**How to avoid:** ConfirmDialog stays controlled (`open`, `onOpenChange`, `onConfirm(reason?)`); each of the 3 sites passes its own state; `isSubmitting`/`error` come from the site's mutation. Keep `reparent-confirm-dialog.tsx` untouched (D-15-12).
**Warning signs:** ConfirmDialog file imports `useCustomersStore` or any `XxxApis`.

### Pitfall 5: EmptyTitle font-weight remap rippling visually
**What goes wrong:** `ui/empty.tsx` EmptyTitle `text-sm font-medium` → `font-semibold` (UI-SPEC Typography remap) changes appearance on today-page and approvals-page empty states — intended, but easy to forget the 500-must-not-survive rule.
**Why it happens:** The 2-weight contract (400/600 only) forbids `font-medium`; the remap touches a shared primitive.
**How to avoid:** Include the empty.tsx base-class edit as an explicit task; grep for `font-medium` in the frozen set afterward (must be zero in new components).
**Warning signs:** `font-medium` reappears in shared components; grep `grep -rn "font-medium" web/src/components/shared/`.

### Pitfall 6: A11y contracts dropped under deadline
**What goes wrong:** Sort buttons without `aria-label`, pagination icon buttons unlabeled, `aria-sort` missing — UI-SPEC marks these as contracts, not suggestions.
**Why it happens:** Icon-only controls look self-evident to the implementer.
**How to avoid:** Copy the exact label strings from UI-SPEC ("Sort by {label}" / "Sorted by {label} ({direction})", "Previous page"/"Next page"); assert in tests via `getByRole("button", { name: "Sort by Date" })`.
**Warning signs:** oxlint jsx-a11y errors on icon-only buttons (type-aware lint runs in CI).

## Code Examples

### 1. TanStack Table v8 core — sorting + client pagination (DataTable internals)
```tsx
// Source: tanstack/table official React pagination example (Context7, main branch — v8 API)
import {
  type ColumnDef, type SortingState, type PaginationState,
  flexRender, getCoreRowModel, getSortedRowModel, getPaginationRowModel,
  useReactTable,
} from "@tanstack/react-table";

const table = useReactTable({
  columns, data,
  state: { sorting, pagination },
  onSortingChange: setSorting,
  onPaginationChange: setPagination,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
});

// header: flexRender(header.column.columnDef.header, header.getContext())
// sort toggle: header.column.getToggleSortingHandler(); state: header.column.getIsSorted()
// pagination: table.getCanPreviousPage() / table.previousPage() / table.nextPage()
// page-size: table.setPageSize(Number(e.target.value)) with [10, 20, 50]
// counts: table.getRowModel().rows.length (current page) vs table.getRowCount() (total)
```
Note: v9 would replace `useReactTable` with `useTable` + `tableFeatures({ rowSortingFeature, createSortedRowModel, rowPaginationFeature, createPaginatedRowModel, sortFns })` and `flexRender` with `table.FlexRender header={header}` — **use v8 for this phase**.

### 2. Tailwind v4 `@theme inline` with runtime variable pairs (tokens)
```css
/* Source: Tailwind v4 docs theme.mdx/colors.mdx (Context7) + in-repo index.css:33-131 shape */
:root { --acme-canvas-color: oklch(0.967 0.003 264.542); }
[data-theme="dark"] { --acme-canvas-color: oklch(0.21 0.034 264.665); }
@theme inline { --color-canvas: var(--acme-canvas-color); }
/* → bg-canvas/50 etc. resolve the runtime var (dark-mode swap works) */
```
Alpha on custom-property colors (docs colors.mdx): `bg-(--my-var)/10` — the exact syntax the subtle badge variant uses.

### 3. Required-reason destructive confirm (ConfirmDialog gate)
```tsx
// Source: 15-UI-SPEC ConfirmDialog contract (D-15-07) — mirrors server 400 invariant (D-13-10)
const [reason, setReason] = useState("");
const reasonInvalid = requiredReason && reason.trim() === "";
// confirm button: disabled={reasonInvalid || isSubmitting}
// validation copy when touched: "A reason is required."
// error prop renders: "Could not complete the action. Try again."
```

### 4. cva variant pattern (StatusBadge / EmptyState shape)
```tsx
// Source: in-repo ui/empty.tsx:28-41 (existing project pattern)
const emptyMediaVariants = cva("mb-2 flex shrink-0 items-center justify-center …", {
  variants: { variant: { default: "bg-transparent", icon: "flex size-8 …" } },
  defaultVariants: { variant: "default" },
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded per-status Tailwind classes (`bg-yellow-100` etc., status-badge.tsx:9-40) | Semantic role tokens via `:root`/`.dark` + `@theme inline` | This phase (D-15-01/02/03/06) | Dark-mode-correct, cross-surface-consistent status colors; SC1 ("no ad-hoc hex") enforceable |
| 3 per-page delete dialogs + reparent dialog | Frozen ConfirmDialog with required-reason support | This phase (D-15-07) | One destructive pattern for phases 16–26 (absence reject, claim unclaim) |
| Hand-rolled EntriesTable/EntriesFilters | Frozen DataTable (@tanstack/react-table) + FilterBar | DataTable now; entries-table migration Phase 21 (D-15-12) | Sorting/pagination built in; consumers get them free |
| `useReactTable` + row models (v8) | `useTable` + `tableFeatures()` (v9) | 2026-08-09 (v9.1.2) | Breaking major — this phase pins v8.21.3; migration later is one-file |

**Deprecated/outdated:**
- **`useReactTable`/`getSortedRowModel`/`flexRender`** (v8 API): still functional and the ecosystem standard; superseded by v9's features architecture in the 3-day-old major — do not use for new code beyond this phase's frozen DataTable, and re-evaluate the version when a consuming phase needs table features v8 lacks.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Pin `@tanstack/react-table@8.21.3` (not 9.1.2) | Standard Stack | If user prefers latest, DataTable internals must use the v9 API (documented above) — the frozen prop contract is unaffected |
| A2 | `bg-(--status-info)/10` parenthesized custom-property alpha syntax works in Tailwind 4.3.3 [CITED: Tailwind v4 docs — parenthesized custom-property form + modifier; in-repo `bg-destructive/10` precedent with `@theme inline`] | Patterns/Code Examples | If the syntax fails at build, fallback is `bg-status-info/10` (registered theme utility — same result) |
| A3 | oklch values for info/success/warning in the UI-SPEC table are correct [CITED: 15-UI-SPEC (verified against Tailwind v4 palette 2026-08-12); neutral + danger values VERIFIED in-repo: `--base-600: oklch(0.4411 0 24.82)` index.css:45, `--base-400: oklch(0.7084 0 23.42)` index.css:43, `--destructive: oklch(0.577 0.245 27.325)` index.css:79 / `.dark` `oklch(0.704 0.191 22.216)` index.css:114] | Standard Stack/Patterns | Visual hue mismatch on badges — caught at human visual check; trivial fix |
| A4 | TypeScript 7.0.2 compiles the new generic components (`StatusBadge<S>`, `DataTable<T>`) without config changes [existing generic `EntriesTable<T>` compiles today] | Validation | Build failures — contained; adjust generics to non-generic where needed |
| A5 | UI-SPEC generation (D-15-10) is already satisfied — file exists, approved, committed | Scope | If user expects a fresh `/gsd-ui-phase 15` run, plan adds a re-verify task — no re-generation (would overwrite the approved contract) |
| A6 | The 3-round sketch cap (D-15-11) is the only hard loop rule; wrap-up (`--wrap-up` → sketch-findings skill) is a separate step from a sketch round | UXFD-02 | If wrap-up counted as a round, phases could exhaust the cap without sketching — contract doc must count only sketch-option rounds |

## Open Questions (RESOLVED)

1. **[RESOLVED] TanStack Table major version (v8 vs v9)**
   - What we know: v9.1.2 is latest (2026-08-09, breaking); v8.21.3 is the stable maintenance line; UI-SPEC pins neither; both support React 19.
   - What's unclear: user preference — the frozen component lives for 10+ phases.
   - Recommendation: default to **v8.21.3** (plan checkpoint `checkpoint:human-verify` doubles as the version confirmation); research documents the v9 API for a deliberate switch.
   - Resolution: implemented in plan 02 — Task 1's blocking human-verify checkpoint confirms the v8 pin, Task 2 installs `@tanstack/react-table@^8.21.3` (RESOLVED 2026-08-12).
2. **[RESOLVED] Exact token variable names + danger handling**
   - What we know: CONTEXT marks names as agent discretion; UI-SPEC says danger reuses `--destructive` as base and prescribes 2 tokens per role + 10 `@theme inline` keys.
   - What's unclear: nothing blocking — planner freedom (`--status-neutral` … `--status-danger` per UI-SPEC table is the obvious reading).
   - Recommendation: follow the UI-SPEC table verbatim; no user confirmation needed.
   - Resolution: implemented in plan 01 Task 1 — the 10 tokens land in `:root`/`.dark` verbatim from the UI-SPEC Color table with exactly 10 `@theme inline` keys (RESOLVED 2026-08-12).
3. **[RESOLVED] FilterBar date-range widget** (calendar popover vs native select)
   - What we know: UI-SPEC `kind: "date-range"`; ui/calendar + react-day-picker present.
   - What's unclear: widget style; planner discretion per CONTEXT.
   - Recommendation: calendar popover (existing pattern in time-entries filters), mirroring `entries-filters.tsx` behavior.
   - Resolution: implemented in plan 02 Task 3 — Popover + ui/calendar DayPicker `mode="range"` with 'dd MMM' labels (RESOLVED 2026-08-12).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| bun | Install @tanstack/react-table + run scripts | ✓ | 1.3.13 | npm (not used in repo) |
| node | build/test toolchain | ✓ | 22.23.1 | — |
| Tailwind CSS (v4) | token system | ✓ | 4.3.3 (package.json) | — |
| React / TS | components | ✓ | 19.2.8 / 7.0.2 (package.json) | — |
| @tanstack/react-table | DataTable | ✗ not installed | — | Must `bun add @tanstack/react-table@^8.21.3` (checkpoint-gated) |
| go / Postgres / Docker | none — UI-only phase | ✓ (go 1.26.1) | — | not required |

**Missing dependencies with no fallback:**
- `@tanstack/react-table` — the DataTable cannot be built without it (D-15-05); install is the first Wave 0/1 task, gated by human-verify.

**Missing dependencies with fallback:** none — everything else is present.

## Validation Architecture

> `workflow.nyquist_validation` absent from `.planning/config.json` → treated as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.10 + @testing-library/react 16.3.2 + jsdom (globals: true) |
| Config file | `web/vitest.config.ts` (setup: `./src/lib/__tests__/setup.ts` — jest-dom + matchMedia polyfill) |
| Quick run command | `bun run test -- src/components/shared/__tests__/<file>.test.tsx` |
| Full suite command | `bun run test` (plus `bun run typecheck`, `bun run lint`, `bun run build` for the phase gate) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UXFD-01 | StatusBadge maps all 5 vocabularies + EntryStatus to roles per D-15-03; unknown → neutral fallback | unit | `bun run test -- src/components/shared/__tests__/status-badge.test.tsx` | ❌ Wave 0 |
| UXFD-01 | StatusBadge variants render role classes; dot variant present | unit | same file (render + classList assertions — classes are static strings) | ❌ Wave 0 |
| UXFD-01 | DataTable: sorting toggles + `aria-label`/`aria-sort` contract; pagination footer counts; page-size 10/20/50; empty state; skeleton on isLoading; em-dash missing cells | unit | `bun run test -- src/components/shared/__tests__/data-table.test.tsx` | ❌ Wave 0 |
| UXFD-01 | FilterBar: active-filter count chip, Reset visibility (≥1 active), stateless onChange | unit | `bun run test -- src/components/shared/__tests__/filter-bar.test.tsx` | ❌ Wave 0 |
| UXFD-01 | PageHeader: summary chips render as dot badges + counts; actions slot | unit | `bun run test -- src/components/shared/__tests__/page-header.test.tsx` | ❌ Wave 0 |
| UXFD-01 | EmptyState: default Inbox icon; title/desc/action slots | unit | `bun run test -- src/components/shared/__tests__/empty-state.test.tsx` | ❌ Wave 0 |
| UXFD-01 | ConfirmDialog: confirm disabled until reason non-empty when `requiredReason`; "A reason is required."; error copy; isSubmitting spinner | unit | `bun run test -- src/components/shared/__tests__/confirm-dialog.test.tsx` | ❌ Wave 0 |
| UXFD-01 | 3 dialogs absorbed: each site renders ConfirmDialog and the old per-page dialogs are deleted; reparent dialog untouched | unit/smoke | existing page test suites (customers/org-hierarchy/working-groups) + `bun run build` | ✅ existing suites cover sites |
| UXFD-01 | New type unions exist and EntryStatus consumers still compile | typecheck | `bun run typecheck` | ❌ (types are compile-time) |
| UXFD-02 | SKETCH-LOOP-CONTRACT.md exists at `.planning/sketches/` pinning 2–3 variants, ≤3 rounds, MANIFEST update, `--wrap-up` | manual/static | file-presence check in phase gate | ❌ Wave 0 (doc) |
| UXFD-01 | SC4 sidebar collapsed-mode human visual check | manual | human verification task (code fix `54f465a` landed) | n/a |

### Sampling Rate
- **Per task commit:** `bun run test -- src/components/shared/__tests__/` (affected component file) + `bun run typecheck`
- **Per wave merge:** `bun run test` + `bun run lint` + `bun run build`
- **Phase gate:** full suite green + `bun run fmt:check` + human visual check (dark-mode badge smoke + sidebar collapsed mode) before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `web/src/components/shared/__tests__/status-badge.test.tsx` — role mapping, variants, unknown fallback (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/data-table.test.tsx` — sorting/pagination/a11y/empty/loading (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/filter-bar.test.tsx` — active count/reset (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/page-header.test.tsx` — summary strip (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/empty-state.test.tsx` — default icon/slots (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/confirm-dialog.test.tsx` — required-reason gate (REQ UXFD-01)
- [ ] `bun add @tanstack/react-table@^8.21.3` — before any DataTable work (framework dep)

*(No new framework config needed — vitest/jsdom/aliases already configured; framework install not required.)*

## Security Domain

> `security_enforcement` absent from config.json → enabled. This is a UI-only phase: no new server surface, no auth/session/access-control changes. The security content is limited to the two client-side patterns the frozen set introduces.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — (no auth surface; existing JWT/cookie flow untouched) |
| V3 Session Management | no | — |
| V4 Access Control | no | — (no new routes/roles; destructive gating stays server-side per D-13-10) |
| V5 Input Validation | yes | ConfirmDialog required-reason is a client-side UX mirror of the server 400 invariant — server remains authoritative; client never bypasses (it only prevents accidental submits) |
| V6 Cryptography | no | — |

### Known Threat Patterns for this phase

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| XSS via server-emitted strings (status labels, direction warning messages rendered verbatim per 13-UI-SPEC D-13-30) | Tampering | React escapes text children by default — badges/warning labels render strings as text, never `dangerouslySetInnerHTML`; icons come from lucide components only |
| Destructive writes without confirmation/reason | Tampering | ConfirmDialog requires explicit confirm (+ reason when the server requires it, D-13-10/D-13-16); server-side 400 on reason-less destructive writes remains the enforcement point |
| A11y-induced misuse (unlabeled sort/page controls) | — | UI-SPEC a11y contracts: `aria-label="Sort by {label}"`, `aria-sort`, `aria-label="Previous page"/"Next page"` — enforced via oxlint jsx-a11y + unit-test assertions |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: in-repo, this session] `web/src/index.css` — token structure lines 33-131 (`:root`/`.dark`), `@theme inline` block lines 133-195; `--destructive: oklch(0.577 0.245 27.325)` (line 79) / dark `oklch(0.704 0.191 22.216)` (line 114); `--base-600: oklch(0.4411 0 24.82)` (line 45); `--base-400: oklch(0.7084 0 23.42)` (line 43)
- [VERIFIED: in-repo, this session] `web/src/components/shared/status-badge.tsx` — hardcoded map lines 9-40, base `"inline-flex items-center px-2 py-0.5 rounded text-xs font-medium"` (line 48)
- [VERIFIED: in-repo, this session] `web/src/components/ui/empty.tsx` — EmptyTitle base `"font-heading text-sm font-medium tracking-tight"` (line 63); cva pattern lines 28-41
- [VERIFIED: in-repo, this session] `web/src/types/models.ts` — `EntryStatus` union (lines 3-9); `Role` incl. `"hr"` (line 1)
- [VERIFIED: in-repo, this session] `web/package.json`, `web/components.json`, `web/vitest.config.ts`, `.planning/codebase/CONVENTIONS.md`, `15-CONTEXT.md`, `15-UI-SPEC.md`, `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `gsd-sketch/SKILL.md` + `sketch.md` workflow, git log (`ff165ed`, `54f465a`)
- [VERIFIED: npm registry] `@tanstack/react-table` versions/dist-tags/peerDeps; package-legitimacy verdict SUS ("too-new", 17.9M weekly downloads, TanStack repo, no postinstall)

### Secondary (MEDIUM confidence)
- [CITED: Context7 / tailwindlabs/tailwindcss.com] `colors.mdx` + `theme.mdx` — `@theme inline` for referencing runtime CSS variables; parenthesized custom-property syntax `bg-(--var)/10`
- [CITED: Context7 / tanstack/table] official React pagination example (v8 API: `useReactTable`, `getCoreRowModel`, `getSortedRowModel`, `getPaginationRowModel`, `flexRender`, `table.getToggleSortingHandler()`); v9 migration guides (React/Preact/Svelte) — `useTable`, `tableFeatures()`, `table.FlexRender`
- [CITED: 15-UI-SPEC.md] token oklch values (info blue-600 `oklch(0.546 0.245 262.881)` etc. — checked against Tailwind v4 palette 2026-08-12), component API contracts, copy contract, typography remap

### Tertiary (LOW confidence)
- [ASSUMED] TypeScript 7.0.2 behavior with the new generic components (no session verification of TS7-specific generic constraints) — mitigations in Assumptions Log A4

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version verified against package.json/npm registry this session; the one judgment call (v8 vs v9) is flagged with full API documentation for either choice
- Architecture: HIGH — token pattern, component APIs, and test approach all verified against in-repo code + approved UI-SPEC + official docs
- Pitfalls: HIGH — each pitfall traced to a verified in-repo fact (line-referenced) or a documented version/API difference

**Research date:** 2026-08-12
**Valid until:** 2026-08-19 (7 days) — the TanStack Table version landscape is moving (v9 released 2026-08-09); re-verify dist-tags if planning extends beyond this window
