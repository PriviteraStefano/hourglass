# Phase 15: UX Foundation — Design Tokens + Shared Components - Context

**Gathered:** 2026-08-12
**Status:** Ready for planning

<domain>
## Phase Boundary

UI-only phase (no backend, no new capabilities). The design system is frozen before any surface or polish work, per UXFD-01 and UXFD-02:

- **Semantic status/state tokens** — role-based status palette (neutral/info/success/warning/danger) lands in `web/src/index.css` as light+dark CSS variable pairs, mapped through `@theme inline` per the existing Tailwind 4 pattern (UXFD-01 / ROADMAP SC1)
- **Frozen shared component set** — PageHeader (with status summary strip), FilterBar, DataTable (on @tanstack/react-table), StatusBadge variants (subtle/solid/outline/dot), EmptyState, ConfirmDialog — consumed by every new and polished page from Phase 16 on (UXFD-01 / ROADMAP SC2)
- **Sketch loop contract** — standalone contract doc at `.planning/sketches/SKETCH-LOOP-CONTRACT.md` establishing the gsd-sketch flow for all surface/polish phases (UXFD-02 / ROADMAP SC3)
- **Sidebar quick task triage** — ROADMAP SC4: the collapsed-mode quick task is already fixed (commit `54f465a`, 2026-08-01); the only remaining item is a human visual verification of collapsed-mode hover/navigation. No further code work expected.
- **UI-SPEC** — this phase produces a UI-SPEC.md via `/gsd-ui-phase` (precedent: `13-UI-SPEC.md`), making the frozen tokens/components a verifiable design contract

Not in scope: migrating the time-entries table/filters or page-level hardcoded colors (deferred to their polish phases), P-011 IA revision (D-O: only after prototypes land), backend changes.

</domain>

<decisions>
## Implementation Decisions

### Status palette (UXFD-01)
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

### Frozen component set (UXFD-01 SC2)
- **D-15-05:** **DataTable built on `@tanstack/react-table`** — new dependency; headless table + the existing `ui/table.tsx` primitives. Sorting, column definitions, and pagination are built into the frozen DataTable. (The current hand-rolled `shared/entries-table.tsx` stays for Phase 21, which migrates it.)
- **D-15-06:** **StatusBadge variants frozen: subtle (default), solid, outline, dot.** Color comes from the D-15-03 status→role mapping; the hardcoded Tailwind classes in the current `status-badge.tsx` are fully replaced. Dot variant = color dot + label for compact lists.
- **D-15-07:** **ConfirmDialog is consolidated NOW** — the shared component absorbs the 3 existing per-page delete-confirm dialogs (customers, org-hierarchy, working-groups) in this phase. It supports the required-reason destructive pattern (precedent D-13-10/D-13-16: reason-less destructive writes are rejected server-side with 400, and the dialog must present the required-reason input). Every future destructive action (absence reject, claim unclaim) uses it.
- **D-15-08:** **PageHeader = title + optional description + right-aligned actions slot + status summary strip** (counts or key metric badges next to the title — chosen over the minimal title+desc+actions). **EmptyState = thin wrapper over the existing `ui/empty.tsx`** composing icon/title/description/action with a default look. **FilterBar** is the generic search/filter/reset component in the frozen set (details below are planner discretion).

### Sketch loop contract (UXFD-02)
- **D-15-09:** **Standalone contract doc: `.planning/sketches/SKETCH-LOOP-CONTRACT.md`** — pins: every surface/polish phase (16–26) runs gsd-sketch first; 2–3 variants shown; user agrees on one; UI-only plans; ≤3 sketch rounds maximum; sketch MANIFEST updated; `--wrap-up` produces sketch-findings. Downstream phases inherit it from this file, not from CONTEXT.md.
- **D-15-10:** **This phase produces a UI-SPEC.md via `/gsd-ui-phase 15`** (precedent: `13-UI-SPEC.md`) — locks the design-system contract (tokens, spacing, typography, color roles) that planner and gsd-ui-checker verify against.
- **D-15-11:** **The 3-round cap is the ONLY hard rule** — no minimum round enforced; phase plans decide how many rounds they actually run (1–2 for polish, 2–3 for new surfaces like the scheduler).

### Migration scope
- **D-15-12:** **Shared-only migration** — this phase refactors only the shared layer: `status-badge.tsx` rewrite (D-15-06), the 3 delete dialogs → ConfirmDialog (D-15-07), plus the type unions (D-15-13). The time-entries table/filters and every page-level hardcoded color stay untouched — their polish phases (21 Track, 23 Approvals+WG, 24 Customers+Contracts) do that migration using the frozen set. The "full sweep" option was explicitly rejected (would make Phase 15 a refactor phase and leave polish phases with nothing to do).
- **D-15-13:** **New status vocabularies typed in this phase** — `TicketStatus` / `DirectionStatus` / `AbsenceStatus` unions added to `web/src/types/models.ts` mirroring the backend vocabularies (tickets: open/triage/planned/in_progress/resolved/closed/dismissed; direction: draft/active/superseded/cancelled; absences: declared/confirmed/rejected/withdrawn), and StatusBadge is typed generically so Phase 16/18/19 surfaces consume it without type churn. No UI for these statuses this phase.

### the agent's Discretion
- Exact token variable names (`--status-{role}`, `--status-{role}-foreground`, dark-mode pairs) and whether danger reuses `--destructive` or gets a parallel `--status-danger` — must follow the existing `:root`/`.dark`/`@theme inline` pattern
- FilterBar's exact API (search input, select/dropdown filters, date range, reset, active-filter count) — it is in the frozen set, details open
- PageHeader breadcrumb slot, EmptyState default icon set, DataTable pagination style (page-size selector etc.)
- Whether governance badges render as StatusBadge neutral variants or a separate small component
- Component file layout under `web/src/components/shared/` (kebab-case, named exports, `__tests__` colocation per CONVENTIONS.md)
- The sidebar human-visual-verification task shape (SC4 — fix already landed)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone docs
- `.planning/ROADMAP.md` — Phase 15 entry (lines ~219–232): goal, requirements UXFD-01/UXFD-02, 4 success criteria, "UI hint: yes"
- `.planning/REQUIREMENTS.md` — UXFD-01/UXFD-02 requirement text; status vocabularies in TICK/COV/DIR/AVAIL requirements the palette must cover
- `.planning/STATE.md` — Deferred Items table (sidebar quick task triage status; UAT debt folding plan for polish phases)

### Prior phase contracts (locked decisions)
- `.planning/phases/13-direction-backend-the-plan-plane/13-UI-SPEC.md` — design system FROZEN state as of Phase 13; status colors explicitly deferred to this phase ("Phase 15's semantic-token job"); accent never used for status semantics; server-emitted warning object contracts (`away`/`partial`/`over-capacity`/`invalid`); required-reason destructive-write pattern (D-13-10/D-13-16)
- `.planning/phases/14-availability-backend-absences-capacity/14-CONTEXT.md` — absence status vocabulary (declared/confirmed/rejected/withdrawn, D-14-08) and confirmed-only read path

### Sketch loop
- `.config/opencode/skills/gsd-sketch/SKILL.md` — gsd-sketch mechanics: 2–3 variants, `.planning/sketches/` location, MANIFEST tracking, `--wrap-up` producing sketch-findings skills
- New file created this phase: `.planning/sketches/SKETCH-LOOP-CONTRACT.md` (D-15-09)

### Codebase (read-only context)
- `web/src/index.css` — current token structure: `:root`/`.dark` CSS variables + `@theme inline` mapping; `--destructive` reuse candidate
- `web/components.json` — shadcn config: `base-mira` style, base color `olive`, Base UI components, lucide icons
- `web/src/components/shared/status-badge.tsx` — the hardcoded Tailwind status map being replaced (D-15-06)
- `web/src/components/ui/empty.tsx` — EmptyState wrapper target (D-15-08)
- `web/src/routes/_authenticated/{customers,org-hierarchy,working-groups}/-components/.../*delete-confirm-dialog.tsx` — the 3 dialogs absorbed into ConfirmDialog (D-15-07)
- `web/src/types/models.ts` — `EntryStatus` union; the pattern for the new `TicketStatus`/`DirectionStatus`/`AbsenceStatus` unions (D-15-13)
- `.planning/quick/260801-got-investigate-sidebar-collapsed-mode-hover/260801-got-SUMMARY.md` — sidebar fix summary; pre-existing build breakage listed there is already resolved (verified `bun run build` passes 2026-08-12)

### Research note
- `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md` — D-O: surfaces/IA deferred to UI prototyping; no P-011 revision until prototypes land (do NOT revise IA in this phase)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/components/ui/` — 55+ shadcn primitives (badge, table, alert-dialog, dialog, button, input, select, calendar) — the building blocks every frozen component composes
- `web/src/components/ui/empty.tsx` — EmptyState wrapper target (cva variants already present)
- `web/src/components/shared/status-badge.tsx` — existing shared component to rewrite onto the role tokens
- `web/src/components/shared/entries-table.tsx` / `entries-filters.tsx` — hand-rolled table/filter patterns DataTable/FilterBar generalize (NOT migrated this phase, D-15-12)
- 3 per-page `delete-confirm-dialog.tsx` (customers, org-hierarchy, working-groups) + `reparent-confirm-dialog.tsx` — consolidation input for ConfirmDialog
- `web/src/lib/utils.ts` (`cn`) + cva (class-variance-authority) — the variant pattern StatusBadge/EmptyState already use

### Established Patterns
- Tailwind 4 CSS-first: `:root`/`.dark` variable pairs + `@theme inline` in `index.css` (verified against current Tailwind v4 docs) — new status tokens follow this exact shape
- shadcn component style: `data-slot` attributes, cva variants, `cn()` composition, named exports only, kebab-case filenames, `__tests__` colocation with `.test.tsx` (CONVENTIONS.md)
- oxfmt (80-col, semicolons, double quotes), `import type` for type-only imports, `.ts`/`.tsx` extensions on `@/` imports
- React Query route-loader pattern unaffected; components stay presentational (server state stays in routes/hooks)
- JSDoc comments cite plan/ADR references (e.g., `(ADR-P-011)`) — new components should cite this phase's plan codes

### Integration Points
- `web/src/index.css` — token definitions land here (`:root`, `.dark`, `@theme inline`)
- `web/src/types/models.ts` + `web/src/types/index.ts` barrel — new status unions (D-15-13)
- `web/src/components/shared/` — frozen components land here
- `web/package.json` — `@tanstack/react-table` added (D-15-05)
- `.planning/sketches/` — SKETCH-LOOP-CONTRACT.md created this phase (D-15-09)
- Routes consuming the frozen set from Phase 16 on: all `_authenticated` pages; the 3 consolidated dialog sites are the first in-repo consumers
- `13-UI-SPEC.md` status — Phase 15's UI-SPEC supersedes the "status colors deferred" note; keep the frozen design-system record updated

</code_context>

<specifics>
## Specific Ideas

- User's rationale for meaning-first hues: green should read "approved/done" only; the current `pending_manager` green is semantically misleading. Users accept re-learning the palette (this is the moment to do it, before new surfaces ship).
- User explicitly rejected a "full sweep" refactor of all pages in this phase — polish phases must have migration work left for them (21/23/24).
- No specific external design references given — the palette comes from the role mapping table (D-15-03).

</specifics>

<deferred>
## Deferred Ideas

- **Time-entries table → DataTable + entries-filters → FilterBar migration** — Phase 21 (Track polish) via the frozen set (D-15-12)
- **Page-level hardcoded status colors sweep** (today, WG, activities, customers, contracts, exports pages) — folded into their polish phases (20–26); ROADMAP SC1 ("no surface/polish phase introduces ad-hoc hex values") applies from Phase 16 on
- **P-011 IA revision** — stays deferred until prototypes land (research D-O); do NOT touch IA in this phase
- **Sidebar collapsed-mode human visual verification** — the code fix is landed (commit `54f465a`); only a human check of collapsed-mode hover/navigation remains, surfaced as a task in this phase (SC4 triage complete)

</deferred>

---

*Phase: 15-UX Foundation — Design Tokens + Shared Components*
*Context gathered: 2026-08-12*
